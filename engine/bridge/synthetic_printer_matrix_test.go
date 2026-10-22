package bridge

import (
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

// TestSyntheticOrcaProfileMatrix exercises ACTUAL bl2std 3MF conversion,
// not a claim that any real-world printer has been validated.
// All machine, process, and PLA entries are synthetic, never user presets.
func TestSyntheticOrcaProfileMatrix(t *testing.T) {
    // Simulate different physical tool counts, source color counts, and nozzle sizes.
    // Input fixtures are Bambu-style 3MF, each with stale sliced G-code to remove.
    counts := []int{1, 2, 4, 8}
    colors := []int{1, 3, 5, 6}
    nozzles := []string{"0.2", "0.4", "0.6", "0.8"}
    for _, toolCount := range counts {
        for _, colorCount := range colors {
            for _, nozzle := range nozzles {
                t.Run(fmt.Sprintf("physical-%d/source-%d/nozzle-%s", toolCount, colorCount, nozzle), func(t *testing.T) {
                    dir := t.TempDir()
                    source := filepath.Join(dir, "source.3mf")
                    destination := filepath.Join(dir, "target.3mf")
                    original := fixture(t, source, colorCount)
                    originalSHA := sha256.Sum256(original)
                    bundle := syntheticOrcaBundle(toolCount)
                    bundle.Nozzle = nozzle
                    model := fmt.Sprintf("Synthetic Printer %dT %smm", toolCount, nozzle)
                    bundle.Profile.DisplayName = model
                    bundle.Profile.ModelID = model
                    bundle.Profile.ProjectSettings["printer_model"] = model
                    physicalNozzles := make([]any, toolCount)
                    for i := range physicalNozzles {
                        physicalNozzles[i] = nozzle
                    }
                    bundle.Profile.ProjectSettings["nozzle_diameter"] = physicalNozzles
                    result, err := ConvertWithOrcaBundle(source, destination, writeOrcaBundle(t, bundle))
                    if err != nil {
                        t.Fatalf("synthetic profile conversion failed: %s: %v", ErrorCode(err), err)
                    }
                    for key, expected := range map[string]int{
                        "source_filament_slots": colorCount, "virtual_filament_slots": colorCount,
                        "physical_toolheads": toolCount,
                    } {
                        got, ok := result[key].(int)
                        if !ok || got != expected {
                            t.Fatalf("%s: got %T(%v), want %d", key, result[key], result[key], expected)
                        }
                    }
                    if needsMapping, ok := result["user_slot_mapping_required"].(bool); !ok || needsMapping != (colorCount > toolCount) {
                        t.Fatalf("virtual-to-physical tool mapping flag incorrect: %v", result["user_slot_mapping_required"])
                    }
                    entries := archiveContents(t, destination)
                    if strings.Contains(entries["[Content_Types].xml"], "plate_1.gcode") ||
                        strings.Contains(entries["_rels/.rels"], "plate_1.gcode") ||
                        entries["Metadata/plate_1.gcode"] != "" || entries["Metadata/plate_1.gcode.md5"] != "" {
                        t.Fatal("stale sliced G-code or references survived")
                    }
                    if strings.Contains(entries["Metadata/project_settings.config"], "UNSAFE_BAMBU_SCRIPT") {
                        t.Fatal("source firmware script leaked")
                    }
                    project := make(map[string]any)
                    if err := json.Unmarshal([]byte(entries["Metadata/project_settings.config"]), &project); err != nil {
                        t.Fatal(err)
                    }
                    if project["printer_model"] != model {
                        t.Fatalf("printer identity incorrect: %v", project["printer_model"])
                    }
                    data, err := os.ReadFile(source)
                    if err != nil || sha256.Sum256(data) != originalSHA {
                        t.Fatal("source changed or became unreadable")
                    }
                    // A second convert must refuse rather than overwrite a prior output.
                    if _, err := ConvertWithOrcaBundle(source, destination, writeOrcaBundle(t, bundle)); err == nil || ErrorCode(err) != "OUTPUT_EXISTS" {
                        t.Fatalf("unsafe overwrite behavior, code %q error %v", ErrorCode(err), err)
                    }
                })
            }
        }
    }
}
