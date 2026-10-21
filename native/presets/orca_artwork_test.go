package presets

import "testing"

func TestArtworkSourceNamesOwningInstallationAndExactModel(t *testing.T) {
	root := writeOrcaTree(t, "Acme", acmeSample())
	locations := []slicerLocation{{id: "test", name: "Test Orca", root: root}}
	target := discoverOrca(locations).Targets[0]

	source, err := resolveOrcaArtworkSource(locations, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if source.Root != root || source.Vendor != "Acme" || source.Model != "Acme Mini" {
		t.Fatalf("artwork source must name the owning install and exact printer_model: %+v", source)
	}
}

func TestArtworkSourceRequiresFreshlyResolvedIdentifier(t *testing.T) {
	root := writeOrcaTree(t, "Acme", acmeSample())
	locations := []slicerLocation{{id: "test", name: "Test Orca", root: root}}
	for _, id := range []string{"orca-acme-mini-0123456789", "../Acme", "Acme Mini", ""} {
		if _, err := resolveOrcaArtworkSource(locations, id); err == nil {
			t.Fatalf("unresolvable identifier produced an artwork source: %q", id)
		}
	}
}
