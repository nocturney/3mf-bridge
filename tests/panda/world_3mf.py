"""Placed 3MF triangle reader for conversion QA (stdlib, synthetic fixtures only).

Traverses the build graph across p:path model parts and recursively applies both
component and build transforms. Unreferenced meshes do NOT count as printed
geometry. This is structural QA, not slicer/import/printability validation.
"""
from __future__ import annotations

from collections import Counter
import math
import posixpath
import xml.etree.ElementTree as ET

IDENTITY = (1., 0., 0., 0., 1., 0., 0., 0., 1., 0., 0., 0.)
UNITS = {"millimeter": 1., "millimetre": 1., "micron": .001,
         "centimeter": 10., "metre": 1000., "meter": 1000.,
         "inch": 25.4, "foot": 304.8}
PAINT = {"paint_color", "mmu_segmentation"}


class SceneError(ValueError):
    pass


def local(tag):
    return tag.rsplit("}", 1)[-1]


def matrix(raw):
    if not raw:
        return IDENTITY
    try:
        values = tuple(float(value) for value in raw.split())
    except ValueError as exc:
        raise SceneError("invalid 3MF transform") from exc
    if len(values) != 12 or not all(math.isfinite(v) for v in values):
        raise SceneError("3MF transform requires 12 finite numbers")
    return values


def compose(a, b):
    # Row-vector affine matrices: apply a, then b.
    rows_a = (a[0:3], a[3:6], a[6:9])
    rows_b = (b[0:3], b[3:6], b[6:9])
    linear = [sum(rows_a[i][k] * rows_b[k][j] for k in range(3))
              for i in range(3) for j in range(3)]
    offset = [sum(a[9+k] * rows_b[k][j] for k in range(3)) + b[9+j]
              for j in range(3)]
    return tuple(linear + offset)


def apply(m, v, scale, decimals):
    # 3MF object coordinates are in model units; transforms share those units.
    out = tuple(round((sum(v[k] * m[3*k+j] for k in range(3)) + m[9+j])
                      * scale, decimals) + 0.0 for j in range(3))
    if not all(math.isfinite(x) for x in out):
        raise SceneError("non-finite placed vertex")
    return out


def related_part(parent, raw):
    if not raw or "\\" in raw or "?" in raw or "#" in raw:
        raise SceneError("invalid cross-part p:path")
    if raw.startswith("/"):
        target = raw.lstrip("/")
    elif raw.startswith("3D/"):
        target = raw
    else:
        target = posixpath.join(posixpath.dirname(parent), raw)
    name = posixpath.normpath(target)
    if (name.startswith("../") or name == ".." or not name.startswith("3D/")
            or not name.lower().endswith(".model") or "\x00" in name):
        raise SceneError("unsafe cross-part p:path")
    return name


def child_part(part, element):
    value = next((v for k, v in element.attrib.items() if local(k) == "path"), None)
    return related_part(part, value) if value else part


class Scene:
    def __init__(self, files, decimals=4, paint_normalizer=None):
        self.files = files
        self.decimals = decimals
        self.paint_normalizer = paint_normalizer or (lambda value: value)
        self._parts = {}
        self.triangles = Counter()
        self.items = []   # (root object id, geometry Counter)
        self._triangle_budget = 2_000_000
        self._build()

    def _load(self, part):
        if part in self._parts:
            return self._parts[part]
        if len(self._parts) >= 512 or part not in self.files:
            raise SceneError("missing or excessive 3MF model part: " + part)
        try:
            root = ET.fromstring(self.files[part])
        except ET.ParseError as exc:
            raise SceneError("invalid 3MF XML part: " + part) from exc
        unit = root.get("unit", "millimeter").lower()
        if unit not in UNITS:
            raise SceneError("unsupported 3MF model unit: " + unit)
        objects = {}
        for resources in root:
            if local(resources.tag) != "resources":
                continue
            for obj in resources:
                if local(obj.tag) == "object":
                    oid = obj.get("id")
                    if not oid or oid in objects:
                        raise SceneError("missing/duplicate 3MF object id")
                    objects[oid] = obj
        build = [i for group in root if local(group.tag) == "build"
                 for i in group if local(i.tag) == "item"]
        data = (objects, build, UNITS[unit])
        self._parts[part] = data
        return data

    def _walk(self, part, oid, m, seen, destination, root_scale):
        identity = (part, oid)
        if identity in seen or len(seen) >= 20:
            raise SceneError("cyclic or excessively nested 3MF components")
        objects, _, scale = self._load(part)
        if abs(scale - root_scale) > 1e-12:
            raise SceneError("mixed model-part units are not supported")
        obj = objects.get(oid)
        if obj is None:
            raise SceneError("unknown 3MF object " + part + ":" + str(oid))
        seen.add(identity)
        try:
            for mesh in obj:
                if local(mesh.tag) != "mesh":
                    continue
                verts = []
                for block in mesh:
                    if local(block.tag) == "vertices":
                        for vertex in block:
                            if local(vertex.tag) != "vertex":
                                continue
                            try:
                                point = tuple(float(vertex.get(k)) for k in "xyz")
                            except (TypeError, ValueError) as exc:
                                raise SceneError("invalid 3MF vertex") from exc
                            if not all(math.isfinite(v) for v in point):
                                raise SceneError("non-finite 3MF vertex")
                            verts.append(apply(m, point, scale, self.decimals))
                for block in mesh:
                    if local(block.tag) != "triangles":
                        continue
                    for tri in block:
                        if local(tri.tag) != "triangle":
                            continue
                        try:
                            idx = [int(tri.get(k)) for k in ("v1", "v2", "v3")]
                        except (TypeError, ValueError) as exc:
                            raise SceneError("invalid 3MF triangle index") from exc
                        if any(i < 0 or i >= len(verts) for i in idx):
                            raise SceneError("3MF triangle index out of range")
                        geom = tuple(sorted(verts[i] for i in idx))
                        code = next((v for k, v in tri.attrib.items() if local(k) in PAINT), "")
                        destination[(geom, self.paint_normalizer(code))] += 1
                        self._triangle_budget -= 1
                        if self._triangle_budget < 0:
                            raise SceneError("3MF placed triangle limit exceeded")
            for group in obj:
                if local(group.tag) != "components":
                    continue
                for comp in group:
                    if local(comp.tag) == "component":
                        self._walk(child_part(part, comp), comp.get("objectid"),
                                   compose(matrix(comp.get("transform")), m),
                                   seen, destination, root_scale)
        finally:
            seen.remove(identity)

    def _build(self):
        root = "3D/3dmodel.model"
        _, build, scale = self._load(root)
        for item in build:
            if item.get("printable", "1") == "0":
                continue
            placed = Counter()
            self._walk(child_part(root, item), item.get("objectid"),
                       matrix(item.get("transform")), set(), placed, scale)
            self.triangles.update(placed)
            self.items.append((item.get("objectid"), placed))

    def geometry(self):
        return Counter({geom: count for geom, count in self._geometry_counted().items()})

    def _geometry_counted(self):
        output = Counter()
        for (coords, _paint), count in self.triangles.items():
            output[coords] += count
        return output

    def paints(self):
        counts = Counter()
        for (_geom, paint), count in self.triangles.items():
            counts[paint] += count
        return counts
