package presets

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// Bounded reader for PrusaSlicer INI profiles. Only "[type:name]" headers and
// "key = value" lines are accepted. Values stay raw text: nothing read here is
// evaluated, expanded, or used to open another file.

const (
	maxPrusaINIBytes    = 8 << 20
	maxPrusaINILine     = 64 << 10
	maxPrusaINISections = 1 << 15
	maxPrusaINIKeys     = 1 << 18
	maxPrusaSectionName = 256
	maxPrusaValueBytes  = 16 << 10
	maxPrusaListItems   = 256
)

// Keys may carry a list index, as in dynamic_overhang_speeds[0].
var prusaKeyPattern = regexp.MustCompile(`^[a-z0-9_]{1,128}(\[[0-9]{1,3}\])?$`)

type prusaSection map[string]string

// prusaINI keeps sections in file order so discovery stays deterministic.
type prusaINI struct {
	order    []string
	sections map[string]prusaSection
}

func parsePrusaINI(data []byte) (*prusaINI, error) {
	if len(data) > maxPrusaINIBytes {
		return nil, fmt.Errorf("file exceeds %d bytes", maxPrusaINIBytes)
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return nil, fmt.Errorf("file contains NUL bytes")
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	ini := &prusaINI{sections: map[string]prusaSection{}}
	var current prusaSection
	keys := 0
	for number, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if len(line) > maxPrusaINILine {
			return nil, fmt.Errorf("line %d exceeds %d bytes", number+1, maxPrusaINILine)
		}
		if hasPrusaControl(line) {
			return nil, fmt.Errorf("line %d contains control characters", number+1)
		}
		text := strings.TrimSpace(line)
		if text == "" || text[0] == '#' || text[0] == ';' {
			continue
		}
		if text[0] == '[' {
			if !strings.HasSuffix(text, "]") {
				return nil, fmt.Errorf("line %d has an unterminated section header", number+1)
			}
			name := strings.TrimSpace(text[1 : len(text)-1])
			if name == "" || len(name) > maxPrusaSectionName || strings.ContainsAny(name, "[]") {
				return nil, fmt.Errorf("line %d has an invalid section name", number+1)
			}
			if _, duplicate := ini.sections[name]; duplicate {
				return nil, fmt.Errorf("section %q is defined twice", name)
			}
			if len(ini.order) >= maxPrusaINISections {
				return nil, fmt.Errorf("more than %d sections", maxPrusaINISections)
			}
			current = prusaSection{}
			ini.sections[name] = current
			ini.order = append(ini.order, name)
			continue
		}
		if current == nil {
			return nil, fmt.Errorf("line %d sits outside any section", number+1)
		}
		separator := strings.IndexByte(text, '=')
		if separator < 0 {
			return nil, fmt.Errorf("line %d is not a key = value setting", number+1)
		}
		key := strings.TrimSpace(text[:separator])
		value := strings.TrimSpace(text[separator+1:])
		if !prusaKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("line %d has an invalid key", number+1)
		}
		if _, duplicate := current[key]; duplicate {
			return nil, fmt.Errorf("line %d repeats key %q", number+1, key)
		}
		if len(value) > maxPrusaValueBytes {
			return nil, fmt.Errorf("line %d value exceeds %d bytes", number+1, maxPrusaValueBytes)
		}
		keys++
		if keys > maxPrusaINIKeys {
			return nil, fmt.Errorf("more than %d settings", maxPrusaINIKeys)
		}
		current[key] = value
	}
	return ini, nil
}

// hasPrusaControl reports control characters other than tab. CR is removed
// before this check, so Windows line endings are accepted.
func hasPrusaControl(line string) bool {
	for i := 0; i < len(line); i++ {
		if (line[i] < 0x20 && line[i] != '\t') || line[i] == 0x7f {
			return true
		}
	}
	return false
}

// splitPrusaValues splits a per-tool value. PrusaSlicer writes numeric vectors
// such as nozzle_diameter with commas, and string lists with semicolons and
// quotes. Quoted or semicolon lists use splitPrusaList. Anything else splits on
// commas.
func splitPrusaValues(raw string) ([]string, error) {
	if strings.ContainsAny(raw, `;"`) {
		return splitPrusaList(raw)
	}
	items := []string{}
	for _, item := range strings.Split(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	if len(items) > maxPrusaListItems {
		return nil, fmt.Errorf("list has more than %d items", maxPrusaListItems)
	}
	return items, nil
}

// splitPrusaList splits a PrusaSlicer list value such as `"A;B";C`. Separators
// inside double quotes are kept, and backslash escapes a quote or backslash.
// Empty items are dropped, so an empty value means an empty list.
func splitPrusaList(raw string) ([]string, error) {
	items := []string{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return items, nil
	}
	var item strings.Builder
	quoted := false
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		switch {
		case quoted && c == '\\' && i+1 < len(raw) && (raw[i+1] == '"' || raw[i+1] == '\\'):
			item.WriteByte(raw[i+1])
			i++
		case c == '"':
			quoted = !quoted
		case c == ';' && !quoted:
			items = append(items, strings.TrimSpace(item.String()))
			item.Reset()
		default:
			item.WriteByte(c)
		}
		if len(items) > maxPrusaListItems {
			return nil, fmt.Errorf("list has more than %d items", maxPrusaListItems)
		}
	}
	if quoted {
		return nil, fmt.Errorf("list has an unterminated quote")
	}
	items = append(items, strings.TrimSpace(item.String()))
	kept := items[:0]
	for _, value := range items {
		if value != "" {
			kept = append(kept, value)
		}
	}
	return kept, nil
}
