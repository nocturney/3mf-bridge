package bridge

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T, filename string, count int) []byte {
	t.Helper()
	var filaments []string
	colors := []string{"#FF0000", "#00FF00", "#0000FF", "#FFFF00", "#FFFFFF"}
	for i := 0; i < count; i++ {
		filaments = append(filaments, "<filament id='"+string(rune('1'+i))+"' type='PLA' color='"+colors[i]+"' used_m='1' used_g='3'/>")
	}
	slice := "<config><plate><metadata key='printer_model_id' value='C13'/>" + strings.Join(filaments, "") + "</plate></config>"
	model := "<model xmlns='http://schemas.microsoft.com/3dmanufacturing/core/2015/02' unit='millimeter'><resources><object id='1' type='model'><mesh><vertices><vertex x='0' y='0' z='0'/><vertex x='10' y='0' z='0'/><vertex x='0' y='10' z='0'/><vertex x='0' y='0' z='10'/></vertices><triangles><triangle v1='0' v2='1' v3='2' paint_color='4'/><triangle v1='0' v2='1' v3='3' paint_color='4'/></triangles></mesh></object></resources><build><item objectid='1'/></build></model>"
	materialTypes := make([]string, count)
	for i := range materialTypes {
		materialTypes[i] = "PLA"
	}
	cfg := map[string]any{
		"printer_model": "Bambu Lab A1", "filament_colour": colors[:count],
		"filament_type": materialTypes, "enable_support": "0",
		"layer_height": "0.16", "wall_loops": "3", "brim_width": "5",
		"machine_start_gcode": "M9999 UNSAFE_BAMBU_SCRIPT",
	}
	types := "<Types xmlns='http://schemas.openxmlformats.org/package/2006/content-types'><Override PartName='/Metadata/plate_1.gcode' ContentType='text/plain'/></Types>"
	rels := "<Relationships><Relationship Id='r1' Target='../Metadata/plate_1.gcode' Type='slice'/></Relationships>"
	proj, _ := json.Marshal(cfg)
	files := map[string][]byte{
		"[Content_Types].xml":              []byte(types),
		"_rels/.rels":                      []byte(rels),
		"3D/3dmodel.model":                 []byte(model),
		"Metadata/project_settings.config": proj,
		"Metadata/model_settings.config":   []byte("<config/>"),
		"Metadata/slice_info.config":       []byte(slice),
		"Metadata/plate_1.gcode":           []byte("M104 S300; Bambu cache"),
		"Metadata/plate_1.gcode.md5":       []byte("fake"),
		"Metadata/plate_1.json":            []byte("{}"),
	}
	var data bytes.Buffer
	zw := zip.NewWriter(&data)
	for name, contents := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(contents); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func archiveContents(t *testing.T, filename string) map[string]string {
	t.Helper()
	zr, err := zip.OpenReader(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	result := map[string]string{}
	for _, f := range zr.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		result[f.Name] = string(b)
	}
	return result
}

func TestU1ConversionSanitizesAndPreservesGeometry(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "source.3mf")
	out := filepath.Join(dir, "U1-source.3mf")
	original := fixture(t, in, 1)
	result, err := Convert(in, out, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if !result["ok"].(bool) {
		t.Fatal(result)
	}
	data := archiveContents(t, out)
	if !strings.Contains(data["Metadata/project_settings.config"], "Snapmaker U1") {
		t.Fatal("U1 identity not present")
	}
	if strings.Contains(data["Metadata/project_settings.config"], "UNSAFE_BAMBU_SCRIPT") {
		t.Fatal("source machine G-code leaked into target")
	}
	if !strings.Contains(data["Metadata/project_settings.config"], "\"wall_loops\": \"3\"") {
		t.Fatal("source wall loop count was not preserved")
	}
	if _, ok := data["Metadata/plate_1.gcode"]; ok {
		t.Fatal("stale G-code copied")
	}
	if _, ok := data["Metadata/plate_1.gcode.md5"]; ok {
		t.Fatal("stale checksum copied")
	}
	if _, ok := data["Metadata/plate_1.json"]; ok {
		t.Fatal("stale slice metadata copied")
	}
	if strings.Contains(data["[Content_Types].xml"], "plate_1.gcode") ||
		strings.Contains(data["_rels/.rels"], "plate_1.gcode") {
		t.Fatal("stale Open Packaging Convention relationships remain")
	}
	if strings.Count(data["3D/3dmodel.model"], "paint_color") != 2 {
		t.Fatal("paint disappeared")
	}
	srcNow, err := os.ReadFile(in)
	if err != nil || !bytes.Equal(srcNow, original) {
		t.Fatal("source mutated")
	}
	before, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Convert(in, out, "u1")
	if err == nil || ErrorCode(err) != "OUTPUT_EXISTS" {
		t.Fatalf("overwrite was not prevented: %v", err)
	}
	after, _ := os.ReadFile(out)
	if !bytes.Equal(before, after) {
		t.Fatal("existing output mutated")
	}
}

func TestRejectUnverifiedTargetsAndExcessColors(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.3mf")
	out := filepath.Join(dir, "result.3mf")
	fixture(t, input, 1)
	_, err := Convert(input, out, "c5")
	if err == nil || ErrorCode(err) != "TARGET_NOT_VALIDATED" {
		t.Fatalf("unverified target: %v", err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("created unsupported target output")
	}
	fixture(t, input, 5)
	_, err = Convert(input, out, "u1")
	if err == nil || ErrorCode(err) != "COLOR_MAPPING_REVIEW_REQUIRED" {
		t.Fatalf("unexpected color mapping policy: %v", err)
	}
}

func TestRejectNonProjectAndTraversal(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "invalid.3mf")
	out := filepath.Join(dir, "output.3mf")
	if err := os.WriteFile(in, []byte("not-a-zip"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := Convert(in, out, "u1")
	if err == nil || ErrorCode(err) != "SOURCE_INVALID" {
		t.Fatalf("accepted invalid archive: %v", err)
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("../escape")
	w.Write([]byte("bad"))
	zw.Close()
	os.WriteFile(in, buf.Bytes(), 0600)
	_, err = Convert(in, out, "u1")
	if err == nil || ErrorCode(err) != "SOURCE_INVALID" {
		t.Fatalf("accepted zip traversal: %v", err)
	}
}

func TestFourFilamentSlotsStayDistinct(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "four.3mf")
	output := filepath.Join(dir, "U1-four.3mf")
	fixture(t, input, 4)
	report, err := Convert(input, output, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if report["source_filament_slots"] != 4 {
		t.Fatal("source slots miscounted")
	}
	data := archiveContents(t, output)
	var cfg map[string]any
	if err := json.Unmarshal([]byte(data["Metadata/project_settings.config"]), &cfg); err != nil {
		t.Fatal(err)
	}
	colors := cfg["filament_colour"].([]any)
	if len(colors) != 4 {
		t.Fatalf("wrong output slots: %v", colors)
	}
	for i, expected := range []string{"#FF0000", "#00FF00", "#0000FF", "#FFFF00"} {
		if colors[i] != expected {
			t.Fatalf("slot %d changed: %v", i+1, colors)
		}
	}
}
