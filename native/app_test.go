package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nocturney/slicebridge/native/presets"
)

func loadFixture(t *testing.T, fixture string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "tests", "fixtures", fixture))
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "source.3mf")
	if err = os.WriteFile(target, data, 0600); err != nil {
		t.Fatal(err)
	}
	return target
}
func TestBuiltinU1WithoutSlicer(t *testing.T) {
	app := &App{}
	file := loadFixture(t, "bambu-one-color.3mf")
	reply := app.Convert(ConvertRequest{FilePath: file, Targets: []string{"u1"}})
	if !reply.OK || len(reply.Results) != 1 || len(reply.Failures) != 0 {
		t.Fatalf("U1 conversion failed: %+v", reply)
	}
	if reply.Results[0].Report["target_nozzle"] != "0.4" {
		t.Fatalf("target nozzle mismatch: %+v", reply.Results[0].Report)
	}
	if _, err := os.Stat(reply.Results[0].OutputPath); err != nil {
		t.Fatal(err)
	}
	again := app.Convert(ConvertRequest{FilePath: file, Targets: []string{"u1"}})
	if !again.OK || again.Results[0].OutputPath == reply.Results[0].OutputPath {
		t.Fatalf("previous output overwritten: %+v", again)
	}
}
func TestInvalidInputsDoNotWriteFiles(t *testing.T) {
	app := &App{}
	file := loadFixture(t, "bambu-one-color.3mf")
	for _, targets := range [][]string{nil, {"u1", "u1"}, {"malicious"}} {
		r := app.Convert(ConvertRequest{FilePath: file, Targets: targets})
		if r.OK || len(r.Results) != 0 {
			t.Fatalf("bad target accepted %+v", r)
		}
	}
}
func TestInstalledProfilesOnHost(t *testing.T) {
	if os.Getenv("BRIDGE_TEST_INSTALLED") != "1" {
		t.Skip("requires local vendor slicer installations")
	}
	for _, target := range []string{"u1", "c5", "cc2", "qidi"} {
		for _, sourceNozzle := range []string{"0.2", "0.4", "0.6", "0.8"} {
			// U1 0.4 has an independently validated builtin engine profile.
			if target == "u1" && sourceNozzle == "0.4" {
				continue
			}
			bundle, err := presets.Load(target, sourceNozzle)
			if err != nil {
				t.Errorf("%s/%s: %v", target, sourceNozzle, err)
				continue
			}
			expected := sourceNozzle
			if target == "c5" && sourceNozzle == "0.2" {
				expected = "0.25"
			}
			if bundle.Nozzle != expected {
				t.Errorf("%s/%s: got %s", target, sourceNozzle, bundle.Nozzle)
			}
			if len(bundle.Profile.ProjectSettings) < 150 {
				t.Errorf("%s/%s: profile unusually small", target, sourceNozzle)
			}
		}
	}
}
func TestFourTargetsConvertedInParallelOnInstalledHost(t *testing.T) {
	if os.Getenv("BRIDGE_TEST_INSTALLED") != "1" {
		t.Skip("requires local vendor slicer installations")
	}
	for _, fixture := range []string{"bambu-nozzle-0.2.3mf", "bambu-mixed-nozzle-0.6.3mf", "bambu-nozzle-0.8.3mf"} {
		source := loadFixture(t, fixture)
		app := &App{}
		response := app.Convert(ConvertRequest{FilePath: source, Targets: []string{"u1", "c5", "cc2", "qidi"}})
		if !response.OK || len(response.Results) != 4 || len(response.Failures) > 0 {
			t.Errorf("%s: failed: %+v", fixture, response)
			continue
		}
		for _, result := range response.Results {
			expected := response.SourceNozzle
			if result.Target == "c5" && expected == "0.2" {
				expected = "0.25"
			}
			if result.Report["target_nozzle"] != expected {
				t.Errorf("%s %s: mismatch expected %s got %v", fixture, result.Target, expected, result.Report["target_nozzle"])
			}
			if _, err := os.Stat(result.OutputPath); err != nil {
				t.Error(err)
			}
		}
	}
}
