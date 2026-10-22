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
	writeOrcaFiles(t, root, vendor, files)
	return root
}

// writeOrcaFiles writes one vendor's files under an existing profiles root.
func writeOrcaFiles(t *testing.T, root, vendor string, files map[string]map[string]any) {
	t.Helper()
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
			"no installed compatible process",
		},
		"process does not list machine": {
			func(f map[string]map[string]any) { f[acmeProcess]["compatible_printers"] = []any{"Other"} },
			"no installed compatible process",
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

// This mirrors the four installed Flash Studio machine variants, including the
// vendor's incorrect AD5M process defaults for 0.4/0.6/0.8. The fixture uses
// standalone mock machine/filament/process data; no vendor files are copied.
func TestFlashforgeCreator5ProRecoversAllFourInstalledNozzleProfiles(t *testing.T) {
	files := map[string]map[string]any{}
	nozzles := []struct {
		diameter, layer, preferred, selected string
	}{
		{"0.25", "0.12", "0.12mm Standard @FF C5 0.25 nozzle", "0.12mm Standard @FF C5 0.25 nozzle"},
		{"0.4", "0.20", "0.20mm Standard @AD5M Pro 0.4", "0.20mm Standard @FF C5"},
		{"0.6", "0.30", "0.20mm Standard @AD5M Pro 0.4", "0.30mm Standard @FF C5 0.6 nozzle"},
		{"0.8", "0.40", "0.20mm Standard @AD5M Pro 0.4", "0.40mm Standard @FF C5 0.8 nozzle"},
	}
	for _, v := range nozzles {
		machine := "Flashforge Creator 5 Pro " + v.diameter + " nozzle"
		files["machine/"+machine+".json"] = map[string]any{
			"name": machine, "printer_model": "Flashforge Creator 5 Pro", "printer_technology": "FFF",
			"nozzle_diameter":     []any{v.diameter, v.diameter, v.diameter, v.diameter},
			"machine_start_gcode": "G28", "machine_end_gcode": "M104 S0",
			"default_print_profile":    v.preferred,
			"default_filament_profile": []any{"Generic PLA @FF C5P " + v.diameter},
		}
		files["process/"+v.selected+".json"] = map[string]any{
			"name": v.selected, "compatible_printers": []any{machine}, "layer_height": v.layer,
			"wall_loops": "3",
		}
		files["filament/Generic PLA @FF C5P "+v.diameter+".json"] = map[string]any{
			"name": "Generic PLA @FF C5P " + v.diameter, "compatible_printers": []any{machine},
			"filament_type": []any{"PLA"}, "nozzle_temperature": []any{"210"},
			"filament_flow_ratio": []any{"0.98"},
		}
	}
	// Known wrong default exists, but belongs to another model and MUST
	// never be used for Creator 5 Pro.
	files["process/0.20mm Standard @AD5M Pro 0.4.json"] = map[string]any{
		"compatible_printers": []any{"Flashforge Adventurer 5M Pro 0.4"},
		"layer_height":        "0.20", "wall_loops": "3",
	}
	root := writeOrcaTree(t, "Flashforge", files)
	discovery := discoverFrom(root)
	if len(discovery.Targets) != 4 || len(discovery.Rejected) != 0 {
		t.Fatalf("expected four valid C5 Pro profiles, got %+v", discovery)
	}
	nozzleToID := map[string]string{}
	for _, target := range discovery.Targets {
		nozzleToID[target.Nozzle] = target.ID
		if target.Model != "Flashforge Creator 5 Pro" || target.PhysicalTools != 4 || target.Title != target.Machine {
			t.Fatalf("C5 Pro model/tool data lost: %+v", target)
		}
		bundle, err := resolveOrca([]slicerLocation{{id: "flash", name: "Flash Studio", root: root}}, target.ID)
		if err != nil {
			t.Fatal(err)
		}
		if bundle.Nozzle != target.Nozzle || bundle.Profile.ProjectSettings["printer_model"] != "Flashforge Creator 5 Pro" {
			t.Fatalf("bad resolved bundle: %+v", bundle)
		}
		if bundle.Profile.ProjectSettings["print_settings_id"] == "0.20mm Standard @AD5M Pro 0.4" {
			t.Fatal("incompatible AD5M default must not be selected")
		}
	}
	for _, v := range nozzles {
		if nozzleToID[v.diameter] == "" {
			t.Fatalf("missing nozzle %s: %+v", v.diameter, discovery.Targets)
		}
		bundle, _ := resolveOrca([]slicerLocation{{id: "flash", name: "Flash Studio", root: root}}, nozzleToID[v.diameter])
		if bundle.Profile.ProjectSettings["print_settings_id"] != v.selected {
			t.Fatalf("nozzle %s: got process %v, want %s", v.diameter, bundle.Profile.ProjectSettings["print_settings_id"], v.selected)
		}
	}
}

func TestOrcaFallbackRequiresExplicitCompatibilityAndSaneLayerHeight(t *testing.T) {
	files := acmeSample()
	files[acmeMachine]["default_print_profile"] = "Nonexistent vendor typo"
	files["process/0.20mm for Other.json"] = map[string]any{
		"compatible_printers": []any{"Other machine"}, "layer_height": "0.2",
	}
	files["process/0.40mm Bad Layer.json"] = map[string]any{
		"compatible_printers": []any{"Acme Mini 0.4 nozzle"}, "layer_height": "0.40",
	}
	files["process/0.20mm Standard Alternate.json"] = map[string]any{
		"compatible_printers": []any{"Acme Mini 0.4 nozzle"}, "layer_height": "0.2",
	}
	d := discoverFrom(writeOrcaTree(t, "Acme", files))
	if len(d.Targets) != 1 || len(d.Rejected) != 0 {
		t.Fatalf("valid fallback dropped: %+v", d)
	}
	// Remove both compatible profiles: an incompatible process must never
	// silently stand in for another printer.
	delete(files, "process/0.20mm Standard Alternate.json")
	delete(files, acmeProcess)
	d = discoverFrom(writeOrcaTree(t, "Acme", files))
	if len(d.Targets) != 0 || len(d.Rejected) != 1 {
		t.Fatalf("unsafe fallback enabled target: %+v", d)
	}
	if !strings.Contains(d.Rejected[0].Reason, "no installed compatible process") {
		t.Fatalf("unexpected rejection: %+v", d.Rejected)
	}
}

func TestOrcaProcessSelectionDeterministicWithoutDefault(t *testing.T) {
	files := acmeSample()
	delete(files[acmeMachine], "default_print_profile")
	files["process/0.32mm Draft.json"] = map[string]any{
		"compatible_printers": []any{"Acme Mini 0.4 nozzle"}, "layer_height": "0.32",
	}
	files["process/0.18mm Standard.json"] = map[string]any{
		"compatible_printers": []any{"Acme Mini 0.4 nozzle"}, "layer_height": "0.18",
	}
	d := discoverFrom(writeOrcaTree(t, "Acme", files))
	if len(d.Targets) != 1 {
		t.Fatalf("machine without default but with valid process hidden: %+v", d)
	}
}

func TestSymlinkedVendorAndMachineFilesAreIgnored(t *testing.T) {
	outside := writeOrcaTree(t, "Acme", acmeSample())
	linkedVendor := t.TempDir()
	if err := os.Symlink(filepath.Join(outside, "Acme"), filepath.Join(linkedVendor, "Acme")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if d := discoverFrom(linkedVendor); len(d.Targets) != 0 || len(d.Rejected) != 0 {
		t.Fatalf("symlinked vendor folder must be ignored: %+v", d)
	}

	files := acmeSample()
	delete(files, acmeMachine)
	root := writeOrcaTree(t, "Acme", files)
	machineDir := filepath.Join(root, "Acme", "machine")
	if err := os.MkdirAll(machineDir, 0700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(acmeSample()[acmeMachine])
	if err != nil {
		t.Fatal(err)
	}
	elsewhere := filepath.Join(t.TempDir(), "machine.json")
	if err := os.WriteFile(elsewhere, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(machineDir, "Acme Mini 0.4.json")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if d := discoverFrom(root); len(d.Targets) != 0 || len(d.Rejected) != 0 {
		t.Fatalf("symlinked machine file must not be read: %+v", d)
	}
}

func TestOrcaChainRejectsSymlinkedApplicationFolder(t *testing.T) {
	base := t.TempDir()
	bundle := filepath.Join(base, "real", "Snapmaker Orca.app")
	profiles := filepath.Join(bundle, "Contents", "Resources", "profiles")
	if err := os.MkdirAll(profiles, 0700); err != nil {
		t.Fatal(err)
	}
	if !orcaChainIsReal(bundle, profiles) {
		t.Fatal("real application folder rejected")
	}
	if orcaChainIsReal(filepath.Join(base, "missing.app"), filepath.Join(base, "missing.app", "Contents", "Resources", "profiles")) {
		t.Fatal("missing application folder accepted")
	}
	link := filepath.Join(base, "linked.app")
	if err := os.Symlink(bundle, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if orcaChainIsReal(link, filepath.Join(link, "Contents", "Resources", "profiles")) {
		t.Fatal("symlinked application folder accepted")
	}
}

func TestVendorForkCopyWinsOverGenericOrcaAndBrokenCopyFallsBack(t *testing.T) {
	fork := writeOrcaTree(t, "Snapmaker", acmeSample())
	generic := writeOrcaTree(t, "Snapmaker", acmeSample())
	d := discoverOrca([]slicerLocation{
		{id: "snapmaker-orca", name: "Snapmaker Orca", root: fork},
		{id: "orca-slicer", name: "OrcaSlicer", root: generic},
	})
	if len(d.Targets) != 1 || d.Targets[0].Slicer != "Snapmaker Orca" || len(d.Rejected) != 0 {
		t.Fatalf("vendor fork must win, and the duplicate must not be offered: %+v", d)
	}

	broken := acmeSample()
	delete(broken, acmeFilament)
	brokenRoot := writeOrcaTree(t, "Snapmaker", broken)
	d = discoverOrca([]slicerLocation{
		{id: "snapmaker-orca", name: "Snapmaker Orca", root: brokenRoot},
		{id: "orca-slicer", name: "OrcaSlicer", root: generic},
	})
	if len(d.Targets) != 1 || d.Targets[0].Slicer != "OrcaSlicer" {
		t.Fatalf("incomplete vendor copy must fall back to generic OrcaSlicer: %+v", d)
	}
	if len(d.Rejected) != 1 || d.Rejected[0].Slicer != "Snapmaker Orca" || !strings.Contains(d.Rejected[0].Reason, "no PLA filament") {
		t.Fatalf("broken vendor copy should be reported: %+v", d.Rejected)
	}
}

func TestBrandedOrcaForksOnlyContributeTheirOwnManufacturer(t *testing.T) {
	// Real branded Orca forks ship whole upstream vendor libraries. Do not
	// offer another maker's profiles as a fork-owned destination just because
	// compatible JSON happens to be bundled beside the manufacturer's files.
	fork := t.TempDir()
	writeOrcaFiles(t, fork, "Snapmaker", acmeSample())
	writeOrcaFiles(t, fork, "Anker", acmeSample())
	generic := writeOrcaTree(t, "Anker", acmeSample())
	locations := []slicerLocation{
		{id: "snapmaker-orca", name: "Snapmaker Orca", root: fork},
		{id: "orca-slicer", name: "OrcaSlicer", root: generic},
	}
	discovery := discoverOrca(locations)
	if len(discovery.Targets) != 2 || len(discovery.Rejected) != 0 {
		t.Fatalf("only owner Snapmaker and generic Orca Anker should be discovered: %+v", discovery)
	}
	owners := map[string]string{}
	for _, target := range discovery.Targets {
		owners[target.Vendor] = target.Slicer
	}
	if owners["Snapmaker"] != "Snapmaker Orca" || owners["Anker"] != "OrcaSlicer" {
		t.Fatalf("unrelated vendor profile leaked through branded fork: %+v", owners)
	}
	// An old ID that pointed into an unrelated fork must never re-resolve.
	oldForeignID := orcaTargetID("Anker", "Acme Mini 0.4 nozzle",
		filepath.Join(fork, "Anker", acmeMachine))
	if _, err := resolveOrca(locations, oldForeignID); err == nil {
		t.Fatal("previous cross-vendor fork target must not be resolved after filtering")
	}
}

func TestOrcaForkOwnershipWhitelistDoesNotFilterGenericOrExplicitProfiles(t *testing.T) {
	for _, tc := range []struct {
		app, vendor string
		allowed     bool
	}{
		{"snapmaker-orca", "Snapmaker", true},
		{"snapmaker-orca", "Anker", false},
		{"elegoo-slicer", "ELEGOO", true},
		{"elegoo-slicer", "Flashforge", false},
		{"flash-studio", "Flashforge", true},
		{"flash-studio", "Anker", false},
		{"qidi-studio", "QIDI", true},
		{"qidi-studio", "X 5 Series", true},
		{"qidi-studio", "Anker", false},
		{"orca-slicer", "Anker", true},
		{"system", "Anker", true},
	} {
		if got := orcaVendorOwnedByApp(tc.app, tc.vendor); got != tc.allowed {
			t.Fatalf("app=%q vendor=%q: allowed=%v want %v", tc.app, tc.vendor, got, tc.allowed)
		}
	}
}

func TestAbstractMachinePresetsAreSkippedSilently(t *testing.T) {
	files := acmeSample()
	files["machine/fdm_acme_common.json"] = map[string]any{
		"name": "fdm_acme_common", "instantiation": "false", "nozzle_diameter": []any{"0.4"},
	}
	d := discoverFrom(writeOrcaTree(t, "Acme", files))
	if len(d.Targets) != 1 || len(d.Rejected) != 0 {
		t.Fatalf("inheritance-only preset must not be a target or a rejection: %+v", d)
	}
}

func TestProcessFilesBeyondScanDepthAreNotIndexed(t *testing.T) {
	files := acmeSample()
	delete(files, acmeProcess)
	files["process/a/b/c/d/0.20mm Acme Standard.json"] = map[string]any{
		"name": "0.20mm Acme Standard", "compatible_printers": []any{"Acme Mini 0.4 nozzle"}, "layer_height": "0.2",
	}
	d := discoverFrom(writeOrcaTree(t, "Acme", files))
	if len(d.Targets) != 0 || len(d.Rejected) != 1 || !strings.Contains(d.Rejected[0].Reason, "no installed compatible process") {
		t.Fatalf("process file below the scan depth must not satisfy the chain: %+v", d)
	}
}

// TestWindowsInstallTreeDiscoversVendorFirstWithoutDuplicates lays out real
// per-user and system install folders, then runs the production inventory.
func TestWindowsInstallTreeDiscoversVendorFirstWithoutDuplicates(t *testing.T) {
	pf := t.TempDir()
	local := t.TempDir()
	forkProfiles := filepath.Join(pf, "Snapmaker_Orca", "resources", "profiles")
	genericProfiles := filepath.Join(local, "Programs", "OrcaSlicer", "resources", "profiles")
	writeOrcaFiles(t, forkProfiles, "Acme", acmeSample())
	writeOrcaFiles(t, genericProfiles, "Acme", acmeSample())
	writeOrcaFiles(t, genericProfiles, "Zed", acmeSample())

	d := discoverOrca(orcaFamilyLocations("windows", pf, "", local, "", nil))
	if len(d.Targets) != 2 || len(d.Rejected) != 0 {
		t.Fatalf("want Acme once and Zed once: %+v", d)
	}
	for _, target := range d.Targets {
		if target.Vendor == "Acme" && target.Slicer != "Snapmaker Orca" {
			t.Fatalf("Acme duplicated from generic OrcaSlicer instead of the vendor fork: %+v", target)
		}
		if target.Vendor == "Zed" && target.Slicer != "OrcaSlicer" {
			t.Fatalf("Zed vendor must come from generic OrcaSlicer: %+v", target)
		}
	}
}

// TestMacInstallTreeDiscoversUserApplications uses a per-user Applications folder,
// so the test never depends on what is installed in the machine's /Applications.
func TestMacInstallTreeDiscoversUserApplications(t *testing.T) {
	home := t.TempDir()
	bundle := filepath.Join(home, "Applications", "OrcaSlicer.app", "Contents", "Resources", "profiles")
	writeOrcaFiles(t, bundle, "Acme", acmeSample())
	var mine []slicerLocation
	for _, loc := range orcaFamilyLocations("darwin", "", "", "", home, nil) {
		if strings.HasPrefix(loc.root, home) {
			mine = append(mine, loc)
		}
	}
	d := discoverOrca(mine)
	if len(d.Targets) != 1 || d.Targets[0].Slicer != "OrcaSlicer" {
		t.Fatalf("user-installed OrcaSlicer not discovered: %+v", d)
	}
}
