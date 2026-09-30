package webhooks

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
	"github.com/sarathsp06/sparrow/pkg/storage"
)

// fakeEventTypeRepo is an in-memory event type store: enough of
// RepositoryInterface for saveEventType, with head rows and version history.
type fakeEventTypeRepo struct {
	store.RepositoryInterface
	heads    map[string]*store.EventRegistration
	versions map[string][]*store.EventRegistrationVersion
}

func newFakeEventTypeRepo() *fakeEventTypeRepo {
	return &fakeEventTypeRepo{
		heads:    map[string]*store.EventRegistration{},
		versions: map[string][]*store.EventRegistrationVersion{},
	}
}

func (f *fakeEventTypeRepo) RunInTransaction(fn func(store.RepositoryInterface) error) error {
	return fn(f)
}

func (f *fakeEventTypeRepo) GetEventByName(_ context.Context, _ uuid.UUID, name string) (*store.EventRegistration, error) {
	h, ok := f.heads[name]
	if !ok {
		return nil, nil
	}
	cp := *h
	return &cp, nil
}

func (f *fakeEventTypeRepo) GetEventByNameForUpdate(ctx context.Context, tenantID uuid.UUID, name string) (*store.EventRegistration, error) {
	return f.GetEventByName(ctx, tenantID, name)
}

func (f *fakeEventTypeRepo) RegisterEvent(_ context.Context, tenantID uuid.UUID, e *store.EventRegistration) error {
	if _, ok := f.heads[e.Name]; ok {
		return storage.ErrAlreadyExists
	}
	if e.Version == 0 {
		e.Version = 1
	}
	e.TenantID = tenantID
	e.CreatedAt, e.UpdatedAt = time.Now(), time.Now()
	cp := *e
	f.heads[e.Name] = &cp
	f.versions[e.Name] = append(f.versions[e.Name], &store.EventRegistrationVersion{
		Name: e.Name, Version: e.Version, Description: e.Description, Schema: e.Schema, SamplePayload: e.SamplePayload,
	})
	return nil
}

func (f *fakeEventTypeRepo) UpdateEvent(_ context.Context, _ uuid.UUID, e *store.EventRegistration) error {
	if _, ok := f.heads[e.Name]; !ok {
		return storage.ErrNotFound
	}
	cp := *e
	f.heads[e.Name] = &cp
	return nil
}

func (f *fakeEventTypeRepo) AddEventTypeVersion(_ context.Context, _ uuid.UUID, v *store.EventRegistrationVersion) error {
	for _, existing := range f.versions[v.Name] {
		if existing.Version == v.Version {
			return storage.ErrAlreadyExists
		}
	}
	cp := *v
	f.versions[v.Name] = append(f.versions[v.Name], &cp)
	return nil
}

func (f *fakeEventTypeRepo) FillInEventTypeVersion(_ context.Context, _ uuid.UUID, v *store.EventRegistrationVersion) error {
	for _, existing := range f.versions[v.Name] {
		if existing.Version == v.Version {
			existing.Schema, existing.SamplePayload, existing.Description = v.Schema, v.SamplePayload, v.Description
			now := time.Now()
			existing.SchemaDefinedAt = &now
			return nil
		}
	}
	return storage.ErrNotFound
}

func orderSchema(total string) map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []any{"order_id", "total"},
		"properties": map[string]any{
			"order_id": map[string]any{"type": "string"},
			"total":    map[string]any{"type": total},
		},
	}
}

