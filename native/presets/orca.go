package presets

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/BryanSmee/bl_to_std/pkg/printer"
)

// Installed Orca-family machine presets become conversion targets only when the
// machine, its default process and a PLA filament are all locally present and
// each one explicitly lists that machine as compatible. Nothing is inferred
// from vendor folder names, and no per-printer approval flag is consulted.

const maxOrcaRejections = 50
const maxOrcaTools = 8

var orcaTargetPattern = regexp.MustCompile(`^orca-[a-z0-9]+(?:-[a-z0-9]+)*-[0-9a-f]{10}$`)
var orcaMaterials = []string{"PLA", "PETG", "ABS", "ASA", "TPU"}
var orcaNozzles = []string{"0.2", "0.25", "0.4", "0.6", "0.8"}

// OrcaTarget is one installed machine that resolved completely. Its ID is
// derived from the machine file location and is re-resolved before each use.
type OrcaTarget struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Slicer        string   `json:"slicer"`
	Vendor        string   `json:"vendor"`
	Machine       string   `json:"machine"`
	Model         string   `json:"model"`
	Nozzle        string   `json:"nozzle"`
	PhysicalTools int      `json:"physicalTools"`
	Materials     []string `json:"materials"`
}

// OrcaRejection explains why an installed machine preset is not offered.
type OrcaRejection struct {
	Vendor  string `json:"vendor"`
	Machine string `json:"machine"`
	Slicer  string `json:"slicer"`
	Reason  string `json:"reason"`
}

// OrcaDiscovery is the frontend view. Install paths never leave this package.
type OrcaDiscovery struct {
	Targets  []OrcaTarget    `json:"targets"`
	Rejected []OrcaRejection `json:"rejected"`
}

// OrcaBundle is the generic target bundle read by engine/bridge.
type OrcaBundle struct {
	Target           string                    `json:"target"`
	Slicer           string                    `json:"slicer"`
	Nozzle           string                    `json:"nozzle"`
	PhysicalTools    int                       `json:"physical_tools"`
	Profile          printer.Profile           `json:"profile"`
	MaterialSettings map[string]map[string]any `json:"material_settings"`
}

type orcaCandidate struct {
	target OrcaTarget
	bundle OrcaBundle
}

// IsOrcaTargetID reports whether an ID has the shape produced by this package.
func IsOrcaTargetID(id string) bool {
	return orcaTargetPattern.MatchString(id)
}

// OrcaPrefix derives a filesystem-safe output prefix from a target ID.
func OrcaPrefix(id string) string {
	body := strings.TrimPrefix(id, "orca-")
	if len(body) > 11 {
		// Drop the "-<10 hex>" suffix that keeps IDs unique per install.
		body = body[:len(body)-11]
	}
	body = strings.ToUpper(body)
	if len(body) > 24 {
		body = body[:24]
	}
	return "ORCA-" + strings.Trim(body, "-")
}

// DiscoverOrca lists installed machine presets that resolve completely, plus
// bounded reasons for every installed machine that does not.
func DiscoverOrca() OrcaDiscovery {
	return discoverOrca(installedOrcaLocations())
}

// ResolveOrcaBundle re-resolves an ID from disk. The ID is matched only against
// freshly discovered targets; it is never treated as a path.
func ResolveOrcaBundle(id string) (*OrcaBundle, error) {
	return resolveOrca(installedOrcaLocations(), id)
}

func installedOrcaLocations() []slicerLocation {
	seen := map[string]bool{}
	locations := []slicerLocation{}
	for _, target := range []string{"u1", "c5", "cc2", "qidi"} {
		for _, loc := range slicerLocations(target) {
			if seen[loc.root] {
				continue
			}
			seen[loc.root] = true
			locations = append(locations, loc)
		}
	}
	return locations
}

func discoverOrca(locations []slicerLocation) OrcaDiscovery {
	candidates, rejected := scanOrca(locations)
	discovery := OrcaDiscovery{Targets: []OrcaTarget{}, Rejected: []OrcaRejection{}}
	for _, candidate := range candidates {
		discovery.Targets = append(discovery.Targets, candidate.target)
	}
	sort.Slice(discovery.Targets, func(i, j int) bool {
		left, right := discovery.Targets[i], discovery.Targets[j]
		return left.Vendor+" "+left.Machine+" "+left.ID < right.Vendor+" "+right.Machine+" "+right.ID
	})
	if len(rejected) > maxOrcaRejections {
		rejected = rejected[:maxOrcaRejections]
	}
	discovery.Rejected = append(discovery.Rejected, rejected...)
	return discovery
}

