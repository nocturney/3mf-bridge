package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/BryanSmee/bl_to_std/pkg/converter"
)

// The Panda2Prusa engine is a vendor Python program shipped as a sidecar
// executable next to SliceBridge. Go never converts geometry or settings: it
// only locates the sidecar, runs it with a bounded timeout and reply size,
// and verifies the published 3MF before reporting success.

const (
	prusaTargetID    = "prusa-project"
	prusaPrefix      = "PRUSA"
	prusaSlicerName  = "PrusaSlicer"
	prusaEngineName  = "slicebridge-prusa-engine"
	prusaEngineID    = "panda2prusa"
	prusaEngineEnv   = "SLICEBRIDGE_PRUSA_ENGINE"
	prusaReplyLimit  = 64 << 10
	prusaTextLimit   = 300
	prusaEntryLimit  = 10000
	prusaUnzipLimit  = 512 << 20
	prusaModelLimit  = 256 << 20
	prusaConfigLimit = 8 << 20
)

// prusaEngineTimeout bounds one sidecar run. Tests shorten it.
var prusaEngineTimeout = 2 * time.Minute

// prusaRun and prusaSourceSlots are seams: production runs the located sidecar
// and reads the real source project; tests substitute deterministic doubles.
var prusaRun = runPrusaProcess
var prusaSourceSlots = sourceFilamentCount

var prusaCodePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,63}$`)
var errPrusaOutputLimit = errors.New("engine reply exceeds the size limit")

// prusaError carries a stable UPPERCASE code that writeConversion reports.
type prusaError struct {
	code string
	msg  string
}

func (e *prusaError) Error() string          { return e.msg }
func (e *prusaError) conversionCode() string { return e.code }

func prusaFail(code, format string, args ...any) error {
	return &prusaError{code: code, msg: fmt.Sprintf(format, args...)}
}

// PrusaEngineStatus reports whether the sidecar file exists. It is a file-level
// probe only and never executes the engine.
type PrusaEngineStatus struct {
	Available bool   `json:"available"`
	Engine    string `json:"engine"`
	Source    string `json:"source,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

func (a *App) PrusaEngineStatus() PrusaEngineStatus {
	exe, source, reason := locatePrusaEngine()
	status := PrusaEngineStatus{Engine: prusaEngineID, Reason: reason}
	if exe != "" {
		status.Available = true
		status.Source = source
	}
	return status
}

// locatePrusaEngine returns the single executable allowed for the sidecar.
// There is no PATH lookup and no system Python fallback in any build.
func locatePrusaEngine() (exe, source, reason string) {
	if override := os.Getenv(prusaEngineEnv); override != "" {
		// Developer and test override only: it must be an absolute path to a
		// real executable, and it replaces the bundled location entirely.
		if !filepath.IsAbs(override) || requireRegularExecutable(override) != nil {
			return "", "developer", "invalid_override"
		}
		return override, "developer", ""
	}
	self, err := os.Executable()
	if err != nil {
		return "", "bundled", "missing"
	}
	name := prusaEngineName
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	candidate := filepath.Join(filepath.Dir(self), name)
	if runtime.GOOS == "darwin" {
		// SliceBridge.app/Contents/MacOS/SliceBridge -> Contents/Resources/
		candidate = filepath.Join(filepath.Dir(self), "..", "Resources", name)
	}
	if requireRegularExecutable(candidate) != nil {
		return "", "bundled", "missing"
	}
	return candidate, "bundled", ""
}

// requireRegularExecutable rejects symlinks, directories and non-executables.
func requireRegularExecutable(file string) error {
	info, err := os.Lstat(file)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file")
	}
	if runtime.GOOS == "windows" {
		if !strings.EqualFold(filepath.Ext(file), ".exe") {
			return fmt.Errorf("not an .exe file")
		}
	} else if info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("not executable")
	}
	return nil
}

func sourceFilamentCount(source string) (int, error) {
	insp, err := converter.Inspect(source)
	if err != nil {
		return 0, err
	}
	return len(insp.Filaments), nil
}

