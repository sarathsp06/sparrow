package receivercode

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"strings"
)

// SpecVersion is the version of the Spec format.
const SpecVersion = 1

// Defaults written into generated receivers, each behind a named constant.
const (
	// DefaultToleranceSeconds is how far webhook-timestamp may be from the
	// receiver's clock.
	DefaultToleranceSeconds = 300
	// DefaultDedupCapacity is how many webhook-ids the in-memory dedup
	// remembers.
	DefaultDedupCapacity = 10000
	// UnknownEventStatus answers a verified delivery no handler matches.
	// It is a 4xx so the delivery fails visibly in Sparrow.
	UnknownEventStatus = 422
)

// Environment variables a generated receiver reads.
const (
	EnvSecret    = "SPARROW_WEBHOOK_SECRET"
	EnvPublicKey = "SPARROW_WEBHOOK_PUBLIC_KEY"
	EnvTestMode  = "SPARROW_RECEIVER_TEST_MODE"
	EnvPort      = "PORT"
)

// Paths a generated receiver serves.
const (
	WebhookPath = "/webhook"
	HealthPath  = "/healthz"
)

// Spec is the machine-readable description of a generated receiver: how it
// verifies and routes deliveries, what each handler decodes, and the test
// vectors its tests replay.
type Spec struct {
	SpecVersion   int                `json:"spec_version"`
	Target        string             `json:"target"`
	Webhook       SpecWebhook        `json:"webhook"`
	Receiver      SpecReceiver       `json:"receiver"`
	Subscriptions []SpecSubscription `json:"subscriptions"`
	Test          SpecTest           `json:"test"`
	Vectors       []Vector           `json:"vectors"`
}

// SpecWebhook is the webhook the receiver is for.
type SpecWebhook struct {
	ID            string `json:"id"`
	Consumer      string `json:"consumer,omitempty"`
	SigningScheme string `json:"signing_scheme"`
	// PublicKey is the webhook's real Ed25519 public key (hex), embedded in
	// the generated code. Empty for HMAC.
	PublicKey string `json:"public_key,omitempty"`
}

// SpecReceiver is the runtime contract of the generated receiver.
type SpecReceiver struct {
	WebhookPath        string   `json:"webhook_path"`
	HealthPath         string   `json:"health_path"`
	SecretEnv          string   `json:"secret_env,omitempty"`
	PublicKeyEnv       string   `json:"public_key_env,omitempty"`
	TestModeEnv        string   `json:"test_mode_env"`
	PortEnv            string   `json:"port_env"`
	DefaultPort        int      `json:"default_port"`
	ToleranceSeconds   int      `json:"tolerance_seconds"`
	DedupCapacity      int      `json:"dedup_capacity"`
	DispatchHeaders    []string `json:"dispatch_headers"`
	UnknownEventStatus int      `json:"unknown_event_status"`
}

// SpecSubscription describes one handler.
type SpecSubscription struct {
	ID           string `json:"id"`
	EventType    string `json:"event_type"`
	EventVersion int    `json:"event_version,omitempty"`
	Handler      string `json:"handler"`
	// Format is the body format: json, xml or text.
	Format      string `json:"format"`
	ContentType string `json:"content_type,omitempty"`
	// Model is how the body is decoded: envelope, json, xml or raw.
	Model string `json:"model"`
	// ModelSource is where the model's type came from: schema or sample.
	ModelSource string   `json:"model_source,omitempty"`
	Notes       []string `json:"notes,omitempty"`
	RenderError string   `json:"render_error,omitempty"`
}

// SpecTest holds the throwaway credentials the vectors are signed with.
// They are derived from the webhook ID and are never the webhook's real
// secret or key.
type SpecTest struct {
	// Now is the Unix time the vectors were signed at; tests fix their
	// clock to it.
	Now int64 `json:"now"`
	// Secret is the throwaway HMAC secret (whsec_ format).
	Secret      string `json:"secret"`
	WrongSecret string `json:"wrong_secret"`
	// PublicKey and PrivateKeySeed are a throwaway Ed25519 keypair (hex).
	PublicKey           string `json:"public_key"`
	PrivateKeySeed      string `json:"private_key_seed"`
	WrongPrivateKeySeed string `json:"wrong_private_key_seed"`
}

