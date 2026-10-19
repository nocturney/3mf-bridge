"""Evidence-only upstream rights ledger; never a redistribution permission grant.

Repository-root license detection is separate from individual vendor
profile/data authorship and can never certify a profile payload for distribution.
"""
from __future__ import annotations

import hashlib
import re
import subprocess
from pathlib import Path


ROOT_LICENSE_FILES = ("LICENSE", "LICENSE.txt", "LICENSE.md", "License", "COPYING", "COPYING.txt")
LICENSES_WITH_AGPL_HEAD = {"GNU AFFERO GENERAL PUBLIC LICENSE": "AGPL-3.0"}


def root_file(repo_dir: Path, relative: str) -> bytes | None:
    proc = subprocess.run(
        ["git", "-C", str(repo_dir), "show", "HEAD:" + relative],
        stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, check=False)
    if proc.returncode != 0:
        return None
    return proc.stdout


def license_entry(checkout: Path, source: dict) -> dict:
    observed = []
    for filename in ROOT_LICENSE_FILES:
        content = root_file(checkout, filename)
        if content is None:
            continue
        text = content[:1500].decode("utf-8-sig", errors="replace")
        license_name = ("AGPL-3.0" if "GNU AFFERO GENERAL PUBLIC LICENSE" in text
                        and "Version 3, 19 November 2007" in text
                        else "unclassified")
        observed.append({
            "path": filename,
            "sha256": hashlib.sha256(content.replace(b"\r\n", b"\n")).hexdigest(),
            "license_id_observed": license_name,
        })
    if len(observed) > 1:
        # Multiple root licenses require human resolution before using a label.
        declared = "multiple_root_licenses_review_required"
    elif len(observed) == 1:
        declared = observed[0]["license_id_observed"]
    else:
        declared = "no_repository_level_license_found"
    return {
        "key": source["key"],
        "repository": source["repository"],
        "commit": source["commit"],
        "pinned_repository_url": (
            "https://github.com/" + source["repository"] + "/tree/" + source["commit"]
        ),
        "repository_root_license_observed": declared,
        "repository_root_license_evidence": observed,
        "individual_profile_licensing_reviewed": False,
        "third_party_profile_payloads_included": False,
        "third_party_profile_payloads_redistribution_cleared": False,
    }


def create_ledger(source_root: Path, sources: list[dict]) -> dict:
    entries = [license_entry(source_root / source["key"], source) for source in sources]
    if len(entries) != len({e["key"] for e in entries}):
        raise ValueError("duplicate source rights entry")
    return {
        "schema_version": 1,
        "purpose": "source provenance and root license evidence, not legal clearance",
        "default_action": "metadata_references_only; use locally installed slicer profiles",
        "items": entries,
    }
