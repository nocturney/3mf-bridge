# Changelog

Public, published releases will be listed here. Development-branch work and draft releases do not count as publicly available versions.

## Unreleased

- Prepared an open-source GPL-3.0-only license for project-owned code and synchronized prototype npm license metadata.
- Documented separate third-party, vendor-profile and trademark scopes without claiming ownership of upstream works.
- Established the SliceBridge product name and a concise bilingual README.
- Published a pre-release landing page in English and Hebrew using the approved Mint / Forest artwork.
- Added GitHub project metadata, reporting and documentation conventions.
- Published development status; no current downloadable community build yet.
- Simplified the release guidance for a small open-source project; signing is optional.

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

## Draft artifacts (not released)

- `v0.4.0-beta.3`: GitHub draft release. **Not published**; an older development preview, not a current community download.

See [the simple community release guide](DISTRIBUTION_CHECKLIST.md) when sharing a future build.
