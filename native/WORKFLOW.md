## SliceBridge Native — experimental dev.8 (0.4.0-beta.4.dev.8)

The native desktop workflow now supports:
1. **Saved per-printer preferences**: actual physical target nozzle
   (Auto/0.2/0.4/0.6/0.8, or 0.25 on Creator 5); preferred
   locally-installed slicer (QIDIStudio or OrcaSlicer on applicable systems);
   optional material-specific native filament presets for PLA/PETG/ABS/ASA/TPU.
   Preferences are validated against the real installed vendor profile data,
   stored as an atomic local JSON file in the OS user config directory
   (Windows: `%AppData%/Velvet Factory/SliceBridge/settings.json`;
   macOS: `~/Library/Application Support/Velvet Factory/SliceBridge/settings.json`).
   No preferences or filenames are sent to any cloud service.
   **Physical nozzle** settings are operator-specified; the tool cannot
   probe your actual printer's hardware. Set them accurately before slicing.
2. **Detailed per-output comparison** of source/target nozzle,
   source vs converted 3MF object/instance/plate counts where available,
   filament slot count, materials and colors, validated triangles,
   stale slice caches removed and preserved print keys. A complete batch
   report can be exported to local JSON. Source 3MF files missing nozzle
   metadata are explicitly shown as assumed 0.4 mm.
3. **Open output in its own installed slicer** after conversion.
   Only successful outputs from the current app session may be opened,
   and executable paths come from hard-coded, locally detected applications,
   never from arbitrary UI-supplied paths.
4. **Multiple source projects / folders**: select several 3MF files or
   recursively add a folder via native dialogs or OS drag/drop. Deduplicate,
   skip past generated outputs during folder traversal, do not follow
   symlinks, and refuse batches above 200 source projects. For each project,
   up to four target conversions run in parallel; projects process
   one-by-one to bound memory and give incremental progress.

### Window behavior / desktop interaction

- OS-native title bar and resize corners. The app no longer automatically
  resizes or recenters itself on language switch, status changes or results.
- Window width/height persist between launches in the same user profile.
  The native app always opens with a bounded minimum size; displays adapt.
- Top toolbars and bottom action bar stay fixed; input queue, printer
  selections and result cards scroll *inside* the workspace. A narrow-window
  fallback scrolls the workspace instead of clipping controls.
- Keyboard: Ctrl/Cmd+O add files, Ctrl/Cmd+Shift+O add folder,
  Ctrl/Cmd+, printer settings.
- The Hebrew RTL / English LTR toggle remains available at any time.
- A closeable on-demand GitHub release update notice is retained.
- Native asset JavaScript is plain JS, no bundled Node, npm, Electron or
  Chromium runtime. The output remains a single Windows executable or
  compact signed-local macOS app bundle.

### QA

Windows:
```powershell
cd native
$env:BRIDGE_TEST_INSTALLED='1'
go test -count=1 -v ./...
wails build -s
# Optional CSS/JS desktop smoke (requires existing dev Electron):
../node_modules/electron/dist/electron.exe tests/layout-smoke.cjs
```

macOS:
```sh
cd native
BRIDGE_TEST_INSTALLED=1 go test -count=1 -v ./...
wails build -s
```

The Electron-powered **layout smoke is development-only**: it exercises
the actual native HTML/CSS/JS with a stub bridge in 1100x750 and 780x580
windows, verifies bilingual direction switching does not change OS window
size, saved-settings modal and dismissal, multi-file rendering, scrolling
of 32-file and 128-result queues, and persistent action footer.
Production bundles still use system WebView2/WKWebView.

Real-slicer opening from the new Native UI and physical prints require
independent operator QA. Beta 2 of the *old* Electron app remains untouched.
