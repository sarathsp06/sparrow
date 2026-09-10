package client

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"testing"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/signature"
)

// TestBuildRequestSignaturesVerifiable guards the signer/verifier contract:
// headers produced by BuildRequest must verify with the public
// pkg/signature package that consumers use.
func TestBuildRequestSignaturesVerifiable(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	payload := []byte(`{"event":"order.created","payload":{"order_id":"ord_123"}}`)
	const secret = "whsec_MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw"

	dr := &DeliveryRequest{
		WebhookID:         uuid.New(),
		DeliveryID:        "delivery-123",
		URL:               "https://example.com/hook",
		Method:            "POST",
		Payload:           payload,
		Secret:            secret,
		Ed25519PrivateKey: priv,
	}

	req, err := BuildRequest(context.Background(), dr)
	if err != nil {
		t.Fatalf("BuildRequest: %v", err)
	}

	if err := signature.VerifyHMAC(payload, req.Header, secret); err != nil {
		t.Errorf("HMAC signature from BuildRequest failed verification: %v", err)
	}
	if err := signature.VerifyEd25519(payload, req.Header, hex.EncodeToString(pub)); err != nil {
		t.Errorf("Ed25519 signature from BuildRequest failed verification: %v", err)
	}

	// Tampered body must fail both.
	if err := signature.VerifyHMAC([]byte("tampered"), req.Header, secret); err == nil {
		t.Error("HMAC verification accepted a tampered payload")
	}
	if err := signature.VerifyEd25519([]byte("tampered"), req.Header, hex.EncodeToString(pub)); err == nil {
		t.Error("Ed25519 verification accepted a tampered payload")
	}
}
