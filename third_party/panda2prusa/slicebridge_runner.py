#!/usr/bin/env python3
"""SliceBridge headless runner for the vendored Panda2Prusa converter.

Stable contract::

    slicebridge_runner.py --source /abs/in.3mf --output /abs/out.3mf --json [--plate N[,N]]

stdout is always exactly one JSON object. Success exits 0. Failure exits 1 with
``{"ok": false, "code": "STABLE_CODE", "error": "..."}``. Bad arguments exit 2 with
code ``BAD_ARGUMENTS``. Nothing is written to stderr.

This file is SliceBridge code. The package in ./panda2prusa is unmodified upstream
(see README-SLICEBRIDGE.md). The runner makes no network calls and never talks to a
printer. Its output is a PrusaSlicer project that still needs the user's own printer
profile; printer compatibility is never claimed. The output also carries
``Metadata/SliceBridge_source_filaments.json``, a backup record of the source filament
slots. It is not a PrusaSlicer filament preset.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import shutil
import sys
import tempfile
import time
import zipfile
import zlib
from collections import Counter

_HERE = os.path.dirname(os.path.abspath(__file__))
if _HERE not in sys.path:  # keeps `python -I` and PyInstaller imports working
    sys.path.insert(0, _HERE)

UPSTREAM_PROJECT = "CupsOhJoe/Panda2Prusa"
UPSTREAM_COMMIT = "95e4d82d22142295e52913a10b07e054458a8e4e"
ENGINE = "panda2prusa"

# Input bounds. Sized for large MakerWorld projects. Anything beyond them fails closed.
MAX_SOURCE_BYTES = 1 << 30  # compressed file size
MAX_ENTRIES = 10000
MAX_ENTRY_BYTES = 512 << 20  # uncompressed size of one entry
MAX_TOTAL_BYTES = 2 << 30  # uncompressed size of all entries
RATIO_CHECK_MIN_BYTES = 16 << 20
MAX_COMPRESSION_RATIO = 100

MODEL_PART = "3D/3dmodel.model"
CONFIG_PART = "Metadata/Slic3r_PE_model.config"
PRINTER_CONFIG_PART = "Metadata/Slic3r_PE.config"  # never written: no printer preset is claimed
CONTENT_TYPES_PART = "[Content_Types].xml"
OPC_NS = "http://schemas.openxmlformats.org/package/2006/content-types"
AUX_PART = "Metadata/SliceBridge_source_filaments.json"
AUX_CONTENT_TYPE = "application/json"
AUX_SCHEMA_VERSION = 1
AUX_SOURCE = "bambu-orca"
XML_SUFFIXES = (".model", ".config", ".rels", ".xml")
FORBIDDEN_XML_MARKERS = (b"<!DOCTYPE", b"<!ENTITY")
CORE_NS = "http://schemas.microsoft.com/3dmanufacturing/core/2015/02"
SLIC3RPE_NS = "http://schemas.slic3r.org/3mf/2017/06"
PAINT_ATTR = "{%s}mmu_segmentation" % SLIC3RPE_NS
ARCHIVE_ERRORS = (zipfile.BadZipFile, zlib.error, EOFError, NotImplementedError, OSError, RuntimeError, ValueError)

try:
    import lxml.etree as ET

    from panda2prusa import __version__ as ENGINE_VERSION
    from panda2prusa.convert import convert_file
    from panda2prusa.reader import ROOT_MODEL, read_bambu_3mf
    from panda2prusa.writer import _collect_objects, _flatten_leaves
except ImportError as _exc:  # lxml missing or vendored tree incomplete
    _ENGINE_ERROR = str(_exc)
else:
    _ENGINE_ERROR = None


class SliceBridgeError(Exception):
    def __init__(self, code, message):
        super().__init__(message)
        self.code = code


def _q(uri, tag):
    return "{%s}%s" % (uri, tag)


def _sha256(path):
    digest = hashlib.sha256()
    with open(path, "rb") as fh:
        for chunk in iter(lambda: fh.read(1 << 20), b""):
            digest.update(chunk)
    return digest.hexdigest()


def _remove_quiet(path):
    try:
        os.unlink(path)
    except OSError:
        pass


def _make_temp(out_dir):
    try:
        fd, path = tempfile.mkstemp(prefix=".slicebridge-", suffix=".partial", dir=out_dir)
    except OSError as exc:
        raise SliceBridgeError("OUTPUT_DIR_UNAVAILABLE", f"cannot create a temporary file in {out_dir}: {exc}") from exc
    os.close(fd)
    return path


# ---------------------------------------------------------------- input checks

def _check_paths(source, output):
    if not (os.path.isabs(source) and os.path.isabs(output)):
        raise SliceBridgeError("BAD_ARGUMENTS", "--source and --output must be absolute paths")
    if not (source.lower().endswith(".3mf") and output.lower().endswith(".3mf")):
        raise SliceBridgeError("BAD_ARGUMENTS", "--source and --output must end in .3mf")
    if os.path.realpath(source) == os.path.realpath(output):
        raise SliceBridgeError("SOURCE_OUTPUT_SAME_PATH", "output must not be the source file")
    if not os.path.isfile(source):
        raise SliceBridgeError("SOURCE_NOT_FOUND", f"source is not a regular file: {source}")
    if os.path.lexists(output):
        raise SliceBridgeError("OUTPUT_EXISTS", f"output already exists and will not be replaced: {output}")
    out_dir = os.path.dirname(output)
    if not (os.path.isdir(out_dir) and os.access(out_dir, os.W_OK)):
        raise SliceBridgeError("OUTPUT_DIR_UNAVAILABLE", f"output directory is missing or not writable: {out_dir}")


def _is_safe_entry_name(name):
    if not name or "\\" in name or "\x00" in name or name.startswith("/"):
        return False
    if re.match(r"^[A-Za-z]:", name):
        return False
    parts = name.split("/")
    if parts[-1] == "":  # directory entry
        parts = parts[:-1]
    return all(part not in ("", ".", "..") for part in parts)


def _check_archive(path):
    """Bounded structural checks on the source zip, before any parsing."""
    if os.path.getsize(path) > MAX_SOURCE_BYTES:
        raise SliceBridgeError("INPUT_TOO_LARGE", "source file exceeds the size limit")
    try:
        zf = zipfile.ZipFile(path)
    except ARCHIVE_ERRORS as exc:
        raise SliceBridgeError("INPUT_NOT_ZIP", f"source is not a readable 3MF container: {exc}") from exc
    try:
        with zf:
            infos = zf.infolist()
            if len(infos) > MAX_ENTRIES:
                raise SliceBridgeError("INPUT_TOO_MANY_ENTRIES", "source has too many archive entries")
            names = set()
            folded = set()
            total = 0
            for info in infos:
                name = info.filename
                if not _is_safe_entry_name(name):
                    raise SliceBridgeError("INPUT_UNSAFE_PATH", f"unsafe archive entry name: {name!r}")
                if name.casefold() in folded:
                    raise SliceBridgeError("INPUT_DUPLICATE_ENTRY", f"duplicate archive entry: {name!r}")
                folded.add(name.casefold())
                names.add(name)
                if info.flag_bits & 0x1:
                    raise SliceBridgeError("INPUT_ENCRYPTED", "encrypted archive entries are not supported")
                if info.compress_type not in (zipfile.ZIP_STORED, zipfile.ZIP_DEFLATED):
                    raise SliceBridgeError("INPUT_UNSUPPORTED_COMPRESSION", f"unsupported compression in {name!r}")
                if info.file_size > MAX_ENTRY_BYTES:
                    raise SliceBridgeError("INPUT_TOO_LARGE", f"archive entry too large: {name!r}")
                total += info.file_size
                if total > MAX_TOTAL_BYTES:
                    raise SliceBridgeError("INPUT_TOO_LARGE", "archive uncompressed size exceeds the limit")
                if info.file_size >= RATIO_CHECK_MIN_BYTES and (
                    info.compress_size == 0 or info.file_size / info.compress_size > MAX_COMPRESSION_RATIO
                ):
                    raise SliceBridgeError("INPUT_SUSPICIOUS_COMPRESSION", f"suspicious compression ratio: {name!r}")
            if MODEL_PART not in names:
                raise SliceBridgeError("INPUT_MISSING_MODEL", f"source has no {MODEL_PART}")
            # Upstream parses XML with entity resolution on. Refuse DTDs and entities outright.
            # Valid Bambu/Orca/PrusaSlicer files never contain them.
            for info in infos:
                if info.is_dir() or not info.filename.lower().endswith(XML_SUFFIXES):
                    continue
                data = zf.read(info.filename)  # zipfile verifies CRC here
                if any(marker in data for marker in FORBIDDEN_XML_MARKERS):
                    raise SliceBridgeError("INPUT_XML_DTD_FORBIDDEN", f"DTD or entity declaration in {info.filename!r}")
    except SliceBridgeError:
        raise
    except ARCHIVE_ERRORS as exc:
        raise SliceBridgeError("INPUT_CORRUPT", f"source archive is corrupt: {exc}") from exc


# ------------------------------------------------- expected vs produced geometry

def _part_for(parts, leaf_idx, leaf_oid, leaf_count):
    """Mirror of the upstream writer's part matching (by id, else by position)."""
    by_id = {p.id: p for p in parts}
    if leaf_oid in by_id:
        return by_id[leaf_oid]
    if len(parts) == leaf_count:
        return parts[leaf_idx]
    return None


