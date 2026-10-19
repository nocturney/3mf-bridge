#!/usr/bin/env python3
"""SliceBridge reproducible, fail-closed profile registry (index only, no G-code)."""
from __future__ import annotations

import argparse
import hashlib
import json
import re
import subprocess
from collections import Counter, defaultdict
from pathlib import Path, PurePosixPath

SOURCES = (
    ("bambu", "manufacturer", "bambulab/BambuStudio", "resources/profiles"),
    ("qidi", "manufacturer", "QIDITECH/QIDIStudio", "resources/profiles"),
    ("elegoo", "manufacturer", "elegooofficial/ElegooSlicer", "resources/profiles"),
    ("orca", "orca", "OrcaSlicer/OrcaSlicer", "resources/profiles"),
    ("prusa_official", "manufacturer", "prusa3d/PrusaSlicer-settings", "live/PrusaResearch"),
    ("prusa_community", "prusa", "prusa3d/PrusaSlicer-settings-non-prusa-fff", "."),
)
PRIORITY = {"manufacturer": 0, "orca": 1, "prusa": 2}
ALIAS = {"BBL": "Bambu Lab", "Qidi": "QIDI", "QIDITechnology": "QIDI",
         "Elegoo": "ELEGOO", "PrusaResearch": "Prusa Research",
         "Lulzbot": "LulzBot", "Ratrig": "RatRig", "Ultimaker": "UltiMaker"}
PREFIXES = {
    "Bambu Lab": ("Bambu Lab ",),
    "QIDI": ("Qidi ", "QIDI "),
    "ELEGOO": ("Elegoo ",),
    "Prusa Research": ("Original Prusa ", "Prusa "),
    "Snapmaker": ("Snapmaker ",),
}
HEADER = re.compile(r"^\s*\[([^]]+)\]\s*$")
ATTR = re.compile(r"^\s*([a-zA-Z_]\w*)\s*=\s*(.*?)\s*$")


class InvalidProfile(ValueError):
    pass


