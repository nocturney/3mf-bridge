"""Create a wholly synthetic 3-color Bambu-like test project."""
from pathlib import Path
from zipfile import ZipFile, ZIP_DEFLATED
import json
import xml.etree.ElementTree as ET

folder = Path(__file__).resolve().parent / 'fixtures'
source = folder / 'bambu-one-color.3mf'
output = folder / 'bambu-mixed-three.3mf'
colors = ['#FF0000', '#00FF00', '#0000FF']
types = ['PLA', 'PETG', 'ABS']

with ZipFile(source, 'r') as archive:
    files = {name:archive.read(name) for name in archive.namelist()}
cfg = json.loads(files['Metadata/project_settings.config'])
cfg['filament_colour'] = colors
cfg['filament_type'] = types
files['Metadata/project_settings.config'] = json.dumps(cfg).encode()

data = files['Metadata/slice_info.config'].decode()
root = ET.fromstring(data)
plate = root.find('plate')
for child in list(plate):
    if child.tag == 'filament':
        plate.remove(child)
for i, (color, material) in enumerate(zip(colors, types), 1):
    ET.SubElement(plate, 'filament', {
        'id':str(i), 'type':material, 'color':color, 'used_m':'1', 'used_g':'3'
    })
files['Metadata/slice_info.config'] = ET.tostring(root, encoding='utf-8')

# Preserve original 3MF namespace and change only first three faces.
model = ET.fromstring(files['3D/3dmodel.model'])
triangles = [tag for tag in model.iter() if tag.tag.endswith('}triangle')]
for triangle, paint in zip(triangles, ('4','8','0C','4')):
    triangle.attrib['paint_color'] = paint
files['3D/3dmodel.model'] = ET.tostring(model, encoding='utf-8')

with ZipFile(output, 'w', ZIP_DEFLATED) as archive:
    for name, contents in sorted(files.items()):
        archive.writestr(name, contents)
print(output, output.stat().st_size)
