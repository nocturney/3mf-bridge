package prusa3mf

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

const (
	xmlHeader      = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
	identityMatrix = "1 0 0 0 0 1 0 0 0 0 1 0 0 0 0 1"

	contentTypesXML = xmlHeader +
		`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Override PartName="/3D/3dmodel.model" ContentType="application/vnd.ms-package.3dmanufacturing-3dmodel+xml"/>` +
		`<Override PartName="/Metadata/Slic3r_PE.config" ContentType="text/plain"/>` +
		`<Override PartName="/Metadata/Slic3r_PE_model.config" ContentType="text/xml"/>` +
		`</Types>` + "\n"

	relsXML = xmlHeader +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rel-1" Target="/3D/3dmodel.model" Type="` + modelRelType + `"/>` +
		`</Relationships>` + "\n"
)

// encodeOutput builds the PrusaSlicer package in memory. Parts are written in
// a fixed order and without timestamps, so identical input gives identical bytes.
func encodeOutput(p *project) ([]byte, error) {
	model, err := modelXML(p)
	if err != nil {
		return nil, err
	}
	parts := []struct {
		name string
		body string
	}{
		{contentTypesPart, contentTypesXML},
		{relsPart, relsXML},
		{modelPart, model},
		{prusaConfigPart, prusaConfig(p)},
		{prusaModelConfigPart, modelConfigXML(p)},
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, part := range parts {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: part.name, Method: zip.Deflate})
		if err != nil {
			return nil, err
		}
		if _, err := w.Write([]byte(part.body)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// modelXML writes geometry, paint and build items. Vertices stay in object
// space and each build item carries its transform, so geometry is unchanged.
func modelXML(p *project) (string, error) {
	var b strings.Builder
	b.WriteString(xmlHeader)
	fmt.Fprintf(&b, "<model unit=\"%s\" xml:lang=\"en-US\" xmlns=\"%s\" xmlns:slic3rpe=\"%s\">\n", p.unit, coreNS, slic3rpeNS)
	b.WriteString(" <metadata name=\"Application\">SliceBridge prusa3mf</metadata>\n")
	b.WriteString(" <resources>\n")
	for _, obj := range p.objects {
		fmt.Fprintf(&b, "  <object id=\"%d\" type=\"model\">\n   <mesh>\n    <vertices>\n", obj.id)
		for _, v := range obj.vertices {
			fmt.Fprintf(&b, "     <vertex x=\"%s\" y=\"%s\" z=\"%s\"/>\n", formatFloat(v[0]), formatFloat(v[1]), formatFloat(v[2]))
		}
		b.WriteString("    </vertices>\n    <triangles>\n")
		for _, t := range obj.triangles {
			fmt.Fprintf(&b, "     <triangle v1=\"%d\" v2=\"%d\" v3=\"%d\"", t.v[0], t.v[1], t.v[2])
			if t.channel != 0 {
				code, err := encodePaint(int(t.channel))
				if err != nil {
					return "", err
				}
				fmt.Fprintf(&b, " slic3rpe:mmu_segmentation=\"%s\"", code)
			}
			b.WriteString("/>\n")
		}
		b.WriteString("    </triangles>\n   </mesh>\n  </object>\n")
	}
	b.WriteString(" </resources>\n <build>\n")
	for _, it := range p.items {
		fmt.Fprintf(&b, "  <item objectid=\"%d\"", it.objectID)
		if it.transform != nil {
			fmt.Fprintf(&b, " transform=\"%s\"", formatTransform(it.transform))
		}
		if it.printable != "" {
			fmt.Fprintf(&b, " printable=\"%s\"", it.printable)
		}
		b.WriteString("/>\n")
	}
	b.WriteString(" </build>\n</model>\n")
	return b.String(), nil
}

// modelConfigXML carries object and volume extruder assignments, names and the
// triangle range of each volume, in PrusaSlicer's Slic3r_PE_model.config layout.
func modelConfigXML(p *project) string {
	instances := make(map[int]int)
	for _, it := range p.items {
		instances[it.objectID]++
	}
	var b strings.Builder
	b.WriteString(xmlHeader)
	b.WriteString("<config>\n")
	for _, obj := range p.objects {
		fmt.Fprintf(&b, " <object id=\"%d\" instances_count=\"%d\">\n", obj.id, instances[obj.id])
		writeMeta(&b, "object", "name", obj.name)
		if obj.extruder != 0 {
			writeMeta(&b, "object", "extruder", strconv.Itoa(obj.extruder))
		}
		fmt.Fprintf(&b, "  <volume firstid=\"0\" lastid=\"%d\">\n", len(obj.triangles)-1)
		writeMeta(&b, "volume", "name", obj.name)
		writeMeta(&b, "volume", "volume_type", "ModelPart")
		writeMeta(&b, "volume", "matrix", identityMatrix)
		if obj.volumeExtruder != 0 {
			writeMeta(&b, "volume", "extruder", strconv.Itoa(obj.volumeExtruder))
		}
		b.WriteString("  </volume>\n </object>\n")
	}
	b.WriteString(" <plate>\n")
	b.WriteString("  <metadata key=\"plater_id\" value=\"1\"/>\n")
	b.WriteString("  <metadata key=\"locked\" value=\"0\"/>\n")
	next := make(map[int]int)
	for _, it := range p.items {
		fmt.Fprintf(&b, "  <model_instance>\n   <metadata key=\"object_id\" value=\"%d\"/>\n   <metadata key=\"instance_id\" value=\"%d\"/>\n  </model_instance>\n", it.objectID, next[it.objectID])
		next[it.objectID]++
	}
	b.WriteString(" </plate>\n</config>\n")
	return b.String()
}

// prusaConfig writes the filament table in PrusaSlicer's "; key = value" form.
// Print, printer and process settings are not transferred, so no printer
// firmware or scripts are written here.
func prusaConfig(p *project) string {
	var colours, types, gaps []string
	for _, c := range p.channels {
		colours = append(colours, c.color)
		types = append(types, c.typ)
		if c.placeholder {
			gaps = append(gaps, strconv.Itoa(c.id))
		}
	}
	var b strings.Builder
	b.WriteString("; generated_by = SliceBridge prusa3mf\n")
	b.WriteString("; source_format = Bambu/Orca 3MF\n")
	if p.sourcePrinter != "" {
		fmt.Fprintf(&b, "; source_printer_model = %s\n", p.sourcePrinter)
	}
	fmt.Fprintf(&b, "; filament_colour = %s\n", strings.Join(colours, ";"))
	fmt.Fprintf(&b, "; filament_type = %s\n", strings.Join(types, ";"))
	fmt.Fprintf(&b, "; slicebridge_placeholder_channels = %s\n", strings.Join(gaps, ";"))
	fmt.Fprintf(&b, "; slicebridge_dropped_entries = %d\n", len(p.dropped))
	b.WriteString("; slicebridge_settings = not transferred; re-slice with a PrusaSlicer printer profile\n")
	return b.String()
}

func writeMeta(b *strings.Builder, typ, key, value string) {
	fmt.Fprintf(b, "  <metadata type=\"%s\" key=\"%s\" value=\"%s\"/>\n", typ, key, xmlEscape(value))
}

func xmlEscape(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}

func formatTransform(t *[12]float64) string {
	parts := make([]string, len(t))
	for i, v := range t {
		parts[i] = formatFloat(v)
	}
	return strings.Join(parts, " ")
}
