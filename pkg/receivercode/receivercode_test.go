package receivercode

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sarathsp06/sparrow/pkg/signature"
)

func TestTargets(t *testing.T) {
	got := Targets()
	want := []string{"go-nethttp", "node-express", "python-fastapi"}
	if len(got) != len(want) {
		t.Fatalf("Targets() = %v", got)
	}
	for i, ti := range got {
		if ti.ID != want[i] || ti.Target.ID() != want[i] || ti.Name == "" || ti.Runtime == "" {
			t.Errorf("Targets()[%d] = %+v, want ID %s", i, ti, want[i])
		}
	}
}

func TestGenerateRejectsBadInput(t *testing.T) {
	good := testCases[0].in
	good.Target = Target{Language: "go", Framework: "nethttp"}
	if _, err := Generate(good); err != nil {
		t.Fatalf("good input: %v", err)
	}
	cases := map[string]func(in *Input){
		"unknown target": func(in *Input) { in.Target = Target{Language: "cobol", Framework: "cics"} },
		"unknown scheme": func(in *Input) { in.Webhook.SigningScheme = "rsa" },
		"ed25519 no key": func(in *Input) { in.Webhook.SigningScheme = SchemeEd25519 },
		"ed25519 bad key": func(in *Input) {
			in.Webhook.SigningScheme, in.Webhook.Ed25519PublicKey = SchemeEd25519, strings.Repeat("zz", 32)
		},
		"no webhook id":     func(in *Input) { in.Webhook.ID = "" },
		"quoted webhook id": func(in *Input) { in.Webhook.ID = `wh"x` },
		"consumer newline":  func(in *Input) { in.Webhook.Consumer = "a\nb" },
		"no subscriptions":  func(in *Input) { in.Subscriptions = nil },
		"empty sub id":      func(in *Input) { in.Subscriptions = []Subscription{{EventType: "a.b"}} },
		"duplicate sub id": func(in *Input) {
			in.Subscriptions = []Subscription{{ID: "s", EventType: "a"}, {ID: "s", EventType: "b"}}
		},
		"quoted sub id":      func(in *Input) { in.Subscriptions = []Subscription{{ID: "s'1", EventType: "a"}} },
		"invalid event type": func(in *Input) { in.Subscriptions = []Subscription{{ID: "s", EventType: `a"; drop`}} },
	}
	for name, mutate := range cases {
		in := good
		in.Webhook = good.Webhook
		mutate(&in)
		if _, err := Generate(in); err == nil {
			t.Errorf("%s: Generate succeeded, want error", name)
		}
	}
}

// TestVectorsAgainstSignature checks every vector with pkg/signature, the
// authoritative verifier: vectors expecting a handler or a duplicate verify,
// the 401 ones do not.
func TestVectorsAgainstSignature(t *testing.T) {
	for _, tc := range testCases {
		in := tc.in
		in.Target = Target{Language: "go", Framework: "nethttp"}
		out, err := Generate(in)
		if err != nil {
			t.Fatal(err)
		}
		v := signature.Verifier{Now: func() time.Time { return time.Unix(out.Spec.Test.Now, 0) }}
		for _, vec := range out.Spec.Vectors {
			h := http.Header{}
			for k, val := range vec.Headers {
				h.Set(k, val)
			}
			if in.Webhook.SigningScheme == SchemeEd25519 {
				err = v.VerifyEd25519([]byte(vec.Body), h, out.Spec.Test.PublicKey)
			} else {
				err = v.VerifyHMAC([]byte(vec.Body), h, out.Spec.Test.Secret)
			}
			if wantOK := vec.Expect.Status != 401; (err == nil) != wantOK {
				t.Errorf("%s/%s/%s: verify err = %v, want ok=%v", tc.name, vec.SubscriptionID, vec.Name, err, wantOK)
			}
		}
	}
}

