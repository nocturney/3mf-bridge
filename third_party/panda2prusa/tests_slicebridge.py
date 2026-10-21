"""Headless tests for slicebridge_runner.py.

Run from this directory::

    python -m unittest tests_slicebridge -v

Refusal, identity and overwrite tests need only the standard library. Conversion tests
need lxml (the isolated /tmp venv, see README-SLICEBRIDGE.md) and skip cleanly without
it. They run on a synthetic multipart, painted, sparse-extruder project and, when present,
on the Bambu fixtures in the repository's tests/fixtures directory (or the directory named
by SLICEBRIDGE_FIXTURES).
"""

import glob
import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest
import warnings
import zipfile
from unittest import mock

import slicebridge_runner as runner

HERE = os.path.dirname(os.path.abspath(__file__))
RUNNER = os.path.join(HERE, "slicebridge_runner.py")
FIXTURE_DIR = os.environ.get(
    "SLICEBRIDGE_FIXTURES", os.path.normpath(os.path.join(HERE, "..", "..", "tests", "fixtures"))
)
FIXTURES = sorted(glob.glob(os.path.join(FIXTURE_DIR, "*.3mf")))
ENGINE_OK = runner._ENGINE_ERROR is None

# Git blob ids of the upstream files at the pinned commit (byte-identity guard).
UPSTREAM_BLOBS = {
    "LICENSE": "45af5a1ba6a00970b688e551436efc596a605c36",
    "README.md": "2ce43defa1e8a07061e8e2c8c071f31ee34a4733",
    "CHANGELOG.md": "83949c88eb60a94f5483fa5885113c01d9ffc75f",
    "requirements.txt": "a366824973f72bd8161c22b7c3f67bc4083616c8",
    "panda2prusa/__init__.py": "fe412edf8731eea1a9f7adb0c1155c1c65d36502",
    "panda2prusa/convert.py": "f7ad236f59319c04b73c7e0d88cec937e15297ae",
    "panda2prusa/ns.py": "8bf2593359bab33b9cfb52d08cb5b36c2de22f10",
    "panda2prusa/paint.py": "3eb8f327e484bcc11d49ed9504dc4f80e016dc5e",
    "panda2prusa/reader.py": "43d83e53d62f97e6c509b6265c9d69c8e110d3c1",
    "panda2prusa/transform.py": "2e6239e09acd3f3be8a13ddd9961a8a6ac448a20",
    "panda2prusa/writer.py": "e768fcdb6dddfc1bbbce46ef4119a9597dcce853",
}

TETRA_VERTS = (
    '<vertices><vertex x="0" y="0" z="0"/><vertex x="10" y="0" z="0"/>'
    '<vertex x="0" y="10" z="0"/><vertex x="0" y="0" z="10"/></vertices>'
)
TETRA_TRIS = (
    '<triangle v1="0" v2="2" v3="1"/><triangle v1="0" v2="1" v3="3"/>'
    '<triangle v1="1" v2="2" v3="3"/><triangle v1="0" v2="3" v3="2"/>'
)
PAINTED_TRIS = (
    '<triangle v1="0" v2="2" v3="1" paint_color="4"/>'
    '<triangle v1="0" v2="1" v3="3"/><triangle v1="1" v2="2" v3="3"/><triangle v1="0" v2="3" v3="2"/>'
)

# One root object (id 3) built from two components: a painted tetrahedron (id 2) and a
# plain tetrahedron (id 4) moved 20 mm along X. Two leaves, two volumes, one painted triangle.
MULTIPART_MODEL = (
    '<?xml version="1.0" encoding="UTF-8"?>'
    '<model unit="millimeter" xmlns="http://schemas.microsoft.com/3dmanufacturing/core/2015/02">'
    "<resources>"
    f'<object id="2" type="model"><mesh>{TETRA_VERTS}<triangles>{PAINTED_TRIS}</triangles></mesh></object>'
    f'<object id="4" type="model"><mesh>{TETRA_VERTS}<triangles>{TETRA_TRIS}</triangles></mesh></object>'
    '<object id="3" type="model"><components>'
    '<component objectid="2"/>'
    '<component objectid="4" transform="1 0 0 0 1 0 0 0 1 20 0 0"/>'
    "</components></object>"
    "</resources>"
    '<build><item objectid="3" transform="1 0 0 0 1 0 0 0 1 5 5 0"/></build>'
    "</model>"
)
# Sparse filament slots: body on slot 1, clip on slot 3, object default slot 3.
MODEL_SETTINGS = (
    '<?xml version="1.0" encoding="UTF-8"?><config>'
    '<object id="3"><metadata key="name" value="Bracket"/><metadata key="extruder" value="3"/>'
    '<part id="2" subtype="normal_part"><metadata key="name" value="Body"/>'
    '<metadata key="extruder" value="1"/></part>'
    '<part id="4" subtype="normal_part"><metadata key="name" value="Clip"/>'
    '<metadata key="extruder" value="3"/></part>'
    "</object>"
    '<plate><metadata key="plater_id" value="1"/><model_instance>'
    '<metadata key="object_id" value="3"/></model_instance></plate>'
    "</config>"
)
NO_BUILD_MODEL = (
    '<?xml version="1.0" encoding="UTF-8"?>'
    '<model unit="millimeter" xmlns="http://schemas.microsoft.com/3dmanufacturing/core/2015/02">'
    "<resources/><build/></model>"
)
DOCTYPE_MODEL = '<!DOCTYPE model [<!ENTITY leak SYSTEM "file:///etc/hosts">]>' + NO_BUILD_MODEL


