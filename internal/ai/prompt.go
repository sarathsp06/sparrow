package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// The drafting prompt has three parts, in this order:
//
//  1. Template reference: static. How Sparrow renders a transform template,
//     the Go text/template language as Sparrow uses it, the pitfalls the
//     render check catches, and the helper catalog. It is identical for
//     every request, so it is the system prompt and prefix-caches.
//  2. Destination: the recipe, when one is chosen. What the destination
//     API is, how it is called, what a good message for it looks like,
//     its params, and its reference template.
//  3. The request: the event type, its schema, sample payload and a field
//     outline derived from both, the receiver's example or docs, the
//     current template, and the user's instructions.

// Recipe is the destination format a draft must follow. It mirrors the
// parts of a shipped recipe (satellites/recipes) the model needs.
type Recipe struct {
	Name        string
	Description string
	// URL is the destination endpoint, with {{param "x"}} tokens unfilled.
	URL string
	// ContentType is the Content-Type the destination is sent, when set.
	ContentType string
	// Guidance describes what an ideal message for the destination looks
	// like. Advice, not a rule.
	Guidance string
	// Params are the values substituted into {{param "x"}} tokens when the
	// recipe is applied.
	Params []RecipeParam
	// Template is the recipe's reference transform template.
	Template string
}

// RecipeParam is one {{param "name"}} value of a recipe.
type RecipeParam struct {
	Name   string
	Prompt string
	// Help says what the value is; Example and Default show its shape; Enum
	// lists the only accepted values. Any may be empty.
	Help    string
	Example string
	Default string
	Enum    []string
}

// PromptBuilder assembles the drafting prompt from a Request. It needs no
// model or key, so a server with no AI provider configured can still hand
// the prompt to a user to paste into any chat assistant.
type PromptBuilder struct {
	helpers []HelperFunc
}

// NewPromptBuilder returns a builder that knows the template helper catalog.
func NewPromptBuilder(helpers []HelperFunc) *PromptBuilder {
	return &PromptBuilder{helpers: helpers}
}

// ChatPrompt returns one self-contained prompt for a human to paste into a
// chat assistant: the same grounding the drafter sends, but asking for the
// template in a code block instead of a JSON object.
func (p *PromptBuilder) ChatPrompt(req Request, docsText string) string {
	return p.systemPrompt(true) + "\n\n" + p.userPrompt(req, docsText) +
		"\nReply with the complete template in one ```-fenced code block (a longer fence if the template itself contains ```) and nothing else inside it, then one or two sentences on which payload fields you used and any assumption or placeholder you introduced. The person will paste the code block into Sparrow's template editor and check it with Run Preview.\n"
}

