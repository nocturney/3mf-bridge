# SliceBridge v1 printer settings → catalogue ID bridge

This is **a read-only metadata contract**, not a migration of any customer
settings, not a slicer discovery implementation, and not a conversion release.
It is derived from the Wails code on
[feature/browser-compat-and-icon at 3bf2928](https://github.com/nocturney/slicebridge/tree/3bf292876406edb891351fe160ca146532299014/native)
(`native/presets/presets.go`, `native/preferences.go`, `native/batch.go`).
The catalogue lives in a separate Draft PR and **does not** import or modify
these Wails source files.

## Exact legacy identity mapping

| Stored target key | Existing device | Catalogue model ID | Legacy target nozzle options |
| --- | --- | --- | --- |
| `u1` | Snapmaker U1 | `snapmaker-u1` | 0.2 / 0.4 / 0.6 / 0.8 mm |
| `c5` | Flashforge Creator 5 | `flashforge-creator-5` | 0.25 / 0.4 / 0.6 / 0.8 mm |
| `cc2` | ELEGOO Centauri Carbon 2 | `elegoo-centauri-carbon-2` | 0.2 / 0.4 / 0.6 / 0.8 mm |
| `qidi` | QIDI X-Plus 5 | `qidi-x-plus-5` | 0.2 / 0.4 / 0.6 / 0.8 mm |

Special cases found in the existing Wails converter:
- **Creator 5 source 0.2 mm → actual target 0.25 mm**, preserving the
  existing owner-approved legacy nozzle remap. No other remaps are authorized.
- Snapmaker U1 has an existing **0.4 mm built-in fast path** when slicer
  preference is Automatic and there are no custom filament overrides.
  Other combinations require installed local profiles.
- Snapmaker U1 **mixed 0.4+0.6** is in the metadata registry but **not in
  the existing single-nozzle conversion adapter**. It must not be auto-enabled.
- Flashforge Creator 5's structurally indexed source currently comes from
  OrcaSlicer; that does *not* imply Flash Studio can import it, or that
  Flashforge provided an original profile for Creator 5.
- Index entries for other models **must be tagged "Catalog only"**. They have
  no existing SliceBridge adapter even when a valid preset chain exists.

## Release and preservation rules

The existing application stores version **1** settings keyed by
`u1`, `c5`, `cc2`, `qidi`, with per-printer `nozzle`, `slicer` and
`filaments` overrides. Those names and saved values are authoritative
until the UI/backend migration is separately reviewed and tested. **Do not
overwrite, delete, silently remap user saved settings or enable catalogued
printer models based on this file.**

`data/legacy-adapters.json` is an offline capability/identity ledger.
`tools/legacy_adapters.py` verifies that each entry points to exactly
one real, structurally indexed model and allowed *physical* nozzle values,
that the four old keys are unique, that unknown models default to
`catalog_only`, and that metadata never asserts a conversion is
tested or turned on. It can create an **in-memory migration preview** from
fixture data, preserving custom/nozzle/slicer/filament preferences and unknown
fields without filesystem access.

**Separate follow-up work in Wails branch**:
1. Embed the vetted metadata snapshot as a read-only resource.
2. Look up local slicer/profile presence using the existing safe resolver,
   retain old working U1 fallback and the Creator 5 remap.
3. Upgrade saved settings atomically with versioned rollback/migration tests.
4. Add first-run setup, "My printers" add/remove/search and explicit status:
   installed supported adapter / missing local profile / catalog-only.
5. Run genuine slicer 3MF opening and physical nozzle/material QA before
   declaring new conversions usable.

No printer actuation or remote profile downloads are part of this contract.
