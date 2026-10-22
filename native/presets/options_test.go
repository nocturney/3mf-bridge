package presets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSlicerLocationsWindowsIncludeUserInstalls(t *testing.T) {
	pf := filepath.Join("C:", "Program Files")
	x86 := filepath.Join("C:", "Program Files (x86)")
	local := filepath.Join("C:", "Users", "Tester", "AppData", "Local")
	got := slicerLocationsForOS("qidi", "windows", pf, x86, local, "", "")
	if len(got) != 6 {
		t.Fatalf("want both QIDI slicers at three install roots, got %#v", got)
	}
	// The vendor app is checked in every base before generic Orca.
	expected := []struct{ id, root string }{
		{"qidi-studio", filepath.Join(pf, "QIDIStudio", "resources", "profiles")},
		{"qidi-studio", filepath.Join(x86, "QIDIStudio", "resources", "profiles")},
		{"qidi-studio", filepath.Join(local, "Programs", "QIDIStudio", "resources", "profiles")},
		{"orca-slicer", filepath.Join(pf, "OrcaSlicer", "resources", "profiles")},
		{"orca-slicer", filepath.Join(x86, "OrcaSlicer", "resources", "profiles")},
		{"orca-slicer", filepath.Join(local, "Programs", "OrcaSlicer", "resources", "profiles")},
	}
	for i, want := range expected {
		if got[i].id != want.id || got[i].root != want.root {
			t.Fatalf("Windows candidate %d: got %+v, want %+v", i, got[i], want)
		}
	}
	if len(slicerLocationsForOS("u1", "windows", "", "", "", "", "")) != 0 {
		t.Fatal("empty system directories must not become relative search paths")
	}
}

func TestSlicerLocationsMacIncludeUserApplications(t *testing.T) {
	home := filepath.Join(string(filepath.Separator), "Users", "tester")
	got := slicerLocationsForOS("qidi", "darwin", "", "", "", home, "")
	if len(got) != 4 {
		t.Fatalf("want QIDIStudio and OrcaSlicer in both application directories, got %#v", got)
	}
	want := filepath.Join(home, "Applications", "OrcaSlicer.app")
	if got[3].executable != want || got[3].root != filepath.Join(want, "Contents", "Resources", "profiles") {
		t.Fatalf("user-local Mac app not considered: %+v", got[3])
	}
	flash := slicerLocationsForOS("c5", "darwin", "", "", "", home, "")
	if len(flash) != 4 || flash[1].executable != filepath.Join("/Applications", "Flash Studio Desktop.app") {
		t.Fatalf("Flash Studio naming alternatives unavailable: %#v", flash)
	}
}

func TestExplicitSlicerOverrideIsLocalAndSingleChoice(t *testing.T) {
	root := filepath.Join(t.TempDir(), "resources", "profiles")
	t.Setenv("BRIDGE_SLICER_PROFILE_QIDI", root)
	got := Candidates("qidi")
	if len(got) != 1 || got[0] != root {
		t.Fatalf("override should not search unrelated app locations: %#v", got)
	}
	opts := Slicers("qidi")
	if len(opts) != 1 || opts[0].ID != "system" || opts[0].Installed {
		t.Fatalf("missing vendor folder must not claim to be installed: %#v", opts)
	}
	if err := os.MkdirAll(filepath.Join(root, Specs["qidi"].Vendor), 0700); err != nil {
		t.Fatal(err)
	}
	opts = Slicers("qidi")
	if len(opts) != 1 || !opts[0].Installed {
		t.Fatalf("valid local installed-profile folder not discovered: %#v", opts)
	}
	resolved, executable, _, err := ResolveSlicer("qidi", "system")
	if err != nil || resolved != root || executable != "" {
		t.Fatalf("local override should not expose arbitrary executable: %q %q %v", resolved, executable, err)
	}
}

// writeVendorFolder marks a profile root as an installed app with the QIDI vendor profiles.
func writeVendorFolder(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, Specs["qidi"].Vendor), 0700); err != nil {
		t.Fatal(err)
	}
}

