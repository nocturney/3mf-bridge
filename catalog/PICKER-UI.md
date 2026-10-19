# Offline My Printers / Catalogue (Draft)

## 0.4.0-beta.4.dev.1 — 19 Oct 2026 (development, not released)

- First-run setup starts with **zero preselected printers**. Existing saved `my-printers.json` remains authoritative and is never reset.
- No model cards or local pictures appear until the user **chooses a manufacturer or searches**. Brand selection filters the existing offline catalogue; image bytes still load lazily from an installed slicer only when results are shown.
- There is exactly **one My Printers entry** and **one Settings entry** in the top bar. Removed the duplicated target-panel picker link and per-printer gear shortcuts; per-printer preferences still live in the existing central Settings dialog.
- At narrower window widths, top-bar labels collapse cleanly to accessible icons and tooltips instead of being clipped. When zero target printers are selected, the workspace explains where to add them.
- Source of truth for release/version policy is tracked in separate [PR #23](https://github.com/nocturney/slicebridge/pull/23). This build milestone is part of [Draft PR #13](https://github.com/nocturney/slicebridge/pull/13) and is **not** a published GitHub Release or approved conversion update.

The new native Wails picker consumes **603 printer identities**, but it does not
install, download or redistribute vendor profiles, printer photos, logos or
third-party slicers.

## User experience

- First launch with no `my-printers.json`: show an explicit setup dialog with
  **no printers selected**. Closing without saving leaves onboarding pending;
  existing saved printer selections and conversion preferences remain untouched.
- After saving: display only selected legacy target adapters on the conversion
  page; additional user-selected printers appear explicitly as **catalogue
  only — conversion not available**.
- On first open, show no cards or images. Choose a manufacturer or type a
  manufacturer/model/alias search to reveal matching models; cap results at
  80 cards. Manufacturer choices are populated from the indexed catalogue,
  with no handpicked defaults or recommended printer models.
- Bambu Lab models remain in the data snapshot but are **hidden from target
  selection and explicitly rejected by the save API**. MakerWorld/Bambu remain
  project source workflows; no Bambu target adapter is claimed.
- Saved printer membership is a separate, versioned, atomic
  `my-printers.json`, next to existing `settings.json`. The old settings file
  and its per-target nozzle, slicer and filament choices are not migrated,
  rewritten or deleted. Unknown/corrupt picker settings fail closed.
- All four legacy adapter keys and existing Creator 5 0.2→0.25 physical nozzle
  logic are preserved in their original Wails code.

## Locally sourced images; no packaged manufacturer artwork

Many slicers put their own model covers in a profile-relative path such as
`Snapmaker/Snapmaker U1_cover.png` or `X 5 Series/X-Plus 5_cover.png`.
The build derives only **relative filename hints** from the pinned catalogue's
machine model paths and manufacturer vendor bundles. The PNG bytes are not
stored or shipped by SliceBridge.

At runtime, the frontend asks Wails for artwork using **only a known catalogue
model ID**. The backend checks a small list of established *installed slicer*
profile roots; denies arbitrary paths, parent traversal, escapes via symlinks,
and non-regular files; checks file size (512 KiB) and PNG magic/decoded image
dimensions; and returns bounded `data:image/png` bytes. The UI loads pictures
lazily on scroll with at most four concurrent local requests and caches
results in memory for the active window. **It never downloads artwork**.

Verified observed files in installed OrcaSlicer on Chris:
- `Snapmaker/Snapmaker U1_cover.png`
- `Flashforge/Flashforge Creator 5_cover.png`
- `Elegoo/Elegoo Centauri Carbon 2_cover.png`
- `X 5 Series/X-Plus 5_cover.png`

Photos and manufacturer logos are not guaranteed across slicers or versions.
Logo handling remains limited to display-name/initials until a verified
locally installed logo is identified for a vendor. Missing/oversized/bad
pictures show the app-owned neutral printer tile. No copied third-party
media, remote image URLs or any claim of rights to redistribute are included.

## Tests and release barriers

- Committed deterministic compact catalogue export and regeneration assertion.
- Catalogue JSON/source/rights integrity, unearned converter flag checks.
- Go tests for settings isolation, fail-closed selection, valid model IDs,
  image type/size/path/symlink barriers and base64 picture decoding.
- JS tests for Bambu exclusion, indexed-vs-supported distinction, search,
  no external network image fetching and new picker controls.
- GitHub Actions builds Windows WebView2 and macOS WebKit binaries.

**PR is Draft and must not self-merge.** Standalone app QA, actual installed
slicer discovery, photo appearance, onboarding dialogs, keyboard navigation
and real 3MF import/conversion remain release gates. Image discovery is
presentation only and does not confer conversion capability.
