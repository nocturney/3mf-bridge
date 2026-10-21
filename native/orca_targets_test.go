package main

import (
	"fmt"
	"testing"
)

func TestInstalledTargetIDsAreAcceptedOnlyInValidatedShape(t *testing.T) {
	good := "orca-acme-mini-0-4-nozzle-0123456789"
	if err := validateTargets([]string{good}); err != nil {
		t.Fatalf("well-formed installed target rejected: %v", err)
	}
	if err := validateTargets([]string{"u1", good}); err != nil {
		t.Fatalf("built-in and installed targets should mix: %v", err)
	}
	for _, bad := range []string{"orca-../x", "orca-acme", "qidi-x", "ORCA-ACME-0123456789", "orca-acme-012345678"} {
		if err := validateTargets([]string{bad}); err == nil {
			t.Fatalf("%q accepted as a conversion target", bad)
		}
	}
	if err := validateTargets([]string{good, good}); err == nil {
		t.Fatal("repeated installed target accepted")
	}
}

func TestInstalledTitlesPreferCatalogueMetadata(t *testing.T) {
	catalog := PrinterCatalogReply{Models: []CatalogPrinter{
		{Manufacturer: "Bambu Lab", Model: "X1C", Aliases: []string{"Acme Mini"}},
		{Manufacturer: "Creality", Model: "K1", Aliases: []string{"K1 Max"}},
	}}
	if got := catalogTitle(catalog, "K1 Max"); got != "Creality K1" {
		t.Fatalf("alias should resolve to catalogue title, got %q", got)
	}
	// Bambu entries stay out of installed-target titles entirely.
	if got := catalogTitle(catalog, "Acme Mini"); got != "Acme Mini" {
		t.Fatalf("unmatched or Bambu-only model should keep installed name, got %q", got)
	}
}

// A large printer selection is bounded by concurrency, not restricted to
// the four historically hard-coded destination models.
func TestManyInstalledOrcaTargetsAllowed(t *testing.T) {
	targets := []string{"u1", "c5", "cc2", "qidi"}
	for i := 0; i < maxMyPrinters-4; i++ {
		targets = append(targets, fmt.Sprintf("orca-acme-model-%d-%010x", i, i))
	}
	if err := validateTargets(targets); err != nil {
		t.Fatalf("up to %d independent targets must be accepted: %v", maxMyPrinters, err)
	}
	targets = append(targets, "orca-acme-model-extra-000000ffff")
	if err := validateTargets(targets); err == nil {
		t.Fatal("target count beyond the selected-printer limit was accepted")
	}
}