// templateReference is part 1 of the prompt. FENCE stands for a Markdown
// code fence, which a Go raw string cannot contain; see referenceText.
const templateReference = `You write Sparrow subscription transform templates.

# Part 1: How Sparrow templates work

Sparrow is a webhook delivery service. A subscription's transform template is a Go text/template that Sparrow renders once per delivery attempt; the rendered text is sent verbatim as the HTTP request body to the receiver. Headers, the URL and signing are configured elsewhere: the template produces the body and nothing else.

## The data (the dot)

The template is executed against a map with exactly these keys:

  .event_id    string  unique id of this event occurrence, stable across retries
  .event_name  string  the event type name, e.g. "order.created"
  .timestamp   string  RFC 3339 time of this delivery attempt, e.g. "2026-03-01T12:00:00Z"
  .attempt     int     delivery attempt number, starting at 1
  .payload     object  the event payload, decoded from JSON (described in part 3)

Keys are lowercase snake_case: .payload, never .Payload. Nothing else is available: no consumer, webhook id, headers or secrets.

The payload is decoded from JSON, so its values are Go types: objects are map[string]any, arrays are []any, strings are string, booleans are bool, null is nil, and every number is float64 (there are no ints in the payload, even for 3 or 42).

## Strict rendering

Templates render with missingkey=error: that is the default for subscriptions and how every draft is checked, so write for it. Consequences:

- {{ .payload.coupon }} fails the delivery when the payload has no "coupon" key. Only read a field directly when it is always present (required in the schema).
- For a field that may be absent, use dig, which never fails: {{ dig "coupon" "" .payload }}, or nested {{ dig "customer" "address" "city" "unknown" .payload }} (keys..., then the default, then the map).
- index also never fails on a missing map key; it yields nil: {{ with index .payload "coupon" }}...{{ end }}.
- default does NOT protect against a missing key: in {{ default "x" .payload.coupon }} the lookup fails before default runs. default only replaces nil or "".
- A key whose value is null reads as nil and prints as "<no value>". Sparrow rejects any rendered output containing "<no value>", so guard nullable fields with {{ with }} / {{ if }}, give them a default via dig, or emit them through json (which prints null).
- Keys that are not Go identifiers (containing "-", ".", spaces, or starting with a digit) cannot be written with dots: use {{ index .payload "first-name" }} or dig.
- Field paths only work through objects. .payload.items.sku is an error when items is an array: iterate it with range, or pick one element with {{ index .payload.items 0 }}.

## Go template syntax, as used here

- Actions are delimited by {{ }}. Everything outside them is copied to the output literally, including newlines and indentation.
- {{- and -}} trim the whitespace (including newlines) before / after the action. Use them to keep loops and conditionals from adding blank lines; it rarely matters for JSON validity, but matters for plain-text and form-encoded bodies.
- Pipelines pass the previous value as the LAST argument: {{ .payload.name | upper }} is {{ upper .payload.name }}; {{ .payload.note | ellipsis 100 }} is {{ ellipsis 100 .payload.note }}.
- Parentheses group calls: {{ printf "%s (%s)" .payload.name (upper .payload.plan) }}.
- Variables: {{ $total := .payload.total }}, reassignment with {{ $n = add $n 1 }}. $ is always the root data, so inside range or with (which rebind the dot) reach the event with $.event_id or $.payload.
- Conditionals: {{ if X }}...{{ else if Y }}...{{ else }}...{{ end }}. Empty values are false: false, 0, nil, "", empty arrays and maps.
- {{ with X }}...{{ end }} runs the body only when X is non-empty, with the dot set to X; {{ else }} is allowed.
- Loops: {{ range $i, $item := .payload.items }}...{{ end }} over arrays (index, element) or maps (key, value, sorted by key); {{ else }} runs when it is empty. Inside a bare {{ range .payload.items }} the dot is the element.
- Comparison and logic builtins: eq, ne, lt, le, gt, ge, and, or, not. {{ if and (eq .payload.status "paid") (gt .payload.total 100.0) }}. eq accepts several values: {{ if eq .payload.status "paid" "settled" }} is true when status is either.
- Other builtins: index (map key or array position, nested: index .payload "a" "b"), printf / print (fmt formatting), urlquery, html, js.
- len and slice are Sparrow helpers that replace the builtins with different behaviour; see the catalog.
- Comments are {{/* ... */}}; don't write them unless asked.

## Numbers

- Payload numbers are float64 and a literal like 3 is an int; comparing the two is an error ("incompatible types for comparison"). Compare against float literals: {{ if gt .payload.total 100.0 }}, {{ if eq .payload.count 0.0 }}. .attempt is an int, so {{ if gt .attempt 1 }} is fine.
- printf "%d" with a payload number prints "%!d(float64=3)". Use %v, or convert first: {{ printf "%d" (toInt .payload.count) }}.
- Large floats print in exponent form ({{ .payload.amount }} can print 1.5e+06). In JSON output pipe numbers through json (prints 1500000); in text use toInt, or printf "%.2f" for fixed decimals.
- add, sub, mul, div return float64 too: {{ mul .payload.amount 100 | toInt }} for cents.

## Producing the body

JSON (most receivers):
- Write the JSON structure literally and emit every interpolated value through the json helper, which adds the quotes and escaping: "name": {{ .payload.name | json }}. Never hand-quote: "name": "{{ .payload.name }}" breaks on a quote or newline in the data.
- Values built with printf go through json last: "text": {{ printf "Order %v paid" .payload.id | json }}.
- Whole objects and arrays can be copied with json: "data": {{ .payload | json }}, "items": {{ .payload.items | json }}.
- Build new objects and arrays with dict and list, then json: {{ dict "id" .payload.id "total" .payload.total | json }}. Collect values in a loop with append: {{ $ids := list }}{{ range .payload.items }}{{ $ids = append $ids .id }}{{ end }}{{ $ids | json }}.
- When writing an array by hand with range, put commas between elements, not after the last: [{{ range $i, $x := .payload.items }}{{ if $i }},{{ end }}{{ $x.sku | json }}{{ end }}].
- Optional members: either always emit the key with a fallback ("coupon": {{ dig "coupon" nil .payload | json }} prints null when absent), or wrap the whole member in {{ with }} and mind the commas around it (simplest: put optional members last and lead them with the comma).
- A value that must be a JSON string containing JSON (a "payload as text" column) is json applied twice: {{ .payload | json | json }}.
- The json helper escapes <, > and & as <, >, &. That is valid JSON and every receiver decodes it.
Sparrow rejects a draft whose output starts with { or [ but is not valid JSON.

Form-encoded (application/x-www-form-urlencoded): key=value pairs joined by &, with every interpolated value through urlencode: Body={{ printf "Order %v" .payload.id | urlencode }}&To=%2B15551230000.

Plain text: write the text; use {{- -}} to control blank lines; strings need no escaping.

Limits: output over 1 MB, or rendering longer than 5 seconds, fails the delivery.

## Rules

- Use only the Go template syntax above and the helpers in the catalog below. Do not invent helpers (there is no sprig, no toJson, no param).
- Check every field name, its case and its nesting against the payload outline and sample in part 3. Read required fields directly and anything optional, nullable or uncertain with dig or index.
- Prefer a small, readable template over a clever one. Keep the destination's literal structure visible rather than generating it all with dict.
- If given a destination recipe (part 2), keep the top-level shape of its reference template (the destination API requires it) and adapt only the content to the instructions. Its reference template shows a valid body, not a good message: use its guidance to decide what the message says and how.
- If given an example of what the receiver expects, match its structure and field names exactly and map event fields onto it. If given a description instead, follow it literally. If given documentation, use only the parts about the request body.
- If given a current template to refine, keep everything the instructions do not ask to change.
- Your template is rendered against the sample payload before anyone sees it; it must render without error, without "<no value>", and (for JSON) to valid JSON.

## Example

Instructions: "Post order id, customer email and total; mention the coupon when there is one."
Payload: {"id": "ord_1", "total": 49.5, "customer": {"email": "a@example.com"}, "coupon": "SPRING"} with coupon optional.

FENCE
{
  "text": {{ printf "Order %v paid: %.2f" .payload.id .payload.total | json }},
  "email": {{ .payload.customer.email | json }},
  "total": {{ .payload.total | json }},
  "attempt": {{ .attempt }}
  {{- with dig "coupon" "" .payload }},
  "coupon": {{ . | json }}
  {{- end }}
}
FENCE
`

