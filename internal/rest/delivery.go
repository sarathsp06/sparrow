package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

type deliveryIDInput struct {
	Consumer   string `path:"consumer" doc:"Tenant consumer the delivery belongs to."`
	DeliveryID string `path:"delivery_id" doc:"Delivery id (UUID)."`
}

// deliveryIDOnlyInput is for single-delivery operations the domain layer
// supports consumer-agnostically (delivery IDs are globally unique).
type deliveryIDOnlyInput struct {
	DeliveryID string `path:"delivery_id"`
}

type deliveryItem struct {
	DeliveryID      string  `json:"delivery_id" doc:"Delivery id (UUID)."`
	WebhookID       string  `json:"webhook_id" doc:"Webhook this delivery was sent to."`
	Consumer        string  `json:"consumer,omitempty" doc:"Consumer of the webhook. Set in delivery listings."`
	EventID         string  `json:"event_id" doc:"Pushed event occurrence this delivery originated from."`
	EventName       string  `json:"event_name,omitempty" doc:"Event type of that occurrence, e.g. order.created. Set in delivery listings."`
	Status          string  `json:"status" enum:"pending,sending,success,failed,retrying,expired,paused" doc:"Current delivery status. paused: created while its subscription was paused; not attempted until retried."`
	AttemptCount    int     `json:"attempt_count" doc:"Number of delivery attempts made so far."`
	MaxAttempts     int     `json:"max_attempts" doc:"Maximum attempts allowed before the delivery is marked failed."`
	ResponseCode    int     `json:"response_code,omitempty" doc:"HTTP status code returned by the endpoint on the most recent attempt, if any."`
	ResponseBody    string  `json:"response_body,omitempty" doc:"Endpoint response body from the most recent attempt, if capture_response_body is enabled."`
	ErrorMessage    string  `json:"error_message,omitempty" doc:"Human-readable failure reason from the most recent attempt."`
	ErrorCategory   string  `json:"error_category,omitempty" enum:"success,client_error,server_error,timeout,dns_error,tls_error,connection_refused,network_error,rate_limited,unexpected_status,template_error,unknown," doc:"Failure classification from the most recent attempt, used to decide retryability. template_error means the subscription's transform failed to render; it is not retried automatically and does not count against the webhook's health."`
	TemplateError   string  `json:"template_error,omitempty" doc:"The payload transform error, when the subscription's template failed to render. Set for both on_transform_error modes: with fail the delivery was not sent; with fallback the default envelope was sent instead."`
	CreatedAt       string  `json:"created_at" doc:"Creation timestamp, RFC3339."`
	LastAttemptedAt *string `json:"last_attempted_at,omitempty" doc:"Timestamp of the most recent attempt, RFC3339."`
	NextRetryAt     *string `json:"next_retry_at,omitempty" doc:"Timestamp of the next scheduled retry, RFC3339, if one is pending."`
}

type deliveryOutput struct {
	Body deliveryItem
}

func toDeliveryItem(dl *store.WebhookDelivery) deliveryItem {
	item := deliveryItem{
		DeliveryID:    dl.ID.String(),
		WebhookID:     dl.WebhookID.String(),
		Consumer:      dl.Consumer,
		EventID:       dl.EventID.String(),
		EventName:     dl.EventName,
		Status:        string(dl.Status),
		AttemptCount:  dl.AttemptCount,
		MaxAttempts:   dl.MaxAttempts,
		ResponseCode:  dl.ResponseCode,
		ResponseBody:  dl.ResponseBody,
		ErrorMessage:  dl.ErrorMessage,
		ErrorCategory: dl.ErrorCategory,
		TemplateError: dl.TemplateError,
		CreatedAt:     dl.CreatedAt.Format(time.RFC3339Nano),
	}
	if dl.LastAttemptedAt != nil {
		s := dl.LastAttemptedAt.Format(time.RFC3339Nano)
		item.LastAttemptedAt = &s
	}
	if dl.NextRetryAt != nil {
		s := dl.NextRetryAt.Format(time.RFC3339Nano)
		item.NextRetryAt = &s
	}
	return item
}

