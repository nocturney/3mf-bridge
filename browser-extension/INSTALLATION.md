# Installing SliceBridge for MakerWorld

**Current status:** Browser extension builds are beta-only and **not yet approved in an extension store**. Do not claim there is a one-click store install or publish invented store links.

## For a normal user

In the SliceBridge desktop app, open **MakerWorld Setup**, choose your browser (Edge, Chrome, Brave or Firefox), and use the installation button. Until store approval, it runs a **guided beta installation**: it prepares the bundled local extension folder and opens the selected browser's extension-management screen. The browser still requires an explicit manual approval. Firefox beta extensions are temporary and must be reloaded after restarting Firefox.

Once the extension is officially listed, the *same button* becomes **Install from browser store**. No Developer mode, ZIP download or manually loaded folder will be part of the normal store flow. The user confirms installation directly in the browser.

## For the publisher (one-time submission)

- **Edge first:** package the contents (not the enclosing directory) of `dist/extensions/chromium` as a ZIP and submit it through [Microsoft Edge Partner Center](https://learn.microsoft.com/en-us/microsoft-edge/extensions/publish/publish-extension). The public or hidden listing only becomes installable after approval.
- **Chrome / Brave:** submit the same Chromium extension to [Chrome Web Store](https://developer.chrome.com/docs/webstore/register/). Developer registration requires a one-time registration fee. Chrome Web Store installs can also be offered to Brave users, subject to Brave's approval flow.
- **Firefox:** package the contents of `dist/extensions/firefox` and submit them for [Mozilla signing / AMO distribution](https://extensionworkshop.com/documentation/publish/). Unsigned Firefox extensions cannot provide a durable normal-user install.

To prepare publisher-only store submission ZIPs, run `node browser-extension/build.cjs` and `python3 browser-extension/pack_store.py`. The ZIPs in `dist/store-submissions/` contain the extension files at their root; they are **not signed or installable consumer store downloads**. CI also keeps these as a publisher-only artifact. Version each add-on upload separately from the SliceBridge desktop app. Check store policies, privacy disclosures, permissions, screenshots and contact details.

**After** a store approves a live extension listing, pin that exact official URL in `approvedExtensionStoreURLs` in `native/browser_setup.go`, add a Go URL allowlist regression, run CI, and release a new app build. Until then, the URL map stays empty and the app never displays a fictitious store install button.

No add-on may be installed silently or change Bambu Studio/system defaults without user consent.
