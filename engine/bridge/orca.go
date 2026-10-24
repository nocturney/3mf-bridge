package bridge

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/BryanSmee/bl_to_std/pkg/converter"
)

// orcaTargetID is the stable identifier the native resolver derives from one
// installed Orca-family machine preset. It can never name a built-in target.
var orcaTargetID = regexp.MustCompile(`^orca-[a-z0-9]+(?:-[a-z0-9]+)*-[0-9a-f]{10}$`)

// maxOrcaPhysicalTools is a sanity bound on installed hardware data. It is not
// a claim about any particular printer.
const maxOrcaPhysicalTools = 8

// orcaBundle adds the installed machine's physical tool count to the shared
// target bundle. The count is the length of the machine's nozzle_diameter array.
type orcaBundle struct {
	targetBundle
	PhysicalTools int `json:"physical_tools"`
}

// loadOrcaBundle validates an installed profile bundle without consulting the
// built-in target table. Every identity, nozzle and tool claim must agree.
func loadOrcaBundle(filename string) (*orcaBundle, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	if len(data) < 100 || len(data) > 4<<20 {
		return nil, fmt.Errorf("profile bundle has invalid size")
	}
	var bundle orcaBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, err
	}
	model := bundle.Profile.DisplayName
	if !orcaTargetID.MatchString(bundle.Target) || bundle.Profile.Name != bundle.Target ||
		model == "" || len(model) > 120 || bundle.Profile.ModelID != model {
		return nil, fmt.Errorf("profile bundle identity is invalid")
	}
	for _, builtIn := range targetModels {
		if builtIn == model {
			return nil, fmt.Errorf("%q is served by a built-in target engine", model)
		}
	}
	if bundle.PhysicalTools < 1 || bundle.PhysicalTools > maxOrcaPhysicalTools ||
		bundle.Profile.FilamentSlots != bundle.PhysicalTools {
		return nil, fmt.Errorf("physical tool count %d is not supported", bundle.PhysicalTools)
	}
	if bundle.Profile.ProjectSettings["printer_model"] != model {
		return nil, fmt.Errorf("machine model does not match the profile identity")
	}
	nozzles, _ := bundle.Profile.ProjectSettings["nozzle_diameter"].([]any)
	if len(nozzles) != bundle.PhysicalTools {
		return nil, fmt.Errorf("nozzle_diameter lists %d tools, expected %d", len(nozzles), bundle.PhysicalTools)
	}
	detected, err := nozzleFromRaw(nozzles)
	if err != nil || detected == "" || detected != bundle.Nozzle {
		return nil, fmt.Errorf("nozzle_diameter does not match bundled %s mm nozzle", bundle.Nozzle)
	}
	if err := bundle.Profile.Validate(); err != nil {
		return nil, err
	}
	if _, ok := bundle.Profile.ProjectSettings["machine_start_gcode"].(string); !ok {
		return nil, fmt.Errorf("machine start script missing")
	}
	if _, ok := bundle.Profile.ProjectSettings["machine_end_gcode"].(string); !ok {
		return nil, fmt.Errorf("machine end script missing")
	}
	if bundle.Profile.FilamentProfiles["PLA"] == "" || len(bundle.MaterialSettings) == 0 {
		return nil, fmt.Errorf("installed PLA material preset missing")
	}
	return &bundle, nil
}

