package rest

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	sparrow "github.com/sarathsp06/sparrow"
	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"github.com/sarathsp06/sparrow/pkg/receivercode"
)

// receiverCodeService is what the receiver code endpoints need.
type receiverCodeService interface {
	ListWebhooks(ctx context.Context, consumer string, webhookID string, event string, activeOnly bool, health string, limit, offset int32) ([]*store.WebhookRegistration, int32, error)
	ListSubscriptions(ctx context.Context, consumer string, webhookID string, eventName string, limit, offset int32) ([]*store.EventSubscription, int32, error)
	GetEvent(ctx context.Context, name string) (*store.EventRegistration, error)
	WebhookSigningPublicKeyHex(encryptedPrivKey []byte) string
	webhooks.TestEventSender
}

// maxReceiverSubscriptions bounds the subscriptions one receiver is
// generated for.
const maxReceiverSubscriptions = 200

type receiverTargetOut struct {
	ID        string `json:"id" doc:"Target id, e.g. go-nethttp."`
	Language  string `json:"language" doc:"Value for the language field of generateReceiverCode."`
	Framework string `json:"framework" doc:"Value for the framework field of generateReceiverCode."`
	Name      string `json:"name" doc:"Display name."`
	Runtime   string `json:"runtime" doc:"Runtime the generated project needs."`
}

// receiverCodeTargets lists the targets generateReceiverCode supports, for
// GET /v1/capabilities.
func receiverCodeTargets() []receiverTargetOut {
	targets := receivercode.Targets()
	out := make([]receiverTargetOut, 0, len(targets))
	for _, t := range targets {
		out = append(out, receiverTargetOut{ID: t.ID, Language: t.Language, Framework: t.Framework, Name: t.Name, Runtime: t.Runtime})
	}
	return out
}

type generateReceiverCodeInput struct {
	Consumer  string `path:"consumer" doc:"Tenant consumer the webhook belongs to."`
	WebhookID string `path:"webhook_id" doc:"Webhook id (UUID)."`
	Body      struct {
		Language        string   `json:"language" required:"true" doc:"Target language: go, python or node (see GET /v1/capabilities, receiver_code.targets)."`
		Framework       string   `json:"framework" required:"true" doc:"Target framework: nethttp (go), fastapi (python) or express (node)."`
		SubscriptionIDs []string `json:"subscription_ids,omitempty" maxItems:"200" doc:"Subscriptions to generate handlers for; omit for every subscription of the webhook."`
	}
}

type receiverFileOut struct {
	Path    string `json:"path" doc:"Slash-separated path relative to the project root."`
	Content string `json:"content" doc:"File content (UTF-8 text)."`
	Mode    uint32 `json:"mode" doc:"Unix permission bits, e.g. 420 (0644) or 493 (0755)."`
}

type generateReceiverCodeOutput struct {
	Body struct {
		Files []receiverFileOut `json:"files" doc:"The project's files, sorted by path."`
		Spec  map[string]any    `json:"spec" doc:"Machine-readable receiver spec: how deliveries are verified and routed, each handler's body format and model, and the signed test vectors the generated tests replay (signed with throwaway credentials, never the webhook's secret). Its format is versioned by spec_version and, like the generated code, experimental."`
		Run   string            `json:"run" doc:"Shell command that starts the receiver, from the project root."`
		Test  string            `json:"test" doc:"Shell command that runs the generated tests, from the project root."`
	}
}

type sendTestEventInput struct {
	Consumer  string `path:"consumer" doc:"Tenant consumer the webhook belongs to."`
	WebhookID string `path:"webhook_id" doc:"Webhook id (UUID)."`
	Body      struct {
		SubscriptionID string `json:"subscription_id" required:"true" doc:"Subscription whose event type sample is sent, rendered through its transform."`
	}
}