func resolveOrca(locations []slicerLocation, id string) (*OrcaBundle, error) {
	if !IsOrcaTargetID(id) {
		return nil, fmt.Errorf("invalid installed profile identifier")
	}
	candidates, _ := scanOrca(locations)
	for _, candidate := range candidates {
		if candidate.target.ID == id {
			bundle := candidate.bundle
			return &bundle, nil
		}
	}
	return nil, fmt.Errorf("installed profile is not available or no longer resolves completely")
}

func scanOrca(locations []slicerLocation) ([]orcaCandidate, []OrcaRejection) {
	candidates := []orcaCandidate{}
	rejected := []OrcaRejection{}
	for _, loc := range locations {
		vendors, err := os.ReadDir(loc.root)
		if err != nil {
			continue
		}
		for _, vendor := range vendors {
			// Bambu remains a source-only vendor in this app.
			if !vendor.IsDir() || vendor.Name() == "BBL" || vendor.Name() == "OrcaFilamentLibrary" {
				continue
			}
			found, problems := scanOrcaVendor(loc, vendor.Name())
			candidates = append(candidates, found...)
			rejected = append(rejected, problems...)
		}
	}
	return candidates, rejected
}

// scanOrcaVendor reads one vendor's preset index once, then checks each of its
// machine files independently so one bad machine never hides the others.
func scanOrcaVendor(loc slicerLocation, vendor string) ([]orcaCandidate, []OrcaRejection) {
	machineDir := filepath.Join(loc.root, vendor, "machine")
	entries, err := os.ReadDir(machineDir)
	if err != nil {
		return nil, nil
	}
	files := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			files = append(files, filepath.Join(machineDir, entry.Name()))
		}
	}
	if len(files) == 0 {
		return nil, nil
	}
	index, err := readCatalog(loc.root, Spec{Vendor: vendor})
	if err != nil {
		return nil, []OrcaRejection{{Vendor: vendor, Slicer: loc.name, Reason: "vendor profiles unreadable: " + err.Error()}}
	}
	byMachine := orcaFilamentsByMachine(index)
	candidates := []orcaCandidate{}
	rejected := []OrcaRejection{}
	for _, file := range files {
		candidate, err := resolveOrcaMachine(index, byMachine, loc, vendor, file)
		if err != nil {
			rejected = append(rejected, OrcaRejection{
				Vendor: vendor, Machine: strings.TrimSuffix(filepath.Base(file), ".json"),
				Slicer: loc.name, Reason: err.Error(),
			})
			continue
		}
		candidates = append(candidates, candidate)
	}
	return candidates, rejected
}

