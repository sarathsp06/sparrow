package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/sarathsp06/sparrow/pkg/signature"
)

const listenHelp = `Receive deliveries on this machine, pretty-print each one, and optionally
forward it to a local app. Ctrl-C stops and cleans up.

By default the CLI starts a listen session: the server keeps the deliveries
for it and the CLI polls for them over HTTPS, so this works against any
Sparrow you can reach, including production behind a VPN, with nothing
listening on this machine. The server needs SPARROW_LISTEN_ENABLED=true.
Each delivery is signed with the session's secret (printed at start, use it
as your app's webhook secret) and the CLI only forwards deliveries whose
signature verifies. Your app's response (or a 200 when not forwarding) is
reported back as the delivery attempt's result: a 5xx is retried, a timeout
(no answer within 10s) too.

With a consumer access token (the customer portal's credential) add
--portal; the session then belongs to that token's consumer.

--direct keeps the older mode: start a local HTTP receiver and register it
as a temporary webhook at http://host.docker.internal:<port> (or
--public-url). That needs the server to reach this machine, so it is for a
Sparrow running locally with SPARROW_ALLOW_PRIVATE_NETWORKS=true.

usage: sparrow listen --event <name> [--event <name>...] [--forward URL]
                      [--ttl DURATION] [--portal]
       sparrow listen --direct --event <name> [--port N] [--bind ADDR]
                      [--public-url URL] [--forward URL]
`

func newListenCmd() *cobra.Command {
	var port int
	var bind string
	var events listFlag
	var publicURL string
	var forward string
	var direct, portal bool
	var ttl time.Duration
	cmd := &cobra.Command{
		Use:   "listen",
		Short: "Receive deliveries on this machine (and forward them to a local app)",
		Long:  listenHelp,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(events) == 0 {
				return fmt.Errorf("at least one --event is required")
			}
			if direct && portal {
				return fmt.Errorf("--portal works only with listen sessions, not --direct")
			}
			client, cfg, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if direct {
				return runListen(cmd.Context(), cmd.OutOrStdout(), client, cfg.Consumer, bind, port, events, publicURL, forward)
			}
			client.portal = portal
			return runRemoteListen(cmd.Context(), cmd.OutOrStdout(), client, cfg.Consumer, events, ttl, forward)
		},
	}
	f := cmd.Flags()
	f.VarP(&events, "event", "e", "event type to subscribe to (repeatable, required)")
	f.StringVar(&forward, "forward", "", "proxy each delivery to this URL and report its response")
	f.DurationVar(&ttl, "ttl", 0, "listen session lifetime (default and maximum: the server's SPARROW_LISTEN_MAX_TTL)")
	f.BoolVar(&portal, "portal", false, "the API key is a consumer access token: go through the portal API")
	f.BoolVar(&direct, "direct", false, "run a local receiver the server delivers to, instead of a listen session")
	f.IntVar(&port, "port", 0, "--direct: local port to listen on (default random)")
	f.StringVar(&bind, "bind", "", "--direct: local address to listen on (default all interfaces, so Docker can reach it; 127.0.0.1 for loopback only)")
	f.StringVar(&publicURL, "public-url", "", "--direct: URL the Sparrow server should deliver to (default http://host.docker.internal:<port>)")
	return cmd
}

