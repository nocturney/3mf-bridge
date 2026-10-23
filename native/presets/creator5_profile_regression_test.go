package presets

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// These regressions mirror how Flashforge ships Creator 5 Pro machine presets:
// each nozzle variant inherits an Adventurer 5M Pro 0.4 machine and overrides
// printer_model and nozzle_diameter (four tools). Everything here is synthetic
// and lives under t.TempDir(); no installed slicer data is read or copied.

const (
	c5ParentName    = "Flashforge Adventurer 5M Pro 0.4 Nozzle"
	c5ParentProcess = "0.20mm Standard @AD5M Pro 0.4"
	c5ParentPLA     = "Generic PLA @AD5M"
	c5ProModel      = "Flashforge Creator 5 Pro"
)

func c5ProMachineName(nozzle string) string {
	return "Flashforge Creator 5 Pro " + nozzle + " nozzle"
}

func c5ProProcessName(nozzle, layer string) string {
	return fmt.Sprintf("%smm Standard @FF C5 %s nozzle", layer, nozzle)
}

func c5ProFilamentName(nozzle string) string {
	return "Generic PLA @FF C5P " + nozzle
}

// creator5ProFixture builds the AD5M Pro parent plus one inheriting Creator 5 Pro
// variant per entry of {nozzle: layer height}. The variants' default_print_profile
// points at the parent's AD5M process, which lists only the parent machine.
func creator5ProFixture(variants map[string]string) map[string]map[string]any {
	files := map[string]map[string]any{
		"machine/" + c5ParentName + ".json": {
			"name": c5ParentName, "printer_model": "Flashforge Adventurer 5M Pro", "printer_technology": "FFF",
			"nozzle_diameter":     []any{"0.4"},
			"machine_start_gcode": "G28 ; inherited", "machine_end_gcode": "M104 S0 ; inherited",
			"default_print_profile":    c5ParentProcess,
			"default_filament_profile": []any{c5ParentPLA},
		},
		"process/" + c5ParentProcess + ".json": {
			"name": c5ParentProcess, "compatible_printers": []any{c5ParentName}, "layer_height": "0.20",
		},
		"filament/" + c5ParentPLA + ".json": {
			"name": c5ParentPLA, "compatible_printers": []any{c5ParentName},
			"filament_type": []any{"PLA"}, "nozzle_temperature": []any{"210"},
		},
	}
	for nozzle, layer := range variants {
		machine, process, filament := c5ProMachineName(nozzle), c5ProProcessName(nozzle, layer), c5ProFilamentName(nozzle)
		defaultProcess := c5ParentProcess
		if nozzle == "0.25" {
			defaultProcess = process // the vendor's 0.25 default is correct
		}
		files["machine/"+machine+".json"] = map[string]any{
			"name": machine, "inherits": c5ParentName, "printer_model": c5ProModel,
			"nozzle_diameter":          []any{nozzle, nozzle, nozzle, nozzle},
			"default_print_profile":    defaultProcess,
			"default_filament_profile": []any{filament},
		}
		files["process/"+process+".json"] = map[string]any{
			"name": process, "compatible_printers": []any{machine}, "layer_height": layer,
		}
		files["filament/"+filament+".json"] = map[string]any{
			"name": filament, "compatible_printers": []any{machine},
			"filament_type": []any{"PLA"}, "nozzle_temperature": []any{"210"},
		}
	}
	return files
}

func targetByMachine(t *testing.T, discovery OrcaDiscovery, machine string) OrcaTarget {
	t.Helper()
	for _, target := range discovery.Targets {
		if target.Machine == machine {
			return target
		}
	}
	t.Fatalf("machine %q not offered: targets %+v rejected %+v", machine, discovery.Targets, discovery.Rejected)
	return OrcaTarget{}
}

func rejectionFor(discovery OrcaDiscovery, machine string) (OrcaRejection, bool) {
	for _, rejection := range discovery.Rejected {
		if rejection.Machine == machine {
			return rejection, true
		}
	}
	return OrcaRejection{}, false
}

