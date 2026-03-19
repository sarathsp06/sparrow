package auth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- ClerkIdentityProvider Tests ----

func TestClerkIdentityProvider_SyncNamespaceRoles(t *testing.T) {
	t.Run("successful sync", func(t *testing.T) {
		var (
			mu        sync.Mutex
			gotMethod string
			gotPath   string
			gotAuth   string
			gotCType  string
			gotBody   map[string]any
		)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			gotMethod = r.Method
			gotPath = r.URL.Path
			gotAuth = r.Header.Get("Authorization")
			gotCType = r.Header.Get("Content-Type")

			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &gotBody)

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_secret123",
			WithClerkBaseURL(srv.URL),
		)

		roles := map[string]Role{
			"customer-a": RoleNamespaceAdmin,
			"customer-b": RoleNamespaceViewer,
		}

		err := provider.SyncNamespaceRoles(context.Background(), "org_abc", "user_xyz", roles)
		require.NoError(t, err)

		mu.Lock()
		defer mu.Unlock()

		assert.Equal(t, http.MethodPatch, gotMethod)
		assert.Equal(t, "/organizations/org_abc/memberships/user_xyz/metadata", gotPath)
		assert.Equal(t, "Bearer sk_test_secret123", gotAuth)
		assert.Equal(t, "application/json", gotCType)

		// Verify body structure
		pubMeta, ok := gotBody["public_metadata"].(map[string]any)
		require.True(t, ok, "body should have public_metadata object")

		nsRoles, ok := pubMeta["namespace_roles"].([]any)
		require.True(t, ok, "public_metadata should have namespace_roles array")
		assert.Len(t, nsRoles, 2)

		// Roles should be sorted
		assert.Equal(t, "namespace:admin:customer-a", nsRoles[0])
		assert.Equal(t, "namespace:viewer:customer-b", nsRoles[1])
	})

	t.Run("empty roles sends empty array", func(t *testing.T) {
		var gotBody map[string]any

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &gotBody)
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key",
			WithClerkBaseURL(srv.URL),
		)

		err := provider.SyncNamespaceRoles(context.Background(), "org_abc", "user_xyz", map[string]Role{})
		require.NoError(t, err)

		pubMeta, ok := gotBody["public_metadata"].(map[string]any)
		require.True(t, ok)

		nsRoles, ok := pubMeta["namespace_roles"].([]any)
		require.True(t, ok, "namespace_roles should be an array, not null")
		assert.Len(t, nsRoles, 0, "empty roles should produce empty array")
	})

	t.Run("nil roles sends empty array", func(t *testing.T) {
		var gotBody map[string]any

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &gotBody)
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key",
			WithClerkBaseURL(srv.URL),
		)

		err := provider.SyncNamespaceRoles(context.Background(), "org_abc", "user_xyz", nil)
		require.NoError(t, err)

		pubMeta := gotBody["public_metadata"].(map[string]any)
		nsRoles, ok := pubMeta["namespace_roles"].([]any)
		require.True(t, ok, "namespace_roles should be an array, not null")
		assert.Len(t, nsRoles, 0)
	})

	t.Run("empty externalTenantID returns error", func(t *testing.T) {
		provider := NewClerkIdentityProvider("sk_test_key")

		err := provider.SyncNamespaceRoles(context.Background(), "", "user_xyz", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "externalTenantID")
	})

	t.Run("empty subjectID returns error", func(t *testing.T) {
		provider := NewClerkIdentityProvider("sk_test_key")

		err := provider.SyncNamespaceRoles(context.Background(), "org_abc", "", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "subjectID")
	})

	t.Run("non-2xx status returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"forbidden","message":"invalid API key"}`))
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_bad_key",
			WithClerkBaseURL(srv.URL),
		)

		err := provider.SyncNamespaceRoles(context.Background(), "org_abc", "user_xyz", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "403")
		assert.Contains(t, err.Error(), "invalid API key")
	})

	t.Run("HTTP request failure returns error", func(t *testing.T) {
		// Create and immediately close server so the URL is unreachable
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL
		srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key",
			WithClerkBaseURL(url),
		)

		err := provider.SyncNamespaceRoles(context.Background(), "org_abc", "user_xyz", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "request failed")
	})
}

