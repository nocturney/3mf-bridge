package main

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func prusaVendorFixture(t *testing.T, root, name string, lines []string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name+".ini"), []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestPrusaVendorDefaultThumbnailMatchesExactly(t *testing.T) {
	root := t.TempDir()
	prusaVendorFixture(t, root, "Anycubic", []string{
		"[vendor]", "name = Anycubic",
		"[printer_model:I3MEGA]", "name = Anycubic i3 Mega", "variants = 0.4",
		"[printer_model:I3MEGAS]", "name = Anycubic i3 Mega S", "variants = 0.4",
	})
	original := writePNGCover(t, root, "Anycubic/I3MEGA_thumbnail.png", color.RGBA{10, 30, 40, 255})
	other := writePNGCover(t, root, "Anycubic/I3MEGAS_thumbnail.png", color.RGBA{100, 30, 40, 255})
	model := CatalogPrinter{ID: "anycubic-anycubic-i3-mega", Manufacturer: "Anycubic", Model: "Anycubic i3 Mega"}
	got := findPrusaThumbnailArtwork(model, []string{root})
	if !got.Found || got.Provenance != provenanceInstalledSlicer || got.DataURI != pngDataURIOf(t, original) {
		t.Fatalf("exact model should use installed PrusaSlicer thumbnail, got %+v", got)
	}
	if got.DataURI == pngDataURIOf(t, other) {
		t.Fatal("borrowed the similar i3 Mega S photograph")
	}
	wrong := CatalogPrinter{Manufacturer: "Anycubic", Model: "Anycubic i3 Mega Plus"}
	if found := findPrusaThumbnailArtwork(wrong, []string{root}); found.Found {
		t.Fatal("fuzzy/printer-family match was incorrectly accepted")
	}
	foreign := CatalogPrinter{Manufacturer: "Creality", Model: "Anycubic i3 Mega"}
	if found := findPrusaThumbnailArtwork(foreign, []string{root}); found.Found {
		t.Fatal("image borrowed from a different manufacturer")
	}
}

func TestPrusaVendorExplicitThumbnailAndProSeparation(t *testing.T) {
	root := t.TempDir()
	prusaVendorFixture(t, root, "Creality", []string{
		"[vendor]", "name = Creality",
		"[printer_model:CR10S]", "name = Creality CR-10 S", "thumbnail = CR10S_sample.png",
		"[printer_model:CR10SPRO]", "name = Creality CR-10 S Pro", "thumbnail = CR10SPRO_sample.png",
	})
	expected := writeSampleCover(t, root, "Creality/CR10S_sample.png")
	pro := writePNGCover(t, root, "Creality/CR10SPRO_sample.png", color.RGBA{120, 10, 30, 255})
	model := CatalogPrinter{Manufacturer: "Creality", Model: "Creality CR-10 S"}
	got := findPrusaThumbnailArtwork(model, []string{root})
	if !got.Found || got.DataURI != pngDataURIOf(t, expected) || got.DataURI == pngDataURIOf(t, pro) {
		t.Fatal("CR-10 S borrowed the Pro image or failed to show its exact thumbnail")
	}
}

func TestPrusaVendorOfficialNamePrefix(t *testing.T) {
	root := t.TempDir()
	prusaVendorFixture(t, root, "PrusaResearch", []string{
		"[vendor]", "name = Prusa Research",
		"[printer_model:MK4ISMMU3]", "name = Original Prusa MK4 MMU3", "thumbnail = MK4ISMMU3_thumbnail.png",
	})
	expected := writeSampleCover(t, root, "PrusaResearch/MK4ISMMU3_thumbnail.png")
	for _, model := range []CatalogPrinter{
		{Manufacturer: "Prusa", Model: "Prusa MK4 MMU3"},
		{Manufacturer: "Prusa Research", Model: "MK4 MMU3"},
	} {
		got := findPrusaThumbnailArtwork(model, []string{root})
		if !got.Found || got.DataURI != pngDataURIOf(t, expected) {
			t.Fatalf("official Prusa alias was not resolved for %+v", model)
		}
	}
	wrong := CatalogPrinter{Manufacturer: "Prusa", Model: "Prusa MK4 MMU3 HF"}
	if found := findPrusaThumbnailArtwork(wrong, []string{root}); found.Found {
		t.Fatal("HF variant borrowed a different product's thumbnail")
	}
}

