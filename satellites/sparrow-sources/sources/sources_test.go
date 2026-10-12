package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakePusher records pushes and optionally fails.
type fakePusher struct {
	events []string
	fail   bool
}

func (f *fakePusher) PushEvent(_ context.Context, event string, _ json.RawMessage, _ map[string]string) error {
	if f.fail {
		return fmt.Errorf("boom")
	}
	f.events = append(f.events, event)
	return nil
}

func TestHandle(t *testing.T) {
	push := &fakePusher{}
	h := Handle("custom", push, nil, func(r *http.Request, body []byte) (string, map[string]string, error) {
		switch {
		case r.Header.Get("X-Token") != "ok":
			return "", nil, ErrBadSignature
		case string(body) == "bad":
			return "", nil, fmt.Errorf("unmappable")
		}
		return "custom.ping", nil, nil
	})
	do := func(token, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("X-Token", token)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	if w := do("", "{}"); w.Code != http.StatusUnauthorized || w.Body.Len() != 0 {
		t.Errorf("unsigned: status %d body %q", w.Code, w.Body)
	}
	if w := do("ok", "bad"); w.Code != http.StatusBadRequest {
		t.Errorf("map error: status %d", w.Code)
	}
	if w := do("ok", "{}"); w.Code != http.StatusOK {
		t.Errorf("signed: status %d", w.Code)
	}
	push.fail = true
	if w := do("ok", "{}"); w.Code != http.StatusBadGateway {
		t.Errorf("push failure: status %d", w.Code)
	}
	if got := fmt.Sprint(push.events); got != "[custom.ping]" {
		t.Errorf("pushed events = %s", got)
	}
}
