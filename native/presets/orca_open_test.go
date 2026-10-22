package presets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A fake application plus deterministic Acme JSON presets exercises executable
// resolution without launching any slicer or reading installed user settings.
func orcaLaunchFixture(t *testing.T, platform, appName, exeName string) (slicerLocation, string) {
	t.Helper()
	base := t.TempDir()
	app := filepath.Join(base, appName)
	exe := filepath.Join(app, exeName)
	root := filepath.Join(app, "resources", "profiles")
	if platform == "darwin" {
		exe = filepath.Join(base, appName+".app")
		root = filepath.Join(exe, "Contents", "Resources", "profiles")
	}
	writeOrcaFiles(t, root, "Acme", acmeSample())
	if platform == "windows" {
		if err := os.WriteFile(exe, []byte("MZ SYNTHETIC TEST ONLY"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	loc := slicerLocation{id: "orca-slicer", name: "OrcaSlicer", root: root, executable: exe}
	targets := discoverOrca([]slicerLocation{loc}).Targets
	if len(targets) != 1 {
		t.Fatalf("expected one completely resolved synthetic target, got %+v", targets)
	}
	return loc, targets[0].ID
}

func TestResolveOrcaExecutableTiedToTheExactInstalledWindowsRoot(t *testing.T) {
	loc, target := orcaLaunchFixture(t, "windows", "OrcaSlicer", "orca-slicer.exe")
	exe, name, err := resolveOrcaExecutableAt([]slicerLocation{loc}, target, "windows")
	if err != nil || exe != loc.executable || name != loc.name {
		t.Fatalf("expected owning app executable %s, got %s / %s (%v)", loc.executable, exe, name, err)
	}
	if _, _, err := resolveOrcaExecutableAt([]slicerLocation{loc}, "orca-unknown-0123456789", "windows"); err == nil {
		t.Fatal("well-formed but unknown profile ID must not select an arbitrary installed slicer")
	}
	if _, _, err := resolveOrcaExecutableAt([]slicerLocation{loc}, "../orca.exe", "windows"); err == nil {
		t.Fatal("untrusted executable path must be rejected as a profile ID")
	}
}

func TestResolveOrcaExecutableUsesQIDIFixedFilenameFallback(t *testing.T) {
	loc, _ := orcaLaunchFixture(t, "windows", "QIDIStudio", "QIDIStudio.exe")
	loc.id, loc.name = "qidi-studio", "QIDIStudio"
	// Model files for a different vendor in QIDIStudio are no longer trusted
	// as QIDI targets. Use the matching synthetic QIDI vendor folder here.
	writeOrcaFiles(t, loc.root, "QIDI", acmeSample())
	targets := discoverOrca([]slicerLocation{loc}).Targets
	if len(targets) != 1 || targets[0].Vendor != "QIDI" {
		t.Fatalf("only the QIDI-owned synthetic machine should resolve: %+v", targets)
	}
	target := targets[0].ID
	if err := os.Remove(loc.executable); err != nil {
		t.Fatal(err)
	}
	alternative := filepath.Join(filepath.Dir(loc.executable), "qidi-studio.exe")
	if err := os.WriteFile(alternative, []byte("MZ TEST"), 0600); err != nil {
		t.Fatal(err)
	}
	exe, name, err := resolveOrcaExecutableAt([]slicerLocation{loc}, target, "windows")
	if err != nil || exe != alternative || name != "QIDIStudio" {
		t.Fatalf("QIDI verified fallback not used: exe=%s name=%s err=%v", exe, name, err)
	}
}

func TestResolveOrcaExecutableRejectsCustomRootAndMissingExecutable(t *testing.T) {
	loc, target := orcaLaunchFixture(t, "windows", "OrcaSlicer", "orca-slicer.exe")
	custom := loc
	custom.executable = ""
	if _, _, err := resolveOrcaExecutableAt([]slicerLocation{custom}, target, "windows"); err == nil ||
		!strings.Contains(err.Error(), "no trusted slicer") {
		t.Fatalf("a profile-only custom root must not invent an executable: %v", err)
	}
	if err := os.Remove(loc.executable); err != nil {
		t.Fatal(err)
	}
	if _, _, err := resolveOrcaExecutableAt([]slicerLocation{loc}, target, "windows"); err == nil {
		t.Fatal("missing executable must fail")
	}
	if err := os.WriteFile(loc.executable, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := resolveOrcaExecutableAt([]slicerLocation{loc}, target, "windows"); err == nil {
		t.Fatal("empty executable must fail")
	}
}

func TestResolveOrcaExecutableRejectsProfileRootOutsideApplication(t *testing.T) {
	loc, target := orcaLaunchFixture(t, "windows", "OrcaSlicer", "orca-slicer.exe")
	moved := loc
	moved.executable = filepath.Join(t.TempDir(), "other.exe")
	if err := os.WriteFile(moved.executable, []byte("MZ TEST"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := resolveOrcaExecutableAt([]slicerLocation{moved}, target, "windows"); err == nil {
		t.Fatal("app executable from a different folder must not open a profile")
	}
}

func TestResolveOrcaExecutableRefusesSymlinkedExecutable(t *testing.T) {
	loc, target := orcaLaunchFixture(t, "windows", "OrcaSlicer", "orca-slicer.exe")
	outside := filepath.Join(t.TempDir(), "outside.exe")
	if err := os.WriteFile(outside, []byte("MZ TEST"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(loc.executable); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, loc.executable); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if _, _, err := resolveOrcaExecutableAt([]slicerLocation{loc}, target, "windows"); err == nil {
		t.Fatal("symlinked executable must never be launched")
	}
}

func TestResolveOrcaExecutableMacBundleAndUnsupportedPlatform(t *testing.T) {
	loc, target := orcaLaunchFixture(t, "darwin", "OrcaSlicer", "")
	exe, name, err := resolveOrcaExecutableAt([]slicerLocation{loc}, target, "darwin")
	if err != nil || exe != loc.executable || name != "OrcaSlicer" {
		t.Fatalf("mac bundle resolution: exe=%s name=%s err=%v", exe, name, err)
	}
	if _, _, err := resolveOrcaExecutableAt([]slicerLocation{loc}, target, "linux"); err == nil {
		t.Fatal("unsupported launch platform must fail closed")
	}
}
