package bridge

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Regression for the dev.24 compatibility path: the unified resolver must
// retain Creality Print's narrowly verified in-memory slot normalization.
// Only synthetic profiles and 3MF fixtures are used, never installed presets.
func TestUnifiedInstalledOrcaCrealityCompatibilityPreservesSlotsOrRefuses(t *testing.T) {
	cases := []struct {
		name   string
		flags  []any
		reject bool
	}{
		{"single support flag expanded", []any{"0"}, false},
		{"ambiguous support flags fail closed", []any{"0", "1"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "source.3mf")
			out := filepath.Join(dir, "result.3mf")
			fixture(t, src, 3)
			setSourceNozzle(t, src, "0.4")
			before, err := os.ReadFile(src)
			if err != nil {
				t.Fatal(err)
			}

			bundle := syntheticOrcaBundle(1)
			bundle.Slicer = "Creality Print"
			bundle.Profile.ModelID = "Creality K2 Plus"
			bundle.Profile.DisplayName = "Creality K2 Plus"
			bundle.Profile.ProjectSettings["printer_model"] = "Creality K2 Plus"
			bundle.Profile.ProjectSettings["filament_is_support"] = tc.flags
			bundle.Profile.ProjectSettings["filament_diameter"] = "1.75"
			_, err = ConvertTarget(src, out, TargetRequest{
				Target: bundle.Target, ProfileFile: writeOrcaBundle(t, bundle),
			})

			if tc.reject {
				if err == nil || ErrorCode(err) != "TARGET_PROFILE_INVALID" {
					t.Fatalf("ambiguous vendor flags must fail closed: %v", err)
				}
				if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
					t.Fatalf("refused conversion wrote output: %v", statErr)
				}
			} else {
				if err != nil {
					t.Fatalf("unified conversion rejected supported flags: %v", err)
				}
				cfg := projectSettings(t, out)
				if !reflect.DeepEqual(cfg["filament_is_support"], []any{"0", "0", "0"}) {
					t.Fatalf("source filament slots lost support flags: %v", cfg["filament_is_support"])
				}
				if !reflect.DeepEqual(cfg["filament_diameter"], []any{"1.75", "1.75", "1.75"}) {
					t.Fatalf("virtual filament diameters not preserved: %v", cfg["filament_diameter"])
				}
				if !reflect.DeepEqual(cfg["nozzle_diameter"], []any{"0.4"}) {
					t.Fatalf("physical nozzle changed: %v", cfg["nozzle_diameter"])
				}
			}

			after, readErr := os.ReadFile(src)
			if readErr != nil || sha256.Sum256(after) != sha256.Sum256(before) {
				t.Fatal("synthetic source 3MF was changed")
			}
		})
	}
}