def _expected(project, plates):
    """What the output must contain, computed from the source project."""
    objs, src_path = _collect_objects(project)
    leaves_of = {}
    summarized = set()
    geometry, painted_counts, extruders, items = [], [], [], []
    paints = Counter()
    for bi in project.build_items:
        plate = project.plate_of_object.get(bi.objectid)
        if plates is not None and plate is not None and plate not in plates:
            continue
        key = (ROOT_MODEL, bi.objectid)
        if key not in objs:
            continue
        if key not in leaves_of:
            leaves_of[key] = _flatten_leaves(key, objs, src_path)
        leaves = leaves_of[key]
        if not leaves:
            continue
        items.append((bi.transform if bi.transform.strip() else "", "1" if bi.printable else "0"))
        if key in summarized:
            continue
        summarized.add(key)
        so = project.settings_objects.get(bi.objectid)
        parts = so.parts if so else []
        tri_total = vert_total = painted = 0
        vol_exts = []
        for idx, (leaf_key, _xf) in enumerate(leaves):
            lmesh = objs[leaf_key].find(_q(CORE_NS, "mesh"))
            lverts = lmesh.find(_q(CORE_NS, "vertices"))
            ltris = lmesh.find(_q(CORE_NS, "triangles"))
            leaf_verts = len(lverts.findall(_q(CORE_NS, "vertex"))) if lverts is not None else 0
            vert_total += leaf_verts
            tri_els = ltris.findall(_q(CORE_NS, "triangle")) if ltris is not None else []
            tri_total += len(tri_els)
            for tri in tri_els:
                # Upstream offsets indices per mesh; an out-of-range index would silently
                # bind to the next mesh's vertices, so refuse it at the source.
                for axis in ("v1", "v2", "v3"):
                    if not 0 <= int(tri.get(axis)) < leaf_verts:
                        raise SliceBridgeError(
                            "INPUT_GEOMETRY_INVALID", f"triangle index out of range in object {leaf_key[1]}"
                        )
                if "paint_color" in tri.attrib:
                    painted += 1
                    paints[tri.attrib["paint_color"]] += 1
            if tri_els:  # upstream only emits a volume for leaves that have triangles
                part = _part_for(parts, idx, leaf_key[1], len(leaves))
                vol_exts.append((part.extruder if part else None) or "")
        geometry.append((tri_total, vert_total, len(vol_exts)))
        painted_counts.append(painted)
        extruders.append(((so.extruder if so else None) or "", tuple(sorted(vol_exts))))
    return {
        "geometry": sorted(geometry),
        "painted": sorted(painted_counts),
        "paints": paints,
        "extruders": sorted(extruders),
        "items": sorted(items),
    }


