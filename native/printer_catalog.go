package main

// Offline printer catalogue selection. This file deliberately never modifies
// conversion adapters, old printer preferences, slicer profiles, or hardware.
import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nocturney/slicebridge/native/presets"
)

const maxMyPrinters = 32
const maxPickerBytes = 1 << 20
const maxSavedPickerBytes = 32 << 10

var catalogIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var legacyCatalogIDs = map[string]string{
	"u1": "snapmaker-u1",
	"c5": "flashforge-creator-5",
	"cc2": "elegoo-centauri-carbon-2",
	"qidi": "qidi-x-plus-5",
}

type CatalogPrinter struct {
	ID            string   `json:"id"`
	Manufacturer  string   `json:"manufacturer"`
	Model         string   `json:"model"`
	Aliases       []string `json:"aliases"`
	ReleaseDate   *string  `json:"releaseDate"`
	StructuralStatus string `json:"structuralStatus"`
	Nozzles       []string `json:"nozzles"`
	SourcePublisher *string `json:"sourcePublisher"`
	LegacyTarget  *string  `json:"legacyTarget"`
	ImagePath     *string  `json:"imagePath"`
	LogoPath      *string  `json:"logoPath"`
	LocalCoverCandidates []string `json:"localCoverCandidates"`
}
type PrinterCatalogReply struct {
	SchemaVersion int               `json:"schemaVersion"`
	Kind          string            `json:"kind"`
	ModelCount    int               `json:"modelCount"`
	Manufacturers int               `json:"manufacturers"`
	LegacyAdapterCount int           `json:"legacyAdapterCount"`
	ThirdPartyProfilePayloadsIncluded bool `json:"thirdPartyProfilePayloadsIncluded"`
	Models        []CatalogPrinter  `json:"models"`
}
type MyPrintersReply struct {
	Version           int      `json:"version"`
	SelectedIDs       []string `json:"selectedIds"`
	OnboardingPending bool     `json:"onboardingPending"`
}
type myPrintersOnDisk struct {
	Version     int      `json:"version"`
	SelectedIDs []string `json:"selectedIds"`
}

func catalogSnapshot() (PrinterCatalogReply, error) {
	raw, err := assets.ReadFile("frontend/dist/printer-catalog.json")
	if err != nil {
		return PrinterCatalogReply{}, err
	}
	if len(raw) > maxPickerBytes {
		return PrinterCatalogReply{}, fmt.Errorf("embedded catalogue exceeds limit")
	}
	var catalog PrinterCatalogReply
	if err = json.Unmarshal(raw, &catalog); err != nil {
		return PrinterCatalogReply{}, fmt.Errorf("invalid embedded catalogue: %w", err)
	}
	if catalog.SchemaVersion != 1 || catalog.Kind != "slicebridge-offline-printer-picker" ||
		catalog.ModelCount != 603 || catalog.Manufacturers != 84 ||
		catalog.LegacyAdapterCount != 4 || catalog.ThirdPartyProfilePayloadsIncluded ||
		len(catalog.Models) != catalog.ModelCount {
		return PrinterCatalogReply{}, fmt.Errorf("embedded catalogue schema or source count mismatch")
	}
	ids, adapters, makers := make(map[string]bool), make(map[string]bool), make(map[string]bool)
	for _, model := range catalog.Models {
		if !catalogIDPattern.MatchString(model.ID) || ids[model.ID] ||
			strings.TrimSpace(model.Manufacturer) == "" || strings.TrimSpace(model.Model) == "" ||
			len(model.Manufacturer) > 100 || len(model.Model) > 160 {
			return PrinterCatalogReply{}, fmt.Errorf("invalid/duplicate embedded printer ID")
		}
		if model.StructuralStatus != "index_only" && model.StructuralStatus != "structural_verified" {
			return PrinterCatalogReply{}, fmt.Errorf("unsupported catalogue status")
		}
		// No remote images or third-party artwork bytes are shipped in the app.
		// LocalCoverCandidates contains relative, vetted filenames only.
		if model.ImagePath != nil || model.LogoPath != nil {
			return PrinterCatalogReply{}, fmt.Errorf("unreviewed remote or local catalogue media")
		}
		if len(model.LocalCoverCandidates) > 12 {
			return PrinterCatalogReply{}, fmt.Errorf("too many local cover candidates")
		}
		for _, rel := range model.LocalCoverCandidates {
			if !validLocalCoverPath(rel) {
				return PrinterCatalogReply{}, fmt.Errorf("unsafe local cover reference in catalogue")
			}
		}
		ids[model.ID] = true
		makers[model.Manufacturer] = true
		if model.LegacyTarget != nil {
			key := *model.LegacyTarget
			if legacyCatalogIDs[key] != model.ID || adapters[key] {
				return PrinterCatalogReply{}, fmt.Errorf("catalogue attempts to claim a new conversion adapter")
			}
			adapters[key] = true
		}
	}
	if len(makers) != catalog.Manufacturers || len(adapters) != 4 {
		return PrinterCatalogReply{}, fmt.Errorf("catalogue maker/adapter totals do not match")
	}
	for key := range legacyCatalogIDs {
		if !adapters[key] {
			return PrinterCatalogReply{}, fmt.Errorf("missing legacy adapter")
		}
	}
	return catalog, nil
}

