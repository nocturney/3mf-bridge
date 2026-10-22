package main

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/BryanSmee/bl_to_std/pkg/converter"
	"github.com/nocturney/slicebridge/engine/bridge"
	"github.com/nocturney/slicebridge/native/presets"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const maxBatchFiles = 200

type SourceSummary struct {
	Path          string               `json:"path"`
	Name          string               `json:"name"`
	Nozzle        string               `json:"nozzle"`
	NozzleAssumed bool                 `json:"nozzleAssumed"`
	Plates        int                  `json:"plates"`
	Objects       int                  `json:"objects"`
	Instances     int                  `json:"instances"`
	Filaments     []converter.Filament `json:"filaments"`
}
type BatchRequest struct {
	FilePaths []string `json:"filePaths"`
	Targets   []string `json:"targets"`
}
type FileBatchResult struct {
	Source   SourceSummary   `json:"source"`
	Results  []ConvertResult `json:"results"`
	Failures []ConvertResult `json:"failures"`
	Error    string          `json:"error,omitempty"`
}
type BatchReply struct {
	StartedAt  string            `json:"startedAt"`
	FinishedAt string            `json:"finishedAt"`
	Files      []FileBatchResult `json:"files"`
	FilesDone  int               `json:"filesDone"`
	Successes  int               `json:"successes"`
	Failures   int               `json:"failures"`
	Error      string            `json:"error,omitempty"`
}
type BatchProgress struct {
	FileIndex int    `json:"fileIndex"`
	FileCount int    `json:"fileCount"`
	FileName  string `json:"fileName"`
	Completed bool   `json:"completed"`
}

// CollectInputs expands files and folders from the native picker or OS drop.
// Symlink directories are not traversed and previously generated output files
// are not requeued when a whole source folder is dropped.
func collectInputs(paths []string) ([]FileInfo, error) {
	if len(paths) == 0 {
		return []FileInfo{}, nil
	}
	if len(paths) > maxBatchFiles {
		return nil, fmt.Errorf("too many dropped items (limit %d)", maxBatchFiles)
	}
	collected := []FileInfo{}
	seen := map[string]bool{}
	add := func(name string, fromFolder bool) error {
		if fromFolder {
			upper := strings.ToUpper(filepath.Base(name))
			for _, prefix := range []string{"U1-", "C5-", "CC2-", "QIDI-", "ORCA-", "PRUSA-"} {
				if strings.HasPrefix(upper, prefix) {
					return nil
				}
			}
		}
		if !strings.EqualFold(filepath.Ext(name), ".3mf") {
			return nil
		}
		info, err := fileInfo(name)
		if err != nil {
			return err
		}
		key := info.Path
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if seen[key] {
			return nil
		}
		if len(collected) >= maxBatchFiles {
			return fmt.Errorf("a maximum of %d projects may be queued", maxBatchFiles)
		}
		seen[key] = true
		collected = append(collected, *info)
		return nil
	}
	for _, item := range paths {
		abs, err := filepath.Abs(item)
		if err != nil {
			return nil, err
		}
		stat, err := os.Lstat(abs)
		if err != nil {
			return nil, err
		}
		if stat.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("symlink inputs are not accepted: %s", abs)
		}
		if stat.IsDir() {
			err = filepath.WalkDir(abs, func(name string, entry os.DirEntry, entryErr error) error {
				if entryErr != nil {
					return entryErr
				}
				if entry.Type()&os.ModeSymlink != 0 {
					return nil
				}
				if entry.IsDir() {
					if name != abs && strings.HasPrefix(entry.Name(), ".") {
						return filepath.SkipDir
					}
					return nil
				}
				return add(name, true)
			})
			if err != nil {
				return nil, err
			}
		} else {
			if !strings.EqualFold(filepath.Ext(abs), ".3mf") {
				return nil, fmt.Errorf("only .3mf files may be added")
			}
			if err = add(abs, false); err != nil {
				return nil, err
			}
		}
	}
	sort.Slice(collected, func(i, j int) bool { return strings.ToLower(collected[i].Path) < strings.ToLower(collected[j].Path) })
	return collected, nil
}
func (a *App) CollectInputs(paths []string) ([]FileInfo, error) { return collectInputs(paths) }
func (a *App) SelectFiles(lang string) ([]FileInfo, error) {
	if a.ctx == nil {
		return nil, fmt.Errorf("desktop window not ready")
	}
	title := "Add Bambu 3MF projects"
	if lang == "he" {
		title = "הוספת פרויקטים של Bambu 3MF"
	}
	paths, err := wailsruntime.OpenMultipleFilesDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: title, Filters: []wailsruntime.FileFilter{{DisplayName: "3MF (*.3mf)", Pattern: "*.3mf"}},
	})
	if err != nil {
		return nil, err
	}
	return collectInputs(paths)
}
func (a *App) SelectFolder(lang string) ([]FileInfo, error) {
	if a.ctx == nil {
		return nil, fmt.Errorf("desktop window not ready")
	}
	title := "Add all 3MF projects in folder"
	if lang == "he" {
		title = "הוספת כל קובצי 3MF מהתיקייה"
	}
	folder, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{Title: title})
	if err != nil {
		return nil, err
	}
	if folder == "" {
		return []FileInfo{}, nil
	}
	return collectInputs([]string{folder})
}