// resolveOrcaMachine applies every fail-closed check to one machine file.
func resolveOrcaMachine(index catalog, byMachine map[string]map[string][]string, loc slicerLocation, vendor, file string) (orcaCandidate, error) {
	stem := strings.TrimSuffix(filepath.Base(file), ".json")
	data, err := os.ReadFile(file)
	if err != nil {
		return orcaCandidate{}, err
	}
	var leaf map[string]any
	if err := json.Unmarshal(data, &leaf); err != nil {
		return orcaCandidate{}, fmt.Errorf("machine JSON is invalid")
	}
	machineName, _ := leaf["name"].(string)
	if machineName == "" {
		machineName = stem
	}
	machine, err := flatten(index, "machine", stem, map[string]bool{})
	if err != nil {
		return orcaCandidate{}, err
	}
	if technology := orcaFirstString(machine["printer_technology"]); technology != "" && technology != "FFF" {
		return orcaCandidate{}, fmt.Errorf("printer technology %s is not supported", technology)
	}
	model := orcaFirstString(machine["printer_model"])
	if model == "" {
		return orcaCandidate{}, fmt.Errorf("printer_model is missing")
	}
	for _, spec := range Specs {
		if spec.Model == model {
			return orcaCandidate{}, fmt.Errorf("%s is already offered by a built-in engine", model)
		}
	}
	nozzle, tools, err := orcaNozzle(machine["nozzle_diameter"])
	if err != nil {
		return orcaCandidate{}, err
	}
	for _, key := range []string{"machine_start_gcode", "machine_end_gcode"} {
		if _, ok := machine[key].(string); !ok {
			return orcaCandidate{}, fmt.Errorf("%s is missing", key)
		}
	}
	processName := orcaFirstString(machine["default_print_profile"])
	if processName == "" {
		return orcaCandidate{}, fmt.Errorf("no default process preset is declared")
	}
	if len(index["process:"+processName]) == 0 {
		return orcaCandidate{}, fmt.Errorf("default process preset %q is not installed", processName)
	}
	if !orcaListsMachine(index, "process", processName, machineName) {
		return orcaCandidate{}, fmt.Errorf("default process %q does not list this machine as compatible", processName)
	}
	process, err := flatten(index, "process", processName, map[string]bool{})
	if err != nil {
		return orcaCandidate{}, err
	}
	materials := map[string]map[string]any{}
	filamentNames := map[string]string{}
	for _, material := range orcaMaterials {
		name := orcaPreferredChoice(byMachine[machineName][material], machine["default_filament_profile"])
		if name == "" {
			if material == "PLA" {
				return orcaCandidate{}, fmt.Errorf("no PLA filament preset lists this machine as compatible")
			}
			continue
		}
		filament, err := flatten(index, "filament", name, map[string]bool{})
		if err != nil {
			return orcaCandidate{}, err
		}
		materials[material] = filament
		filamentNames[material] = name
	}
	project := map[string]any{}
	for _, layer := range []map[string]any{machine, process, materials["PLA"]} {
		for key, value := range layer {
			project[key] = value
		}
	}
	// Firmware and hardware values belong to the installed machine.
	for key, value := range machine {
		if strings.HasPrefix(key, "machine_") || key == "nozzle_diameter" || key == "change_filament_gcode" ||
			key == "layer_change_gcode" || key == "before_layer_change_gcode" {
			project[key] = value
		}
	}
	project["printer_model"] = model
	project["printer_settings_id"] = machineName
	project["print_settings_id"] = processName
	project["filament_settings_id"] = []string{filamentNames["PLA"]}
	project["filament_type"] = []string{"PLA"}
	project["filament_colour"] = []string{"#FFFFFF"}

	target := orcaTargetID(vendor, machineName, file)
	materialNames := []string{}
	for _, material := range orcaMaterials {
		if _, ok := filamentNames[material]; ok {
			materialNames = append(materialNames, material)
		}
	}
	bundle := OrcaBundle{
		Target: target, Slicer: loc.name, Nozzle: nozzle, PhysicalTools: tools,
		Profile: printer.Profile{
			Name: target, DisplayName: model, ModelID: model, FilamentSlots: tools,
			FilamentProfiles: filamentNames, DefaultFilamentProfile: filamentNames["PLA"], ProjectSettings: project,
		},
		MaterialSettings: materials,
	}
	return orcaCandidate{
		target: OrcaTarget{
			ID: target, Slicer: loc.name, Vendor: vendor, Machine: machineName, Model: model,
			Nozzle: nozzle, PhysicalTools: tools, Materials: materialNames,
		},
		bundle: bundle,
	}, nil
}

// orcaNozzle requires one uniform, supported diameter per installed tool. The
// tool count is the length of that installed nozzle array.
func orcaNozzle(raw any) (string, int, error) {
	entries, _ := raw.([]any)
	if len(entries) < 1 || len(entries) > maxOrcaTools {
		return "", 0, fmt.Errorf("nozzle_diameter must list 1..%d tools", maxOrcaTools)
	}
	nozzle := ""
	for _, entry := range entries {
		value, err := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(entry)), 64)
		if err != nil {
			return "", 0, fmt.Errorf("nozzle_diameter has a non-numeric entry")
		}
		match := ""
		for _, supported := range orcaNozzles {
			reference, _ := strconv.ParseFloat(supported, 64)
			if math.Abs(reference-value) < 0.000001 {
				match = supported
			}
		}
		if match == "" {
			return "", 0, fmt.Errorf("unsupported nozzle diameter %v mm", entry)
		}
		if nozzle != "" && match != nozzle {
			return "", 0, fmt.Errorf("mixed nozzle diameters are not supported")
		}
		nozzle = match
	}
	return nozzle, len(entries), nil
}