// referenceText is templateReference with real code fences, built once.
var referenceText = strings.ReplaceAll(templateReference, "FENCE", "```")

// systemPrompt is part 1 plus the output-format instruction and the helper
// catalog. It is stable across requests so it can be prefix-cached. chat
// swaps the structured-output instruction for a code-block one.
func (p *PromptBuilder) systemPrompt(chat bool) string {
	var b strings.Builder
	b.WriteString(referenceText)
	if chat {
		b.WriteString("\nReply with the complete template in one fenced code block, followed by one or two plain sentences of notes.\n")
	} else {
		b.WriteString("\nRespond only with the JSON object described by the output schema. The template field is the full template text; the notes field is one or two plain sentences for the user: the payload fields used, and any assumption or placeholder.\n")
	}
	b.WriteString("\n## Helper catalog\n\nThese are the only functions available besides the builtins above. Name, then its documentation.\n")
	for _, h := range p.helpers {
		b.WriteString("\n### ")
		b.WriteString(h.Name)
		b.WriteString("\n")
		b.WriteString(h.Description)
		b.WriteString("\n")
	}
	return b.String()
}

// userPrompt is parts 2 and 3: the destination recipe, if any, then the
// event and the user's request.
func (p *PromptBuilder) userPrompt(req Request, docsText string) string {
	var b strings.Builder
	if req.Recipe != nil {
		writeRecipe(&b, req.Recipe, strings.TrimSpace(req.CurrentTemplate) != "")
	}

	b.WriteString("# Part 3: The request\n\n")
	fmt.Fprintf(&b, "Event type: %s (this is .event_name)\n\n", req.EventName)
	if outline := payloadOutline(req.Schema, req.SamplePayload); outline != "" {
		b.WriteString("## Payload fields\n\n")
		b.WriteString(outline)
		b.WriteString("\n")
	}
	if len(req.Schema) > 0 {
		b.WriteString("## JSON Schema of the payload\n\n")
		writeFenced(&b, "json", mustJSON(req.Schema))
		b.WriteString("\n")
	}
	if len(req.SamplePayload) > 0 {
		b.WriteString("## Sample payload\n\nThis is .payload for the render check:\n\n")
		writeFenced(&b, "json", mustJSON(req.SamplePayload))
		b.WriteString("\n")
	} else {
		b.WriteString("No sample payload is registered for this event type; rely on the schema and the instructions, and read every field with dig.\n\n")
	}
	if strings.TrimSpace(req.TargetExample) != "" {
		b.WriteString("## What the receiver expects\n\nAn example body, or a description of its shape:\n\n")
		writeFenced(&b, "", strings.TrimSpace(req.TargetExample))
		b.WriteString("\n")
	}
	if strings.TrimSpace(docsText) != "" {
		fmt.Fprintf(&b, "## Receiver documentation\n\nText extracted from %s (use only what concerns the request body):\n\n", strings.TrimSpace(req.DocsURL))
		writeFenced(&b, "", strings.TrimSpace(docsText))
		b.WriteString("\n")
	}
	if strings.TrimSpace(req.CurrentTemplate) != "" {
		b.WriteString("## Current template to refine\n\n")
		writeFenced(&b, "", strings.TrimSpace(req.CurrentTemplate))
		b.WriteString("\n")
	}
	b.WriteString("## Instructions\n\n")
	b.WriteString(strings.TrimSpace(req.Instructions))
	b.WriteString("\n")
	return b.String()
}