// TestRequestRebuildsAtAnotherTime checks the recipe a contract harness
// uses: re-signing a valid vector now verifies with the real clock.
func TestRequestRebuildsAtAnotherTime(t *testing.T) {
	for _, tc := range testCases {
		in := tc.in
		in.Target = Target{Language: "python", Framework: "fastapi"}
		out, err := Generate(in)
		if err != nil {
			t.Fatal(err)
		}
		now := time.Now().Unix()
		for _, vec := range out.Spec.Vectors {
			if vec.Name != "valid" {
				continue
			}
			headers, body := out.Spec.Test.Rebuild(vec, now)
			h := http.Header{}
			for k, val := range headers {
				h.Set(k, val)
			}
			if in.Webhook.SigningScheme == SchemeEd25519 {
				err = signature.VerifyEd25519(body, h, out.Spec.Test.PublicKey)
			} else {
				err = signature.VerifyHMAC(body, h, out.Spec.Test.Secret)
			}
			if err != nil {
				t.Errorf("%s/%s: rebuilt request does not verify: %v", tc.name, vec.SubscriptionID, err)
			}
			if h.Get("X-Sparrow-Subscription-ID") != vec.SubscriptionID || h.Get("X-Sparrow-Event-Type") == "" {
				t.Errorf("%s: routing headers missing: %v", tc.name, h)
			}
		}
	}
}

// TestRebuildMatchesTheVectors rebuilds every vector at its own time: the
// result must be the vector itself.
func TestRebuildMatchesTheVectors(t *testing.T) {
	for _, tc := range testCases {
		in := tc.in
		in.Target = Target{Language: "go", Framework: "nethttp"}
		out, err := Generate(in)
		if err != nil {
			t.Fatal(err)
		}
		for _, vec := range out.Spec.Vectors {
			headers, body := out.Spec.Test.Rebuild(vec, out.Spec.Test.Now)
			if string(body) != vec.Body || len(headers) != len(vec.Headers) {
				t.Fatalf("%s/%s: rebuilt request differs", tc.name, vec.Name)
			}
			for k, v := range vec.Headers {
				if headers[k] != v {
					t.Errorf("%s/%s: %s = %q, want %q", tc.name, vec.Name, k, headers[k], v)
				}
			}
		}
	}
}

func TestTestCredentialsAreNotSecrets(t *testing.T) {
	a, b := testCreds("wh_1"), testCreds("wh_2")
	if a.Secret == b.Secret || a.PublicKey == b.PublicKey {
		t.Fatal("credentials do not depend on the webhook ID")
	}
	if a.Secret == a.WrongSecret || a.PrivateKeySeed == a.WrongPrivateKeySeed {
		t.Fatal("wrong credentials equal the right ones")
	}
	if testCreds("wh_1") != a {
		t.Fatal("credentials are not deterministic")
	}
}

func TestOutputNeverContainsSecretEnvValue(t *testing.T) {
	t.Setenv(EnvSecret, "whsec_c2VjcmV0LXRoYXQtbXVzdC1ub3QtbGVhaw==")
	for _, target := range Targets() {
		in := testCases[0].in
		in.Target = target.Target
		out, err := Generate(in)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range out.Files {
			if strings.Contains(f.Content, "c2VjcmV0LXRoYXQtbXVzdC1ub3QtbGVhaw") {
				t.Fatalf("%s/%s embeds the environment's secret", target.ID, f.Path)
			}
		}
	}
}

