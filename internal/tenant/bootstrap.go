package tenant

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// BootstrapConfig holds configuration for the auto-bootstrap process.
type BootstrapConfig struct {
	// AutoBootstrap controls whether the default tenant is verified
	// on first boot. Default: true.
	AutoBootstrap bool

	// Logger for bootstrap events.
	Logger *slog.Logger
}

// DefaultBootstrapConfig returns the default bootstrap configuration.
func DefaultBootstrapConfig() BootstrapConfig {
	return BootstrapConfig{
		AutoBootstrap: true,
		Logger:        slog.Default(),
	}
}

// Bootstrap ensures the default tenant exists (created by migration 000004).
// This is idempotent — safe to call on every startup.
func Bootstrap(ctx context.Context, svc *Service, cfg BootstrapConfig) error {
	if !cfg.AutoBootstrap {
		cfg.Logger.InfoContext(ctx, "auto-bootstrap disabled, skipping")
		return nil
	}

	logger := cfg.Logger

	// Check that the default tenant exists (created by migration 000004)
	_, err := svc.repo.GetTenantByID(ctx, DefaultTenantID)
	if err != nil {
		if storage.IsNotFound(err) {
			logger.WarnContext(ctx, "default tenant not found — run migrations first")
			return fmt.Errorf("bootstrap: default tenant %s not found; run database migrations", DefaultTenantID)
		}
		return fmt.Errorf("bootstrap: check default tenant: %w", err)
	}

	logger.InfoContext(ctx, "bootstrap: default tenant verified", "tenant_id", DefaultTenantID)
	return nil
}
