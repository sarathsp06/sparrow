// cmd/gendocs generates API documentation TypeScript data files from proto/webhook.proto.
//
// It parses the proto file for structure (services, RPCs, messages, fields, types,
// deprecation markers, error codes, descriptions) and merges with an overlay YAML
// for data that can't be derived from proto (example values, notes, footers).
//
// Usage:
//
//	go run ./cmd/gendocs
//	go run ./cmd/gendocs -proto proto/webhook.proto -overlay docs/api-overrides.yaml -outdir docs/src/data/api
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/emicklei/proto"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Proto model (parsed from .proto file)
// ---------------------------------------------------------------------------

type ProtoField struct {
	Name        string
	Type        string // mapped type: "string", "int32", "Timestamp", "Struct (JSON)", etc.
	RawType     string // original proto type (for message resolution)
	Description string
	Required    bool
	Deprecated  bool
	IsRepeated  bool
	IsMap       bool
	MapKey      string
	MapValue    string
	IsOptional  bool
	IsMessage   bool // whether the type references a message
	IsEnum      bool // whether the type references an enum
}

type ProtoError struct {
	Code        string
	Description string
}

type ProtoRPC struct {
	Name         string
	Description  string
	RequestType  string
	ResponseType string
	Errors       []ProtoError
}

type ProtoService struct {
	Name        string
	Description string
	RPCs        []ProtoRPC
}

type ProtoMessage struct {
	Name   string
	Fields []ProtoField
}

type ProtoEnum struct {
	Name string
}

type ProtoFile struct {
	Services map[string]*ProtoService
	Messages map[string]*ProtoMessage
	Enums    map[string]*ProtoEnum
}

// ---------------------------------------------------------------------------
// Overlay model (from YAML)
// ---------------------------------------------------------------------------

type OverlayFieldConfig struct {
	Example     any    `yaml:"example"`
	Description string `yaml:"description"` // override proto description
	Required    *bool  `yaml:"required"`    // override auto-detected required
}

type OverlayRPCConfig struct {
	Description string                        `yaml:"description"` // override
	Fields      map[string]OverlayFieldConfig `yaml:"fields"`
}

type OverlayServiceConfig struct {
	Description string                      `yaml:"description"` // override
	Notes       string                      `yaml:"notes"`
	Footer      string                      `yaml:"footer"`
	RPCs        map[string]OverlayRPCConfig `yaml:"rpcs"`
}

type Overlay struct {
	// Services that should be generated (in order)
	ServiceOrder []string `yaml:"service_order"`
	// Message types that should NOT be flattened when they appear as fields
	EntityTypes []string `yaml:"entity_types"`
	// Per-service overrides
	Services map[string]OverlayServiceConfig `yaml:"services"`
}

// ---------------------------------------------------------------------------
// Proto parsing
// ---------------------------------------------------------------------------

var (
	errorLineRE = regexp.MustCompile(`^Errors?:\s+([A-Z_]+)\s+(?:if\s+)?(.+?)\.?$`)
	requiredRE  = regexp.MustCompile(`\bRequired\b`)
)

func parseProto(path string) (*ProtoFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open proto: %w", err)
	}
	defer func() { _ = f.Close() }()

	parser := proto.NewParser(f)
	definition, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse proto: %w", err)
	}

	pf := &ProtoFile{
		Services: make(map[string]*ProtoService),
		Messages: make(map[string]*ProtoMessage),
		Enums:    make(map[string]*ProtoEnum),
	}

	// First pass: collect all message and enum names (for type resolution)
	proto.Walk(definition,
		proto.WithMessage(func(m *proto.Message) {
			msg := &ProtoMessage{Name: m.Name}
			for _, e := range m.Elements {
				switch el := e.(type) {
				case *proto.NormalField:
					field := parseNormalField(el)
					msg.Fields = append(msg.Fields, field)
				case *proto.MapField:
					field := parseMapField(el)
					msg.Fields = append(msg.Fields, field)
				}
			}
			pf.Messages[m.Name] = msg
		}),
		proto.WithEnum(func(e *proto.Enum) {
			pf.Enums[e.Name] = &ProtoEnum{Name: e.Name}
		}),
	)

	// Second pass: collect services and RPCs
	proto.Walk(definition,
		proto.WithService(func(s *proto.Service) {
			svc := &ProtoService{
				Name:        s.Name,
				Description: extractComment(s.Comment),
			}
			for _, e := range s.Elements {
				if rpc, ok := e.(*proto.RPC); ok {
					protoRPC := parseRPC(rpc)
					svc.RPCs = append(svc.RPCs, protoRPC)
				}
			}
			pf.Services[s.Name] = svc
		}),
	)

	// Mark message/enum references on fields
	for _, msg := range pf.Messages {
		for i := range msg.Fields {
			f := &msg.Fields[i]
			if _, ok := pf.Messages[f.RawType]; ok {
				f.IsMessage = true
			}
			if _, ok := pf.Enums[f.RawType]; ok {
				f.IsEnum = true
			}
		}
	}

	return pf, nil
}

