package infer

import (
	"fmt"
	"strings"
)

// FromSchema infers a type from a JSON Schema. It understands type,
// properties, required, items, additionalProperties, enum, const, local
// $refs and simple anyOf/oneOf/allOf; anything else becomes Any with a Note.
func FromSchema(schema []byte) (*Type, error) {
	v, err := decodeJSON(schema)
	if err != nil {
		return nil, err
	}
	root, ok := v.(map[string]any)
	if !ok {
		if b, isBool := v.(bool); isBool && b {
			return anyType("schema accepts any value"), nil
		}
		return nil, fmt.Errorf("infer: schema must be a JSON object")
	}
	s := &schemaWalker{root: root, visiting: map[string]bool{}}
	return s.walk(root, 0), nil
}

type schemaWalker struct {
	root     map[string]any
	visiting map[string]bool
}

func (s *schemaWalker) walk(node any, depth int) *Type {
	if depth > maxDepth {
		return anyType("schema nested too deeply to type")
	}
	m, ok := node.(map[string]any)
	if !ok {
		return anyType("")
	}

	if ref, ok := m["$ref"].(string); ok {
		return s.ref(ref, depth)
	}
	if c, ok := m["const"]; ok {
		return fromValue(c, depth)
	}
	if enum, ok := m["enum"].([]any); ok && len(enum) > 0 {
		var t *Type
		for _, e := range enum {
			t = merge(t, fromValue(e, depth))
		}
		return t
	}
	if all, ok := m["allOf"].([]any); ok {
		return s.allOf(all, depth)
	}
	for _, key := range []string{"anyOf", "oneOf"} {
		if branches, ok := m[key].([]any); ok {
			return s.union(key, branches, depth)
		}
	}

	types, nullable := schemaTypes(m)
	if len(types) > 1 {
		t := anyType("schema allows several types (" + strings.Join(types, ", ") + ")")
		t.Nullable = nullable
		return t
	}
	var t *Type
	switch first(types) {
	case "string":
		t = &Type{Kind: String}
	case "integer":
		t = &Type{Kind: Int}
	case "number":
		t = &Type{Kind: Float}
	case "boolean":
		t = &Type{Kind: Bool}
	case "null":
		t = anyType(noteNull)
		nullable = true
	case "array":
		t = &Type{Kind: Array, Elem: s.items(m, depth)}
	case "object":
		t = s.object(m, depth)
	case "":
		switch {
		case m["properties"] != nil || m["additionalProperties"] != nil:
			t = s.object(m, depth)
		case m["items"] != nil:
			t = &Type{Kind: Array, Elem: s.items(m, depth)}
		default:
			t = anyType("schema does not restrict the type")
		}
	default:
		t = anyType("unknown schema type " + first(types))
	}
	t.Nullable = t.Nullable || nullable
	return t
}

// schemaTypes returns the non-null types a schema allows and whether null
// is one of them.
func schemaTypes(m map[string]any) (types []string, nullable bool) {
	switch v := m["type"].(type) {
	case string:
		if v == "null" {
			return []string{v}, true
		}
		types = []string{v}
	case []any:
		for _, e := range v {
			name, _ := e.(string)
			if name == "null" {
				nullable = true
				continue
			}
			types = append(types, name)
		}
		if len(types) == 0 && nullable {
			return []string{"null"}, true
		}
	}
	if n, ok := m["nullable"].(bool); ok && n {
		nullable = true
	}
	return types, nullable
}

func first(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func (s *schemaWalker) items(m map[string]any, depth int) *Type {
	switch items := m["items"].(type) {
	case map[string]any:
		return s.walk(items, depth+1)
	case []any:
		return anyType("tuple arrays are not typed")
	}
	return anyType("array items are not described")
}

func (s *schemaWalker) object(m map[string]any, depth int) *Type {
	props, _ := m["properties"].(map[string]any)
	if len(props) == 0 {
		if ap, ok := m["additionalProperties"].(map[string]any); ok {
			return &Type{Kind: Map, Elem: s.walk(ap, depth+1)}
		}
		return &Type{Kind: Map, Elem: anyType(""), Note: "object without described properties"}
	}
	required := map[string]bool{}
	if req, ok := m["required"].([]any); ok {
		for _, r := range req {
			if name, ok := r.(string); ok {
				required[name] = true
			}
		}
	}
	fields := make([]Field, 0, len(props))
	for name, p := range props {
		fields = append(fields, Field{Name: name, Type: s.walk(p, depth+1), Required: required[name]})
	}
	sortFields(fields)
	return &Type{Kind: Object, Fields: fields}
}

// ref follows a local reference ("#/$defs/x", "#/definitions/x"). Remote
// and recursive references become Any.
func (s *schemaWalker) ref(ref string, depth int) *Type {
	path, ok := strings.CutPrefix(ref, "#/")
	if !ok {
		if ref == "#" {
			return anyType("recursive schema reference")
		}
		return anyType("external schema reference " + ref + " is not followed")
	}
	if s.visiting[ref] {
		return anyType("recursive schema reference " + ref)
	}
	var node any = s.root
	for _, part := range strings.Split(path, "/") {
		part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		m, ok := node.(map[string]any)
		if !ok {
			return anyType("unresolved schema reference " + ref)
		}
		if node, ok = m[part]; !ok {
			return anyType("unresolved schema reference " + ref)
		}
	}
	s.visiting[ref] = true
	defer delete(s.visiting, ref)
	return s.walk(node, depth+1)
}

// allOf merges object branches into one object; any other combination is
// Any.
func (s *schemaWalker) allOf(branches []any, depth int) *Type {
	var fields []Field
	seen := map[string]int{}
	for _, b := range branches {
		t := s.walk(b, depth+1)
		if t.Kind != Object {
			return anyType("allOf over non-object schemas is not typed")
		}
		for _, f := range t.Fields {
			if i, ok := seen[f.Name]; ok {
				fields[i].Required = fields[i].Required || f.Required
				continue
			}
			seen[f.Name] = len(fields)
			fields = append(fields, f)
		}
	}
	if len(fields) == 0 {
		return anyType("empty allOf")
	}
	sortFields(fields)
	return &Type{Kind: Object, Fields: fields}
}

// union types anyOf/oneOf only when every non-null branch is the same
// scalar kind; null branches make it nullable.
func (s *schemaWalker) union(key string, branches []any, depth int) *Type {
	var out *Type
	nullable := false
	for _, b := range branches {
		t := s.walk(b, depth+1)
		if t.Kind == Any && t.Note == noteNull {
			nullable = true
			continue
		}
		switch {
		case out == nil:
			out = t
		case out.Kind != t.Kind || out.Kind == Object || out.Kind == Array || out.Kind == Map:
			u := anyType(key + " with different shapes is not typed")
			u.Nullable = nullable
			return u
		}
	}
	if out == nil {
		out = anyType(noteNull)
	}
	c := *out
	c.Nullable = c.Nullable || nullable
	return &c
}
