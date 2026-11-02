package webhooks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	jsonschema "github.com/kaptinlin/jsonschema"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	svcerrors "github.com/sarathsp06/sparrow/internal/errors"
	"github.com/sarathsp06/sparrow/internal/storage"
	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// maxEventNameLength matches the VARCHAR(255) event-name columns. Postgres
// counts characters, not bytes, so the check does too.
const maxEventNameLength = 255

const eventNameTooLong = "event name must be at most 255 characters"

// maxScheduleAhead caps how far in the future an event can be scheduled.
// Far-future events would sit in the queue indefinitely and outlive any
// reasonable retention window.
const maxScheduleAhead = 365 * 24 * time.Hour

// PushEvent pushes an event.
// When idempotencyKey is non-nil and non-empty, duplicate detection is
// performed: if an event with the same key already exists within the
// (tenant, consumer), the existing event_id is returned with
// isDuplicate=true and no new event or deliveries are created.
// Re-push/re-enqueue flows pass nil, so they are never deduplicated.
// A non-nil scheduledAt delays the fan-out (and so all deliveries) until
// that time; a past or zero value means immediate.
func (s *WebhookService) PushEvent(ctx context.Context, consumer string, event string, payload map[string]any, ttlSeconds int64, metadata map[string]string, labels map[string]string, idempotencyKey *string, scheduledAt *time.Time) (string, bool, bool, []string, error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("consumer", consumer),
		attribute.String("event", event),
	)

	s.logger.InfoContext(ctx, "Processing push event request",
		"consumer", consumer,
		"event", event,
	)

	tenantID := tenant.FromContext(ctx)

	// Validate required fields
	if consumer == "" {
		err := svcerrors.Error(svcerrors.InvalidArgument, "consumer is required")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "consumer is required")
		return "", false, false, nil, err
	}
	if event == "" {
		err := svcerrors.Error(svcerrors.InvalidArgument, "event is required")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "event is required")
		return "", false, false, nil, err
	}
	if utf8.RuneCountInString(event) > maxEventNameLength {
		err := svcerrors.Error(svcerrors.InvalidArgument, eventNameTooLong)
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "event name too long")
		return "", false, false, nil, err
	}
	// Both are VARCHAR(255): a longer value failed the INSERT with a 500.
	if utf8.RuneCountInString(consumer) > maxEventNameLength {
		err := svcerrors.Errorf(svcerrors.InvalidArgument, "consumer must be at most %d characters, got %d", maxEventNameLength, utf8.RuneCountInString(consumer))
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "consumer too long")
		return "", false, false, nil, err
	}
	if idempotencyKey != nil && utf8.RuneCountInString(*idempotencyKey) > maxEventNameLength {
		err := svcerrors.Errorf(svcerrors.InvalidArgument, "idempotency_key must be at most %d characters, got %d", maxEventNameLength, utf8.RuneCountInString(*idempotencyKey))
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "idempotency_key too long")
		return "", false, false, nil, err
	}
	if err := validateLabels(labels, "labels"); err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "invalid labels")
		return "", false, false, nil, err
	}
	if IsReservedEventName(event) {
		// Only Sparrow itself emits sparrow.* events (see queue.pushSystemEvent).
		err := reservedEventNameError(event)
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "reserved event name")
		return "", false, false, nil, err
	}

	// Normalize scheduling: past or zero means immediate; cap the horizon.
	now := time.Now()
	if scheduledAt != nil && !scheduledAt.After(now) {
		scheduledAt = nil
	}
	if scheduledAt != nil && scheduledAt.After(now.Add(maxScheduleAhead)) {
		err := svcerrors.Error(svcerrors.InvalidArgument, "scheduled_at must be within one year from now")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "scheduled_at too far in the future")
		return "", false, false, nil, err
	}

	// Idempotency check: if the caller provided an idempotency key, look up
	// an existing event with the same key in this (tenant, consumer). When
	// found, return the existing event_id immediately — no new record, no
	// new deliveries. This check is intentionally skipped for re-push flows
	// (which pass nil) so that replays always create new events.
	if idempotencyKey != nil && *idempotencyKey != "" {
		existing, err := s.webhookRepo.GetEventByIdempotencyKey(ctx, tenantID, consumer, *idempotencyKey)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, "idempotency lookup failed")
			s.logger.ErrorContext(ctx, "Failed to check idempotency key", "idempotency_key", *idempotencyKey, "error", err)
			return "", false, false, nil, fmt.Errorf("failed to check idempotency key: %w", err)
		}
		if existing != nil {
			span.SetAttributes(attribute.Bool("duplicate", true))
			span.SetStatus(otelcodes.Ok, "duplicate event (idempotent)")
			s.logger.InfoContext(ctx, "Duplicate event detected via idempotency key",
				"idempotency_key", *idempotencyKey,
				"existing_event_id", existing.ID.String(),
			)
			return existing.ID.String(), true, existing.SchemaValid, nil, nil
		}
	}

	// Look up the registered event type. Unknown names are rejected unless
	// auto-registration is enabled (a development convenience; production
	// definitions arrive by register or import).
	eventReg, err := s.webhookRepo.GetEventByName(ctx, tenantID, event)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "event lookup failed")
		s.logger.ErrorContext(ctx, "Failed to lookup event registration", "event", event, "error", err)
		return "", false, false, nil, fmt.Errorf("failed to lookup event registration: %w", err)
	}
	if eventReg == nil {
		if !s.autoRegisterEvents {
			err := svcerrors.Errorf(svcerrors.NotFound, "event type %q is not registered; register or import it before pushing", event)
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, "event type not registered")
			return "", false, false, nil, err
		}
		eventReg, err = s.autoRegisterEvent(ctx, tenantID, event)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, "auto-registration failed")
			return "", false, false, nil, err
		}
	}
	if !eventReg.Active {
		err := svcerrors.Errorf(svcerrors.FailedPrecondition, "event '%s' is inactive", event)
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "event inactive")
		s.logger.ErrorContext(ctx, "Event is inactive", "event", event)
		return "", false, false, nil, err
	}

	// Soft schema validation: validate payload against event schema if present.
	// Events are always accepted and stored regardless of schema match.
	// Invalid payloads are tagged (schema_valid=false) with per-field warnings.
	var warnings []string
	schemaValid := true
	if len(eventReg.Schema) != 0 && payload != nil {
		if err := ValidateJSONSchema(eventReg.Schema, payload); err != nil {
			var schemaErr *SchemaValidationError
			if errors.As(err, &schemaErr) {
				schemaValid = false
				warnings = schemaErr.Warnings()
				s.logger.WarnContext(ctx, "Payload does not match event schema (accepted with warnings)",
					"event", event,
					"warning_count", len(warnings),
				)
				span.SetAttributes(attribute.Bool("schema_valid", false))
			} else {
				// Non-schema error (e.g., schema compilation failure) -- still accept
				schemaValid = false
				warnings = []string{err.Error()}
				s.logger.WarnContext(ctx, "Schema validation encountered unexpected error (accepted with warnings)",
					"event", event,
					"error", err,
				)
			}
		}
	}

	// TTL=0 means no expiry (default). Only positive values enable expiry.
	ttl := ttlSeconds
	if ttl < 0 {
		ttl = 0
	}

	// Generate event ID
	eventID := uuid.New().String()

	// Store the event record in database first
	eventRecord := &store.EventRecord{
		ID:             uuid.MustParse(eventID),
		Consumer:       consumer,
		Event:          event,
		Payload:        payload,
		TTL:            ttl,
		Metadata:       metadata,
		Labels:         labels,
		SchemaValid:    schemaValid,
		EventVersion:   eventReg.Version,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      time.Now(),
		ScheduledAt:    scheduledAt,
	}
	if scheduledAt != nil && ttl > 0 {
		// TTL counts from fan-out, which for a scheduled event is the
		// scheduled time, not the push. Without this the event row (and
		// retention) would expire it before it ever fires.
		eventRecord.ExpiresAt = scheduledAt.Add(time.Duration(ttl) * time.Second)
	}

	eventArgs := queue.EventArgs{
		EventID:    eventID,
		Consumer:   consumer,
		Event:      event,
		TTLSeconds: ttl,
		Metadata:   metadata,
		Labels:     labels,
		CreatedAt:  eventRecord.CreatedAt,
		TenantID:   tenantID.String(),
	}
	if scheduledAt != nil {
		eventArgs.ScheduledAt = *scheduledAt
	}

	// The event row and its processing job commit together: an event is
	// never stored without the job that delivers it, which an idempotent
	// retry would then answer as a duplicate.
	err = s.webhookRepo.StoreEventAndEnqueue(ctx, tenantID, eventRecord, func(ctx context.Context, tx *sql.Tx) error {
		return s.jobInserter.BatchInsertTx(ctx, tx, []river.JobArgs{eventArgs})
	})
	if err != nil {
		// Idempotency race: a concurrent push with the same key won the insert
		// between our lookup above and this insert. Return the winner as a duplicate.
		if errors.Is(err, storage.ErrAlreadyExists) && idempotencyKey != nil && *idempotencyKey != "" {
			existing, lookupErr := s.webhookRepo.GetEventByIdempotencyKey(ctx, tenantID, consumer, *idempotencyKey)
			if lookupErr == nil && existing != nil {
				span.SetAttributes(attribute.Bool("duplicate", true))
				span.SetStatus(otelcodes.Ok, "duplicate event (idempotent)")
				s.logger.InfoContext(ctx, "Duplicate event detected on insert (idempotency race)",
					"idempotency_key", *idempotencyKey,
					"existing_event_id", existing.ID.String(),
				)
				return existing.ID.String(), true, existing.SchemaValid, nil, nil
			}
		}
		s.logger.ErrorContext(ctx, "Failed to store and schedule event", "error", err, "event_id", eventID)
		return "", false, false, nil, fmt.Errorf("failed to store event: %w", err)
	}

	// Record metrics
	if s.metrics != nil {
		s.metrics.EventsPushed.Add(ctx, 1)
	}

	span.SetStatus(otelcodes.Ok, "event scheduled successfully")

	s.logger.InfoContext(ctx, "Event processing scheduled successfully",
		"event_id", eventID,
		"consumer", consumer,
		"event", event,
	)
	return eventID, false, schemaValid, warnings, nil
}

