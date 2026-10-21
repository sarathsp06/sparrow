package rest

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sarathsp06/sparrow/internal/webhooks"
)

type sendTestEventInput struct {
	Consumer       string `path:"consumer" doc:"Tenant consumer the subscription belongs to."`
	SubscriptionID string `path:"subscription_id" doc:"Subscription to send the event to: its transform renders the payload, and its webhook receives it."`
	Body           struct {
		EventName string         `json:"event_name" required:"true" doc:"Event to send: the subscription's event, or any event for a catch-all (*) subscription."`
		Payload   map[string]any `json:"payload,omitempty" doc:"Event payload to send instead of the event type's sample payload. It is rendered through the subscription's transform like a pushed event."`
	}
}

type sendTestEventOutput struct {
	Body struct {
		StatusCode int    `json:"status_code" doc:"The receiver's HTTP status; 0 when no response was received."`
		LatencyMs  int64  `json:"latency_ms" doc:"Time to the response headers, in milliseconds."`
		ErrorClass string `json:"error_class,omitempty" doc:"Delivery error category when the send did not get a 2xx: client_error, server_error, rate_limited, timeout, connection_refused, dns_error, tls_error, network_error, redirect, template_error (nothing was sent) or unknown."`
	}
}

func registerTestEventRoutes(api huma.API, svc webhooks.TestEventSender) {
	huma.Register(api, huma.Operation{
		OperationID: "sendTestEvent",
		Method:      http.MethodPost,
		Path:        "/v1/consumers/{consumer}/subscriptions/{subscription_id}:sendTestEvent",
		Summary:     "Send a test event to a subscription",
		Description: "Sends event_name to the subscription: one signed delivery of payload, or of the event type's sample when payload is omitted, rendered through the subscription's transform as a real delivery would be, to its webhook's URL, synchronously, and returns what the receiver answered. " +
			"It uses the same request builder, signing, network policy (SSRF protection) and timeout as real deliveries and adds the header X-Sparrow-Test: true. " +
			"It creates no delivery and never counts toward the webhook's health or auto-disable. A receiver error is reported in the response (status_code, error_class), not as an API error.\n\n" +
			"One test event per webhook every 2 seconds on each server: more answer 429. 404 for a subscription not in this consumer. 400 when the subscription does not receive event_name, or when the event type is not registered and no payload is given.\n\nPortal tokens: allowed, as POST /portal/api/subscriptions/{subscription_id}:sendTestEvent.",
		Errors: []int{400, 404, 429, 503},
		Tags:   []string{"Subscriptions"},
	}, func(ctx context.Context, in *sendTestEventInput) (*sendTestEventOutput, error) {
		res, err := svc.SendTestEvent(ctx, in.Consumer, in.SubscriptionID, in.Body.EventName, in.Body.Payload)
		if err != nil {
			return nil, mapError(ctx, err, "failed to send the test event")
		}
		out := &sendTestEventOutput{}
		out.Body.StatusCode = res.StatusCode
		out.Body.LatencyMs = res.Latency.Milliseconds()
		out.Body.ErrorClass = res.ErrorClass
		return out, nil
	})
}