// ---- encodeNamespaceRoles Tests ----

func TestEncodeNamespaceRoles(t *testing.T) {
	t.Run("multiple roles sorted", func(t *testing.T) {
		roles := map[string]Role{
			"customer-b": RoleNamespaceViewer,
			"customer-a": RoleNamespaceAdmin,
			"customer-c": RoleNamespaceMember,
		}

		encoded := encodeNamespaceRoles(roles)
		require.Len(t, encoded, 3)
		assert.Equal(t, "namespace:admin:customer-a", encoded[0])
		assert.Equal(t, "namespace:member:customer-c", encoded[1])
		assert.Equal(t, "namespace:viewer:customer-b", encoded[2])
	})

	t.Run("single role", func(t *testing.T) {
		roles := map[string]Role{
			"my-ns": RoleNamespaceAdmin,
		}

		encoded := encodeNamespaceRoles(roles)
		require.Len(t, encoded, 1)
		assert.Equal(t, "namespace:admin:my-ns", encoded[0])
	})

	t.Run("empty map returns empty slice", func(t *testing.T) {
		encoded := encodeNamespaceRoles(map[string]Role{})
		require.NotNil(t, encoded, "should return empty slice, not nil")
		assert.Len(t, encoded, 0)
	})

	t.Run("nil map returns empty slice", func(t *testing.T) {
		encoded := encodeNamespaceRoles(nil)
		require.NotNil(t, encoded, "should return empty slice, not nil")
		assert.Len(t, encoded, 0)
	})
}

// ---- DecodeNamespaceRoles Tests ----

func TestDecodeNamespaceRoles(t *testing.T) {
	t.Run("valid entries", func(t *testing.T) {
		encoded := []string{
			"namespace:admin:customer-a",
			"namespace:viewer:customer-b",
			"namespace:member:customer-c",
		}

		roles := DecodeNamespaceRoles(encoded)
		require.Len(t, roles, 3)
		assert.Equal(t, RoleNamespaceAdmin, roles["customer-a"])
		assert.Equal(t, RoleNamespaceViewer, roles["customer-b"])
		assert.Equal(t, RoleNamespaceMember, roles["customer-c"])
	})

	t.Run("invalid entries silently skipped", func(t *testing.T) {
		encoded := []string{
			"bad-entry",
			"namespace:admin:customer-a",
			"also:bad",
		}

		roles := DecodeNamespaceRoles(encoded)
		require.Len(t, roles, 1)
		assert.Equal(t, RoleNamespaceAdmin, roles["customer-a"])
	})

	t.Run("wrong prefix skipped", func(t *testing.T) {
		encoded := []string{
			"tenant:admin:customer-a",
			"other:viewer:customer-b",
		}

		roles := DecodeNamespaceRoles(encoded)
		assert.Empty(t, roles, "no valid entries should produce empty map")
	})

	t.Run("empty level skipped", func(t *testing.T) {
		encoded := []string{
			"namespace::customer-a",
		}

		roles := DecodeNamespaceRoles(encoded)
		assert.Empty(t, roles)
	})

	t.Run("empty name skipped", func(t *testing.T) {
		encoded := []string{
			"namespace:admin:",
		}

		roles := DecodeNamespaceRoles(encoded)
		assert.Empty(t, roles)
	})

	t.Run("invalid role level skipped", func(t *testing.T) {
		// "namespace:superadmin" is not a valid namespace role
		encoded := []string{
			"namespace:superadmin:customer-a",
		}

		roles := DecodeNamespaceRoles(encoded)
		assert.Empty(t, roles)
	})

	t.Run("empty input returns nil", func(t *testing.T) {
		roles := DecodeNamespaceRoles([]string{})
		assert.Nil(t, roles)
	})

	t.Run("nil input returns nil", func(t *testing.T) {
		roles := DecodeNamespaceRoles(nil)
		assert.Nil(t, roles)
	})
}