// prusaConvert writes one generic PrusaSlicer project. The printer model is
// deliberately not chosen here; the user selects it in PrusaSlicer after import.
func prusaConvert(source string) ConvertResult {
	result := ConvertResult{Target: prusaTargetID, Prefix: prusaPrefix}
	exe, _, reason := locatePrusaEngine()
	switch reason {
	case "":
	case "invalid_override":
		result.Code = "PRUSA_ENGINE_INVALID"
		result.Error = prusaEngineEnv + " must name an absolute regular executable"
		return result
	default:
		result.Code = "PRUSA_ENGINE_MISSING"
		result.Error = "The PrusaSlicer project engine is not installed with this build"
		return result
	}
	abs, err := filepath.Abs(source)
	if err != nil {
		result.Code = "SOURCE_INVALID"
		result.Error = err.Error()
		return result
	}
	sourceSlots, err := prusaSourceSlots(abs)
	if err != nil {
		result.Code = "SOURCE_INVALID"
		result.Error = err.Error()
		return result
	}
	if sourceSlots == 0 {
		result.Code = "SOURCE_INVALID"
		result.Error = "source project has no filament slots"
		return result
	}
	return writeConversion(result, abs, prusaSlicerName, map[string]string{}, func(out string) (map[string]any, error) {
		return runPrusaConversion(exe, abs, out, sourceSlots)
	})
}

// runPrusaConversion runs the sidecar once for one output path and returns a
// report only after the published 3MF passes content checks.
func runPrusaConversion(exe, source, out string, sourceSlots int) (map[string]any, error) {
	if samePrusaPath(source, out) {
		return nil, prusaFail("PRUSA_SAME_PATH", "engine output must differ from the source project")
	}
	ctx, cancel := context.WithTimeout(context.Background(), prusaEngineTimeout)
	defer cancel()
	stdout, runErr := prusaRun(ctx, exe, []string{"--source", source, "--output", out, "--json"})
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, prusaFail("PRUSA_ENGINE_TIMEOUT", "PrusaSlicer project engine exceeded %s", prusaEngineTimeout)
	}
	if errors.Is(runErr, errPrusaOutputLimit) {
		return nil, prusaFail("PRUSA_ENGINE_OUTPUT_LIMIT", "engine reply exceeded %d bytes", prusaReplyLimit)
	}
	if err := prusaOutcome(stdout, runErr); err != nil {
		return nil, err
	}
	// The engine publishes atomically. This check decides whether the file is
	// really a usable project; a rejected file is removed so it cannot be opened.
	facts, err := validatePrusaOutput(out)
	if err != nil {
		_ = os.Remove(out)
		return nil, err
	}
	if err := requirePreservedSlots(sourceSlots, facts.filamentSlots); err != nil {
		_ = os.Remove(out)
		return nil, err
	}
	return map[string]any{
		"prusa_engine":               prusaEngineID,
		"printer_profile_generic":    true,
		"source_filament_slots":      sourceSlots,
		"target_filament_slots":      facts.filamentSlots,
		"slots_preserved":            true,
		"source_slots_metadata_only": true,
		"warnings": []string{"Source filament IDs are retained in SliceBridge 3MF metadata. Select your PrusaSlicer printer and filament presets before slicing; those presets are not installed or changed."},
		"target_triangles":           facts.triangles,
		"target_plates":              0, // PrusaSlicer plates are not read; shown as unknown.
		"native_filament_overrides":  map[string]string{},
		"user_slot_mapping_required": false,
	}, nil
}

func samePrusaPath(a, b string) bool {
	ca, cb := filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(ca, cb)
	}
	return ca == cb
}

// runPrusaProcess runs the sidecar and returns its bounded stdout. The child is
// killed as soon as the reply overflows, so it cannot block on a full pipe.
func runPrusaProcess(ctx context.Context, exe string, args []string) ([]byte, error) {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stdout := &boundedBuffer{limit: prusaReplyLimit, stop: cancel}
	cmd := exec.CommandContext(runCtx, exe, args...)
	cmd.Dir = filepath.Dir(exe)
	cmd.Stdout = stdout
	cmd.Stderr = io.Discard
	err := cmd.Run()
	if stdout.overflow {
		return nil, errPrusaOutputLimit
	}
	return stdout.data.Bytes(), err
}

type boundedBuffer struct {
	data     bytes.Buffer
	limit    int
	overflow bool
	stop     func()
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.overflow || b.data.Len()+len(p) > b.limit {
		b.overflow = true
		if b.stop != nil {
			b.stop()
		}
		return 0, errPrusaOutputLimit
	}
	return b.data.Write(p)
}

type prusaEngineReply struct {
	OK     *bool  `json:"ok"`
	Engine string `json:"engine"`
	Code   string `json:"code"`
	Error  string `json:"error"`
}

