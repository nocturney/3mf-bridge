#!/usr/bin/env python3
"""Validate a single current development version and its changelog entry.

Lightweight, offline check; never tags, publishes, or writes repository files.
"""
from __future__ import annotations
import argparse
from pathlib import Path
import re

SEMVER = re.compile(
    r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)"
    r"(?:-((?:0|[1-9][0-9]*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*)"
    r"(?:\.(?:0|[1-9][0-9]*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*))*))?"
    r"(?:\+[0-9A-Za-z.-]+)?"
)

def validate(root: Path) -> str:
    raw = (root / "VERSION").read_text(encoding="utf-8")
    if not raw.endswith("\n") or raw.strip() != raw.rstrip("\r\n") or "\n" in raw[:-1]:
        raise ValueError("VERSION must be a single newline-terminated SemVer")
    version = raw.strip()
    if not SEMVER.fullmatch(version):
        raise ValueError("VERSION is not a valid SemVer value")
    changelog = (root / "CHANGELOG.md").read_text(encoding="utf-8")
    marker = f"## [{version}]"
    if changelog.count(marker) != 1 or not re.search(
        r"^## \[" + re.escape(version) + r"\](?:[ \t-]|$)", changelog, re.M
    ):
        raise ValueError("CHANGELOG must contain exactly one matching version heading")
    section = changelog.split(marker, 1)[1].split("\n## ", 1)[0]
    if not re.search(r"(?im)^[-*] ", section):
        raise ValueError("versioned milestone requires at least one substantive change")
    if "not released" not in section.lower() and "development" not in section.lower():
        raise ValueError("development milestone must be labeled unpublished")
    return version

def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, default=Path(__file__).resolve().parents[1])
    args = parser.parse_args()
    print("VERSION/CHANGELOG PASS:", validate(args.repo.resolve()))

if __name__ == "__main__":
    main()
