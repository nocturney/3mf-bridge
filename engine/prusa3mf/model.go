package prusa3mf

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"slices"
	"strconv"
	"strings"
)

const (
	coreNS       = "http://schemas.microsoft.com/3dmanufacturing/core/2015/02"
	productionNS = "http://schemas.microsoft.com/3dmanufacturing/production/2015/06"
	slic3rpeNS   = "http://schemas.slic3r.org/3mf/2017/06"
	maxObjects   = 1024
	maxItems     = 4096
	maxVertices  = 4000000
	maxTriangles = 4000000
)

var validUnits = map[string]bool{
	"micron": true, "millimeter": true, "centimeter": true,
	"inch": true, "foot": true, "meter": true,
}

// triangle keeps indices as int32 and the painted channel as a byte so that
// multi-million-triangle models stay within a predictable memory budget.
type triangle struct {
	v       [3]int32
	channel uint8 // 1-based filament channel; 0 = unpainted
}

type object struct {
	id             int
	name           string
	hasMesh        bool
	vertices       [][3]float64
	triangles      []triangle
	extruder       int // object-level channel from model settings; 0 = unset
	volumeExtruder int // single-volume channel from model settings; 0 = unset
}

type buildItem struct {
	objectID  int
	transform *[12]float64 // nil = not written (identity)
	printable string       // "", "0" or "1"
}

type modelDoc struct {
	unit    string
	objects []*object
	byID    map[int]*object
	items   []buildItem
}

type modelReader struct {
	dec        *xml.Decoder
	depth      int
	prusaPaint bool // also accept slic3rpe:mmu_segmentation; used only to verify our own output
	doc        modelDoc
	totalVerts int
	totalTris  int
}

// parseModel streams the 3D model part. Every element and attribute outside
// the supported set is refused rather than silently dropped, so nothing that
// affects printing can disappear without an error.
func parseModel(data []byte, prusaPaint bool) (*modelDoc, error) {
	m := &modelReader{dec: xml.NewDecoder(bytes.NewReader(data)), prusaPaint: prusaPaint}
	m.doc.unit = "millimeter"
	m.doc.byID = make(map[int]*object)
	if err := m.root(); err != nil {
		return nil, err
	}
	return &m.doc, nil
}

func (m *modelReader) root() error {
	for {
		tok, err := m.dec.Token()
		if err == io.EOF {
			return fail("SOURCE_INVALID", "3D model part has no root element")
		}
		if err != nil {
			return fail("SOURCE_INVALID", "invalid 3D model XML: %v", err)
		}
		switch t := tok.(type) {
		case xml.Directive:
			return fail("UNSUPPORTED_XML", "DTD and entity declarations are not allowed")
		case xml.StartElement:
			if t.Name.Space != coreNS || t.Name.Local != "model" {
				return fail("UNSUPPORTED_XML", "3D model root %q is not a 3MF core model", t.Name.Local)
			}
			if u, ok := attr(t, "unit"); ok {
				if !validUnits[u] {
					return fail("UNSUPPORTED_UNIT", "unit %q is not supported", u)
				}
				m.doc.unit = u
			}
			return m.children(m.modelChild)
		}
	}
}

