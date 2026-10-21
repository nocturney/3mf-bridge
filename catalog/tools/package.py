#!/usr/bin/env python3
"""Create a deterministic, metadata-only SliceBridge catalogue artifact.

Never publishes or downloads raw upstream slicer profiles. Fails if the
committed index/manifest/reference hashes do not pass offline integrity QA.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import zipfile
from pathlib import Path

import check

_FIXED_ZIP_DATE = (1980, 1, 1, 0, 0, 0)
_PREFIX = "catalog/"
_MAX_MEMBER_BYTES = 32 * 1024 * 1024


def canonical_bytes(content: bytes) -> bytes:
    return content.replace(b"\r\n", b"\n")


def checksum(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def entry_bytes(root: Path, filename: str) -> bytes:
    check.check_relative(filename)
    root = root.resolve()
    target = (root / filename).resolve()
    if not target.is_relative_to(root) or not target.is_file():
        raise ValueError("missing/escaped catalogue file: " + filename)
    if target.stat().st_size > _MAX_MEMBER_BYTES:
        raise ValueError("oversized metadata file: " + filename)
    return canonical_bytes(target.read_bytes())


def manifest_payload(root: Path):
    overview = check.check_registry(root)
    index = json.loads((root / "data" / "index.json").read_text(encoding="utf-8"))
    if index["conversion_tested"] != 0:
        raise ValueError("catalogue bundle cannot certify untested conversions")
    paths = ["schema-v1.json", "sources.lock.json", "data/index.json", "data/audit.json",
             "data/attribution.json", "data/legacy-adapters.json"]
    paths += ["data/" + item["path"] for item in index["maker_files"]]
    if len(set(paths)) != len(paths):
        raise ValueError("duplicate archive entry")
    files = []
    content = {}
    for filename in paths:
        name = _PREFIX + filename
        blob = entry_bytes(root, filename)
        content[name] = blob
        files.append({"path": name, "sha256": checksum(blob), "bytes": len(blob)})
    receipt = {
        "schema_version": 1,
        "bundle_kind": "printer-profile-metadata-only",
        "source_profile_payloads_included": False,
        "conversion_tested_models": 0,
        "indexed_models": overview["models"],
        "structurally_verified_models": overview["structurally_verified"],
        "sources": overview["sources"],
        "files": files,
    }
    return receipt, content


def write_zip_member(handle: zipfile.ZipFile, name: str, data: bytes):
    check.check_relative(name)
    info = zipfile.ZipInfo(name, date_time=_FIXED_ZIP_DATE)
    # No zlib recompression: exact ZIP bytes are reproducible across OSes.
    info.compress_type = zipfile.ZIP_STORED
    info.create_system = 3
    info.external_attr = 0o100644 << 16
    handle.writestr(info, data, compress_type=zipfile.ZIP_STORED)


def validate_archive(path: Path) -> dict:
    with zipfile.ZipFile(path, "r") as archive:
        members = archive.namelist()
        if len(members) != len(set(members)):
            raise ValueError("duplicate archive entries")
        for name in members:
            check.check_relative(name)
            if not name.startswith(_PREFIX) or not name.endswith(".json"):
                raise ValueError("unexpected bundled file: " + name)
        if "catalog/bundle-manifest.json" not in members:
            raise ValueError("missing metadata bundle-manifest")
        payload = archive.read("catalog/bundle-manifest.json")
        manifest = json.loads(payload)
        if manifest["bundle_kind"] != "printer-profile-metadata-only":
            raise ValueError("archive misidentified as runnable profiles")
        if manifest["conversion_tested_models"] != 0 or manifest["source_profile_payloads_included"]:
            raise ValueError("archive makes unsupported conversion claims")
        expected = {x["path"]: x for x in manifest["files"]}
        if set(members) != set(expected) | {"catalog/bundle-manifest.json"}:
            raise ValueError("archive has unlisted or missing files")
        if sum(archive.getinfo(x).file_size for x in members) > 32 * 1024 * 1024:
            raise ValueError("archive expanded payload exceeds 32 MiB")
        for name, item in expected.items():
            if archive.getinfo(name).file_size != item["bytes"]:
                raise ValueError("archive content length mismatch: " + name)
            if checksum(archive.read(name)) != item["sha256"]:
                raise ValueError("archive metadata checksum mismatch: " + name)
        index = json.loads(archive.read("catalog/data/index.json"))
        if (index["models"] != manifest["indexed_models"] or
                index["structurally_verified"] != manifest["structurally_verified_models"]):
            raise ValueError("archive model counts inconsistent")
    return manifest


def build_archive(root: Path, output: Path) -> dict:
    manifest, blobs = manifest_payload(root.resolve())
    digest = checksum(blobs["catalog/data/index.json"])[:12]
    filename = "slicebridge-printer-catalog-v1-" + digest + ".zip"
    output.mkdir(parents=True, exist_ok=True)
    target = output / filename
    sumfile = output / (filename + ".sha256")
    if target.exists() or sumfile.exists():
        raise FileExistsError("refusing to overwrite an existing metadata artifact")
    encoded = (json.dumps(manifest, indent=2, ensure_ascii=False) + "\n").encode("utf-8")
    temporary = output / (filename + ".tmp")
    if temporary.exists():
        raise FileExistsError("temporary output already exists")
    try:
        with zipfile.ZipFile(temporary, "w") as archive:
            for name in sorted(blobs):
                write_zip_member(archive, name, blobs[name])
            write_zip_member(archive, "catalog/bundle-manifest.json", encoded)
        verified = validate_archive(temporary)
        archive_sha = checksum(temporary.read_bytes())
        os.replace(temporary, target)
        sumfile.write_bytes((archive_sha + "  " + filename + "\n").encode("ascii"))
    finally:
        if temporary.exists():
            temporary.unlink()
    return {
        "file": str(target),
        "sha256": archive_sha,
        "models": verified["indexed_models"],
        "structurally_verified": verified["structurally_verified_models"],
        "verified_metadata_files": len(verified["files"]),
        "status": "metadata-only, conversion not certified"
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--catalog", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    print(json.dumps(build_archive(args.catalog.resolve(), args.output.resolve()), indent=2))


if __name__ == "__main__":
    main()
