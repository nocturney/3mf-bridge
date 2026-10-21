"""Fast fixture and committed-snapshot tests for the printer catalog."""
from __future__ import annotations

import json
import sys
import tempfile
import unittest
from pathlib import Path

TOOLS = Path(__file__).resolve().parents[1] / "tools"
sys.path.insert(0, str(TOOLS))
import build
import check


class CatalogFixtures(unittest.TestCase):
    def test_untrusted_paths_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root/"good.json").write_text("{}",encoding="utf-8")
            self.assertEqual(build.read_json(root,"good.json")[0],{})
            for path in ("../secret.json", "/tmp/secret.json", "C:/system",r"..\secret.json",
                         "", "abc/../../secret.json"):
                with self.subTest(path=path):
                    with self.assertRaises(build.InvalidProfile):
                        build.safe_path(root,path)

    def test_json_snapshot_checksums_ignore_only_checkout_newlines(self):
        self.assertEqual(check.snapshot_digest(b'{"ok":1}\n'),
                         check.snapshot_digest(b'{"ok":1}\r\n'))
        self.assertNotEqual(check.snapshot_digest(b'{"ok":1}\n'),
                            check.snapshot_digest(b'{"ok":2}\n'))

    def test_nozzle_dimensions_are_checked(self):
        self.assertEqual(build.diameters("0.4;0.2;0.4;0.6"), ["0.2","0.4","0.6"])
        for raw in ("foo", "0", "40", "-0.2"):
            with self.subTest(raw=raw):
                with self.assertRaises(build.InvalidProfile):
                    build.diameters(raw)

    def test_inheritance_cannot_cycle(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for name, parent in (("a","b"),("b","a")):
                (root/(name+".json")).write_text(
                    json.dumps({"type":"machine","inherits":parent}),encoding="utf-8"
                )
            with self.assertRaisesRegex(build.InvalidProfile, "cycle"):
                build.resolve_machine("a",root,{"a":"a.json","b":"b.json"},{})

    def test_fixture_profile_must_link_real_machine_process_and_filament(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            payloads = {
                "model.json":{"type":"machine_model","name":"Sample X1",
                              "machine_tech":"FFF","nozzle_diameter":"0.4;0.6"},
                "base.json":{"type":"machine","name":"base",
                             "printer_model":"Sample X1","default_print_profile":"Standard",
                             "default_filament_profile":["Generic PLA"]},
                "machine.json":{"type":"machine","name":"X1 0.4","inherits":"base",
                                "nozzle_diameter":["0.4"]},
                "process.json":{"type":"process","name":"Standard"},
                "filament.json":{"type":"filament","name":"Generic PLA"},
            }
            for filename, obj in payloads.items():
                (root/filename).write_text(json.dumps(obj), encoding="utf-8")
            maps = {"machine":{"base":"base.json","X1 0.4":"machine.json"},
                    "process":{"Standard":"process.json"},
                    "filament":{"Generic PLA":"filament.json"},"machine_cache":{}}
            original = build.validation_for_model(root,{"name":"Sample X1","sub_path":"model.json"},{},maps)
            self.assertEqual(original["validation"],"structural_verified")
            self.assertEqual(len(original["variants"]),1)
            self.assertEqual(original["variants"][0]["nozzle_mm"],"0.4")
            # A missing filament is a hard block: machine cannot be advertised ready.
            maps["filament"].clear()
            maps["machine_cache"].clear()
            blocked = build.validation_for_model(root,{"name":"Sample X1","sub_path":"model.json"},{},maps)
            self.assertEqual(blocked["validation"],"index_only")
            self.assertEqual(blocked["variants"],[])

    def test_official_precedence_only_when_verified(self):
        def candidate(provider, publisher, verified):
            return {"provider":provider, "publisher":publisher, "repository":publisher,
                    "commit":"1"*40, "manifest":{"path":publisher+".json","sha256":"a"*64},
                    "model":{"path":publisher+".json","sha256":"a"*64},
                    "validation":"structural_verified" if verified else "index_only",
                    "nozzles_mm":["0.4"],
                    "variants":[{"nozzle_mm":"0.4","machine":{"name":"machine"},
                                 "process":{"name":"process"},"filament":{"name":"filament"}}]
                                if verified else []}
        def produce(vendor_ok):
            incoming=[
                {"manufacturer":"QIDI","name":"X-Plus 5",
                 "candidate":candidate("orca","orca",True)},
                {"manufacturer":"QIDI","name":"Qidi X-Plus 5",
                 "candidate":candidate("manufacturer","qidi",vendor_ok)},
                {"manufacturer":"QIDI","name":"X-Plus 5",
                 "candidate":candidate("prusa","prusa_community",False)},
            ]
            indexed, quarantine = build.combine(incoming)
            self.assertFalse(quarantine)
            self.assertEqual(len(indexed),1)
            return indexed[0]
        self.assertEqual(produce(True)["preferred_profile"]["publisher"],"qidi")
        self.assertEqual(produce(False)["preferred_profile"]["publisher"],"orca")

    def test_nozzle_specific_vendor_then_orca_fallback(self):
        def candidate(publisher, provider, nozzles):
            return {"publisher":publisher,"provider":provider,"repository":publisher,
                    "commit":"1"*40,"manifest":{"path":publisher+".json","sha256":"a"*64},
                    "model":{"path":publisher+".json","sha256":"a"*64},
                    "validation":"structural_verified","nozzles_mm":nozzles,
                    "variants":[{"nozzle_mm":n,"machine":{"name":publisher+"-machine"},
                                 "process":{"name":publisher+"-process"},
                                 "filament":{"name":publisher+"-filament"}} for n in nozzles]}
        records, quarantine = build.combine([
            {"manufacturer":"QIDI","name":"X-Plus 5",
             "candidate":candidate("qidi","manufacturer",["0.4"])},
            {"manufacturer":"QIDI","name":"X-Plus 5",
             "candidate":candidate("orca","orca",["0.4","0.6"])}])
        self.assertFalse(quarantine)
        preferred = records[0]["preferred_by_nozzle"]
        self.assertEqual(preferred["0.4"]["publisher"],"qidi")
        self.assertEqual(preferred["0.6"]["publisher"],"orca")

    def test_release_dates_are_not_fabricated(self):
        items, _ = build.combine([
            {"manufacturer":"Bambu Lab","name":"Bambu Lab P1S",
             "candidate":{"provider":"orca","publisher":"orca","repository":"r",
                          "commit":"1"*40,"manifest":{"path":"BBL.json"},"validation":"index_only",
                          "variants":[],"nozzles_mm":["0.4"]}},
        ])
        self.assertIsNone(items[0]["release_date"])
        self.assertFalse(items[0]["conversion_tested"])

    def test_cited_release_dates_survive_rebuild(self):
        url = "https://www.example.com/press-release"
        with tempfile.TemporaryDirectory() as tmp:
            makers = Path(tmp)
            (makers / "x.json").write_text(json.dumps({"models": [
                {"id": "acme-one", "release_date": "2024-01-02", "release_date_evidence": url},
                {"id": "acme-two", "release_date": None, "release_date_evidence": None},
            ]}), encoding="utf-8")
            models = [{"id": "acme-one", "release_date": None, "release_date_evidence": None},
                      {"id": "acme-two", "release_date": None, "release_date_evidence": None}]
            build.carry_release_evidence(models, makers)
        self.assertEqual(models[0]["release_date"], "2024-01-02")
        self.assertEqual(models[0]["release_date_evidence"], url)
        self.assertIsNone(models[1]["release_date"])


class SavedSnapshot(unittest.TestCase):
    def test_committed_registry_integrity(self):
        root = Path(__file__).resolve().parents[1]
        result = check.check_registry(root)
        self.assertGreater(result["models"], 450)
        self.assertGreater(result["structurally_verified"], 200)
        self.assertGreater(result["manufacturers"], 30)

    def test_prusa_blog_release_dates_pinned_to_exact_identities(self):
        prusa = json.loads((Path(__file__).resolve().parents[1]/"data"/"makers"/"prusa.json")
                           .read_text(encoding="utf-8"))
        dated = {m["id"]: (m["release_date"], m["release_date_evidence"])
                 for m in prusa["models"] if m["release_date"]}
        self.assertEqual(dated["prusa-prusa-mini"],
                         ("2019-10-12", "https://blog.prusa3d.com/original-prusa-mini-is-here-smart-and-compact-3d-printer_30887/"))
        self.assertEqual(dated["prusa-prusa-mk4s"],
                         ("2024-08-12", "https://blog.prusa3d.com/the-original-prusa-mk4s-is-here_100605/"))
        self.assertEqual(dated["prusa-prusa-xl"],
                         ("2021-11-18", "https://blog.prusa3d.com/original-prusa-xl-first-look_58312/"))
        # Profile groups shared with base models must not carry a "+" variant date.
        profile_groups = json.loads((Path(__file__).resolve().parents[1]/"data"/"makers"/"prusa-research.json")
                                    .read_text(encoding="utf-8"))
        shared = {m["id"]: m["release_date"] for m in profile_groups["models"]
                  if m["id"] in ("prusa-research-i3-mk3s-mk3s", "prusa-research-mini-mini")}
        self.assertEqual(shared, {"prusa-research-i3-mk3s-mk3s": None, "prusa-research-mini-mini": None})

    def test_existing_target_machines_have_official_profile_candidates(self):
        root = Path(__file__).resolve().parents[1] / "data"/"makers"
        for brand, model, publisher in (
            ("bambu-lab","P1S","bambu"),
            ("qidi","X-Plus 5","qidi"),
            ("elegoo","Centauri Carbon 2","elegoo"),
        ):
            doc = json.loads((root/(brand+".json")).read_text(encoding="utf-8"))
            matches = [m for m in doc["models"] if m["model"] == model]
            self.assertEqual(len(matches),1)
            self.assertEqual(matches[0]["preferred_profile"]["publisher"],publisher)
            self.assertTrue(matches[0]["preferred_profile"]["nozzles_mm"])


if __name__ == "__main__":
    unittest.main()
