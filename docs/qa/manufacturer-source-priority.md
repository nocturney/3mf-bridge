# Manufacturer source priority

Branch: `fix/manufacturer-slicer-first-20261022`, based on `origin/fix/full-catalog-picker-integrated` (5d6bf7d).

## Rule

When a physical printer has several installed targets, the card default and the install
search order follow one source tier order:

| Tier | Source | Examples |
|---|---|---|
| 0 | Manufacturer's own app, or built-in adapter | Snapmaker Orca, Flash Studio Desktop, ElegooSlicer, QIDIStudio, built-in legacy adapter |
| 1 | Generic OrcaSlicer (or a target with no app name) | OrcaSlicer |
| 2 | Generic PrusaSlicer, or the project-only export | PrusaSlicer, `prusa-project` |
| 3 | Catalogue-only, not ready | catalogue ID with no installed target |

Within the top tier of a card, the built-in adapter comes first, then a 0.4 mm variant, then the lowest nozzle.

Install search (`native/presets/options.go`): applications are tried in priority groups.
Each group is checked in every install base before the next group, so a vendor app in
LocalAppData beats generic OrcaSlicer in Program Files. Windows QIDI order is QIDIStudio,
then OrcaSlicer. An explicit `BRIDGE_SLICER_PROFILE_*` override replaces the search and
fails closed when its vendor folder is missing.

## Evidence

- `node --test tests/*.test.js` in `native/`: 139 pass, 0 fail (baseline before this change: 132 pass).
- `node --test tests/printer-cards-ui.test.js`: 20 pass. Against the original `printer-picker.js`, the
  Flash Studio, QIDIStudio, Snapmaker Orca, ELEGOO and PrusaSlicer tests fail. Those fail on the old code and pass on the new.
- `node --test browser-extension/build.test.cjs browser-extension/setup-flow.test.cjs`: 8 pass.
- `native/presets/options_test.go`: updated Windows QIDI order test and new tests for Windows
  auto and fallback, explicit override, macOS order, and malformed targets. **Not run**: no Go toolchain on the host.

## Behaviour changes to note

- The Windows QIDI candidate order changed. Existing saved selections and overrides are unchanged.
- Vendor tier outranks generic Orca regardless of nozzle, so a vendor 0.6 mm target is the default over a generic 0.4 mm one.
- The cover lookup (`my-printers.js`) walks the same variant order. It now asks the vendor slicer first and stops once a cover is found.
  The existing test expectation was updated from two lookups to one.

## Not done

- No workflow, release, or version changes. No new engine. No Bambu or Cura target.
- Go tests are unverified here and must be run with `go test -count=1 ./presets/...` in CI.