// Vector is one signed request and what the receiver must do with it.
// Vectors are replayed in order against one receiver: "duplicate" repeats
// "valid".
type Vector struct {
	Name           string `json:"name"`
	SubscriptionID string `json:"subscription_id"`
	// MessageID, Timestamp, Signatures and Tamper say how the request was
	// built, so a harness can rebuild it at another time: Timestamp is
	// relative to Now, Signatures name the scheme and key ("v1:test",
	// "v1:wrong", "v1a:test", "v1a:wrong"), Tamper alters the body after
	// signing, and no Signatures means no webhook-signature header.
	MessageID       string   `json:"message_id"`
	TimestampOffset int64    `json:"timestamp_offset"`
	Signatures      []string `json:"signatures"`
	Tamper          bool     `json:"tamper,omitempty"`
	// OmitSubscriptionHeader drops X-Sparrow-Subscription-ID to exercise
	// the event type fallback; EventType overrides the event type header.
	OmitSubscriptionHeader bool   `json:"omit_subscription_header,omitempty"`
	EventType              string `json:"event_type"`
	EventVersion           int    `json:"event_version,omitempty"`
	ContentType            string `json:"content_type"`
	// Headers and Body are the request as signed at Test.Now.
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	Expect  Expect            `json:"expect"`
}

// Expect is a vector's expected outcome.
type Expect struct {
	Status int `json:"status"`
	// Handler is the handler that must be called, or "" for none.
	Handler string `json:"handler"`
}

// testCreds derives the throwaway credentials from the webhook ID, so they
// are stable across runs and unrelated to the real secret.
func testCreds(webhookID string) SpecTest {
	derive := func(label string) []byte {
		sum := sha256.Sum256([]byte("sparrow-receivercode/" + label + "/" + webhookID))
		return sum[:]
	}
	seed := derive("ed25519")
	pub := ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)
	return SpecTest{
		Now:                 sampleTime.Unix(),
		Secret:              "whsec_" + base64.StdEncoding.EncodeToString(derive("hmac")),
		WrongSecret:         "whsec_" + base64.StdEncoding.EncodeToString(derive("hmac-wrong")),
		PublicKey:           hex.EncodeToString(pub),
		PrivateKeySeed:      hex.EncodeToString(seed),
		WrongPrivateKeySeed: hex.EncodeToString(derive("ed25519-wrong")),
	}
}

// Sign returns the webhook-signature header value for a message signed as
// sigs describe (see Vector.Signatures).
func (t SpecTest) Sign(msgID string, timestamp int64, body []byte, sigs []string) string {
	msg := []byte(msgID + "." + strconv.FormatInt(timestamp, 10) + ".")
	msg = append(msg, body...)
	parts := make([]string, 0, len(sigs))
	for _, s := range sigs {
		switch s {
		case "v1:test", "v1:wrong":
			secret := t.Secret
			if s == "v1:wrong" {
				secret = t.WrongSecret
			}
			key, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
			mac := hmac.New(sha256.New, key)
			mac.Write(msg)
			parts = append(parts, "v1,"+base64.StdEncoding.EncodeToString(mac.Sum(nil)))
		case "v1a:test", "v1a:wrong":
			seedHex := t.PrivateKeySeed
			if s == "v1a:wrong" {
				seedHex = t.WrongPrivateKeySeed
			}
			seed, _ := hex.DecodeString(seedHex)
			sig := ed25519.Sign(ed25519.NewKeyFromSeed(seed), msg)
			parts = append(parts, "v1a,"+base64.StdEncoding.EncodeToString(sig))
		}
	}
	return strings.Join(parts, " ")
}

// TamperBody is how a vector with Tamper alters the body after signing.
func TamperBody(body []byte) []byte { return append(append([]byte{}, body...), ' ') }

// Request builds the headers and body of v signed at now with t's
// credentials.
func (t SpecTest) Request(v Vector, now int64, body []byte) (map[string]string, []byte) {
	ts := now + v.TimestampOffset
	h := map[string]string{
		"Content-Type":         v.ContentType,
		"webhook-id":           v.MessageID,
		"webhook-timestamp":    strconv.FormatInt(ts, 10),
		"X-Sparrow-Event-Type": v.EventType,
	}
	if v.EventVersion > 0 {
		h["X-Sparrow-Event-Version"] = strconv.Itoa(v.EventVersion)
	}
	if !v.OmitSubscriptionHeader {
		h["X-Sparrow-Subscription-ID"] = v.SubscriptionID
	}
	if len(v.Signatures) > 0 {
		h["webhook-signature"] = t.Sign(v.MessageID, ts, body, v.Signatures)
	}
	if v.Tamper {
		body = TamperBody(body)
	}
	return h, body
}

