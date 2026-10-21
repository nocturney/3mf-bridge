package presets

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// ManufacturerSlicer is a popular slicer that may be installed on this computer.
// Presence comes only from file-type checks on fixed, trusted locations. The
// manufacturers name the vendor a slicer ships profiles for; they never imply
// that any particular printer model converts or prints correctly.
type ManufacturerSlicer struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Manufacturers []string `json:"manufacturers"`
	Installed     bool     `json:"installed"`
	Evidence      string   `json:"evidence"`
	SourceOnly    bool     `json:"sourceOnly"`
}

const (
	manufacturerEvidenceApplication = "application"
	manufacturerEvidenceProfiles    = "profiles"
)

// windowsInstall is an install folder below a trusted Windows base plus the
// executable inside it. Folder parts stay separate so they join per platform.
type windowsInstall struct {
	folder []string
	exe    string
}

// manufacturerSlicerSpec lists the trusted install locations for one slicer.
// Orca-family apps keep vendor profiles under resources/profiles.
type manufacturerSlicerSpec struct {
	id            string
	name          string
	manufacturers []string
	sourceOnly    bool
	generic       bool
	profiles      bool
	windows       []windowsInstall
	bundles       []string
}

// manufacturerSlicerSpecs is the coverage table. Entries are added only when the
// install folder or bundle name is known, so a missing slicer is a false negative
// rather than a wrong detection.
var manufacturerSlicerSpecs = []manufacturerSlicerSpec{
	{id: "prusa-slicer", name: "PrusaSlicer", manufacturers: []string{"Prusa Research", "Prusa"}, windows: []windowsInstall{{folder: []string{"Prusa3D", "PrusaSlicer"}, exe: "prusa-slicer.exe"}}, bundles: []string{"PrusaSlicer.app"}},
	{id: "bambu-studio", name: "Bambu Studio", manufacturers: []string{"Bambu Lab"}, sourceOnly: true, windows: []windowsInstall{{folder: []string{"Bambu Studio"}, exe: "bambu-studio.exe"}}, bundles: []string{"BambuStudio.app", "Bambu Studio.app"}},
	{id: "orca-slicer", name: "OrcaSlicer", generic: true, profiles: true, windows: []windowsInstall{{folder: []string{"OrcaSlicer"}, exe: "orca-slicer.exe"}}, bundles: []string{"OrcaSlicer.app"}},
	{id: "elegoo-slicer", name: "ElegooSlicer", manufacturers: []string{"ELEGOO"}, profiles: true, windows: []windowsInstall{{folder: []string{"ElegooSlicer"}, exe: "elegoo-slicer.exe"}}, bundles: []string{"ElegooSlicer.app"}},
	{id: "qidi-studio", name: "QIDIStudio", manufacturers: []string{"QIDI"}, profiles: true, windows: []windowsInstall{{folder: []string{"QIDIStudio"}, exe: "QIDIStudio.exe"}}, bundles: []string{"QIDIStudio.app"}},
	{id: "snapmaker-orca", name: "Snapmaker Orca", manufacturers: []string{"Snapmaker"}, profiles: true, windows: []windowsInstall{{folder: []string{"Snapmaker_Orca"}, exe: "snapmaker-orca.exe"}}, bundles: []string{"Snapmaker Orca.app"}},
	{id: "flash-studio", name: "Flash Studio", manufacturers: []string{"Flashforge"}, profiles: true, windows: []windowsInstall{{folder: []string{"Flashforge", "Flash Studio Desktop"}, exe: "flash studio.exe"}}, bundles: []string{"Flash Studio.app", "Flash Studio Desktop.app"}},
	{id: "creality-print", name: "Creality Print", manufacturers: []string{"Creality"}, windows: []windowsInstall{{folder: []string{"Creality", "CrealityPrint"}, exe: "CrealityPrint.exe"}, {folder: []string{"CrealityPrint"}, exe: "CrealityPrint.exe"}, {folder: []string{"Creality Print"}, exe: "CrealityPrint.exe"}}, bundles: []string{"CrealityPrint.app", "Creality Print.app"}},
	{id: "anycubic-slicer-next", name: "Anycubic Slicer Next", manufacturers: []string{"Anycubic"}, windows: []windowsInstall{{folder: []string{"AnycubicSlicerNext"}, exe: "AnycubicSlicerNext.exe"}, {folder: []string{"Anycubic Slicer Next"}, exe: "AnycubicSlicerNext.exe"}, {folder: []string{"Anycubic", "Anycubic Slicer Next"}, exe: "Anycubic Slicer Next.exe"}}, bundles: []string{"AnycubicSlicerNext.app", "Anycubic Slicer Next.app"}},
	{id: "sovol-slicer", name: "Sovol Slicer", manufacturers: []string{"Sovol"}, windows: []windowsInstall{{folder: []string{"SovolSlicer"}, exe: "SovolSlicer.exe"}, {folder: []string{"Sovol Slicer"}, exe: "Sovol Slicer.exe"}}, bundles: []string{"SovolSlicer.app", "Sovol Slicer.app"}},
	{id: "flsun-slicer", name: "FLSUN Slicer", manufacturers: []string{"FLSun"}, windows: []windowsInstall{{folder: []string{"FLSUN Slicer"}, exe: "FLSUN Slicer.exe"}, {folder: []string{"FLSUNSlicer"}, exe: "FLSUNSlicer.exe"}}, bundles: []string{"FLSUN Slicer.app", "FLSUNSlicer.app"}},
	{id: "flashprint", name: "FlashPrint", manufacturers: []string{"Flashforge"}, windows: []windowsInstall{{folder: []string{"Flashforge", "FlashPrint 5"}, exe: "FlashPrint.exe"}, {folder: []string{"FlashPrint 5"}, exe: "FlashPrint.exe"}, {folder: []string{"FlashPrint"}, exe: "FlashPrint.exe"}}, bundles: []string{"FlashPrint.app", "FlashPrint 5.app"}},
	{id: "qidi-slicer", name: "QIDISlicer", manufacturers: []string{"QIDI"}, windows: []windowsInstall{{folder: []string{"QIDISlicer"}, exe: "QIDISlicer.exe"}}, bundles: []string{"QIDISlicer.app"}},
	{id: "snapmaker-luban", name: "Snapmaker Luban", manufacturers: []string{"Snapmaker"}, windows: []windowsInstall{{folder: []string{"Snapmaker Luban"}, exe: "Snapmaker Luban.exe"}, {folder: []string{"SnapmakerLuban"}, exe: "Snapmaker Luban.exe"}}, bundles: []string{"Snapmaker Luban.app"}},
	{id: "ankermake-studio", name: "AnkerMake Studio", manufacturers: []string{"Anker"}, windows: []windowsInstall{{folder: []string{"AnkerMake Studio"}, exe: "AnkerMake Studio.exe"}, {folder: []string{"AnkerMakeStudio"}, exe: "AnkerMake Studio.exe"}}, bundles: []string{"AnkerMake Studio.app"}},
	{id: "artillery-slicer", name: "Artillery Slicer", manufacturers: []string{"Artillery"}, windows: []windowsInstall{{folder: []string{"Artillery Slicer"}, exe: "Artillery Slicer.exe"}, {folder: []string{"ArtillerySlicer"}, exe: "ArtillerySlicer.exe"}}, bundles: []string{"Artillery Slicer.app", "ArtillerySlicer.app"}},
	{id: "voxelmaker", name: "VoxelMaker", manufacturers: []string{"Voxelab"}, windows: []windowsInstall{{folder: []string{"VoxelMaker"}, exe: "VoxelMaker.exe"}, {folder: []string{"Voxelab", "VoxelMaker"}, exe: "VoxelMaker.exe"}}, bundles: []string{"VoxelMaker.app"}},
	{id: "ratos-slicer", name: "RatOS Slicer", manufacturers: []string{"RatRig"}, windows: []windowsInstall{{folder: []string{"RatOS Slicer"}, exe: "RatOS Slicer.exe"}, {folder: []string{"RatOSSlicer"}, exe: "RatOSSlicer.exe"}}, bundles: []string{"RatOS Slicer.app", "RatOSSlicer.app"}},
	{id: "lulzbot-cura", name: "Cura LulzBot Edition", manufacturers: []string{"LulzBot"}, windows: []windowsInstall{{folder: []string{"Cura LulzBot Edition"}, exe: "CuraLE.exe"}, {folder: []string{"Cura LulzBot Edition"}, exe: "Cura.exe"}}, bundles: []string{"Cura LulzBot Edition.app"}},
	{id: "creality-slicer", name: "Creality Slicer", manufacturers: []string{"Creality"}, windows: []windowsInstall{{folder: []string{"Creality Slicer"}, exe: "Creality Slicer.exe"}}, bundles: []string{"Creality Slicer.app"}},
	{id: "anycubic-slicer", name: "Anycubic Slicer", manufacturers: []string{"Anycubic"}, windows: []windowsInstall{{folder: []string{"AnycubicSlicer"}, exe: "AnycubicSlicer.exe"}, {folder: []string{"Anycubic Slicer"}, exe: "Anycubic Slicer.exe"}}, bundles: []string{"AnycubicSlicer.app", "Anycubic Slicer.app"}},
	{id: "dremel-digilab", name: "Dremel DigiLab 3D Slicer", manufacturers: []string{"Dremel"}, windows: []windowsInstall{{folder: []string{"Dremel DigiLab 3D Slicer"}, exe: "Dremel DigiLab 3D Slicer.exe"}}, bundles: []string{"Dremel DigiLab 3D Slicer.app"}},
}

