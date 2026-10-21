package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const prusaTestModel = `<?xml version="1.0" encoding="UTF-8"?>
<model unit="millimeter" xmlns="http://schemas.microsoft.com/3dmanufacturing/core/2015/02"><resources><object id="1" type="model"><mesh><vertices><vertex x="0" y="0" z="0"/><vertex x="1" y="0" z="0"/><vertex x="0" y="1" z="0"/></vertices><triangles><triangle v1="0" v2="1" v3="2"/></triangles></mesh></object></resources><build><item objectid="1"/></build></model>`

const prusaTestContentTypes = `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>`

const prusaOKReply = `{"ok":true,"engine":"panda2prusa"}`

// validPrusaEntries is a minimal PrusaSlicer-style project with one slot per
// comma-free colour entry in slotColours (for example "#FF0000;#00FF00").
func validPrusaEntries(slotColours string) map[string]string {
	colours := strings.Split(slotColours, ";")
	slots := make([]map[string]any, 0, len(colours))
	for i, colour := range colours {
		slots = append(slots, map[string]any{
			"id": i + 1, "color": colour, "material": "PLA",
		})
	}
	record, _ := json.Marshal(map[string]any{
		"schemaVersion": 1, "source": "bambu-orca", "slots": slots,
	})
	return map[string]string{
		"[Content_Types].xml":                          prusaTestContentTypes,
		"3D/3dmodel.model":                             prusaTestModel,
		"Metadata/SliceBridge_source_filaments.json": string(record),
	}
}

