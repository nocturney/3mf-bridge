#!/usr/bin/env python3
"""Conversion fidelity QA for SliceBridge 3MF output (stdlib only).

Compares a source project with a converted output. It never slices, never
touches a printer or the network, and never claims that the output prints.
Checks that cannot be compared across formats are reported as "skip".

  python3 tests/compat/threemf_compat.py --source SRC.3mf --output OUT.3mf \
      [--family auto|orca|bambu|prusa] [--expect-printer-model NAME] \
      [--source-sha256 HEX] [--prior-output-sha256 HEX]

Exit codes: 0 no failed checks, 1 at least one failed check, 2 unreadable input.
"""
import argparse
import hashlib
import json
import os
import posixpath
import re
import sys
import xml.etree.ElementTree as ET
import zipfile
import zlib
from collections import Counter
from pathlib import Path

# Shared placement-aware parser lives with Panda QA. Keep direct CLI invocation
# (python tests/compat/threemf_compat.py) working from any current directory.
sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "panda"))
from world_3mf import Scene, SceneError

FAMILIES = ("orca", "bambu", "prusa")
# Orca and Bambu share the JSON-config layout and the paint codec; Prusa differs in both.
CODEC = {"orca": "bbl-json", "bambu": "bbl-json", "prusa": "prusa-ini"}
MAX_ENTRIES = 10000
MAX_UNCOMPRESSED = 1 << 30
MODEL_REL_SUFFIX = "/3dmanufacturing/2013/01/3dmodel"
MODEL_CONFIGS = ("Metadata/model_settings.config", "Metadata/Slic3r_PE_model.config")
PAINT_ATTRS = ("paint_color", "mmu_segmentation")
GCODE_SUFFIXES = (".gcode", ".gcode.md5")
# Numeric process intent the engine documents as carried over
# (mirrors safePrintParameters in engine/bridge/settings.go).
PROCESS_KEYS = (
    "layer_height", "initial_layer_print_height", "wall_loops", "top_shell_layers",
    "bottom_shell_layers", "sparse_infill_density", "support_threshold_angle",
    "brim_width", "skirt_loops", "skirt_distance", "elefant_foot_compensation",
    "elephant_foot_compensation", "xy_contour_compensation",
)
NOTES = [
    "Structural comparison only: nothing is sliced and no firmware or printer is validated.",
    "A passing report does not prove that the output prints or that it is safe to print.",
]


class InputError(Exception):
    """A file cannot be read as a 3MF package, or its layout is not understood."""


def _local(tag):
    return tag.rsplit("}", 1)[-1]


def _iter(root, name):
    return [el for el in root.iter() if _local(el.tag) == name]


def _unsafe(name):
    parts = name.split("/")
    return name.startswith("/") or "\\" in name or ".." in parts or ":" in parts[0]


class Package:
    """A 3MF ZIP. Defects are recorded in ``problems`` so reports can name them."""

    def __init__(self, path):
        if not os.path.isfile(path):
            raise InputError(f"not a file: {path}")
        self.path = path
        self.files = {}
        self.problems = []
        try:
            archive = zipfile.ZipFile(path)
        except (zipfile.BadZipFile, OSError) as exc:
            raise InputError(f"not a readable ZIP/3MF: {exc}")
        with archive:
            infos = archive.infolist()
            if len(infos) > MAX_ENTRIES:
                self.problems.append(f"too many entries: {len(infos)}")
            for name, count in sorted(Counter(i.filename for i in infos).items()):
                if count > 1:
                    self.problems.append(f"duplicate entry: {name}")
            total = 0
            for info in infos:
                name = info.filename
                if name.endswith("/"):
                    continue
                if _unsafe(name):
                    self.problems.append(f"unsafe entry name: {name}")
                    continue
                if info.flag_bits & 0x1:
                    self.problems.append(f"encrypted entry: {name}")
                    continue
                total += info.file_size
                if total > MAX_UNCOMPRESSED:
                    self.problems.append("decompressed size limit exceeded")
                    break
                try:
                    self.files[name] = archive.read(info)
                except (zipfile.BadZipFile, NotImplementedError, RuntimeError, zlib.error, OSError) as exc:
                    self.problems.append(f"unreadable entry {name}: {exc}")

    def xml(self, name):
        if name not in self.files:
            raise InputError(f"missing {name}")
        try:
            return ET.fromstring(self.files[name])
        except ET.ParseError as exc:
            raise InputError(f"invalid XML in {name}: {exc}")

    def json_config(self, name):
        if name not in self.files:
            return None
        try:
            data = json.loads(self.files[name].decode("utf-8"))
        except (ValueError, UnicodeDecodeError) as exc:
            raise InputError(f"invalid JSON in {name}: {exc}")
        if not isinstance(data, dict):
            raise InputError(f"{name} is not a JSON object")
        return data


