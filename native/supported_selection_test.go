package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/nocturney/slicebridge/native/presets"
)

const readyOrca = "orca-acme-mini-0-4-nozzle-0123456789"

// stubInstalledOrca makes the listed Orca IDs look installed and resolvable for
// one test. The real slicer folders are never scanned.
func stubInstalledOrca(t *testing.T, ids ...string) {
	t.Helper()
	ready := map[string]bool{}
	targets := []presets.OrcaTarget{}
	for _, id := range ids {
		ready[id] = true
		targets = append(targets, presets.OrcaTarget{ID: id})
	}
	prevDiscover, prevResolve := discoverInstalledOrca, resolveInstalledOrca
	t.Cleanup(func() {
		discoverInstalledOrca, resolveInstalledOrca = prevDiscover, prevResolve
	})
	discoverInstalledOrca = func() presets.OrcaDiscovery {
		return presets.OrcaDiscovery{Targets: targets}
	}
	resolveInstalledOrca = func(id string) (*presets.OrcaBundle, error) {
		if !ready[id] {
			return nil, fmt.Errorf("installed profile is not available")
		}
		return &presets.OrcaBundle{Target: id}, nil
	}
}

// writeMyPrintersFile places a raw selection file exactly as an older version
// would have left it, and returns its path.
func writeMyPrintersFile(t *testing.T, raw string) string {
	t.Helper()
	path, err := myPrintersPath()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNewSelectionAcceptsOnlyWorkingDestinations(t *testing.T) {
	withFreshSettings(t)
	stubInstalledOrca(t, readyOrca)
	app := &App{}
	picked := []string{
		"snapmaker-u1", "flashforge-creator-5", "elegoo-centauri-carbon-2", "qidi-x-plus-5",
		"prusa-project", readyOrca,
	}
	saved, err := app.SaveMyPrinters(picked)
	if err != nil || !reflect.DeepEqual(saved.SelectedIDs, picked) {
		t.Fatalf("working destinations rejected: %+v %v", saved, err)
	}
	again, err := app.GetMyPrinters()
	if err != nil || !reflect.DeepEqual(again.SelectedIDs, picked) {
		t.Fatalf("working destinations not reloaded: %+v %v", again, err)
	}
}

func TestNewSelectionRejectsIndexOnlyBambuAndMalformedIDs(t *testing.T) {
	withFreshSettings(t)
	stubInstalledOrca(t, readyOrca)
	app := &App{}
	// Index-only model without an adapter, Bambu (source-only), a bare adapter
	// key, an unknown ID, malformed installed-looking IDs, and empty input.
	for _, id := range []string{
		"prusa-prusa-mk4",
		"bambu-lab-p1s",
		"u1",
		"unknown-printer",
		"orca-../x",
		"Snapmaker-U1",
		"snapmaker-u1 ",
		"",
	} {
		if _, err := app.SaveMyPrinters([]string{id}); err == nil {
			t.Fatalf("%q saved as a new printer selection", id)
		}
	}
	result, err := app.GetMyPrinters()
	if err != nil || !result.OnboardingPending {
		t.Fatalf("rejected saves changed onboarding state: %+v %v", result, err)
	}
}

func TestNewSelectionRequiresInstalledOrcaProfileToResolve(t *testing.T) {
	withFreshSettings(t)
	stubInstalledOrca(t)
	app := &App{}
	if _, err := app.SaveMyPrinters([]string{readyOrca}); err == nil {
		t.Fatal("profile that is not installed saved as a destination")
	}
	path, err := myPrintersPath()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("failed save wrote selection state: %v", err)
	}
}

func TestNewSelectionRejectsDuplicatesAndOverLimit(t *testing.T) {
	withFreshSettings(t)
	app := &App{}
	if _, err := app.SaveMyPrinters([]string{"snapmaker-u1", "snapmaker-u1"}); err == nil {
		t.Fatal("duplicate selection saved")
	}
	over := make([]string, 0, maxMyPrinters+1)
	for i := 0; i <= maxMyPrinters; i++ {
		over = append(over, fmt.Sprintf("orca-acme-model-%d-%010x", i, i))
	}
	stubInstalledOrca(t, over...)
	if _, err := app.SaveMyPrinters(over); err == nil || !strings.Contains(err.Error(), "at most") {
		t.Fatalf("selection over the limit saved: %v", err)
	}
	if _, err := app.SaveMyPrinters(over[:maxMyPrinters]); err != nil {
		t.Fatalf("%d ready targets rejected: %v", maxMyPrinters, err)
	}
}

