package receivercode

import (
	"bytes"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"text/template/parse"
	"time"
	"unicode"

	"github.com/sarathsp06/sparrow/pkg/receivercode/infer"
	"github.com/sarathsp06/sparrow/pkg/template"
)

// Model kinds: how a subscription's body is decoded.
const (
	// ModelEnvelope is Sparrow's JSON envelope with a typed payload.
	ModelEnvelope = "envelope"
	// ModelJSON is a transformed JSON body with an inferred type.
	ModelJSON = "json"
	// ModelXML is a transformed XML body with an inferred element tree.
	ModelXML = "xml"
	// ModelRaw is the body as a string, not decoded.
	ModelRaw = "raw"
)

// sampleEventID and sampleTime make the sample delivery deterministic.
const sampleEventID = "00000000-0000-4000-8000-000000000001"

var sampleTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// plan is what Generate learned about one subscription.
type plan struct {
	Sub Subscription
	// Name is the PascalCase base for the handler and model names, unique
	// within the receiver.
	Name   string
	Format string
	// ContentType is the subscription's Content-Type header, if set.
	ContentType string
	// Body is the sample delivery body: the rendered template, or the
	// envelope.
	Body  []byte
	Model string
	// Type is the payload type (ModelEnvelope) or body type (ModelJSON).
	Type *infer.Type
	// XML is the body's element tree (ModelXML).
	XML *infer.Element
	// ModelSource says where Type came from: "schema", "sample" or "".
	ModelSource string
	Notes       []string
	RenderError string
}

func analyze(s Subscription) *plan {
	p := &plan{Sub: s, ContentType: headerValue(s.Headers, "Content-Type")}
	if len(bytes.TrimSpace(s.Sample)) == 0 && len(bytes.TrimSpace(s.Schema)) > 0 {
		// Like the server, derive placeholder values from the schema, so
		// the model and the test vectors agree with it.
		if t, err := infer.FromSchema(s.Schema); err == nil && t.Kind == infer.Object {
			s.Sample = mustJSON(placeholder(t, 0))
			p.Sub.Sample = s.Sample
			p.Notes = append(p.Notes, "the event type has no sample payload; the sample delivery uses placeholder values from its JSON Schema")
		}
	}
	payload, payloadNote := samplePayload(s.Sample)
	if payloadNote != "" {
		p.Notes = append(p.Notes, payloadNote)
	}
	envelope := envelopeBody(s.EventType, payload)

	if s.Transform == "" {
		p.Body = envelope
		p.Format = FormatJSON
		p.Model = ModelEnvelope
		p.Type, p.ModelSource = payloadType(s, &p.Notes)
		return p
	}

	body, err := render(s, payload)
	if err != nil {
		p.RenderError = err.Error()
		p.Notes = append(p.Notes, "the transform template failed to render against the sample, so the body is handled as raw text")
		p.Body = envelope
		p.Format = p.formatOr(FormatJSON)
		p.Model = ModelRaw
		return p
	}
	p.Body = body
	p.Format = p.formatOr(DetectFormat(body))

	if why := variableShape(s.Transform); why != "" {
		p.Model = ModelRaw
		p.Notes = append(p.Notes, why)
		return p
	}
	switch p.Format {
	case FormatJSON:
		t, err := infer.FromJSON(body)
		if err != nil {
			p.Model = ModelRaw
			p.Notes = append(p.Notes, "Content-Type says JSON but the rendered sample is not valid JSON")
			return p
		}
		p.Model, p.Type, p.ModelSource = ModelJSON, t, "sample"
		p.Notes = append(p.Notes, "the model is inferred from one rendered sample: best effort, check it against the template")
	case FormatXML:
		el, err := infer.FromXML(body)
		if err != nil {
			p.Model = ModelRaw
			p.Notes = append(p.Notes, "Content-Type says XML but the rendered sample is not well-formed XML")
			return p
		}
		p.Model, p.XML, p.ModelSource = ModelXML, el, "sample"
		p.Notes = append(p.Notes, "the model is inferred from one rendered sample: best effort, check it against the template")
	default:
		p.Model = ModelRaw
	}
	return p
}

func (p *plan) formatOr(detected string) string {
	if f := formatFromContentType(p.ContentType); f != "" {
		return f
	}
	return detected
}

// samplePayload decodes the sample payload the way the server stores it.
func samplePayload(sample json.RawMessage) (map[string]any, string) {
	if len(bytes.TrimSpace(sample)) == 0 || bytes.Equal(bytes.TrimSpace(sample), []byte("null")) {
		return map[string]any{}, "the event type has no sample payload; the sample delivery uses an empty payload"
	}
	var payload map[string]any
	if err := json.Unmarshal(sample, &payload); err != nil || payload == nil {
		return map[string]any{}, "the sample payload is not a JSON object; the sample delivery uses an empty payload"
	}
	return payload, ""
}

