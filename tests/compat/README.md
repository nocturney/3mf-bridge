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
| `geometry_counts`, `geometry_hash` | Vertex/triangle counts; order-independent hash of canonical triangles | Yes |
| `paint_state` | Per-triangle paint code multiset (triangle identity = sorted vertex coordinates) | Only same codec (Orca/Bambu share one) |
| `palette_slots` | Ordered (colour, material) per slot; duplicates kept as separate slots | Yes |
| `virtual_channels` | Channels used by objects and paint (incl. fifth) still exist with the same identity | Paint channels only for Orca/Bambu |
| `object_extruder_mapping` | Ordered object extruder values | Yes (same metadata idea) |
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
- **Paint decoding.** Bambu/Orca codes are decoded only as 4→1, 8→2, C→3, 1C→4,
  2C→5. That scheme is inferred from repo fixtures and generators, not from the
  upstream codec. Paint is still compared on the raw code, so unknown codes are
  compared, not dropped.
- **Prusa is unverified.** `Slic3r_PE*.config` names, the `mmu_segmentation`
  attribute and the INI-style filament lists were written from memory. They
  have not been checked against a real PrusaSlicer export. Prusa checks should
  be treated as a harness-logic smoke test until a real sample is reviewed.
- **Engine output is not exercised here.** The `third_party/bl2std` submodule is
  empty in this checkout and no Go toolchain is installed, so the engine cannot
  produce output locally. The harness can still validate any output file that
  is supplied.
- **Not verified.** Build transforms, object names, supports, nozzle settings,
  firmware scripts, slicing results and real-printer behaviour are not checked.

## Files

- `threemf_compat.py` – library and CLI (`compare()`, `main()`).
- `synthetic.py` – deterministic tiny 3MF builders and mutation helpers.
- `test_threemf_compat.py` – unit and mutation tests, including sanitized copies of
  every `tests/fixtures/*.3mf`.
- `../../.github/workflows/conversion-compat.yml` – runs the unit tests on push
  and pull request.
