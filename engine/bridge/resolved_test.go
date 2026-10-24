package bridge

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// All fixtures here are synthetic: generated Bambu-style 3MF files and invented
// machine profiles. No user project or real slicer profile is read.

var fixtureColors = []any{"#FF0000", "#00FF00", "#0000FF", "#FFFF00", "#FFFFFF"}

// setSourceNozzle rewrites the synthetic source so the Bambu project explicitly
// declares its nozzle, as a real exported project does.
func setSourceNozzle(t *testing.T, filename, nozzle string) {
	t.Helper()
	archive, err := zip.OpenReader(filename)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		if file.Name == "Metadata/project_settings.config" {
			cfg := map[string]any{}
			if err := json.Unmarshal(data, &cfg); err != nil {
				t.Fatal(err)
			}
			cfg["nozzle_diameter"] = []any{nozzle}
			if data, err = json.Marshal(cfg); err != nil {
				t.Fatal(err)
			}
		}
		dst, err := zw.Create(file.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := dst.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	archive.Close()
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, out.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
}

// syntheticVendorBundleFile writes a Creator 5 style installed vendor bundle
// whose four physical tools carry the given nozzle.
func syntheticVendorBundleFile(t *testing.T, nozzle string) string {
	t.Helper()
	bundle := syntheticMaterialBundle()
	bundle.Nozzle = nozzle
	bundle.Profile.ProjectSettings["nozzle_diameter"] = []any{nozzle, nozzle, nozzle, nozzle}
	data, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "vendor-profile.json")
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	return file
}

// syntheticOrcaFile writes an installed Orca-family bundle with its own ID,
// model, nozzle and physical tool count.
func syntheticOrcaFile(t *testing.T, id, model, nozzle string, tools int) string {
	t.Helper()
	bundle := syntheticOrcaBundle(tools)
	bundle.Target = id
	bundle.Nozzle = nozzle
	bundle.Profile.Name = id
	bundle.Profile.DisplayName = model
	bundle.Profile.ModelID = model
	bundle.Profile.ProjectSettings["printer_model"] = model
	nozzles := make([]any, tools)
	for i := range nozzles {
		nozzles[i] = nozzle
	}
	bundle.Profile.ProjectSettings["nozzle_diameter"] = nozzles
	return writeOrcaBundle(t, bundle)
}

