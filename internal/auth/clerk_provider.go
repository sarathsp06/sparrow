package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"
)

// ClerkIdentityProvider syncs namespace roles to Clerk's organization
// membership publicMetadata so they appear in future JWT session tokens.
//
// It calls the Clerk Backend API directly (no SDK dependency) using:
//
//	PATCH /organizations/{org_id}/memberships/{user_id}/metadata
//
// The namespace roles are stored as a sorted string slice in publicMetadata:
//
//	{
//	  "public_metadata": {
//	    "namespace_roles": [
//	      "namespace:admin:customer-a",
//	      "namespace:viewer:customer-b"
//	    ]
//	  }
//	}
//
// Format: "namespace:{role_level}:{namespace_name}" where role_level is
// the part after the colon in the Sparrow role (e.g., "admin" from "namespace:admin").
//
// This format is compact, easy to parse in JWT claim extraction, and
// alphabetically sortable for idempotency checks.
type ClerkIdentityProvider struct {
	secretKey  string
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// ClerkProviderOption configures the Clerk identity provider.
type ClerkProviderOption func(*ClerkIdentityProvider)

// WithClerkBaseURL overrides the Clerk API base URL (for testing).
func WithClerkBaseURL(url string) ClerkProviderOption {
	return func(p *ClerkIdentityProvider) {
		p.baseURL = strings.TrimRight(url, "/")
	}
}

// WithClerkHTTPClient sets a custom HTTP client (for testing or custom transport).
func WithClerkHTTPClient(client *http.Client) ClerkProviderOption {
	return func(p *ClerkIdentityProvider) {
		p.httpClient = client
	}
}

// WithClerkLogger sets the logger for the Clerk provider.
func WithClerkLogger(logger *slog.Logger) ClerkProviderOption {
	return func(p *ClerkIdentityProvider) {
		p.logger = logger
	}
}

// NewClerkIdentityProvider creates a Clerk identity provider that syncs
// namespace roles to organization membership publicMetadata.
//
// secretKey is the Clerk Backend API secret key (sk_live_* or sk_test_*).
func NewClerkIdentityProvider(secretKey string, opts ...ClerkProviderOption) *ClerkIdentityProvider {
	p := &ClerkIdentityProvider{
		secretKey: secretKey,
		baseURL:   "https://api.clerk.com/v1",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: slog.Default(),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// clerkMetadataRequest is the JSON body for the Clerk metadata PATCH endpoint.
type clerkMetadataRequest struct {
	PublicMetadata map[string]any `json:"public_metadata"`
}

// SyncNamespaceRoles pushes a user's complete namespace role set to Clerk
// organization membership publicMetadata.
//
// The roles are encoded as a sorted string slice for deterministic comparison:
//
//	["namespace:admin:customer-a", "namespace:viewer:customer-b"]
//
// An empty/nil roles map results in an empty slice (clearing all namespace roles).
func (p *ClerkIdentityProvider) SyncNamespaceRoles(ctx context.Context, externalTenantID string, subjectID string, roles map[string]Role) error {
	if externalTenantID == "" {
		return fmt.Errorf("clerk: externalTenantID (org_id) is required")
	}
	if subjectID == "" {
		return fmt.Errorf("clerk: subjectID (user_id) is required")
	}

	// Encode roles as sorted "namespace:{level}:{name}" strings
	encoded := encodeNamespaceRoles(roles)

	body := clerkMetadataRequest{
		PublicMetadata: map[string]any{
			"namespace_roles": encoded,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("clerk: marshal metadata: %w", err)
	}

	url := fmt.Sprintf("%s/organizations/%s/memberships/%s/metadata",
		p.baseURL, externalTenantID, subjectID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("clerk: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.secretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("clerk: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		p.logger.InfoContext(ctx, "clerk: synced namespace roles",
			slog.String("org_id", externalTenantID),
			slog.String("user_id", subjectID),
			slog.Int("role_count", len(roles)),
		)
		return nil
	}

	// Read error response for diagnostics
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	p.logger.ErrorContext(ctx, "clerk: failed to sync namespace roles",
		slog.String("org_id", externalTenantID),
		slog.String("user_id", subjectID),
		slog.Int("status_code", resp.StatusCode),
		slog.String("response", string(respBody)),
	)

	return fmt.Errorf("clerk: API returned %d: %s", resp.StatusCode, string(respBody))
}

// encodeNamespaceRoles converts a namespace->Role map into a sorted string slice.
// Format: "namespace:{level}:{name}" e.g. "namespace:admin:customer-a"
//
// This is the canonical encoding used in both Clerk publicMetadata and JWT claims.
func encodeNamespaceRoles(roles map[string]Role) []string {
	if len(roles) == 0 {
		return []string{}
	}

	encoded := make([]string, 0, len(roles))
	for ns, role := range roles {
		// Role is e.g. "namespace:admin" -- extract the level part after ":"
		level := string(role)
		if parts := strings.SplitN(level, ":", 2); len(parts) == 2 {
			level = parts[1]
		}
		encoded = append(encoded, fmt.Sprintf("namespace:%s:%s", level, ns))
	}
	sort.Strings(encoded)
	return encoded
}

// DecodeNamespaceRoles parses the sorted string slice from JWT claims or Clerk
// metadata back into a namespace->Role map.
//
// Input:  ["namespace:admin:customer-a", "namespace:viewer:customer-b"]
// Output: {"customer-a": "namespace:admin", "customer-b": "namespace:viewer"}
//
// Invalid entries are silently skipped.
func DecodeNamespaceRoles(encoded []string) map[string]Role {
	if len(encoded) == 0 {
		return nil
	}

	roles := make(map[string]Role, len(encoded))
	for _, entry := range encoded {
		// Expected format: "namespace:{level}:{name}"
		parts := strings.SplitN(entry, ":", 3)
		if len(parts) != 3 || parts[0] != "namespace" {
			continue
		}
		level := parts[1]
		name := parts[2]
		if name == "" || level == "" {
			continue
		}
		role := Role("namespace:" + level)
		if IsNamespaceRole(role) {
			roles[name] = role
		}
	}
	return roles
}

// Compile-time check
var _ IdentityProvider = (*ClerkIdentityProvider)(nil)
var _ TeamManager = (*ClerkIdentityProvider)(nil)

// TeamManagement returns self — Clerk supports full team management.
func (p *ClerkIdentityProvider) TeamManagement() TeamManager {
	return p
}

// ---- TeamManager implementation (Clerk Backend API) ----

// clerkPaginatedResponse is the generic paginated response shape from Clerk.
type clerkPaginatedResponse[T any] struct {
	Data       []T `json:"data"`
	TotalCount int `json:"total_count"`
}

// clerkMembership maps the Clerk OrganizationMembership JSON response.
type clerkMembership struct {
	ID             string              `json:"id"`
	Role           string              `json:"role"`
	PublicUserData clerkPublicUserData `json:"public_user_data"`
	CreatedAt      int64               `json:"created_at"`
	UpdatedAt      int64               `json:"updated_at"`
}

type clerkPublicUserData struct {
	UserID     string `json:"user_id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Identifier string `json:"identifier"`
	ImageURL   string `json:"image_url"`
	HasImage   bool   `json:"has_image"`
}

// clerkInvitation maps the Clerk OrganizationInvitation JSON response.
type clerkInvitation struct {
	ID           string `json:"id"`
	EmailAddress string `json:"email_address"`
	Role         string `json:"role"`
	Status       string `json:"status"`
	CreatedAt    int64  `json:"created_at"`
	ExpiresAt    int64  `json:"expires_at,omitempty"`
}

// ListMembers lists all members of an organization via Clerk Backend API.
func (p *ClerkIdentityProvider) ListMembers(ctx context.Context, externalOrgID string, limit, offset int) ([]TeamMember, int, error) {
	url := fmt.Sprintf("%s/organizations/%s/memberships?limit=%d&offset=%d",
		p.baseURL, externalOrgID, limit, offset)

	var result clerkPaginatedResponse[clerkMembership]
	if err := p.doJSON(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, 0, fmt.Errorf("clerk: list members: %w", err)
	}

	members := make([]TeamMember, len(result.Data))
	for i, m := range result.Data {
		members[i] = TeamMember{
			UserID:    m.PublicUserData.UserID,
			FirstName: m.PublicUserData.FirstName,
			LastName:  m.PublicUserData.LastName,
			Email:     m.PublicUserData.Identifier,
			ImageURL:  m.PublicUserData.ImageURL,
			Role:      m.Role,
			JoinedAt:  time.UnixMilli(m.CreatedAt),
		}
	}

	return members, result.TotalCount, nil
}

// InviteMember invites a new member to the organization by email.
func (p *ClerkIdentityProvider) InviteMember(ctx context.Context, externalOrgID, email, role string) (*TeamInvitation, error) {
	url := fmt.Sprintf("%s/organizations/%s/invitations", p.baseURL, externalOrgID)

	body := map[string]string{
		"email_address":   email,
		"role":            role,
		"inviter_user_id": "", // Optional: leave empty to send from the org
	}

	var result clerkInvitation
	if err := p.doJSON(ctx, http.MethodPost, url, body, &result); err != nil {
		return nil, fmt.Errorf("clerk: invite member: %w", err)
	}

	inv := clerkInvitationToTeamInvitation(result)
	return &inv, nil
}

// RemoveMember removes a member from the organization.
func (p *ClerkIdentityProvider) RemoveMember(ctx context.Context, externalOrgID, userID string) error {
	url := fmt.Sprintf("%s/organizations/%s/memberships/%s",
		p.baseURL, externalOrgID, userID)

	if err := p.doJSON(ctx, http.MethodDelete, url, nil, nil); err != nil {
		return fmt.Errorf("clerk: remove member: %w", err)
	}

	return nil
}

// UpdateMemberRole updates a member's organization-level role.
func (p *ClerkIdentityProvider) UpdateMemberRole(ctx context.Context, externalOrgID, userID, role string) (*TeamMember, error) {
	url := fmt.Sprintf("%s/organizations/%s/memberships/%s",
		p.baseURL, externalOrgID, userID)

	body := map[string]string{
		"role": role,
	}

	var result clerkMembership
	if err := p.doJSON(ctx, http.MethodPatch, url, body, &result); err != nil {
		return nil, fmt.Errorf("clerk: update member role: %w", err)
	}

	member := TeamMember{
		UserID:    result.PublicUserData.UserID,
		FirstName: result.PublicUserData.FirstName,
		LastName:  result.PublicUserData.LastName,
		Email:     result.PublicUserData.Identifier,
		ImageURL:  result.PublicUserData.ImageURL,
		Role:      result.Role,
		JoinedAt:  time.UnixMilli(result.CreatedAt),
	}
	return &member, nil
}

// ListInvitations lists invitations for an organization.
func (p *ClerkIdentityProvider) ListInvitations(ctx context.Context, externalOrgID string, status string, limit, offset int) ([]TeamInvitation, int, error) {
	url := fmt.Sprintf("%s/organizations/%s/invitations?limit=%d&offset=%d",
		p.baseURL, externalOrgID, limit, offset)

	if status != "" {
		url += fmt.Sprintf("&status=%s", status)
	}

	var result clerkPaginatedResponse[clerkInvitation]
	if err := p.doJSON(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, 0, fmt.Errorf("clerk: list invitations: %w", err)
	}

	invitations := make([]TeamInvitation, len(result.Data))
	for i, inv := range result.Data {
		invitations[i] = clerkInvitationToTeamInvitation(inv)
	}

	return invitations, result.TotalCount, nil
}

// RevokeInvitation revokes a pending invitation.
func (p *ClerkIdentityProvider) RevokeInvitation(ctx context.Context, externalOrgID, invitationID string) error {
	url := fmt.Sprintf("%s/organizations/%s/invitations/%s/revoke",
		p.baseURL, externalOrgID, invitationID)

	if err := p.doJSON(ctx, http.MethodPost, url, nil, nil); err != nil {
		return fmt.Errorf("clerk: revoke invitation: %w", err)
	}

	return nil
}

// ---- Shared HTTP helper ----

// doJSON performs an HTTP request to the Clerk API, optionally encoding a JSON body
// and decoding the response into result. If result is nil, the response body is discarded.
func (p *ClerkIdentityProvider) doJSON(ctx context.Context, method, url string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.secretKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

func clerkInvitationToTeamInvitation(inv clerkInvitation) TeamInvitation {
	ti := TeamInvitation{
		ID:        inv.ID,
		Email:     inv.EmailAddress,
		Role:      inv.Role,
		Status:    inv.Status,
		CreatedAt: time.UnixMilli(inv.CreatedAt),
	}
	if inv.ExpiresAt > 0 {
		ti.ExpiresAt = time.UnixMilli(inv.ExpiresAt)
	}
	return ti
}
