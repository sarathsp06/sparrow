package receivercode

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Signing schemes of a webhook (its signature_type).
const (
	SchemeHMAC    = "hmac"
	SchemeEd25519 = "ed25519"
)

// Target names a language and framework pair; Targets lists the supported
// ones.
type Target struct {
	Language  string `json:"language"`
	Framework string `json:"framework"`
}

// ID is the target's short name, e.g. "go-nethttp".
func (t Target) ID() string { return t.Language + "-" + t.Framework }

// Input is everything Generate needs, as plain data.
type Input struct {
	Target Target
	// SparrowVersion goes into each file's header comment. Empty means
	// "dev".
	SparrowVersion string
	Webhook        Webhook
	// Subscriptions are the subscriptions to generate handlers for. Order
	// does not matter: output is sorted by event type and ID.
	Subscriptions []Subscription
}

// Webhook describes the receiving endpoint.
type Webhook struct {
	ID       string
	Consumer string
	// SigningScheme is SchemeHMAC or SchemeEd25519.
	SigningScheme string
	// Ed25519PublicKey is the hex public key of an ed25519 webhook. It is
	// public and is embedded in the generated code.
	Ed25519PublicKey string
}

// Subscription is one event type the receiver handles.
type Subscription struct {
	ID           string
	EventType    string
	EventVersion int
	// Schema is the event type's JSON Schema, if any.
	Schema json.RawMessage
	// Sample is a sample event payload (the envelope's "payload").
	Sample json.RawMessage
	// Transform is the transform template; empty sends the envelope.
	Transform string
	// StrictMissingKeys renders Transform with missingkey=error, as the
	// subscription's template_missing_key "error" does.
	StrictMissingKeys bool
	// Headers are the subscription's custom delivery headers. Only
	// Content-Type is read.
	Headers map[string]string
}

var eventTypePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]*$`)

func (in *Input) validate() error {
	if _, ok := targetInfo(in.Target); !ok {
		return fmt.Errorf("receivercode: unsupported target %q", in.Target.ID())
	}
	switch in.Webhook.SigningScheme {
	case SchemeHMAC:
	case SchemeEd25519:
		if !isHexKey(in.Webhook.Ed25519PublicKey) {
			return fmt.Errorf("receivercode: ed25519 webhook needs a 64-character hex public key")
		}
	default:
		return fmt.Errorf("receivercode: unknown signing scheme %q", in.Webhook.SigningScheme)
	}
	if in.Webhook.ID == "" || strings.ContainsAny(in.Webhook.ID, "\"'`\\\n\r") {
		return fmt.Errorf("receivercode: webhook ID must be set and contain no quotes or control characters")
	}
	if strings.ContainsAny(in.Webhook.Consumer, "\n\r`") {
		return fmt.Errorf("receivercode: consumer contains backticks or line breaks")
	}
	if len(in.Subscriptions) == 0 {
		return fmt.Errorf("receivercode: no subscriptions to generate handlers for")
	}
	seen := map[string]bool{}
	for _, s := range in.Subscriptions {
		if s.ID == "" || seen[s.ID] {
			return fmt.Errorf("receivercode: subscription IDs must be set and unique (got %q)", s.ID)
		}
		seen[s.ID] = true
		if !eventTypePattern.MatchString(s.EventType) {
			return fmt.Errorf("receivercode: subscription %s: invalid event type %q", s.ID, s.EventType)
		}
		if strings.ContainsAny(s.ID, "\"'`\\\n\r") {
			return fmt.Errorf("receivercode: subscription ID %q contains quotes or control characters", s.ID)
		}
	}
	return nil
}

func isHexKey(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return true
}