func parseNormalField(f *proto.NormalField) ProtoField {
	field := ProtoField{
		Name:       f.Name,
		RawType:    f.Type,
		IsRepeated: f.Repeated,
		IsOptional: f.Optional,
	}
	field.Description = extractComment(f.Comment)
	if field.Description == "" {
		field.Description = extractInlineComment(f.InlineComment)
	}
	field.Required = requiredRE.MatchString(field.Description)
	field.Deprecated = hasDeprecatedOption(f.Options) || isCommentDeprecated(field.Description)
	field.Type = mapProtoType(f.Type, f.Repeated, false, "", "")
	return field
}

func parseMapField(f *proto.MapField) ProtoField {
	field := ProtoField{
		Name:     f.Name,
		RawType:  f.Type,
		IsMap:    true,
		MapKey:   f.KeyType,
		MapValue: f.Type,
	}
	field.Description = extractComment(f.Comment)
	if field.Description == "" {
		field.Description = extractInlineComment(f.InlineComment)
	}
	field.Required = requiredRE.MatchString(field.Description)
	field.Deprecated = hasDeprecatedOption(f.Options) || isCommentDeprecated(field.Description)
	field.Type = mapProtoType(f.Type, false, true, f.KeyType, f.Type)
	return field
}

func parseRPC(rpc *proto.RPC) ProtoRPC {
	fullComment := extractComment(rpc.Comment)
	desc, errors := splitRPCComment(fullComment)
	return ProtoRPC{
		Name:         rpc.Name,
		Description:  desc,
		RequestType:  rpc.RequestType,
		ResponseType: rpc.ReturnsType,
		Errors:       errors,
	}
}

func extractComment(c *proto.Comment) string {
	if c == nil {
		return ""
	}
	var lines []string
	for _, line := range c.Lines {
		line = strings.TrimSpace(line)
		// Remove leading "// " style prefix if present
		line = strings.TrimPrefix(line, " ")
		lines = append(lines, line)
	}
	return joinCommentLines(lines)
}

func extractInlineComment(c *proto.Comment) string {
	if c == nil {
		return ""
	}
	var lines []string
	for _, line := range c.Lines {
		lines = append(lines, strings.TrimSpace(line))
	}
	return strings.Join(lines, " ")
}

func joinCommentLines(lines []string) string {
	// Join consecutive non-empty lines into paragraphs
	var result []string
	var current []string
	for _, line := range lines {
		if line == "" {
			if len(current) > 0 {
				result = append(result, strings.Join(current, " "))
				current = nil
			}
		} else {
			current = append(current, line)
		}
	}
	if len(current) > 0 {
		result = append(result, strings.Join(current, " "))
	}
	return strings.Join(result, " ")
}

func splitRPCComment(comment string) (string, []ProtoError) {
	// Split on sentence boundaries, extracting "Errors: CODE if condition." lines
	lines := strings.Split(comment, " ")

	// Re-join and split on "Errors:" boundaries
	fullText := comment
	var errors []ProtoError
	var descParts []string

	// Split by sentence, looking for "Errors:" pattern
	sentences := splitSentences(fullText)
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		m := errorLineRE.FindStringSubmatch(s)
		if m != nil {
			errors = append(errors, ProtoError{
				Code:        m[1],
				Description: cleanErrorDescription(m[2]),
			})
		} else {
			descParts = append(descParts, s)
		}
	}

	_ = lines // unused but was part of earlier logic
	return strings.Join(descParts, " "), errors
}

