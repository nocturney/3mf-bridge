"""Rights evidence and human-readable third-party credits regression gates."""
from __future__ import annotations

import json
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "tools"))
import check


class AttributionEvidence(unittest.TestCase):
    def setUp(self):
        self.root = Path(__file__).resolve().parents[1]

    def test_all_eight_sources_credited_and_no_rights_invented(self):
        sources = json.loads((self.root / "sources.lock.json").read_text(encoding="utf-8"))["sources"]
        ledger = check.check_attributions(self.root, {x["key"]: x for x in sources})
        found = {x["key"]: x for x in ledger["items"]}
        self.assertEqual(len(found), 8)
        expected_without_root = {"snapmaker", "prusa_community"}
        for key, item in found.items():
            if key in expected_without_root:
                self.assertEqual(item["repository_root_license_observed"],
                                 "no_repository_level_license_found")
            else:
                self.assertEqual(item["repository_root_license_observed"], "AGPL-3.0")
                self.assertEqual(len(item["repository_root_license_evidence"]), 1)
            self.assertFalse(item["third_party_profile_payloads_included"])
            self.assertFalse(item["third_party_profile_payloads_redistribution_cleared"])
            self.assertFalse(item["individual_profile_licensing_reviewed"])

    def test_human_notice_mentions_all_sources(self):
        notice = (self.root / "THIRD_PARTY_NOTICES.md").read_text(encoding="utf-8")
        sources = json.loads((self.root / "sources.lock.json").read_text(encoding="utf-8"))["sources"]
        for source in sources:
            self.assertIn("https://github.com/" + source["repository"], notice)

    def test_archive_includes_machine_readable_credits_only(self):
        from package import manifest_payload
        receipt, content = manifest_payload(self.root)
        self.assertIn("catalog/data/attribution.json", content)
        self.assertEqual(len(receipt["files"]), 90)
        rights = json.loads(content["catalog/data/attribution.json"])
        self.assertEqual(len(rights["items"]), 8)
        self.assertFalse(any(x["third_party_profile_payloads_included"] for x in rights["items"]))


if __name__ == "__main__":
    unittest.main()