// manufacturerCandidate is one trusted location to test. base is the trusted
// root, and app and profiles are the parts below it. Nothing is joined to a
// relative path, so an empty root never matches the working directory.
type manufacturerCandidate struct {
	index    int
	base     string
	app      []string
	profiles []string
}

// manufacturerFS is the only host access during detection. It exposes Lstat
// and ReadDir, never Stat or open, so callers cannot follow links by accident.
type manufacturerFS interface {
	Lstat(name string) (fs.FileInfo, error)
	ReadDir(name string) ([]fs.DirEntry, error)
}

type hostManufacturerFS struct{}

func (hostManufacturerFS) Lstat(name string) (fs.FileInfo, error) { return os.Lstat(name) }

func (hostManufacturerFS) ReadDir(name string) ([]fs.DirEntry, error) { return os.ReadDir(name) }

// DetectManufacturerSlicers reports every supported slicer, installed or not.
// It checks only fixed Windows and macOS locations. It runs no process, reads
// no registry, and takes no path from the frontend.
func DetectManufacturerSlicers() []ManufacturerSlicer {
	programFiles := os.Getenv("ProgramFiles")
	programFilesX86 := os.Getenv("ProgramFiles(x86)")
	if runtime.GOOS == "windows" {
		if programFiles == "" {
			programFiles = `C:\Program Files`
		}
		if programFilesX86 == "" {
			programFilesX86 = `C:\Program Files (x86)`
		}
	}
	applications, userApplications := "", ""
	if runtime.GOOS == "darwin" {
		applications = "/Applications"
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			userApplications = filepath.Join(home, "Applications")
		}
	}
	return detectManufacturerSlicers(runtime.GOOS, hostManufacturerFS{}, programFiles, programFilesX86,
		os.Getenv("LOCALAPPDATA"), applications, userApplications)
}

