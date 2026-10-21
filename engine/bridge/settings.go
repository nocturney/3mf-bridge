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
	"layer_height":               {0.04, 0.64},
	"initial_layer_print_height": {0.04, 0.64},
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
		if key == "layer_height" || key == "initial_layer_print_height" {
			// A Bambu 0.2mm layer height is not necessarily valid for a
			// Creator 5 0.25mm toolhead or another target nozzle. If out
			// of range, retain the target slicer's native process value.
			min, minOK := profileLayerLimit(p.ProjectSettings["min_layer_height"])
			max, maxOK := profileLayerLimit(p.ProjectSettings["max_layer_height"])
			if (minOK && val < min) || (maxOK && val > max) {
				continue
			}
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

func profileLayerLimit(raw any) (float64, bool) {
	if raw == nil {
		return 0, false
	}
	var vals []any
	switch value := raw.(type) {
	case []any:
		vals = value
	case []string:
		for _, item := range value {
			vals = append(vals, item)
		}
	default:
		vals = []any{value}
	}
	if len(vals) == 0 {
		return 0, false
	}
	var first float64
	for i, item := range vals {
		var s string
		switch v := item.(type) {
		case string:
			s = v
		case float64:
			s = strconv.FormatFloat(v, 'f', -1, 64)
		default:
			return 0, false
		}
		n, err := parseNumberSetting(s)
		if err != nil || n <= 0 || n > 1 {
			return 0, false
		}
		if i > 0 && (n-first > 0.000001 || first-n > 0.000001) {
			return 0, false
		}
		first = n
	}
	return first, true
}
