package bridge

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/BryanSmee/bl_to_std/pkg/printer"
)

func TestNozzleTargetMapping(t *testing.T) {
	for _, tc := range []struct {
		source, target, want string
		fail                 bool
	}{
		{"0.2", "u1", "0.2", false},
		{"0.2", "c5", "0.25", false},
		{"0.2", "cc2", "0.2", false},
		{"0.2", "qidi", "0.2", false},
		{"0.4", "c5", "0.4", false},
		{"0.6", "c5", "0.6", false},
		{"0.8", "c5", "0.8", false},
		{"0.25", "c5", "0.25", false},
		{"0.25", "cc2", "", true},
		{"0.25", "u1", "", true},
	} {
		got, err := targetNozzleForSource(tc.target, tc.source)
		if (err != nil) != tc.fail || got != tc.want {
			t.Errorf("%s %s: got %q error %v, want %q fail %t", tc.target, tc.source, got, err, tc.want, tc.fail)
		}
	}
}

func TestNozzleParser(t *testing.T) {
	for _, pair := range [][2]string{
		{"Bambu Lab A1 0.2 nozzle", "0.2"},
		{"Bambu Lab X1 Carbon 0.4 nozzle", "0.4"},
		{"Bambu A1 0.6 mm nozzle", "0.6"},
		{"Creator 5 0.25 nozzle", "0.25"},
		{"Bambu Lab A1", ""},
	} {
		if got := nozzleFromPresetName(pair[0]); got != pair[1] {
			t.Errorf("%q returned %q, expected %q", pair[0], got, pair[1])
		}
	}
	vals := []any{"0.6", "0.8"}
	if _, err := nozzleFromRaw(vals); err == nil {
		t.Fatal("mixed nozzle sizes were accepted")
	}
	if got, err := nozzleFromRaw([]any{"0.2", "0.2"}); err != nil || got != "0.2" {
		t.Fatalf("uniform nozzle sizes failed %q: %v", got, err)
	}
}

func TestSourceWithUnspecifiedNozzleUsesReportedDefault(t *testing.T) {
	input := filepath.Join(t.TempDir(), "legacy.3mf")
	fixture(t, input, 1)
	info, err := InspectNozzle(input)
	if err != nil {
		t.Fatal(err)
	}
	if info["source_nozzle"] != "0.4" || info["source_nozzle_assumed"] != true {
		t.Fatalf("expected an explicit 0.4mm assumption: %v", info)
	}
}

func TestLayerHeightRespectsNozzleRange(t *testing.T) {
	input := filepath.Join(t.TempDir(), "source.3mf")
	fixture(t, input, 1) // original layer_height=0.16
	makeProfile := func(min, max, layer string) *printer.Profile {
		return &printer.Profile{
			Name: "test",
			ProjectSettings: map[string]any{
				"min_layer_height": []any{min},
				"max_layer_height": []any{max},
				"layer_height":     layer,
			},
		}
	}
	p, keys, err := preservedProfile(input, makeProfile("0.08", "0.14", "0.12"))
	if err != nil {
		t.Fatal(err)
	}
	if p.ProjectSettings["layer_height"] != "0.12" {
		t.Fatalf("invalid 0.16 copied to 0.25mm toolhead: %v", p.ProjectSettings["layer_height"])
	}
	for _, key := range keys {
		if key == "layer_height" {
			t.Fatal("invalid layer listed as preserved")
		}
	}
	p, keys, err = preservedProfile(input, makeProfile("0.12", "0.42", "0.30"))
	if err != nil {
		t.Fatal(err)
	}
	if p.ProjectSettings["layer_height"] != "0.16" {
		t.Fatalf("safe source layer was not preserved: %v", p.ProjectSettings["layer_height"])
	}
	if !strings.Contains(strings.Join(keys, ","), "layer_height") {
		t.Fatalf("preserved layer missing: %v", keys)
	}
}

func TestTargetNozzleProfileMismatchRejected(t *testing.T) {
	bundle := syntheticMaterialBundle()
	bundle.Nozzle = "0.25"
	if err := verifyTargetBundleNozzle(bundle); err == nil {
		t.Fatal("0.4mm machine mislabeled as 0.25mm was accepted")
	}
	bundle.Nozzle = "0.4"
	if err := verifyTargetBundleNozzle(bundle); err != nil {
		t.Fatalf("matching machine dimensions failed: %v", err)
	}
}
