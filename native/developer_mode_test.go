package main

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func developerModeFixture(t *testing.T) string {
	t.Helper()
	withFreshSettings(t)
	path, err := settingsPath()
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDeveloperModeDefaultsDisabled(t *testing.T) {
	developerModeFixture(t)
	app := &App{}
	got, err := app.GetDeveloperMode()
	if err != nil || got {
		t.Fatalf("fresh install developer mode = %v, %v; want disabled", got, err)
	}
	if defaultSettings().DeveloperMode {
		t.Fatal("defaultSettings enables developer mode")
	}
	if err := app.requireDeveloperMode(); !errors.Is(err, errDeveloperModeOff) {
		t.Fatalf("gate while disabled = %v, want errDeveloperModeOff", err)
	}
}

func TestDeveloperModeEnableAndDisable(t *testing.T) {
	developerModeFixture(t)
	app := &App{}
	if got, err := app.SetDeveloperMode(true); err != nil || !got {
		t.Fatalf("enable = %v, %v", got, err)
	}
	if err := app.requireDeveloperMode(); err != nil {
		t.Fatalf("gate while enabled = %v", err)
	}
	if got, err := app.SetDeveloperMode(false); err != nil || got {
		t.Fatalf("disable = %v, %v", got, err)
	}
	if got, err := app.GetDeveloperMode(); err != nil || got {
		t.Fatalf("after disable = %v, %v", got, err)
	}
	if err := app.requireDeveloperMode(); !errors.Is(err, errDeveloperModeOff) {
		t.Fatalf("gate after disable = %v, want errDeveloperModeOff", err)
	}
}

func TestDeveloperModePersistsAcrossRestarts(t *testing.T) {
	path := developerModeFixture(t)
	first := &App{}
	if _, err := first.SetDeveloperMode(true); err != nil {
		t.Fatal(err)
	}
	// A restarted process has no in-memory state; it must read the file.
	restarted := &App{}
	if got, err := restarted.GetDeveloperMode(); err != nil || !got {
		t.Fatalf("after restart = %v, %v; want enabled", got, err)
	}
	if err := checkDeveloperMode(); err != nil {
		t.Fatalf("bridge gate after restart = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"developerMode": true`) {
		t.Fatalf("persisted file lacks developerMode:true: %s", data)
	}
}

func TestSettingsWithoutDeveloperModeFieldStayDisabled(t *testing.T) {
	path := developerModeFixture(t)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	legacy := `{"version":1,"printers":{"qidi":{"nozzle":"0.4","slicer":"auto","filaments":{}}},"window":{"width":1100,"height":750}}`
	if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	app := &App{}
	got, err := app.GetDeveloperMode()
	if err != nil || got {
		t.Fatalf("file without field = %v, %v; want disabled", got, err)
	}
	loaded, err := app.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Printers["qidi"].Nozzle != "0.4" || loaded.Window != (WindowSettings{Width: 1100, Height: 750}) {
		t.Fatalf("existing preferences changed on read: %+v", loaded)
	}
}

func TestDeveloperModeLeavesUnrelatedPreferencesUnchanged(t *testing.T) {
	developerModeFixture(t)
	saved := defaultSettings()
	saved.Printers["qidi"] = PrinterSettings{Nozzle: "0.4", Slicer: "auto", Filaments: map[string]string{}}
	saved.Window = WindowSettings{Width: 1100, Height: 750}
	if err := writeSettings(saved); err != nil {
		t.Fatal(err)
	}
	app := &App{}
	before, err := app.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.SetDeveloperMode(true); err != nil {
		t.Fatal(err)
	}
	after, err := app.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if !after.DeveloperMode {
		t.Fatal("developer mode did not persist")
	}
	if !reflect.DeepEqual(after.Printers, before.Printers) || after.Window != before.Window {
		t.Fatalf("unrelated preferences changed:\nbefore %+v\nafter  %+v", before, after)
	}
	if _, err := app.SetDeveloperMode(false); err != nil {
		t.Fatal(err)
	}
	restored, err := app.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored, before) {
		t.Fatalf("disable did not restore prior settings:\nbefore %+v\nafter  %+v", before, restored)
	}
}

func TestCorruptSettingsFailDeveloperModeClosed(t *testing.T) {
	path := developerModeFixture(t)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"not json":   "{ definitely not JSON }",
		"wrong type": `{"version":1,"printers":{},"window":{"width":1280,"height":900},"developerMode":"true"}`,
	}
	bridgeURI := "slicebridge://import?url=" + url.QueryEscape("https://makerworld.com/api/model.3mf?signed=secret")
	for name, body := range cases {
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		app := &App{}
		if got, err := app.GetDeveloperMode(); err == nil || got {
			t.Errorf("%s: get = %v, %v; want error and disabled", name, got, err)
		}
		if got, err := app.SetDeveloperMode(true); err == nil || got {
			t.Errorf("%s: set = %v, %v; want error and no change", name, got, err)
		}
		if err := app.requireDeveloperMode(); err == nil {
			t.Errorf("%s: gate passed on corrupt settings", name)
		}
		bridge := &BrowserBridge{active: map[string]bool{}}
		bridge.acceptArg(bridgeURI)
		if len(bridge.waiting) != 0 {
			t.Errorf("%s: MakerWorld URI accepted on corrupt settings", name)
		}
		data, err := os.ReadFile(path)
		if err != nil || string(data) != body {
			t.Errorf("%s: corrupt settings were overwritten: %q, %v", name, data, err)
		}
	}
}

