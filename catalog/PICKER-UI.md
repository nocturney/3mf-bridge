# Offline My Printers / Catalogue (Draft)

The new native Wails picker consumes **603 printer identities**, but it does not
install, download or redistribute vendor profiles, printer photos, logos or
third-party slicers.

## User experience

- First launch with no `my-printers.json`: show an explicit setup dialog with
  the four legacy targets preselected. Closing without saving leaves onboarding
  pending; the existing conversion preferences remain untouched.
- After saving: display only selected legacy target adapters on the conversion
  page; additional user-selected printers appear explicitly as **catalogue
  only — conversion not available**.
- Search all models by manufacturer, model or alias. Limit the visible search
  results to 80 at a time; guide the user to narrow searches rather than
  rendering hundreds of cards at once.
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
