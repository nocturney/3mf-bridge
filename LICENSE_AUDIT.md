# Preliminary license inventory — source manifests only

**As of 19 October 2026 — NOT a release SBOM or legal sign-off.** The counts below were derived from committed `package-lock.json` data, not binary inspection. They are a snapshot; new changes require a new inventory. A lockfile license field is not itself proof that every bundled notice is satisfied.

## npm dependency lock snapshots

| Committed lockfile | Package entries (excluding root) | Declared licenses |
| --- | ---: | --- |
| `main/package-lock.json` at `0f94026168f3` | 13 | 1 Apache-2.0 · 1 BSD-2-Clause · 2 ISC · 9 MIT |
| `feature/browser-compat-and-icon/package-lock.json` at `aae9595530aa` | 51 | 2 Apache-2.0 · 5 BlueOak-1.0.0 · 4 BSD-2-Clause · 5 ISC · 35 MIT |

**Direct npm development tools:** prototype `electron ^44.8.0`; development also declares `@electron/packager ^20.3.0`. Neither count is an inventory of Go/Wails dependencies, native app packaging, browser extension assets, or remote build tools.

## Go/Wails and conversion components

- `native/go.mod` on the development branch: Go 1.25, `github.com/wailsapp/wails/v2`, `golang.org/x/mod`, additional indirect modules and local replacements. Each Go module's **exact-version license and packaged notices still require review**.
- `engine/go.mod` on the development branch: Go 1.24.7, depends on `BryanSmee/bl_to_std` via pinned local submodule. Confirm the upstream **GPLv3** license and preserve upstream copyright, source and build instructions.
- The website's project-approved icon and banner and all generated app/extension icons should be checked for artwork rights and trademark usage before redistribution.
- Vendor printer profiles are designed to be read from user's local slicer installations; **notices do not grant permission to republish them**. The proposed catalogue's index/hash metadata needs a separate provenance/rights audit before release.

## Required before publishing a binary

1. Build from a precise source commit; record submodule SHAs and dependency locks used.
2. Generate a version-specific SBOM from **the actual distributed binary/dependency graph**, not just this snapshot; include original license notices and corresponding source where required.
3. Review GPLv3 obligations for the combined converter, Wails/Electron packaging, installers, webview/browser extension and user-visible third-party artwork; retain required credits.
4. Verify vendor asset exclusion and downstream project data licensing. A profile path or hash reference does not authorize distribution of the underlying file.
5. Confirm clear `LICENSE`, `THIRD_PARTY_NOTICES.md`, copyright and trademark permissions and independent QA before declaring rights clearance complete.

Tracking: [legal/license #9](https://github.com/nocturney/slicebridge/issues/9) · [public release #11](https://github.com/nocturney/slicebridge/issues/11) · [LICENSING.md](LICENSING.md).

## Go/Wails follow-up (19 October 2026)

Examined 32 manifest entries, including 30 external version-specific module ZIPs with candidate license files and two local replacements. The preliminary classifications and remaining limitations are in [GO_LICENSE_AUDIT.md](GO_LICENSE_AUDIT.md). Draft ZIP inspection and approved artwork hash inventory are in [RELEASE_PREFLIGHT.md](RELEASE_PREFLIGHT.md).
