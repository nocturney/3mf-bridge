package presets

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nocturney/slicebridge/engine/bridge"
)

// Manual, opt-in release QA: this test is skipped by normal local/PR test
// runs. It tests an extracted, SHA-verified official Anycubic Linux package,
// never downloads software or writes printer/slicer preferences.
func TestQAAnycubicOfficialLinuxProfiles(t *testing.T) {
	profiles := os.Getenv("QA_ANYCUBIC_OFFICIAL_PROFILES")
	output := os.Getenv("QA_ANYCUBIC_OFFICIAL_OUTPUT")
	if profiles == "" || output == "" {
		t.Skip("requires the official extracted Anycubic release and disposable QA output")
	}
	loc := slicerLocation{id: "anycubic-slicer-next", name: "Anycubic Slicer Next", root: profiles}
	targets, rejected := scanOrcaVendor(loc, "Anycubic")
	if len(targets) == 0 {
		t.Fatal("no eligible Anycubic destinations from official release")
	}
	if err := os.MkdirAll(output, 0700); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join("..", "..", "tests", "fixtures", "bambu-one-color.3mf")
	converted := 0
	kobraVerified := false
	machines := make(map[string]bool)
	seen := make(map[string]bool)
	nozzles := make(map[string]int)
	var failures []string
	for i, target := range targets {
		info := target.target
		if seen[info.ID] || info.Model == "" || info.Nozzle == "" || info.PhysicalTools < 1 || len(info.Materials) == 0 {
			t.Fatalf("invalid/duplicate discovered target %+v", info)
		}
		seen[info.ID] = true
		machines[info.Model] = true
		nozzles[info.Nozzle]++
		// Explicitly select every official target nozzle in this synthetic QA.
		// Only a temporary 3MF is generated; saved user selections are untouched.
		data, err := json.Marshal(target.bundle)
		if err != nil {
			t.Fatal(err)
		}
		bundleFile := filepath.Join(output, fmt.Sprintf("official-target-%03d.json", i))
		if err := os.WriteFile(bundleFile, data, 0600); err != nil {
			t.Fatal(err)
		}
		name := fmt.Sprintf("official-target-%03d.3mf", i)
		if info.Model == "Anycubic Kobra S1" && info.Nozzle == "0.4" {
			name = "Anycubic-Kobra-S1-0.4.3mf"
		}
		dst := filepath.Join(output, name)
		// Remove only the disposable output from a previous QA run; never user files.
		if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if _, err := bridge.ConvertWithOrcaBundle(src, dst, bundleFile); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", info.Machine, err))
			continue
		}
		z, err := zip.OpenReader(dst)
		if err != nil {
			t.Fatal(err)
		}
		var settings map[string]any
		for _, entry := range z.File {
			if entry.Name != "Metadata/project_settings.config" {
				continue
			}
			f, err := entry.Open()
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(io.LimitReader(f, 8<<20))
			f.Close()
			if err != nil || json.Unmarshal(body, &settings) != nil {
				t.Fatalf("invalid converted project settings for %s: %v", info.Machine, err)
			}
			break
		}
		z.Close()
		if settings == nil || settings["printer_model"] != info.Model {
			t.Fatalf("converted model not preserved for %s: %v", info.Machine, settings["printer_model"])
		}
		values, ok := settings["nozzle_diameter"].([]any)
		if !ok || len(values) == 0 {
			t.Fatalf("no target nozzle array for %s: %v", info.Machine, settings["nozzle_diameter"])
		}
		for _, v := range values {
			if fmt.Sprint(v) != info.Nozzle {
				t.Fatalf("converted nozzle mismatch for %s: %v; target %s", info.Machine, values, info.Nozzle)
			}
		}
		if code, _ := settings["machine_start_gcode"].(string); code == "" || strings.Contains(code, "UNSAFE_BAMBU_SCRIPT") {
			t.Fatalf("unsafe/missing target start script for %s", info.Machine)
		}
		converted++
		if info.Model == "Anycubic Kobra S1" && info.Nozzle == "0.4" {
			kobraVerified = true
		}
	}
	t.Logf("ANYCUBIC_RELEASE_MATRIX eligible=%d unique_models=%d rejected_non_targets=%d nozzles=%v converted_all_variants=%d failures=%d",
		len(targets), len(machines), len(rejected), nozzles, converted, len(failures))
	for _, f := range failures {
		t.Logf("PROFILE_CONVERSION_FAILED %s", f)
	}
	if !kobraVerified {
		t.Fatal("official Anycubic Kobra S1 0.4 source conversion did not pass")
	}
	if len(failures) > 0 {
		t.Fatalf("%d official 0.4 profile conversions failed; see report", len(failures))
	}
}