def _parse_xml(data):
    parser = ET.XMLParser(resolve_entities=False, no_network=True, huge_tree=True)
    return ET.fromstring(data, parser=parser)


def _meta_value(el, key):
    for md in el.findall("metadata"):
        if md.get("key") == key:
            return md.get("value")
    return None


def _looks_like_gcode(name):
    low = name.lower()
    return "gcode" in low or low.endswith((".gco", ".g"))


def _actual(model_bytes, config_bytes):
    """Summarize the produced model in the same shape as _expected()."""
    root = _parse_xml(model_bytes)
    resources = root.find(_q(CORE_NS, "resources"))
    build = root.find(_q(CORE_NS, "build"))
    if resources is None or build is None:
        raise SliceBridgeError("OUTPUT_INVALID", "output model lacks resources or build")

    config = {}
    if config_bytes is not None:
        cfg_root = _parse_xml(config_bytes)
        for obj in cfg_root.findall("object"):
            vols = []
            for vol in obj.findall("volume"):
                vols.append((int(vol.get("firstid")), int(vol.get("lastid")), _meta_value(vol, "extruder") or ""))
            config[obj.get("id")] = (_meta_value(obj, "extruder") or "", vols)

    geometry, painted_counts, extruders = [], [], []
    paints = Counter()
    object_ids = set()
    for obj in resources.findall(_q(CORE_NS, "object")):
        oid = obj.get("id")
        object_ids.add(oid)
        mesh = obj.find(_q(CORE_NS, "mesh"))
        if mesh is None:
            raise SliceBridgeError("OUTPUT_INVALID", f"output object {oid} has no mesh")
        verts = mesh.find(_q(CORE_NS, "vertices"))
        tris = mesh.find(_q(CORE_NS, "triangles"))
        vert_count = len(verts.findall(_q(CORE_NS, "vertex"))) if verts is not None else 0
        tri_els = tris.findall(_q(CORE_NS, "triangle")) if tris is not None else []
        painted = 0
        for tri in tri_els:
            for axis in ("v1", "v2", "v3"):
                if not 0 <= int(tri.get(axis)) < vert_count:
                    raise SliceBridgeError("GEOMETRY_MISMATCH", f"triangle index out of range in object {oid}")
            code = tri.get(PAINT_ATTR)
            if code is not None:
                painted += 1
                paints[code] += 1
        obj_ext, vols = config.get(oid, ("", []))
        cursor = 0  # volume ranges must tile the object's triangles exactly
        for first, last, _ext in sorted(vols):
            if first != cursor or last < first:
                raise SliceBridgeError("GEOMETRY_MISMATCH", f"volume ranges are not contiguous in object {oid}")
            cursor = last + 1
        if cursor != len(tri_els):
            raise SliceBridgeError("GEOMETRY_MISMATCH", f"volume ranges do not cover object {oid}")
        geometry.append((len(tri_els), vert_count, len(vols)))
        painted_counts.append(painted)
        extruders.append((obj_ext, tuple(sorted(ext for _f, _l, ext in vols))))

    if set(config) - object_ids:
        raise SliceBridgeError("OUTPUT_INVALID", "config references objects missing from the model")
    items = []
    for item in build.findall(_q(CORE_NS, "item")):
        if item.get("objectid") not in object_ids:
            raise SliceBridgeError("GEOMETRY_MISMATCH", "build item references a missing object")
        items.append((item.get("transform", ""), item.get("printable", "")))
    return {
        "geometry": sorted(geometry),
        "painted": sorted(painted_counts),
        "paints": paints,
        "extruders": sorted(extruders),
        "items": sorted(items),
    }


