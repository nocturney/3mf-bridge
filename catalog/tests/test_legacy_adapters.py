"""Regression checks for catalogue-to-legacy-adapter boundary.

Tests use only a frozen vendor catalogue and fixture settings; never read or
write a person's installed printer profiles, application preferences or files.
"""
import copy
import json
import shutil
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "tools"))
import legacy_adapters


class ExistingAdapters(unittest.TestCase):
    def setUp(self):
        self.doc = json.loads((ROOT / "data" / "legacy-adapters.json").read_text(encoding="utf-8"))
        self.by_key = {x["legacy_target"]: x for x in self.doc["mappings"]}

    def test_exact_four_legacy_ids_and_catalog_only_default(self):
        status = legacy_adapters.validate_mapping(ROOT)
        self.assertEqual(status["legacy_adapter_count"], 4)
        self.assertEqual(status["catalog_only_model_count"], 599)
        self.assertEqual(status["conversion_tested"], 0)
        self.assertFalse(status["runtime_adapter_activated_by_catalog"])
        self.assertEqual(status["mapped_model_ids"], sorted([
            "snapmaker-u1", "flashforge-creator-5",
            "elegoo-centauri-carbon-2", "qidi-x-plus-5"
        ]))
        self.assertEqual(set(self.by_key), {"u1", "c5", "cc2", "qidi"})

    def test_source_nozzle_remap_and_mixed_variant_are_not_auto_enabled(self):
        c5 = self.by_key["c5"]
        self.assertEqual(c5["source_nozzle_remap_mm"], {"0.2": "0.25"})
        self.assertIn("0.25", c5["target_nozzles_mm"])
        self.assertNotIn("0.2", c5["target_nozzles_mm"])
        self.assertEqual(self.by_key["u1"]["builtin_fast_path_nozzle_mm"], "0.4")
        self.assertNotIn("0.4+0.6", self.by_key["u1"]["target_nozzles_mm"])
        for entry in self.doc["mappings"]:
            self.assertFalse(entry["catalog_activates_conversion"])
            self.assertFalse(entry["conversion_tested"])
            self.assertEqual(entry["profile_location_policy"], "installed_slicer_only")

    def test_read_only_settings_preview_preserves_all_preference_fields(self):
        sample = {
            "version": 1,
            "printers": {
                "u1": {"nozzle": "0.4", "slicer": "auto",
                       "filaments": {"PLA": "Custom PLA"}},
                "c5": {"nozzle": "0.25", "slicer": "Flash Studio Desktop",
                       "filaments": {"PETG": "PETG Special"}, "future_key": "unchanged"},
                "qidi": {"nozzle": "0.8", "filaments": {"ABS": "ABS Glass"}},
                "unknown_future": {"nozzle": "1.0", "extraneous": {"a": [1, 2]}},
            },
            "window": {"width": 1150, "height": 750},
            "future_setting": {"nested": True},
        }
        frozen = copy.deepcopy(sample)
        plan = legacy_adapters.preview_settings_migration(sample, self.doc)
        self.assertEqual(sample, frozen)
        self.assertEqual(plan["preserved_original_v1_settings"], frozen)
        self.assertFalse(plan["changes_written"])
        mapped = {entry["legacy_target"]: entry for entry in plan["mapped_printers"]}
        self.assertEqual(set(mapped), {"u1", "c5", "qidi"})
        self.assertEqual(mapped["c5"]["catalog_model_id"], "flashforge-creator-5")
        self.assertEqual(mapped["c5"]["original_preference"], frozen["printers"]["c5"])
        self.assertEqual(mapped["u1"]["original_preference"]["filaments"],
                         {"PLA": "Custom PLA"})
        self.assertEqual(plan["unknown_printers_preserved"],
                         {"unknown_future": frozen["printers"]["unknown_future"]})
        # Plan must hold deep independent copies even if caller edits later.
        sample["printers"]["c5"]["filaments"]["PETG"] = "changed"
        self.assertEqual(mapped["c5"]["original_preference"]["filaments"]["PETG"],
                         "PETG Special")

    def test_unsupported_settings_versions_and_bad_shapes_fail_closed(self):
        for sample in (None, [], {"version": 2}, {"version": 1, "printers": []},
                       {"version": 1, "printers": {"u1": "invalid"}}):
            with self.subTest(sample=sample):
                with self.assertRaises(ValueError):
                    legacy_adapters.preview_settings_migration(sample, self.doc)

    def test_tampered_binding_or_enabled_converter_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "data" / "makers").mkdir(parents=True)
            for item in self.doc["mappings"]:
                src = ROOT / "data" / item["maker_file"]
                dest = root / "data" / item["maker_file"]
                shutil.copyfile(src, dest)
            shutil.copyfile(ROOT / "data" / "index.json", root / "data" / "index.json")
            dest = root / "data" / "legacy-adapters.json"
            cases = [
                ("auto enable", ("u1", "catalog_activates_conversion", True)),
                ("unknown model", ("qidi", "model_id", "fabricated-model")),
                ("duplicate ID", ("cc2", "model_id", "snapmaker-u1")),
                ("missing nozzle", ("c5", "target_nozzles_mm", ["0.2", "0.4", "0.6", "0.8"])),
                ("silent remap change", ("c5", "source_nozzle_remap_mm", {})),
                ("mixed nozzle lie", ("u1", "target_nozzles_mm", ["0.2", "0.4", "0.4+0.6", "0.8"])),
            ]
            for name, (key, prop, invalid) in cases:
                with self.subTest(name=name):
                    altered = copy.deepcopy(self.doc)
                    item = next(x for x in altered["mappings"] if x["legacy_target"] == key)
                    item[prop] = invalid
                    dest.write_text(json.dumps(altered), encoding="utf-8")
                    with self.assertRaises(ValueError):
                        legacy_adapters.validate_mapping(root)

    def test_go_wails_contract_is_traceable_and_not_wired_yet(self):
        self.assertEqual(self.doc["source_settings_schema_version"], 1)
        self.assertEqual(self.doc["registration_policy"],
                         "metadata_only_never_auto_enable_catalogued_models")
        self.assertTrue(self.doc["preserve_saved_preferences"])
        self.assertEqual(self.doc["default_profile_acquisition"],
                         "read_local_slicer_installation_only")
        self.assertFalse(self.doc["profile_payloads_distributed"])


if __name__ == "__main__":
    unittest.main()