// Rebuild re-signs v at now: the request a harness sends to a running
// receiver, whose clock is real. Tampered vectors are tampered again after
// signing.
func (t SpecTest) Rebuild(v Vector, now int64) (map[string]string, []byte) {
	body := []byte(v.Body)
	if v.Tamper && len(body) > 0 {
		body = body[:len(body)-1]
	}
	return t.Request(v, now, body)
}

// buildVectors makes the vectors for every subscription, in plan order.
func buildVectors(plans []*plan, scheme string, creds SpecTest) []Vector {
	good, bad := []string{"v1:test"}, []string{"v1:wrong"}
	if scheme == SchemeEd25519 {
		good, bad = []string{"v1a:test"}, []string{"v1a:wrong"}
	}
	firstByType := map[string]*plan{}
	for _, p := range plans {
		if _, ok := firstByType[p.Sub.EventType]; !ok {
			firstByType[p.Sub.EventType] = p
		}
	}

	var out []Vector
	for i, p := range plans {
		ok := Expect{Status: 200, Handler: handlerName(p)}
		unauthorized := Expect{Status: 401}
		id := func(name string) string { return "msg_test_" + strconv.Itoa(i+1) + "_" + name }
		cases := []Vector{
			{Name: "valid", MessageID: id("valid"), Signatures: good, Expect: ok},
			{Name: "duplicate", MessageID: id("valid"), Signatures: good, Expect: Expect{Status: 200}},
			{Name: "tampered_body", MessageID: id("tampered"), Signatures: good, Tamper: true, Expect: unauthorized},
			{Name: "wrong_key", MessageID: id("wrong_key"), Signatures: bad, Expect: unauthorized},
			{Name: "timestamp_too_old", MessageID: id("old"), TimestampOffset: -2 * DefaultToleranceSeconds, Signatures: good, Expect: unauthorized},
			{Name: "timestamp_in_future", MessageID: id("future"), TimestampOffset: 2 * DefaultToleranceSeconds, Signatures: good, Expect: unauthorized},
			{Name: "missing_signature", MessageID: id("unsigned"), Expect: unauthorized},
		}
		if scheme == SchemeEd25519 {
			cases = append(cases,
				Vector{Name: "valid_v1a_invalid_v1", MessageID: id("v1a_ok"), Signatures: []string{"v1:wrong", "v1a:test"}, Expect: ok},
				Vector{Name: "invalid_v1a_valid_v1", MessageID: id("v1_only"), Signatures: []string{"v1:test", "v1a:wrong"}, Expect: unauthorized},
			)
		}
		first := firstByType[p.Sub.EventType]
		cases = append(cases,
			Vector{Name: "event_type_fallback", MessageID: id("fallback"), Signatures: good, OmitSubscriptionHeader: true,
				Expect: Expect{Status: 200, Handler: handlerName(first)}},
		)
		if i == len(plans)-1 {
			cases = append(cases, Vector{Name: "unknown_event", MessageID: id("unknown"), Signatures: good,
				OmitSubscriptionHeader: true, EventType: "sparrow.receivercode.unknown", Expect: Expect{Status: UnknownEventStatus}})
		}
		for _, v := range cases {
			v.SubscriptionID = p.Sub.ID
			v.EventVersion = p.Sub.EventVersion
			v.ContentType = p.ContentType
			if v.ContentType == "" {
				v.ContentType = "application/json"
			}
			if v.EventType == "" {
				v.EventType = p.Sub.EventType
			}
			body := p.Body
			if v.Name == "event_type_fallback" {
				// The fallback handler decodes with its own model.
				body = first.Body
			}
			headers, sent := creds.Request(v, creds.Now, body)
			v.Headers, v.Body = headers, string(sent)
			out = append(out, v)
		}
	}
	return out
}

// handlerName is the language-neutral handler name used in the spec and in
// test-mode output.
func handlerName(p *plan) string { return "handle" + p.Name }