func toDeliveryOutput(dl *store.WebhookDelivery) *deliveryOutput {
	return &deliveryOutput{Body: toDeliveryItem(dl)}
}

// DeliveryListParams are the query filters shared by the consumer-scoped and
// global delivery list routes.
type DeliveryListParams struct {
	WebhookID      string `query:"webhook_id,omitempty" doc:"Filter to deliveries for one webhook."`
	EventID        string `query:"event_id,omitempty" doc:"Filter to deliveries for one pushed event occurrence."`
	Status         string `query:"status,omitempty" doc:"Filter by delivery status (e.g. pending, success, failed, retrying, paused)."`
	ErrorCategory  string `query:"error_category,omitempty" doc:"Filter by failure classification (e.g. server_error, client_error, timeout)."`
	SubscriptionID string `query:"subscription_id,omitempty" doc:"Filter to deliveries created by one subscription, e.g. its paused deliveries."`
	EventName      string `query:"event_name,omitempty" doc:"Filter to deliveries of one event type, e.g. order.created. Also matches deliveries whose subscription has since been replaced or deleted."`
	CreatedAfter   string `query:"created_after,omitempty" doc:"Filter to deliveries created on or after this date (YYYY-MM-DD) or exact time (RFC3339, e.g. an import's imported_at)."`
	CreatedBefore  string `query:"created_before,omitempty" doc:"Filter to deliveries created on or before this date (YYYY-MM-DD) or exact time (RFC3339)."`
	PrepareRetry   bool   `query:"prepare_retry" default:"false" doc:"If true, snapshot the matching deliveries into a retry_id you can pass to the batch retry endpoint."`
	Limit          int32  `query:"limit" default:"50" minimum:"1" maximum:"1000" doc:"Maximum items to return."`
	Cursor         string `query:"cursor,omitempty" doc:"Continue after the page this came from: pagination.next_cursor of the previous response. Stable while new deliveries arrive."`
	Offset         int32  `query:"offset" default:"0" doc:"Deprecated: use cursor. Number of items to skip."`
}

type listDeliveriesInput struct {
	Consumer string `path:"consumer" doc:"Tenant consumer to list deliveries in."`
	DeliveryListParams
}

type listDeliveriesGlobalInput struct {
	Consumer string `query:"consumer,omitempty" doc:"Filter to one consumer; omit to list deliveries across all consumers."`
	DeliveryListParams
}

type listDeliveriesOutput struct {
	Body struct {
		Items      []deliveryItem         `json:"items"`
		Pagination CursorPaginationOutput `json:"pagination"`
		RetryID    string                 `json:"retry_id,omitempty" doc:"Snapshot id for the batch retry endpoint, present when prepare_retry was set and something matched."`
		RetryTotal int                    `json:"retry_total,omitempty" doc:"Number of deliveries in that snapshot."`
	}
}

type retryDeliveryInput struct {
	Consumer   string `path:"consumer" doc:"Tenant consumer the delivery belongs to."`
	DeliveryID string `path:"delivery_id" doc:"Delivery id (UUID) to retry."`
}

type retryDeliveriesByWebhookInput struct {
	Consumer string `path:"consumer" doc:"Tenant consumer the webhook belongs to."`
	Body     struct {
		WebhookID string `json:"webhook_id" required:"true" doc:"Retry every eligible delivery for this webhook."`
		Force     bool   `json:"force,omitempty" doc:"If true, also retry deliveries that already exhausted their max attempts."`
	}
}

type retryOutput struct {
	Body struct {
		Count       int32    `json:"count" doc:"Number of deliveries retried."`
		DeliveryIDs []string `json:"delivery_ids,omitempty" doc:"Ids of the deliveries that were retried."`
	}
}

type attemptItem struct {
	Success       bool   `json:"success" doc:"Whether this attempt succeeded (matched an expected status code)."`
	ResponseTime  int    `json:"response_time" doc:"Round-trip time of this attempt, in milliseconds."`
	ResponseCode  int    `json:"response_code" doc:"HTTP status code returned by the endpoint on this attempt."`
	ErrorMessage  string `json:"error_message,omitempty" doc:"Human-readable failure reason for this attempt."`
	ErrorCategory string `json:"error_category,omitempty" enum:"success,client_error,server_error,timeout,dns_error,tls_error,connection_refused,network_error,rate_limited,unexpected_status,template_error,unknown," doc:"Failure classification for this attempt."`
	Timestamp     string `json:"timestamp" doc:"When this attempt was made, RFC3339."`
}