func TestSpecDescribesEachSubscription(t *testing.T) {
	in := testCases[len(testCases)-1].in // multi
	in.Target = Target{Language: "node", Framework: "express"}
	out, err := Generate(in)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]SpecSubscription{}
	for _, s := range out.Spec.Subscriptions {
		byID[s.ID] = s
	}
	checks := []struct{ id, model, source, handler string }{
		{"sub_a", ModelEnvelope, "schema", "handleOrderPaid"},
		{"sub_b", ModelJSON, "sample", "handleOrderPaid2"},
		{"sub_c", ModelEnvelope, "sample", "handleInvoiceFailedV2"},
		{"sub_d", ModelRaw, "", "handleInvoiceVoided"},
	}
	for _, c := range checks {
		s := byID[c.id]
		if s.Model != c.model || s.ModelSource != c.source || s.Handler != c.handler {
			t.Errorf("%s = %+v, want model %s from %q, handler %s", c.id, s, c.model, c.source, c.handler)
		}
	}
	if byID["sub_d"].RenderError == "" {
		t.Error("sub_d: render error not recorded")
	}
	if out.Run != "SPARROW_WEBHOOK_SECRET=<your webhook secret> npm start" || out.Test != "npm test" {
		t.Errorf("run, test = %q, %q", out.Run, out.Test)
	}
	var round Spec
	b, _ := json.Marshal(out.Spec)
	if err := json.Unmarshal(b, &round); err != nil || len(round.Vectors) != len(out.Spec.Vectors) {
		t.Fatalf("spec does not round-trip: %v", err)
	}
}

func TestFormatDetection(t *testing.T) {
	for body, want := range map[string]string{
		`{"a":1}`:                             FormatJSON,
		` [1, 2] `:                            FormatJSON,
		`{"a":`:                               FormatText,
		`<a><b/></a>`:                         FormatXML,
		`<?xml version="1.0"?><a x="1">t</a>`: FormatXML,
		`<a></b>`:                             FormatText,
		`<a/><b/>`:                            FormatText,
		`<a/>trailing`:                        FormatText,
		`hello`:                               FormatText,
		``:                                    FormatText,
	} {
		if got := DetectFormat([]byte(body)); got != want {
			t.Errorf("DetectFormat(%q) = %s, want %s", body, got, want)
		}
	}
	for ct, want := range map[string]string{
		"application/json":                  FormatJSON,
		"application/vnd.api+json":          FormatJSON,
		"text/xml; charset=utf-8":           FormatXML,
		"application/atom+xml":              FormatXML,
		"application/x-www-form-urlencoded": FormatText,
		"":                                  "",
		";;":                                "",
	} {
		if got := formatFromContentType(ct); got != want {
			t.Errorf("formatFromContentType(%q) = %q, want %q", ct, got, want)
		}
	}
}

func TestContentTypeOverridesDetection(t *testing.T) {
	p := analyze(Subscription{ID: "s", EventType: "e", Sample: json.RawMessage(`{"a":1}`),
		Transform: `{"a": {{.payload.a}}}`, Headers: map[string]string{"Content-Type": "text/plain"}})
	if p.Format != FormatText || p.Model != ModelRaw {
		t.Errorf("format, model = %s, %s; want text, raw", p.Format, p.Model)
	}
	p = analyze(Subscription{ID: "s", EventType: "e", Sample: json.RawMessage(`{"a":1}`),
		Transform: `not json`, Headers: map[string]string{"Content-Type": "application/json"}})
	if p.Format != FormatJSON || p.Model != ModelRaw || len(p.Notes) == 0 {
		t.Errorf("JSON content type with text body: %+v", p)
	}
	p = analyze(Subscription{ID: "s", EventType: "e", Sample: json.RawMessage(`{"a":1}`),
		Transform: `<a>`, Headers: map[string]string{"Content-Type": "application/xml"}})
	if p.Format != FormatXML || p.Model != ModelRaw {
		t.Errorf("XML content type with broken body: %+v", p)
	}
}

