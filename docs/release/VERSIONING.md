# SliceBridge versioning and changelog

Small, predictable development milestones. A source VERSION bump does not publish a release.

## Source of truth

- `VERSION` is the **source app/development build version**, which may or may not yet be published. It follows [Semantic Versioning](https://semver.org/); during beta work, use `0.4.0-beta.4.dev.N`, incrementing `N` for each completed, separately reviewable development stage.
- `CHANGELOG.md` is the timeline. Keep `Unreleased` for pending changes and create or update a heading for the matching `VERSION` at each finished, pushed milestone. Include a link to the PR/commit and state if tests are pending.
- The existing `package.json` version `1.0.0` describes the older Electron prototype package, **not a publicly shipped SliceBridge desktop version**. Don't silently equate it with Wails builds.
- The Wails native app has been merged into main. Each development source milestone synchronizes root `VERSION` with native `appVersion` (checked by `tests/test_versioning.py`). This does not alter previous published binaries. New release artifacts must still embed and verify the exact source commit at packaging time.
- [v0.4.0-beta.4.dev.8](https://github.com/nocturney/slicebridge/releases/tag/v0.4.0-beta.4.dev.8) was published as an experimental pre-release on 2026-10-21 from source commit `87de87cfbcbc40813e11e9e1c7322f084fed7f8f`. This is not stable; dev.7 and beta.3 remain older experimental builds.

## One stage = a pushed commit + a changelog note

1. Develop one reviewable change on its own branch. Increment `VERSION` to the next `.dev.N` if the stage changes the intended desktop version.
2. Add a short `CHANGELOG.md` entry describing **actual** code and the test status. If the change lives in an unmerged PR, note the PR and its dependency; don't imply it's on `main`.
3. Run `python3 tools/check_versioning.py` (GitHub Actions also enforces it), relevant unit tests, and verify that the diff contains no generated user assets or secrets.
4. Commit **and push each stage** to GitHub, and link the commit or Draft PR. Pushed commits are visible for review, but publishing a CI artifact or pushing a branch is **not a GitHub Release**.
5. Do not merge self-authored code without independent QA. Versioned, approved releases use a `v<version>` tag pointing to the exact approved source commit and matching verified binary + release notes. Never force-update release tags.

### Version types

| Example | Meaning |
| --- | --- |
| `0.4.0-beta.4.dev.0` | Next development snapshot; **not released** |
| `0.4.0-beta.4.dev.1` | Next pushed, testable development stage; **not released** |
| `0.4.0-beta.4` | Candidate for beta release **only after QA/owner approval** |
| `v0.4.0-beta.4` | Immutable published release tag, created **only at publication** |

Avoid automatically making GitHub Releases for every commit; keep the repo simple and distinguish development progress from what users can safely install.
