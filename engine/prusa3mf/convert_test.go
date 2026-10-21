package prusa3mf

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const fixturePath = "../../tests/fixtures/bambu-five-colors.3mf"

// copyFixture gives each test a private copy of the repository fixture so no
// test can write to, or depend on mutations of, the shared bytes.
func copyFixture(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "bambu-five-colors.3mf")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// writePackage writes files into a new ZIP in a fresh temporary directory.
func writePackage(t *testing.T, files map[string][]byte) string {
	t.Helper()
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range names {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(files[name]); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "source.3mf")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// readOutput returns every entry of a generated package by name.
func readOutput(t *testing.T, path string) map[string][]byte {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	out := make(map[string][]byte, len(zr.File))
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		out[f.Name] = data
	}
	return out
}

const tetraVertices = `<vertices><vertex x="0" y="0" z="0"/><vertex x="10" y="0" z="0"/>` +
	`<vertex x="0" y="10" z="0"/><vertex x="0" y="0" z="10"/></vertices>`

const defaultBuild = `<item objectid="1"/>`

// tetraObject is a four-vertex, four-triangle object. paints[i] is the paint
// code for triangle i; an empty string leaves that triangle unpainted.
func tetraObject(id int, paints ...string) string {
	tris := [4][3]int{{0, 1, 2}, {0, 1, 3}, {0, 2, 3}, {1, 2, 3}}
	var b strings.Builder
	fmt.Fprintf(&b, `<object id="%d" type="model"><mesh>%s<triangles>`, id, tetraVertices)
	for i, t := range tris {
		paint := ""
		if i < len(paints) && paints[i] != "" {
			paint = fmt.Sprintf(` paint_color="%s"`, paints[i])
		}
		fmt.Fprintf(&b, `<triangle v1="%d" v2="%d" v3="%d"%s/>`, t[0], t[1], t[2], paint)
	}
	b.WriteString(`</triangles></mesh></object>`)
	return b.String()
}

