package presets

import (
	"strings"
	"testing"
)

// Narrow-nozzle equivalence is a profile-selection rule only. Never write a
// 0.2 mm physical nozzle into a 0.25 mm Flashforge preset or vice versa.
func TestVariantNarrowNozzleCategoryKeepsPhysicalManufacturerDiameter(t *testing.T) {
	cases := []struct{ target, source, want string }{
		{"c5", "0.2", "0.25"},
		{"c5", "0.25", "0.25"},
		{"u1", "0.25", "0.2"},
		{"cc2", "0.25", "0.2"},
		{"qidi", "0.25", "0.2"},
		{"u1", "0.2", "0.2"},
		{"cc2", "0.2", "0.2"},
		{"qidi", "0.2", "0.2"},
		{"c5", "0.4", "0.4"},
		{"u1", "0.6", "0.6"},
		{"cc2", "0.8", "0.8"},
	}
	for _, tc := range cases {
		got, machine, process, materials, err := Variant(tc.target, tc.source)
		if err != nil {
			t.Fatalf("%s source %s: %v", tc.target, tc.source, err)
		}
		if got != tc.want || !strings.Contains(machine, tc.want) || process == "" || materials["PLA"] == "" {
			t.Fatalf("%s from %s: target nozzle=%q machine=%q process=%q materials=%v; want %s",
				tc.target, tc.source, got, machine, process, materials, tc.want)
		}
	}
	for _, tc := range []struct{ target, nozzle string }{{"u1", "0.3"}, {"c5", "0.15"}, {"cc2", "1.0"}, {"qidi", "0.5"}} {
		if got, _, _, _, err := Variant(tc.target, tc.nozzle); err == nil {
			t.Fatalf("unsupported physical source %s mapped unexpectedly to %q", tc.nozzle, got)
		}
	}
}
