# Changelog

Published development builds are listed under [GitHub Releases](https://github.com/nocturney/slicebridge/releases). They are experimental pre-releases, not stable versions. Unmerged branch work is not considered released.

## Unreleased

- Prepared an open-source GPL-3.0-only license for project-owned code and synchronized prototype npm license metadata.
- Documented separate third-party, vendor-profile and trademark scopes without claiming ownership of upstream works.
- Established the SliceBridge product name and a concise bilingual README.
- Published a pre-release landing page in English and Hebrew using the approved Mint / Forest artwork.
- Added GitHub project metadata, reporting and documentation conventions.
- Published experimental Windows and macOS development builds on GitHub Releases.
- Simplified the release guidance for a small open-source project; signing is optional.

## [0.4.0-beta.4.dev.8] — 2026-10-21 (planned next development milestone, NOT RELEASED)

- This is a planned next source/build number, not an available download. The latest published pre-release is [v0.4.0-beta.4.dev.7](https://github.com/nocturney/slicebridge/releases/tag/v0.4.0-beta.4.dev.7), built from source 986161df3cdc9485e71bda2bbcd9a4bcd877dab7 on October 20, 2026.
- PRs #1–#6 and #8 are now merged into `main`, including the four conversion targets, MakerWorld integration and the 603-model reference catalogue (not 603 conversion adapters). Code integration does not automatically publish an installer.
- Future dev.8 binaries require an approved source SHA, matching Windows/macOS builds, GPL and notices, and SHA256 checksums. VERSION alone cannot publish a stable release.

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