func tetraModel(build string, paints ...string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<model unit="millimeter" xmlns="` + coreNS + `"><resources>` +
		tetraObject(1, paints...) +
		`</resources><build>` + build + `</build></model>`
}

func filament(id int, typ, color string) string {
	return fmt.Sprintf(`<filament id="%d" type="%s" color="%s" used_m="1" used_g="3"/>`, id, typ, color)
}

func sliceInfo(filaments ...string) string {
	return `<config><plate><metadata key="printer_model_id" value="C13"/>` +
		strings.Join(filaments, "") + `</plate></config>`
}

// basePackage is a valid two-filament package with one unpainted tetrahedron.
// Tests change one part at a time to exercise a single rule.
func basePackage() map[string][]byte {
	return map[string][]byte{
		contentTypesPart:  []byte(contentTypesXML),
		relsPart:          []byte(relsXML),
		modelPart:         []byte(tetraModel(defaultBuild)),
		modelSettingsPart: []byte(`<config><object id="1"><metadata key="extruder" value="1"/></object></config>`),
		sliceInfoPart: []byte(sliceInfo(
			filament(1, "PLA", "#FF0000"),
			filament(2, "PLA", "#00FF00"),
		)),
	}
}

func TestConvertFixture(t *testing.T) {
	src := copyFixture(t)
	before, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "prusa.3mf")

	report, err := Convert(src, dst)
	if err != nil {
		t.Fatalf("Convert(fixture) failed with %s: %v", ErrorCode(err), err)
	}
	if report.Objects != 1 || report.BuildItems != 1 || report.Vertices != 4 || report.Triangles != 4 {
		t.Fatalf("geometry counts = objects %d items %d vertices %d triangles %d", report.Objects, report.BuildItems, report.Vertices, report.Triangles)
	}
	if report.PaintedTriangles != 4 || fmt.Sprint(report.PaintChannels) != "[1 5]" {
		t.Fatalf("paint = %d triangles on channels %v, want 4 on [1 5]", report.PaintedTriangles, report.PaintChannels)
	}
	if len(report.Channels) != 5 || report.DuplicateMaterialSlots != 0 {
		t.Fatalf("channels = %d (duplicates %d), want 5 (0)", len(report.Channels), report.DuplicateMaterialSlots)
	}
	wantDropped := "[Metadata/plate_1.gcode Metadata/plate_1.gcode.md5 Metadata/plate_1.json]"
	if fmt.Sprint(report.DroppedEntries) != wantDropped {
		t.Fatalf("dropped = %v, want %s", report.DroppedEntries, wantDropped)
	}

	out := readOutput(t, dst)
	if len(out) != 5 {
		t.Fatalf("output has %d entries, want 5", len(out))
	}
	doc, err := parseModel(out[modelPart], true)
	if err != nil {
		t.Fatal(err)
	}
	gotCodes := []string{}
	for _, tri := range doc.objects[0].triangles {
		gotCodes = append(gotCodes, doc.paints[tri.paint].code)
	}
	if fmt.Sprint(gotCodes) != "[2C 4 4 4]" {
		t.Fatalf("triangle paint codes = %v, want [2C 4 4 4]", gotCodes)
	}
	model := string(out[modelPart])
	if !strings.Contains(model, `slic3rpe:mmu_segmentation="2C"`) || !strings.Contains(model, `slic3rpe:mmu_segmentation="4"`) {
		t.Fatalf("output model lacks the expected paint codes")
	}
	modelConfig := string(out[prusaModelConfigPart])
	if !strings.Contains(modelConfig, `<metadata type="object" key="extruder" value="5"/>`) {
		t.Fatalf("object extruder 5 not written:\n%s", modelConfig)
	}
	filamentCfg := string(out[prusaConfigPart])
	if !strings.Contains(filamentCfg, "; slicebridge_source_filament_colour = #FF0000;#00FF00;#0000FF;#FFFF00;#FFFFFF\n") {
		t.Fatalf("filament colours not preserved in order:\n%s", filamentCfg)
	}
	for name, body := range out {
		if strings.Contains(string(body), "UNSAFE_BAMBU_SCRIPT") || strings.Contains(string(body), "M104") {
			t.Fatalf("%s carries source firmware or G-code", name)
		}
	}

	after, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("source package was modified")
	}
}

func TestConvertIsDeterministic(t *testing.T) {
	p, err := readSource(copyFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	first, err := encodeOutput(p)
	if err != nil {
		t.Fatal(err)
	}
	second, err := encodeOutput(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("two encodings of the same project differ")
	}
}

func TestConvertPreservesSparseAndDuplicateChannels(t *testing.T) {
	files := basePackage()
	files[modelPart] = []byte(tetraModel(defaultBuild, "4C", "4"))
	files[modelSettingsPart] = []byte(`<config><object id="1"><metadata key="extruder" value="7"/></object></config>`)
	files[sliceInfoPart] = []byte(sliceInfo(
		filament(1, "PLA", "#FF0000"),
		filament(3, "PETG", "#00FF00"),
		filament(7, "PLA", "#FF0000"),
	))
	dst := filepath.Join(t.TempDir(), "sparse.3mf")

	report, err := Convert(writePackage(t, files), dst)
	if err != nil {
		t.Fatalf("Convert failed with %s: %v", ErrorCode(err), err)
	}
	if len(report.Channels) != 7 {
		t.Fatalf("channels = %d, want 7 (numbering preserved to the highest id)", len(report.Channels))
	}
	for _, c := range report.Channels {
		wantPlaceholder := c.ID == 2 || c.ID == 4 || c.ID == 5 || c.ID == 6
		if c.Placeholder != wantPlaceholder {
			t.Fatalf("channel %d placeholder = %v, want %v", c.ID, c.Placeholder, wantPlaceholder)
		}
	}
	if report.DuplicateMaterialSlots != 1 {
		t.Fatalf("duplicate material slots = %d, want 1 (channel 7 repeats channel 1)", report.DuplicateMaterialSlots)
	}
	if fmt.Sprint(report.PaintChannels) != "[1 7]" {
		t.Fatalf("paint channels = %v, want [1 7]", report.PaintChannels)
	}

	out := readOutput(t, dst)
	filamentCfg := string(out[prusaConfigPart])
	for _, want := range []string{
		"; slicebridge_source_filament_colour = #FF0000;;#00FF00;;;;#FF0000\n",
		"; slicebridge_source_filament_type = PLA;;PETG;;;;PLA\n",
		"; slicebridge_placeholder_channels = 2;4;5;6\n",
	} {
		if !strings.Contains(filamentCfg, want) {
			t.Fatalf("filament table missing %q:\n%s", want, filamentCfg)
		}
	}
	if !strings.Contains(string(out[prusaModelConfigPart]), `<metadata type="object" key="extruder" value="7"/>`) {
		t.Fatal("object extruder 7 not written")
	}
}

// TestConvertPreservesNestedPaint checks that split-tree paint codes reach the
// output unchanged, including the state 0 leaf and the state 18 extension.
func TestConvertPreservesNestedPaint(t *testing.T) {
	files := basePackage()
	nested := []string{"841", "0CFC2C906", "84C04201CF", "4"}
	files[modelPart] = []byte(tetraModel(defaultBuild, nested...))
	var filaments []string
	for id := 1; id <= 18; id++ {
		filaments = append(filaments, filament(id, "PLA", fmt.Sprintf("#%06X", id)))
	}
	files[sliceInfoPart] = []byte(sliceInfo(filaments...))
	dst := filepath.Join(t.TempDir(), "nested.3mf")

	report, err := Convert(writePackage(t, files), dst)
	if err != nil {
		t.Fatalf("Convert failed with %s: %v", ErrorCode(err), err)
	}
	if fmt.Sprint(report.PaintChannels) != "[1 2 3 4 5 7 18]" || report.PaintedTriangles != 4 {
		t.Fatalf("paint = %d triangles on channels %v, want 4 on [1 2 3 4 5 7 18]", report.PaintedTriangles, report.PaintChannels)
	}
	model := string(readOutput(t, dst)[modelPart])
	for _, code := range nested {
		if want := fmt.Sprintf(`slic3rpe:mmu_segmentation="%s"`, code); !strings.Contains(model, want) {
			t.Fatalf("paint %q not written unchanged:\n%s", code, model)
		}
	}
	if !strings.Contains(model, `name="slic3rpe:MmPaintingVersion">1<`) {
		t.Fatalf("output lacks the Prusa paint version metadata:\n%s", model)
	}
}

func TestConvertPreservesBuildTransform(t *testing.T) {
	files := basePackage()
	files[modelPart] = []byte(tetraModel(`<item objectid="1" transform="1 0 0 0 1 0 0 0 1 10.5 20 -30"/>`))
	dst := filepath.Join(t.TempDir(), "transform.3mf")

	if _, err := Convert(writePackage(t, files), dst); err != nil {
		t.Fatalf("Convert failed with %s: %v", ErrorCode(err), err)
	}
	model := string(readOutput(t, dst)[modelPart])
	if !strings.Contains(model, `transform="1 0 0 0 1 0 0 0 1 10.5 20 -30"`) {
		t.Fatalf("build transform not preserved:\n%s", model)
	}
}

func TestConvertRefusesExistingOutput(t *testing.T) {
	src := copyFixture(t)
	dst := filepath.Join(t.TempDir(), "existing.3mf")
	if err := os.WriteFile(dst, []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Convert(src, dst)
	if ErrorCode(err) != "OUTPUT_EXISTS" {
		t.Fatalf("code = %s (%v), want OUTPUT_EXISTS", ErrorCode(err), err)
	}
	kept, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(kept) != "keep me" {
		t.Fatal("existing output was overwritten")
	}
}

func TestConvertRefusesSourceAsOutput(t *testing.T) {
	src := copyFixture(t)
	before, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Convert(src, src)
	if ErrorCode(err) != "SOURCE_EQUALS_OUTPUT" {
		t.Fatalf("code = %s (%v), want SOURCE_EQUALS_OUTPUT", ErrorCode(err), err)
	}
	after, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("source was modified")
	}
}

func TestConvertRefusesNonThreeMFNames(t *testing.T) {
	src := copyFixture(t)
	dir := t.TempDir()
	_, err := Convert(src, filepath.Join(dir, "out.zip"))
	if ErrorCode(err) != "EXTENSION_INVALID" {
		t.Fatalf("code = %s (%v), want EXTENSION_INVALID", ErrorCode(err), err)
	}
}