// ---- Roundtrip Test ----

func TestEncodeDecodeNamespaceRoles_Roundtrip(t *testing.T) {
	original := map[string]Role{
		"customer-a": RoleNamespaceAdmin,
		"customer-b": RoleNamespaceViewer,
		"customer-c": RoleNamespaceMember,
	}

	encoded := encodeNamespaceRoles(original)
	decoded := DecodeNamespaceRoles(encoded)

	require.Len(t, decoded, len(original))
	for ns, role := range original {
		assert.Equal(t, role, decoded[ns], "role mismatch for namespace %q", ns)
	}
}

// ---- TeamManagement() Tests ----

func TestClerkIdentityProvider_TeamManagement(t *testing.T) {
	provider := NewClerkIdentityProvider("sk_test_key")
	tm := provider.TeamManagement()
	assert.NotNil(t, tm, "Clerk provider should return a non-nil TeamManager")
	assert.Equal(t, provider, tm, "TeamManagement should return the provider itself")
}

func TestNoopIdentityProvider_TeamManagement(t *testing.T) {
	provider := NewNoopIdentityProvider(nil)
	tm := provider.TeamManagement()
	assert.Nil(t, tm, "Noop provider should return nil TeamManager")
}

// ---- TeamManager: ListMembers Tests ----

func TestClerkTeamManager_ListMembers(t *testing.T) {
	t.Run("successful list with members", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/organizations/org_abc/memberships", r.URL.Path)
			assert.Equal(t, "10", r.URL.Query().Get("limit"))
			assert.Equal(t, "0", r.URL.Query().Get("offset"))
			assert.Equal(t, "Bearer sk_test_key", r.Header.Get("Authorization"))

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":   "mem_1",
						"role": "org:admin",
						"public_user_data": map[string]any{
							"user_id":    "user_001",
							"first_name": "Alice",
							"last_name":  "Smith",
							"identifier": "alice@example.com",
							"image_url":  "https://img.example.com/alice.jpg",
							"has_image":  true,
						},
						"created_at": 1700000000000,
					},
					{
						"id":   "mem_2",
						"role": "org:member",
						"public_user_data": map[string]any{
							"user_id":    "user_002",
							"first_name": "Bob",
							"last_name":  "Jones",
							"identifier": "bob@example.com",
							"image_url":  "",
							"has_image":  false,
						},
						"created_at": 1700001000000,
					},
				},
				"total_count": 2,
			})
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key", WithClerkBaseURL(srv.URL))
		members, total, err := provider.ListMembers(context.Background(), "org_abc", 10, 0)

		require.NoError(t, err)
		assert.Equal(t, 2, total)
		require.Len(t, members, 2)

		assert.Equal(t, "user_001", members[0].UserID)
		assert.Equal(t, "Alice", members[0].FirstName)
		assert.Equal(t, "Smith", members[0].LastName)
		assert.Equal(t, "alice@example.com", members[0].Email)
		assert.Equal(t, "https://img.example.com/alice.jpg", members[0].ImageURL)
		assert.Equal(t, "org:admin", members[0].Role)
		assert.False(t, members[0].JoinedAt.IsZero())

		assert.Equal(t, "user_002", members[1].UserID)
		assert.Equal(t, "Bob", members[1].FirstName)
		assert.Equal(t, "org:member", members[1].Role)
	})

	t.Run("empty list", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data":        []any{},
				"total_count": 0,
			})
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key", WithClerkBaseURL(srv.URL))
		members, total, err := provider.ListMembers(context.Background(), "org_abc", 10, 0)

		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, members)
	})

	t.Run("API error returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"forbidden"}`))
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key", WithClerkBaseURL(srv.URL))
		_, _, err := provider.ListMembers(context.Background(), "org_abc", 10, 0)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "403")
	})
}

