package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBrowserSetupValidatesBrowserAndMode(t *testing.T) {
	for _, browser := range []string{"edge", "chrome", "brave", "firefox"} {
		if _, err := browserExtensionKind(browser); err != nil {
			t.Fatalf("rejected %q: %v", browser, err)
		}
	}
	for _, browser := range []string{"", "safari", "../firefox", "edge;rm", "unknown"} {
		if _, err := browserExtensionKind(browser); err == nil {
			t.Errorf("accepted unsupported browser %q", browser)
		}
	}
	for _, mode := range []string{"off", "ask", "always"} {
		url, err := makerWorldSetupURL(mode)
		if err != nil || url != "https://makerworld.com/#slicebridge-setup="+mode {
			t.Errorf("invalid setup URL for %q: %q %v", mode, url, err)
		}
	}
	for _, mode := range []string{"", "on", "ask?url=file://", "always#x", "https://evil.test"} {
		if _, err := makerWorldSetupURL(mode); err == nil {
			t.Errorf("accepted untrusted setup mode %q", mode)
		}
	}
}

func TestBundledExtensionsExtractToLocalFolder(t *testing.T) {
	for _, kind := range []string{"chromium", "firefox"} {
		destination, err := prepareExtensionFiles(t.TempDir(), kind)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(destination, "SliceBridge") {
			t.Fatal(destination)
		}
		for _, resource := range extensionResourceNames {
			want, err := extensionFile(kind, resource)
			if err != nil || len(want) == 0 {
				t.Fatalf("%s/%s not embedded: %v", kind, resource, err)
			}
			got, err := os.ReadFile(filepath.Join(destination, resource))
			if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("extracted data does not match embedded asset %s/%s: %v", kind, resource, err)
			}
		}
		if _, err = prepareExtensionFiles(t.TempDir(), "../bad"); err == nil {
			t.Error("invalid extension source accepted")
		}
	}
	if _, err := extensionFile("chromium", "../index.html"); err == nil {
		t.Error("path traversal accepted")
	}
}

func TestPrepareExtensionRejectsSymlinkDestination(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, "SliceBridge", "BrowserExtensions")
	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(path, "chromium")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := prepareExtensionFiles(home, "chromium"); err == nil {
		t.Error("symlink output accepted")
	}
}