def _verify_output(path, expected):
    """Check the written file against the source. Raises SliceBridgeError on any mismatch."""
    try:
        with zipfile.ZipFile(path) as zf:
            if zf.testzip() is not None:
                raise SliceBridgeError("OUTPUT_INVALID", "output zip failed its CRC check")
            names = zf.namelist()
            if MODEL_PART not in names:
                raise SliceBridgeError("OUTPUT_INVALID", f"output has no {MODEL_PART}")
            if any(_looks_like_gcode(name) for name in names):
                raise SliceBridgeError("OUTPUT_CONTAINS_GCODE", "output unexpectedly contains G-code")
            if PRINTER_CONFIG_PART in names:
                raise SliceBridgeError("OUTPUT_INVALID", f"output unexpectedly contains {PRINTER_CONFIG_PART}")
            model_bytes = zf.read(MODEL_PART)
            config_bytes = zf.read(CONFIG_PART) if CONFIG_PART in names else None
            aux_bytes = zf.read(AUX_PART) if AUX_PART in names else None
            content_types = zf.read(CONTENT_TYPES_PART) if CONTENT_TYPES_PART in names else None
        actual = _actual(model_bytes, config_bytes)
    except SliceBridgeError:
        raise
    except (*ARCHIVE_ERRORS, ET.LxmlError, TypeError, AttributeError) as exc:
        raise SliceBridgeError("OUTPUT_INVALID", f"output could not be verified: {exc}") from exc

    if expected["geometry"] != actual["geometry"] or expected["items"] != actual["items"]:
        raise SliceBridgeError(
            "GEOMETRY_MISMATCH", "output objects, volumes, triangles, vertices or build items differ from the source"
        )
    if expected["painted"] != actual["painted"] or expected["paints"] != actual["paints"]:
        raise SliceBridgeError("PAINT_MISMATCH", "per-triangle paint data differs from the source")
    if expected["extruders"] != actual["extruders"]:
        raise SliceBridgeError("EXTRUDER_MISMATCH", "extruder assignments differ from the source")
    _verify_source_filaments(aux_bytes, content_types, expected["filament_slots"])


