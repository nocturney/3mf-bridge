package bridge

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

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
