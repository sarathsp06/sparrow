package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sarathsp06/sparrow/pkg/receivercode"
)

const scaffoldHelp = `Generate a ready-to-run receiver project for a webhook.

The project verifies Sparrow's signatures on the raw body, rejects stale and
repeated deliveries, routes each event to its own handler and ships with tests.
Your webhook secret is never written: the receiver reads SPARROW_WEBHOOK_SECRET.

  sparrow scaffold --webhook <id> --lang go --framework nethttp -o ./receiver
  sparrow scaffold --webhook <id> --lang python --framework fastapi --test
  sparrow scaffold --offline --schema order.json --event-type order.paid \
      --signing hmac --lang node --framework express

Targets: go/nethttp, python/fastapi, node/express (TypeScript). Without
--offline the server generates the project from the webhook's subscriptions,
event types and transforms (POST :generateReceiverCode). --offline generates
it here from a JSON Schema and/or sample payload, with no server. The schema
types the model; a --sample that contradicts it makes the project's own
tests fail.

--test sends one test event per subscription after writing (POST
:sendTestEvent): run the receiver first, at the webhook's URL.`

type scaffoldOpts struct {
	webhookID     string
	lang          string
	framework     string
	outDir        string
	force         bool
	test          bool
	subscriptions []string

	offline     bool
	schemaFile  string
	sampleFile  string
	eventType   string
	signing     string
	publicKey   string
	transform   string
	contentType string
}

func newScaffoldCmd() *cobra.Command {
	var o scaffoldOpts
	cmd := &cobra.Command{
		Use:   "scaffold",
		Short: "Generate a receiver project for a webhook",
		Long:  scaffoldHelp,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runScaffold(cmd, o)
		},
	}
	f := cmd.Flags()
	f.StringVar(&o.webhookID, "webhook", "", "webhook id to generate a receiver for")
	f.StringVar(&o.lang, "lang", "", "language: go, python or node")
	f.StringVar(&o.framework, "framework", "", "framework: nethttp (go), fastapi (python) or express (node)")
	f.StringVarP(&o.outDir, "out", "o", "sparrow-receiver", "directory to write the project to")
	f.BoolVar(&o.force, "force", false, "overwrite files that already exist")
	f.BoolVar(&o.test, "test", false, "send a test event per subscription after writing (not with --offline)")
	f.StringSliceVar(&o.subscriptions, "subscription", nil, "only these subscription ids (repeatable; default all)")
	f.BoolVar(&o.offline, "offline", false, "generate locally without a server")
	f.StringVar(&o.schemaFile, "schema", "", "offline: JSON Schema file of the event payload")
	f.StringVar(&o.sampleFile, "sample", "", "offline: sample payload JSON file")
	f.StringVar(&o.eventType, "event-type", "example.event", "offline: event type name")
	f.StringVar(&o.signing, "signing", receivercode.SchemeHMAC, "offline: signing scheme, hmac or ed25519")
	f.StringVar(&o.publicKey, "public-key", "", "offline: the webhook's hex Ed25519 public key (with --signing ed25519)")
	f.StringVar(&o.transform, "transform", "", "offline: transform template file the subscription uses")
	f.StringVar(&o.contentType, "content-type", "", "offline: the subscription's Content-Type header, if it sets one")
	return cmd
}

// generatedProject is the generateReceiverCode response, or the offline
// equivalent.
type generatedProject struct {
	Files []receivercode.File `json:"files"`
	Spec  struct {
		Subscriptions []struct {
			ID        string   `json:"id"`
			EventType string   `json:"event_type"`
			Notes     []string `json:"notes"`
		} `json:"subscriptions"`
	} `json:"spec"`
	Run     string `json:"run"`
	Test    string `json:"test"`
	HasMore bool   `json:"has_more"`
}

func runScaffold(cmd *cobra.Command, o scaffoldOpts) error {
	if o.lang == "" || o.framework == "" {
		return errors.New("--lang and --framework are required (see sparrow scaffold --help)")
	}
	out := cmd.OutOrStdout()
	var (
		proj   *generatedProject
		client *apiClient
		cfg    config
		err    error
	)
	if o.offline {
		if o.webhookID != "" || o.test || len(o.subscriptions) > 0 {
			return errors.New("--offline cannot be combined with --webhook, --subscription or --test")
		}
		proj, err = scaffoldOffline(o)
	} else {
		if o.webhookID == "" {
			return errors.New("--webhook is required (or use --offline)")
		}
		client, cfg, err = clientFromCmd(cmd)
		if err != nil {
			return err
		}
		proj = &generatedProject{}
		body := map[string]any{"language": o.lang, "framework": o.framework}
		if len(o.subscriptions) > 0 {
			body["subscription_ids"] = o.subscriptions
		}
		err = client.do(cmd.Context(), http.MethodPost, webhookPath(cfg.Consumer, o.webhookID)+":generateReceiverCode", body, proj)
	}
	if err != nil {
		return err
	}

	if err := writeProject(o.outDir, proj.Files, o.force); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "Wrote %d files to %s\n\n  cd %s\n  %s   # run\n  %s   # test\n", len(proj.Files), o.outDir, o.outDir, proj.Run, proj.Test)
	for _, s := range proj.Spec.Subscriptions {
		for _, n := range s.Notes {
			_, _ = fmt.Fprintf(out, "\nnote: %s: %s", s.EventType, n)
		}
	}
	if proj.HasMore {
		_, _ = fmt.Fprintf(out, "\nwarning: the webhook has more than %d subscriptions; only the first %d have handlers. Pass --subscription to choose them.", len(proj.Spec.Subscriptions), len(proj.Spec.Subscriptions))
	}
	_, _ = fmt.Fprintln(out)

	if !o.test {
		return nil
	}
	return sendTestEvents(cmd.Context(), out, client, cfg.Consumer, o.webhookID, proj)
}