// ConvertWithOrcaBundle converts through an installed Orca-family profile
// bundle prepared by the native resolver. The bundle supplies the target ID,
// printer model, physical nozzle and physical tool count; source slots beyond
// the physical tools stay virtual, exactly as in the built-in engines.
func ConvertWithOrcaBundle(source, destination, profileFile string) (map[string]any, error) {
	src, dst, err := resolvePaths(source, destination)
	if err != nil {
		return nil, err
	}
	bundle, err := loadOrcaBundle(profileFile)
	if err != nil {
		return nil, fail("TARGET_PROFILE_INVALID", "installed profile: %v", err)
	}
	before, err := inspectPackage(src)
	if err != nil {
		return nil, fail("SOURCE_INVALID", "%v", err)
	}
	sourceNozzle, err := detectSourceNozzle(src)
	if err != nil {
		return nil, fail("SOURCE_NOZZLE_INVALID", "%v", err)
	}
	insp, err := converter.Inspect(src)
	if err != nil {
		return nil, fail("SOURCE_INVALID", "cannot read source filament assignments: %v", err)
	}
	if len(insp.Filaments) == 0 {
		return nil, fail("NO_FILAMENTS", "source contains no usable filament data")
	}
	slots, mapping, err := sourceSlotPlan(insp.Filaments)
	if err != nil {
		return nil, fail("SOURCE_FILAMENTS_INVALID", "%v", err)
	}
	if len(slots) > bundle.Profile.FilamentSlots {
		bundle.Profile.FilamentSlots = len(slots)
	}
	if err = bindNativeFilamentSettings(&bundle.targetBundle, insp.Filaments); err != nil {
		return nil, fail("MATERIAL_REVIEW_REQUIRED", "target %s: %v", bundle.Target, err)
	}
	// Creality Print 7.3 refuses a valid multi-filament project when the
	// installed single-nozzle preset's support flags still have one entry.
	if err = normalizeCrealitySupportFlags(bundle, len(insp.Filaments)); err != nil {
		return nil, fail("TARGET_PROFILE_INVALID", "target %s: %v", bundle.Target, err)
	}
	if err = normalizeCrealityFilamentDiameters(bundle, len(insp.Filaments)); err != nil {
		return nil, fail("TARGET_PROFILE_INVALID", "target %s: %v", bundle.Target, err)
	}
	if err = normalizeFlashforgeCreator5ProSupportFlags(bundle, len(insp.Filaments)); err != nil {
		return nil, fail("TARGET_PROFILE_INVALID", "target %s: %v", bundle.Target, err)
	}
	if err = normalizeObservedOrcaMultitoolSupportFlags(bundle, len(insp.Filaments)); err != nil {
		return nil, fail("TARGET_PROFILE_INVALID", "target %s: %v", bundle.Target, err)
	}
	if err = normalizeObservedOrcaLegacyProjectCompatibility(bundle); err != nil {
		return nil, fail("TARGET_PROFILE_INVALID", "target %s: %v", bundle.Target, err)
	}
	result, err := publishConversion(conversionPlan{
		source: src, destination: dst, target: bundle.Target,
		expectedModel: bundle.Profile.ModelID,
		base:          &bundle.Profile, filaments: insp.Filaments, slots: slots, mapping: mapping,
		before: before, physicalTools: bundle.PhysicalTools, sourceNozzle: sourceNozzle,
		targetNozzle: bundle.Nozzle, requestedNozzle: bundle.Nozzle,
	})
	if err != nil {
		return nil, err
	}
	result["profile_kind"] = "installed_orca_json"
	return result, nil
}

// normalizeCrealitySupportFlags fixes one observed Creality Print 7.3 CLI
// incompatibility without modifying any other Orca fork, machine preset,
// physical nozzle, filament selection, or source project. A single vendor
// default may be repeated for virtual slots, but conflicting values are
// never guessed or remapped.
func normalizeCrealitySupportFlags(bundle *orcaBundle, sourceSlots int) error {
	if bundle.Slicer != "Creality Print" || !strings.HasPrefix(bundle.Profile.ModelID, "Creality ") ||
		sourceSlots <= bundle.PhysicalTools {
		return nil
	}
	// bl_to_std defaults missing support-only flags to one "0" entry.
	// Write the same documented default for each virtual filament before
	// conversion, rather than allowing a one-entry fallback in the output.
	raw := []any{"0"}
	if value, exists := bundle.Profile.ProjectSettings["filament_is_support"]; exists {
		switch flags := value.(type) {
		case []any:
			if len(flags) == 0 {
				return fmt.Errorf("Creality Print support flags are empty")
			}
			raw = flags
		case string:
			if flags != "0" && flags != "1" {
				return fmt.Errorf("Creality Print support flag %q is invalid", flags)
			}
			raw = []any{flags}
		default:
			return fmt.Errorf("Creality Print support flags are malformed")
		}
	}
	if len(raw) == sourceSlots {
		return nil
	}
	if len(raw) != 1 {
		return fmt.Errorf("Creality Print support flags have %d values for %d filaments", len(raw), sourceSlots)
	}
	flags := make([]any, sourceSlots)
	for i := range flags {
		flags[i] = raw[0]
	}
	bundle.Profile.ProjectSettings["filament_is_support"] = flags
	return nil
}

