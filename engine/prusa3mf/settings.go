package prusa3mf

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// maxChannelID caps the numbered filament table. Channels are preserved exactly
// as numbered; this only bounds hostile or accidental palettes.
const maxChannelID = 64

var (
	colorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	typePattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 +._-]{0,31}$`)
)

// objectSettings holds the Bambu model_settings.config facts for one object.
type objectSettings struct {
	name           string
	extruder       int
	volumeExtruder int
}

type filamentEntry struct {
	id    int
	typ   string
	color string
}

type projectSettings struct {
	printerModel string
	colors       []string
	types        []string
	hasColors    bool
	hasTypes     bool
}

// channel is one numbered virtual filament. Placeholder channels fill numbering
// gaps so that channel N is always written at position N; they are never valid
// targets for an object, a volume or a paint code.
type channel struct {
	id          int
	color       string
	typ         string
	placeholder bool
}

func parseModelSettings(data []byte) (map[int]objectSettings, error) {
	root, err := parseSmallXML(data)
	if err != nil {
		return nil, err
	}
	if root.name != "config" {
		return nil, fail("SOURCE_INVALID", "model settings root is %q, expected config", root.name)
	}
	out := make(map[int]objectSettings)
	for _, obj := range root.childrenNamed("object") {
		id, err := strconv.Atoi(obj.attrs["id"])
		if err != nil || id <= 0 {
			return nil, fail("SOURCE_INVALID", "model settings object id %q is invalid", obj.attrs["id"])
		}
		if _, dup := out[id]; dup {
			return nil, fail("SOURCE_INVALID", "model settings list object %d twice", id)
		}
		var s objectSettings
		s.name, _ = obj.metadata("name")
		if s.extruder, err = channelMetadata(obj, "extruder"); err != nil {
			return nil, err
		}
		parts := obj.childrenNamed("part")
		if len(parts) > 1 {
			return nil, fail("UNSUPPORTED_MULTIPART", "object %d has %d parts; multi-volume objects are not supported", id, len(parts))
		}
		for _, part := range parts {
			if sub := part.attrs["subtype"]; sub != "" && sub != "normal_part" {
				return nil, fail("UNSUPPORTED_MODIFIER_PART", "object %d has a %q part; modifier volumes are not supported", id, sub)
			}
			if _, ok := part.metadata("firstid"); ok {
				return nil, fail("UNSUPPORTED_MULTIPART", "object %d declares triangle ranges for its parts; not supported", id)
			}
			if _, ok := part.metadata("lastid"); ok {
				return nil, fail("UNSUPPORTED_MULTIPART", "object %d declares triangle ranges for its parts; not supported", id)
			}
			if s.volumeExtruder, err = channelMetadata(part, "extruder"); err != nil {
				return nil, err
			}
		}
		out[id] = s
	}
	return out, nil
}

func channelMetadata(n *xmlNode, key string) (int, error) {
	s, ok := n.metadata(key)
	if !ok {
		return 0, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 || v > maxChannelID {
		return 0, fail("SOURCE_INVALID", "%s %q is not a valid filament channel", key, s)
	}
	return v, nil
}

// parseSliceInfo collects every <filament id type color/> entry. Duplicate and
// conflicting ids are rejected later, in buildChannels.
func parseSliceInfo(data []byte) ([]filamentEntry, error) {
	root, err := parseSmallXML(data)
	if err != nil {
		return nil, err
	}
	var out []filamentEntry
	var walk func(n *xmlNode) error
	walk = func(n *xmlNode) error {
		if n.name == "filament" {
			id, err := strconv.Atoi(n.attrs["id"])
			if err != nil || id < 1 {
				return fail("SOURCE_INVALID", "filament id %q is invalid", n.attrs["id"])
			}
			out = append(out, filamentEntry{id: id, typ: n.attrs["type"], color: n.attrs["color"]})
			return nil
		}
		for _, c := range n.children {
			if err := walk(c); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(root); err != nil {
		return nil, err
	}
	return out, nil
}

// parseProjectSettings reads only the filament arrays and printer name. The
// rest of the Bambu process/printer configuration is deliberately not carried
// over, and its firmware scripts are never copied.
func parseProjectSettings(data []byte) (projectSettings, error) {
	var ps projectSettings
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return ps, fail("SOURCE_INVALID", "project settings JSON: %v", err)
	}
	if v, ok := raw["printer_model"]; ok {
		_ = json.Unmarshal(v, &ps.printerModel)
	}
	if v, ok := raw["filament_colour"]; ok {
		if err := json.Unmarshal(v, &ps.colors); err != nil {
			return ps, fail("SOURCE_INVALID", "filament_colour must be a list of strings")
		}
		ps.hasColors = true
	}
	if v, ok := raw["filament_type"]; ok {
		if err := json.Unmarshal(v, &ps.types); err != nil {
			return ps, fail("SOURCE_INVALID", "filament_type must be a list of strings")
		}
		ps.hasTypes = true
	}
	return ps, nil
}

// buildChannels returns the numbered table. Slice info is authoritative when
// present; project filament arrays are used when it is absent and must agree
// with it where both list a channel.
func buildChannels(slice []filamentEntry, ps projectSettings) ([]channel, error) {
	entries := make(map[int]filamentEntry)
	switch {
	case len(slice) > 0:
		for _, e := range slice {
			if e.id > maxChannelID {
				return nil, fail("CHANNEL_INVALID", "filament id %d exceeds the %d channel limit", e.id, maxChannelID)
			}
			if prev, ok := entries[e.id]; ok {
				if prev != e {
					return nil, fail("FILAMENT_TABLE_CONFLICT", "filament id %d is listed twice with different settings", e.id)
				}
				continue
			}
			entries[e.id] = e
		}
		for id, e := range entries {
			if ps.hasColors && id <= len(ps.colors) && !strings.EqualFold(ps.colors[id-1], e.color) {
				return nil, fail("FILAMENT_TABLE_CONFLICT", "filament %d colour differs between slice info and project settings", id)
			}
			if ps.hasTypes && id <= len(ps.types) && !strings.EqualFold(ps.types[id-1], e.typ) {
				return nil, fail("FILAMENT_TABLE_CONFLICT", "filament %d material differs between slice info and project settings", id)
			}
		}
	case ps.hasColors:
		if !ps.hasTypes || len(ps.types) != len(ps.colors) {
			return nil, fail("SOURCE_INVALID", "filament_type must list one material per filament_colour entry")
		}
		if len(ps.colors) > maxChannelID {
			return nil, fail("CHANNEL_INVALID", "project lists %d filaments; the limit is %d", len(ps.colors), maxChannelID)
		}
		for i := range ps.colors {
			entries[i+1] = filamentEntry{id: i + 1, typ: ps.types[i], color: ps.colors[i]}
		}
	default:
		return nil, fail("NO_FILAMENTS", "source has no filament table (slice info or project filament_colour)")
	}
	if len(entries) == 0 {
		return nil, fail("NO_FILAMENTS", "source filament table is empty")
	}
	maxID := 0
	for id := range entries {
		maxID = max(maxID, id)
	}
	table := make([]channel, 0, maxID)
	for id := 1; id <= maxID; id++ {
		e, ok := entries[id]
		if !ok {
			table = append(table, channel{id: id, placeholder: true})
			continue
		}
		if !colorPattern.MatchString(e.color) {
			return nil, fail("CHANNEL_INVALID", "filament %d has invalid colour %q", id, e.color)
		}
		if !typePattern.MatchString(e.typ) {
			return nil, fail("CHANNEL_INVALID", "filament %d has invalid material %q", id, e.typ)
		}
		table = append(table, channel{id: id, color: e.color, typ: e.typ})
	}
	return table, nil
}
