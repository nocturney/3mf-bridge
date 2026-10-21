# SliceBridge — optional release helper

The script [`tools/release_preflight.py`](../../tools/release_preflight.py) is an **optional convenience**, not a release-approval system. It can assemble a ZIP of Git-tracked source (including initialized pinned submodules), a preliminary dependency list in CycloneDX format and SHA-256 checksums.

## In GitHub Actions

Run [Release preflight (evidence only)](https://github.com/nocturney/slicebridge/actions/workflows/release-preflight.yml) on `main`. The run packages that commit's source and uploads a short-lived **evidence artifact**, not a public application release. It has read-only permissions and does not sign or publish anything.

The workflow **cannot read unpublished draft Release files** using its standard token. Inspect those separately on an authorized local computer if desired.

## Locally

From a clean checkout of the specific source commit:

```bash
git submodule update --init --recursive
python3 -m unittest discover -s tests -p 'test_release_preflight.py' -v
python3 tools/release_preflight.py --repo . --source-sha "$(git rev-parse HEAD)" --out /tmp/slicebridge-evidence
```

An empty output folder outside the repository is required. To inspect ZIPs that you already downloaded from a private draft, add `--assets-dir /path/to/draft-zips` locally. The tool **reads** them for hashes and ZIP integrity, but does not republish or execute the apps. If a ZIP's embedded build commit differs from the chosen source commit, the report flags the difference.

### What it produces

- `slicebridge-source-<SHA12>.zip` — source snapshot at that Git commit.
- `manifest-sbom.cdx.json` — **preliminary** dependency inventory based on manifests and any available Go binary metadata (not a complete shipped-binary SBOM).
- `release-evidence.json`, `SHA256SUMS.txt`; with local draft ZIPs, also `DRAFT_ASSET_SHA256SUMS.txt`.

This isn't a certificate of correct printing, license compliance or platform trust. **Digital signing/notarization are optional for a hobby distribution**, although unsigned apps may trigger or be blocked by OS security warnings. For the actual simple steps to share a build, see [the community release checklist](DISTRIBUTION_CHECKLIST.md).
