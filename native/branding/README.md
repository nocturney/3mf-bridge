# SliceBridge — Approved Identity / Direction 1

The owner selected **SliceBridge**, direction 1, on October 19, 2026.

**Positioning:** A focused lightweight 3MF project bridge between Bambu
and native printer slicers. This is *not* another slicer, cloud service,
or 3D geometry editing suite.

**Wordmark:** SliceBridge (no space). Sentence-case display.
**English strapline:** Bridge your prints.
**Hebrew strapline:** מחברים בין סלייסרים.
**User interface:** Hebrew RTL + English LTR only.

## Mint / Forest palette

| Role | Name | Hex |
| --- | --- | --- |
| Very light surface | Mint Foam | `#DFF6EE` |
| Secondary soft mint | Light Mint | `#A9E5D0` |
| Bright accent | Mint | `#5BBFA8` |
| Primary controls | Forest | `#2F7F73` |
| Primary text, deep contrast | Deep Forest | `#193F38` |

Prefer white/near-white working surfaces and deep forest for readable
text. Avoid applying pale mint as foreground on white without contrast.

## Logo / app icon

The approved direction uses a **single arch formed from five stacked,
flowing 3D-print layers**. Mint highlights on the upper arch and deep
forest towards the base signify different slicers joined by one bridge.
A rounded pale-mint icon tile matches both native OS environments.

**Approved visual sources (do not redraw):**
- `direction1-reference/approved-icon.png`: exact user-approved icon crop.
- `direction1-reference/approved-wave.png`: exact user-approved banner crop.
- The earlier `slicebridge-symbol.svg` is a rejected interpretation, NOT the production icon master.

`build-icons.cjs` resizes the approved icon into platform files, crops only the wave region of the approved banner so the accessible HTML text appears once, and retains the source crops for reproducibility.

Derived generated assets (committed):
- `../build/appicon.png`: Wails app icon master (1024 px)
- `../build/windows/icon.ico`: Windows taskbar/executable multi-resolution
- `../build/darwin/icon.icns`: macOS Dock/application icon
- `../frontend/dist/brand-icon.png`: app toolbar mark from approved pixels
- `../frontend/dist/slicebridge-banner.png`: wave-only approved-pixel crop, with live banner text in HTML
- `../../browser-extension/shared/icon-{16,32,48,128}.png`: extension icons
- `slicebridge-preview.png`: earlier design review only

For development rebuilds: `npm install --no-save --prefix native/branding sharp`,
then `node native/branding/build-icons.cjs` from the repository root.
Package browser builds with `node browser-extension/build.cjs`; install `dist/extensions/chromium` as an unpacked Edge/Chrome extension and load `dist/extensions/firefox` as a temporary Firefox add-on for testing. These are unsigned development artifacts, not store-ready releases.
Go/Wails production binaries do not depend on Sharp or Node.

## UI decisions retained

- OS chrome, resize and scroll are entirely native WebView2 / WebKit.
- The update checker and application version live together in the **bottom
  status/footer bar**, not amid title and language controls.
- The results panel has **Clear results**; adding genuinely new files
  clears the prior report automatically; no JSON export button.
- Sliders and sync icons use fixed 17 px stroke SVGs, never text glyphs
  or platform-dependent emoji.
- Existing printer/nozzle and window preferences migrate automatically
  from the old `3MF Bridge` config folder to `SliceBridge` when edited.
  The legacy settings file is left untouched.

Do not rename printer target IDs, change G-code safety, alter the
approved C5 nozzle map, or merge the old Electron release as part of
the visual rebrand.
