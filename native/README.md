# SliceBridge Native (Wails / Go)

3MF Bridge Native replaces Electron with native OS webviews:
- Windows 10/11: Microsoft Edge WebView2 Runtime (installed in Windows 11 and verified on Chris)
- macOS: system WKWebView / WebKit

This project lives on its own `feature/native-webview-shell` branch and
preserves Electron Beta 2 on `feature/real-3mf-engine`.

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
- U1 0.4 mm continues using the field-validated builtin converter profile,
  including on machines without Snapmaker Orca installed.
- Detects material or profile mismatches and fails that target instead of
  issuing fake output; checks mesh integrity, removes stale sliced G-code,
  and uses atomic no-overwrite publishing (source project is unchanged).
- Source file picker, native drag/drop, output-folder opener, independent
  per-target successes/errors, source/target nozzle in completion report.

## Checking for GitHub updates

The **Check for updates / בדיקת עדכונים** button is explicit and manual.
It requests GitHub Releases for `nocturney/3mf-bridge`, compares the
installed SemVer version against published stable **and beta** releases, and
checks for an archive built for the running Windows or macOS architecture.
The UI displays the new version and opens its GitHub Release page in the
system browser when selected. **No silent download, installation, or
periodic background requests.**

The repository is currently **private**: public GitHub API requests return
HTTP 404. The updater therefore uses the existing signed-in GitHub CLI
(`gh`) on the local machine, with credentials managed by GitHub CLI and
the operating system keychain. The app does **not** bundle, request,
store or expose a GitHub token to the HTML frontend. On a new machine,
install GitHub CLI and run `gh auth login` with an account authorized to
access this repository. If authentication is missing, the UI says so
instead of pretending that the software is up to date.

At the time of the implementation there are **no published Releases**,
only CI preview artifacts. In this case the updater explicitly reports
"No GitHub Releases yet" and offers to open the native beta build page.
CI artifacts are **not** counted as versioned Releases, and expire after
14 days.

For future Releases, tag with a full SemVer (e.g. `v0.4.0-beta.2`),
bump `native/updates_bridge.go`'s `appVersion`, and attach the matching
asset file to the GitHub Release:
- Windows x64: `3MF-Bridge-Windows-native.zip`
- macOS Apple Silicon: `3MF-Bridge-macOS-Apple-Silicon.zip`

There is no Intel Mac binary yet; its update checker will not offer an
Apple Silicon download. All release pages are constructed from a fixed
trusted `github.com/nocturney/3mf-bridge` path, never an arbitrary URL
returned by GitHub.

Update backend tests use an isolated local HTTP mock. Optionally run
`BRIDGE_TEST_GITHUB=1 go test -v ./updates` to test the real authenticated
private-repository request. The bilingual update-button JavaScript test
is `node --test tests/update-ui.test.js`.

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
is claimed. Do not merge the stacked PR until native platform builds and
owner smoke QA pass.

## Upstream

Go conversion code: BryanSmee/bl_to_std pinned as the existing GPL-3.0
Git submodule. Wails v2: https://github.com/wailsapp/wails.
Installed vendor preset JSON files are not included in the repository,
embedded binary or redistribution artifacts.
