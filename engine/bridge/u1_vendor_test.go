package bridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// A validated installed U1 vendor profile must take precedence over the
// historical field-validated offline fallback. This protects process and
// machine settings chosen by the manufacturer's slicer.
func TestInstalledSnapmakerProfileIsNotSilentlyReplacedByFallback(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.3mf")
	fixture(t, source, 3)
	bundle := syntheticMaterialBundle()
	bundle.Target = "u1"
	bundle.Nozzle = "0.4"
	bundle.Profile.Name = "u1"
	bundle.Profile.DisplayName = "Snapmaker U1"
	bundle.Profile.ModelID = "Snapmaker U1"
	bundle.Profile.ProjectSettings["printer_model"] = "Snapmaker U1"
	bundle.Profile.ProjectSettings["machine_start_gcode"] = "G28 ; vendor snapmaker profile"
	bundle.Profile.ProjectSettings["print_settings_id"] = "Vendor process @Snapmaker U1"
	data, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	profile := filepath.Join(dir, "vendor-profile.json")
	if err := os.WriteFile(profile, data, 0600); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(dir, "target.3mf")
	report, err := ConvertTarget(source, out, TargetRequest{
		Target: "u1", ProfileFile: profile, Nozzle: "0.4",
	})
	if err != nil {
		t.Fatalf("%s: %v", ErrorCode(err), err)
	}
	if report["profile_kind"] != string(ProfileInstalledVendor) {
		t.Fatalf("installed vendor bundle ignored: %v", report["profile_kind"])
	}
	settings := projectSettings(t, out)
	if settings["machine_start_gcode"] != "G28 ; vendor snapmaker profile" {
		t.Fatalf("the installed vendor machine settings were ignored: %v", settings["machine_start_gcode"])
	}
	if settings["print_settings_id"] != "Vendor process @Snapmaker U1" {
		t.Fatalf("the installed vendor process was ignored: %v", settings["print_settings_id"])
	}
}
