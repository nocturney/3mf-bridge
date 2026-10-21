# Changelog

Published development builds are listed under [GitHub Releases](https://github.com/nocturney/slicebridge/releases). They are experimental pre-releases, not stable versions. Unmerged branch work is not considered released.

## Unreleased

- Future development changes will be listed here.

## [0.4.0-beta.4.dev.12] — 2026-10-21 (development milestone; release status: see GitHub Releases)

- The manual MakerWorld beta extension installer is greyed out for normal users. It becomes available only after an explicit Developer Mode opt-in in About. Normal users should not attempt an unpacked extension install.
- README (English and Hebrew) and the extension installation guide state that the browser-store extension is not officially listed, and that the beta manual install requires advanced browser setup.
- Files downloaded from MakerWorld and manual 3MF import are unchanged.
- Native development version set to `0.4.0-beta.4.dev.12`.

## [0.4.0-beta.4.dev.11] — 2026-10-21 (development milestone; release status: see GitHub Releases)

- Removed labels through [PR #34](https://github.com/nocturney/slicebridge/pull/34).
- Added a source family detector through [PR #35](https://github.com/nocturney/slicebridge/pull/35) to identify the slicer family of a source project.
- Added independent lossless QA through [PR #36](https://github.com/nocturney/slicebridge/pull/36).
- Added generic Orca-family conversion through [PR #37](https://github.com/nocturney/slicebridge/pull/37). It uses a locally installed machine, process and filament profile set that matches. No other Orca profile combination is inferred.
- Added generic PrusaSlicer project output through [PR #39](https://github.com/nocturney/slicebridge/pull/39), using the bundled MIT-licensed Python converter Panda2Prusa. The output is a PrusaSlicer 3MF project. You choose the printer and filament profile in PrusaSlicer; SliceBridge does not map a specific Prusa printer and filament profile automatically.
- Bambu Studio and MakerWorld remain source formats; Bambu Lab is not a conversion target. The 603-model catalogue is for discovery and selection, not 603 validated conversions. The four historical targets are unchanged.
- Added local installation discovery for 22 desktop slicers from popular manufacturers on Windows and macOS, including Creality, Anycubic, Prusa, QIDI, ELEGOO, Flashforge, Snapmaker, Sovol, FLSun and other catalogue makers. UltiMaker and Raise3D were excluded at the owner's request. My Printers shows detected application names without certification badges. Presence alone never enables per-model conversion; detection does not modify slicers or profiles. [PR #40](https://github.com/nocturney/slicebridge/pull/40).
- Physical print validation is not claimed for this milestone. Check printer and slicer settings before printing.
- Native development version set to `0.4.0-beta.4.dev.11`. The published build, if available, is listed under GitHub Releases and must match the tagged source commit.

## [0.4.0-beta.4.dev.10] — 2026-10-21 (development milestone; release status: see GitHub Releases)

- Corrected the dev.9 palette behavior at user request through [PR #32](https://github.com/nocturney/slicebridge/pull/32): every source color/material filament ID remains a **separate virtual project slot** in converted 3MF, even when the printer has only four physical tools. No implicit merging of identical or distinct source colors/materials. Users make any necessary merges/removals inside the destination slicer before printing.
- Preserved per-triangle paint and per-object filament references, source material labels and colors, printer-specific manufacturer profile safety checks, native settings isolation and G-code-cache sanitation. Output highlights extra virtual channels for manual review; local vendor filament presets remain required.
- Regression QA: source project with five independent colors was converted to all four existing printer targets with installed vendor profiles in an isolated Windows host; archive readback confirmed five palette entries, original painting/object assignments and no stale machine code. Extra tests cover a fifth PETG material, duplicate colors that must stay separate, six virtual slots and source immutability. All exact-head Windows/macOS CI checks passed on PR #32. Target-slicer GUI round-trip of extra virtual slots and physical print validation are **not claimed**.
- The published [dev.9](https://github.com/nocturney/slicebridge/releases/tag/v0.4.0-beta.4.dev.9) binaries were built **before** PR #32 merged and retain the earlier slot-coalescing behavior. Do not use dev.9 for the new all-channels-preserved behavior. dev.10 is the first development version intended to contain it; exact download/release status is shown on GitHub Releases.

## [0.4.0-beta.4.dev.9] — 2026-10-21 (development milestone; release status: see GitHub Releases)

- Improved Windows and macOS slicer discovery through [PR #29](https://github.com/nocturney/slicebridge/pull/29): system-wide and per-user application folders, shared trusted profile location resolver, no automatic edits to saved slicer or filament settings.
- Added strictly lossless consolidation of redundant source material slots through [PR #30](https://github.com/nocturney/slicebridge/pull/30): more than four declared Bambu filament IDs may map into four hardware slots only when excess IDs have exactly the same color and material. Distinct colors, distinct materials and mixed source nozzle sizes remain guarded; no silent nearest-color mapping.
- New Go and bilingual UI regression tests; both original PRs passed GitHub Windows and macOS native build/test checks. On an isolated Windows Chris test environment at merged source d7df76902f201692bbc48ea180047ce19b2150f3, all eight actual installed-profile conversions for two test projects across Snapmaker U1, Flashforge Creator 5, ELEGOO CC2 and QIDI Plus5 passed. This is backend integration QA, **not** a full graphical slicer-launch or physical print validation.
- Source projects, existing per-printer profiles and user settings remain unchanged. The 603-model catalogue still has only four implemented converter adapters.

## [0.4.0-beta.4.dev.8] — 2026-10-21 (published experimental pre-release)

- Prepared an open-source GPL-3.0-only license for project-owned code and synchronized prototype npm license metadata.
- Documented separate third-party, vendor-profile and trademark scopes without claiming ownership of upstream works.
- Established the SliceBridge product name and a concise bilingual README.
- Published a pre-release landing page in English and Hebrew using the approved Mint / Forest artwork.
- Added GitHub project metadata, reporting and documentation conventions.
- Published experimental Windows and macOS development builds on GitHub Releases.
- Simplified the release guidance for a small open-source project; signing is optional.

- Published [v0.4.0-beta.4.dev.8](https://github.com/nocturney/slicebridge/releases/tag/v0.4.0-beta.4.dev.8) as an experimental Windows/macOS Apple Silicon pre-release built from exact merged source `87de87cfbcbc40813e11e9e1c7322f084fed7f8f`, with GPL license, third-party notices, corresponding source archive and SHA-256 checksums. It is not stable.
- PRs #1–#6 and #8 are merged into `main`, and PR #13's My Printers/UI updates reached `main` through #6. The result includes four conversion targets and a separate 603-model reference catalogue (not 603 conversion adapters). Code integration does not automatically publish an installer.
- Version metadata was merged through [PR #23](https://github.com/nocturney/slicebridge/pull/23); GitHub native Windows/macOS CI and release-preflight checks passed on the exact released source commit. Source `native/updates_bridge.go` reports `dev.8`, with regression tests guarding drift from `VERSION`.
- Corrected the desktop packaging description, which still said “Beta 3” in macOS Finder; the native About panel retains the exact dev.8 build version and the OS bundle version remains numeric 0.4.0.
- The released binaries embed the same tagged source commit and passed ZIP integrity and bounded application-launch smoke checks on both platforms. No full slicer, printer, Chromium-extension, or macOS URL-handler takeover acceptance test is claimed.

Historical dev.0 through dev.6 headings describe the state at the time they were written. All dev.1 through dev.6 packages were subsequently published as experimental pre-releases; the actual GitHub Releases list is authoritative.

## [0.4.0-beta.4.dev.7] — 2026-10-20 (published experimental pre-release)

- Published [v0.4.0-beta.4.dev.7](https://github.com/nocturney/slicebridge/releases/tag/v0.4.0-beta.4.dev.7) with experimental Windows/macOS downloads and corresponding source `986161df3cdc9485e71bda2bbcd9a4bcd877dab7`. This **does not include** code merged into `main` on October 21, 2026, or any later planned dev.8 changes.

## [0.4.0-beta.4.dev.6] — 2026-10-20 (development milestone, not released)

- Correct the `dev.5` source-panel overexpansion: **one or two files no longer push the selected target-printer cards out of view**. The file panel now sizes itself by the number of rows rather than reserving a fixed minimum height of 390 pixels.
- Keep the source list readable (wrapped filenames, folder, size and remove button), cap only *long* queues at a 196 px internal scroll area, and remove the duplicated selected-file heading and footnote while files are queued. The source heading already shows the count.
- Reclaim vertical space in shorter desktop windows by compacting the header/action/banner/progress rows without removing the fixed banner or changing printer cards, and preserve existing saved window sizes. Fresh-install default remains 1280×900.
- Renderer regression checks now explicitly verify that **both the source file(s) and the target-printer cards are visible without workspace scrolling** at a normal desktop viewport, including the two-file case; retain overflow QA for larger queues and narrow windows.
- App code is in [Draft PR #13](https://github.com/nocturney/slicebridge/pull/13). This milestone alone does not constitute a published Release or independent Wails visual approval.

## [0.4.0-beta.4.dev.5] — 2026-10-20 (development milestone, not released)

- Make attached source files **actually readable in the main desktop converter**, after feedback that a selected 3MF appeared only as a cut-off thin strip: expand the queue panel whenever it has files, provide at least 145 px of scrollable file list, show a selected-files heading and count, and wrap long project filenames across up to two lines while preserving path and size information.
- Replace the left column's cramped grid row with predictable flex-height source and printer sections; let the workspace scroll on smaller window dimensions without cropping the queue. Keep the drop target accessible for subsequent uploads.
- Increase the **new-install default** desktop window from 1100×750 to 1280×900. **Existing manually saved window sizes, printer choices, nozzle and filament preferences remain unchanged**; resized windows continue to persist as before.
- Add structural Node tests, Go preference-preservation tests, and repeatable real-engine rendered-geometry QA with test cases for 1/15 source files, Hebrew/English and small windows. Does not touch conversion behavior or printer hardware.
- Source changes reside in [Draft PR #13](https://github.com/nocturney/slicebridge/pull/13). New binaries must be validated at the exact source SHA and published separately as a clearly marked **Pre-release** after CI; this metadata update is not itself a release.

## [0.4.0-beta.4.dev.4] — 2026-10-19 (development milestone, not released)

- [Draft PR #13](https://github.com/nocturney/slicebridge/pull/13), commit [`a3dd8f3`](https://github.com/nocturney/slicebridge/commit/a3dd8f377d93421884d3f8f77c7a6e56062bde1a): update the in-app banner to **“DIFFERENT SLICERS. ONE SIMPLE WORKFLOW.”** with no unsupported community-marketing claim.
- Commit [`69df829`](https://github.com/nocturney/slicebridge/commit/69df829ef98388f41274e08e7f6574333c54b4e9): simplify MakerWorld browser extension installation for nontechnical users. One main button chooses a **publisher-verified browser-store page** when available; until then, a **guided beta fallback** prepares the bundled extension and opens the selected browser's add-on management page. Developer-mode instructions stay hidden until necessary; consent remains mandatory.
- Prepare separate publisher-only Chromium and Firefox submission ZIPs (`browser-extension/pack_store.py`) for later Edge/Chrome/Firefox store review. These are **not signed/approved store listings** and are not consumer installation downloads.
- Add Hebrew/English install guidance, exact host allowlists, no arbitrary external-link opening, backend route tests and Node UI flow regression tests. Existing settings, conversion logic and Bambu default handler remain unchanged.
- Bump native development version to `0.4.0-beta.4.dev.4`; no app binary or browser-store add-on has been publicly released at this milestone yet.

## [0.4.0-beta.4.dev.3] — 2026-10-19 (development milestone, not released)

- [Draft PR #13](https://github.com/nocturney/slicebridge/pull/13), commit [`4da9ba7`](https://github.com/nocturney/slicebridge/commit/4da9ba75102a36c272ab93551ae6ab47324a161e): add local printer-cover thumbnails to the actual converter target cards using the existing trusted Wails image reader. Only selected legacy adapters request local PNG files; icons remain as fallback. No third-party images redistributed.
- Remove the extra community-themed slogan from the in-app banner, retaining the shorter brand headline.
- Remove the small unwanted scrollbar below the empty 3MF drop area while preserving scrollability for queued files.
- Change the native development build version to `0.4.0-beta.4.dev.3`; this remains **unmerged source code**, pending new Windows/macOS CI binaries, visual QA and a separately verified Pre-release.

## [0.4.0-beta.4.dev.2] — 2026-10-19 (development milestone, not released)

- [Draft PR #13](https://github.com/nocturney/slicebridge/pull/13), commit [`84289a4`](https://github.com/nocturney/slicebridge/commit/84289a43759f820bc2a937f81069409282ea5336): prevent source queue/upload area clipping in short windows by preserving minimum queue height and allowing workspace scrolling.
- Show existing conversion-target printers as responsive **two-column selection cards** (one column on narrow windows), while keeping their existing supported adapters and locally stored preferences unchanged.
- Keep **visible text alongside** the My Printers, Settings and MakerWorld navigation icons. Use a wrapped two-row header when necessary instead of hiding labels. All are **unmerged development changes**, not a stable release.
- Initial local QA: 26 Node UI/extension tests and 40 Python registry tests passed; Wails Windows/macOS builds and visual acceptance must be checked independently.

## [0.4.0-beta.4.dev.1] — 2026-10-19 (development milestone, not released)

- [Draft PR #13](https://github.com/nocturney/slicebridge/pull/13), commit [`068f050`](https://github.com/nocturney/slicebridge/commit/068f050106a5f6d522bb4723f91fc8aab69c833a): no predefined printers or immediate catalogue gallery, choose a manufacturer or search before showing images, and one navigation entry each for printer selection and settings. Existing saved selections remain unchanged; code is **not merged into `main` or released**.
- Native development binary identifies itself as `0.4.0-beta.4.dev.1`; GitHub Actions checks are pending independent QA. This is a test build, not an approved release.

## [0.4.0-beta.4.dev.0] — 2026-10-19 (development milestone, not released)

- Established a single tracked `VERSION`, a stage-by-stage changelog policy and an automated version/changelog contract check. This stage contains repository metadata only; it does not ship a new app binary.
- Kept the existing `v0.4.0-beta.3` draft distinct from public releases and from the next development version.

## Published development builds

- [Development releases](https://github.com/nocturney/slicebridge/releases), including `v0.4.0-beta.4.dev.*`: experimental downloads for community testing.
- `v0.4.0-beta.3`: an older published development preview; not a stable version.

See [the simple community release guide](DISTRIBUTION_CHECKLIST.md) when sharing a future build.
