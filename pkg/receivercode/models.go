package receivercode

import (
	"encoding/json"
	"strconv"

	"github.com/sarathsp06/sparrow/pkg/receivercode/infer"
)

// objDecl is one named object type to declare.
type objDecl struct {
	Name string
	Type *infer.Type
	// Parents lists this declaration's place, for a doc comment.
	Doc string
}

// typeNamer assigns unique type names to the object types of a model graph.
// Names are global to the generated project, so two subscriptions never
// declare the same type.
type typeNamer struct {
	used  map[string]bool
	names map[*infer.Type]string
	decls []objDecl
	// fieldOK reports whether a JSON key can be a field in the target
	// language; an object with a key that cannot becomes a map.
	fieldOK func(key string) bool
}

func newTypeNamer(fieldOK func(string) bool, reserved ...string) *typeNamer {
	n := &typeNamer{used: map[string]bool{}, names: map[*infer.Type]string{}, fieldOK: fieldOK}
	for _, r := range reserved {
		n.used[r] = true
	}
	return n
}

func (n *typeNamer) unique(base string) string {
	name := base
	for i := 2; n.used[name]; i++ {
		name = base + strconv.Itoa(i)
	}
	n.used[name] = true
	return name
}

// declare walks t depth-first and names every object type in it, the root
// as name and nested ones as name + field.
func (n *typeNamer) declare(t *infer.Type, name, doc string) {
	if t == nil {
		return
	}
	switch t.Kind {
	case infer.Object:
		for _, f := range t.Fields {
			if !n.fieldOK(f.Name) {
				t.Kind, t.Elem, t.Fields = infer.Map, &infer.Type{}, nil
				t.Note = "a key cannot be a field name in this language"
				return
			}
		}
		if _, done := n.names[t]; done {
			return
		}
		typeName := n.unique(name)
		n.names[t] = typeName
		n.decls = append(n.decls, objDecl{Name: typeName, Type: t, Doc: doc})
		for _, f := range t.Fields {
			n.declare(f.Type, typeName+pascal(f.Name), typeName+"."+f.Name)
		}
	case infer.Array:
		n.declare(t.Elem, name+"Item", doc+"[]")
	case infer.Map:
		n.declare(t.Elem, name+"Value", doc+"{}")
	}
}

// dependencyOrder returns the declarations with every object type before
// the types that use it (for languages that resolve names eagerly).
func (n *typeNamer) dependencyOrder() []objDecl {
	byType := make(map[*infer.Type]objDecl, len(n.decls))
	for _, d := range n.decls {
		byType[d.Type] = d
	}
	done := map[*infer.Type]bool{}
	var out []objDecl
	var visit func(t *infer.Type)
	visit = func(t *infer.Type) {
		if t == nil {
			return
		}
		switch t.Kind {
		case infer.Array, infer.Map:
			visit(t.Elem)
		case infer.Object:
			d, declared := byType[t]
			if !declared || done[t] {
				return
			}
			done[t] = true
			for _, f := range t.Fields {
				visit(f.Type)
			}
			out = append(out, d)
		}
	}
	for _, d := range n.decls {
		visit(d.Type)
	}
	return out
}

// envelopeType is the envelope as a model type around payload.
func envelopeType(payload *infer.Type) *infer.Type {
	return &infer.Type{Kind: infer.Object, Fields: []infer.Field{
		{Name: "attempt", Type: &infer.Type{Kind: infer.Int}, Required: true},
		{Name: "event_id", Type: &infer.Type{Kind: infer.String}, Required: true},
		{Name: "event_name", Type: &infer.Type{Kind: infer.String}, Required: true},
		{Name: "payload", Type: payload, Required: true},
		{Name: "timestamp", Type: &infer.Type{Kind: infer.String}, Required: true},
		{Name: "version", Type: &infer.Type{Kind: infer.String}, Required: true},
	}}
}

// placeholder is a value of type t, for a sample built from a schema.
func placeholder(t *infer.Type, depth int) any {
	if t == nil || depth > 16 {
		return nil
	}
	switch t.Kind {
	case infer.String:
		return "string"
	case infer.Int:
		return 1
	case infer.Float:
		return 1.5
	case infer.Bool:
		return true
	case infer.Array:
		return []any{placeholder(t.Elem, depth+1)}
	case infer.Map:
		return map[string]any{}
	case infer.Object:
		m := make(map[string]any, len(t.Fields))
		for _, f := range t.Fields {
			m[f.Name] = placeholder(f.Type, depth+1)
		}
		return m
	}
	return nil
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
