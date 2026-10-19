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