func splitSentences(text string) []string {
	// Split on "Errors:" boundaries first
	parts := strings.Split(text, "Errors:")
	var result []string
	if len(parts) > 0 {
		// First part is the description
		desc := strings.TrimSpace(parts[0])
		if desc != "" {
			result = append(result, desc)
		}
		// Remaining parts are error lines
		for _, p := range parts[1:] {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, "Errors: "+p)
			}
		}
	}
	return result
}

func cleanErrorDescription(desc string) string {
	desc = strings.TrimSpace(desc)
	desc = strings.TrimSuffix(desc, ".")
	// Capitalize first letter
	if len(desc) > 0 {
		desc = strings.ToUpper(desc[:1]) + desc[1:]
	}
	return desc + "."
}

func hasDeprecatedOption(opts []*proto.Option) bool {
	for _, opt := range opts {
		if opt.Name == "deprecated" && opt.Constant.Source == "true" {
			return true
		}
	}
	return false
}

// isCommentDeprecated checks if a field description starts with "Deprecated:"
// which is the Go convention for marking deprecated fields via comments.
func isCommentDeprecated(description string) bool {
	return strings.HasPrefix(strings.TrimSpace(description), "Deprecated:")
}

func mapProtoType(typeName string, repeated, isMap bool, mapKey, mapValue string) string {
	if isMap {
		k := mapScalarType(mapKey)
		v := mapScalarType(mapValue)
		return fmt.Sprintf("map<%s, %s>", k, v)
	}
	base := mapScalarType(typeName)
	if repeated {
		return base + "[]"
	}
	return base
}

func mapScalarType(t string) string {
	switch t {
	case "string":
		return "string"
	case "int32":
		return "int32"
	case "int64":
		return "int64"
	case "bool":
		return "bool"
	case "double", "float":
		return "double"
	case "google.protobuf.Timestamp":
		return "Timestamp"
	case "google.protobuf.Struct":
		return "Struct (JSON)"
	default:
		// Message or enum type — return as-is
		return t
	}
}

// ---------------------------------------------------------------------------
// Overlay loading
// ---------------------------------------------------------------------------

func loadOverlay(path string) (*Overlay, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read overlay: %w", err)
	}
	var overlay Overlay
	if err := yaml.Unmarshal(data, &overlay); err != nil {
		return nil, fmt.Errorf("parse overlay: %w", err)
	}
	return &overlay, nil
}

// ---------------------------------------------------------------------------
// TypeScript generation
// ---------------------------------------------------------------------------

type TSField struct {
	Name        string
	Type        string
	Required    bool
	HasRequired bool // whether to emit the required field at all
	Description string
	Example     any // nil = omit
}

type TSError struct {
	Code        string
	Description string
}

type TSRPC struct {
	Name        string
	Description string
	Request     []TSField
	Response    []TSField
	Errors      []TSError
}

type TSService struct {
	Service     string
	Description string
	Notes       string
	Footer      string
	RPCs        []TSRPC
}

func buildTSService(pf *ProtoFile, svcName string, overlay *Overlay) (*TSService, error) {
	svc, ok := pf.Services[svcName]
	if !ok {
		return nil, fmt.Errorf("service %q not found in proto", svcName)
	}

	entityTypes := makeSet(overlay.EntityTypes)
	svcOverlay := overlay.Services[svcName]

	ts := &TSService{
		Service:     svcName,
		Description: coalesce(svcOverlay.Description, svc.Description),
		Notes:       svcOverlay.Notes,
		Footer:      svcOverlay.Footer,
	}

	for _, rpc := range svc.RPCs {
		rpcOverlay := svcOverlay.RPCs[rpc.Name]
		tsRPC := TSRPC{
			Name:        rpc.Name,
			Description: coalesce(rpcOverlay.Description, rpc.Description),
		}

		// Request fields
		reqMsg := pf.Messages[rpc.RequestType]
		if reqMsg != nil {
			tsRPC.Request = buildFields(pf, reqMsg, "", true, entityTypes, rpcOverlay.Fields)
		}

		// Response fields
		respMsg := pf.Messages[rpc.ResponseType]
		if respMsg != nil {
			tsRPC.Response = buildFields(pf, respMsg, "", false, entityTypes, rpcOverlay.Fields)
		}

		// Errors
		for _, e := range rpc.Errors {
			tsRPC.Errors = append(tsRPC.Errors, TSError(e))
		}

		ts.RPCs = append(ts.RPCs, tsRPC)
	}

	return ts, nil
}