// RePushEvent replays a previously pushed event as if it were pushed fresh.
// It loads the original event record and calls PushEvent with the same payload,
// consumer, event name, metadata, and labels. The payload is validated against
// the CURRENT event type schema. Returns a new event_id and any warnings.
func (s *WebhookService) RePushEvent(ctx context.Context, eventID string) (string, []string, error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("original_event_id", eventID),
	)

	s.logger.InfoContext(ctx, "Processing single event re-push",
		"original_event_id", eventID,
	)

	tenantID := tenant.FromContext(ctx)

	// Parse and validate event ID
	id, err := parseUUID(eventID, "event ID")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "invalid event ID")
		return "", nil, err
	}

	// Load original event record
	original, err := s.webhookRepo.GetEventByID(ctx, tenantID, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "failed to load original event")
		s.logger.ErrorContext(ctx, "Failed to load original event for re-push",
			"event_id", eventID,
			"error", err,
		)
		return "", nil, fmt.Errorf("failed to load original event: %w", err)
	}
	if original == nil {
		err := svcerrors.Errorf(svcerrors.NotFound, "event not found: %s", eventID)
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "event not found")
		return "", nil, err
	}

	// Re-push through the standard PushEvent pipeline with nil idempotency key.
	// This ensures re-pushes always create new events and are never deduplicated.
	// This gives us: current schema validation, new event_id, fan-out to matching subscriptions.
	newEventID, _, _, warnings, err := s.PushEvent(ctx, original.Consumer, original.Event, original.Payload, original.TTL, original.Metadata, original.Labels, nil, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "re-push failed")
		s.logger.ErrorContext(ctx, "Failed to re-push event",
			"original_event_id", eventID,
			"error", err,
		)
		return "", nil, fmt.Errorf("failed to re-push event: %w", err)
	}

	span.SetStatus(otelcodes.Ok, "event re-pushed successfully")
	span.SetAttributes(attribute.String("new_event_id", newEventID))

	s.logger.InfoContext(ctx, "Event re-pushed successfully",
		"original_event_id", eventID,
		"new_event_id", newEventID,
	)
	return newEventID, warnings, nil
}

