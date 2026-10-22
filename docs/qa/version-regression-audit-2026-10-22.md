# SliceBridge — Historical regression audit (2026-10-22)

**Scope:** All **18 published version tags** visible in GitHub Releases:
`v0.4.0-beta.3`, `v0.4.0-beta.4.dev.1` through `dev.17`.
This is a source/release/test comparison, **not** a claim that 18 historical desktop
binaries were individually opened, sliced or physically printed. Tag commit IDs and
release asset lists were checked against GitHub. PR #59 is a **new, unreleased
integration branch** at the time of the audit. The latest published application
is dev.17, whose binaries do **not** contain the later fixes.

## Per-release behaviour ledger

| Tagged release | Function or change established | Later fate / audit result |
|---|---|---|
| beta.3 | Mint/Forest desktop identity; manual/automatic Clear Results; intentionally removed standalone JSON report export | Branding and Clear Results are historic requirements; the JSON export was **removed intentionally**, not lost |
| dev.1 | 603 indexed models; separate My Printers inventory and pre-existing four real conversion adapters; no settings reset | **Regressed in dev.13** when inventory was limited to available conversion targets; corrected in backend PR #56 and frontend PR #59 |
| dev.2 | Responsive target-printer cards and scrollable source workspace | Current source/target geometry checks remain; My Printers later developed a separate clipping problem |
| dev.3 | Genuine exact-model thumbnails from **locally installed slicers**, no network images | **Regressed in dev.17** by bundling an external U1 photo; corrected in PR #54 |
| dev.4 | Guided MakerWorld browser-extension beta onboarding | Reclassified as advanced/Developer Mode by dev.12; deliberate restriction, not a lost conversion engine |
| dev.5 | Source queue expanded from clipped one-line strip; independent scrolling and selected count | dev.6 changed oversized always-on height intentionally; source queue layout QA remains in CI |
| dev.6 | Compact source queue sizing avoids unnecessary workspace scrolling | Current queue layout QA preserved; needs real Wails UX confirmation, not just DOM |
| dev.7 | About dialog, version and manual Check for updates moved from footer | Feature moved rather than removed; about/update tests still run |
| dev.8 | Consolidated Windows/macOS native packaging, bilingual site and licensing | Repository restructuring PR #53 updated paths and CI; metadata and license checks continue |
| dev.9 | Better per-user installed-slicer discovery; allowed *redundant* same-color/material source channel coalescing | **Explicitly reversed in dev.10** at owner request; do not restore channel merging |
| dev.10 | Every source filament ID remains a distinct virtual 3MF slot, even when printer has four physical tools; preserve paint/object mappings | Regression guards remain. 11 Panda2Prusa synthetic outputs and existing four converted user outputs received structural checks; target slicer round-trip/printing unverified |
| dev.11 | Installed Orca-family machine/process/filament discovery, generic PrusaSlicer 3MF project via Panda2Prusa, detection of 22 slicer apps | Both conversion pathways remain. Important limit: generic Prusa export **never** promised automatic target-printer profile mapping |
| dev.12 | MakerWorld beta extension requires opt-in Developer Mode and existing browser authorization | Security requirement remains; ordinary local 3MF input unaffected |
| dev.13 | New "show available conversions only" filter in My Printers | **Confirmed product regression**: 603 catalogue records became a 249-model UI depending on local discoveries. PRs #56/#59 restore separate inventory and conversion readiness |
| dev.14 | Single portable Windows EXE including the Prusa engine; macOS ZIP app | Release packaging checks remain and validate embedded engine. Later public release must rebuild exact SHA |
| dev.15 | Preferred maker dropdown, saved choices first and publicly verified launch-date ordering | PR #59 preserves saved order, maker A-Z, within-maker newest verified dates; no dates or support categories printed on cards |
| dev.16 | Installed Orca nozzle/process fallback restored when vendor default process is unavailable | Go profile-scanning path still present and Windows/macOS tests run; never infer a physical print from structural profile discovery |
| dev.17 | One physical card per printer with independently selectable nozzle/slicer variants; U1 photo fallback and smaller card layout | **Visible RTL card/variant clipping** and external photo contradicted dev.3 design. PRs #54/#59 remove external image and switch to intrinsic grid sizing with Chromium geometry checks |

### Cross-version findings, ranked