func inspectModelCount(path string) (objects, instances int, err error) {
	f, err := zip.OpenReader(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	found := false
	for _, item := range f.File {
		if !strings.HasPrefix(item.Name, "3D/") || !strings.HasSuffix(strings.ToLower(item.Name), ".model") {
			continue
		}
		if item.UncompressedSize64 > 64<<20 {
			return 0, 0, fmt.Errorf("source model XML too large")
		}
		reader, err := item.Open()
		if err != nil {
			return 0, 0, err
		}
		decoder := xml.NewDecoder(io.LimitReader(reader, 64<<20))
		for {
			tok, readErr := decoder.Token()
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				reader.Close()
				return 0, 0, readErr
			}
			elem, ok := tok.(xml.StartElement)
			if !ok {
				continue
			}
			switch elem.Name.Local {
			case "object":
				objects++
			case "item":
				instances++
			}
			if objects > 100000 || instances > 100000 {
				reader.Close()
				return 0, 0, fmt.Errorf("unusual model object count")
			}
		}
		reader.Close()
		found = true
	}
	if !found {
		return 0, 0, fmt.Errorf("no 3D model XML found")
	}
	return objects, instances, nil
}
func sourceSummary(path string) (SourceSummary, error) {
	s := SourceSummary{Path: path, Name: filepath.Base(path)}
	if _, err := fileInfo(path); err != nil {
		return s, err
	}
	nozzle, err := bridge.InspectNozzle(path)
	if err != nil {
		return s, err
	}
	insp, err := converter.Inspect(path)
	if err != nil {
		return s, err
	}
	objects, instances, err := inspectModelCount(path)
	if err != nil {
		return s, err
	}
	s.Nozzle, _ = nozzle["source_nozzle"].(string)
	s.NozzleAssumed, _ = nozzle["source_nozzle_assumed"].(bool)
	s.Plates = insp.Plates
	s.Objects = objects
	s.Instances = instances
	s.Filaments = insp.Filaments
	return s, nil
}
func (a *App) InspectFile(path string) (SourceSummary, error) { return sourceSummary(path) }

// selectSnapmakerProfile chooses an actual installed, validated Snapmaker Orca
// profile over the historical built-in 0.4 mm U1 template. The template remains
// an offline fallback ONLY for the old default auto/0.4 path. An explicit user
// nozzle, slicer or filament preference never triggers a silent replacement.
func selectSnapmakerProfile(target, nozzle string, preference PrinterSettings,
	load func(string, string, string, map[string]string) (*presets.Bundle, error)) (bool, *presets.Bundle) {
	eligibleBuiltin := target == "u1" && nozzle == "0.4" &&
		preference.Slicer == "auto" && len(preference.Filaments) == 0
	if !eligibleBuiltin {
		return false, nil
	}
	vendor, err := load(target, nozzle, "auto", nil)
	if err != nil {
		return true, nil
	}
	return false, vendor
}