// GetEventRecord retrieves a single pushed event instance by UUID with delivery statistics.
func (s *WebhookService) GetEventRecord(ctx context.Context, eventID string) (*store.EventRecord, int32, int32, int32, int32, error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("event_id", eventID),
	)

	tenantID := tenant.FromContext(ctx)

	// Parse and validate event ID
	id, err := parseUUID(eventID, "event ID")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "invalid event ID")
		return nil, 0, 0, 0, 0, err
	}

	// Load event record
	record, err := s.webhookRepo.GetEventByID(ctx, tenantID, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "failed to load event record")
		return nil, 0, 0, 0, 0, fmt.Errorf("failed to load event record: %w", err)
	}
	if record == nil {
		return nil, 0, 0, 0, 0, svcerrors.Error(svcerrors.NotFound, "event record not found")
	}

	// Get delivery statistics
	webhookCount, successCount, failedCount, pendingCount, err := s.webhookRepo.GetEventDeliveryStats(ctx, tenantID, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "failed to get delivery stats")
		s.logger.ErrorContext(ctx, "Failed to get delivery stats for event record",
			"event_id", eventID,
			"error", err,
		)
		// Return the record without stats rather than failing entirely
		return record, 0, 0, 0, 0, nil
	}

	return record, webhookCount, successCount, failedCount, pendingCount, nil
}

