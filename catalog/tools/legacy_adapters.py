"""Pure legacy-ID bridge for planned future SliceBridge printer selection.

This module never opens slicers, reads customer settings, downloads profiles,
changes conversion settings, or enables any untested printer. It validates
the committed compatibility contract and can PREVIEW (not apply) a migration
of caller-supplied settings data.
"""
from __future__ import annotations

import copy
import json
import re
from pathlib import Path, PurePosixPath

EXPECTED_LEGACY_TARGETS = {"u1", "c5", "cc2", "qidi"}
SAFE_ID = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")
SAFE_NOZZLE = re.compile(r"^[0-9]+(?:\.[0-9]+)?$")
SOURCE_NOZZLES = {"0.2", "0.4", "0.6", "0.8"}


def demand(condition: bool, description: str):
    if not condition:
        raise ValueError(description)


def _data_path(root: Path, name: str) -> Path:
    demand(isinstance(name, str), "non-string maker path")
    posix = PurePosixPath(name)
    demand(not posix.is_absolute() and "\\" not in name and ":" not in name and
           ".." not in posix.parts and len(posix.parts) == 2 and
           posix.parts[0] == "makers" and name.endswith(".json"),
           "unsafe maker path")
    target = (root / "data" / name).resolve()
    demand(target.is_relative_to((root / "data").resolve()),
           "maker escaped catalogue data directory")
    return target


def validate_mapping(root: Path) -> dict:
    root = root.resolve()
    document = json.loads((root / "data" / "legacy-adapters.json").read_text(encoding="utf-8"))
    demand(document.get("schema_version") == 1, "wrong legacy adapter mapping version")
    demand(document.get("format") == "slicebridge-existing-adapter-compatibility-map",
           "wrong legacy adapter format")
    demand(document.get("source_settings_schema_version") == 1,
           "legacy settings version must stay v1 until migration is implemented")
    observed = document.get("observed_wails_backend", {})
    demand(observed.get("repository") == "nocturney/slicebridge" and
           re.fullmatch(r"[0-9a-f]{40}", observed.get("commit", "")) is not None and
           observed.get("files") == ["native/presets/presets.go",
                                     "native/preferences.go", "native/batch.go"],
           "missing fixed Wails adapter source provenance")
    demand(document.get("registration_policy") ==
           "metadata_only_never_auto_enable_catalogued_models",
           "catalogue must not enable conversion directly")
    demand(document.get("unmapped_catalogue_models") == "catalog_only",
           "new printers must default to catalog-only")
    demand(document.get("preserve_saved_preferences") is True,
           "legacy settings retention disabled")
    demand(document.get("profile_payloads_distributed") is False and
           document.get("default_profile_acquisition") == "read_local_slicer_installation_only",
           "third-party profile download or distribution is forbidden")
    demand(document.get("runtime_adapter_qa_not_certified") is True,
           "metadata falsely claims runtime adapter QA")

    keys: set[str] = set()
    ids: set[str] = set()
    for item in document["mappings"]:
        key, model_id = item["legacy_target"], item["model_id"]
        demand(key in EXPECTED_LEGACY_TARGETS and key not in keys,
               "unknown or duplicate legacy target")
        demand(isinstance(model_id, str) and SAFE_ID.fullmatch(model_id) is not None and
               model_id not in ids, "invalid or reused stable model ID")
        keys.add(key)
        ids.add(model_id)
        demand(item["adapter_status"] == "legacy_adapter_exists_not_catalog_qa_certified",
               "legacy adapter incorrectly certified")
        demand(item["profile_location_policy"] == "installed_slicer_only" and
               item["catalog_activates_conversion"] is False and
               item["conversion_tested"] is False, "unearned conversion-ready claim")
        maker = json.loads(_data_path(root, item["maker_file"]).read_text(encoding="utf-8"))
        model = [model for model in maker["models"] if model["id"] == model_id]
        demand(len(model) == 1 and model[0]["status"] == "structural_verified",
               "legacy model cannot be uniquely resolved to a structurally indexed model")
        demand(model[0]["conversion_tested"] is False,
               "catalogue claims tested converter without runtime QA")
        nozzles = item["target_nozzles_mm"]
        demand(isinstance(nozzles, list) and len(nozzles) == 4 and
               len(set(nozzles)) == 4 and
               all(isinstance(n, str) and SAFE_NOZZLE.fullmatch(n) for n in nozzles),
               "unsupported or duplicate legacy nozzle")
        available = model[0]["preferred_by_nozzle"]
        demand(all(n in available for n in nozzles),
               "mapped nozzle is not present in structurally indexed model")
        demand(not any("+" in nozzle for nozzle in nozzles),
               "legacy single-nozzle adapter cannot claim multi-toolhead support")
        redirect = item["source_nozzle_remap_mm"]
        demand(isinstance(redirect, dict), "invalid source-nozzle redirect map")
        demand(redirect == ({"0.2": "0.25"} if key == "c5" else {}),
               "legacy source-nozzle remapping drift")
        demand(all(src in SOURCE_NOZZLES and dst in nozzles
                   for src, dst in redirect.items()), "invalid source nozzle redirect")
        for src in SOURCE_NOZZLES:
            mapped = redirect.get(src, src)
            demand(mapped in nozzles, "unmapped legacy source nozzle")
        builtin = item["builtin_fast_path_nozzle_mm"]
        demand(builtin == ("0.4" if key == "u1" else None),
               "built-in Snapmaker U1 compatibility path drift")
        demand(item.get("builtin_fast_path_preconditions") ==
               ({"slicer": "auto", "custom_filaments": "none"} if key == "u1" else None),
               "built-in U1 fast path requires auto slicer and no custom materials")
    demand(keys == EXPECTED_LEGACY_TARGETS and len(document["mappings"]) == 4,
           "exactly four existing adapters must be mapped")
    return {"legacy_adapter_count": len(keys), "mapped_model_ids": sorted(ids),
            "catalog_only_model_count": _indexed_count(root) - len(ids),
            "conversion_tested": 0,
            "runtime_adapter_activated_by_catalog": False}


def _indexed_count(root: Path) -> int:
    manifest = json.loads((root / "data" / "index.json").read_text(encoding="utf-8"))
    return manifest["models"]


def preview_settings_migration(settings: dict, mapping: dict) -> dict:
    """Never touch a disk file or modify the caller's v1 settings object.

    Return an explicit plan with verbatim copies of preference records so a
    future Go migration can be tested for lossless nozzle/slicer/filament
    preservation before it is allowed to change persisted user settings.
    """
    demand(isinstance(settings, dict) and settings.get("version") == 1,
           "settings must be a v1 object")
    printers = settings.get("printers", {})
    demand(isinstance(printers, dict), "legacy printers field must be an object")
    mapping_by_key = {item["legacy_target"]: item for item in mapping["mappings"]}
    planned = []
    unknown = {}
    for key, value in printers.items():
        demand(isinstance(key, str) and isinstance(value, dict),
               "invalid legacy printer settings record")
        if key not in mapping_by_key:
            unknown[key] = copy.deepcopy(value)
            continue
        planned.append({
            "legacy_target": key,
            "catalog_model_id": mapping_by_key[key]["model_id"],
            "original_preference": copy.deepcopy(value),
            "catalog_activates_conversion": False,
        })
    return {
        "kind": "read_only_migration_preview",
        "saved_settings_schema_version": 1,
        "mapped_printers": planned,
        "unknown_printers_preserved": unknown,
        "preserved_original_v1_settings": copy.deepcopy(settings),
        "changes_written": False,
    }
