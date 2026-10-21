package prusa3mf

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Report summarises a successful conversion for logs and the CLI.
type Report struct {
	Output                 string          `json:"output"`
	Objects                int             `json:"objects"`
	BuildItems             int             `json:"build_items"`
	Vertices               int             `json:"vertices"`
	Triangles              int             `json:"triangles"`
	PaintedTriangles       int             `json:"painted_triangles"`
	PaintChannels          []int           `json:"paint_channels"`
	Channels               []ChannelReport `json:"channels"`
	DuplicateMaterialSlots int             `json:"duplicate_material_slots"`
	DroppedEntries         []string        `json:"dropped_entries"`
	Warnings               []string        `json:"warnings"`
}

// ChannelReport describes one numbered filament channel in the output.
type ChannelReport struct {
	ID          int    `json:"id"`
	Color       string `json:"color"`
	Type        string `json:"type"`
	Placeholder bool   `json:"placeholder,omitempty"`
}

var fixedWarnings = []string{
	"Print, printer and process settings were not transferred; choose a PrusaSlicer printer profile and re-slice before printing.",
	"Source firmware start and end scripts were not copied.",
}

// Convert reads the Bambu/Orca project at src and publishes a PrusaSlicer 3MF
// at dst. It never overwrites dst and never modifies src. The generated package
// is re-parsed and compared with the source before it becomes visible at dst.
func Convert(src, dst string) (Report, error) {
	if !strings.EqualFold(filepath.Ext(src), ".3mf") || !strings.EqualFold(filepath.Ext(dst), ".3mf") {
		return Report{}, fail("EXTENSION_INVALID", "input and output must have .3mf extensions")
	}
	srcInfo, err := os.Stat(src)
	if err != nil {
		return Report{}, fail("SOURCE_INVALID", "%v", err)
	}
	if dstInfo, err := os.Stat(dst); err == nil {
		if os.SameFile(srcInfo, dstInfo) {
			return Report{}, fail("SOURCE_EQUALS_OUTPUT", "source and destination cannot be the same file")
		}
		return Report{}, fail("OUTPUT_EXISTS", "output already exists: %s", dst)
	}
	if st, err := os.Stat(filepath.Dir(dst)); err != nil || !st.IsDir() {
		return Report{}, fail("OUTPUT_INVALID", "output directory does not exist: %s", filepath.Dir(dst))
	}
	p, err := readSource(src)
	if err != nil {
		return Report{}, err
	}
	data, err := encodeOutput(p)
	if err != nil {
		return Report{}, err
	}
	if err := verifyOutput(data, p); err != nil {
		return Report{}, fail("OUTPUT_VALIDATION_FAILED", "%v", err)
	}
	if err := publish(dst, data); err != nil {
		return Report{}, err
	}
	return buildReport(dst, p), nil
}

// publish writes to a temporary file in the destination directory and then
// hard-links it into place. Link fails if dst exists, so a concurrent writer
// cannot be overwritten.
func publish(dst string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".prusa3mf-*.tmp")
	if err != nil {
		return fail("OUTPUT_INVALID", "%v", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fail("OUTPUT_PUBLISH_FAILED", "%v", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return fail("OUTPUT_PUBLISH_FAILED", "%v", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fail("OUTPUT_PUBLISH_FAILED", "%v", err)
	}
	if err := tmp.Close(); err != nil {
		return fail("OUTPUT_PUBLISH_FAILED", "%v", err)
	}
	if err := os.Link(tmpName, dst); err != nil {
		if os.IsExist(err) {
			return fail("OUTPUT_EXISTS", "output already exists: %s", dst)
		}
		return fail("OUTPUT_PUBLISH_FAILED", "cannot publish without overwrite: %v", err)
	}
	return nil
}

func buildReport(output string, p *project) Report {
	r := Report{
		Output:         output,
		Objects:        len(p.objects),
		BuildItems:     len(p.items),
		DroppedEntries: p.dropped,
	}
	painted := make(map[int]bool)
	for _, obj := range p.objects {
		r.Vertices += len(obj.vertices)
		r.Triangles += len(obj.triangles)
		for _, t := range obj.triangles {
			if t.channel != 0 {
				r.PaintedTriangles++
				painted[int(t.channel)] = true
			}
		}
	}
	for ch := range painted {
		r.PaintChannels = append(r.PaintChannels, ch)
	}
	sort.Ints(r.PaintChannels)

	seen := make(map[string]bool)
	gaps := 0
	for _, c := range p.channels {
		r.Channels = append(r.Channels, ChannelReport{ID: c.id, Color: c.color, Type: c.typ, Placeholder: c.placeholder})
		if c.placeholder {
			gaps++
			continue
		}
		key := strings.ToUpper(c.color) + "|" + strings.ToUpper(c.typ)
		if seen[key] {
			r.DuplicateMaterialSlots++
		}
		seen[key] = true
	}
	r.Warnings = append(r.Warnings, fixedWarnings...)
	r.Warnings = append(r.Warnings, p.warnings...)
	if gaps > 0 {
		r.Warnings = append(r.Warnings, fmt.Sprintf("%d sparse channel gap(s) were kept as placeholder filaments; no object or paint uses them", gaps))
	}
	return r
}
