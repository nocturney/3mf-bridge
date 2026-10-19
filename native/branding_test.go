package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Upgrading must not erase printer selections or the manually sized window.
func TestSliceBridgeMigratesLegacyPrinterPreferences(t *testing.T) {
	root := t.TempDir()
	t.Setenv("BRIDGE_SETTINGS_FILE", "")
	switch runtime.GOOS {
	case "windows":
		t.Setenv("APPDATA", root)
	case "darwin":
		t.Setenv("HOME", root)
	default:
		t.Setenv("XDG_CONFIG_HOME", root)
	}
	pref := defaultSettings()
	pref.Printers["cc2"] = PrinterSettings{Nozzle: "0.6", Slicer: "auto", Filaments: map[string]string{}}
	pref.Window = WindowSettings{Width: 1370, Height: 820}
	base, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	oldPath := filepath.Join(base, "Velvet Factory", "3MF Bridge", "settings.json")
	newPath, err := settingsPath()
	if err != nil {
		t.Fatal(err)
	}
	if oldPath == newPath {
		t.Fatal("branding must use distinct settings path")
	}
	if err := os.MkdirAll(filepath.Dir(oldPath), 0700); err != nil {
		t.Fatal(err)
	}
	original, err := json.Marshal(pref)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldPath, original, 0600); err != nil {
		t.Fatal(err)
	}
	actual, err := loadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if actual.Window != pref.Window || actual.Printers["cc2"].Nozzle != "0.6" {
		t.Fatalf("legacy settings lost: %+v", actual)
	}
	if err := writeSettings(actual); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatal(err)
	}
	stillThere, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(original) != string(stillThere) {
		t.Fatal("legacy settings were altered")
	}
	// New-brand settings take precedence after migration.
	actual.Window = WindowSettings{Width: 1024, Height: 700}
	if err := writeSettings(actual); err != nil {
		t.Fatal(err)
	}
	migrated, err := loadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if migrated.Window != actual.Window {
		t.Fatalf("new settings not preferred: %+v", migrated.Window)
	}
}