func projectSettings(t *testing.T, filename string) map[string]any {
	t.Helper()
	cfg := map[string]any{}
	if err := json.Unmarshal([]byte(archiveContents(t, filename)["Metadata/project_settings.config"]), &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

// requireNoOutput proves a refusal happened before anything was published or
// left behind in the output directory.
func requireNoOutput(t *testing.T, dir string, keep ...string) {
	t.Helper()
	allowed := map[string]bool{}
	for _, name := range keep {
		allowed[name] = true
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !allowed[entry.Name()] {
			t.Fatalf("refused conversion left %q behind", entry.Name())
		}
	}
}

func TestMixedBuiltinAndInstalledTargetsShareOneConversionCore(t *testing.T) {
	const (
		orcaOne = "orca-acme-mini-0-4-nozzle-0123456789"
		orcaTwo = "orca-flashforge-creator-5-pro-0-25-nozzle-abcdef0123"
	)
	cases := []struct {
		name   string
		req    TargetRequest
		kind   ProfileKind
		origin NozzleOrigin
		nozzle string
		tools  int
		model  string
	}{
		{"u1 template", TargetRequest{Target: "u1"}, ProfileBuiltinTemplate, NozzleSourceMatched, "0.4", 4, "Snapmaker U1"},
		{"c5 vendor selected 0.25", TargetRequest{Target: "c5", ProfileFile: syntheticVendorBundleFile(t, "0.25"), Nozzle: "0.25"},
			ProfileInstalledVendor, NozzleSelected, "0.25", 4, "Flashforge Creator 5"},
		{"orca single tool", TargetRequest{Target: orcaOne, ProfileFile: syntheticOrcaFile(t, orcaOne, "Acme Mini", "0.4", 1)},
			ProfileInstalledOrca, NozzleFromProfile, "0.4", 1, "Acme Mini"},
		{"orca two tools 0.25", TargetRequest{Target: orcaTwo, ProfileFile: syntheticOrcaFile(t, orcaTwo, "Flashforge Creator 5 Pro", "0.25", 2)},
			ProfileInstalledOrca, NozzleFromProfile, "0.25", 2, "Flashforge Creator 5 Pro"},
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "source.3mf")
	original := fixture(t, source, 5)
	setSourceNozzle(t, source, "0.4")
	original, _ = os.ReadFile(source)
	originalSHA := sha256.Sum256(original)

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := filepath.Join(dir, fmt.Sprintf("out-%d.3mf", i))
			report, err := ConvertTarget(source, out, tc.req)
			if err != nil {
				t.Fatalf("%s: %v", ErrorCode(err), err)
			}
			if report["profile_kind"] != string(tc.kind) || report["target_nozzle_origin"] != string(tc.origin) {
				t.Fatalf("provenance not reported: %v / %v", report["profile_kind"], report["target_nozzle_origin"])
			}
			if report["target"] != tc.req.Target || report["target_nozzle"] != tc.nozzle || report["source_nozzle"] != "0.4" {
				t.Fatalf("identity/nozzle wrong: %+v", report)
			}
			// Physical tools come from the machine; source colors stay virtual slots.
			if report["physical_toolheads"] != tc.tools || report["virtual_filament_slots"] != 5 ||
				report["source_filament_slots"] != 5 || report["user_slot_mapping_required"] != (5 > tc.tools) {
				t.Fatalf("slot/tool accounting wrong: %+v", report)
			}
			mapping := report["source_to_target_slot_map"].(map[int]int)
			for id := 1; id <= 5; id++ {
				if mapping[id] != id {
					t.Fatalf("slot %d remapped: %v", id, mapping)
				}
			}
			cfg := projectSettings(t, out)
			if cfg["printer_model"] != tc.model {
				t.Fatalf("printer identity %v, want %q", cfg["printer_model"], tc.model)
			}
			if !reflect.DeepEqual(cfg["filament_colour"], fixtureColors) {
				t.Fatalf("colors changed: %v", cfg["filament_colour"])
			}
			if types, _ := cfg["filament_type"].([]any); len(types) != 5 {
				t.Fatalf("material list not preserved for 5 slots: %v", cfg["filament_type"])
			}
			if strings.Contains(archiveContents(t, out)["Metadata/project_settings.config"], "UNSAFE_BAMBU_SCRIPT") {
				t.Fatal("source firmware script leaked")
			}
			// The target's physical nozzle is the machine's, never re-derived.
			nozzles, _ := cfg["nozzle_diameter"].([]any)
			for _, n := range nozzles {
				if n != tc.nozzle {
					t.Fatalf("output nozzle_diameter %v, want %s", nozzles, tc.nozzle)
				}
			}
		})
	}
	now, err := os.ReadFile(source)
	if err != nil || sha256.Sum256(now) != originalSHA {
		t.Fatal("source project changed")
	}
}

func TestSelectedQuarterMillimeterNozzleIsNotOverwrittenBySource(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.3mf")
	out := filepath.Join(dir, "C5-source.3mf")
	fixture(t, source, 3)
	setSourceNozzle(t, source, "0.4")

	report, err := ConvertTarget(source, out, TargetRequest{
		Target: "c5", ProfileFile: syntheticVendorBundleFile(t, "0.25"), Nozzle: "0.25",
	})
	if err != nil {
		t.Fatalf("%s: %v", ErrorCode(err), err)
	}
	if report["source_nozzle"] != "0.4" || report["target_nozzle"] != "0.25" {
		t.Fatalf("source nozzle leaked into target: %+v", report)
	}
	if report["nozzle_remapped"] != true || report["nozzle_selected_by_user"] != true ||
		report["target_nozzle_origin"] != string(NozzleSelected) {
		t.Fatalf("selection not reported as the user's: %+v", report)
	}
	for _, n := range projectSettings(t, out)["nozzle_diameter"].([]any) {
		if n != "0.25" {
			t.Fatalf("0.25 mm silently replaced in output: %v", n)
		}
	}
}