def _application(pkg):
    if "3D/3dmodel.model" not in pkg.files:
        return ""
    try:
        root = ET.fromstring(pkg.files["3D/3dmodel.model"])
    except ET.ParseError:
        return ""
    for el in _iter(root, "metadata"):
        if el.get("name") == "Application":
            return (el.text or "").strip()
    return ""


def detect_family(pkg):
    """Return (family, basis). Family is None when no known layout is present."""
    if "Metadata/Slic3r_PE.config" in pkg.files or "Metadata/Slic3r_PE_model.config" in pkg.files:
        return "prusa", "layout"
    if "Metadata/project_settings.config" in pkg.files or "Metadata/model_settings.config" in pkg.files:
        app = _application(pkg)
        if app.startswith("BambuStudio"):
            return "bambu", "application-metadata"
        if app.startswith("OrcaSlicer"):
            return "orca", "application-metadata"
        cfg = pkg.json_config("Metadata/project_settings.config") or {}
        if str(cfg.get("printer_model", "")).startswith("Bambu"):
            return "bambu", "printer-model"
        # The JSON layout is shared; without a hint the label is a default only.
        return "orca", "layout-default"
    return None, "no-known-layout"


def model_parts(pkg):
    """Find root and child 3D model parts by their OPC model relationships.

    Real Bambu Studio / MakerWorld projects commonly keep actual triangle mesh
    in 3D/Objects/*.model, linked from 3D/_rels/3dmodel.model.rels; counting
    only the root 3D/3dmodel.model silently records zero triangles.
    """
    if "[Content_Types].xml" not in pkg.files:
        raise InputError("missing [Content_Types].xml")
    pkg.xml("[Content_Types].xml")
    pending=[]
    for rel in pkg.xml("_rels/.rels"):
        if _local(rel.tag) == "Relationship" and rel.get("Type", "").endswith(MODEL_REL_SUFFIX):
            if rel.get("TargetMode", "").lower() == "external":
                raise InputError("external model relationship is not allowed")
            pending.append((None, rel.get("Target", "")))
    if not pending:
        raise InputError("no 3D model relationship in _rels/.rels")
    parts=[]
    seen=set()
    while pending:
        parent,target=pending.pop(0)
        if not target or ("?" in target) or ("#" in target) or "\\" in target:
            raise InputError("invalid model relationship target")
        if target.startswith("/"):
            name=posixpath.normpath(target.lstrip("/"))
        else:
            name=posixpath.normpath(posixpath.join(
                posixpath.dirname(parent) if parent else "", target))
        if not name.startswith("3D/") or name=="3D/" or name.startswith("../") or ".." in name.split("/"):
            raise InputError(f"unsafe model relationship target: {target}")
        if not name.lower().endswith(".model"):
            raise InputError(f"non-model 3D relationship target: {target}")
        if name not in pkg.files:
            raise InputError(f"model relationship target missing: {name}")
        if name in seen:
            continue
        seen.add(name)
        parts.append(name)
        if len(parts)>512:
            raise InputError("too many related 3D model parts")
        relfile=posixpath.join(posixpath.dirname(name), "_rels",
                               posixpath.basename(name)+".rels")
        if relfile not in pkg.files:
            continue
        for rel in pkg.xml(relfile):
            if _local(rel.tag)!="Relationship" or not rel.get("Type", "").endswith(MODEL_REL_SUFFIX):
                continue
            if rel.get("TargetMode", "").lower()=="external":
                raise InputError("external model relationship is not allowed")
            pending.append((name,rel.get("Target","")))
    return parts


def _coord(el, key):
    try:
        return round(float(el.get(key)), 6) + 0.0  # +0.0 folds -0.0 into 0.0
    except (TypeError, ValueError):
        raise InputError(f"vertex has no numeric {key} coordinate")


