# Panda2Prusa sidecar: packaging and conversion QA

This directory holds the QA for the bundled Panda2Prusa sidecar (`slicebridge-prusa-engine`).
It is stdlib `unittest` only and never touches a printer, a slicer install, the network, or
an external conversion service.

| File | What it checks | Needs the vendor? |
| --- | --- | --- |
| `test_packaging.py` | Staging into `native/build/bin` (Windows) and `SliceBridge.app/Contents/Resources` (macOS), manifest hash, MIT license gate, symlink and overwrite refusals, pins, CI text | No |
| `test_conversion_contract.py` | `FixtureFacts` and `HarnessSelfChecks` run always. `EngineContract` runs the sidecar on `tests/fixtures/bambu-five-colors.3mf` | `EngineContract` only |
| `panda_qa.py` | Shared stdlib helpers: 3MF reader, comparator, stale-gcode scan, runner invocation | n/a |

## Run

```sh
python3 -m unittest discover -s tests/panda -p 'test_*.py' -v
```

Requires Python 3.11 or newer in CI. The tests also run on older interpreters, because they
only use the standard library.

## Skip and failure semantics

- `EngineContract` is skipped **only** while `third_party/panda2prusa/` lacks any of
  `slicebridge_runner.py`, `panda2prusa/`, or `LICENSE`.
- Once the vendor is present, a missing or failing engine is a test **error**, not a skip.
- `HarnessSelfChecks` prove the comparator catches each defect class. They compare against a
  Prusa-shaped reference built from the fixture. That reference is a harness check, not engine
  output, and passing it says nothing about the real converter.

## Which engine runs

`panda_qa.engine_command()` picks the first available:

1. `SLICEBRIDGE_PRUSA_ENGINE=/path/to/exe` (explicit override)
2. `dist/prusa-sidecar/slicebridge-prusa-engine[.exe]` (output of the build subcommand)
3. `python third_party/panda2prusa/slicebridge_runner.py` (needs `lxml` in that interpreter)

The contract is `--source SRC --output OUT --json`: exit 0, one JSON document on stdout, and a
3MF ZIP at `OUT`.

## What the engine contract asserts

Each test is an independent claim about the converted 3MF, checked against the Bambu source:

- **Geometry and triangles**: object-local triangle records (coordinates and paint) match as a
  multiset. Coordinates are compared at 0.0001 mm.
- **Build transforms**: every build item's 4x4 matrix matches, in count and value.
- **Paint bitstreams**: every `paint_color` / `mmu_segmentation` string is carried verbatim,
  including nested codes such as `2C`.
- **Extruder assignments**: object-level `extruder` values match, with objects matched by
  geometry signature rather than by ID. Volume-level values are compared as a sorted multiset.
- **Numbered slots**: the output filament colour list equals the source list, in slot order,
  with duplicates kept. A derived fixture gives slots 1, 2 and 4 the same colour. All five slots
  must survive.
- **No stale output**: no `*.gcode` or `*.gcode.md5` entries, and no bytes from the source's
  stale gcode or `UNSAFE_BAMBU_SCRIPT` anywhere in the package.
- **No overwrites**: the source bytes are unchanged, converting onto the source path is refused,
  and a symlinked output path does not write through to its target.

### Assumptions to confirm against the real vendor

These are the formats the comparator reads. They are taken from the brief and from
`tests/compat/threemf_compat.py`, and the upstream Panda2Prusa output was not inspected while
writing these tests:

- Prusa object and volume extruders live in `Metadata/Slic3r_PE_model.config` as
  `<metadata key="extruder">` (object level, and inside `<volume firstid lastid>`).
- Prusa filament colours live in `Metadata/Slic3r_PE.config` as `filament_colour = "#…;#…"`.
  If that is absent, `Metadata/project_settings.config` is used.
- Paint is compared under `paint_color` or `mmu_segmentation`, by exact string.

If the real output differs, fix the reader in `panda_qa.py`. Do not loosen the assertions.

## Packaging

`scripts/package-prusa-sidecar.py` is the single entry point. Relative paths resolve from the
repository root. `scripts/prusa-sidecar-requirements.txt` pins PyInstaller and sets the lxml
floor.

```sh
python3 -m pip install -r scripts/prusa-sidecar-requirements.txt
python3 scripts/package-prusa-sidecar.py build                 # -> dist/prusa-sidecar/
python3 scripts/package-prusa-sidecar.py stage  --platform windows --bin-dir native/build/bin
python3 scripts/package-prusa-sidecar.py stage  --platform macos   --bin-dir native/build/bin
python3 scripts/package-prusa-sidecar.py verify --platform macos   --bin-dir native/build/bin --smoke
python3 scripts/package-prusa-sidecar.py print-ci-steps            # the workflow steps for CI
```

- `build` needs Python 3.11+, the pinned PyInstaller, lxml 5 or newer, and a complete vendor
  tree. It refuses a non-MIT `LICENSE`. It runs a smoke conversion of the fixture and writes
  `manifest.json` with the sidecar SHA-256.
- `stage` refuses symlinked destinations and never writes to `SliceBridge.exe` or
  `SliceBridge.app`. Writes go through a temp file and an atomic replace. On macOS it ad-hoc
  re-signs the sidecar and then the app, because adding files to `Resources` breaks the seal.
- `verify` checks the staged hash against the manifest, the license text, and (macOS) the seal.
  With `--smoke` it also runs the staged sidecar on the fixture.
- Builds are verified by manifest hash, not by byte identity. Two builds of the same inputs
  produced different bytes during scratch testing, so compare manifests, not raw hashes.

Windows and macOS differ in layout:

| Platform | Sidecar and license land in | Shipped by |
| --- | --- | --- |
| Windows | `native/build/bin/` beside `SliceBridge.exe` | the zip step, which must name the sidecar and the license |
| macOS | `SliceBridge.app/Contents/Resources/` | the existing `ditto` step, which packs the whole `.app` |

`python3 scripts/package-prusa-sidecar.py print-ci-steps` prints the exact workflow steps.