// autoRegisterEvent creates a schema-less event type on first push, when
// SPARROW_AUTO_REGISTER_EVENTS is on. A later register or import that adds a
// schema fills in this version 1 rather than creating version 2.
func (s *WebhookService) autoRegisterEvent(ctx context.Context, tenantID uuid.UUID, event string) (*store.EventRegistration, error) {
	if err := validateNewEventTypeName(event); err != nil {
		return nil, err
	}
	reg := &store.EventRegistration{Name: event, Active: true, Version: 1}
	err := s.webhookRepo.RegisterEvent(ctx, tenantID, reg)
	if errors.Is(err, storage.ErrAlreadyExists) {
		// A concurrent push registered it first; use that one.
		existing, getErr := s.webhookRepo.GetEventByName(ctx, tenantID, event)
		if getErr != nil {
			return nil, fmt.Errorf("failed to reload auto-registered event: %w", getErr)
		}
		if existing != nil {
			return existing, nil
		}
	}
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to auto-register event", "event", event, "error", err)
		return nil, fmt.Errorf("failed to auto-register event: %w", err)
	}
	s.logger.InfoContext(ctx, "Auto-registered new event type", "event", event)
	return reg, nil
}

// RegisterEvent registers a new event type at version 1. It fails with
// AlreadyExists if the name is taken; use UpdateEvent to change a type.
func (s *WebhookService) RegisterEvent(ctx context.Context, name string, description string, schema map[string]any, metadata map[string]string, active bool) (string, time.Time, error) {
	s.logger.InfoContext(ctx, "Processing event registration request", "name", name, "description", description)
	res, err := s.saveEventType(ctx, EventTypeDefinition{
		Name:        name,
		Description: description,
		Schema:      schema,
		Metadata:    metadata,
		Active:      active,
	}, saveCreateOnly, saveEventTypeOptions{})
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to register event", "name", name, "error", err)
		return "", time.Time{}, err
	}
	s.logger.InfoContext(ctx, "Event registered successfully", "name", name, "version", res.Version)
	return res.Name, res.Event.CreatedAt, nil
}

// ListEvents lists the event types consumer can subscribe to (see
// checkEventForConsumer): Sparrow's own sparrow.* events for the system
// consumer, tenant event types for any other consumer or none.
func (s *WebhookService) ListEvents(ctx context.Context, activeOnly bool, consumer string, limit, offset int32) ([]*store.EventRegistration, int32, error) {
	s.logger.InfoContext(ctx, "Processing list events request",
		"active_only", activeOnly, "consumer", consumer, "limit", limit, "offset", offset)

	tenantID := tenant.FromContext(ctx)

	// Event types are tenant-scoped

	l, o := normalizePagination(int(limit), int(offset))
	limit, offset = clampInt32(l), clampInt32(o)

	filter := store.EventTypeFilter{ActiveOnly: activeOnly, System: eventTypesForConsumer(consumer)}
	events, totalCount, err := s.webhookRepo.ListEventsPaginated(ctx, tenantID, filter, int(limit), int(offset))
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to list events", "error", err)
		return nil, 0, fmt.Errorf("failed to retrieve events: %w", err)
	}
	s.logger.InfoContext(ctx, "Listed events successfully",
		"count", len(events),
		"total", totalCount,
	)
	return events, clampInt32(totalCount), nil
}