func TestCreator5ProNozzleAndToolsFollowTargetOverrideNotInheritedParent(t *testing.T) {
	root := writeOrcaTree(t, "Flashforge", creator5ProFixture(map[string]string{"0.25": "0.12", "0.4": "0.20"}))
	locations := []slicerLocation{{id: "flash-studio", name: "Flash Studio", root: root}}
	discovery := discoverOrca(locations)
	if len(discovery.Rejected) != 0 {
		t.Fatalf("unexpected rejections: %+v", discovery.Rejected)
	}
	parent := targetByMachine(t, discovery, c5ParentName)
	if parent.Nozzle != "0.4" || parent.PhysicalTools != 1 {
		t.Fatalf("parent must keep its own single 0.4 tool: %+v", parent)
	}
	for _, nozzle := range []string{"0.25", "0.4"} {
		target := targetByMachine(t, discovery, c5ProMachineName(nozzle))
		if target.Model != c5ProModel || target.Nozzle != nozzle || target.PhysicalTools != 4 {
			t.Fatalf("nozzle %s: model/nozzle/tools must come from the Creator 5 Pro override: %+v", nozzle, target)
		}
		bundle, err := resolveOrca(locations, target.ID)
		if err != nil {
			t.Fatal(err)
		}
		if bundle.Nozzle != nozzle || bundle.PhysicalTools != 4 || bundle.Profile.FilamentSlots != 4 {
			t.Fatalf("nozzle %s: bundle nozzle %s tools %d slots %d", nozzle, bundle.Nozzle, bundle.PhysicalTools, bundle.Profile.FilamentSlots)
		}
		settings := bundle.Profile.ProjectSettings
		diameters, _ := settings["nozzle_diameter"].([]any)
		if len(diameters) != 4 {
			t.Fatalf("nozzle %s: project nozzle_diameter %v is not four tools", nozzle, settings["nozzle_diameter"])
		}
		for _, d := range diameters {
			if d != nozzle {
				t.Fatalf("nozzle %s: project nozzle_diameter %v leaked the inherited 0.4 value", nozzle, diameters)
			}
		}
		if settings["printer_model"] != c5ProModel || settings["printer_settings_id"] != c5ProMachineName(nozzle) {
			t.Fatalf("nozzle %s: identity wrong: %v / %v", nozzle, settings["printer_model"], settings["printer_settings_id"])
		}
		// Firmware values missing from the leaf still come from the inherited machine.
		if settings["machine_start_gcode"] != "G28 ; inherited" || settings["machine_end_gcode"] != "M104 S0 ; inherited" {
			t.Fatalf("nozzle %s: inherited gcode lost: %v", nozzle, settings["machine_start_gcode"])
		}
	}
}

func TestCreator5ProProcessAndFilamentAreExplicitlyCompatibleNeverTheParents(t *testing.T) {
	root := writeOrcaTree(t, "Flashforge", creator5ProFixture(map[string]string{"0.25": "0.12", "0.4": "0.20"}))
	locations := []slicerLocation{{id: "flash-studio", name: "Flash Studio", root: root}}
	discovery := discoverOrca(locations)
	for _, nozzle := range []string{"0.25", "0.4"} {
		layer := map[string]string{"0.25": "0.12", "0.4": "0.20"}[nozzle]
		bundle, err := resolveOrca(locations, targetByMachine(t, discovery, c5ProMachineName(nozzle)).ID)
		if err != nil {
			t.Fatal(err)
		}
		settings := bundle.Profile.ProjectSettings
		if settings["print_settings_id"] != c5ProProcessName(nozzle, layer) {
			t.Fatalf("nozzle %s: process %v, want %s", nozzle, settings["print_settings_id"], c5ProProcessName(nozzle, layer))
		}
		if settings["print_settings_id"] == c5ParentProcess {
			t.Fatalf("nozzle %s: the AD5M process default must not be used for Creator 5 Pro", nozzle)
		}
		if bundle.Profile.FilamentProfiles["PLA"] != c5ProFilamentName(nozzle) ||
			bundle.Profile.DefaultFilamentProfile != c5ProFilamentName(nozzle) {
			t.Fatalf("nozzle %s: PLA binding %+v", nozzle, bundle.Profile.FilamentProfiles)
		}
		if settings["layer_height"] != layer {
			t.Fatalf("nozzle %s: layer_height %v, want %s", nozzle, settings["layer_height"], layer)
		}
	}
}