// parsePrusaReply requires exactly one JSON object with an ok field.
func parsePrusaReply(data []byte) (prusaEngineReply, error) {
	var reply prusaEngineReply
	if len(data) == 0 || len(data) > prusaReplyLimit {
		return reply, errors.New("empty or oversized reply")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&reply); err != nil {
		return reply, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return reply, errors.New("trailing data after the JSON object")
	}
	if reply.OK == nil {
		return reply, errors.New("missing ok field")
	}
	return reply, nil
}

// prusaOutcome maps the sidecar's stdout and exit status to one stable result.
// Every malformed reply collapses to PRUSA_ENGINE_BAD_JSON, so the same fault
// always produces the same UI code.
func prusaOutcome(stdout []byte, runErr error) error {
	reply, err := parsePrusaReply(stdout)
	if err != nil {
		if runErr != nil {
			return prusaFail("PRUSA_ENGINE_FAILED", "PrusaSlicer project engine exited with an error")
		}
		return prusaFail("PRUSA_ENGINE_BAD_JSON", "engine reply is not a single JSON object")
	}
	if !*reply.OK {
		if !prusaCodePattern.MatchString(reply.Code) {
			return prusaFail("PRUSA_ENGINE_BAD_JSON", "engine failure carries no valid code")
		}
		message := sanitizeEngineText(reply.Error)
		if message == "" {
			message = reply.Code
		}
		return &prusaError{code: reply.Code, msg: message}
	}
	if runErr != nil {
		return prusaFail("PRUSA_ENGINE_FAILED", "PrusaSlicer project engine exited with an error")
	}
	if reply.Engine != prusaEngineID {
		return prusaFail("PRUSA_ENGINE_BAD_JSON", "unexpected engine identity in reply")
	}
	return nil
}

// sanitizeEngineText removes control characters and bounds length before the
// engine's message is shown in the UI.
func sanitizeEngineText(text string) string {
	runes := make([]rune, 0, len(text))
	for _, r := range text {
		if unicode.IsControl(r) {
			r = ' '
		}
		runes = append(runes, r)
	}
	runes = []rune(strings.TrimSpace(string(runes)))
	if len(runes) > prusaTextLimit {
		runes = runes[:prusaTextLimit]
	}
	return string(runes)
}

type prusaOutputFacts struct {
	triangles     int
	filamentSlots int
}

// validatePrusaOutput inspects the ZIP content itself: safe entry names, the
// content-types part, editable mesh geometry, a lossless source-slot record,
// and no stale sliced G-code.
func validatePrusaOutput(output string) (prusaOutputFacts, error) {
	var facts prusaOutputFacts
	invalid := func(message string) error {
		return prusaFail("PRUSA_OUTPUT_INVALID", "%s", message)
	}
	info, err := os.Lstat(output)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > max3MFSize {
		return facts, invalid("engine output is not a regular 3MF file")
	}
	zr, err := zip.OpenReader(output)
	if err != nil {
		return facts, invalid("engine output is not a valid 3MF archive")
	}
	defer zr.Close()
	if len(zr.File) > prusaEntryLimit {
		return facts, invalid("engine output has too many archive entries")
	}
	seen := map[string]bool{}
	var total uint64
	hasContentTypes, hasSlotConfig := false, false
	for _, f := range zr.File {
		name := strings.TrimSuffix(f.Name, "/")
		if name == "" || path.Clean(name) != name || strings.HasPrefix(name, "/") ||
			strings.HasPrefix(name, "../") || strings.Contains(name, "\\") {
			return facts, invalid("engine output contains an unsafe archive entry")
		}
		if seen[name] {
			return facts, invalid("engine output contains a duplicate archive entry")
		}
		seen[name] = true
		if f.UncompressedSize64 > prusaUnzipLimit || total+f.UncompressedSize64 > prusaUnzipLimit {
			return facts, invalid("engine output exceeds the decompressed size limit")
		}
		total += f.UncompressedSize64
		lower := strings.ToLower(name)
		switch {
		case name == "[Content_Types].xml":
			hasContentTypes = true
		case strings.HasPrefix(name, "Metadata/") && strings.Contains(lower, ".gcode"):
			return facts, prusaFail("PRUSA_STALE_GCODE", "engine output still contains sliced machine code")
		case strings.HasPrefix(name, "3D/") && strings.HasSuffix(lower, ".model"):
			triangles, err := countPrusaTriangles(f)
			if err != nil {
				return facts, invalid("engine output contains invalid 3D model XML")
			}
			facts.triangles += triangles
		case name == "Metadata/SliceBridge_source_filaments.json":
			// Panda2Prusa intentionally does not write a Prusa printer or
			// filament preset. Our auxiliary JSON record preserves EVERY
			// original numbered source slot without overriding user presets.
			data, err := readZipEntry(f, prusaConfigLimit)
			if err != nil {
				return facts, invalid("engine output source slot record cannot be read")
			}
			slots, err := countPreservedSourceSlots(data)
			if err != nil {
				return facts, invalid("engine output source slot record is invalid: " + err.Error())
			}
			facts.filamentSlots = slots
			hasSlotConfig = true
		}
	}
	if !hasContentTypes {
		return facts, invalid("engine output is missing [Content_Types].xml")
	}
	if facts.triangles == 0 {
		return facts, invalid("engine output has no editable 3D mesh geometry")
	}
	if !hasSlotConfig {
		return facts, invalid("engine output does not declare filament slots")
	}
	return facts, nil
}

