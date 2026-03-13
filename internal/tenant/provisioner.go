package tenant

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/auth"
)

// MaxTenantsPerUser is the maximum number of tenants a single user (JWT sub)
// can create via auto-provisioning.
const MaxTenantsPerUser = 2

// Compile-time check that AutoProvisioner implements auth.TenantProvisioner.
var _ auth.TenantProvisioner = (*AutoProvisioner)(nil)

// AutoProvisioner implements auth.TenantProvisioner. It creates a new tenant
// when a JWT contains an org_id that has no corresponding tenant in the
// database, and enforces per-user tenant creation limits.
type AutoProvisioner struct {
	svc    *Service
	logger *slog.Logger
}

// NewAutoProvisioner creates a new auto-provisioner backed by the tenant service.
func NewAutoProvisioner(svc *Service, logger *slog.Logger) *AutoProvisioner {
	if logger == nil {
		logger = slog.Default()
	}
	return &AutoProvisioner{svc: svc, logger: logger}
}

// ProvisionTenant creates a new tenant for the given external ID (e.g., Clerk
// org_id). It enforces a limit of MaxTenantsPerUser tenants per user.
//
// The tenant name is derived from the external ID (e.g., "org_2xYz..." becomes
// the name "org_2xYz..."). The org name from Clerk is not available in the JWT,
// so we use the external ID as a placeholder. The user can rename it later.
func (p *AutoProvisioner) ProvisionTenant(ctx context.Context, externalID string, createdBy string) (uuid.UUID, error) {
	if createdBy == "" {
		return uuid.Nil, fmt.Errorf("cannot auto-provision tenant: no user identity (JWT sub claim missing)")
	}

	// Enforce per-user tenant limit
	count, err := p.svc.repo.CountTenantsByCreator(ctx, createdBy)
	if err != nil {
		return uuid.Nil, fmt.Errorf("check tenant count for user %q: %w", createdBy, err)
	}
	if count >= MaxTenantsPerUser {
		return uuid.Nil, fmt.Errorf("user %q has reached the maximum of %d tenants", createdBy, MaxTenantsPerUser)
	}

	// Use the external ID as a placeholder name. The Clerk org name is not
	// available in the JWT claims, so this is the best we can do at
	// auto-provision time. Users can rename the tenant later.
	name := externalID

	t, err := p.svc.CreateTenant(ctx, name, CreateTenantOpts{
		ExternalID: &externalID,
		CreatedBy:  &createdBy,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("auto-provision tenant: %w", err)
	}

	p.logger.Info("auto-provisioned tenant",
		"tenant_id", t.ID.String(),
		"external_id", externalID,
		"created_by", createdBy,
	)

	return t.ID, nil
}
