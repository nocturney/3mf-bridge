"""Targeted regression tests for Prusa INI inheritance and mixed Snapmaker U1 nozzles."""
import hashlib
import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "tools"))
import build
import check
import prusa_ini


def valid_ini_sections():
    return {
        "printer_model:X1": {"name": "Sample X1", "variants": "0.4; 0.6",
                             "technology": "FFF"},
        "printer:*machine*": {"printer_model": "X1",
                              "nozzle_diameter": "0.4,0.4",
                              "printer_variant": "0.4",
                              "default_print_profile": "Standard",
                              "default_filament_profile": "PLA,PETG"},
        "printer:Real X1": {"inherits": "*machine*"},
        "print:*common*": {"speed": "30", "first_layer_speed": "15"},
        "print:*fast*": {"speed": "60"},
        "print:Standard": {"inherits": "*common*; *fast*", "speed": "45"},
        "filament:*base*": {"temperature": "210"},
        "filament:PLA": {"inherits": "*base*"},
        "filament:PETG": {"inherits": "*base*", "temperature": "235"},
    }


class PrusaIniCases(unittest.TestCase):
    def test_multiple_inheritance_and_child_wins(self):
        sections = valid_ini_sections()
        props, chain = prusa_ini.resolve("Standard",
                                         prusa_ini.sections_of_type(sections, "print"),
                                         "print", {})
        self.assertEqual(props["speed"], "45")
        self.assertEqual(props["first_layer_speed"], "15")
        self.assertEqual(chain, ["*common*", "*fast*", "Standard"])

    def test_missing_dependency_and_cycles_are_fail_closed(self):
        profiles = {"A": {"inherits": "B"}, "B": {"inherits": "A"}}
        with self.assertRaisesRegex(prusa_ini.InvalidIniProfile, "cycle"):
            prusa_ini.resolve("A", profiles, "print", {})
        with self.assertRaisesRegex(prusa_ini.InvalidIniProfile, "missing preset"):
            prusa_ini.resolve("A", {"A": {"inherits": "missing"}}, "print", {})

    def test_printer_print_and_all_filaments_are_verified(self):
        sections = valid_ini_sections()
        report = prusa_ini.validate_model(sections, "printer_model:X1",
                                          ["0.4", "0.6"], "bundles/example.ini", "a"*64)
        self.assertEqual(report["validation"], "structural_verified")
        self.assertEqual(report["candidate_printer_sections"], 1)
        self.assertEqual(report["variants"][0]["nozzle_mm"], "0.4")
        self.assertEqual([x["name"] for x in report["variants"][0]["filaments"]],
                         ["PLA", "PETG"])
        self.assertEqual(report["variants"][0]["process"]["inheritance"][1]["name"],
                         "*fast*")
        for fil in report["variants"][0]["filaments"]:
            check.check_preset_reference(fil, "filament")

    def test_invalid_second_filament_cannot_promote_model(self):
        sections = valid_ini_sections()
        del sections["filament:PETG"]
        report = prusa_ini.validate_model(sections, "printer_model:X1",
                                          ["0.4"], "bundles/example.ini", "a"*64)
        self.assertEqual(report["validation"], "index_only")
        self.assertEqual(report["variants"], [])
        self.assertTrue(any("missing" in s for s in report["warnings"]))

    def test_nozzle_variant_mismatch_blocks_printer(self):
        sections = valid_ini_sections()
        sections["printer:*machine*"]["nozzle_diameter"] = "0.6"
        report = prusa_ini.validate_model(sections, "printer_model:X1",
                                          ["0.4","0.6"], "bundles/example.ini","a"*64)
        self.assertEqual(report["validation"], "index_only")

    def test_wrong_ini_section_chain_cannot_pass_registry_qa(self):
        sections = valid_ini_sections()
        report = prusa_ini.validate_model(sections, "printer_model:X1",
                                          ["0.4"], "example.ini", "b"*64)
        good = report["variants"][0]["process"]
        check.check_preset_reference(good, "process")
        tampered = {**good, "section": "filament:Standard"}
        with self.assertRaisesRegex(ValueError, "typed section"):
            check.check_preset_reference(tampered, "process")

    def test_malformed_default_material_list(self):
        with self.assertRaises(prusa_ini.InvalidIniProfile):
            prusa_ini.declared_filaments("PLA,,PETG")