func TestPrusaVendorAmbiguousOrUnsafeThumbnailFailsClosed(t *testing.T) {
	root := t.TempDir()
	prusaVendorFixture(t, root, "Creality", []string{
		"[printer_model:ONE]", "name = Creality CR-10 S", "thumbnail = CR10S_thumbnail.png",
		"[printer_model:TWO]", "name = Creality CR-10 S", "thumbnail = CR10SPRO_thumbnail.png",
	})
	writeSampleCover(t, root, "Creality/CR10S_thumbnail.png")
	writeSampleCover(t, root, "Creality/CR10SPRO_thumbnail.png")
	m := CatalogPrinter{Manufacturer: "Creality", Model: "Creality CR-10 S"}
	if findPrusaThumbnailArtwork(m, []string{root}).Found {
		t.Fatal("ambiguous official vendor picture mapping must not silently choose")
	}
	bad := []string{
		"../elsewhere.png", "..\\elsewhere.png", "https://elsewhere.png",
		"file:///tmp/elsewhere.png", "/etc/passwd", "a/b.png",
		"image.svg", "a..png", "  image.png", "image.png ", "x\nbad.png",
	}
	for _, filename := range bad {
		if validPrusaThumbnailPath("Creality/" + filename) {
			t.Fatalf("unsafe thumbnail path accepted: %q", filename)
		}
	}
	prusaVendorFixture(t, root, "Creality", []string{
		"[printer_model:CR10S]", "name = Creality CR-10 S", "thumbnail = ../secret.png",
	})
	if findPrusaThumbnailArtwork(m, []string{root}).Found {
		t.Fatal("unsafe thumbnail from INI must not be opened")
	}
}

func TestPrusaVendorRejectsNonImagesAndSymlinks(t *testing.T) {
	root := t.TempDir()
	prusaVendorFixture(t, root, "Anycubic", []string{
		"[printer_model:AK]", "name = Anycubic Kossel Pulley (Linear)", "thumbnail = AK_thumbnail.png",
	})
	imagePath := filepath.Join(root, "Anycubic", "AK_thumbnail.png")
	if err := os.MkdirAll(filepath.Dir(imagePath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(imagePath, []byte("not a png"), 0600); err != nil {
		t.Fatal(err)
	}
	m := CatalogPrinter{Manufacturer: "Anycubic", Model: "Anycubic Kossel Pulley (Linear)"}
	if findPrusaThumbnailArtwork(m, []string{root}).Found {
		t.Fatal("non-PNG bytes treated as printer photo")
	}
	outside := writeSampleCover(t, t.TempDir(), "outside.png")
	if err := os.Remove(imagePath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, imagePath); err == nil {
		if findPrusaThumbnailArtwork(m, []string{root}).Found {
			t.Fatal("symlink leading outside trusted install accepted")
		}
	}
	if findPrusaThumbnailArtwork(m, []string{root, filepath.Join(root, "missing")}).Found {
		t.Fatal("untrusted or absent image accepted")
	}
}

func TestPrusaThumbnailIntegrationForCatalogID(t *testing.T) {
	root := t.TempDir()
	prusaVendorFixture(t, root, "Anycubic", []string{
		"[printer_model:I3MEGA]", "name = Anycubic i3 Mega",
	})
	expected := writeSampleCover(t, root, "Anycubic/I3MEGA_thumbnail.png")
	previous := prusaThumbnailRootList
	prusaThumbnailRootList = func() []string { return []string{root} }
	t.Cleanup(func() { prusaThumbnailRootList = previous })

	model := CatalogPrinter{ID: "anycubic-anycubic-i3-mega", Manufacturer: "Anycubic", Model: "Anycubic i3 Mega"}
	if !prusaPictureMatches(model, "Anycubic i3 Mega") {
		t.Fatal("simple vendor model identity must match")
	}
	reply, err := (&App{}).PrinterArtwork(model.ID)
	if err != nil || !reply.Found || reply.DataURI != pngDataURIOf(t, expected) {
		t.Fatalf("catalogue endpoint did not serve installed thumbnail: %+v %v", reply, err)
	}
	source, err := os.ReadFile(expected)
	if err != nil || !strings.HasPrefix(reply.DataURI, "data:image/png;base64,") || len(source) == 0 {
		t.Fatal("missing source preserved or invalid local data URI")
	}
}