// manufacturerCandidates builds the trusted locations for one platform. It is a
// pure function of its arguments so tests can pass fake roots.
func manufacturerCandidates(platform, programFiles, programFilesX86, localAppData, applications, userApplications string) []manufacturerCandidate {
	result := []manufacturerCandidate{}
	for i, spec := range manufacturerSlicerSpecs {
		switch platform {
		case "windows":
			bases := []string{programFiles, programFilesX86}
			if localAppData != "" {
				bases = append(bases, filepath.Join(localAppData, "Programs"))
			}
			for _, base := range bases {
				for _, install := range spec.windows {
					candidate := manufacturerCandidate{index: i, base: base, app: append(append([]string{}, install.folder...), install.exe)}
					if spec.profiles {
						candidate.profiles = append(append([]string{}, install.folder...), "resources", "profiles")
					}
					result = append(result, candidate)
				}
			}
		case "darwin":
			for _, base := range []string{applications, userApplications} {
				for _, bundle := range spec.bundles {
					candidate := manufacturerCandidate{index: i, base: base, app: []string{bundle, "Contents", "Info.plist"}}
					if spec.profiles {
						candidate.profiles = []string{bundle, "Contents", "Resources", "profiles"}
					}
					result = append(result, candidate)
				}
			}
		}
	}
	return result
}

func detectManufacturerSlicers(platform string, fsys manufacturerFS, programFiles, programFilesX86, localAppData, applications, userApplications string) []ManufacturerSlicer {
	candidates := manufacturerCandidates(platform, programFiles, programFilesX86, localAppData, applications, userApplications)
	items := make([]ManufacturerSlicer, 0, len(manufacturerSlicerSpecs))
	for i, spec := range manufacturerSlicerSpecs {
		item := ManufacturerSlicer{ID: spec.id, Name: spec.name, Manufacturers: []string{}, SourceOnly: spec.sourceOnly}
		if !spec.generic {
			item.Manufacturers = append([]string{}, spec.manufacturers...)
		}
		appFound, profilesFound := false, false
		var vendors []string
		for _, candidate := range candidates {
			if candidate.index != i {
				continue
			}
			if info, ok := lstatWithin(fsys, candidate.base, candidate.app); ok && info.Mode().IsRegular() && info.Size() > 0 {
				appFound = true
			}
			if !profilesFound && candidate.profiles != nil {
				if names, ok := localVendorNames(fsys, candidate.base, candidate.profiles); ok && len(names) > 0 {
					profilesFound, vendors = true, names
				}
			}
		}
		switch {
		case appFound:
			item.Installed, item.Evidence = true, manufacturerEvidenceApplication
		case profilesFound:
			item.Installed, item.Evidence = true, manufacturerEvidenceProfiles
		}
		// A generic Orca-family app names a manufacturer only when its own
		// profile tree really ships that vendor.
		if spec.generic && profilesFound {
			item.Manufacturers = vendors
		}
		items = append(items, item)
	}
	return items
}

