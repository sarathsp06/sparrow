package receivercode

import (
	"fmt"
	"go/format"
	"strconv"
	"strings"
	"unicode"

	"github.com/sarathsp06/sparrow/pkg/receivercode/infer"
)

var goTarget = targetDef{
	info: TargetInfo{
		Target:  Target{Language: "go", Framework: "nethttp"},
		ID:      "go-nethttp",
		Name:    "Go (net/http)",
		Runtime: "Go 1.22 or later",
	},
	run: func(g *gen) string {
		return g.envPrefix() + "go run ."
	},
	test:  "go test ./...",
	build: buildGo,
}

// goReserved are names the Go templates declare in package main.
var goReserved = []string{"Config", "Receiver", "DecodeError", "NewReceiver", "ConfigFromEnv", "Tolerance",
	"DedupCapacity", "MaxBodyBytes", "SigningPublicKey", "UnknownEventStatus", "main"}

type goSub struct {
	ID, EventType, Name, Handler, Func, Model, ModelType, Format string
	Notes                                                        []string
}

func buildGo(g *gen) (map[string]any, []File, error) {
	namer := newTypeNamer(goFieldOK, goReserved...)
	var subs []goSub
	xmlRoots := map[*plan]string{}
	usesXML := false
	for _, p := range g.plans {
		s := goSub{ID: p.Sub.ID, EventType: p.Sub.EventType, Name: p.Name, Handler: handlerName(p),
			Func: "Handle" + p.Name, Model: p.Model, Format: p.Format, Notes: p.Notes}
		switch p.Model {
		case ModelEnvelope:
			payload := p.Type
			namer.declare(payload, p.Name+"Payload", p.Sub.EventType+" payload")
			env := envelopeType(payload)
			namer.declare(env, p.Name+"Event", p.Sub.EventType+" envelope")
			s.ModelType = namer.names[env]
		case ModelJSON:
			// declare may turn an object Go cannot express into a map, so
			// the type expression is taken afterwards.
			namer.declare(p.Type, p.Name+"Body", p.Sub.EventType+" body")
			s.ModelType = goTypeExpr(p.Type, namer)
		case ModelXML:
			usesXML = true
			s.ModelType = namer.unique(p.Name + "XML")
			xmlRoots[p] = s.ModelType
		default:
			s.ModelType = "string"
		}
		subs = append(subs, s)
	}

	models, err := goModels(g, namer, xmlRoots)
	if err != nil {
		return nil, nil, err
	}
	data := g.commonData()
	data["Subs"] = subs
	data["UsesXML"] = usesXML
	data["Setup"] = []string{}
	data["ToleranceName"] = "Tolerance"
	data["DedupName"] = "DedupCapacity"
	data["HelperPath"] = "signature/signature.go"
	files := []File{
		{Path: "models.go", Content: models, Mode: 0o644},
		{Path: "signature/signature.go", Content: goHelper, Mode: 0o644},
	}
	return data, files, nil
}

// goFieldOK reports whether key can be an encoding/json struct tag name.
func goFieldOK(key string) bool {
	if key == "" || key == "-" {
		return false
	}
	for _, c := range key {
		switch {
		case strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", c):
		case !unicode.IsLetter(c) && !unicode.IsDigit(c):
			return false
		}
	}
	return true
}

func goTypeExpr(t *infer.Type, n *typeNamer) string {
	switch t.Kind {
	case infer.String:
		return "string"
	case infer.Int:
		return "int64"
	case infer.Float:
		return "float64"
	case infer.Bool:
		return "bool"
	case infer.Object:
		return n.names[t]
	case infer.Array:
		return "[]" + goTypeExpr(t.Elem, n)
	case infer.Map:
		return "map[string]" + goTypeExpr(t.Elem, n)
	}
	return "any"
}

// goInitialisms are spelled in capitals in Go identifiers.
var goInitialisms = map[string]bool{"Api": true, "Html": true, "Http": true, "Https": true, "Id": true, "Ip": true,
	"Json": true, "Sku": true, "Sql": true, "Uri": true, "Url": true, "Uuid": true, "Xml": true}

