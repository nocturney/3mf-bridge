package prusa3mf

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// verifyOutput re-reads the generated package and compares every converted fact
// with the in-memory project: entry set, vertices, triangles, paint channels,
// build transforms, object and volume extruders, and the filament table.
// Publication happens only after this passes.
func verifyOutput(data []byte, p *project) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	want := []string{contentTypesPart, relsPart, modelPart, prusaConfigPart, prusaModelConfigPart}
	if len(zr.File) != len(want) {
		return fmt.Errorf("output has %d entries, want %d", len(zr.File), len(want))
	}
	files := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		files[f.Name] = f
	}
	bodies := make(map[string][]byte, len(want))
	for _, name := range want {
		f := files[name]
		if f == nil {
			return fmt.Errorf("output is missing %s", name)
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		body, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}
		bodies[name] = body
	}
	doc, err := parseModel(bodies[modelPart], true)
	if err != nil {
		return err
	}
	if err := compareModel(doc, p); err != nil {
		return err
	}
	if err := compareModelConfig(bodies[prusaModelConfigPart], p); err != nil {
		return err
	}
	return compareFilaments(bodies[prusaConfigPart], p)
}

func compareModel(doc *modelDoc, p *project) error {
	if doc.unit != p.unit {
		return fmt.Errorf("unit %q, want %q", doc.unit, p.unit)
	}
	if len(doc.objects) != len(p.objects) {
		return fmt.Errorf("output has %d objects, want %d", len(doc.objects), len(p.objects))
	}
	for i, want := range p.objects {
		got := doc.objects[i]
		if got.id != want.id {
			return fmt.Errorf("object %d out of order", want.id)
		}
		if len(got.vertices) != len(want.vertices) {
			return fmt.Errorf("object %d has %d vertices, want %d", want.id, len(got.vertices), len(want.vertices))
		}
		for j := range want.vertices {
			if got.vertices[j] != want.vertices[j] {
				return fmt.Errorf("object %d vertex %d changed", want.id, j)
			}
		}
		if len(got.triangles) != len(want.triangles) {
			return fmt.Errorf("object %d has %d triangles, want %d", want.id, len(got.triangles), len(want.triangles))
		}
		for j := range want.triangles {
			g, w := got.triangles[j], want.triangles[j]
			if g.v != w.v {
				return fmt.Errorf("object %d triangle %d indices changed", want.id, j)
			}
			if gc, wc := doc.paints[g.paint].code, p.paints[w.paint].code; gc != wc {
				return fmt.Errorf("object %d triangle %d paint %q, want %q", want.id, j, gc, wc)
			}
		}
	}
	if len(doc.items) != len(p.items) {
		return fmt.Errorf("output has %d build items, want %d", len(doc.items), len(p.items))
	}
	for i, want := range p.items {
		got := doc.items[i]
		if got.objectID != want.objectID || got.printable != want.printable {
			return fmt.Errorf("build item %d changed", i)
		}
		if (got.transform == nil) != (want.transform == nil) {
			return fmt.Errorf("build item %d transform presence changed", i)
		}
		if got.transform != nil && *got.transform != *want.transform {
			return fmt.Errorf("build item %d transform changed", i)
		}
	}
	return nil
}

func compareModelConfig(data []byte, p *project) error {
	root, err := parseSmallXML(data)
	if err != nil {
		return err
	}
	objs := root.childrenNamed("object")
	if len(objs) != len(p.objects) {
		return fmt.Errorf("model config has %d objects, want %d", len(objs), len(p.objects))
	}
	instances := make(map[int]int)
	for _, it := range p.items {
		instances[it.objectID]++
	}
	for i, want := range p.objects {
		got := objs[i]
		if got.attrs["id"] != strconv.Itoa(want.id) {
			return fmt.Errorf("model config object %d out of order", want.id)
		}
		if got.attrs["instances_count"] != strconv.Itoa(instances[want.id]) {
			return fmt.Errorf("model config object %d instance count differs", want.id)
		}
		if name, _ := got.metadata("name"); name != want.name {
			return fmt.Errorf("model config object %d name differs", want.id)
		}
		if ext, _ := got.metadata("extruder"); ext != intText(want.extruder) {
			return fmt.Errorf("object %d extruder %q, want %q", want.id, ext, intText(want.extruder))
		}
		vols := got.childrenNamed("volume")
		if len(vols) != 1 {
			return fmt.Errorf("model config object %d has %d volumes, want 1", want.id, len(vols))
		}
		vol := vols[0]
		if vol.attrs["firstid"] != "0" || vol.attrs["lastid"] != strconv.Itoa(len(want.triangles)-1) {
			return fmt.Errorf("model config object %d volume range does not cover its triangles", want.id)
		}
		if ext, _ := vol.metadata("extruder"); ext != intText(want.volumeExtruder) {
			return fmt.Errorf("volume of object %d extruder %q, want %q", want.id, ext, intText(want.volumeExtruder))
		}
	}
	return nil
}

func compareFilaments(data []byte, p *project) error {
	var colours, types string
	for _, line := range strings.Split(string(data), "\n") {
		switch {
		case strings.HasPrefix(line, "; slicebridge_source_filament_colour = "):
			colours = strings.TrimPrefix(line, "; slicebridge_source_filament_colour = ")
		case strings.HasPrefix(line, "; slicebridge_source_filament_type = "):
			types = strings.TrimPrefix(line, "; slicebridge_source_filament_type = ")
		}
	}
	var wantColours, wantTypes []string
	for _, c := range p.channels {
		wantColours = append(wantColours, c.color)
		wantTypes = append(wantTypes, c.typ)
	}
	if colours != strings.Join(wantColours, ";") || types != strings.Join(wantTypes, ";") {
		return fmt.Errorf("source filament table in %s does not match the source channels", prusaConfigPart)
	}
	return nil
}

// intText renders an optional channel where 0 means "not written".
func intText(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}