def _paint_code(el):
    for key, value in el.attrib.items():
        if _local(key) in PAINT_ATTRS:
            return value.strip().upper().lstrip("0")
    return ""  # unpainted; "0" and an absent attribute both normalise to this


def geometry(pkg, parts):
    """Placed build geometry, not orphan meshes or raw pre-transform vertices.

    Root/child part availability was already validated by model_parts(). A
    cloned component counts once per *instance*, and coordinates are in mm.
    """
    if not parts:
        raise InputError("no model parts")
    try:
        scene = Scene(pkg.files, decimals=4,
                      paint_normalizer=lambda code: code.strip().upper().lstrip("0"))
    except SceneError as exc:
        raise InputError("cannot resolve placed 3MF geometry: " + str(exc)) from exc
    if not scene.triangles:
        raise InputError("3MF has no printable build triangles")
    placed = scene.geometry()
    digest = hashlib.sha256(json.dumps(sorted(placed.items())).encode()).hexdigest()
    vertices = len({vertex for triangle in placed for vertex in triangle})
    tris = [(triangle, paint) for (triangle, paint), count in scene.triangles.items()
            for _ in range(count)]
    return {"vertices": vertices, "triangles": sum(placed.values()),
            "hash": digest, "tris": tris, "_scene": scene}


def bbl_channel(code):
    """Decode a Bambu/Orca paint code to a 1-based extruder channel, or None if unknown.

    Scheme inferred from repo fixtures and generators: 4->1, 8->2, C->3, 1C->4, 2C->5.
    """
    if not re.fullmatch(r"[0-9A-F]+", code or ""):
        return None
    value = int(code, 16)
    if value in (0x4, 0x8, 0xC):
        return value >> 2
    if value & 0xF == 0xC and value >> 4:
        return 3 + (value >> 4)
    return None


def _slice_slots(pkg):
    found = _iter(pkg.xml("Metadata/slice_info.config"), "filament")
    try:
        found.sort(key=lambda e: int(e.get("id", "0")))
    except ValueError:
        raise InputError("slice_info filament id is not an integer")
    return [e.get("color", "") for e in found], [e.get("type", "") for e in found]


def _ini_list(text, key):
    match = re.search(r"^\s*;?\s*%s\s*=\s*(.*?)\s*$" % key, text, re.M)
    if not match or not match.group(1).strip().strip('"'):
        return []
    return [item.strip().strip('"') for item in match.group(1).strip().strip('"').split(";")]


def _normalized_palette_color(color):
    """Opaque 8-digit Bambu RGBA and 6-digit RGB specify the same physical color.

    Non-opaque alpha is not silently discarded. This affects QA comparison
    only; it does not mutate or rewrite a 3MF project.
    """
    color = str(color).strip().upper()
    if re.fullmatch(r"#[0-9A-F]{8}", color) and color.endswith("FF"):
        return color[:7]
    return color


def palette(pkg, family):
    """Ordered (colour, material) per filament slot, or None when the package has none."""
    if CODEC[family] == "bbl-json":
        cfg = pkg.json_config("Metadata/project_settings.config") or {}
        if "filament_colour" in cfg:
            colors, types = cfg["filament_colour"], cfg.get("filament_type", [])
        elif "Metadata/slice_info.config" in pkg.files:
            colors, types = _slice_slots(pkg)
        else:
            return None
    else:
        raw = pkg.files.get("Metadata/Slic3r_PE.config")
        if raw is None and "Metadata/SliceBridge_source_filaments.json" in pkg.files:
            # Panda2Prusa's generic project cannot install native Prusa filament
            # presets. It preserves the source identity in a separate metadata
            # document; compare that record, without calling it a native preset.
            data = pkg.json_config("Metadata/SliceBridge_source_filaments.json")
            slots = data.get("slots") if data is not None else None
            if data.get("schemaVersion") != 1 or not isinstance(slots, list):
                raise InputError("invalid SliceBridge source-filament metadata")
            if not slots:
                return None
            if len(slots) > 256:
                raise InputError("too many SliceBridge source-filament slots")
            ordered = []
            for slot in slots:
                if not isinstance(slot, dict) or not isinstance(slot.get("id"), int):
                    raise InputError("malformed SliceBridge source-filament slot")
                if not isinstance(slot.get("color"), str) or not isinstance(slot.get("material"), str):
                    raise InputError("missing SliceBridge source-filament colour or material")
                ordered.append((slot["id"], slot["color"], slot["material"]))
            ordered.sort(key=lambda row: row[0])
            if [row[0] for row in ordered] != list(range(1, len(ordered) + 1)):
                raise InputError("non-contiguous SliceBridge source-filament slot IDs")
            return [(colour.strip().upper(), material.strip().upper())
                    for _, colour, material in ordered]
        if raw is None:
            return None
        text = raw.decode("utf-8", "replace")
        colors, types = _ini_list(text, "filament_colour"), _ini_list(text, "filament_type")
    if not isinstance(colors, list) or not isinstance(types, list) or len(colors) != len(types):
        raise InputError("filament colour and material lists are missing or differ in length")
    if not colors:
        return None
    return [(_normalized_palette_color(c), str(t).strip().upper()) for c, t in zip(colors, types)]


