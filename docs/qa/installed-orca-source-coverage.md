# Installed Orca-family source coverage

Discovery of installed Orca-compatible machine presets, used by `DiscoverOrca` and
`ResolveOrcaBundle` in `native/presets/orca.go`. Inventory paths come from
`orcaFamilyLocations` in `native/presets/options.go`.

## Scanned applications (priority order)

Each application is checked in every trusted install base, in this order:
Windows `Program Files`, `Program Files (x86)`, `%LOCALAPPDATA%\Programs`;
macOS `/Applications`, `~/Applications`.

| Priority | Application | Windows folder | macOS bundle |
|---|---|---|---|
| 1 | Snapmaker Orca | `Snapmaker_Orca` | `Snapmaker Orca.app` |
| 2 | Flash Studio Desktop | `Flashforge\Flash Studio Desktop` | `Flash Studio Desktop.app` |
| 2 | Flash Studio | (not a separate Windows folder) | `Flash Studio.app` |
| 3 | ElegooSlicer | `ElegooSlicer` | `ElegooSlicer.app` |
| 4 | QIDIStudio | `QIDIStudio` | `QIDIStudio.app` |
| 5 | OrcaSlicer (universal fallback) | `OrcaSlicer` | `OrcaSlicer.app` |

Before these, any `BRIDGE_SLICER_PROFILE_<TARGET>` override is scanned as an explicit
local preference. Profiles live under `resources/profiles/<Vendor>/{machine,process,filament}`.

Only applications whose installed layout was checked against Orca-compatible JSON are
listed. Other manufacturer forks (Creality Print, Anycubic, Sovol, and so on) are not
scanned until their installed layout is verified.

## Selection rules

- A machine is offered only when its machine, nozzle, default process and PLA filament
  form a complete chain. Each link must explicitly list the machine as compatible.
- A machine is offered once per vendor and printer name. The first complete copy in
  priority order wins, so a vendor fork beats the generic OrcaSlicer copy. A broken fork
  copy falls back to the complete generic copy.
- Bambu (`BBL`) and `OrcaFilamentLibrary` are never offered as Orca targets.
- Inheritance-only machine presets (`instantiation` false) are skipped silently.
- Models already served by a built-in engine (Snapmaker U1, Flashforge Creator 5,
  ELEGOO CC2, QIDI Plus5) are rejected with a reason.

## Bounds and link protection

| Limit | Value |
|---|---|
| Application roots scanned | fixed list above, plus overrides |
| Vendor folders per root | 64 |
| Machine files per vendor | 512 |
| Profile JSON files per vendor | 20,000 (over this rejects the vendor) |
| Folder depth below a vendor | 4 |
| Bytes per profile file | 4 MiB (larger files are skipped) |

- Recursive searching is not used. Only the listed folders are read.
- Symlinks are not followed. Symlinked vendor folders, machine files and profile files
  are ignored. An application folder whose chain (bundle, Contents, Resources, profiles)
  contains a link is not scanned.
- Install paths never reach the frontend. Invalid profile JSON is reported without the
  file path.

## Test coverage

- `options_test.go`: Windows vendor-first ordering, Mac system and per-user bundles,
  Flash Studio naming, Bambu exclusion, empty bases, override precedence, and the four
  original targets keeping every legacy root.
- `orca_test.go`: symlinked vendor and machine files, symlinked application chain,
  vendor fork over generic duplicate, broken fork falling back, abstract presets,
  scan depth, and end-to-end Windows and Mac install trees using temporary folders.

Go is not installed on the development Mac. These tests run in CI (`native.yml`).

## Not covered (remaining limitations)

- Validated user-created presets under the Orca `user/` folders are not scanned. Their
  inheritance depends on system presets, so they need a separate validation design.
- Apps outside the table above are not scanned, even if they are Orca forks.
- A single unreadable or invalid JSON file rejects its whole vendor, as before.
- The Windows 'Flash Studio' folder name and the Mac bundle names are taken from the
  existing built-in mapping and have not been checked against a live install here.
