# Third-party notices / הודעות צד שלישי

This document is a **source inventory and attribution starting point**, not a final bill of materials for a release. Do not strip upstream notices, and do not treat GitHub's public visibility as permission to copy proprietary files.

## bl2std — GPLv3 dependency on the development branch

- Project: [BryanSmee/bl_to_std](https://github.com/BryanSmee/bl_to_std)
- Pinned upstream commit recorded in development: `4252f5f0a880463fd2e6aeed3125ec3653de60ee`
- License: [GNU GPL version 3](https://github.com/BryanSmee/bl_to_std/blob/4252f5f0a880463fd2e6aeed3125ec3653de60ee/LICENSE), including its original copyright notices
- Source: the development branch's `third_party/bl2std` submodule (initialize with `git submodule update --init --recursive`), plus the authoritative upstream project

The conversion branch imports `bl2std/pkg/converter` and `bl2std/pkg/printer`. The relevant GPLv3 obligations for a combined binary and its corresponding source must be met at distribution time. We do not claim authorship of bl2std or its machine templates.

## Installed slicer/printer profiles — NOT redistributed here

SliceBridge is designed to read **user-installed**, compatible target machine, process and filament presets **at conversion time**. Notably:

- [FlashForge/Orca-Flashforge](https://github.com/FlashForge/Orca-Flashforge) / Flash Studio Desktop
- [elegooofficial/ElegooSlicer](https://github.com/elegooofficial/ElegooSlicer) / ELEGOO
- [OrcaSlicer/OrcaSlicer](https://github.com/OrcaSlicer/OrcaSlicer) / compatible installed target profiles
- [Snapmaker/Orca_Presets](https://github.com/Snapmaker/Orca_Presets) / manufacturer-published presets
- [PrusaSlicer](https://github.com/prusa3d/PrusaSlicer) vendor presets / compatibility fallback
- Official slicer applications from Bambu Lab, QIDI, Prusa Research, and other manufacturers where applicable

**Listing a source does not mean it is incorporated, bundled or freely redistributable.** Each source's terms must be evaluated individually. SliceBridge's profile registry proposal stores reference metadata and source hashes, not original preset JSON/INI payloads; validation of authorship, attribution, licensing and redistribution for any new files remains a release blocker. A generated 3MF can contain settings derived from locally installed presets; applicable upstream terms may still matter.

## Runtime and build dependencies

The Electron prototype, Wails runtime, Go, Node packages and browser-extension components each retain their own third-party licenses. Package manifests and lockfiles record dependency versions where available. A release-specific **SBOM (software bill of materials) and third-party license bundle** must be generated for the **exact binaries being shipped** and reviewed before distribution. This document alone is not an exhaustive license list.

## Graphics and product names

The approved SliceBridge icon/banner are included as project identity materials and are **not** adapted vendor logos. The project identity and trademark guidance in [TRADEMARKS.md](TRADEMARKS.md) does not transfer trademark rights from any vendor to SliceBridge.

## Avoid unsupported claims

- Open source is not equivalent to public-domain or warranty-free rights to third-party resources.
- No PolyForm Noncommercial code used during research is claimed to be incorporated in this project; validate that statement again as new PRs are integrated.
- A profile being readable from a local slicer does not establish that redistribution would be allowed.

See [LICENSING.md](LICENSING.md) and [DISTRIBUTION_CHECKLIST.md](DISTRIBUTION_CHECKLIST.md) before any public build.