func writePrusaTestZip(t *testing.T, file string, entries map[string]string) {
	t.Helper()
	out, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	zw := zip.NewWriter(out)
	for name, body := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

// useFakePrusaEngine points the developer override at a regular executable
// placeholder. Tests replace what actually runs via stubPrusaEngine.
func useFakePrusaEngine(t *testing.T) string {
	t.Helper()
	name := "panda-test-engine"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	exe := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(exe, []byte("placeholder"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(prusaEngineEnv, exe)
	return exe
}

type prusaRunner = func(context.Context, string, []string) ([]byte, error)

// stubPrusaEngine replaces the sidecar run and the source slot reader for one
// test, restoring the production values afterwards.
func stubPrusaEngine(t *testing.T, sourceSlots int, run prusaRunner) {
	t.Helper()
	savedRun, savedSlots, savedTimeout := prusaRun, prusaSourceSlots, prusaEngineTimeout
	t.Cleanup(func() {
		prusaRun, prusaSourceSlots, prusaEngineTimeout = savedRun, savedSlots, savedTimeout
	})
	prusaRun = run
	prusaSourceSlots = func(string) (int, error) { return sourceSlots, nil }
}

func prusaArg(t *testing.T, args []string, flag string) string {
	t.Helper()
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	t.Fatalf("engine was not given %s: %v", flag, args)
	return ""
}

// engineWriting simulates a sidecar that publishes an output and replies ok.
func engineWriting(t *testing.T, write func(t *testing.T, out string)) prusaRunner {
	return func(_ context.Context, _ string, args []string) ([]byte, error) {
		write(t, prusaArg(t, args, "--output"))
		return []byte(prusaOKReply), nil
	}
}

func zipOutput(entries map[string]string) func(*testing.T, string) {
	return func(t *testing.T, out string) { writePrusaTestZip(t, out, entries) }
}

func prusaTestSource(t *testing.T) string {
	t.Helper()
	source := filepath.Join(t.TempDir(), "Project.3mf")
	if err := os.WriteFile(source, []byte("source-project-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	return source
}

func readTestFile(t *testing.T, file string) string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func countTestEntries(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

func TestPrusaSidecarAbsenceFailsClosedWithoutRunning(t *testing.T) {
	t.Setenv(prusaEngineEnv, "")
	status := (&App{}).PrusaEngineStatus()
	if status.Available || status.Reason != "missing" || status.Engine != prusaEngineID {
		t.Fatalf("absent sidecar must report unavailable, got %+v", status)
	}
	source := prusaTestSource(t)
	stubPrusaEngine(t, 2, func(context.Context, string, []string) ([]byte, error) {
		t.Fatal("engine must not run when the sidecar is absent")
		return nil, nil
	})
	result := configuredConvert(source, prusaTargetID, "0.4", PrinterSettings{})
	if result.OK || result.Code != "PRUSA_ENGINE_MISSING" {
		t.Fatalf("absent sidecar must fail closed, got ok=%v code=%q", result.OK, result.Code)
	}
	if n := countTestEntries(t, filepath.Dir(source)); n != 1 {
		t.Fatalf("no output may be created when the sidecar is absent, folder has %d entries", n)
	}
}

func TestPrusaStatusDeveloperOverrideMustBeAbsoluteRegularExecutable(t *testing.T) {
	t.Setenv(prusaEngineEnv, "relative-engine")
	if status := (&App{}).PrusaEngineStatus(); status.Available || status.Reason != "invalid_override" {
		t.Fatalf("relative override must be rejected, got %+v", status)
	}
	t.Setenv(prusaEngineEnv, t.TempDir())
	if status := (&App{}).PrusaEngineStatus(); status.Available || status.Reason != "invalid_override" {
		t.Fatalf("directory override must be rejected, got %+v", status)
	}
	plain := filepath.Join(t.TempDir(), "engine-without-exec")
	if err := os.WriteFile(plain, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(prusaEngineEnv, plain)
	if status := (&App{}).PrusaEngineStatus(); status.Available {
		t.Fatalf("non-executable override must be rejected, got %+v", status)
	}

	exe := useFakePrusaEngine(t)
	status := (&App{}).PrusaEngineStatus()
	if !status.Available || status.Source != "developer" || status.Reason != "" {
		t.Fatalf("valid developer override must be available, got %+v", status)
	}
	encoded, err := json.Marshal(status)
	if err != nil || !strings.Contains(string(encoded), `"available":true`) {
		t.Fatalf("status must marshal with an available flag: %s %v", encoded, err)
	}
	if strings.Contains(string(encoded), exe) {
		t.Fatal("status must not expose the sidecar path to the UI")
	}

	source := prusaTestSource(t)
	t.Setenv(prusaEngineEnv, "relative-engine")
	if result := configuredConvert(source, prusaTargetID, "0.4", PrinterSettings{}); result.Code != "PRUSA_ENGINE_INVALID" {
		t.Fatalf("invalid override must fail closed, got %q", result.Code)
	}
}

func TestPrusaReplyProtocolMapsToStableCodes(t *testing.T) {
	cases := []struct {
		name   string
		stdout string
		runErr error
		code   string
	}{
		{"success", prusaOKReply, nil, ""},
		{"wrong engine identity", `{"ok":true,"engine":"other"}`, nil, "PRUSA_ENGINE_BAD_JSON"},
		{"ok but exit error", prusaOKReply, errors.New("exit status 1"), "PRUSA_ENGINE_FAILED"},
		{"engine failure code kept", `{"ok":false,"code":"OUTPUT_EXISTS","error":"exists"}`, nil, "OUTPUT_EXISTS"},
		{"failure with exit error keeps code", `{"ok":false,"code":"SOURCE_INVALID"}`, errors.New("exit status 2"), "SOURCE_INVALID"},
		{"lowercase code rejected", `{"ok":false,"code":"bad_code"}`, nil, "PRUSA_ENGINE_BAD_JSON"},
		{"traceback without JSON", "Traceback (most recent call last)", errors.New("exit status 1"), "PRUSA_ENGINE_FAILED"},
		{"plain text on success exit", "hello", nil, "PRUSA_ENGINE_BAD_JSON"},
		{"trailing data rejected", prusaOKReply + " {}", nil, "PRUSA_ENGINE_BAD_JSON"},
		{"missing ok rejected", `{"engine":"panda2prusa"}`, nil, "PRUSA_ENGINE_BAD_JSON"},
		{"array rejected", `[]`, nil, "PRUSA_ENGINE_BAD_JSON"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := prusaOutcome([]byte(tc.stdout), tc.runErr)
			got := ""
			if err != nil {
				got = conversionErrorCode(err)
			}
			if got != tc.code {
				t.Fatalf("code = %q, want %q (err %v)", got, tc.code, err)
			}
		})
	}
}

func TestPrusaEngineMessagesAreSanitizedAndBounded(t *testing.T) {
	got := sanitizeEngineText("line\x00one\n" + strings.Repeat("x", 400))
	if strings.ContainsRune(got, 0) || strings.ContainsRune(got, '\n') {
		t.Fatalf("control characters survived: %q", got)
	}
	if n := len([]rune(got)); n != prusaTextLimit {
		t.Fatalf("message length = %d runes, want %d", n, prusaTextLimit)
	}
}

func TestPrusaVerifiedOutputPublishesWithPrusaPrefix(t *testing.T) {
	useFakePrusaEngine(t)
	source := prusaTestSource(t)
	before := readTestFile(t, source)
	stubPrusaEngine(t, 2, func(ctx context.Context, exe string, args []string) ([]byte, error) {
		if exe != os.Getenv(prusaEngineEnv) {
			t.Fatalf("engine must run from the located sidecar, got %q", exe)
		}
		if prusaArg(t, args, "--source") != source || args[len(args)-1] != "--json" {
			t.Fatalf("unexpected engine arguments: %v", args)
		}
		return engineWriting(t, zipOutput(validPrusaEntries("#FF0000;#00FF00")))(ctx, exe, args)
	})

	result := configuredConvert(source, prusaTargetID, "0.4", PrinterSettings{})
	if !result.OK {
		t.Fatalf("verified output must succeed: %s %s", result.Code, result.Error)
	}
	if result.OutputName != "PRUSA-Project.3mf" || !strings.HasSuffix(result.OutputPath, "PRUSA-Project.3mf") {
		t.Fatalf("output naming = %q, want PRUSA-Project.3mf", result.OutputName)
	}
	if got := readTestFile(t, source); got != before {
		t.Fatal("the source project must not change")
	}
	report := result.Report
	if report["native_slicer"] != "PrusaSlicer" || report["prusa_engine"] != prusaEngineID {
		t.Fatalf("report identity wrong: %v", report)
	}
	if report["source_filament_slots"] != 2 || report["target_filament_slots"] != 2 || report["slots_preserved"] != true {
		t.Fatalf("slot preservation not reported: %v", report)
	}
	if report["printer_profile_generic"] != true {
		t.Fatalf("generic printer profile must be declared: %v", report)
	}
}

func TestPrusaSlotLossIsRejectedAndOutputRemoved(t *testing.T) {
	useFakePrusaEngine(t)
	source := prusaTestSource(t)
	before := readTestFile(t, source)
	stubPrusaEngine(t, 2, engineWriting(t, zipOutput(validPrusaEntries("#FF0000"))))

	result := configuredConvert(source, prusaTargetID, "0.4", PrinterSettings{})
	if result.OK || result.Code != "PRUSA_SLOTS_NOT_PRESERVED" {
		t.Fatalf("merged slots must fail closed, got ok=%v code=%q", result.OK, result.Code)
	}
	if n := countTestEntries(t, filepath.Dir(source)); n != 1 {
		t.Fatalf("rejected output must be removed, folder has %d entries", n)
	}
	if readTestFile(t, source) != before {
		t.Fatal("the source project must not change")
	}
}

func TestPrusaInvalidOutputIsRejectedBeforeSuccess(t *testing.T) {
	noContentTypes := validPrusaEntries("#FF0000;#00FF00")
	delete(noContentTypes, "[Content_Types].xml")
	noGeometry := validPrusaEntries("#FF0000;#00FF00")
	noGeometry["3D/3dmodel.model"] = `<model><resources/></model>`
	noSlots := validPrusaEntries("#FF0000;#00FF00")
	delete(noSlots, "Metadata/SliceBridge_source_filaments.json")
	stale := validPrusaEntries("#FF0000;#00FF00")
	stale["Metadata/plate_1.gcode"] = "G28"
	unsafe := validPrusaEntries("#FF0000;#00FF00")
	unsafe["../escape.model"] = prusaTestModel

	cases := []struct {
		name  string
		write func(t *testing.T, out string)
		code  string
	}{
		{"not a zip", func(t *testing.T, out string) {
			if err := os.WriteFile(out, []byte("not a zip"), 0o600); err != nil {
				t.Fatal(err)
			}
		}, "PRUSA_OUTPUT_INVALID"},
		{"missing content types", zipOutput(noContentTypes), "PRUSA_OUTPUT_INVALID"},
		{"no editable geometry", zipOutput(noGeometry), "PRUSA_OUTPUT_INVALID"},
		{"no declared slots", zipOutput(noSlots), "PRUSA_OUTPUT_INVALID"},
		{"sliced machine code", zipOutput(stale), "PRUSA_STALE_GCODE"},
		{"unsafe entry", zipOutput(unsafe), "PRUSA_OUTPUT_INVALID"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			useFakePrusaEngine(t)
			source := prusaTestSource(t)
			stubPrusaEngine(t, 2, engineWriting(t, tc.write))
			result := configuredConvert(source, prusaTargetID, "0.4", PrinterSettings{})
			if result.OK || result.Code != tc.code {
				t.Fatalf("code = %q ok=%v, want %q", result.Code, result.OK, tc.code)
			}
			if n := countTestEntries(t, filepath.Dir(source)); n != 1 {
				t.Fatalf("rejected output must be removed, folder has %d entries", n)
			}
		})
	}
}

func TestPrusaEngineFailureLeavesNoOutput(t *testing.T) {
	useFakePrusaEngine(t)
	source := prusaTestSource(t)
	stubPrusaEngine(t, 2, func(context.Context, string, []string) ([]byte, error) {
		return []byte(`{"ok":false,"code":"CONVERSION_FAILED","error":"boom"}`), errors.New("exit status 1")
	})
	result := configuredConvert(source, prusaTargetID, "0.4", PrinterSettings{})
	if result.OK || result.Code != "CONVERSION_FAILED" || result.Error != "boom" {
		t.Fatalf("engine failure must pass through its code, got %+v", result)
	}
	if n := countTestEntries(t, filepath.Dir(source)); n != 1 {
		t.Fatalf("failed run must leave no output, folder has %d entries", n)
	}
}

func TestPrusaTimeoutAndOversizedReplyFailClosed(t *testing.T) {
	useFakePrusaEngine(t)
	source := prusaTestSource(t)

	stubPrusaEngine(t, 2, func(ctx context.Context, _ string, _ []string) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	prusaEngineTimeout = 20 * time.Millisecond
	if result := configuredConvert(source, prusaTargetID, "0.4", PrinterSettings{}); result.Code != "PRUSA_ENGINE_TIMEOUT" {
		t.Fatalf("timeout code = %q", result.Code)
	}

	stubPrusaEngine(t, 2, func(context.Context, string, []string) ([]byte, error) {
		return nil, errPrusaOutputLimit
	})
	if result := configuredConvert(source, prusaTargetID, "0.4", PrinterSettings{}); result.Code != "PRUSA_ENGINE_OUTPUT_LIMIT" {
		t.Fatalf("oversized reply code = %q", result.Code)
	}
	if n := countTestEntries(t, filepath.Dir(source)); n != 1 {
		t.Fatalf("fail-closed runs must leave no output, folder has %d entries", n)
	}
}

func TestPrusaBoundedBufferStopsAtLimit(t *testing.T) {
	stopped := false
	buf := &boundedBuffer{limit: 4, stop: func() { stopped = true }}
	if _, err := buf.Write([]byte("1234")); err != nil {
		t.Fatalf("write within limit failed: %v", err)
	}
	if _, err := buf.Write([]byte("5")); !errors.Is(err, errPrusaOutputLimit) {
		t.Fatalf("write past limit must fail, got %v", err)
	}
	if !buf.overflow || !stopped {
		t.Fatal("overflow must be recorded and the child stopped")
	}
}

func TestPrusaSourceAndOutputMustDiffer(t *testing.T) {
	source := prusaTestSource(t)
	stubPrusaEngine(t, 2, func(context.Context, string, []string) ([]byte, error) {
		t.Fatal("engine must not run when source and output are the same file")
		return nil, nil
	})
	_, err := runPrusaConversion(filepath.Join(t.TempDir(), "engine"), source, source, 2)
	if conversionErrorCode(err) != "PRUSA_SAME_PATH" {
		t.Fatalf("same path code = %q (err %v)", conversionErrorCode(err), err)
	}
}

func TestPrusaSourceSlotRecordAndPreservationRule(t *testing.T) {
	cases := []struct {
		config string
		slots  int
		valid  bool
	}{
		{`{"schemaVersion":1,"source":"bambu-orca","slots":[{"id":1,"color":"#FF0000","material":"PLA"},{"id":2,"color":"#FF0000","material":"PLA"},{"id":3,"color":"#00FF00","material":"PETG"}]}`, 3, true},
		{`{"schemaVersion":1,"source":"bambu-orca","slots":[{"id":1,"color":null,"material":null}]}`, 1, true},
		{`{"schemaVersion":1,"source":"bambu-orca","slots":[]}`, 0, false},
		{`{"schemaVersion":2,"source":"bambu-orca","slots":[{"id":1}]}`, 0, false},
		{`{"schemaVersion":1,"source":"bambu-orca","slots":[{"id":1},{"id":1}]}`, 0, false},
		{`{"schemaVersion":1,"source":"bambu-orca","slots":[{"id":2}]}`, 0, false},
		{`{"schemaVersion":1,"source":"bambu-orca","slots":[}`, 0, false},
	}
	for _, tc := range cases {
		slots, err := countPreservedSourceSlots([]byte(tc.config))
		if (err == nil) != tc.valid || (tc.valid && slots != tc.slots) {
			t.Fatalf("countPreservedSourceSlots(%q) = %d,%v want slots=%d valid=%v", tc.config, slots, err, tc.slots, tc.valid)
		}
	}
	if err := requirePreservedSlots(2, 1); err == nil || conversionErrorCode(err) != "PRUSA_SLOTS_NOT_PRESERVED" {
		t.Fatalf("losing a slot must be refused, got %v", err)
	}
	if err := requirePreservedSlots(2, 2); err != nil {
		t.Fatalf("equal slot count must pass: %v", err)
	}
	if err := requirePreservedSlots(2, 3); err != nil {
		t.Fatalf("extra output slots must pass: %v", err)
	}
}

func TestPrusaTargetValidationAllowsOnlyOneFixedTarget(t *testing.T) {
	if err := validateTargets([]string{prusaTargetID}); err != nil {
		t.Fatalf("fixed prusa target must be accepted: %v", err)
	}
	if err := validateTargets([]string{"u1", prusaTargetID, "c5"}); err != nil {
		t.Fatalf("prusa target must combine with built-in targets: %v", err)
	}
	if err := validateTargets([]string{prusaTargetID, prusaTargetID}); err == nil {
		t.Fatal("repeated prusa target must be rejected")
	}
	for _, bad := range []string{"prusa", "prusa-project-2", "PRUSA-PROJECT", "../prusa-project"} {
		if err := validateTargets([]string{bad}); err == nil {
			t.Fatalf("target %q must be rejected", bad)
		}
	}
}

func TestPrusaOutputsAreNotRequeuedFromDroppedFolders(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"PRUSA-first.3mf", "source.3mf"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	files, err := collectInputs([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Name != "source.3mf" {
		t.Fatalf("generated PRUSA output must not be requeued, got %+v", files)
	}
}