// orcaFilamentsByMachine groups filament presets by the machines that name
// them as compatible, then by resolved filament_type.
func orcaFilamentsByMachine(index catalog) map[string]map[string][]string {
	result := map[string]map[string][]string{}
	seen := map[string]bool{}
	for key := range index {
		if !strings.HasPrefix(key, "filament:") {
			continue
		}
		name := strings.TrimPrefix(key, "filament:")
		if seen[name] {
			continue
		}
		seen[name] = true
		machines, ok := orcaCompatibility(index, "filament", name)
		if !ok {
			continue
		}
		flat, err := flatten(index, "filament", name, map[string]bool{})
		if err != nil {
			continue
		}
		material := orcaFirstString(flat["filament_type"])
		if material == "" {
			continue
		}
		for _, machine := range machines {
			if result[machine] == nil {
				result[machine] = map[string][]string{}
			}
			result[machine][material] = append(result[machine][material], name)
		}
	}
	for _, byMaterial := range result {
		for material := range byMaterial {
			sort.Strings(byMaterial[material])
		}
	}
	return result
}

// orcaCompatibility returns the explicit compatible_printers list of a preset
// or its nearest ancestor. A missing list is never read as "all printers".
func orcaCompatibility(index catalog, typ, name string) ([]string, bool) {
	for depth := 0; depth < 8; depth++ {
		choices := index[typ+":"+name]
		if len(choices) == 0 {
			return nil, false
		}
		leaf := choose(choices, name).settings
		if raw, ok := leaf["compatible_printers"]; ok {
			list, isList := raw.([]any)
			if !isList {
				return nil, false
			}
			names := []string{}
			for _, item := range list {
				if s, isString := item.(string); isString {
					names = append(names, s)
				}
			}
			return names, true
		}
		parent, _ := leaf["inherits"].(string)
		if parent == "" {
			return nil, false
		}
		name = parent
	}
	return nil, false
}

func orcaListsMachine(index catalog, typ, name, machine string) bool {
	names, ok := orcaCompatibility(index, typ, name)
	if !ok {
		return false
	}
	for _, listed := range names {
		if listed == machine {
			return true
		}
	}
	return false
}

// orcaPreferredChoice prefers the machine's declared default filament for this
// material, then the first compatible sorted name. An empty result means the
// material is unavailable for this machine and will fail if a project uses it.
func orcaPreferredChoice(choices []string, defaults any) string {
	if len(choices) == 0 {
		return ""
	}
	var listed []any
	switch value := defaults.(type) {
	case []any:
		listed = value
	case string:
		listed = []any{value}
	}
	for _, item := range listed {
		s, _ := item.(string)
		for _, choice := range choices {
			if s != "" && choice == s {
				return s
			}
		}
	}
	return choices[0]
}

// choose mirrors flatten: prefer the file whose stem matches the requested name.
func choose(choices []preset, name string) preset {
	for _, candidate := range choices {
		if strings.TrimSuffix(filepath.Base(candidate.file), ".json") == name {
			return candidate
		}
	}
	return choices[0]
}

func orcaFirstString(raw any) string {
	switch value := raw.(type) {
	case string:
		return value
	case []any:
		if len(value) > 0 {
			s, _ := value[0].(string)
			return s
		}
	}
	return ""
}

// orcaTargetID hashes the machine file location so IDs stay unique per install
// without exposing the path to the frontend.
func orcaTargetID(vendor, machine, file string) string {
	sum := sha256.Sum256([]byte(file))
	return "orca-" + orcaSlug(vendor+" "+machine, 40) + "-" + hex.EncodeToString(sum[:])[:10]
}

func orcaSlug(value string, limit int) string {
	var b strings.Builder
	dash := true
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if len(slug) > limit {
		slug = strings.Trim(slug[:limit], "-")
	}
	if slug == "" {
		return "printer"
	}
	return slug
}