type sendTestEventOutput struct {
	Body struct {
		StatusCode      int    `json:"status_code" doc:"The receiver's HTTP status; 0 when no response was received."`
		LatencyMs       int64  `json:"latency_ms" doc:"Time to the response headers, in milliseconds."`
		ResponseExcerpt string `json:"response_excerpt" doc:"The first KiB of the response body."`
		ErrorClass      string `json:"error_class,omitempty" doc:"Delivery error category when the send did not get a 2xx: client_error, server_error, rate_limited, timeout, connection_refused, dns_error, tls_error, network_error, redirect, template_error or unknown."`
		Error           string `json:"error,omitempty" doc:"What went wrong when no response was received, or the template error."`
	}
}

func registerReceiverCodeRoutes(api huma.API, svc receiverCodeService) {
	huma.Register(api, huma.Operation{
		OperationID: "generateReceiverCode",
		Method:      http.MethodPost,
		Path:        "/v1/consumers/{consumer}/webhooks/{webhook_id}:generateReceiverCode",
		Summary:     "Generate receiver code for a webhook",
		Description: "Generates a ready-to-run receiver project for the webhook in the chosen language and framework (GET /v1/capabilities lists them under receiver_code.targets). " +
			"The project verifies Sparrow's signatures on the raw body before parsing (with a copy of Sparrow's tested verification helper), rejects stale timestamps, drops redeliveries by webhook-id, " +
			"routes on X-Sparrow-Subscription-ID then X-Sparrow-Event-Type, decodes each subscription's body (its transform rendered against the event type's sample decides JSON, XML or text) " +
			"and ships with tests that replay signed vectors. Secrets are never included: the receiver reads SPARROW_WEBHOOK_SECRET, and the tests use throwaway credentials. " +
			"Nothing is stored. The same webhook, subscriptions and Sparrow version always produce the same files; the generated code itself may improve between releases.\n\n" +
			"400 for an unsupported target or a subscription that is not on the webhook, 404 for an unknown webhook.\n\nPortal tokens: allowed, as POST /portal/api/webhooks/{webhook_id}:generateReceiverCode.",
		Errors: []int{400, 404},
		Tags:   []string{"Webhooks"},
	}, func(ctx context.Context, in *generateReceiverCodeInput) (*generateReceiverCodeOutput, error) {
		input, err := receiverCodeInput(ctx, svc, in)
		if err != nil {
			return nil, err
		}
		gen, err := receivercode.Generate(input)
		if err != nil {
			return nil, huma.Error400BadRequest(strings.TrimPrefix(err.Error(), "receivercode: "))
		}
		out := &generateReceiverCodeOutput{}
		out.Body.Run, out.Body.Test = gen.Run, gen.Test
		for _, f := range gen.Files {
			out.Body.Files = append(out.Body.Files, receiverFileOut(f))
		}
		raw, err := json.Marshal(gen.Spec)
		if err != nil {
			return nil, mapError(ctx, err, "failed to encode the receiver spec")
		}
		if err := json.Unmarshal(raw, &out.Body.Spec); err != nil {
			return nil, mapError(ctx, err, "failed to encode the receiver spec")
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "sendTestEvent",
		Method:      http.MethodPost,
		Path:        "/v1/consumers/{consumer}/webhooks/{webhook_id}:sendTestEvent",
		Summary:     "Send a test event to a webhook",
		Description: "Sends one signed delivery of the subscription's event type sample (rendered through its transform, as a real delivery would be) to the webhook's URL, synchronously, and returns what the receiver answered. " +
			"It uses the same request builder, signing, network policy (SSRF protection) and timeout as real deliveries and adds the header X-Sparrow-Test: true. " +
			"It creates no delivery and never counts toward the webhook's health or auto-disable. A receiver error is reported in the response (status_code, error_class), not as an API error.\n\n" +
			"One test event per webhook every 2 seconds on each server: more answer 429. 404 for an unknown webhook or a subscription not on it.\n\nPortal tokens: allowed, as POST /portal/api/webhooks/{webhook_id}:sendTestEvent.",
		Errors: []int{400, 404, 429, 503},
		Tags:   []string{"Webhooks"},
	}, func(ctx context.Context, in *sendTestEventInput) (*sendTestEventOutput, error) {
		res, err := svc.SendTestEvent(ctx, in.Consumer, in.WebhookID, in.Body.SubscriptionID)
		if err != nil {
			return nil, mapError(ctx, err, "failed to send the test event")
		}
		out := &sendTestEventOutput{}
		out.Body.StatusCode = res.StatusCode
		out.Body.LatencyMs = res.Latency.Milliseconds()
		out.Body.ResponseExcerpt = res.ResponseExcerpt
		out.Body.ErrorClass = res.ErrorClass
		out.Body.Error = res.Error
		return out, nil
	})
}

// receiverCodeInput collects the webhook, its subscriptions and their event
// types as receivercode's plain input.
func receiverCodeInput(ctx context.Context, svc receiverCodeService, in *generateReceiverCodeInput) (receivercode.Input, error) {
	regs, _, err := svc.ListWebhooks(ctx, in.Consumer, in.WebhookID, "", false, "", 1, 0)
	if err != nil {
		return receivercode.Input{}, mapError(ctx, err, "failed to get webhook")
	}
	if len(regs) == 0 {
		return receivercode.Input{}, huma.Error404NotFound("webhook not found")
	}
	reg := regs[0]

	subs, _, err := svc.ListSubscriptions(ctx, in.Consumer, reg.ID.String(), "", maxReceiverSubscriptions, 0)
	if err != nil {
		return receivercode.Input{}, mapError(ctx, err, "failed to list subscriptions")
	}
	if len(in.Body.SubscriptionIDs) > 0 {
		byID := make(map[string]*store.EventSubscription, len(subs))
		for _, s := range subs {
			byID[s.ID.String()] = s
		}
		picked := make([]*store.EventSubscription, 0, len(in.Body.SubscriptionIDs))
		for _, id := range in.Body.SubscriptionIDs {
			s, ok := byID[strings.ToLower(id)]
			if !ok {
				return receivercode.Input{}, huma.Error400BadRequest("subscription " + id + " is not on this webhook")
			}
			picked = append(picked, s)
		}
		subs = picked
	}
	if len(subs) == 0 {
		return receivercode.Input{}, huma.Error400BadRequest("the webhook has no subscriptions to generate handlers for")
	}

	version := sparrow.Version
	if strings.HasPrefix(version, "-") {
		version = "dev"
	}
	input := receivercode.Input{
		Target:         receivercode.Target{Language: in.Body.Language, Framework: in.Body.Framework},
		SparrowVersion: version,
		Webhook: receivercode.Webhook{
			ID:            reg.ID.String(),
			Consumer:      reg.Consumer,
			SigningScheme: string(reg.SignatureType),
		},
	}
	if input.Webhook.SigningScheme == "" {
		input.Webhook.SigningScheme = receivercode.SchemeHMAC
	}
	if input.Webhook.SigningScheme == receivercode.SchemeEd25519 {
		input.Webhook.Ed25519PublicKey = svc.WebhookSigningPublicKeyHex(reg.Ed25519PrivateKey)
	}

	events := map[string]*store.EventRegistration{}
	for _, s := range subs {
		ev, seen := events[s.EventName]
		if !seen {
			// An event type that cannot be read leaves the handler untyped
			// rather than failing the whole receiver.
			ev, _ = svc.GetEvent(ctx, s.EventName)
			events[s.EventName] = ev
		}
		// Delivery headers are the webhook's, overridden by the
		// subscription's; only Content-Type matters here.
		headers := maps.Clone(map[string]string(reg.Headers))
		if headers == nil {
			headers = map[string]string{}
		}
		maps.Copy(headers, s.Headers)
		sub := receivercode.Subscription{
			ID:        s.ID.String(),
			EventType: s.EventName,
			Headers:   headers,
		}
		if s.HasTransform() {
			sub.Transform = s.TransformTemplate
			sub.StrictMissingKeys = s.StrictTemplate()
		}
		if ev != nil {
			sub.EventVersion = ev.Version
			if len(ev.Schema) > 0 {
				sub.Schema, _ = json.Marshal(ev.Schema)
			}
			if len(ev.SamplePayload) > 0 {
				sub.Sample, _ = json.Marshal(ev.SamplePayload)
			}
		}
		input.Subscriptions = append(input.Subscriptions, sub)
	}
	return input, nil
}
