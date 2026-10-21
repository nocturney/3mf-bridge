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

// slicerLocationsForOS builds a bounded list of known install locations.
// Candidate paths are local only; no registry scans, network access, or
// executable paths supplied by untrusted frontend input.
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
    if platform == "windows" {
        // Keep the original Program Files priority. LocalAppData/Programs
        // also hosts legitimate per-user installs without admin privileges.
        bases := []string{programFiles, programFilesX86}
        if localAppData != "" {
            bases = append(bases, filepath.Join(localAppData, "Programs"))
        }
        for _, base := range bases {
            if base == "" {
                continue
            }
            app := func(id, name, folder, exe string) {
                dir := filepath.Join(base, folder)
                add(id, name, filepath.Join(dir, "resources", "profiles"), filepath.Join(dir, exe))
            }
            switch target {
            case "u1":
                app("snapmaker-orca", "Snapmaker Orca", "Snapmaker_Orca", "snapmaker-orca.exe")
            case "c5":
                dir := filepath.Join("Flashforge", "Flash Studio Desktop")
                app("flash-studio", "Flash Studio Desktop", dir, "flash studio.exe")
            case "cc2":
                app("elegoo-slicer", "ElegooSlicer", "ElegooSlicer", "elegoo-slicer.exe")
            case "qidi":
                app("orca-slicer", "OrcaSlicer", "OrcaSlicer", "orca-slicer.exe")
                app("qidi-studio", "QIDIStudio", "QIDIStudio", "QIDIStudio.exe")
            }
        }
    }
    if platform == "darwin" {
        bases := []string{"/Applications"}
        if home != "" {
            bases = append(bases, filepath.Join(home, "Applications"))
        }
        for _, base := range bases {
            app := func(id, name, bundleName string) {
                bundle := filepath.Join(base, bundleName+".app")
                add(id, name, filepath.Join(bundle, "Contents", "Resources", "profiles"), bundle)
            }
            switch target {
            case "u1":
                app("snapmaker-orca", "Snapmaker Orca", "Snapmaker Orca")
            case "c5":
                app("flash-studio", "Flash Studio", "Flash Studio")
                app("flash-studio", "Flash Studio Desktop", "Flash Studio Desktop")
            case "cc2":
                app("elegoo-slicer", "ElegooSlicer", "ElegooSlicer")
            case "qidi":
                app("qidi-studio", "QIDIStudio", "QIDIStudio")
                app("orca-slicer", "OrcaSlicer", "OrcaSlicer")
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
	if id == "" {
		id = "auto"
	}
	for _, loc := range slicerLocations(target) {
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
