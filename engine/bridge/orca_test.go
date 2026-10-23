package bridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/BryanSmee/bl_to_std/pkg/converter"
	"github.com/BryanSmee/bl_to_std/pkg/printer"
)

const syntheticOrcaID = "orca-acme-mini-0-4-nozzle-0123456789"

// syntheticOrcaBundle mirrors what the native resolver writes for an installed
// single-nozzle machine. Tests mutate it to exercise each fail-closed branch.
func syntheticOrcaBundle(tools int) *orcaBundle {
	nozzles := make([]any, tools)
	for i := range nozzles {
		nozzles[i] = "0.4"
	}
	return &orcaBundle{
		targetBundle: targetBundle{
			Target: syntheticOrcaID, Slicer: "Synthetic Orca", Nozzle: "0.4",
			Profile: printer.Profile{
				Name: syntheticOrcaID, DisplayName: "Acme Mini", ModelID: "Acme Mini",
				FilamentSlots:    tools,
				FilamentProfiles: map[string]string{"PLA": "Acme PLA"},
				ProjectSettings: map[string]any{
					"printer_model":       "Acme Mini",
					"nozzle_diameter":     nozzles,
					"machine_start_gcode": "G28 ; installed",
					"machine_end_gcode":   "M104 S0 ; installed",
				},
			},
			MaterialSettings: map[string]map[string]any{
				"PLA": {
					"nozzle_temperature":   []any{"210"},
					"filament_flow_ratio":  []any{"0.98"},
					"filament_type":        []any{"PLA"},
					"filament_settings_id": []any{"Acme PLA"},
				},
			},
		},
		PhysicalTools: tools,
	}
}

func writeOrcaBundle(t *testing.T, bundle *orcaBundle) string {
	t.Helper()
	data, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "orca-profile.json")
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestOrcaBundleAcceptsStructurallyValidSingleToolSample(t *testing.T) {
	bundle, err := loadOrcaBundle(writeOrcaBundle(t, syntheticOrcaBundle(1)))
	if err != nil {
		t.Fatal(err)
	}
	if bundle.PhysicalTools != 1 || bundle.Profile.FilamentSlots != 1 {
		t.Fatalf("physical tools must come from the machine, got %d/%d", bundle.PhysicalTools, bundle.Profile.FilamentSlots)
	}
	if bundle.Profile.ModelID != "Acme Mini" || bundle.Nozzle != "0.4" {
		t.Fatalf("identity not preserved: model %q nozzle %q", bundle.Profile.ModelID, bundle.Nozzle)
	}
}

func TestOrcaBundleRejectsUnverifiedIdentityAndTools(t *testing.T) {
	cases := map[string]func(*orcaBundle){
		"tool count not backed by nozzle array": func(b *orcaBundle) {
			b.PhysicalTools = 2
			b.Profile.FilamentSlots = 2
		},
		"mixed nozzle diameters": func(b *orcaBundle) {
			b.PhysicalTools = 2
			b.Profile.FilamentSlots = 2
			b.Profile.ProjectSettings["nozzle_diameter"] = []any{"0.4", "0.6"}
		},
		"model already served by a built-in engine": func(b *orcaBundle) {
			b.Profile.DisplayName = "X-Plus 5"
			b.Profile.ModelID = "X-Plus 5"
			b.Profile.ProjectSettings["printer_model"] = "X-Plus 5"
		},
		"legacy target name claimed": func(b *orcaBundle) {
			b.Target = "qidi"
			b.Profile.Name = "qidi"
		},
		"missing PLA preset": func(b *orcaBundle) {
			b.Profile.FilamentProfiles = map[string]string{"PETG": "Acme PETG"}
		},
	}
	for name, mutate := range cases {
		bundle := syntheticOrcaBundle(1)
		mutate(bundle)
		if _, err := loadOrcaBundle(writeOrcaBundle(t, bundle)); err == nil {
			t.Fatalf("%s was accepted", name)
		}
	}
}

