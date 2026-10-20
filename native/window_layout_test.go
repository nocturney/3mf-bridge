package main

import (
    "path/filepath"
    "testing"
)

func TestNewDesktopWindowStartsLargeEnoughForSourceQueue(t *testing.T) {
    got := defaultSettings().Window
    want := (WindowSettings{Width: 1280, Height: 900})
    if got != want {
        t.Fatalf("fresh install window = %+v, want %+v", got, want)
    }
}

func TestExistingCustomWindowSizeIsNeverOverwrittenByNewDefault(t *testing.T) {
    t.Setenv("BRIDGE_SETTINGS_FILE", filepath.Join(t.TempDir(), "settings.json"))
    settings := defaultSettings()
    settings.Window = WindowSettings{Width: 1100, Height: 750}
    settings.Printers["qidi"] = PrinterSettings{Nozzle: "0.4", Slicer: "auto", Filaments: map[string]string{}}
    if err := writeSettings(settings); err != nil {
        t.Fatal(err)
    }
    actual, err := loadSettings()
    if err != nil {
        t.Fatal(err)
    }
    if actual.Window != settings.Window {
        t.Fatalf("custom window changed: got %+v, want %+v", actual.Window, settings.Window)
    }
    if actual.Printers["qidi"].Nozzle != "0.4" {
        t.Fatalf("saved nozzle changed: %+v", actual.Printers["qidi"])
    }
}
