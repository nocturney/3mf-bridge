package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Files embedded in the standalone desktop binary. No GitHub access, external
// download, or browser credentials are required to prepare an extension.
var extensionResourceNames = []string{
	"manifest.json", "content.js", "popup.html", "popup.css", "popup.js",
	"icon-16.png", "icon-32.png", "icon-48.png", "icon-128.png",
}

func browserExtensionKind(browser string) (string, error) {
	switch browser {
	case "edge", "chrome", "brave":
		return "chromium", nil
	case "firefox":
		return "firefox", nil
	default:
		return "", fmt.Errorf("unsupported browser %q", browser)
	}
}

func makerWorldSetupURL(mode string) (string, error) {
	switch mode {
	case "off", "ask", "always":
		// A content script running ONLY on makerworld.com asks for explicit
		// confirmation. It is the extension, not this app, that persists mode.
		return "https://makerworld.com/#slicebridge-setup=" + mode, nil
	default:
		return "", errors.New("unsupported MakerWorld mode")
	}
}

func extensionFile(kind, filename string) ([]byte, error) {
	if kind != "chromium" && kind != "firefox" {
		return nil, errors.New("invalid extension kind")
	}
	approved := false
	for _, item := range extensionResourceNames {
		if filename == item {
			approved = true
			break
		}
	}
	if !approved {
		return nil, errors.New("invalid extension resource")
	}
	return assets.ReadFile("frontend/dist/extension-packages/" + kind + "/" + filename)
}

// prepareExtensionFiles has no OS/browser side effects, suitable for isolated QA.
func prepareExtensionFiles(base, kind string) (string, error) {
	if kind != "chromium" && kind != "firefox" {
		return "", errors.New("invalid extension kind")
	}
	dest := filepath.Join(base, "SliceBridge", "BrowserExtensions", kind)
	if info, err := os.Lstat(dest); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("extension output folder cannot be a symlink")
	}
	if err := os.MkdirAll(dest, 0700); err != nil {
		return "", err
	}
	for _, name := range extensionResourceNames {
		data, err := extensionFile(kind, name)
		if err != nil {
			return "", fmt.Errorf("missing bundled extension resource %s: %w", name, err)
		}
		output := filepath.Join(dest, name)
		if err := os.WriteFile(output, data, 0600); err != nil {
			return "", fmt.Errorf("write %s: %w", name, err)
		}
	}
	return dest, nil
}

// PrepareBrowserExtension prepares a local unpacked extension and opens its
// directory. The browser still requires a user gesture to load it. Firefox
// development loading is temporary; signed store releases are a separate task.
func (a *App) PrepareBrowserExtension(browser string) (map[string]string, error) {
	kind, err := browserExtensionKind(strings.ToLower(strings.TrimSpace(browser)))
	if err != nil {
		return nil, err
	}
	userConfig, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	folder, err := prepareExtensionFiles(userConfig, kind)
	if err != nil {
		return nil, err
	}
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("explorer.exe", folder).Start()
	case "darwin":
		_ = exec.Command("open", folder).Start()
	}
	return map[string]string{"folder": folder, "kind": kind}, nil
}