// UpdateEvent replaces an existing event type's definition. A schema change
// creates a new version; everything else changes the current version in
// place. See planEventTypeSave for the exact rules.
//
// A breaking schema change (see ClassifySchemaChange) to an event type that
// subscriptions receive fails with FailedPrecondition unless allowBreaking.
func (s *WebhookService) UpdateEvent(ctx context.Context, name string, description string, schema map[string]any, metadata map[string]string, active bool, allowBreaking bool) (*EventTypeSaveResult, error) {
	s.logger.InfoContext(ctx, "Processing event update request", "name", name)
	res, err := s.saveEventType(ctx, EventTypeDefinition{
		Name:        name,
		Description: description,
		Schema:      schema,
		Metadata:    metadata,
		Active:      active,
	}, saveMustExist, saveEventTypeOptions{AllowBreaking: allowBreaking})
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to update event", "name", name, "error", err)
		return nil, err
	}
	s.logger.InfoContext(ctx, "Event updated", "name", name, "action", res.Action, "version", res.Version)
	return res, nil
}

// ListEventTypeVersions returns every version of an event type, newest first.
func (s *WebhookService) ListEventTypeVersions(ctx context.Context, name string) ([]*store.EventRegistrationVersion, error) {
	if _, err := s.GetEvent(ctx, name); err != nil {
		return nil, err
	}
	versions, err := s.webhookRepo.ListEventTypeVersions(ctx, tenant.FromContext(ctx), name)
	if err != nil {
		return nil, fmt.Errorf("failed to list event type versions: %w", err)
	}
	return versions, nil
}

// GetEventTypeVersion returns one version of an event type.
func (s *WebhookService) GetEventTypeVersion(ctx context.Context, name string, version int) (*store.EventRegistrationVersion, error) {
	if name == "" {
		return nil, svcerrors.Error(svcerrors.InvalidArgument, "event name is required")
	}
	v, err := s.webhookRepo.GetEventTypeVersion(ctx, tenant.FromContext(ctx), name, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get event type version: %w", err)
	}
	if v == nil {
		return nil, svcerrors.Errorf(svcerrors.NotFound, "event type %q has no version %d", name, version)
	}
	return v, nil
}

// GetEvent retrieves an event registration by name
func (s *WebhookService) GetEvent(ctx context.Context, name string) (*store.EventRegistration, error) {
	s.logger.InfoContext(ctx, "Processing get event request", "name", name)
	if name == "" {
		return nil, svcerrors.Error(svcerrors.InvalidArgument, "event name is required")
	}

	tenantID := tenant.FromContext(ctx)

	// Event types are tenant-scoped

	event, err := s.webhookRepo.GetEventByName(ctx, tenantID, name)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get event", "error", err)
		return nil, fmt.Errorf("failed to retrieve event: %w", err)
	}
	if event == nil {
		return nil, svcerrors.Error(svcerrors.NotFound, "event type not found")
	}

	return event, nil
}

// SchemaValidationError represents a structured schema validation failure
// with per-field error details that can be displayed to the user.
type SchemaValidationError struct {
	Message string            `json:"message"`
	Details map[string]string `json:"details"` // field path -> error message
}

func (e *SchemaValidationError) Error() string {
	if len(e.Details) == 0 {
		return e.Message
	}
	var parts []string
	for field, msg := range e.Details {
		if field == "" {
			parts = append(parts, msg)
		} else {
			parts = append(parts, fmt.Sprintf("%s: %s", field, msg))
		}
	}
	return fmt.Sprintf("%s: %s", e.Message, strings.Join(capWarnings(parts), "; "))
}

// maxSchemaWarnings bounds the messages reported for one payload. A large
// array of wrongly typed items gets one per item, which made a 1 MiB push
// answer with a response about 20 times larger.
const maxSchemaWarnings = 100

// capWarnings sorts warnings, so the ones kept do not change from push to
// push, and replaces those past maxSchemaWarnings with a count.
func capWarnings(warnings []string) []string {
	if len(warnings) <= maxSchemaWarnings {
		return warnings
	}
	slices.Sort(warnings)
	more := len(warnings) - maxSchemaWarnings
	return append(warnings[:maxSchemaWarnings:maxSchemaWarnings], fmt.Sprintf("... and %d more schema warnings", more))
}