// lstatWithin walks parts below base. Every directory on the way must be a real
// directory, so a symlink or junction under a trusted root cannot redirect the
// lookup elsewhere. The final component is returned unfollowed for the caller
// to type-check.
func lstatWithin(fsys manufacturerFS, base string, parts []string) (fs.FileInfo, bool) {
	if base == "" || len(parts) == 0 {
		return nil, false
	}
	dir := base
	if info, err := fsys.Lstat(dir); err != nil || !info.IsDir() {
		return nil, false
	}
	for _, part := range parts[:len(parts)-1] {
		dir = filepath.Join(dir, part)
		if info, err := fsys.Lstat(dir); err != nil || !info.IsDir() {
			return nil, false
		}
	}
	info, err := fsys.Lstat(filepath.Join(dir, parts[len(parts)-1]))
	if err != nil {
		return nil, false
	}
	return info, true
}

// localVendorNames maps vendor folders in an installed profile root to exact
// catalogue manufacturer names. A folder counts only when it has a machine
// directory. Bambu is excluded because it stays source-only in this app.
func localVendorNames(fsys manufacturerFS, base string, parts []string) ([]string, bool) {
	info, ok := lstatWithin(fsys, base, parts)
	if !ok || !info.IsDir() {
		return nil, false
	}
	root := filepath.Join(append([]string{base}, parts...)...)
	entries, err := fsys.ReadDir(root)
	if err != nil {
		return nil, false
	}
	names := []string{}
	seen := map[string]bool{}
	for _, entry := range entries {
		// DirEntry reports a symlink as a non-directory, so links are skipped here.
		if !entry.IsDir() {
			continue
		}
		name, known := catalogManufacturerKeys[catalogKey(entry.Name())]
		if !known || excludedVendors[name] || seen[name] {
			continue
		}
		machine, err := fsys.Lstat(filepath.Join(root, entry.Name(), "machine"))
		if err != nil || !machine.IsDir() {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	sort.Strings(names)
	return names, true
}

// excludedVendors never appear as manufacturers of a generic Orca-family app.
var excludedVendors = map[string]bool{"Bambu Lab": true, "Custom": true, "UltiMaker": true, "Raise3D": true}

// catalogKey folds a manufacturer or folder name to lowercase letters and
// digits, so "PrusaResearch" and "Prusa Research" compare equal.
func catalogKey(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// catalogManufacturerKeys maps a folded key back to its exact catalogue name.
var catalogManufacturerKeys = func() map[string]string {
	keys := make(map[string]string, len(catalogManufacturerNames))
	for _, name := range catalogManufacturerNames {
		keys[catalogKey(name)] = name
	}
	return keys
}()

// catalogManufacturerNames are the 84 exact manufacturer names in the offline
// printer catalogue. Tests check this list against the embedded catalogue.
var catalogManufacturerNames = []string{
	"Afinia",
	"Anker",
	"Anycubic",
	"Artillery",
	"Bambu Lab",
	"BIBO",
	"BIQU",
	"Blocks",
	"Caribou",
	"Chuanying",
	"Co Print",
	"CoLiDo",
	"Comgrow",
	"CONSTRUCT3D",
	"Creality",
	"Cubicon",
	"Custom",
	"DeltaMaker",
	"Dremel",
	"E2D",
	"ELEGOO",
	"Eryone",
	"FELIX Printers",
	"Flashforge",
	"FLSun",
	"FlyingBear",
	"Folgertech",
	"gCreate",
	"Geeetech",
	"Goofoo",
	"HartSmartProducts",
	"IdeaFormer",
	"INAT",
	"InfiMech",
	"Infinity3D",
	"iQ",
	"Kingroon",
	"LH",
	"LNL3D",
	"LONGER",
	"LulzBot",
	"M3D",
	"MagicMaker",
	"MakerGear",
	"Mellow",
	"Meltingplot",
	"OpenEYE",
	"OrcaArena",
	"PapapiuLab",
	"Peopoly",
	"Phrozen",
	"Positron3D",
	"Print4Taste",
	"Printcepts",
	"Prusa",
	"Prusa Research",
	"QIDI",
	"Raise3D",
	"RatRig",
	"re3D",
	"RH3D",
	"Rigid3D",
	"RolohaunDesign",
	"SecKit",
	"SeeMeCNC",
	"Snapmaker",
	"Sovol",
	"Tiertime",
	"TriLAB",
	"Trimaker",
	"Tronxy",
	"TwoTrees",
	"UltiMaker",
	"Vivedino",
	"Volumic",
	"Voron",
	"Voxelab",
	"Vzbot",
	"Wanhao",
	"Wanhao France",
	"WEMAKE3D",
	"WonderMaker",
	"Z-Bolt",
	"Zonestar",
}
