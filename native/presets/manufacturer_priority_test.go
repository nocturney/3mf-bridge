package presets

import (
    "encoding/json"
    "os"
    "path/filepath"
    "runtime"
    "testing"
)

// A present manufacturer directory isn't proof its printer, process and PLA
// chain can be loaded. Auto must try the next trusted source, while a manual
// QIDIStudio selection must fail rather than silently switch slicers.
func TestWindowsAutoChoosesCompleteManufacturerProfileThenFallback(t *testing.T) {
    if runtime.GOOS != "windows" {
        t.Skip("tests Windows trusted install base selection using temporary ProgramFiles directories")
    }
    pf := t.TempDir()
    local := t.TempDir()
    t.Setenv("ProgramFiles", pf)
    t.Setenv("ProgramFiles(x86)", "")
    t.Setenv("LOCALAPPDATA", local)
    t.Setenv("BRIDGE_SLICER_PROFILE_QIDI", "")

    vendor := filepath.Join(local, "Programs", "QIDIStudio", "resources", "profiles")
    generic := filepath.Join(pf, "OrcaSlicer", "resources", "profiles")
    _, machine, process, filaments, err := Variant("qidi", "0.4")
    if err != nil { t.Fatal(err) }

    write := func(root, category, name string, data map[string]any) {
        t.Helper()
        filename := filepath.Join(root, Specs["qidi"].Vendor, category, name+".json")
        if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil { t.Fatal(err) }
        payload, err := json.Marshal(data)
        if err != nil { t.Fatal(err) }
        if err := os.WriteFile(filename, payload, 0600); err != nil { t.Fatal(err) }
    }
    complete := func(root string) {
        t.Helper()
        write(root, "machine", machine, map[string]any{
            "printer_model": Specs["qidi"].Model,
            "nozzle_diameter": []string{"0.4"},
            "machine_start_gcode": "G28",
        })
        write(root, "process", process, map[string]any{"layer_height": "0.2"})
        write(root, "filament", filaments["PLA"], map[string]any{
            "filament_type": []string{"PLA"},
            "filament_colour": []string{"#FFFFFF"},
        })
    }

    complete(generic)
    // This is deliberately only a vendor folder; QIDIStudio cannot convert.
    if err := os.MkdirAll(filepath.Join(vendor, Specs["qidi"].Vendor), 0700); err != nil { t.Fatal(err) }
    selected, err := LoadConfigured("qidi", "0.4", "auto", nil)
    if err != nil || selected.Slicer != "OrcaSlicer" {
        t.Fatalf("incomplete manufacturer profile must fall back to valid Orca: bundle=%+v err=%v", selected, err)
    }
    if _, err := LoadConfigured("qidi", "0.4", "qidi-studio", nil); err == nil {
        t.Fatal("explicit QIDIStudio with incomplete profiles must fail closed")
    }

    complete(vendor)
    selected, err = LoadConfigured("qidi", "0.4", "auto", nil)
    if err != nil || selected.Slicer != "QIDIStudio" || selected.Root != vendor {
        t.Fatalf("complete manufacturer profile must outrank Orca: bundle=%+v err=%v", selected, err)
    }

    if err := os.Remove(filepath.Join(vendor, Specs["qidi"].Vendor, "process", process+".json")); err != nil { t.Fatal(err) }
    selected, err = LoadConfigured("qidi", "0.4", "auto", nil)
    if err != nil || selected.Slicer != "OrcaSlicer" {
        t.Fatalf("broken manufacturer process must fall back to valid Orca: bundle=%+v err=%v", selected, err)
    }
}