type attemptsOutput struct {
	Body struct {
		Items []attemptItem `json:"items"`
	}
}

// deliveryRouteService is the service slice delivery routes consume: delivery
// status/retry plus bulk retry jobs.
type deliveryRouteService interface {
	webhooks.DeliveryManager
	webhooks.BatchManager
}

func registerDeliveryRoutes(api huma.API, svc deliveryRouteService) {
	huma.Register(api, huma.Operation{
		OperationID: "getDelivery",
		Method:      http.MethodGet,
		Path:        "/v1/consumers/{consumer}/deliveries/{delivery_id}",
		Summary:     "Get a delivery by id",
		Description: "Fetches one delivery's status, response code/body, and error classification.",
		Errors:      []int{404},
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *deliveryIDInput) (*deliveryOutput, error) {
		dl, err := svc.GetDeliveryStatus(ctx, in.DeliveryID, in.Consumer)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get delivery")
		}
		return toDeliveryOutput(dl), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listDeliveries",
		Method:      http.MethodGet,
		Path:        "/v1/consumers/{consumer}/deliveries",
		Summary:     "List deliveries",
		Description: "Lists deliveries in a consumer, optionally filtered by webhook, event occurrence, or status. Set prepare_retry to snapshot the filtered set for the batch retry endpoint.",
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *listDeliveriesInput) (*listDeliveriesOutput, error) {
		return listDeliveriesImpl(ctx, svc, in.Consumer, in.DeliveryListParams)
	})

	huma.Register(api, huma.Operation{
		OperationID: "listDeliveriesGlobal",
		Method:      http.MethodGet,
		Path:        "/v1/deliveries",
		Summary:     "List deliveries across all consumers",
		Description: "Cross-consumer delivery listing with the same filters as the per-consumer route. Pass consumer to scope to one consumer.",
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *listDeliveriesGlobalInput) (*listDeliveriesOutput, error) {
		return listDeliveriesImpl(ctx, svc, in.Consumer, in.DeliveryListParams)
	})

	huma.Register(api, huma.Operation{
		OperationID: "retryDelivery",
		Method:      http.MethodPost,
		Path:        "/v1/consumers/{consumer}/deliveries/{delivery_id}:retry",
		Summary:     "Retry a single delivery",
		Description: "Immediately re-attempts one delivery, regardless of its current status or remaining attempt budget.",
		Errors:      []int{404},
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *retryDeliveryInput) (*retryOutput, error) {
		ids, count, err := svc.RetryDelivery(ctx, in.Consumer, in.DeliveryID, "", false)
		if err != nil {
			return nil, mapError(ctx, err, "failed to retry delivery")
		}
		out := &retryOutput{}
		out.Body.Count = count
		out.Body.DeliveryIDs = ids
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "retryDeliveriesByWebhook",
		Method:      http.MethodPost,
		Path:        "/v1/consumers/{consumer}/deliveries:retry",
		Summary:     "Retry deliveries in bulk for a webhook",
		Description: "Retries every eligible (failed/pending) delivery for one webhook. Set force to also retry deliveries that already exhausted max_attempts.",
		Errors:      []int{400, 404},
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *retryDeliveriesByWebhookInput) (*retryOutput, error) {
		ids, count, err := svc.RetryDelivery(ctx, in.Consumer, "", in.Body.WebhookID, in.Body.Force)
		if err != nil {
			return nil, mapError(ctx, err, "failed to retry deliveries")
		}
		out := &retryOutput{}
		out.Body.Count = count
		out.Body.DeliveryIDs = ids
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getDeliveryAttempts",
		Method:      http.MethodGet,
		Path:        "/v1/consumers/{consumer}/deliveries/{delivery_id}/attempts",
		Summary:     "Get a delivery's per-attempt history",
		Description: "Returns the full per-attempt record for one delivery: response code, timing, and error classification for every attempt made so far.",
		Errors:      []int{404},
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *deliveryIDInput) (*attemptsOutput, error) {
		attempts, err := svc.GetDeliveryAttempts(ctx, in.DeliveryID)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get delivery attempts")
		}
		out := &attemptsOutput{}
		out.Body.Items = make([]attemptItem, 0, len(attempts))
		for _, a := range attempts {
			out.Body.Items = append(out.Body.Items, attemptItem{
				Success:       a.Success,
				ResponseTime:  a.ResponseTime,
				ResponseCode:  a.ResponseCode,
				ErrorMessage:  a.ErrorMessage,
				ErrorCategory: a.ErrorCategory,
				Timestamp:     a.Timestamp.Format(time.RFC3339Nano),
			})
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "startDeliveryRetryJob",
		Method:        http.MethodPost,
		Path:          "/v1/consumers/{consumer}/deliveries:retryBatch",
		Summary:       "Start a batch retry job from a prepared snapshot",
		Description:   "Starts an async job that retries every delivery captured by an earlier prepare_retry=true list call. Poll the returned job with getDeliveryRetryJob.",
		Errors:        []int{400, 404},
		Tags:          []string{"Deliveries"},
		DefaultStatus: http.StatusAccepted,
	}, func(ctx context.Context, in *repushBatchInput) (*batchJobOutput, error) {
		if err := svc.RetryDeliveries(ctx, in.Body.RepushID); err != nil {
			return nil, mapError(ctx, err, "failed to start retry job")
		}
		job, err := svc.GetRetryStatus(ctx, in.Body.RepushID)
		if err != nil {
			return nil, mapError(ctx, err, "failed to load retry job status")
		}
		return toBatchJobOutput(job), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getDeliveryRetryJob",
		Method:      http.MethodGet,
		Path:        "/v1/consumers/{consumer}/retry-jobs/{job_id}",
		Summary:     "Get batch retry job progress",
		Description: "Returns a batch retry job's status and processed/failed/total counts.",
		Errors:      []int{404},
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *jobIDInput) (*batchJobOutput, error) {
		job, err := svc.GetRetryStatus(ctx, in.JobID)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get retry job status")
		}
		return toBatchJobOutput(job), nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "cancelDeliveryRetryJob",
		Method:        http.MethodPost,
		Path:          "/v1/consumers/{consumer}/retry-jobs/{job_id}:cancel",
		Summary:       "Cancel a pending or in-progress batch retry job",
		Description:   "Requests cancellation of a batch retry job. Deliveries already retried are not rolled back.",
		Errors:        []int{404, 409},
		Tags:          []string{"Deliveries"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *jobIDInput) (*emptyOutput, error) {
		if err := svc.CancelRetry(ctx, in.JobID); err != nil {
			return nil, mapError(ctx, err, "failed to cancel retry job")
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getDeliveryGlobal",
		Method:      http.MethodGet,
		Path:        "/v1/deliveries/{delivery_id}",
		Summary:     "Get a delivery by id (any consumer)",
		Description: "Consumer-agnostic lookup by delivery id, for callers that only have the id (e.g. a webhook-signature verification failure report) and don't know which consumer it belongs to.",
		Errors:      []int{404},
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *deliveryIDOnlyInput) (*deliveryOutput, error) {
		dl, err := svc.GetDeliveryStatus(ctx, in.DeliveryID, "")
		if err != nil {
			return nil, mapError(ctx, err, "failed to get delivery")
		}
		return toDeliveryOutput(dl), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getDeliveryAttemptsGlobal",
		Method:      http.MethodGet,
		Path:        "/v1/deliveries/{delivery_id}/attempts",
		Summary:     "Get a delivery's per-attempt history (any consumer)",
		Description: "Consumer-agnostic variant of getDeliveryAttempts.",
		Errors:      []int{404},
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *deliveryIDOnlyInput) (*attemptsOutput, error) {
		attempts, err := svc.GetDeliveryAttempts(ctx, in.DeliveryID)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get delivery attempts")
		}
		out := &attemptsOutput{}
		out.Body.Items = make([]attemptItem, 0, len(attempts))
		for _, a := range attempts {
			out.Body.Items = append(out.Body.Items, attemptItem{
				Success:       a.Success,
				ResponseTime:  a.ResponseTime,
				ResponseCode:  a.ResponseCode,
				ErrorMessage:  a.ErrorMessage,
				ErrorCategory: a.ErrorCategory,
				Timestamp:     a.Timestamp.Format(time.RFC3339Nano),
			})
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "retryDeliveryGlobal",
		Method:      http.MethodPost,
		Path:        "/v1/deliveries/{delivery_id}:retry",
		Summary:     "Retry a single delivery (any consumer)",
		Description: "Consumer-agnostic variant of retryDelivery.",
		Errors:      []int{404},
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *deliveryIDOnlyInput) (*retryOutput, error) {
		ids, count, err := svc.RetryDelivery(ctx, "", in.DeliveryID, "", false)
		if err != nil {
			return nil, mapError(ctx, err, "failed to retry delivery")
		}
		out := &retryOutput{}
		out.Body.Count = count
		out.Body.DeliveryIDs = ids
		return out, nil
	})
}