def _verify_source_filaments(aux_bytes, content_types, slots):
    """The backup slot record must match the source exactly and be registered in OPC."""
    if aux_bytes is None or content_types is None:
        raise SliceBridgeError("OUTPUT_INVALID", f"output has no {AUX_PART}")
    try:
        aux_doc = json.loads(aux_bytes.decode("utf-8"))
    except (UnicodeDecodeError, ValueError) as exc:
        raise SliceBridgeError("METADATA_MISMATCH", f"{AUX_PART} is not valid UTF-8 JSON: {exc}") from exc
    if aux_doc != _aux_document(slots):
        raise SliceBridgeError("METADATA_MISMATCH", f"{AUX_PART} does not match the source filament slots")
    try:
        root = _parse_xml(content_types)
    except ET.LxmlError as exc:
        raise SliceBridgeError("OUTPUT_INVALID", f"[Content_Types].xml is not valid XML: {exc}") from exc
    overrides = [
        el
        for el in root.iter(_q(OPC_NS, "Override"))
        if el.get("PartName") == "/" + AUX_PART and el.get("ContentType") == AUX_CONTENT_TYPE
    ]
    if not overrides:
        raise SliceBridgeError("OUTPUT_INVALID", f"[Content_Types].xml has no override for {AUX_PART}")


# ------------------------------------------------- source filament slot record

PROFILE_WARNING = (
    "Printer profile compatibility was not verified. Open the output in PrusaSlicer "
    "and select your own Prusa printer profile before slicing."
)
RECORD_WARNING = (
    f"{AUX_PART} is a backup record of the source filament slots. It is not a PrusaSlicer "
    "filament preset, and PrusaSlicer does not read it. Set each filament in your own printer profile."
)
_HEX_COLOR = re.compile(r"#([0-9A-Fa-f]{6})")


def _source_filament_slots(project):
    """Every source filament slot in source order. Duplicates and unused slots are kept.

    Ids are the 1-based positions in the source's filament list. A colour that is not
    ``#RRGGBB`` is recorded as null rather than guessed.
    """
    colors = list(project.filament_colors or [])
    types = list(project.filament_types or [])
    slots = []
    for idx in range(max(len(colors), len(types))):
        match = _HEX_COLOR.fullmatch(colors[idx].strip()) if idx < len(colors) else None
        material = types[idx].strip() if idx < len(types) else ""
        slots.append(
            {
                "id": idx + 1,
                "color": "#" + match.group(1).upper() if match else None,
                "material": material or None,
            }
        )
    return slots