func TestPlanEventTypeSave(t *testing.T) {
	base := &store.EventRegistration{
		Name:        "order.created",
		Description: "An order",
		Schema:      orderSchema("number"),
		Metadata:    map[string]string{"owner": "payments"},
		Active:      true,
		Version:     3,
	}
	def := func(mut func(*EventTypeDefinition)) EventTypeDefinition {
		d := EventTypeDefinition{
			Name:        base.Name,
			Description: base.Description,
			Schema:      orderSchema("number"),
			Metadata:    map[string]string{"owner": "payments"},
			Active:      true,
		}
		if mut != nil {
			mut(&d)
		}
		return d
	}

	tests := []struct {
		name         string
		current      *store.EventRegistration
		in           EventTypeDefinition
		action       EventTypeSaveAction
		version      int
		changes      []string
		activeChange string
	}{
		{name: "new type starts at v1", current: nil, in: def(nil), action: EventTypeCreated, version: 1},
		{name: "identical definition is unchanged", current: base, in: def(nil), action: EventTypeUnchanged, version: 3},
		{
			name:    "equal schema built in a different order is unchanged",
			current: base,
			in: def(func(d *EventTypeDefinition) {
				d.Schema = map[string]any{
					"properties": map[string]any{"total": map[string]any{"type": "number"}, "order_id": map[string]any{"type": "string"}},
					"required":   []any{"order_id", "total"},
					"type":       "object",
				}
			}),
			action: EventTypeUnchanged, version: 3,
		},
		{
			name: "schema change creates the next version", current: base,
			in:     def(func(d *EventTypeDefinition) { d.Schema = orderSchema("string") }),
			action: EventTypeNewVersion, version: 4, changes: []string{ChangeSchema},
		},
		{
			name: "removing the schema creates the next version", current: base,
			in:     def(func(d *EventTypeDefinition) { d.Schema = nil }),
			action: EventTypeNewVersion, version: 4, changes: []string{ChangeSchema},
		},
		{
			name: "description change is in place", current: base,
			in:     def(func(d *EventTypeDefinition) { d.Description = "A placed order" }),
			action: EventTypeUpdated, version: 3, changes: []string{ChangeDescription},
		},
		{
			name: "deactivating is in place", current: base,
			in:     def(func(d *EventTypeDefinition) { d.Active = false }),
			action: EventTypeUpdated, version: 3, changes: []string{ChangeActive}, activeChange: "deactivates",
		},
		{
			name: "metadata change is in place", current: base,
			in:     def(func(d *EventTypeDefinition) { d.Metadata = map[string]string{"owner": "orders"} }),
			action: EventTypeUpdated, version: 3, changes: []string{ChangeMetadata},
		},
		{
			name:    "nil and empty metadata are equal",
			current: &store.EventRegistration{Name: "x", Active: true, Version: 1, Metadata: map[string]string{}},
			in:      EventTypeDefinition{Name: "x", Active: true},
			action:  EventTypeUnchanged, version: 1,
		},
		{
			name:    "first schema on a schema-less version fills it in without a new version",
			current: &store.EventRegistration{Name: "x", Active: true, Version: 1},
			in:      EventTypeDefinition{Name: "x", Active: true, Schema: orderSchema("number"), Description: "now documented"},
			action:  EventTypeUpdated, version: 1, changes: []string{ChangeSchemaDefined, ChangeDescription},
		},
		{
			name: "schema and description change together is one new version", current: base,
			in: def(func(d *EventTypeDefinition) {
				d.Schema = orderSchema("integer")
				d.Description = "changed"
			}),
			action: EventTypeNewVersion, version: 4, changes: []string{ChangeSchema, ChangeDescription},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := planEventTypeSave(tt.current, tt.in)
			assert.Equal(t, tt.action, got.Action)
			assert.Equal(t, tt.version, got.Version)
			assert.Equal(t, tt.changes, got.Changes)
			assert.Equal(t, tt.activeChange, got.ActiveChange)
		})
	}
}

func TestIsReservedEventName(t *testing.T) {
	for name, want := range map[string]bool{
		"sparrow.webhook.health_changed": true,
		"Sparrow.x":                      true,
		"SPARROW.":                       true,
		"sparrowish.event":               false,
		"order.sparrow.created":          false,
		"sparrow":                        false,
	} {
		assert.Equal(t, want, IsReservedEventName(name), name)
	}
}

