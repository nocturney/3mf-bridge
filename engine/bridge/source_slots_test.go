package bridge

import (
    "archive/zip"
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "reflect"
    "strings"
    "testing"

    "github.com/BryanSmee/bl_to_std/pkg/converter"
    "github.com/BryanSmee/bl_to_std/pkg/paint"
)

func TestSourceSlotPlanKeepsEqualColorsAndDifferentMaterialsSeparate(t *testing.T) {
    filaments := []converter.Filament{
        {ID: 1, Color: "#FF0000", Type: "PLA"},
        {ID: 2, Color: "#00FF00", Type: "PLA"},
        {ID: 3, Color: "#0000FF", Type: "PLA"},
        {ID: 4, Color: "#FFFF00", Type: "PLA"},
        {ID: 5, Color: "#FF0000", Type: "PLA"},
        {ID: 6, Color: "#FF0000", Type: "PETG"},
    }
    slots, mapping, err := sourceSlotPlan(filaments)
    if err != nil || len(slots) != 6 {
        t.Fatalf("six original channels must survive, slots=%+v error=%v", slots, err)
    }
    if mapping[1] != 1 || mapping[5] != 5 || mapping[6] != 6 {
        t.Fatalf("duplicate colors and distinct material were silently folded: %+v", mapping)
    }
    for i, f := range filaments {
        if slots[i].Color != f.Color || slots[i].Type != f.Type {
            t.Fatalf("slot %d changed color/material: %+v versus %+v", i, slots[i], f)
        }
    }
}

func TestSourceSlotPlanRejectsOnlyInvalidIdentitiesOrOversizedInputs(t *testing.T) {
    for _, source := range [][]converter.Filament{
        {{ID: 1, Color: "#123456", Type: "PLA"}, {ID: 1, Color: "#123456", Type: "PLA"}},
        {{ID: 0, Color: "#123456", Type: "PLA"}},
        {{ID: 1, Color: "#123456", Type: ""}},
        make([]converter.Filament, maxVirtualFilaments+1),
    } {
        if _, _, err := sourceSlotPlan(source); err == nil {
            t.Fatalf("invalid source identities accepted: %+v", source)
        }
    }
}

func mutateFiveSlotSource(t *testing.T, filename string, duplicateColor bool, fifthMaterial string) string {
    t.Helper()
    archive, err := zip.OpenReader(filename)
    if err != nil {
        t.Fatal(err)
    }
    defer archive.Close()
    var out bytes.Buffer
    zw := zip.NewWriter(&out)
    painted := paint.Node{State: 5}.Encode()
    changed := map[string]bool{}
    for _, file := range archive.File {
        reader, err := file.Open()
        if err != nil { t.Fatal(err) }
        data, err := io.ReadAll(reader)
        reader.Close()
        if err != nil { t.Fatal(err) }
        switch file.Name {
        case "3D/3dmodel.model":
            replacement := "paint_color='" + painted + "'"
            if !bytes.Contains(data, []byte("paint_color='4'")) { t.Fatal("test model has no source painting") }
            data = bytes.Replace(data, []byte("paint_color='4'"), []byte(replacement), -1)
            changed[file.Name] = true
        case "Metadata/model_settings.config":
            data = []byte("<config><object id='1'><metadata key='extruder' value='5'/></object></config>")
            changed[file.Name] = true
        case "Metadata/slice_info.config":
            if duplicateColor {
                data = bytes.Replace(data, []byte("id='5' type='PLA' color='#FFFFFF'"),
                    []byte("id='5' type='PLA' color='#FF0000'"), 1)
            } else if fifthMaterial != "PLA" {
                data = bytes.Replace(data, []byte("id='5' type='PLA' color='#FFFFFF'"),
                    []byte(fmt.Sprintf("id='5' type='%s' color='#FFFFFF'", fifthMaterial)), 1)
            }
            changed[file.Name] = true
        case "Metadata/project_settings.config":
            cfg := map[string]any{}
            if err := json.Unmarshal(data, &cfg); err != nil { t.Fatal(err) }
            colors, ok := cfg["filament_colour"].([]any)
            if !ok || len(colors) != 5 { t.Fatal("fixture missing five material colors") }
            types, ok := cfg["filament_type"].([]any)
            if !ok || len(types) != 5 { t.Fatal("fixture missing five material types") }
            if duplicateColor { colors[4] = "#FF0000" }
            types[4] = fifthMaterial
            data, err = json.Marshal(cfg)
            if err != nil { t.Fatal(err) }
            changed[file.Name] = true
        }
        dst, err := zw.Create(file.Name)
        if err != nil { t.Fatal(err) }
        if _, err = dst.Write(data); err != nil { t.Fatal(err) }
    }
    if err := zw.Close(); err != nil { t.Fatal(err) }
    if len(changed) != 4 { t.Fatalf("not all source metadata edited: %+v",changed) }
    if err := os.WriteFile(filename, out.Bytes(), 0600); err != nil { t.Fatal(err) }
    return painted
}

