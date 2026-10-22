package presets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A vendor bundle shaped like the real QIDITechnology.ini: an abstract common
// parent, multi-parent inherits, and compatibility by condition.
const qidiVendorFixture = `# fixture
[vendor]
name = QIDITechnology

[printer:*common*]
bed_shape = 0x0,350x0,350x350,0x350
printer_technology = FFF
printer_model =
nozzle_diameter = 0.4
machine_start_gcode = G28

[printer:*X-Plus 4*]
inherits = *common*
printer_model = X-Plus 4

[printer:X-Plus 4 0.4 nozzle]
inherits = *X-Plus 4*
nozzle_diameter = 0.4

[print:*common*]
layer_height = 0.2
first_layer_height = 0.2

[print:0.20mm Standard @X-Plus 4]
inherits = *common*
compatible_printers_condition = printer_model=="X-Plus 4"

[print:0.12mm Fine @X-Plus 4]
inherits = *common*
layer_height = 0.12
compatible_printers_condition = printer_model=="X-Plus 4" and nozzle_diameter[0]==0.4

[print:0.20mm Other printer]
inherits = *common*
compatible_printers = "Some Other Printer"

[print:0.20mm regex only]
inherits = *common*
compatible_printers_condition = printer_notes=~/.*PRINTER_VENDOR_QIDI.*/

[filament:*common*]
filament_type = PLA

[filament:Generic PLA @X-Plus 4]
inherits = *common*
compatible_printers_condition = printer_model=="X-Plus 4"

[filament:Generic ABS @X-Plus 4]
filament_type = ABS
compatible_printers_condition = printer_model=="X-Plus 4"
`

const userQIDIPrinter = `[printer:QIDI Plus5 0.4 nozzle]
inherits = X-Plus 4 0.4 nozzle
`

// prusaFixture writes a vendor folder and a user config folder into temp dirs.
func prusaFixture(t *testing.T) (prusaLocations, string) {
	t.Helper()
	root := t.TempDir()
	loc := prusaLocations{
		user:   filepath.Join(root, "user"),
		vendor: filepath.Join(root, "vendor"),
	}
	writePrusaFile(t, filepath.Join(loc.vendor, "QIDITechnology.ini"), qidiVendorFixture)
	writePrusaFile(t, filepath.Join(loc.user, "printer", "QIDI Plus5 0.4 nozzle.ini"), userQIDIPrinter)
	return loc, root
}

func writePrusaFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverPrusaSelectsCompletePrinterWithSourcePresets(t *testing.T) {
	loc, root := prusaFixture(t)
	discovery := discoverPrusa(loc)
	if len(discovery.Rejected) != 0 {
		t.Fatalf("unexpected rejections: %+v", discovery.Rejected)
	}
	if len(discovery.Targets) != 1 {
		t.Fatalf("targets = %+v", discovery.Targets)
	}
	target := discovery.Targets[0]
	if target.Title != "QIDI Plus5 0.4 nozzle" || target.Vendor != "QIDITechnology" ||
		target.Model != "X-Plus 4" || target.Nozzle != "0.4" || target.Slicer != "PrusaSlicer" {
		t.Fatalf("unexpected target fields: %+v", target)
	}
	// Two prints are compatible. The one whose layer height is nearest half the
	// nozzle wins, and the Standard PLA filament is picked by name.
	if target.Print != "0.20mm Standard @X-Plus 4" {
		t.Fatalf("print preset = %q", target.Print)
	}
	if target.Filament != "Generic PLA @X-Plus 4" {
		t.Fatalf("filament preset = %q", target.Filament)
	}
	if !IsPrusaTargetID(target.ID) {
		t.Fatalf("ID %q does not match the shape", target.ID)
	}
	encoded, err := json.Marshal(discovery)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), root) || strings.Contains(string(encoded), `"print"`) || strings.Contains(string(encoded), `"filament"`) {
		t.Fatalf("install paths or preset internals leaked into JSON: %s", encoded)
	}
}

