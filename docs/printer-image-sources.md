# Printer image sources and licence record

Internal record. Last updated: 2026-10-21, base commit `eb38540`.

SliceBridge bundles **no printer photographs, manifests or manufacturer
artwork**. This file records that policy and how printer pictures are sourced.
It is not user-facing.

## Current policy

- Printer pictures come only from slicers already installed on the user's
  computer. SliceBridge reads the cover PNG that the installed slicer ships for
  the exact physical model, and passes it to the UI as a bounded
  `data:image/png` value.
- SliceBridge never downloads pictures, never fetches a remote image URL and
  never reads an image path supplied by the frontend.
- A card with no installed cover shows the app's neutral "Image unavailable"
  caption. It never borrows another model's picture.

## How a cover is chosen

The UI first checks exact catalogue covers, then one installed target per slicer family when necessary (never every nozzle). Two backend methods in `native/printer_artwork.go` supply pictures:

1. `PrinterArtworkForInstalledTarget(targetID)` is used for installed
   Orca-family targets. The ID is re-resolved against freshly discovered
   machine profiles. The cover is `<Vendor>/<printer_model>_cover.png`, read
   first from the target's own installation and then, for that exact file
   only, from other allowlisted installations.
2. `PrinterArtwork(catalogID)` is kept for the four legacy converter models and
   offline catalogue entries. It uses the catalogue's exact
   `localCoverCandidates` hints.

Matching is exact. There is no substring or fuzzy matching, so "Creator 5" never
borrows the "Creator 5 Pro" cover, and the reverse is also true. Bambu
installations are not read.

Every returned picture must pass these checks:

- the path is a single vendor folder plus one `_cover.png` file name, with no
  `..`, separators, drive letters or control characters;
- the file is a regular file, not a symlink, and its resolved location stays
  inside its allowlisted installed root;
- size is at most 512 KiB;
- the PNG signature is present and the decoded size is 32–2048 px per side,
  with at most 4 million pixels;
- SVG, JPEG and other formats are refused even when they are renamed.

The reply carries only `found`, `dataUri` and `provenance`. No installation path
or profile bytes reach the frontend.

## Naming evidence

The `<Vendor>/<printer_model>_cover.png` convention matches the observed
installed-OrcaSlicer files listed in `catalog/PICKER-UI.md`:

- `Snapmaker/Snapmaker U1_cover.png`
- `Flashforge/Flashforge Creator 5_cover.png`
- `Elegoo/Elegoo Centauri Carbon 2_cover.png`
- `X 5 Series/X-Plus 5_cover.png`

Read-only checks of installed slicer resource directories on 2026-10-21
confirmed these names on both a Windows workstation (Chris) and a Mac
(MacMiniOffice). Additional verified exact-model files include
`Flashforge/Flashforge Creator 5 Pro_cover.png` and
`Elegoo/Elegoo Centauri Carbon_cover.png`. QIDIStudio also ships
`X 5 Series/X-Plus 5_cover.png`. No photos or profiles were copied from
those installations. Other OS/version combinations may differ; missing
files fall back to the neutral caption, never to an approximate match.

## Removed in this change

| Former artifact | Reason |
|---|---|
| `native/frontend/dist/printer-images/snapmaker-u1.jpg` | Unsplash photo of a workshop scene. Its identity rested on visual inspection, not a spec sheet, and it was a third-party photograph. |
| `native/frontend/dist/printer-image-manifest.json` | It pointed the UI at the bundled photo. The UI no longer fetches any manifest. |

The JPEG and manifest remain in git history at `eb38540` and earlier. Removing
them from history, if that is wanted, is a separate decision.

## Rules that still apply

- Only images whose redistribution right can be verified may be bundled. This
  change bundles none, so no new rights review is needed.
- Manufacturer and retailer photos are copyrighted unless the manufacturer
  publishes an explicit licence. Adding one would need its own record here.
- Pictures must show the exact physical model, not a sibling variant.
- No product images are generated or invented.
