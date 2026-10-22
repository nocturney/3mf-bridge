package presets

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// SlicerChoice is an installed, trusted slicer location discovered on this host.
// The renderer selects only one of these hard-coded IDs, never an executable path.
type SlicerChoice struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
}
type slicerLocation struct {
	id, name, root, executable string
}

// appInstall is one install name of an application. Windows folders sit under each
// install base; macOS bundle names sit in each application directory.
type appInstall struct {
	id, name, folder, exe string
}

// appGroups lists a target's applications in priority order. Vendor applications come
// before generic Orca. Alternative install names of one app share a group.
func appGroups(target, platform string) [][]appInstall {
	switch platform {
	case "windows":
		switch target {
		case "u1":
			return [][]appInstall{{{"snapmaker-orca", "Snapmaker Orca", "Snapmaker_Orca", "snapmaker-orca.exe"}}}
		case "c5":
			return [][]appInstall{{{"flash-studio", "Flash Studio Desktop", filepath.Join("Flashforge", "Flash Studio Desktop"), "flash studio.exe"}}}
		case "cc2":
			return [][]appInstall{{{"elegoo-slicer", "ElegooSlicer", "ElegooSlicer", "elegoo-slicer.exe"}}}
		case "qidi":
			return [][]appInstall{
				{{"qidi-studio", "QIDIStudio", "QIDIStudio", "QIDIStudio.exe"}},
				{{"orca-slicer", "OrcaSlicer", "OrcaSlicer", "orca-slicer.exe"}},
			}
		}
	case "darwin":
		switch target {
		case "u1":
			return [][]appInstall{{{"snapmaker-orca", "Snapmaker Orca", "Snapmaker Orca", ""}}}
		case "c5":
			return [][]appInstall{{
				{"flash-studio", "Flash Studio", "Flash Studio", ""},
				{"flash-studio", "Flash Studio Desktop", "Flash Studio Desktop", ""},
			}}
		case "cc2":
			return [][]appInstall{{{"elegoo-slicer", "ElegooSlicer", "ElegooSlicer", ""}}}
		case "qidi":
			return [][]appInstall{
				{{"qidi-studio", "QIDIStudio", "QIDIStudio", ""}},
				{{"orca-slicer", "OrcaSlicer", "OrcaSlicer", ""}},
			}
		}
	}
	return nil
}

// slicerLocationsForOS builds a bounded list of known install locations.
// Candidate paths are local only; no registry scans, network access, or
// executable paths supplied by untrusted frontend input.
// Applications are ordered by priority group, and every install base is checked
// inside a group before the next group. So a vendor app in LocalAppData still
// beats a generic Orca in Program Files. An unknown target yields no paths.
func slicerLocationsForOS(target, platform, programFiles, programFilesX86, localAppData, home, override string) []slicerLocation {
	if override != "" {
		return []slicerLocation{{"system", "Custom local profile", override, ""}}
	}
	result := []slicerLocation{}
	seen := map[string]bool{}
	add := func(id, name, root, executable string) {
		if root == "" || seen[root] {
			return
		}
		seen[root] = true
		result = append(result, slicerLocation{id, name, root, executable})
	}
	var bases []string
	switch platform {
	case "windows":
		// Keep the original Program Files priority. LocalAppData/Programs
		// also hosts legitimate per-user installs without admin privileges.
		bases = []string{programFiles, programFilesX86}
		if localAppData != "" {
			bases = append(bases, filepath.Join(localAppData, "Programs"))
		}
	case "darwin":
		bases = []string{"/Applications"}
		if home != "" {
			bases = append(bases, filepath.Join(home, "Applications"))
		}
	}
	for _, group := range appGroups(target, platform) {
		for _, base := range bases {
			if base == "" {
				continue
			}
			for _, app := range group {
				if platform == "windows" {
					dir := filepath.Join(base, app.folder)
					add(app.id, app.name, filepath.Join(dir, "resources", "profiles"), filepath.Join(dir, app.exe))
					continue
				}
				bundle := filepath.Join(base, app.folder+".app")
				add(app.id, app.name, filepath.Join(bundle, "Contents", "Resources", "profiles"), bundle)
			}
		}
	}
	return result
}