func TestPrusaTargetIDsAreStableAndTiedToTheExactPreset(t *testing.T) {
	loc, root := prusaFixture(t)
	first := discoverPrusa(loc).Targets[0].ID
	if again := discoverPrusa(loc).Targets[0].ID; again != first {
		t.Fatalf("ID changed between runs: %s vs %s", first, again)
	}
	other := prusaLocations{user: filepath.Join(root, "elsewhere"), vendor: loc.vendor}
	writePrusaFile(t, filepath.Join(other.user, "printer", "QIDI Plus5 0.4 nozzle.ini"), userQIDIPrinter)
	if moved := discoverPrusa(other).Targets[0].ID; moved == first {
		t.Fatal("same preset name in another folder reused the same ID")
	}
	if !regexp.MustCompile(`^prusa-qidi-plus5-0-4-nozzle-[0-9a-f]{10}$`).MatchString(first) {
		t.Fatalf("unexpected ID shape: %s", first)
	}
}

func TestPrusaRejectsMalformedAndUnsupportedPrinters(t *testing.T) {
	loc, root := prusaFixture(t)
	printers := map[string]string{
		"sla.ini":       "[printer:SLA]\ninherits = X-Plus 4 0.4 nozzle\nprinter_technology = SLA\n",
		"two tools.ini": "[printer:Two tools]\ninherits = X-Plus 4 0.4 nozzle\nnozzle_diameter = 0.4,0.4\n",
		"half.ini":      "[printer:Half]\ninherits = X-Plus 4 0.4 nozzle\nnozzle_diameter = 0.5\n",
		"no bed.ini":    "[printer:No bed]\nprinter_technology = FFF\nbed_shape =\nnozzle_diameter = 0.4\n",
		"broken.ini":    "[printer:Broken]\nnot a setting\n",
		"orphan.ini":    "[printer:Orphan]\ninherits = Missing Parent\n",
		"self loop.ini": "[printer:Loop]\ninherits = Loop\n",
		"lonely.ini":    "[printer:Lonely]\nprinter_technology = FFF\nbed_shape = 0x0,1x0,1x1\nnozzle_diameter = 0.4\nprinter_model = Lonely\n",
		"duplicate.ini": "[printer:Dup]\nnozzle_diameter = 0.4\n[printer:Dup]\n",
	}
	for name, content := range printers {
		writePrusaFile(t, filepath.Join(loc.user, "printer", name), content)
	}
	discovery := discoverPrusa(loc)
	if len(discovery.Targets) != 1 || discovery.Targets[0].Title != "QIDI Plus5 0.4 nozzle" {
		t.Fatalf("a bad printer hid the good one: %+v", discovery.Targets)
	}
	reasons := map[string]string{}
	for _, rejected := range discovery.Rejected {
		reasons[rejected.Machine] = rejected.Reason
	}
	expect := map[string]string{
		"SLA":       "not supported",
		"Two tools": "multi-extruder",
		"Half":      "unsupported nozzle",
		"No bed":    "bed_shape",
		"Orphan":    "not installed",
		"Lonely":    "no compatible print",
		"Loop":      "deeper than",
		"broken":    "malformed",
		"duplicate": "malformed",
	}
	for machine, fragment := range expect {
		if !strings.Contains(reasons[machine], fragment) {
			t.Errorf("%s: reason %q lacks %q", machine, reasons[machine], fragment)
		}
	}
	for _, rejected := range discovery.Rejected {
		if strings.Contains(rejected.Reason, root) {
			t.Errorf("rejection leaks an install path: %q", rejected.Reason)
		}
	}
}

func TestPrusaSkipsSymlinksAndOversizedVendorFiles(t *testing.T) {
	loc, root := prusaFixture(t)
	outside := filepath.Join(root, "outside.ini")
	writePrusaFile(t, outside, userQIDIPrinter)
	link := filepath.Join(loc.user, "printer", "linked.ini")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	writePrusaFile(t, filepath.Join(loc.vendor, "huge.ini"), strings.Repeat("#", maxPrusaFileBytes+1))
	discovery := discoverPrusa(loc)
	if len(discovery.Targets) != 1 {
		t.Fatalf("symlinked or oversized files changed targets: %+v", discovery.Targets)
	}
	found := false
	for _, rejected := range discovery.Rejected {
		if rejected.Machine == "huge" && strings.Contains(rejected.Reason, "cannot be read safely") {
			found = true
		}
	}
	if !found {
		t.Fatalf("oversized vendor file not reported: %+v", discovery.Rejected)
	}
}

