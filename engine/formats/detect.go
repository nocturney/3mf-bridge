// Package formats detects the source 3MF project family from package structure.
// Detection is intentionally separate from conversion capability: knowing a
// producer's format does not mean a target machine is supported.
package formats

import (
	"archive/zip"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
)

// Family identifies the project-metadata dialect, not the printer model.
// Bambu Studio and OrcaSlicer use the same relevant 3MF configuration family.
type Family string

const (
	Unknown   Family = "unknown"
	BambuOrca Family = "bambu-orca"
	Prusa     Family = "prusa"
)

const (
	maxSourceBytes    int64 = 250 << 20
	maxArchiveEntries       = 12000
)

var ErrConflictingMarkers = errors.New("3MF contains conflicting source-family metadata")

// Result is a metadata classification. It is NOT a conversion readiness verdict.
type Result struct {
	Family   Family   `json:"family"`
	Evidence []string `json:"evidence"`
}

// Detect inspects archive filenames only; it never imports, slices or executes a
// project, and never touches a slicer installation or hardware.
func Detect(filename string) (Result, error) {
	st, err := os.Stat(filename)
	if err != nil {
		return Result{}, err
	}
	if !st.Mode().IsRegular() || st.Size() == 0 || st.Size() > maxSourceBytes {
		return Result{}, fmt.Errorf("3MF must be a nonempty regular file up to 250 MiB")
	}
	r, err := zip.OpenReader(filename)
	if err != nil {
		return Result{}, fmt.Errorf("opening 3MF: %w", err)
	}
	defer r.Close()
	return DetectArchive(&r.Reader)
}

// DetectArchive accepts zip.Reader for deterministic tests and future use on
// in-memory 3MF bytes. Unknown does not imply invalid geometry or printability.
func DetectArchive(r *zip.Reader) (Result, error) {
	if r == nil {
		return Result{}, fmt.Errorf("missing 3MF archive")
	}
	if len(r.File) == 0 || len(r.File) > maxArchiveEntries {
		return Result{}, fmt.Errorf("invalid 3MF archive entry count")
	}
	names := map[string]bool{}
	hasModel := false
	bambu := false
	prusa := false
	evid := []string{}
	for _, f := range r.File {
		name := strings.ReplaceAll(f.Name, "\\", "/")
		clean := path.Clean(name)
		if name == "" || strings.HasPrefix(name, "/") || clean == ".." ||
			strings.HasPrefix(clean, "../") || clean != strings.TrimSuffix(name, "/") ||
			strings.ContainsRune(name, '\x00') {
			return Result{}, fmt.Errorf("unsafe 3MF archive entry: %q", f.Name)
		}
		// Casefold is used only for recognition; original casing stays untouched.
		key := strings.ToLower(strings.TrimSuffix(name, "/"))
		if names[key] {
			return Result{}, fmt.Errorf("duplicate 3MF archive entry: %q", f.Name)
		}
		names[key] = true
		if strings.HasPrefix(key, "3d/") && strings.HasSuffix(key, ".model") {
			hasModel = true
		}
		switch key {
		case "metadata/project_settings.config":
			bambu = true
			evid = append(evid, "Metadata/project_settings.config")
		case "metadata/slic3r_pe.config", "metadata/slic3r_pe_model.config":
			prusa = true
			evid = append(evid, f.Name)
		}
	}
	if !hasModel {
		return Result{}, fmt.Errorf("3MF has no 3D model entry")
	}
	if !names["[content_types].xml"] || !names["_rels/.rels"] {
		return Result{}, fmt.Errorf("3MF is missing required package relationships")
	}
	if bambu && prusa {
		return Result{}, ErrConflictingMarkers
	}
	switch {
	case bambu:
		return Result{Family: BambuOrca, Evidence: evid}, nil
	case prusa:
		return Result{Family: Prusa, Evidence: evid}, nil
	default:
		return Result{Family: Unknown, Evidence: []string{}}, nil
	}
}
