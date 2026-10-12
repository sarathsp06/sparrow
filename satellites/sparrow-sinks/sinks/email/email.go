// Package email is the email sink: each verified Sparrow delivery becomes one
// plain-text email sent over SMTP.
package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/sarathsp06/sparrow/satellites/sparrow-sinks/sinks"
)

const (
	defaultSubjectTemplate = "[sparrow] {{.event_name}}"
	defaultBodyTemplate    = `Event:     {{.event_name}}
Event ID:  {{.event_id}}
Timestamp: {{.timestamp}}
Attempt:   {{.attempt}}

Payload:
{{.payload_pretty}}
`
)

// SMTPConfig is the SMTP server the sink sends through.
type SMTPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	StartTLS bool   `yaml:"starttls"`
}

// Config is the email: section of sinks.yaml. The templates are Go
// text/template over the envelope's snake_case fields (event_name, event_id,
// timestamp, attempt, payload, payload_pretty).
type Config struct {
	WebhookSecret    string     `yaml:"webhook_secret"`
	SigningPublicKey string     `yaml:"signing_public_key"`
	SMTP             SMTPConfig `yaml:"smtp"`
	From             string     `yaml:"from"`
	To               []string   `yaml:"to"`
	SubjectTemplate  string     `yaml:"subject_template"`
	BodyTemplate     string     `yaml:"body_template"`
}

// Handler returns the email sink. It verifies with SigningPublicKey when
// set, else WebhookSecret; one of them is required.
func Handler(cfg Config, logger *slog.Logger) (http.Handler, error) {
	switch {
	case cfg.SMTP.Host == "" || cfg.SMTP.Port == 0:
		return nil, fmt.Errorf("email: smtp host and port are required")
	case cfg.From == "" || len(cfg.To) == 0:
		return nil, fmt.Errorf("email: from and to are required")
	}
	verify, err := sinks.VerifierFor(cfg.WebhookSecret, cfg.SigningPublicKey)
	if err != nil {
		return nil, fmt.Errorf("email: %w", err)
	}
	sink, err := newEmailSink(cfg)
	if err != nil {
		return nil, err
	}
	return sinks.Handle("email", verify, logger, sink.deliver), nil
}

type emailSink struct {
	cfg     Config
	subject *template.Template
	body    *template.Template
}

func newEmailSink(cfg Config) (*emailSink, error) {
	subjectTmpl := cfg.SubjectTemplate
	if subjectTmpl == "" {
		subjectTmpl = defaultSubjectTemplate
	}
	bodyTmpl := cfg.BodyTemplate
	if bodyTmpl == "" {
		bodyTmpl = defaultBodyTemplate
	}
	subject, err := template.New("subject").Parse(subjectTmpl)
	if err != nil {
		return nil, fmt.Errorf("email: parse subject_template: %w", err)
	}
	body, err := template.New("body").Parse(bodyTmpl)
	if err != nil {
		return nil, fmt.Errorf("email: parse body_template: %w", err)
	}
	return &emailSink{cfg: cfg, subject: subject, body: body}, nil
}

// render produces the subject line and body text for an envelope.
func (s *emailSink) render(env sinks.Envelope) (subject, body string, err error) {
	data := templateContext(env)
	var subj, bod bytes.Buffer
	if err := s.subject.Execute(&subj, data); err != nil {
		return "", "", fmt.Errorf("render subject: %w", err)
	}
	if err := s.body.Execute(&bod, data); err != nil {
		return "", "", fmt.Errorf("render body: %w", err)
	}
	// Strip CR/LF from the subject: payload text must not inject headers.
	subject = strings.NewReplacer("\r", " ", "\n", " ").Replace(subj.String())
	return subject, bod.String(), nil
}

func (s *emailSink) deliver(_ context.Context, env sinks.Envelope, _ []byte) error {
	subject, body, err := s.render(env)
	if err != nil {
		return err
	}
	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s\r\n", s.cfg.From)
	fmt.Fprintf(&msg, "To: %s\r\n", strings.Join(s.cfg.To, ", "))
	fmt.Fprintf(&msg, "Subject: %s\r\n", subject)
	fmt.Fprintf(&msg, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	msg.WriteString("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n")
	msg.WriteString(body)
	return s.send(msg.Bytes())
}

// send speaks SMTP directly (rather than smtp.SendMail) so an explicit
// starttls: true fails loudly when the server cannot upgrade, instead of
// silently sending plaintext.
func (s *emailSink) send(msg []byte) error {
	host := s.cfg.SMTP.Host
	addr := net.JoinHostPort(host, strconv.Itoa(s.cfg.SMTP.Port))
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("smtp dial %s: %w", addr, err)
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close() //nolint:errcheck
		return fmt.Errorf("smtp greeting: %w", err)
	}
	defer c.Close() //nolint:errcheck
	if s.cfg.SMTP.StartTLS {
		if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	if s.cfg.SMTP.Username != "" {
		auth := smtp.PlainAuth("", s.cfg.SMTP.Username, s.cfg.SMTP.Password, host)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := c.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	for _, rcpt := range s.cfg.To {
		if err := c.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", rcpt, err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp finish data: %w", err)
	}
	return c.Quit()
}

// templateContext exposes the envelope to text/template with the same
// snake_case keys Sparrow's server-side transform templates use.
func templateContext(env sinks.Envelope) map[string]any {
	var payload any
	_ = json.Unmarshal(env.Payload, &payload) // nil payload renders as <no value>
	pretty, _ := json.MarshalIndent(payload, "", "  ")
	return map[string]any{
		"version":        env.Version,
		"event_id":       env.EventID,
		"event_name":     env.EventName,
		"timestamp":      env.Timestamp,
		"attempt":        env.Attempt,
		"payload":        payload,
		"payload_pretty": string(pretty),
	}
}
