# SliceBridge printer registry: third-party sources and attribution

This repository contains a **metadata-only index** derived from openly viewable slicer profile indexes. This document credits the repositories and records the root-repository license evidence. It is not a sublicence and does not grant permissions that the upstream authors did not grant.

**SliceBridge does not copy upstream JSON/INI machine, process or filament profile payloads into its distributed metadata ZIP.** The registry stores names, stable references, pinned Git commit IDs, paths and checksums. The user's locally installed slicer/profile data remains the authority for actual conversion. The app must not imply that a catalogued printer is supported unless its adapter has been independently tested.

Source versions and links below are generated from / verified against `sources.lock.json`; a machine-readable counterpart is shipped in `data/attribution.json`.

| Source publisher | Upstream repository | Observed root license evidence | Profile payload reuse |
| --- | --- | --- | --- |
| Bambu Lab | [BambuStudio](https://github.com/bambulab/BambuStudio) | `LICENSE`: AGPL-3.0 | Not included; asset-specific rights not assessed |
| QIDI | [QIDIStudio](https://github.com/QIDITECH/QIDIStudio) | `LICENSE`: AGPL-3.0 | Not included; asset-specific rights not assessed |
| ELEGOO | [ElegooSlicer](https://github.com/elegooofficial/ElegooSlicer) | `LICENSE.txt`: AGPL-3.0 | Not included; asset-specific rights not assessed |
| Snapmaker | [Orca_Presets](https://github.com/Snapmaker/Orca_Presets) | No repository-level licence file found at pinned commit | Not included; permission to redistribute not established |
| FlashForge | [Orca-Flashforge](https://github.com/FlashForge/Orca-Flashforge) | `LICENSE.txt`: AGPL-3.0 | Not included; asset-specific rights not assessed |
| OrcaSlicer | [OrcaSlicer](https://github.com/OrcaSlicer/OrcaSlicer) | `LICENSE.txt`: AGPL-3.0 | Not included; asset-specific rights not assessed |
| Prusa Research | [PrusaSlicer-settings](https://github.com/prusa3d/PrusaSlicer-settings) | `License`: AGPL-3.0 | Not included; asset-specific rights not assessed |
| Community Prusa bundles | [PrusaSlicer-settings-non-prusa-fff](https://github.com/prusa3d/PrusaSlicer-settings-non-prusa-fff) | No repository-wide licence found; individual vendor bundles may carry separate terms | Not included; permission to redistribute not established |

The six AGPL-3.0 findings refer **only to the repository's root licence document**. Some vendor profile assets may be third-party contributions with different terms. A root licence is not a verified per-profile licence. The two sources with absent root-licence evidence must not be presumed public domain, open source, or freely redistributable merely because GitHub makes their files visible.

**If SliceBridge later redistributes verbatim profile payloads:** review the licence of each asset and the distribution architecture, preserve original copyright notices, include the applicable licence text and satisfy corresponding source obligations where required. Credit by itself is not sufficient permission and non-commercial/non-modifying distribution is still distribution. The legal effect of combining code and data depends on the actual architecture.

For the existing metadata-only integration, retain this attribution manifest and source-permalink information. Do not mirror the vendor profile contents, bundle external slicer software, inject unreviewed remote downloads, or upgrade a printer's conversion-ready status based on attribution.

This is an engineering evidence ledger, not a legal opinion or a warranty of title. Before distributing copied third-party assets or enabling a remote update channel, obtain an appropriate rights review.
