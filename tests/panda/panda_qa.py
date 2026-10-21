"""Stdlib helpers for Panda2Prusa conversion QA (Bambu 3MF source vs converted output).

Read-only with respect to inputs: packages are read into memory, and mutated copies are
written to a caller-chosen temp path. Paint decoding mirrors tests/compat/threemf_compat.py
(4->1, 8->2, C->3, 1C->4, 2C->5); the paint strings themselves are compared verbatim.
"""
from __future__ import annotations

import collections
import hashlib
import json
import os
import re
import subprocess
import sys
import xml.etree.ElementTree as ET
import zipfile
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
FIXTURES = REPO / "tests" / "fixtures"
FIVE_COLORS = FIXTURES / "bambu-five-colors.3mf"
VENDOR = REPO / "third_party" / "panda2prusa"
VENDOR_RUNNER = VENDOR / "slicebridge_runner.py"
ENGINE_NAME = "slicebridge-prusa-engine" + (".exe" if os.name == "nt" else "")
DIST_ENGINE = REPO / "dist" / "prusa-sidecar" / ENGINE_NAME
PAINT_ATTRS = ("paint_color", "mmu_segmentation")
STALE_MARKERS = (b"stale printer-specific slice", b"UNSAFE_BAMBU_SCRIPT")
GCODE_SUFFIXES = (".gcode", ".gcode.md5")
COORD_DECIMALS = 4  # coordinates compare to 0.0001 mm
CONVERT_TIMEOUT = 300


def sha256_file(path):
    digest = hashlib.sha256()
    with open(path, "rb") as handle:
        for chunk in iter(lambda: handle.read(1 << 20), b""):
            digest.update(chunk)
    return digest.hexdigest()


def local(tag):
    return tag.rsplit("}", 1)[-1]


def children(element, name):
    return [child for child in element if local(child.tag) == name]


def descendants(element, name):
    return [el for el in element.iter() if local(el.tag) == name]


def decode_paint_channel(code):
    """1-based extruder channel for a Bambu paint code, or None when the code is not recognised."""
    if not re.fullmatch(r"[0-9A-Fa-f]+", code or ""):
        return None
    value = int(code, 16)
    if value in (0x4, 0x8, 0xC):
        return value >> 2
    if value & 0xF == 0xC and value >> 4:
        return 3 + (value >> 4)
    return None


class Package:
    """A 3MF ZIP held in memory, with the model and the config files the QA reads."""

    def __init__(self, path):
        self.path = Path(path)
        with zipfile.ZipFile(self.path) as archive:
            self.names = archive.namelist()
            self.files = {name: archive.read(name) for name in self.names}

    def text(self, name):
        return self.files[name].decode("utf-8", "replace")

    def has(self, name):
        return name in self.files

    def json_config(self, name):
        return json.loads(self.text(name))

    def xml(self, name):
        return ET.fromstring(self.files[name])


def _matrix(text):
    """A 3MF transform (12 numbers, row-vector form) as a 4x4 matrix."""
    values = [float(v) for v in (text or "1 0 0 0 1 0 0 0 1 0 0 0").split()]
    if len(values) != 12:
        raise ValueError("3MF transform must have 12 numbers")
    return [[values[0], values[1], values[2], 0.0],
            [values[3], values[4], values[5], 0.0],
            [values[6], values[7], values[8], 0.0],
            [values[9], values[10], values[11], 1.0]]


def _multiply(a, b):
    return [[sum(a[i][k] * b[k][j] for k in range(4)) for j in range(4)] for i in range(4)]


def _apply(matrix, vertex):
    x, y, z = vertex
    return tuple(round(x * matrix[0][j] + y * matrix[1][j] + z * matrix[2][j] + matrix[3][j],
                       COORD_DECIMALS) for j in range(3))


def _round_tuple(values):
    return tuple(round(v, COORD_DECIMALS) for v in values)


