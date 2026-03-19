package namespace

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/internal/auth"
	"github.com/sarathsp06/sparrow/pkg/storage"
)

// ---- Test mocks ----

// mockNamespaceRepo implements the subset of Repository used by the service
// under test. Methods not listed here will panic if called (indicates a test
// problem).
type mockNamespaceRepo struct {
	Repository // embed interface — unused methods panic with nil receiver

	mu          sync.Mutex
	namespaces  map[string]*Namespace  // key: "tenantID/name"
	memberships map[string]*Membership // key: "tenantID/subjectID/namespace"
}

func newMockNamespaceRepo() *mockNamespaceRepo {
	return &mockNamespaceRepo{
		namespaces:  make(map[string]*Namespace),
		memberships: make(map[string]*Membership),
	}
}

func nsKey(tenantID uuid.UUID, name string) string {
	return tenantID.String() + "/" + name
}

func memKey(tenantID uuid.UUID, subjectID, namespace string) string {
	return tenantID.String() + "/" + subjectID + "/" + namespace
}

func (r *mockNamespaceRepo) GetNamespaceByName(_ context.Context, tenantID uuid.UUID, name string) (*Namespace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ns, ok := r.namespaces[nsKey(tenantID, name)]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return ns, nil
}

func (r *mockNamespaceRepo) UpsertMembership(_ context.Context, m *Membership) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now
	r.memberships[memKey(m.TenantID, m.SubjectID, m.Namespace)] = m
	return nil
}

func (r *mockNamespaceRepo) DeleteMembership(_ context.Context, tenantID uuid.UUID, subjectID, namespace string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.memberships, memKey(tenantID, subjectID, namespace))
	return nil
}

func (r *mockNamespaceRepo) ListNamespacesBySubject(_ context.Context, tenantID uuid.UUID, subjectID string) ([]*Membership, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*Membership
	prefix := tenantID.String() + "/" + subjectID + "/"
	for k, m := range r.memberships {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			result = append(result, m)
		}
	}
	return result, nil
}

func (r *mockNamespaceRepo) addNamespace(tenantID uuid.UUID, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.namespaces[nsKey(tenantID, name)] = &Namespace{
		ID:       uuid.New(),
		TenantID: tenantID,
		Name:     name,
	}
}

// mockIdentityProvider captures SyncNamespaceRoles calls and signals completion.
type mockIdentityProvider struct {
	mu    sync.Mutex
	calls []syncCall
	err   error
	done  chan struct{}
	once  sync.Once
}

type syncCall struct {
	ExternalTenantID string
	SubjectID        string
	Roles            map[string]auth.Role
}

func newMockIdentityProvider() *mockIdentityProvider {
	return &mockIdentityProvider{
		done: make(chan struct{}),
	}
}

func (m *mockIdentityProvider) SyncNamespaceRoles(_ context.Context, extID, subID string, roles map[string]auth.Role) error {
	m.mu.Lock()
	m.calls = append(m.calls, syncCall{extID, subID, roles})
	m.mu.Unlock()
	m.once.Do(func() { close(m.done) })
	return m.err
}

func (m *mockIdentityProvider) TeamManagement() auth.TeamManager {
	return nil
}

func (m *mockIdentityProvider) waitForSync(t *testing.T) {
	t.Helper()
	select {
	case <-m.done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for identity provider sync")
	}
}

func (m *mockIdentityProvider) getCalls() []syncCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]syncCall, len(m.calls))
	copy(out, m.calls)
	return out
}

// mockExternalTenantLookup returns configurable values.
type mockExternalTenantLookup struct {
	externalID string
	err        error
}

func (m *mockExternalTenantLookup) LookupExternalIDByTenantID(_ context.Context, _ uuid.UUID) (string, error) {
	return m.externalID, m.err
}

// ---- Tests ----

func TestService_AssignNamespaceRole_SyncsToProvider(t *testing.T) {
	tenantID := uuid.New()
	repo := newMockNamespaceRepo()
	repo.addNamespace(tenantID, "ns-a")

	provider := newMockIdentityProvider()
	lookup := &mockExternalTenantLookup{externalID: "org_ext_123"}

	svc := NewService(repo,
		WithIdentityProvider(provider),
		WithExternalTenantLookup(lookup),
	)

	_, err := svc.AssignNamespaceRole(context.Background(), AssignMembershipRequest{
		TenantID:  tenantID,
		SubjectID: "user_1",
		Namespace: "ns-a",
		Role:      auth.RoleNamespaceAdmin,
	})
	require.NoError(t, err)

	// Wait for async sync goroutine
	provider.waitForSync(t)

	calls := provider.getCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "org_ext_123", calls[0].ExternalTenantID)
	assert.Equal(t, "user_1", calls[0].SubjectID)
	assert.Equal(t, auth.RoleNamespaceAdmin, calls[0].Roles["ns-a"])
}

