package presets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// writeManufacturerFile creates a fixture under a temp root. Tests never read
// or change real slicer installs or profiles.
func writeManufacturerFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}

func itemsByID(items []ManufacturerSlicer) map[string]ManufacturerSlicer {
	byID := map[string]ManufacturerSlicer{}
	for _, item := range items {
		byID[item.ID] = item
	}
	return byID
}

func candidatePath(c manufacturerCandidate) string {
	return filepath.Join(append([]string{c.base}, c.app...)...)
}

func TestManufacturerNamesMatchEmbeddedCatalogue(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "frontend", "dist", "printer-catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	var catalogue struct {
		Models []struct {
			Manufacturer string `json:"manufacturer"`
		} `json:"models"`
	}
	if err := json.Unmarshal(raw, &catalogue); err != nil {
		t.Fatal(err)
	}
	embedded := map[string]bool{}
	for _, model := range catalogue.Models {
		embedded[model.Manufacturer] = true
	}
	if len(embedded) != 84 || len(catalogManufacturerNames) != 84 {
		t.Fatalf("want 84 catalogue manufacturers, embedded %d, list %d", len(embedded), len(catalogManufacturerNames))
	}
	for _, name := range catalogManufacturerNames {
		if !embedded[name] {
			t.Fatalf("%q is not an exact embedded catalogue manufacturer", name)
		}
	}
	if len(catalogManufacturerKeys) != len(catalogManufacturerNames) {
		t.Fatal("two catalogue manufacturers fold to the same key")
	}
	for _, spec := range manufacturerSlicerSpecs {
		for _, name := range spec.manufacturers {
			if !embedded[name] {
				t.Fatalf("%s declares %q, which is not a catalogue manufacturer", spec.id, name)
			}
		}
	}
}

func TestCriticalManufacturerAliasesResolveToExactCatalogueNames(t *testing.T) {
	cases := map[string]string{
		"prusaresearch": "Prusa Research",
		"prusa":         "Prusa",
		"elegoo":        "ELEGOO",
		"qidi":          "QIDI",
		"flsun":         "FLSun",
		"flashforge":    "Flashforge",
		"snapmaker":     "Snapmaker",
		"ratrig":        "RatRig",
	}
	for key, want := range cases {
		if got := catalogManufacturerKeys[key]; got != want {
			t.Fatalf("key %q resolved to %q, want %q", key, got, want)
		}
	}
	if catalogKey("PrusaResearch") != "prusaresearch" {
		t.Fatal("folder names must fold to letters and digits only")
	}
	// Bambu folders are excluded from manufacturer lists even though they map.
	if !excludedVendors["Bambu Lab"] {
		t.Fatal("Bambu Lab must stay excluded from generic Orca manufacturers")
	}
}

func TestManufacturerSlicerCoverageAndStableIDs(t *testing.T) {
	want := []string{"prusa-slicer", "bambu-studio", "orca-slicer", "elegoo-slicer", "qidi-studio", "snapmaker-orca", "flash-studio"}
	items := detectManufacturerSlicers("plan9", hostManufacturerFS{}, "", "", "", "", "")
	if len(items) != len(want) {
		t.Fatalf("coverage changed: want %d slicers, got %d", len(want), len(items))
	}
	idPattern := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	seen := map[string]bool{}
	for i, item := range items {
		if item.ID != want[i] {
			t.Fatalf("item %d ID %q, want stable ID %q", i, item.ID, want[i])
		}
		if !idPattern.MatchString(item.ID) || seen[item.ID] {
			t.Fatalf("invalid or duplicate ID %q", item.ID)
		}
		seen[item.ID] = true
		if item.Installed || item.Evidence != "" || item.Manufacturers == nil {
			t.Fatalf("empty roots must report nothing installed: %+v", item)
		}
	}
}

func TestManufacturerSlicerJSONShape(t *testing.T) {
	items := detectManufacturerSlicers("plan9", hostManufacturerFS{}, "", "", "", "", "")
	raw, err := json.Marshal(items)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, item := range decoded {
		for _, key := range []string{"id", "name", "manufacturers", "installed", "evidence", "sourceOnly"} {
			if _, ok := item[key]; !ok {
				t.Fatalf("missing JSON field %q in %v", key, item)
			}
		}
		if _, ok := item["manufacturers"].([]any); !ok {
			t.Fatalf("manufacturers must be an array, got %v", item["manufacturers"])
		}
	}
	if strings.Contains(string(raw), "null") {
		t.Fatalf("payload must not contain null fields: %s", raw)
	}
}

func TestLiveDetectionReturnsEverySupportedSlicer(t *testing.T) {
	// Shape only: the host may or may not have these installed.
	items := DetectManufacturerSlicers()
	if len(items) != len(manufacturerSlicerSpecs) {
		t.Fatalf("want %d items, got %d", len(manufacturerSlicerSpecs), len(items))
	}
}

