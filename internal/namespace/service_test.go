package namespace

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// ---- Test mocks ----

// mockNamespaceRepo implements the subset of Repository used by the service
// under test.
type mockNamespaceRepo struct {
	Repository // embed interface — unused methods panic with nil receiver

	namespaces map[string]*Namespace // key: "tenantID/name"
}

func newMockNamespaceRepo() *mockNamespaceRepo {
	return &mockNamespaceRepo{
		namespaces: make(map[string]*Namespace),
	}
}

func nsKey(tenantID uuid.UUID, name string) string {
	return tenantID.String() + "/" + name
}

func (r *mockNamespaceRepo) CreateNamespace(_ context.Context, ns *Namespace) error {
	key := nsKey(ns.TenantID, ns.Name)
	if _, exists := r.namespaces[key]; exists {
		return storage.ErrAlreadyExists
	}
	if ns.ID == uuid.Nil {
		ns.ID = uuid.New()
	}
	r.namespaces[key] = ns
	return nil
}

func (r *mockNamespaceRepo) GetNamespaceByName(_ context.Context, tenantID uuid.UUID, name string) (*Namespace, error) {
	ns, ok := r.namespaces[nsKey(tenantID, name)]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return ns, nil
}

func (r *mockNamespaceRepo) GetNamespaceByID(_ context.Context, id uuid.UUID) (*Namespace, error) {
	for _, ns := range r.namespaces {
		if ns.ID == id {
			return ns, nil
		}
	}
	return nil, storage.ErrNotFound
}

func (r *mockNamespaceRepo) addNamespace(tenantID uuid.UUID, name string) *Namespace {
	ns := &Namespace{
		ID:       uuid.New(),
		TenantID: tenantID,
		Name:     name,
	}
	r.namespaces[nsKey(tenantID, name)] = ns
	return ns
}

// ---- Tests ----

func TestService_CreateNamespace(t *testing.T) {
	tenantID := uuid.New()
	repo := newMockNamespaceRepo()
	svc := NewService(repo)

	ns, err := svc.CreateNamespace(context.Background(), CreateNamespaceRequest{
		TenantID:    tenantID,
		Name:        "test-ns",
		Description: "A test namespace",
	})
	require.NoError(t, err)
	assert.Equal(t, "test-ns", ns.Name)
	assert.Equal(t, tenantID, ns.TenantID)
}

func TestService_CreateNamespace_InvalidName(t *testing.T) {
	svc := NewService(newMockNamespaceRepo())

	_, err := svc.CreateNamespace(context.Background(), CreateNamespaceRequest{
		TenantID: uuid.New(),
		Name:     "INVALID NAME",
	})
	assert.Error(t, err)
}

func TestService_CreateNamespace_EmptyName(t *testing.T) {
	svc := NewService(newMockNamespaceRepo())

	_, err := svc.CreateNamespace(context.Background(), CreateNamespaceRequest{
		TenantID: uuid.New(),
		Name:     "",
	})
	assert.Error(t, err)
}

func TestService_CreateNamespace_Duplicate(t *testing.T) {
	tenantID := uuid.New()
	repo := newMockNamespaceRepo()
	repo.addNamespace(tenantID, "existing")
	svc := NewService(repo)

	_, err := svc.CreateNamespace(context.Background(), CreateNamespaceRequest{
		TenantID: tenantID,
		Name:     "existing",
	})
	assert.Error(t, err)
}