func countPrusaTriangles(f *zip.File) (int, error) {
	rc, err := f.Open()
	if err != nil {
		return 0, err
	}
	defer rc.Close()
	count := 0
	decoder := xml.NewDecoder(io.LimitReader(rc, prusaModelLimit))
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			return count, nil
		}
		if err != nil {
			return 0, err
		}
		if start, ok := tok.(xml.StartElement); ok && start.Name.Local == "triangle" {
			count++
		}
	}
}

func readZipEntry(f *zip.File, limit int64) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("entry exceeds the size limit")
	}
	return data, nil
}

// countPreservedSourceSlots validates SliceBridge's lossless auxiliary metadata.
// It is not a native PrusaSlicer filament preset; its job is to retain each
// source slot, including unused and identically coloured distinct slots.
func countPreservedSourceSlots(data []byte) (int, error) {
	var record struct {
		SchemaVersion int    `json:"schemaVersion"`
		Source        string `json:"source"`
		Slots         []struct {
			ID       int     `json:"id"`
			Color    *string `json:"color"`
			Material *string `json:"material"`
		} `json:"slots"`
	}
	if err := json.Unmarshal(data, &record); err != nil {
		return 0, fmt.Errorf("invalid JSON: %w", err)
	}
	if record.SchemaVersion != 1 || record.Source != "bambu-orca" {
		return 0, fmt.Errorf("unknown schema or source")
	}
	if len(record.Slots) == 0 || len(record.Slots) > 64 {
		return 0, fmt.Errorf("source slot count outside 1..64")
	}
	for index, slot := range record.Slots {
		if slot.ID != index+1 {
			return 0, fmt.Errorf("source slot IDs must remain ordered and distinct")
		}
	}
	return len(record.Slots), nil
}

// requirePreservedSlots enforces the project rule that every source filament
// remains its own slot. Fewer output slots means the engine merged or dropped one.
func requirePreservedSlots(sourceSlots, outputSlots int) error {
	if outputSlots < sourceSlots {
		return prusaFail("PRUSA_SLOTS_NOT_PRESERVED", "source has %d filament slots but the output has %d", sourceSlots, outputSlots)
	}
	return nil
}

// launchPrusaSlicer opens a generated project only from fixed standard install
// locations. Otherwise it returns an error and the UI offers the output folder.
func launchPrusaSlicer(outputPath string) error {
	for _, candidate := range prusaSlicerCandidates() {
		info, err := os.Lstat(candidate)
		if err != nil {
			continue
		}
		if runtime.GOOS == "darwin" {
			if info.IsDir() {
				return exec.Command("open", "-a", candidate, outputPath).Start()
			}
			continue
		}
		if info.Mode().IsRegular() {
			return exec.Command(candidate, outputPath).Start()
		}
	}
	return fmt.Errorf("PrusaSlicer is not installed in a standard location; use Open output folder instead")
}

func prusaSlicerCandidates() []string {
	switch runtime.GOOS {
	case "darwin":
		candidates := []string{"/Applications/PrusaSlicer.app"}
		if home, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates, filepath.Join(home, "Applications", "PrusaSlicer.app"))
		}
		return candidates
	case "windows":
		if programFiles := os.Getenv("ProgramFiles"); programFiles != "" {
			return []string{filepath.Join(programFiles, "Prusa3D", "PrusaSlicer", "prusa-slicer.exe")}
		}
	}
	return nil
}
