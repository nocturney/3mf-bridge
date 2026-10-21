"""Tests for scripts/package-prusa-sidecar.py: staging, verification and refusal paths (stdlib unittest).

These tests exercise file placement, hashing, license gating and symlink refusal only.
Their inputs are stand-in bytes written into temp folders, not a built engine. The real
build and the engine behaviour are covered by test_conversion_contract.py.
"""
import contextlib
import hashlib
import importlib.util
import io
import json
import os
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

SCRIPT = Path(__file__).resolve().parents[2] / "scripts" / "package-prusa-sidecar.py"
REQUIREMENTS = SCRIPT.with_name("prusa-sidecar-requirements.txt")
MIT_TEXT = ("MIT License\n\nCopyright (c) 2026 Test Author\n\n"
            "Permission is hereby granted, free of charge, to any person obtaining a copy\n")
GPL_TEXT = "GNU GENERAL PUBLIC LICENSE\nVersion 3, 29 June 2007\n"

_spec = importlib.util.spec_from_file_location("package_prusa_sidecar", SCRIPT)
pps = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(pps)


def quiet_main(argv):
    """Run the CLI, returning (exit status, stdout, stderr)."""
    out, err = io.StringIO(), io.StringIO()
    with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
        # Packing only runs on Windows in production. These fixture tests are
        # platform-independent and use fake MZ files, not real executables.
        if argv and argv[0] == "pack-windows":
            with patch.object(pps, "host_platform", return_value="windows"):
                status = pps.main(argv)
        else:
            status = pps.main(argv)
    return status, out.getvalue(), err.getvalue()


def sha(data):
    return hashlib.sha256(data).hexdigest()


