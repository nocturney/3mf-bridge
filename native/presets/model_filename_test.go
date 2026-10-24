package presets

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestModelFilenamePrefixUsesExactModelRatherThanSlicerBrand(t *testing.T) {
	tests := []struct{ model, vendor, want string }{
		{"Snapmaker U1", "Snapmaker", "U1"},
		{"Bambu Lab H2D", "BBL", "H2D"},
		{"Creality CR-10", "Creality", "CR10"},
		{"Elegoo Neptune 4 Pro", "Elegoo", "Neptune4Pro"},
		{"Flashforge Creator 5 Pro", "Flashforge", "Creator5Pro"},
		{"Flashforge Creator 5", "Flashforge", "Creator5"},
		{"Anycubic Kobra 3", "Anycubic", "Kobra3"},
		{"X-Plus 5", "X 5 Series", "XPlus5"},
		{"QIDI Plus5", "QIDITechnology", "Plus5"},
		{"Acme Mini V2", "Acme", "MiniV2"},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := ModelFilenamePrefix(tt.model, tt.vendor); got != tt.want {
				t.Fatalf("ModelFilenamePrefix(%q,%q) = %q; want %q", tt.model, tt.vendor, got, tt.want)
			}
		})
	}
	if ModelFilenamePrefix("Flashforge Creator 5", "Flashforge") ==
		ModelFilenamePrefix("Flashforge Creator 5 Pro", "Flashforge") {
		t.Fatal("Creator 5 and Creator 5 Pro cannot share an output filename prefix")
	}
}

func TestModelFilenamePrefixNeverLeaksPathsOrUnboundedNames(t *testing.T) {
	for _, raw := range []string{"../../secrets\\data:3", "🚫🚫", strings.Repeat("LongMachineName", 20)} {
		got := ModelFilenamePrefix(raw, "Acme")
		if got == "" || len(got) > 48 || strings.ContainsAny(got, "/\\.:<>|?* ") {
			t.Fatalf("unsafe model filename prefix from %q: %q", raw, got)
		}
		if got != ModelFilenamePrefix(raw, "Acme") {
			t.Fatalf("unstable prefix %q", got)
		}
	}
}

func TestArtworkCandidateRootsAcrossInstalledSlicers(t *testing.T) {
	programFiles := filepath.Join(t.TempDir(), "Program Files")
	custom := t.TempDir()
	locations := artworkRootsForOS("windows", programFiles, "", "", "", []string{custom})
	roots := make([]string, len(locations))
	for i, loc := range locations {
		roots[i] = loc.root
	}
	mustFind := func(root string) int {
		t.Helper()
		for i, got := range roots {
			if got == root {
				return i
			}
		}
		t.Fatalf("missing allowed artwork root %q in %v", root, roots)
		return -1
	}
	if mustFind(custom) != 0 {
		t.Fatal("explicit installed target artwork must take precedence")
	}
	orca := mustFind(filepath.Join(programFiles, "OrcaSlicer", "resources", "profiles"))
	bambu := mustFind(filepath.Join(programFiles, "Bambu Studio", "resources", "profiles"))
	prusa := mustFind(filepath.Join(programFiles, "Prusa3D", "PrusaSlicer", "resources", "profiles"))
	if !(orca < bambu && bambu < prusa) {
		t.Fatalf("unexpected generic artwork fallback order: Orca=%d Bambu=%d Prusa=%d", orca, bambu, prusa)
	}
	seen := map[string]bool{}
	for _, root := range roots {
		if seen[root] {
			t.Fatalf("duplicate allowed artwork root: %q", root)
		}
		seen[root] = true
	}
	macs := artworkRootsForOS("darwin", "", "", "", "/Users/qa", nil)
	macFound := false
	for _, loc := range macs {
		if loc.root == "/Applications/BambuStudio.app/Contents/Resources/profiles" {
			macFound = true
		}
	}
	if !macFound {
		t.Fatal("macOS Bambu Studio local artwork fallback missing")
	}
}
