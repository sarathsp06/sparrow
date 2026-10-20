package receivercode

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"

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

// conforms reports whether a JSON value decodes into t without error in
// every target: nulls only where nullable, integers without a fraction,
// required keys present.
func conforms(v any, t *infer.Type) bool {
	if t == nil || t.Kind == infer.Any {
		return true
	}
	if v == nil {
		return t.Nullable
	}
	switch t.Kind {
	case infer.String:
		_, ok := v.(string)
		return ok
	case infer.Bool:
		_, ok := v.(bool)
		return ok
	case infer.Int:
		n, ok := v.(json.Number)
		if !ok || strings.ContainsAny(string(n), ".eE") {
			return false
		}
		_, err := strconv.ParseInt(string(n), 10, 64)
		return err == nil
	case infer.Float:
		_, ok := v.(json.Number)
		return ok
	case infer.Array:
		arr, ok := v.([]any)
		if !ok {
			return false
		}
		for _, e := range arr {
			if !conforms(e, t.Elem) {
				return false
			}
		}
		return true
	case infer.Map:
		m, ok := v.(map[string]any)
		if !ok {
			return false
		}
		for _, e := range m {
			if !conforms(e, t.Elem) {
				return false
			}
		}
		return true
	case infer.Object:
		m, ok := v.(map[string]any)
		if !ok {
			return false
		}
		for _, f := range t.Fields {
			val, present := m[f.Name]
			if !present {
				if f.Required {
					return false
				}
				continue
			}
			if !conforms(val, f.Type) {
				return false
			}
		}
		return true
	}
	return false
}

// project keeps the parts of v that a typed decode of t preserves: fields
// t declares and non-null values. Its result is what a generated handler
// logs at least, so tests compare it as a subset.
func project(v any, t *infer.Type) any {
	if t == nil || t.Kind == infer.Any || v == nil {
		return v
	}
	switch t.Kind {
	case infer.Object:
		m, _ := v.(map[string]any)
		out := map[string]any{}
		for _, f := range t.Fields {
			if val, ok := m[f.Name]; ok && val != nil {
				out[f.Name] = project(val, f.Type)
			}
		}
		return out
	case infer.Map:
		m, _ := v.(map[string]any)
		out := map[string]any{}
		for k, val := range m {
			if val != nil {
				out[k] = project(val, t.Elem)
			}
		}
		return out
	case infer.Array:
		arr, _ := v.([]any)
		out := make([]any, 0, len(arr))
		for _, e := range arr {
			out = append(out, project(e, t.Elem))
		}
		return out
	}
	return v
}

// decodeNumbers decodes JSON keeping numbers as json.Number.
func decodeNumbers(b []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	err := dec.Decode(&v)
	return v, err
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

// bodyType is the model type of p's whole body (the envelope or the
// transformed JSON), or nil for xml and raw.
func (p *plan) bodyType() *infer.Type {
	switch p.Model {
	case ModelEnvelope:
		return envelopeType(p.Type)
	case ModelJSON:
		return p.Type
	}
	return nil
}

// settleModel makes sure the sample body decodes into the model: a schema
// type the sample does not satisfy is replaced by the sample's own type,
// and the expected handler value is the projected sample.
func (p *plan) settleModel() {
	t := p.bodyType()
	if t == nil {
		return
	}
	v, err := decodeNumbers(p.Body)
	if err != nil {
		p.Model, p.Type, p.ModelSource = ModelRaw, nil, ""
		return
	}
	if conforms(v, t) {
		return
	}
	if p.Model == ModelEnvelope {
		env, _ := v.(map[string]any)
		if st, err := infer.FromJSON(mustJSON(env["payload"])); err == nil && st.Kind == infer.Object {
			p.Type, p.ModelSource = st, "sample"
			p.Notes = append(p.Notes, "the sample payload does not match the JSON Schema, so the payload type is inferred from the sample: best effort")
			if conforms(v, p.bodyType()) {
				return
			}
		}
	}
	p.Model, p.Type, p.ModelSource = ModelRaw, nil, ""
	p.Notes = append(p.Notes, "the sample does not fit any inferred model, so the body is handled as raw text")
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// expectedValue is what a handler for p logs for its sample body (at
// least, for the subset match).
func expectedValue(p *plan) (json.RawMessage, string) {
	if t := p.bodyType(); t != nil {
		var v any
		if err := json.Unmarshal(p.Body, &v); err == nil {
			return mustJSON(project(v, t)), "subset"
		}
	}
	return mustJSON(string(p.Body)), "equal"
}
