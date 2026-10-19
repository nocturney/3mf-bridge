# SliceBridge — release notes template (copy into a future GitHub Release)

> **Template only. No release is authorized.** Replace bracketed placeholders with independently verified information.

## [Version] — [platform(s)] — [date]

**Stage:** [alpha / limited beta / public beta / stable]  
**Exact source commit:** `[SHA]`  
**Release approval:** [named authorized maintainer and independent QA evidence]

### What changed
[Short user-facing description]

### Compatibility actually tested

| Target printer | Target slicer/version | Nozzle/material | Converted | Opened | Sliced | Physically printed |
| --- | --- | --- | --- | --- | --- | --- |
| [model] | [name/version] | [variant] | [PASS/FAIL/not tested] | [PASS/FAIL/not tested] | [PASS/FAIL/not tested] | [PASS/FAIL/not tested] |

Do **not** replace `not tested` with `PASS` or infer support solely from a profile catalogue.

### Downloads (publish only approved assets)

| OS / CPU | File | SHA-256 | Signature / notarization |
| --- | --- | --- | --- |
| Windows x64 | [file] | [hash] | [verified evidence] |
| macOS Apple Silicon | [file] | [hash] | [verified evidence] |

### Safety and privacy
[Validation of intended printer, nozzle, filament and slicing; privacy review outcomes]

### Source, license and notices
**GPL-3.0-only** for covered original source: [repository LICENSE](https://github.com/nocturney/slicebridge/blob/main/LICENSE). Include corresponding source for the **exact release build**, submodule revisions, [third-party notices](THIRD_PARTY_NOTICES.md), and a release-specific dependency/asset notice bundle.

### Known limitations / breaking changes
[List remaining constraints without speculation]

### Links
[Full changelog](CHANGELOG.md) · [Support](SUPPORT.md) · [Security](SECURITY.md) · [Release checklist](DISTRIBUTION_CHECKLIST.md)
