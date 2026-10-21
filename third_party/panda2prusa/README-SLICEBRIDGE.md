# Panda2Prusa in SliceBridge (vendored)

This directory vendors the Python converter [CupsOhJoe/Panda2Prusa](https://github.com/CupsOhJoe/Panda2Prusa)
for SliceBridge's headless conversion path. It converts Bambu Studio / OrcaSlicer `.3mf` projects
to PrusaSlicer-native `.3mf` projects.

- Upstream: https://github.com/CupsOhJoe/Panda2Prusa
- Pinned commit: `95e4d82d22142295e52913a10b07e054458a8e4e`
- License: MIT, Copyright (c) 2026 Joseph (CupsOhJoe). See [LICENSE](LICENSE).
- Vendoring: plain files copied from that commit, not a git submodule, so builds are reproducible
  and work offline. Every file under `panda2prusa/` plus `LICENSE`, `README.md`, `CHANGELOG.md` and
  `requirements.txt` is byte-identical to the pinned commit. `tests_slicebridge.py` checks this against
  the upstream git blob ids.

## Layout

| Path | Origin | Notes |
| --- | --- | --- |
| `panda2prusa/__init__.py`, `convert.py`, `reader.py`, `writer.py`, `paint.py`, `transform.py`, `ns.py` | upstream, unmodified | Headless conversion core |
| `LICENSE`, `README.md`, `CHANGELOG.md`, `requirements.txt` | upstream, unmodified | Attribution and dependency pin |
| `slicebridge_runner.py` | SliceBridge | Entry point, JSON contract, input and output checks |
| `tests_slicebridge.py` | SliceBridge | Headless unittest suite |
| `README-SLICEBRIDGE.md` | SliceBridge | This file |

Not vendored on purpose: `panda2prusa/cli.py`, `gui.py`, `__main__.py`, `gui_launcher.py`,
`pyproject.toml`, the upstream `tests/` directory and `.github/`. The GUI needs tkinter and is unused.
The upstream test suite expects user sample files that are not in git.

## Runner contract

```
python slicebridge_runner.py --source /abs/in.3mf --output /abs/out.3mf --json [--plate N[,N...]]
```

- `--source` and `--output` must be absolute and end in `.3mf`. `--plate` is optional and repeatable;
  without it, every plate is converted.
- stdout is exactly one JSON object and nothing is written to stderr.
- Success: exit `0`, `{"ok": true, "engine": "panda2prusa", ...}`. The payload has source and output
  SHA-256 values, `stats` (objects, volumes, build items, triangles, vertices, painted triangles),
  `extruder_remap: null`, the `verified` checks, and `printer_profile: {"verified": false}`.
- Failure: exit `1`, `{"ok": false, "code": "...", "error": "..."}`.
- Bad arguments: exit `2`, code `BAD_ARGUMENTS`.

Stable codes:

| Code | Meaning |
| --- | --- |
| `BAD_ARGUMENTS` | Usage error, relative path, wrong extension, bad plate value |
| `SOURCE_OUTPUT_SAME_PATH` | Output resolves to the source, including symlinks and hard links |
| `SOURCE_NOT_FOUND` | Source is missing or not a regular file |
| `OUTPUT_EXISTS` | Output exists, or appeared during conversion. Never replaced |
| `OUTPUT_DIR_UNAVAILABLE` | Output directory is missing or not writable |
| `INPUT_NOT_ZIP`, `INPUT_CORRUPT` | Not a zip, or CRC or deflate failure |
| `INPUT_TOO_LARGE`, `INPUT_TOO_MANY_ENTRIES` | Above the size or entry-count limits |
| `INPUT_SUSPICIOUS_COMPRESSION` | Compression ratio above 100:1 on an entry of at least 16 MiB |
| `INPUT_UNSAFE_PATH`, `INPUT_DUPLICATE_ENTRY` | Entry name escapes the archive, or duplicates another entry case-insensitively |
| `INPUT_ENCRYPTED`, `INPUT_UNSUPPORTED_COMPRESSION` | Only stored and deflated entries are accepted |
| `INPUT_MISSING_MODEL` | No `3D/3dmodel.model` |
| `INPUT_XML_DTD_FORBIDDEN` | `<!DOCTYPE` or `<!ENTITY` in an XML part. Upstream resolves entities, so these are refused |
| `INPUT_PARSE_FAILED` | Upstream could not read the project |
| `INPUT_GEOMETRY_INVALID` | A triangle index points outside its mesh's vertices |
| `NO_GEOMETRY` | No build items with mesh data on the selected plates |
| `ENGINE_UNAVAILABLE` | lxml or the vendored package cannot be imported |
| `CONVERSION_FAILED` | Upstream raised during conversion |
| `OUTPUT_INVALID` | Output zip, model or config could not be read or failed its CRC |
| `OUTPUT_CONTAINS_GCODE` | Output contains a G-code entry |
| `GEOMETRY_MISMATCH` | Object, volume, triangle, vertex or build-item summary differs from the source |
| `PAINT_MISMATCH` | Paint data differs from the source |
| `EXTRUDER_MISMATCH` | Object or volume extruder identity differs from the source |
| `SOURCE_MODIFIED` | Source changed during conversion. Nothing is published |
| `PUBLISH_FAILED` | Hard link failed. The filesystem must support hard links |
| `INTERNAL_ERROR` | Unexpected failure. Reported as JSON, never a traceback |

## Publish and verification flow

1. Validate arguments and paths. Refuse before any parsing.
2. Hash the source. Run bounded archive checks: entry count, name safety, duplicates, encryption,
   compression method, per-entry and total sizes, compression ratio, and no DTD or entities.
3. Parse with upstream `read_bambu_3mf` and compute the expected summary from the source. Out-of-range
   source indices fail here, because upstream would otherwise bind them silently to the next mesh.
4. Create a temporary `.slicebridge-*.partial` file beside the output. Run upstream
   `convert_file(..., plates=..., extruder_map={}, compact_extruders=False)` into it. Empty `{}`
   means no slot remapping, so sparse slots and same-colour distinct slots keep their original numbers.
5. Verify the temporary file. It must pass the zip CRC check, contain no G-code, and match the
   expected summary: object, volume, triangle and vertex counts, build items and transforms, paint
   strings and per-object paint counts, and object and volume extruders. Volume ranges must tile each
   object's triangles exactly.
6. Re-hash the source. If it changed, refuse.
7. `os.link` the temporary file to the output. `link` fails if the output exists, so this is an atomic
   no-replace publish. The temporary file is always removed.

## What this does not do

- It does not claim printer profile compatibility. The output is a PrusaSlicer project. Open it in
  PrusaSlicer and pick your own Prusa printer profile before slicing.
- It does not carry over print settings beyond part names and extruders, such as custom layer heights,
  modifiers, or per-part profiles. See upstream `CHANGELOG.md`.
- It does not keep Bambu plate membership. Selected objects are written together with their build
  transforms, as upstream does.
- It makes no network calls and does not talk to any printer.

## Dependencies and packaging

- Runtime: Python 3.9+ and `lxml>=5.0` (`requirements.txt`). `lxml` is BSD-3-Clause. It must appear
  in the SBOM and third-party bundle for any shipped binary.
- The runner inserts its own directory into `sys.path`, so `python -I slicebridge_runner.py` works.
  A PyInstaller build should use `slicebridge_runner.py` as its entry point. The static imports of
  `panda2prusa.*` and `lxml.etree` are what PyInstaller follows.
- Upstream version strings disagree: `panda2prusa/__init__.py` says `0.1.0` and `pyproject.toml` says
  `1.0.0`. The runner reports the `__init__` value in `engine_version`. The pinned commit is the source
  of truth.

## Tests

```
python3 -m venv /tmp/slicebridge-p2p-venv
/tmp/slicebridge-p2p-venv/bin/pip install -r requirements.txt
cd third_party/panda2prusa
/tmp/slicebridge-p2p-venv/bin/python -B -m unittest tests_slicebridge -v
```

Refusal, archive-check, CLI-refusal and byte-identity tests need only the standard library. The
conversion tests need lxml and skip without it. They run on a synthetic project that has two
components, a painted triangle, and sparse filament slots (1 and 3), and on the Bambu fixtures in the
repository's `tests/fixtures`, or in the directory named by `SLICEBRIDGE_FIXTURES`. Fixtures are read,
never copied or modified.