def _write_3mf(path, entries):
    """Write a zip from (name, bytes) pairs; duplicate names are allowed on purpose."""
    with warnings.catch_warnings():
        warnings.simplefilter("ignore", UserWarning)
        with zipfile.ZipFile(path, "w", zipfile.ZIP_DEFLATED) as zf:
            for name, data in entries:
                zf.writestr(name, data)


def _synthetic_3mf(path, model=MULTIPART_MODEL, settings=MODEL_SETTINGS):
    entries = [(runner.MODEL_PART, model.encode("utf-8"))]
    if settings is not None:
        entries.append(("Metadata/model_settings.config", settings.encode("utf-8")))
    _write_3mf(path, entries)
    return path


def _sha(path):
    with open(path, "rb") as fh:
        return hashlib.sha256(fh.read()).hexdigest()


def _leftovers(directory):
    return [name for name in os.listdir(directory) if name.startswith(".slicebridge-")]


class _TempCase(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.mkdtemp(prefix="slicebridge-test-")
        self.addCleanup(shutil.rmtree, self.tmp, True)
        self.source = _synthetic_3mf(os.path.join(self.tmp, "in.3mf"))
        self.output = os.path.join(self.tmp, "out.3mf")



class RefusalTests(_TempCase):
    """Refusals decided before any parsing, so they need no lxml."""

    def test_source_equal_to_output_refused_and_untouched(self):
        before = _sha(self.source)
        with self.assertRaises(runner.SliceBridgeError) as ctx:
            runner.run_conversion(self.source, self.source)
        self.assertEqual(ctx.exception.code, "SOURCE_OUTPUT_SAME_PATH")
        self.assertEqual(_sha(self.source), before)

    def test_existing_output_refused_and_untouched(self):
        with open(self.output, "wb") as fh:
            fh.write(b"precious")
        with self.assertRaises(runner.SliceBridgeError) as ctx:
            runner.run_conversion(self.source, self.output)
        self.assertEqual(ctx.exception.code, "OUTPUT_EXISTS")
        with open(self.output, "rb") as fh:
            self.assertEqual(fh.read(), b"precious")

    def test_dangling_symlink_output_refused(self):
        link = os.path.join(self.tmp, "link.3mf")
        os.symlink(os.path.join(self.tmp, "nowhere.3mf"), link)
        with self.assertRaises(runner.SliceBridgeError) as ctx:
            runner.run_conversion(self.source, link)
        self.assertEqual(ctx.exception.code, "OUTPUT_EXISTS")

    def test_symlink_to_source_as_output_refused(self):
        link = os.path.join(self.tmp, "alias.3mf")
        os.symlink(self.source, link)
        with self.assertRaises(runner.SliceBridgeError) as ctx:
            runner.run_conversion(self.source, link)
        self.assertEqual(ctx.exception.code, "SOURCE_OUTPUT_SAME_PATH")

    def test_relative_paths_refused(self):
        with self.assertRaises(runner.SliceBridgeError) as ctx:
            runner.run_conversion("in.3mf", self.output)
        self.assertEqual(ctx.exception.code, "BAD_ARGUMENTS")

    def test_non_3mf_extension_refused(self):
        with self.assertRaises(runner.SliceBridgeError) as ctx:
            runner.run_conversion(self.source, os.path.join(self.tmp, "out.zip"))
        self.assertEqual(ctx.exception.code, "BAD_ARGUMENTS")

    def test_missing_source_refused(self):
        with self.assertRaises(runner.SliceBridgeError) as ctx:
            runner.run_conversion(os.path.join(self.tmp, "absent.3mf"), self.output)
        self.assertEqual(ctx.exception.code, "SOURCE_NOT_FOUND")

    def test_missing_output_directory_refused(self):
        out = os.path.join(self.tmp, "no-such-dir", "out.3mf")
        with self.assertRaises(runner.SliceBridgeError) as ctx:
            runner.run_conversion(self.source, out)
        self.assertEqual(ctx.exception.code, "OUTPUT_DIR_UNAVAILABLE")

    def test_not_a_zip_refused(self):
        bogus = os.path.join(self.tmp, "bogus.3mf")
        with open(bogus, "wb") as fh:
            fh.write(b"this is not a zip file at all" * 10)
        with self.assertRaises(runner.SliceBridgeError) as ctx:
            runner.run_conversion(bogus, self.output)
        self.assertEqual(ctx.exception.code, "INPUT_NOT_ZIP")

    def test_path_traversal_entry_refused(self):
        evil = os.path.join(self.tmp, "evil.3mf")
        _write_3mf(evil, [("../escape.model", b"x"), (runner.MODEL_PART, NO_BUILD_MODEL.encode())])
        with self.assertRaises(runner.SliceBridgeError) as ctx:
            runner.run_conversion(evil, self.output)
        self.assertEqual(ctx.exception.code, "INPUT_UNSAFE_PATH")

    def test_absolute_entry_name_refused(self):
        evil = os.path.join(self.tmp, "abs.3mf")
        _write_3mf(evil, [("/3D/3dmodel.model", NO_BUILD_MODEL.encode())])
        with self.assertRaises(runner.SliceBridgeError) as ctx:
            runner.run_conversion(evil, self.output)
        self.assertEqual(ctx.exception.code, "INPUT_UNSAFE_PATH")

    def test_duplicate_entry_refused(self):
        dup = os.path.join(self.tmp, "dup.3mf")
        model = NO_BUILD_MODEL.encode()
        _write_3mf(dup, [(runner.MODEL_PART, model), (runner.MODEL_PART.lower(), model)])
        with self.assertRaises(runner.SliceBridgeError) as ctx:
            runner.run_conversion(dup, self.output)
        self.assertEqual(ctx.exception.code, "INPUT_DUPLICATE_ENTRY")

    def test_missing_model_refused(self):
        bare = os.path.join(self.tmp, "bare.3mf")
        _write_3mf(bare, [("readme.txt", b"no model here")])
        with self.assertRaises(runner.SliceBridgeError) as ctx:
            runner.run_conversion(bare, self.output)
        self.assertEqual(ctx.exception.code, "INPUT_MISSING_MODEL")

    def test_doctype_and_entities_refused(self):
        dtd = os.path.join(self.tmp, "dtd.3mf")
        _write_3mf(dtd, [(runner.MODEL_PART, DOCTYPE_MODEL.encode())])
        with self.assertRaises(runner.SliceBridgeError) as ctx:
            runner.run_conversion(dtd, self.output)
        self.assertEqual(ctx.exception.code, "INPUT_XML_DTD_FORBIDDEN")

    def test_compression_bomb_refused(self):
        bomb = os.path.join(self.tmp, "bomb.3mf")
        _write_3mf(bomb, [(runner.MODEL_PART, NO_BUILD_MODEL.encode()), ("Metadata/pad.bin", bytes(32 << 20))])
        with self.assertRaises(runner.SliceBridgeError) as ctx:
            runner.run_conversion(bomb, self.output)
        self.assertEqual(ctx.exception.code, "INPUT_SUSPICIOUS_COMPRESSION")

    def test_entry_count_limit_refused(self):
        with mock.patch.object(runner, "MAX_ENTRIES", 1):
            with self.assertRaises(runner.SliceBridgeError) as ctx:
                runner.run_conversion(self.source, self.output)
        self.assertEqual(ctx.exception.code, "INPUT_TOO_MANY_ENTRIES")

    def test_source_size_limit_refused(self):
        with mock.patch.object(runner, "MAX_SOURCE_BYTES", 10):
            with self.assertRaises(runner.SliceBridgeError) as ctx:
                runner.run_conversion(self.source, self.output)
        self.assertEqual(ctx.exception.code, "INPUT_TOO_LARGE")

    def test_unsafe_entry_name_helper(self):
        self.assertTrue(runner._is_safe_entry_name("3D/Objects/object_1.model"))
        self.assertTrue(runner._is_safe_entry_name("Metadata/plate_1.png"))
        for bad in ("", "../x", "a/../b", "a//b", "/abs", "C:/x", "a\\b", "a/./b"):
            self.assertFalse(runner._is_safe_entry_name(bad), bad)


@unittest.skipUnless(ENGINE_OK, f"engine unavailable: {runner._ENGINE_ERROR}")
class ConversionTests(_TempCase):
    """Real conversions through the vendored upstream engine."""

    def test_multipart_painted_sparse_round_trip(self):
        before = _sha(self.source)
        payload = runner.run_conversion(self.source, self.output)
        self.assertTrue(payload["ok"])
        self.assertEqual(payload["engine"], "panda2prusa")
        self.assertEqual(payload["source_sha256"], before)
        self.assertEqual(_sha(self.source), before)
        self.assertIsNone(payload["extruder_remap"])
        self.assertFalse(payload["printer_profile"]["verified"])
        self.assertEqual(payload["stats"]["objects"], 1)
        self.assertEqual(payload["stats"]["volumes"], 2)
        self.assertEqual(payload["stats"]["triangles"], 8)
        self.assertEqual(payload["stats"]["vertices"], 8)
        self.assertEqual(payload["stats"]["painted_triangles"], 1)
        self.assertEqual(payload["stats"]["build_items"], 1)
        self.assertEqual(payload["output_sha256"], _sha(self.output))

        with zipfile.ZipFile(self.output) as zf:
            names = zf.namelist()
            self.assertFalse([n for n in names if "gcode" in n.lower()])
            model = zf.read(runner.MODEL_PART).decode("utf-8")
            config = runner._parse_xml(zf.read(runner.CONFIG_PART))
        self.assertIn('slic3rpe:mmu_segmentation="4"', model)
        self.assertNotIn("paint_color", model)
        obj = config.find("object")
        self.assertEqual(runner._meta_value(obj, "extruder"), "3")
        volumes = obj.findall("volume")
        self.assertEqual(sorted(runner._meta_value(v, "extruder") for v in volumes), ["1", "3"])

    def test_second_run_to_same_output_refused(self):
        runner.run_conversion(self.source, self.output)
        digest = _sha(self.output)
        with self.assertRaises(runner.SliceBridgeError) as ctx:
            runner.run_conversion(self.source, self.output)
        self.assertEqual(ctx.exception.code, "OUTPUT_EXISTS")
        self.assertEqual(_sha(self.output), digest)
        self.assertEqual(_leftovers(self.tmp), [])

    def test_out_of_range_source_index_refused(self):
        bad_model = MULTIPART_MODEL.replace('v1="0" v2="2" v3="1" paint_color', 'v1="9" v2="2" v3="1" paint_color')
        self.assertNotEqual(bad_model, MULTIPART_MODEL)
        bad = _synthetic_3mf(os.path.join(self.tmp, "bad-index.3mf"), model=bad_model)
        with self.assertRaises(runner.SliceBridgeError) as ctx:
            runner.run_conversion(bad, self.output)
        self.assertEqual(ctx.exception.code, "INPUT_GEOMETRY_INVALID")
        self.assertFalse(os.path.lexists(self.output))

    def test_no_geometry_refused(self):
        empty = _synthetic_3mf(os.path.join(self.tmp, "empty.3mf"), model=NO_BUILD_MODEL)
        with self.assertRaises(runner.SliceBridgeError) as ctx:
            runner.run_conversion(empty, self.output)
        self.assertEqual(ctx.exception.code, "NO_GEOMETRY")
        self.assertEqual(_leftovers(self.tmp), [])
        self.assertFalse(os.path.lexists(self.output))

    def test_publish_failure_cleans_up_and_publishes_nothing(self):
        with mock.patch.object(runner.os, "link", side_effect=OSError(1, "Operation not permitted")):
            with self.assertRaises(runner.SliceBridgeError) as ctx:
                runner.run_conversion(self.source, self.output)
        self.assertEqual(ctx.exception.code, "PUBLISH_FAILED")
        self.assertFalse(os.path.lexists(self.output))
        self.assertEqual(_leftovers(self.tmp), [])

    def test_output_appearing_during_conversion_is_not_replaced(self):
        def other_writer_wins(src, dst):
            with open(dst, "wb") as fh:
                fh.write(b"someone else")
            raise FileExistsError(17, "File exists", dst)

        with mock.patch.object(runner.os, "link", side_effect=other_writer_wins):
            with self.assertRaises(runner.SliceBridgeError) as ctx:
                runner.run_conversion(self.source, self.output)
        self.assertEqual(ctx.exception.code, "OUTPUT_EXISTS")
        with open(self.output, "rb") as fh:
            self.assertEqual(fh.read(), b"someone else")
        self.assertEqual(_leftovers(self.tmp), [])

    def test_verification_failure_publishes_nothing(self):
        failing = runner.SliceBridgeError("GEOMETRY_MISMATCH", "forced for test")
        with mock.patch.object(runner, "_verify_output", side_effect=failing):
            with self.assertRaises(runner.SliceBridgeError) as ctx:
                runner.run_conversion(self.source, self.output)
        self.assertEqual(ctx.exception.code, "GEOMETRY_MISMATCH")
        self.assertFalse(os.path.lexists(self.output))
        self.assertEqual(_leftovers(self.tmp), [])

    def test_source_modified_during_conversion_publishes_nothing(self):
        real_convert = runner.convert_file

        def tampering_convert(src, dst, **kwargs):
            result = real_convert(src, dst, **kwargs)
            with open(src, "ab") as fh:
                fh.write(b"\0tampered")
            return result

        with mock.patch.object(runner, "convert_file", side_effect=tampering_convert):
            with self.assertRaises(runner.SliceBridgeError) as ctx:
                runner.run_conversion(self.source, self.output)
        self.assertEqual(ctx.exception.code, "SOURCE_MODIFIED")
        self.assertFalse(os.path.lexists(self.output))
        self.assertEqual(_leftovers(self.tmp), [])

    def test_plate_selection_keeps_only_requested_plate(self):
        project_plates = runner.read_bambu_3mf(self.source).plate_ids
        self.assertEqual(project_plates, [1])
        payload = runner.run_conversion(self.source, self.output, plates=[1])
        self.assertEqual(payload["plates_selected"], [1])
        empty_plate = os.path.join(self.tmp, "other.3mf")
        with self.assertRaises(runner.SliceBridgeError) as ctx:
            runner.run_conversion(self.source, empty_plate, plates=[7])
        self.assertEqual(ctx.exception.code, "NO_GEOMETRY")
        self.assertFalse(os.path.lexists(empty_plate))

    @unittest.skipUnless(FIXTURES, f"no fixtures found in {FIXTURE_DIR}")
    def test_repository_fixtures_convert_and_verify(self):
        for fixture in FIXTURES:
            with self.subTest(fixture=os.path.basename(fixture)):
                out = os.path.join(self.tmp, "fixture-" + os.path.basename(fixture))
                before = _sha(fixture)
                payload = runner.run_conversion(fixture, out)
                self.assertTrue(payload["ok"])
                self.assertGreater(payload["stats"]["objects"], 0)
                self.assertEqual(_sha(fixture), before)
                self.assertEqual(payload["output_sha256"], _sha(out))

class VendorTests(unittest.TestCase):
    def test_vendored_upstream_is_byte_identical(self):
        for rel, expected in UPSTREAM_BLOBS.items():
            with open(os.path.join(HERE, rel), "rb") as fh:
                data = fh.read()
            blob = hashlib.sha1(b"blob %d\0" % len(data) + data).hexdigest()
            self.assertEqual(blob, expected, rel)


class CliContractTests(_TempCase):
    """The stable CLI contract: one JSON object on stdout, exit codes as documented."""

    def run_cli(self, *args):
        proc = subprocess.run(
            [sys.executable, "-I", RUNNER, *args],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            universal_newlines=True,
            timeout=600,
        )
        return proc

    def test_missing_json_flag_is_bad_arguments(self):
        proc = self.run_cli("--source", self.source, "--output", self.output)
        self.assertEqual(proc.returncode, 2)
        payload = json.loads(proc.stdout)
        self.assertEqual(payload["code"], "BAD_ARGUMENTS")
        self.assertFalse(payload["ok"])
        self.assertEqual(proc.stderr, "")

    def test_refusal_is_json_with_nonzero_exit(self):
        proc = self.run_cli("--source", self.source, "--output", self.source, "--json")
        self.assertEqual(proc.returncode, 1)
        self.assertEqual(len(proc.stdout.strip().splitlines()), 1)
        payload = json.loads(proc.stdout)
        self.assertFalse(payload["ok"])
        self.assertEqual(payload["code"], "SOURCE_OUTPUT_SAME_PATH")
        self.assertIn("error", payload)

    @unittest.skipUnless(ENGINE_OK, f"engine unavailable: {runner._ENGINE_ERROR}")
    def test_success_is_single_json_object(self):
        proc = self.run_cli("--source", self.source, "--output", self.output, "--json")
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        self.assertEqual(proc.stderr, "")
        lines = proc.stdout.strip().splitlines()
        self.assertEqual(len(lines), 1)
        payload = json.loads(lines[0])
        self.assertTrue(payload["ok"])
        self.assertEqual(payload["engine"], "panda2prusa")
        self.assertTrue(os.path.isfile(self.output))


if __name__ == "__main__":
    unittest.main()
