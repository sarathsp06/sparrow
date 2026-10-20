package receivercode

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sarathsp06/sparrow/pkg/receivercode/infer"
)

// Pinned versions of the Node target's dependencies.
const (
	nodeExpress      = "5.3.0"
	nodeTypesExpress = "5.0.6"
	nodeTypesNode    = "22.20.5"
	nodeTypeScript   = "7.0.2"
	// nodeEngine is the first Node release that strips TypeScript types
	// without a flag (also in 23.6+).
	nodeEngine = ">=22.18.0"
)

var nodeTarget = targetDef{
	info: TargetInfo{
		Target:  Target{Language: "node", Framework: "express"},
		ID:      "node-express",
		Name:    "Node.js (Express, TypeScript)",
		Runtime: "Node.js 22.18 or later (runs TypeScript directly)",
	},
	run: func(g *gen) string {
		return g.envPrefix() + "npm start"
	},
	test:  "npm test",
	build: buildNode,
}

type nodeSub struct {
	ID, EventType, Name, Handler, Func, Model, ModelType, Format string
	Notes                                                        []string
}

var tsIdentPattern = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)

func buildNode(g *gen) (map[string]any, []File, error) {
	namer := newTypeNamer(func(string) bool { return true })
	var subs []nodeSub
	var aliases []string
	for _, p := range g.plans {
		s := nodeSub{ID: p.Sub.ID, EventType: p.Sub.EventType, Name: p.Name, Handler: handlerName(p),
			Func: handlerName(p), Model: p.Model, Format: p.Format, Notes: p.Notes}
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
				aliases = append(aliases, fmt.Sprintf("export type %s = %s;", alias, tsTypeExpr(p.Type, namer)))
				s.ModelType = "models." + alias
			}
		default:
			s.ModelType = "string"
		}
		subs = append(subs, s)
	}

	data := g.commonData()
	data["Subs"] = subs
	hasModels := false
	for _, s := range subs {
		hasModels = hasModels || strings.HasPrefix(s.ModelType, "models.")
	}
	data["HasModels"] = hasModels
	data["Setup"] = []string{"npm install"}
	data["ToleranceName"] = "TOLERANCE_SECONDS"
	data["DedupName"] = "DEDUP_CAPACITY"
	data["HelperPath"] = "src/sparrow-verify.ts"
	data["Express"] = nodeExpress
	data["TypesExpress"] = nodeTypesExpress
	data["TypesNode"] = nodeTypesNode
	data["TypeScript"] = nodeTypeScript
	data["NodeEngine"] = nodeEngine
	files := []File{
		{Path: "src/models.ts", Content: tsModels(g, namer, aliases), Mode: 0o644},
		{Path: "src/sparrow-verify.ts", Content: tsHelper, Mode: 0o644},
	}
	return data, files, nil
}

func tsTypeExpr(t *infer.Type, n *typeNamer) string {
	var expr string
	switch t.Kind {
	case infer.String:
		expr = "string"
	case infer.Int, infer.Float:
		expr = "number"
	case infer.Bool:
		expr = "boolean"
	case infer.Object:
		expr = n.names[t]
	case infer.Array:
		elem := tsTypeExpr(t.Elem, n)
		if strings.Contains(elem, " ") {
			elem = "(" + elem + ")"
		}
		expr = elem + "[]"
	case infer.Map:
		expr = "Record<string, " + tsTypeExpr(t.Elem, n) + ">"
	default:
		return "unknown"
	}
	if t.Nullable {
		expr += " | null"
	}
	return expr
}

func tsKey(key string) string {
	if tsIdentPattern.MatchString(key) {
		return key
	}
	return strconv.Quote(key)
}

func tsModels(g *gen, namer *typeNamer, aliases []string) string {
	var b strings.Builder
	b.WriteString("// " + g.header() + "\n\n")
	b.WriteString("// Models of the delivery bodies, inferred from each event type's JSON Schema\n// or from a rendered sample: best effort. They are types only: bodies are not\n// validated at runtime. Check them against your event types and adjust freely.\n")
	for _, d := range namer.decls {
		fmt.Fprintf(&b, "\n/** The %s. */\nexport interface %s {\n", commentText(d.Doc), d.Name)
		for _, f := range d.Type.Fields {
			if note := fieldNote(f.Type); note != "" {
				fmt.Fprintf(&b, "  // %s\n", commentText(note))
			}
			opt := ""
			if !f.Required {
				opt = "?"
			}
			fmt.Fprintf(&b, "  %s%s: %s;\n", tsKey(f.Name), opt, tsTypeExpr(f.Type, namer))
		}
		b.WriteString("}\n")
	}
	for _, a := range aliases {
		b.WriteString("\n" + a + "\n")
	}
	if len(namer.decls) == 0 && len(aliases) == 0 {
		b.WriteString("\nexport {};\n")
	}
	return b.String()
}

// commentText makes s safe inside a // or /* */ comment and a Python
// docstring: one line, no comment terminator, no quotes or backslashes.
func commentText(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r < 0x20 || r == 0x7f:
			return ' '
		case r == '\\' || r == '"':
			return '\''
		}
		return r
	}, s)
	return strings.ReplaceAll(s, "*/", "* /")
}
