package main

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"
)

func testEnvelope() envelope {
	return envelope{
		Version:   "1",
		EventID:   "evt_42",
		EventName: "order.paid",
		Timestamp: "2026-09-12T10:00:00Z",
		Attempt:   2,
		Payload:   json.RawMessage(`{"order_id":"ord_9","total":12.5}`),
	}
}

func TestEmailRender_Defaults(t *testing.T) {
	sink, err := newEmailSink(emailConfig{From: "a@b", To: []string{"c@d"}})
	if err != nil {
		t.Fatal(err)
	}
	subject, body, err := sink.render(testEnvelope())
	if err != nil {
		t.Fatal(err)
	}
	if subject != "[sparrow] order.paid" {
		t.Errorf("subject = %q", subject)
	}
	for _, want := range []string{"evt_42", "order.paid", "2026-09-12T10:00:00Z", `"order_id": "ord_9"`} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q:\n%s", want, body)
		}
	}
}

func TestEmailRender_CustomTemplatesAndHeaderInjection(t *testing.T) {
	sink, err := newEmailSink(emailConfig{
		SubjectTemplate: "{{.payload.order_id}} paid on attempt {{.attempt}}",
		BodyTemplate:    "total={{.payload.total}}",
	})
	if err != nil {
		t.Fatal(err)
	}
	env := testEnvelope()
	env.Payload = json.RawMessage(`{"order_id":"ord_9\r\nBcc: evil@x","total":1}`)
	subject, body, err := sink.render(env)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(subject, "\r\n") {
		t.Errorf("subject allows header injection: %q", subject)
	}
	if !strings.HasSuffix(subject, "paid on attempt 2") {
		t.Errorf("subject = %q", subject)
	}
	if body != "total=1" {
		t.Errorf("body = %q", body)
	}
}

func TestEmailRender_BadTemplate(t *testing.T) {
	if _, err := newEmailSink(emailConfig{SubjectTemplate: "{{.unclosed"}); err == nil {
		t.Fatal("expected parse error")
	}
}

// smtpMessage is what the in-test SMTP server captured from one session.
type smtpMessage struct {
	From string
	To   []string
	Data string
}

// startSMTPServer runs a minimal SMTP server: greet, EHLO, MAIL, RCPT, DATA,
// QUIT. Each completed session is sent on the returned channel.
func startSMTPServer(t *testing.T) (host string, port int, msgs <-chan smtpMessage) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() }) //nolint:errcheck
	ch := make(chan smtpMessage, 16)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveSMTP(conn, ch)
		}
	}()

	addr := ln.Addr().(*net.TCPAddr)
	return "127.0.0.1", addr.Port, ch
}

func serveSMTP(conn net.Conn, ch chan<- smtpMessage) {
	defer conn.Close() //nolint:errcheck
	r := bufio.NewReader(conn)
	reply := func(s string) { conn.Write([]byte(s + "\r\n")) } //nolint:errcheck

	reply("220 test.local SMTP")
	var msg smtpMessage
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		cmd := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			reply("250 test.local")
		case strings.HasPrefix(cmd, "MAIL FROM:"):
			msg.From = strings.Trim(line[len("MAIL FROM:"):], "<> ")
			reply("250 ok")
		case strings.HasPrefix(cmd, "RCPT TO:"):
			msg.To = append(msg.To, strings.Trim(line[len("RCPT TO:"):], "<> "))
			reply("250 ok")
		case cmd == "DATA":
			reply("354 go ahead")
			var data strings.Builder
			for {
				dl, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(dl, "\r\n") == "." {
					break
				}
				data.WriteString(dl)
			}
			msg.Data = data.String()
			reply("250 queued")
			ch <- msg
			msg = smtpMessage{}
		case cmd == "QUIT":
			reply("221 bye")
			return
		default:
			reply("250 ok")
		}
	}
}

func TestEmailSink_SendsViaSMTP(t *testing.T) {
	host, port, msgs := startSMTPServer(t)
	sink, err := newEmailSink(emailConfig{
		SMTP: smtpConfig{Host: host, Port: port},
		From: "sparrow@example.com",
		To:   []string{"ops@example.com", "dev@example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := sink.deliver(context.Background(), testEnvelope(), nil); err != nil {
		t.Fatalf("deliver: %v", err)
	}

	select {
	case got := <-msgs:
		if got.From != "sparrow@example.com" {
			t.Errorf("MAIL FROM = %q", got.From)
		}
		if len(got.To) != 2 || got.To[0] != "ops@example.com" || got.To[1] != "dev@example.com" {
			t.Errorf("RCPT TO = %v", got.To)
		}
		for _, want := range []string{"Subject: [sparrow] order.paid", "To: ops@example.com, dev@example.com", "evt_42", "ord_9"} {
			if !strings.Contains(got.Data, want) {
				t.Errorf("DATA missing %q:\n%s", want, got.Data)
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no message received")
	}
}

func TestEmailSink_SMTPDownIsError(t *testing.T) {
	// Reserve a port and close it so nothing listens there.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close() //nolint:errcheck

	sink, err := newEmailSink(emailConfig{
		SMTP: smtpConfig{Host: "127.0.0.1", Port: port},
		From: "a@b", To: []string{"c@d"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := sink.deliver(context.Background(), testEnvelope(), nil); err == nil {
		t.Fatal("expected connection error")
	}
}
