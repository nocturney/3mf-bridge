# prusa3mf

Pure-Go conversion of Bambu/Orca 3MF projects into PrusaSlicer 3MF packages.
Standard library only; no Python or G-code at runtime.

```go
report, err := prusa3mf.Convert("project.3mf", "project.prusa.3mf")
if err != nil {
    code := prusa3mf.ErrorCode(err) // stable code, e.g. UNSUPPORTED_PAINT
}
```

CLI: `go run ./cmd/prusa3mf --source in.3mf --output out.3mf` (run from `engine/`).
It prints one JSON object with the report or the error code.

## What is converted

| Source | Output |
| --- | --- |
| Vertices, triangles (exact float round trip) | Same, in object space |
| Build item `transform`, `printable` | Same |
| Per-triangle `paint_color` (TriangleSelector tree, channels 1-18), copied unchanged | `slic3rpe:mmu_segmentation` with the same code |
| Object `extruder` and single-part `extruder` (`model_settings.config`) | `Metadata/Slic3r_PE_model.config` |
| Numbered filament table (`slice_info.config`, else project `filament_colour`/`filament_type`) | `slicebridge_source_filament_colour` / `slicebridge_source_filament_type` at position N = channel N (metadata only) |

Channels are never renumbered. Duplicate colour/material entries are kept and
counted (`duplicate_material_slots`). Gaps in numbering become placeholder
channels, which are listed in `slicebridge_placeholder_channels`, have empty
colour and material entries, and may not be used by any object or paint code.
The filament table is not written as PrusaSlicer `filament_colour` or
`filament_type`, so no PrusaSlicer filament preset is claimed to be set.

Print, printer and process settings are **not** transferred. Firmware
start/end scripts and slice caches (`Metadata/plate_*.gcode`, `.md5`, `.json`)
are dropped and listed in the report; their content is never read or copied.

## Refused inputs

Every refusal returns a stable code and writes no output file.

| Code | Meaning |
| --- | --- |
| `UNSUPPORTED_PAINT` | Paint code is malformed, truncated, not hexadecimal, has bits after its root node, or is longer than 4096 characters |
| `CHANNEL_NOT_IN_SOURCE` | Object, volume or paint names a channel missing from the table or a placeholder |
| `CHANNEL_INVALID`, `FILAMENT_TABLE_CONFLICT`, `NO_FILAMENTS` | Filament table is missing, malformed or contradictory |
| `UNSUPPORTED_COMPONENTS`, `UNSUPPORTED_MULTIPART`, `UNSUPPORTED_MODIFIER_PART` | Multi-volume or modifier geometry |
| `UNSUPPORTED_UNREFERENCED_OBJECT`, `UNSUPPORTED_OBJECT_TYPE` | Objects not placed in the build, or non-model objects |
| `UNSUPPORTED_ELEMENT`, `UNSUPPORTED_ATTRIBUTE`, `UNSUPPORTED_UNIT`, `UNSUPPORTED_XML` | Anything the parser does not explicitly support (including DTDs) |
| `UNSUPPORTED_MULTI_MODEL`, `UNSUPPORTED_EXTERNAL_MODEL` | Additional model parts or production-extension references |
| `UNSAFE_ENTRY` | Absolute path, `..`, backslash, colon, control character, empty segment, or case-only duplicate |
| `ZIP_BOMB`, `ARCHIVE_LIMIT` | Compression ratio above 100:1 on large entries; more than 512 entries; over 512 MiB expanded; over 250 MB archive |
| `SOURCE_INVALID`, `SOURCE_EQUALS_OUTPUT`, `OUTPUT_EXISTS`, `OUTPUT_VALIDATION_FAILED` | Input or publication problems |

## Safety

- Entry count is checked from the End of Central Directory before `archive/zip` builds its table.
- Declared sizes are not trusted: reads are capped and compression ratio is checked.
- XML DTD directives are refused, so entity declarations and entity expansion cannot be used. Element depth and node counts are bounded.
- Nothing is extracted to disk. Output is written to a temporary file in the destination directory and hard-linked into place, so an existing file is never replaced.
- Before publication the output is re-parsed and compared with the source: vertices, triangle indices, paint channels, transforms, extruder assignments, volume ranges and the filament table must all match.

## Verification status (read this before relying on output)

- Paint decoding follows the bitstream grammar in Panda2Prusa's `paint.py` (MIT): hex digits are read from the end of the string, least significant bit first; a leaf is `00` plus a 2-bit state, with `11` plus a 4-bit extension for states 3 to 18; a split is `01`-`11` plus a side index and its children. Codes that are truncated or have non-zero bits after the root are refused. The codec was checked against that reference's decoder for the codes in the tests, but **no code has been checked against a PrusaSlicer-written file**.
- The repository fixture `tests/fixtures/bambu-five-colors.3mf` is synthetic. Its paint codes are `4` (channel 1) and `2C` (channel 5).
- The 4096-character bound on one paint code is a local safety limit, not a PrusaSlicer rule.
- Output carries `slic3rpe:Version3mf` and `slic3rpe:MmPaintingVersion` = 1, as in Panda2Prusa. The volume type is written as `ModelPart`; Panda2Prusa writes `model_part`, which this package does not follow.
- The output part layout (`Slic3r_PE.config`, `Slic3r_PE_model.config`, content types, relationships) was compared with Panda2Prusa's writer. It has not been opened in PrusaSlicer. Confirm with a real round trip before shipping.
- Panda2Prusa (MIT) is the reference project. No code is copied from it or from PrusaSlicer (AGPL-3.0).
- This package does not claim to support all 138 Prusa printer models. It makes no printer-specific decisions and writes no printer profile.

## Tests

```
cd engine && go test ./prusa3mf/...
```

The tests copy the repository fixture to a temporary directory and never write
to it. They cover the fixture round trip (entries, geometry, channel 5 paint,
transforms, extruder and filament table, dropped G-code), sparse and duplicate
channels, no-overwrite and source-protection rules, determinism, the verifier,
the codec, and every refusal code above.
