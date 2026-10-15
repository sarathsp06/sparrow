package rest

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/middleware"
	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// --- Event types (definitions) ---

// eventTypeBody defines or updates an event type's shape. Name is required
// on register; on update it identifies which fields changed (empty/nil
// fields are left untouched — see registerEventTypeInput/patchEventTypeInput).
type eventTypeBody struct {
	Name        string            `json:"name,omitempty" doc:"Unique event type name, e.g. order.created. Required on register, ignored on update (the path names the type). At most 255 characters; the sparrow. prefix is reserved. Immutable after creation."`
	Description string            `json:"description,omitempty" doc:"Human-readable summary of what this event represents. On update, an empty string leaves it unchanged."`
	JSONSchema  map[string]any    `json:"event_schema,omitempty" doc:"Optional JSON Schema for the payload. Validation is soft: non-conforming payloads are still accepted, with warnings returned from the push call. On update, sending a different schema creates a new version."`
	Metadata    map[string]string `json:"metadata,omitempty" doc:"Arbitrary key/value metadata for your own tooling. On update, replaces the whole map."`
	Active      *bool             `json:"active,omitempty" doc:"Whether events of this type can be pushed; a push to an inactive type fails with 409. Defaults to true on register; unchanged on update when omitted."`
}

type registerEventTypeInput struct {
	Body eventTypeBody
}

type eventTypeNameInput struct {
	Name string `path:"name" doc:"Event type name, e.g. order.created."`
}

type patchEventTypeInput struct {
	Name          string `path:"name" doc:"Event type name, e.g. order.created."`
	AllowBreaking bool   `query:"allow_breaking" default:"false" doc:"Apply a breaking schema change even though subscriptions receive this event type. Without it, a breaking change to a subscribed type is refused with 409 and the reasons."`
	Body          eventTypeBody
}

type EventTypeItem struct {
	Name          string            `json:"name" doc:"Event type name, e.g. order.created."`
	Description   string            `json:"description,omitempty" doc:"Human-readable summary of what this event represents."`
	JSONSchema    map[string]any    `json:"event_schema,omitempty" doc:"JSON Schema payloads are softly validated against, if one is registered."`
	SamplePayload map[string]any    `json:"sample_payload,omitempty" doc:"Example payload derived from the schema, used to preview subscription transform templates."`
	Metadata      map[string]string `json:"metadata,omitempty" doc:"Arbitrary key/value metadata."`
	Active        bool              `json:"active" doc:"Whether events of this type can currently be pushed."`
	Version       int               `json:"version" doc:"Current schema version. Starts at 1 and increases only when the schema changes; every version is kept (see listEventTypeVersions)."`
	CreatedAt     string            `json:"created_at" format:"date-time" doc:"Creation timestamp, RFC3339."`
	UpdatedAt     string            `json:"updated_at" format:"date-time" doc:"Last-modified timestamp, RFC3339."`
}

type eventTypeOutput struct {
	Body EventTypeItem
}

// eventTypeChange reports what a write did to an event type.
type eventTypeChange struct {
	Action                string                   `json:"action" enum:"created,new_version,updated,unchanged" doc:"created: new event type at version 1. new_version: the schema changed, so a new version was created and the previous one kept. updated: the current version changed in place (description, metadata, active, or a first schema added to a schema-less version). unchanged: nothing was written."`
	Version               int                      `json:"version" doc:"Version after the write."`
	PreviousVersion       int                      `json:"previous_version,omitempty" doc:"Version before the write; absent when the event type was created."`
	Changes               []string                 `json:"changes,omitempty" doc:"Fields that changed: schema, schema_defined (a first schema was added in place), description, metadata, active."`
	ActiveChange          string                   `json:"active_change,omitempty" enum:"deactivates,reactivates" doc:"Set when the write flips the active flag."`
	Compatibility         *schemaCompatibilityItem `json:"compatibility,omitempty" doc:"For new_version: whether the schema change could break a subscription's payload transformation."`
	AffectedSubscriptions int                      `json:"affected_subscriptions,omitempty" doc:"For new_version: subscriptions that receive this event type, by name or catch-all."`
}

type schemaCompatibilityItem struct {
	Result  string   `json:"result" enum:"compatible,breaking" doc:"breaking when the change could break a payload transformation: a required property removed or made optional, a type widened, the schema removed, or a change under oneOf/anyOf/allOf/$ref/patternProperties that cannot be checked."`
	Reasons []string `json:"reasons,omitempty" doc:"Each breaking change with the path it affects."`
}