func buildFields(pf *ProtoFile, msg *ProtoMessage, prefix string, isRequest bool, entityTypes map[string]bool, fieldOverrides map[string]OverlayFieldConfig) []TSField {
	var fields []TSField

	for _, f := range msg.Fields {
		if f.Deprecated {
			continue
		}

		fieldName := f.Name
		if prefix != "" {
			fieldName = prefix + "." + f.Name
		}

		// Should we flatten this message-type field?
		if f.IsMessage && !f.IsMap && shouldFlatten(f.RawType, entityTypes) {
			nestedMsg, ok := pf.Messages[f.RawType]
			if ok {
				nestedPrefix := fieldName
				if f.IsRepeated {
					nestedPrefix = fieldName + "[]"
				}
				nested := buildFields(pf, nestedMsg, nestedPrefix, isRequest, entityTypes, fieldOverrides)
				fields = append(fields, nested...)
				continue
			}
		}

		tsField := TSField{
			Name:        fieldName,
			Type:        f.Type,
			Description: cleanFieldDescription(f.Description),
		}

		// Required is only meaningful for request fields
		if isRequest && f.Required {
			tsField.Required = true
			tsField.HasRequired = true
		}

		// Apply overlay
		if ov, ok := fieldOverrides[fieldName]; ok {
			if ov.Description != "" {
				tsField.Description = ov.Description
			}
			if ov.Example != nil {
				tsField.Example = ov.Example
			}
			if ov.Required != nil {
				tsField.Required = *ov.Required
				tsField.HasRequired = true
			}
		}

		fields = append(fields, tsField)
	}

	return fields
}

func shouldFlatten(typeName string, entityTypes map[string]bool) bool {
	return !entityTypes[typeName]
}

func cleanFieldDescription(desc string) string {
	// Take the first sentence or two — the proto comments can be very verbose
	desc = strings.TrimSpace(desc)
	// Remove "Required." prefix if present (we handle that via the required flag)
	desc = strings.TrimPrefix(desc, "Required. ")
	return desc
}

func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func makeSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}

// ---------------------------------------------------------------------------
// TypeScript file emitter
// ---------------------------------------------------------------------------

func emitService(ts *TSService) string {
	var b strings.Builder

	b.WriteString("import type { ApiService } from \"./types\";\n\n")
	b.WriteString("const service: ApiService = {\n")
	fmt.Fprintf(&b, "  service: %s,\n", jsString(ts.Service))
	fmt.Fprintf(&b, "  description:\n    %s,\n", jsString(ts.Description))

	if ts.Notes != "" {
		fmt.Fprintf(&b, "  notes: %s,\n", jsTemplateLiteral(ts.Notes))
	}

	b.WriteString("  rpcs: [\n")
	for _, rpc := range ts.RPCs {
		emitRPC(&b, &rpc)
	}
	b.WriteString("  ],\n")

	if ts.Footer != "" {
		fmt.Fprintf(&b, "  footer: %s,\n", jsTemplateLiteral(ts.Footer))
	}

	b.WriteString("};\n\n")
	b.WriteString("export default service;\n")

	return b.String()
}

func emitRPC(b *strings.Builder, rpc *TSRPC) {
	b.WriteString("    {\n")
	fmt.Fprintf(b, "      name: %s,\n", jsString(rpc.Name))
	fmt.Fprintf(b, "      description:\n        %s,\n", jsString(rpc.Description))

	// Request
	b.WriteString("      request: [\n")
	for _, f := range rpc.Request {
		emitField(b, &f, "        ")
	}
	b.WriteString("      ],\n")

	// Response
	if len(rpc.Response) > 0 {
		b.WriteString("      response: [\n")
		for _, f := range rpc.Response {
			emitField(b, &f, "        ")
		}
		b.WriteString("      ],\n")
	}

	// Errors
	if len(rpc.Errors) > 0 {
		b.WriteString("      errors: [\n")
		for _, e := range rpc.Errors {
			fmt.Fprintf(b, "        { code: %s, description: %s },\n", jsString(e.Code), jsString(e.Description))
		}
		b.WriteString("      ],\n")
	}

	b.WriteString("    },\n")
}