// payloadType types the envelope payload: from the schema when there is
// one, else from the sample.
func payloadType(s Subscription, notes *[]string) (*infer.Type, string) {
	if len(bytes.TrimSpace(s.Schema)) > 0 {
		t, err := infer.FromSchema(s.Schema)
		if err == nil {
			return t, "schema"
		}
		*notes = append(*notes, "the JSON Schema could not be read ("+err.Error()+"); the payload type comes from the sample")
	}
	if len(bytes.TrimSpace(s.Sample)) > 0 {
		if t, err := infer.FromJSON(s.Sample); err == nil && t.Kind == infer.Object {
			*notes = append(*notes, "the payload type is inferred from the sample payload: best effort")
			return t, "sample"
		}
	}
	return &infer.Type{Kind: infer.Map, Elem: &infer.Type{}, Note: "no schema or sample to type the payload"}, ""
}

// envelopeBody builds the envelope a delivery without a transform carries,
// with the same field order as the server's.
func envelopeBody(eventType string, payload map[string]any) []byte {
	env := struct {
		Version   string         `json:"version"`
		EventID   string         `json:"event_id"`
		EventName string         `json:"event_name"`
		Timestamp string         `json:"timestamp"`
		Attempt   int            `json:"attempt"`
		Payload   map[string]any `json:"payload"`
	}{"1", sampleEventID, eventType, sampleTime.Format(time.RFC3339), 1, payload}
	b, _ := json.Marshal(env) // a decoded JSON map always re-encodes
	return b
}

var engine = template.New()

// render runs the transform against the sample with the server's engine and
// limits, with the clock fixed so a template that reads it stays
// deterministic.
func render(s Subscription, payload map[string]any) ([]byte, error) {
	ctx := template.NewWebhookContext(sampleEventID, s.EventType, sampleTime.Format(time.RFC3339), 1, payload)
	return engine.ExecuteWith(s.Transform, ctx, template.ExecOptions{
		StrictMissingKeys: s.StrictMissingKeys,
		Now:               func() time.Time { return sampleTime },
	})
}

// parseTrees parses a template with the engine's helper names, for
// inspection only.
func parseTrees(src string) map[string]*parse.Tree {
	funcs := map[string]any{}
	for name, fn := range template.FuncMap() {
		funcs[name] = fn
	}
	trees, err := parse.Parse("webhook", src, "", "", funcs, map[string]any{"html": nil, "js": nil})
	if err != nil {
		return nil
	}
	return trees
}

// variableShape explains why a template's output shape can vary between
// events (a conditional), or returns "".
func variableShape(src string) string {
	why := ""
	walkTemplate(src, func(n parse.Node) {
		if why != "" {
			return
		}
		switch n.(type) {
		case *parse.IfNode, *parse.WithNode:
			why = "the template has conditionals ({{if}} or {{with}}), so its output shape can vary between events; the body is handled as raw text"
		}
	})
	return why
}

func walkTemplate(src string, visit func(parse.Node)) {
	trees := parseTrees(src)
	names := make([]string, 0, len(trees))
	for name := range trees {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		walkNode(trees[name].Root, visit)
	}
}

func walkNode(n parse.Node, visit func(parse.Node)) {
	if n == nil {
		return
	}
	visit(n)
	switch n := n.(type) {
	case *parse.ListNode:
		if n == nil {
			return
		}
		for _, c := range n.Nodes {
			walkNode(c, visit)
		}
	case *parse.ActionNode:
		walkNode(n.Pipe, visit)
	case *parse.PipeNode:
		if n == nil {
			return
		}
		for _, c := range n.Cmds {
			walkNode(c, visit)
		}
	case *parse.CommandNode:
		for _, a := range n.Args {
			walkNode(a, visit)
		}
	case *parse.IfNode:
		walkBranch(&n.BranchNode, visit)
	case *parse.RangeNode:
		walkBranch(&n.BranchNode, visit)
	case *parse.WithNode:
		walkBranch(&n.BranchNode, visit)
	case *parse.TemplateNode:
		walkNode(n.Pipe, visit)
	}
}

func walkBranch(b *parse.BranchNode, visit func(parse.Node)) {
	walkNode(b.Pipe, visit)
	walkNode(b.List, visit)
	if b.ElseList != nil {
		walkNode(b.ElseList, visit)
	}
}

// pascal turns an event type or key into a PascalCase identifier part.
func pascal(s string) string {
	var b strings.Builder
	upper := true
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) || r > unicode.MaxASCII {
			upper = true
			continue
		}
		if upper {
			b.WriteRune(unicode.ToUpper(r))
			upper = false
			continue
		}
		b.WriteRune(r)
	}
	out := b.String()
	if out == "" || unicode.IsDigit(rune(out[0])) {
		out = "X" + out
	}
	return out
}

// assignNames sorts plans by event type and ID and gives each a unique
// PascalCase name.
func assignNames(plans []*plan) {
	sort.Slice(plans, func(i, j int) bool {
		a, b := plans[i].Sub, plans[j].Sub
		if a.EventType != b.EventType {
			return a.EventType < b.EventType
		}
		return a.ID < b.ID
	})
	used := map[string]bool{}
	for _, p := range plans {
		base := pascal(p.Sub.EventType)
		name := base
		for n := 2; used[name]; n++ {
			name = base + strconv.Itoa(n)
		}
		used[name] = true
		p.Name = name
	}
}
