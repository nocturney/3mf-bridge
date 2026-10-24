package main

// PrusaSlicer vendor bundles describe printer thumbnail filenames independently
// of Orca's "<model>_cover.png" convention. Read only explicitly installed
// PrusaSlicer system profiles; never edit or import vendor printer settings.
import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

const maxPrusaThumbnailBundleBytes = 2 << 20

type prusaThumbnailModel struct {
	name     string
	filename string
}

func artworkIdentity(text string) string {
	var b strings.Builder
	for _, char := range strings.ToLower(text) {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			b.WriteRune(char)
		}
	}
	return b.String()
}

func prusaVendorIdentity(name string) string {
	key := artworkIdentity(name)
	if key == "prusa" {
		return "prusaresearch"
	}
	return key
}

// The sole spelling alias is Prusa's official "Original Prusa" product prefix.
// Every other manufacturer requires complete normalized-model equality; e.g.
// CR-10 S cannot borrow CR-10 S Pro, or Kobra 3 borrow Kobra 3 Max.
func prusaPictureModelIdentity(name, vendor string) string {
	value := artworkIdentity(name)
	if vendor == "prusaresearch" {
		value = strings.TrimPrefix(value, "originalprusa")
		value = strings.TrimPrefix(value, "prusa")
	}
	return value
}

func prusaPictureMatches(model CatalogPrinter, sourceName string) bool {
	vendor := prusaVendorIdentity(model.Manufacturer)
	if vendor == "" || sourceName == "" {
		return false
	}
	source := prusaPictureModelIdentity(sourceName, vendor)
	candidates := []string{model.Model}
	if vendor != "prusaresearch" {
		candidates = append(candidates, model.Manufacturer+" "+model.Model)
	}
	candidates = append(candidates, model.Aliases...)
	for _, candidate := range candidates {
		if value := prusaPictureModelIdentity(candidate, vendor); value != "" && value == source {
			return true
		}
	}
	return false
}

// Only [printer_model:ID] name/thumbnail metadata is parsed. No slicer
// profiles, scripts, or machine settings are read or trusted for conversion.
// When no thumbnail is named, PrusaSlicer's documented ID_thumbnail.png
// convention applies. Files remain in their original installed location.
func readPrusaThumbnailModels(root, bundle string) []prusaThumbnailModel {
	file := filepath.Join(root, bundle+".ini")
	info, err := os.Lstat(file)
	if err != nil || !info.Mode().IsRegular() ||
		info.Size() <= 0 || info.Size() > maxPrusaThumbnailBundleBytes {
		return nil
	}
	data, err := os.ReadFile(file)
	if err != nil || len(data) == 0 || len(data) > maxPrusaThumbnailBundleBytes {
		return nil
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), 256<<10)

	var rows []prusaThumbnailModel
	var current *prusaThumbnailModel
	var modelID string
	flush := func() {
		if current != nil && current.name != "" {
			if current.filename == "" {
				current.filename = modelID + "_thumbnail.png"
			}
			rows = append(rows, *current)
		}
	}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flush()
			current = nil
			modelID = ""
			if strings.HasPrefix(line, "[printer_model:") {
				modelID = strings.TrimSuffix(strings.TrimPrefix(line, "[printer_model:"), "]")
				if modelID != "" && len(modelID) <= 120 {
					current = &prusaThumbnailModel{}
				}
			}
			continue
		}
		if current == nil {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "name":
			current.name = strings.TrimSpace(value)
		case "thumbnail":
			current.filename = strings.TrimSpace(value)
		}
	}
	if scanner.Err() != nil {
		return nil
	}
	flush()
	return rows
}

// findPrusaThumbnailArtwork is a fallback ONLY for catalogue models for which
// no exact installed Orca-family cover was found. It never selects a printer
// profile or contacts a remote image server. The matching vendor INI must name
// the exact physical model and yield one unambiguous, locally valid PNG.
func findPrusaThumbnailArtwork(model CatalogPrinter, roots []string) PrinterArtworkReply {
	vendor := prusaVendorIdentity(model.Manufacturer)
	if vendor == "" {
		return PrinterArtworkReply{Found: false}
	}
	for _, root := range roots {
		bundles, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, bundle := range bundles {
			filename := bundle.Name()
			if bundle.IsDir() || !strings.HasSuffix(strings.ToLower(filename), ".ini") {
				continue
			}
			stem := filename[:len(filename)-4]
			if prusaVendorIdentity(stem) != vendor {
				continue
			}
			models := readPrusaThumbnailModels(root, stem)
			relative := ""
			ambiguous := false
			for _, row := range models {
				if !prusaPictureMatches(model, row.name) {
					continue
				}
				candidate := stem + "/" + row.filename
				if !validPrusaThumbnailPath(candidate) {
					ambiguous = true
					break
				}
				if relative != "" && relative != candidate {
					ambiguous = true
					break
				}
				relative = candidate
			}
			if ambiguous || relative == "" {
				continue
			}
			if uri, ok := localPrusaPNGDataURI(root, relative); ok {
				return PrinterArtworkReply{
					Found: true, DataURI: uri, Provenance: provenanceInstalledSlicer,
				}
			}
		}
	}
	return PrinterArtworkReply{Found: false}
}