func configuredConvert(source, target, sourceNozzle string, preference PrinterSettings) ConvertResult {
	// The generic PrusaSlicer project has no per-printer settings entry, so the
	// preference argument is intentionally ignored for it.
	if target == prusaTargetID {
		return prusaConvert(source)
	}
	if presets.IsOrcaTargetID(target) {
		return installedConvert(source, target)
	}
	if presets.IsPrusaTargetID(target) {
		return prusaInstalledConvert(target)
	}
	result := ConvertResult{Target: target, Prefix: outputs[target]}
	preferred := preference.Nozzle
	if preferred == "auto" {
		preferred = ""
	}
	nativeNozzle := sourceNozzle
	if preferred != "" {
		nativeNozzle = preferred
	}
	_, _, _, _, vErr := presets.Variant(target, nativeNozzle)
	if vErr != nil {
		result.Code = "TARGET_NOZZLE_UNAVAILABLE"
		result.Error = vErr.Error()
		return result
	}
	if target == "c5" && nativeNozzle == "0.2" {
		nativeNozzle = "0.25"
	}
	useBuiltin, bundle := selectSnapmakerProfile(target, nativeNozzle, preference, presets.LoadConfigured)
	usedSlicer := "Built-in Snapmaker U1 template"
	profileFile := ""
	if !useBuiltin {
		if bundle == nil {
			var loadErr error
			bundle, loadErr = presets.LoadConfigured(target, nativeNozzle, preference.Slicer, preference.Filaments)
			if loadErr != nil {
				result.Code = "TARGET_PROFILE_INVALID"
				result.Error = loadErr.Error()
				return result
			}
		}
		usedSlicer = bundle.Slicer
		f, cleanup, err := tempVendorBundle(bundle)
		if err != nil {
			result.Code = "TARGET_PROFILE_INVALID"
			result.Error = err.Error()
			return result
		}
		defer cleanup()
		profileFile = f
	}
	return writeConversion(result, source, usedSlicer, preference.Filaments, func(out string) (map[string]any, error) {
		return bridge.ConvertWithTargetNozzle(source, out, target, profileFile, preferred)
	})
}

// installedConvert converts through an installed Orca-family machine profile.
// The profile itself fixes nozzle, tool count and model, so the per-printer
// preferences of the built-in targets intentionally do not apply.
func installedConvert(source, target string) ConvertResult {
	result := ConvertResult{Target: target, Prefix: presets.OrcaPrefix(target)}
	bundle, err := presets.ResolveOrcaBundle(target)
	if err != nil {
		result.Code = "TARGET_PROFILE_INVALID"
		result.Error = err.Error()
		return result
	}
	profileFile, cleanup, err := tempOrcaBundle(bundle)
	if err != nil {
		result.Code = "TARGET_PROFILE_INVALID"
		result.Error = err.Error()
		return result
	}
	defer cleanup()
	return writeConversion(result, source, bundle.Slicer, map[string]string{}, func(out string) (map[string]any, error) {
		return bridge.ConvertWithOrcaBundle(source, out, profileFile)
	})
}

