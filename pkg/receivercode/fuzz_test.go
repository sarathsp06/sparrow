package receivercode

import (
	"encoding/json"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"sync"
	"testing"

	"github.com/sarathsp06/sparrow/pkg/receivercode/infer"
)

// sourceImporter type-checks the standard library from source, shared
// across fuzz iterations so each package is loaded once.
var (
	importerOnce sync.Once
	fset         = token.NewFileSet()
	srcImporter  types.Importer
	importerMu   sync.Mutex
)

// FuzzInfer feeds arbitrary bytes to every inference entry point, which must
// never panic, and generates the Go target from any JSON object sample:
// its models and handlers must type-check.
func FuzzInfer(f *testing.F) {
	for _, seed := range []string{
		orderSample, orderSchema, `{}`, `[]`, `null`, `{"a":{"b":{"c":[1,"x",null]}}}`,
		`{"a b":1,"a,b":2,"-":3,"":4,"ü":5,"class":6,"model_x":7}`,
		`{"type":"object","properties":{"x":{"$ref":"#"}}}`,
		`{"oneOf":[{"type":"string"},{"type":"null"}]}`,
		`<a x="1"><b>t</b><b/></a>`, `<a>`, `not json`,
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = infer.FromJSON(data)
		_, _ = infer.FromSchema(data)
		_, _ = infer.FromXML(data)
		_ = DetectFormat(data)

		var obj map[string]any
		if json.Unmarshal(data, &obj) != nil || obj == nil {
			return
		}
		in := Input{
			Target:  Target{Language: "go", Framework: "nethttp"},
			Webhook: Webhook{ID: "wh_fuzz", SigningScheme: SchemeHMAC},
			Subscriptions: []Subscription{
				{ID: "s1", EventType: "fuzz.envelope", Sample: data, Schema: data},
				{ID: "s2", EventType: "fuzz.transform", Sample: data, Transform: `{{json .payload}}`},
			},
		}
		out, err := Generate(in)
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		typeCheckGo(t, out)
	})
}

// typeCheckGo type-checks the generated models.go and handlers.go.
func typeCheckGo(t *testing.T, out Output) {
	t.Helper()
	importerOnce.Do(func() { srcImporter = importer.ForCompiler(fset, "source", nil) })
	importerMu.Lock()
	defer importerMu.Unlock()
	var files []*ast.File
	for _, f := range out.Files {
		if f.Path != "models.go" && f.Path != "handlers.go" {
			continue
		}
		parsed, err := parser.ParseFile(fset, f.Path, f.Content, 0)
		if err != nil {
			t.Fatalf("%s does not parse: %v\n%s", f.Path, err, f.Content)
		}
		files = append(files, parsed)
	}
	conf := types.Config{Importer: srcImporter}
	if _, err := conf.Check("main", fset, files, nil); err != nil {
		t.Fatalf("generated Go does not type-check: %v", err)
	}
}