// Warnings returns per-field validation messages suitable for API responses.
// Each warning is a human-readable string describing a specific validation failure.
func (e *SchemaValidationError) Warnings() []string {
	var warnings []string
	for field, msg := range e.Details {
		if field == "" {
			warnings = append(warnings, msg)
		} else {
			warnings = append(warnings, fmt.Sprintf("field '%s': %s", field, msg))
		}
	}
	if len(warnings) == 0 {
		warnings = append(warnings, e.Message)
	}
	return capWarnings(warnings)
}

// newSchemaCompiler returns a compiler that never fetches a remote $ref. The
// library registers plain http/https loaders by default, which would let
// anyone who can save a schema make the server GET any URL, outside the
// delivery network policy and without a size limit.
func newSchemaCompiler() *jsonschema.Compiler {
	c := jsonschema.NewCompiler()
	clear(c.Loaders)
	return c
}

// ValidateJSONSchema validates a payload against a JSON schema string.
// Returns a SchemaValidationError with detailed per-field errors on failure.
func ValidateJSONSchema(schema map[string]any, payload map[string]any) error {
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}
	sch, err := newSchemaCompiler().Compile(schemaJSON)
	if err != nil {
		return fmt.Errorf("invalid event schema: %w", err)
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	result := sch.ValidateJSON(payloadBytes)
	if result == nil || result.Valid {
		return nil
	}

	// Extract detailed per-field errors from the evaluation result
	details := result.DetailedErrors()

	return &SchemaValidationError{
		Message: "payload validation failed",
		Details: details,
	}
}

// ListEventReports lists event records with delivery statistics in descending order by creation time.
// Supports filtering by consumer, event name, schema_valid, labels, and time range.
// When PrepareRepush is true, snapshots all matching event IDs into a batch job and returns the batch ID.
func (s *WebhookService) ListEventReports(ctx context.Context, filter store.EventReportFilter) (*EventReportPage, error) {
	span := trace.SpanFromContext(ctx)

	s.logger.InfoContext(ctx, "Processing list event reports request",
		"consumer", filter.Consumer,
		"event_name", filter.EventName,
		"prepare_repush", filter.PrepareRepush,
		"limit", filter.Limit)

	tenantID := tenant.FromContext(ctx)

	filter.Limit, _ = normalizePagination(filter.Limit, 0)

	events, hasMore, err := s.webhookRepo.ListEventReportsFiltered(ctx, tenantID, filter)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to list event reports", "consumer", filter.Consumer, "event_name", filter.EventName, "error", err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, fmt.Errorf("failed to list event reports: %w", err)
	}

	// Snapshot matching IDs into a batch job if requested
	page := &EventReportPage{Items: events, HasMore: hasMore}
	if filter.PrepareRepush {
		ids, err := s.webhookRepo.SnapshotEventIDs(ctx, tenantID, filter)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to snapshot event IDs for repush", "error", err)
			return nil, fmt.Errorf("failed to prepare repush: %w", err)
		}
		if len(ids) > 0 {
			filterMap := map[string]any{
				"consumer": filter.Consumer,
			}
			if filter.EventName != nil {
				filterMap["event_name"] = *filter.EventName
			}
			if filter.SchemaValid != nil {
				filterMap["schema_valid"] = *filter.SchemaValid
			}
			if len(filter.Labels) > 0 {
				filterMap["labels"] = filter.Labels
			}
			batchData := &store.BatchJobData{
				ItemIDs: ids,
				Filter:  filterMap,
			}
			batchJob, err := s.webhookRepo.CreateBatchJob(ctx, tenantID, filter.Consumer, store.BatchTypeEventRepush, batchData)
			if err != nil {
				s.logger.ErrorContext(ctx, "Failed to create batch job for repush", "error", err)
				return nil, fmt.Errorf("failed to create repush batch: %w", err)
			}
			page.RepushID = batchJob.ID.String()
			page.RepushTotal = len(ids)
			s.logger.InfoContext(ctx, "Created repush batch job",
				"repush_id", page.RepushID,
				"event_count", len(ids))
		}
	}

	s.logger.InfoContext(ctx, "Successfully listed event reports",
		"consumer", filter.Consumer,
		"event_name", filter.EventName,
		"count", len(events),
		"has_more", hasMore)

	span.SetAttributes(
		attribute.String("consumer", filter.Consumer),
		attribute.Int("count", len(events)),
		attribute.Bool("has_more", hasMore),
	)
	if filter.EventName != nil {
		span.SetAttributes(attribute.String("event_name", *filter.EventName))
	}

	return page, nil
}
