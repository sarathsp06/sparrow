// Package infer turns a JSON Schema, a JSON sample or an XML sample into a
// small, language-neutral model that the receiver targets render as types.
//
// Inference is best effort and never fails on a shape it cannot represent:
// such a value becomes Any (an untyped map, dict or unknown) with a Note
// saying why. Only input that is not JSON or XML at all is an error.
package infer

import (
	"sort"
)

// Kind is the shape of a value.
type Kind int

// Kinds, in no particular order. Any is the zero value: an untyped value.
const (
	Any Kind = iota
	String
	Int
	Float
	Bool
	Object // fixed fields
	Array  // Elem is the element type
	Map    // string keys, Elem is the value type
)

// String returns the kind's name, for tests and notes.
func (k Kind) String() string {
	switch k {
	case String:
		return "string"
	case Int:
		return "int"
	case Float:
		return "float"
	case Bool:
		return "bool"
	case Object:
		return "object"
	case Array:
		return "array"
	case Map:
		return "map"
	default:
		return "any"
	}
}

// Type is an inferred value type.
type Type struct {
	Kind Kind
	// Fields are an Object's fields, sorted by Name.
	Fields []Field
	// Elem is the element type of an Array or the value type of a Map.
	Elem *Type
	// Nullable is set when the value may be null.
	Nullable bool
	// Note explains why a value is Any or a Map, for a code comment.
	Note string
}

// Field is one field of an Object.
type Field struct {
	// Name is the JSON key.
	Name string
	Type *Type
	// Required is false when the key may be absent.
	Required bool
}

// maxDepth bounds nesting so a hostile schema or sample cannot exhaust the
// stack; anything deeper becomes Any.
const maxDepth = 32

func anyType(note string) *Type { return &Type{Kind: Any, Note: note} }

func sortFields(fields []Field) {
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
}

// Untyped reports whether t carries no type information at all.
func (t *Type) Untyped() bool { return t == nil || t.Kind == Any }

// merge combines two observations of the same value (array elements). The
// result accepts both: differing kinds become Any, Int and Float become
// Float, and object fields missing on one side become optional.
func merge(a, b *Type) *Type {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	nullable := a.Nullable || b.Nullable
	switch {
	case a.Kind == Any && a.Note == noteNull:
		c := *b
		c.Nullable = true
		return &c
	case b.Kind == Any && b.Note == noteNull:
		c := *a
		c.Nullable = true
		return &c
	case a.Kind != b.Kind:
		if (a.Kind == Int && b.Kind == Float) || (a.Kind == Float && b.Kind == Int) {
			return &Type{Kind: Float, Nullable: nullable}
		}
		t := anyType("values of different types in the sample")
		t.Nullable = nullable
		return t
	}
	switch a.Kind {
	case Array, Map:
		return &Type{Kind: a.Kind, Elem: merge(a.Elem, b.Elem), Nullable: nullable, Note: a.Note}
	case Object:
		return &Type{Kind: Object, Fields: mergeFields(a.Fields, b.Fields), Nullable: nullable}
	}
	c := *a
	c.Nullable = nullable
	return &c
}

func mergeFields(a, b []Field) []Field {
	byName := make(map[string]Field, len(a)+len(b))
	inA := make(map[string]bool, len(a))
	for _, f := range a {
		byName[f.Name] = f
		inA[f.Name] = true
	}
	inB := make(map[string]bool, len(b))
	for _, f := range b {
		inB[f.Name] = true
		if prev, ok := byName[f.Name]; ok {
			byName[f.Name] = Field{Name: f.Name, Type: merge(prev.Type, f.Type), Required: prev.Required && f.Required}
			continue
		}
		byName[f.Name] = f
	}
	out := make([]Field, 0, len(byName))
	for name, f := range byName {
		if !inA[name] || !inB[name] {
			f.Required = false
		}
		out = append(out, f)
	}
	sortFields(out)
	return out
}
