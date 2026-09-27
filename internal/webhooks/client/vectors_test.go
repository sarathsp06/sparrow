package client

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Signature verification vectors shared by every verify helper
// (pkg/signature and client/verify/<lang>). They are produced by the same
// signing functions deliveries use, so a change to the signer that the
// helpers don't follow fails here first.
//
// Regenerate after an intentional signing change:
//
//	go test ./internal/webhooks/client -run TestSignatureVectors -update-vectors
var updateVectors = flag.Bool("update-vectors", false, "rewrite pkg/signature/testdata vectors")

const vectorDir = "../../../pkg/signature/testdata"

// vectorNow is the verifier clock every vector is checked against.
const (
	vectorNow       = 1790000000
	vectorTolerance = 300
)

type vector struct {
	Name string `json:"name"`
	// Scheme is "hmac" (verify with Key as the webhook secret) or "ed25519"
	// (verify with Key as the hex public key).
	Scheme     string            `json:"scheme"`
	Key        string            `json:"key"`
	Headers    map[string]string `json:"headers"`
	PayloadB64 string            `json:"payload_b64"`
	Valid      bool              `json:"valid"`
}

type vectorFile struct {
	Comment          string   `json:"_comment"`
	Now              int64    `json:"now"`
	ToleranceSeconds int64    `json:"tolerance_seconds"`
	Cases            []vector `json:"cases"`
}