class NozzleCombinations(unittest.TestCase):
    def test_source_profile_digests_are_portable_across_line_endings(self):
        self.assertEqual(build.sha(b"{\\\"nozzle\\\": 0.4}\n"),
                         build.sha(b"{\\\"nozzle\\\": 0.4}\r\n"))
        self.assertNotEqual(build.sha(b"{\\\"nozzle\\\": 0.4}\n"),
                            build.sha(b"{\\\"nozzle\\\": 0.6}\n"))

    def test_nonuniform_machine_has_distinct_variant(self):
        self.assertEqual(build.diameters("0.4;0.4+0.6;0.2"),
                         ["0.2", "0.4", "0.4+0.6"])
        self.assertEqual(build.diameters(["0.4","0.6","0.4"]),["0.4","0.6"])
        self.assertEqual(build.nozzle_sort_key("0.4+0.6"),(0.4,0.6))
        for invalid in ("0.4+abc", "0.4+", "0.4+20", "0.4++0.6"):
            with self.assertRaises(build.InvalidProfile):
                build.normalized_nozzle_token(invalid)

    def test_compound_variant_requires_all_physical_nozzles(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            fixtures={
                "model.json":{"type":"machine_model","name":"Printer X",
                              "nozzle_diameter":"0.4;0.4+0.6"},
                "base.json":{"type":"machine","name":"base","printer_model":"Printer X",
                             "default_print_profile":"Standard",
                             "default_filament_profile":["PLA"]},
                "mixed.json":{"type":"machine","name":"Printer X mixed",
                              "inherits":"base","printer_variant":"0.4+0.6",
                              "nozzle_diameter":["0.4","0.6","0.4","0.6"]},
                "print.json":{"type":"process","name":"Standard"},
                "filament.json":{"type":"filament","name":"PLA"},
            }
            for name, data in fixtures.items():
                (root/name).write_text(json.dumps(data),encoding="utf-8")
            maps={"machine":{"base":"base.json","mixed":"mixed.json"},
                  "process":{"Standard":"print.json"},
                  "filament":{"PLA":"filament.json"},"machine_cache":{}}
            found=build.validation_for_model(root,{"name":"Printer X","sub_path":"model.json"}, {}, maps)
            self.assertEqual([v["nozzle_mm"] for v in found["variants"]],["0.4+0.6"])
            fixtures["mixed.json"]["nozzle_diameter"]=["0.4","0.4"]
            (root/"mixed.json").write_text(json.dumps(fixtures["mixed.json"]),encoding="utf-8")
            maps["machine_cache"]={}
            wrong=build.validation_for_model(root,{"name":"Printer X","sub_path":"model.json"}, {}, maps)
            self.assertEqual(wrong["validation"],"index_only")


class PinnedSourceSamples(unittest.TestCase):
    def test_prusa_research_mk4_is_now_structurally_verified(self):
        root=Path(__file__).resolve().parents[1]/"data"/"makers"
        models=json.loads((root/"prusa-research.json").read_text(encoding="utf8"))["models"]
        mk4=next(x for x in models if x["model"]=="MK4")
        self.assertEqual(mk4["preferred_profile"]["publisher"], "prusa_official")
        self.assertTrue(mk4["preferred_by_nozzle"])
        self.assertFalse(mk4["conversion_tested"])

    def test_snapmaker_u1_pins_official_before_orca_and_has_mixed_variant(self):
        root=Path(__file__).resolve().parents[1]/"data"/"makers"
        models=json.loads((root/"snapmaker.json").read_text(encoding="utf8"))["models"]
        u1=next(x for x in models if x["id"]=="snapmaker-u1")
        sources={x["publisher"]:x for x in u1["profile_sources"]}
        self.assertIn("snapmaker",sources)
        self.assertIn("orca",sources)
        # Current pinned official bundle is incomplete for U1; no false claim.
        self.assertEqual(sources["snapmaker"]["validation"],"index_only")
        self.assertEqual(u1["preferred_by_nozzle"]["0.4+0.6"]["publisher"],"orca")
        self.assertFalse(u1["conversion_tested"])
        variant=next(v for v in sources["orca"]["variants"] if v["nozzle_mm"]=="0.4+0.6")
        chain=variant["machine"]["inheritance"]
        self.assertGreater(len(chain),1)
        self.assertEqual(chain[-1]["name"],variant["machine"]["name"])
        for inherited in chain:
            self.assertTrue(inherited["path"].startswith("machine/"))
        check.check_preset_reference(variant["machine"],"machine")


    def test_flashforge_uses_vendor_when_complete_and_orca_for_creator5(self):
        root=Path(__file__).resolve().parents[1]/"data"/"makers"
        models=json.loads((root/"flashforge.json").read_text(encoding="utf8"))["models"]
        ad5x=next(x for x in models if x["model"]=="Flashforge AD5X")
        creator=next(x for x in models if x["model"]=="Flashforge Creator 5")
        self.assertEqual(ad5x["preferred_profile"]["publisher"],"flashforge")
        self.assertEqual(creator["preferred_profile"]["publisher"],"orca")
        self.assertFalse(any(x["publisher"]=="flashforge" for x in creator["profile_sources"]))
        self.assertFalse(ad5x["conversion_tested"] or creator["conversion_tested"])

    def test_all_eight_source_revisions_are_pinned(self):
        root=Path(__file__).resolve().parents[1]
        locked=json.loads((root/"sources.lock.json").read_text(encoding="utf8"))["sources"]
        self.assertEqual(len(locked),8)
        for publisher in ("flashforge","snapmaker","prusa_official","prusa_community"):
            self.assertIn(publisher,[x["key"] for x in locked])
        summary=json.loads((root/"data"/"index.json").read_text(encoding="utf8"))
        self.assertGreaterEqual(summary["structurally_verified"],400)
        self.assertEqual(summary["conversion_tested"],0)


if __name__ == "__main__":
    unittest.main()