// Select only known browser executables. The user-supplied browser code and
// mode are allowlisted; no executable paths or arbitrary URLs can be supplied.
func findWindowsBrowser(browser string) (string, error) {
	var relative []string
	switch browser {
	case "edge":
		relative = []string{"Microsoft/Edge/Application/msedge.exe"}
	case "chrome":
		relative = []string{"Google/Chrome/Application/chrome.exe"}
	case "brave":
		relative = []string{"BraveSoftware/Brave-Browser/Application/brave.exe"}
	case "firefox":
		relative = []string{"Mozilla Firefox/firefox.exe"}
	default:
		return "", errors.New("unsupported browser")
	}
	for _, parent := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LOCALAPPDATA")} {
		if parent == "" {
			continue
		}
		for _, part := range relative {
			candidate := filepath.Join(parent, filepath.FromSlash(part))
			if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("%s is not installed in a standard location", browser)
}

// The official store links are deliberately empty until the publisher has a
// live, verified extension listing. Never send nontechnical users to a store
// search page or promise one-click installation before approval.
var approvedExtensionStoreURLs = map[string]string{}

func browserExtensionsPage(browser string) (string, error) {
	switch browser {
	case "edge":
		return "edge://extensions", nil
	case "chrome":
		return "chrome://extensions", nil
	case "brave":
		return "brave://extensions", nil
	case "firefox":
		return "about:debugging#/runtime/this-firefox", nil
	default:
		return "", fmt.Errorf("unsupported browser %q", browser)
	}
}

func verifiedExtensionStoreURL(browser string) (string, error) {
	if _, err := browserExtensionKind(browser); err != nil {
		return "", err
	}
	storeBrowser := browser
	if browser == "brave" { storeBrowser = "chrome" }
	candidate := approvedExtensionStoreURLs[storeBrowser]
	if candidate == "" { return "", nil }
	parsed, err := url.Parse(candidate)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("invalid official extension store URL")
	}
	path := parsed.EscapedPath()
	switch storeBrowser {
	case "edge":
		if parsed.Host != "microsoftedge.microsoft.com" || !strings.HasPrefix(path, "/addons/detail/") {
			return "", errors.New("unapproved Edge store URL")
		}
	case "chrome":
		if parsed.Host != "chromewebstore.google.com" || !strings.HasPrefix(path, "/detail/") {
			return "", errors.New("unapproved Chrome store URL")
		}
	case "firefox":
		if parsed.Host != "addons.mozilla.org" || !strings.HasPrefix(path, "/en-US/firefox/addon/") {
			return "", errors.New("unapproved Firefox store URL")
		}
	default:
		return "", errors.New("unknown extension store")
	}
	return candidate, nil
}

// BrowserExtensionStoreURL exposes a read-only approved listing, if one exists.
func (a *App) BrowserExtensionStoreURL(browser string) (string, error) {
	return verifiedExtensionStoreURL(strings.ToLower(strings.TrimSpace(browser)))
}

// Only internal callers can open a browser URL, after validating an exact
// allowlisted settings page, official store listing, or MakerWorld mode URL.
// Never accept a URL string from the webview as a public Wails method.
func (a *App) launchTrustedBrowserURL(browser, targetURL string) error {
	if a.ctx == nil {
		return errors.New("desktop not ready")
	}
	if runtime.GOOS == "windows" {
		binary, err := findWindowsBrowser(browser)
		if err != nil { return err }
		proc := exec.Command(binary, targetURL)
		if err := proc.Start(); err != nil { return err }
		return proc.Process.Release()
	}
	if runtime.GOOS == "darwin" {
		apps := map[string]string{"edge": "Microsoft Edge", "chrome": "Google Chrome", "brave": "Brave Browser", "firefox": "Firefox"}
		return exec.Command("open", "-a", apps[browser], targetURL).Run()
	}
	wailsruntime.BrowserOpenURL(a.ctx, targetURL)
	return nil
}

// OpenBrowserExtensionsPage opens only the selected browser's own extensions
// manager, as an explicit user action. No extension is installed silently.
func (a *App) OpenBrowserExtensionsPage(browser string) error {
	browser = strings.ToLower(strings.TrimSpace(browser))
	page, err := browserExtensionsPage(browser)
	if err != nil { return err }
	return a.launchTrustedBrowserURL(browser, page)
}

// OpenBrowserExtensionStore is enabled only after an approved listing URL has
// been added to the pinned allowlist in a code-reviewed, versioned release.
func (a *App) OpenBrowserExtensionStore(browser string) error {
	browser = strings.ToLower(strings.TrimSpace(browser))
	page, err := verifiedExtensionStoreURL(browser)
	if err != nil { return err }
	if page == "" { return errors.New("browser extension is not published in an approved store yet") }
	return a.launchTrustedBrowserURL(browser, page)
}

// Clicking a mode opens the exact selected browser; only the extension
// confirmation (not this app) persists a MakerWorld preference.
func (a *App) OpenMakerWorldMode(browser, mode string) error {
	page, err := makerWorldSetupURL(mode)
	if err != nil { return err }
	browser = strings.ToLower(strings.TrimSpace(browser))
	if _, err = browserExtensionKind(browser); err != nil { return err }
	return a.launchTrustedBrowserURL(browser, page)
}
