"""PrusaSlicer INI preset-chain resolver (pure metadata, no slicing).

Each profile reference points to a pinned vendor bundle INI and its section.
A resolved chain proves syntax and referenced sections, not G-code safety.
"""
from __future__ import annotations

import csv
import re
from collections import Counter


class InvalidIniProfile(ValueError):
    pass


def decode_value(value: str) -> str:
    """Remove only surrounding syntactic quotes, retaining interior content."""
    value = value.strip()
    if len(value) >= 2 and value[0] == value[-1] == '"':
        return value[1:-1]
    return value


def declared_filaments(value: str) -> list[str]:
    if not value or not value.strip():
        return []
    try:
        result = next(csv.reader([value], skipinitialspace=True, strict=True))
    except csv.Error as exc:
        raise InvalidIniProfile("malformed filament list") from exc
    parsed = [decode_value(v) for v in result]
    if not parsed or any(not name for name in parsed):
        raise InvalidIniProfile("missing default filament")
    return parsed


def inherited_parent_names(value: str) -> list[str]:
    # Prusa upstream uses semicolon-separated multiple inheritance in
    # printers, print profiles, and material profiles.
    parents = [decode_value(p) for p in value.split(";") if p.strip()]
    if len(parents) > 32:
        raise InvalidIniProfile("unbounded parent preset list")
    return parents


def sections_of_type(sections: dict, prefix: str) -> dict:
    return {key.split(":", 1)[1]: value
            for key, value in sections.items() if key.startswith(prefix + ":")}


def resolve(preset: str, profiles: dict, kind: str, cache: dict,
            visiting: tuple[str, ...] = ()) -> tuple[dict, list[str]]:
    """Resolve ordered inherited overlays, reject cycles and absent parents.

    Prusa uses left-to-right precedence with the last parent overriding
    earlier parent keys, then the concrete child overrides its parents.
    """
    if preset in cache:
        return cache[preset]
    if preset in visiting or len(visiting) >= 32:
        raise InvalidIniProfile(kind + " inheritance cycle")
    if preset not in profiles:
        raise InvalidIniProfile(kind + " missing preset: " + preset)
    obj = profiles[preset]
    config: dict = {}
    ancestry: list[str] = []
    parents = inherited_parent_names(obj.get("inherits", ""))
    for parent in parents:
        parent_props, chain = resolve(parent, profiles, kind, cache, visiting + (preset,))
        config.update(parent_props)
        for inherited_name in chain:
            if inherited_name not in ancestry:
                ancestry.append(inherited_name)
    config.update({key: value for key, value in obj.items() if key != "inherits"})
    ancestry.append(preset)
    cache[preset] = config, ancestry
    return cache[preset]


def matching_nozzle(config: dict, available: set[str]) -> str:
    variant_text = decode_value(config.get("printer_variant", ""))
    nozzle_text = decode_value(config.get("nozzle_diameter", ""))
    if not variant_text or not nozzle_text:
        raise InvalidIniProfile("printer variant/nozzle not fully defined")
    try:
        variant = float(variant_text)
        values = [float(token.strip()) for token in re.split("[,;]", nozzle_text) if token.strip()]
    except ValueError as exc:
        raise InvalidIniProfile("unparseable machine nozzle") from exc
    if not values or any(not 0.1 <= v <= 1.2 for v in values):
        raise InvalidIniProfile("machine nozzle outside supported range")
    if any(abs(v - variant) > 1e-8 for v in values):
        raise InvalidIniProfile("physical nozzle and model variant differ")
    diameter = f"{variant:g}"
    if diameter not in available:
        raise InvalidIniProfile("nozzle absent from model declaration")
    return diameter


def section_ref(path: str, file_hash: str, kind: str, preset: str,
                chain: list[str]) -> dict:
    return {
        "name": preset, "path": path, "sha256": file_hash,
        "section": kind + ":" + preset,
        "inheritance": [
            {"name": name, "path": path, "sha256": file_hash,
             "section": kind + ":" + name}
            for name in chain
        ],
    }


def validate_model(sections: dict, model_section: str,
                   nozzle_list: list[str], file_path: str, file_hash: str):
    printers = sections_of_type(sections, "printer")
    processes = sections_of_type(sections, "print")
    filaments = sections_of_type(sections, "filament")
    caches = {"printer": {}, "print": {}, "filament": {}}
    wanted_model = model_section.removeprefix("printer_model:")
    variants: list[dict] = []
    errors: Counter = Counter()
    linked = 0
    for printer, props in printers.items():
        if printer.startswith("*"):
            continue
        try:
            config, machine_chain = resolve(printer, printers, "printer", caches["printer"])
            if config.get("printer_model", "").strip() != wanted_model:
                continue
            linked += 1
            if config.get("printer_technology", "FFF") not in ("FFF", "FDM"):
                raise InvalidIniProfile("printer uses unsupported technology")
            nozzle = matching_nozzle(config, set(nozzle_list))
            process = decode_value(config.get("default_print_profile", ""))
            if not process:
                raise InvalidIniProfile("printer missing default print process")
            _, process_chain = resolve(process, processes, "print", caches["print"])
            names = declared_filaments(config.get("default_filament_profile", ""))
            if not names:
                raise InvalidIniProfile("printer missing default filament")
            filament_refs = []
            for filament in names:
                _, fil_chain = resolve(filament, filaments, "filament",
                                       caches["filament"])
                filament_refs.append(section_ref(file_path, file_hash,
                                                 "filament", filament, fil_chain))
            variant = {
                "nozzle_mm": nozzle,
                "machine": section_ref(file_path, file_hash, "printer",
                                       printer, machine_chain),
                "process": section_ref(file_path, file_hash, "print",
                                       process, process_chain),
                "filament": filament_refs[0],
            }
            if len(filament_refs) > 1:
                variant["filaments"] = filament_refs
            variants.append(variant)
        except InvalidIniProfile as exc:
            errors[str(exc)] += 1
    variants.sort(key=lambda x: (float(x["nozzle_mm"]), x["machine"]["name"]))
    return {
        "candidate_printer_sections": linked,
        "variants": variants,
        "validation": "structural_verified" if variants else "index_only",
        "warnings": [f"{msg} ({count})" for msg, count in errors.most_common(5)]
                    if errors else [],
    }
