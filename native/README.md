# SliceBridge Native (Wails / Go)

SliceBridge Native is the primary desktop app and uses native OS webviews:
- Windows 10/11: Microsoft Edge WebView2 Runtime (installed in Windows 11 and verified on Chris)
- macOS: system WKWebView / WebKit

The Wails desktop app is merged into `main`; older Electron code remains in the source tree as a legacy prototype.

**Single Go executable**: no Electron/Chromium payload, no Node.js runtime
in production, no external 3MF converter binary. English and Hebrew are the
only shipped UI languages. Toggle between them in the top bar; Hebrew is RTL.

## Conversion features retained

- Real editable project conversion for Snapmaker U1, Flashforge Creator 5,
  ELEGOO Centauri Carbon 2 and QIDI Plus5, with parallel execution.
- Bambu 3MF nozzle auto-detection: 0.2/0.4/0.6/0.8 mm.
  Creator 5 automatically maps source 0.2 mm to its 0.25 mm profile,
  with no confirmation dialog, as approved by the owner.
- Reads *installed* machine, process and filament presets directly from
  Flash Studio Desktop, ElegooSlicer, Snapmaker Orca and OrcaSlicer/QIDIStudio.
  Vendor G-code and temperature curves are never redistributed in the app.
  Discovery covers system-wide Program Files (including x86) and user-local
  Programs on Windows, and both /Applications and ~/Applications on macOS.
  Supported app IDs and vendor folders remain allowlisted; no filesystem-wide
  scan or executable path supplied by the UI is accepted.
- U1 0.4 mm continues using the field-validated builtin converter profile,
  including on machines without Snapmaker Orca installed.
- Detects material or profile mismatches and fails that target instead of
  issuing fake output; checks mesh integrity, removes stale sliced G-code,
  and uses atomic no-overwrite publishing (source project is unchanged).
- **Output naming:** model-specific results use the normalized physical printer model,
  not the slicer vendor (`Creator5Pro-project.3mf`, `CR10-project.3mf`,
  `Neptune4Pro-project.3mf` or `U1-project.3mf`). Repeated outputs use
  `-2`, `-3`, etc. Source names remain unchanged. The generic unbound
  PrusaSlicer project has no selected printer and therefore retains `PRUSA-`.
- **Printer pictures:** local installed machine cover PNGs only. The exact
  image in the selected printer profile's slicer is preferred; if missing,
  matching vendor/model images in other allowlisted locally installed Orca
  forks, Bambu Studio and PrusaSlicer are eligible fallbacks. A non-OEM cover
  does *not* supply a machine/filament preset, and similar model names are
  never treated as interchangeable. No network lookup or bundled art.
- For projects with more than four Bambu source color/material channels,
  preserve **every original virtual filament**, its material type, its color
  and its painted/object assignments in the converted 3MF. This is NOT
  permission to print with more tools than the printer physically has:
  users choose merges/removals and verify settings in the destination slicer.
  The app never combines slots automatically, even if colors match.
  Native vendor materials still require compatible installed profiles;
  some slicer builds may limit imported virtual slots to physical tools.
- Source file picker, native drag/drop, output-folder opener, independent
  per-target successes/errors, source/target nozzle in completion report.

## PrusaSlicer project (generic)

The fixed **PrusaSlicer project** target writes `PRUSA-<name>.3mf` beside the
source by running the vendor Panda2Prusa engine. The row appears only when the
bundled engine is available:

- Windows: one `SliceBridge.exe` includes the engine and upstream MIT license.
  On first use, a SHA-256 verified payload is extracted automatically to the
  user's versioned application cache; no other download or Python is needed.
- macOS: the engine stays inside `SliceBridge.app/Contents/Resources/`.

Contract: `slicebridge-prusa-engine --source <abs .3mf> --output <abs .3mf> --json`
prints one JSON object, either `{"ok":true,"engine":"panda2prusa",...}` or
`{"ok":false,"code":"UPPERCASE","error":"..."}`. Go never converts geometry.
Each output must pass a ZIP content check and keep every source filament slot;
otherwise the output is removed and the target reports a coded failure.
Missing or damaged embedded engine files fail closed. Release builds never
fall back to PATH or system Python. Developers may set
`SLICEBRIDGE_PRUSA_ENGINE` to an absolute executable for tests only.

The output is a generic PrusaSlicer project. Choose the exact printer model in
PrusaSlicer after opening it. Nothing here claims a printer-specific profile or
a physical print result.

## About, version and community updates

The **About / אודות** window shows the installed build version, a manual **Check
for updates / בדיקת עדכונים** button, GPL-3.0-only, third-party credits, and
links to the exact stamped source commit and [public development
Releases](https://github.com/nocturney/slicebridge/releases). The complete
license and notices are **bundled locally** and readable offline.

The update check queries the public `nocturney/slicebridge` GitHub Releases
API only when clicked; no GitHub CLI, account, automatic installation or
background polling is needed. It considers published prereleases and offers
only platform-matching downloads. Any network failure remains visible rather
than silently claiming the build is current.

Development downloads can be unsigned, so Windows/macOS may show an
unknown-publisher warning. Inspect target slicer, nozzle and material
settings before printing. Go tests for this updater live in `updates/`
and native UI tests in `tests/about-ui.test.js` and `tests/update-ui.test.js`.

## Development

Requires Go 1.25+, Wails v2.17.0 and native build tools (WebView2 on Windows,
Xcode Command Line Tools on macOS). No npm install is needed to build the
static frontend.

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0
git submodule update --init
cd native
go test ./...
wails build -s
```

Binaries will be in `native/build/bin`: Windows `SliceBridge.exe`,
macOS `SliceBridge.app`. The macOS application may need signing and
notarization before distributing to other users. Do not ship an unsigned
Mac app as a fully public installer.

### Host integration tests

On a PC or Mac where all four target slicers and presets are installed:

```sh
cd native
BRIDGE_TEST_INSTALLED=1 go test -v ./...
```

These tests exercise four-way parallel conversions against the installed
profile data and verify nozzle mapping, no-overwrite and process integrity.
On Windows PowerShell use `$env:BRIDGE_TEST_INSTALLED='1'; go test -v ./...`.

Override a profile location with
`BRIDGE_SLICER_PROFILE_U1`, `BRIDGE_SLICER_PROFILE_C5`,
`BRIDGE_SLICER_PROFILE_CC2`, or `BRIDGE_SLICER_PROFILE_QIDI`,
each pointing to its parent `resources/profiles` folder.

## Trust / QA scope

The owner previously reported successful loading and slicing for all four
printers with the original Electron app and a single four-printer operation.
The native-Wails rewrite reuses the **same Go conversion implementation**
but reimplements the profile filesystem resolver. Treat Wails GUI and macOS
native-runtime smoke as independent QA tasks. No physical print validation
is claimed. Publish future native binaries only after exact-version Windows/macOS CI and representative QA.

## Upstream

Go conversion code: BryanSmee/bl_to_std pinned as the existing GPL-3.0
Git submodule. Wails v2: https://github.com/wailsapp/wails.
Installed vendor preset JSON files are not included in the repository,
embedded binary or redistribution artifacts.
