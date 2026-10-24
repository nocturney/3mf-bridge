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

// installBases returns the trusted install bases for a platform in priority order.
// Empty system directories are dropped, so they never turn into relative paths.
func installBases(platform, programFiles, programFilesX86, localAppData, home string) []string {
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
	result := []string{}
	for _, base := range bases {
		if base != "" {
			result = append(result, base)
		}
	}
	return result
}

// installLocation is the profile root and executable (Windows) or bundle (macOS)
// of one application folder inside an install base.
func installLocation(platform, base string, app appInstall) slicerLocation {
	if platform == "windows" {
		dir := filepath.Join(base, app.folder)
		return slicerLocation{app.id, app.name, filepath.Join(dir, "resources", "profiles"), filepath.Join(dir, app.exe)}
	}
	bundle := filepath.Join(base, app.folder+".app")
	return slicerLocation{app.id, app.name, filepath.Join(bundle, "Contents", "Resources", "profiles"), bundle}
}

// appendUnique adds a location once per profile root.
func appendUnique(result []slicerLocation, seen map[string]bool, loc slicerLocation) []slicerLocation {
	if loc.root == "" || seen[loc.root] {
		return result
	}
	seen[loc.root] = true
	return append(result, loc)
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
	bases := installBases(platform, programFiles, programFilesX86, localAppData, home)
	for _, group := range appGroups(target, platform) {
		for _, base := range bases {
			for _, app := range group {
				result = appendUnique(result, seen, installLocation(platform, base, app))
			}
		}
	}
	return result
}

// orcaFamilyApps is the bounded, explicit list of trusted applications whose
// installed resources/profiles tree may hold Orca-compatible machine, process and
// filament JSON. Vendor forks come first; universal OrcaSlicer is the last
// fallback. Each entry was verified against that layout. Bambu Studio is absent
// because it stays source-only, and apps with other layouts are not listed.
func orcaFamilyApps(platform string) []appInstall {
	switch platform {
	case "windows":
		return []appInstall{
			{"snapmaker-orca", "Snapmaker Orca", "Snapmaker_Orca", "snapmaker-orca.exe"},
			{"flash-studio", "Flash Studio Desktop", filepath.Join("Flashforge", "Flash Studio Desktop"), "flash studio.exe"},
			{"elegoo-slicer", "ElegooSlicer", "ElegooSlicer", "elegoo-slicer.exe"},
			{"qidi-studio", "QIDIStudio", "QIDIStudio", "QIDIStudio.exe"},
			// Creality Print 7.3 still installs under the 7.0 folder on Windows.
			{"creality-print", "Creality Print", filepath.Join("Creality", "Creality Print 7.0"), "CrealityPrint.exe"},
			{"creality-print", "Creality Print", filepath.Join("Creality", "Creality Print"), "CrealityPrint.exe"},
			// This is the Orca-based Slicer Next, not the old Prusa-based Anycubic Slicer.
			{"anycubic-slicer-next", "Anycubic Slicer Next", "AnycubicSlicerNext", "AnycubicSlicerNext.exe"},
			{"anycubic-slicer-next", "Anycubic Slicer Next", "Anycubic Slicer Next", "AnycubicSlicerNext.exe"},
			{"orca-slicer", "OrcaSlicer", "OrcaSlicer", "orca-slicer.exe"},
		}
	case "darwin":
		return []appInstall{
			{"snapmaker-orca", "Snapmaker Orca", "Snapmaker Orca", ""},
			{"flash-studio", "Flash Studio", "Flash Studio", ""},
			{"flash-studio", "Flash Studio Desktop", "Flash Studio Desktop", ""},
			{"elegoo-slicer", "ElegooSlicer", "ElegooSlicer", ""},
			{"qidi-studio", "QIDIStudio", "QIDIStudio", ""},
			{"creality-print", "Creality Print", "Creality Print", ""},
			{"anycubic-slicer-next", "Anycubic Slicer Next", "AnycubicSlicerNext", ""},
			{"anycubic-slicer-next", "Anycubic Slicer Next", "Anycubic Slicer Next", ""},
			{"orca-slicer", "OrcaSlicer", "OrcaSlicer", ""},
		}
	}
	return nil
}

// orcaFamilyLocations is the discovery inventory for installed Orca-family
// profiles. Explicit local overrides come first, then each trusted application
// in priority order across every install base. It never searches recursively.
func orcaFamilyLocations(platform, programFiles, programFilesX86, localAppData, home string, overrides []string) []slicerLocation {
	result := []slicerLocation{}
	seen := map[string]bool{}
	for _, override := range overrides {
		if override != "" {
			result = appendUnique(result, seen, slicerLocation{"system", "Custom local profile", override, ""})
		}
	}
	bases := installBases(platform, programFiles, programFilesX86, localAppData, home)
	for _, app := range orcaFamilyApps(platform) {
		for _, base := range bases {
			result = appendUnique(result, seen, installLocation(platform, base, app))
		}
	}
	return result
}

