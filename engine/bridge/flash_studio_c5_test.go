package bridge

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Flash Studio Desktop on Windows rejected this synthetic four-tool Creator 5
// output with "filament_is_support's count 1 not equal to filament_colour's size 4".
// Every profile is invented; no real vendor preset or user model is modified.
func TestFlashStudioCreator5VendorSupportFlagsAreAligned(t *testing.T) {
	cases := []struct {
		name, slicer string
		existing     any
		hasExisting  bool
		want         []any
		fail         bool
	}{
		{"Mac app missing flags", "Flash Studio", nil, false, []any{"0", "0", "0", "0"}, false},
		{"Windows app missing flags", "Flash Studio Desktop", nil, false, []any{"0", "0", "0", "0"}, false},
		{"one manufacturer zero", "Flash Studio Desktop", []any{"0"}, true, []any{"0", "0", "0", "0"}, false},
		{"complete manufacturer flags retained", "Flash Studio", []any{"0", "1", "0", "1"}, true, []any{"0", "1", "0", "1"}, false},
		{"conflicting partial flags rejected", "Flash Studio Desktop", []any{"0", "1"}, true, nil, true},
		{"single manufacturer one rejected", "Flash Studio", []any{"1"}, true, nil, true},
		{"malformed flags rejected", "Flash Studio Desktop", "0", true, nil, true},
		{"other slicer untouched", "OrcaSlicer", nil, false, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, "source.3mf")
			destination := filepath.Join(dir, "result.3mf")
			original := fixture(t, source, 1)
			bundle := syntheticMaterialBundle()
			bundle.Nozzle = "0.4"
			bundle.Slicer = tc.slicer
			if tc.hasExisting {
				bundle.Profile.ProjectSettings["filament_is_support"] = tc.existing
			}
			bytes, err := json.Marshal(bundle)
			if err != nil {
				t.Fatal(err)
			}
			profile := filepath.Join(dir, "vendor-profile.json")
			if err = os.WriteFile(profile, bytes, 0600); err != nil {
				t.Fatal(err)
			}
			result, err := ConvertTarget(source, destination, TargetRequest{Target: "c5", ProfileFile: profile, Nozzle: "0.4"})
			if tc.fail {
				if err == nil || ErrorCode(err) != "TARGET_PROFILE_INVALID" {
					t.Fatalf("must fail closed, got result=%v err=%v", result, err)
				}
				if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
					t.Fatalf("refusal created output: %v", statErr)
				}
			} else {
				if err != nil {
					t.Fatalf("conversion failed: %v", err)
				}
				cfg := projectSettings(t, destination)
				if tc.want == nil {
					if _, exists := cfg["filament_is_support"]; exists {
						t.Fatalf("other slicer received Flash Studio-only flags: %v", cfg["filament_is_support"])
					}
				} else if !reflect.DeepEqual(cfg["filament_is_support"], tc.want) {
					t.Fatalf("support flags=%v, want %v", cfg["filament_is_support"], tc.want)
				}
				if !reflect.DeepEqual(cfg["nozzle_diameter"], []any{"0.4", "0.4", "0.4", "0.4"}) {
					t.Fatalf("physical nozzles changed: %v", cfg["nozzle_diameter"])
				}
				if colors, ok := cfg["filament_colour"].([]any); !ok || len(colors) != 4 {
					t.Fatalf("four physical slots lost: %v", cfg["filament_colour"])
				}
				if result["target_nozzle"] != "0.4" || result["profile_kind"] != "installed_vendor_json" {
					t.Fatalf("wrong target/nozzle provenance: %v", result)
				}
			}
			unchanged, readErr := os.ReadFile(source)
			if readErr != nil || sha256.Sum256(original) != sha256.Sum256(unchanged) {
				t.Fatal("source changed")
			}
		})
	}
}

// More virtual source colors than physical tools must remain separate in the
// 3MF project while explicitly requiring human mapping before any print.
func TestFlashStudioCreator5FiveVirtualColorsNeverClaimPhysicalReadiness(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.3mf")
	destination := filepath.Join(dir, "five-virtual.3mf")
	original := fixture(t, source, 5)
	bundle := syntheticMaterialBundle()
	bundle.Nozzle = "0.4"
	bundle.Slicer = "Flash Studio Desktop"
	bytes, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	profile := filepath.Join(dir, "profile.json")
	if err = os.WriteFile(profile, bytes, 0600); err != nil {
		t.Fatal(err)
	}
	result, err := ConvertTarget(source, destination, TargetRequest{Target: "c5", ProfileFile: profile, Nozzle: "0.4"})
	if err != nil {
		t.Fatal(err)
	}
	if result["physical_toolheads"] != 4 || result["source_filament_slots"] != 5 ||
		result["virtual_filament_slots"] != 5 || result["user_slot_mapping_required"] != true {
		t.Fatalf("source vs hardware tool mapping is not explicit: %+v", result)
	}
	cfg := projectSettings(t, destination)
	if colors, ok := cfg["filament_colour"].([]any); !ok || len(colors) != 5 {
		t.Fatalf("source palette collapsed: %v", cfg["filament_colour"])
	}
	if flags, ok := cfg["filament_is_support"].([]any); !ok || len(flags) != 5 {
		t.Fatalf("support flags do not cover independent source slots: %v", cfg["filament_is_support"])
	}
	if diameters, ok := cfg["nozzle_diameter"].([]any); !ok || !reflect.DeepEqual(diameters, []any{"0.4", "0.4", "0.4", "0.4"}) {
		t.Fatalf("virtual source channels silently became physical tools: %v", cfg["nozzle_diameter"])
	}
	unchanged, err := os.ReadFile(source)
	if err != nil || sha256.Sum256(original) != sha256.Sum256(unchanged) {
		t.Fatal("source changed")
	}
}

func TestFlashStudioCreator5MalformedPhysicalToolCountIsNotHidden(t *testing.T) {
	for _, n := range []int{0, 1, 2, 3, 5} {
		t.Run(fmt.Sprintf("%d physical nozzle entries", n), func(t *testing.T) {
			bundle := syntheticMaterialBundle()
			bundle.Nozzle = "0.4"
			bundle.Slicer = "Flash Studio Desktop"
			nozzles := make([]any, n)
			for i := range nozzles {
				nozzles[i] = "0.4"
			}
			bundle.Profile.ProjectSettings["nozzle_diameter"] = nozzles
			if err := normalizeFlashStudioCreator5SupportFlags(bundle, 1); err == nil {
				t.Fatalf("malformed %d physical tools accepted", n)
			}
			if _, exists := bundle.Profile.ProjectSettings["filament_is_support"]; exists {
				t.Fatal("malformed machine gained synthetic support flags")
			}
		})
	}
}
