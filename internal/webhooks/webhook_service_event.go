package webhooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	jsonschema "github.com/kaptinlin/jsonschema"
	"github.com/sarathsp06/schemagen"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
	"github.com/sarathsp06/sparrow/pkg/storage"
)

// maxEventNameLength matches the VARCHAR(255) event-name columns. Postgres
// counts characters, not bytes, so the check does too.
const maxEventNameLength = 255

const eventNameTooLong = "event name must be at most 255 characters"

// PushEvent pushes an event.
// When idempotencyKey is non-nil and non-empty, duplicate detection is
// performed: if an event with the same key already exists within the
// (tenant, consumer), the existing event_id is returned with
// isDuplicate=true and no new event or deliveries are created.
// Re-push/re-enqueue flows pass nil, so they are never deduplicated.
func (s *WebhookService) PushEvent(ctx context.Context, consumer string, event string, payload map[string]any, ttlSeconds int64, metadata map[string]string, labels map[string]string, idempotencyKey *string) (string, bool, bool, []string, error) {
	ctx, span := s.tracer.Start(ctx, "event.push",
		trace.WithAttributes(
			attribute.String("consumer", consumer),
			attribute.String("event", event),
		),
	)
	defer span.End()

	s.logger.InfoContext(ctx, "Processing push event request",
		"consumer", consumer,
		"event", event,
	)

	tenantID := tenant.DefaultTenantID

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
	}

	if err := s.webhookRepo.StoreEvent(ctx, tenantID, eventRecord); err != nil {
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
		s.logger.ErrorContext(ctx, "Failed to store event record", "error", err, "event_id", eventID)
		return "", false, false, nil, fmt.Errorf("failed to store event record: %w", err)
	}

	// Create event processing job with minimal data.
	//
	// NOTE: Cross-driver transaction gap — the event INSERT above uses sqlx
	// while the River job INSERT below uses pgx. These cannot share a
	// database transaction. If the River insert fails, we compensate by
	// deleting the orphaned event record (see error handling below).
	// This is an accepted architectural tradeoff; making both drivers share
	// a transaction would require migrating all app queries to pgx.
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

	// Insert the event processing job
	_, err = s.jobInserter.Insert(ctx, eventArgs)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to schedule event processing job",
			"event_id", eventID,
			"consumer", consumer,
			"event", event,
			"error", err,
		)
		// Compensation: delete the orphaned event record since the job failed.
		// This prevents events from existing in the DB without a corresponding
		// processing job (cross-driver: sqlx event store + pgx River job).
		if delErr := s.webhookRepo.DeleteEventByID(ctx, tenantID, eventRecord.ID); delErr != nil {
			s.logger.ErrorContext(ctx, "Failed to compensate: could not delete orphaned event record",
				"event_id", eventID,
				"delete_error", delErr,
			)
		}
		return "", false, false, nil, fmt.Errorf("failed to schedule event processing: %w", err)
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
	ctx, span := s.tracer.Start(ctx, "event.repush",
		trace.WithAttributes(
			attribute.String("original_event_id", eventID),
		),
	)
	defer span.End()

	s.logger.InfoContext(ctx, "Processing single event re-push",
		"original_event_id", eventID,
	)

	tenantID := tenant.DefaultTenantID

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
	newEventID, _, _, warnings, err := s.PushEvent(ctx, original.Consumer, original.Event, original.Payload, original.TTL, original.Metadata, original.Labels, nil)
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
	ctx, span := s.tracer.Start(ctx, "event.get_record",
		trace.WithAttributes(
			attribute.String("event_id", eventID),
		),
	)
	defer span.End()

	tenantID := tenant.DefaultTenantID

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

// generateSamplePayload generates a sample payload from the given schema using schemagen
func generateSamplePayload(schema map[string]any) (map[string]any, error) {
	if len(schema) == 0 {
		return map[string]any{}, nil
	}

	// Convert schema to JSON bytes
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	// Create generator and generate sample
	generator := schemagen.NewGenerator().SetGenerateAllFields(true)
	sample, err := generator.Generate(schemaBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate sample payload: %w", err)
	}

	// Convert to map[string]any
	if sampleMap, ok := sample.(map[string]any); ok {
		return sampleMap, nil
	}

	return map[string]any{}, nil
}