def _aux_document(slots):
    return {"schemaVersion": AUX_SCHEMA_VERSION, "source": AUX_SOURCE, "slots": slots}


def _filament_warnings(slots):
    if not slots:
        return ["The source project lists no filament slots, so the backup record is empty."]
    unreadable = [str(slot["id"]) for slot in slots if slot["color"] is None]
    if unreadable:
        return [f"Source filament slot(s) {', '.join(unreadable)} have no #RRGGBB colour and are recorded as null."]
    return []


def _with_aux_override(content_types):
    try:
        root = _parse_xml(content_types)
    except ET.LxmlError as exc:
        raise SliceBridgeError("OUTPUT_INVALID", f"[Content_Types].xml is not valid XML: {exc}") from exc
    override = ET.SubElement(root, _q(OPC_NS, "Override"))
    override.set("PartName", "/" + AUX_PART)
    override.set("ContentType", AUX_CONTENT_TYPE)
    return ET.tostring(root, xml_declaration=True, encoding="UTF-8")


def _write_with_source_filaments(src_path, dst_path, slots):
    """Repack ``src_path`` into the new file ``dst_path`` and append the slot record.

    Entries are streamed in bounded chunks, never extracted. Each keeps its name,
    compression method, timestamp and bytes. Only [Content_Types].xml changes, gaining
    one Override for the record.
    """
    with zipfile.ZipFile(src_path) as zin, zipfile.ZipFile(dst_path, "w") as zout:
        names = zin.namelist()
        if CONTENT_TYPES_PART not in names or AUX_PART in names:
            raise SliceBridgeError("OUTPUT_INVALID", "converted package has an unexpected layout")
        for info in zin.infolist():
            zi = zipfile.ZipInfo(info.filename, date_time=info.date_time)
            zi.compress_type = info.compress_type
            zi.external_attr = info.external_attr
            if info.filename == CONTENT_TYPES_PART:
                zout.writestr(zi, _with_aux_override(zin.read(info)))
                continue
            large = info.file_size > zipfile.ZIP64_LIMIT
            with zin.open(info) as src, zout.open(zi, "w", force_zip64=large) as dst:
                shutil.copyfileobj(src, dst, 1 << 20)
        aux = zipfile.ZipInfo(AUX_PART, date_time=time.localtime()[:6])
        aux.compress_type = zipfile.ZIP_DEFLATED
        zout.writestr(aux, json.dumps(_aux_document(slots), separators=(",", ":")).encode("utf-8"))


# --------------------------------------------------------------- conversion

