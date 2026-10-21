package formats

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, more map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	base := map[string]string{
		"[Content_Types].xml": `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>`,
		"_rels/.rels":         `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"/>`,
		"3D/3dmodel.model":    `<model xmlns="http://schemas.microsoft.com/3dmanufacturing/core/2015/02"/>`,
	}
	for k, v := range more {
		base[k] = v
	}
	for name, contents := range base {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "input.3mf")
	if err := os.WriteFile(file, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestBambuOrcaSource(t *testing.T) {
	p := fixture(t, map[string]string{"Metadata/project_settings.config": `{"filament_colour":["#000000"]}`})
	got, err := Detect(p)
	if err != nil || got.Family != BambuOrca || len(got.Evidence) != 1 {
		t.Fatalf("got %+v %v", got, err)
	}
}
func TestPrusaSource(t *testing.T) {
	p := fixture(t, map[string]string{"Metadata/Slic3r_PE_model.config": `<config/>`})
	got, err := Detect(p)
	if err != nil || got.Family != Prusa {
		t.Fatalf("got %+v %v", got, err)
	}
}
func TestPlainThreeMFIsUnknown(t *testing.T) {
	got, err := Detect(fixture(t, nil))
	if err != nil || got.Family != Unknown {
		t.Fatalf("got %+v %v", got, err)
	}
}
func TestConflictingMarkersFail(t *testing.T) {
	p := fixture(t, map[string]string{"Metadata/project_settings.config": "{}", "Metadata/Slic3r_PE.config": "abc"})
	_, err := Detect(p)
	if !errors.Is(err, ErrConflictingMarkers) {
		t.Fatalf("wanted conflict, got %v", err)
	}
}
func TestRejectUnsafeAndIncompleteArchives(t *testing.T) {
	for _, extra := range []map[string]string{
		{"../escape": "x"},
		{"3D/../../escape": "x"},
	} {
		p := fixture(t, extra)
		_, err := Detect(p)
		if err == nil {
			t.Fatalf("expected unsafe archive rejection")
		}
	}
	if _, err := Detect(fixture(t, map[string]string{"[Content_Types].xml": ""})); err != nil {
		t.Fatalf("structural metadata detection should not validate XML content: %v", err)
	}
}
func TestCorruptZipRejected(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.3mf")
	if err := os.WriteFile(p, []byte("not zip"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Detect(p); err == nil {
		t.Fatal("corrupt zip accepted")
	}
}