// writeRecipe writes part 2. refining says a current template exists, whose
// already-substituted param values the draft should keep.
func writeRecipe(b *strings.Builder, r *Recipe, refining bool) {
	fmt.Fprintf(b, "# Part 2: Destination recipe %q\n\n", r.Name)
	if r.Description != "" {
		b.WriteString(r.Description)
		b.WriteString(".\n\n")
	}
	if r.URL != "" {
		fmt.Fprintf(b, "The body is POSTed to %s", r.URL)
		if r.ContentType != "" {
			fmt.Fprintf(b, " with Content-Type: %s", r.ContentType)
		}
		b.WriteString(". The destination accepts only this format, never Sparrow's default envelope.\n\n")
	}
	if g := strings.TrimSpace(r.Guidance); g != "" {
		b.WriteString("## What a good message looks like\n\nGuidance for this destination, not a rule: follow it where the instructions and any receiver example leave room, and let them win where they differ. The example is a rendered message, not a template.\n\n")
		writeFenced(b, "", g)
		b.WriteString("\n")
	}
	if len(r.Params) > 0 {
		b.WriteString("## Recipe params\n\nWhen the recipe is applied, each {{param \"name\"}} token is replaced by plain text substitution with the user's value:\n\n")
		for _, prm := range r.Params {
			writeParam(b, prm)
		}
		b.WriteString("\nYour template is saved and rendered as-is, after that step, so it must not contain {{param}} tokens (param is not a template function). ")
		if refining {
			b.WriteString("The current template already carries the substituted values; keep them unchanged. ")
		}
		b.WriteString("Where the body needs a param value you do not have, write a literal placeholder such as \"REPLACE_WITH_ROUTING_KEY\" and say so in your notes.\n\n")
	}
	if strings.TrimSpace(r.Template) != "" {
		b.WriteString("## Reference template\n\nThe recipe's own template. Keep its top-level shape and adapt the content; its {{param}} tokens follow the rule above:\n\n")
		writeFenced(b, "", strings.TrimSpace(r.Template))
		b.WriteString("\n")
	}
}