func (a *App) PrinterCatalog() (PrinterCatalogReply, error) {
	return catalogSnapshot()
}

// InstalledOrcaTargets lists locally installed Orca-family machines that
// resolved completely. Selection needs no catalogue status flag or approval.
// Titles come from the offline catalogue when a model matches; otherwise the
// installed printer_model is shown unchanged.
func (a *App) InstalledOrcaTargets() presets.OrcaDiscovery {
	discovery := presets.DiscoverOrca()
	catalog, err := catalogSnapshot()
	if err != nil {
		return discovery
	}
	for i := range discovery.Targets {
		discovery.Targets[i].Title = catalogTitle(catalog, discovery.Targets[i].Model)
	}
	return discovery
}

// catalogTitle returns "Manufacturer Model" for an exact catalogue model or
// alias match, and the raw installed model name otherwise.
func catalogTitle(catalog PrinterCatalogReply, installedModel string) string {
	for _, item := range catalog.Models {
		if strings.EqualFold(item.Manufacturer, "Bambu Lab") {
			continue
		}
		names := append([]string{item.Model}, item.Aliases...)
		for _, name := range names {
			if strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(installedModel)) {
				return strings.TrimSpace(item.Manufacturer + " " + item.Model)
			}
		}
	}
	return installedModel
}

func myPrintersPath() (string, error) {
	oldSettingsPath, err := settingsPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(oldSettingsPath), "my-printers.json"), nil
}

// Injectable for tests. Production reads only locally installed Orca presets,
// through the same resolver conversion uses before it reads a target.
var discoverInstalledOrca = presets.DiscoverOrca
var resolveInstalledOrca = presets.ResolveOrcaBundle

// catalogueSelectable reports whether a catalogue ID may sit in My Printers as
// user-owned inventory. Every known catalogue model qualifies except Bambu Lab,
// which is source-only. Being selectable never makes an ID a conversion target:
// batch and single conversion keep their own gates (outputs, IsOrcaTargetID,
// prusaTargetID), and none of them read this list.
func catalogueSelectable(catalog PrinterCatalogReply, id string) bool {
	for _, item := range catalog.Models {
		if item.ID == id {
			return !strings.EqualFold(item.Manufacturer, "Bambu Lab")
		}
	}
	return false
}

// validateNewSelection accepts the IDs a user may save: any known catalogue model
// except Bambu Lab, the generic PrusaSlicer engine, and installed Orca targets
// that resolve right now. Unknown, malformed, duplicate and Bambu IDs are rejected.
func validateNewSelection(ids []string, catalog PrinterCatalogReply) error {
	if len(ids) > maxMyPrinters {
		return fmt.Errorf("at most %d printers can be selected", maxMyPrinters)
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !catalogIDPattern.MatchString(id) || seen[id] {
			return fmt.Errorf("invalid or duplicate printer selection")
		}
		seen[id] = true
		switch {
		case id == prusaTargetID:
			// Generic engine: readiness is reported by PrusaEngineStatus.
		case presets.IsOrcaTargetID(id):
			if _, err := resolveInstalledOrca(id); err != nil {
				return fmt.Errorf("installed printer profile is not ready")
			}
		case !catalogueSelectable(catalog, id):
			return fmt.Errorf("printer is not in the offline catalogue")
		}
	}
	return nil
}