func TestWindowsCandidatesUseProgramFilesUserProgramsAndExplicitPrusaFolder(t *testing.T) {
	pf := filepath.Join("fake", "Program Files")
	x86 := filepath.Join("fake", "Program Files (x86)")
	local := filepath.Join("fake", "Local")
	candidates := manufacturerCandidates("windows", pf, x86, local, "", "")
	want := filepath.Join(local, "Programs", "Prusa3D", "PrusaSlicer", "prusa-slicer.exe")
	found := false
	for _, c := range candidates {
		if c.base == "" {
			t.Fatal("an empty root must never become a candidate")
		}
		if manufacturerSlicerSpecs[c.index].id == "prusa-slicer" && candidatePath(c) == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("LocalAppData\\Programs PrusaSlicer candidate missing (want %s)", want)
	}
	if got := manufacturerCandidates("windows", "", "", "", "", ""); len(got) != 0 {
		t.Fatalf("empty Windows roots produced candidates: %+v", got)
	}
}

func TestMacCandidatesIncludeApplicationsAndUserApplications(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "tmp")
	apps := filepath.Join(root, "Applications")
	userApps := filepath.Join(root, "Users", "tester", "Applications")
	candidates := manufacturerCandidates("darwin", "", "", "", apps, userApps)
	wantPlist := filepath.Join(userApps, "OrcaSlicer.app", "Contents", "Info.plist")
	wantProfiles := filepath.Join(userApps, "OrcaSlicer.app", "Contents", "Resources", "profiles")
	found := false
	for _, c := range candidates {
		if len(c.app) != 3 {
			t.Fatalf("macOS candidates must check Info.plist inside a bundle: %+v", c)
		}
		if candidatePath(c) == wantPlist && filepath.Join(c.base, filepath.Join(c.profiles...)) == wantProfiles {
			found = true
		}
	}
	if !found {
		t.Fatal("user-level OrcaSlicer bundle candidate missing")
	}
	if got := manufacturerCandidates("plan9", "", "", "", apps, userApps); len(got) != 0 {
		t.Fatalf("unsupported platforms must not produce candidates: %+v", got)
	}
}

func TestWindowsDetectionFindsRealExecutablesAndVendorProfiles(t *testing.T) {
	root := t.TempDir()
	pf := filepath.Join(root, "Program Files")
	x86 := filepath.Join(root, "Program Files (x86)")
	local := filepath.Join(root, "Local")
	writeManufacturerFile(t, filepath.Join(pf, "OrcaSlicer", "orca-slicer.exe"), "MZ")
	writeManufacturerFile(t, filepath.Join(pf, "OrcaSlicer", "resources", "profiles", "Creality", "machine", "Ender.json"), "{}")
	writeManufacturerFile(t, filepath.Join(pf, "OrcaSlicer", "resources", "profiles", "PrusaResearch", "machine", "MK4.json"), "{}")
	writeManufacturerFile(t, filepath.Join(pf, "OrcaSlicer", "resources", "profiles", "BBL", "machine", "X1.json"), "{}")
	writeManufacturerFile(t, filepath.Join(local, "Programs", "Prusa3D", "PrusaSlicer", "prusa-slicer.exe"), "MZ")
	writeManufacturerFile(t, filepath.Join(pf, "Bambu Studio", "bambu-studio.exe"), "MZ")
	byID := itemsByID(detectManufacturerSlicers("windows", hostManufacturerFS{}, pf, x86, local, "", ""))

	orca := byID["orca-slicer"]
	if !orca.Installed || orca.Evidence != manufacturerEvidenceApplication {
		t.Fatalf("OrcaSlicer exe in Program Files not detected: %+v", orca)
	}
	if got := strings.Join(orca.Manufacturers, "|"); got != "Creality|Prusa Research" {
		t.Fatalf("generic Orca must list only local vendors (BBL excluded): %q", got)
	}
	prusa := byID["prusa-slicer"]
	if !prusa.Installed || prusa.Evidence != manufacturerEvidenceApplication || strings.Join(prusa.Manufacturers, "|") != "Prusa Research|Prusa" {
		t.Fatalf("per-user PrusaSlicer not detected: %+v", prusa)
	}
	bambu := byID["bambu-studio"]
	if !bambu.Installed || !bambu.SourceOnly {
		t.Fatalf("Bambu Studio must be detected as source-only: %+v", bambu)
	}
	if elegoo := byID["elegoo-slicer"]; elegoo.Installed || elegoo.Evidence != "" {
		t.Fatalf("absent slicer must not be installed: %+v", elegoo)
	}
}

func TestProfileOnlyInstallReportsProfilesEvidence(t *testing.T) {
	root := t.TempDir()
	pf := filepath.Join(root, "Program Files")
	writeManufacturerFile(t, filepath.Join(pf, "ElegooSlicer", "resources", "profiles", "ELEGOO", "machine", "CC2.json"), "{}")
	elegoo := itemsByID(detectManufacturerSlicers("windows", hostManufacturerFS{}, pf, "", "", "", ""))["elegoo-slicer"]
	if !elegoo.Installed || elegoo.Evidence != manufacturerEvidenceProfiles {
		t.Fatalf("profile-only install should report profiles evidence: %+v", elegoo)
	}
	if strings.Join(elegoo.Manufacturers, "|") != "ELEGOO" {
		t.Fatalf("declared manufacturer missing: %+v", elegoo.Manufacturers)
	}
}