// artworkRootsForOS extends the installed Orca-family asset inventory with
// Bambu Studio and PrusaSlicer. Their profile artwork may be used for the
// EXACT same model even though their conversion profiles are not eligible
// Orca destination profiles. Roots remain explicit trusted app installations:
// no recursive drive search, no remote images, no frontend-supplied paths.
func artworkRootsForOS(platform, programFiles, programFilesX86, localAppData, home string, overrides []string) []slicerLocation {
	locations := orcaFamilyLocations(platform, programFiles, programFilesX86, localAppData, home, overrides)
	seen := map[string]bool{}
	for _, loc := range locations {
		seen[loc.root] = true
	}
	var extras []appInstall
	switch platform {
	case "windows":
		extras = []appInstall{
			{"bambu-studio", "Bambu Studio", "Bambu Studio", ""},
			{"prusa-slicer", "PrusaSlicer", filepath.Join("Prusa3D", "PrusaSlicer"), ""},
		}
	case "darwin":
		extras = []appInstall{
			{"bambu-studio", "Bambu Studio", "BambuStudio", ""},
			{"bambu-studio", "Bambu Studio", "Bambu Studio", ""},
			{"prusa-slicer", "PrusaSlicer", "PrusaSlicer", ""},
		}
	}
	for _, app := range extras {
		for _, base := range installBases(platform, programFiles, programFilesX86, localAppData, home) {
			locations = appendUnique(locations, seen, installLocation(platform, base, app))
		}
	}
	return locations
}

// ArtworkRoots reports only roots that really exist on this machine. Printer
// model matching, PNG validation and symlink containment remain enforced by
// the artwork reader, not by untrusted slicer preset metadata.
func ArtworkRoots() []string {
	pf := os.Getenv("ProgramFiles")
	if runtime.GOOS == "windows" && pf == "" {
		pf = `C:\Program Files`
	}
	home, _ := os.UserHomeDir()
	overrides := []string{}
	for _, target := range []string{"u1", "c5", "cc2", "qidi"} {
		overrides = append(overrides, os.Getenv("BRIDGE_SLICER_PROFILE_"+strings.ToUpper(target)))
	}
	locations := artworkRootsForOS(runtime.GOOS, pf, os.Getenv("ProgramFiles(x86)"),
		os.Getenv("LOCALAPPDATA"), home, overrides)
	result := []string{}
	seen := map[string]bool{}
	for _, loc := range locations {
		absolute, err := filepath.Abs(loc.root)
		if err != nil || seen[absolute] {
			continue
		}
		if info, err := os.Stat(absolute); err == nil && info.IsDir() {
			seen[absolute] = true
			result = append(result, absolute)
		}
	}
	return result
}

// installedOrcaFamilyLocations reads the live environment for orcaFamilyLocations.
// The per-target BRIDGE_SLICER_PROFILE_* overrides are kept as explicit preferences.
func installedOrcaFamilyLocations() []slicerLocation {
	pf := os.Getenv("ProgramFiles")
	if runtime.GOOS == "windows" && pf == "" {
		pf = `C:\Program Files`
	}
	home, _ := os.UserHomeDir()
	overrides := []string{}
	for _, target := range []string{"u1", "c5", "cc2", "qidi"} {
		overrides = append(overrides, os.Getenv("BRIDGE_SLICER_PROFILE_"+strings.ToUpper(target)))
	}
	return orcaFamilyLocations(runtime.GOOS, pf, os.Getenv("ProgramFiles(x86)"),
		os.Getenv("LOCALAPPDATA"), home, overrides)
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

// qidiStudioExecutable retains the older Windows filename but also accepts the
// real qidi-studio.exe used by current installed builds. Only the two fixed names
// in the same trusted app folder are allowed; links and empty files do not qualify.
func qidiStudioExecutable(loc slicerLocation) string {
	if loc.id != "qidi-studio" || filepath.Base(loc.executable) != "QIDIStudio.exe" {
		return loc.executable
	}
	if info, err := os.Lstat(loc.executable); err == nil && info.Mode().IsRegular() && info.Size() > 0 {
		return loc.executable
	}
	alternative := filepath.Join(filepath.Dir(loc.executable), "qidi-studio.exe")
	if info, err := os.Lstat(alternative); err == nil && info.Mode().IsRegular() && info.Size() > 0 {
		return alternative
	}
	return loc.executable
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
		return loc.root, qidiStudioExecutable(loc), loc.name, nil
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
