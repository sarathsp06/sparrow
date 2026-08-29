package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// --- Event types (definitions) ---

// eventTypeBody defines or updates an event type's shape. Name is required
// on register; on update it identifies which fields changed (empty/nil
// fields are left untouched — see registerEventTypeInput/patchEventTypeInput).
type eventTypeBody struct {
	Name        string            `json:"name,omitempty" doc:"Unique event type name, e.g. order.created. Immutable after creation."`
	Description string            `json:"description,omitempty" doc:"Human-readable summary of what this event represents."`
	JSONSchema  map[string]any    `json:"event_schema,omitempty" doc:"Optional JSON Schema for the payload. Validation is soft: non-conforming payloads are still accepted, with warnings returned from the push call."`
	Metadata    map[string]string `json:"metadata,omitempty" doc:"Arbitrary key/value metadata for your own tooling."`
	Active      *bool             `json:"active,omitempty" doc:"Whether events of this type can be pushed. Defaults to true."`
}

type registerEventTypeInput struct {
	Body eventTypeBody
}

type eventTypeNameInput struct {
	Name string `path:"name"`
}

type patchEventTypeInput struct {
	Name string `path:"name"`
	Body eventTypeBody
}

type eventTypeItem struct {
	Name          string            `json:"name" doc:"Event type name, e.g. order.created."`
	Description   string            `json:"description,omitempty" doc:"Human-readable summary of what this event represents."`
	JSONSchema    map[string]any    `json:"event_schema,omitempty" doc:"JSON Schema payloads are softly validated against, if one is registered."`
	SamplePayload map[string]any    `json:"sample_payload,omitempty" doc:"Example payload derived from the schema, used to preview subscription transform templates."`
	Metadata      map[string]string `json:"metadata,omitempty" doc:"Arbitrary key/value metadata."`
	Active        bool              `json:"active" doc:"Whether events of this type can currently be pushed."`
	CreatedAt     string            `json:"created_at" doc:"Creation timestamp, RFC3339."`
	UpdatedAt     string            `json:"updated_at" doc:"Last-modified timestamp, RFC3339."`
}

type eventTypeOutput struct {
	Body eventTypeItem
}

