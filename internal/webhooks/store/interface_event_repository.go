package store

import (
	"context"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// RegisterEvent registers a new event type within a tenant
func (r *Repository) RegisterEvent(ctx context.Context, tenantID uuid.UUID, event *EventRegistration) error {
	event.TenantID = tenantID
	query := `
		INSERT INTO event_registrations (
			tenant_id, name, description, schema, sample_payload, metadata, active
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.conn.ExecContext(ctx, query,
		event.TenantID,
		event.Name,
		event.Description,
		event.Schema,
		event.SamplePayload,
		event.Metadata,
		event.Active,
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
	var events []*EventRegistration
	err = r.conn.SelectContext(ctx, &events, query, tenantID, activeOnly, limit, offset)
	if err != nil {
		return nil, 0, storage.Error(err)
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