// normalizeCrealityFilamentDiameters repeats the installed single-tool
// filament diameter for each preserved source slot. Creality Print 7.3's
// G-code tool writer requires an entry per virtual material; otherwise the
// 3MF imports with the correct palette but silently slices only material T0.
// This changes neither the physical nozzle nor any saved slicer preference.
func normalizeCrealityFilamentDiameters(bundle *orcaBundle, sourceSlots int) error {
	if bundle.Slicer != "Creality Print" || !strings.HasPrefix(bundle.Profile.ModelID, "Creality ") ||
		bundle.PhysicalTools != 1 || sourceSlots <= 1 {
		return nil
	}
	raw, ok := bundle.Profile.ProjectSettings["filament_diameter"]
	if !ok {
		return fmt.Errorf("installed Creality Print filament diameter is missing")
	}
	var diameter any
	switch v := raw.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("installed Creality Print filament diameter is empty")
		}
		diameter = v
	case []any:
		if len(v) == sourceSlots {
			return nil
		}
		if len(v) != 1 || v[0] == nil {
			return fmt.Errorf("Creality filament diameters have %d values for %d slots", len(v), sourceSlots)
		}
		diameter = v[0]
	default:
		return fmt.Errorf("installed Creality Print filament diameter is malformed")
	}
	values := make([]any, sourceSlots)
	for i := range values {
		values[i] = diameter
	}
	bundle.Profile.ProjectSettings["filament_diameter"] = values
	return nil
}

// normalizeFlashforgeCreator5ProSupportFlags fixes a verified *project-output*
// inconsistency in Flash Studio 2.4.2. Its four-tool Creator 5 Pro preset may
// omit the per-material support flags; the slicer then expands filament_colour
// to 4 tool slots but keeps only one implicit support flag and rejects the 3MF.
// Repeat only the slicer's known 0 default. Do not edit installed printer
// profiles, nozzle diameters, physical tool assignments, or other Orca forks.
func normalizeFlashforgeCreator5ProSupportFlags(bundle *orcaBundle, sourceSlots int) error {
	if bundle.Slicer != "Flash Studio" ||
		bundle.Profile.ModelID != "Flashforge Creator 5 Pro" || bundle.PhysicalTools != 4 {
		return nil
	}
	count := max(bundle.PhysicalTools, sourceSlots)
	if raw, exists := bundle.Profile.ProjectSettings["filament_is_support"]; exists {
		flags, ok := raw.([]any)
		if !ok {
			return fmt.Errorf("Flashforge Creator 5 Pro support flag type is invalid")
		}
		if len(flags) == count {
			return nil // Preserve manufacturer-provided per-slot flags verbatim.
		}
		if len(flags) != 1 || fmt.Sprint(flags[0]) != "0" {
			return fmt.Errorf("Flashforge Creator 5 Pro support flags mismatch %d material slots", count)
		}
	}
	flags := make([]any, count)
	for i := range flags {
		flags[i] = "0"
	}
	bundle.Profile.ProjectSettings["filament_is_support"] = flags
	return nil
}

// normalizeObservedOrcaMultitoolSupportFlags fixes the documented native
// Orca 2.4.2 exit-251 validator for physical multi-tool profiles of four
// installed vendor families. With missing support flags, the upstream
// converter emits a one-slot implicit zero while filament_colour contains
// max(physical tools, source slots) entries. A disposable QA 3MF with only
// this output field aligned produced real nozzle/model-matching G-code.
// This is output-only; manufacturer presets and physical tool/nozzle arrays
// are never rewritten. Other slicers and vendors remain unchanged.
func normalizeObservedOrcaMultitoolSupportFlags(bundle *orcaBundle, sourceSlots int) error {
	if bundle.Slicer != "OrcaSlicer" || bundle.PhysicalTools < 2 {
		return nil
	}
	observed := false
	for _, prefix := range []string{"orca-prusa-", "orca-ratrig-", "orca-volumic-", "orca-re3d-"} {
		if strings.HasPrefix(bundle.Target, prefix) {
			observed = true
			break
		}
	}
	if !observed {
		return nil
	}
	count := max(bundle.PhysicalTools, sourceSlots)
	if raw, exists := bundle.Profile.ProjectSettings["filament_is_support"]; exists {
		switch flags := raw.(type) {
		case []any:
			if len(flags) == count {
				return nil // Preserve explicit manufacturer per-slot flags verbatim.
			}
			if len(flags) != 1 || fmt.Sprint(flags[0]) != "0" {
				return fmt.Errorf("Orca multi-tool support flags mismatch %d material slots", count)
			}
		case string:
			if flags != "0" {
				return fmt.Errorf("Orca multi-tool support flag %q cannot be expanded", flags)
			}
		default:
			return fmt.Errorf("Orca multi-tool support flag type is invalid")
		}
	}
	flags := make([]any, count)
	for i := range flags {
		flags[i] = "0" // The converter's existing implicit default, made per-slot.
	}
	bundle.Profile.ProjectSettings["filament_is_support"] = flags
	return nil
}

