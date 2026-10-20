package receivercode

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/sarathsp06/sparrow/pkg/receivercode/infer"
)

// Pinned versions of the Python target's dependencies.
const (
	pyFastAPI      = "0.150.0"
	pyUvicorn      = "0.54.0"
	pyPydantic     = "2.14.1"
	pyCryptography = "50.0.2"
	pyPytest       = "9.1.1"
	pyHTTPX        = "2.15.1"
)

var pythonTarget = targetDef{
	info: TargetInfo{
		Target:  Target{Language: "python", Framework: "fastapi"},
		ID:      "python-fastapi",
		Name:    "Python (FastAPI)",
		Runtime: "Python 3.10 or later",
	},
	run: func(g *gen) string {
		return g.envPrefix() + "python receiver.py"
	},
	test:  "pytest",
	build: buildPython,
}

var pyKeywords = map[string]bool{}

func init() {
	for _, k := range strings.Fields(`False None True and as assert async await break class continue def del elif
		else except finally for from global if import in is lambda nonlocal not or pass raise return try while
		with yield`) {
		pyKeywords[k] = true
	}
	// BaseModel attributes a field must not shadow.
	for _, k := range strings.Fields(`construct copy dict fields from_orm json parse_file parse_obj parse_raw
		schema schema_json update_forward_refs validate`) {
		pyKeywords[k] = true
	}
}

type pySub struct {
	ID, EventType, Name, Handler, Func, Model, ModelType, Adapter, Format string
	Notes                                                                 []string
}

func buildPython(g *gen) (map[string]any, []File, error) {
	namer := newTypeNamer(func(string) bool { return true }, "BaseModel", "Field", "Any")
	var subs []pySub
	var aliases []string
	for _, p := range g.plans {
		s := pySub{ID: p.Sub.ID, EventType: p.Sub.EventType, Name: p.Name, Handler: handlerName(p),
			Func: "handle_" + snake(p.Name), Model: p.Model, Format: p.Format, Notes: p.Notes,
			Adapter: "_" + strings.ToUpper(snake(p.Name))}
		switch p.Model {
		case ModelEnvelope:
			namer.declare(p.Type, p.Name+"Payload", p.Sub.EventType+" payload")
			env := envelopeType(p.Type)
			namer.declare(env, p.Name+"Event", p.Sub.EventType+" envelope")
			s.ModelType = "models." + namer.names[env]
		case ModelJSON:
			namer.declare(p.Type, p.Name+"Body", p.Sub.EventType+" body")
			if p.Type.Kind == infer.Object {
				s.ModelType = "models." + namer.names[p.Type]
			} else {
				alias := namer.unique(p.Name + "Body")
				aliases = append(aliases, fmt.Sprintf("%s = %s", alias, pyTypeExpr(p.Type, namer)))
				s.ModelType = "models." + alias
			}
		case ModelXML:
			s.ModelType = "ET.Element"
		default:
			s.ModelType = "str"
		}
		subs = append(subs, s)
	}

	data := g.commonData()
	data["Subs"] = subs
	data["HasXML"], data["HasModels"] = false, false
	for _, s := range subs {
		data["HasXML"] = data["HasXML"].(bool) || s.Model == ModelXML
		data["HasModels"] = data["HasModels"].(bool) || strings.HasPrefix(s.ModelType, "models.")
	}
	data["Setup"] = []string{"python -m venv .venv && . .venv/bin/activate", "pip install -r requirements.txt"}
	data["ToleranceName"] = "TOLERANCE_SECONDS"
	data["DedupName"] = "DEDUP_CAPACITY"
	data["HelperPath"] = "sparrow_verify.py"
	data["Pins"] = map[string]string{
		"FastAPI": pyFastAPI, "Uvicorn": pyUvicorn, "Pydantic": pyPydantic,
		"Cryptography": pyCryptography, "Pytest": pyPytest, "HTTPX2": pyHTTPX,
	}
	files := []File{
		{Path: "models.py", Content: pyModels(g, namer, aliases), Mode: 0o644},
		{Path: "sparrow_verify.py", Content: pythonHelper, Mode: 0o644},
	}
	return data, files, nil
}

