#!/usr/bin/env python3
"""Offline integrity gate for committed SliceBridge printer profile metadata."""
from __future__ import annotations

import hashlib
import json
import re
import sys
from pathlib import Path, PurePosixPath

ORDER = {"manufacturer": 0, "orca": 1, "prusa": 2}
VENDOR_PUBLISHERS = {"bambu", "qidi", "elegoo", "prusa_official"}


def ensure(condition: bool, message: str):
    if not condition:
        raise ValueError(message)


def load_json(file: Path):
    return json.loads(file.read_text(encoding="utf-8"))


def is_digest(value):
    return isinstance(value, str) and re.fullmatch("[a-f0-9]{64}", value) is not None


def is_commit(value):
    return isinstance(value, str) and re.fullmatch("[a-f0-9]{40}", value) is not None


def snapshot_digest(raw: bytes) -> str:
    """Stable across Git's LF and CRLF working-tree policies."""
    return hashlib.sha256(raw.replace(b"\r\n", b"\n")).hexdigest()


def check_relative(path: str):
    ensure(isinstance(path, str) and path and "\\" not in path and ":" not in path,
           "invalid path: " + repr(path))
    pure = PurePosixPath(path)
    ensure(not pure.is_absolute() and ".." not in pure.parts and "\x00" not in path,
           "unsafe relative source path: " + repr(path))