func emitField(b *strings.Builder, f *TSField, indent string) {
	b.WriteString(indent + "{ ")
	fmt.Fprintf(b, "name: %s, ", jsString(f.Name))
	fmt.Fprintf(b, "type: %s, ", jsString(f.Type))
	if f.HasRequired {
		fmt.Fprintf(b, "required: %v, ", f.Required)
	}
	fmt.Fprintf(b, "description: %s", jsString(f.Description))
	if f.Example != nil {
		fmt.Fprintf(b, ", example: %s", jsValue(f.Example))
	}
	b.WriteString(" },\n")
}

func jsString(s string) string {
	// Use JSON encoder with HTML escaping disabled to avoid \u003c / \u003e
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	// Encode appends a newline; trim it
	return strings.TrimSpace(buf.String())
}

func jsValue(v any) string {
	switch val := v.(type) {
	case string:
		return jsString(val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		// Check if it's a whole number
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case []any:
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = jsValue(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		return jsObject(val)
	default:
		data, _ := json.Marshal(v)
		return string(data)
	}
}

func jsObject(m map[string]any) string {
	if len(m) == 0 {
		return "{}"
	}
	var parts []string
	// Iterate in a stable order (sorted keys)
	keys := sortedKeys(m)
	for _, k := range keys {
		v := m[k]
		// Use unquoted key if it's a valid JS identifier
		if isJSIdentifier(k) {
			parts = append(parts, fmt.Sprintf("%s: %s", k, jsValue(v)))
		} else {
			parts = append(parts, fmt.Sprintf("%s: %s", jsString(k), jsValue(v)))
		}
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func jsTemplateLiteral(s string) string {
	// Escape backticks and ${
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "`", "\\`")
	s = strings.ReplaceAll(s, "${", "\\${")
	return "`" + s + "`"
}

func isJSIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, c := range s {
		if i == 0 {
			if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && c != '_' && c != '$' {
				return false
			}
		} else {
			if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' && c != '$' {
				return false
			}
		}
	}
	return true
}

func sortedKeys(m map[string]any) []string {
	// Preserve insertion order by using a simple approach
	// Since Go maps don't preserve order, we sort alphabetically
	// But for JSON-like objects, we want the order from the YAML
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Sort for deterministic output
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

// ---------------------------------------------------------------------------
// Service filename mapping
// ---------------------------------------------------------------------------

func serviceFileName(svcName string) string {
	// WebhookService -> webhook-service.ts
	name := strings.TrimSuffix(svcName, "Service")
	// CamelCase to kebab-case
	var parts []string
	current := ""
	for _, c := range name {
		if c >= 'A' && c <= 'Z' {
			if current != "" {
				parts = append(parts, strings.ToLower(current))
			}
			current = string(c)
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, strings.ToLower(current))
	}
	return strings.Join(parts, "-") + "-service.ts"
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	protoPath := flag.String("proto", "proto/webhook.proto", "Path to proto file")
	overlayPath := flag.String("overlay", "docs/api-overrides.yaml", "Path to overlay YAML")
	outDir := flag.String("outdir", "docs/src/data/api", "Output directory for generated TS files")
	flag.Parse()

	// Parse proto
	pf, err := parseProto(*protoPath)
	if err != nil {
		log.Fatalf("Failed to parse proto: %v", err)
	}
	log.Printf("Parsed proto: %d services, %d messages, %d enums",
		len(pf.Services), len(pf.Messages), len(pf.Enums))

	// Load overlay
	overlay, err := loadOverlay(*overlayPath)
	if err != nil {
		log.Fatalf("Failed to load overlay: %v", err)
	}
	log.Printf("Loaded overlay: %d services configured", len(overlay.Services))

	// Generate TypeScript files
	for _, svcName := range overlay.ServiceOrder {
		ts, err := buildTSService(pf, svcName, overlay)
		if err != nil {
			log.Fatalf("Failed to build %s: %v", svcName, err)
		}

		output := emitService(ts)
		filename := serviceFileName(svcName)
		outPath := filepath.Join(*outDir, filename)

		if err := os.WriteFile(outPath, []byte(output), 0644); err != nil {
			log.Fatalf("Failed to write %s: %v", outPath, err)
		}
		log.Printf("Generated %s (%d RPCs, %d bytes)", filename, len(ts.RPCs), len(output))
	}

	log.Println("Done!")
}
