package presets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeOrcaTree writes a synthetic Orca-family vendor under a temp root. Real
// installed slicer profiles are never read or modified by these tests.
func writeOrcaTree(t *testing.T, vendor string, files map[string]map[string]any) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		path := filepath.Join(root, vendor, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const acmeMachine = "machine/Acme Mini 0.4.json"
const acmeProcess = "process/0.20mm Acme Standard.json"
const acmeFilament = "filament/Acme PLA.json"

// acmeSample is a structurally complete single-tool Orca-family installation.
func acmeSample() map[string]map[string]any {
	return map[string]map[string]any{
		acmeMachine: {
			"name": "Acme Mini 0.4 nozzle", "printer_model": "Acme Mini", "printer_technology": "FFF",
			"nozzle_diameter":          []any{"0.4"},
			"machine_start_gcode":      "G28",
			"machine_end_gcode":        "M104 S0",
			"default_print_profile":    "0.20mm Acme Standard",
			"default_filament_profile": []any{"Acme PLA"},
		},
		acmeProcess: {
			"name": "0.20mm Acme Standard", "compatible_printers": []any{"Acme Mini 0.4 nozzle"},
			"layer_height": "0.2", "wall_loops": "3",
		},
		acmeFilament: {
			"name": "Acme PLA", "compatible_printers": []any{"Acme Mini 0.4 nozzle"},
			"filament_type": []any{"PLA"}, "nozzle_temperature": []any{"210"},
			"filament_flow_ratio": []any{"0.98"},
		},
	}
}

func discoverFrom(root string) OrcaDiscovery {
	return discoverOrca([]slicerLocation{{id: "test", name: "Test Orca", root: root}})
}

func TestStructurallyValidOrcaSampleBecomesSelectableTarget(t *testing.T) {
	root := writeOrcaTree(t, "Acme", acmeSample())
	discovery := discoverFrom(root)
	if len(discovery.Targets) != 1 || len(discovery.Rejected) != 0 {
		t.Fatalf("want exactly one target, got %+v", discovery)
	}
	target := discovery.Targets[0]
	if !IsOrcaTargetID(target.ID) || target.Model != "Acme Mini" || target.Nozzle != "0.4" || target.PhysicalTools != 1 {
		t.Fatalf("unexpected target %+v", target)
	}
	if len(target.Materials) != 1 || target.Materials[0] != "PLA" {
		t.Fatalf("only the explicitly compatible PLA must be offered: %v", target.Materials)
	}
	// The frontend payload must never reveal where the profile lives.
	payload, err := json.Marshal(discovery)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), root) {
		t.Fatal("install path leaked into the discovery payload")
	}
	bundle, err := resolveOrca([]slicerLocation{{id: "test", name: "Test Orca", root: root}}, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Target != target.ID || bundle.Profile.Name != target.ID || bundle.Profile.ModelID != "Acme Mini" {
		t.Fatalf("bundle identity wrong: target %q name %q model %q", bundle.Target, bundle.Profile.Name, bundle.Profile.ModelID)
	}
	if bundle.PhysicalTools != 1 || bundle.Profile.FilamentSlots != 1 || bundle.Nozzle != "0.4" {
		t.Fatalf("physical data wrong: tools %d slots %d nozzle %s", bundle.PhysicalTools, bundle.Profile.FilamentSlots, bundle.Nozzle)
	}
	if bundle.Profile.FilamentProfiles["PLA"] != "Acme PLA" || bundle.Profile.ProjectSettings["printer_model"] != "Acme Mini" {
		t.Fatalf("material or model binding wrong: %+v", bundle.Profile.FilamentProfiles)
	}
}

func TestPhysicalToolsComeFromInstalledNozzleArray(t *testing.T) {
	files := acmeSample()
	files[acmeMachine]["nozzle_diameter"] = []any{"0.4", "0.4"}
	discovery := discoverFrom(writeOrcaTree(t, "Acme", files))
	if len(discovery.Targets) != 1 || discovery.Targets[0].PhysicalTools != 2 {
		t.Fatalf("two-nozzle machine must report two physical tools: %+v", discovery)
	}
}

