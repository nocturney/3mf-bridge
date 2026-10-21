package prusa3mf

import (
	"bytes"
	"encoding/xml"
	"io"
)

// Limits for the small XML parts (model settings, slice info, relationships).
// The large model part is streamed by model.go instead of building a tree.
const (
	maxXMLDepth      = 32
	maxSmallXMLNodes = 100000
)

// xmlNode is a deliberately small tree for configuration parts. Namespaces are
// ignored here because these parts use unqualified names.
type xmlNode struct {
	name     string
	attrs    map[string]string
	children []*xmlNode
}

// parseSmallXML builds a bounded tree. DTD directives are refused outright,
// which also rules out entity declarations and therefore entity expansion.
func parseSmallXML(data []byte) (*xmlNode, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var root *xmlNode
	var stack []*xmlNode
	nodes := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fail("SOURCE_INVALID", "invalid XML: %v", err)
		}
		switch t := tok.(type) {
		case xml.Directive:
			return nil, fail("UNSUPPORTED_XML", "DTD and entity declarations are not allowed")
		case xml.StartElement:
			nodes++
			if nodes > maxSmallXMLNodes || len(stack) >= maxXMLDepth {
				return nil, fail("ARCHIVE_LIMIT", "XML part exceeds the element or depth limit")
			}
			n := &xmlNode{name: t.Name.Local, attrs: make(map[string]string, len(t.Attr))}
			for _, a := range t.Attr {
				n.attrs[a.Name.Local] = a.Value
			}
			if len(stack) == 0 {
				if root != nil {
					return nil, fail("SOURCE_INVALID", "XML part has more than one root element")
				}
				root = n
			} else {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, n)
			}
			stack = append(stack, n)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if root == nil {
		return nil, fail("SOURCE_INVALID", "XML part is empty")
	}
	return root, nil
}

func (n *xmlNode) childrenNamed(name string) []*xmlNode {
	var out []*xmlNode
	for _, c := range n.children {
		if c.name == name {
			out = append(out, c)
		}
	}
	return out
}

// metadata returns the value of a direct <metadata key="..." value="..."/> child.
func (n *xmlNode) metadata(key string) (string, bool) {
	for _, c := range n.children {
		if c.name == "metadata" && c.attrs["key"] == key {
			return c.attrs["value"], true
		}
	}
	return "", false
}
