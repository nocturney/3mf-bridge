"""Metadata-only bundle packaging and upstream reference routing tests."""
from __future__ import annotations

import json
import hashlib
import sys
import tempfile
import unittest
import zipfile
from pathlib import Path

TOOLS = Path(__file__).resolve().parents[1] / "tools"
sys.path.insert(0, str(TOOLS))
import package
import attest_sources


class MetadataBundleTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.root = Path(__file__).resolve().parents[1]

    def test_reproducible_metadata_bundle_without_source_payloads(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            left = package.build_archive(self.root, root / "left")
            right = package.build_archive(self.root, root / "right")
            first, second = Path(left["file"]), Path(right["file"])
            self.assertEqual(first.name, second.name)
            self.assertEqual(first.read_bytes(), second.read_bytes())
            self.assertEqual(left["sha256"], hashlib.sha256(first.read_bytes()).hexdigest())
            self.assertEqual(left["sha256"], right["sha256"])
            self.assertEqual(left["models"], 603)
            manifest = package.validate_archive(first)
            self.assertEqual(manifest["conversion_tested_models"], 0)
            self.assertFalse(manifest["source_profile_payloads_included"])
            self.assertEqual(manifest["sources"], 8)
            with zipfile.ZipFile(first) as archive:
                self.assertTrue(all(n.startswith("catalog/") and n.endswith(".json")
                                    for n in archive.namelist()))
                self.assertFalse(any(n.endswith((".ini", ".3mf", ".gcode", ".stl"))
                                     for n in archive.namelist()))
                self.assertIn("catalog/data/makers/snapmaker.json", archive.namelist())
                self.assertIn("catalog/data/makers/flashforge.json", archive.namelist())
                self.assertIn("catalog/data/legacy-adapters.json", archive.namelist())
                self.assertEqual(json.loads(archive.read("catalog/data/legacy-adapters.json"))
                                 ["registration_policy"],
                                 "metadata_only_never_auto_enable_catalogued_models")
            self.assertIn(left["sha256"], (first.parent / (first.name + ".sha256")).read_text())
            with self.assertRaises(FileExistsError):
                package.build_archive(self.root, root / "left")

    def test_tampered_archive_is_rejected(self):
        with tempfile.TemporaryDirectory() as temp:
            result = package.build_archive(self.root, Path(temp) / "base")
            clean = Path(result["file"])
            tampered = Path(temp) / "tampered.zip"
            with zipfile.ZipFile(clean) as source, zipfile.ZipFile(tampered, "w") as bad:
                for info in source.infolist():
                    contents = source.read(info.filename)
                    if info.filename == "catalog/data/index.json":
                        contents = contents.replace(b'"models": 603', b'"models": 604', 1)
                    bad.writestr(info, contents)
            with self.assertRaisesRegex(ValueError, "checksum mismatch"):
                package.validate_archive(tampered)

    def test_source_reference_routing_and_traversal(self):
        self.assertEqual(
            attest_sources.profile_source_path("resources/profiles/Snapmaker.json",
                                               "machine/Snapmaker U1.json"),
            "resources/profiles/Snapmaker/machine/Snapmaker U1.json",
        )
        self.assertEqual(
            attest_sources.profile_source_path("live/PrusaResearch/1.14.2.ini",
                                               "live/PrusaResearch/1.14.2.ini"),
            "live/PrusaResearch/1.14.2.ini",
        )
        for value in ("../../secrets.txt", "/root", "C:/Windows/config"):
            with self.assertRaises(ValueError):
                attest_sources.profile_source_path("resources/profiles/BBL.json", value)
        with self.assertRaises(ValueError):
            attest_sources.profile_source_path("live/PrusaResearch/1.14.2.ini",
                                               "live/PrusaResearch/wrong.ini")


if __name__ == "__main__":
    unittest.main()
