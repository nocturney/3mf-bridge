# SliceBridge catalogue: release boundaries and at-home slicer QA

## Community beta: one engine family, representative QA

For experimental development builds, **do not require a separate full
slicer test for every rebranded or closely related slicer**. Prefer a
representative import/conversion check per shared 3MF/slicing engine
family, then check *known* differences (machine profile, extruders,
nozzles, G-code dialect and materials). Accept targeted user bug reports
to refine compatibility instead of holding up every beta for exhaustive
combinatorial testing.

Show other slicer variants as **experimental / not individually
verified** until tested. A shared engine is a strong compatibility
indicator, but does not certify a vendor's printer presets or physical
hardware. The four existing conversion adapters remain the only working
conversion targets; the other catalogue entries are still **catalogue
only**, not automatically activated or described as supported converters.
Never silently change user slicer/nozzle settings or start a physical
print. Actual model-specific verification can be added based on reported
problems or whenever a stronger compatibility claim is needed.


## What can ship without physical slicer QA

**Data-only candidate:** the metadata registry in PR #8 has no converter
adapter, no feature activation, no profile payload mirror and no UI changes.
A reviewer may approve and merge the **inert index** after verifying the
exact head, pinned source hashes, legal provenance, and metadata CI. This
does not create a SliceBridge desktop release or authorize printer support.

GitHub Actions "Printer profile registry QA" builds a metadata-only archive
only after Windows, macOS and Linux tests pass. Download the ZIP and its
.sha256 file from that run's **Artifacts**. They contain only JSON metadata,
a JSON schema, source-lock file and integrity manifest; no raw upstream
machine/filament presets, 3MF, STL, G-code, or executable code. Retention:
30 days. The run is **not** a release or an update feed.

To check an artifact offline:

1. Verify sha256sum --check CATALOG_ZIP.sha256 in the archive directory.
2. Inspect catalog/bundle-manifest.json inside the ZIP.
3. Confirm bundle_kind is printer-profile-metadata-only,
   conversion_tested_models is 0 and source_profile_payloads_included is false.
4. The archive itself is reverified during packaging; no client-side automatic
   activation is implemented. Do not point a public app to a PR artifact URL.

**Do not merge a PR authored by the same agent without independent review,
and do not release/enable a converter based on metadata CI alone.**

## Explicit remaining product-release gates

- [ ] independent review of exact PR head and upstream attribution/licensing
- [ ] verify selected printer adapters and existing saved-preference migration
- [ ] actual slicer import of produced 3MF projects on Windows/macOS
- [ ] intended slicers launch and select correct nozzle/material/print profiles
- [ ] MakerWorld installed-browser + custom protocol E2E (separate PR #6)
- [ ] signing/notarization/installer tests and downgrade/rollback
- [ ] full release QA verdict and approval

## At-home test plan — read-only previews, no printer movement

Record **app commit, source 3MF, slicer name/version, OS, target printer,
selected nozzle(s), materials and a screenshot of the actual opened slicer
project**. A CLI success code or JSON source-reference check is insufficient.

Use one known Bambu source project with distinct colors and another with
supports/complex geometry. For each relevant target below:

| Target | Intended project checks | Known registry limitation |
| --- | --- | --- |
| Snapmaker U1 | all bodies, assigned extruders, four heads, 0.4 and 0.4+0.6 configs, rotation and sizes | manufacturer bundle U1 process unresolved; Orca structural fallback only |
| Flashforge Creator 5 | colors, tool mapping, 3MF opening, 0.4 preset; native and Orca separately if installed | current FlashForge source omits Creator 5; Orca is not OEM |
| ELEGOO Centauri Carbon 2 | size, PLA preset, filament mapping and support settings | structural validation does not certify produced 3MF |
| QIDI Plus5 / X-Plus 5 | correct machine, nozzle, print process, filament, G-code dialect setting in preview | exact model aliases must be checked with actual slicer |

For each imported project:
1. Confirm slicer opens without "unsupported/invalid project" errors.
2. Compare geometry, orientation, bounding box, plate placements and item
   count against the source.
3. Confirm individual material/color-to-tool mapping is preserved; report
   every silent reassignment. Check supports, modifiers and process presets.
4. Record missing/changed profile, missing filament, nozzle mismatch and
   app version. Treat any unexplained warning as FAIL.
5. Do **not** start a print or upload G-code as part of catalogue QA.
6. Add an exact model/nozzle/slicer/version/3MF evidence record before
   setting a model/variant's conversion_tested status to true.

No release should claim "all 603 printers supported": 603 are indexed,
not 603 working converters. Keep last-known-good settings and the currently
supported four adapter configurations until measured regression is green.
