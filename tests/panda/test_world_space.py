"""Synthetic cross-part 3MF regressions. No user models or slicer presets."""
import json
import tempfile
import unittest
import zipfile
from pathlib import Path

import panda_qa as qa
from world_3mf import Scene

CORE = "http://schemas.microsoft.com/3dmanufacturing/core/2015/02"
PROD = "http://schemas.microsoft.com/3dmanufacturing/production/2015/06"
RNS = "http://schemas.openxmlformats.org/package/2006/relationships"
MREL = "http://schemas.microsoft.com/3dmanufacturing/2013/01/3dmodel"
COLORS = ["#FF0000", "#00FF00", "#00AAFF", "#FFFF00", "#FFFFFF"]


def archive(path, members):
    with zipfile.ZipFile(path, "w", zipfile.ZIP_DEFLATED) as z:
        for name, value in members.items():
            z.writestr(name, value if isinstance(value, bytes) else value.encode())
    return path


def members(flat=False, tool=5, paint="2C", path="/3D/Objects/shape.model",
            copied=False, duplicate=False):
    header = f"<model xmlns='{CORE}' xmlns:p='{PROD}' unit='millimeter'>"
    footer = "</model>"
    rels = f"<Relationships xmlns='{RNS}'><Relationship Id='main' Target='/3D/3dmodel.model' Type='{MREL}'/></Relationships>"
    content = ("<Types xmlns='http://schemas.openxmlformats.org/package/2006/content-types'>"
               "<Default Extension='model' ContentType='application/vnd.ms-package.3dmanufacturing-3dmodel+xml'/>"
               "</Types>")
    members = {"[Content_Types].xml": content, "_rels/.rels": rels}
    if not flat:
        # Two affine transforms: mesh -> component (x+10) -> build (y+5).
        items = ("<item objectid='5' transform='1 0 0 0 1 0 0 0 1 0 5 0'/>"
                 + ("<item objectid='5' transform='1 0 0 0 1 0 0 0 1 0 5 0'/>" if duplicate else ""))
        root = (header + "<resources><object id='5' type='model'><components>"
                + f"<component p:path='{path}' objectid='22' transform='1 0 0 0 1 0 0 0 1 10 0 0'/>"
                + "</components></object></resources><build>" + items + "</build>" + footer)
        members["3D/3dmodel.model"] = root
        members["3D/_rels/3dmodel.model.rels"] = (
            f"<Relationships xmlns='{RNS}'><Relationship Id='child' "
            + f"Target='{path}' Type='{MREL}'/></Relationships>")
        verts = [(0, 0, 0), (2, 0, 0), (0, 2, 0), (0, 0, 2)]
        obj_id = "22"
        members["Metadata/model_settings.config"] = (
            "<config><object id='5'><metadata key='extruder' value='5'/></object></config>")
        members["Metadata/project_settings.config"] = json.dumps({"filament_colour": COLORS})
    else:
        verts = [(10, 5, 0), (12, 5, 0), (10, 7, 0), (10, 5, 2)]
        obj_id = "81"
        members["Metadata/Slic3r_PE_model.config"] = (
            f"<config><object id='81'><metadata key='extruder' value='{tool}'/></object></config>")
        members["Metadata/SliceBridge_source_filaments.json"] = json.dumps({
            "schemaVersion": 1, "source": "bambu-orca",
            "slots": [{"id": i + 1, "color": col, "material": "PLA"}
                      for i, col in enumerate(COLORS)]})
        items = "<item objectid='81'/>"
    v = "".join(f"<vertex x='{x}' y='{y}' z='{z}'/>" for x, y, z in verts)
    t = (f"<triangle v1='0' v2='1' v3='2' paint_color='{paint}'/>"
         "<triangle v1='0' v2='1' v3='3'/>")
    mesh = (header + f"<resources><object id='{obj_id}' type='model'><mesh><vertices>{v}</vertices>"
            + f"<triangles>{t}</triangles></mesh></object></resources>"
            + ("<build>" + items + "</build>" if flat else "<build/>") + footer)
    if flat:
        members["3D/3dmodel.model"] = mesh
    else:
        members["3D/Objects/shape.model"] = mesh
    return members


class WorldSpaceGraphTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory(prefix="synthetic-world-3mf-")
        self.addCleanup(temp.cleanup)
        self.home = Path(temp.name)

    def pair(self, src=None, dest=None):
        a = qa.Package(archive(self.home / "src.3mf", src or members()))
        b = qa.Package(archive(self.home / "out.3mf", dest or members(flat=True)))
        return qa.compare(a, b)

    def test_flattened_renumbered_p_path_keeps_geometry_paint_and_tool(self):
        self.assertEqual(self.pair(), {})

    def test_relative_cross_part_path(self):
        src = members(path="Objects/shape.model")
        src["3D/_rels/3dmodel.model.rels"] = src["3D/_rels/3dmodel.model.rels"].replace(
            "Objects/shape.model", "/3D/Objects/shape.model")
        self.assertEqual(self.pair(src), {})

    def test_changed_colour_on_placed_triangle_fails(self):
        self.assertIn("paints", self.pair(dest=members(flat=True, paint="4")))

    def test_changed_tool_on_renumbered_object_fails(self):
        self.assertIn("extruders", self.pair(dest=members(flat=True, tool=2)))

    def test_missing_one_build_instance_fails(self):
        self.assertIn("geometry", self.pair(src=members(duplicate=True)))

    def test_cyclic_components_fail_closed(self):
        src = members()
        src["3D/3dmodel.model"] = src["3D/3dmodel.model"].replace(
            "objectid='22'", "objectid='5'").replace(
            "p:path='/3D/Objects/shape.model' ", "")
        self.assertIn("model", self.pair(src))

    def test_traversal_rejects_parent_directory_escape(self):
        src = members(path="../../private.model")
        self.assertIn("model", self.pair(src))

    def test_scene_sees_only_referenced_child_not_unused_mesh(self):
        src = members()
        src["3D/Objects/orphan.model"] = src["3D/Objects/shape.model"]
        src_path = archive(self.home / "referenced.3mf", src)
        scene = Scene(qa.Package(src_path).files)
        self.assertEqual(sum(scene.triangles.values()), 2)


if __name__ == "__main__":
    unittest.main()