func TestSaveEventType_VersionLifecycle(t *testing.T) {
	repo := newFakeEventTypeRepo()
	svc := NewWebhookService(nil, repo, nil)
	ctx := testContext()

	res, err := svc.saveEventType(ctx, EventTypeDefinition{Name: "order.created", Active: true}, saveUpsert)
	require.NoError(t, err)
	assert.Equal(t, EventTypeCreated, res.Action)
	assert.Equal(t, 1, res.Event.Version)

	// A first schema fills in v1 rather than creating v2.
	res, err = svc.saveEventType(ctx, EventTypeDefinition{Name: "order.created", Active: true, Schema: orderSchema("number")}, saveUpsert)
	require.NoError(t, err)
	assert.Equal(t, EventTypeUpdated, res.Action)
	assert.Equal(t, []string{ChangeSchemaDefined}, res.Changes)
	require.Len(t, repo.versions["order.created"], 1)
	v1 := repo.versions["order.created"][0]
	assert.NotNil(t, v1.SchemaDefinedAt)
	assert.Equal(t, "number", v1.Schema["properties"].(map[string]any)["total"].(map[string]any)["type"])
	assert.NotEmpty(t, res.Event.SamplePayload, "sample payload is generated for the filled-in schema")

	// A description change stays on v1 and keeps the sample.
	sample := res.Event.SamplePayload
	res, err = svc.saveEventType(ctx, EventTypeDefinition{Name: "order.created", Active: true, Schema: orderSchema("number"), Description: "An order"}, saveUpsert)
	require.NoError(t, err)
	assert.Equal(t, EventTypeUpdated, res.Action)
	assert.Equal(t, 1, res.Event.Version)
	assert.Equal(t, sample, res.Event.SamplePayload)

	// A schema change creates v2 and keeps v1.
	res, err = svc.saveEventType(ctx, EventTypeDefinition{Name: "order.created", Active: true, Schema: orderSchema("string"), Description: "An order"}, saveUpsert)
	require.NoError(t, err)
	assert.Equal(t, EventTypeNewVersion, res.Action)
	assert.Equal(t, 2, res.Event.Version)
	assert.Equal(t, 1, res.PreviousVersion)
	require.Len(t, repo.versions["order.created"], 2)
	assert.Equal(t, "number", repo.versions["order.created"][0].Schema["properties"].(map[string]any)["total"].(map[string]any)["type"], "v1 is kept as it was")

	// Saving the same thing again writes nothing.
	res, err = svc.saveEventType(ctx, EventTypeDefinition{Name: "order.created", Active: true, Schema: orderSchema("string"), Description: "An order"}, saveUpsert)
	require.NoError(t, err)
	assert.Equal(t, EventTypeUnchanged, res.Action)
	assert.Len(t, repo.versions["order.created"], 2)
}

func TestSaveEventType_Modes(t *testing.T) {
	ctx := testContext()

	t.Run("create-only rejects an existing name", func(t *testing.T) {
		repo := newFakeEventTypeRepo()
		svc := NewWebhookService(nil, repo, nil)
		_, err := svc.saveEventType(ctx, EventTypeDefinition{Name: "a.b", Active: true}, saveCreateOnly)
		require.NoError(t, err)
		_, err = svc.saveEventType(ctx, EventTypeDefinition{Name: "a.b", Active: true}, saveCreateOnly)
		assertStatus(t, err, svcerrors.AlreadyExists)
	})

	t.Run("must-exist rejects an unknown name", func(t *testing.T) {
		svc := NewWebhookService(nil, newFakeEventTypeRepo(), nil)
		_, err := svc.saveEventType(ctx, EventTypeDefinition{Name: "a.b", Active: true}, saveMustExist)
		assertStatus(t, err, svcerrors.NotFound)
	})

	t.Run("reserved and invalid names are rejected before storage", func(t *testing.T) {
		repo := newFakeEventTypeRepo()
		svc := NewWebhookService(nil, repo, nil)
		for _, name := range []string{"sparrow.webhook.health_changed", "Sparrow.custom", ""} {
			_, err := svc.saveEventType(ctx, EventTypeDefinition{Name: name, Active: true}, saveUpsert)
			assertStatus(t, err, svcerrors.InvalidArgument)
		}
		assert.Empty(t, repo.heads)
	})

	t.Run("upsert retries once when a concurrent writer created the type first", func(t *testing.T) {
		repo := &racingRepo{
			fakeEventTypeRepo:     newFakeEventTypeRepo(),
			createOnFirstRegister: &store.EventRegistration{Name: "a.b", Active: true, Version: 1},
		}
		svc := NewWebhookService(nil, repo, nil)
		res, err := svc.saveEventType(ctx, EventTypeDefinition{Name: "a.b", Active: true, Description: "mine"}, saveUpsert)
		require.NoError(t, err)
		assert.Equal(t, EventTypeUpdated, res.Action, "the retry applies the change on top of the row that won")
		assert.Equal(t, "mine", res.Event.Description)
	})
}

