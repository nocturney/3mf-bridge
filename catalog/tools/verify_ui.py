#!/usr/bin/env python3
"""Verify that the committed UI catalogue is byte-for-byte reproducible.

Checks the *committed Git blob*, never the platform's working tree status:
git checkout may convert LF/CRLF on Windows. No file is modified.
"""
from __future__ import annotations
import argparse
import hashlib
import subprocess
from pathlib import Path

from export_ui import build_picker


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--catalog", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--repo", type=Path, default=Path(__file__).resolve().parents[2])
    args = parser.parse_args()
    expected = build_picker(args.catalog.resolve())
    tracked_path = "native/frontend/dist/printer-catalog.json"
    committed = subprocess.check_output([
        "git", "-C", str(args.repo.resolve()), "show", "HEAD:" + tracked_path
    ]).replace(b"\r\n", b"\n")
    if expected != committed:
        raise SystemExit(
            "Embedded picker snapshot differs from the committed Git blob: "
            "regenerate and commit. "
            f"expected sha256={hashlib.sha256(expected).hexdigest()}, "
            f"committed sha256={hashlib.sha256(committed).hexdigest()}"
        )
    print("Offline picker Git-blob provenance PASS; sha256=" +
          hashlib.sha256(committed).hexdigest())


if __name__ == "__main__":
    main()