// runListen (--direct) receives deliveries on a local HTTP server via a temp
// webhook.
func runListen(ctx context.Context, out io.Writer, client *apiClient, consumer, bind string, port int, events listFlag, publicURL, forward string) error {
	ln, err := net.Listen("tcp", net.JoinHostPort(bind, strconv.Itoa(port)))
	if err != nil {
		return err
	}
	localPort := ln.Addr().(*net.TCPAddr).Port
	registerURL := publicURL
	if registerURL == "" {
		registerURL = fmt.Sprintf("http://host.docker.internal:%d", localPort)
	}

	hook, err := client.registerWebhook(ctx, consumer, webhookRequest{
		URL:         registerURL,
		Events:      events,
		Active:      true,
		Description: "sparrow listen (temporary)",
	})
	if err != nil {
		ln.Close() //nolint:errcheck
		return fmt.Errorf("register webhook: %w", err)
	}
	secret := hook.HTTPConfig.WebhookSecret

	srv := &http.Server{Handler: listenHandler(out, secret, forward), ReadHeaderTimeout: 10 * time.Second}
	go srv.Serve(ln) //nolint:errcheck

	_, _ = fmt.Fprintf(out, "listening on %s, registered webhook %s -> %s (events: %v)\n", ln.Addr(), hook.WebhookID, registerURL, []string(events))
	_, _ = fmt.Fprintln(out, "press Ctrl-C to stop and delete the webhook")

	<-ctx.Done()

	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(cleanupCtx) //nolint:errcheck
	if err := client.deleteWebhook(cleanupCtx, consumer, hook.WebhookID); err != nil {
		return fmt.Errorf("delete temporary webhook %s: %w", hook.WebhookID, err)
	}
	_, _ = fmt.Fprintf(out, "\ndeleted temporary webhook %s\n", hook.WebhookID)
	return nil
}

// listenHandler prints each delivery and acknowledges (or forwards) it only
// when it carries a valid signature for secret.
func listenHandler(out io.Writer, secret, forward string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(io.LimitReader(r.Body, maxListenBody))
		if !printReceived(out, r, body, secret) {
			// Not from Sparrow: the port may be reachable by others on the
			// network, so never forward or acknowledge unsigned requests.
			http.Error(w, "invalid or missing webhook signature", http.StatusUnauthorized)
			return
		}
		if forward != "" {
			mirrorForward(out, w, r, body, forward)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

// maxListenBody caps a received delivery (Sparrow's own default body limit).
const maxListenBody = 5 << 20

// printReceived pretty-prints one delivery: signature check, headers, body.
// It reports whether the delivery is authentic (always true when the
// webhook has no secret, e.g. a server without signing).
func printReceived(out io.Writer, r *http.Request, body []byte, secret string) bool {
	_, _ = fmt.Fprintf(out, "\n── delivery %s ── %s %s\n", time.Now().Format("15:04:05"), r.Method, r.URL.Path)
	if secret != "" {
		if err := signature.VerifyHMAC(body, r.Header, secret); err != nil {
			_, _ = fmt.Fprintf(out, "signature: INVALID (%v) — rejected with 401\n", err)
			return false
		}
		_, _ = fmt.Fprintln(out, "signature: verified (v1, HMAC-SHA256)")
	}
	keys := make([]string, 0, len(r.Header))
	for k := range r.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		_, _ = fmt.Fprintf(out, "%s: %s\n", k, r.Header.Get(k))
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, body, "", "  ") == nil {
		_, _ = fmt.Fprintf(out, "%s\n", pretty.Bytes())
	} else {
		out.Write(body) //nolint:errcheck
		_, _ = fmt.Fprintln(out)
	}
	return true
}

// mirrorForward proxies the delivery to forwardURL and mirrors its status.
func mirrorForward(out io.Writer, w http.ResponseWriter, r *http.Request, body []byte, forwardURL string) {
	resp, err := forwardRequest(r.Context(), r.Method, forwardURL, r.Header, body)
	if err != nil {
		_, _ = fmt.Fprintf(out, "forward: %v\n", err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	_, _ = fmt.Fprintf(out, "forward: %s -> %d\n", forwardURL, resp.status)
	w.WriteHeader(resp.status)
	w.Write(resp.body) //nolint:errcheck
}

// maxForwardResponse caps the local app's response body kept and reported.
const maxForwardResponse = 64 << 10

type forwardResponse struct {
	status  int
	headers map[string]string
	body    []byte
}

// forwardRequest sends one delivery to forwardURL with its original headers.
func forwardRequest(ctx context.Context, method, forwardURL string, header http.Header, body []byte) (forwardResponse, error) {
	req, err := http.NewRequestWithContext(ctx, method, forwardURL, bytes.NewReader(body))
	if err != nil {
		return forwardResponse{}, err
	}
	req.Header = header.Clone()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return forwardResponse{}, err
	}
	defer resp.Body.Close() //nolint:errcheck
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxForwardResponse))
	headers := make(map[string]string, len(resp.Header))
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}
	return forwardResponse{status: resp.StatusCode, headers: headers, body: respBody}, nil
}