func TestFiveVirtualSlotsOnOneToolKeepEverySourceSlot(t *testing.T) {
	bundle, err := loadOrcaBundle(writeOrcaBundle(t, syntheticOrcaBundle(1)))
	if err != nil {
		t.Fatal(err)
	}
	filaments := []converter.Filament{
		{ID: 1, Type: "PLA", Color: "#FF0000"},
		// Identical color and material must stay a separate virtual slot.
		{ID: 2, Type: "PLA", Color: "#FF0000"},
		{ID: 3, Type: "PLA", Color: "#00FF00"},
		{ID: 4, Type: "PLA", Color: "#0000FF"},
		{ID: 5, Type: "PLA", Color: "#FFFFFF"},
	}
	slots, mapping, err := sourceSlotPlan(filaments)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 5 || len(mapping) != 5 || mapping[2] != 2 {
		t.Fatalf("source slots folded: %d slots, mapping %v", len(slots), mapping)
	}
	if slots[0].Color != slots[1].Color {
		t.Fatal("test precondition: first two slots should share a color")
	}
	bundle.Profile.FilamentSlots = len(slots)
	if err := bindNativeFilamentSettings(&bundle.targetBundle, filaments); err != nil {
		t.Fatal(err)
	}
	cfg := bundle.Profile.ProjectSettings
	if !reflect.DeepEqual(cfg["filament_type"], []any{"PLA", "PLA", "PLA", "PLA", "PLA"}) {
		t.Fatalf("virtual slot material list wrong: %v", cfg["filament_type"])
	}
	if got, _ := cfg["nozzle_temperature"].([]any); len(got) != 5 {
		t.Fatalf("per-slot temperatures should cover 5 virtual slots: %v", cfg["nozzle_temperature"])
	}
	if bundle.PhysicalTools != 1 {
		t.Fatalf("virtual slots must not change the physical tool count: %d", bundle.PhysicalTools)
	}
}

func TestPhysicalToolWarningWordingIsStable(t *testing.T) {
	if got := physicalToolPhrase(4); got != "four physical tools" {
		t.Fatalf("legacy four-tool wording changed: %q", got)
	}
	if got := physicalToolPhrase(1); got != "one physical tool" {
		t.Fatalf("single-tool wording wrong: %q", got)
	}
	if got := physicalToolPhrase(2); got != "2 physical tools" {
		t.Fatalf("two-tool wording wrong: %q", got)
	}
}

// Creality Print 7.3 rejects output when support flags are shorter than the
// project palette, even when the same installed PLA preset backs every slot.
func TestCrealitySupportFlagsMatchVirtualFilamentSlots(t *testing.T) {
    b := syntheticOrcaBundle(1)
    b.Slicer = "Creality Print"
    b.Profile.ModelID = "Creality K2 Plus"
    b.Profile.ProjectSettings["filament_is_support"] = []any{"0"}
    if err := normalizeCrealitySupportFlags(b, 2); err != nil {
        t.Fatal(err)
    }
    if got := b.Profile.ProjectSettings["filament_is_support"]; !reflect.DeepEqual(got, []any{"0", "0"}) {
        t.Fatalf("multi-colour support flags misaligned: %v", got)
    }
    if got := b.Profile.ProjectSettings["nozzle_diameter"]; !reflect.DeepEqual(got, []any{"0.4"}) {
        t.Fatalf("physical nozzle was modified: %v", got)
    }
    if b.PhysicalTools != 1 {
        t.Fatalf("physical tool count changed to %d", b.PhysicalTools)
    }
}

func TestCrealityFlagNormalizationDoesNotTouchOtherOrcaForks(t *testing.T) {
    for _, tc := range []struct{ slicer, model string }{
        {"QIDIStudio", "QIDI X-Plus 5"},
        {"OrcaSlicer", "Creality K2 Plus"},
        {"Creality Print", "Other Vendor"},
    } {
        b := syntheticOrcaBundle(1)
        b.Slicer, b.Profile.ModelID = tc.slicer, tc.model
        b.Profile.ProjectSettings["filament_is_support"] = []any{"0"}
        if err := normalizeCrealitySupportFlags(b, 3); err != nil {
            t.Fatal(err)
        }
        if got := b.Profile.ProjectSettings["filament_is_support"]; !reflect.DeepEqual(got, []any{"0"}) {
            t.Fatalf("slicer %s / model %s mutated: %v", tc.slicer, tc.model, got)
        }
    }
}

func TestCrealitySupportFlagsRefuseAmbiguousValues(t *testing.T) {
    b := syntheticOrcaBundle(1)
    b.Slicer, b.Profile.ModelID = "Creality Print", "Creality K2 Plus"
    b.Profile.ProjectSettings["filament_is_support"] = []any{"0", "1"}
    if err := normalizeCrealitySupportFlags(b, 3); err == nil {
        t.Fatal("ambiguous per-filament flags must not be silently remapped")
    }
    if got := b.Profile.ProjectSettings["filament_is_support"]; !reflect.DeepEqual(got, []any{"0", "1"}) {
        t.Fatalf("ambiguous flags changed: %v", got)
    }
}

// The installed Creality profile may omit this optional setting; the upstream
// converter's single-slot default is "0". Replicating it is necessary for
// Creality Print's multi-filament 3MF validator.
func TestCrealityMissingSupportFlagsUseInstalledDefault(t *testing.T) {
    b := syntheticOrcaBundle(1)
    b.Slicer, b.Profile.ModelID = "Creality Print", "Creality K2 Plus"
    if err := normalizeCrealitySupportFlags(b, 2); err != nil {
        t.Fatal(err)
    }
    if got := b.Profile.ProjectSettings["filament_is_support"]; !reflect.DeepEqual(got, []any{"0", "0"}) {
        t.Fatalf("missing support flags should become explicit defaults: %v", got)
    }
}
