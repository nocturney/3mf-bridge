# SliceBridge versioning and changelog

Small, predictable development milestones. No public release is implied.

## Source of truth

- `VERSION` is the intended **next app/development build version**. It follows [Semantic Versioning](https://semver.org/); during beta work, use `0.4.0-beta.4.dev.N`, incrementing `N` for each completed, separately reviewable development stage.
- `CHANGELOG.md` is the timeline. Keep `Unreleased` for pending changes and create or update a heading for the matching `VERSION` at each finished, pushed milestone. Include a link to the PR/commit and state if tests are pending.
- The existing `package.json` version `1.0.0` describes the older Electron prototype package, **not a publicly shipped SliceBridge desktop version**. Don't silently equate it with Wails builds.
- Wails currently has a hard-coded `appVersion` in its unmerged development branch. Until the runtime branch is integrated, `VERSION` on the documentation/main branch is a planned next build, **not the version embedded in an old binary**. Future Wails CI must stamp the exact source `VERSION` into `main.appVersion` using existing Go ldflags, then verify the binary readback alongside the source commit.
- `v0.4.0-beta.3` is an existing **draft, unpublished** GitHub Release; it is not the current downloadable community build.

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
