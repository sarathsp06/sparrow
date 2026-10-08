package store

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// EventTypeRepository defines operations for event type registrations and
// their version history.
//
// event_registrations holds the current definition of each event type;
// event_registration_versions holds every version, including the current one.
// Event types are never deleted.
type EventTypeRepository interface {
	RegisterEvent(ctx context.Context, tenantID uuid.UUID, event *EventRegistration) error
	GetEventByName(ctx context.Context, tenantID uuid.UUID, eventName string) (*EventRegistration, error)
	GetEventByNameForUpdate(ctx context.Context, tenantID uuid.UUID, eventName string) (*EventRegistration, error)
	ListEvents(ctx context.Context, tenantID uuid.UUID, activeOnly bool) ([]*EventRegistration, error)
	ListEventsPaginated(ctx context.Context, tenantID uuid.UUID, filter EventTypeFilter, limit, offset int) ([]*EventRegistration, int, error)
	UpdateEvent(ctx context.Context, tenantID uuid.UUID, event *EventRegistration) error
	AddEventTypeVersion(ctx context.Context, tenantID uuid.UUID, version *EventRegistrationVersion) error
	FillInEventTypeVersion(ctx context.Context, tenantID uuid.UUID, version *EventRegistrationVersion) error
	ListEventTypeVersions(ctx context.Context, tenantID uuid.UUID, eventName string) ([]*EventRegistrationVersion, error)
	GetEventTypeVersion(ctx context.Context, tenantID uuid.UUID, eventName string, version int) (*EventRegistrationVersion, error)
}

const eventRegistrationColumns = `tenant_id, name, description, schema, sample_payload, metadata, active, version, created_at, updated_at`

const eventRegistrationVersionColumns = `tenant_id, name, version, description, schema, sample_payload, schema_defined_at, created_at`

// eventRegistrationVersionSelect tolerates NULL descriptions copied in by the
// migration backfill.
const eventRegistrationVersionSelect = `tenant_id, name, version, COALESCE(description, '') AS description, schema, sample_payload, schema_defined_at, created_at`

// RegisterEvent creates an event type at version 1 together with its first
// history row. Both rows are written by a single statement, so the pair is
// atomic without an enclosing transaction.
func (r *Repository) RegisterEvent(ctx context.Context, tenantID uuid.UUID, event *EventRegistration) error {
	event.TenantID = tenantID
	now := time.Now()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	event.UpdatedAt = now
	if event.Version <= 0 {
		event.Version = 1
	}

	query := `
		WITH reg AS (
			INSERT INTO event_registrations (
				tenant_id, name, description, schema, sample_payload, metadata, active, version, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING tenant_id, name, version, description, schema, sample_payload, created_at
		)
		INSERT INTO event_registration_versions (tenant_id, name, version, description, schema, sample_payload, created_at)
		SELECT tenant_id, name, version, description, schema, sample_payload, created_at FROM reg
	`
	_, err := r.conn.ExecContext(ctx, query,
		event.TenantID,
		event.Name,
		event.Description,
		event.Schema,
		event.SamplePayload,
		event.Metadata,
		event.Active,
		event.Version,
		event.CreatedAt,
		event.UpdatedAt,
	)
	return storage.Error(err)
}

// GetEventByName gets an event registration by name within a tenant.
// Returns nil, nil when it does not exist.
func (r *Repository) GetEventByName(ctx context.Context, tenantID uuid.UUID, eventName string) (*EventRegistration, error) {
	return r.getEventByName(ctx, tenantID, eventName, "")
}

// GetEventByNameForUpdate is GetEventByName with a row lock, for use inside a
// transaction that decides the next version. Returns nil, nil when the event
// type does not exist.
func (r *Repository) GetEventByNameForUpdate(ctx context.Context, tenantID uuid.UUID, eventName string) (*EventRegistration, error) {
	return r.getEventByName(ctx, tenantID, eventName, " FOR UPDATE")
}

func (r *Repository) getEventByName(ctx context.Context, tenantID uuid.UUID, eventName, lock string) (*EventRegistration, error) {
	query := `SELECT ` + eventRegistrationColumns + ` FROM event_registrations WHERE tenant_id = $1 AND name = $2` + lock
	var event EventRegistration
	err := r.conn.GetContext(ctx, &event, query, tenantID, eventName)
	if err != nil {
		if storage.IsNotFound(storage.Error(err)) {
			return nil, nil
		}
		return nil, storage.Error(err)
	}
	return &event, nil
}

// ListEvents returns all registered events for a tenant
func (r *Repository) ListEvents(ctx context.Context, tenantID uuid.UUID, activeOnly bool) ([]*EventRegistration, error) {
	events, _, err := r.ListEventsPaginated(ctx, tenantID, EventTypeFilter{ActiveOnly: activeOnly}, 1000, 0)
	return events, err
}

// SystemEventPrefix is the name prefix of Sparrow's own system event types
// (e.g. sparrow.webhook.health_changed).
const SystemEventPrefix = "sparrow."

// EventTypeFilter narrows ListEventsPaginated.
type EventTypeFilter struct {
	ActiveOnly bool
	// System, when set, keeps only Sparrow's system event types (true) or
	// only tenant event types (false). Nil keeps both.
	System *bool
}