// writeParam writes one recipe param as a list item: its label, then what
// the model can use to write a realistic placeholder.
func writeParam(b *strings.Builder, prm RecipeParam) {
	fmt.Fprintf(b, "- %s: %s", prm.Name, prm.Prompt)
	if prm.Help != "" {
		fmt.Fprintf(b, ". %s", strings.TrimSuffix(prm.Help, "."))
	}
	var shape []string
	if len(prm.Enum) > 0 {
		shape = append(shape, "one of "+strings.Join(prm.Enum, ", "))
	}
	if prm.Example != "" {
		shape = append(shape, "e.g. "+prm.Example)
	}
	if prm.Default != "" {
		shape = append(shape, "default "+prm.Default)
	}
	if len(shape) > 0 {
		fmt.Fprintf(b, " (%s)", strings.Join(shape, "; "))
	}
	b.WriteString("\n")
}

// Bounds on the payload outline, so a huge schema or sample cannot crowd
// out the rest of the prompt.
const (
	outlineMaxFields = 150
	outlineMaxDepth  = 8
	outlineMaxValue  = 40
	outlineMaxNote   = 120
)

// outlineField is one row of the payload outline.
type outlineField struct {
	path     string
	typ      string
	presence string
	sample   string
	notes    string
}

// payloadOutline flattens the schema and sample into one row per field: its
// template path, JSON type, whether it is always present, and its sample
// value. It tells the model at a glance which fields it can read directly
// and which need dig. Returns "" when there is nothing to describe.
func payloadOutline(schema, sample map[string]any) string {
	var rows []outlineField
	walkObject(&rows, ".payload", schema, sample, sample != nil, false, 0)
	if len(rows) == 0 {
		return ""
	}
	truncated := len(rows) > outlineMaxFields
	if truncated {
		rows = rows[:outlineMaxFields]
	}
	pathW, typW, presW := 0, 0, 0
	for _, r := range rows {
		pathW = max(pathW, len(r.path))
		typW = max(typW, len(r.typ))
		presW = max(presW, len(r.presence))
	}
	var b strings.Builder
	b.WriteString("One row per field: template path, JSON type, presence, sample value, then the schema's rules for it (allowed values, format, description) after a #. ")
	b.WriteString("\"required\" fields can be read directly; \"required in parent\" fields are always in their parent, but the parent may be absent or null, so read them inside a {{ with }} on the parent or with dig; read \"optional\", \"nullable\" and \"unknown\" (no schema says) fields with dig or index. ")
	b.WriteString("[] marks the elements of an array: reach them with range. ")
	b.WriteString("Paths shown with [\"key\"] are not identifiers and need index or dig.\n\n")
	var t strings.Builder
	for _, r := range rows {
		line := fmt.Sprintf("%-*s  %-*s  %-*s  %s", pathW, r.path, typW, r.typ, presW, r.presence, r.sample)
		if r.notes != "" {
			line += "  # " + r.notes
		}
		t.WriteString(strings.TrimRight(line, " "))
		t.WriteString("\n")
	}
	if truncated {
		fmt.Fprintf(&t, "... (outline cut at %d fields; see the schema and sample below)\n", outlineMaxFields)
	}
	writeFenced(&b, "", strings.TrimSuffix(t.String(), "\n"))
	return b.String()
}

