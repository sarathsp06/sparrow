package webhooks

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
	"github.com/sarathsp06/sparrow/pkg/storage"
)

// ReservedEventPrefix is the event name prefix reserved for Sparrow's own
// system events (e.g. sparrow.webhook.health_changed). Users can subscribe to
// these events but cannot register, change, import or push them.
const ReservedEventPrefix = "sparrow."

// IsReservedEventName reports whether name uses the reserved prefix. The match
// is case-insensitive so "Sparrow.x" cannot slip past it.
func IsReservedEventName(name string) bool {
	return strings.HasPrefix(strings.ToLower(name), ReservedEventPrefix)
}

// EventTypeDefinition is the desired state of an event type: what a register,
// patch or import asks the event type to become.
type EventTypeDefinition struct {
	Name        string
	Description string
	Schema      map[string]any
	Metadata    map[string]string
	Active      bool
}

// EventTypeSaveAction is the outcome of saving an event type definition.
type EventTypeSaveAction string

const (
	// EventTypeCreated means the event type did not exist and was created at version 1.
	EventTypeCreated EventTypeSaveAction = "created"
	// EventTypeNewVersion means the schema changed and a new version was created.
	EventTypeNewVersion EventTypeSaveAction = "new_version"
	// EventTypeUpdated means the current version was changed in place: its
	// description, metadata or active flag, or a first schema filled in.
	EventTypeUpdated EventTypeSaveAction = "updated"
	// EventTypeUnchanged means the definition already matched; nothing was written.
	EventTypeUnchanged EventTypeSaveAction = "unchanged"
)

// Field names reported in EventTypeSaveResult.Changes.
const (
	ChangeSchema        = "schema"
	ChangeSchemaDefined = "schema_defined"
	ChangeDescription   = "description"
	ChangeMetadata      = "metadata"
	ChangeActive        = "active"
)

// EventTypeSaveResult describes what saving a definition did.
type EventTypeSaveResult struct {
	Name            string
	Action          EventTypeSaveAction
	Version         int
	PreviousVersion int
	Changes         []string
	// ActiveChange is "deactivates" or "reactivates" when the active flag
	// flips, and empty otherwise.
	ActiveChange string
	// Event is the resulting current definition.
	Event *store.EventRegistration
}

// eventTypeSaveMode restricts which outcomes a save may have.
type eventTypeSaveMode int

const (
	// saveUpsert creates the event type or changes it (import).
	saveUpsert eventTypeSaveMode = iota
	// saveCreateOnly fails with AlreadyExists if the event type exists (register).
	saveCreateOnly
	// saveMustExist fails with NotFound if the event type does not exist (patch).
	saveMustExist
)

// planEventTypeSave decides what saving `in` over `current` should do,
// without touching storage. current is nil when the event type does not exist.
//
// Only a schema change creates a new version. Filling in a first schema on a
// version that had none defines the contract rather than changing it, so it
// updates that version in place. Description, metadata and active always
// change in place.
func planEventTypeSave(current *store.EventRegistration, in EventTypeDefinition) EventTypeSaveResult {
	res := EventTypeSaveResult{Name: in.Name}
	if current == nil {
		res.Action = EventTypeCreated
		res.Version = 1
		return res
	}

	res.PreviousVersion = current.Version
	res.Version = current.Version

	curEmpty, inEmpty := len(current.Schema) == 0, len(in.Schema) == 0
	switch {
	case curEmpty && !inEmpty:
		res.Changes = append(res.Changes, ChangeSchemaDefined)
	case !schemasEqual(current.Schema, in.Schema):
		res.Changes = append(res.Changes, ChangeSchema)
		res.Version = current.Version + 1
	}
	if current.Description != in.Description {
		res.Changes = append(res.Changes, ChangeDescription)
	}
	if !metadataEqual(current.Metadata, in.Metadata) {
		res.Changes = append(res.Changes, ChangeMetadata)
	}
	if current.Active != in.Active {
		res.Changes = append(res.Changes, ChangeActive)
		if in.Active {
			res.ActiveChange = "reactivates"
		} else {
			res.ActiveChange = "deactivates"
		}
	}

	switch {
	case res.Version != current.Version:
		res.Action = EventTypeNewVersion
	case len(res.Changes) > 0:
		res.Action = EventTypeUpdated
	default:
		res.Action = EventTypeUnchanged
	}
	return res
}

// schemasEqual compares two decoded JSON Schemas by value, so key order and
// whitespace never count as a change. No schema and an empty schema are equal.
func schemasEqual(a, b map[string]any) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

func metadataEqual(a, b map[string]string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return maps.Equal(a, b)
}

// validateEventTypeName checks a user-supplied event type name.
func validateEventTypeName(name string) error {
	if name == "" {
		return svcerrors.Error(svcerrors.InvalidArgument, "event name is required")
	}
	if utf8.RuneCountInString(name) > maxEventNameLength {
		return svcerrors.Error(svcerrors.InvalidArgument, eventNameTooLong)
	}
	if IsReservedEventName(name) {
		return reservedEventNameError(name)
	}
	return nil
}