func TestCreator5ProTargetIDsAreDistinctAndStable(t *testing.T) {
	root := writeOrcaTree(t, "Flashforge", creator5ProFixture(map[string]string{"0.25": "0.12", "0.4": "0.20"}))
	locations := []slicerLocation{{id: "flash-studio", name: "Flash Studio", root: root}}
	first, second := discoverOrca(locations), discoverOrca(locations)
	if len(first.Targets) != 3 || len(second.Targets) != 3 {
		t.Fatalf("want parent plus two Creator 5 Pro variants: %+v", first)
	}
	seen := map[string]string{}
	for i, target := range first.Targets {
		if target.ID != second.Targets[i].ID {
			t.Fatalf("ID for %q changed between scans: %s vs %s", target.Machine, target.ID, second.Targets[i].ID)
		}
		if !IsOrcaTargetID(target.ID) {
			t.Fatalf("malformed ID %q", target.ID)
		}
		if other, dup := seen[target.ID]; dup {
			t.Fatalf("%q and %q share ID %s", target.Machine, other, target.ID)
		}
		seen[target.ID] = target.Machine
	}
	// The saved selection format: vendor + machine slug, then a 10-hex file hash.
	id025 := targetByMachine(t, first, c5ProMachineName("0.25")).ID
	if !strings.HasPrefix(id025, "orca-flashforge-flashforge-creator-5-pro-0-25-") {
		t.Fatalf("0.25 ID lost its stable slug: %s", id025)
	}
	id04 := targetByMachine(t, first, c5ProMachineName("0.4")).ID
	if !strings.HasPrefix(id04, "orca-flashforge-flashforge-creator-5-pro-0-4-") || id025 == id04 {
		t.Fatalf("0.4 ID wrong or colliding with 0.25: %s vs %s", id04, id025)
	}
	if prefix := OrcaPrefix(id025); !strings.HasPrefix(prefix, "ORCA-FLASHFORGE-") || len(prefix) > len("ORCA-")+24 {
		t.Fatalf("output prefix %q", prefix)
	}
	if _, err := resolveOrca(locations, id025); err != nil {
		t.Fatal(err)
	}
}

func TestCreator5ProWithoutCompatibleProcessOrPLAFailsClosed(t *testing.T) {
	cases := map[string]struct {
		mutate func(files map[string]map[string]any)
		reason string
	}{
		"own process file removed, parent AD5M default exists": {
			func(f map[string]map[string]any) {
				delete(f, "process/"+c5ProProcessName("0.4", "0.20")+".json")
			},
			"no installed compatible process",
		},
		"own process no longer lists the machine": {
			func(f map[string]map[string]any) {
				f["process/"+c5ProProcessName("0.4", "0.20")+".json"]["compatible_printers"] = []any{c5ParentName}
			},
			"no installed compatible process",
		},
		"only process has a layer height too thick for the nozzle": {
			func(f map[string]map[string]any) {
				f["process/"+c5ProProcessName("0.4", "0.20")+".json"]["layer_height"] = "0.40"
			},
			"no installed compatible process",
		},
		"no compatible PLA": {
			func(f map[string]map[string]any) {
				f["filament/"+c5ProFilamentName("0.4")+".json"]["compatible_printers"] = []any{c5ParentName}
			},
			"no PLA filament",
		},
		"mixed nozzle override": {
			func(f map[string]map[string]any) {
				f["machine/"+c5ProMachineName("0.4")+".json"]["nozzle_diameter"] = []any{"0.4", "0.4", "0.25", "0.25"}
			},
			"mixed nozzle",
		},
	}
	for name, tc := range cases {
		files := creator5ProFixture(map[string]string{"0.4": "0.20"})
		tc.mutate(files)
		root := writeOrcaTree(t, "Flashforge", files)
		locations := []slicerLocation{{id: "flash-studio", name: "Flash Studio", root: root}}
		discovery := discoverOrca(locations)
		machine := c5ProMachineName("0.4")
		for _, target := range discovery.Targets {
			if target.Machine == machine {
				t.Fatalf("%s: broken Creator 5 Pro was offered: %+v", name, target)
			}
		}
		rejection, ok := rejectionFor(discovery, machine)
		if !ok || !strings.Contains(rejection.Reason, tc.reason) {
			t.Fatalf("%s: want rejection containing %q, got %+v", name, tc.reason, discovery.Rejected)
		}
		// The unaffected parent is still offered; one bad machine hides nothing.
		targetByMachine(t, discovery, c5ParentName)
		// Re-resolving an ID from before the breakage must also fail closed.
		staleID := orcaTargetID("Flashforge", machine, pathOfMachine(root, machine))
		if _, err := resolveOrca(locations, staleID); err == nil {
			t.Fatalf("%s: stale ID still resolved", name)
		}
	}
}

