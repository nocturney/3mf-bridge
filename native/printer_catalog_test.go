package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestOfflinePickerHas603EntriesButOnlyFourAdapters(t *testing.T) {
	got, err := (&App{}).PrinterCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if got.ModelCount != 603 || len(got.Models) != 603 || got.LegacyAdapterCount != 4 ||
		got.ThirdPartyProfilePayloadsIncluded {
		t.Fatalf("catalogue metadata contract mismatch: %+v", got)
	}
	count, hasBambu := 0, false
	for _, model := range got.Models {
		if model.Manufacturer == "Bambu Lab" {
			hasBambu = true
			if model.LegacyTarget != nil {
				t.Fatal("Bambu must not be a conversion adapter")
			}
		}
		if model.LegacyTarget != nil {
			count++
			if legacyCatalogIDs[*model.LegacyTarget] != model.ID {
				t.Fatalf("legacy target mis-bound: %+v", model)
			}
		}
		if model.ImagePath != nil || model.LogoPath != nil {
			t.Fatal("unreviewed third-party image included")
		}
	}
	if !hasBambu || count != 4 {
		t.Fatalf("Bambu must remain indexed, but only four converters exist: %v %d", hasBambu, count)
	}
}

func TestMyPrintersOnboardingDefaultsAndAtomicSave(t *testing.T) {
	withFreshSettings(t)
	app := &App{}
	fresh, err := app.GetMyPrinters()
	if err != nil {
		t.Fatal(err)
	}
	if !fresh.OnboardingPending || fresh.Version != 1 ||
		!reflect.DeepEqual(fresh.SelectedIDs, defaultMyPrinterIDs) {
		t.Fatalf("old four printers not preserved on first boot: %+v", fresh)
	}
	path, err := myPrintersPath()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("first-run read wrote to disk unexpectedly: %v", err)
	}
	picked := []string{"elegoo-centauri-carbon-2", "qidi-x-plus-5", "snapmaker-u1"}
	saved, err := app.SaveMyPrinters(picked)
	if err != nil {
		t.Fatal(err)
	}
	if saved.OnboardingPending || !reflect.DeepEqual(saved.SelectedIDs, picked) {
		t.Fatalf("incorrect persisted selected models: %+v", saved)
	}
	again, err := app.GetMyPrinters()
	if err != nil || again.OnboardingPending || !reflect.DeepEqual(again.SelectedIDs, picked) {
		t.Fatalf("selected printers were not reloaded: %+v %v", again, err)
	}
	empty, err := app.SaveMyPrinters([]string{})
	if err != nil || empty.SelectedIDs == nil || len(empty.SelectedIDs) != 0 {
		t.Fatalf("empty printer selection must be valid: %+v %v", empty, err)
	}
	reloaded, err := app.GetMyPrinters()
	if err != nil || len(reloaded.SelectedIDs) != 0 || reloaded.OnboardingPending {
		t.Fatalf("empty selection reverted to four printers: %+v %v", reloaded, err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"selectedIds": []`)) {
		t.Fatalf("empty selection must be stored explicitly as array: %s", raw)
	}
}

func TestMyPrintersDoNotChangeLegacyHardwareAndFilamentSettings(t *testing.T) {
	withFreshSettings(t)
	original := defaultSettings()
	pref := original.Printers["cc2"]
	pref.Nozzle = "0.6"
	pref.Filaments["PETG"] = "My PETG profile"
	original.Printers["cc2"] = pref
	original.Window = WindowSettings{Width: 1400, Height: 790}
	if err := writeSettings(original); err != nil {
		t.Fatal(err)
	}
	settingsFile, err := settingsPath()
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatal(err)
	}
	app := &App{}
	if _, err = app.SaveMyPrinters([]string{"flashforge-creator-5"}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("printer picker modified existing nozzle, slicer or filament preferences")
	}
	loaded, err := app.GetSettings()
	if err != nil || loaded.Printers["cc2"].Nozzle != "0.6" ||
		loaded.Printers["cc2"].Filaments["PETG"] != "My PETG profile" ||
		loaded.Window.Width != 1400 {
		t.Fatalf("legacy printer preferences lost: %+v %v", loaded, err)
	}
}

func TestMyPrinterPickerFailClosedOnUnknownHiddenOrRepeatedIDs(t *testing.T) {
	withFreshSettings(t)
	app := &App{}
	catalog, err := app.PrinterCatalog()
	if err != nil {
		t.Fatal(err)
	}
	var hidden string
	for _, m := range catalog.Models {
		if m.Manufacturer == "Bambu Lab" {
			hidden = m.ID
			break
		}
	}
	if hidden == "" {
		t.Fatal("expected indexed Bambu profile")
	}
	more := make([]string, 0, 33)
	for _, m := range catalog.Models {
		if m.Manufacturer != "Bambu Lab" {
			more = append(more, m.ID)
			if len(more) == 33 {
				break
			}
		}
	}
	for _, ids := range [][]string{
		{"unknown-printer"},
		{hidden},
		{"snapmaker-u1", "snapmaker-u1"},
		more,
	} {
		if _, err = app.SaveMyPrinters(ids); err == nil {
			t.Fatalf("invalid/hidden/too many targets saved: %#v", ids)
		}
	}
	result, err := app.GetMyPrinters()
	if err != nil || !result.OnboardingPending {
		t.Fatalf("failed write changed onboarding state: %+v %v", result, err)
	}
}

func TestCorruptMyPrinterSelectionNotOverwritten(t *testing.T) {
	withFreshSettings(t)
	path, err := myPrintersPath()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	invalid := []byte(`{"version":900,"selectedIds":["qidi-x-plus-5"]}`)
	if err = os.WriteFile(path, invalid, 0600); err != nil {
		t.Fatal(err)
	}
	app := &App{}
	if _, err = app.GetMyPrinters(); err == nil {
		t.Fatal("corrupt/unknown version silently accepted")
	}
	if _, err = app.SaveMyPrinters([]string{"qidi-x-plus-5"}); err == nil ||
		!strings.Contains(err.Error(), "version") {
		t.Fatalf("corrupt selection overwritten instead of failing closed: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(after, invalid) {
		t.Fatalf("corrupt original state not preserved: %v", err)
	}
	if err = os.WriteFile(path, []byte(`{"version":1,"selectedIds":["bambu-lab-p1s"]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = app.GetMyPrinters(); err == nil {
		t.Fatal("Bambu selection incorrectly granted as adapter")
	}
}

func TestMyPrinterSavedPayloadIsSmallAndHasNoProfileBytes(t *testing.T) {
	withFreshSettings(t)
	app := &App{}
	if _, err := app.SaveMyPrinters([]string{"snapmaker-u1"}); err != nil {
		t.Fatal(err)
	}
	path, _ := myPrintersPath()
	blob, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]any
	if err = json.Unmarshal(blob, &data); err != nil {
		t.Fatal(err)
	}
	if len(data) != 2 {
		t.Fatalf("unexpected fields written to local picker: %#v", data)
	}
	if bytes.Contains(blob, []byte("machine_settings")) || bytes.Contains(blob, []byte("filament_presets")) {
		t.Fatal("upstream profiles must not be copied to My Printers state")
	}
}