func TestTargetNozzleEvidenceFailsClosed(t *testing.T) {
	const orcaID = "orca-acme-mini-0-4-nozzle-0123456789"
	const otherID = "orca-acme-other-0-4-nozzle-9876543210"
	quarter := func(t *testing.T) string { return syntheticVendorBundleFile(t, "0.25") }
	cases := []struct {
		name string
		req  func(t *testing.T) TargetRequest
		code string
	}{
		{"source-matched 0.4 against a 0.25 profile",
			func(t *testing.T) TargetRequest { return TargetRequest{Target: "c5", ProfileFile: quarter(t)} }, "TARGET_NOZZLE_MISMATCH"},
		{"explicit 0.4 against a 0.25 profile",
			func(t *testing.T) TargetRequest {
				return TargetRequest{Target: "c5", ProfileFile: quarter(t), Nozzle: "0.4"}
			}, "TARGET_NOZZLE_MISMATCH"},
		{"explicit 0.2 is never remapped to 0.25",
			func(t *testing.T) TargetRequest {
				return TargetRequest{Target: "c5", ProfileFile: quarter(t), Nozzle: "0.2"}
			}, "TARGET_NOZZLE_UNAVAILABLE"},
		{"nozzle the model does not offer",
			func(t *testing.T) TargetRequest { return TargetRequest{Target: "u1", Nozzle: "0.25"} }, "TARGET_NOZZLE_UNAVAILABLE"},
		{"non-numeric nozzle",
			func(t *testing.T) TargetRequest { return TargetRequest{Target: "qidi", Nozzle: "wide"} }, "TARGET_NOZZLE_UNAVAILABLE"},
		{"selected nozzle without an installed profile",
			func(t *testing.T) TargetRequest { return TargetRequest{Target: "c5", Nozzle: "0.25"} }, "TARGET_PROFILE_REQUIRED"},
		{"unknown alias",
			func(t *testing.T) TargetRequest { return TargetRequest{Target: "bogus"} }, "TARGET_NOT_VALIDATED"},
		{"installed profile disagrees with requested nozzle",
			func(t *testing.T) TargetRequest {
				return TargetRequest{Target: orcaID, ProfileFile: syntheticOrcaFile(t, orcaID, "Acme Mini", "0.4", 1), Nozzle: "0.6"}
			}, "TARGET_NOZZLE_MISMATCH"},
		{"installed bundle belongs to another target",
			func(t *testing.T) TargetRequest {
				return TargetRequest{Target: orcaID, ProfileFile: syntheticOrcaFile(t, otherID, "Acme Other", "0.4", 1)}
			}, "TARGET_PROFILE_INVALID"},
		{"installed target without a bundle",
			func(t *testing.T) TargetRequest { return TargetRequest{Target: orcaID} }, "TARGET_PROFILE_REQUIRED"},
		{"installed bundle with no verified nozzle array",
			func(t *testing.T) TargetRequest {
				bundle := syntheticOrcaBundle(1)
				bundle.Profile.ProjectSettings["nozzle_diameter"] = []any{}
				return TargetRequest{Target: orcaID, ProfileFile: writeOrcaBundle(t, bundle)}
			}, "TARGET_PROFILE_INVALID"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, "source.3mf")
			fixture(t, source, 2)
			setSourceNozzle(t, source, "0.4")
			_, err := ConvertTarget(source, filepath.Join(dir, "out.3mf"), tc.req(t))
			if err == nil || ErrorCode(err) != tc.code {
				t.Fatalf("want %s, got %q (%v)", tc.code, ErrorCode(err), err)
			}
			requireNoOutput(t, dir, "source.3mf")
		})
	}
}