// eventTypeFilterWhere matches EventTypeFilter with $2 = ActiveOnly and
// $3 = System. The prefix match is case-insensitive, like
// webhooks.IsReservedEventName.
const eventTypeFilterWhere = `
	tenant_id = $1
	AND ($2 IS FALSE OR active = true)
	AND ($3::boolean IS NULL OR (lower(name) LIKE '` + SystemEventPrefix + `%') = $3)`

// ListEventsPaginated returns registered events for a tenant with pagination
func (r *Repository) ListEventsPaginated(ctx context.Context, tenantID uuid.UUID, filter EventTypeFilter, limit, offset int) ([]*EventRegistration, int, error) {
	countQuery := `SELECT COUNT(*) FROM event_registrations WHERE ` + eventTypeFilterWhere
	var totalCount int
	err := r.conn.GetContext(ctx, &totalCount, countQuery, tenantID, filter.ActiveOnly, filter.System)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	query := `
		SELECT ` + eventRegistrationColumns + `
		FROM event_registrations
		WHERE ` + eventTypeFilterWhere + `
		ORDER BY name ASC
		LIMIT $4 OFFSET $5
	`
	var events []*EventRegistration
	err = r.conn.SelectContext(ctx, &events, query, tenantID, filter.ActiveOnly, filter.System, limit, offset)
	if err != nil {
		return nil, 0, storage.Error(err)
	}
	return events, totalCount, nil
}

// UpdateEvent overwrites the current definition of an event type, including
// its version number. It does not touch the version history; callers that
// create a new version pair it with AddEventTypeVersion in one transaction.
// Returns storage.ErrNotFound when no registration matches the tenant and name.
func (r *Repository) UpdateEvent(ctx context.Context, tenantID uuid.UUID, event *EventRegistration) error {
	if event.Version <= 0 {
		event.Version = 1
	}
	query := `
		UPDATE event_registrations
		SET description = $3, schema = $4, sample_payload = $5, metadata = $6, active = $7, version = $8, updated_at = NOW()
		WHERE tenant_id = $1 AND name = $2
	`

	res, err := r.conn.ExecContext(ctx, query,
		tenantID,
		event.Name,
		event.Description,
		event.Schema,
		event.SamplePayload,
		event.Metadata,
		event.Active,
		event.Version,
	)
	if err != nil {
		return storage.Error(err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return storage.Error(err)
	}
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// AddEventTypeVersion appends a version to an event type's history.
// Returns storage.ErrAlreadyExists if that version number is taken, which is
// the backstop against two writers claiming the same next version.
func (r *Repository) AddEventTypeVersion(ctx context.Context, tenantID uuid.UUID, version *EventRegistrationVersion) error {
	version.TenantID = tenantID
	if version.CreatedAt.IsZero() {
		version.CreatedAt = time.Now()
	}
	query := `
		INSERT INTO event_registration_versions (` + eventRegistrationVersionColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.conn.ExecContext(ctx, query,
		tenantID,
		version.Name,
		version.Version,
		version.Description,
		version.Schema,
		version.SamplePayload,
		version.SchemaDefinedAt,
		version.CreatedAt,
	)
	return storage.Error(err)
}

// FillInEventTypeVersion writes a first schema into an existing version that
// had none, and records when that happened. This is the only in-place change
// the history table allows. Returns storage.ErrNotFound if the version does
// not exist.
func (r *Repository) FillInEventTypeVersion(ctx context.Context, tenantID uuid.UUID, version *EventRegistrationVersion) error {
	if version.SchemaDefinedAt == nil {
		now := time.Now()
		version.SchemaDefinedAt = &now
	}
	query := `
		UPDATE event_registration_versions
		SET description = $4, schema = $5, sample_payload = $6, schema_defined_at = $7
		WHERE tenant_id = $1 AND name = $2 AND version = $3
	`
	res, err := r.conn.ExecContext(ctx, query,
		tenantID,
		version.Name,
		version.Version,
		version.Description,
		version.Schema,
		version.SamplePayload,
		version.SchemaDefinedAt,
	)
	if err != nil {
		return storage.Error(err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return storage.Error(err)
	}
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// ListEventTypeVersions returns every version of an event type, newest first.
func (r *Repository) ListEventTypeVersions(ctx context.Context, tenantID uuid.UUID, eventName string) ([]*EventRegistrationVersion, error) {
	query := `
		SELECT ` + eventRegistrationVersionSelect + `
		FROM event_registration_versions
		WHERE tenant_id = $1 AND name = $2
		ORDER BY version DESC
	`
	var versions []*EventRegistrationVersion
	if err := r.conn.SelectContext(ctx, &versions, query, tenantID, eventName); err != nil {
		return nil, storage.Error(err)
	}
	return versions, nil
}

// GetEventTypeVersion returns one version of an event type, or nil, nil when
// it does not exist.
func (r *Repository) GetEventTypeVersion(ctx context.Context, tenantID uuid.UUID, eventName string, version int) (*EventRegistrationVersion, error) {
	query := `
		SELECT ` + eventRegistrationVersionSelect + `
		FROM event_registration_versions
		WHERE tenant_id = $1 AND name = $2 AND version = $3
	`
	var v EventRegistrationVersion
	if err := r.conn.GetContext(ctx, &v, query, tenantID, eventName, version); err != nil {
		if storage.IsNotFound(storage.Error(err)) {
			return nil, nil
		}
		return nil, storage.Error(err)
	}
	return &v, nil
}