func toEventTypeItem(e *store.EventRegistration) eventTypeItem {
	return eventTypeItem{
		Name:          e.Name,
		Description:   e.Description,
		JSONSchema:    e.Schema,
		SamplePayload: e.SamplePayload,
		Metadata:      e.Metadata,
		Active:        e.Active,
		CreatedAt:     e.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:     e.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func toEventTypeOutput(e *store.EventRegistration) *eventTypeOutput {
	return &eventTypeOutput{Body: toEventTypeItem(e)}
}

type listEventTypesInput struct {
	ActiveOnly bool  `query:"active_only" default:"false" doc:"Only return active event types."`
	Limit      int32 `query:"limit" default:"50" doc:"Maximum items to return."`
	Offset     int32 `query:"offset" default:"0" doc:"Number of items to skip, for pagination."`
}

type listEventTypesOutput struct {
	Body struct {
		Items      []eventTypeItem  `json:"items"`
		Pagination PaginationOutput `json:"pagination"`
	}
}

// --- Event occurrences (pushed instances) ---

type pushEventBody struct {
	Payload        map[string]any    `json:"payload" required:"true" doc:"Event payload, validated (softly) against the event type's JSON Schema if one is registered."`
	TTLSeconds     int64             `json:"ttl_seconds,omitempty" doc:"Optional time-to-live in seconds; deliveries are no longer attempted after this expires."`
	Metadata       map[string]string `json:"metadata,omitempty" doc:"Arbitrary key/value metadata stored with the occurrence."`
	Labels         map[string]string `json:"labels,omitempty" doc:"Key/value labels a subscription's label filters can match against to select which subscribers receive this occurrence."`
	IdempotencyKey *string           `json:"idempotency_key,omitempty" doc:"Optional client-supplied key; re-pushing with the same key returns the original event instead of creating a duplicate."`
}

type pushEventInput struct {
	Namespace string `path:"namespace" doc:"Tenant namespace to record the occurrence in."`
	Event     string `query:"event" required:"true" doc:"Name of a registered event type."`
	Body      pushEventBody
}

type pushEventOutput struct {
	Body struct {
		EventID     string   `json:"event_id" doc:"Id (UUID) of the created event occurrence."`
		SchemaValid bool     `json:"schema_valid" doc:"Whether the payload validated against the event type's JSON Schema. False does not mean the push was rejected — see warnings."`
		Warnings    []string `json:"warnings,omitempty" doc:"Non-fatal schema validation warnings. The event is stored and delivered regardless."`
	}
}

// eventIDOnlyInput is used for occurrence lookups that are namespace-agnostic
// in the domain layer (event IDs are globally unique UUIDs).
type eventIDOnlyInput struct {
	EventID string `path:"event_id"`
}

type eventOccurrenceItem struct {
	EventID              string            `json:"event_id" doc:"Event occurrence id (UUID)."`
	Namespace            string            `json:"namespace" doc:"Tenant namespace this occurrence was pushed into."`
	Event                string            `json:"event" doc:"Event type name."`
	Payload              map[string]any    `json:"payload" doc:"The pushed payload."`
	Metadata             map[string]string `json:"metadata,omitempty" doc:"Arbitrary key/value metadata stored with the occurrence."`
	Labels               map[string]string `json:"labels,omitempty" doc:"Key/value labels used to match against subscription label filters."`
	SchemaValid          bool              `json:"schema_valid" doc:"Whether the payload validated against the event type's JSON Schema at push time."`
	WebhookCount         int32             `json:"webhook_count" doc:"Number of webhooks whose subscriptions matched this occurrence."`
	SuccessfulDeliveries int32             `json:"successful_deliveries" doc:"Deliveries that succeeded."`
	FailedDeliveries     int32             `json:"failed_deliveries" doc:"Deliveries that failed (exhausted retries or non-retryable error)."`
	PendingDeliveries    int32             `json:"pending_deliveries" doc:"Deliveries still pending or retrying."`
	CreatedAt            string            `json:"created_at" doc:"When this occurrence was pushed, RFC3339."`
}

type eventOccurrenceOutput struct {
	Body eventOccurrenceItem
}

type listEventOccurrencesInput struct {
	Namespace     string `path:"namespace" doc:"Tenant namespace to list occurrences in."`
	Event         string `query:"event,omitempty" doc:"Filter to occurrences of this event type name."`
	PrepareRepush bool   `query:"prepare_repush" default:"false" doc:"If true, snapshot the matching occurrences into a repush_id you can pass to the batch re-push endpoint."`
	Limit         int32  `query:"limit" default:"50" doc:"Maximum items to return."`
	Offset        int32  `query:"offset" default:"0" doc:"Number of items to skip, for pagination."`
}

type listEventOccurrencesOutput struct {
	Body struct {
		Items      []eventOccurrenceItem `json:"items"`
		Pagination PaginationOutput      `json:"pagination"`
		RepushID   string                `json:"repush_id,omitempty" doc:"Snapshot id for the batch re-push endpoint, present when prepare_repush was set."`
	}
}

type repushEventOutput struct {
	Body struct {
		EventID  string   `json:"event_id" doc:"Id of the re-pushed event occurrence (same id, replayed)."`
		Warnings []string `json:"warnings,omitempty" doc:"Non-fatal schema validation warnings from re-validating the stored payload."`
	}
}

type repushBatchInput struct {
	Namespace string `path:"namespace"`
	Body      struct {
		RepushID string `json:"repush_id" required:"true" doc:"Snapshot id from an earlier prepare_repush=true or prepare_retry=true list call."`
	}
}

type jobIDInput struct {
	Namespace string `path:"namespace"`
	JobID     string `path:"job_id"`
}

type batchJobOutput struct {
	Body struct {
		ID        string `json:"id" doc:"Batch job id (UUID) — the repush_id/retry_id used elsewhere."`
		Namespace string `json:"namespace" doc:"Tenant namespace the job runs in."`
		JobType   string `json:"job_type" enum:"event_repush,delivery_retry" doc:"Which batch operation this job performs."`
		Status    string `json:"status" enum:"pending,processing,completed,failed,cancelled" doc:"Current job status."`
		Total     int    `json:"total" doc:"Total items in the batch snapshot."`
		Processed int    `json:"processed" doc:"Items processed so far (successes + failures)."`
		Failed    int    `json:"failed" doc:"Items that failed to process."`
		CreatedAt string `json:"created_at" doc:"Creation timestamp, RFC3339."`
		ExpiresAt string `json:"expires_at" doc:"When the job's underlying snapshot expires, RFC3339."`
	}
}

func toBatchJobOutput(b *store.BatchJob) *batchJobOutput {
	out := &batchJobOutput{}
	out.Body.ID = b.ID.String()
	out.Body.Namespace = b.Namespace
	out.Body.JobType = string(b.JobType)
	out.Body.Status = string(b.Status)
	out.Body.Total = b.Total
	out.Body.Processed = b.Processed
	out.Body.Failed = b.Failed
	out.Body.CreatedAt = b.CreatedAt.Format(time.RFC3339Nano)
	out.Body.ExpiresAt = b.ExpiresAt.Format(time.RFC3339Nano)
	return out
}

func registerEventRoutes(api huma.API, d *Deps) {
	huma.Register(api, huma.Operation{
		OperationID:   "registerEventType",
		Method:        http.MethodPost,
		Path:          "/v1/event-types",
		Summary:       "Register an event type definition",
		Description:   "Defines a named event type, optionally with a JSON Schema. Subscriptions reference event types by name; pushing an event requires that its type already be registered.",
		Errors:        []int{400, 409},
		Tags:          []string{"Event Types"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *registerEventTypeInput) (*eventTypeOutput, error) {
		active := true
		if in.Body.Active != nil {
			active = *in.Body.Active
		}
		if _, _, err := d.Svc.RegisterEvent(ctx, in.Body.Name, in.Body.Description, in.Body.JSONSchema, in.Body.Metadata, active); err != nil {
			return nil, mapError(ctx, err, "failed to register event type")
		}
		e, err := d.Svc.GetEvent(ctx, in.Body.Name)
		if err != nil {
			return nil, mapError(ctx, err, "failed to reload event type")
		}
		return toEventTypeOutput(e), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listEventTypes",
		Method:      http.MethodGet,
		Path:        "/v1/event-types",
		Summary:     "List event type definitions",
		Description: "Lists registered event types, optionally filtered to only active ones.",
		Tags:        []string{"Event Types"},
	}, func(ctx context.Context, in *listEventTypesInput) (*listEventTypesOutput, error) {
		limit, offset := in.Limit, in.Offset
		activeOnly := in.ActiveOnly
		regs, total, err := d.Svc.ListEvents(ctx, activeOnly, limit, offset)
		if err != nil {
			return nil, mapError(ctx, err, "failed to list event types")
		}
		out := &listEventTypesOutput{}
		for _, e := range regs {
			out.Body.Items = append(out.Body.Items, toEventTypeItem(e))
		}
		out.Body.Pagination = newPagination(limit, offset, total)
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getEventType",
		Method:      http.MethodGet,
		Path:        "/v1/event-types/{name}",
		Summary:     "Get an event type by name",
		Description: "Fetches one event type's schema, sample payload, and metadata.",
		Errors:      []int{404},
		Tags:        []string{"Event Types"},
	}, func(ctx context.Context, in *eventTypeNameInput) (*eventTypeOutput, error) {
		e, err := d.Svc.GetEvent(ctx, in.Name)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get event type")
		}
		return toEventTypeOutput(e), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateEventType",
		Method:      http.MethodPatch,
		Path:        "/v1/event-types/{name}",
		Summary:     "Update an event type definition",
		Description: "Merge-patches an event type: only fields present in the request body are changed.",
		Errors:      []int{400, 404},
		Tags:        []string{"Event Types"},
	}, func(ctx context.Context, in *patchEventTypeInput) (*eventTypeOutput, error) {
		existing, err := d.Svc.GetEvent(ctx, in.Name)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get event type")
		}
		desc := existing.Description
		if in.Body.Description != "" {
			desc = in.Body.Description
		}
		schema := existing.Schema
		if in.Body.JSONSchema != nil {
			schema = in.Body.JSONSchema
		}
		meta := map[string]string(existing.Metadata)
		if in.Body.Metadata != nil {
			meta = in.Body.Metadata
		}
		active := existing.Active
		if in.Body.Active != nil {
			active = *in.Body.Active
		}
		if err := d.Svc.UpdateEvent(ctx, in.Name, desc, schema, meta, active); err != nil {
			return nil, mapError(ctx, err, "failed to update event type")
		}
		updated, err := d.Svc.GetEvent(ctx, in.Name)
		if err != nil {
			return nil, mapError(ctx, err, "failed to reload event type")
		}
		return toEventTypeOutput(updated), nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "deleteEventType",
		Method:        http.MethodDelete,
		Path:          "/v1/event-types/{name}",
		Summary:       "Delete an event type definition",
		Description:   "Permanently deletes an event type definition. Existing pushed occurrences of this type are not deleted.",
		Errors:        []int{404},
		Tags:          []string{"Event Types"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *eventTypeNameInput) (*emptyOutput, error) {
		if err := d.Svc.DeleteEvent(ctx, in.Name); err != nil {
			return nil, mapError(ctx, err, "failed to delete event type")
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "pushEvent",
		Method:        http.MethodPost,
		Path:          "/v1/namespaces/{namespace}/events",
		Summary:       "Push an event occurrence for asynchronous delivery",
		Description:   "Records one occurrence of a registered event type and asynchronously fans it out to every subscription whose event name and label filters match. Returns immediately; delivery happens in the background — check Deliveries for outcomes.",
		Errors:        []int{400, 404},
		Tags:          []string{"Events"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *pushEventInput) (*pushEventOutput, error) {
		eventID, schemaValid, warnings, err := d.Svc.PushEvent(ctx, in.Namespace, in.Event, in.Body.Payload, in.Body.TTLSeconds, in.Body.Metadata, in.Body.Labels, in.Body.IdempotencyKey)
		if err != nil {
			return nil, mapError(ctx, err, "failed to push event")
		}
		out := &pushEventOutput{}
		out.Body.EventID = eventID
		out.Body.SchemaValid = schemaValid
		out.Body.Warnings = warnings
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listEventOccurrences",
		Method:      http.MethodGet,
		Path:        "/v1/namespaces/{namespace}/events",
		Summary:     "List pushed event occurrences",
		Description: "Lists pushed event occurrences in a namespace, with delivery outcome counts per occurrence. Set prepare_repush to snapshot the filtered set for the batch re-push endpoint.",
		Tags:        []string{"Events"},
	}, func(ctx context.Context, in *listEventOccurrencesInput) (*listEventOccurrencesOutput, error) {
		limit, offset := in.Limit, in.Offset
		filter := store.EventReportFilter{
			Namespace:     in.Namespace,
			Limit:         int(limit),
			Offset:        int(offset),
			PrepareRepush: in.PrepareRepush,
		}
		if in.Event != "" {
			filter.EventName = &in.Event
		}
		reports, total, repushID, err := d.Svc.ListEventReports(ctx, filter)
		if err != nil {
			return nil, mapError(ctx, err, "failed to list event occurrences")
		}
		out := &listEventOccurrencesOutput{}
		for _, r := range reports {
			var o eventOccurrenceOutput
			o.Body.EventID = r.ID.String()
			o.Body.Namespace = r.Namespace
			o.Body.Event = r.Event
			o.Body.Payload = r.Payload
			o.Body.Metadata = r.Metadata
			o.Body.Labels = r.Labels
			o.Body.SchemaValid = r.SchemaValid
			o.Body.WebhookCount = r.WebhookCount
			o.Body.SuccessfulDeliveries = r.SuccessfulDeliveries
			o.Body.FailedDeliveries = r.FailedDeliveries
			o.Body.PendingDeliveries = r.PendingDeliveries
			o.Body.CreatedAt = r.CreatedAt.Format(time.RFC3339Nano)
			out.Body.Items = append(out.Body.Items, o.Body)
		}
		out.Body.Pagination = newPagination(limit, offset, total)
		out.Body.RepushID = repushID
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getEventOccurrence",
		Method:      http.MethodGet,
		Path:        "/v1/events/{event_id}",
		Summary:     "Get a pushed event occurrence by id",
		Description: "Fetches one event occurrence's payload, labels, and delivery outcome counts. Event ids are globally unique, so no namespace is required.",
		Errors:      []int{400, 404},
		Tags:        []string{"Events"},
	}, func(ctx context.Context, in *eventIDOnlyInput) (*eventOccurrenceOutput, error) {
		if _, err := uuid.Parse(in.EventID); err != nil {
			return nil, huma.Error400BadRequest("event_id must be a valid UUID")
		}
		rec, webhookCount, successCount, failedCount, pendingCount, err := d.Svc.GetEventRecord(ctx, in.EventID)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get event occurrence")
		}
		out := &eventOccurrenceOutput{}
		out.Body.EventID = rec.ID.String()
		out.Body.Namespace = rec.Namespace
		out.Body.Event = rec.Event
		out.Body.Payload = rec.Payload
		out.Body.Metadata = rec.Metadata
		out.Body.Labels = rec.Labels
		out.Body.SchemaValid = rec.SchemaValid
		out.Body.WebhookCount = webhookCount
		out.Body.SuccessfulDeliveries = successCount
		out.Body.FailedDeliveries = failedCount
		out.Body.PendingDeliveries = pendingCount
		out.Body.CreatedAt = rec.CreatedAt.Format(time.RFC3339Nano)
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "repushEvent",
		Method:      http.MethodPost,
		Path:        "/v1/events/{event_id}:repush",
		Summary:     "Replay a single pushed event occurrence",
		Description: "Re-runs fan-out for one already-pushed event occurrence: matches subscriptions again and creates new deliveries. Useful after fixing a subscription or webhook that was misconfigured.",
		Errors:      []int{404},
		Tags:        []string{"Events"},
	}, func(ctx context.Context, in *eventIDOnlyInput) (*repushEventOutput, error) {
		eventID, warnings, err := d.Svc.RePushEvent(ctx, in.EventID)
		if err != nil {
			return nil, mapError(ctx, err, "failed to re-push event")
		}
		out := &repushEventOutput{}
		out.Body.EventID = eventID
		out.Body.Warnings = warnings
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "startEventRepushJob",
		Method:        http.MethodPost,
		Path:          "/v1/namespaces/{namespace}/events:rePush",
		Summary:       "Start a batch re-push job from a prepared snapshot",
		Description:   "Starts an async job that replays every event occurrence captured by an earlier prepare_repush=true list call. Poll the returned job with getEventRepushJob.",
		Errors:        []int{400, 404},
		Tags:          []string{"Events"},
		DefaultStatus: http.StatusAccepted,
	}, func(ctx context.Context, in *repushBatchInput) (*batchJobOutput, error) {
		if err := d.Svc.RePushEvents(ctx, in.Body.RepushID); err != nil {
			return nil, mapError(ctx, err, "failed to start re-push job")
		}
		job, err := d.Svc.GetRepushStatus(ctx, in.Body.RepushID)
		if err != nil {
			return nil, mapError(ctx, err, "failed to load re-push job status")
		}
		return toBatchJobOutput(job), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getEventRepushJob",
		Method:      http.MethodGet,
		Path:        "/v1/namespaces/{namespace}/repush-jobs/{job_id}",
		Summary:     "Get batch re-push job progress",
		Description: "Returns a batch re-push job's status and processed/failed/total counts.",
		Errors:      []int{404},
		Tags:        []string{"Events"},
	}, func(ctx context.Context, in *jobIDInput) (*batchJobOutput, error) {
		job, err := d.Svc.GetRepushStatus(ctx, in.JobID)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get re-push job status")
		}
		return toBatchJobOutput(job), nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "cancelEventRepushJob",
		Method:        http.MethodPost,
		Path:          "/v1/namespaces/{namespace}/repush-jobs/{job_id}:cancel",
		Summary:       "Cancel a pending or in-progress batch re-push job",
		Description:   "Requests cancellation of a batch re-push job. Occurrences already re-pushed are not rolled back.",
		Errors:        []int{404, 409},
		Tags:          []string{"Events"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *jobIDInput) (*emptyOutput, error) {
		if err := d.Svc.CancelRepush(ctx, in.JobID); err != nil {
			return nil, mapError(ctx, err, "failed to cancel re-push job")
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})
}
