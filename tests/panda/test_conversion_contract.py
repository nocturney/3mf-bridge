"""Bambu source vs Prusa output contract for the bundled Panda2Prusa sidecar (stdlib unittest).

Two layers, both anchored to the real fixture tests/fixtures/bambu-five-colors.3mf:

- HarnessSelfChecks always run. They prove the comparator reads the fixture correctly and
  catches each kind of defect. The Prusa-shaped reference they compare against is built here
  from the fixture. It is a harness check, not engine output.
- EngineContract runs the sidecar through the runner contract (--source --output --json).
  It is skipped only while third_party/panda2prusa is not vendored. Once the vendor is present,
  a missing or failing engine is a test failure, not a skip.

Run: python3 -m unittest discover -s tests/panda -p 'test_*.py' -v
"""
import json
import os
import shutil
import tempfile
import unittest
import zipfile
from pathlib import Path

import panda_qa as qa

VENDOR_READY = all((qa.VENDOR / item).exists() for item in ("slicebridge_runner.py", "panda2prusa", "LICENSE"))
VENDOR_SKIP = "third_party/panda2prusa is not vendored yet; the engine contract runs once it is integrated"
PRUSA_MODEL_CONFIG = "Metadata/Slic3r_PE_model.config"
PRUSA_INI = "Metadata/Slic3r_PE.config"
SOURCE_PALETTE = ["#FF0000", "#00FF00", "#0000FF", "#FFFF00", "#FFFFFF"]
# Slots 1, 2 and 4 deliberately share a colour. All five numbered slots must survive conversion.
DUPLICATE_PALETTE = ["#FF0000", "#FF0000", "#0000FF", "#FF0000", "#FFFFFF"]


def prusa_reference(extruder=5, palette=SOURCE_PALETTE):
    """Members of a Prusa-shaped copy of the fixture: object extruder and a palette INI.

    Used only to exercise the comparator's Prusa reading path. It says nothing about the engine.
    """
    model_config = (
        "<?xml version='1.0' encoding='UTF-8'?>\n<config>\n"
        " <object id='1' instances_count='1'>\n"
        "  <metadata type='object' key='extruder' value='%d'/>\n"
        "  <volume firstid='0' lastid='3'/>\n"
        " </object>\n</config>\n" % extruder
    ).encode()
    ini = ('filament_colour = "%s"\n' % ";".join(palette)).encode()
    return {PRUSA_MODEL_CONFIG: model_config, PRUSA_INI: ini}


def fixture_copy(directory, name="source.3mf", replacements=None):
    """A writable copy of the fixture inside a temp directory. The repo fixture is never written."""
    path = Path(directory) / name
    if replacements is None:
        shutil.copyfile(str(qa.FIVE_COLORS), str(path))
        return path
    return qa.write_variant(qa.FIVE_COLORS, path, replacements)


def duplicate_palette_variant(directory):
    """The fixture with slots 1, 2 and 4 all set to the same colour in both config sources."""
    src = qa.Package(qa.FIVE_COLORS)
    project = json.loads(src.text("Metadata/project_settings.config"))
    project["filament_colour"] = list(DUPLICATE_PALETTE)
    slice_info = src.text("Metadata/slice_info.config")
    slice_info = slice_info.replace("color='#00FF00'", "color='#FF0000'")
    slice_info = slice_info.replace("color='#FFFF00'", "color='#FF0000'")
    return fixture_copy(directory, "duplicate-colours.3mf", {
        "Metadata/project_settings.config": json.dumps(project).encode(),
        "Metadata/slice_info.config": slice_info.encode(),
    })


class FixtureFacts(unittest.TestCase):
    """The fixture carries the things the contract tests depend on."""

    @classmethod
    def setUpClass(cls):
        cls.src = qa.Package(qa.FIVE_COLORS)

    def test_fixture_is_the_five_colour_bambu_project(self):
        self.assertIn("3D/3dmodel.model", self.src.names)
        self.assertEqual(qa.source_palette(self.src), SOURCE_PALETTE)
        self.assertEqual(qa.source_settings(self.src), {"1": {"extruder": 5, "parts": {}}})

    def test_fixture_has_nested_and_plain_paint_codes(self):
        paints = sorted(paint for _, paint in qa.Model(self.src).mesh_records().elements())
        self.assertEqual(paints, ["2C", "4", "4", "4"])
        self.assertEqual([qa.decode_paint_channel(p) for p in paints], [5, 1, 1, 1])

    def test_fixture_carries_stale_gcode_the_output_must_drop(self):
        problems = qa.stale_gcode_problems(self.src)
        self.assertTrue(any("plate_1.gcode" in p for p in problems), problems)
        self.assertTrue(any("UNSAFE_BAMBU_SCRIPT" in p for p in problems), problems)