func buildVectors(t *testing.T) vectorFile {
	t.Helper()
	const (
		whsec  = "whsec_MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw"
		raw    = "plain-shared-secret"
		msgID  = "msg_4f1c2a8e-delivery"
		other  = "whsec_" + "b3RoZXItc2VjcmV0LW90aGVyLXNlY3JldA=="
		broken = "whsec_!!not-base64!!"
	)
	payload := []byte("{\"order_id\":\"ord_123\",\"note\":\"café ☕\\n\"}\n")
	seed := bytes.Repeat([]byte{7}, ed25519.SeedSize)
	priv := ed25519.NewKeyFromSeed(seed)
	pubHex := hex.EncodeToString(priv.Public().(ed25519.PublicKey))
	otherPubHex := hex.EncodeToString(ed25519.NewKeyFromSeed(bytes.Repeat([]byte{9}, ed25519.SeedSize)).Public().(ed25519.PublicKey))

	ts := func(offset int64) string { return strconv.FormatInt(vectorNow+offset, 10) }
	hmacSig := func(secret, stamp string, body []byte) string {
		sig, err := generateHMACSignature(body, secret, msgID, stamp)
		if err != nil {
			t.Fatal(err)
		}
		return "v1," + sig
	}
	edSig := func(stamp string, body []byte) string {
		return "v1a," + generateEd25519Signature(body, priv, msgID, stamp)
	}
	hdr := func(stamp, sig string) map[string]string {
		return map[string]string{"webhook-id": msgID, "webhook-timestamp": stamp, "webhook-signature": sig}
	}
	b64 := func(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
	p := b64(payload)
	now := ts(0)
	good := hmacSig(whsec, now, payload)
	goodEd := edSig(now, payload)

	cases := []vector{
		{"hmac valid whsec secret", "hmac", whsec, hdr(now, good), p, true},
		{"hmac valid raw secret", "hmac", raw, hdr(now, hmacSig(raw, now, payload)), p, true},
		{"hmac tampered body", "hmac", whsec, hdr(now, good), b64(append(bytes.Clone(payload[:len(payload)-1]), ' ')), false},
		{"hmac wrong secret", "hmac", other, hdr(now, good), p, false},
		{"hmac invalid whsec base64", "hmac", broken, hdr(now, good), p, false},
		{"hmac empty secret", "hmac", "", hdr(now, hmacSig("", now, payload)), p, false},
		{"hmac empty whsec secret", "hmac", "whsec_", hdr(now, hmacSig("whsec_", now, payload)), p, false},
		{"hmac timestamp at past tolerance edge", "hmac", whsec, hdr(ts(-vectorTolerance), hmacSig(whsec, ts(-vectorTolerance), payload)), p, true},
		{"hmac timestamp past tolerance", "hmac", whsec, hdr(ts(-vectorTolerance-1), hmacSig(whsec, ts(-vectorTolerance-1), payload)), p, false},
		{"hmac timestamp at future tolerance edge", "hmac", whsec, hdr(ts(vectorTolerance), hmacSig(whsec, ts(vectorTolerance), payload)), p, true},
		{"hmac timestamp beyond future tolerance", "hmac", whsec, hdr(ts(vectorTolerance+1), hmacSig(whsec, ts(vectorTolerance+1), payload)), p, false},
		{"hmac non-numeric timestamp", "hmac", whsec, hdr("17900000x0", good), p, false},
		{"hmac signature for another timestamp", "hmac", whsec, hdr(ts(1), good), p, false},
		{"hmac rotation: second signature matches", "hmac", whsec, hdr(now, hmacSig(other, now, payload)+" "+good), p, true},
		{"hmac extra whitespace between signatures", "hmac", whsec, hdr(now, "  "+hmacSig(other, now, payload)+"   "+good+" "), p, true},
		{"hmac unknown versions are ignored", "hmac", whsec, hdr(now, "v2,AAAA v0,"+strings.TrimPrefix(good, "v1,")+" "+good), p, true},
		{"hmac invalid base64 candidate skipped", "hmac", whsec, hdr(now, "v1,@@@ "+good), p, true},
		{"hmac only v1a present", "hmac", whsec, hdr(now, goodEd), p, false},
		{"hmac v1 prefix required", "hmac", whsec, hdr(now, strings.TrimPrefix(good, "v1,")), p, false},
		{"hmac mixed-case header names", "hmac", whsec, map[string]string{"Webhook-Id": msgID, "WEBHOOK-TIMESTAMP": now, "Webhook-Signature": good}, p, true},
		{"hmac missing webhook-id", "hmac", whsec, map[string]string{"webhook-timestamp": now, "webhook-signature": good}, p, false},
		{"hmac missing webhook-timestamp", "hmac", whsec, map[string]string{"webhook-id": msgID, "webhook-signature": good}, p, false},
		{"hmac missing webhook-signature", "hmac", whsec, map[string]string{"webhook-id": msgID, "webhook-timestamp": now}, p, false},
		{"hmac empty body", "hmac", whsec, hdr(now, hmacSig(whsec, now, nil)), "", true},
		{"ed25519 valid", "ed25519", pubHex, hdr(now, goodEd), p, true},
		{"ed25519 valid alongside v1", "ed25519", pubHex, hdr(now, good+" "+goodEd), p, true},
		{"ed25519 tampered body", "ed25519", pubHex, hdr(now, goodEd), b64([]byte("{}")), false},
		{"ed25519 wrong public key", "ed25519", otherPubHex, hdr(now, goodEd), p, false},
		{"ed25519 invalid hex key", "ed25519", "zz" + pubHex[2:], hdr(now, goodEd), p, false},
		{"ed25519 short key", "ed25519", pubHex[:62], hdr(now, goodEd), p, false},
		{"ed25519 timestamp past tolerance", "ed25519", pubHex, hdr(ts(-vectorTolerance-1), edSig(ts(-vectorTolerance-1), payload)), p, false},
		{"ed25519 only v1 present", "ed25519", pubHex, hdr(now, good), p, false},
	}
	return vectorFile{
		Comment:          "Generated by internal/webhooks/client TestSignatureVectors with the production signer. Verify each case with the given key, headers and base64 payload at clock `now` with `tolerance_seconds`; `valid` is the expected outcome. Do not edit by hand.",
		Now:              vectorNow,
		ToleranceSeconds: vectorTolerance,
		Cases:            cases,
	}
}

// encodeTSV renders the vectors for languages without a JSON parser in the
// standard library (Java, Kotlin): a "# now <n> tolerance <n>" line, then
// one tab-separated line per case:
//
//	name  scheme  valid(1|0)  base64(key)  base64(payload)  headers
//
// where headers is a comma-separated list of base64(name):base64(value).
func encodeTSV(vf vectorFile) []byte {
	var b strings.Builder
	b64 := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	b.WriteString("# now " + strconv.FormatInt(vf.Now, 10) + " tolerance " + strconv.FormatInt(vf.ToleranceSeconds, 10) + "\n")
	for _, c := range vf.Cases {
		names := make([]string, 0, len(c.Headers))
		for k := range c.Headers {
			names = append(names, k)
		}
		sort.Strings(names)
		pairs := make([]string, 0, len(names))
		for _, k := range names {
			pairs = append(pairs, b64(k)+":"+b64(c.Headers[k]))
		}
		valid := "0"
		if c.Valid {
			valid = "1"
		}
		b.WriteString(strings.Join([]string{c.Name, c.Scheme, valid, b64(c.Key), c.PayloadB64, strings.Join(pairs, ",")}, "\t") + "\n")
	}
	return []byte(b.String())
}

func TestSignatureVectors(t *testing.T) {
	vf := buildVectors(t)
	jsonBytes, err := json.MarshalIndent(vf, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	jsonBytes = append(jsonBytes, '\n')
	files := map[string][]byte{
		"vectors.json": jsonBytes,
		"vectors.tsv":  encodeTSV(vf),
	}
	for name, want := range files {
		path := filepath.Join(vectorDir, name)
		if *updateVectors {
			if err := os.MkdirAll(vectorDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, want, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v (regenerate with -update-vectors)", path, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%s is stale: the signer changed; regenerate with `go test ./internal/webhooks/client -run TestSignatureVectors -update-vectors` and make every verify helper pass", path)
		}
	}
}