func toEventTypeChange(r *webhooks.EventTypeSaveResult) eventTypeChange {
	c := eventTypeChange{
		Action:          string(r.Action),
		Version:         r.Version,
		PreviousVersion: r.PreviousVersion,
		Changes:         r.Changes,
		ActiveChange:    r.ActiveChange,
	}
	if r.Compatibility != nil {
		c.Compatibility = &schemaCompatibilityItem{Result: r.Compatibility.Result(), Reasons: r.Compatibility.Reasons}
		c.AffectedSubscriptions = len(r.AffectedSubscriptions)
	}
	return c
}

type updateEventTypeOutput struct {
	Body struct {
		EventTypeItem
		Change eventTypeChange `json:"change" doc:"What this update did."`
	}
}

type eventTypeVersionItem struct {
	Name            string         `json:"name" doc:"Event type name."`
	Version         int            `json:"version" doc:"Version number."`
	Description     string         `json:"description,omitempty" doc:"Description as it was when this version was created."`
	JSONSchema      map[string]any `json:"event_schema,omitempty" doc:"JSON Schema of this version. Absent for a schema-less version."`
	SamplePayload   map[string]any `json:"sample_payload,omitempty" doc:"Example payload generated from this version's schema."`
	SchemaDefinedAt *string        `json:"schema_defined_at,omitempty" format:"date-time" doc:"When a first schema was added to this version in place, RFC3339. Events pushed before this time were accepted while the version had no schema."`
	CreatedAt       string         `json:"created_at" format:"date-time" doc:"When this version was created, RFC3339."`
}

func toEventTypeVersionItem(v *store.EventRegistrationVersion) eventTypeVersionItem {
	item := eventTypeVersionItem{
		Name:          v.Name,
		Version:       v.Version,
		Description:   v.Description,
		JSONSchema:    v.Schema,
		SamplePayload: v.SamplePayload,
		CreatedAt:     v.CreatedAt.Format(time.RFC3339Nano),
	}
	if v.SchemaDefinedAt != nil {
		t := v.SchemaDefinedAt.Format(time.RFC3339Nano)
		item.SchemaDefinedAt = &t
	}
	return item
}

type eventTypeVersionInput struct {
	Name    string `path:"name" doc:"Event type name, e.g. order.created."`
	Version int    `path:"version" minimum:"1" doc:"Version number."`
}

type eventTypeVersionOutput struct {
	Body eventTypeVersionItem
}

type listEventTypeVersionsOutput struct {
	Body struct {
		Items []eventTypeVersionItem `json:"items" doc:"Every version of the event type, newest first."`
	}
}

type validateEventPayloadInput struct {
	Name string `path:"name" doc:"Event type name, e.g. order.created."`
	Body struct {
		Payload map[string]any `json:"payload" required:"true" doc:"Payload to validate against the event type's registered JSON Schema."`
	}
}

type validateEventPayloadOutput struct {
	Body struct {
		Valid    bool     `json:"valid" doc:"Whether the payload matches the event type's JSON Schema. Always true when the event type has no schema registered."`
		Warnings []string `json:"warnings,omitempty" doc:"Per-field validation errors, present when valid is false."`
	}
}