// children calls handle for each start element until the enclosing end element.
// Each handler must consume its own element completely.
func (m *modelReader) children(handle func(xml.StartElement) error) error {
	m.depth++
	defer func() { m.depth-- }()
	if m.depth > maxXMLDepth {
		return fail("UNSUPPORTED_XML", "XML nesting is deeper than %d levels", maxXMLDepth)
	}
	for {
		tok, err := m.dec.Token()
		if err != nil {
			return fail("SOURCE_INVALID", "invalid 3D model XML: %v", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Space != "" && t.Name.Space != coreNS {
				return fail("UNSUPPORTED_ELEMENT", "element %q in namespace %q is not supported", t.Name.Local, t.Name.Space)
			}
			if err := handle(t); err != nil {
				return err
			}
		case xml.EndElement:
			return nil
		case xml.Directive:
			return fail("UNSUPPORTED_XML", "DTD and entity declarations are not allowed")
		}
	}
}

func (m *modelReader) modelChild(el xml.StartElement) error {
	switch el.Name.Local {
	case "metadata":
		return m.dec.Skip()
	case "resources":
		return m.children(m.resource)
	case "build":
		return m.children(m.item)
	}
	return fail("UNSUPPORTED_ELEMENT", "model element %q is not supported", el.Name.Local)
}

func (m *modelReader) resource(el xml.StartElement) error {
	switch el.Name.Local {
	case "metadata":
		return m.dec.Skip()
	case "object":
		return m.object(el)
	}
	return fail("UNSUPPORTED_ELEMENT", "resource %q is not supported; only mesh objects are", el.Name.Local)
}

func (m *modelReader) object(el xml.StartElement) error {
	if err := checkAttrs(el, "id", "type", "name", "partnumber"); err != nil {
		return err
	}
	id, err := positiveInt(el, "id")
	if err != nil {
		return err
	}
	if m.doc.byID[id] != nil {
		return fail("SOURCE_INVALID", "duplicate object id %d", id)
	}
	if len(m.doc.objects) >= maxObjects {
		return fail("ARCHIVE_LIMIT", "model has more than %d objects", maxObjects)
	}
	if typ, _ := attr(el, "type"); typ != "" && typ != "model" {
		return fail("UNSUPPORTED_OBJECT_TYPE", "object %d has type %q; only model objects are supported", id, typ)
	}
	name, _ := attr(el, "name")
	obj := &object{id: id, name: name}
	m.doc.objects = append(m.doc.objects, obj)
	m.doc.byID[id] = obj
	if err := m.children(func(child xml.StartElement) error { return m.objectChild(child, obj) }); err != nil {
		return err
	}
	if !obj.hasMesh {
		return fail("UNSUPPORTED_OBJECT", "object %d has no mesh", id)
	}
	if len(obj.triangles) == 0 {
		return fail("SOURCE_INVALID", "object %d has no triangles", id)
	}
	return nil
}

func (m *modelReader) objectChild(el xml.StartElement, obj *object) error {
	switch el.Name.Local {
	case "metadata":
		return m.dec.Skip()
	case "mesh":
		if obj.hasMesh {
			return fail("SOURCE_INVALID", "object %d has more than one mesh", obj.id)
		}
		obj.hasMesh = true
		return m.children(func(child xml.StartElement) error { return m.meshChild(child, obj) })
	case "components":
		return fail("UNSUPPORTED_COMPONENTS", "object %d uses components; only single-mesh objects are supported", obj.id)
	}
	return fail("UNSUPPORTED_ELEMENT", "object %d contains unsupported element %q", obj.id, el.Name.Local)
}

func (m *modelReader) meshChild(el xml.StartElement, obj *object) error {
	switch el.Name.Local {
	case "metadata":
		return m.dec.Skip()
	case "vertices":
		return m.children(func(v xml.StartElement) error { return m.vertex(v, obj) })
	case "triangles":
		return m.children(func(t xml.StartElement) error { return m.triangle(t, obj) })
	}
	return fail("UNSUPPORTED_ELEMENT", "mesh of object %d contains unsupported element %q", obj.id, el.Name.Local)
}

func (m *modelReader) vertex(el xml.StartElement, obj *object) error {
	if el.Name.Local == "metadata" {
		return m.dec.Skip()
	}
	if el.Name.Local != "vertex" {
		return fail("UNSUPPORTED_ELEMENT", "vertices of object %d contain unsupported element %q", obj.id, el.Name.Local)
	}
	if m.totalVerts >= maxVertices {
		return fail("ARCHIVE_LIMIT", "model has more than %d vertices", maxVertices)
	}
	var v [3]float64
	for i, key := range [3]string{"x", "y", "z"} {
		s, ok := attr(el, key)
		if !ok {
			return fail("SOURCE_INVALID", "vertex of object %d is missing %s", obj.id, key)
		}
		f, err := parseFinite(s)
		if err != nil {
			return fail("SOURCE_INVALID", "vertex of object %d has invalid %s %q", obj.id, key, s)
		}
		v[i] = f
	}
	obj.vertices = append(obj.vertices, v)
	m.totalVerts++
	return m.dec.Skip()
}

func (m *modelReader) triangle(el xml.StartElement, obj *object) error {
	if el.Name.Local == "metadata" {
		return m.dec.Skip()
	}
	if el.Name.Local != "triangle" {
		return fail("UNSUPPORTED_ELEMENT", "triangles of object %d contain unsupported element %q", obj.id, el.Name.Local)
	}
	if m.prusaPaint {
		// The verifier reads PrusaSlicer paint; rename it so one code path handles both.
		for i := range el.Attr {
			if el.Attr[i].Name.Space == slic3rpeNS && el.Attr[i].Name.Local == "mmu_segmentation" {
				el.Attr[i].Name = xml.Name{Local: "paint_color"}
			}
		}
	}
	if err := checkAttrs(el, "v1", "v2", "v3", "paint_color"); err != nil {
		return err
	}
	var t triangle
	for i, key := range [3]string{"v1", "v2", "v3"} {
		s, ok := attr(el, key)
		if !ok {
			return fail("SOURCE_INVALID", "triangle of object %d is missing %s", obj.id, key)
		}
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 || n > 2147483647 || n >= len(obj.vertices) {
			return fail("SOURCE_INVALID", "triangle of object %d references vertex %q outside the mesh", obj.id, s)
		}
		// The 3MF writer uses int32 indices; reject oversized values instead
		// of allowing architecture-dependent truncation on 64-bit hosts.
		t.v[i] = int32(n)
	}
	if code, ok := attr(el, "paint_color"); ok {
		channel, err := decodePaint(code)
		if err != nil {
			return err
		}
		t.channel = uint8(channel)
	}
	if m.totalTris >= maxTriangles {
		return fail("ARCHIVE_LIMIT", "model has more than %d triangles", maxTriangles)
	}
	obj.triangles = append(obj.triangles, t)
	m.totalTris++
	return m.dec.Skip()
}

func (m *modelReader) item(el xml.StartElement) error {
	if el.Name.Local == "metadata" {
		return m.dec.Skip()
	}
	if el.Name.Local != "item" {
		return fail("UNSUPPORTED_ELEMENT", "build contains unsupported element %q", el.Name.Local)
	}
	if err := checkAttrs(el, "objectid", "transform", "printable", "partnumber"); err != nil {
		return err
	}
	if len(m.doc.items) >= maxItems {
		return fail("ARCHIVE_LIMIT", "build has more than %d items", maxItems)
	}
	id, err := positiveInt(el, "objectid")
	if err != nil {
		return err
	}
	it := buildItem{objectID: id}
	if s, ok := attr(el, "transform"); ok {
		t, err := parseTransform(s)
		if err != nil {
			return err
		}
		it.transform = &t
	}
	if s, ok := attr(el, "printable"); ok {
		if s != "0" && s != "1" {
			return fail("UNSUPPORTED_ATTRIBUTE", "item printable=%q is not supported", s)
		}
		it.printable = s
	}
	m.doc.items = append(m.doc.items, it)
	return m.children(func(child xml.StartElement) error {
		if child.Name.Local == "metadata" {
			return m.dec.Skip()
		}
		return fail("UNSUPPORTED_ELEMENT", "build item %d contains unsupported element %q", id, child.Name.Local)
	})
}

// checkAttrs refuses any attribute outside allowed. Namespaced attributes are
// refused too, except production-extension identifiers (such as p:UUID), which
// carry no mesh data and are dropped deliberately.
func checkAttrs(el xml.StartElement, allowed ...string) error {
	for _, a := range el.Attr {
		if a.Name.Space == "xmlns" || (a.Name.Space == "" && a.Name.Local == "xmlns") {
			continue
		}
		if a.Name.Space == productionNS {
			if a.Name.Local == "path" {
				return fail("UNSUPPORTED_EXTERNAL_MODEL", "%s references an external model part", el.Name.Local)
			}
			continue
		}
		if a.Name.Space == "" && slices.Contains(allowed, a.Name.Local) {
			continue
		}
		return fail("UNSUPPORTED_ATTRIBUTE", "%s attribute %q is not supported", el.Name.Local, a.Name.Local)
	}
	return nil
}

func attr(el xml.StartElement, name string) (string, bool) {
	for _, a := range el.Attr {
		if a.Name.Space == "" && a.Name.Local == name {
			return a.Value, true
		}
	}
	return "", false
}

func positiveInt(el xml.StartElement, name string) (int, error) {
	s, ok := attr(el, name)
	if !ok {
		return 0, fail("SOURCE_INVALID", "%s element is missing %s", el.Name.Local, name)
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, fail("SOURCE_INVALID", "%s has invalid %s %q", el.Name.Local, name, s)
	}
	return n, nil
}

func parseFinite(s string) (float64, error) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("non-finite value %q", s)
	}
	return f, nil
}

// parseTransform reads the twelve row-major values of a 3MF build transform.
func parseTransform(s string) ([12]float64, error) {
	var out [12]float64
	fields := strings.Fields(s)
	if len(fields) != 12 {
		return out, fail("SOURCE_INVALID", "item transform must have 12 values, found %d", len(fields))
	}
	for i, f := range fields {
		v, err := parseFinite(f)
		if err != nil {
			return out, fail("SOURCE_INVALID", "item transform has invalid value %q", f)
		}
		out[i] = v
	}
	return out, nil
}