def sha(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def safe_path(root: Path, relative: str) -> Path:
    if not isinstance(relative, str) or not relative or "\x00" in relative or "\\" in relative:
        raise InvalidProfile("invalid relative profile path")
    pure = PurePosixPath(relative)
    if pure.is_absolute() or ".." in pure.parts or ":" in relative:
        raise InvalidProfile("profile path escapes pinned source")
    path = (root / relative).resolve()
    if not path.is_relative_to(root.resolve()) or not path.is_file():
        raise InvalidProfile("missing or escaped source profile")
    return path


def read_json(root: Path, relative: str) -> tuple[dict, str]:
    path = safe_path(root, relative)
    if path.stat().st_size > 16 * 1024 * 1024:
        raise InvalidProfile("oversized JSON profile")
    content = path.read_bytes()
    try:
        value = json.loads(content.decode("utf-8-sig"))
    except (UnicodeError, ValueError) as exc:
        raise InvalidProfile("malformed JSON") from exc
    if not isinstance(value, dict):
        raise InvalidProfile("profile is not a JSON object")
    return value, sha(content)


def normalized(value) -> str:
    return " ".join(str(value).casefold().split())


def model_label(brand: str, model: str) -> str:
    text = " ".join(model.strip().split())
    for prefix in PREFIXES.get(brand, ()):
        if text.casefold().startswith(prefix.casefold()):
            return text[len(prefix):].strip()
    return text


def model_key(brand: str, model: str) -> str:
    plain = model_label(brand, model)
    key = re.sub(r"[^a-z0-9]+", "-", (brand + " " + plain).lower()).strip("-")
    if not plain or not key:
        raise InvalidProfile("no stable model identifier")
    return key


def diameters(value) -> list[str]:
    if isinstance(value, str):
        tokens = value.split(";")
    elif isinstance(value, list):
        tokens = value
    else:
        tokens = []
    parsed = set()
    for token in tokens:
        token = str(token).strip()
        if not token:
            continue
        try:
            number = float(token)
        except ValueError as exc:
            raise InvalidProfile("unparseable nozzle diameter") from exc
        if not (0.1 <= number <= 1.2):
            raise InvalidProfile("nozzle diameter outside supported range")
        parsed.add(f"{number:g}")
    return sorted(parsed, key=float)


def by_name(manifest: dict, key: str) -> dict[str, str]:
    result = {}
    for item in manifest.get(key, []):
        if isinstance(item, dict):
            name, sub = item.get("name"), item.get("sub_path")
            if isinstance(name, str) and isinstance(sub, str):
                if name and sub:
                    result.setdefault(name, sub)
    return result


def resolve_machine(name: str, folder: Path, index: dict[str, str],
                    cache: dict, visiting: tuple[str, ...] = ()):
    if name in cache:
        return cache[name]
    if name in visiting or len(visiting) > 24:
        raise InvalidProfile("machine inheritance cycle")
    if name not in index:
        raise InvalidProfile("machine inherits a missing profile")
    values, filehash = read_json(folder, index[name])
    if values.get("type") not in ("machine", None):
        raise InvalidProfile("wrong machine profile type")
    parent = values.get("inherits")
    if parent:
        inherited, _, _ = resolve_machine(parent, folder, index, cache, visiting + (name,))
        values = {**inherited, **values}
    cache[name] = (values, filehash, index[name])
    return cache[name]



def resolve_preset(name: str, folder: Path, indexed: dict[str, str], kind: str,
                   cache: dict, visiting: tuple[str, ...] = ()):
    """Validate all inherited process/filament resources and pin every dependency."""
    if name in cache:
        return cache[name]
    if name in visiting or len(visiting) > 24:
        raise InvalidProfile(kind + " inheritance cycle")
    if name not in indexed:
        raise InvalidProfile(kind + " inherits a missing preset")
    obj, filehash = read_json(folder, indexed[name])
    if obj.get("type") not in (kind, None):
        raise InvalidProfile("not a " + kind + " preset")
    chain = []
    parent = obj.get("inherits")
    if parent:
        inherited, dependencies = resolve_preset(
            str(parent), folder, indexed, kind, cache, visiting + (name,))
        obj = {**inherited, **obj}
        chain.extend(dependencies)
    chain.append({"name": name, "path": indexed[name], "sha256": filehash})
    cache[name] = (obj, chain)
    return cache[name]


def validation_for_model(folder: Path, item: dict, manifest: dict, maps: dict):
    name, relative = item.get("name"), item.get("sub_path")
    if not isinstance(name, str) or not isinstance(relative, str) or not name:
        raise InvalidProfile("invalid model manifest entry")
    model, modelhash = read_json(folder, relative)
    if model.get("type") not in ("machine_model", None):
        raise InvalidProfile("not a machine model profile")
    if normalized(model.get("name")) != normalized(name):
        raise InvalidProfile("manifest and model name differ")
    if model.get("machine_tech", "FFF") not in ("FFF", "FDM"):
        raise InvalidProfile("not an FDM printer")
    declared = diameters(model.get("nozzle_diameter"))
    if not declared:
        raise InvalidProfile("model missing nozzle variants")
    machine, process, filament = maps["machine"], maps["process"], maps["filament"]
    cache = maps["machine_cache"]
    variants = []
    errors = Counter()
    for machine_name in machine:
        try:
            config, mach_hash, mach_path = resolve_machine(machine_name, folder, machine, cache)
            if normalized(config.get("printer_model")) != normalized(name):
                continue
            nozzle = diameters(config.get("nozzle_diameter"))
            if len(nozzle) != 1 or nozzle[0] not in declared:
                raise InvalidProfile("nozzle/model mismatch")
            print_name = config.get("default_print_profile")
            if not isinstance(print_name, str) or print_name not in process:
                raise InvalidProfile("missing process preset")
            _, print_chain = resolve_preset(print_name, folder, process, "process",
                                            maps.setdefault("process_cache", {}))
            default = config.get("default_filament_profile")
            default = default if isinstance(default, list) else [default] if isinstance(default, str) else []
            fil, filament_chain = None, None
            for candidate_filament in default:
                if candidate_filament not in filament:
                    continue
                try:
                    _, filament_chain = resolve_preset(candidate_filament, folder, filament, "filament",
                                                       maps.setdefault("filament_cache", {}))
                    fil = candidate_filament
                    break
                except InvalidProfile:
                    continue
            if fil is None or not filament_chain:
                raise InvalidProfile("no valid default filament dependency chain")
            variants.append({
                "nozzle_mm": nozzle[0],
                "machine": {"name": machine_name, "path": mach_path, "sha256": mach_hash},
                "process": {"name": print_name, "path": process[print_name],
                            "sha256": print_chain[-1]["sha256"], "inheritance": print_chain},
                "filament": {"name": fil, "path": filament[fil],
                             "sha256": filament_chain[-1]["sha256"], "inheritance": filament_chain},
            })
        except InvalidProfile as exc:
            # Bad variants cannot be used as a verified machine configuration.
            errors[str(exc)] += 1
    variants.sort(key=lambda v: (float(v["nozzle_mm"]), v["machine"]["name"]))
    return {
        "model": {"path": relative, "sha256": modelhash,
                  "manufacturer_model_id": model.get("model_id")},
        "nozzles_mm": declared,
        "variants": variants,
        "validation": "structural_verified" if variants else "index_only",
        "warnings": [f"{name} ({count})" for name, count in errors.most_common(4)]
                    if not variants else [],
    }


def scan_orca_json(source: dict, checkout: Path):
    root = checkout / source["base"]
    if not root.is_dir():
        raise FileNotFoundError(str(root))
    for manifest_path in sorted(root.glob("*.json")):
        manifest, manifest_sha = read_json(root, manifest_path.name)
        if not isinstance(manifest.get("machine_model_list"), list):
            continue
        vendor = manifest_path.stem
        publisher = source["key"]
        manufacturer = ALIAS.get(vendor, vendor)
        if source["provider"] == "manufacturer":
            permitted = {
                "bambu": {"BBL"},
                "qidi": {"Q Series", "X 3 Series", "X 4 Series", "X 5 Series"},
                "elegoo": {"Elegoo"}
            }[publisher]
            if vendor not in permitted:
                continue
            if publisher == "qidi":
                manufacturer = "QIDI"
        folder = root / vendor
        maps = {k: by_name(manifest, k + "_list") for k in ("machine", "process", "filament")}
        maps["machine_cache"] = {}
        for item in manifest["machine_model_list"]:
            if not isinstance(item, dict):
                continue
            try:
                profile = validation_for_model(folder, item, manifest, maps)
                yield {
                    "manufacturer": manufacturer,
                    "name": item["name"],
                    "candidate": {
                        "provider": source["provider"],
                        "publisher": publisher,
                        "repository": source["repository"],
                        "commit": source["commit"],
                        "bundle_version": str(manifest.get("version", "")),
                        "manifest": {"path": (Path(source["base"]) / manifest_path.name).as_posix(),
                                     "sha256": manifest_sha},
                        **profile,
                    }
                }
            except (KeyError, InvalidProfile) as exc:
                yield {"quarantine": {
                    "publisher": publisher,
                    "model": item.get("name"),
                    "manifest": manifest_path.name,
                    "problem": str(exc),
                }}


def ini_revision(file: Path):
    match = re.fullmatch(r"(\d+)\.(\d+)\.(\d+)", file.stem)
    return tuple(int(part) for part in match.groups()) if match else (-1, -1, -1)


def read_ini_sections(path: Path):
    sections = {}
    active = None
    with path.open(encoding="utf-8-sig", errors="replace") as handle:
        for line in handle:
            head = HEADER.match(line)
            if head:
                active = head.group(1)
                sections[active] = {}
            elif active:
                setting = ATTR.match(line)
                if setting:
                    sections[active][setting.group(1)] = setting.group(2)
    return sections


def scan_prusa(source: dict, checkout: Path):
    root = checkout / source["base"]
    if not root.is_dir():
        raise FileNotFoundError(str(root))
    vendors = ([("Prusa Research", root)] if source["key"] == "prusa_official"
               else [(ALIAS.get(x.name, x.name), x) for x in root.iterdir()
                     if x.is_dir() and x.name != ".git"])
    for brand, folder in sorted(vendors):
        choices = [x for x in folder.glob("*.ini") if ini_revision(x)[0] >= 0]
        if not choices:
            continue
        chosen = max(choices, key=ini_revision)
        relative = chosen.relative_to(checkout).as_posix()
        content = chosen.read_bytes()
        sections = read_ini_sections(chosen)
        printers = [values for section, values in sections.items()
                    if section.startswith("printer:")]
        for section, values in sections.items():
            if not section.startswith("printer_model:") or values.get("technology", "FFF") not in ("FFF","FDM"):
                continue
            name = values.get("name")
            try:
                nozzle = diameters(values.get("variants"))
                if not name or not nozzle:
                    raise InvalidProfile("Prusa model missing variant/nozzle")
                linked = sum(p.get("printer_model") == section.split(":",1)[1] for p in printers)
                yield {"manufacturer": brand, "name": name,
                       "candidate": {
                           "provider": source["provider"],
                           "publisher": source["key"],
                           "repository": source["repository"],
                           "commit": source["commit"],
                           "bundle_version": chosen.stem,
                           "manifest": {"path": relative, "sha256": sha(content)},
                           "model": {"path": relative, "section": section, "sha256": sha(content)},
                           "nozzles_mm": nozzle,
                           "candidate_printer_sections": linked,
                           "variants": [],
                           "validation": "index_only",
                           "warnings": ["INI dependency resolution not yet verified"],
                       }}
            except InvalidProfile as exc:
                yield {"quarantine": {"publisher": source["key"],
                                      "model": name, "problem": str(exc)}}


def combine(incoming):
    models, quarantine = {}, []
    for item in incoming:
        if "quarantine" in item:
            quarantine.append(item["quarantine"])
            continue
        try:
            brand, name = item["manufacturer"], item["name"]
            key = model_key(brand, name)
            if key not in models:
                models[key] = {
                    "id": key, "manufacturer": brand, "model": model_label(brand, name),
                    "release_date": None, "release_date_evidence": None,
                    "conversion_tested": False,
                    "profile_sources": [],
                }
            model = models[key]
            candidate = item["candidate"]
            dedup = (candidate["publisher"], candidate["manifest"]["path"])
            if any((existing["publisher"], existing["manifest"]["path"]) == dedup
                   for existing in model["profile_sources"]):
                quarantine.append({"publisher": candidate["publisher"],
                                   "model": name, "problem": "duplicate model in source"})
                continue
            model["profile_sources"].append(candidate)
        except InvalidProfile as exc:
            quarantine.append({"input": item, "problem": str(exc)})
    for model in models.values():
        model["profile_sources"].sort(key=lambda c: (
            PRIORITY[c["provider"]],
            c["validation"] != "structural_verified",
            c["publisher"],
            c["manifest"]["path"]
        ))
        preferred = next((c for c in model["profile_sources"]
                          if c["validation"] == "structural_verified"), None)
        model["preferred_profile"] = (
            {"provider": preferred["provider"], "publisher": preferred["publisher"],
             "manifest": preferred["manifest"]["path"],
             "nozzles_mm": sorted({v["nozzle_mm"] for v in preferred["variants"]}, key=float)}
            if preferred else None
        )
        # Resolve each physical nozzle separately: original manufacturer can
        # win at 0.4mm while Orca safely fills a missing 0.6mm profile.
        by_nozzle = {}
        for candidate in model["profile_sources"]:
            if candidate["validation"] != "structural_verified":
                continue
            for variant in candidate["variants"]:
                diameter = variant["nozzle_mm"]
                if diameter not in by_nozzle:
                    by_nozzle[diameter] = {
                        "provider": candidate["provider"],
                        "publisher": candidate["publisher"],
                        "manifest": candidate["manifest"]["path"],
                        "machine": variant["machine"]["name"],
                        "process": variant["process"]["name"],
                        "filament": variant["filament"]["name"],
                    }
        model["preferred_by_nozzle"] = dict(sorted(by_nozzle.items(), key=lambda x: float(x[0])))
        model["status"] = "structural_verified" if preferred else "index_only"
    return sorted(models.values(), key=lambda x: (x["manufacturer"].casefold(), x["model"].casefold())), quarantine


def build(sources_root: Path, target: Path, update_lock: bool = False):
    previous = target / "sources.lock.json"
    existing = json.loads(previous.read_text(encoding="utf-8")) if previous.exists() else None
    pinned = {x["key"]: x for x in existing["sources"]} if existing else {}
    incoming, sources = [], []
    for key, provider, repo, base in SOURCES:
        checkout = sources_root / key
        if not (checkout / ".git").exists():
            raise FileNotFoundError(f"Missing pinned git checkout: {checkout}")
        commit = subprocess.check_output(["git","-C",str(checkout),"rev-parse","HEAD"],
                                         text=True).strip()
        if not re.fullmatch(r"[0-9a-f]{40}", commit):
            raise InvalidProfile("unreadable git revision")
        if existing and not update_lock:
            if key not in pinned or pinned[key]["commit"] != commit:
                raise InvalidProfile("upstream revision changed for " + key + "; review changes and pass --update-lock")
        source = {
            "key": key, "provider": provider, "repository": repo, "commit": commit,
            "base": base, "license_status": "review_before_redistribution",
        }
        sources.append(source)
        ingester = scan_prusa if key.startswith("prusa_") else scan_orca_json
        incoming.extend(ingester(source, checkout))
    models, quarantine = combine(incoming)
    grouped = defaultdict(list)
    for model in models:
        grouped[model["manufacturer"]].append(model)
    files = []
    makers_dir = target / "data" / "makers"
    makers_dir.mkdir(parents=True, exist_ok=True)
    # This directory is generated output, not editable user state.
    for stale in makers_dir.glob("*.json"):
        stale.unlink()
    used_files = set()
    for brand, items in sorted(grouped.items()):
        slug = re.sub(r"[^a-z0-9]+","-",brand.casefold()).strip("-")
        file = makers_dir / (slug + ".json")
        if file.name in used_files:
            raise InvalidProfile("manufacturer output name collision: " + file.name)
        used_files.add(file.name)
        file.write_text(json.dumps({"schema_version": 1, "manufacturer": brand, "models": items},
                                   ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
        files.append({"manufacturer": brand, "path": "makers/" + file.name,
                      "models": len(items),
                      "structurally_verified": sum(x["status"] == "structural_verified" for x in items),
                      "sha256": sha(file.read_bytes())})
    manifest = {
        "schema_version": 1, "format": "slicebridge-printer-profile-registry",
        "source_priority": ["manufacturer", "orca", "prusa"],
        "selection_rule": "highest-ranked structurally verified candidate",
        "models": len(models), "manufacturers": len(grouped),
        "structurally_verified": sum(x["status"] == "structural_verified" for x in models),
        "conversion_tested": 0,
        "maker_files": files,
    }
    (target / "data"/"index.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    (target / "sources.lock.json").write_text(
        json.dumps({"schema_version": 1, "sources": sources}, indent=2) + "\n", encoding="utf-8"
    )
    (target / "data"/"audit.json").write_text(
        json.dumps({"schema_version": 1, "quarantined": len(quarantine),
                    "examples": quarantine[:120]}, indent=2, ensure_ascii=False) + "\n", encoding="utf-8"
    )
    summary = {"models": manifest["models"], "manufacturers": manifest["manufacturers"],
               "structurally_verified": manifest["structurally_verified"],
               "quarantined": len(quarantine),
               "source_candidates": dict(Counter(c["publisher"] for m in models for c in m["profile_sources"]))}
    print(json.dumps(summary, indent=2))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--sources-root", type=Path, required=True)
    parser.add_argument("--output", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--update-lock", action="store_true",
                        help="Explicitly authorize a reviewed upstream profile revision update")
    args = parser.parse_args()
    build(args.sources_root.resolve(), args.output.resolve(), args.update_lock)


if __name__ == "__main__":
    main()
