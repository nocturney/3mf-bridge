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

// sourceSlotPlan preserves distinct color/material assignments. More than four
// Bambu source IDs can be converted only when the extra IDs are EXACT matches
// of another source color and material, never by nearest-color substitution.
// Projects with <= four IDs retain their established, injective mapping.
func sourceSlotPlan(filaments []converter.Filament, maxSlots int) ([]converter.Slot, map[int]int, []converter.Filament, int, error) {
	if len(filaments) == 0 || maxSlots <= 0 {
		return nil, nil, nil, 0, fmt.Errorf("source has no usable material slots")
	}
	type identity struct {
		color    string
		material string
	}
	slots := make([]converter.Slot, 0, maxSlots)
	mapped := make(map[int]int, len(filaments))
	unique := make([]converter.Filament, 0, maxSlots)
	seen := make(map[identity]int)
	compact := len(filaments) > maxSlots
	for _, filament := range filaments {
		if filament.ID <= 0 {
			return nil, nil, nil, 0, fmt.Errorf("invalid source filament ID %d", filament.ID)
		}
		if _, duplicate := mapped[filament.ID]; duplicate {
			return nil, nil, nil, 0, fmt.Errorf("repeated source filament ID %d", filament.ID)
		}
		key := identity{color: strings.ToUpper(filament.Color), material: filament.Type}
		if compact {
			if slot, exists := seen[key]; exists {
				mapped[filament.ID] = slot
				continue
			}
		}
		if len(slots) >= maxSlots {
			return nil, nil, nil, 0, fmt.Errorf(
				"source has %d slots with more than %d distinct color/material combinations; refusing to merge different materials or colors",
				len(filaments), maxSlots)
		}
		slot := len(slots) + 1
		seen[key] = slot
		mapped[filament.ID] = slot
		slots = append(slots, converter.Slot{
			Color: filament.Color, Type: filament.Type,
			Support: converter.IsSupportType(filament.Type, ""),
		})
		unique = append(unique, filament)
	}
	return slots, mapped, unique, len(filaments) - len(unique), nil
}

// Convert keeps the original validated U1 entrypoint for compatibility.
func Convert(source, destination, target string) (map[string]any, error) {
	return ConvertWithProfile(source, destination, target, "")
}

// ConvertWithProfile uses manufacturer slicer presets for C5/CC2/QIDI.
// A missing or mismatched installed profile fails closed.
func ConvertWithProfile(source, destination, target, profileFile string) (map[string]any, error) {
	return ConvertWithTargetNozzle(source, destination, target, profileFile, "")
}

// ConvertWithTargetNozzle allows the operator to choose the target's actual
// installed physical nozzle. The default retains historical source matching;
// an explicit target selection is validated against manufacturer sizes.
func ConvertWithTargetNozzle(source, destination, target, profileFile, requestedNozzle string) (map[string]any, error) {
	target = strings.ToLower(target)
	if targetModels[target] == "" {
		return nil, fail("TARGET_NOT_VALIDATED", "unknown target %q", target)
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
	sourceNozzle, err := detectSourceNozzle(src)
	if err != nil {
		return nil, fail("SOURCE_NOZZLE_INVALID", "%v", err)
	}
	targetNozzle := requestedNozzle
	if targetNozzle == "" {
		targetNozzle, err = targetNozzleForSource(target, sourceNozzle.Nozzle)
		if err != nil {
			return nil, fail("TARGET_NOZZLE_UNAVAILABLE", "%v", err)
		}
	} else if !validTargetNozzle(target, targetNozzle) {
		return nil, fail("TARGET_NOZZLE_UNAVAILABLE", "target %s does not support physical nozzle %q", target, requestedNozzle)
	}
	if (target != "u1" || targetNozzle != "0.4") && profileFile == "" {
		return nil, fail("TARGET_PROFILE_REQUIRED", "installed %s %smm slicer nozzle profile is required", target, targetNozzle)
	}
	insp, err := converter.Inspect(src)
	if err != nil {
		return nil, fail("SOURCE_INVALID", "cannot read source filament assignments: %v", err)
	}
	if len(insp.Filaments) == 0 {
		return nil, fail("NO_FILAMENTS", "source contains no usable filament data")
	}
	slots, mapping, uniqueFilaments, folded, err := sourceSlotPlan(insp.Filaments, 4)
	if err != nil {
		return nil, fail("COLOR_MAPPING_REVIEW_REQUIRED", "%v", err)
	}
	var base *printer.Profile
	var bundle *targetBundle
	if target == "u1" && targetNozzle == "0.4" {
		// Keep the already field-validated 0.4mm U1 engine behavior unchanged.
		base = printer.Builtin("snapmaker-u1")
	} else {
		bundle, err = loadTargetBundle(profileFile, target)
		if err != nil {
			return nil, fail("TARGET_PROFILE_INVALID", "target %s installed slicer profile: %v", target, err)
		}
		if bundle.Nozzle != targetNozzle {
			return nil, fail("TARGET_NOZZLE_MISMATCH",
				"requested %s nozzle %s mm but profile bundle is %s mm", target, targetNozzle, bundle.Nozzle)
		}
		if err = bindNativeFilamentSettings(bundle, uniqueFilaments); err != nil {
			return nil, fail("MATERIAL_REVIEW_REQUIRED", "target %s: %v", target, err)
		}
		base = &bundle.Profile
	}
	if base == nil {
		return nil, fail("ENGINE_INVALID", "target base profile missing")
	}
	merged, preserved, err := preservedProfile(src, base)
	if err != nil {
		return nil, fail("SOURCE_INVALID", "%v", err)
	}
	for _, filament := range insp.Filaments {
		if _, supported := merged.FilamentProfiles[filament.Type]; !supported {
			return nil, fail("MATERIAL_REVIEW_REQUIRED", "unsupported material %q in source slot %d; refusing silent PLA substitution", filament.Type, filament.ID)
		}
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
	expectedModel := "Snapmaker U1"
	if target != "u1" {
		expectedModel = targetModels[target]
	}
	if after.printerModel != expectedModel {
		return nil, fail("TARGET_IDENTITY_INVALID", "converted project has unexpected printer model %q (expected %q)", after.printerModel, expectedModel)
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
	warnings := []string{
		fmt.Sprintf("Open this project in the target slicer for %s and re-slice before printing.", expectedModel),
		"Review material slots, supports, nozzle and user print parameters; printer-specific firmware scripts come from the installed target profile.",
	}
	if folded > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"%d redundant source filament slots were coalesced because their colors and material types match exactly; review spool assignments.", folded))
	}
	return map[string]any{
		"ok":                               true,
		"target":                           target,
		"output":                           dst,
		"engine":                           "bl2std",
		"source_nozzle":                    sourceNozzle.Nozzle,
		"source_nozzle_source":             sourceNozzle.Source,
		"source_nozzle_assumed":            sourceNozzle.Assumed,
		"target_nozzle":                    targetNozzle,
		"nozzle_remapped":                  sourceNozzle.Nozzle != targetNozzle,
		"nozzle_selected_by_user":          requestedNozzle != "",
		"source_filament_slots":            len(insp.Filaments),
		"target_filament_slots_used":       len(slots),
		"identical_source_slots_coalesced": folded,
		"source_to_target_slot_map":        mapping,
		"geometry_vertices":                after.vertices,
		"geometry_triangles":               after.triangles,
		"removed_slice_cache_entries":      removed,
		"preserved_print_keys":             preserved,
		"warnings":                         warnings,
	}, nil
}