func TestUnsupportedOrMissingProfilesAreRejectedWithReasons(t *testing.T) {
	cases := map[string]struct {
		mutate func(files map[string]map[string]any)
		reason string
	}{
		"default process file missing": {
			func(f map[string]map[string]any) { delete(f, acmeProcess) },
			"default process preset",
		},
		"process does not list machine": {
			func(f map[string]map[string]any) { f[acmeProcess]["compatible_printers"] = []any{"Other"} },
			"does not list this machine",
		},
		"no PLA lists machine": {
			func(f map[string]map[string]any) { f[acmeFilament]["compatible_printers"] = []any{"Other"} },
			"no PLA filament",
		},
		"mixed nozzle sizes": {
			func(f map[string]map[string]any) { f[acmeMachine]["nozzle_diameter"] = []any{"0.4", "0.6"} },
			"mixed nozzle",
		},
		"unsupported nozzle size": {
			func(f map[string]map[string]any) { f[acmeMachine]["nozzle_diameter"] = []any{"0.3"} },
			"unsupported nozzle diameter",
		},
		"resin machine": {
			func(f map[string]map[string]any) { f[acmeMachine]["printer_technology"] = "SLA" },
			"printer technology SLA",
		},
		"start/end script missing": {
			func(f map[string]map[string]any) { delete(f[acmeMachine], "machine_end_gcode") },
			"machine_end_gcode is missing",
		},
		"model already served by built-in engine": {
			func(f map[string]map[string]any) { f[acmeMachine]["printer_model"] = "X-Plus 5" },
			"already offered by a built-in engine",
		},
	}
	for name, tc := range cases {
		files := acmeSample()
		tc.mutate(files)
		discovery := discoverFrom(writeOrcaTree(t, "Acme", files))
		if len(discovery.Targets) != 0 {
			t.Fatalf("%s: unsafe machine offered as target: %+v", name, discovery.Targets)
		}
		if len(discovery.Rejected) != 1 || !strings.Contains(discovery.Rejected[0].Reason, tc.reason) {
			t.Fatalf("%s: expected rejection containing %q, got %+v", name, tc.reason, discovery.Rejected)
		}
	}
}

func TestBambuVendorIsNeverOfferedAsOrcaTarget(t *testing.T) {
	discovery := discoverFrom(writeOrcaTree(t, "BBL", acmeSample()))
	if len(discovery.Targets) != 0 || len(discovery.Rejected) != 0 {
		t.Fatalf("Bambu vendor must stay source-only: %+v", discovery)
	}
}

func TestMalformedOrUnknownIdentifiersFailClosed(t *testing.T) {
	locations := []slicerLocation{{id: "test", name: "Test Orca", root: writeOrcaTree(t, "Acme", acmeSample())}}
	for _, id := range []string{"", "u1", "qidi", "../../etc/passwd", "orca-acme-mini-0-4-nozzle-XYZ", "orca-acme-012345678"} {
		if IsOrcaTargetID(id) {
			t.Fatalf("%q must not be a valid installed identifier", id)
		}
		if _, err := resolveOrca(locations, id); err == nil {
			t.Fatalf("%q resolved to a bundle", id)
		}
	}
	if _, err := resolveOrca(locations, "orca-acme-mini-0-4-nozzle-0123456789"); err == nil {
		t.Fatal("well-formed but uninstalled identifier must not resolve")
	}
}

func TestOrcaPrefixIsFilesystemSafeAndStable(t *testing.T) {
	got := OrcaPrefix("orca-acme-mini-0-4-nozzle-0123456789")
	if got != "ORCA-ACME-MINI-0-4-NOZZLE" {
		t.Fatalf("prefix %q", got)
	}
	long := OrcaPrefix("orca-" + strings.Repeat("abc-", 20) + "0123456789")
	if len(long) > len("ORCA-")+24 || strings.ContainsAny(long, "abcdefghijklmnopqrstuvwxyz/\\:") {
		t.Fatalf("prefix not bounded or safe: %q", long)
	}
}
