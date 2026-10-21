"""Synthetic Bambu-like 3MF for automated bridge tests (not a printer validation corpus)."""
from pathlib import Path
from zipfile import ZipFile, ZIP_DEFLATED
import json

out = Path(__file__).resolve().parent / "fixtures" / "bambu-one-color.3mf"
out.parent.mkdir(parents=True, exist_ok=True)

model = """<?xml version='1.0' encoding='UTF-8'?>
<model xmlns='http://schemas.microsoft.com/3dmanufacturing/core/2015/02' unit='millimeter'>
  <resources><object id='1' type='model'><mesh>
    <vertices><vertex x='0' y='0' z='0'/><vertex x='10' y='0' z='0'/>
      <vertex x='0' y='10' z='0'/><vertex x='0' y='0' z='10'/></vertices>
    <triangles><triangle v1='0' v2='1' v3='2' paint_color='4'/>
      <triangle v1='0' v2='1' v3='3' paint_color='4'/>
      <triangle v1='0' v2='2' v3='3' paint_color='4'/>
      <triangle v1='1' v2='2' v3='3' paint_color='4'/></triangles>
  </mesh></object></resources>
  <build><item objectid='1'/></build>
</model>"""

settings = {
    "printer_model": "Bambu Lab A1",
    "filament_colour": ["#FF0000"],
    "filament_type": ["PLA"],
    "enable_support": "0",
    "layer_height": "0.16",
    "wall_loops": "3",
    "brim_width": "5",
    "machine_start_gcode": "M9999 UNSAFE_BAMBU_SCRIPT",
}
entries = {
    "[Content_Types].xml": """<Types xmlns='http://schemas.openxmlformats.org/package/2006/content-types'>
        <Override PartName='/3D/3dmodel.model' ContentType='application/vnd.ms-package.3dmanufacturing-3dmodel+xml'/>
        <Override PartName='/Metadata/plate_1.gcode' ContentType='text/plain'/>
        </Types>""",
    "_rels/.rels": """<Relationships xmlns='http://schemas.openxmlformats.org/package/2006/relationships'>
        <Relationship Id='model' Target='/3D/3dmodel.model' Type='http://schemas.microsoft.com/3dmanufacturing/2013/01/3dmodel'/>
        <Relationship Id='old' Target='/Metadata/plate_1.gcode' Type='slice'/>
        </Relationships>""",
    "3D/3dmodel.model": model,
    "Metadata/project_settings.config": json.dumps(settings),
    "Metadata/model_settings.config": "<config><object id='1'><metadata key='extruder' value='1'/></object></config>",
    "Metadata/slice_info.config": """<config><plate>
        <metadata key='printer_model_id' value='C13'/>
        <filament id='1' type='PLA' color='#FF0000' used_m='1' used_g='3'/>
        </plate></config>""",
    "Metadata/plate_1.gcode": "M104 S300 ; stale printer-specific slice",
    "Metadata/plate_1.gcode.md5": "synthetic",
    "Metadata/plate_1.json": "{}",
}
with ZipFile(out, "w", ZIP_DEFLATED) as archive:
    for name, text in sorted(entries.items()):
        archive.writestr(name, text.encode("utf-8"))
print(out, out.stat().st_size)