func TestArbitraryInstalledTargetIDsKeepTheirOwnIdentity(t *testing.T) {
	machines := []struct {
		id, model, nozzle string
		tools             int
	}{
		{"orca-vendor-a-single-0-2-nozzle-aaaaaaaaaa", "Vendor A Single", "0.2", 1},
		{"orca-vendor-b-dual-0-6-nozzle-bbbbbbbbbb", "Vendor B Dual", "0.6", 2},
		{"orca-vendor-c-toolchanger-0-8-nozzle-cccccccccc", "Vendor C Toolchanger", "0.8", 6},
		{"orca-vendor-d-pro-0-25-nozzle-dddddddddd", "Vendor D Pro", "0.25", 4},
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "source.3mf")
	fixture(t, source, 4)
	for i, m := range machines {
		out := filepath.Join(dir, fmt.Sprintf("out-%d.3mf", i))
		report, err := ConvertTarget(source, out, TargetRequest{
			Target: m.id, ProfileFile: syntheticOrcaFile(t, m.id, m.model, m.nozzle, m.tools),
		})
		if err != nil {
			t.Fatalf("%s: %s: %v", m.id, ErrorCode(err), err)
		}
		if report["target"] != m.id || report["target_nozzle"] != m.nozzle || report["physical_toolheads"] != m.tools {
			t.Fatalf("%s: %+v", m.id, report)
		}
		if got := projectSettings(t, out)["printer_model"]; got != m.model {
			t.Fatalf("%s: printer_model %v, want %s", m.id, got, m.model)
		}
	}
}

// The exported legacy entry points are thin adapters; they must yield the same
// report as the unified entry point for the same inputs.
func TestLegacyEntryPointsMatchUnifiedEntryPoint(t *testing.T) {
	const id = "orca-acme-mini-0-4-nozzle-0123456789"
	dir := t.TempDir()
	source := filepath.Join(dir, "source.3mf")
	fixture(t, source, 3)
	orca := syntheticOrcaFile(t, id, "Acme Mini", "0.4", 1)
	vendor := syntheticVendorBundleFile(t, "0.25")

	normalize := func(report map[string]any, out string) map[string]any {
		clone := map[string]any{}
		for k, v := range report {
			clone[k] = v
		}
		if clone["output"] != out {
			t.Fatalf("output path %v, want %s", clone["output"], out)
		}
		delete(clone, "output")
		return clone
	}
	type pair struct {
		name            string
		legacy, unified func(out string) (map[string]any, error)
	}
	pairs := []pair{
		{"u1",
			func(out string) (map[string]any, error) { return ConvertWithProfile(source, out, "u1", "") },
			func(out string) (map[string]any, error) {
				return ConvertTarget(source, out, TargetRequest{Target: "u1"})
			}},
		{"c5 selected 0.25",
			func(out string) (map[string]any, error) {
				return ConvertWithTargetNozzle(source, out, "c5", vendor, "0.25")
			},
			func(out string) (map[string]any, error) {
				return ConvertTarget(source, out, TargetRequest{Target: "c5", ProfileFile: vendor, Nozzle: "0.25"})
			}},
		{"orca",
			func(out string) (map[string]any, error) { return ConvertWithOrcaBundle(source, out, orca) },
			func(out string) (map[string]any, error) {
				return ConvertTarget(source, out, TargetRequest{Target: id, ProfileFile: orca})
			}},
	}
	for _, p := range pairs {
		a, b := filepath.Join(dir, p.name+"-legacy.3mf"), filepath.Join(dir, p.name+"-unified.3mf")
		legacy, err := p.legacy(a)
		if err != nil {
			t.Fatalf("%s legacy: %v", p.name, err)
		}
		unified, err := p.unified(b)
		if err != nil {
			t.Fatalf("%s unified: %v", p.name, err)
		}
		if !reflect.DeepEqual(normalize(legacy, a), normalize(unified, b)) {
			t.Fatalf("%s: legacy and unified reports differ:\n%+v\n%+v", p.name, legacy, unified)
		}
	}
}

// Built-in alias names are part of the saved-settings contract.
func TestBuiltinAliasesRemainStable(t *testing.T) {
	want := map[string]string{
		"u1": "Snapmaker U1", "c5": "Flashforge Creator 5",
		"cc2": "Elegoo Centauri Carbon 2", "qidi": "X-Plus 5",
	}
	if !reflect.DeepEqual(targetModels, want) {
		t.Fatalf("built-in alias table changed: %v", targetModels)
	}
	for alias := range want {
		if orcaTargetID.MatchString(alias) {
			t.Fatalf("built-in alias %q collides with the installed ID shape", alias)
		}
	}
}