// goPascal is pascal with Go's initialism style: event_id -> EventID.
func goPascal(key string) string {
	var b strings.Builder
	for _, part := range strings.FieldsFunc(key, func(r rune) bool {
		return r > unicode.MaxASCII || !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		p := pascal(part)
		if goInitialisms[p] {
			p = strings.ToUpper(p)
		}
		b.WriteString(p)
	}
	if b.Len() == 0 {
		return pascal(key)
	}
	out := b.String()
	if unicode.IsDigit(rune(out[0])) {
		out = "X" + out
	}
	return out
}

// goIdent makes a unique exported Go identifier for a key.
func goIdent(key string, used map[string]bool) string {
	base := goPascal(key)
	name := base
	for i := 2; used[name]; i++ {
		name = base + strconv.Itoa(i)
	}
	used[name] = true
	return name
}

func goModels(g *gen, namer *typeNamer, xmlRoots map[*plan]string) (string, error) {
	var b strings.Builder
	b.WriteString("// " + g.header() + "\n\n")
	b.WriteString("// Models are inferred from each event type's JSON Schema or from a rendered\n// sample: best effort. Check them against your event types and adjust freely.\n\n")
	b.WriteString("package main\n\n")
	if len(xmlRoots) > 0 {
		b.WriteString("import \"encoding/xml\"\n\n")
	}
	for _, d := range namer.decls {
		fmt.Fprintf(&b, "// %s is the %s.\n", d.Name, commentText(d.Doc))
		fmt.Fprintf(&b, "type %s struct {\n", d.Name)
		used := map[string]bool{}
		for _, f := range d.Type.Fields {
			if note := fieldNote(f.Type); note != "" {
				fmt.Fprintf(&b, "\t// %s\n", commentText(note))
			}
			fmt.Fprintf(&b, "\t%s %s `json:%s`\n", goIdent(f.Name, used), goTypeExpr(f.Type, namer), strconv.Quote(f.Name))
		}
		b.WriteString("}\n\n")
	}
	for _, p := range g.plans {
		if root, ok := xmlRoots[p]; ok {
			goXMLDecl(&b, p, root, namer)
		}
	}
	src, err := format.Source([]byte(b.String()))
	if err != nil {
		return "", fmt.Errorf("receivercode: generated Go models do not parse: %w", err)
	}
	return string(src), nil
}

// fieldNote is the comment for a field whose type had to be loosened.
func fieldNote(t *infer.Type) string {
	for t != nil && (t.Kind == infer.Array || t.Kind == infer.Map) && t.Note == "" {
		t = t.Elem
	}
	if t == nil || t.Note == "" {
		return ""
	}
	return "Untyped: " + t.Note + "."
}

// goXMLDecl declares the encoding/xml structs for p's element tree.
func goXMLDecl(b *strings.Builder, p *plan, rootName string, namer *typeNamer) {
	type pending struct {
		name string
		el   *infer.Element
		root bool
	}
	queue := []pending{{name: rootName, el: p.XML, root: true}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		fmt.Fprintf(b, "// %s is the <%s> element of %s deliveries.\n", cur.name, commentText(cur.el.Name), p.Sub.EventType)
		fmt.Fprintf(b, "type %s struct {\n", cur.name)
		used := map[string]bool{"XMLName": true}
		if cur.root {
			fmt.Fprintf(b, "\tXMLName xml.Name `xml:%s`\n", strconv.Quote(cur.el.Name))
		}
		for _, a := range cur.el.Attrs {
			fmt.Fprintf(b, "\t%s string `xml:%s`\n", goIdent(goPascal(a)+"Attr", used), strconv.Quote(a+",attr"))
		}
		if cur.el.Text && (len(cur.el.Children) > 0 || len(cur.el.Attrs) > 0) {
			fmt.Fprintf(b, "\t%s string `xml:\",chardata\"`\n", goIdent("Text", used))
		}
		for _, c := range cur.el.Children {
			typ := "string"
			if len(c.Children) > 0 || len(c.Attrs) > 0 {
				typ = namer.unique(cur.name + pascal(c.Name))
				queue = append(queue, pending{name: typ, el: c})
			}
			if c.Repeated {
				typ = "[]" + typ
			}
			fmt.Fprintf(b, "\t%s %s `xml:%s`\n", goIdent(c.Name, used), typ, strconv.Quote(c.Name))
		}
		b.WriteString("}\n\n")
	}
}

// formatSource gofmts generated Go files; other files pass through.
func formatSource(name string, src []byte) (string, error) {
	if !strings.HasSuffix(name, ".go") {
		return string(src), nil
	}
	out, err := format.Source(src)
	if err != nil {
		return "", fmt.Errorf("receivercode: generated %s does not parse: %w", name, err)
	}
	return string(out), nil
}

// commonData is the template data every target shares.
func (g *gen) commonData() map[string]any {
	return map[string]any{
		"Header":           g.header(),
		"Version":          g.in.SparrowVersion,
		"Target":           g.spec.Target,
		"WebhookID":        g.in.Webhook.ID,
		"Consumer":         g.in.Webhook.Consumer,
		"Ed25519":          g.in.Webhook.SigningScheme == SchemeEd25519,
		"PublicKey":        g.spec.Webhook.PublicKey,
		"ToleranceSeconds": DefaultToleranceSeconds,
		"DedupCapacity":    DefaultDedupCapacity,
		"UnknownStatus":    UnknownEventStatus,
		"Port":             g.spec.Receiver.DefaultPort,
		"EnvSecret":        EnvSecret,
		"EnvPublicKey":     EnvPublicKey,
		"EnvTestMode":      EnvTestMode,
		"WebhookPath":      WebhookPath,
		"HealthPath":       HealthPath,
		"Spec":             g.spec,
		"EventRoutes":      g.eventRoutes(),
	}
}

// eventRoute maps an event type to the first subscription handling it, for
// the X-Sparrow-Event-Type fallback.
type eventRoute struct{ EventType, Name, Handler string }

func (g *gen) eventRoutes() []eventRoute {
	var out []eventRoute
	seen := map[string]bool{}
	for _, p := range g.plans {
		if seen[p.Sub.EventType] {
			continue
		}
		seen[p.Sub.EventType] = true
		out = append(out, eventRoute{EventType: p.Sub.EventType, Name: p.Name, Handler: handlerName(p)})
	}
	return out
}