// ---- TeamManager: InviteMember Tests ----

func TestClerkTeamManager_InviteMember(t *testing.T) {
	t.Run("successful invite", func(t *testing.T) {
		var gotBody map[string]string

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/organizations/org_abc/invitations", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &gotBody)

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":            "inv_123",
				"email_address": "new@example.com",
				"role":          "org:member",
				"status":        "pending",
				"created_at":    1700000000000,
				"expires_at":    1700604800000,
			})
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key", WithClerkBaseURL(srv.URL))
		inv, err := provider.InviteMember(context.Background(), "org_abc", "new@example.com", "org:member")

		require.NoError(t, err)
		require.NotNil(t, inv)
		assert.Equal(t, "inv_123", inv.ID)
		assert.Equal(t, "new@example.com", inv.Email)
		assert.Equal(t, "org:member", inv.Role)
		assert.Equal(t, "pending", inv.Status)
		assert.False(t, inv.CreatedAt.IsZero())
		assert.False(t, inv.ExpiresAt.IsZero())

		// Verify request body
		assert.Equal(t, "new@example.com", gotBody["email_address"])
		assert.Equal(t, "org:member", gotBody["role"])
	})

	t.Run("API error returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"error":"already invited"}`))
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key", WithClerkBaseURL(srv.URL))
		_, err := provider.InviteMember(context.Background(), "org_abc", "dup@example.com", "org:member")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "422")
	})
}

// ---- TeamManager: RemoveMember Tests ----

func TestClerkTeamManager_RemoveMember(t *testing.T) {
	t.Run("successful remove", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Equal(t, "/organizations/org_abc/memberships/user_001", r.URL.Path)
			assert.Equal(t, "Bearer sk_test_key", r.Header.Get("Authorization"))

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key", WithClerkBaseURL(srv.URL))
		err := provider.RemoveMember(context.Background(), "org_abc", "user_001")

		require.NoError(t, err)
	})

	t.Run("API error returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key", WithClerkBaseURL(srv.URL))
		err := provider.RemoveMember(context.Background(), "org_abc", "user_999")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "404")
	})
}

// ---- TeamManager: UpdateMemberRole Tests ----

