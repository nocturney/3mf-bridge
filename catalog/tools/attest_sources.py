#!/usr/bin/env python3
"""Read-only on-disk attestation for every pinned SliceBridge profile reference.

Usage: python catalog/tools/attest_sources.py --sources-root DIRECTORY
Requires the eight pinned, clean upstream Git checkouts. Does not run slicers
or copy any upstream profile payload into the catalog.
"""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
from collections import Counter

import build
import check


def digest_source(data: bytes) -> str:
    # Profile text hashes are canonicalized by the importer.
    return hashlib.sha256(data.replace(b"\r\n", b"\n")).hexdigest()


def profile_source_path(manifest: str, relative: str) -> str:
    """Find JSON model/preset path relative to the source vendor directory.

    Prusa INI references already refer to their full repository-relative INI.
    """
    check.check_relative(manifest)
    check.check_relative(relative)
    if manifest.lower().endswith(".ini"):
        if relative != manifest:
            raise ValueError("Prusa profile reference is not the pinned manifest")
        return manifest
    directory = Path(manifest).parent / Path(manifest).stem
    return (directory / relative).as_posix()


def attest(root: Path, sources_root: Path) -> dict:
    snapshot = check.check_registry(root)
    locks = json.loads((root / "sources.lock.json").read_text(encoding="utf-8"))["sources"]
    source_by_key = {x["key"]: x for x in locks}
    for source in locks:
        checkout = sources_root / source["key"]
        if not (checkout / ".git").exists():
            raise ValueError("missing source checkout " + source["key"])
        import subprocess
        revision = subprocess.check_output(
            ["git", "-C", str(checkout), "rev-parse", "HEAD"], text=True).strip()
        if revision != source["commit"]:
            raise ValueError("source revision does not match lock: " + source["key"])
        remote = subprocess.check_output(
            ["git", "-C", str(checkout), "remote", "get-url", "origin"],
            text=True).strip().rstrip("/").lower().removesuffix(".git")
        if remote != ("https://github.com/" + source["repository"]).lower():
            raise ValueError("unexpected source origin: " + source["key"])
        dirty = subprocess.check_output(
            ["git", "-C", str(checkout), "status", "--porcelain",
             "--untracked-files=all"], text=True)
        if dirty.strip():
            raise ValueError("source checkout not clean: " + source["key"])

    data = root / "data" / "makers"
    byte_cache: dict[Path, str] = {}
    ini_sections: dict[Path, set[str]] = {}
    references = Counter()
    for maker in sorted(data.glob("*.json")):
        doc = json.loads(maker.read_text(encoding="utf-8"))
        for model in doc["models"]:
            for candidate in model["profile_sources"]:
                checkout = sources_root / candidate["publisher"]
                manifest_rel = candidate["manifest"]["path"]
                refs = [("manifest", manifest_rel, candidate["manifest"])]
                refs.append((
                    "model",
                    profile_source_path(manifest_rel, candidate["model"]["path"]),
                    candidate["model"]
                ))
                for variant in candidate["variants"]:
                    for kind in ("machine", "process", "filament"):
                        part = variant[kind]
                        relatives = [part]
                        relatives += part.get("inheritance", [])
                        for ancestor in relatives:
                            refs.append((
                                kind,
                                profile_source_path(manifest_rel, ancestor["path"]),
                                ancestor,
                            ))
                    for filament in variant.get("filaments", []):
                        for ancestor in [filament, *filament.get("inheritance", [])]:
                            refs.append(("filament", profile_source_path(
                                manifest_rel, ancestor["path"]), ancestor))
                for kind, relative, ref in refs:
                    target = build.safe_path(checkout, relative)
                    if target not in byte_cache:
                        byte_cache[target] = digest_source(target.read_bytes())
                    if byte_cache[target] != ref["sha256"]:
                        raise ValueError("profile hash mismatch: " + candidate["publisher"] +
                                         "/" + relative + " (" + kind + ")")
                    section = ref.get("section")
                    if section:
                        if target.suffix.lower() != ".ini":
                            raise ValueError("section refers to non-INI source file")
                        if target not in ini_sections:
                            ini_sections[target] = set(build.read_ini_sections(target))
                        if section not in ini_sections[target]:
                            raise ValueError("missing INI section: " + relative + ":" + section)
                    references[kind] += 1
    return {
        "source_checkouts": len(locks),
        "models": snapshot["models"],
        "structurally_verified": snapshot["structurally_verified"],
        "distinct_upstream_files": len(byte_cache),
        "checked_ini_bundles": len(ini_sections),
        "checked_references": dict(sorted(references.items())),
        "status": "PASS",
        "meaning": "profile paths and hashes exist in pinned upstream checkout; no 3MF conversion verified"
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--sources-root", required=True, type=Path)
    parser.add_argument("--catalog", default=Path(__file__).resolve().parents[1], type=Path)
    args = parser.parse_args()
    print(json.dumps(attest(args.catalog.resolve(), args.sources_root.resolve()),
                     indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
