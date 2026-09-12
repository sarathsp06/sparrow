//go:build integration

package integration

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// sinkSMTPMessage is one mail session captured by the in-test SMTP server.
type sinkSMTPMessage struct {
	From string
	To   []string
	Data string
}

// startSinkSMTPServer runs a minimal SMTP server (greet, EHLO, MAIL, RCPT,
// DATA, QUIT) and streams completed sessions on the returned channel.
func startSinkSMTPServer(t *testing.T) (port int, msgs <-chan sinkSMTPMessage) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() }) //nolint:errcheck
	ch := make(chan sinkSMTPMessage, 16)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close() //nolint:errcheck
				r := bufio.NewReader(conn)
				reply := func(s string) { conn.Write([]byte(s + "\r\n")) } //nolint:errcheck
				reply("220 test.local SMTP")
				var msg sinkSMTPMessage
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
						msg = sinkSMTPMessage{}
					case cmd == "QUIT":
						reply("221 bye")
						return
					default:
						reply("250 ok")
					}
				}
			}(conn)
		}
	}()

	return ln.Addr().(*net.TCPAddr).Port, ch
}

// freePort reserves an ephemeral port and releases it for the sinks binary.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())
	return port
}

// startSinksBinary builds and runs cmd/sparrow-sinks with the given config,
// waiting until /healthz answers.
func startSinksBinary(t *testing.T, ctx context.Context, configYAML string) (baseURL string) {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "sparrow-sinks")
	build := exec.CommandContext(ctx, "go", "build", "-o", bin, "github.com/sarathsp06/sparrow/cmd/sparrow-sinks")
	build.Stderr = os.Stderr
	require.NoError(t, build.Run(), "build sparrow-sinks")

	cfgPath := filepath.Join(dir, "sinks.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(configYAML), 0o600))

	cmd := exec.Command(bin, "--config", cfgPath)
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	// listen addr is the first line of the config: "listen: 127.0.0.1:PORT"
	addr := strings.TrimSpace(strings.TrimPrefix(strings.SplitN(configYAML, "\n", 2)[0], "listen:"))
	baseURL = "http://" + addr
	require.Eventually(t, func() bool {
		resp, err := http.Get(baseURL + "/healthz")
		if err != nil {
			return false
		}
		resp.Body.Close() //nolint:errcheck
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 100*time.Millisecond, "sinks /healthz never became ready")
	return baseURL
}

// TestE2E_EmailSink proves the full chain: Sparrow -> signed delivery ->
// sparrow-sinks signature verification -> SMTP, using the real binary and an
// in-test SMTP server.
func TestE2E_EmailSink(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		namespace = "sinks-test"
		eventName = "sinks.email.event"
	)

	smtpPort, msgs := startSinkSMTPServer(t)
	sinksPort := freePort(t)

	registerEventType(t, c, ctx, eventName)

	// Register the webhook first: its plaintext secret is only returned once,
	// at registration, and the sinks binary needs it before it can verify.
	var webhookOut struct {
		WebhookID  string `json:"webhook_id"`
		HTTPConfig struct {
			WebhookSecret string `json:"webhook_secret"`
		} `json:"http_config"`
	}
	resp, err := c.post(ctx, "/v1/namespaces/"+namespace+"/webhooks", map[string]any{
		"events": []string{eventName},
		"url":    fmt.Sprintf("http://127.0.0.1:%d/sinks/email", sinksPort),
		"active": true,
		"http_config": map[string]any{
			"max_retries":             3,
			"retry_backoff_seconds":   1,
			"request_timeout_seconds": 5,
		},
	}, &webhookOut)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotEmpty(t, webhookOut.HTTPConfig.WebhookSecret, "registration must return the plaintext webhook secret")

	configYAML := fmt.Sprintf(`listen: 127.0.0.1:%d
webhook_secret: %q
email:
  smtp: {host: 127.0.0.1, port: %d, starttls: false}
  from: sparrow@example.com
  to: [ops@example.com]
`, sinksPort, webhookOut.HTTPConfig.WebhookSecret, smtpPort)
	startSinksBinary(t, ctx, configYAML)

	eventID := pushTestEvent(t, c, ctx, namespace, eventName)

	select {
	case mail := <-msgs:
		require.Equal(t, "sparrow@example.com", mail.From)
		require.Equal(t, []string{"ops@example.com"}, mail.To)
		require.Contains(t, mail.Data, eventID, "mail body must contain the event id")
		require.Contains(t, mail.Data, "Subject: [sparrow] "+eventName)
	case <-time.After(60 * time.Second):
		t.Fatal("timed out waiting for the email sink to deliver")
	}
}