class HarnessSelfChecks(unittest.TestCase):
    """The comparator must accept a faithful Prusa-shaped copy and reject each defect."""

    @classmethod
    def setUpClass(cls):
        cls.temp = tempfile.TemporaryDirectory(prefix="panda-qa-self-")
        cls.dir = Path(cls.temp.name)
        cls.src = qa.Package(qa.FIVE_COLORS)

    @classmethod
    def tearDownClass(cls):
        cls.temp.cleanup()

    def reference(self, name, **kwargs):
        path = qa.write_variant(qa.FIVE_COLORS, self.dir / name, prusa_reference(**kwargs))
        return qa.Package(path)

    def model_variant(self, name, old, new, extra=None):
        model = self.src.text("3D/3dmodel.model")
        self.assertIn(old, model, "fixture no longer contains the expected token")
        replacements = dict(prusa_reference())
        replacements["3D/3dmodel.model"] = model.replace(old, new).encode()
        replacements.update(extra or {})
        return qa.Package(qa.write_variant(qa.FIVE_COLORS, self.dir / name, replacements))

    def test_faithful_reference_matches(self):
        self.assertEqual(qa.compare(self.src, self.reference("ok.3mf")), {})

    def test_changed_paint_bitstream_is_reported(self):
        out = self.model_variant("paint.3mf", "paint_color='2C'", "paint_color='4'")
        self.assertIn("paints", qa.compare(self.src, out))

    def test_nested_paint_string_must_stay_verbatim(self):
        out = self.model_variant("nested.3mf", "paint_color='2C'", "paint_color='2c'")
        self.assertIn("paints", qa.compare(self.src, out))

    def test_changed_vertex_is_reported_as_geometry(self):
        out = self.model_variant("geom.3mf", "<vertex x='10' y='0' z='0'/>", "<vertex x='11' y='0' z='0'/>")
        self.assertIn("geometry", qa.compare(self.src, out))

    def test_changed_build_transform_is_reported(self):
        out = self.model_variant("xform.3mf", "<item objectid='1'/>",
                                 "<item objectid='1' transform='1 0 0 0 1 0 0 0 1 5 0 0'/>")
        self.assertIn("transforms", qa.compare(self.src, out))

    def test_wrong_object_extruder_is_reported(self):
        out = self.reference("ext.3mf", extruder=3)
        self.assertIn("extruders", qa.compare(self.src, out))

    def test_compacted_palette_is_reported(self):
        out = self.reference("compact.3mf", palette=SOURCE_PALETTE[:4])
        self.assertIn("palette", qa.compare(self.src, out))

    def test_duplicate_colours_compared_by_slot_not_by_unique_colour(self):
        dup_src = qa.Package(duplicate_palette_variant(self.dir))
        exact = qa.Package(qa.write_variant(qa.FIVE_COLORS, self.dir / "dup-out.3mf",
                                            prusa_reference(palette=DUPLICATE_PALETTE)))
        self.assertEqual(qa.compare(dup_src, exact), {})
        collapsed = qa.Package(qa.write_variant(qa.FIVE_COLORS, self.dir / "dup-bad.3mf",
                                                prusa_reference(palette=["#FF0000", "#0000FF", "#FFFFFF"])))
        self.assertIn("palette", qa.compare(dup_src, collapsed))

    def test_stale_gcode_entry_is_reported(self):
        stale = qa.Package(qa.write_variant(qa.FIVE_COLORS, self.dir / "stale.3mf",
                                            {"Metadata/plate_1.gcode": b"M104 S300 ; stale printer-specific slice"}))
        self.assertTrue(qa.stale_gcode_problems(stale))
        clean_path = self.dir / "clean.3mf"
        with zipfile.ZipFile(clean_path, "w") as archive:
            archive.writestr("3D/3dmodel.model", self.src.files["3D/3dmodel.model"])
        self.assertEqual(qa.stale_gcode_problems(qa.Package(clean_path)), [])


