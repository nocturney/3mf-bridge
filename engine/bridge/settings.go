package bridge

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/BryanSmee/bl_to_std/pkg/printer"
)

type parameterRange struct{ min, max float64 }

var safePrintParameters = map[string]parameterRange{
	"layer_height":               {0.06, 0.32},
	"initial_layer_print_height": {0.08, 0.32},
	"wall_loops":                 {1, 12},
	"top_shell_layers":           {0, 20},
	"bottom_shell_layers":        {0, 20},
	"sparse_infill_density":      {0, 100},
	"support_threshold_angle":    {0, 90},
	"brim_width":                 {0, 30},
	"skirt_loops":                {0, 12},
	"skirt_distance":             {0, 30},
	"elefant_foot_compensation":  {0, 3},
	"elephant_foot_compensation": {0, 3},
	"xy_contour_compensation":    {-2, 2},
}

// Carry over only documented numeric process intent. Hardware limits, scripts,
// temperatures, motion and other slicer-dependent enum values are not copied.
func preservedProfile(source string, baseline *printer.Profile) (*printer.Profile, []string, error) {
	p := *baseline
	p.ProjectSettings = make(map[string]any, len(baseline.ProjectSettings))
	for k, v := range baseline.ProjectSettings {
		p.ProjectSettings[k] = v
	}
	zr, err := zip.OpenReader(source)
	if err != nil {
		return nil, nil, err
	}
	defer zr.Close()
	var raw []byte
	for _, f := range zr.File {
		if f.Name != "Metadata/project_settings.config" {
			continue
		}
		if f.UncompressedSize64 > 8<<20 {
			return nil, nil, fmt.Errorf("oversized project settings")
		}
		r, err := f.Open()
		if err != nil {
			return nil, nil, err
		}
		raw, err = io.ReadAll(io.LimitReader(r, 8<<20))
		r.Close()
		if err != nil {
			return nil, nil, err
		}
		break
	}
	if len(raw) == 0 {
		return &p, nil, nil
	}
	var settings map[string]any
	if err := json.Unmarshal(raw, &settings); err != nil {
		return nil, nil, err
	}
	var preserved []string
	for key, bounds := range safePrintParameters {
		original, ok := settings[key].(string)
		if !ok {
			continue
		}
		if _, supported := p.ProjectSettings[key]; !supported {
			continue
		}
		val, err := parseNumberSetting(original)
		if err != nil || val < bounds.min || val > bounds.max {
			continue
		}
		p.ProjectSettings[key] = original
		preserved = append(preserved, key)
	}
	return &p, preserved, nil
}

func parseNumberSetting(value string) (float64, error) {
	v := strings.TrimSpace(value)
	if len(v) == 0 || len(v) > 24 {
		return 0, fmt.Errorf("invalid number")
	}
	v = strings.TrimSuffix(v, "%")
	number, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, err
	}
	if number != number {
		return 0, fmt.Errorf("NaN not allowed")
	}
	return number, nil
}
