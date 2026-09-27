package signature

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestVectors runs the shared verification vectors (generated from the
// server's signer, see testdata/vectors.json) that every verify helper in
// client/verify must also pass.
func TestVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vf struct {
		Now              int64 `json:"now"`
		ToleranceSeconds int64 `json:"tolerance_seconds"`
		Cases            []struct {
			Name       string            `json:"name"`
			Scheme     string            `json:"scheme"`
			Key        string            `json:"key"`
			Headers    map[string]string `json:"headers"`
			PayloadB64 string            `json:"payload_b64"`
			Valid      bool              `json:"valid"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &vf); err != nil {
		t.Fatal(err)
	}
	v := Verifier{
		Tolerance: time.Duration(vf.ToleranceSeconds) * time.Second,
		Now:       func() time.Time { return time.Unix(vf.Now, 0) },
	}
	for _, c := range vf.Cases {
		t.Run(c.Name, func(t *testing.T) {
			payload, err := base64.StdEncoding.DecodeString(c.PayloadB64)
			if err != nil {
				t.Fatal(err)
			}
			h := http.Header{}
			for k, val := range c.Headers {
				h.Set(k, val)
			}
			switch c.Scheme {
			case "hmac":
				err = v.VerifyHMAC(payload, h, c.Key)
			case "ed25519":
				err = v.VerifyEd25519(payload, h, c.Key)
			default:
				t.Fatalf("unknown scheme %q", c.Scheme)
			}
			if c.Valid && err != nil {
				t.Fatalf("expected valid, got %v", err)
			}
			if !c.Valid && err == nil {
				t.Fatal("expected verification to fail")
			}
		})
	}
}