class PackagingFixture(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="panda-pkg-test-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.dist = self.root / "dist"
        self.dist.mkdir()
        self.sidecar_bytes = b"MZSTAND-IN SIDECAR BYTES (test input, not an engine)"
        self.license_text = MIT_TEXT

    def make_dist(self, platform, sidecar=None, license_text=None, manifest_sha=None):
        sidecar = self.sidecar_bytes if sidecar is None else sidecar
        license_text = self.license_text if license_text is None else license_text
        (self.dist / pps.exe_name(platform)).write_bytes(sidecar)
        (self.dist / pps.LICENSE_NAME).write_text(license_text, encoding="utf-8")
        manifest = {"name": pps.SIDECAR, "platform": platform,
                    "sha256": manifest_sha or sha(self.sidecar_bytes)}
        (self.dist / pps.MANIFEST_NAME).write_text(json.dumps(manifest), encoding="utf-8")

    def make_windows_bin(self):
        bin_dir = self.root / "native" / "build" / "bin"
        bin_dir.mkdir(parents=True)
        (bin_dir / "SliceBridge.exe").write_bytes(b"MZWAILS APP EXE")
        return bin_dir

    def make_macos_bin(self):
        bin_dir = self.root / "native" / "build" / "bin"
        (bin_dir / "SliceBridge.app" / "Contents" / "MacOS").mkdir(parents=True)
        (bin_dir / "SliceBridge.app" / "Contents" / "MacOS" / "SliceBridge").write_bytes(b"WAILS APP")
        return bin_dir


class StageAndVerify(PackagingFixture):
    def test_windows_packs_engine_and_license_inside_one_app_exe(self):
        self.make_dist("windows")
        bin_dir = self.make_windows_bin()
        status, _, err = quiet_main(["pack-windows", "--bin-dir", str(bin_dir),
                                     "--dist-dir", str(self.dist)])
        self.assertEqual(status, 0, err)
        self.assertEqual([p.name for p in bin_dir.iterdir()], ["SliceBridge.exe"])
        self.assertTrue((bin_dir / "SliceBridge.exe").read_bytes().startswith(b"MZWAILS APP EXE"))
        info = pps.inspect_windows_bundle(bin_dir / "SliceBridge.exe", sha(self.sidecar_bytes))
        self.assertEqual(info["engine"], self.sidecar_bytes)
        self.assertFalse(list(bin_dir.glob("*.packing")), "temp file left behind")

    def test_macos_stage_targets_app_resources_only(self):
        self.make_dist("macos")
        bin_dir = self.make_macos_bin()
        status, _, err = quiet_main(["stage", "--platform", "macos", "--bin-dir", str(bin_dir),
                                     "--dist-dir", str(self.dist), "--no-codesign"])
        self.assertEqual(status, 0, err)
        resources = bin_dir / "SliceBridge.app" / "Contents" / "Resources"
        self.assertEqual((resources / "slicebridge-prusa-engine").read_bytes(), self.sidecar_bytes)
        self.assertEqual((resources / pps.LICENSE_NAME).read_text(encoding="utf-8"), MIT_TEXT)
        self.assertFalse((bin_dir / "slicebridge-prusa-engine").exists())
        if os.name == "posix":
            self.assertTrue(os.access(str(resources / "slicebridge-prusa-engine"), os.X_OK))

    def test_verify_accepts_a_correct_embedded_windows_package(self):
        self.make_dist("windows")
        bin_dir = self.make_windows_bin()
        quiet_main(["pack-windows", "--bin-dir", str(bin_dir), "--dist-dir", str(self.dist)])
        status, out, err = quiet_main(["verify", "--platform", "windows", "--bin-dir", str(bin_dir),
                                       "--dist-dir", str(self.dist)])
        self.assertEqual(status, 0, err)
        self.assertEqual(json.loads(out)["engineSha256"], sha(self.sidecar_bytes))

    def test_verify_accepts_a_correct_macos_stage_without_seal_check(self):
        self.make_dist("macos")
        bin_dir = self.make_macos_bin()
        quiet_main(["stage", "--platform", "macos", "--bin-dir", str(bin_dir),
                    "--dist-dir", str(self.dist), "--no-codesign"])
        status, _, err = quiet_main(["verify", "--platform", "macos", "--bin-dir", str(bin_dir),
                                     "--dist-dir", str(self.dist), "--no-codesign"])
        self.assertEqual(status, 0, err)

    def test_verify_rejects_a_tampered_embedded_bundle(self):
        self.make_dist("windows")
        bin_dir = self.make_windows_bin()
        quiet_main(["pack-windows", "--bin-dir", str(bin_dir), "--dist-dir", str(self.dist)])
        app = bin_dir / "SliceBridge.exe"
        raw = bytearray(app.read_bytes())
        raw[-1] ^= 0xFF
        app.write_bytes(raw)
        status, _, err = quiet_main(["verify", "--platform", "windows", "--bin-dir", str(bin_dir),
                                     "--dist-dir", str(self.dist)])
        self.assertEqual(status, 1)
        self.assertIn("SHA-256 mismatch", err)

    def test_stage_refuses_a_sidecar_that_does_not_match_its_manifest(self):
        self.make_dist("windows", sidecar=b"changed after the build", manifest_sha=sha(self.sidecar_bytes))
        bin_dir = self.make_windows_bin()
        status, _, err = quiet_main(["pack-windows", "--bin-dir", str(bin_dir),
                                     "--dist-dir", str(self.dist)])
        self.assertEqual(status, 1)
        self.assertIn("manifest", err)
        self.assertFalse((bin_dir / "slicebridge-prusa-engine.exe").exists())

    def test_stage_refuses_a_license_that_is_not_mit(self):
        self.make_dist("windows", license_text=GPL_TEXT)
        bin_dir = self.make_windows_bin()
        status, _, err = quiet_main(["pack-windows", "--bin-dir", str(bin_dir),
                                     "--dist-dir", str(self.dist)])
        self.assertEqual(status, 1)
        self.assertIn("MIT", err)
        self.assertFalse((bin_dir / "slicebridge-prusa-engine.exe").exists())

    def test_stage_requires_the_wails_output_first(self):
        self.make_dist("windows")
        empty = self.root / "empty-bin"
        empty.mkdir()
        status, _, err = quiet_main(["pack-windows", "--bin-dir", str(empty),
                                     "--dist-dir", str(self.dist)])
        self.assertEqual(status, 2)
        self.assertIn("Wails app", err)

    def test_stage_refuses_a_manifest_for_the_other_platform(self):
        self.make_dist("macos")
        (self.dist / "slicebridge-prusa-engine.exe").write_bytes(self.sidecar_bytes)
        bin_dir = self.make_windows_bin()
        status, _, err = quiet_main(["pack-windows", "--bin-dir", str(bin_dir),
                                     "--dist-dir", str(self.dist)])
        self.assertEqual(status, 1)
        self.assertIn("built for macos", err)


class RefusesLinks(PackagingFixture):
    def test_windows_symlinked_app_is_refused_without_touching_target(self):
        self.make_dist("windows")
        bin_dir = self.make_windows_bin()
        outside = self.root / "outside.txt"
        outside.write_bytes(b"OUTSIDE")
        (bin_dir / "SliceBridge.exe").unlink()
        try:
            os.symlink(str(outside), str(bin_dir / "SliceBridge.exe"))
        except (OSError, NotImplementedError) as error:
            self.skipTest("symlinks unavailable on this host: %s" % error)
        status, _, err = quiet_main(["pack-windows", "--bin-dir", str(bin_dir),
                                     "--dist-dir", str(self.dist)])
        self.assertEqual(status, 1)
        self.assertIn("symlink", err)
        self.assertEqual(outside.read_bytes(), b"OUTSIDE")

    def test_macos_symlinked_resources_folder_is_refused(self):
        self.make_dist("macos")
        bin_dir = self.make_macos_bin()
        elsewhere = self.root / "elsewhere"
        elsewhere.mkdir()
        contents = bin_dir / "SliceBridge.app" / "Contents"
        try:
            os.symlink(str(elsewhere), str(contents / "Resources"))
        except (OSError, NotImplementedError) as error:
            self.skipTest("symlinks unavailable on this host: %s" % error)
        status, _, err = quiet_main(["stage", "--platform", "macos", "--bin-dir", str(bin_dir),
                                     "--dist-dir", str(self.dist), "--no-codesign"])
        self.assertEqual(status, 1)
        self.assertEqual(list(elsewhere.iterdir()), [])


class BuildGates(PackagingFixture):
    def test_build_writes_nothing_when_the_vendor_tree_is_missing(self):
        out = self.root / "out"
        status, _, err = quiet_main(["build", "--vendor-dir", str(self.root / "no-vendor"),
                                     "--output-dir", str(out)])
        self.assertEqual(status, 2)
        self.assertFalse(out.exists())

    def test_build_refuses_interpreters_older_than_3_11(self):
        if sys.version_info >= (3, 11):
            self.skipTest("this interpreter satisfies the 3.11 floor")
        status, _, err = quiet_main(["build", "--vendor-dir", str(self.root), "--output-dir",
                                     str(self.root / "out")])
        self.assertEqual(status, 2)
        self.assertIn("3.11", err)

    def test_vendor_license_must_be_mit(self):
        vendor = self.root / "vendor"
        (vendor / "panda2prusa").mkdir(parents=True)
        (vendor / "slicebridge_runner.py").write_text("print('x')\n", encoding="utf-8")
        (vendor / "LICENSE").write_text(GPL_TEXT, encoding="utf-8")
        with self.assertRaises(pps.CheckFailed):
            pps.check_vendor(vendor)
        (vendor / "LICENSE").write_text(MIT_TEXT, encoding="utf-8")
        pps.check_vendor(vendor)

    def test_vendor_missing_items_are_named(self):
        vendor = self.root / "partial"
        vendor.mkdir()
        (vendor / "LICENSE").write_text(MIT_TEXT, encoding="utf-8")
        with self.assertRaises(pps.Failure) as caught:
            pps.check_vendor(vendor)
        self.assertIn("slicebridge_runner.py", str(caught.exception))
        self.assertIn("panda2prusa", str(caught.exception))


class PinsAndCiText(unittest.TestCase):
    def test_pyinstaller_is_pinned_exactly_and_lxml_has_a_floor(self):
        self.assertEqual(pps.pinned_pyinstaller(), "6.22.3")
        text = REQUIREMENTS.read_text(encoding="utf-8")
        self.assertIn("lxml>=5", text)

    def test_relative_paths_resolve_from_the_repository_root(self):
        self.assertEqual(pps.resolve("native/build/bin"), pps.ROOT / "native" / "build" / "bin")
        absolute = Path(tempfile.gettempdir()).resolve()
        self.assertEqual(pps.resolve(str(absolute)), absolute)

    def test_ci_steps_cover_single_windows_exe_and_bundled_macos_app(self):
        status, out, _ = quiet_main(["print-ci-steps"])
        self.assertEqual(status, 0)
        self.assertIn("pack-windows --bin-dir native/build/bin", out)
        self.assertIn("stage --platform macos --bin-dir native/build/bin", out)
        self.assertIn("SliceBridge.exe", out)
        self.assertNotIn("Compress-Archive", out)
        self.assertNotIn("stage --platform windows", out)


if __name__ == "__main__":
    unittest.main()
