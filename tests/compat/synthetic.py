"""Deterministic, tiny synthetic 3MF builders for the conversion QA tests.

Nothing here is a printer or slicer profile. Machine scripts and G-code are
obvious placeholders, and every archive has a fixed timestamp so output bytes
are reproducible. Prusa-style markers are stand-ins written from memory, not
copied from a real PrusaSlicer export.
"""
import json
import re
import warnings
import xml.etree.ElementTree as ET
import zipfile

MODEL_NS = "http://schemas.microsoft.com/3dmanufacturing/core/2015/02"
MODEL_REL = "http://schemas.microsoft.com/3dmanufacturing/2013/01/3dmodel"
TETRA = [(0, 0, 0), (10, 0, 0), (0, 10, 0), (0, 0, 10)]
TRIS = [(0, 1, 2), (0, 1, 3), (0, 2, 3), (1, 2, 3)]
SCRIPT = "M9999 UNSAFE_SOURCE_SCRIPT"

CONTENT_TYPES = (
    "<Types xmlns='http://schemas.openxmlformats.org/package/2006/content-types'>"
    "<Override PartName='/3D/3dmodel.model' "
    "ContentType='application/vnd.ms-package.3dmanufacturing-3dmodel+xml'/></Types>"
)
RELS = (
    "<Relationships xmlns='http://schemas.openxmlformats.org/package/2006/relationships'>"
    f"<Relationship Id='model' Target='/3D/3dmodel.model' Type='{MODEL_REL}'/></Relationships>"
)
GCODE_NOTE = "M104 S300 ; stale printer-specific slice"


def model_xml(vertices=TETRA, triangles=TRIS, paints=None, attr="paint_color"):
    paints = paints or [None] * len(triangles)
    verts = "".join(f"<vertex x='{x}' y='{y}' z='{z}'/>" for x, y, z in vertices)
    tris = ""
    for (a, b, c), paint in zip(triangles, paints):
        extra = f" {attr}='{paint}'" if paint else ""
        tris += f"<triangle v1='{a}' v2='{b}' v3='{c}'{extra}/>"
    return (
        "<?xml version='1.0' encoding='UTF-8'?>"
        f"<model xmlns='{MODEL_NS}' unit='millimeter'>"
        "<resources><object id='1' type='model'><mesh>"
        f"<vertices>{verts}</vertices><triangles>{tris}</triangles>"
        "</mesh></object></resources>"
        "<build><item objectid='1'/></build></model>"
    ).encode()


def bbl_package(colors, types, extruder=1, paints=None, vertices=TETRA, triangles=TRIS,
                printer_model="Bambu Lab A1", scripts=True, gcode=True, process=None):
    """A Bambu/Orca-style project. ``gcode=True`` adds the stale slice cache a source may carry."""
    cfg = {
        "printer_model": printer_model,
        "filament_colour": list(colors),
        "filament_type": list(types),
        "enable_support": "0",
        "layer_height": "0.16",
        "wall_loops": "3",
        "brim_width": "5",
        "machine_start_gcode": SCRIPT if scripts else "",
    }
    cfg.update(process or {})
    slices = "".join(
        f"<filament id='{i}' type='{t}' color='{c}' used_m='1' used_g='3'/>"
        for i, (c, t) in enumerate(zip(colors, types), 1)
    )
    entries = {
        "[Content_Types].xml": CONTENT_TYPES,
        "_rels/.rels": RELS,
        "3D/3dmodel.model": model_xml(vertices, triangles, paints),
        "Metadata/model_settings.config":
            f"<config><object id='1'><metadata key='extruder' value='{extruder}'/></object></config>",
        "Metadata/project_settings.config": json.dumps(cfg, sort_keys=True),
        "Metadata/slice_info.config":
            f"<config><plate><metadata key='printer_model_id' value='C13'/>{slices}</plate></config>",
        "Metadata/plate_1.json": "{}",
    }
    if gcode:
        entries["Metadata/plate_1.gcode"] = GCODE_NOTE
        entries["Metadata/plate_1.gcode.md5"] = "0" * 32
        entries["_rels/.rels"] = RELS.replace(
            "</Relationships>",
            "<Relationship Id='old' Target='/Metadata/plate_1.gcode' Type='slice'/></Relationships>")
        entries["[Content_Types].xml"] = CONTENT_TYPES.replace(
            "</Types>", "<Override PartName='/Metadata/plate_1.gcode' ContentType='text/plain'/></Types>")
    return entries


def prusa_package(colors, types, extruder=1, paints=None, vertices=TETRA, triangles=TRIS):
    """A PrusaSlicer-style stand-in: Slic3r_PE*.config names and a namespaced mmu attribute.

    UNVERIFIED against a real PrusaSlicer export; the QA tests only exercise harness logic.
    """
    ini = (
        f"; filament_colour = {';'.join(colors)}\n"
        f"; filament_type = {';'.join(types)}\n"
        "; layer_height = 0.2\n"
    )
    model_cfg = (
        "<config><object id='1' instances_count='1'>"
        f"<metadata type='object' key='extruder' value='{extruder}'/>"
        "</object></config>"
    )
    model = model_xml(vertices, triangles, paints, attr="slic3rpe:mmu_segmentation")
    model = model.replace(b"<model ", b"<model xmlns:slic3rpe='urn:synthetic:slic3rpe' ", 1)
    return {
        "[Content_Types].xml": CONTENT_TYPES,
        "_rels/.rels": RELS,
        "3D/3dmodel.model": model,
        "Metadata/Slic3r_PE.config": ini,
        "Metadata/Slic3r_PE_model.config": model_cfg,
    }