func TestClerkTeamManager_UpdateMemberRole(t *testing.T) {
	t.Run("successful role update", func(t *testing.T) {
		var gotBody map[string]string

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPatch, r.Method)
			assert.Equal(t, "/organizations/org_abc/memberships/user_001", r.URL.Path)

			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &gotBody)

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":   "mem_1",
				"role": "org:admin",
				"public_user_data": map[string]any{
					"user_id":    "user_001",
					"first_name": "Alice",
					"last_name":  "Smith",
					"identifier": "alice@example.com",
					"image_url":  "https://img.example.com/alice.jpg",
				},
				"created_at": 1700000000000,
			})
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key", WithClerkBaseURL(srv.URL))
		member, err := provider.UpdateMemberRole(context.Background(), "org_abc", "user_001", "org:admin")

		require.NoError(t, err)
		require.NotNil(t, member)
		assert.Equal(t, "user_001", member.UserID)
		assert.Equal(t, "Alice", member.FirstName)
		assert.Equal(t, "Smith", member.LastName)
		assert.Equal(t, "org:admin", member.Role)

		// Verify request body
		assert.Equal(t, "org:admin", gotBody["role"])
	})

	t.Run("API error returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid role"}`))
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key", WithClerkBaseURL(srv.URL))
		_, err := provider.UpdateMemberRole(context.Background(), "org_abc", "user_001", "invalid")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})
}

// ---- TeamManager: ListInvitations Tests ----

func TestClerkTeamManager_ListInvitations(t *testing.T) {
	t.Run("successful list with status filter", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/organizations/org_abc/invitations", r.URL.Path)
			assert.Equal(t, "pending", r.URL.Query().Get("status"))
			assert.Equal(t, "20", r.URL.Query().Get("limit"))
			assert.Equal(t, "0", r.URL.Query().Get("offset"))

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":            "inv_1",
						"email_address": "pending@example.com",
						"role":          "org:member",
						"status":        "pending",
						"created_at":    1700000000000,
						"expires_at":    1700604800000,
					},
				},
				"total_count": 1,
			})
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key", WithClerkBaseURL(srv.URL))
		invitations, total, err := provider.ListInvitations(context.Background(), "org_abc", "pending", 20, 0)

		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, invitations, 1)

		assert.Equal(t, "inv_1", invitations[0].ID)
		assert.Equal(t, "pending@example.com", invitations[0].Email)
		assert.Equal(t, "org:member", invitations[0].Role)
		assert.Equal(t, "pending", invitations[0].Status)
		assert.False(t, invitations[0].CreatedAt.IsZero())
		assert.False(t, invitations[0].ExpiresAt.IsZero())
	})

	t.Run("list without status filter", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "", r.URL.Query().Get("status"), "no status filter should be sent")

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data":        []any{},
				"total_count": 0,
			})
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key", WithClerkBaseURL(srv.URL))
		invitations, total, err := provider.ListInvitations(context.Background(), "org_abc", "", 20, 0)

		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, invitations)
	})

	t.Run("invitation without expires_at", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":            "inv_2",
						"email_address": "no-expiry@example.com",
						"role":          "org:admin",
						"status":        "pending",
						"created_at":    1700000000000,
					},
				},
				"total_count": 1,
			})
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key", WithClerkBaseURL(srv.URL))
		invitations, _, err := provider.ListInvitations(context.Background(), "org_abc", "", 20, 0)

		require.NoError(t, err)
		require.Len(t, invitations, 1)
		assert.True(t, invitations[0].ExpiresAt.IsZero(), "ExpiresAt should be zero when not provided")
	})
}

// ---- TeamManager: RevokeInvitation Tests ----

func TestClerkTeamManager_RevokeInvitation(t *testing.T) {
	t.Run("successful revoke", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/organizations/org_abc/invitations/inv_123/revoke", r.URL.Path)
			assert.Equal(t, "Bearer sk_test_key", r.Header.Get("Authorization"))

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key", WithClerkBaseURL(srv.URL))
		err := provider.RevokeInvitation(context.Background(), "org_abc", "inv_123")

		require.NoError(t, err)
	})

	t.Run("API error returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"invitation not found"}`))
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key", WithClerkBaseURL(srv.URL))
		err := provider.RevokeInvitation(context.Background(), "org_abc", "inv_999")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "404")
	})

	t.Run("unreachable server returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL
		srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key", WithClerkBaseURL(url))
		err := provider.RevokeInvitation(context.Background(), "org_abc", "inv_123")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "request failed")
	})
}

// ---- doJSON shared helper Tests ----

func TestClerkDoJSON_ContentTypeHandling(t *testing.T) {
	t.Run("GET request has no Content-Type", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Empty(t, r.Header.Get("Content-Type"), "GET should not set Content-Type")
			_, _ = w.Write([]byte(`{}`))
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key", WithClerkBaseURL(srv.URL))
		err := provider.doJSON(context.Background(), http.MethodGet, srv.URL+"/test", nil, nil)
		require.NoError(t, err)
	})

	t.Run("POST with body sets Content-Type application/json", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			_, _ = w.Write([]byte(`{}`))
		}))
		defer srv.Close()

		provider := NewClerkIdentityProvider("sk_test_key", WithClerkBaseURL(srv.URL))
		err := provider.doJSON(context.Background(), http.MethodPost, srv.URL+"/test", map[string]string{"key": "val"}, nil)
		require.NoError(t, err)
	})
}