| Priority | Finding and evidence | Corrective state |
|---|---|---|
| **P0** | **Published-binary drift:** installed dev.17 and its immutable release tag predate all October 22 fixes. GitHub main/PR improvements cannot appear in that binary | **Open for distribution:** do not overwrite dev.17; publish a distinct version only after Windows/macOS CI, exact-source packaging and visual review |
| **P0** | **Inventory vs conversion mixed:** dev.13 introduced available-only selection; older catalogue entries could not be saved | Backend PR #56 merged; frontend PR #59 pending checks, shows 589 non-Bambu IDs in 584 physical cards without making unsupported conversions ready |
| **P1** | **My Printers overflow:** historical 91px / constrained-height cards hid nozzle options, as observed in owner's screenshot | PR #59 uses content-sized cards and independent scrolling; 8 RTL/LTR Chromium geometry cases passed on Windows; real Wails user confirmation remains |
| **P1** | **Wrong slicer precedence:** generic Orca could beat QIDIStudio on Windows; UI gave builtin/0.4 priority before validating manufacturer source | PR #59 includes explicit manufacturer/slicer allowlist, QIDIStudio-first across install bases, fully validated vendor profile before generic fallback, preserving explicit overrides |
| **P1** | **Snapmaker U1 special case:** built-in 0.4 template could beat an installed, complete Snapmaker Orca profile | PR #59 prefers exact manufacturer profile; keeps old template only if vendor profile incomplete/missing and old auto settings apply |
| **P1** | **External image substitution:** dev.17 bundled a non-local U1 photo though the original design read installed covers only | PR #54 merged; exact-model local PNG source and transparent missing-image state, no image-network downloads |
| **P1** | **Visual QA blind spot:** standard DOM/unit CI passed but failed to detect clipping. An early new QA helper also triggered two CodeQL alerts due broad HTML regex | PR #59 includes real headless geometry QA in native CI and replaces regex stripping with exact known script-tag removal; CodeQL recheck required on final SHA |
| **P2** | **Virtual palette regression risk:** dev.9 coalescing deliberately undone by dev.10; future "optimization" could lose fifth color/paint state | Protect discrete source IDs and per-object paints; multi-color fixture tests remain; still need target GUI and print validation |
| **P2** | **Prusa profile mapping gap:** generic output requires choosing printer/filament profiles in PrusaSlicer | Never actually implemented as automatic per-model mapping; **not a historical regression**. Do not mark all 603 models ready |
| **P2** | **Manual MakerWorld setup:** beta extension is Developer Mode only | Intended dev.12 policy, not feature disappearance; explicit opt-in retained |

## Acceptance checklist before next published release

- [ ] All native Windows WebView2 and macOS WebKit CI jobs, security and metadata green at **exact release source SHA**
- [ ] Real Windows app at normal and compact sizes: 5 saved printers shown, variants inside their own cards, RTL/LTR and filters readable
- [ ] Exactly 589 non-Bambu catalogue IDs remain selectable; 14 Bambu models excluded as source-only; catalogue-only selection never enables conversion
- [ ] Manufacturer's installed matching machine + print process + PLA chain first; universal Orca fallback only after validation; generic Prusa project is a separate explicit export
- [ ] Existing saved printer IDs, explicit nozzle/slicer/filament preferences, local settings file and selection order survive upgrade without migration
- [ ] Exact model covers only from installed slicer PNGs; no download, no cross-model image, no bundled third-party photo
- [ ] 3MF keeps all source filament slots, colors, per-object/paint references; no stale sliced G-code; a fifth slot must not silently merge
- [ ] Source file queue, folder drag/drop, Clear Results, About/version/update, MakerWorld Developer Mode and bilingual site still work
- [ ] Windows single EXE contains matching Prusa engine; macOS ZIP/Info.plist/signature and source SHA/checksums verified
- [ ] Run real destination slicer GUI round-trip for representative multi-material projects; physical print tests remain separate and are **not** claimed complete

**Audit method/limitations:** Compared tagged changelogs/release descriptions, actual source
trees, selected commits and tests for all 18 releases; the current branch was tested
with 141 Node UI tests and 8 Chromium geometry scenarios. No manual full GUI run of
each historic Windows/macOS release was performed. Release notes are historical
claims; a tag or passing test cannot prove every hardware or UI workflow in practice.
No new slicer format engines or physical printer profiles were installed during the audit.
