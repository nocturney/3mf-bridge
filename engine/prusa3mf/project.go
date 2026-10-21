package prusa3mf

import (
	"fmt"
	"sort"
)

// project is the validated source content that the writer emits. channels[i]
// is always channel i+1: numbers are never renumbered or compacted on the way out.
type project struct {
	unit          string
	objects       []*object
	items         []buildItem
	channels      []channel
	sourcePrinter string
	dropped       []string
	warnings      []string
}

// bind attaches model settings to objects and checks every channel reference
// against the source filament table. A placeholder channel is never a valid
// target for an object, a volume or a paint code.
func (p *project) bind(doc *modelDoc, settings map[int]objectSettings) error {
	placed := make(map[int]bool, len(p.items))
	for _, it := range p.items {
		if doc.byID[it.objectID] == nil {
			return fail("SOURCE_INVALID", "build item references missing object %d", it.objectID)
		}
		placed[it.objectID] = true
	}
	var orphaned []int
	for id := range settings {
		if doc.byID[id] == nil {
			orphaned = append(orphaned, id)
		}
	}
	sort.Ints(orphaned)
	for _, id := range orphaned {
		p.warnings = append(p.warnings, fmt.Sprintf("model settings for object %d have no mesh and were ignored", id))
	}
	for _, obj := range p.objects {
		if !placed[obj.id] {
			return fail("UNSUPPORTED_UNREFERENCED_OBJECT", "object %d is not placed in the build; unplaced objects are not supported", obj.id)
		}
		if s, ok := settings[obj.id]; ok {
			if s.name != "" {
				obj.name = s.name
			}
			obj.extruder = s.extruder
			obj.volumeExtruder = s.volumeExtruder
		}
		if obj.name == "" {
			obj.name = fmt.Sprintf("Object %d", obj.id)
		}
		if err := p.checkChannel(obj.extruder, "object", obj.id); err != nil {
			return err
		}
		if err := p.checkChannel(obj.volumeExtruder, "volume of object", obj.id); err != nil {
			return err
		}
		for _, t := range obj.triangles {
			if err := p.checkChannel(int(t.channel), "painted triangle in object", obj.id); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkChannel accepts 0 (unset / unpainted) or a non-placeholder channel of the table.
func (p *project) checkChannel(ch int, owner string, id int) error {
	if ch == 0 {
		return nil
	}
	if ch > len(p.channels) || p.channels[ch-1].placeholder {
		return fail("CHANNEL_NOT_IN_SOURCE", "%s %d uses filament channel %d, which is not in the source filament table", owner, id, ch)
	}
	return nil
}
