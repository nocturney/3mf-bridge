# SliceBridge — public distribution checklist

**Status: HOLD — no general release authorized.** This checklist is a release gate, not a claim that each item is complete. Coordinate implementation-dependent work with the separate app-development workflow. Do not publish the draft Beta 3 solely because artifacts exist.

## 1. Rights and attribution

- [x] The owner selected **open-source distribution**. A consistent **GPL-3.0-only** code license and matching prototype npm metadata are proposed in the licensing PR, matching the pre-existing development branch license. **This does not validate ownership of every upstream asset.**
- [ ] Confirm all SliceBridge-original copyright owners can license their contributions under GPL-3.0-only and retain original copyright notices.
- [ ] Ensure code and metadata from all feature branches are reconciled to the same license during reviewed merges; do not misrepresent unmerged feature code as released.
- [ ] Confirm compliance for the `BryanSmee/bl_to_std` GPL-3.0 component and any derivative/distributed binaries, including corresponding source and notices where required.
- [ ] Audit all bundled third-party source, art, binaries, icons and scripts for licenses and required notices. Review development branch `THIRD_PARTY_NOTICES.md` and correct inherited `3MF Bridge` wording.
- [ ] Ensure **vendor printer/material presets are not bundled or republished** without verified rights. Availability on a third-party public Git repository is not, by itself, permission to redistribute.
- [ ] Confirm permission to use the approved SliceBridge artwork and the Velvet Factory name.

## 2. Technical release gate (owned by app-development workflow)

- [ ] Merge only reviewed changes after independent QA and green CI on the precise commit being released.
- [ ] Resolve old `nocturney/3mf-bridge` references and Go module/update-check paths on development branches; verify after migration.
- [ ] Run clean Windows and macOS builds and automated tests at the release commit.
- [ ] Verify actual slicer import, intended supported printers, matching nozzle/material profiles and representative physical prints. Disclose untested combinations explicitly.
- [ ] Test MakerWorld and browser-extension opt-in behavior without silently hijacking file/protocol associations.
- [ ] Check security of untrusted 3MF inputs, external links, update mechanism, temporary output and permissions.

## 3. Packaging and publishing

- [ ] Sign Windows executable/installer as appropriate; sign and notarize macOS distribution, with Apple Silicon/Intel scope explained accurately.
- [ ] Produce versioned archives with clear platform/architecture labels and SHA-256 checksums.
- [ ] Review installer/download instructions, README, [privacy notice](PRIVACY.md), [security reporting](SECURITY.md) and third-party notices.
- [ ] Attach verified assets to a **published** GitHub Release with accurate release notes and limitations; do not treat temporary CI artifacts or drafts as general downloads.
- [ ] Update the site and changelog only after the release has passed approval.
- [ ] Check fresh-machine installation and upgrade paths, then monitor issues.

**Release owner and sign-off:** to be named before publication. No auto-publishing is configured through this document.
