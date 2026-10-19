# SliceBridge Printer Profile Registry (v1)

This is a **profile index with validation**, not a menu of printer names.
It is deliberately separate from the UI, conversion engine and printer settings.

## Preference: manufacturer > Orca > Prusa

1. Original manufacturer: Bambu Lab / Bambu Studio, QIDI / QIDIStudio,
   ELEGOO / ElegooSlicer, Prusa Research / official Prusa profile bundle.
2. OrcaSlicer if an original source has no structurally verified matching profile.
3. Community/vendor PrusaSlicer bundles, provided they pass the same checks.

A manufacturer's slicer can ship third-party profiles; those are NOT
automatically relabeled "original manufacturer". No candidate wins solely
because its source is official: it must pass compatibility and integrity gates.

## Contents

- data/index.json — list of manufacturers, file checksums, exact model and
  validated counts.
- data/makers/*.json — one document per manufacturer, stable printer IDs,
  model and nozzle declarations, alternative sources, source revisions,
  checksums of machine/process/filament profiles and their inherited presets.
- sources.lock.json — full 40-digit upstream Git commit IDs, not moving HEAD.
- data/audit.json — quarantined source/model entries and error explanations.
- schema-v1.json — published machine-readable registry format.
- tools/build.py — Python stdlib source importer; rejects source revision drift.
- tools/check.py — offline semantic and integrity validation.
- tests/test_catalog.py — fixture tests and committed snapshot regression.

## Meaning of validation states

- index_only: model and nozzle declaration exists, but has no complete
  validated chain. Not eligible as a conversion profile.
- structural_verified: at least one machine variant exists with a nozzle
  matching the parent model and valid print-process and filament-default
  references. JSON and all referenced process/filament inheritance chains
  parse, exist and are SHA-256 recorded. Broken sources are quarantined.
- conversion_tested: reserved for independent real 3MF import/conversion QA;
  currently FALSE for every catalogued model.

**Structural verification does NOT mean correct generated 3MF, reliable
machine behavior, valid print G-code, actual slicer compatibility, or a
successful print.** Do not auto-enable conversion for new catalogue entries
without a real tested adapter for the requested printer, nozzle and material.

Prusa INI model profiles are currently indexed only; complete inheritance
and process/filament validation is not yet implemented for INI format.
Release dates are intentionally null until supported by manufacturer evidence.

## Rebuild and test

Python 3.11+, standard library; no slicer is invoked and nothing is printed.

    python catalog/tools/build.py --sources-root PATH_TO_CHECKOUTS
    python catalog/tools/check.py catalog
    python -m unittest discover -s catalog/tests -v

The parent folder needs pinned Git checkouts named:
bambu, qidi, elegoo, orca, prusa_official and prusa_community.
Each checkout must be at the commit in sources.lock.json. A normal rebuild
rejects unexpected SHA updates. After reviewing an upstream change, use
--update-lock to authorize an intentional new snapshot.

Only catalogue-owned data/makers/*.json files are rewritten.
There are no user profiles, conversion settings or UI files in this branch.

## Source lineage and redistribution

These source repositories are indexed at the specific commits recorded in
sources.lock.json. The profile file contents are not bundled or mirrored here;
we store paths, references, hashes and minimal metadata.

- Manufacturer Bambu Lab: https://github.com/bambulab/BambuStudio
- Manufacturer QIDI: https://github.com/QIDITECH/QIDIStudio
- Manufacturer ELEGOO: https://github.com/elegooofficial/ElegooSlicer
- Alternate Orca: https://github.com/OrcaSlicer/OrcaSlicer
- Manufacturer Prusa Research: https://github.com/prusa3d/PrusaSlicer-settings
- Prusa non-manufacturer community bundles: https://github.com/prusa3d/PrusaSlicer-settings-non-prusa-fff

Licensing and attribution are not yet cleared for redistribution.
Do not mirror/download bundled payloads to end users until licensing,
manufacturer assets and attribution requirements are reviewed.

Next phases: full Prusa INI resolver, official Snapmaker/Flashforge sources,
source metadata/license review, compatibility adapters, and actual per-model
slicer/3MF QA. Manufacturer release dates are separate, sourced metadata.