func TestConvertFiveDistinctChannelsPreservesPaintPaletteAndMaterial(t *testing.T) {
    dir := t.TempDir()
    source := filepath.Join(dir, "original-five.3mf")
    output := filepath.Join(dir, "U1-five.3mf")
    fixture(t, source, 5)
    painted := mutateFiveSlotSource(t, source, false, "PETG")
    original, err := os.ReadFile(source)
    if err != nil { t.Fatal(err) }
    got, err := Convert(source, output, "u1")
    if err != nil { t.Fatal(err) }
    if got["source_filament_slots"] != 5 || got["virtual_filament_slots"] != 5 ||
        got["physical_toolheads"] != 4 || got["user_slot_mapping_required"] != true {
        t.Fatalf("incorrect review and slot metadata: %+v", got)
    }
    mapping := got["source_to_target_slot_map"].(map[int]int)
    for id := 1; id <= 5; id++ {
        if mapping[id] != id { t.Fatalf("slot %d incorrectly remapped: %+v", id, mapping) }
    }
    converted, err := converter.Inspect(output)
    if err != nil { t.Fatal(err) }
    if len(converted.Filaments) != 5 ||
        converted.Filaments[4].Color != "#FFFFFF" ||
        converted.Filaments[4].Type != "PETG" {
        t.Fatalf("fifth material/color missing from project: %+v", converted.Filaments)
    }
    files := archiveContents(t, output)
    cfg := map[string]any{}
    if err := json.Unmarshal([]byte(files["Metadata/project_settings.config"]), &cfg); err != nil { t.Fatal(err) }
    wantColors := []any{"#FF0000","#00FF00","#0000FF","#FFFF00","#FFFFFF"}
    if !reflect.DeepEqual(cfg["filament_colour"],wantColors) { t.Fatalf("colors changed: %+v",cfg["filament_colour"]) }
    if types, ok := cfg["filament_type"].([]any); !ok || len(types)!=5 || types[4]!="PETG" {
        t.Fatalf("material types altered: %+v", cfg["filament_type"])
    }
    if !strings.Contains(files["3D/3dmodel.model"], "paint_color=\""+painted+"\"") {
        t.Fatalf("triangle painting reference for fifth filament lost: %s", files["3D/3dmodel.model"])
    }
    if !strings.Contains(files["Metadata/model_settings.config"], "value=\"5\"") {
        t.Fatalf("object fifth-filament assignment missing: %s",files["Metadata/model_settings.config"])
    }
    if strings.Contains(files["Metadata/project_settings.config"],"UNSAFE_BAMBU_SCRIPT") {
        t.Fatal("source firmware script leaked")
    }
    if _, present := files["Metadata/plate_1.gcode"]; present { t.Fatal("stale sliced gcode leaked") }
    unchanged, err := os.ReadFile(source)
    if err != nil || !bytes.Equal(unchanged, original) { t.Fatal("source mutated") }
}

func TestDuplicateIdenticalSourceFilamentsRemainSeparateInOutput(t *testing.T) {
    dir := t.TempDir()
    source := filepath.Join(dir, "five-matching.3mf")
    output := filepath.Join(dir, "U1-matching.3mf")
    fixture(t, source, 5)
    mutateFiveSlotSource(t, source, true, "PLA")
    got, err := Convert(source, output, "u1")
    if err != nil { t.Fatal(err) }
    mapping := got["source_to_target_slot_map"].(map[int]int)
    if mapping[1] != 1 || mapping[5] != 5 {
        t.Fatalf("two distinct spool identities were combined: %+v", mapping)
    }
    converted, err := converter.Inspect(output)
    if err != nil { t.Fatal(err) }
    if len(converted.Filaments) != 5 || converted.Filaments[0].Color != converted.Filaments[4].Color {
        t.Fatalf("duplicate source colors not retained as separate spools: %+v",converted.Filaments)
    }
}