def run_conversion(source, output, plates=None):
    """Convert ``source`` to ``output`` and return the success payload.

    Raises SliceBridgeError with a stable code on refusal or failure. The output is
    published only after it verifies, via a hard link from a temporary file in the
    same directory, so a file is never seen half-written and never replaced.
    """
    _check_paths(source, output)
    source_sha = _sha256(source)
    _check_archive(source)
    if _ENGINE_ERROR is not None:
        raise SliceBridgeError("ENGINE_UNAVAILABLE", f"vendored panda2prusa cannot be imported: {_ENGINE_ERROR}")
    try:
        project = read_bambu_3mf(source)
        expected = _expected(project, plates)
        expected["filament_slots"] = _source_filament_slots(project)
    except SliceBridgeError:
        raise
    except Exception as exc:
        raise SliceBridgeError("INPUT_PARSE_FAILED", f"could not read the 3MF project: {exc}") from exc
    if not expected["geometry"]:
        raise SliceBridgeError("NO_GEOMETRY", "no build items with mesh geometry were selected")

    out_dir = os.path.dirname(output)
    partial = repacked = None
    try:
        partial = _make_temp(out_dir)
        repacked = _make_temp(out_dir)
        try:
            # Keep source filament slots exactly as written: no compaction, no remap.
            convert_file(source, partial, plates=plates, extruder_map={}, compact_extruders=False)
        except Exception as exc:
            raise SliceBridgeError("CONVERSION_FAILED", f"upstream panda2prusa failed: {exc}") from exc
        try:
            _write_with_source_filaments(partial, repacked, expected["filament_slots"])
        except SliceBridgeError:
            raise
        except (*ARCHIVE_ERRORS, ET.LxmlError) as exc:
            raise SliceBridgeError("CONVERSION_FAILED", f"could not add the source filament record: {exc}") from exc
        _verify_output(repacked, expected)
        if _sha256(source) != source_sha:
            raise SliceBridgeError("SOURCE_MODIFIED", "source changed during conversion; nothing was published")
        output_sha = _sha256(repacked)
        output_bytes = os.path.getsize(repacked)
        os.chmod(repacked, 0o644)
        try:
            os.link(repacked, output)
        except FileExistsError as exc:
            raise SliceBridgeError("OUTPUT_EXISTS", f"output appeared during conversion: {output}") from exc
        except OSError as exc:
            raise SliceBridgeError("PUBLISH_FAILED", f"could not publish output (hard links required): {exc}") from exc
    finally:
        for path in (partial, repacked):
            if path is not None:
                _remove_quiet(path)

    geometry = expected["geometry"]
    slots = expected["filament_slots"]
    return {
        "ok": True,
        "engine": ENGINE,
        "engine_version": ENGINE_VERSION,
        "upstream": {"project": UPSTREAM_PROJECT, "commit": UPSTREAM_COMMIT, "license": "MIT"},
        "source": source,
        "output": output,
        "source_sha256": source_sha,
        "output_sha256": output_sha,
        "output_bytes": output_bytes,
        "producer": project.producer,
        "plates_available": project.plate_ids,
        "plates_selected": sorted(plates) if plates else None,
        "stats": {
            "objects": len(geometry),
            "volumes": sum(g[2] for g in geometry),
            "build_items": len(expected["items"]),
            "triangles": sum(g[0] for g in geometry),
            "vertices": sum(g[1] for g in geometry),
            "painted_triangles": sum(expected["painted"]),
        },
        "extruder_remap": None,
        "slot_count": len(slots),
        "verified": {
            "source_unchanged": True,
            "archive_valid": True,
            "gcode_free": True,
            "geometry_preserved": True,
            "paint_preserved": True,
            "extruders_preserved": True,
            "filament_slots_preserved": True,
        },
        "printer_profile": {"verified": False},
        "warnings": [PROFILE_WARNING, RECORD_WARNING, *_filament_warnings(slots)],
    }


# ---------------------------------------------------------------------- CLI

class _JsonArgumentParser(argparse.ArgumentParser):
    def error(self, message):
        _emit({"ok": False, "code": "BAD_ARGUMENTS", "error": message})
        sys.exit(2)


def _emit(payload):
    sys.stdout.write(json.dumps(payload, sort_keys=True) + "\n")
    sys.stdout.flush()


def _parse_plates(chunks):
    if not chunks:
        return None
    plates = []
    for chunk in chunks:
        for text in chunk.replace(",", " ").split():
            try:
                plate = int(text)
            except ValueError:
                raise SliceBridgeError("BAD_ARGUMENTS", f"plate must be a number: {text!r}")
            if plate < 1:
                raise SliceBridgeError("BAD_ARGUMENTS", f"plate numbers start at 1: {plate}")
            plates.append(plate)
    return plates or None


def main(argv=None):
    parser = _JsonArgumentParser(prog="slicebridge_runner.py", add_help=False, allow_abbrev=False)
    parser.add_argument("--source", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--json", action="store_true", required=True)
    parser.add_argument("--plate", action="append")
    args = parser.parse_args(argv)
    try:
        payload = run_conversion(args.source, args.output, _parse_plates(args.plate))
    except SliceBridgeError as exc:
        _emit({"ok": False, "code": exc.code, "error": str(exc)})
        return 1
    except Exception as exc:  # boundary: never print a traceback on stdout
        _emit({"ok": False, "code": "INTERNAL_ERROR", "error": f"unexpected failure: {exc}"})
        return 1
    _emit(payload)
    return 0


if __name__ == "__main__":
    sys.exit(main())
