package tenant

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/sarathsp06/sparrow/internal/auth"
	"github.com/sarathsp06/sparrow/pkg/storage"
)

// BootstrapConfig holds configuration for the auto-bootstrap process.
type BootstrapConfig struct {
	// AutoBootstrap controls whether the default tenant and root API key
	// are created on first boot. Default: true.
	AutoBootstrap bool

	// RootAPIKey is an optional pre-defined root API key.
	// If set, this exact key is used instead of generating a random one.
	// Useful for deterministic deployments (e.g., Docker Compose, CI).
	RootAPIKey string

	// Logger for bootstrap events.
	Logger *slog.Logger
}

// DefaultBootstrapConfig returns the default bootstrap configuration,
// reading from environment variables.
func DefaultBootstrapConfig() BootstrapConfig {
	cfg := BootstrapConfig{
		AutoBootstrap: true,
		Logger:        slog.Default(),
	}

	if v := os.Getenv("SPARROW_AUTO_BOOTSTRAP"); v == "false" || v == "0" {
		cfg.AutoBootstrap = false
	}
	if v := os.Getenv("SPARROW_ROOT_API_KEY"); v != "" {
		cfg.RootAPIKey = v
	}

	return cfg
}

// Bootstrap ensures the default tenant exists and creates a root API key
// if one hasn't been created yet. This is idempotent — safe to call on
// every startup.
//
// On first boot:
//   - Verifies the default tenant exists (created by the migration)
//   - Creates a root API key with tenant:admin role + platform_admin flag
//   - Prints the key to logs (it's shown only once)
//
// On subsequent boots:
//   - Checks that API keys exist for the default tenant
//   - If they do, skips key creation (already bootstrapped)
func Bootstrap(ctx context.Context, svc *Service, cfg BootstrapConfig) error {
	if !cfg.AutoBootstrap {
		cfg.Logger.InfoContext(ctx, "auto-bootstrap disabled, skipping")
		return nil
	}

	logger := cfg.Logger

	// Check that the default tenant exists (created by migration 000004)
	_, err := svc.repo.GetTenantByID(ctx, auth.DefaultTenantID)
	if err != nil {
		if storage.IsNotFound(err) {
			logger.WarnContext(ctx, "default tenant not found — run migrations first")
			return fmt.Errorf("bootstrap: default tenant %s not found; run database migrations", auth.DefaultTenantID)
		}
		return fmt.Errorf("bootstrap: check default tenant: %w", err)
	}

	// Check if any API keys already exist for the default tenant
	keys, total, err := svc.repo.ListAPIKeys(ctx, auth.DefaultTenantID, 1, 0)
	if err != nil {
		return fmt.Errorf("bootstrap: list API keys: %w", err)
	}
	_ = keys
	if total > 0 {
		logger.InfoContext(ctx, "bootstrap: default tenant already has API keys, skipping key creation")
		return nil
	}

	// Create the root API key
	var result *CreateAPIKeyResult
	if cfg.RootAPIKey != "" {
		// Use the pre-defined key
		result, err = createRootKeyWithValue(ctx, svc, cfg.RootAPIKey)
	} else {
		// Generate a random key
		result, err = svc.CreateAPIKey(ctx, CreateAPIKeyRequest{
			TenantID:        auth.DefaultTenantID,
			Name:            "Root API Key (auto-generated)",
			Role:            auth.RoleTenantAdmin,
			IsPlatformAdmin: true,
		})
	}
	if err != nil {
		return fmt.Errorf("bootstrap: create root API key: %w", err)
	}

	// Print the key — this is the only time it's shown
	logger.InfoContext(ctx, "========================================================")
	logger.InfoContext(ctx, "  ROOT API KEY CREATED (save this — it won't be shown again)")
	logger.InfoContext(ctx, "  "+result.RawKey)
	logger.InfoContext(ctx, "========================================================")

	return nil
}

// createRootKeyWithValue creates a root API key using a specific raw key value.
func createRootKeyWithValue(ctx context.Context, svc *Service, rawKey string) (*CreateAPIKeyResult, error) {
	keyHash := auth.HashAPIKey(rawKey)
	prefix := rawKey
	if idx := nthIndex(rawKey, '_', 2); idx > 0 && idx < len(rawKey) {
		prefix = rawKey[:idx+1]
	} else {
		prefix = "sk_default_"
	}

	key := &APIKey{
		TenantID:        auth.DefaultTenantID,
		Name:            "Root API Key (pre-configured)",
		KeyPrefix:       prefix,
		KeyHash:         keyHash,
		Role:            auth.RoleTenantAdmin,
		IsPlatformAdmin: true,
	}

	if err := svc.repo.CreateAPIKey(ctx, key); err != nil {
		return nil, err
	}

	return &CreateAPIKeyResult{
		Key:    key,
		RawKey: rawKey,
	}, nil
}

// nthIndex returns the index of the nth occurrence of sep in s, or -1.
func nthIndex(s string, sep byte, n int) int {
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			count++
			if count == n {
				return i
			}
		}
	}
	return -1
}