@unittest.skipUnless(VENDOR_READY, VENDOR_SKIP)
class EngineContract(unittest.TestCase):
    """Real conversions of the fixture through the sidecar. Each test is an independent claim."""

    @classmethod
    def setUpClass(cls):
        found = qa.engine_command()
        if found is None:
            raise AssertionError("third_party/panda2prusa is vendored but no engine is available")
        cls.engine, cls.engine_source = found
        cls.temp = tempfile.TemporaryDirectory(prefix="panda-qa-engine-")
        cls.dir = Path(cls.temp.name)
        cls.source = fixture_copy(cls.dir)
        cls.source_sha = qa.sha256_file(cls.source)
        cls.fixture_sha = qa.sha256_file(qa.FIVE_COLORS)
        cls.output = cls.dir / "out" / "five-colors-prusa.3mf"
        cls.output.parent.mkdir()
        cls.result = qa.convert(cls.engine, cls.source, cls.output, cls.dir)
        cls.package = qa.Package(cls.output) if cls.output.is_file() and zipfile.is_zipfile(cls.output) else None

    @classmethod
    def tearDownClass(cls):
        cls.temp.cleanup()

    def assertConverted(self):
        if self.result.returncode != 0:
            self.fail("engine (%s) exited %d:\n%s" % (self.engine_source, self.result.returncode,
                                                      (self.result.stderr or self.result.stdout)[-2000:]))
        self.assertIsNotNone(self.package, "engine did not write a 3MF ZIP to --output")
        return self.package

    def assertNoProblem(self, *checks):
        package = self.assertConverted()
        problems = qa.compare(qa.Package(self.source), package)
        for check in checks:
            self.assertNotIn(check, problems, problems.get(check))

    def test_runner_contract_json_and_exit_status(self):
        self.assertConverted()
        self.assertEqual(self.result.returncode, 0)
        json.loads(self.result.stdout)
        self.assertIn("3D/3dmodel.model", self.package.names)

    def test_geometry_and_triangles(self):
        self.assertNoProblem("geometry")

    def test_build_transforms(self):
        self.assertNoProblem("transforms")

    def test_paint_bitstreams_verbatim_including_nested(self):
        self.assertNoProblem("paints")
        codes = [paint for _, paint in qa.Model(self.assertConverted()).mesh_records().elements()]
        self.assertIn("2C", codes, "nested paint code 2C must survive verbatim")

    def test_object_and_volume_extruder_assignments(self):
        self.assertNoProblem("extruders")

    def test_all_numbered_palette_slots_kept(self):
        self.assertNoProblem("palette")
        self.assertEqual(qa.output_palette(self.assertConverted()), SOURCE_PALETTE)

    def test_duplicate_colour_slots_are_not_compacted(self):
        dup_source = duplicate_palette_variant(self.dir)
        dup_out = self.dir / "out" / "duplicate-colours-prusa.3mf"
        result = qa.convert(self.engine, dup_source, dup_out, self.dir)
        self.assertEqual(result.returncode, 0, (result.stderr or result.stdout)[-2000:])
        out = qa.Package(dup_out)
        self.assertEqual(qa.output_palette(out), DUPLICATE_PALETTE)
        self.assertNotIn("palette", qa.compare(qa.Package(dup_source), out))

    def test_no_stale_gcode_or_bambu_script_survives(self):
        self.assertEqual(qa.stale_gcode_problems(self.assertConverted()), [])

    def test_source_bytes_unchanged_by_conversion(self):
        self.assertConverted()
        self.assertEqual(qa.sha256_file(self.source), self.source_sha)
        self.assertEqual(qa.sha256_file(qa.FIVE_COLORS), self.fixture_sha)

    def test_output_onto_source_path_is_refused(self):
        before = qa.sha256_file(self.source)
        result = qa.convert(self.engine, self.source, self.source, self.dir)
        self.assertNotEqual(result.returncode, 0, "converting onto the source path must be refused")
        self.assertEqual(qa.sha256_file(self.source), before)

    def test_symlinked_output_does_not_write_through(self):
        target = self.dir / "sentinel-target.txt"
        target.write_bytes(b"SENTINEL: must not change")
        link = self.dir / "out" / "symlink-output.3mf"
        try:
            os.symlink(str(target), str(link))
        except (OSError, NotImplementedError) as error:
            self.skipTest("symlinks unavailable on this host: %s" % error)
        qa.convert(self.engine, self.source, link, self.dir)
        self.assertEqual(target.read_bytes(), b"SENTINEL: must not change")
        self.assertEqual(qa.sha256_file(self.source), self.source_sha)


if __name__ == "__main__":
    unittest.main()
