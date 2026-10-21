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
	expected := []struct{ id, root string }{
		{"orca-slicer", filepath.Join(pf, "OrcaSlicer", "resources", "profiles")},
		{"qidi-studio", filepath.Join(pf, "QIDIStudio", "resources", "profiles")},
		{"orca-slicer", filepath.Join(x86, "OrcaSlicer", "resources", "profiles")},
		{"qidi-studio", filepath.Join(x86, "QIDIStudio", "resources", "profiles")},
		{"orca-slicer", filepath.Join(local, "Programs", "OrcaSlicer", "resources", "profiles")},
		{"qidi-studio", filepath.Join(local, "Programs", "QIDIStudio", "resources", "profiles")},
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
