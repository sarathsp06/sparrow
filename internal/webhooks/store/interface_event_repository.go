package store

import (
	"context"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// RegisterEvent registers a new event type
func (r *Repository) RegisterEvent(ctx context.Context, event *EventRegistration) error {
	event.ID = uuid.New()
	query := `
		INSERT INTO event_registrations (
			id, name, description, schema, metadata, active
		) VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		event.ID,
		event.Name,
		event.Description,
		event.Schema,
		event.Metadata,
		event.Active,
	)
	return storage.Error(err)
}

// GetEventByName gets an event registration by name
func (r *Repository) GetEventByName(ctx context.Context, eventName string) (*EventRegistration, error) {
	query := `
		SELECT id, name, description, schema, metadata, active, created_at, updated_at
		FROM event_registrations 
		WHERE name = $1
	`
	var event EventRegistration
	err := r.db.GetContext(ctx, &event, query, eventName)
	if err != nil {
		if storage.IsNotFound(storage.Error(err)) {
			return nil, nil
		}
		return nil, storage.Error(err)
	}
	return &event, nil
}

// ListEvents returns all registered events
func (r *Repository) ListEvents(ctx context.Context, activeOnly bool) ([]*EventRegistration, error) {
	query := `
		SELECT id, name, description, schema, metadata, active, created_at, updated_at
		FROM event_registrations
		WHERE ($1 IS FALSE OR active = true)
		ORDER BY name ASC
	`
	var events []*EventRegistration
	err := r.db.SelectContext(ctx, &events, query, activeOnly)
	if err != nil {
		return nil, storage.Error(err)
	}
	return events, nil
}

// UpdateEvent updates an event registration
func (r *Repository) UpdateEvent(ctx context.Context, event *EventRegistration) error {
	query := `
		UPDATE event_registrations 
		SET description = $2, schema = $3, metadata = $4, active = $5
		WHERE name = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		event.Name,
		event.Description,
		event.Schema,
		event.Metadata,
		event.Active,
	)
	return storage.Error(err)
}

// DeleteEvent deletes an event registration
func (r *Repository) DeleteEvent(ctx context.Context, eventName string) error {
	query := `DELETE FROM event_registrations WHERE name = $1`
	_, err := r.db.ExecContext(ctx, query, eventName)
	return storage.Error(err)
}
