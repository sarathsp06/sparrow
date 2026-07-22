package store

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

type eventRegistrationRow struct {
	TenantID      uuid.UUID `db:"tenant_id"`
	Name          string    `db:"name"`
	Description   string    `db:"description"`
	Schema        []byte    `db:"schema"`
	SamplePayload []byte    `db:"sample_payload"`
	Metadata      []byte    `db:"metadata"`
	Active        bool      `db:"active"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

func decodeJSONMap(raw []byte) (map[string]any, error) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func decodeJSONStringMap(raw []byte) (map[string]string, error) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, nil
	}

	var decoded map[string]string
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func buildEventRegistration(row eventRegistrationRow) (*EventRegistration, error) {
	schema, err := decodeJSONMap(row.Schema)
	if err != nil {
		return nil, err
	}

	samplePayload, err := decodeJSONMap(row.SamplePayload)
	if err != nil {
		return nil, err
	}

	metadata, err := decodeJSONStringMap(row.Metadata)
	if err != nil {
		return nil, err
	}

	return &EventRegistration{
		TenantID:      row.TenantID,
		Name:          row.Name,
		Description:   row.Description,
		Schema:        schema,
		SamplePayload: samplePayload,
		Metadata:      metadata,
		Active:        row.Active,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}, nil
}

// RegisterEvent registers a new event type within a tenant
func (r *Repository) RegisterEvent(ctx context.Context, tenantID uuid.UUID, event *EventRegistration) error {
	event.TenantID = tenantID
	now := time.Now()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	event.UpdatedAt = now

	query := `
		INSERT INTO event_registrations (
			tenant_id, name, description, schema, sample_payload, metadata, active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.conn.ExecContext(ctx, query,
		event.TenantID,
		event.Name,
		event.Description,
		event.Schema,
		event.SamplePayload,
		event.Metadata,
		event.Active,
		event.CreatedAt,
		event.UpdatedAt,
	)
	return storage.Error(err)
}

// GetEventByName gets an event registration by name within a tenant
func (r *Repository) GetEventByName(ctx context.Context, tenantID uuid.UUID, eventName string) (*EventRegistration, error) {
	query := `
		SELECT tenant_id, name, description, schema, sample_payload, metadata, active, created_at, updated_at
		FROM event_registrations 
		WHERE tenant_id = $1 AND name = $2
	`
	var row eventRegistrationRow
	err := r.conn.GetContext(ctx, &row, query, tenantID, eventName)
	if err != nil {
		if storage.IsNotFound(storage.Error(err)) {
			return nil, nil
		}
		return nil, storage.Error(err)
	}
	event, err := buildEventRegistration(row)
	if err != nil {
		return nil, storage.Error(err)
	}
	return event, nil
}

// ListEvents returns all registered events for a tenant
func (r *Repository) ListEvents(ctx context.Context, tenantID uuid.UUID, activeOnly bool) ([]*EventRegistration, error) {
	events, _, err := r.ListEventsPaginated(ctx, tenantID, activeOnly, 1000, 0)
	return events, err
}

// ListEventsPaginated returns registered events for a tenant with pagination
func (r *Repository) ListEventsPaginated(ctx context.Context, tenantID uuid.UUID, activeOnly bool, limit, offset int) ([]*EventRegistration, int, error) {
	countQuery := `SELECT COUNT(*) FROM event_registrations WHERE tenant_id = $1 AND ($2 IS FALSE OR active = true)`
	var totalCount int
	err := r.conn.GetContext(ctx, &totalCount, countQuery, tenantID, activeOnly)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	query := `
		SELECT tenant_id, name, description, schema, sample_payload, metadata, active, created_at, updated_at
		FROM event_registrations
		WHERE tenant_id = $1 AND ($2 IS FALSE OR active = true)
		ORDER BY name ASC
		LIMIT $3 OFFSET $4
	`
	var rows []eventRegistrationRow
	err = r.conn.SelectContext(ctx, &rows, query, tenantID, activeOnly, limit, offset)
	if err != nil {
		return nil, 0, storage.Error(err)
	}
	events := make([]*EventRegistration, 0, len(rows))
	for _, row := range rows {
		event, buildErr := buildEventRegistration(row)
		if buildErr != nil {
			return nil, 0, storage.Error(buildErr)
		}
		events = append(events, event)
	}
	return events, totalCount, nil
}

// UpdateEvent updates an event registration within a tenant
func (r *Repository) UpdateEvent(ctx context.Context, tenantID uuid.UUID, event *EventRegistration) error {
	query := `
		UPDATE event_registrations 
		SET description = $3, schema = $4, sample_payload = $5, metadata = $6, active = $7
		WHERE tenant_id = $1 AND name = $2
	`

	_, err := r.conn.ExecContext(ctx, query,
		tenantID,
		event.Name,
		event.Description,
		event.Schema,
		event.SamplePayload,
		event.Metadata,
		event.Active,
	)
	return storage.Error(err)
}

// DeleteEvent deletes an event registration within a tenant
func (r *Repository) DeleteEvent(ctx context.Context, tenantID uuid.UUID, eventName string) error {
	query := `DELETE FROM event_registrations WHERE tenant_id = $1 AND name = $2`
	_, err := r.conn.ExecContext(ctx, query, tenantID, eventName)
	return storage.Error(err)
}
