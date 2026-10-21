# SliceBridge — a simple community release checklist

SliceBridge is a small personal, open-source project. **No corporate approval process, paid signing certificate or notarization is required just to share an experimental build.**

Before sharing a build:

- [ ] **Try that exact build** on the Windows/macOS platforms you name. Check that a sample project converts and opens in the target slicer. Clearly label printers or combinations you haven't tested. Always verify nozzle/material settings before printing.
- [ ] **Include the matching source code** and build information for GPL-covered components, the [GPL-3.0-only license](../../LICENSE), and required upstream notices/attribution. Don't package vendor profiles you have no right to redistribute.
- [ ] **Make it clear that it is experimental.** List supported OS/architectures, known limitations and anything a tester should check.
- [ ] Upload the files to a GitHub Release, ideally with SHA-256 checksums, and link the release from the README/site when it actually exists.
- [ ] Give people a simple way to [report problems](../../.github/SUPPORT.md).

**Unsigned builds are allowed.** Windows may display a SmartScreen or unknown-publisher warning; macOS Gatekeeper may block opening an unnotarized app by default. Say this clearly rather than claiming the apps are trusted or universally easy to install. Signing/notarization can be added later if it's worth the effort.

The existing `v0.4.0-beta.3` is an **old draft**, not a tested release for today's code. Whether to share a build is a practical decision after app testing, not a mandatory checklist of certificates and approvals.

Optional helpers: [source ZIP, preliminary SBOM and checksums](RELEASE_OPERATIONS.md). The [historical draft inspection](RELEASE_PREFLIGHT.md) is evidence about that old build, not a universal release-blocking rule.
