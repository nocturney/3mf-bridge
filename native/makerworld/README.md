# MakerWorld integration — safe technical groundwork

**Status: experimental, not auto-enabled.** Do not claim browser click
interception, URL retrieval, or background auto-open are working yet.

## Stage 1: implemented, no OS/browser modifications

- `makerworld.Detect(path)` uses the Windows `:Zone.Identifier`
  HostUrl / ReferrerUrl evidence and macOS Spotlight Where From metadata
  (`kMDItemWhereFroms`). It NEVER reads the user's web cookies.
- Only a confirmed `makerworld.com` hostname in origin/referrer
  earns the **MW** badge in the native source queue. `bblmw.com`
  CDN alone has *medium* confidence, not a MakerWorld confirmation.
- Absent provenance does NOT mean a 3MF is unrelated to MakerWorld.
  Files may be copied between volumes, downloaded by another tool, or
  have metadata intentionally cleared.
- `makerworld.ParseLink(raw)` strictly parses:
  - Windows `bambustudio://open?file=<percent-encoded HTTPS URL>`
  - macOS `bambustudioopen://<percent-encoded HTTPS URL>`
  - Future opt-in `slicebridge://import?url=<percent-encoded HTTPS URL>`
- URL hostname matching is exact or well-formed subdomain based:
  `makerworld.com` and `bblmw.com`. Local IP addresses, HTTP,
  malformed schemes and spoofed domains are rejected.
- Signed query strings are NOT written to UI status, logs, metadata
  or analytics. The native bridge preview function never fetches URLs.
- These capabilities work regardless of Bambu Studio installation.
- Integration entrypoints: Go `App.DetectMakerWorldOrigins(paths)` and
  `App.PreviewMakerWorldLink(raw)`.

## Proposed next stage: browser action and automatic reception

**Recommended:** a Chromium Chrome/Edge extension (scope:
`https://makerworld.com/*` and localized pages) that offers an
explicit "Open in SliceBridge" action. The extension must use an
opt-in `slicebridge://` URL scheme or native messaging handoff.

This avoids taking ownership of Bambu Studio's `bambustudio://`
protocol system-wide. A separate user-approved "Replace Bambu button"
mode could register both native Bambu schemes, but that would replace
the handler for ALL such links on the OS, not only MakerWorld pages,
so it must be reversible and off by default. Never silently seize
Bambu Studio's protocol registration.

**File auto-open:** an opt-in watcher of the user's selected download
folder. Trigger only on fully downloaded .3mf files with high-confidence
MakerWorld provenance; dedupe by file identity and wait for writes to
finish. Do not continuously scan private unrelated directories.

**Important blocker:** MakerWorld generates short-lived download links
and sometimes demands session verification (CAPTCHA/HTTP 418). A native
client cannot assume arbitrary signed CDN URLs are downloadable without
browser authentication. Do not bypass rate limits, site authentication,
or verification. If the download is inaccessible outside the browser,
fall back to the browser's normal user-authorized download flow.

## Quality gates

Run `go test ./makerworld` on both Windows and macOS, including URL
rejection, Win ADS parsing and macOS metadata output parsing.
UI layout smoke uses mocked high-confidence provenance to verify MW
badge rendering and preserved drag/drop, 4-printer queue behavior.

**Pending:** actual in-browser MakerWorld button captured as an example,
browser type selected, authorized opt-in, and real MakerWorld downloaded
3MF on Windows/Mac to validate provenance in that browser.