// writeConversion publishes into the first free same-name output and enriches
// the report with target-side counts. Built-in and installed targets share it.
func writeConversion(result ConvertResult, source, slicer string, overrides map[string]string, convert func(out string) (map[string]any, error)) ConvertResult {
	dir := filepath.Dir(source)
	basename := filepath.Base(source)
	stem := strings.TrimSuffix(basename, filepath.Ext(basename))
	for attempt := 1; attempt <= 100; attempt++ {
		filename := result.Prefix + "-" + basename
		if attempt > 1 {
			filename = fmt.Sprintf("%s-%s-%d%s", result.Prefix, stem, attempt, filepath.Ext(basename))
		}
		out := filepath.Join(dir, filename)
		if _, err := os.Stat(out); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			result.Code = "OUTPUT_INVALID"
			result.Error = err.Error()
			return result
		}
		conversion, err := convert(out)
		if err != nil {
			if conversionErrorCode(err) == "OUTPUT_EXISTS" {
				continue
			}
			result.Code = conversionErrorCode(err)
			result.Error = err.Error()
			return result
		}
		conversion["native_slicer"] = slicer
		conversion["native_filament_overrides"] = overrides
		// Reinspect the written project so reports show real target-side
		// object/plate counts instead of assuming they equal the source.
		if objects, instances, scanErr := inspectModelCount(out); scanErr == nil {
			conversion["target_objects"] = objects
			conversion["target_instances"] = instances
		}
		// Callers that already measured the output (the PrusaSlicer path reads
		// its own slot count) keep their values; the Bambu-based reader is
		// not applied to foreign output formats.
		if targetInspection, scanErr := converter.Inspect(out); scanErr == nil {
			setIfAbsent(conversion, "target_plates", targetInspection.Plates)
			setIfAbsent(conversion, "target_filament_slots", len(targetInspection.Filaments))
		}
		result.OK = true
		result.OutputPath = out
		result.OutputName = filename
		result.Report = conversion
		return result
	}
	result.Code = "OUTPUT_COLLISIONS"
	result.Error = "Too many same-name outputs"
	return result
}

// conversionErrorCode prefers a code carried by a native engine error and falls
// back to the bridge codes used by the built-in and installed-profile paths.
func conversionErrorCode(err error) string {
	var coded interface{ conversionCode() string }
	if errors.As(err, &coded) {
		return coded.conversionCode()
	}
	return bridge.ErrorCode(err)
}

func setIfAbsent(report map[string]any, key string, value any) {
	if _, exists := report[key]; !exists {
		report[key] = value
	}
}

