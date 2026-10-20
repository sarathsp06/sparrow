package infer

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
)

// Element is an XML element inferred from a sample.
type Element struct {
	// Name is the local name, without namespace prefix.
	Name string
	// Attrs are the attribute local names, sorted.
	Attrs []string
	// Children are the distinct child elements, sorted by name.
	Children []*Element
	// Repeated is set when the element occurs more than once under its
	// parent.
	Repeated bool
	// Text is set when the element has non-whitespace character data.
	Text bool
}

// FromXML infers the element tree of one XML document. Elements with the
// same name under one parent are folded into one, marked Repeated when any
// parent instance has more than one.
func FromXML(sample []byte) (*Element, error) {
	dec := xml.NewDecoder(bytes.NewReader(sample))
	dec.Strict = true
	type frame struct {
		el *Element
		// counts are this instance's children by name.
		counts map[string]int
	}
	var stack []frame
	var root *Element
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("infer: invalid XML: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if len(stack) > maxDepth {
				return nil, errors.New("infer: XML nested too deeply")
			}
			var attrs []string
			for _, a := range t.Attr {
				if a.Name.Space == "xmlns" || a.Name.Local == "xmlns" {
					continue
				}
				attrs = appendUnique(attrs, a.Name.Local)
			}
			var el *Element
			if len(stack) == 0 {
				if root != nil {
					return nil, errors.New("infer: invalid XML: more than one root element")
				}
				root = &Element{Name: t.Name.Local, Attrs: attrs}
				el = root
			} else {
				parent := stack[len(stack)-1]
				parent.counts[t.Name.Local]++
				el = parent.el.child(t.Name.Local)
				el.Attrs = appendUnique(el.Attrs, attrs...)
				if parent.counts[t.Name.Local] > 1 {
					el.Repeated = true
				}
			}
			stack = append(stack, frame{el: el, counts: map[string]int{}})
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) > 0 && len(bytes.TrimSpace(t)) > 0 {
				stack[len(stack)-1].el.Text = true
			}
		}
	}
	if root == nil {
		return nil, errors.New("infer: invalid XML: no root element")
	}
	root.sort()
	return root, nil
}

// child returns e's child element called name, adding it on first sight.
func (e *Element) child(name string) *Element {
	for _, c := range e.Children {
		if c.Name == name {
			return c
		}
	}
	c := &Element{Name: name}
	e.Children = append(e.Children, c)
	return c
}

func (e *Element) sort() {
	sort.Strings(e.Attrs)
	sort.Slice(e.Children, func(i, j int) bool { return e.Children[i].Name < e.Children[j].Name })
	for _, c := range e.Children {
		c.sort()
	}
}

func appendUnique(list []string, names ...string) []string {
	for _, n := range names {
		if !slices.Contains(list, n) {
			list = append(list, n)
		}
	}
	return list
}