// racingRepo makes the first RegisterEvent lose a race: another writer's row
// appears and the insert fails with a unique violation.
type racingRepo struct {
	*fakeEventTypeRepo
	createOnFirstRegister *store.EventRegistration
}

func (r *racingRepo) RunInTransaction(fn func(store.RepositoryInterface) error) error {
	return fn(r)
}

func (r *racingRepo) RegisterEvent(ctx context.Context, tenantID uuid.UUID, e *store.EventRegistration) error {
	if r.createOnFirstRegister != nil {
		other := r.createOnFirstRegister
		r.createOnFirstRegister = nil
		_ = r.fakeEventTypeRepo.RegisterEvent(ctx, tenantID, other)
		return storage.ErrAlreadyExists
	}
	return r.fakeEventTypeRepo.RegisterEvent(ctx, tenantID, e)
}

func assertStatus(t *testing.T, err error, want svcerrors.Status) {
	t.Helper()
	require.Error(t, err)
	assert.Equal(t, want, svcerrors.Classify(err, "").Status, err.Error())
}

func TestPushEvent_UnknownEventTypeIsRejectedByDefault(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	svc := NewWebhookService(inserter, repo, nil)

	repo.On("GetEventByName", mock.Anything, mock.Anything, "user.signup").Return(nil, nil)

	_, _, _, _, err := svc.PushEvent(testContext(), "default", "user.signup", map[string]any{}, 0, nil, nil, nil)
	assertStatus(t, err, svcerrors.NotFound)
	assert.Contains(t, err.Error(), "not registered")
	repo.AssertNotCalled(t, "RegisterEvent", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "StoreEvent", mock.Anything, mock.Anything, mock.Anything)
}

func TestPushEvent_ReservedNameIsRejected(t *testing.T) {
	repo := new(mockRepo)
	svc := NewWebhookService(new(mockJobInserter), repo, nil, WithAutoRegisterEvents(true))

	for _, name := range []string{"sparrow.webhook.health_changed", "SPARROW.fake"} {
		_, _, _, _, err := svc.PushEvent(testContext(), "default", name, map[string]any{}, 0, nil, nil, nil)
		assertStatus(t, err, svcerrors.InvalidArgument)
	}
	repo.AssertNotCalled(t, "GetEventByName", mock.Anything, mock.Anything, mock.Anything)
}

func TestPushEvent_PinsTheCurrentVersion(t *testing.T) {
	repo := new(mockRepo)
	inserter := new(mockJobInserter)
	svc := NewWebhookService(inserter, repo, nil)

	repo.On("GetEventByName", mock.Anything, mock.Anything, "order.created").
		Return(&store.EventRegistration{Name: "order.created", Active: true, Version: 3}, nil)
	repo.On("StoreEvent", mock.Anything, mock.Anything, mock.MatchedBy(func(e *store.EventRecord) bool {
		return e.EventVersion == 3
	})).Return(nil)
	inserter.On("Insert", mock.Anything, mock.Anything).Return(&rivertype.JobInsertResult{}, nil)

	_, _, _, _, err := svc.PushEvent(testContext(), "default", "order.created", map[string]any{}, 0, nil, nil, nil)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}