class Model:
    """Objects, meshes and build items from 3D/3dmodel.model.

    Triangles are compared in object-local space, paired with the paint string they carry.
    Build transforms are compared as 4x4 matrices.
    """

    def __init__(self, package):
        root = package.xml("3D/3dmodel.model")
        self.objects = {}
        for obj in descendants(root, "object"):
            oid = obj.get("id")
            mesh = children(obj, "mesh")
            entry = {"mesh": None, "components": []}
            if mesh:
                vertices = [_round_tuple((float(v.get("x")), float(v.get("y")), float(v.get("z"))))
                            for v in descendants(mesh[0], "vertex")]
                triangles = []
                for tri in descendants(mesh[0], "triangle"):
                    corners = [int(tri.get(k)) for k in ("v1", "v2", "v3")]
                    # Prusa writes the paint attribute in the slic3rpe XML
                    # namespace; Bambu writes an unqualified paint_color.
                    paint = next((value for key, value in tri.attrib.items()
                                  if local(key) in PAINT_ATTRS), "")
                    triangles.append((corners, paint))
                entry["mesh"] = {"vertices": vertices, "triangles": triangles}
            for comp in descendants(obj, "component"):
                entry["components"].append((comp.get("objectid"), _matrix(comp.get("transform"))))
            self.objects[oid] = entry
        self.items = [(item.get("objectid"), _matrix(item.get("transform")))
                      for item in descendants(root, "item")]

    def local_triangles(self, oid, matrix=None, depth=0):
        """All triangles of an object and its components, in the requested space."""
        if depth > 16:
            raise ValueError("component nesting too deep")
        base = matrix or _matrix(None)
        entry = self.objects.get(oid)
        if entry is None:
            raise ValueError("build item references unknown object " + str(oid))
        found = []
        if entry["mesh"]:
            verts = [_apply(base, v) for v in entry["mesh"]["vertices"]]
            for corners, paint in entry["mesh"]["triangles"]:
                key = tuple(sorted(verts[c] for c in corners))
                found.append((key, paint))
        for child, child_matrix in entry["components"]:
            found.extend(self.local_triangles(child, _multiply(child_matrix, base), depth + 1))
        return found

    def mesh_records(self):
        """Every mesh triangle as (object-local key, paint), independent of build placement."""
        records = collections.Counter()
        # A Bambu project can store geometry behind component transforms;
        # Panda2Prusa flattens those transforms into a new mesh. Compare
        # placed, world-space triangles rather than raw object-local vertices.
        for oid, transform in self.items:
            for key, paint in self.local_triangles(oid, transform):
                records[(key, paint)] += 1
        return records

    def signature(self, oid):
        """Hash of an object's local geometry plus paint, used to match objects across formats."""
        records = sorted(self.local_triangles(oid))
        return hashlib.sha256(json.dumps(records).encode()).hexdigest()

    def transforms(self):
        return sorted(tuple(_round_tuple(sum(matrix, []))) for _, matrix in self.items)


def _metadata(element):
    return {m.get("key"): m.get("value") for m in children(element, "metadata")}


def source_settings(package):
    """Bambu per-object and per-part extruder assignments from Metadata/model_settings.config."""
    if not package.has("Metadata/model_settings.config"):
        return {}
    root = package.xml("Metadata/model_settings.config")
    result = {}
    for obj in descendants(root, "object"):
        oid = obj.get("id")
        parts = {}
        for part in children(obj, "part"):
            meta = _metadata(part)
            if "extruder" in meta:
                parts[part.get("id")] = int(meta["extruder"])
        meta = _metadata(obj)
        result[oid] = {"extruder": int(meta["extruder"]) if "extruder" in meta else None,
                       "parts": parts}
    return result