func TestPrusaMissingFoldersYieldEmptyLists(t *testing.T) {
	root := t.TempDir()
	discovery := discoverPrusa(prusaLocations{user: filepath.Join(root, "none"), vendor: filepath.Join(root, "none2")})
	encoded, err := json.Marshal(discovery)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"targets":[],"rejected":[]}` {
		t.Fatalf("empty discovery encodes as %s", encoded)
	}
}

func TestPrusaCompatibilityFollowsPrusaSlicerPrecedence(t *testing.T) {
	printer := prusaSection{"printer_model": "X-Plus 4"}
	cases := []struct {
		name   string
		preset prusaSection
		want   bool
	}{
		{"no list and no condition applies everywhere", prusaSection{}, true},
		{"explicit list names this printer", prusaSection{"compatible_printers": `"Mine";"Other"`}, true},
		{"explicit list omits this printer", prusaSection{"compatible_printers": `"Other"`}, false},
		{"explicit list beats a matching condition", prusaSection{"compatible_printers": `"Other"`, "compatible_printers_condition": `printer_model=="X-Plus 4"`}, false},
		{"condition decides when the list is empty", prusaSection{"compatible_printers_condition": `printer_model=="X-Plus 4"`}, true},
	}
	for _, tc := range cases {
		got, err := prusaCompatible(tc.preset, "Mine", printer)
		if err != nil || got != tc.want {
			t.Errorf("%s: got %v, %v; want %v", tc.name, got, err, tc.want)
		}
	}
	if got, err := prusaCompatible(prusaSection{"compatible_printers_condition": `printer_model=~/^X-Plus/`}, "Mine", printer); err != nil || !got {
		t.Errorf("valid regex condition: got %v, %v", got, err)
	}
	if _, err := prusaCompatible(prusaSection{"compatible_printers_condition": `printer_notes=~/[/`}, "Mine", printer); err == nil {
		t.Error("invalid regex condition did not fail closed")
	}
}

func TestResolvePrusaTargetOnlyReturnsCompleteInstalledPresets(t *testing.T) {
	loc, _ := prusaFixture(t)
	id := discoverPrusa(loc).Targets[0].ID
	if target, err := resolvePrusaTarget(loc, id); err != nil || target.ID != id {
		t.Fatalf("installed target did not resolve: %+v, %v", target, err)
	}
	for _, bad := range []string{"prusa-project", "prusa-../x", "PRUSA-QIDI-0123456789", "prusa-qidi-0123456789", ""} {
		if _, err := resolvePrusaTarget(loc, bad); err == nil {
			t.Errorf("%q resolved", bad)
		}
	}
	if _, err := resolvePrusaTarget(loc, "prusa-qidi-plus5-0-4-nozzle-0000000000"); err == nil {
		t.Error("well-formed but uninstalled ID resolved")
	}
}

func TestPrusaSavedSelectionRanksOnlyForItsOwnPrinter(t *testing.T) {
	loc, _ := prusaFixture(t)
	saved := filepath.Join(loc.user, "PrusaSlicer.ini")
	writePrusaFile(t, saved, "[presets]\nprinter = QIDI Plus5 0.4 nozzle\nprint = 0.12mm Fine @X-Plus 4\n")
	if got := discoverPrusa(loc).Targets[0].Print; got != "0.12mm Fine @X-Plus 4" {
		t.Fatalf("saved selection ignored for its own printer: %q", got)
	}
	writePrusaFile(t, saved, "[presets]\nprinter = Someone Else\nprint = 0.12mm Fine @X-Plus 4\n")
	if got := discoverPrusa(loc).Targets[0].Print; got != "0.20mm Standard @X-Plus 4" {
		t.Fatalf("selection for another printer applied: %q", got)
	}
	writePrusaFile(t, saved, "[presets\nbroken\n")
	if targets := discoverPrusa(loc).Targets; len(targets) != 1 {
		t.Fatalf("corrupt PrusaSlicer.ini hid the printer: %+v", targets)
	}
}