def check_registry(root: Path) -> dict:
    index = load_json(root / "data"/"index.json")
    lock = load_json(root / "sources.lock.json")
    ensure(index["schema_version"] == 1 and lock["schema_version"] == 1,
           "unsupported registry schema")
    ensure(index["source_priority"] == ["manufacturer", "orca", "prusa"],
           "incorrect profile source priority")
    entries = lock["sources"]
    ensure(len(entries) >= 5 and len({e["key"] for e in entries}) == len(entries),
           "missing or duplicate sources")
    by_key = {}
    for item in entries:
        ensure(is_commit(item.get("commit")), "unpinned source revision")
        ensure(item["provider"] in ORDER and item["license_status"] == "review_before_redistribution",
               "unknown source provider or distribution rights")
        ensure(re.fullmatch(r"[a-z0-9_-]+", item["key"]) is not None, "invalid source identifier")
        by_key[item["key"]] = item
    ensure(index["manufacturers"] == len(index["maker_files"]),
           "incorrect manufacturer count")
    seen_files, seen_ids = set(), set()
    total_models, total_verified = 0, 0
    for record in index["maker_files"]:
        filename = record["path"]
        check_relative(filename)
        ensure(filename.startswith("makers/") and filename.endswith(".json"),
               "maker file has invalid location")
        ensure(filename not in seen_files, "duplicate manufacturer output file")
        seen_files.add(filename)
        raw = (root / "data" / filename).read_bytes()
        # Git may convert LF to CRLF on Windows checkout. Verify the
        # canonical UTF-8/LF serialized document, not platform newlines.
        ensure(snapshot_digest(raw) == record["sha256"],
               "manufacturer file checksum mismatch")
        document = json.loads(raw)
        ensure(document["manufacturer"] == record["manufacturer"], "wrong maker label")
        ensure(len(document["models"]) == record["models"], "wrong maker count")
        verified_in_maker = 0
        for model in document["models"]:
            identity = model["id"]
            ensure(identity not in seen_ids, "duplicate printer id")
            seen_ids.add(identity)
            ensure(model["manufacturer"] == document["manufacturer"], "manufacturer mismatch")
            ensure(model["release_date"] is None or model.get("release_date_evidence"),
                   "release date without source proof")
            ensure(model["conversion_tested"] is False, "unverified runtime certification")
            candidates = model["profile_sources"]
            ensure(candidates, "printer without any profile candidate")
            seen_source = set()
            eligible = []
            actual_order = []
            for candidate in candidates:
                publisher = candidate["publisher"]
                ensure(publisher in by_key, "profile candidate not in locked sources")
                locked = by_key[publisher]
                ensure(candidate["commit"] == locked["commit"] and
                       candidate["repository"] == locked["repository"] and
                       candidate["provider"] == locked["provider"],
                       "profile source hash/repository does not match lock")
                ensure(candidate["provider"] != "manufacturer" or publisher in VENDOR_PUBLISHERS,
                       "third-party profiles mislabeled as manufacturer original")
                check_relative(candidate["manifest"]["path"])
                check_relative(candidate["model"]["path"])
                ensure(is_digest(candidate["manifest"]["sha256"]) and
                       is_digest(candidate["model"]["sha256"]), "missing model/source checksums")
                key = (publisher, candidate["manifest"]["path"])
                ensure(key not in seen_source, "duplicate source candidate")
                seen_source.add(key)
                verified = candidate["validation"] == "structural_verified"
                variants = candidate["variants"]
                ensure(verified == bool(variants), "empty/misreported profile variants")
                declared = set(candidate["nozzles_mm"])
                ensure(declared, "printer has no nozzle measurements")
                for variant in variants:
                    ensure(variant["nozzle_mm"] in declared, "variant nozzle mismatch")
                    for required in ("machine", "process", "filament"):
                        part = variant[required]
                        check_relative(part["path"])
                        ensure(is_digest(part["sha256"]) and part["name"],
                               "unverifiable machine/process/filament reference")
                        if required in ("process", "filament"):
                            chain = part.get("inheritance", [])
                            ensure(chain and chain[-1]["sha256"] == part["sha256"]
                                   and chain[-1]["name"] == part["name"],
                                   "missing/mismatched preset inheritance chain")
                            names = set()
                            for ancestor in chain:
                                check_relative(ancestor["path"])
                                ensure(ancestor["name"] not in names and is_digest(ancestor["sha256"]),
                                       "preset inheritance cycle or unhashed ancestor")
                                names.add(ancestor["name"])
                actual_order.append((ORDER[candidate["provider"]],
                                     not verified, publisher, candidate["manifest"]["path"]))
                if verified:
                    eligible.append(candidate)
            ensure(actual_order == sorted(actual_order), "source preference order modified")
            chosen = next((c for c in candidates if c["validation"] == "structural_verified"), None)
            expected = model["preferred_profile"]
            if chosen:
                ensure(model["status"] == "structural_verified" and expected is not None,
                       "verified source not selected")
                ensure(expected["provider"] == chosen["provider"] and
                       expected["publisher"] == chosen["publisher"] and
                       expected["manifest"] == chosen["manifest"]["path"],
                       "wrong preferred provider or fallback priority")
                actual = sorted({v["nozzle_mm"] for v in chosen["variants"]}, key=float)
                ensure(expected["nozzles_mm"] == actual, "incorrect validated nozzle choices")
                verified_in_maker += 1
            else:
                ensure(model["status"] == "index_only" and expected is None,
                       "unverified model incorrectly approved")
            expected_nozzles = {}
            for profile_source in candidates:
                if profile_source["validation"] != "structural_verified":
                    continue
                for variant in profile_source["variants"]:
                    diameter = variant["nozzle_mm"]
                    expected_nozzles.setdefault(diameter, {
                        "provider": profile_source["provider"],
                        "publisher": profile_source["publisher"],
                        "manifest": profile_source["manifest"]["path"],
                        "machine": variant["machine"]["name"],
                        "process": variant["process"]["name"],
                        "filament": variant["filament"]["name"],
                    })
            ensure(model["preferred_by_nozzle"] == expected_nozzles,
                   "nozzle-specific priority or validated profile references are incorrect")
            total_models += 1
        ensure(verified_in_maker == record["structurally_verified"],
               "maker validation count mismatch")
        total_verified += verified_in_maker
    ensure(total_models == index["models"], "registry has missing models")
    ensure(total_verified == index["structurally_verified"],
           "registry validation count mismatch")
    ensure(index["conversion_tested"] == 0, "unearned runtime validation claim")
    actual_files = {p.relative_to(root / "data").as_posix() for p in
                    (root / "data"/"makers").glob("*.json")}
    ensure(actual_files == seen_files, "unindexed or missing data files")
    return {"models": total_models, "structurally_verified": total_verified,
            "manufacturers": len(seen_files), "sources": len(entries)}


if __name__ == "__main__":
    source = Path(sys.argv[1] if len(sys.argv) > 1 else Path(__file__).resolve().parents[1])
    print(json.dumps(check_registry(source.resolve()), indent=2))
