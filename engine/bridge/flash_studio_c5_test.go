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

// On Windows the installed Orca-family app reports "Flash Studio Desktop",
// not the macOS "Flash Studio" label. Previously the Pro-only normalization
// skipped Windows, and its native CLI rejected the 4-tool palette.
func TestWindowsFlashStudioCreator5ProUsesInstalledFourToolFlags(t *testing.T) {
	for _, tc := range []struct {
		name, slicer string
		existing     []any
		want         []any
		fail         bool
	}{
		{"mac app", "Flash Studio", nil, []any{"0", "0", "0", "0"}, false},
		{"Windows app", "Flash Studio Desktop", nil, []any{"0", "0", "0", "0"}, false},
		{"explicit full flags", "Flash Studio Desktop", []any{"0", "1", "0", "1"}, []any{"0", "1", "0", "1"}, false},
		{"ambiguous partial", "Flash Studio Desktop", []any{"0", "1"}, nil, true},
		{"other vendor app unchanged", "OrcaSlicer", nil, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, "three-PLA.3mf")
			original := fixture(t, source, 3)
			out := filepath.Join(dir, "converted.3mf")
			bundle := syntheticOrcaBundle(4)
			bundle.Target = "orca-flashforge-creator-5-pro-0-25-abcdef0123"
			bundle.Nozzle = "0.25"
			bundle.Slicer = tc.slicer
			bundle.Profile.Name = bundle.Target
			bundle.Profile.DisplayName = "Flashforge Creator 5 Pro"
			bundle.Profile.ModelID = "Flashforge Creator 5 Pro"
			bundle.Profile.ProjectSettings["printer_model"] = bundle.Profile.ModelID
			bundle.Profile.ProjectSettings["nozzle_diameter"] = []any{"0.25", "0.25", "0.25", "0.25"}
			if tc.existing != nil {
				bundle.Profile.ProjectSettings["filament_is_support"] = tc.existing
			}
			profileFile := writeOrcaBundle(t, bundle)
			profileBefore, err := os.ReadFile(profileFile)
			if err != nil {
				t.Fatal(err)
			}
			report, err := ConvertTarget(source, out, TargetRequest{Target: bundle.Target, ProfileFile: profileFile, Nozzle: "0.25"})
			if tc.fail {
				if err == nil || ErrorCode(err) != "TARGET_PROFILE_INVALID" {
					t.Fatalf("ambiguous manufacturer flags unexpectedly accepted: %v %v", report, err)
				}
				if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
					t.Fatalf("refusal created output: %v", statErr)
				}
			} else {
				if err != nil {
					t.Fatalf("conversion failed %s: %v", ErrorCode(err), err)
				}
				cfg := projectSettings(t, out)
				if tc.want == nil {
					if _, exists := cfg["filament_is_support"]; exists {
						t.Fatalf("unrelated OrcaSlicer profile altered: %v", cfg["filament_is_support"])
					}
				} else if !reflect.DeepEqual(cfg["filament_is_support"], tc.want) {
					t.Fatalf("support flags=%v want=%v", cfg["filament_is_support"], tc.want)
				}
				if !reflect.DeepEqual(cfg["nozzle_diameter"], []any{"0.25", "0.25", "0.25", "0.25"}) {
					t.Fatalf("physical nozzle array changed: %v", cfg["nozzle_diameter"])
				}
				if got := cfg["filament_colour"].([]any); len(got) != 4 ||
					got[0] != "#FF0000" || got[1] != "#00FF00" || got[2] != "#0000FF" {
					t.Fatalf("independent source colors were collapsed: %v", got)
				}
				if report["physical_toolheads"] != 4 || report["target_nozzle"] != "0.25" ||
					report["source_filament_slots"] != 3 || report["user_slot_mapping_required"] != false {
					t.Fatalf("target report inconsistent: %v", report)
				}
			}
			sourceAfter, err := os.ReadFile(source)
			if err != nil || sha256.Sum256(original) != sha256.Sum256(sourceAfter) {
				t.Fatal("source 3MF changed")
			}
			profileAfter, err := os.ReadFile(profileFile)
			if err != nil || sha256.Sum256(profileBefore) != sha256.Sum256(profileAfter) {
				t.Fatal("installed bundle file was modified")
			}
		})
	}
}