func TestSampleEdgeCases(t *testing.T) {
	for name, sample := range map[string]string{"none": "", "null": "null", "array": "[1]", "invalid": "{"} {
		p := analyze(Subscription{ID: "s", EventType: "e", Sample: json.RawMessage(sample)})
		if p.Model != ModelEnvelope || len(p.Notes) == 0 {
			t.Errorf("%s: %+v", name, p)
		}
	}
	// The schema types the payload, even when the sample disagrees with it.
	p := analyze(Subscription{ID: "s", EventType: "e", Schema: json.RawMessage(`{"type":"object","required":["n"],"properties":{"n":{"type":"integer"}}}`),
		Sample: json.RawMessage(`{"n": "not a number"}`)})
	if p.ModelSource != "schema" || p.Type.Fields[0].Type.Kind.String() != "int" {
		t.Errorf("schema typing: %+v", p)
	}
	// Without a sample, placeholders from the schema make the sample body.
	p = analyze(Subscription{ID: "s", EventType: "e", Schema: json.RawMessage(`{"type":"object","required":["n"],"properties":{"n":{"type":"integer"}}}`)})
	if !strings.Contains(string(p.Body), `"payload":{"n":1}`) {
		t.Errorf("placeholder body = %s", p.Body)
	}
	// An unreadable schema is noted and the sample is used.
	p = analyze(Subscription{ID: "s", EventType: "e", Schema: json.RawMessage(`[`), Sample: json.RawMessage(`{"a":true}`)})
	if p.ModelSource != "sample" {
		t.Errorf("bad schema: %+v", p)
	}
}

func TestPascalAndSnake(t *testing.T) {
	for in, want := range map[string]string{"order.paid": "OrderPaid", "a-b_c d": "ABCD", "9lives": "X9lives", "": "X", "ünï": "N"} {
		if got := pascal(in); got != want {
			t.Errorf("pascal(%q) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[string]string{"OrderPaid2": "order_paid2", "userID": "user_id", "HTTPServer": "http_server", "a--b": "a_b"} {
		if got := snake(in); got != want {
			t.Errorf("snake(%q) = %q, want %q", in, got, want)
		}
	}
	used := map[string]bool{}
	for in, want := range map[string]string{"class": "class_", "2fa": "f_2fa", "model_type": "f_model_type", "_": "field", "json": "json_"} {
		if got := pyIdent(in, used); got != want {
			t.Errorf("pyIdent(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUnrepresentableGoKeysBecomeMaps(t *testing.T) {
	in := Input{
		Target:  Target{Language: "go", Framework: "nethttp"},
		Webhook: Webhook{ID: "wh", SigningScheme: SchemeHMAC},
		Subscriptions: []Subscription{{ID: "s", EventType: "e",
			Sample: json.RawMessage(`{"ok": 1, "bad,key": 2, "-": 3}`)}},
	}
	out, err := Generate(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range out.Files {
		if f.Path == "models.go" && !strings.Contains(f.Content, "Payload   map[string]any") {
			t.Errorf("models.go:\n%s", f.Content)
		}
	}
}

// TestVendoredHelpersAreIdentical fails when a vendored helper drifts from
// its original. Outside the repository (a module download) it skips.
func TestVendoredHelpersAreIdentical(t *testing.T) {
	pairs := map[string]string{
		"helpers/signature.go.txt":  "../signature/signature.go",
		"helpers/sparrow_verify.py": "../../client/verify/python/sparrow_verify.py",
		"helpers/sparrow-verify.ts": "../../client/verify/js/sparrow-verify.ts",
	}
	for copyPath, original := range pairs {
		want, err := os.ReadFile(filepath.FromSlash(original))
		if os.IsNotExist(err) {
			t.Skipf("%s not found: not in the Sparrow repository", original)
		}
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.FromSlash(copyPath))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Errorf("%s differs from %s: copy the original over it", copyPath, original)
		}
	}
	// And the generated projects carry them unchanged.
	for _, target := range Targets() {
		in := testCases[0].in
		in.Target = target.Target
		out, _ := Generate(in)
		found := false
		for _, f := range out.Files {
			if f.Content == goHelper || f.Content == pythonHelper || f.Content == tsHelper {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: no file is the vendored helper", target.ID)
		}
	}
}