def output_settings(package):
    """Prusa per-object and per-volume extruder assignments from Metadata/Slic3r_PE_model.config."""
    if not package.has("Metadata/Slic3r_PE_model.config"):
        return {}
    root = package.xml("Metadata/Slic3r_PE_model.config")
    result = {}
    for obj in descendants(root, "object"):
        meta = _metadata(obj)
        volumes = []
        for volume in descendants(obj, "volume"):
            vmeta = _metadata(volume)
            count = int(volume.get("lastid")) - int(volume.get("firstid")) + 1
            volumes.append((count, int(vmeta["extruder"]) if "extruder" in vmeta else None))
        result[obj.get("id")] = {"extruder": int(meta["extruder"]) if "extruder" in meta else None,
                                 "volumes": volumes}
    return result


def source_palette(package):
    """Ordered filament colours for slots 1..N (upper case, duplicates kept)."""
    if package.has("Metadata/project_settings.config"):
        cfg = package.json_config("Metadata/project_settings.config")
        if isinstance(cfg.get("filament_colour"), list):
            return [str(c).strip().upper() for c in cfg["filament_colour"]]
    if package.has("Metadata/slice_info.config"):
        root = package.xml("Metadata/slice_info.config")
        found = descendants(root, "filament")
        found.sort(key=lambda el: int(el.get("id", "0")))
        return [el.get("color", "").strip().upper() for el in found]
    return None


def output_palette(package):
    """Ordered filament colours from the Prusa config (INI), falling back to a JSON project config."""
    if package.has("Metadata/SliceBridge_source_filaments.json"):
        # Panda2Prusa intentionally does not install or forge a Prusa
        # filament preset. SliceBridge retains the exact source slot list
        # in an auxiliary 3MF part for lossless reference and manual mapping.
        data = package.json_config("Metadata/SliceBridge_source_filaments.json")
        slots = data.get("slots")
        if data.get("schemaVersion") == 1 and isinstance(slots, list):
            if [slot.get("id") for slot in slots] == list(range(1, len(slots) + 1)):
                return [str(slot.get("color", "")).strip().upper() for slot in slots]
    if package.has("Metadata/Slic3r_PE.config"):
        match = re.search(r"^\s*filament_colour\s*=\s*(.*?)\s*$", package.text("Metadata/Slic3r_PE.config"), re.M)
        if match:
            raw = match.group(1).strip().strip('"')
            return [item.strip().strip('"').upper() for item in raw.split(";") if item.strip()]
    if package.has("Metadata/project_settings.config"):
        cfg = package.json_config("Metadata/project_settings.config")
        if isinstance(cfg.get("filament_colour"), list):
            return [str(c).strip().upper() for c in cfg["filament_colour"]]
    return None


