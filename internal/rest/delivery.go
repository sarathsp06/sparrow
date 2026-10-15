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
	Consumer   string `path:"consumer" doc:"Consumer the delivery belongs to."`
	DeliveryID string `path:"delivery_id" doc:"Delivery id (UUID)."`
}

// deliveryIDOnlyInput is for single-delivery operations the domain layer
// supports consumer-agnostically (delivery IDs are globally unique).
type deliveryIDOnlyInput struct {
	DeliveryID string `path:"delivery_id" doc:"Delivery id (UUID)."`
}

type deliveryItem struct {
	DeliveryID         string  `json:"delivery_id" doc:"Delivery id (UUID)."`
	WebhookID          string  `json:"webhook_id" doc:"Webhook this delivery was sent to."`
	WebhookDescription string  `json:"webhook_description,omitempty" doc:"Description of that webhook. Set in delivery listings."`
	WebhookURL         string  `json:"webhook_url,omitempty" doc:"URL of that webhook. Set in delivery listings."`
	Consumer           string  `json:"consumer,omitempty" doc:"Consumer of the webhook. Set in delivery listings."`
	EventID            string  `json:"event_id" doc:"Pushed event occurrence this delivery originated from."`
	EventName          string  `json:"event_name,omitempty" doc:"Event type of that occurrence, e.g. order.created. Set in delivery listings."`
	Status             string  `json:"status" enum:"pending,sending,success,failed,retrying,expired,paused" doc:"Current delivery status. pending: queued for its first attempt (or re-queued by a retry). sending: an attempt is in flight. retrying: an attempt failed and another is scheduled (next_retry_at). success: an attempt got an expected status code. failed: attempts exhausted, or a non-retryable error such as template_error. expired: the event's ttl_seconds passed before the attempt. paused: held because the webhook or subscription was paused; not attempted until retried."`
	AttemptCount       int     `json:"attempt_count" doc:"Number of delivery attempts made so far, including earlier rounds before a manual retry."`
	MaxAttempts        int     `json:"max_attempts" doc:"Attempts allowed per round before the delivery is marked failed: the webhook's max_retries + 1 when the delivery was created. A manual retry starts a new round."`
	ResponseCode       int     `json:"response_code,omitempty" doc:"HTTP status code returned by the endpoint on the most recent attempt, if any."`
	ResponseBody       string  `json:"response_body,omitempty" doc:"Endpoint response body from the most recent attempt: up to 1 KiB, or up to SPARROW_MAX_CAPTURED_RESPONSE_BYTES when the webhook has capture_response_body."`
	ErrorMessage       string  `json:"error_message,omitempty" doc:"Human-readable failure reason from the most recent attempt."`
	ErrorCategory      string  `json:"error_category,omitempty" enum:"success,client_error,server_error,timeout,dns_error,tls_error,connection_refused,network_error,rate_limited,unexpected_status,template_error,unknown," doc:"Outcome classification of the most recent attempt, used to decide retryability. server_error (5xx), timeout, connection_refused, network_error and rate_limited (429) are retried; client_error (other 4xx), dns_error, tls_error and unexpected_status (a 2xx/3xx not in expected_status_codes) are not. template_error means the subscription's transform failed to render; it is not retried automatically and does not count against the webhook's health."`
	TemplateError      string  `json:"template_error,omitempty" doc:"The payload transform error, when the subscription's template failed to render. Set for both on_transform_error modes: with fail the delivery was not sent; with fallback the default envelope was sent instead."`
	CreatedAt          string  `json:"created_at" doc:"When the delivery was created by fan-out, RFC3339."`
	LastAttemptedAt    *string `json:"last_attempted_at,omitempty" doc:"Timestamp of the most recent attempt, RFC3339."`
	NextRetryAt        *string `json:"next_retry_at,omitempty" doc:"Timestamp of the next scheduled retry, RFC3339, if one is pending."`
}

type deliveryOutput struct {
	Body deliveryItem
}

