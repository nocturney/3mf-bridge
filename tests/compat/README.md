# Conversion fidelity QA (`tests/compat`)

Stdlib-only harness that compares a source 3MF with a converted output. It never
slices, never contacts a printer or the network, and never uses real slicer
settings. Its report is structural evidence only: a pass does not prove the
output prints.

```sh
# Unit tests (synthetic projects generated in temp dirs; repo fixtures are read-only)
python3 -m unittest discover -s tests/compat -p 'test_*.py' -v

# Compare one conversion. Exit 0 = no failed checks, 1 = failure, 2 = unreadable input.
python3 tests/compat/threemf_compat.py --source SRC.3mf --output OUT.3mf \
    [--family auto|orca|bambu|prusa] [--expect-printer-model NAME] \
    [--source-sha256 HEX] [--prior-output-sha256 HEX]
```

`--prior-output-sha256` is the hash of a file that already existed at the output
path before the run; the check fails if the conversion replaced it.

## What is compared

| Check | Meaning | Comparable across families? |
| --- | --- | --- |
| `source_zip_sanity`, `output_zip_sanity` | Readable ZIP, no duplicate or unsafe names, no encryption | Yes |
| `output_components` | `[Content_Types].xml`, `_rels/.rels`, and a 3D model relationship whose target exists | Yes |
| `no_sliced_gcode` | No `*.gcode` / `*.gcode.md5` entries or references | Yes |
| `geometry_counts`, `geometry_hash` | Placed build geometry: recursively resolves `p:path` child `.model` parts and component/build transforms; compares world-space triangles and unique placed vertices | Yes |
| `paint_state` | Paint codes bound to **placed** triangle geometry, including component instances | Only same codec (Orca/Bambu share one) |
| `palette_slots` | Ordered (colour, material) per slot; duplicates kept separate. Generic Panda2Prusa outputs preserve source slots in SliceBridge JSON metadata, **not** native Prusa filament presets. | Yes |
| `virtual_channels` | Channels used by objects and paint (incl. fifth) still exist with the same identity | Paint channels only for Orca/Bambu |
| `object_extruder_mapping` | Per-build-instance extruder assignments aligned by world-space geometry, not metadata order or object IDs; unprovable bindings marked `skip` | When object-to-build binding is provable |
| `machine_scripts_not_carried` | Source `*_gcode` printer scripts are not copied | JSON layout only |
| `process_intent` | Numeric process keys the engine documents as carried over | JSON layout only |
| `non_overwrite` | Output path is not the source; source hash and any pre-existing output hash unchanged | Yes |
| `printer_model` | Only checked when `--expect-printer-model` is given | Yes |

Skipped checks are reported as `skip`, never `pass`.

## Honest limits

- **Families.** Orca and Bambu share one JSON-config layout and one paint codec,
  so they are checked identically. Labels come from `Application` metadata or
  `printer_model`; an unlabelled JSON project defaults to `orca` and says so
  (`basis: layout-default`).
- **Paint decoding.** Only the simple Bambu/Orca fixture codes 4→1, 8→2,
  C→3, 1C→4 and 2C→5 have independently verified direct-channel meanings.
  Real subtriangle paint trees such as BC, CC and 8C are not decoded as a
  single tool/channel: `virtual_channels` is marked `skip` for those projects
  (unless an explicit, provable object-tool mapping fails), while `paint_state`
  still compares the complete raw per-triangle paint codes and world-space
  geometry. A raw-paint match must not be described as validation of physical
  multicolor spool/tool changes. Opaque eight-digit Bambu colors `#RRGGBBFF`
  and six-digit `#RRGGBB` compare equal for QA only; nonopaque alpha does not.
- **Script provenance.** Identical active G-code scripts for two *different*
  printer models fail. If the source and an explicitly named target are the
  same model, identical scripts are `skip` rather than a false copy error or
  an unearned pass: only the installed OEM preset could prove their origin.
- **Prusa is unverified.** `Slic3r_PE*.config` names, the `mmu_segmentation`
  attribute and the INI-style filament lists were written from memory. They
  have not been checked against a real PrusaSlicer export. Prusa checks should
  be treated as a harness-logic smoke test until a real sample is reviewed.
- **Engine output is not exercised here.** The `third_party/bl2std` submodule is
  empty in this checkout and no Go toolchain is installed, so the engine cannot
  produce output locally. The harness can still validate any output file that
  is supplied.
- **Not verified.** Geometric equivalence checks build/component transforms, but not
  slicer interpretation. Non-build object/volume bindings, object names, supports,
  nozzle settings, firmware scripts, slicing and real-printer behaviour remain unverified.

## Files

- `threemf_compat.py` – library and CLI (`compare()`, `main()`).
- `synthetic.py` – deterministic tiny 3MF builders and mutation helpers.
- `test_threemf_compat.py` – unit and mutation tests, including sanitized copies of
  every `tests/fixtures/*.3mf`.
- `../../.github/workflows/conversion-compat.yml` – runs the unit tests on push
  and pull request.