func TestOldCatalogueOnlySelectionLoadsWithoutCorruption(t *testing.T) {
	withFreshSettings(t)
	stubInstalledOrca(t)
	raw := `{"version":1,"selectedIds":["snapmaker-u1","prusa-prusa-mk4","bambu-lab-p1s","unknown-printer","qidi-x-plus-5"]}`
	path := writeMyPrintersFile(t, raw)
	app := &App{}
	got, err := app.GetMyPrinters()
	if err != nil || got.OnboardingPending ||
		!reflect.DeepEqual(got.SelectedIDs, []string{"snapmaker-u1", "qidi-x-plus-5"}) {
		t.Fatalf("old catalogue-only selection not hidden cleanly: %+v %v", got, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != raw {
		t.Fatalf("reading rewrote the user's file: %q %v", after, err)
	}
}

func TestMissingInstalledProfileIsHiddenNotDeleted(t *testing.T) {
	withFreshSettings(t)
	raw := `{"version":1,"selectedIds":["snapmaker-u1","` + readyOrca + `"]}`
	path := writeMyPrintersFile(t, raw)
	app := &App{}
	stubInstalledOrca(t)
	got, err := app.GetMyPrinters()
	if err != nil || !reflect.DeepEqual(got.SelectedIDs, []string{"snapmaker-u1"}) {
		t.Fatalf("missing installed profile not hidden: %+v %v", got, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != raw {
		t.Fatalf("missing profile removed from the file on read: %q %v", after, err)
	}
	stubInstalledOrca(t, readyOrca)
	back, err := app.GetMyPrinters()
	if err != nil || !reflect.DeepEqual(back.SelectedIDs, []string{"snapmaker-u1", readyOrca}) {
		t.Fatalf("reinstalled profile not restored from untouched file: %+v %v", back, err)
	}
}

func TestSavingOverStaleSelectionKeepsPreferences(t *testing.T) {
	withFreshSettings(t)
	stubInstalledOrca(t)
	if err := writeSettings(defaultSettings()); err != nil {
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
	writeMyPrintersFile(t, `{"version":1,"selectedIds":["prusa-prusa-mk4","snapmaker-u1","`+readyOrca+`"]}`)
	app := &App{}
	saved, err := app.SaveMyPrinters([]string{"elegoo-centauri-carbon-2"})
	if err != nil || !reflect.DeepEqual(saved.SelectedIDs, []string{"elegoo-centauri-carbon-2"}) {
		t.Fatalf("save over stale selection failed: %+v %v", saved, err)
	}
	reloaded, err := app.GetMyPrinters()
	if err != nil || !reflect.DeepEqual(reloaded.SelectedIDs, []string{"elegoo-centauri-carbon-2"}) {
		t.Fatalf("stale entries survived the save: %+v %v", reloaded, err)
	}
	after, err := os.ReadFile(settingsFile)
	if err != nil || string(before) != string(after) {
		t.Fatalf("printer selection changed slicer or nozzle preferences: %v", err)
	}
}

func TestMalformedSavedSelectionStillFailsClosed(t *testing.T) {
	for _, raw := range []string{
		`{"version":1,"selectedIds":["Snapmaker-U1"]}`,
		`{"version":1,"selectedIds":["snapmaker-u1","snapmaker-u1"]}`,
		`{"version":1,"selectedIds":["orca-../x"]}`,
	} {
		withFreshSettings(t)
		stubInstalledOrca(t)
		path := writeMyPrintersFile(t, raw)
		app := &App{}
		if _, err := app.GetMyPrinters(); err == nil {
			t.Fatalf("malformed saved selection accepted on read: %s", raw)
		}
		if _, err := app.SaveMyPrinters([]string{"snapmaker-u1"}); err == nil {
			t.Fatalf("malformed saved selection overwritten: %s", raw)
		}
		after, err := os.ReadFile(path)
		if err != nil || string(after) != raw {
			t.Fatalf("corrupt original state not preserved: %q %v", after, err)
		}
	}
}

func TestPrusaProjectSelectionRoundTrips(t *testing.T) {
	withFreshSettings(t)
	stubInstalledOrca(t)
	app := &App{}
	if _, err := app.SaveMyPrinters([]string{"prusa-project"}); err != nil {
		t.Fatal(err)
	}
	got, err := app.GetMyPrinters()
	if err != nil || !reflect.DeepEqual(got.SelectedIDs, []string{"prusa-project"}) {
		t.Fatalf("generic PrusaSlicer selection not reloaded: %+v %v", got, err)
	}
}