func TestWindowsQIDIAutoPrefersVendorAppAcrossBases(t *testing.T) {
	pf := t.TempDir()
	local := t.TempDir()
	orcaRoot := filepath.Join(pf, "OrcaSlicer", "resources", "profiles")
	qidiRoot := filepath.Join(local, "Programs", "QIDIStudio", "resources", "profiles")
	writeVendorFolder(t, orcaRoot)
	writeVendorFolder(t, qidiRoot)
	locations := slicerLocationsForOS("qidi", "windows", pf, "", local, "", "")

	// Generic Orca sits in the higher-priority Program Files base, yet QIDIStudio wins.
	root, _, name, err := resolveSlicerAt(locations, Specs["qidi"], "auto")
	if err != nil || root != qidiRoot || name != "QIDIStudio" {
		t.Fatalf("auto should pick QIDIStudio from LocalAppData: root=%q name=%q err=%v", root, name, err)
	}
	// Fallback: without the vendor app, generic Orca is used.
	if err := os.RemoveAll(filepath.Join(local, "Programs", "QIDIStudio")); err != nil {
		t.Fatal(err)
	}
	root, _, name, err = resolveSlicerAt(locations, Specs["qidi"], "auto")
	if err != nil || root != orcaRoot || name != "OrcaSlicer" {
		t.Fatalf("auto should fall back to OrcaSlicer: root=%q name=%q err=%v", root, name, err)
	}
	// An explicit preference is honoured, and fails closed when that app has no profile.
	if root, _, _, err := resolveSlicerAt(locations, Specs["qidi"], "orca-slicer"); err != nil || root != orcaRoot {
		t.Fatalf("explicit orca-slicer not honoured: root=%q err=%v", root, err)
	}
	if _, _, _, err := resolveSlicerAt(locations, Specs["qidi"], "qidi-studio"); err == nil {
		t.Fatal("explicit qidi-studio without a profile must fail closed, not fall back to Orca")
	}
}

func TestMacQIDIVendorAppOutranksGenericAcrossApplicationDirs(t *testing.T) {
	home := filepath.Join(string(filepath.Separator), "Users", "tester")
	got := slicerLocationsForOS("qidi", "darwin", "", "", "", home, "")
	wantIDs := []string{"qidi-studio", "qidi-studio", "orca-slicer", "orca-slicer"}
	if len(got) != len(wantIDs) {
		t.Fatalf("want four Mac candidates, got %#v", got)
	}
	for i, id := range wantIDs {
		if got[i].id != id {
			t.Fatalf("Mac candidate %d: got %q, want %q (%+v)", i, got[i].id, id, got[i])
		}
	}
	if want := filepath.Join(home, "Applications", "QIDIStudio.app"); got[1].executable != want {
		t.Fatalf("user-local QIDIStudio should precede generic Orca in /Applications: %+v", got[1])
	}
}

func TestExplicitOverrideIsHonouredAndFailsClosed(t *testing.T) {
	pf := t.TempDir()
	writeVendorFolder(t, filepath.Join(pf, "QIDIStudio", "resources", "profiles"))
	override := filepath.Join(t.TempDir(), "profiles")
	locations := slicerLocationsForOS("qidi", "windows", pf, "", "", "", override)
	if len(locations) != 1 || locations[0].id != "system" || locations[0].root != override {
		t.Fatalf("override must replace every installed app location: %#v", locations)
	}
	if _, _, _, err := resolveSlicerAt(locations, Specs["qidi"], "auto"); err == nil {
		t.Fatal("override without a vendor folder must not fall back to an installed app")
	}
	writeVendorFolder(t, override)
	root, executable, _, err := resolveSlicerAt(locations, Specs["qidi"], "system")
	if err != nil || root != override || executable != "" {
		t.Fatalf("valid override not honoured: root=%q executable=%q err=%v", root, executable, err)
	}
}

func TestMalformedTargetHasNoPathsOrSelection(t *testing.T) {
	pf := t.TempDir()
	for _, target := range []string{"", "QIDI", "../qidi", "qidi/../u1", "unknown"} {
		if got := slicerLocationsForOS(target, "windows", pf, "", "", "", ""); len(got) != 0 {
			t.Fatalf("Windows target %q must yield no paths: %#v", target, got)
		}
		if got := slicerLocationsForOS(target, "darwin", "", "", "", "/Users/tester", ""); len(got) != 0 {
			t.Fatalf("Mac target %q must yield no paths: %#v", target, got)
		}
	}
	if _, _, _, err := ResolveSlicer("../qidi", "auto"); err == nil {
		t.Fatal("malformed target must not resolve to a slicer")
	}
	if IsValidSlicer("../qidi", "qidi-studio") {
		t.Fatal("malformed target must not accept a slicer choice")
	}
}
