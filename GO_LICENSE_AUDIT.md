# Go module license inventory — 19 October 2026

**Preliminary source-manifest audit, NOT a legal conclusion or complete binary SBOM.**

**Snapshot:** [`feature/browser-compat-and-icon` at `3bf292876406edb891351fe160ca146532299014`](https://github.com/nocturney/slicebridge/tree/3bf292876406edb891351fe160ca146532299014/native). Examined `native/go.mod`, `engine/go.mod`, and the version-specific module ZIPs downloaded from `proxy.golang.org` **without executing those dependencies**. Workstation: Chris Windows. The working tree was clean and matched that commit at the time of inspection.

| Classification from license text | Count |
| --- | ---: |
| MIT-family | 18 |
| BSD-family | 10 |
| ISC | 1 |
| Apache-2.0 | 1 |
| Local replacement — assess separately | 2 |
| **Total manifest entries** | **32** |

**30 external module archives** provided candidate LICENSE/COPYING files in their exact requested versions. Classifications are heuristic text matches (asterisk below), **not verified SPDX classifications**, not proof of compliance, and not a determination that every compiled file uses the same license. Re-run when `go.mod` or lock dependencies change.

| Go module | Manifest version | Detected candidate license |
| --- | --- | --- |
| `git.sr.ht/~jackmordaunt/go-toast/v2` | `v2.0.3` | MIT-family* |
| `github.com/bep/debounce` | `v1.2.1` | MIT-family* |
| `github.com/BryanSmee/bl_to_std` | `v0.0.0` | local replacement |
| `github.com/gabriel-vasile/mimetype` | `v1.4.16` | MIT-family* |
| `github.com/go-ole/go-ole` | `v1.3.0` | MIT-family* |
| `github.com/godbus/dbus/v5` | `v5.1.0` | BSD-family* |
| `github.com/google/uuid` | `v1.6.0` | BSD-family* |
| `github.com/gorilla/websocket` | `v1.5.3` | BSD-family* |
| `github.com/jchv/go-winloader` | `v0.0.0-20210711035445-715c2860da7e` | ISC* |
| `github.com/labstack/echo/v4` | `v4.15.3` | MIT-family* |
| `github.com/labstack/gommon` | `v0.5.0` | MIT-family* |
| `github.com/leaanthony/go-ansi-parser` | `v1.6.1` | MIT-family* |
| `github.com/leaanthony/gosod` | `v1.0.4` | MIT-family* |
| `github.com/leaanthony/slicer` | `v1.6.0` | MIT-family* |
| `github.com/leaanthony/u` | `v1.1.1` | MIT-family* |
| `github.com/mattn/go-colorable` | `v0.1.14` | MIT-family* |
| `github.com/mattn/go-isatty` | `v0.0.22` | MIT-family* |
| `github.com/nocturney/3mf-bridge/engine` | `v0.0.0-00010101000000-000000000000` | local replacement |
| `github.com/pkg/browser` | `v0.0.0-20240102092130-5ac0b6a4141c` | BSD-family* |
| `github.com/pkg/errors` | `v0.9.1` | BSD-family* |
| `github.com/rivo/uniseg` | `v0.4.7` | MIT-family* |
| `github.com/samber/lo` | `v1.49.1` | MIT-family* |
| `github.com/tkrajina/go-reflector` | `v0.5.8` | Apache-2.0* |
| `github.com/valyala/bytebufferpool` | `v1.0.0` | MIT-family* |
| `github.com/valyala/fasttemplate` | `v1.2.2` | MIT-family* |
| `github.com/wailsapp/go-webview2` | `v1.0.22` | MIT-family* |
| `github.com/wailsapp/wails/v2` | `v2.17.0` | MIT-family* |
| `golang.org/x/crypto` | `v0.53.0` | BSD-family* |
| `golang.org/x/mod` | `v0.37.0` | BSD-family* |
| `golang.org/x/net` | `v0.56.0` | BSD-family* |
| `golang.org/x/sys` | `v0.46.0` | BSD-family* |
| `golang.org/x/text` | `v0.39.0` | BSD-family* |

## Local replacements

- `github.com/BryanSmee/bl_to_std` resolves to `../third_party/bl2std`. The pinned submodule is commit `4252f5f0a880463fd2e6aeed3125ec3653de60ee`, whose [LICENSE](https://github.com/BryanSmee/bl_to_std/blob/4252f5f0a880463fd2e6aeed3125ec3653de60ee/LICENSE) is **GNU GPL version 3**. Preserve its separate authorship, copyright notices and corresponding source requirements.
- `github.com/nocturney/3mf-bridge/engine` resolves to `../engine`. The project's own code is intended to be GPL-3.0-only where its contributors hold licensing rights. The legacy module path is tracked in [issue #10](https://github.com/nocturney/slicebridge/issues/10).

## Release-time blockers remaining

- Confirm upstream copyright and NOTICE files, any file- or directory-level licenses and requirements for bundling source/binaries.
- Re-run for **the exact release commit**, not this working branch; generate a machine-readable SBOM and complete dependency/source notice bundle for the binary actually shipped.
- Review browser-extension resources, Go compiler/toolchain, Windows/macOS WebView packaging and vendor preset policy. Go module archives do not prove the contents of final installers.
- Ensure GPL-covered combined works have their corresponding source and installation/build information available as required.
- Do not republish manufacturer profiles merely because they are installed on a tester's machine or publicly hosted elsewhere.

[General licensing](LICENSING.md) · [Third-party notices](THIRD_PARTY_NOTICES.md) · [Distribution checklist](DISTRIBUTION_CHECKLIST.md)