// keptSavedSelection returns the saved IDs to show, in saved order. Known
// catalogue models stay. Installed Orca targets stay only while installed, and
// the generic engine stays. Unknown IDs, Bambu entries and missing Orca profiles
// are dropped from the reply only; reads never rewrite the file. Malformed or
// duplicate IDs remain corruption.
func keptSavedSelection(ids []string, catalog PrinterCatalogReply, installed map[string]bool) ([]string, error) {
	seen := make(map[string]bool, len(ids))
	kept := []string{}
	for _, id := range ids {
		if !catalogIDPattern.MatchString(id) || seen[id] {
			return nil, fmt.Errorf("invalid or duplicate printer selection")
		}
		seen[id] = true
		switch {
		case id == prusaTargetID:
			kept = append(kept, id)
		case presets.IsOrcaTargetID(id):
			if installed[id] {
				kept = append(kept, id)
			}
		case catalogueSelectable(catalog, id):
			kept = append(kept, id)
		}
	}
	return kept, nil
}

// installedOrcaIDs scans installed Orca targets only when a saved selection names
// one, so ordinary reads never touch the slicer folders.
func installedOrcaIDs(ids []string) map[string]bool {
	installed := map[string]bool{}
	for _, id := range ids {
		if presets.IsOrcaTargetID(id) {
			for _, target := range discoverInstalledOrca().Targets {
				installed[target.ID] = true
			}
			break
		}
	}
	return installed
}

func loadMyPrinters(catalog PrinterCatalogReply) (MyPrintersReply, error) {
	path, err := myPrintersPath()
	if err != nil {
		return MyPrintersReply{}, err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return MyPrintersReply{
			Version: 1, SelectedIDs: []string{},
			OnboardingPending: true,
		}, nil
	}
	if err != nil {
		return MyPrintersReply{}, err
	}
	if len(raw) > maxSavedPickerBytes {
		return MyPrintersReply{}, fmt.Errorf("saved printer selections exceed limit")
	}
	var cfg myPrintersOnDisk
	if err = json.Unmarshal(raw, &cfg); err != nil {
		return MyPrintersReply{}, fmt.Errorf("corrupt saved printer selections: %w", err)
	}
	if cfg.Version != 1 || cfg.SelectedIDs == nil {
		return MyPrintersReply{}, fmt.Errorf("unrecognized saved printer selection version")
	}
	if len(cfg.SelectedIDs) > maxMyPrinters {
		return MyPrintersReply{}, fmt.Errorf("saved printer selections invalid: at most %d printers", maxMyPrinters)
	}
	kept, err := keptSavedSelection(cfg.SelectedIDs, catalog, installedOrcaIDs(cfg.SelectedIDs))
	if err != nil {
		return MyPrintersReply{}, fmt.Errorf("saved printer selections invalid: %w", err)
	}
	return MyPrintersReply{Version: 1, SelectedIDs: kept}, nil
}

func (a *App) GetMyPrinters() (MyPrintersReply, error) {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	catalog, err := catalogSnapshot()
	if err != nil {
		return MyPrintersReply{}, err
	}
	return loadMyPrinters(catalog)
}

func (a *App) SaveMyPrinters(ids []string) (MyPrintersReply, error) {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	catalog, err := catalogSnapshot()
	if err != nil {
		return MyPrintersReply{}, err
	}
	if err = validateNewSelection(ids, catalog); err != nil {
		return MyPrintersReply{}, err
	}
	// Corrupt state is never overwritten: the user can repair/recover it.
	if _, err = loadMyPrinters(catalog); err != nil {
		return MyPrintersReply{}, err
	}
	path, err := myPrintersPath()
	if err != nil {
		return MyPrintersReply{}, err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return MyPrintersReply{}, err
	}
	data, err := json.MarshalIndent(myPrintersOnDisk{
		Version: 1, SelectedIDs: append([]string{}, ids...),
	}, "", "  ")
	if err != nil {
		return MyPrintersReply{}, err
	}
	data = append(data, '\n')
	temporary, err := os.CreateTemp(filepath.Dir(path), ".my-printers-*.tmp")
	if err != nil {
		return MyPrintersReply{}, err
	}
	defer os.Remove(temporary.Name())
	if err = temporary.Chmod(0600); err != nil {
		temporary.Close()
		return MyPrintersReply{}, err
	}
	if _, err = temporary.Write(data); err != nil {
		temporary.Close()
		return MyPrintersReply{}, err
	}
	if err = temporary.Sync(); err != nil {
		temporary.Close()
		return MyPrintersReply{}, err
	}
	if err = temporary.Close(); err != nil {
		return MyPrintersReply{}, err
	}
	if err = os.Rename(temporary.Name(), path); err != nil {
		return MyPrintersReply{}, err
	}
	return MyPrintersReply{Version: 1, SelectedIDs: append([]string{}, ids...)}, nil
}
