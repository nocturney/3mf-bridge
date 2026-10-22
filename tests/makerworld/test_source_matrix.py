"""Deterministic checks of the MakerWorld/Bambu source fixtures (stdlib unittest, no engine).

These tests read the repo fixtures only and never write to them. They do not run a conversion:
conversion evidence comes from smoke_conversion.py, and only where an engine is installed.

Run: python3 -m unittest discover -s tests/makerworld -p 'test_*.py' -v
"""
import json
import re
import sys
import unittest
import zipfile
from pathlib import Path
import xml.etree.ElementTree as ET

sys.path.insert(0, str(Path(__file__).resolve().parent))
import matrix  # noqa: E402

PAINT_CHANNEL = {"4": 1, "8": 2, "0C": 3, "1C": 4, "2C": 5}  # inferred scheme, see docs/qa matrix
STALE_GCODE = ["Metadata/plate_1.gcode", "Metadata/plate_1.gcode.md5"]


class FixtureMatrix(unittest.TestCase):

    def test_matrix_covers_every_fixture(self):
        # Adding or removing a fixture must update EXPECTED and the QA matrix doc together.
        self.assertEqual(sorted(p.stem for p in matrix.fixture_paths()), sorted(matrix.EXPECTED))

    def test_zip_structure_is_sound(self):
        for path in matrix.fixture_paths():
            with self.subTest(fixture=path.name):
                self.assertEqual(matrix.structure_problems(path), [])

    def test_counts_and_settings_match_pinned_expectations(self):
        for path in matrix.fixture_paths():
            expected = dict(matrix.SHARED, **matrix.EXPECTED[path.stem])
            facts = matrix.facts(path)
            with self.subTest(fixture=path.name):
                self.assertEqual(facts["objects"], expected["objects"])
                self.assertEqual(facts["build_items"], expected["build_items"])
                self.assertEqual(facts["vertices"], expected["vertices"])
                self.assertEqual(facts["triangles"], expected["triangles"])
                self.assertEqual(facts["printer_model"], expected["printer_model"])
                self.assertEqual(facts["paint"], expected["paint"])
                self.assertEqual(facts["extruders"], [expected["extruder"]])
                self.assertEqual(facts["nozzle"], expected["nozzle"])
                self.assertEqual(len(facts["filament_colours"]), expected["filaments"])
                self.assertEqual(len(facts["filament_types"]), expected["filaments"])

    def test_every_source_carries_stale_bambu_gcode(self):
        # The "no stale G-code survives conversion" expectation is only meaningful if the
        # source actually carries G-code. Pin that here.
        for path in matrix.fixture_paths():
            with self.subTest(fixture=path.name):
                self.assertEqual(matrix.facts(path)["gcode"], STALE_GCODE)

    def test_used_paint_channels_fit_the_palette(self):
        for path in matrix.fixture_paths():
            facts = matrix.facts(path)
            used = {PAINT_CHANNEL[code] for code in facts["paint"] if code in PAINT_CHANNEL}
            with self.subTest(fixture=path.name):
                self.assertEqual(set(facts["paint"]) - set(PAINT_CHANNEL), set(),
                                 "paint code outside the documented scheme")
                self.assertTrue(used <= set(range(1, len(facts["filament_colours"]) + 1)))

    def test_nozzle_variant_names_match_their_nozzle_setting(self):
        for path in matrix.fixture_paths():
            match = re.search(r"nozzle-(\d\.\d)", path.stem)
            if match:
                with self.subTest(fixture=path.name):
                    self.assertEqual(matrix.facts(path)["nozzle"], [match.group(1)])

    def test_slice_info_colours_agree_with_project_palette_when_present(self):
        for path in matrix.fixture_paths():
            with zipfile.ZipFile(path) as archive:
                if "Metadata/slice_info.config" not in archive.namelist():
                    continue
                root = ET.fromstring(archive.read("Metadata/slice_info.config"))
                slots = sorted(
                    (el for el in root.iter() if el.tag.rsplit("}", 1)[-1] == "filament"),
                    key=lambda el: int(el.get("id", "0")),
                )
                slice_colours = [el.get("color", "").upper() for el in slots]
            project = matrix.facts(path)["filament_colours"]
            with self.subTest(fixture=path.name):
                self.assertEqual(slice_colours, [c.upper() for c in project])

    def test_overheight_fixture_differs_only_in_layer_height(self):
        # Documents the finding: the "overheight" name is not backed by a taller model.
        base = zipfile.ZipFile(matrix.FIXTURES / "bambu-nozzle-0.2.3mf")
        over = zipfile.ZipFile(matrix.FIXTURES / "bambu-nozzle-0.2-overheight.3mf")
        with base, over:
            self.assertEqual(base.namelist(), over.namelist())
            changed = [n for n in base.namelist() if base.read(n) != over.read(n)]
            self.assertEqual(changed, ["Metadata/project_settings.config"])
            a = json.loads(base.read("Metadata/project_settings.config"))
            b = json.loads(over.read("Metadata/project_settings.config"))
            differing = sorted(k for k in set(a) | set(b) if a.get(k) != b.get(k))
            self.assertEqual(differing, ["layer_height"])
            self.assertEqual((a["layer_height"], b["layer_height"]), ("0.10", "0.16"))


if __name__ == "__main__":
    unittest.main()
