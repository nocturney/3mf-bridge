"""Unit tests for the conversion fidelity harness.

Run from the repository root:  python3 -m unittest discover -s tests/compat -v
Every project is generated in a temporary directory. Repository fixtures are
only read, never written.
"""
import contextlib
import hashlib
import io
import json
import os
import tempfile
import unittest

import synthetic as syn
import threemf_compat as qa

HERE = os.path.dirname(os.path.abspath(__file__))
FIXTURES = os.path.normpath(os.path.join(HERE, "..", "fixtures"))
COLORS = ["#FF0000", "#00FF00", "#FF0000", "#FFFF00", "#FFFFFF"]  # slot 3 duplicates slot 1
TYPES = ["PLA", "PLA", "PLA", "PLA", "PLA"]
PAINTS = ["4", None, None, "2C"]  # channel 1 and channel 5 (fifth) on source triangles


def statuses(report):
    return {c["name"]: c["status"] for c in report["checks"]}


def failed(report):
    return sorted(name for name, status in statuses(report).items() if status == "fail")


class Base(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.dir = self._tmp.name
        self.addCleanup(self._tmp.cleanup)

    def write(self, name, entries):
        return syn.write_zip(os.path.join(self.dir, name), entries)

    def detail(self, report, name):
        return next(c["detail"] for c in report["checks"] if c["name"] == name)

    def pair(self, src_entries, out_entries, **kwargs):
        src = self.write("src.3mf", src_entries)
        out = self.write("out.3mf", out_entries)
        return qa.compare(src, out, **kwargs)

    def source(self, **kwargs):
        opts = dict(colors=COLORS, types=TYPES, extruder=5, paints=PAINTS)
        opts.update(kwargs)
        return syn.bbl_package(**opts)

    def clean(self, entries):
        return syn.sanitized(entries)


class BblSyntheticTests(Base):
    def test_clean_conversion_passes_and_reports_channels(self):
        report = self.pair(self.source(), self.clean(self.source()), source_sha256=None)
        self.assertTrue(report["ok"], failed(report))
        self.assertEqual(report["facts"]["source_channels"], [1, 5])
        self.assertEqual(statuses(report)["virtual_channels"], "pass")
        self.assertEqual(statuses(report)["palette_slots"], "pass")
        self.assertIn("1 duplicate colour/material slots", self.detail(report, "palette_slots"))

    def test_dropping_fifth_slot_fails_palette_and_channel(self):
        out = syn.set_bbl_palette(self.clean(self.source()), COLORS[:4], TYPES[:4])
        report = self.pair(self.source(), out)
        self.assertFalse(report["ok"])
        self.assertIn("palette_slots", failed(report))
        self.assertIn("virtual_channels", failed(report))

    def test_collapsing_duplicate_slot_fails(self):
        # Folding the duplicate #FF0000 slot away is the silent data loss the engine forbids.
        out = syn.set_bbl_palette(self.clean(self.source()), [COLORS[0], COLORS[1], COLORS[3], COLORS[4]], TYPES[:4])
        report = self.pair(self.source(), out)
        self.assertIn("palette_slots", failed(report))

    def test_material_change_fails(self):
        types = list(TYPES)
        types[1] = "PETG"
        out = syn.set_bbl_palette(self.clean(self.source()), COLORS, types)
        report = self.pair(self.source(), out)
        self.assertEqual(statuses(report)["palette_slots"], "fail")
        self.assertIn("slot 2", self.detail(report, "palette_slots"))

    def test_colour_swap_within_palette_fails(self):
        colors = list(COLORS)
        colors[0], colors[1] = colors[1], colors[0]
        out = syn.set_bbl_palette(self.clean(self.source()), colors, TYPES)
        self.assertIn("palette_slots", failed(self.pair(self.source(), out)))

    def test_paint_code_change_on_fifth_channel_fails(self):
        out = syn.set_triangle_paint(self.clean(self.source()), 3, "4")
        report = self.pair(self.source(), out)
        self.assertEqual(statuses(report)["paint_state"], "fail")

    def test_missing_painted_triangle_fails_counts(self):
        out = syn.drop_triangle(self.clean(self.source()), 3)
        report = self.pair(self.source(), out)
        self.assertEqual(statuses(report)["geometry_counts"], "fail")

    def test_moved_vertex_keeps_counts_but_fails_hash(self):
        out = syn.move_vertex(self.clean(self.source()), 1, 0.5)
        report = self.pair(self.source(), out)
        self.assertEqual(statuses(report)["geometry_counts"], "pass")
        self.assertEqual(statuses(report)["geometry_hash"], "fail")

    def test_object_extruder_remap_fails(self):
        out = syn.set_extruder(self.clean(self.source()), 1)
        self.assertEqual(statuses(self.pair(self.source(), out))["object_extruder_mapping"], "fail")

    def test_sliced_gcode_left_behind_fails(self):
        out = syn.add_gcode(self.clean(self.source()))
        self.assertEqual(statuses(self.pair(self.source(), out))["no_sliced_gcode"], "fail")

    def test_source_machine_script_copied_fails(self):
        out = self.clean(self.source())
        cfg_source = syn.bbl_package(COLORS, TYPES)  # carries the source script
        out["Metadata/project_settings.config"] = cfg_source["Metadata/project_settings.config"]
        report = self.pair(self.source(), out)
        self.assertEqual(statuses(report)["machine_scripts_not_carried"], "fail")

    def test_changed_process_value_fails(self):
        out = syn.set_process(self.clean(self.source()), "layer_height", "0.2")
        self.assertEqual(statuses(self.pair(self.source(), out))["process_intent"], "fail")

    def test_same_numeric_process_value_in_other_spelling_passes(self):
        out = syn.set_process(self.clean(self.source()), "layer_height", 0.16)
        self.assertEqual(statuses(self.pair(self.source(), out))["process_intent"], "pass")

    def test_missing_model_relationship_fails_components(self):
        out = self.clean(self.source())
        out["_rels/.rels"] = syn.RELS.replace("3dmodel", "other").encode()
        report = self.pair(self.source(), out)
        self.assertEqual(statuses(report)["output_components"], "fail")
        self.assertFalse(report["ok"])

    def test_missing_content_types_fails_components(self):
        out = self.clean(self.source())
        del out["[Content_Types].xml"]
        self.assertEqual(statuses(self.pair(self.source(), out))["output_components"], "fail")

    def test_dangling_model_relationship_fails(self):
        out = self.clean(self.source())
        out["_rels/.rels"] = syn.RELS.replace("/3D/3dmodel.model", "/3D/missing.model").encode()
        self.assertEqual(statuses(self.pair(self.source(), out))["output_components"], "fail")

    def test_invalid_triangle_index_is_a_failure_not_a_crash(self):
        def point_outside(root):
            first = [el for el in root.iter() if el.tag.endswith("triangle")][0]
            first.set("v3", "99")
        out = syn.edit_model(self.clean(self.source()), point_outside)
        report = self.pair(self.source(), out)
        self.assertEqual(statuses(report)["output_geometry_parse"], "fail")

    def test_duplicate_zip_entry_fails_zip_sanity(self):
        out = self.clean(self.source())
        pairs = list(out.items()) + [("3D/3dmodel.model", out["3D/3dmodel.model"])]
        self.assertEqual(statuses(self.pair(self.source(), pairs))["output_zip_sanity"], "fail")

    def test_unsafe_entry_name_fails_zip_sanity(self):
        out = self.clean(self.source())
        pairs = list(out.items()) + [("../escape.txt", b"x")]
        self.assertEqual(statuses(self.pair(self.source(), pairs))["output_zip_sanity"], "fail")

    def test_truncated_archive_is_an_input_error(self):
        src = self.write("src.3mf", self.source())
        bad = os.path.join(self.dir, "bad.3mf")
        with open(src, "rb") as fh, open(bad, "wb") as out:
            out.write(fh.read()[:200])
        with self.assertRaises(qa.InputError):
            qa.compare(src, bad)

    def test_source_without_palette_is_skipped_not_passed(self):
        src = self.source()
        del src["Metadata/project_settings.config"]
        src["Metadata/slice_info.config"] = b"<config><plate></plate></config>"
        report = self.pair(src, self.clean(self.source()))
        self.assertEqual(statuses(report)["palette_slots"], "skip")

    def test_expected_printer_model_is_enforced(self):
        report = self.pair(self.source(), self.clean(self.source()), expect_model="Snapmaker U1")
        self.assertEqual(statuses(report)["printer_model"], "fail")
        report = self.pair(self.source(), self.clean(self.source()), expect_model="Bambu Lab A1")
        self.assertEqual(statuses(report)["printer_model"], "pass")

    def test_report_never_claims_printability(self):
        report = self.pair(self.source(), self.clean(self.source()))
        self.assertTrue(any("does not prove" in note for note in report["notes"]))


class NonOverwriteTests(Base):
    def setUp(self):
        super().setUp()
        self.src = self.write("src.3mf", self.source())
        self.out = self.write("out.3mf", self.clean(self.source()))
        self.src_hash = qa._file_sha256(self.src)
        self.out_hash = qa._file_sha256(self.out)

    def status(self, **kwargs):
        return statuses(qa.compare(self.src, self.out, **kwargs))["non_overwrite"]

    def test_output_equal_to_source_path_fails(self):
        report = qa.compare(self.src, self.src)
        self.assertEqual(statuses(report)["non_overwrite"], "fail")

    def test_no_hashes_is_skip_not_pass(self):
        self.assertEqual(self.status(), "skip")

    def test_verified_hashes_pass(self):
        self.assertEqual(self.status(source_sha256=self.src_hash, prior_output_sha256=self.out_hash), "pass")

    def test_overwritten_preexisting_output_fails(self):
        other = hashlib.sha256(b"something else").hexdigest()
        self.assertEqual(self.status(prior_output_sha256=other), "fail")

    def test_mutated_source_fails(self):
        self.assertEqual(self.status(source_sha256="0" * 64), "fail")


class FamilyTests(Base):
    def test_prusa_identity_passes_and_mutation_fails(self):
        colors = ["#FF0000", "#00FF00"]
        types = ["PLA", "PETG"]
        src = syn.prusa_package(colors, types, extruder=2, paints=["4", "8"])
        self.assertTrue(self.pair(src, src)["ok"])
        report = self.pair(src, syn.set_prusa_palette(src, colors, ["PLA", "PLA"]))
        self.assertEqual(statuses(report)["palette_slots"], "fail")
        self.assertEqual(report["family"]["source"], "prusa")

    def test_cross_family_skips_paint_but_keeps_palette_check(self):
        src = self.source()
        prusa_out = syn.prusa_package(COLORS, TYPES, extruder=5, paints=PAINTS)
        report = self.pair(src, prusa_out)
        self.assertEqual(report["family"]["output"], "prusa")
        self.assertFalse(report["family"]["paint_codec_comparable"])
        self.assertEqual(statuses(report)["paint_state"], "skip")
        self.assertEqual(statuses(report)["palette_slots"], "pass")

    def test_cross_family_palette_loss_still_fails(self):
        src = self.source()
        prusa_out = syn.prusa_package(COLORS[:4], TYPES[:4], extruder=5, paints=PAINTS)
        self.assertIn("palette_slots", failed(self.pair(src, prusa_out)))

    def test_declared_family_must_match_source_layout(self):
        with self.assertRaises(qa.InputError):
            qa.compare(self.write("s.3mf", self.source()), self.write("o.3mf", self.source()), family="prusa")

    def test_unknown_layout_requires_family(self):
        plain = {"[Content_Types].xml": syn.CONTENT_TYPES, "_rels/.rels": syn.RELS,
                 "3D/3dmodel.model": syn.model_xml()}
        with self.assertRaises(qa.InputError):
            qa.compare(self.write("p.3mf", plain), self.write("q.3mf", plain))
        report = qa.compare(self.write("p.3mf", plain), self.write("q.3mf", plain), family="orca")
        self.assertEqual(report["family"]["source"], "orca")
        self.assertEqual(report["family"]["source_basis"], "declared")


class FixtureTests(Base):
    """Repository fixtures are read-only inputs; outputs are sanitized temporary copies."""

    def fixture(self, name):
        return os.path.join(FIXTURES, name)

    def test_every_repo_fixture_is_self_consistent_after_sanitizing(self):
        names = sorted(n for n in os.listdir(FIXTURES) if n.endswith(".3mf"))
        self.assertGreaterEqual(len(names), 5)
        for name in names:
            with self.subTest(fixture=name):
                before = qa._file_sha256(self.fixture(name))
                out = self.write("clean.3mf", self.clean(syn.read_zip(self.fixture(name))))
                report = qa.compare(self.fixture(name), out)
                self.assertTrue(report["ok"], (name, failed(report)))
                self.assertEqual(qa._file_sha256(self.fixture(name)), before)

    def test_fifth_channel_fixture_drop_fails(self):
        entries = syn.read_zip(self.fixture("bambu-five-colors.3mf"))
        clean = self.clean(entries)
        self.assertEqual(qa.compare(self.fixture("bambu-five-colors.3mf"),
                                    self.write("ok.3mf", clean))["facts"]["source_channels"], [1, 5])
        palette = qa.palette(qa.Package(self.fixture("bambu-five-colors.3mf")), "bambu")
        self.assertEqual(len(palette), 5)
        out = syn.set_bbl_palette(clean, [c for c, _ in palette][:4], [t for _, t in palette][:4])
        report = qa.compare(self.fixture("bambu-five-colors.3mf"), self.write("drop.3mf", out))
        self.assertIn("virtual_channels", failed(report))

    def test_fifth_paint_fixture_recode_fails(self):
        clean = self.clean(syn.read_zip(self.fixture("bambu-five-colors.3mf")))
        model = clean["3D/3dmodel.model"].replace(b"paint_color='2C'", b"paint_color='4'")
        out = dict(clean, **{"3D/3dmodel.model": model})
        report = qa.compare(self.fixture("bambu-five-colors.3mf"), self.write("recode.3mf", out))
        self.assertEqual(statuses(report)["paint_state"], "fail")

    def test_mixed_fixture_channels_are_one_to_three(self):
        report = qa.compare(self.fixture("bambu-mixed-three.3mf"),
                            self.write("m.3mf", self.clean(syn.read_zip(self.fixture("bambu-mixed-three.3mf")))))
        self.assertEqual(report["facts"]["source_channels"], [1, 2, 3])


class NestedRealWorldPackageTests(Base):
    def nested(self):
        entries=dict(self.source())
        actual=entries["3D/3dmodel.model"]
        entries["3D/Objects/object_1.model"]=actual
        entries["3D/3dmodel.model"]=(
            b'<model xmlns="http://schemas.microsoft.com/3dmanufacturing/core/2015/02">'
            b'<resources/><build/></model>')
        entries["3D/_rels/3dmodel.model.rels"]=(
            b'<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
            b'<Relationship Id="child" Target="/3D/Objects/object_1.model" '
            b'Type="http://schemas.microsoft.com/3dmanufacturing/2013/01/3dmodel"/>'
            b'</Relationships>')
        return entries

    def test_child_model_geometry_is_counted(self):
        source=self.nested()
        src=self.write("src.3mf",source)
        source_package=qa.Package(src)
        paths=qa.model_parts(source_package)
        self.assertEqual(paths, ["3D/3dmodel.model","3D/Objects/object_1.model"])
        self.assertGreater(qa.geometry(source_package,paths)["triangles"],0)
        report=self.pair(source,self.clean(source))
        self.assertTrue(report["ok"],failed(report))
        self.assertEqual(statuses(report)["geometry_counts"],"pass")

    def test_unreferenced_gcode_mime_default_is_not_slice_cache(self):
        out=self.clean(self.source())
        out["[Content_Types].xml"]=out["[Content_Types].xml"].replace(
            b"</Types>", b'<Default Extension="gcode" ContentType="text/x.gcode"/></Types>')
        report=self.pair(self.source(),out)
        self.assertEqual(statuses(report)["no_sliced_gcode"],"pass")

    def test_explicit_gcode_part_reference_is_rejected(self):
        out=self.clean(self.source())
        out["_rels/.rels"]=out["_rels/.rels"].replace(
            b"</Relationships>",
            b'<Relationship Id="bad" Target="/Metadata/plate_1.gcode" '
            b'Type="http://example.com/sliced-cache"/></Relationships>')
        report=self.pair(self.source(),out)
        self.assertEqual(statuses(report)["no_sliced_gcode"],"fail")

    def test_missing_nested_model_fails_even_when_root_shell_exists(self):
        entries=self.nested()
        del entries["3D/Objects/object_1.model"]
        with self.assertRaisesRegex(qa.InputError,"model relationship target missing"):
            qa.compare(self.write("src.3mf",entries),
                       self.write("out.3mf",self.clean(self.source())))


class PrusaMetadataOnlyTests(Base):
    def generic_prusa(self, slots=None):
        if slots is None:
            slots=[{"id":i + 1, "color":c, "material":t}
                   for i, (c, t) in enumerate(zip(COLORS, TYPES))]
        entries=syn.prusa_package(COLORS, TYPES, extruder=5, paints=PAINTS)
        entries.pop("Metadata/Slic3r_PE.config", None)
        entries["Metadata/SliceBridge_source_filaments.json"]=json.dumps(
            {"schemaVersion":1, "source":"bambu-orca", "slots":slots}).encode()
        return entries

    def test_metadata_only_prusa_palette_is_preserved_without_native_preset_claim(self):
        report=self.pair(self.source(), self.generic_prusa())
        self.assertTrue(report["ok"], failed(report))
        self.assertEqual(statuses(report)["palette_slots"], "pass")
        self.assertEqual(statuses(report)["virtual_channels"], "pass")
        self.assertEqual(statuses(report)["paint_state"], "skip")
        self.assertEqual(statuses(report)["process_intent"], "skip")
        self.assertEqual(statuses(report)["machine_scripts_not_carried"], "skip")
        self.assertIn("metadata only", report["facts"]["output_palette_basis"])

    def test_metadata_only_prusa_palette_drop_fails(self):
        out=self.generic_prusa()
        record=json.loads(out["Metadata/SliceBridge_source_filaments.json"])
        record["slots"].pop()
        out["Metadata/SliceBridge_source_filaments.json"]=json.dumps(record).encode()
        report=self.pair(self.source(),out)
        self.assertEqual(statuses(report)["palette_slots"], "fail")
        self.assertEqual(statuses(report)["virtual_channels"], "fail")

    def test_duplicate_slot_id_fails_explicitly(self):
        out=self.generic_prusa()
        record=json.loads(out["Metadata/SliceBridge_source_filaments.json"])
        record["slots"][3]["id"]=3
        out["Metadata/SliceBridge_source_filaments.json"]=json.dumps(record).encode()
        report=self.pair(self.source(),out)
        self.assertEqual(statuses(report)["output_palette"], "fail")
        self.assertEqual(statuses(report)["palette_slots"], "fail")


class CliTests(Base):
    def run_cli(self, *args):
        buffer = io.StringIO()
        with contextlib.redirect_stdout(buffer):
            code = qa.main(list(args))
        return code, buffer.getvalue()

    def test_exit_codes(self):
        src = self.write("src.3mf", self.source())
        ok = self.write("ok.3mf", self.clean(self.source()))
        bad = self.write("bad.3mf", syn.set_bbl_palette(self.clean(self.source()), COLORS[:4], TYPES[:4]))
        self.assertEqual(self.run_cli("--source", src, "--output", ok)[0], 0)
        self.assertEqual(self.run_cli("--source", src, "--output", bad)[0], 1)
        code, text = self.run_cli("--source", src, "--output", os.path.join(self.dir, "missing.3mf"))
        self.assertEqual(code, 2)
        self.assertIn('"ok": false', text)


if __name__ == "__main__":
    unittest.main()