// listDeliveriesImpl is the shared body of the consumer-scoped and global
// delivery list routes. An empty consumer lists across all consumers.
func listDeliveriesImpl(ctx context.Context, svc deliveryRouteService, consumer string, p DeliveryListParams) (*listDeliveriesOutput, error) {
	filter := store.DeliveryFilter{
		Consumer:     consumer,
		Limit:        int(p.Limit),
		Offset:       int(p.Offset),
		PrepareRetry: p.PrepareRetry,
	}
	after, err := decodeCursor(p.Cursor)
	if err != nil {
		return nil, err
	}
	filter.After = after
	if p.WebhookID != "" {
		id, err := uuid.Parse(p.WebhookID)
		if err != nil {
			return nil, huma.Error400BadRequest("webhook_id must be a valid UUID")
		}
		filter.WebhookID = &id
	}
	if p.EventID != "" {
		id, err := uuid.Parse(p.EventID)
		if err != nil {
			return nil, huma.Error400BadRequest("event_id must be a valid UUID")
		}
		filter.EventID = &id
	}
	if p.SubscriptionID != "" {
		id, err := uuid.Parse(p.SubscriptionID)
		if err != nil {
			return nil, huma.Error400BadRequest("subscription_id must be a valid UUID")
		}
		filter.SubscriptionID = &id
	}
	if p.EventName != "" {
		filter.EventName = &p.EventName
	}
	if p.Status != "" {
		filter.Status = &p.Status
	}
	if p.ErrorCategory != "" {
		filter.ErrorCategory = &p.ErrorCategory
	}
	createdAfter, err := parseDateFilter(p.CreatedAfter, false)
	if err != nil {
		return nil, huma.Error400BadRequest("created_after " + err.Error())
	}
	filter.CreatedAfter = createdAfter
	createdBefore, err := parseDateFilter(p.CreatedBefore, true)
	if err != nil {
		return nil, huma.Error400BadRequest("created_before " + err.Error())
	}
	filter.CreatedBefore = createdBefore
	page, err := svc.ListDeliveries(ctx, filter)
	if err != nil {
		return nil, mapError(ctx, err, "failed to list deliveries")
	}
	out := &listDeliveriesOutput{}
	out.Body.Items = make([]deliveryItem, 0, len(page.Items))
	for _, dl := range page.Items {
		out.Body.Items = append(out.Body.Items, toDeliveryItem(dl))
	}
	if n := len(page.Items); n > 0 {
		last := page.Items[n-1]
		out.Body.Pagination = newCursorPagination(p.Limit, page.HasMore, last.CreatedAt, last.ID)
	} else {
		out.Body.Pagination = newCursorPagination(p.Limit, false, time.Time{}, uuid.Nil)
	}
	out.Body.RetryID = page.RetryID
	out.Body.RetryTotal = page.RetryTotal
	return out, nil
}