// tempOrcaBundle writes one resolved installed profile to a private temp file.
func tempOrcaBundle(bundle *presets.OrcaBundle) (string, func(), error) {
	dir, err := os.MkdirTemp("", "3mf-bridge-installed-profile-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	data, err := json.Marshal(bundle)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	file := filepath.Join(dir, "installed-profile.json")
	if err := os.WriteFile(file, data, 0600); err != nil {
		cleanup()
		return "", nil, err
	}
	return file, cleanup, nil
}

func validateTargets(targets []string) error {
	if len(targets) == 0 || len(targets) > maxMyPrinters {
		return fmt.Errorf("select between 1 and %d printers", maxMyPrinters)
	}
	seen := map[string]bool{}
	for _, t := range targets {
		_, builtIn := outputs[t]
		if (!builtIn && !presets.IsOrcaTargetID(t) && !presets.IsPrusaTargetID(t) && t != prusaTargetID) || seen[t] {
			return fmt.Errorf("invalid or repeated target %q", t)
		}
		seen[t] = true
	}
	return nil
}
func (a *App) emitProgress(progress BatchProgress) {
	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "bridge:batch-progress", progress)
	}
}
func (a *App) ConvertBatch(request BatchRequest) BatchReply {
	report := BatchReply{StartedAt: time.Now().UTC().Format(time.RFC3339), Files: []FileBatchResult{}}
	if err := validateTargets(request.Targets); err != nil {
		report.Error = err.Error()
		return report
	}
	if len(request.FilePaths) == 0 || len(request.FilePaths) > maxBatchFiles {
		report.Error = fmt.Sprintf("select between 1 and %d projects", maxBatchFiles)
		return report
	}
	settings, err := a.GetSettings()
	if err != nil {
		report.Error = err.Error()
		return report
	}
	seen := map[string]bool{}
	validated := []string{}
	for _, path := range request.FilePaths {
		file, err := fileInfo(path)
		if err != nil {
			report.Error = err.Error()
			return report
		}
		key := file.Path
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if seen[key] {
			report.Error = "repeated project path in batch"
			return report
		}
		seen[key] = true
		validated = append(validated, file.Path)
	}
	// Bound conversions: one source project at a time, with at most four
	// concurrent printer targets. Prevent large queues from exhausting RAM.
	for index, source := range validated {
		a.emitProgress(BatchProgress{FileIndex: index + 1, FileCount: len(validated), FileName: filepath.Base(source)})
		entry := FileBatchResult{
			Source:  SourceSummary{Path: source, Name: filepath.Base(source)},
			Results: []ConvertResult{}, Failures: []ConvertResult{},
		}
		summary, err := sourceSummary(source)
		if err != nil {
			entry.Error = err.Error()
			report.Failures += len(request.Targets)
			for _, target := range request.Targets {
				entry.Failures = append(entry.Failures, ConvertResult{Target: target, Code: "SOURCE_INVALID", Error: err.Error()})
			}
		} else {
			entry.Source = summary
			results := make([]ConvertResult, len(request.Targets))
			var wg sync.WaitGroup
			// Even with up to 32 selected printers, at most four 3MF
			// conversions run concurrently to bound CPU, RAM and disk use.
			permits := make(chan struct{}, 4)
			for i, target := range request.Targets {
				wg.Add(1)
				go func(index int, t string) {
					defer wg.Done()
					permits <- struct{}{}
					defer func() { <-permits }()
					results[index] = configuredConvert(source, t, summary.Nozzle, settings.Printers[t])
				}(i, target)
			}
			wg.Wait()
			for _, result := range results {
				if result.OK {
					entry.Results = append(entry.Results, result)
					report.Successes++
					a.mutex.Lock()
					if a.generated == nil {
						a.generated = map[string]string{}
					}
					a.generated[result.OutputPath] = result.Target
					a.lastFolder = filepath.Dir(result.OutputPath)
					a.mutex.Unlock()
				} else {
					entry.Failures = append(entry.Failures, result)
					report.Failures++
				}
			}
		}
		report.Files = append(report.Files, entry)
		report.FilesDone++
		a.emitProgress(BatchProgress{FileIndex: index + 1, FileCount: len(validated), FileName: filepath.Base(source), Completed: true})
	}
	report.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	a.mutex.Lock()
	a.latestBatch = report
	a.mutex.Unlock()
	return report
}

func (a *App) OpenInSlicer(target, outputPath string) error {
	a.mutex.Lock()
	actual := a.generated[outputPath]
	a.mutex.Unlock()
	if target == "" || target != actual {
		return fmt.Errorf("file was not generated for this printer in the current session")
	}
	f, err := fileInfo(outputPath)
	if err != nil {
		return err
	}
	if f.Path != outputPath {
		return fmt.Errorf("invalid output path")
	}
	if target == prusaTargetID {
		return launchPrusaSlicer(outputPath)
	}
	var exe, slicer string
	if presets.IsOrcaTargetID(target) {
		// Re-resolve the installed machine profile from its opaque target ID,
		// and launch only the verified app that owns those exact profiles.
		// Custom profile-only roots must fail closed (no guessed executable).
		exe, slicer, err = presets.ResolveOrcaExecutable(target)
	} else {
		var settings Settings
		settings, err = a.GetSettings()
		if err == nil {
			_, exe, slicer, err = presets.ResolveSlicer(target, settings.Printers[target].Slicer)
		}
	}
	if err != nil {
		return err
	}
	if exe == "" {
		return fmt.Errorf("%s executable is not available on this computer", slicer)
	}
	if info, err := os.Stat(exe); err != nil || (runtime.GOOS == "darwin" && !info.IsDir()) || (runtime.GOOS == "windows" && info.IsDir()) {
		return fmt.Errorf("%s executable is missing", slicer)
	}
	if runtime.GOOS == "darwin" {
		return exec.Command("open", "-a", exe, outputPath).Start()
	}
	return exec.Command(exe, outputPath).Start()
}
