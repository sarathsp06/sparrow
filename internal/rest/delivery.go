package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

type deliveryIDInput struct {
	Namespace  string `path:"namespace" doc:"Tenant namespace the delivery belongs to."`
	DeliveryID string `path:"delivery_id" doc:"Delivery id (UUID)."`
}

// deliveryIDOnlyInput is for single-delivery operations the domain layer
// supports namespace-agnostically (delivery IDs are globally unique).
type deliveryIDOnlyInput struct {
	DeliveryID string `path:"delivery_id"`
}

type deliveryItem struct {
	DeliveryID      string  `json:"delivery_id" doc:"Delivery id (UUID)."`
	WebhookID       string  `json:"webhook_id" doc:"Webhook this delivery was sent to."`
	EventID         string  `json:"event_id" doc:"Pushed event occurrence this delivery originated from."`
	Status          string  `json:"status" enum:"pending,sending,success,failed,retrying,expired" doc:"Current delivery status."`
	AttemptCount    int     `json:"attempt_count" doc:"Number of delivery attempts made so far."`
	MaxAttempts     int     `json:"max_attempts" doc:"Maximum attempts allowed before the delivery is marked failed."`
	ResponseCode    int     `json:"response_code,omitempty" doc:"HTTP status code returned by the endpoint on the most recent attempt, if any."`
	ResponseBody    string  `json:"response_body,omitempty" doc:"Endpoint response body from the most recent attempt, if capture_response_body is enabled."`
	ErrorMessage    string  `json:"error_message,omitempty" doc:"Human-readable failure reason from the most recent attempt."`
	ErrorCategory   string  `json:"error_category,omitempty" enum:"success,client_error,server_error,timeout,dns_error,tls_error,connection_refused,network_error,rate_limited,unexpected_status,unknown," doc:"Failure classification from the most recent attempt, used to decide retryability."`
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
		EventID:       dl.EventID.String(),
		Status:        string(dl.Status),
		AttemptCount:  dl.AttemptCount,
		MaxAttempts:   dl.MaxAttempts,
		ResponseCode:  dl.ResponseCode,
		ResponseBody:  dl.ResponseBody,
		ErrorMessage:  dl.ErrorMessage,
		ErrorCategory: dl.ErrorCategory,
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

type listDeliveriesInput struct {
	Namespace    string `path:"namespace" doc:"Tenant namespace to list deliveries in."`
	WebhookID    string `query:"webhook_id,omitempty" doc:"Filter to deliveries for one webhook."`
	EventID      string `query:"event_id,omitempty" doc:"Filter to deliveries for one pushed event occurrence."`
	Status       string `query:"status,omitempty" doc:"Filter by delivery status (e.g. pending, success, failed, retrying)."`
	PrepareRetry bool   `query:"prepare_retry" default:"false" doc:"If true, snapshot the matching deliveries into a retry_id you can pass to the batch retry endpoint."`
	Limit        int32  `query:"limit" default:"50" doc:"Maximum items to return."`
	Offset       int32  `query:"offset" default:"0" doc:"Number of items to skip, for pagination."`
}

type listDeliveriesOutput struct {
	Body struct {
		Items      []deliveryItem   `json:"items"`
		Pagination PaginationOutput `json:"pagination"`
		RetryID    string           `json:"retry_id,omitempty" doc:"Snapshot id for the batch retry endpoint, present when prepare_retry was set."`
	}
}

type retryDeliveryInput struct {
	Namespace  string `path:"namespace" doc:"Tenant namespace the delivery belongs to."`
	DeliveryID string `path:"delivery_id" doc:"Delivery id (UUID) to retry."`
}

type retryDeliveriesByWebhookInput struct {
	Namespace string `path:"namespace" doc:"Tenant namespace the webhook belongs to."`
	Body      struct {
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
	ErrorCategory string `json:"error_category,omitempty" enum:"success,client_error,server_error,timeout,dns_error,tls_error,connection_refused,network_error,rate_limited,unexpected_status,unknown," doc:"Failure classification for this attempt."`
	Timestamp     string `json:"timestamp" doc:"When this attempt was made, RFC3339."`
}

type attemptsOutput struct {
	Body struct {
		Items []attemptItem `json:"items"`
	}
}

func registerDeliveryRoutes(api huma.API, d *Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "getDelivery",
		Method:      http.MethodGet,
		Path:        "/v1/namespaces/{namespace}/deliveries/{delivery_id}",
		Summary:     "Get a delivery by id",
		Description: "Fetches one delivery's status, response code/body, and error classification.",
		Errors:      []int{404},
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *deliveryIDInput) (*deliveryOutput, error) {
		dl, err := d.Svc.GetDeliveryStatus(ctx, in.DeliveryID, in.Namespace)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get delivery")
		}
		return toDeliveryOutput(dl), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listDeliveries",
		Method:      http.MethodGet,
		Path:        "/v1/namespaces/{namespace}/deliveries",
		Summary:     "List deliveries",
		Description: "Lists deliveries in a namespace, optionally filtered by webhook, event occurrence, or status. Set prepare_retry to snapshot the filtered set for the batch retry endpoint.",
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *listDeliveriesInput) (*listDeliveriesOutput, error) {
		limit, offset := in.Limit, in.Offset
		filter := store.DeliveryFilter{
			Namespace:    in.Namespace,
			Limit:        int(limit),
			Offset:       int(offset),
			PrepareRetry: in.PrepareRetry,
		}
		if in.WebhookID != "" {
			if id, err := uuid.Parse(in.WebhookID); err == nil {
				filter.WebhookID = &id
			}
		}
		if in.EventID != "" {
			if id, err := uuid.Parse(in.EventID); err == nil {
				filter.EventID = &id
			}
		}
		if in.Status != "" {
			filter.Status = &in.Status
		}
		deliveries, total, retryID, err := d.Svc.ListDeliveries(ctx, filter)
		if err != nil {
			return nil, mapError(ctx, err, "failed to list deliveries")
		}
		out := &listDeliveriesOutput{}
		out.Body.Items = make([]deliveryItem, 0, len(deliveries))
		for _, dl := range deliveries {
			out.Body.Items = append(out.Body.Items, toDeliveryItem(dl))
		}
		out.Body.Pagination = newPagination(limit, offset, total)
		out.Body.RetryID = retryID
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "retryDelivery",
		Method:      http.MethodPost,
		Path:        "/v1/namespaces/{namespace}/deliveries/{delivery_id}:retry",
		Summary:     "Retry a single delivery",
		Description: "Immediately re-attempts one delivery, regardless of its current status or remaining attempt budget.",
		Errors:      []int{404},
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *retryDeliveryInput) (*retryOutput, error) {
		ids, count, err := d.Svc.RetryDelivery(ctx, in.Namespace, in.DeliveryID, "", false)
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
		Path:        "/v1/namespaces/{namespace}/deliveries:retry",
		Summary:     "Retry deliveries in bulk for a webhook",
		Description: "Retries every eligible (failed/pending) delivery for one webhook. Set force to also retry deliveries that already exhausted max_attempts.",
		Errors:      []int{400, 404},
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *retryDeliveriesByWebhookInput) (*retryOutput, error) {
		ids, count, err := d.Svc.RetryDelivery(ctx, in.Namespace, "", in.Body.WebhookID, in.Body.Force)
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
		Path:        "/v1/namespaces/{namespace}/deliveries/{delivery_id}/attempts",
		Summary:     "Get a delivery's per-attempt history",
		Description: "Returns the full per-attempt record for one delivery: response code, timing, and error classification for every attempt made so far.",
		Errors:      []int{404},
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *deliveryIDInput) (*attemptsOutput, error) {
		attempts, err := d.Svc.GetDeliveryAttempts(ctx, in.DeliveryID)
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
		Path:          "/v1/namespaces/{namespace}/deliveries:retryBatch",
		Summary:       "Start a batch retry job from a prepared snapshot",
		Description:   "Starts an async job that retries every delivery captured by an earlier prepare_retry=true list call. Poll the returned job with getDeliveryRetryJob.",
		Errors:        []int{400, 404},
		Tags:          []string{"Deliveries"},
		DefaultStatus: http.StatusAccepted,
	}, func(ctx context.Context, in *repushBatchInput) (*batchJobOutput, error) {
		if err := d.Svc.RetryDeliveries(ctx, in.Body.RepushID); err != nil {
			return nil, mapError(ctx, err, "failed to start retry job")
		}
		job, err := d.Svc.GetRetryStatus(ctx, in.Body.RepushID)
		if err != nil {
			return nil, mapError(ctx, err, "failed to load retry job status")
		}
		return toBatchJobOutput(job), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getDeliveryRetryJob",
		Method:      http.MethodGet,
		Path:        "/v1/namespaces/{namespace}/retry-jobs/{job_id}",
		Summary:     "Get batch retry job progress",
		Description: "Returns a batch retry job's status and processed/failed/total counts.",
		Errors:      []int{404},
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *jobIDInput) (*batchJobOutput, error) {
		job, err := d.Svc.GetRetryStatus(ctx, in.JobID)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get retry job status")
		}
		return toBatchJobOutput(job), nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "cancelDeliveryRetryJob",
		Method:        http.MethodPost,
		Path:          "/v1/namespaces/{namespace}/retry-jobs/{job_id}:cancel",
		Summary:       "Cancel a pending or in-progress batch retry job",
		Description:   "Requests cancellation of a batch retry job. Deliveries already retried are not rolled back.",
		Errors:        []int{404, 409},
		Tags:          []string{"Deliveries"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *jobIDInput) (*emptyOutput, error) {
		if err := d.Svc.CancelRetry(ctx, in.JobID); err != nil {
			return nil, mapError(ctx, err, "failed to cancel retry job")
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getDeliveryGlobal",
		Method:      http.MethodGet,
		Path:        "/v1/deliveries/{delivery_id}",
		Summary:     "Get a delivery by id (any namespace)",
		Description: "Namespace-agnostic lookup by delivery id, for callers that only have the id (e.g. a webhook-signature verification failure report) and don't know which namespace it belongs to.",
		Errors:      []int{404},
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *deliveryIDOnlyInput) (*deliveryOutput, error) {
		dl, err := d.Svc.GetDeliveryStatus(ctx, in.DeliveryID, "")
		if err != nil {
			return nil, mapError(ctx, err, "failed to get delivery")
		}
		return toDeliveryOutput(dl), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getDeliveryAttemptsGlobal",
		Method:      http.MethodGet,
		Path:        "/v1/deliveries/{delivery_id}/attempts",
		Summary:     "Get a delivery's per-attempt history (any namespace)",
		Description: "Namespace-agnostic variant of getDeliveryAttempts.",
		Errors:      []int{404},
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *deliveryIDOnlyInput) (*attemptsOutput, error) {
		attempts, err := d.Svc.GetDeliveryAttempts(ctx, in.DeliveryID)
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
		Summary:     "Retry a single delivery (any namespace)",
		Description: "Namespace-agnostic variant of retryDelivery.",
		Errors:      []int{404},
		Tags:        []string{"Deliveries"},
	}, func(ctx context.Context, in *deliveryIDOnlyInput) (*retryOutput, error) {
		ids, count, err := d.Svc.RetryDelivery(ctx, "", in.DeliveryID, "", false)
		if err != nil {
			return nil, mapError(ctx, err, "failed to retry delivery")
		}
		out := &retryOutput{}
		out.Body.Count = count
		out.Body.DeliveryIDs = ids
		return out, nil
	})
}