// autoRegisterEvent creates a schema-less event type on first push, when
// SPARROW_AUTO_REGISTER_EVENTS is on. A later register or import that adds a
// schema fills in this version 1 rather than creating version 2.
func (s *WebhookService) autoRegisterEvent(ctx context.Context, tenantID uuid.UUID, event string) (*store.EventRegistration, error) {
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
	ctx, span := s.tracer.Start(ctx, "WebhookService.RegisterEvent")
	defer span.End()

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

// ListEvents lists all registered events
func (s *WebhookService) ListEvents(ctx context.Context, activeOnly bool, limit, offset int32) ([]*store.EventRegistration, int32, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListEvents")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing list events request",
		"active_only", activeOnly, "limit", limit, "offset", offset)

	tenantID := tenant.DefaultTenantID

	// Event types are tenant-scoped

	l, o := normalizePagination(int(limit), int(offset))
	limit, offset = int32(l), int32(o)

	events, totalCount, err := s.webhookRepo.ListEventsPaginated(ctx, tenantID, activeOnly, int(limit), int(offset))
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to list events", "error", err)
		return nil, 0, fmt.Errorf("failed to retrieve events: %w", err)
	}
	s.logger.InfoContext(ctx, "Listed events successfully",
		"count", len(events),
		"total", totalCount,
	)
	return events, int32(totalCount), nil
}

// UpdateEvent replaces an existing event type's definition. A schema change
// creates a new version; everything else changes the current version in
// place. See planEventTypeSave for the exact rules.
//
// A breaking schema change (see ClassifySchemaChange) to an event type that
// subscriptions receive fails with FailedPrecondition unless allowBreaking.
func (s *WebhookService) UpdateEvent(ctx context.Context, name string, description string, schema map[string]any, metadata map[string]string, active bool, allowBreaking bool) (*EventTypeSaveResult, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.UpdateEvent")
	defer span.End()

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
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListEventTypeVersions")
	defer span.End()

	if _, err := s.GetEvent(ctx, name); err != nil {
		return nil, err
	}
	versions, err := s.webhookRepo.ListEventTypeVersions(ctx, tenant.DefaultTenantID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to list event type versions: %w", err)
	}
	return versions, nil
}

// GetEventTypeVersion returns one version of an event type.
func (s *WebhookService) GetEventTypeVersion(ctx context.Context, name string, version int) (*store.EventRegistrationVersion, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetEventTypeVersion")
	defer span.End()

	if name == "" {
		return nil, svcerrors.Error(svcerrors.InvalidArgument, "event name is required")
	}
	v, err := s.webhookRepo.GetEventTypeVersion(ctx, tenant.DefaultTenantID, name, version)
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
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetEvent")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing get event request", "name", name)
	if name == "" {
		return nil, svcerrors.Error(svcerrors.InvalidArgument, "event name is required")
	}

	tenantID := tenant.DefaultTenantID

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
	return fmt.Sprintf("%s: %s", e.Message, strings.Join(parts, "; "))
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
	return warnings
}

// ValidateJSONSchema validates a payload against a JSON schema string.
// Returns a SchemaValidationError with detailed per-field errors on failure.
func ValidateJSONSchema(schema map[string]any, payload map[string]any) error {
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	sch, err := compiler.Compile(schemaJSON)
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
func (s *WebhookService) ListEventReports(ctx context.Context, filter store.EventReportFilter) ([]*store.EventReportWithStats, int32, string, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.ListEventReports")
	defer span.End()

	s.logger.InfoContext(ctx, "Processing list event reports request",
		"consumer", filter.Consumer,
		"event_name", filter.EventName,
		"prepare_repush", filter.PrepareRepush,
		"limit", filter.Limit,
		"offset", filter.Offset)

	tenantID := tenant.DefaultTenantID

	filter.Limit, filter.Offset = normalizePagination(filter.Limit, filter.Offset)

	events, totalCount, err := s.webhookRepo.ListEventReportsFiltered(ctx, tenantID, filter)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to list event reports", "consumer", filter.Consumer, "event_name", filter.EventName, "error", err)
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, 0, "", fmt.Errorf("failed to list event reports: %w", err)
	}

	// Snapshot matching IDs into a batch job if requested
	var repushID string
	if filter.PrepareRepush {
		ids, err := s.webhookRepo.SnapshotEventIDs(ctx, tenantID, filter)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to snapshot event IDs for repush", "error", err)
			return nil, 0, "", fmt.Errorf("failed to prepare repush: %w", err)
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
				return nil, 0, "", fmt.Errorf("failed to create repush batch: %w", err)
			}
			repushID = batchJob.ID.String()
			s.logger.InfoContext(ctx, "Created repush batch job",
				"repush_id", repushID,
				"event_count", len(ids))
		}
	}

	s.logger.InfoContext(ctx, "Successfully listed event reports",
		"consumer", filter.Consumer,
		"event_name", filter.EventName,
		"count", len(events),
		"total", totalCount)

	span.SetAttributes(
		attribute.String("consumer", filter.Consumer),
		attribute.Int("count", len(events)),
		attribute.Int("total", totalCount),
	)
	if filter.EventName != nil {
		span.SetAttributes(attribute.String("event_name", *filter.EventName))
	}

	return events, int32(totalCount), repushID, nil
}
