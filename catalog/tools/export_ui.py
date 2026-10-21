"""Build a compact, attribution-conscious, OFFLINE UI picker snapshot.

The Wails app bundles this JSON as metadata. It never contains third-party
machine/process/filament configurations, downloaded photos, printer icons,
G-code, or a claim of conversion support.
"""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path

from check import check_registry


def json_bytes(payload: dict) -> bytes:
    return (json.dumps(payload, ensure_ascii=False, separators=(",", ":")) + "\n").encode("utf-8")


def local_cover_candidates(model: dict) -> list[str]:
    """References only; image bytes are read from a user's installed slicers.

    Orca/Bambu/QIDI/Elegoo/FlashForge machine vendors usually place
    Vendor/Model_cover.png next to their own machine preset folders.
    These paths are suggestions, NEVER downloads or redistributed assets.
    """
    seen, found = set(), []
    for entry in model["profile_sources"]:
        manifest = entry["manifest"]["path"]
        if not manifest.endswith(".json"):
            continue  # Prusa INI can have other thumbnail conventions
        vendor_dir = Path(manifest).stem
        source_name = Path(entry["model"]["path"]).stem
        suggestions = [
            source_name,
            model["manufacturer"] + " " + model["model"],
            model["model"],
        ]
        for name in suggestions:
            parts = (vendor_dir, name)
            if any(not v or len(v) > 140 or "/" in v or "\\" in v
                   or ":" in v or ".." in v
                   or any(ord(c) < 32 for c in v) for v in parts):
                continue
            relative = vendor_dir + "/" + name + "_cover.png"
            if relative not in seen:
                seen.add(relative)
                found.append(relative)
            if len(found) == 12:
                return found
    return found


def build_picker(root: Path) -> bytes:
    stats = check_registry(root)
    index_bytes = (root / "data" / "index.json").read_bytes().replace(b"\r\n", b"\n")
    index = json.loads(index_bytes)
    bridge = json.loads((root / "data" / "legacy-adapters.json").read_text(encoding="utf-8"))
    adapters = {x["model_id"]: x for x in bridge["mappings"]}
    makers, models = set(), []
    for doc in index["maker_files"]:
        vendor = json.loads((root / "data" / doc["path"]).read_text(encoding="utf-8"))
        makers.add(vendor["manufacturer"])
        for source in vendor["models"]:
            adapter = adapters.get(source["id"])
            aliases = []
            if adapter:
                alias = adapter["legacy_display_name"]
                if alias.casefold() not in {source["model"].casefold(),
                                            (source["manufacturer"]+" "+source["model"]).casefold()}:
                    aliases.append(alias)
            models.append({
                "id": source["id"],
                "manufacturer": source["manufacturer"],
                "model": source["model"],
                "aliases": aliases,
                "releaseDate": source["release_date"],
                "structuralStatus": source["status"],
                "nozzles": list(source["preferred_by_nozzle"]),
                "sourcePublisher": source["preferred_profile"]["publisher"]
                                   if source["preferred_profile"] else None,
                "legacyTarget": adapter["legacy_target"] if adapter else None,
                "imagePath": None,
                "logoPath": None,
                "localCoverCandidates": local_cover_candidates(source),
            })
    models.sort(key=lambda x: (x["manufacturer"].casefold(),
                               x["model"].casefold(), x["id"]))
    if len(models) != stats["models"] or len({m["id"] for m in models}) != len(models):
        raise ValueError("duplicate/missing model in UI metadata")
    if len(makers) != stats["manufacturers"]:
        raise ValueError("inconsistent vendor names")
    if sum(bool(x["legacyTarget"]) for x in models) != 4:
        raise ValueError("only the four original adapters may be tagged")
    return json_bytes({
        "schemaVersion": 1,
        "kind": "slicebridge-offline-printer-picker",
        "catalogIndexSHA256": hashlib.sha256(index_bytes).hexdigest(),
        "mediaPolicy": "local installed-slicer PNG cover references only; fallback initials; no image bytes bundled",
        "thirdPartyProfilePayloadsIncluded": False,
        "modelCount": len(models),
        "manufacturers": len(makers),
        "legacyAdapterCount": 4,
        "models": models,
    })


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--catalog", type=Path, default=Path(__file__).resolve().parents[1])
    p.add_argument("--output", type=Path, required=True)
    args = p.parse_args()
    root, destination = args.catalog.resolve(), args.output.resolve()
    data = build_picker(root)
    destination.parent.mkdir(parents=True, exist_ok=True)
    destination.write_bytes(data)
    print(json.dumps({"path": str(destination),
                      "bytes": len(data),
                      "sha256": hashlib.sha256(data).hexdigest(),
                      "models": json.loads(data)["modelCount"]}, indent=2))


if __name__ == "__main__":
    main()
