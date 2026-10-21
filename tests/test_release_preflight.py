"""Offline tests for the preflight tool. No network or real binaries required."""
from __future__ import annotations
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
import zipfile

SCRIPT = Path(__file__).resolve().parents[1] / "tools" / "release_preflight.py"


class PreflightTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="slicebridge-release-test-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.repo = self.root / "repo"
        self.repo.mkdir()
        self.git(self.repo, "init", "-q")
        self.git(self.repo, "config", "user.name", "Release Audit Test")
        self.git(self.repo, "config", "user.email", "test@example.invalid")
        (self.repo / "README.md").write_text("# Synthetic project\n", encoding="utf-8")
        (self.repo / "LICENSE").write_text("GNU GENERAL PUBLIC LICENSE\nVersion 3\n", encoding="utf-8")
        (self.repo / "package-lock.json").write_text(json.dumps({
            "lockfileVersion": 3,
            "packages": {"": {"name": "test", "license": "GPL-3.0-only"},
                         "node_modules/example": {"version": "1.2.3", "license": "MIT"},
                         "node_modules/@scope/tool": {"version": "2.3.4", "license": "Apache-2.0"}}
        }), encoding="utf-8")
        (self.repo / "go.mod").write_text("module example.invalid/test\n\ngo 1.25\nrequire github.com/test/example v1.2.3\n")
        self.commit()

    def git(self, path, *args):
        result = subprocess.run(["git", "-C", str(path), *args], capture_output=True, text=True)
        if result.returncode:
            self.fail("git " + " ".join(args) + ": " + result.stderr)
        return result.stdout.strip()

    def commit(self):
        self.git(self.repo, "add", ".")
        self.git(self.repo, "commit", "-m", "Fixture", "-q")
        self.sha = self.git(self.repo, "rev-parse", "HEAD")

    def run_preflight(self, out, sha=None, assets=None, expected=0):
        args = [sys.executable, str(SCRIPT), "--repo", str(self.repo), "--out", str(out),
                "--source-sha", sha or self.sha]
        if assets is not None:
            args += ["--assets-dir", str(assets)]
        result = subprocess.run(args, capture_output=True, text=True)
        self.assertEqual(result.returncode, expected, result.stdout + "\n" + result.stderr)
        return result

    def test_deterministic_source_archive_and_bom(self):
        out1, out2 = self.root / "out1", self.root / "out2"
        self.run_preflight(out1)
        self.run_preflight(out2)
        arc = f"slicebridge-source-{self.sha[:12]}.zip"
        self.assertEqual(hashlib.sha256((out1 / arc).read_bytes()).digest(),
                         hashlib.sha256((out2 / arc).read_bytes()).digest())
        with zipfile.ZipFile(out1 / arc) as z:
            names = z.namelist()
            self.assertIn(f"slicebridge-source-{self.sha[:12]}/README.md", names)
            self.assertIn(f"slicebridge-source-{self.sha[:12]}/LICENSE", names)
            self.assertIsNone(z.testzip())
        bom = json.loads((out1 / "manifest-sbom.cdx.json").read_text())
        self.assertEqual(bom["bomFormat"], "CycloneDX")
        self.assertEqual(len(bom["components"]), 3)
        self.assertIn("pkg:npm/%40scope/tool@2.3.4", [c["purl"] for c in bom["components"]])
        self.assertIn("pkg:npm/example@1.2.3", [c["purl"] for c in bom["components"]])
        self.assertIn("pkg:golang/github.com/test/example@v1.2.3", [c["purl"] for c in bom["components"]])
        report = json.loads((out1 / "release-evidence.json").read_text())
        self.assertEqual(report["sourceCommit"], self.sha)
        self.assertFalse(report["releaseGates"]["approved"])
        for line in (out1 / "SHA256SUMS.txt").read_text().splitlines():
            h, name = line.split("  ", 1)
            self.assertEqual(h, hashlib.sha256((out1 / name).read_bytes()).hexdigest())

    def test_requires_immutable_commit(self):
        self.run_preflight(self.root / "wrong", sha="0" * 40, expected=1)
        self.run_preflight(self.root / "invalid", sha="main", expected=1)

    def test_refuses_dirty_tracked_files(self):
        (self.repo / "README.md").write_text("modified", encoding="utf-8")
        self.run_preflight(self.root / "dirty", expected=1)

    def test_archive_validation_rejects_path_traversal(self):
        asset_dir = self.root / "assets"
        asset_dir.mkdir()
        with zipfile.ZipFile(asset_dir / "unsafe.zip", "w") as z:
            z.writestr("../outside.txt", "hello")
        self.run_preflight(self.root / "unsafe", assets=asset_dir, expected=1)

    def test_draft_asset_hash_does_not_redistribute_binary(self):
        asset_dir = self.root / "draft"
        asset_dir.mkdir()
        with zipfile.ZipFile(asset_dir / "test-draft.zip", "w") as z:
            z.writestr("bin/README.txt", "Synthetic binary placeholder")
        out = self.root / "evidence"
        self.run_preflight(out, assets=asset_dir)
        report = json.loads((out / "release-evidence.json").read_text())
        self.assertEqual(len(report["draftAssets"]), 1)
        self.assertEqual(report["draftAssets"][0]["zipCrcPass"], True)
        self.assertFalse((out / "test-draft.zip").exists())
        self.assertTrue((out / "DRAFT_ASSET_SHA256SUMS.txt").exists())

    def test_recursive_pinned_submodule_source(self):
        child = self.root / "upstream"
        child.mkdir()
        self.git(child, "init", "-q")
        self.git(child, "config", "user.name", "Upstream Fixture")
        self.git(child, "config", "user.email", "upstream@example.invalid")
        (child / "LICENSE").write_text("Synthetic GPL test fixture")
        self.git(child, "add", ".")
        self.git(child, "commit", "-qm", "Initial")
        self.git(self.repo, "-c", "protocol.file.allow=always", "submodule", "add", "-q",
                 str(child), "third_party/bl2std")
        self.commit()
        out = self.root / "submodule-out"
        self.run_preflight(out)
        archive = out / f"slicebridge-source-{self.sha[:12]}.zip"
        with zipfile.ZipFile(archive) as z:
            self.assertIn(f"slicebridge-source-{self.sha[:12]}/third_party/bl2std/LICENSE",
                          z.namelist())
        report = json.loads((out / "release-evidence.json").read_text())
        self.assertTrue(any(x["path"] == "third_party/bl2std/" for x in report["pinnedSourceTrees"]))


if __name__ == "__main__":
    unittest.main()
