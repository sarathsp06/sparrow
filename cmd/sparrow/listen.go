package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"time"

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

// runListen receives deliveries on a local HTTP server via a temp webhook.
func runListen(ctx context.Context, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("listen", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(out, listenHelp); fs.PrintDefaults() }
	urlFlag, apiKeyFlag, nsFlag := configFlags(fs)
	port := fs.Int("port", 0, "local port to listen on (default random)")
	var events listFlag
	fs.Var(&events, "event", "event type to subscribe to (repeatable, required)")
	publicURL := fs.String("public-url", "", "URL the Sparrow server should deliver to (default http://host.docker.internal:<port>)")
	forward := fs.String("forward", "", "proxy each delivery to this URL and mirror its status")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(events) == 0 {
		return fmt.Errorf("at least one --event is required\n\n%s", listenHelp)
	}
	cfg, err := resolveConfig(*urlFlag, *apiKeyFlag, *nsFlag)
	if err != nil {
		return err
	}
	client := newAPIClient(cfg)

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		return err
	}
	localPort := ln.Addr().(*net.TCPAddr).Port
	registerURL := *publicURL
	if registerURL == "" {
		registerURL = fmt.Sprintf("http://host.docker.internal:%d", localPort)
	}

	hook, err := client.registerWebhook(ctx, cfg.Namespace, webhookRequest{
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
		if *forward != "" {
			mirrorForward(out, w, r, body, *forward)
			return
		}
		w.WriteHeader(http.StatusOK)
	})}
	go srv.Serve(ln) //nolint:errcheck

	fmt.Fprintf(out, "listening on :%d, registered webhook %s -> %s (events: %v)\n", localPort, hook.WebhookID, registerURL, []string(events))
	fmt.Fprintln(out, "press Ctrl-C to stop and delete the webhook")

	<-ctx.Done()

	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(cleanupCtx) //nolint:errcheck
	if err := client.deleteWebhook(cleanupCtx, cfg.Namespace, hook.WebhookID); err != nil {
		return fmt.Errorf("delete temporary webhook %s: %w", hook.WebhookID, err)
	}
	fmt.Fprintf(out, "\ndeleted temporary webhook %s\n", hook.WebhookID)
	return nil
}

// printReceived pretty-prints one delivery: signature check, headers, body.
func printReceived(out io.Writer, r *http.Request, body []byte, secret string) {
	fmt.Fprintf(out, "\n── delivery %s ── %s %s\n", time.Now().Format("15:04:05"), r.Method, r.URL.Path)
	if secret != "" {
		if err := signature.VerifyHMAC(body, r.Header, secret); err != nil {
			fmt.Fprintf(out, "signature: INVALID (%v)\n", err)
		} else {
			fmt.Fprintln(out, "signature: verified (v1, HMAC-SHA256)")
		}
	}
	keys := make([]string, 0, len(r.Header))
	for k := range r.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(out, "%s: %s\n", k, r.Header.Get(k))
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, body, "", "  ") == nil {
		fmt.Fprintf(out, "%s\n", pretty.Bytes())
	} else {
		out.Write(body) //nolint:errcheck
		fmt.Fprintln(out)
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
		fmt.Fprintf(out, "forward: %v\n", err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close() //nolint:errcheck
	fmt.Fprintf(out, "forward: %s -> %d\n", forwardURL, resp.StatusCode)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body) //nolint:errcheck
}
