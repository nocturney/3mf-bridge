# Changelog

Published development builds are listed under [GitHub Releases](https://github.com/nocturney/slicebridge/releases). They are experimental pre-releases, not stable versions. Unmerged branch work is not considered released.

## Unreleased

- **Model-specific output filenames:** converted files now begin with the physical target model (for example `U1-`, `Creator5Pro-`, `H2D-`, `CR10-`, `Neptune4Pro-`) rather than the generic Orca slicer name. The original filename and no-overwrite collision numbering are retained. The generic Prusa project export keeps `PRUSA-` because no printer model is selected. Legacy Electron adapter outputs and folder-import generated-output filters follow the same new model naming; previously generated file prefixes remain recognized.
- **Local printer artwork fallback across slicers:** prefer the exact matching PNG in the target's own installed slicer, then the same vendor/model cover in other trusted installed Orca-family roots, Bambu Studio and PrusaSlicer resources. The additional apps supply **images only**, never foreign conversion settings, and only exact-model PNGs from bounded allowed paths are read. No network images, approximate model matching, user-preference resets or redistributed manufacturer art.

## [0.4.0-beta.4.dev.22] - 2026-10-24 (published experimental Pre-release; PR #79 merged)

- **More real installed slicer profiles, not more fictional printer support.** [PR #79](https://github.com/nocturney/slicebridge/pull/79) discovers valid manufacturer-owned Creality Print / Anycubic Slicer Next Orca-family machine, process and filament bundles, and fixes narrow profile-resolution failures without copying settings from unrelated models. Exact physical nozzle diameters are retained, including 0.25 mm hardware with 0.2 mm *source-category* equivalence only.
- **Native G-code QA expanded to 483/483 selectable installed printer/nozzle targets on at least one appropriate native CLI + reproducible QA project.** This is *heterogeneous coverage*, not 483/483 on a single slicer or a real-product guarantee: 474/483 passed the original uniform single-filament case, four Anycubic Kobra 3 combinations used an alternate source and five Creality configurations used the official Windows Creality Print 7.3 CLI. Genuine slicer output was checked for identity, physical nozzle and positive motion/extrusion, with exact G-code hashes in the independent local validation report. Do not interpret this as certification of physical printing or multi-material behavior.
- **Protect manufacturer mechanics and user presets.** Retain Creality's official active 28/30 mm filament-cut retraction values rather than falsifying them to satisfy a generic Orca 18 mm limit. Remove an *inactive, zero-value* Anycubic cut override only for a strictly identified single-material/single-tool path, failing closed if any active or ambiguous vendor value or script exists. Source 3MF paint, geometry, ordered material slots, object assignments and target nozzle remain guarded; no saved slicer profile or user's printer selections are edited by conversion.
- **More honest independent QA:** fixes ZIP relationship namespace serialization, per-source compatibility fields and regression handling for complex Bambu paint subdivision trees, opaque RGBA/RGB color equivalence, inert script comments, and identical same-model OEM scripts. Unknown paint-to-physical-tool mapping is **SKIP**, not falsely PASS. All 55 Python compatibility regressions passed; full engine/native Go suites and targeted UI tests passed on PR #79.
- **Visible multi-material acceptance warning:** converted output requiring manual color/filament-slot mapping no longer appears as an unconditional green success in the results list. It receives a conspicuous **Review mapping** status and a visible caution above the collapsed technical details, with RTL/English copy. The file is still considered successfully *converted*; the application does not silently drop the fourth color, pretend all physical tools are mapped, or auto-send a print. The actual four-source-slot/one-physical-tool result view is covered by a live DOM regression; normal mappings retain their original success status.
- **Known limitations: this remains an experimental development build, not a print-safety approval.** Tests with public real four-color projects proved exact 3MF geometry and raw painted-triangle preservation but exposed native CLI version incompatibility on newer BambuStudio projects, target-specific wipe/prime-tower collision, and a *missing T3 tool* in a Kobra 3 QA output compared with the source's four-tool G-code. The five official Creality Windows CLI 10 mm cubes had different placement from macOS GUI on the same 3MF inputs. Consequently, multicolor/tool mapping, printer-bound geometric placement and safe physical printing remain **unverified**, even where G-code is emitted. Reopen in the installed target slicer, inspect exact material assignments, collision warnings and geometry before any manual print. No unattended print/send path is added.
- **Release status:** [dev.22](https://github.com/nocturney/slicebridge/releases/tag/v0.4.0-beta.4.dev.22) was published after owner approval, merging PR #79 to `5ed729a`, exact-main native Windows/macOS binary builds and SHA-256 verification. It remains experimental, with explicitly documented multi-material, print-placement and signing limitations.


## [0.4.0-beta.4.dev.21] - 2026-10-23 (experimental development build; see GitHub Releases for publication status)

- **Manufacturer slicer first for verified targets** ([PR #76](https://github.com/nocturney/slicebridge/pull/76)): on a new selected physical printer, if a compatible manufacturer slicer has a fully resolved installed machine/process/filament profile at that **same nozzle**, prefer it to generic Orca; otherwise retain a validated fallback. The four existing built-in target Settings options already use manufacturer-first `Auto` with safe fallback; their guidance now reflects this. A manually saved target/profile ID is never changed automatically.
- **QIDI naming corrected without altering profiles:** installed QIDIStudio's real `X 5 Series` profiles appear under `QIDI`, and canonical `QIDI X-Smart 5` model prefixes normalize to one physical printer card alongside equivalent Orca targets. The actual opaque profile IDs, installations and settings files remain untouched.
- **No slicer/nozzle controls in the manufacturer catalogue:** hardware cards stay one checkbox per physical printer. A small slicer preference appears only for an **already selected** printer with two or more completely verified installed slicers for its **current nozzle**; it never allows changing nozzle diameters. The explicit default button names the actual preferred installed slicer rather than pretending to persist a dynamic `Auto` identifier. Any change still needs Save Printers; Cancel and existing manual overrides remain intact. Compact RTL/LTR selected-printer summary controls no longer truncate printer names.
- **Scope / safeguards:** this is an experimental development pre-release, not a certified physical-print workflow or a fix for unresolved Prusa multi-filament/tool binding. An installed slicer executable alone is not proof of a compatible profile. The source's nozzle cannot alter physical target hardware; review printer, nozzle, filament, color and slicer warnings before printing.

## [0.4.0-beta.4.dev.20] - 2026-10-23 (experimental development build; release status: see GitHub Releases)

- **Return to the simple pre-Pro printer selection experience** ([PR #74](https://github.com/nocturney/slicebridge/pull/74)): My Printers has one compact card/checkbox per physical model and a clean model-only list of selected printers, without in-card technical nozzle/slicer variants, conversion rankings or misleading unavailable-image captions. A prominent Show more manufacturers control preserves access to the entire underlying 603-record catalogue (589 non-Bambu entries), while the curated manufacturer shortlist stays the default.
- **Profile options moved into separate Settings** inside My Printers. Explicitly switching one model's installed nozzle/slicer target changes only that model's draft selection; Save Printers persists the exact target ID. Rechecking a temporarily deselected model restores its original saved profile (including Flashforge Creator 5 Pro 0.25 mm) rather than silently switching to 0.4 mm. Existing user settings, printer identities and slicer/filament presets are never rewritten by updating.
- **Select all now really selects all conversion destinations**: the four built-in targets plus actual backend-resolved installed Orca and ready PrusaSlicer targets. Newly seen installed profiles start checked if selected by the user; later re-discovery preserves manual unchecks. Main destination cards show printer name only; detailed nozzle, slicer and physical tools stay out of the primary interface. Backend conversion availability, profile ownership checks and safety restrictions are unchanged.
- **Release QA:** added functional five-printer regression (including exact 0.25 mm Creator 5 Pro conversion target), alias retention, profile-dialog cancellation/reselection, and real Chrome RTL/LTR layout and modal checks. This is still an experimental pre-release: no real physical printing approval and no claim of complete multi-colour Prusa preset/tool mapping.

## [0.4.0-beta.4.dev.19] - 2026-10-22 (experimental development build; release status: see GitHub Releases)

- **My Printers is a hardware inventory, not a conversion leaderboard.** [PR #71](https://github.com/nocturney/slicebridge/pull/71) removes preferred-maker groups and source-tier ranking, groups duplicate Anker/AnkerMake spellings into one model card, keeps each saved catalogue ID selectable, and blocks foreign manufacturer profiles borrowed from branded Orca forks. No conversion capability is inferred from a catalogue row.
- **Short, configurable manufacturer view.** [PR #72](https://github.com/nocturney/slicebridge/pull/72) shows 13 selected catalogue manufacturer names (11 destination menu labels after combining Prusa and excluding source-only Bambu). `Show more manufacturers` temporarily reveals all 589 non-Bambu catalogue IDs; the full 603-record catalogue, hidden installed conversion targets and saved choices remain untouched. The five stored printer selections are preserved.
- **Safer and more useful installed Orca flows.** [PR #64](https://github.com/nocturney/slicebridge/pull/64) preserves preferences when installed slicers temporarily disappear; [PR #66](https://github.com/nocturney/slicebridge/pull/66) recognizes the real QIDIStudio executable variants; [PR #68](https://github.com/nocturney/slicebridge/pull/68) permits opening `orca-...` outputs only through the trusted installed owning app, while refusing unrelated paths, untrusted links and unknown profile IDs. This does not certify every slicer's import or slicing.
- **3MF conversion regression quality.** [PR #65](https://github.com/nocturney/slicebridge/pull/65) expands a synthetic Orca profile matrix; [PR #67](https://github.com/nocturney/slicebridge/pull/67) resolves recursive `p:path` parts, build/component transforms, placed triangle paint, and geometry-aligned object extruder checks. One strict precision discrepancy in the local 14-project sample remains distinguishable from passing cases; do not claim universal conversion equivalence.
- **Scope:** experimental development pre-release, not a stable build or validation of physical printing. PrusaSlicer generic projects may import and slice while still lacking verified exact printer/print/filament preset binding or multi-colour tool mapping. Bambu/MakerWorld remain sources only. Check material, tool, nozzle, paint assignments and slicer warnings before attempting any print. Installer/binary signing and macOS notarization are not claimed.

## [0.4.0-beta.4.dev.18] - 2026-10-22 (development build; release status: see GitHub Releases)

- Restore the full non-Bambu printer inventory: 589 catalogue IDs shown as 584 distinct physical-model cards, with each matching nozzle/slicer target preserved. Keeping a catalogue model in My Printers does not assert that it can be converted; conversion remains guarded by actual installed and validated profile chains.
- Default to each manufacturer's matching, fully resolved local slicer profile before a generic Orca/Prusa alternative; preserve explicit nozzle, slicer, filament preferences and the five existing saved printers. The Snapmaker U1 built-in 0.4 mm template remains an offline fallback when its installed vendor profile is not usable.
- Repair the RTL/LTR My Printers layout and intrinsic-height nozzle variant cards, and require real Windows/macOS Chromium geometry regression checks. Restore exact-model photos from locally installed slicers only; remove the external U1 photo and image manifest.
- Expand MakerWorld/Bambu-origin 3MF compatibility QA to nested model parts, source filament metadata, multi-color/virtual-slot checks and explicit format-inapplicable comparisons; preserve dev.10's independent source color/material channels and reject stale sliced G-code. Generic PrusaSlicer projects still require choosing compatible printer/material profiles in the slicer.
- Keep Bambu as a source, not a conversion target; no Cura, ideaMaker, UP Studio or other niche-format engines were added. Move historical auxiliary files out of the repository root, maintain the short README, and record a per-release regression audit from beta.3 to dev.17. Source tests and CI do not constitute physical print validation.

## [0.4.0-beta.4.dev.17] - 2026-10-21 (development build; see GitHub Releases)

- My Printers now shows one card per physical printer model, with each installed nozzle/slicer profile available as a distinct selectable variant. Existing printer choices, engine target identifiers and source-to-slicer conversion routes are retained.
- Deduplicate manufacturer spelling aliases in the picker, keep integrated makers first, and offer the generic PrusaSlicer export without presenting it as a printer manufacturer.
- Resolve printer pictures by exact physical model using existing local slicer cover assets and, where redistribution rights are known, a small bundled photo. Display an explicit unavailable state instead of misleading artwork.
- Keep official launch-announcement dates only as internal model-ordering metadata; remove dates from printer cards. Source-backed dates are recorded for a limited set of popular models, with unknown dates left unknown.
- Fix a frontend startup race and expand Windows/macOS UI and catalogue regression coverage. No printer profiles, existing conversion adapters, or saved configurations are reset.


## [0.4.0-beta.4.dev.16] - 2026-10-21 (development build; see GitHub Releases)

- Restore Orca-family printer nozzle profiles hidden because the slicer's declared default process was missing, misspelled, or intended for another printer. Select a locally installed process only if it explicitly lists the exact machine, flattens successfully, and has an appropriate layer height.
- Detect compatible process alternatives once per slicer/vendor to avoid repeatedly rescanning them for every machine. Never substitute a process from another printer model.
- Show the machine/nozzle designation in the installed profile's display name so Creator 5 Pro 0.25, 0.4, 0.6, and 0.8 mm profiles can be distinguished when available.
- Add test fixtures modeling the installed Flash Studio Creator 5 Pro profile anomaly, safety rejections, and unavailable defaults. Keep manufacturer slicer files, existing printer selections, and other conversion routes unchanged.

## [0.4.0-beta.4.dev.15] - 2026-10-21 (development build; see GitHub Releases)

- In My Printers, show the principal slicer-integration manufacturers first, a disabled visual separator, and other detected manufacturers alphabetically. Existing selections stay first in saved order.
- Sort other printer choices by verified public launch date newest-first when known; undated choices follow, with familiar integrated manufacturers before alphabetic fallback. No release dates are invented from model names.
- Show the launch month/year only when sourced. All target enablement, physical nozzle handling and conversion adapters remain unchanged in this UI-only build.

## [0.4.0-beta.4.dev.14] - 2026-10-21 (development build; see GitHub Releases)

- Windows portable download is one `SliceBridge.exe` with the Prusa conversion engine embedded. No external helper executable, Python installation or installer is required; the verified engine extracts automatically to a per-user cache.
- macOS still packages the engine inside the single `SliceBridge.app`; conversion functionality and source material/color preservation are unchanged.

## [0.4.0-beta.4.dev.13] - 2026-10-21 (development build; see GitHub Releases)

- Show only available conversion destinations in My Printers and the main target list.
- Include resolved local Orca targets and PrusaSlicer projects; hide unsupported catalogue entries and unavailable options.
- Keep older saved selections readable when a profile is no longer installed.

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

See [the simple community release guide](docs/release/DISTRIBUTION_CHECKLIST.md) when sharing a future build.