func slicerLocations(target string) []slicerLocation {
	pf := os.Getenv("ProgramFiles")
	if runtime.GOOS == "windows" && pf == "" {
		pf = `C:\Program Files`
	}
	home, _ := os.UserHomeDir()
	return slicerLocationsForOS(target, runtime.GOOS, pf, os.Getenv("ProgramFiles(x86)"),
		os.Getenv("LOCALAPPDATA"), home, os.Getenv("BRIDGE_SLICER_PROFILE_"+strings.ToUpper(target)))
}
func Slicers(target string) []SlicerChoice {
	spec, ok := Specs[target]
	if !ok {
		return nil
	}
	items := make([]SlicerChoice, 0, 3)
	positions := map[string]int{}
	for _, loc := range slicerLocations(target) {
		info, err := os.Stat(filepath.Join(loc.root, spec.Vendor))
		installed := err == nil && info.IsDir()
		if at, exists := positions[loc.id]; exists {
			items[at].Installed = items[at].Installed || installed
			continue
		}
		positions[loc.id] = len(items)
		items = append(items, SlicerChoice{ID: loc.id, Name: loc.name, Installed: installed})
	}
	return items
}
func ResolveSlicer(target, id string) (root, execPath, slicerName string, err error) {
	spec, ok := Specs[target]
	if !ok {
		return "", "", "", fmt.Errorf("unsupported target %q", target)
	}
	return resolveSlicerAt(slicerLocations(target), spec, id)
}

// resolveSlicerAt takes the first location, in priority order, whose vendor folder
// exists. Fails closed when none matches the preference.
func resolveSlicerAt(locations []slicerLocation, spec Spec, id string) (root, execPath, slicerName string, err error) {
	if id == "" {
		id = "auto"
	}
	for _, loc := range locations {
		if id != "auto" && id != loc.id {
			continue
		}
		if s, e := os.Stat(filepath.Join(loc.root, spec.Vendor)); e != nil || !s.IsDir() {
			continue
		}
		return loc.root, loc.executable, loc.name, nil
	}
	return "", "", "", fmt.Errorf("no installed %s slicer matching preference %q", spec.Name, id)
}
func IsValidSlicer(target, id string) bool {
	if id == "" || id == "auto" {
		return true
	}
	for _, choice := range slicerLocations(target) {
		if choice.id == id {
			return true
		}
	}
	return false
}

// FilamentPresets lists real vendor JSON names that resolve to a matching
// material type and nozzle. These names are the *only* accepted overrides.
func FilamentPresets(target, nozzle, slicerID, material string) ([]string, error) {
	spec, ok := Specs[target]
	if !ok {
		return nil, fmt.Errorf("unknown target %q", target)
	}
	if material != "PLA" && material != "PETG" && material != "ABS" && material != "ASA" && material != "TPU" {
		return nil, fmt.Errorf("unsupported material type %q", material)
	}
	root, _, _, err := ResolveSlicer(target, slicerID)
	if err != nil {
		return nil, err
	}
	ix, err := readCatalog(root, spec)
	if err != nil {
		return nil, err
	}
	_, _, _, defaults, err := Variant(target, nozzle)
	if err != nil {
		return nil, err
	}
	results := []string{}
	seen := map[string]bool{}
	for key := range ix {
		if !strings.HasPrefix(key, "filament:") {
			continue
		}
		name := strings.TrimPrefix(key, "filament:")
		if seen[name] {
			continue
		}
		if !filamentCompatibleName(target, name, nozzle) {
			continue
		}
		preset, err := flatten(ix, "filament", name, map[string]bool{})
		if err != nil {
			continue
		}
		raw, ok := preset["filament_type"].([]any)
		if !ok || len(raw) == 0 || fmt.Sprint(raw[0]) != material {
			continue
		}
		// Machine/profile compatibility must be asserted by its installed
		// canonical name; generic presets from other printer families
		// must never silently bind to this model.
		if len(preset) == 0 {
			continue
		}
		seen[name] = true
		results = append(results, name)
	}
	sort.Strings(results)
	if fallback := defaults[material]; fallback != "" {
		for i, n := range results {
			if n == fallback {
				results = append([]string{n}, append(results[:i], results[i+1:]...)...)
				break
			}
		}
	}
	return results, nil
}

func filamentCompatibleName(target, name, nozzle string) bool {
	marker := ""
	switch target {
	case "u1":
		marker = "@U1"
	case "c5":
		marker = "@FF C5"
	case "cc2":
		marker = "@ECC2"
	case "qidi":
		marker = "@Qidi X-Plus 5"
	}
	if marker == "" || !strings.Contains(name, marker) {
		return false
	}
	// If a native preset names a nozzle, it must match the physical nozzle
	// selected for the target. Generic base presets may inherit freely.
	lower := strings.ToLower(name)
	for _, candidate := range []string{"0.2", "0.25", "0.4", "0.6", "0.8"} {
		if strings.Contains(lower, candidate+" nozzle") && candidate != nozzle {
			return false
		}
	}
	return true
}

func ValidateOverrides(target, nozzle, slicerID string, overrides map[string]string) error {
	if len(overrides) == 0 {
		return nil
	}
	for material, name := range overrides {
		if name == "" {
			continue
		}
		choices, err := FilamentPresets(target, nozzle, slicerID, material)
		if err != nil {
			return err
		}
		found := false
		for _, option := range choices {
			if option == name {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%s profile %q is not an installed, matching %s %smm preset", material, name, target, nozzle)
		}
	}
	return nil
}