func TestGenericOrcaWithoutLocalProfilesNamesNoManufacturer(t *testing.T) {
	root := t.TempDir()
	pf := filepath.Join(root, "Program Files")
	writeManufacturerFile(t, filepath.Join(pf, "OrcaSlicer", "orca-slicer.exe"), "MZ")
	orca := itemsByID(detectManufacturerSlicers("windows", hostManufacturerFS{}, pf, "", "", "", ""))["orca-slicer"]
	if !orca.Installed || len(orca.Manufacturers) != 0 {
		t.Fatalf("generic Orca with no profiles must not name a manufacturer: %+v", orca)
	}
}

func TestIncorrectBinariesDoNotPass(t *testing.T) {
	root := t.TempDir()
	pf := filepath.Join(root, "Program Files")
	// A directory where the executable should be.
	if err := os.MkdirAll(filepath.Join(pf, "Bambu Studio", "bambu-studio.exe"), 0700); err != nil {
		t.Fatal(err)
	}
	// A zero-byte executable.
	writeManufacturerFile(t, filepath.Join(pf, "QIDIStudio", "QIDIStudio.exe"), "")
	byID := itemsByID(detectManufacturerSlicers("windows", hostManufacturerFS{}, pf, "", "", "", ""))
	if byID["bambu-studio"].Installed {
		t.Fatal("a directory named like an executable must not count")
	}
	if byID["qidi-studio"].Installed {
		t.Fatal("a zero-byte executable must not count")
	}
}

func TestSymlinksCannotRedirectDetection(t *testing.T) {
	root := t.TempDir()
	pf := filepath.Join(root, "Program Files")
	outside := filepath.Join(root, "outside")
	writeManufacturerFile(t, filepath.Join(outside, "snapmaker-orca.exe"), "MZ")
	writeManufacturerFile(t, filepath.Join(outside, "elegoo-slicer.exe"), "MZ")
	if err := os.MkdirAll(pf, 0700); err != nil {
		t.Fatal(err)
	}
	// A linked install folder that points outside the trusted root.
	if err := os.Symlink(outside, filepath.Join(pf, "Snapmaker_Orca")); err != nil {
		t.Skipf("symlinks unavailable here: %v", err)
	}
	// A linked executable file inside a real folder.
	if err := os.MkdirAll(filepath.Join(pf, "ElegooSlicer"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "elegoo-slicer.exe"), filepath.Join(pf, "ElegooSlicer", "elegoo-slicer.exe")); err != nil {
		t.Fatal(err)
	}
	byID := itemsByID(detectManufacturerSlicers("windows", hostManufacturerFS{}, pf, "", "", "", ""))
	if byID["snapmaker-orca"].Installed {
		t.Fatal("symlinked install folder escaped the trusted root")
	}
	if byID["elegoo-slicer"].Installed {
		t.Fatal("symlinked executable must not count")
	}
}

func TestMacBundlesNeedInfoPlistAndUserApplicationsCount(t *testing.T) {
	root := t.TempDir()
	apps := filepath.Join(root, "Applications")
	userApps := filepath.Join(root, "Home", "Applications")
	writeManufacturerFile(t, filepath.Join(apps, "PrusaSlicer.app", "Contents", "Info.plist"), "<plist/>")
	writeManufacturerFile(t, filepath.Join(apps, "OrcaSlicer.app", "Contents", "Info.plist"), "<plist/>")
	writeManufacturerFile(t, filepath.Join(apps, "OrcaSlicer.app", "Contents", "Resources", "profiles", "Anker", "machine", "A1.json"), "{}")
	writeManufacturerFile(t, filepath.Join(apps, "ElegooSlicer.app", "Contents", "MacOS", "ElegooSlicer"), "bin")
	writeManufacturerFile(t, filepath.Join(userApps, "Flash Studio Desktop.app", "Contents", "Info.plist"), "<plist/>")
	byID := itemsByID(detectManufacturerSlicers("darwin", hostManufacturerFS{}, "", "", "", apps, userApps))
	if p := byID["prusa-slicer"]; !p.Installed || p.Evidence != manufacturerEvidenceApplication {
		t.Fatalf("PrusaSlicer bundle not detected: %+v", p)
	}
	if o := byID["orca-slicer"]; !o.Installed || strings.Join(o.Manufacturers, "|") != "Anker" {
		t.Fatalf("OrcaSlicer bundle with local Anker profile not detected: %+v", o)
	}
	if e := byID["elegoo-slicer"]; e.Installed {
		t.Fatal("a bundle without Info.plist must not count")
	}
	if f := byID["flash-studio"]; !f.Installed {
		t.Fatalf("user-level Flash Studio Desktop bundle not detected: %+v", f)
	}
}
