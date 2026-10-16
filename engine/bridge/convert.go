package bridge

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BryanSmee/bl_to_std/pkg/converter"
	"github.com/BryanSmee/bl_to_std/pkg/printer"
)

type codedError struct {
	code  string
	cause error
}

func (e *codedError) Error() string { return e.cause.Error() }
func (e *codedError) Unwrap() error { return e.cause }
func fail(code string, format string, args ...any) error {
	return &codedError{code: code, cause: fmt.Errorf(format, args...)}
}
func ErrorCode(err error) string {
	var ce *codedError
	if errors.As(err, &ce) {
		return ce.code
	}
	return "CONVERSION_FAILED"
}

// Convert is deliberately fail-closed: only the U1 adapter is validated here.
// Unverified targets must not be represented by a renamed copy of the input.
func Convert(source, destination, target string) (map[string]any, error) {
	if strings.ToLower(target) != "u1" {
		return nil, fail("TARGET_NOT_VALIDATED", "target %q has no verified profile adapter", target)
	}
	src, err := filepath.Abs(source)
	if err != nil {
		return nil, fail("SOURCE_INVALID", "%v", err)
	}
	dst, err := filepath.Abs(destination)
	if err != nil {
		return nil, fail("OUTPUT_INVALID", "%v", err)
	}
	if strings.EqualFold(src, dst) {
		return nil, fail("SOURCE_EQUALS_OUTPUT", "source and destination cannot be the same file")
	}
	if !strings.EqualFold(filepath.Ext(src), ".3mf") || !strings.EqualFold(filepath.Ext(dst), ".3mf") {
		return nil, fail("EXTENSION_INVALID", "input and output must have .3mf extensions")
	}
	before, err := inspectPackage(src)
	if err != nil {
		return nil, fail("SOURCE_INVALID", "%v", err)
	}
	insp, err := converter.Inspect(src)
	if err != nil {
		return nil, fail("SOURCE_INVALID", "cannot read source filament assignments: %v", err)
	}
	if len(insp.Filaments) > 4 {
		return nil, fail("COLOR_MAPPING_REVIEW_REQUIRED", "source has %d material slots; automatic reduction to four could change the painted model", len(insp.Filaments))
	}
	if len(insp.Filaments) == 0 {
		return nil, fail("NO_FILAMENTS", "source contains no usable filament data")
	}
	base := printer.Builtin("snapmaker-u1")
	if base == nil {
		return nil, fail("ENGINE_INVALID", "upstream U1 profile missing")
	}
	merged, preserved, err := preservedProfile(src, base)
	if err != nil {
		return nil, fail("SOURCE_INVALID", "%v", err)
	}
	slots := make([]converter.Slot, 0, len(insp.Filaments))
	mapping := make(map[int]int, len(insp.Filaments))
	for i, filament := range insp.Filaments {
		if _, supported := merged.FilamentProfiles[filament.Type]; !supported {
			return nil, fail("MATERIAL_REVIEW_REQUIRED", "unsupported material %q in source slot %d; refusing silent PLA substitution", filament.Type, filament.ID)
		}
		slots = append(slots, converter.Slot{
			Color:   filament.Color,
			Type:    filament.Type,
			Support: converter.IsSupportType(filament.Type, ""),
		})
		mapping[filament.ID] = i + 1
	}
	outDir := filepath.Dir(dst)
	if st, err := os.Stat(outDir); err != nil || !st.IsDir() {
		return nil, fail("OUTPUT_INVALID", "output directory does not exist: %s", outDir)
	}
	raw, err := os.CreateTemp(outDir, ".bridge-conversion-*.3mf")
	if err != nil {
		return nil, fail("OUTPUT_INVALID", "%v", err)
	}
	rawPath := raw.Name()
	raw.Close()
	defer os.Remove(rawPath)
	clean, err := os.CreateTemp(outDir, ".bridge-sanitized-*.3mf")
	if err != nil {
		return nil, fail("OUTPUT_INVALID", "%v", err)
	}
	cleanPath := clean.Name()
	clean.Close()
	defer os.Remove(cleanPath)

	_, err = converter.Convert(src, rawPath, converter.Options{
		Printer: merged, Slots: slots, Mapping: mapping, Supports: converter.SupportsAuto,
	})
	if err != nil {
		return nil, fail("UPSTREAM_CONVERSION_FAILED", "bl2std: %v", err)
	}
	removed, err := stripSliceCache(rawPath, cleanPath, preserved)
	if err != nil {
		return nil, fail("CACHE_SANITIZE_FAILED", "%v", err)
	}
	after, err := inspectPackage(cleanPath)
	if err != nil {
		return nil, fail("OUTPUT_VALIDATION_FAILED", "%v", err)
	}
	if before.vertices != after.vertices || before.triangles != after.triangles || after.triangles == 0 {
		return nil, fail("GEOMETRY_CHANGED", "mesh triangle/vertex counts differ: input %d/%d, output %d/%d", before.vertices, before.triangles, after.vertices, after.triangles)
	}
	if after.sliceCaches != 0 {
		return nil, fail("STALE_GCODE", "converted project still contains sliced machine code")
	}
	if !strings.Contains(strings.ToLower(after.printerModel), "snapmaker u1") {
		return nil, fail("TARGET_IDENTITY_INVALID", "converted project has unexpected printer model %q", after.printerModel)
	}
	// A hard link in the same directory publishes atomically without replacing
	// an existing output. This protects the source and all previous conversions.
	if err := os.Link(cleanPath, dst); err != nil {
		if os.IsExist(err) {
			return nil, fail("OUTPUT_EXISTS", "output already exists: %s", dst)
		}
		return nil, fail("OUTPUT_PUBLISH_FAILED", "cannot publish without overwrite: %v", err)
	}
	sort.Strings(preserved)
	return map[string]any{
		"ok":                          true,
		"target":                      "u1",
		"output":                      dst,
		"engine":                      "bl2std",
		"source_filament_slots":       len(insp.Filaments),
		"geometry_vertices":           after.vertices,
		"geometry_triangles":          after.triangles,
		"removed_slice_cache_entries": removed,
		"preserved_print_keys":        preserved,
		"warnings": []string{
			"Open in Snapmaker Orca, review the complete project and re-slice before printing.",
			"Only a reviewed subset of source print parameters is transferred; machine and firmware values are taken from the U1 reference.",
		},
	}, nil
}
