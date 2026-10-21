package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/nocturney/slicebridge/native/presets"
)

// catalogueIDFor looks up a real catalogue model by exact manufacturer and model
// name, so tests never invent IDs or select manufacturer-only pseudo IDs.
func catalogueIDFor(t *testing.T, catalog PrinterCatalogReply, manufacturer, model string) string {
	t.Helper()
	for _, item := range catalog.Models {
		if item.Manufacturer == manufacturer && item.Model == model {
			return item.ID
		}
	}
	t.Fatalf("catalogue has no %s %s", manufacturer, model)
	return ""
}

func TestCatalogueOnlyKnownModelsSaveAndReloadWithoutBecomingTargets(t *testing.T) {
	withFreshSettings(t)
	stubInstalledOrca(t)
	catalog, err := catalogSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	// Prusa and Creality include index_only entries; Sovol and RatRig are index-only
	// models, so this covers catalogue-only manufacturers as well as ready targets.
	picked := []string{
		catalogueIDFor(t, catalog, "Prusa", "Prusa MK4"),
		catalogueIDFor(t, catalog, "Sovol", "Sovol SV06"),
		catalogueIDFor(t, catalog, "Creality", "Creality K1C"),
		catalogueIDFor(t, catalog, "RatRig", "RatRig V-Minion"),
		"snapmaker-u1",
		"prusa-project",
	}
	settingsFile, err := settingsPath()
	if err != nil {
		t.Fatal(err)
	}
	if err = writeSettings(defaultSettings()); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatal(err)
	}
	app := &App{}
	saved, err := app.SaveMyPrinters(picked)
	if err != nil || !reflect.DeepEqual(saved.SelectedIDs, picked) {
		t.Fatalf("catalogue-only selection rejected or reordered: %+v %v", saved, err)
	}
	again, err := app.GetMyPrinters()
	if err != nil || again.OnboardingPending || !reflect.DeepEqual(again.SelectedIDs, picked) {
		t.Fatalf("catalogue-only selection not reloaded: %+v %v", again, err)
	}
	path, err := myPrintersPath()
	if err != nil {
		t.Fatal(err)
	}
	blob, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var disk myPrintersOnDisk
	if err = json.Unmarshal(blob, &disk); err != nil || disk.Version != 1 ||
		!reflect.DeepEqual(disk.SelectedIDs, picked) {
		t.Fatalf("version 1 schema not preserved on disk: %s %v", blob, err)
	}
	after, err := os.ReadFile(settingsFile)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("catalogue selection changed slicer or hardware settings: %v", err)
	}
	// Selection is inventory only: the catalogue-only models may not pass the gate.
	for _, id := range picked[:4] {
		if err = validateTargets([]string{id}); err == nil {
			t.Fatalf("catalogue-only %s accepted as a conversion target", id)
		}
	}
}

func TestEveryKnownCatalogueModelIsSelectableExceptBambu(t *testing.T) {
	catalog, err := catalogSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range catalog.Models {
		bambu := strings.EqualFold(item.Manufacturer, "Bambu Lab")
		err := validateNewSelection([]string{item.ID}, catalog)
		if bambu && err == nil {
			t.Fatalf("Bambu source-only model %s selectable", item.ID)
		}
		if !bambu && err != nil {
			t.Fatalf("known catalogue model %s rejected: %v", item.ID, err)
		}
	}
}

func TestCatalogueSelectionNeverGrantsConversionTargets(t *testing.T) {
	catalog, err := catalogSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range catalog.Models {
		if item.LegacyTarget != nil {
			continue
		}
		if presets.IsOrcaTargetID(item.ID) {
			t.Fatalf("catalogue ID %s collides with the Orca target namespace", item.ID)
		}
		if err = validateTargets([]string{item.ID}); err == nil {
			t.Fatalf("catalogue-only %s accepted as a conversion target", item.ID)
		}
	}
	if err = validateTargets([]string{"u1", "c5", "cc2", "qidi", prusaTargetID}); err != nil {
		t.Fatalf("legacy adapters and generic engine lost conversion gate: %v", err)
	}
}

func TestCatalogueSelectionCapsAtThirtyTwoAndSavesAtLimit(t *testing.T) {
	withFreshSettings(t)
	catalog, err := catalogSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, item := range catalog.Models {
		if !strings.EqualFold(item.Manufacturer, "Bambu Lab") {
			ids = append(ids, item.ID)
		}
		if len(ids) == maxMyPrinters+1 {
			break
		}
	}
	if len(ids) != maxMyPrinters+1 {
		t.Fatalf("catalogue has too few selectable models: %d", len(ids))
	}
	app := &App{}
	if _, err = app.SaveMyPrinters(ids); err == nil || !strings.Contains(err.Error(), "at most") {
		t.Fatalf("33 catalogue selections saved: %v", err)
	}
	if _, err = app.GetMyPrinters(); err != nil {
		t.Fatalf("failed save changed state: %v", err)
	}
	saved, err := app.SaveMyPrinters(ids[:maxMyPrinters])
	if err != nil || !reflect.DeepEqual(saved.SelectedIDs, ids[:maxMyPrinters]) {
		t.Fatalf("%d catalogue selections rejected: %v", maxMyPrinters, err)
	}
	reloaded, err := app.GetMyPrinters()
	if err != nil || !reflect.DeepEqual(reloaded.SelectedIDs, ids[:maxMyPrinters]) {
		t.Fatalf("limit selection not reloaded: %+v %v", reloaded, err)
	}
}

func TestExistingFiveSelectionsRoundTripUnchanged(t *testing.T) {
	withFreshSettings(t)
	stubInstalledOrca(t)
	existing := []string{"snapmaker-u1", "flashforge-creator-5", "elegoo-centauri-carbon-2", "qidi-x-plus-5", "prusa-project"}
	quoted := make([]string, 0, len(existing))
	for _, id := range existing {
		quoted = append(quoted, fmt.Sprintf("%q", id))
	}
	writeMyPrintersFile(t, fmt.Sprintf(`{"version":1,"selectedIds":[%s]}`, strings.Join(quoted, ",")))
	app := &App{}
	got, err := app.GetMyPrinters()
	if err != nil || !reflect.DeepEqual(got.SelectedIDs, existing) {
		t.Fatalf("existing five selections not kept: %+v %v", got, err)
	}
	saved, err := app.SaveMyPrinters(existing)
	if err != nil || !reflect.DeepEqual(saved.SelectedIDs, existing) {
		t.Fatalf("existing five selections not re-saved: %+v %v", saved, err)
	}
}
