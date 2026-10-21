"""Independent single-EXE packer contract tests (no Windows tools required)."""
import hashlib
import importlib.util
import json
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

SCRIPT = Path(__file__).resolve().parents[1] / "scripts" / "package-prusa-sidecar.py"
SPEC = importlib.util.spec_from_file_location("prusa_sidecar_packaging", SCRIPT)
packager = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(packager)


class WindowsOneFileTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.dist = self.root / "dist"
        self.dist.mkdir()
        self.bin = self.root / "bin"
        self.bin.mkdir()
        self.engine = b"MZ" + b"private-python-runtime" * 300
        self.license = b"MIT License\nPermission is hereby granted, free of charge"
        self.sha = hashlib.sha256(self.engine).hexdigest()
        (self.dist / "slicebridge-prusa-engine.exe").write_bytes(self.engine)
        (self.dist / "panda2prusa-LICENSE.txt").write_bytes(self.license)
        (self.dist / "manifest.json").write_text(json.dumps({
            "platform": "windows", "sha256": self.sha
        }))
        self.app = self.bin / "SliceBridge.exe"
        self.original = b"MZ" + b"native-go-wails-content" * 100
        self.app.write_bytes(self.original)

    def pack(self):
        args = SimpleNamespace(bin_dir=self.bin, dist_dir=self.dist)
        with patch.object(packager, "host_platform", return_value="windows"):
            return packager.cmd_pack_windows(args)

    def test_one_exe_contains_engine_and_mit_license(self):
        self.assertEqual(self.pack(), 0)
        self.assertEqual([p.name for p in self.bin.iterdir()], ["SliceBridge.exe"])
        blob = self.app.read_bytes()
        self.assertTrue(blob.startswith(self.original))
        details = packager.inspect_windows_bundle(self.app, self.sha)
        self.assertEqual(details["engine"], self.engine)
        self.assertLess(details["payloadBytes"], packager.PACK_MAX_BYTES)
        args = SimpleNamespace(bin_dir=self.bin, dist_dir=self.dist, platform="windows",
                               smoke=False, no_codesign=False)
        self.assertEqual(packager.cmd_verify(args), 0)

    def test_reject_duplicate_packing(self):
        self.pack()
        with self.assertRaises(packager.CheckFailed):
            self.pack()

    def test_reject_tampered_payload_hash(self):
        self.pack()
        data = bytearray(self.app.read_bytes())
        data[-1] ^= 0xFF
        self.app.write_bytes(data)
        with self.assertRaisesRegex(packager.CheckFailed, "SHA-256 mismatch"):
            packager.inspect_windows_bundle(self.app)

    def test_reject_missing_license(self):
        (self.dist / "panda2prusa-LICENSE.txt").unlink()
        with self.assertRaises(packager.Failure):
            self.pack()

    def test_refuse_symlinked_destination(self):
        self.app.unlink()
        try:
            self.app.symlink_to(self.bin / "target")
        except OSError:
            if __import__("os").name == "nt":
                self.skipTest("Windows runner has no symlink privilege")
            raise
        with self.assertRaises(packager.CheckFailed):
            self.pack()


if __name__ == "__main__":
    unittest.main()