func reservedEventNameError(name string) error {
	return svcerrors.Errorf(svcerrors.InvalidArgument,
		"event name %q uses the reserved prefix %q, which is for Sparrow's own events", name, ReservedEventPrefix)
}

// saveEventType validates def and saves it in its own transaction.
func (s *WebhookService) saveEventType(ctx context.Context, def EventTypeDefinition, mode eventTypeSaveMode) (*EventTypeSaveResult, error) {
	if err := validateEventTypeName(def.Name); err != nil {
		return nil, err
	}
	tenantID := tenant.DefaultTenantID

	var res *EventTypeSaveResult
	run := func() error {
		return s.webhookRepo.RunInTransaction(func(tx store.RepositoryInterface) error {
			var err error
			res, err = s.saveEventTypeTx(ctx, tx, tenantID, def, mode)
			return err
		})
	}
	err := run()
	// Two writers can both see "does not exist" (there is no row to lock) and
	// race to create it; the loser gets a unique violation. For an upsert the
	// row now exists, so run again and apply the change on top of it.
	if errors.Is(err, storage.ErrAlreadyExists) && mode == saveUpsert {
		err = run()
	}
	if err != nil {
		return nil, err
	}
	return res, nil
}

// saveEventTypeTx applies def inside the transaction tx. It locks the current
// row, so concurrent writers to one event type are serialized and cannot
// claim the same next version.
func (s *WebhookService) saveEventTypeTx(ctx context.Context, tx store.RepositoryInterface, tenantID uuid.UUID, def EventTypeDefinition, mode eventTypeSaveMode) (*EventTypeSaveResult, error) {
	current, err := tx.GetEventByNameForUpdate(ctx, tenantID, def.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to load event type: %w", err)
	}
	if current != nil && mode == saveCreateOnly {
		return nil, svcerrors.Errorf(svcerrors.AlreadyExists, "event type %q already exists", def.Name)
	}
	if current == nil && mode == saveMustExist {
		return nil, svcerrors.Error(svcerrors.NotFound, "event type not found")
	}

	plan := planEventTypeSave(current, def)
	res := &plan

	next := &store.EventRegistration{
		Name:        def.Name,
		Description: def.Description,
		Schema:      def.Schema,
		Metadata:    def.Metadata,
		Active:      def.Active,
		Version:     plan.Version,
	}

	switch plan.Action {
	case EventTypeUnchanged:
		res.Event = current
		return res, nil

	case EventTypeCreated:
		next.SamplePayload = s.samplePayloadFor(ctx, def.Schema)
		if err := tx.RegisterEvent(ctx, tenantID, next); err != nil {
			return nil, fmt.Errorf("failed to register event type: %w", err)
		}

	case EventTypeNewVersion:
		next.SamplePayload = s.samplePayloadFor(ctx, def.Schema)
		next.CreatedAt = current.CreatedAt
		if err := tx.UpdateEvent(ctx, tenantID, next); err != nil {
			return nil, fmt.Errorf("failed to update event type: %w", err)
		}
		if err := tx.AddEventTypeVersion(ctx, tenantID, &store.EventRegistrationVersion{
			Name:          def.Name,
			Version:       plan.Version,
			Description:   def.Description,
			Schema:        def.Schema,
			SamplePayload: next.SamplePayload,
		}); err != nil {
			return nil, fmt.Errorf("failed to record event type version: %w", err)
		}

	case EventTypeUpdated:
		next.CreatedAt = current.CreatedAt
		fillIn := len(current.Schema) == 0 && len(def.Schema) != 0
		if fillIn {
			next.SamplePayload = s.samplePayloadFor(ctx, def.Schema)
		} else {
			// Same contract: keep the stored schema (identical by value) and
			// its sample payload.
			next.Schema = current.Schema
			next.SamplePayload = current.SamplePayload
		}
		if err := tx.UpdateEvent(ctx, tenantID, next); err != nil {
			return nil, fmt.Errorf("failed to update event type: %w", err)
		}
		if fillIn {
			if err := tx.FillInEventTypeVersion(ctx, tenantID, &store.EventRegistrationVersion{
				Name:          def.Name,
				Version:       plan.Version,
				Description:   def.Description,
				Schema:        def.Schema,
				SamplePayload: next.SamplePayload,
			}); err != nil {
				return nil, fmt.Errorf("failed to fill in event type schema: %w", err)
			}
		}
	}

	saved, err := tx.GetEventByName(ctx, tenantID, def.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to reload event type: %w", err)
	}
	res.Event = saved
	return res, nil
}

// samplePayloadFor generates the example payload stored with a schema. A
// generation failure is not fatal: the definition is saved with an empty
// sample.
func (s *WebhookService) samplePayloadFor(ctx context.Context, schema map[string]any) map[string]any {
	sample, err := generateSamplePayload(schema)
	if err != nil {
		s.logger.WarnContext(ctx, "Failed to generate sample payload, using empty payload", "error", err)
		return map[string]any{}
	}
	return sample
}