func toDeliveryItem(dl *store.WebhookDelivery) deliveryItem {
	item := deliveryItem{
		DeliveryID:         dl.ID.String(),
		WebhookID:          dl.WebhookID.String(),
		WebhookDescription: dl.WebhookDescription,
		WebhookURL:         dl.WebhookURL,
		Consumer:           dl.Consumer,
		EventID:            dl.EventID.String(),
		EventName:          dl.EventName,
		Status:             string(dl.Status),
		AttemptCount:       dl.AttemptCount,
		MaxAttempts:        dl.MaxAttempts,
		ResponseCode:       dl.ResponseCode,
		ResponseBody:       dl.ResponseBody,
		ErrorMessage:       dl.ErrorMessage,
		ErrorCategory:      dl.ErrorCategory,
		TemplateError:      dl.TemplateError,
		CreatedAt:          dl.CreatedAt.Format(time.RFC3339Nano),
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
	WebhookID      string `query:"webhook_id,omitempty" doc:"Filter to deliveries for one webhook (UUID; 400 otherwise)."`
	EventID        string `query:"event_id,omitempty" doc:"Filter to deliveries for one pushed event occurrence (UUID; 400 otherwise)."`
	Status         string `query:"status,omitempty" enum:"pending,sending,success,failed,retrying,expired,paused" doc:"Filter by delivery status."`
	ErrorCategory  string `query:"error_category,omitempty" doc:"Filter by the most recent attempt's error_category, e.g. server_error, client_error, timeout."`
	SubscriptionID string `query:"subscription_id,omitempty" doc:"Filter to deliveries created by one subscription (UUID), e.g. its paused deliveries."`
	EventName      string `query:"event_name,omitempty" doc:"Filter to deliveries of one event type, e.g. order.created. Also matches deliveries whose subscription has since been replaced or deleted."`
	CreatedAfter   string `query:"created_after,omitempty" doc:"Only deliveries created at or after this time: YYYY-MM-DD (start of that day, UTC) or an RFC3339 timestamp (e.g. an import's imported_at)."`
	CreatedBefore  string `query:"created_before,omitempty" doc:"Only deliveries created at or before this time: YYYY-MM-DD (through the end of that day, UTC) or an RFC3339 timestamp."`
	PrepareRetry   bool   `query:"prepare_retry" default:"false" doc:"Side effect: when true, also snapshots the ids of every delivery matching the filters (not just this page; at most 10,000, more is an error) into a batch job and returns its id as retry_id. Pass it as repush_id to POST .../deliveries:retryBatch within 15 minutes. Each call creates a new snapshot."`
	Limit          int32  `query:"limit" default:"50" minimum:"1" maximum:"1000" doc:"Maximum items to return, 1 to 1000."`
	Cursor         string `query:"cursor,omitempty" doc:"Continue after the page this came from: pagination.next_cursor of the previous response. Stable while new deliveries arrive."`
	Offset         int32  `query:"offset" default:"0" doc:"Removed: page with cursor. Any value other than 0 is rejected with 400, so a client still paging by offset fails loudly instead of re-reading the first page."`
}

type listDeliveriesInput struct {
	Consumer string `path:"consumer" doc:"Consumer to list deliveries in."`
	DeliveryListParams
}

type listDeliveriesGlobalInput struct {
	Consumer string `query:"consumer,omitempty" doc:"Filter to one consumer; omit to list deliveries across all consumers."`
	DeliveryListParams
}

type listDeliveriesOutput struct {
	Body struct {
		Items      []deliveryItem         `json:"items" doc:"Deliveries, newest first. Includes deliveries of deleted webhooks."`
		Pagination CursorPaginationOutput `json:"pagination" doc:"Cursor pagination; there is no total count."`
		RetryID    string                 `json:"retry_id,omitempty" doc:"Snapshot (batch job) id, present when prepare_retry was set and something matched. Send it as the repush_id field of POST .../deliveries:retryBatch. Expires after 15 minutes if not started."`
		RetryTotal int                    `json:"retry_total,omitempty" doc:"Number of deliveries in that snapshot (all matches, not just this page)."`
	}
}

type retryDeliveryInput struct {
	Consumer   string `path:"consumer" doc:"Consumer the delivery belongs to."`
	DeliveryID string `path:"delivery_id" doc:"Delivery id (UUID) to retry."`
}

type retryDeliveriesByWebhookInput struct {
	Consumer string `path:"consumer" doc:"Consumer the webhook belongs to."`
	Body     struct {
		WebhookID string `json:"webhook_id" required:"true" doc:"Webhook id (UUID) whose deliveries to retry."`
		Force     bool   `json:"force,omitempty" doc:"false (default) retries deliveries in status failed, pending, retrying or paused. true retries every delivery of the webhook, including successful and expired ones, which resends them."`
	}
}

type retryOutput struct {
	Body struct {
		Count       int32    `json:"count" doc:"Number of deliveries queued for another round of attempts."`
		DeliveryIDs []string `json:"delivery_ids,omitempty" doc:"Ids of the deliveries queued. Absent when nothing matched."`
	}
}

type attemptItem struct {
	Success       bool   `json:"success" doc:"Whether this attempt succeeded (got one of the webhook's expected_status_codes)."`
	ResponseTime  int    `json:"response_time" doc:"Round-trip time of this attempt, in milliseconds."`
	ResponseCode  int    `json:"response_code" doc:"HTTP status code returned by the endpoint on this attempt; 0 when no response was received."`
	ErrorMessage  string `json:"error_message,omitempty" doc:"Human-readable failure reason for this attempt."`
	ErrorCategory string `json:"error_category,omitempty" enum:"success,client_error,server_error,timeout,dns_error,tls_error,connection_refused,network_error,rate_limited,unexpected_status,template_error,unknown," doc:"Failure classification for this attempt."`
	Timestamp     string `json:"timestamp" doc:"When this attempt was made, RFC3339."`
}

type attemptsOutput struct {
	Body struct {
		Items []attemptItem `json:"items" doc:"Every recorded attempt, oldest first."`
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
		Description: "Returns one delivery's status, latest response code/body, and error classification. 404 when it does not exist in this consumer.\n\nPortal tokens: allowed, as GET /portal/api/deliveries/{delivery_id}.",
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
		Description: "Lists deliveries in a consumer, newest first, optionally filtered by webhook, event occurrence, subscription, event type, status, error category, or creation time. Cursor paginated: pass pagination.next_cursor as cursor for the next page; there is no total count and offset is rejected.\n\n" +
			"Side effect: prepare_retry=true also snapshots every match (up to 10,000) into a batch job and returns retry_id for POST .../deliveries:retryBatch.\n\nPortal tokens: allowed, as GET /portal/api/deliveries.",
		Tags: []string{"Deliveries"},
	}, func(ctx context.Context, in *listDeliveriesInput) (*listDeliveriesOutput, error) {
		return listDeliveriesImpl(ctx, svc, in.Consumer, in.DeliveryListParams)
	})

	huma.Register(api, huma.Operation{
		OperationID: "listDeliveriesGlobal",
		Method:      http.MethodGet,
		Path:        "/v1/deliveries",
		Summary:     "List deliveries across all consumers",
		Description: "Lists deliveries across every consumer, newest first, with the same filters, cursor pagination and prepare_retry side effect as the per-consumer route. Pass consumer to scope to one consumer.\n\nPortal tokens: not allowed.",
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *listDeliveriesGlobalInput) (*listDeliveriesOutput, error) {
		return listDeliveriesImpl(ctx, svc, in.Consumer, in.DeliveryListParams)
	})

	huma.Register(api, huma.Operation{
		OperationID: "retryDelivery",
		Method:      http.MethodPost,
		Path:        "/v1/consumers/{consumer}/deliveries/{delivery_id}:retry",
		Summary:     "Retry a single delivery",
		Description: "Queues one delivery for another round of attempts right away: it is reset to pending and gets up to max_attempts new attempts, without TTL expiry. Works for failed, expired, paused, pending and retrying deliveries; a successful one is refused with 409 (use the bulk route with force to resend it). If the webhook or subscription is still paused the delivery is held as paused again. Not idempotent: each call queues another round.\n\n" +
			"Portal tokens: allowed, as POST /portal/api/deliveries/{delivery_id}:retry.",
		Errors: []int{400, 404, 409},
		Tags:   []string{"Deliveries"},
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
		Description: "Queues every failed, pending, retrying or paused delivery of one webhook for another round of attempts (see retryDelivery), in one request. With force, every delivery of the webhook is resent, including successful and expired ones. Returns count 0 when nothing matched. For a filtered or very large set, use prepare_retry and :retryBatch instead.\n\n" +
			"404 when the webhook does not exist in this consumer.\n\nPortal tokens: allowed, as POST /portal/api/deliveries:retry.",
		Errors: []int{400, 404},
		Tags:   []string{"Deliveries"},
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
		Description: "Returns the per-attempt record for one delivery, oldest first: response code, timing, and error classification for every attempt made so far, across manual retries. A delivery that does not exist or belongs to another consumer answers 404.\n\nPortal tokens: allowed, as GET /portal/api/deliveries/{delivery_id}/attempts.",
		Errors:      []int{404},
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *deliveryIDInput) (*attemptsOutput, error) {
		attempts, err := svc.GetDeliveryAttempts(ctx, in.DeliveryID, in.Consumer)
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
		OperationID: "startDeliveryRetryJob",
		Method:      http.MethodPost,
		Path:        "/v1/consumers/{consumer}/deliveries:retryBatch",
		Summary:     "Start a batch retry job from a prepared snapshot",
		Description: "Starts an async job that retries every delivery in a snapshot taken by an earlier GET .../deliveries?prepare_retry=true call, like retryDelivery for each one. The body field is named repush_id, but it takes the retry_id returned by the deliveries list. Returns 202 with the job; poll it with getDeliveryRetryJob.\n\n" +
			"A snapshot can be started once: 409 when it is not pending (already started or cancelled), expired (15 minutes after it was taken), or is an event re-push snapshot; 404 when the id is unknown, 400 when it is not a UUID.\n\nPortal tokens: allowed, as POST /portal/api/deliveries:retryBatch.",
		Errors:        []int{400, 404, 409},
		Tags:          []string{"Deliveries"},
		DefaultStatus: http.StatusAccepted,
	}, func(ctx context.Context, in *repushBatchInput) (*batchJobOutput, error) {
		if err := svc.RetryDeliveries(ctx, batchScope(ctx, in.Consumer), in.Body.RepushID); err != nil {
			return nil, mapError(ctx, err, "failed to start retry job")
		}
		job, err := svc.GetRetryStatus(ctx, batchScope(ctx, in.Consumer), in.Body.RepushID)
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
		Description: "Returns a batch retry job's status and processed/failed/total counts. 404 when the id is unknown, 409 when the id belongs to an event re-push job.\n\nPortal tokens: allowed, as GET /portal/api/retry-jobs/{job_id}.",
		Errors:      []int{400, 404, 409},
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *jobIDInput) (*batchJobOutput, error) {
		job, err := svc.GetRetryStatus(ctx, batchScope(ctx, in.Consumer), in.JobID)
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
		Description:   "Cancels a pending or running batch retry job; a running job stops at its next progress check (every 25 items). Deliveries already queued are not rolled back. Returns 204. 409 when the job is already completed or cancelled.\n\nPortal tokens: allowed, as POST /portal/api/retry-jobs/{job_id}:cancel.",
		Errors:        []int{400, 404, 409},
		Tags:          []string{"Deliveries"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *jobIDInput) (*emptyOutput, error) {
		if err := svc.CancelRetry(ctx, batchScope(ctx, in.Consumer), in.JobID); err != nil {
			return nil, mapError(ctx, err, "failed to cancel retry job")
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getDeliveryGlobal",
		Method:      http.MethodGet,
		Path:        "/v1/deliveries/{delivery_id}",
		Summary:     "Get a delivery by id (any consumer)",
		Description: "Looks a delivery up by id in any consumer, for callers that only have the id (e.g. the X-Sparrow-Delivery-ID header a receiver logged). 404 when it does not exist.\n\nPortal tokens: not allowed.",
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
		Description: "Same as getDeliveryAttempts, without a consumer: the delivery is looked up by id alone, and an unknown id returns an empty list.\n\nPortal tokens: not allowed.",
		Errors:      []int{404},
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *deliveryIDOnlyInput) (*attemptsOutput, error) {
		attempts, err := svc.GetDeliveryAttempts(ctx, in.DeliveryID, "")
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
		Description: "Same as retryDelivery, for a delivery in any consumer.\n\nPortal tokens: not allowed.",
		Errors:      []int{400, 404, 409},
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
	if p.Offset != 0 {
		return nil, errOffsetRemoved
	}
	filter := store.DeliveryFilter{
		Consumer:     consumer,
		Limit:        int(p.Limit),
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