// normalizeObservedOrcaLegacyProjectCompatibility addresses only four locally
// reproduced OrcaSlicer 2.4.2 CLI failures. It modifies the in-memory project
// assembled for conversion, NEVER installed printer presets or user settings.
// The script changes are guarded by exact vendor/model and observed script text:
// unknown/new manufacturer variants are left untouched, not guessed.
//
// No motion, homing, heater, or physical nozzle setting is modified.
//   - Three Anycubic models require resetting E after each layer when Orca emits
//     relative extrusion. Their installed script has only commented instructions.
//   - Dremel's legacy G92 E0 before each layer conflicts with absolute E mode.
//   - FLSun's M117 LCD message contains an unsupported filename placeholder.
//   - RH3D's firmware retraction is incompatible with simultaneous wipe.
func normalizeObservedOrcaLegacyProjectCompatibility(bundle *orcaBundle) error {
	if bundle.Slicer != "OrcaSlicer" || bundle.PhysicalTools != 1 {
		return nil
	}
	cfg := bundle.Profile.ProjectSettings
	model := bundle.Profile.ModelID
	switch {
	case strings.HasPrefix(bundle.Target, "orca-anycubic-") &&
		(model == "Anycubic Kobra Max" || model == "Anycubic Kobra Plus" || model == "Anycubic Vyper"):
		script, ok := cfg["layer_change_gcode"].(string)
		if !ok {
			return fmt.Errorf("installed Anycubic layer change script is missing")
		}
		reset := regexp.MustCompile(`(?m)^[ \t]*G92[ \t]+E0(?:\.0+)?(?:[ \t]*;[^\n]*)?[ \t]*$`)
		if reset.MatchString(script) {
			return nil // Installed profile already has an active reset; never duplicate.
		}
		if strings.TrimSpace(script) != ";AFTER_LAYER_CHANGE\n;[layer_z]" {
			return nil // Unknown vendor customization: preserve it unchanged.
		}
		cfg["layer_change_gcode"] = strings.TrimRight(script, "\r\n") + "\nG92 E0\n"
	case strings.HasPrefix(bundle.Target, "orca-dremel-") &&
		(model == "Dremel 3D40" || model == "Dremel 3D45"):
		const old = ";BEFORE_LAYER_CHANGE\n;[layer_z]\nG92 E0\n"
		if script, ok := cfg["before_layer_change_gcode"].(string); ok && script == old {
			cfg["before_layer_change_gcode"] = ";BEFORE_LAYER_CHANGE\n;[layer_z]\n"
		}
	case strings.HasPrefix(bundle.Target, "orca-flsun-") && model == "FLSun QQ-S Pro":
		script, ok := cfg["machine_start_gcode"].(string)
		if !ok {
			return fmt.Errorf("installed FLSun start script is missing")
		}
		const legacyLine = "M117 Print [output_filename_format]"
		if strings.Count(script, legacyLine) == 1 {
			cfg["machine_start_gcode"] = strings.Replace(script, legacyLine, "M117 Print started", 1)
		} // Otherwise preserve the installed script without guessing replacements.
	case strings.HasPrefix(bundle.Target, "orca-rh3d-") && model == "E3NG v1.2S":
		if cfg["use_firmware_retraction"] != "1" {
			return nil
		}
		switch cfg["wipe"] {
		case "1":
			cfg["wipe"] = "0" // Native Orca rejects firmware retraction plus wipe.
		case "0":
			return nil
		default:
			return fmt.Errorf("RH3D firmware retraction has unknown wipe setting")
		}
	}
	return nil
}
