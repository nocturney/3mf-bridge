#!/usr/bin/env python3
"""Build browser-store submission ZIPs from the already-verified extension folders.

These are PUBLISHER-ONLY archives, not signed browser installs or consumer downloads.
"""
from __future__ import annotations

import hashlib
import json
from pathlib import Path
import zipfile

ROOT = Path(__file__).resolve().parents[1]
FILES = (
    "manifest.json", "content.js", "popup.html", "popup.css", "popup.js",
    "icon-16.png", "icon-32.png", "icon-48.png", "icon-128.png",
)

def main() -> None:
    output = ROOT / "dist" / "store-submissions"
    output.mkdir(parents=True, exist_ok=True)
    for kind in ("chromium", "firefox"):
        source = ROOT / "dist" / "extensions" / kind
        if not source.is_dir():
            raise SystemExit("Run node browser-extension/build.cjs first")
        actual = {p.name for p in source.iterdir()}
        if actual != set(FILES) or any(not (source / name).is_file() for name in FILES):
            raise SystemExit(f"Unexpected or missing store resources in {kind}: {actual}")
        manifest = json.loads((source / "manifest.json").read_text(encoding="utf-8"))
        if manifest.get("manifest_version") != 3 or manifest.get("permissions") != ["storage"]:
            raise SystemExit("Unexpected extension permission or manifest")
        for entry in manifest.get("content_scripts", []):
            if set(entry.get("matches", [])) != {
                "https://makerworld.com/*", "https://*.makerworld.com/*"
            }:
                raise SystemExit("Extension must only inject into MakerWorld")
        version = manifest["version"]
        filename = f"SliceBridge-MakerWorld-{kind}-{version}-STORE-SUBMISSION.zip"
        target = output / filename
        if target.exists():
            raise SystemExit(f"Refusing to overwrite existing submission: {target}")
        with zipfile.ZipFile(target, "w") as archive:
            for name in sorted(FILES):
                info = zipfile.ZipInfo(name, date_time=(1980, 1, 1, 0, 0, 0))
                info.compress_type = zipfile.ZIP_STORED
                info.external_attr = 0o644 << 16
                archive.writestr(info, (source / name).read_bytes())
        with zipfile.ZipFile(target) as archive:
            if archive.testzip() is not None or set(archive.namelist()) != set(FILES):
                raise SystemExit(f"Invalid ZIP: {target}")
        sha = hashlib.sha256(target.read_bytes()).hexdigest()
        print(f"{filename}: SHA256={sha}")
    print("Publisher-only packages ready; not submitted, signed or publicly installed.")

if __name__ == "__main__":
    main()