def object_extruders(pkg):
    """Ordered (object id, extruder value) pairs from the first model config present."""
    for name in MODEL_CONFIGS:
        if name in pkg.files:
            pairs = []
            for obj in _iter(pkg.xml(name), "object"):
                values = [m.get("value") for m in obj
                          if _local(m.tag) == "metadata" and m.get("key") == "extruder"]
                pairs.append((obj.get("id"), values[-1] if values else None))
            return pairs
    return []


def mapped_object_tools(src_geo, out_geo, source_meta, output_meta):
    """Compare tools by placed build-item geometry, not XML metadata order/IDs.

    When per-build-item mapping cannot be proven, return 'skip' rather than
    claiming an unverified match. The geometry checks gate the full report.
    """
    src_scene, out_scene = src_geo["_scene"], out_geo["_scene"]
    src_map, out_map = dict(source_meta), dict(output_meta)
    src_build_ids = {oid for oid, _ in src_scene.items}
    mapped_ids = [oid for oid in src_build_ids if src_map.get(oid) is not None]
    if not mapped_ids:
        return "skip", "no source object extruder bound to a verified build item"

    def geometry_only(item):
        records = Counter()
        for (coords, _), count in item[1].items():
            records[coords] += count
        return records

    remaining = set(range(len(out_scene.items)))
    targets = [geometry_only(item) for item in out_scene.items]
    verified = 0
    for item in src_scene.items:
        oid = item[0]
        if src_map.get(oid) is None:
            continue
        source_triangles = geometry_only(item)
        matches = [i for i in sorted(remaining) if source_triangles == targets[i]]
        if not matches:
            return ("skip" if src_geo["hash"] != out_geo["hash"] else "fail",
                    "cannot align build object %s with output by placed geometry" % oid)
        best = next((i for i in matches if out_map.get(out_scene.items[i][0]) == src_map[oid]),
                    matches[0])
        remaining.remove(best)
        output_id = out_scene.items[best][0]
        actual = out_map.get(output_id)
        if src_map[oid] != actual:
            return "fail", ("source build object %s tool %s != aligned output %s tool %s"
                            % (oid, src_map[oid], output_id, actual))
        verified += 1
    unsupported = sorted(oid for oid, val in source_meta
                         if val is not None and oid not in src_build_ids)
    if unsupported:
        return "skip", ("%d build items matched, %d non-build metadata bindings unverified"
                        % (verified, len(unsupported)))
    return "pass", ("%d extruder-bound build items matched by world-space geometry"
                    % verified)


def _active_script_commands(value):
    """True only for non-comment slicer/firmware commands, not inert flags.

    An identical vendor placeholder comment is not proof that the target
    inherited the source printer's active firmware script.
    """
    values = value if isinstance(value, list) else [value]
    return any(isinstance(script, str) and any(
        line.strip() and not line.lstrip().startswith((";", "#"))
        for line in script.splitlines())
        for script in values)


def _scalar(value):
    try:
        return repr(float(value))
    except (TypeError, ValueError):
        return str(value).strip()


