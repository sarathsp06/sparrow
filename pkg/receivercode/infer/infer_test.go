package infer

import (
	"strings"
	"testing"
)

// shape renders a type compactly for comparisons: object fields as
// name[?]:shape, nullable as a trailing "|null".
func shape(t *Type) string {
	if t == nil {
		return "<nil>"
	}
	var s string
	switch t.Kind {
	case Object:
		parts := make([]string, 0, len(t.Fields))
		for _, f := range t.Fields {
			opt := ""
			if !f.Required {
				opt = "?"
			}
			parts = append(parts, f.Name+opt+":"+shape(f.Type))
		}
		s = "{" + strings.Join(parts, ",") + "}"
	case Array:
		s = "[" + shape(t.Elem) + "]"
	case Map:
		s = "map<" + shape(t.Elem) + ">"
	default:
		s = t.Kind.String()
	}
	if t.Nullable {
		s += "|null"
	}
	return s
}

func TestFromJSON(t *testing.T) {
	cases := map[string]string{
		`{"a":1,"b":1.5,"c":"x","d":true,"e":null}`: "{a:int,b:float,c:string,d:bool,e:any|null}",
		`{"big":12345678901234567890,"exp":1e3}`:    "{big:float,exp:float}",
		`{}`:                                        "map<any>",
		`[]`:                                        "[any]",
		`[1, 2.5]`:                                  "[float]",
		`[1, "x"]`:                                  "[any]",
		`[null, "x"]`:                               "[string|null]",
		`["x", null]`:                               "[string|null]",
		`[{"a":1},{"a":2,"b":"x"}]`:                 "[{a:int,b?:string}]",
		`[{"a":1},{"b":"x"}]`:                       "[{a?:int,b?:string}]",
		`[[1],[2.5]]`:                               "[[float]]",
		`[{},{}]`:                                   "[map<any>]",
		`"s"`:                                       "string",
	}
	for in, want := range cases {
		got, err := FromJSON([]byte(in))
		if err != nil {
			t.Errorf("FromJSON(%s): %v", in, err)
			continue
		}
		if shape(got) != want {
			t.Errorf("FromJSON(%s) = %s, want %s", in, shape(got), want)
		}
	}
	for _, bad := range []string{``, `{`, `{} {}`, `nul`} {
		if _, err := FromJSON([]byte(bad)); err == nil {
			t.Errorf("FromJSON(%q) succeeded, want error", bad)
		}
	}
	deep := strings.Repeat("[", 40) + strings.Repeat("]", 40)
	got, err := FromJSON([]byte(deep))
	if err != nil || !strings.Contains(shape(got), "any") {
		t.Errorf("deep nesting = %s, %v; want Any below the limit", shape(got), err)
	}
}

