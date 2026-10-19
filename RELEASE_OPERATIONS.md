# SliceBridge release operations — evidence only

This process prepares verifiable release evidence WITHOUT publishing software. It reads an exact Git commit, packages its tracked source and pinned submodules, inventories dependencies from committed npm/Go manifests and optionally reads Go metadata from EXISTING draft binaries without executing them.

**Release remains on HOLD.** A source ZIP, SBOM, checksums and green CI do not replace copyright/third-party rights review, actual version-matched GPL corresponding-source obligations, app security and printer QA, qualified signing, Apple notarization or explicit owner approval. See [DISTRIBUTION_CHECKLIST.md](DISTRIBUTION_CHECKLIST.md).

## Run in GitHub Actions

Open [Actions → Release preflight (evidence only)](https://github.com/nocturney/slicebridge/actions/workflows/release-preflight.yml), choose Run workflow, and supply an immutable 40-character hexadecimal source SHA, NOT a branch or tag name.

Optionally supply the tag of an EXISTING draft release (for example v0.4.0-beta.3). This causes only READ-ONLY ZIP integrity, hash and embedded Go build-info checks. For the historical Beta 3 draft, the recorded build source commit is 5b9cd0a1f961a872f00d4d7e82ced567efd170ee. Only select that SHA when inspecting those specific assets.

The workflow uses contents: read. Draft binaries are fetched into an ephemeral CI directory and explicitly EXCLUDED from uploaded evidence artifacts. The job never creates GitHub Releases, pushes commits, changes branch protection, publishes binaries, or signs anything.

## Outputs of a manual run

| File | Purpose |
| --- | --- |
| slicebridge-source-SHA12.zip | Deterministic ZIP from exact Git-tracked source objects; includes initialized recursive submodules at their pinned revisions |
| manifest-sbom.cdx.json | CycloneDX 1.5 **preliminary** manifest SBOM from package-lock and go.mod, plus available embedded Go build metadata |
| release-evidence.json | Exact source commit, tracked source trees, dependency counts, archive integrity, explicit remaining release blockers |
| SHA256SUMS.txt | SHA-256 hashes of each file in the evidence artifact |
| DRAFT_ASSET_SHA256SUMS.txt | Only with draft downloads; records hashes of existing draft files, which are NOT republished |

The source ZIP sorts Git-tracked paths and uses fixed timestamps for byte-identical reproducibility at the same pinned commit. The script fails on wrong or dirty source checkouts, missing or mismatched submodules, duplicate/malicious archive names and bad ZIP CRC.

**Limitations:** The dependency inventory cannot attest to the complete final shipped-binary SBOM; the archive is not automatically a legally sufficient GPL corresponding-source offer without source, build/installation instructions and notices. It does not check Windows/macOS code-signing trust or run physical printer/slicer tests. Embedded Go source-commit mismatches are flagged; they must be resolved before publishing.

## Local usage

Run these commands in a clean checkout of the *exact desired source commit*:

    git submodule update --init --recursive
    python3 -m unittest discover -s tests -p 'test_release_preflight.py' -v
    python3 tools/release_preflight.py \
      --repo . \
      --source-sha "$(git rev-parse HEAD)" \
      --out /tmp/slicebridge-evidence

Optionally append --assets-dir /path/to/draft-zips. The output directory must be EMPTY and outside the repository.

## What must still be done for a real release

1. Choose a new, QA-approved release commit; record actual printer, nozzle, material and supported-slicer results separately from profile catalogue presence.
2. Produce the actual **per-platform** binary SBOM and license bundle, audit every packaged dependency and artwork asset, include applicable GPL corresponding source and copyright notices.
3. Sign Windows and macOS builds with authorized publisher credentials, notarize and staple the macOS app, validate on clean systems and record exact file checksums AFTER signing.
4. Complete privacy/security checks and confirm rights in the approved branding and all bundled third-party materials. Never commit signing credentials.
5. Obtain independent QA and explicit release-owner sign-off, then (in a separate controlled operation) publish release notes, assets and site download links.
6. Set the GitHub repository Social Preview in Settings using the already-prepared image; the website Open Graph image is already configured.

See [RELEASE_PREFLIGHT.md](RELEASE_PREFLIGHT.md), [GO_LICENSE_AUDIT.md](GO_LICENSE_AUDIT.md) and issues [#9](https://github.com/nocturney/slicebridge/issues/9) / [#11](https://github.com/nocturney/slicebridge/issues/11).
