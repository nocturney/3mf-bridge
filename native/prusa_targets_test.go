package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nocturney/slicebridge/native/presets"
)

const prusaTestID = "prusa-qidi-plus5-0-4-nozzle-0123456789"
const prusaOtherID = "prusa-qidi-plus5-0-6-nozzle-fedcba9876"

func prusaTestTarget(id string) presets.PrusaTarget {
	return presets.PrusaTarget{ID: id, Title: "QIDI Plus5 0.4 nozzle", Slicer: "PrusaSlicer", Vendor: "QIDITechnology", Model: "X-Plus 4", Nozzle: "0.4"}
}

// stubPrusa replaces the installed-preset seams for one test and restores them.
func stubPrusa(t *testing.T, installed ...string) {
	t.Helper()
	oldDiscover, oldResolve, oldVerified := discoverInstalledPrusa, resolveInstalledPrusa, prusaExportVerified
	t.Cleanup(func() {
		discoverInstalledPrusa, resolveInstalledPrusa, prusaExportVerified = oldDiscover, oldResolve, oldVerified
	})
	discoverInstalledPrusa = func() presets.PrusaDiscovery {
		discovery := presets.PrusaDiscovery{Targets: []presets.PrusaTarget{}, Rejected: []presets.PrusaRejection{}}
		for _, id := range installed {
			discovery.Targets = append(discovery.Targets, prusaTestTarget(id))
		}
		return discovery
	}
	resolveInstalledPrusa = func(id string) (presets.PrusaTarget, error) {
		for _, candidate := range installed {
			if candidate == id {
				return prusaTestTarget(id), nil
			}
		}
		return presets.PrusaTarget{}, errors.New("installed printer profile is not available or no longer resolves completely")
	}
}

func TestPrusaTargetIDsAreAcceptedOnlyInValidatedShape(t *testing.T) {
	if err := validateTargets([]string{"u1", prusaTestID, prusaTargetID}); err != nil {
		t.Fatalf("installed Prusa and generic targets should mix with built-ins: %v", err)
	}
	for _, bad := range []string{"prusa-../x", "prusa-qidi", "PRUSA-QIDI-0123456789", "prusa-qidi-012345678", "prusa-project-extra"} {
		if err := validateTargets([]string{bad}); err == nil {
			t.Fatalf("%q accepted as a conversion target", bad)
		}
	}
	if err := validateTargets([]string{prusaTestID, prusaTestID}); err == nil {
		t.Fatal("repeated installed Prusa target accepted")
	}
}

func TestInstalledPrusaReplyIsNeverReadyWhileExportIsUnverified(t *testing.T) {
	stubPrusa(t, prusaTestID)
	reply := (&App{}).InstalledPrusaTargets()
	if len(reply.Targets) != 1 || reply.Targets[0].ID != prusaTestID {
		t.Fatalf("complete installed printer not listed: %+v", reply.Targets)
	}
	if reply.Targets[0].Ready || reply.Targets[0].Reason != prusaExportNotVerified {
		t.Fatalf("unverified export reported as ready: %+v", reply.Targets[0])
	}
	encoded, err := json.Marshal(reply)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"targets":`, `"rejected":`, `"ready":false`, `"slicer":"PrusaSlicer"`, `"vendor":"QIDITechnology"`, `"nozzle":"0.4"`} {
		if !strings.Contains(string(encoded), key) {
			t.Fatalf("reply JSON lacks %s: %s", key, encoded)
		}
	}
	// With the gate open, ready follows the verified state. Readiness never
	// depends on a per-target flag.
	prusaExportVerified = true
	if ready := installedPrusaReply(discoverInstalledPrusa()).Targets[0].Ready; !ready {
		t.Fatal("verified export should mark a complete target ready")
	}
}

func TestPrusaSelectionIsInventoryEvenWhenConversionIsUnverified(t *testing.T) {
	catalog := PrinterCatalogReply{}
	stubPrusa(t, prusaTestID)
	if err := validateNewSelection([]string{prusaTestID}, catalog); err != nil {
		t.Fatalf("complete installed Prusa printer must be saveable as inventory: %v", err)
	}
	if ready, _ := prusaReadiness(); ready {
		t.Fatal("saving installed Prusa inventory must not enable conversion")
	}
	if err := validateNewSelection([]string{prusaOtherID}, catalog); err == nil {
		t.Fatal("uninstalled Prusa printer saved as a new selection")
	}
	if err := validateNewSelection([]string{prusaTargetID}, catalog); err != nil {
		t.Fatalf("generic PrusaSlicer project selection must keep working: %v", err)
	}
	prusaExportVerified = true
	if err := validateNewSelection([]string{prusaTestID}, catalog); err != nil {
		t.Fatalf("complete installed Prusa printer rejected: %v", err)
	}
	if err := validateNewSelection([]string{prusaTestID, prusaTestID}, catalog); err == nil {
		t.Fatal("duplicate Prusa selection accepted")
	}
}

func TestSavedSelectionsKeepLegacyGenericAndInstalledPrusaOnly(t *testing.T) {
	catalog := PrinterCatalogReply{Models: []CatalogPrinter{
		{ID: "snapmaker-u1", Manufacturer: "Snapmaker", Model: "U1"},
	}}
	stubPrusa(t, prusaTestID)
	// Five saved selections: generic engine, a catalogue model, an installed
	// Prusa printer, a Prusa printer that was uninstalled, and a legacy
	// adapter ID that is not in the catalogue. Order must survive.
	saved := []string{prusaTargetID, "snapmaker-u1", prusaTestID, prusaOtherID, "u1"}
	installed := installedTargetIDs(saved)
	if !installed[prusaTestID] || installed[prusaOtherID] {
		t.Fatalf("installed map wrong: %v", installed)
	}
	kept, err := keptSavedSelection(saved, catalog, installed)
	if err != nil {
		t.Fatalf("valid saved selections rejected: %v", err)
	}
	want := []string{prusaTargetID, "snapmaker-u1", prusaTestID}
	if strings.Join(kept, ",") != strings.Join(want, ",") {
		t.Fatalf("kept = %v, want %v", kept, want)
	}
	if _, err := keptSavedSelection([]string{prusaTestID, prusaTestID}, catalog, installed); err == nil {
		t.Fatal("duplicate saved Prusa selection treated as valid")
	}
	if _, err := keptSavedSelection([]string{"prusa-../x"}, catalog, installed); err == nil {
		t.Fatal("malformed saved Prusa selection treated as valid")
	}
}

func TestPrusaConversionReResolvesAndNeverWritesWhileUnverified(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "project.3mf")
	if err := os.WriteFile(source, []byte("not a real project"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	stubPrusa(t, prusaTestID)

	result := configuredConvert(source, prusaOtherID, "0.4", PrinterSettings{})
	if result.OK || result.Code != "TARGET_PROFILE_INVALID" {
		t.Fatalf("uninstalled Prusa ID converted or mis-coded: %+v", result)
	}
	result = configuredConvert(source, prusaTestID, "0.4", PrinterSettings{})
	if result.OK || result.Code != "PRUSA_TARGET_NOT_READY" || result.Error != prusaExportNotVerified {
		t.Fatalf("unverified Prusa target not refused explicitly: %+v", result)
	}
	prusaExportVerified = true
	result = configuredConvert(source, prusaTestID, "0.4", PrinterSettings{})
	if result.OK || result.Code != "PRUSA_EXPORT_NOT_IMPLEMENTED" {
		t.Fatalf("export path must refuse until implemented: %+v", result)
	}
	after, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("a refused Prusa conversion wrote files: before %d after %d", len(before), len(after))
	}
}