def write_zip(path, entries):
    """Write ``entries`` (dict or list of (name, data) pairs) with a fixed timestamp."""
    items = list(entries.items()) if isinstance(entries, dict) else list(entries)
    with zipfile.ZipFile(path, "w", zipfile.ZIP_DEFLATED) as archive, warnings.catch_warnings():
        # Duplicate names are written on purpose by the zip-sanity mutation tests.
        warnings.simplefilter("ignore", UserWarning)
        for name, data in items:
            info = zipfile.ZipInfo(name, date_time=(2020, 1, 1, 0, 0, 0))
            info.compress_type = zipfile.ZIP_DEFLATED
            archive.writestr(info, data)
    return path


def read_zip(path):
    with zipfile.ZipFile(path) as archive:
        return {name: archive.read(name) for name in archive.namelist()}


def _txt(data):
    return data.decode("utf-8") if isinstance(data, bytes) else data


def set_bbl_palette(entries, colors, types):
    """Rewrite the palette the way an engine output would carry it (project + slice info)."""
    entries = dict(entries)
    cfg = json.loads(_txt(entries["Metadata/project_settings.config"]))
    cfg["filament_colour"], cfg["filament_type"] = list(colors), list(types)
    entries["Metadata/project_settings.config"] = json.dumps(cfg, sort_keys=True)
    slices = "".join(
        f"<filament id='{i}' type='{t}' color='{c}' used_m='1' used_g='3'/>"
        for i, (c, t) in enumerate(zip(colors, types), 1)
    )
    entries["Metadata/slice_info.config"] = (
        f"<config><plate><metadata key='printer_model_id' value='C13'/>{slices}</plate></config>")
    return entries


def set_prusa_palette(entries, colors, types):
    entries = dict(entries)
    text = _txt(entries["Metadata/Slic3r_PE.config"])
    text = re.sub(r"(?m)^(; filament_colour = ).*$", lambda m: m.group(1) + ";".join(colors), text)
    text = re.sub(r"(?m)^(; filament_type = ).*$", lambda m: m.group(1) + ";".join(types), text)
    entries["Metadata/Slic3r_PE.config"] = text
    return entries


def edit_model(entries, edit):
    """Apply ``edit(root)`` to the model XML and re-serialise it."""
    entries = dict(entries)
    root = ET.fromstring(entries["3D/3dmodel.model"])
    edit(root)
    entries["3D/3dmodel.model"] = ET.tostring(root, encoding="utf-8")
    return entries


def set_triangle_paint(entries, index, code):
    def edit(root):
        tris = [el for el in root.iter() if el.tag.rsplit("}", 1)[-1] == "triangle"]
        for key in [k for k in tris[index].attrib if k.rsplit("}", 1)[-1] in ("paint_color", "mmu_segmentation")]:
            del tris[index].attrib[key]
        tris[index].set("paint_color", code)
    return edit_model(entries, edit)


def move_vertex(entries, index, delta):
    def edit(root):
        verts = [el for el in root.iter() if el.tag.rsplit("}", 1)[-1] == "vertex"]
        verts[index].set("x", str(float(verts[index].get("x")) + delta))
    return edit_model(entries, edit)


def drop_triangle(entries, index):
    def edit(root):
        tris = [el for el in root.iter() if el.tag.rsplit("}", 1)[-1] == "triangle"]
        for parent in root.iter():
            if tris[index] in list(parent):
                parent.remove(tris[index])
                return
    return edit_model(entries, edit)


def set_extruder(entries, value):
    entries = dict(entries)
    for name in ("Metadata/model_settings.config", "Metadata/Slic3r_PE_model.config"):
        if name in entries:
            entries[name] = re.sub(r"(key=['\"]extruder['\"] value=['\"])[^'\"]*", r"\g<1>%s" % value,
                                   _txt(entries[name]))
    return entries


def set_process(entries, key, value):
    entries = dict(entries)
    cfg = json.loads(_txt(entries["Metadata/project_settings.config"]))
    cfg[key] = value
    entries["Metadata/project_settings.config"] = json.dumps(cfg, sort_keys=True)
    return entries


def add_gcode(entries):
    entries = dict(entries)
    entries["Metadata/plate_1.gcode"] = GCODE_NOTE
    return entries


def sanitized(entries):
    """What a clean conversion looks like: no slice cache, no copied machine scripts.

    Used to simulate engine output for fixtures, which must never be edited.
    """
    kept = {}
    for name, data in entries.items():
        if re.search(r"plate_[0-9]+\.gcode", name, re.I):
            continue
        text = _txt(data) if name.endswith((".config", ".rels", ".xml")) else None
        if text is not None and name in ("_rels/.rels", "[Content_Types].xml"):
            text = re.sub(r"<(?:Override|Relationship)\b[^>]*plate_[0-9]+\.gcode[^>]*/>", "", text)
            data = text.encode("utf-8")
        kept[name] = data
    if "Metadata/project_settings.config" in kept:
        cfg = json.loads(_txt(kept["Metadata/project_settings.config"]))
        for key in list(cfg):
            if key.endswith("_gcode"):
                cfg[key] = ""
        kept["Metadata/project_settings.config"] = json.dumps(cfg, sort_keys=True).encode("utf-8")
    return kept