func pathOfMachine(root, machine string) string {
	return filepath.Join(root, "Flashforge", "machine", machine+".json")
}

func TestPlainCreator5ModelStaysWithBuiltInEngine(t *testing.T) {
	files := creator5ProFixture(map[string]string{"0.25": "0.12"})
	plain := "Flashforge Creator 5 0.4 nozzle"
	files["machine/"+plain+".json"] = map[string]any{
		"name": plain, "inherits": c5ParentName, "printer_model": "Flashforge Creator 5",
		"nozzle_diameter":       []any{"0.4", "0.4", "0.4", "0.4"},
		"default_print_profile": c5ParentProcess,
	}
	files["process/0.20mm Standard @FF C5.json"] = map[string]any{
		"name": "0.20mm Standard @FF C5", "compatible_printers": []any{plain}, "layer_height": "0.20",
	}
	files["filament/Generic PLA @FF C5.json"] = map[string]any{
		"name": "Generic PLA @FF C5", "compatible_printers": []any{plain},
		"filament_type": []any{"PLA"},
	}
	discovery := discoverFrom(writeOrcaTree(t, "Flashforge", files))
	for _, target := range discovery.Targets {
		if target.Model == "Flashforge Creator 5" {
			t.Fatalf("plain Creator 5 belongs to the built-in c5 engine, not the Orca list: %+v", target)
		}
	}
	rejection, ok := rejectionFor(discovery, plain)
	if !ok || !strings.Contains(rejection.Reason, "built-in engine") {
		t.Fatalf("plain Creator 5 must be reported as built-in: %+v", discovery.Rejected)
	}
	// The distinct Pro model is not caught by that guard.
	targetByMachine(t, discovery, c5ProMachineName("0.25"))
}

func TestCreator5ProOwnershipAndBambuSourceOnlyRules(t *testing.T) {
	files := creator5ProFixture(map[string]string{"0.25": "0.12"})
	flash := writeOrcaTree(t, "Flashforge", files)

	// Flashforge's own fork and generic Orca both offer the profile.
	for _, loc := range []slicerLocation{
		{id: "flash-studio", name: "Flash Studio", root: flash},
		{id: "orca-slicer", name: "OrcaSlicer", root: flash},
	} {
		discovery := discoverOrca([]slicerLocation{loc})
		if target := targetByMachine(t, discovery, c5ProMachineName("0.25")); target.Slicer != loc.name {
			t.Fatalf("%s: owner wrong: %+v", loc.id, target)
		}
	}
	// Other makers' branded forks bundle Flashforge libraries but do not own them.
	for _, id := range []string{"elegoo-slicer", "snapmaker-orca", "qidi-studio"} {
		discovery := discoverOrca([]slicerLocation{{id: id, name: id, root: flash}})
		if len(discovery.Targets) != 0 || len(discovery.Rejected) != 0 {
			t.Fatalf("%s must not offer Flashforge profiles: %+v", id, discovery)
		}
	}
	// A fork copy listed first owns the target; the generic copy is not a duplicate.
	generic := writeOrcaTree(t, "Flashforge", files)
	discovery := discoverOrca([]slicerLocation{
		{id: "flash-studio", name: "Flash Studio", root: flash},
		{id: "orca-slicer", name: "OrcaSlicer", root: generic},
	})
	if len(discovery.Targets) != 2 {
		t.Fatalf("parent plus Creator 5 Pro expected exactly once each: %+v", discovery.Targets)
	}
	for _, target := range discovery.Targets {
		if target.Slicer != "Flash Studio" {
			t.Fatalf("manufacturer fork must win over generic Orca: %+v", target)
		}
	}
	// Bambu stays source-only even when it carries identical, complete data.
	bbl := discoverFrom(writeOrcaTree(t, "BBL", files))
	if len(bbl.Targets) != 0 || len(bbl.Rejected) != 0 {
		t.Fatalf("BBL must never become a conversion destination: %+v", bbl)
	}
}