// snake turns a PascalCase or mixed key into snake_case.
func snake(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		switch {
		case unicode.IsUpper(r):
			if i > 0 && (unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1]) ||
				i+1 < len(runes) && unicode.IsLower(runes[i+1]) && unicode.IsUpper(runes[i-1])) {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	return out
}

// pyIdent makes a unique, safe pydantic field name for a key.
func pyIdent(key string, used map[string]bool) string {
	base := snake(key)
	switch {
	case base == "":
		base = "field"
	case unicode.IsDigit(rune(base[0])) || strings.HasPrefix(base, "model_"):
		base = "f_" + base
	case pyKeywords[base]:
		base += "_"
	}
	name := base
	for i := 2; used[name]; i++ {
		name = base + "_" + strconv.Itoa(i)
	}
	used[name] = true
	return name
}

func pyTypeExpr(t *infer.Type, n *typeNamer) string {
	var expr string
	switch t.Kind {
	case infer.String:
		expr = "str"
	case infer.Int:
		expr = "int"
	case infer.Float:
		expr = "float"
	case infer.Bool:
		expr = "bool"
	case infer.Object:
		expr = n.names[t]
	case infer.Array:
		expr = "list[" + pyTypeExpr(t.Elem, n) + "]"
	case infer.Map:
		expr = "dict[str, " + pyTypeExpr(t.Elem, n) + "]"
	default:
		return "Any"
	}
	if t.Nullable {
		expr += " | None"
	}
	return expr
}

func pyModels(g *gen, namer *typeNamer, aliases []string) string {
	var b strings.Builder
	b.WriteString("# " + g.header() + "\n")
	b.WriteString(`"""Models of the delivery bodies.

They are inferred from each event type's JSON Schema or from a rendered sample:
best effort. Check them against your event types and adjust freely. Unknown
keys are ignored.
"""

from typing import Any

from pydantic import BaseModel, Field
`)
	// Children are declared before their parents, so every annotation
	// resolves when its class is created.
	for _, d := range namer.dependencyOrder() {
		fmt.Fprintf(&b, "\n\nclass %s(BaseModel):\n    \"\"\"The %s.\"\"\"\n\n", d.Name, commentText(d.Doc))
		used := map[string]bool{}
		for _, f := range d.Type.Fields {
			if note := fieldNote(f.Type); note != "" {
				fmt.Fprintf(&b, "    # %s\n", commentText(note))
			}
			ident := pyIdent(f.Name, used)
			typ := pyTypeExpr(f.Type, namer)
			optional := !f.Required
			if optional && !strings.HasSuffix(typ, " | None") && typ != "Any" {
				typ += " | None"
			}
			var def []string
			if optional {
				def = append(def, "default=None")
			}
			if ident != f.Name {
				def = append(def, "alias="+pyString(f.Name))
			}
			switch {
			case len(def) == 1 && def[0] == "default=None":
				fmt.Fprintf(&b, "    %s: %s = None\n", ident, typ)
			case len(def) > 0:
				fmt.Fprintf(&b, "    %s: %s = Field(%s)\n", ident, typ, strings.Join(def, ", "))
			default:
				fmt.Fprintf(&b, "    %s: %s\n", ident, typ)
			}
		}
		if len(d.Type.Fields) == 0 {
			b.WriteString("    pass\n")
		}
	}
	if len(aliases) > 0 {
		b.WriteString("\n")
		for _, a := range aliases {
			b.WriteString("\n" + a + "\n")
		}
	}
	return b.String()
}

// pyString is a Python string literal (JSON escapes are valid Python).
func pyString(s string) string {
	q := strconv.Quote(s)
	return q
}
