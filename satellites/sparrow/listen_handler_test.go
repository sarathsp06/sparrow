package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func signedRequest(t *testing.T, secret string, body []byte) *http.Request {
	t.Helper()
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
	if err != nil {
		t.Fatal(err)
	}
	id, ts := "msg_1", strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id + "." + ts + "." + string(body)))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("webhook-id", id)
	req.Header.Set("webhook-timestamp", ts)
	req.Header.Set("webhook-signature", "v1,"+base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	return req
}

func TestListenHandlerRejectsUnsignedAndNeverForwardsThem(t *testing.T) {
	secret := "whsec_" + base64.StdEncoding.EncodeToString([]byte("0123456789abcdef01234567"))
	var forwarded atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		forwarded.Add(1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer upstream.Close()
	h := listenHandler(io.Discard, secret, upstream.URL)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"forged":true}`)))
	if rec.Code != http.StatusUnauthorized || forwarded.Load() != 0 {
		t.Fatalf("unsigned: status %d, forwarded %d; want 401 and nothing forwarded", rec.Code, forwarded.Load())
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, signedRequest(t, secret, []byte(`{"ok":true}`)))
	if rec.Code != http.StatusAccepted || forwarded.Load() != 1 {
		t.Fatalf("signed: status %d, forwarded %d; want upstream's 202 and one forward", rec.Code, forwarded.Load())
	}
}

func TestListenHandlerWithoutSecretAcceptsAll(t *testing.T) {
	rec := httptest.NewRecorder()
	listenHandler(io.Discard, "", "").ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200 when the webhook has no secret", rec.Code)
	}
}