func toEventTypeItem(e *store.EventRegistration) EventTypeItem {
	return EventTypeItem{
		Name:          e.Name,
		Description:   e.Description,
		JSONSchema:    e.Schema,
		SamplePayload: e.SamplePayload,
		Metadata:      e.Metadata,
		Active:        e.Active,
		Version:       e.Version,
		CreatedAt:     e.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:     e.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func toEventTypeOutput(e *store.EventRegistration) *eventTypeOutput {
	return &eventTypeOutput{Body: toEventTypeItem(e)}
}

type listEventTypesInput struct {
	ActiveOnly bool   `query:"active_only" default:"false" doc:"true returns only active event types; false returns all."`
	Consumer   string `query:"consumer" doc:"Return the event types this consumer can subscribe to: Sparrow's own sparrow.* events for the _sparrow consumer, tenant event types for any other consumer or when omitted."`
	Limit      int32  `query:"limit" default:"50" minimum:"1" maximum:"1000" doc:"Maximum items to return, 1 to 1000."`
	Offset     int32  `query:"offset" default:"0" doc:"Number of items to skip, for pagination."`
}

type listEventTypesOutput struct {
	Body struct {
		Items      []EventTypeItem  `json:"items" doc:"Event types, sorted by name."`
		Pagination PaginationOutput `json:"pagination" doc:"Offset pagination with a total count."`
	}
}

// --- Event occurrences (pushed instances) ---

type pushEventBody struct {
	Payload        map[string]any    `json:"payload" required:"true" doc:"Event payload, a JSON object. Validated softly against the event type's JSON Schema if one is registered. The whole request body is limited by SPARROW_MAX_BODY_BYTES (default 5 MiB)."`
	TTLSeconds     int64             `json:"ttl_seconds,omitempty" doc:"Time-to-live in seconds, counted from fan-out. A delivery attempt that would start after it expires is not made and the delivery fails as expired. 0 or omitted means no expiry. Manual retries ignore it."`
	Metadata       map[string]string `json:"metadata,omitempty" doc:"Arbitrary key/value metadata stored with the occurrence."`
	Labels         map[string]string `json:"labels,omitempty" doc:"Key/value labels a subscription's label_filters match against to decide which subscriptions receive this occurrence. At most 20 pairs; keys up to 64 characters of letters, digits, '.', '_' and '-'; values up to 256 characters."`
	IdempotencyKey *string           `json:"idempotency_key,omitempty" doc:"Client-supplied deduplication key, unique per consumer. A push whose key matches an earlier event in the same consumer still returns 201, with duplicate: true and the original event_id, and creates no event or deliveries. The key is kept as long as the event (see SPARROW_EVENT_RETENTION_DAYS)."`
}

type pushEventInput struct {
	Consumer string `path:"consumer" doc:"Consumer to record the occurrence in. Created implicitly on first use."`
	Event    string `query:"event" required:"true" doc:"Name of a registered, active event type (at most 255 characters). An unregistered name is 404 unless SPARROW_AUTO_REGISTER_EVENTS=true; sparrow.* names are reserved (400)."`
	Body     pushEventBody
}

type pushEventOutput struct {
	Body struct {
		EventID     string   `json:"event_id" doc:"Id (UUID) of the created event occurrence. When duplicate is true, this is the id of the PREVIOUSLY created event, not a new one."`
		Duplicate   bool     `json:"duplicate" doc:"True when idempotency_key matched an existing event occurrence in this consumer; no new event or deliveries were created and the status is still 201."`
		SchemaValid bool     `json:"schema_valid" doc:"Whether the payload validated against the event type's JSON Schema (true when it has none). False does not mean the push was rejected; see warnings. For a duplicate, the original event's result."`
		Warnings    []string `json:"warnings,omitempty" doc:"Non-fatal schema validation warnings. The event is stored and delivered regardless. Absent for a duplicate."`
	}
}

// eventIDOnlyInput is used for occurrence lookups that are consumer-agnostic
// in the domain layer (event IDs are globally unique UUIDs).
type eventIDOnlyInput struct {
	EventID string `path:"event_id" doc:"Event occurrence id (UUID)."`
}

type eventOccurrenceItem struct {
	EventID              string            `json:"event_id" doc:"Event occurrence id (UUID)."`
	Consumer             string            `json:"consumer" doc:"Tenant consumer this occurrence was pushed into."`
	Event                string            `json:"event" doc:"Event type name."`
	Payload              map[string]any    `json:"payload" doc:"The pushed payload."`
	Metadata             map[string]string `json:"metadata,omitempty" doc:"Arbitrary key/value metadata stored with the occurrence."`
	Labels               map[string]string `json:"labels,omitempty" doc:"Key/value labels used to match against subscription label filters."`
	SchemaValid          bool              `json:"schema_valid" doc:"Whether the payload validated against the event type's JSON Schema at push time."`
	EventVersion         int               `json:"event_version" doc:"Event type version this occurrence was accepted under."`
	WebhookCount         int32             `json:"webhook_count" doc:"Number of distinct webhooks that got a delivery for this occurrence."`
	SuccessfulDeliveries int32             `json:"successful_deliveries" doc:"Successful delivery attempts for this occurrence."`
	FailedDeliveries     int32             `json:"failed_deliveries" doc:"Failed delivery attempts for this occurrence. Counts attempts, not deliveries: a delivery that failed twice and then succeeded adds 2 here and 1 to successful_deliveries."`
	PendingDeliveries    int32             `json:"pending_deliveries" doc:"Deliveries not finished yet: pending, sending or retrying."`
	CreatedAt            string            `json:"created_at" format:"date-time" doc:"When this occurrence was pushed, RFC3339."`
}

type eventOccurrenceOutput struct {
	Body eventOccurrenceItem
}

// EventOccurrenceListParams are the query filters shared by the
// consumer-scoped and global event occurrence list routes.
type EventOccurrenceListParams struct {
	Event         string `query:"event,omitempty" doc:"Filter to occurrences of this event type name."`
	SchemaValid   string `query:"schema_valid,omitempty" doc:"Filter by schema validation result: valid or invalid. Other values are 400."`
	Labels        string `query:"labels,omitempty" doc:"Filter by labels as comma-separated key=value pairs, e.g. region=eu,tier=gold. Matches occurrences whose labels contain every pair."`
	CreatedAfter  string `query:"created_after,omitempty" doc:"Only occurrences created at or after this time: YYYY-MM-DD (start of that day, UTC) or an RFC3339 timestamp."`
	CreatedBefore string `query:"created_before,omitempty" doc:"Only occurrences created at or before this time: YYYY-MM-DD (through the end of that day, UTC) or an RFC3339 timestamp."`
	PrepareRepush bool   `query:"prepare_repush" default:"false" doc:"Side effect: when true, also snapshots the ids of every occurrence matching the filters (not just this page; at most 10,000, more is an error) into a batch job and returns its id as repush_id. Pass it to POST .../events:rePush within 15 minutes. Each call creates a new snapshot."`
	Limit         int32  `query:"limit" default:"50" minimum:"1" maximum:"1000" doc:"Maximum items to return, 1 to 1000."`
	Cursor        string `query:"cursor,omitempty" doc:"Continue after the page this came from: pagination.next_cursor of the previous response. Stable while new events arrive."`
	Offset        int32  `query:"offset" default:"0" doc:"Removed: page with cursor. Any value other than 0 is rejected with 400, so a client still paging by offset fails loudly instead of re-reading the first page."`
}

type listEventOccurrencesInput struct {
	Consumer string `path:"consumer" doc:"Consumer to list occurrences in."`
	EventOccurrenceListParams
}

type listEventOccurrencesGlobalInput struct {
	Consumer string `query:"consumer,omitempty" doc:"Filter to one consumer; omit to list occurrences across all consumers."`
	EventOccurrenceListParams
}

type listEventOccurrencesOutput struct {
	Body struct {
		Items       []eventOccurrenceItem  `json:"items" doc:"Occurrences, newest first."`
		Pagination  CursorPaginationOutput `json:"pagination" doc:"Cursor pagination; there is no total count."`
		RepushID    string                 `json:"repush_id,omitempty" doc:"Snapshot (batch job) id for POST .../events:rePush, present when prepare_repush was set and something matched. Expires after 15 minutes if not started."`
		RepushTotal int                    `json:"repush_total,omitempty" doc:"Number of occurrences in that snapshot (all matches, not just this page)."`
	}
}

type repushEventOutput struct {
	Body struct {
		EventID  string   `json:"event_id" doc:"Id (UUID) of the new event occurrence created by the replay. The original occurrence is unchanged."`
		Warnings []string `json:"warnings,omitempty" doc:"Non-fatal warnings from validating the stored payload against the event type's current schema."`
	}
}

type repushBatchInput struct {
	Consumer string `path:"consumer" doc:"Consumer the job belongs to. A job snapshotted for another consumer answers 404; a job snapshotted across all consumers is reachable under any consumer, except with a portal token."`
	Body     struct {
		RepushID string `json:"repush_id" required:"true" doc:"repush_id returned by an earlier GET .../events?prepare_repush=true call."`
	}
}

// batchScope is the batch-job scope for a request under
// /v1/consumers/{consumer}: portal (consumer-scoped) callers reach only their
// own consumer's jobs; tenant-wide callers also reach jobs snapshotted across
// all consumers, which the UI starts under any consumer path.
func batchScope(ctx context.Context, consumer string) webhooks.BatchScope {
	return webhooks.BatchScope{Consumer: consumer, ConsumerOnly: middleware.PortalAuthorized(ctx)}
}

type jobIDInput struct {
	Consumer string `path:"consumer" doc:"Consumer the job belongs to. A job snapshotted for another consumer answers 404; a job snapshotted across all consumers is reachable under any consumer, except with a portal token."`
	JobID    string `path:"job_id" doc:"Batch job id (UUID): the repush_id or retry_id from the list call."`
}

type batchJobOutput struct {
	Body struct {
		ID        string `json:"id" doc:"Batch job id (UUID): the repush_id or retry_id from the list call."`
		Consumer  string `json:"consumer" doc:"Consumer the snapshot was taken in; empty for a snapshot taken from a cross-consumer list."`
		JobType   string `json:"job_type" enum:"event_repush,delivery_retry" doc:"Which batch operation this job performs."`
		Status    string `json:"status" enum:"pending,processing,completed,failed,cancelled" doc:"pending: snapshot taken, not started. processing: running. completed: every item was handled (see failed for the ones that could not be). failed: every item failed. cancelled: stopped by a cancel call."`
		Total     int    `json:"total" doc:"Total items in the batch snapshot."`
		Processed int    `json:"processed" doc:"Items handled successfully so far."`
		Failed    int    `json:"failed" doc:"Items that could not be re-pushed or retried (e.g. the event or delivery no longer exists)."`
		CreatedAt string `json:"created_at" format:"date-time" doc:"Creation timestamp, RFC3339."`
		ExpiresAt string `json:"expires_at" format:"date-time" doc:"When the snapshot expires, RFC3339: 15 minutes after it was taken. An expired snapshot can no longer be started."`
	}
}

func toBatchJobOutput(b *store.BatchJob) *batchJobOutput {
	out := &batchJobOutput{}
	out.Body.ID = b.ID.String()
	out.Body.Consumer = b.Consumer
	out.Body.JobType = string(b.JobType)
	out.Body.Status = string(b.Status)
	out.Body.Total = b.Total
	out.Body.Processed = b.Processed
	out.Body.Failed = b.Failed
	out.Body.CreatedAt = b.CreatedAt.Format(time.RFC3339Nano)
	out.Body.ExpiresAt = b.ExpiresAt.Format(time.RFC3339Nano)
	return out
}

// eventRouteService is the service slice event routes consume: event types,
// event publishing, and bulk re-push jobs.
type eventRouteService interface {
	webhooks.EventManager
	webhooks.BatchManager
}

func registerEventRoutes(api huma.API, svc eventRouteService) {
	huma.Register(api, huma.Operation{
		OperationID: "registerEventType",
		Method:      http.MethodPost,
		Path:        "/v1/event-types",
		Summary:     "Register an event type definition",
		Description: "Defines a named event type at version 1, optionally with a JSON Schema; the response includes a sample_payload generated from the schema. Pushing an event requires its type to be registered (unless SPARROW_AUTO_REGISTER_EVENTS=true). Event types are never deleted; deactivate one with active: false.\n\n" +
			"Not idempotent: 409 when the name already exists (use PATCH, or :import, to change it). 400 for a missing, too long or reserved (sparrow.*) name or an invalid schema.\n\nPortal tokens: not allowed.",
		Errors:        []int{400, 409},
		Tags:          []string{"Event Types"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *registerEventTypeInput) (*eventTypeOutput, error) {
		active := true
		if in.Body.Active != nil {
			active = *in.Body.Active
		}
		if _, _, err := svc.RegisterEvent(ctx, in.Body.Name, in.Body.Description, in.Body.JSONSchema, in.Body.Metadata, active); err != nil {
			return nil, mapError(ctx, err, "failed to register event type")
		}
		e, err := svc.GetEvent(ctx, in.Body.Name)
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
		Description: "Lists registered event types sorted by name, optionally only active ones. Sparrow's own sparrow.* events are listed only with consumer=_sparrow. Offset paginated with total_count.\n\nPortal tokens: allowed, as GET /portal/api/event-types; consumer is set to the token's consumer.",
		Tags:        []string{"Event Types"},
	}, func(ctx context.Context, in *listEventTypesInput) (*listEventTypesOutput, error) {
		limit, offset := in.Limit, in.Offset
		activeOnly := in.ActiveOnly
		regs, total, err := svc.ListEvents(ctx, activeOnly, in.Consumer, limit, offset)
		if err != nil {
			return nil, mapError(ctx, err, "failed to list event types")
		}
		out := &listEventTypesOutput{}
		out.Body.Items = make([]EventTypeItem, 0, len(regs))
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
		Description: "Returns one event type's current version: schema, sample payload, and metadata. 404 when the name is not registered.\n\nPortal tokens: allowed, as GET /portal/api/event-types/{name}.",
		Errors:      []int{404},
		Tags:        []string{"Event Types"},
	}, func(ctx context.Context, in *eventTypeNameInput) (*eventTypeOutput, error) {
		e, err := svc.GetEvent(ctx, in.Name)
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
		Description: "Partially updates an event type: only fields present in the request body are changed (metadata, when sent, replaces the whole map). A schema change creates a new version and keeps the previous one; adding a first schema to a schema-less version, and any change to description, metadata or active, updates the current version in place. The response's change field says which happened; repeating the same body is a no-op (change.action unchanged).\n\n" +
			"A breaking schema change (see change.compatibility) to an event type that subscriptions receive is refused with 409 unless allow_breaking=true. 404 when the name is not registered.\n\nPortal tokens: not allowed.",
		Errors: []int{400, 404, 409},
		Tags:   []string{"Event Types"},
	}, func(ctx context.Context, in *patchEventTypeInput) (*updateEventTypeOutput, error) {
		existing, err := svc.GetEvent(ctx, in.Name)
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
		res, err := svc.UpdateEvent(ctx, in.Name, desc, schema, meta, active, in.AllowBreaking)
		if err != nil {
			return nil, mapError(ctx, err, "failed to update event type")
		}
		out := &updateEventTypeOutput{}
		out.Body.EventTypeItem = toEventTypeItem(res.Event)
		out.Body.Change = toEventTypeChange(res)
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listEventTypeVersions",
		Method:      http.MethodGet,
		Path:        "/v1/event-types/{name}/versions",
		Summary:     "List every version of an event type",
		Description: "Returns the version history of an event type, newest first. Not paginated. Event types are never deleted; to retire one, set active to false.\n\nPortal tokens: allowed, as GET /portal/api/event-types/{name}/versions.",
		Errors:      []int{404},
		Tags:        []string{"Event Types"},
	}, func(ctx context.Context, in *eventTypeNameInput) (*listEventTypeVersionsOutput, error) {
		versions, err := svc.ListEventTypeVersions(ctx, in.Name)
		if err != nil {
			return nil, mapError(ctx, err, "failed to list event type versions")
		}
		out := &listEventTypeVersionsOutput{}
		out.Body.Items = make([]eventTypeVersionItem, 0, len(versions))
		for _, v := range versions {
			out.Body.Items = append(out.Body.Items, toEventTypeVersionItem(v))
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getEventTypeVersion",
		Method:      http.MethodGet,
		Path:        "/v1/event-types/{name}/versions/{version}",
		Summary:     "Get one version of an event type",
		Description: "Returns one version's schema, sample payload and description. 404 when the name or version does not exist.\n\nPortal tokens: allowed, as GET /portal/api/event-types/{name}/versions/{version}.",
		Errors:      []int{404},
		Tags:        []string{"Event Types"},
	}, func(ctx context.Context, in *eventTypeVersionInput) (*eventTypeVersionOutput, error) {
		v, err := svc.GetEventTypeVersion(ctx, in.Name, in.Version)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get event type version")
		}
		return &eventTypeVersionOutput{Body: toEventTypeVersionItem(v)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "validateEventPayload",
		Method:      http.MethodPost,
		Path:        "/v1/event-types/{name}:validate",
		Summary:     "Validate a payload against an event type's JSON Schema",
		Description: "Checks a payload against the event type's current JSON Schema without pushing an event or storing anything. pushEvent accepts events regardless of schema validity; this lets callers (e.g. the Push Test Event UI) enforce a hard gate before submitting. An invalid payload is still 200, with valid: false. 404 when the name is not registered.\n\nPortal tokens: not allowed.",
		Errors:      []int{404},
		Tags:        []string{"Event Types"},
	}, func(ctx context.Context, in *validateEventPayloadInput) (*validateEventPayloadOutput, error) {
		e, err := svc.GetEvent(ctx, in.Name)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get event type")
		}
		out := &validateEventPayloadOutput{}
		out.Body.Valid = true
		if len(e.Schema) > 0 {
			if err := webhooks.ValidateJSONSchema(e.Schema, in.Body.Payload); err != nil {
				var schemaErr *webhooks.SchemaValidationError
				if !errors.As(err, &schemaErr) {
					return nil, mapError(ctx, err, "failed to validate payload")
				}
				out.Body.Valid = false
				out.Body.Warnings = schemaErr.Warnings()
			}
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "pushEvent",
		Method:      http.MethodPost,
		Path:        "/v1/consumers/{consumer}/events",
		Summary:     "Push an event occurrence for asynchronous delivery",
		Description: "Records one occurrence of a registered event type and asynchronously fans it out to every subscription whose event name and label filters match. Returns 201 as soon as the event is stored and queued; delivery happens in the background, so check Deliveries for outcomes. A payload that fails the event type's JSON Schema is still accepted (schema_valid: false, with warnings).\n\n" +
			"Idempotency: with idempotency_key, a repeat push with the same key in the same consumer returns 201 with duplicate: true and the original event_id, and creates nothing. Without a key every call creates a new event.\n\n" +
			"Failures: 404 for an unregistered event type (unless SPARROW_AUTO_REGISTER_EVENTS=true), 409 for an inactive one, 400 for a reserved sparrow.* name, a name over 255 characters or invalid labels, 413 for a body over SPARROW_MAX_BODY_BYTES.\n\n" +
			"Portal tokens: not allowed (pushing is the producer's job).",
		Errors:        []int{400, 404, 409, 413},
		Tags:          []string{"Events"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *pushEventInput) (*pushEventOutput, error) {
		eventID, isDuplicate, schemaValid, warnings, err := svc.PushEvent(ctx, in.Consumer, in.Event, in.Body.Payload, in.Body.TTLSeconds, in.Body.Metadata, in.Body.Labels, in.Body.IdempotencyKey)
		if err != nil {
			return nil, mapError(ctx, err, "failed to push event")
		}
		out := &pushEventOutput{}
		out.Body.EventID = eventID
		out.Body.Duplicate = isDuplicate
		out.Body.SchemaValid = schemaValid
		out.Body.Warnings = warnings
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listEventOccurrences",
		Method:      http.MethodGet,
		Path:        "/v1/consumers/{consumer}/events",
		Summary:     "List pushed event occurrences",
		Description: "Lists pushed event occurrences in a consumer, newest first, with delivery outcome counts per occurrence. Cursor paginated: pass pagination.next_cursor as cursor for the next page; there is no total count and offset is rejected.\n\n" +
			"Side effect: prepare_repush=true also snapshots every match (up to 10,000) into a batch job and returns repush_id for POST .../events:rePush.\n\nPortal tokens: allowed, as GET /portal/api/events.",
		Tags: []string{"Events"},
	}, func(ctx context.Context, in *listEventOccurrencesInput) (*listEventOccurrencesOutput, error) {
		return listEventOccurrencesImpl(ctx, svc, in.Consumer, in.EventOccurrenceListParams)
	})

	huma.Register(api, huma.Operation{
		OperationID: "listEventOccurrencesGlobal",
		Method:      http.MethodGet,
		Path:        "/v1/events",
		Summary:     "List pushed event occurrences across all consumers",
		Description: "Lists event occurrences across every consumer, newest first, with the same filters, cursor pagination and prepare_repush side effect as the per-consumer route. Pass consumer to scope to one consumer.\n\nPortal tokens: not allowed.",
		Tags:        []string{"Events"},
	}, func(ctx context.Context, in *listEventOccurrencesGlobalInput) (*listEventOccurrencesOutput, error) {
		return listEventOccurrencesImpl(ctx, svc, in.Consumer, in.EventOccurrenceListParams)
	})

	huma.Register(api, huma.Operation{
		OperationID: "getEventOccurrence",
		Method:      http.MethodGet,
		Path:        "/v1/events/{event_id}",
		Summary:     "Get a pushed event occurrence by id",
		Description: "Returns one event occurrence's payload, labels, and delivery outcome counts. Event ids are globally unique, so no consumer is required. 400 when event_id is not a UUID, 404 when it does not exist (or was removed by retention).\n\nPortal tokens: not allowed.",
		Errors:      []int{400, 404},
		Tags:        []string{"Events"},
	}, func(ctx context.Context, in *eventIDOnlyInput) (*eventOccurrenceOutput, error) {
		if _, err := uuid.Parse(in.EventID); err != nil {
			return nil, huma.Error400BadRequest("event_id must be a valid UUID")
		}
		rec, webhookCount, successCount, failedCount, pendingCount, err := svc.GetEventRecord(ctx, in.EventID)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get event occurrence")
		}
		out := &eventOccurrenceOutput{}
		out.Body.EventID = rec.ID.String()
		out.Body.Consumer = rec.Consumer
		out.Body.Event = rec.Event
		out.Body.Payload = rec.Payload
		out.Body.Metadata = rec.Metadata
		out.Body.Labels = rec.Labels
		out.Body.SchemaValid = rec.SchemaValid
		out.Body.EventVersion = rec.EventVersion
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
		Description: "Pushes a stored event occurrence again: creates a new event occurrence (new event_id) with the same consumer, event type, payload, labels, metadata and TTL, validates the payload against the current schema, and fans it out to the subscriptions that match now. The original occurrence and its deliveries are unchanged. Not idempotent: each call creates another event. To resend one existing delivery instead, retry the delivery.\n\n" +
			"404 when the event does not exist; the same errors as pushEvent when the event type is now inactive or unregistered.\n\nPortal tokens: not allowed.",
		Errors: []int{400, 404, 409},
		Tags:   []string{"Events"},
	}, func(ctx context.Context, in *eventIDOnlyInput) (*repushEventOutput, error) {
		eventID, warnings, err := svc.RePushEvent(ctx, in.EventID)
		if err != nil {
			return nil, mapError(ctx, err, "failed to re-push event")
		}
		out := &repushEventOutput{}
		out.Body.EventID = eventID
		out.Body.Warnings = warnings
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "startEventRepushJob",
		Method:      http.MethodPost,
		Path:        "/v1/consumers/{consumer}/events:rePush",
		Summary:     "Start a batch re-push job from a prepared snapshot",
		Description: "Starts an async job that re-pushes every event occurrence in a snapshot taken by an earlier GET .../events?prepare_repush=true call. Each re-push creates a new event occurrence with the original payload, without re-validating it; sparrow.* system events are skipped and counted as failed. Returns 202 with the job; poll it with getEventRepushJob.\n\n" +
			"A snapshot can be started once: 409 when it is not pending (already started or cancelled) or expired (15 minutes after it was taken), 404 when the id is unknown, 400 when it is not a UUID.\n\nPortal tokens: allowed, as POST /portal/api/events:rePush.",
		Errors:        []int{400, 404, 409},
		Tags:          []string{"Events"},
		DefaultStatus: http.StatusAccepted,
	}, func(ctx context.Context, in *repushBatchInput) (*batchJobOutput, error) {
		if err := svc.RePushEvents(ctx, batchScope(ctx, in.Consumer), in.Body.RepushID); err != nil {
			return nil, mapError(ctx, err, "failed to start re-push job")
		}
		job, err := svc.GetRepushStatus(ctx, batchScope(ctx, in.Consumer), in.Body.RepushID)
		if err != nil {
			return nil, mapError(ctx, err, "failed to load re-push job status")
		}
		return toBatchJobOutput(job), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getEventRepushJob",
		Method:      http.MethodGet,
		Path:        "/v1/consumers/{consumer}/repush-jobs/{job_id}",
		Summary:     "Get batch re-push job progress",
		Description: "Returns a batch re-push job's status and processed/failed/total counts. 404 when the id is unknown, 409 when the id belongs to a delivery retry job.\n\nPortal tokens: allowed, as GET /portal/api/repush-jobs/{job_id}.",
		Errors:      []int{400, 404, 409},
		Tags:        []string{"Events"},
	}, func(ctx context.Context, in *jobIDInput) (*batchJobOutput, error) {
		job, err := svc.GetRepushStatus(ctx, batchScope(ctx, in.Consumer), in.JobID)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get re-push job status")
		}
		return toBatchJobOutput(job), nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "cancelEventRepushJob",
		Method:        http.MethodPost,
		Path:          "/v1/consumers/{consumer}/repush-jobs/{job_id}:cancel",
		Summary:       "Cancel a pending or in-progress batch re-push job",
		Description:   "Cancels a pending or running batch re-push job; a running job stops at its next progress check (every 25 items). Occurrences already re-pushed are not rolled back. Returns 204. 409 when the job is already completed or cancelled.\n\nPortal tokens: allowed, as POST /portal/api/repush-jobs/{job_id}:cancel.",
		Errors:        []int{400, 404, 409},
		Tags:          []string{"Events"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *jobIDInput) (*emptyOutput, error) {
		if err := svc.CancelRepush(ctx, batchScope(ctx, in.Consumer), in.JobID); err != nil {
			return nil, mapError(ctx, err, "failed to cancel re-push job")
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})
}

// listEventOccurrencesImpl is the shared body of the consumer-scoped and
// global event occurrence list routes. An empty consumer lists across all
// consumers.
func listEventOccurrencesImpl(ctx context.Context, svc eventRouteService, consumer string, p EventOccurrenceListParams) (*listEventOccurrencesOutput, error) {
	if p.Offset != 0 {
		return nil, errOffsetRemoved
	}
	filter := store.EventReportFilter{
		Consumer:      consumer,
		Limit:         int(p.Limit),
		PrepareRepush: p.PrepareRepush,
	}
	after, err := decodeCursor(p.Cursor)
	if err != nil {
		return nil, err
	}
	filter.After = after
	if p.Event != "" {
		filter.EventName = &p.Event
	}
	switch p.SchemaValid {
	case "":
	case "valid":
		v := true
		filter.SchemaValid = &v
	case "invalid":
		v := false
		filter.SchemaValid = &v
	default:
		return nil, huma.Error400BadRequest("schema_valid must be 'valid' or 'invalid'")
	}
	labels, err := parseLabelFilter(p.Labels)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	filter.Labels = labels
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
	page, err := svc.ListEventReports(ctx, filter)
	if err != nil {
		return nil, mapError(ctx, err, "failed to list event occurrences")
	}
	out := &listEventOccurrencesOutput{}
	out.Body.Items = make([]eventOccurrenceItem, 0, len(page.Items))
	for _, r := range page.Items {
		var o eventOccurrenceOutput
		o.Body.EventID = r.ID.String()
		o.Body.Consumer = r.Consumer
		o.Body.Event = r.Event
		o.Body.Payload = r.Payload
		o.Body.Metadata = r.Metadata
		o.Body.Labels = r.Labels
		o.Body.SchemaValid = r.SchemaValid
		o.Body.EventVersion = r.EventVersion
		o.Body.WebhookCount = r.WebhookCount
		o.Body.SuccessfulDeliveries = r.SuccessfulDeliveries
		o.Body.FailedDeliveries = r.FailedDeliveries
		o.Body.PendingDeliveries = r.PendingDeliveries
		o.Body.CreatedAt = r.CreatedAt.Format(time.RFC3339Nano)
		out.Body.Items = append(out.Body.Items, o.Body)
	}
	if n := len(page.Items); n > 0 {
		last := page.Items[n-1]
		out.Body.Pagination = newCursorPagination(p.Limit, page.HasMore, last.CreatedAt, last.ID)
	} else {
		out.Body.Pagination = newCursorPagination(p.Limit, false, time.Time{}, uuid.Nil)
	}
	out.Body.RepushID = page.RepushID
	out.Body.RepushTotal = page.RepushTotal
	return out, nil
}
