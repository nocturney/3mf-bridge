"""Deterministic synthetic five-color Bambu project for virtual-slot QA.

Uses the project-owned one-color fixture as a harmless skeleton; never uses a
customer model or printer-generated G-code. Its stale cache is intentional.
"""
from pathlib import Path
from zipfile import ZipFile, ZipInfo, ZIP_DEFLATED
import json

base = Path(__file__).parent / "fixtures"
source = base / "bambu-one-color.3mf"
destination = base / "bambu-five-colors.3mf"
if not source.is_file():
    raise SystemExit("Source fixture missing")
if destination.exists():
    raise SystemExit("Refusing to overwrite existing five-slot fixture")

colors = ["#FF0000", "#00FF00", "#0000FF", "#FFFF00", "#FFFFFF"]
with ZipFile(source) as archive:
    entries = {name: archive.read(name) for name in archive.namelist()}

cfg = json.loads(entries["Metadata/project_settings.config"])
cfg["filament_colour"] = colors
cfg["filament_type"] = ["PLA"] * len(colors)
entries["Metadata/project_settings.config"] = json.dumps(
    cfg, separators=(",", ":"), sort_keys=True
).encode()

entries["Metadata/slice_info.config"] = (
    "<config><plate><metadata key='printer_model_id' value='C13'/>"
    + "".join(
        f"<filament id='{i}' type='PLA' color='{c}' used_m='1' used_g='3'/>"
        for i, c in enumerate(colors, 1)
    )
    + "</plate></config>"
).encode()
entries["Metadata/model_settings.config"] = (
    b"<config><object id='1'><metadata key='extruder' value='5'/></object></config>"
)
model = entries["3D/3dmodel.model"]
if model.count(b"paint_color='4'") < 1:
    raise SystemExit("Source fixture paint encoding unexpected")
# Prusa/Bambu/Orca paint encoding "2C" means leaf extruder/filament 5.
entries["3D/3dmodel.model"] = model.replace(
    b"paint_color='4'", b"paint_color='2C'", 1
)

with ZipFile(destination, "w") as archive:
    for name, content in sorted(entries.items()):
        info = ZipInfo(name, date_time=(2020, 1, 1, 0, 0, 0))
        info.compress_type = ZIP_DEFLATED
        info.external_attr = 0o644 << 16
        archive.writestr(info, content)

print(f"Created {destination} with {len(colors)} independent synthetic slots")