def compare(source, output, family="auto", expect_model=None,
            source_sha256=None, prior_output_sha256=None):
    src = Package(source)
    out = Package(output)

    src_family, src_basis = detect_family(src)
    if family != "auto":
        if src_family and CODEC[src_family] != CODEC[family]:
            raise InputError(f"--family {family} conflicts with source layout ({src_family})")
        src_family, src_basis = family, "declared"
    if src_family is None:
        raise InputError("source has no known 3MF layout; pass --family")
    try:
        out_family, out_basis = detect_family(out)
    except InputError:
        out_family, out_basis = None, "unreadable-config"
    if out_family is None:
        out_family, out_basis = src_family, "inferred-from-source"
    codec_match = CODEC[src_family] == CODEC[out_family]

    # Source defects make the reference itself invalid: report, do not guess.
    src_parts = model_parts(src)
    src_geo = geometry(src, src_parts)
    src_ext = object_extruders(src)
    src_pal = palette(src, src_family)
    src_cfg = src.json_config("Metadata/project_settings.config") if CODEC[src_family] == "bbl-json" else None
    src_paint = [p for _, p in src_geo["tris"] if p]
    # Real Bambu/Orca paint_color may contain a serialized sub-triangle
    # subdivision tree (e.g. BC, CC, 8C), NOT a hexadecimal slot number.
    # Only the small legacy single-state codes used in repo fixtures have
    # independently proven direct channel semantics here. For other codes,
    # raw geometry/paint equality remains checked in paint_state, but
    # paint-to-virtual-slot inference must be explicitly skipped.
    paint_channels_unverified = (CODEC[src_family] == "bbl-json" and any(
        str(code).upper() not in ("4", "8", "C", "1C", "2C") for code in src_paint))
    src_channels = used_channels(src_family, src_ext,
                                 [] if paint_channels_unverified else src_paint)

    checks = []

    def add(name, status, detail):
        checks.append({"name": name, "status": status, "detail": detail})

    def guarded(name, fn):
        try:
            return fn()
        except InputError as exc:
            add(name, "fail", str(exc))
            return None

    def zip_check(label, pkg):
        if pkg.problems:
            add(label, "fail", "; ".join(pkg.problems))
        else:
            add(label, "pass", f"{len(pkg.files)} entries readable, no duplicates or unsafe names")

    zip_check("source_zip_sanity", src)
    zip_check("output_zip_sanity", out)

    out_parts, out_geo = [], None
    try:
        out_parts = model_parts(out)
        add("output_components", "pass", f"content types, package relationships and model parts: {out_parts}")
    except InputError as exc:
        add("output_components", "fail", str(exc))
    if out_parts:
        out_geo = guarded("output_geometry_parse", lambda: geometry(out, out_parts))
        if out_geo is not None:
            add("output_geometry_parse", "pass", f"{out_geo['vertices']} vertices, {out_geo['triangles']} triangles")

    # An unused Default Extension="gcode" in [Content_Types].xml does
    # not mean there is a slice cache. Only a real part, an explicit part
    # Override or a relationship to G-code is evidence of a retained slice.
    gcode = [n for n in out.files if n.lower().endswith(GCODE_SUFFIXES)]
    for relfile in (n for n in out.files if n.endswith(".rels")):
        try:
            root=out.xml(relfile)
        except InputError as exc:
            add("no_sliced_gcode", "fail", str(exc))
            continue
        for rel in root:
            if (_local(rel.tag)=="Relationship" and
                re.search(r"\.gcode(?:\.md5)?(?:$|[?#])", rel.get("Target",""), re.I)):
                gcode.append(f"{relfile}: {rel.get('Target')}")
    if "[Content_Types].xml" in out.files:
        for child in out.xml("[Content_Types].xml"):
            if _local(child.tag)=="Override" and child.get("PartName","").lower().endswith(GCODE_SUFFIXES):
                gcode.append(f"content type override: {child.get('PartName')}")
    add("no_sliced_gcode", "fail" if gcode else "pass",
        f"output carries slice cache: {sorted(set(gcode))}" if gcode else
        "no sliced G-code entries, part overrides or relationships")

    if out_geo is None:
        add("geometry_counts", "fail", "output geometry unavailable")
        add("geometry_hash", "fail", "output geometry unavailable")
    else:
        counts_ok = (src_geo["vertices"], src_geo["triangles"]) == (out_geo["vertices"], out_geo["triangles"])
        add("geometry_counts", "pass" if counts_ok and out_geo["triangles"] else "fail",
            f"source {src_geo['vertices']}/{src_geo['triangles']} vertices/triangles, "
            f"output {out_geo['vertices']}/{out_geo['triangles']}")
        same_hash = src_geo["hash"] == out_geo["hash"]
        add("geometry_hash", "pass" if same_hash else "fail",
            f"canonical triangle hash {'matches' if same_hash else 'differs'} "
            f"(source {src_geo['hash'][:12]}, output {out_geo['hash'][:12]})")

    if out_geo is None:
        add("paint_state", "fail", "output geometry unavailable")
    elif not codec_match:
        add("paint_state", "skip", f"paint codecs differ ({CODEC[src_family]} vs {CODEC[out_family]}); raw codes not comparable")
    else:
        src_counts, out_counts = Counter(src_geo["tris"]), Counter(out_geo["tris"])
        lost = sum((src_counts - out_counts).values())
        added = sum((out_counts - src_counts).values())
        painted_src, painted_out = len(src_paint), sum(1 for _, p in out_geo["tris"] if p)
        detail = (f"painted triangles source {painted_src}, output {painted_out}; "
                  f"unmatched source-side {lost}, output-side {added}")
        add("paint_state", "pass" if not lost and not added else "fail", detail)

    out_pal = guarded("output_palette", lambda: palette(out, out_family))
    if src_pal is None:
        add("palette_slots", "skip", "source has no filament palette")
    elif out_pal is None:
        add("palette_slots", "fail", "output has no filament palette")
    else:
        mismatch = next((i for i, slot in enumerate(src_pal) if i >= len(out_pal) or out_pal[i] != slot), None)
        duplicates = sum(n - 1 for n in Counter(src_pal).values() if n > 1)
        if mismatch is None:
            add("palette_slots", "pass",
                f"{len(src_pal)} ordered slots preserved; {duplicates} duplicate colour/material "
                f"slots kept separate; extra output slots {len(out_pal) - len(src_pal)}")
        elif mismatch >= len(out_pal):
            add("palette_slots", "fail", f"output has {len(out_pal)} slots, source has {len(src_pal)}")
        else:
            add("palette_slots", "fail",
                f"slot {mismatch + 1} is {out_pal[mismatch]} in output, {src_pal[mismatch]} in source")

    if not src_channels:
        add("virtual_channels", "skip", "no object extruder or paint channels referenced")
    elif out_pal is None:
        add("virtual_channels", "fail", "output has no palette to hold channels")
    else:
        gaps = [c for c in sorted(src_channels) if c > len(out_pal)]
        unlike = [c for c in sorted(src_channels) if c <= len(out_pal) and src_pal and c <= len(src_pal)
                  and out_pal[c - 1] != src_pal[c - 1]]
        status = ("fail" if gaps or unlike else
                  "skip" if paint_channels_unverified else "pass")
        note = ("; complex Bambu paint codes cannot be safely decoded into "
                "virtual channels; raw codes are compared in paint_state"
                if paint_channels_unverified else "")
        add("virtual_channels", status,
            f"explicitly provable source channels {sorted(src_channels)}; "
            f"missing in output {gaps}; changed identity {unlike}{note}")

    out_ext = guarded("output_object_extruders", lambda: object_extruders(out))
    if not src_ext and not out_ext:
        add("object_extruder_mapping", "skip", "no object extruder metadata")
    elif out_ext is None:
        add("object_extruder_mapping", "fail", "output extruder metadata unreadable")
    elif out_geo is None:
        add("object_extruder_mapping", "skip", "output geometry cannot be aligned")
    else:
        state, detail = mapped_object_tools(src_geo, out_geo, src_ext, out_ext)
        add("object_extruder_mapping", state, detail)

    out_cfg = None
    if src_cfg is None:
        add("machine_scripts_not_carried", "skip", "JSON project layout not used by this family")
        add("process_intent", "skip", "JSON project layout not used by this family")
    elif out_family == "prusa":
        # Panda2Prusa emits a generic project without Bambu/Orca JSON settings.
        # JSON-process comparisons cannot be asserted against absent fields.
        add("machine_scripts_not_carried", "skip",
            "cross-family Prusa output has no JSON printer-script settings")
        add("process_intent", "skip",
            "cross-family Prusa output has no comparable JSON process settings")
    else:
        detail = "output project settings missing"
        try:
            out_cfg = out.json_config("Metadata/project_settings.config")
        except InputError as exc:
            out_cfg, detail = None, str(exc)
        if out_cfg is None:
            add("machine_scripts_not_carried", "fail", detail)
            add("process_intent", "fail", detail)
        else:
            carried = sorted(k for k, v in src_cfg.items()
                             if k.endswith("_gcode")
                             and k != "emit_machine_limits_to_gcode"
                             and _active_script_commands(v) and out_cfg.get(k) == v)
            add("machine_scripts_not_carried", "fail" if carried else "pass",
                f"source machine scripts copied into output: {carried}" if carried
                else "no source printer scripts copied; target profile owns them")
            keys = [k for k in PROCESS_KEYS if k in src_cfg]
            diffs = [k for k in keys if k not in out_cfg or _scalar(out_cfg[k]) != _scalar(src_cfg[k])]
            if not keys:
                add("process_intent", "skip", "source has no documented numeric process keys")
            else:
                add("process_intent", "fail" if diffs else "pass",
                    f"{len(keys)} numeric keys compared; changed or missing: {diffs}")

    model_name = out_cfg.get("printer_model") if out_cfg else None
    if expect_model:
        ok = model_name == expect_model
        add("printer_model", "pass" if ok else "fail", f"output printer_model {model_name!r}, expected {expect_model!r}")
    else:
        add("printer_model", "skip", f"not requested; output reports {model_name!r}")

    add_overwrite_check(checks, source, output, source_sha256, prior_output_sha256)

    facts = {
        "source_sha256": _file_sha256(source),
        "output_sha256": _file_sha256(output),
        "source_palette": src_pal,
        "output_palette": out_pal,
        "output_palette_basis": ("SliceBridge source-slot metadata only; not PrusaSlicer native presets"
                                if out_family == "prusa" and
                                "Metadata/SliceBridge_source_filaments.json" in out.files and
                                "Metadata/Slic3r_PE.config" not in out.files else "native or JSON profile"),
        "source_channels": sorted(src_channels),
        "source_paint_channels_unverified": paint_channels_unverified,
        "source_object_extruders": src_ext,
    }
    return {
        "ok": all(c["status"] != "fail" for c in checks),
        "family": {"source": src_family, "source_basis": src_basis,
                   "output": out_family, "output_basis": out_basis,
                   "paint_codec_comparable": codec_match},
        "checks": checks,
        "facts": facts,
        "notes": NOTES,
    }


