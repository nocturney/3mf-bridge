package bridge

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
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
// the physical tools stay virtual. It shares ConvertTarget's plan and publish
// path and differs only in loader.
func ConvertWithOrcaBundle(source, destination, profileFile string) (map[string]any, error) {
	return convertWith(source, destination, func(NozzleInfo) (*ResolvedTargetProfile, error) {
		return resolveInstalledOrca(profileFile, "", "")
	})
}
