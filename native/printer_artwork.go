package main

// Local printer artwork is read ONLY from explicit, installed slicer resource
// roots, never from a user-controlled arbitrary path or a remote URL.
// Raw profiles, photographs and logos are not bundled with SliceBridge.
import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nocturney/slicebridge/native/presets"
)

const maxArtworkBytes = 512 << 10
const provenanceInstalledSlicer = "local_installed_slicer"

type PrinterArtworkReply struct {
	Found      bool   `json:"found"`
	DataURI    string `json:"dataUri,omitempty"`
	Provenance string `json:"provenance,omitempty"`
}

// resolveInstalledArtwork is a variable so tests can use a fixed installation.
// Production always re-resolves the target from disk.
var resolveInstalledArtwork = presets.ResolveOrcaArtworkSource

func validLocalCoverPath(relative string) bool {
	if len(relative) > 320 || strings.Count(relative, "/") != 1 ||
		!strings.HasSuffix(relative, "_cover.png") ||
		strings.ContainsAny(relative, "\\:\x00\r\n") ||
		strings.Contains(relative, "..") {
		return false
	}
	pair := strings.Split(relative, "/")
	for _, part := range pair {
		if len(part) == 0 || strings.HasPrefix(part, ".") ||
			strings.TrimSpace(part) != part {
			return false
		}
		for _, c := range part {
			if c < ' ' || c == 127 {
				return false
			}
		}
	}
	return true
}

func allowedProfileRoots() []string {
	// Dynamic and legacy printer cards share one complete, explicitly allowed
	// local installed-slicer artwork inventory. No arbitrary filesystem search.
	return presets.ArtworkRoots()
}

func localPNGDataURI(profileRoot, relative string) (string, bool) {
	if !validLocalCoverPath(relative) {
		return "", false
	}
	root, err := filepath.EvalSymlinks(profileRoot)
	if err != nil {
		return "", false
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", false
	}
	path := filepath.Join(root, filepath.FromSlash(relative))
	info, err = os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() ||
		info.Size() <= 0 || info.Size() > maxArtworkBytes {
		return "", false
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", false
	}
	within, err := filepath.Rel(root, resolved)
	if err != nil || within == ".." || strings.HasPrefix(within, ".."+string(os.PathSeparator)) {
		return "", false
	}
	f, err := os.Open(resolved)
	if err != nil {
		return "", false
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxArtworkBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxArtworkBytes {
		return "", false
	}
	if !bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		return "", false
	}
	dimensions, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != "png" || dimensions.Width < 32 ||
		dimensions.Height < 32 || dimensions.Width > 2048 ||
		dimensions.Height > 2048 ||
		dimensions.Width*dimensions.Height > 4_000_000 {
		return "", false
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data), true
}

func findPrinterArtwork(model CatalogPrinter, roots []string) PrinterArtworkReply {
	for _, relative := range model.LocalCoverCandidates {
		for _, root := range roots {
			if uri, ok := localPNGDataURI(root, relative); ok {
				return PrinterArtworkReply{
					Found: true, DataURI: uri, Provenance: provenanceInstalledSlicer,
				}
			}
		}
	}
	return PrinterArtworkReply{Found: false}
}

// installedTargetCoverPath is the only file an installed target may use: its
// vendor folder plus its exact printer_model, as slicers name cover pictures
// ("Snapmaker/Snapmaker U1_cover.png"). There is no substring or fuzzy matching,
// so "Creator 5" never borrows "Creator 5 Pro" and the reverse is also true.
func installedTargetCoverPath(source presets.OrcaArtworkSource) (string, bool) {
	if source.Model == "" {
		return "", false
	}
	relative := source.Vendor + "/" + source.Model + "_cover.png"
	return relative, validLocalCoverPath(relative)
}

// findInstalledTargetArtwork prefers the target's own installation. Other
// allowlisted installations are checked only for that same exact file.
func findInstalledTargetArtwork(source presets.OrcaArtworkSource, roots []string) PrinterArtworkReply {
	relative, ok := installedTargetCoverPath(source)
	if !ok {
		return PrinterArtworkReply{Found: false}
	}
	// source.Root comes from the same allowlisted location inventory as roots.
	for _, root := range append([]string{source.Root}, roots...) {
		if uri, ok := localPNGDataURI(root, relative); ok {
			return PrinterArtworkReply{
				Found: true, DataURI: uri, Provenance: provenanceInstalledSlicer,
			}
		}
	}
	return PrinterArtworkReply{Found: false}
}

// PrinterArtwork exposes an allowlisted model ID, not an arbitrary file path.
// The Wails caller never controls installed-root paths or image filenames.
// Kept for the four legacy converter models and offline catalogue entries.
func (a *App) PrinterArtwork(modelID string) (PrinterArtworkReply, error) {
	if !catalogIDPattern.MatchString(modelID) {
		return PrinterArtworkReply{}, fmt.Errorf("invalid printer model ID")
	}
	catalog, err := catalogSnapshot()
	if err != nil {
		return PrinterArtworkReply{}, err
	}
	for _, model := range catalog.Models {
		if model.ID == modelID {
			return findPrinterArtwork(model, allowedProfileRoots()), nil
		}
	}
	return PrinterArtworkReply{}, fmt.Errorf("unknown printer model")
}

// PrinterArtworkForInstalledTarget takes an installed Orca-family target ID only.
// The ID is matched against freshly discovered machines, so the frontend never
// supplies a path, a filename or a catalogue model.
func (a *App) PrinterArtworkForInstalledTarget(targetID string) (PrinterArtworkReply, error) {
	source, err := resolveInstalledArtwork(targetID)
	if err != nil {
		return PrinterArtworkReply{}, fmt.Errorf("installed printer profile is not ready")
	}
	return findInstalledTargetArtwork(source, allowedProfileRoots()), nil
}
