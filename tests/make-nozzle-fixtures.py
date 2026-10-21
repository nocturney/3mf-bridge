"""Generate synthetic Bambu 3MF fixtures for nozzle-profile regression tests."""
from pathlib import Path
from zipfile import ZipFile, ZIP_DEFLATED
import json

ROOT = Path(__file__).resolve().parent / "fixtures"
SOURCE = ROOT / "bambu-one-color.3mf"
VARIANTS = {
    "0.2": "0.10",
    "0.4": "0.16",
    "0.6": "0.30",
    "0.8": "0.40",
}
with ZipFile(SOURCE, "r") as arc:
    base = {name: arc.read(name) for name in arc.namelist()}

for size, layer_height in VARIANTS.items():
    files = dict(base)
    settings = json.loads(files["Metadata/project_settings.config"])
    settings["nozzle_diameter"] = [size]
    settings["printer_settings_id"] = f"Bambu Lab A1 {size} nozzle"
    settings["layer_height"] = layer_height
    files["Metadata/project_settings.config"] = json.dumps(settings).encode("utf-8")
    output = ROOT / f"bambu-nozzle-{size}.3mf"
    with ZipFile(output, "w", compression=ZIP_DEFLATED) as archive:
        for name, payload in sorted(files.items()):
            archive.writestr(name, payload)
    print(output.name, output.stat().st_size)

# Invalid process-intent fixture verifies fallback to the target native layer
# range on Creator 5 0.25 mm instead of copying a too-thick 0.16 mm layer.
settings = json.loads(base["Metadata/project_settings.config"])
settings["nozzle_diameter"] = ["0.2"]
settings["printer_settings_id"] = "Bambu Lab A1 0.2 nozzle"
settings["layer_height"] = "0.16"
files = dict(base)
files["Metadata/project_settings.config"] = json.dumps(settings).encode("utf-8")
output = ROOT / "bambu-nozzle-0.2-overheight.3mf"
with ZipFile(output, "w", compression=ZIP_DEFLATED) as archive:
    for name, payload in sorted(files.items()):
        archive.writestr(name, payload)
print(output.name, output.stat().st_size)

# Additional multi-material fixtures test nozzle-specific filament temperatures
# on every target, rather than relying on PLA-only profile smoke tests.
MIXED = ROOT / "bambu-mixed-three.3mf"
with ZipFile(MIXED, "r") as arc:
    base_mixed = {name: arc.read(name) for name in arc.namelist()}
for size in ("0.2", "0.6", "0.8"):
    files = dict(base_mixed)
    cfg = json.loads(files["Metadata/project_settings.config"])
    cfg["nozzle_diameter"] = [size]
    cfg["printer_settings_id"] = f"Bambu Lab A1 {size} nozzle"
    cfg["layer_height"] = VARIANTS[size]
    files["Metadata/project_settings.config"] = json.dumps(cfg).encode("utf-8")
    output = ROOT / f"bambu-mixed-nozzle-{size}.3mf"
    with ZipFile(output, "w", compression=ZIP_DEFLATED) as archive:
        for name, payload in sorted(files.items()):
            archive.writestr(name, payload)
    print(output.name, output.stat().st_size)
