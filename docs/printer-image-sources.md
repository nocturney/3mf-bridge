# Printer image sources and licence record

Internal record. Audit date: 2026-10-21. Base commit: `5cf7d5d`.

This file records provenance and licensing for printer images considered for
`native/frontend/dist/printer-images/`, and the manifest at
`native/frontend/dist/printer-image-manifest.json`. It is not user-facing.

## Rules applied

- Only images whose redistribution right can be verified are bundled.
- Images must depict the exact physical model, not a sibling variant.
- No product photos are generated or invented.
- When licence or provenance cannot be verified, the image is left unavailable.
- Manufacturer and retailer photos are treated as copyrighted unless the
  manufacturer publishes an explicit licence for them.
- The existing product rule in `catalog/PICKER-UI.md` still holds: no
  external images are redistributed by default. This record is the only
  exception path, and it applies per image.

## Bundled images

| Manifest key | File | Source | Licence | Dimensions / size | Identity check |
|---|---|---|---|---|---|
| `snapmaker|u1` | `printer-images/snapmaker-u1.jpg` | Unsplash, "Photo by Snapmaker 3D Printer" (Snapmaker's account, as listed on Unsplash) | Unsplash License (free to use, no attribution required; verify current terms at unsplash.com/license before any commercial sale of an unmodified copy) | 800×533 JPEG, q78, 103,562 bytes | Visual check: white enclosed printer with a "snapmaker" logo and a four-tube toolchanger feed, consistent with U1. Not confirmed against a spec sheet. |

Source page: https://unsplash.com/photos/3d-printer-with-filament-spools-and-printed-objects-IBffrp08OIo
Original download: 1000×667 JPEG, SHA-256 `45195e45aaf0f6bc3ebf04dfee3b963316419683750c34e0e7a858eed977b12d`
Bundled file SHA-256: `1bc4bd241a870499ac06a46e54ca4728afd2b17bc8fcb02bb699b34ed94d2f9a`

Known limits of this image:

- It is a lifestyle shot in a workshop, not a clean studio product photo.
- Incidental third-party content is visible: a "Carlos 3D" neon sign and
  workshop items. No people are visible.
- Model identity rests on visual inspection and the uploader's description.
  Confirm it against Snapmaker's official U1 product page before release.
- WebP was not used: the available macOS `sips` tool cannot write WebP.
  The file is a JPEG, and the manifest path matches it.

## Audited and not bundled

| Manifest key | Model | Result | Reason |
|---|---|---|---|
| `elegoo|centauri carbon 2` | ELEGOO Centauri Carbon 2 | Unavailable | Search found only retailer and press-site product images with no licence. The All3DP CC BY 4.0 licence covers article text, not images. Needs permission from Elegoo. |
| `flashforge|creator 5` | Flashforge Creator 5 | Not audited | Not searched individually; no candidate found. |
| `flashforge|creator 5 pro` | Flashforge Creator 5 Pro | Unavailable | No Wikimedia Commons file found. Vendor, retailer and review-site photos are copyrighted; a review site carries a photographer copyright notice. |
| `qidi|x-plus 5` | QIDI X-Plus 5 | Unavailable | No open licence found. QIDI's product photos carry no licence. Needs permission from QIDI. |
| `snapmaker|u1` | Snapmaker U1 | Bundled (above) | Unsplash License, visual identity check. |

Popular integrated models outside the five requested were not audited
individually. None is bundled.

## Repository assets reviewed

- Repository brand and identity images (`native/frontend/dist/brand-icon.*`,
  `slicebridge-banner.png`, `extension-packages/*`) are SliceBridge's own
  project identity. They depict no printer.
- Vendor profile cover PNGs are read only from locally installed slicer
  resource roots at runtime, by `native/printer_artwork.go`. They are not
  bundled and not redistributed.
- No printer photograph exists in the repository.

## Runtime notes

- `native/printer_artwork.go` serves installed-slicer covers only. It does not
  read this manifest. Wiring the manifest into the UI is outside this record.
- Keys follow `manufacturer|physical model`, lowercased, trimmed, with
  internal spaces kept. For example, `snapmaker|u1`.