func TestDisabledDeveloperModeBlocksBrowserActionsBeforeContext(t *testing.T) {
	developerModeFixture(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	// ctx is nil on purpose: the gate must answer before the desktop-ready check.
	app := &App{}
	calls := []struct {
		name string
		call func() error
	}{
		{"PrepareBrowserExtension", func() error { _, err := app.PrepareBrowserExtension("chrome"); return err }},
		{"OpenBrowserSetup", app.OpenBrowserSetup},
		{"OpenBrowserExtensionsPage", func() error { return app.OpenBrowserExtensionsPage("edge") }},
		{"OpenBrowserExtensionStore", func() error { return app.OpenBrowserExtensionStore("chrome") }},
		{"OpenMakerWorldMode", func() error { return app.OpenMakerWorldMode("edge", "ask") }},
		{"EnableBrowserLink", func() error { _, err := app.EnableBrowserLink(); return err }},
	}
	for _, c := range calls {
		if err := c.call(); !errors.Is(err, errDeveloperModeOff) {
			t.Errorf("%s while disabled = %v, want errDeveloperModeOff", c.name, err)
		}
	}
	userConfig, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(userConfig, "SliceBridge")); !os.IsNotExist(err) {
		t.Fatalf("disabled developer mode still wrote extension files: %v", err)
	}
}

func TestMakerWorldURIAcceptedOnlyWithDeveloperMode(t *testing.T) {
	developerModeFixture(t)
	app := &App{}
	uri := "slicebridge://import?url=" + url.QueryEscape("https://makerworld.com/api/model.3mf?signed=secret")
	// No context is set, so an accepted URI is queued and nothing is downloaded.
	bridge := &BrowserBridge{active: map[string]bool{}}
	bridge.acceptArg(uri)
	if len(bridge.waiting) != 0 {
		t.Fatal("URI accepted while developer mode is disabled")
	}
	if _, err := app.SetDeveloperMode(true); err != nil {
		t.Fatal(err)
	}
	bridge.acceptArg(uri)
	if len(bridge.waiting) != 1 {
		t.Fatalf("URI not accepted while developer mode is enabled: %d queued", len(bridge.waiting))
	}
}

func TestDeveloperModeConcurrentToggleAndRead(t *testing.T) {
	developerModeFixture(t)
	app := &App{}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func(enable bool) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if _, err := app.SetDeveloperMode(enable); err != nil {
					t.Errorf("set: %v", err)
					return
				}
			}
		}(i%2 == 0)
		go func() {
			defer wg.Done()
			for j := 0; j < 40; j++ {
				if _, err := app.GetDeveloperMode(); err != nil {
					t.Errorf("get: %v", err)
					return
				}
				// The lock-free bridge gate may see a missing file before the first
				// write, but never a partial one.
				if err := checkDeveloperMode(); err != nil && !errors.Is(err, errDeveloperModeOff) {
					t.Errorf("gate saw partial settings: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
	if _, err := app.GetDeveloperMode(); err != nil {
		t.Fatalf("settings unreadable after concurrent writes: %v", err)
	}
}