def used_channels(family, extruders, paint_codes):
    channels = set()
    for _, value in extruders:
        if value is not None and str(value).strip().isdigit() and int(value) > 0:
            channels.add(int(value))
    if CODEC[family] == "bbl-json":
        for code in paint_codes:
            channel = bbl_channel(code)
            if channel:
                channels.add(channel)
    return channels


def _file_sha256(path):
    with open(path, "rb") as fh:
        return hashlib.sha256(fh.read()).hexdigest()


def add_overwrite_check(checks, source, output, source_sha256, prior_output_sha256):
    """Non-overwrite policy: distinct paths, source untouched, pre-existing output not replaced."""
    failures = []
    verified = []
    if os.path.realpath(source) == os.path.realpath(output) or os.path.samefile(source, output):
        failures.append("output path is the source file")
    if source_sha256:
        if _file_sha256(source) != source_sha256.lower():
            failures.append("source file changed during conversion")
        else:
            verified.append("source hash")
    if prior_output_sha256:
        if _file_sha256(output) != prior_output_sha256.lower():
            failures.append("pre-existing output file was overwritten")
        else:
            verified.append("pre-existing output hash")
    if failures:
        status, detail = "fail", "; ".join(failures)
    elif verified:
        status, detail = "pass", "verified: " + ", ".join(verified)
    else:
        status, detail = "skip", "distinct paths only; pass --source-sha256 / --prior-output-sha256 to verify"
    checks.append({"name": "non_overwrite", "status": status, "detail": detail})


def main(argv=None):
    parser = argparse.ArgumentParser(description="SliceBridge 3MF conversion fidelity QA (no slicing).")
    parser.add_argument("--source", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--family", choices=("auto",) + FAMILIES, default="auto")
    parser.add_argument("--expect-printer-model")
    parser.add_argument("--source-sha256")
    parser.add_argument("--prior-output-sha256")
    args = parser.parse_args(argv)
    try:
        report = compare(args.source, args.output, args.family, args.expect_printer_model,
                         args.source_sha256, args.prior_output_sha256)
    except InputError as exc:
        print(json.dumps({"ok": False, "error": str(exc)}, indent=2))
        return 2
    print(json.dumps(report, indent=2, sort_keys=True))
    return 0 if report["ok"] else 1


if __name__ == "__main__":
    sys.exit(main())