func TestFromSchema(t *testing.T) {
	cases := map[string]string{
		`{"type":"object","required":["a"],"properties":{"a":{"type":"string"},"b":{"type":"integer"}}}`: "{a:string,b?:int}",
		`{"properties":{"a":{"type":"number"}}}`:                                                         "{a?:float}",
		`{"type":"object"}`:                                                                              "map<any>",
		`{"type":"object","additionalProperties":{"type":"boolean"}}`:                                    "map<bool>",
		`{"additionalProperties":{"type":"string"}}`:                                                     "map<string>",
		`{"type":"array","items":{"type":"string"}}`:                                                     "[string]",
		`{"items":{"type":"string"}}`:                                                                    "[string]",
		`{"type":"array"}`:                                                                               "[any]",
		`{"type":"array","items":[{"type":"string"}]}`:                                                   "[any]",
		`{"type":["string","null"]}`:                                                                     "string|null",
		`{"type":["string","integer"]}`:                                                                  "any",
		`{"type":"null"}`:                                                                                "any|null",
		`{"type":["null"]}`:                                                                              "any|null",
		`{"type":"string","nullable":true}`:                                                              "string|null",
		`{"type":"weird"}`:                                                                               "any",
		`{}`:                                                                                             "any",
		`{"enum":["a","b"]}`:                                                                             "string",
		`{"enum":[1,"b"]}`:                                                                               "any",
		`{"const":3}`:                                                                                    "int",
		`{"anyOf":[{"type":"string"},{"type":"null"}]}`:                                                  "string|null",
		`{"oneOf":[{"type":"string"},{"type":"integer"}]}`:                                               "any",
		`{"oneOf":[{"type":"object"},{"type":"object"}]}`:                                                "any",
		`{"oneOf":[{"type":"null"}]}`:                                                                    "any|null",
		`{"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}}}]}`: "{a:string,b?:int}",
		`{"allOf":[{"properties":{"a":{"type":"string"}}},{"properties":{"a":{"type":"string"}},"required":["a"]}]}`:  "{a:string}",
		`{"allOf":[{"type":"string"}]}`: "any",
		`{"allOf":[]}`:                  "any",
		`{"$ref":"#/$defs/x","$defs":{"x":{"type":"integer"}}}`:                                           "int",
		`{"$ref":"#/definitions/a~1b","definitions":{"a/b":{"type":"boolean"}}}`:                          "bool",
		`{"$ref":"#/$defs/missing"}`:                                                                      "any",
		`{"$ref":"#/$defs/x/deeper","$defs":{"x":"notamap"}}`:                                             "any",
		`{"$ref":"https://example.com/s.json"}`:                                                           "any",
		`{"type":"object","properties":{"self":{"$ref":"#"}}}`:                                            "{self?:any}",
		`{"$defs":{"n":{"type":"object","properties":{"next":{"$ref":"#/$defs/n"}}}},"$ref":"#/$defs/n"}`: "{next?:any}",
	}
	for in, want := range cases {
		got, err := FromSchema([]byte(in))
		if err != nil {
			t.Errorf("FromSchema(%s): %v", in, err)
			continue
		}
		if shape(got) != want {
			t.Errorf("FromSchema(%s) = %s, want %s", in, shape(got), want)
		}
	}
	if got, err := FromSchema([]byte(`true`)); err != nil || got.Kind != Any {
		t.Errorf("FromSchema(true) = %v, %v", got, err)
	}
	for _, bad := range []string{`false`, `"x"`, `[`} {
		if _, err := FromSchema([]byte(bad)); err == nil {
			t.Errorf("FromSchema(%s) succeeded, want error", bad)
		}
	}
	notes, _ := FromSchema([]byte(`{"type":["string","integer"]}`))
	if !strings.Contains(notes.Note, "several types") {
		t.Errorf("note = %q", notes.Note)
	}
}

func TestFromXML(t *testing.T) {
	el, err := FromXML([]byte(`<?xml version="1.0"?>
<order id="1" xmlns="urn:x" xmlns:p="urn:p">
  <item sku="a"><qty>1</qty></item>
  <item sku="b" extra="y"><qty>2</qty></item>
  <note>hi</note>
  <p:tag/>
</order>`))
	if err != nil {
		t.Fatal(err)
	}
	if el.Name != "order" || strings.Join(el.Attrs, ",") != "id" || len(el.Children) != 3 {
		t.Fatalf("root = %+v", el)
	}
	item := el.Children[0]
	if item.Name != "item" || !item.Repeated || strings.Join(item.Attrs, ",") != "extra,sku" {
		t.Errorf("item = %+v", item)
	}
	if qty := item.Children[0]; qty.Repeated || !qty.Text {
		t.Errorf("qty is one per item: %+v", qty)
	}
	if note := el.Children[1]; note.Name != "note" || !note.Text || note.Repeated {
		t.Errorf("note = %+v", note)
	}
	if tag := el.Children[2]; tag.Name != "tag" {
		t.Errorf("namespaced child = %+v", tag)
	}
	for _, bad := range []string{``, `<a>`, `<a/><b/>`, `<a></b>`, `text`, strings.Repeat("<a>", 40) + strings.Repeat("</a>", 40)} {
		if _, err := FromXML([]byte(bad)); err == nil {
			t.Errorf("FromXML(%q) succeeded, want error", bad)
		}
	}
}

func TestKindString(t *testing.T) {
	for k, want := range map[Kind]string{Any: "any", String: "string", Int: "int", Float: "float", Bool: "bool", Object: "object", Array: "array", Map: "map"} {
		if k.String() != want {
			t.Errorf("%d.String() = %s", k, k.String())
		}
	}
	if !(*Type)(nil).Untyped() || (&Type{Kind: Int}).Untyped() {
		t.Error("Untyped")
	}
}