func webhookPath(consumer, webhookID string) string {
	return "/v1/consumers/" + url.PathEscape(consumer) + "/webhooks/" + url.PathEscape(webhookID)
}

// scaffoldOffline generates a one-subscription project locally.
func scaffoldOffline(o scaffoldOpts) (*generatedProject, error) {
	if o.schemaFile == "" && o.sampleFile == "" && o.transform == "" {
		return nil, errors.New("--offline needs --schema, --sample or --transform to describe the payload")
	}
	read := func(path string) ([]byte, error) {
		if path == "" {
			return nil, nil
		}
		return os.ReadFile(path) // #nosec G304 -- the user names their own files
	}
	schema, err := read(o.schemaFile)
	if err != nil {
		return nil, err
	}
	sample, err := read(o.sampleFile)
	if err != nil {
		return nil, err
	}
	tmpl, err := read(o.transform)
	if err != nil {
		return nil, err
	}
	for name, b := range map[string][]byte{"--schema": schema, "--sample": sample} {
		if len(b) > 0 && !json.Valid(b) {
			return nil, fmt.Errorf("%s is not valid JSON", name)
		}
	}
	sub := receivercode.Subscription{
		ID: "offline", EventType: o.eventType, EventVersion: 1,
		Schema: schema, Sample: sample, Transform: string(tmpl), StrictMissingKeys: true,
	}
	if o.contentType != "" {
		sub.Headers = map[string]string{"Content-Type": o.contentType}
	}
	gen, err := receivercode.Generate(receivercode.Input{
		Target:         receivercode.Target{Language: o.lang, Framework: o.framework},
		SparrowVersion: version,
		Webhook: receivercode.Webhook{ID: "offline", SigningScheme: o.signing,
			Ed25519PublicKey: o.publicKey},
		Subscriptions: []receivercode.Subscription{sub},
	})
	if err != nil {
		return nil, err
	}
	proj := &generatedProject{Files: gen.Files, Run: gen.Run, Test: gen.Test}
	for _, s := range gen.Spec.Subscriptions {
		proj.Spec.Subscriptions = append(proj.Spec.Subscriptions, struct {
			ID        string   `json:"id"`
			EventType string   `json:"event_type"`
			Notes     []string `json:"notes"`
		}{s.ID, s.EventType, s.Notes})
	}
	return proj, nil
}

// writeProject writes files under dir. Without force it refuses before
// writing anything if a file already exists.
func writeProject(dir string, files []receivercode.File, force bool) error {
	if len(files) == 0 {
		return errors.New("the server returned no files")
	}
	targets := make([]string, len(files))
	var existing []string
	for i, f := range files {
		clean := filepath.Clean(filepath.FromSlash(f.Path))
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("refusing to write %q outside %s", f.Path, dir)
		}
		targets[i] = filepath.Join(dir, clean)
		if _, err := os.Lstat(targets[i]); err == nil {
			existing = append(existing, f.Path)
		}
	}
	if len(existing) > 0 && !force {
		sort.Strings(existing)
		return fmt.Errorf("%s already has %s; use --force to overwrite", dir, strings.Join(existing, ", "))
	}
	for i, f := range files {
		if err := os.MkdirAll(filepath.Dir(targets[i]), 0o750); err != nil {
			return err
		}
		mode := fs.FileMode(f.Mode) & 0o777
		if mode == 0 {
			mode = 0o644
		}
		if err := os.WriteFile(targets[i], []byte(f.Content), mode); err != nil {
			return err
		}
		// WriteFile keeps an existing file's mode; make it match.
		if err := os.Chmod(targets[i], mode); err != nil {
			return err
		}
	}
	return nil
}

// testEventRetryWait is how long --test waits when the server limits test
// events (one per webhook every 2 seconds).
var testEventRetryWait = 2100 * time.Millisecond

type testEventResult struct {
	StatusCode int    `json:"status_code"`
	LatencyMs  int64  `json:"latency_ms"`
	ErrorClass string `json:"error_class"`
}

// sendTestEvents sends one test event per subscription of the project and
// prints the outcome; it fails when any receives no 2xx.
func sendTestEvents(ctx context.Context, out io.Writer, client *apiClient, consumer, webhookID string, proj *generatedProject) error {
	_, _ = fmt.Fprintln(out, "Test events:")
	failed := 0
	for _, s := range proj.Spec.Subscriptions {
		var res testEventResult
		var err error
		for attempt := 0; attempt < 3; attempt++ {
			err = client.do(ctx, http.MethodPost, webhookPath(consumer, webhookID)+":sendTestEvent", map[string]string{"subscription_id": s.ID}, &res)
			var apiErr *apiError
			if !errors.As(err, &apiErr) || apiErr.Status != http.StatusTooManyRequests {
				break
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(testEventRetryWait):
			}
		}
		switch {
		case err != nil:
			failed++
			_, _ = fmt.Fprintf(out, "  ✗ %s: %v\n", s.EventType, err)
		case res.StatusCode >= 200 && res.StatusCode < 300:
			_, _ = fmt.Fprintf(out, "  ✓ %s: %d in %d ms\n", s.EventType, res.StatusCode, res.LatencyMs)
		default:
			failed++
			_, _ = fmt.Fprintf(out, "  ✗ %s: %d %s in %d ms\n", s.EventType, res.StatusCode, res.ErrorClass, res.LatencyMs)
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d test events did not get a 2xx", failed, len(proj.Spec.Subscriptions))
	}
	return nil
}
