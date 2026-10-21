package bridge

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BryanSmee/bl_to_std/pkg/converter"
)

func TestSourceSlotPlanExactDuplicateBeyondFour(t *testing.T) {
	source := []converter.Filament{
		{ID: 1, Color: "#FF0000", Type: "PLA"},
		{ID: 2, Color: "#00FF00", Type: "PLA"},
		{ID: 3, Color: "#0000FF", Type: "PLA"},
		{ID: 4, Color: "#FFFF00", Type: "PLA"},
		{ID: 5, Color: "#FF0000", Type: "PLA"},
	}
	slots, mapping, unique, folded, err := sourceSlotPlan(source, 4)
	if err != nil || len(slots) != 4 || len(unique) != 4 || folded != 1 {
		t.Fatalf("expected lossless four-slot map, got %v %v %d %v", slots, mapping, folded, err)
	}
	for id, want := range map[int]int{1: 1, 2: 2, 3: 3, 4: 4, 5: 1} {
		if mapping[id] != want {
			t.Fatalf("filament %d mapped to %d instead of %d", id, mapping[id], want)
		}
	}
	// Before exceeding capacity, existing distinct spool identities remain
	// untouched even when their colors and materials happen to match.
	short := []converter.Filament{source[0], source[4]}
	_, mapping, _, folded, err = sourceSlotPlan(short, 4)
	if err != nil || folded != 0 || mapping[5] != 2 {
		t.Fatalf("legacy under-capacity mapping changed: %v %d %v", mapping, folded, err)
	}
}

func TestSourceSlotPlanNeverCombinesDifferentColorsOrMaterials(t *testing.T) {
	base := []converter.Filament{
		{ID: 1, Color: "#FF0000", Type: "PLA"},
		{ID: 2, Color: "#00FF00", Type: "PLA"},
		{ID: 3, Color: "#0000FF", Type: "PLA"},
		{ID: 4, Color: "#FFFF00", Type: "PLA"},
	}
	for _, fifth := range []converter.Filament{
		{ID: 5, Color: "#FFFFFF", Type: "PLA"},
		{ID: 5, Color: "#FF0000", Type: "PETG"},
	} {
		_, _, _, _, err := sourceSlotPlan(append(append([]converter.Filament{}, base...), fifth), 4)
		if err == nil || !strings.Contains(err.Error(), "refusing") {
			t.Fatalf("a distinct color/material was silently combined: %+v %v", fifth, err)
		}
	}
}

func overwriteFifthColorWithFirst(t *testing.T, filename string) {
	t.Helper()
	archive, err := zip.OpenReader(filename)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	changed := 0
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		switch file.Name {
		case "Metadata/slice_info.config":
			if !bytes.Contains(data, []byte("color='#FFFFFF'")) {
				t.Fatal("fifth filament not present in fixture")
			}
			data = bytes.Replace(data, []byte("color='#FFFFFF'"), []byte("color='#FF0000'"), 1)
			changed++
		case "Metadata/project_settings.config":
			if !bytes.Contains(data, []byte("\"#FFFFFF\"")) {
				t.Fatal("fifth profile color missing")
			}
			data = bytes.Replace(data, []byte("\"#FFFFFF\""), []byte("\"#FF0000\""), 1)
			changed++
		}
		dst, err := writer.Create(file.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = dst.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	archive.Close()
	if changed != 2 {
		t.Fatalf("expected two source updates, got %d", changed)
	}
	if err := os.WriteFile(filename, out.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestFiveSlotsWithIdenticalMaterialConvertWithoutChangingSource(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "five-duplicates.3mf")
	output := filepath.Join(dir, "U1-five-duplicates.3mf")
	fixture(t, input, 5)
	overwriteFifthColorWithFirst(t, input)
	before, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Convert(input, output, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if result["source_filament_slots"] != 5 || result["identical_source_slots_coalesced"] != 1 {
		t.Fatalf("unexpected coalescing report: %+v", result)
	}
	mapping, ok := result["source_to_target_slot_map"].(map[int]int)
	if !ok || mapping[1] != 1 || mapping[5] != 1 {
		t.Fatalf("identity-preserving slot map missing: %+v", result)
	}
	after, err := os.ReadFile(input)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("source file mutated")
	}
	files := archiveContents(t, output)
	if strings.Contains(files["Metadata/project_settings.config"], "UNSAFE_BAMBU_SCRIPT") {
		t.Fatal("unsafe source firmware script leaked")
	}
	if _, ok := files["Metadata/plate_1.gcode"]; ok {
		t.Fatal("stale sliced gcode leaked")
	}
}
