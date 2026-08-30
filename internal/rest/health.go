package rest

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

type webhookHealthOutput struct {
	Body struct {
		WebhookID              string  `json:"webhook_id" doc:"Webhook id (UUID)."`
		Health                 string  `json:"health" enum:"healthy,degraded,unhealthy,unknown" doc:"Computed rolling health status."`
		TotalDeliveries        int     `json:"total_deliveries" doc:"Total deliveries recorded for this webhook."`
		SuccessfulDeliveries   int     `json:"successful_deliveries" doc:"Deliveries that succeeded."`
		FailedDeliveries       int     `json:"failed_deliveries" doc:"Deliveries that failed."`
		ConsecutiveFailures    int     `json:"consecutive_failures" doc:"Current streak of consecutive failed deliveries; resets on success."`
		SuccessRate            float64 `json:"success_rate" doc:"Rolling success rate, 0.0 to 1.0."`
		AvgResponseTime        int     `json:"avg_response_time" doc:"Average endpoint response time, in milliseconds."`
		ClientErrors           int     `json:"client_errors" doc:"Count of 4xx failures."`
		ServerErrors           int     `json:"server_errors" doc:"Count of 5xx failures."`
		TimeoutErrors          int     `json:"timeout_errors" doc:"Count of request/connection timeout failures."`
		NetworkErrors          int     `json:"network_errors" doc:"Count of other network-level failures (DNS, TLS, connection refused, reset)."`
		UnexpectedStatusErrors int     `json:"unexpected_status_errors" doc:"Count of responses outside the webhook's configured expected_status_codes."`
	}
}

type healthSummaryOutput struct {
	Body struct {
		HealthyCount   int `json:"healthy_count" doc:"Webhooks currently healthy, across all namespaces."`
		DegradedCount  int `json:"degraded_count" doc:"Webhooks currently degraded, across all namespaces."`
		UnhealthyCount int `json:"unhealthy_count" doc:"Webhooks currently unhealthy, across all namespaces."`
		UnknownCount   int `json:"unknown_count" doc:"Webhooks with no recent delivery attempts, across all namespaces."`
	}
}

// listWebhooksGlobalInput is global (no namespace) — used both for
// cross-namespace health-filtered dashboards and for looking up a specific
// webhook by id when its namespace is unknown (ListWebhooks supports empty
// namespace = search all namespaces).
type listWebhooksGlobalInput struct {
	Health    string `query:"health,omitempty" enum:"healthy,degraded,unhealthy,unknown," doc:"Filter to webhooks with this computed health status."`
	WebhookID string `query:"webhook_id,omitempty" doc:"Look up a single webhook by id when its namespace is unknown."`
	Limit     int32  `query:"limit" default:"50" doc:"Maximum items to return."`
	Offset    int32  `query:"offset" default:"0" doc:"Number of items to skip, for pagination."`
}

func registerHealthRoutes(api huma.API, d *Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "getWebhookHealth",
		Method:      http.MethodGet,
		Path:        "/v1/namespaces/{namespace}/webhooks/{webhook_id}/health",
		Summary:     "Get health status and metrics for a webhook",
		Description: "Returns one webhook's computed health status (healthy/degraded/unhealthy) plus rolling delivery counts and per-category error counts (client, server, timeout, network, unexpected-status).",
		Errors:      []int{404},
		Tags:        []string{"Health"},
	}, func(ctx context.Context, in *webhookIDInput) (*webhookHealthOutput, error) {
		h, err := d.Svc.GetWebhookHealth(ctx, in.WebhookID, in.Namespace)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get webhook health")
		}
		out := &webhookHealthOutput{}
		out.Body.WebhookID = h.WebhookID
		out.Body.Health = string(h.Health)
		out.Body.TotalDeliveries = h.TotalDeliveries
		out.Body.SuccessfulDeliveries = h.SuccessfulDeliveries
		out.Body.FailedDeliveries = h.FailedDeliveries
		out.Body.ConsecutiveFailures = h.ConsecutiveFailures
		out.Body.SuccessRate = h.SuccessRate
		out.Body.AvgResponseTime = h.AvgResponseTime
		out.Body.ClientErrors = h.ClientErrors
		out.Body.ServerErrors = h.ServerErrors
		out.Body.TimeoutErrors = h.TimeoutErrors
		out.Body.NetworkErrors = h.NetworkErrors
		out.Body.UnexpectedStatusErrors = h.UnexpectedStatusErrors
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getHealthSummary",
		Method:      http.MethodGet,
		Path:        "/v1/health-summary",
		Summary:     "Get aggregate webhook health counts across all namespaces",
		Description: "Returns how many webhooks are currently healthy, degraded, unhealthy, or unknown, across every namespace — for a top-level dashboard tile.",
		Tags:        []string{"Health"},
	}, func(ctx context.Context, in *struct{}) (*healthSummaryOutput, error) {
		s, err := d.Svc.GetHealthSummary(ctx)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get health summary")
		}
		out := &healthSummaryOutput{}
		out.Body.HealthyCount = s.HealthyCount
		out.Body.DegradedCount = s.DegradedCount
		out.Body.UnhealthyCount = s.UnhealthyCount
		out.Body.UnknownCount = s.UnknownCount
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listWebhooksByHealth",
		Method:      http.MethodGet,
		Path:        "/v1/webhooks",
		Summary:     "List webhooks across all namespaces filtered by computed health status",
		Description: "Cross-namespace webhook listing. Pass health to filter by computed status, or webhook_id for an id-only lookup when the namespace isn't known.",
		Tags:        []string{"Health"},
	}, func(ctx context.Context, in *listWebhooksGlobalInput) (*listWebhooksOutput, error) {
		var regs []*store.WebhookRegistration
		var total int32
		var err error
		if in.Health != "" {
			regs, total, err = d.Svc.ListWebhooksByHealth(ctx, store.WebhookHealth(in.Health), in.Limit, in.Offset)
		} else {
			regs, total, err = d.Svc.ListWebhooks(ctx, "", in.WebhookID, "", false, in.Limit, in.Offset)
		}
		if err != nil {
			return nil, mapError(ctx, err, "failed to list webhooks")
		}
		eventsMap := getWebhookEventsMap(ctx, d.Svc, regs)
		out := &listWebhooksOutput{}
		out.Body.Items = make([]WebhookOut, len(regs))
		for i, r := range regs {
			out.Body.Items[i] = toWebhookOut(r, eventsMap[r.ID.String()], d.Svc)
		}
		out.Body.Pagination = newPagination(in.Limit, in.Offset, total)
		return out, nil
	})
}