func TestService_RemoveNamespaceRole_SyncsToProvider(t *testing.T) {
	tenantID := uuid.New()
	repo := newMockNamespaceRepo()
	repo.addNamespace(tenantID, "ns-a")
	repo.addNamespace(tenantID, "ns-b")

	// Pre-populate two memberships
	repo.memberships[memKey(tenantID, "user_1", "ns-a")] = &Membership{
		ID: uuid.New(), TenantID: tenantID, SubjectID: "user_1",
		Namespace: "ns-a", Role: auth.RoleNamespaceAdmin,
	}
	repo.memberships[memKey(tenantID, "user_1", "ns-b")] = &Membership{
		ID: uuid.New(), TenantID: tenantID, SubjectID: "user_1",
		Namespace: "ns-b", Role: auth.RoleNamespaceViewer,
	}

	provider := newMockIdentityProvider()
	lookup := &mockExternalTenantLookup{externalID: "org_ext_123"}

	svc := NewService(repo,
		WithIdentityProvider(provider),
		WithExternalTenantLookup(lookup),
	)

	// Remove ns-a membership
	err := svc.RemoveNamespaceRole(context.Background(), tenantID, "user_1", "ns-a")
	require.NoError(t, err)

	provider.waitForSync(t)

	calls := provider.getCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "user_1", calls[0].SubjectID)

	// Should only have ns-b remaining
	require.Len(t, calls[0].Roles, 1)
	assert.Equal(t, auth.RoleNamespaceViewer, calls[0].Roles["ns-b"])
}

func TestService_SyncNoProvider(t *testing.T) {
	tenantID := uuid.New()
	repo := newMockNamespaceRepo()
	repo.addNamespace(tenantID, "ns-a")

	// No identity provider configured
	svc := NewService(repo)

	_, err := svc.AssignNamespaceRole(context.Background(), AssignMembershipRequest{
		TenantID:  tenantID,
		SubjectID: "user_1",
		Namespace: "ns-a",
		Role:      auth.RoleNamespaceAdmin,
	})
	require.NoError(t, err, "should succeed without provider")

	// Give goroutine a moment to run (it shouldn't, but verify no panic)
	time.Sleep(50 * time.Millisecond)
}

func TestService_SyncNoTenantLookup(t *testing.T) {
	tenantID := uuid.New()
	repo := newMockNamespaceRepo()
	repo.addNamespace(tenantID, "ns-a")

	provider := newMockIdentityProvider()
	// No tenant lookup configured — only provider, no lookup
	svc := NewService(repo,
		WithIdentityProvider(provider),
	)

	_, err := svc.AssignNamespaceRole(context.Background(), AssignMembershipRequest{
		TenantID:  tenantID,
		SubjectID: "user_1",
		Namespace: "ns-a",
		Role:      auth.RoleNamespaceAdmin,
	})
	require.NoError(t, err, "should succeed without tenant lookup")

	// Give goroutine a moment — syncRolesToProvider should return immediately
	time.Sleep(50 * time.Millisecond)

	calls := provider.getCalls()
	assert.Len(t, calls, 0, "no sync should happen without tenant lookup")
}

func TestService_SyncLookupFails(t *testing.T) {
	tenantID := uuid.New()
	repo := newMockNamespaceRepo()
	repo.addNamespace(tenantID, "ns-a")

	provider := newMockIdentityProvider()
	lookup := &mockExternalTenantLookup{err: fmt.Errorf("db connection failed")}

	svc := NewService(repo,
		WithIdentityProvider(provider),
		WithExternalTenantLookup(lookup),
	)

	_, err := svc.AssignNamespaceRole(context.Background(), AssignMembershipRequest{
		TenantID:  tenantID,
		SubjectID: "user_1",
		Namespace: "ns-a",
		Role:      auth.RoleNamespaceAdmin,
	})
	require.NoError(t, err, "assign should succeed even when lookup fails")

	// Give goroutine time to run and hit the error path
	time.Sleep(100 * time.Millisecond)

	calls := provider.getCalls()
	assert.Len(t, calls, 0, "provider should not be called when lookup fails")
}

func TestService_SyncEmptyExternalID(t *testing.T) {
	tenantID := uuid.New()
	repo := newMockNamespaceRepo()
	repo.addNamespace(tenantID, "ns-a")

	provider := newMockIdentityProvider()
	lookup := &mockExternalTenantLookup{externalID: ""} // empty = no external ID

	svc := NewService(repo,
		WithIdentityProvider(provider),
		WithExternalTenantLookup(lookup),
	)

	_, err := svc.AssignNamespaceRole(context.Background(), AssignMembershipRequest{
		TenantID:  tenantID,
		SubjectID: "user_1",
		Namespace: "ns-a",
		Role:      auth.RoleNamespaceAdmin,
	})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	calls := provider.getCalls()
	assert.Len(t, calls, 0, "provider should not be called when external ID is empty")
}

func TestService_SyncProviderError(t *testing.T) {
	tenantID := uuid.New()
	repo := newMockNamespaceRepo()
	repo.addNamespace(tenantID, "ns-a")

	provider := newMockIdentityProvider()
	provider.err = fmt.Errorf("clerk API unavailable")
	lookup := &mockExternalTenantLookup{externalID: "org_ext_123"}

	svc := NewService(repo,
		WithIdentityProvider(provider),
		WithExternalTenantLookup(lookup),
	)

	_, err := svc.AssignNamespaceRole(context.Background(), AssignMembershipRequest{
		TenantID:  tenantID,
		SubjectID: "user_1",
		Namespace: "ns-a",
		Role:      auth.RoleNamespaceAdmin,
	})
	require.NoError(t, err, "assign should succeed even when provider fails")

	// Wait for async sync to complete (provider will be called, then error logged)
	provider.waitForSync(t)

	calls := provider.getCalls()
	require.Len(t, calls, 1, "provider should still be called")
	assert.Equal(t, "org_ext_123", calls[0].ExternalTenantID)
}
