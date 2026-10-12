// Package github is the GitHub source: it verifies the X-Hub-Signature-256
// header and republishes each GitHub event as a Sparrow event named
// <sparrow_event_prefix>.<github event>[.<action>].
package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/sarathsp06/sparrow/satellites/sparrow-sources/sources"
)

// Config is the github: provider section of sources.yaml.
type Config struct {
	// Path is where the sparrow-sources binary mounts this provider; set the
	// matching Payload URL on the GitHub webhook. Default: /webhooks/github.
	// Handler ignores it: mount the handler wherever you like.
	Path string `yaml:"path"`
	// Secret is the shared secret configured on the GitHub webhook, used to
	// verify the incoming signature.
	Secret string `yaml:"secret"`
	// SparrowEventPrefix names the OUTBOUND event pushed to Sparrow (not the
	// incoming GitHub event): it is prefixed to the GitHub event name.
	// Default: github.
	SparrowEventPrefix string `yaml:"sparrow_event_prefix"`
}

// Handler returns the GitHub source: X-Hub-Signature-256 verified, each event
// pushed as <prefix>.<event>[.<action>] with source and repo labels.
func Handler(cfg Config, p sources.Pusher, log *slog.Logger) (http.Handler, error) {
	if cfg.Secret == "" {
		return nil, errors.New("github: secret is required")
	}
	if cfg.SparrowEventPrefix == "" {
		cfg.SparrowEventPrefix = "github"
	}
	return sources.Handle("github", p, log, func(r *http.Request, body []byte) (string, map[string]string, error) {
		if !verifySignature(r.Header.Get("X-Hub-Signature-256"), body, cfg.Secret) {
			return "", nil, sources.ErrBadSignature
		}
		return mapEvent(cfg.SparrowEventPrefix, r.Header.Get("X-GitHub-Event"), body)
	}), nil
}

// verifySignature checks an X-Hub-Signature-256 header
// ("sha256=<hex hmac>") against HMAC-SHA256(secret, body).
func verifySignature(header string, body []byte, secret string) bool {
	sig, ok := strings.CutPrefix(header, "sha256=")
	if !ok {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig))
}

// mapEvent maps X-GitHub-Event (+ body "action" when present) to
// <prefix>.<event>[.<action>], with source/repo labels.
func mapEvent(prefix, ghEvent string, body []byte) (string, map[string]string, error) {
	if ghEvent == "" {
		return "", nil, fmt.Errorf("missing X-GitHub-Event header")
	}
	var evt struct {
		Action     string `json:"action"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(body, &evt); err != nil {
		return "", nil, fmt.Errorf("invalid github event body")
	}
	name := prefix + "." + ghEvent
	if evt.Action != "" {
		name += "." + evt.Action
	}
	labels := map[string]string{"source": "github"}
	if evt.Repository.FullName != "" {
		labels["repo"] = evt.Repository.FullName
	}
	return name, labels, nil
}
