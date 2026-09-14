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
	"time"

	"github.com/spf13/cobra"

	"github.com/sarathsp06/sparrow/pkg/signature"
)

const listenHelp = `Start a local HTTP receiver, register it as a temporary webhook, and
pretty-print every delivery. The webhook is deleted on Ctrl-C.

Without --public-url the webhook is registered as
http://host.docker.internal:<port>, which assumes Sparrow runs in Docker on
this machine (Docker Desktop resolves host.docker.internal to your host).
Sparrow blocks private-network delivery URLs by default (SSRF guard); for
local receivers start the server with SPARROW_ALLOW_PRIVATE_NETWORKS=true.

usage: sparrow listen --event <name> [--event <name>...] [--port N]
                      [--public-url URL] [--forward URL]
`

func newListenCmd() *cobra.Command {
	var port int
	var events listFlag
	var publicURL string
	var forward string
	cmd := &cobra.Command{
		Use:   "listen",
		Short: "Receive deliveries on a local HTTP server via a temp webhook",
		Long:  listenHelp,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(events) == 0 {
				return fmt.Errorf("at least one --event is required")
			}
			client, cfg, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			return runListen(cmd.Context(), cmd.OutOrStdout(), client, cfg.Consumer, port, events, publicURL, forward)
		},
	}
	f := cmd.Flags()
	f.IntVar(&port, "port", 0, "local port to listen on (default random)")
	f.VarP(&events, "event", "e", "event type to subscribe to (repeatable, required)")
	f.StringVar(&publicURL, "public-url", "", "URL the Sparrow server should deliver to (default http://host.docker.internal:<port>)")
	f.StringVar(&forward, "forward", "", "proxy each delivery to this URL and mirror its status")
	return cmd
}

// runListen receives deliveries on a local HTTP server via a temp webhook.
func runListen(ctx context.Context, out io.Writer, client *apiClient, consumer string, port int, events listFlag, publicURL, forward string) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
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

	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		printReceived(out, r, body, secret)
		if forward != "" {
			mirrorForward(out, w, r, body, forward)
			return
		}
		w.WriteHeader(http.StatusOK)
	})}
	go srv.Serve(ln) //nolint:errcheck

	_, _ = fmt.Fprintf(out, "listening on :%d, registered webhook %s -> %s (events: %v)\n", localPort, hook.WebhookID, registerURL, []string(events))
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

// printReceived pretty-prints one delivery: signature check, headers, body.
func printReceived(out io.Writer, r *http.Request, body []byte, secret string) {
	_, _ = fmt.Fprintf(out, "\n── delivery %s ── %s %s\n", time.Now().Format("15:04:05"), r.Method, r.URL.Path)
	if secret != "" {
		if err := signature.VerifyHMAC(body, r.Header, secret); err != nil {
			_, _ = fmt.Fprintf(out, "signature: INVALID (%v)\n", err)
		} else {
			_, _ = fmt.Fprintln(out, "signature: verified (v1, HMAC-SHA256)")
		}
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
}

// mirrorForward proxies the delivery to forwardURL and mirrors its status.
func mirrorForward(out io.Writer, w http.ResponseWriter, r *http.Request, body []byte, forwardURL string) {
	req, err := http.NewRequestWithContext(r.Context(), r.Method, forwardURL, bytes.NewReader(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	req.Header = r.Header.Clone()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		_, _ = fmt.Fprintf(out, "forward: %v\n", err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close() //nolint:errcheck
	_, _ = fmt.Fprintf(out, "forward: %s -> %d\n", forwardURL, resp.StatusCode)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body) //nolint:errcheck
}
