"""Offline UI snapshot regression and locally installed artwork reference policy."""
import json
import sys
import unittest
from pathlib import Path

CATALOG = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(CATALOG / "tools"))
import export_ui


class UIRegistryExportTests(unittest.TestCase):
    def test_compact_vetted_snapshot_without_third_party_images(self):
        content = export_ui.build_picker(CATALOG)
        doc = json.loads(content)
        self.assertEqual(doc["modelCount"], 603)
        self.assertEqual(doc["legacyAdapterCount"], 4)
        self.assertFalse(doc["thirdPartyProfilePayloadsIncluded"])
        self.assertEqual(len(doc["models"]), 603)
        self.assertEqual(len({model["id"] for model in doc["models"]}), 603)
        self.assertTrue(all(model["imagePath"] is None and model["logoPath"] is None
                            for model in doc["models"]))
        self.assertTrue(all(len(model["localCoverCandidates"]) <= 12
                            for model in doc["models"]))

    def test_committed_frontend_snapshot_reproducible_from_catalogue(self):
        checked = CATALOG.parent / "native" / "frontend" / "dist" / "printer-catalog.json"
        self.assertTrue(checked.is_file())
        self.assertEqual(export_ui.build_picker(CATALOG), checked.read_bytes().replace(b"\r\n", b"\n"))

    def test_fixtures_have_precise_local_cover_references(self):
        content = json.loads(export_ui.build_picker(CATALOG))
        mapped = {m["id"]: m for m in content["models"]}
        wanted = {
            "snapmaker-u1": "Snapmaker/Snapmaker U1_cover.png",
            "flashforge-creator-5": "Flashforge/Flashforge Creator 5_cover.png",
            "elegoo-centauri-carbon-2": "Elegoo/Elegoo Centauri Carbon 2_cover.png",
            "qidi-x-plus-5": "X 5 Series/X-Plus 5_cover.png",
        }
        for identifier, image in wanted.items():
            self.assertIn(image, mapped[identifier]["localCoverCandidates"])
        self.assertIn("bambu-lab-p1s", mapped)
        self.assertIsNone(mapped["bambu-lab-p1s"]["legacyTarget"])

    def test_no_absolute_paths_remote_urls_or_preset_bytes(self):
        catalog = json.loads(export_ui.build_picker(CATALOG))
        forbidden = ("://", "\\\\", "../", ".ini", ".gcode", ".3mf")
        for model in catalog["models"]:
            for relative in model["localCoverCandidates"]:
                self.assertTrue(relative.endswith("_cover.png"))
                self.assertEqual(relative.count("/"), 1)
                self.assertTrue(all(s not in relative for s in forbidden), relative)


if __name__ == "__main__":
    unittest.main()
