# SliceBridge — release artifact & artwork preflight

**19 October 2026. Evidence only; public release remains ON HOLD.**

## Snapshot and scope

The draft [`v0.4.0-beta.3` Release](https://github.com/nocturney/slicebridge/releases/tag/v0.4.0-beta.3) references source commit `5b9cd0a1f961a872f00d4d7e82ced567efd170ee`, **not** the latest development HEAD. Neither draft ZIP was published as an approved stable/beta download. This report inspects the existing draft artifacts; it does **not** certify actual printing, source correspondence, privacy or security.

### Draft ZIP checks (read-only)

| Draft file | SHA-256 | ZIP integrity / path safety | Signature |
| --- | --- | --- | --- |
| `SliceBridge-Windows.zip` | `0ea7bbc64853351f9f14cf17c3439c122240c79e5dfb9dd670ee6b180563af2c` | CRC test passed; 1 file; no duplicate or unsafe archive paths | Extracted `SliceBridge.exe`: Windows Authenticode **NotSigned** |
| `SliceBridge-macOS-Apple-Silicon.zip` | `759111dd9a015b9be0fc68f788f2e990d4b817d40152dfb6a0fea40e3588d3d5` | CRC test passed; 9 entries; no duplicate or unsafe archive paths | `codesign --verify` passed **ad-hoc** integrity, but `spctl --assess` **rejected** the app. No Apple TeamIdentifier. |

The Windows archive contains an executable; the macOS archive contains an arm64 `SliceBridge.app`. Hashes describe the **draft assets only** and do not apply to any rebuilt/signed files.

### Signing environment observed

- **Chris Windows:** 0 visible CurrentUser Code Signing certificates; 0 with a private key. `signtool.exe` was not found on PATH. This says nothing about credentials available on other machines or CI.
- **MacMiniOffice:** `codesign`, `notarytool` and Apple Command Line Tools are available; `security find-identity -v -p codesigning` reported **0 valid signing identities**. This says nothing about certificates on other Macs or in CI secrets.
- **Required later:** valid publisher signing identity/credentials (Windows as appropriate, Apple Developer ID Application and notary access for general macOS distribution), secure CI configuration, signed rebuilt binaries, Apple notarization/stapling, fresh-machine Gatekeeper/SmartScreen observations and verified per-release hashes. Never commit private keys or passwords.

## Approved artwork provenance — tracked blobs

The [branding handoff](https://github.com/nocturney/slicebridge/blob/feature/browser-compat-and-icon/native/branding/README.md) records that the owner approved **Direction 1 / Mint–Forest** icon and wave. Git blob hashes confirm the website and README **reuse precisely the approved sources**, without a substituted image.

| Asset / use | Ref | Git blob SHA-1 |
| --- | --- | --- |
| Approved icon — source: `native/branding/direction1-reference/approved-icon.png` | `feature/browser-compat-and-icon` | `fedf2854f8e0da4b19f637e2a1f576bb4fcf26aa` |
| Approved wave — source: `native/branding/direction1-reference/approved-wave.png` | `feature/browser-compat-and-icon` | `65bb6eef9a5f093626bf0c6bca358679f8b59567` |
| Website approved icon: `docs/assets/approved-icon.png` | `main` | `fedf2854f8e0da4b19f637e2a1f576bb4fcf26aa` |
| Website approved wave: `docs/assets/approved-wave.png` | `main` | `65bb6eef9a5f093626bf0c6bca358679f8b59567` |
| README top banner: `assets/slicebridge-approved-banner.png` | `main` | `65bb6eef9a5f093626bf0c6bca358679f8b59567` |
| Website social card (composite): `docs/assets/social-preview.png` | `main` | `0f39ea29fcf4a9cdbdcb4f7b0555012c8273bf8f` |
| Native Windows .ico: `native/build/windows/icon.ico` | `feature/browser-compat-and-icon` | `57a6ff373f658b978a9518790b1966948973ac1e` |
| Native macOS .icns: `native/build/darwin/icon.icns` | `feature/browser-compat-and-icon` | `e796480b9f045e0a4c9bd71cdbdf05b4ab3bc98b` |
| Browser extension icon 128: `browser-extension/shared/icon-128.png` | `feature/browser-compat-and-icon` | `d37aad75ea0cbe11af0d90a07f489a94d9ac76d4` |
| Legacy rejected SVG — do not use: `native/branding/slicebridge-symbol.svg` | `feature/browser-compat-and-icon` | `fbb81e2d654d1e81e84d44cf5ee51eda51509130` |

- `docs/assets/social-preview.png` is a 1280×640 **composite made from the approved icon and wave**, not an untouched original image. Reuse on the website is technically confirmed; repository *Social preview* in GitHub Settings still requires a separate upload.
- Native Windows/macOS and extension icons are described as crops/resizes of approved pixels by `native/branding/build-icons.cjs`. This is supported by the build script and design README, **not an independent licensing/copyright assignment**.
- `native/branding/slicebridge-symbol.svg` is specifically marked as a **rejected** interpretation. The active native `index.html` uses `brand-icon.png`, not that SVG. Remove unused rejected art before packaging if it could confuse downstream builders; coordinate any app-side change with the development workflow.
- **Open issue:** provenance/owner approval is not, by itself, independent proof of copyright ownership, trademark registration or a transferable image license. Confirm rights to the exact artwork and distribution permissions before final release. Third-party vendor logos are not documented as part of the approved brand assets.

## Next release gate

1. Resolve [rights and licenses #9](https://github.com/nocturney/slicebridge/issues/9) with exact-build module/asset notices and GPL corresponding source.
2. Validate the **current** release candidate on Windows and macOS and record slicer import, physical nozzle/material and safe print outcomes separately.
3. Sign the rebuilt Windows artifacts as appropriate; sign/notarize/staple macOS artifacts; re-test on clean machines.
4. Recompute SHA-256 **after signing and packaging**, record exact source commit and approved CI/QA evidence, then seek explicit release approval.
5. Keep GitHub Release draft and website download links unpublished until all gates pass.

This document does not change code or release settings, and **cannot substitute for independent QA or legal review**.