def compare(source, output):
    """Compare a Bambu source package with a converted output.

    Returns {check: [messages]}. An empty dict means every check matched.
    """
    problems = collections.defaultdict(list)
    try:
        src_model = Model(source)
        out_model = Model(output)
    except (ValueError, ET.ParseError, KeyError) as error:
        return {"model": ["cannot parse 3dmodel.model: %s" % error]}

    src_mesh, out_mesh = src_model.mesh_records(), out_model.mesh_records()
    if src_mesh != out_mesh:
        missing = sum((src_mesh - out_mesh).values())
        extra = sum((out_mesh - src_mesh).values())
        problems["geometry"].append("triangle records differ: %d missing, %d extra" % (missing, extra))

    if len(src_model.items) != len(out_model.items):
        problems["transforms"].append("build item count %d != %d" % (len(src_model.items), len(out_model.items)))
    elif src_model.transforms() != out_model.transforms():
        problems["transforms"].append("build transforms differ")

    src_paints = collections.Counter(paint for _, paint in src_mesh.elements())
    out_paints = collections.Counter(paint for _, paint in out_mesh.elements())
    if src_paints != out_paints:
        problems["paints"].append("paint bitstream multiset differs: source %s, output %s"
                                  % (dict(src_paints), dict(out_paints)))

    src_obj, out_obj = source_settings(source), output_settings(output)
    out_by_signature = {}
    for oid, entry in out_model.objects.items():
        if entry["mesh"]:
            out_by_signature.setdefault(out_model.signature(oid), oid)
    for oid, entry in src_model.objects.items():
        if not entry["mesh"] and not entry["components"]:
            continue
        src_entry = src_obj.get(oid, {"extruder": None, "parts": {}})
        match = out_by_signature.get(src_model.signature(oid))
        if match is None:
            if src_entry["extruder"] is not None or src_entry["parts"]:
                problems["extruders"].append("source object %s has no geometry-equal output object" % oid)
            continue
        out_entry = out_obj.get(match, {"extruder": None, "volumes": []})
        if src_entry["extruder"] is not None and out_entry["extruder"] != src_entry["extruder"]:
            problems["extruders"].append("object %s extruder %s != output %s"
                                         % (oid, src_entry["extruder"], out_entry["extruder"]))
        src_parts = sorted(src_entry["parts"].values())
        out_parts = sorted(e for _, e in out_entry["volumes"] if e is not None)
        if src_parts and src_parts != out_parts:
            problems["extruders"].append("object %s volume extruders %s != output %s"
                                         % (oid, src_parts, out_parts))

    src_palette_list, out_palette_list = source_palette(source), output_palette(output)
    if src_palette_list is not None:
        if out_palette_list is None:
            problems["palette"].append("output has no filament colour list")
        elif src_palette_list != out_palette_list:
            problems["palette"].append("numbered slots differ: source %s, output %s"
                                       % (src_palette_list, out_palette_list))
        used = {decode_paint_channel(p) for _, p in src_mesh.elements() if p}
        used |= {entry["extruder"] for entry in src_obj.values() if entry["extruder"] is not None}
        bad = sorted(c for c in used if c is not None and c > len(src_palette_list))
        if bad:
            problems["palette"].append("source uses slots beyond its palette: %s" % bad)
    return {check: messages for check, messages in problems.items() if messages}


def stale_gcode_problems(output):
    """Gcode carried over from the Bambu project must not survive into the converted file."""
    problems = []
    for name in output.names:
        if name.endswith(GCODE_SUFFIXES):
            problems.append("stale gcode entry %s" % name)
    for name, blob in output.files.items():
        for marker in STALE_MARKERS:
            if marker in blob:
                problems.append("stale source text %r found in %s" % (marker.decode(), name))
    return problems


def write_variant(source, dest, replacements):
    """Write a copy of a 3MF with members replaced or added. The source is only read."""
    with zipfile.ZipFile(source) as archive:
        members = [(info.filename, archive.read(info.filename)) for info in archive.infolist()]
    names = {name for name, _ in members}
    with zipfile.ZipFile(dest, "w", zipfile.ZIP_DEFLATED) as archive:
        for name, blob in members:
            archive.writestr(name, replacements.get(name, blob))
        for name, blob in replacements.items():
            if name not in names:
                archive.writestr(name, blob)
    return Path(dest)


def engine_command():
    """How to run the converter: a dist build, an explicit override, or the vendored runner.

    Returns (argv prefix, description) or None when nothing is available yet.
    """
    override = os.environ.get("SLICEBRIDGE_PRUSA_ENGINE")
    if override:
        return [override], "SLICEBRIDGE_PRUSA_ENGINE"
    if DIST_ENGINE.is_file():
        return [str(DIST_ENGINE)], "dist/prusa-sidecar"
    if VENDOR_RUNNER.is_file():
        return [sys.executable, str(VENDOR_RUNNER)], "vendored runner (python)"
    return None


def convert(engine, source, output, cwd):
    """Run the runner contract: --source SRC --output OUT --json. Returns the completed process."""
    command = list(engine) + ["--source", str(source), "--output", str(output), "--json"]
    return subprocess.run(command, capture_output=True, text=True, timeout=CONVERT_TIMEOUT, cwd=str(cwd))