// walkObject adds a row per property of an object node. schema and sample
// may each be nil; inSample says whether the object exists in the sample at
// all, so a field missing from it can be told apart from an absent parent.
// mayBeAbsent says the object itself may be missing or null, so a field the
// schema requires inside it is only "required in parent".
func walkObject(rows *[]outlineField, path string, schema, sample map[string]any, inSample, mayBeAbsent bool, depth int) {
	if depth >= outlineMaxDepth || len(*rows) > outlineMaxFields {
		return
	}
	props, _ := schema["properties"].(map[string]any)
	required := map[string]bool{}
	if req, ok := schema["required"].([]any); ok {
		for _, k := range req {
			if s, ok := k.(string); ok {
				required[s] = true
			}
		}
	}
	keys := make([]string, 0, len(props)+len(sample))
	seen := map[string]bool{}
	for k := range sample {
		keys, seen[k] = append(keys, k), true
	}
	for k := range props {
		if !seen[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		if len(*rows) >= outlineMaxFields+1 {
			return // enough to show the cut; payloadOutline truncates
		}
		childSchema, _ := props[k].(map[string]any)
		val, present := sample[k]
		childPath := path + fieldStep(k)

		row := outlineField{path: childPath, typ: valueType(childSchema, val, present), notes: schemaNotes(childSchema)}
		switch {
		case schema != nil && props != nil && required[k] && mayBeAbsent:
			row.presence = "required in parent"
		case schema != nil && props != nil && required[k]:
			row.presence = "required"
		case schema != nil && props != nil:
			row.presence = "optional"
		default:
			row.presence = "unknown"
		}
		if schemaAllowsNull(childSchema) || (present && val == nil) {
			if row.presence != "unknown" {
				row.presence += ", nullable"
			}
		}
		switch {
		case !present && inSample:
			row.sample = "(absent)"
		case !present:
			row.sample = ""
		default:
			row.sample = sampleValue(val)
		}
		*rows = append(*rows, row)

		// Children of anything but a plain required field may find their
		// parent missing or null.
		childAbsent := row.presence != "required"
		switch v := val.(type) {
		case map[string]any:
			walkObject(rows, childPath, childSchema, v, true, childAbsent, depth+1)
		case []any:
			walkArray(rows, childPath, childSchema, v, childAbsent, depth+1)
		default:
			if !present && childSchema != nil {
				if items, ok := childSchema["items"].(map[string]any); ok {
					walkObject(rows, childPath+"[]", items, nil, false, childAbsent, depth+1)
				} else {
					walkObject(rows, childPath, childSchema, nil, false, childAbsent, depth+1)
				}
			}
		}
	}
}

// walkArray describes an array's elements from its first sample element and
// the schema's items.
func walkArray(rows *[]outlineField, path string, schema map[string]any, sample []any, mayBeAbsent bool, depth int) {
	items, _ := schema["items"].(map[string]any)
	if len(sample) > 0 {
		switch first := sample[0].(type) {
		case map[string]any:
			walkObject(rows, path+"[]", items, first, true, mayBeAbsent, depth)
		case []any:
			walkArray(rows, path+"[]", items, first, mayBeAbsent, depth)
		}
		return
	}
	if items != nil {
		walkObject(rows, path+"[]", items, nil, false, mayBeAbsent, depth)
	}
}

var identifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// fieldStep renders one path step: .key when it is a Go identifier the dot
// syntax accepts, ["key"] otherwise.
func fieldStep(k string) string {
	if identifier.MatchString(k) {
		return "." + k
	}
	return fmt.Sprintf("[%q]", k)
}

// valueType names a field's JSON type, from the sample when present and the
// schema otherwise.
func valueType(schema map[string]any, val any, present bool) string {
	if present && val != nil {
		switch val.(type) {
		case map[string]any:
			return "object"
		case []any:
			return "array"
		case string:
			return "string"
		case bool:
			return "boolean"
		case float64:
			return "number"
		default:
			return fmt.Sprintf("%T", val)
		}
	}
	switch t := schema["type"].(type) {
	case string:
		return t
	case []any:
		var names []string
		for _, n := range t {
			if s, ok := n.(string); ok && s != "null" {
				names = append(names, s)
			}
		}
		if len(names) > 0 {
			return strings.Join(names, "|")
		}
	}
	if present {
		return "null"
	}
	return "?"
}

// schemaNotes summarises the schema keywords that constrain what a template
// may assume about a field: const, enum, format, numeric and length bounds,
// and its description.
func schemaNotes(schema map[string]any) string {
	if schema == nil {
		return ""
	}
	var notes []string
	if c, ok := schema["const"]; ok {
		notes = append(notes, "always "+sampleValue(c))
	}
	if enum, ok := schema["enum"].([]any); ok && len(enum) > 0 {
		vals := make([]string, 0, len(enum))
		for _, v := range enum {
			vals = append(vals, sampleValue(v))
		}
		notes = append(notes, "one of "+strings.Join(vals, ", "))
	}
	if f, ok := schema["format"].(string); ok && f != "" {
		notes = append(notes, "format "+f)
	}
	for _, k := range []string{"minimum", "maximum", "minLength", "maxLength", "minItems", "maxItems"} {
		if v, ok := schema[k]; ok {
			notes = append(notes, fmt.Sprintf("%s %v", k, v))
		}
	}
	if d, ok := schema["description"].(string); ok && strings.TrimSpace(d) != "" {
		notes = append(notes, strings.Join(strings.Fields(d), " "))
	}
	out := strings.Join(notes, "; ")
	if r := []rune(out); len(r) > outlineMaxNote {
		out = string(r[:outlineMaxNote-1]) + "…"
	}
	return out
}

func schemaAllowsNull(schema map[string]any) bool {
	switch t := schema["type"].(type) {
	case string:
		return t == "null"
	case []any:
		for _, n := range t {
			if n == "null" {
				return true
			}
		}
	}
	if n, ok := schema["nullable"].(bool); ok && n {
		return true
	}
	return false
}

// sampleValue shows a sample value compactly: scalars as JSON (cut to
// outlineMaxValue), containers by size.
func sampleValue(v any) string {
	switch t := v.(type) {
	case map[string]any:
		return fmt.Sprintf("{%d keys}", len(t))
	case []any:
		return fmt.Sprintf("[%d items]", len(t))
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	s := string(b)
	if r := []rune(s); len(r) > outlineMaxValue {
		s = string(r[:outlineMaxValue-1]) + "…"
	}
	return s
}

// writeFenced writes content in a Markdown code block whose fence is longer
// than any backtick run inside it, so content that contains ``` (the slack
// and discord templates do) cannot close the block early.
func writeFenced(b *strings.Builder, lang, content string) {
	longest, run := 0, 0
	for _, r := range content {
		if r == '`' {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	fence := strings.Repeat("`", max(3, longest+1))
	b.WriteString(fence)
	b.WriteString(lang)
	b.WriteString("\n")
	b.WriteString(content)
	b.WriteString("\n")
	b.WriteString(fence)
	b.WriteString("\n")
}

func mustJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}
