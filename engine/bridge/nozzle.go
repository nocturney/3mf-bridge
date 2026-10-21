package bridge

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// NozzleInfo reports what the Bambu project itself says about installed
// nozzle diameters. Ambiguous/mixed extruder diameters must not be folded.
type NozzleInfo struct {
	Nozzle  string `json:"source_nozzle"`
	Source  string `json:"source_nozzle_source"`
	Assumed bool   `json:"source_nozzle_assumed"`
}

var nozzleInPreset = regexp.MustCompile(`(?i)(?:^|[^0-9])(0\.2[05]?|0\.4|0\.6|0\.8)\s*(?:mm\s*)?nozzle\b`)

func normalizedNozzle(v any) (string, error) {
	var s string
	switch x := v.(type) {
	case string:
		s = strings.TrimSpace(x)
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return "", fmt.Errorf("invalid nozzle diameter")
		}
		s = strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return "", fmt.Errorf("invalid nozzle diameter type %T", v)
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return "", fmt.Errorf("invalid nozzle diameter %q", s)
	}
	for _, supported := range []string{"0.2", "0.25", "0.4", "0.6", "0.8"} {
		ref, _ := strconv.ParseFloat(supported, 64)
		if math.Abs(ref-n) < 1e-6 {
			return supported, nil
		}
	}
	return "", fmt.Errorf("unsupported nozzle diameter %q", s)
}

func nozzleFromRaw(raw any) (string, error) {
	if raw == nil {
		return "", nil
	}
	entries, ok := raw.([]any)
	if !ok {
		entries = []any{raw}
	}
	if len(entries) == 0 {
		return "", nil
	}
	if len(entries) > 16 {
		return "", fmt.Errorf("too many nozzle dimensions")
	}
	result := ""
	for _, v := range entries {
		next, err := normalizedNozzle(v)
		if err != nil {
			return "", err
		}
		if result == "" {
			result = next
		}
		if next != result {
			return "", fmt.Errorf("mixed nozzle diameters %s and %s are not supported", result, next)
		}
	}
	return result, nil
}

func nozzleFromPresetName(name string) string {
	found := nozzleInPreset.FindStringSubmatch(name)
	if len(found) != 2 {
		return ""
	}
	val, err := normalizedNozzle(found[1])
	if err != nil {
		return ""
	}
	return val
}

// detectSourceNozzle expects the caller to have validated the archive with
// inspectPackage. Missing Bambu diameter uses the old 0.4mm workflow, but is
// explicitly flagged and displayed as an assumption.
func detectSourceNozzle(source string) (NozzleInfo, error) {
	var result NozzleInfo
	archive, err := zip.OpenReader(source)
	if err != nil {
		return result, err
	}
	defer archive.Close()
	var settings map[string]any
	for _, f := range archive.File {
		if f.Name != "Metadata/project_settings.config" {
			continue
		}
		if f.UncompressedSize64 > 8<<20 {
			return result, fmt.Errorf("oversized project settings")
		}
		reader, err := f.Open()
		if err != nil {
			return result, err
		}
		data, err := io.ReadAll(io.LimitReader(reader, 8<<20))
		reader.Close()
		if err != nil {
			return result, err
		}
		if err := json.Unmarshal(data, &settings); err != nil {
			return result, err
		}
		break
	}
	explicit, err := nozzleFromRaw(settings["nozzle_diameter"])
	if err != nil {
		return result, err
	}
	if explicit == "" {
		explicit, err = nozzleFromRaw(settings["nozzle_size"])
		if err != nil {
			return result, err
		}
	}
	fromPreset := ""
	for _, field := range []string{"printer_settings_id", "printer_model"} {
		value, ok := settings[field].(string)
		if !ok {
			continue
		}
		found := nozzleFromPresetName(value)
		if found == "" {
			continue
		}
		if fromPreset != "" && fromPreset != found {
			return result, fmt.Errorf("conflicting nozzle sizes in project printer identifiers")
		}
		fromPreset = found
	}
	if explicit != "" && fromPreset != "" && explicit != fromPreset {
		return result, fmt.Errorf("nozzle size in project settings (%s) conflicts with printer preset (%s)", explicit, fromPreset)
	}
	if explicit != "" {
		return NozzleInfo{Nozzle: explicit, Source: "nozzle_diameter"}, nil
	}
	if fromPreset != "" {
		return NozzleInfo{Nozzle: fromPreset, Source: "printer_settings_id"}, nil
	}
	return NozzleInfo{Nozzle: "0.4", Source: "assumed_default", Assumed: true}, nil
}

// InspectNozzle performs the same ZIP/geometry safety checks as conversion.
func InspectNozzle(source string) (map[string]any, error) {
	if _, err := inspectPackage(source); err != nil {
		return nil, fail("SOURCE_INVALID", "%v", err)
	}
	nozzle, err := detectSourceNozzle(source)
	if err != nil {
		return nil, fail("SOURCE_NOZZLE_INVALID", "%v", err)
	}
	return map[string]any{
		"ok":                    true,
		"source_nozzle":         nozzle.Nozzle,
		"source_nozzle_source":  nozzle.Source,
		"source_nozzle_assumed": nozzle.Assumed,
	}, nil
}
