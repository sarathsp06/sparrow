package webhooks

import (
	"context"
	"log/slog"
	"regexp"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/sarathsp06/sparrow/internal/observability"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"github.com/sarathsp06/sparrow/pkg/crypto"
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
)

type WebhookService struct {
	jobInserter          queue.JobInserter
	webhookRepo          store.RepositoryInterface
	crypto               *crypto.Service
	logger               *slog.Logger
	tracer               trace.Tracer
	metrics              *observability.SparrowMetrics
	allowPrivateNetworks bool
}

// WebhookManager manages webhook registrations and their lifecycle.
type WebhookManager interface {
	RegisterWebhook(ctx context.Context, namespace string, events []string, url string, headers map[string]string, timeout int, active bool, description string, secretHeaders map[string]string) (string, time.Time, error)
	CreateWebhook(ctx context.Context, req WebhookRegistrationRequest) (*WebhookRegistration, error)
	UnregisterWebhook(ctx context.Context, webhookID string, namespace string) error
	ListWebhooks(ctx context.Context, namespace string, webhookID string, event string, activeOnly bool, health string, limit, offset int32) ([]*store.WebhookRegistration, int32, error)
	UpdateWebhookConfig(ctx context.Context, webhookID string, namespace string, events []string, url string, headers map[string]string, active bool, description string, httpConfig *HTTPConfigUpdate, secretHeaders map[string]string, signatureType string, updateMask []string) error
	PauseWebhook(ctx context.Context, webhookID string, namespace string, reason string) error
	ResumeWebhook(ctx context.Context, webhookID string, namespace string) error
	GetNamespaceStats(ctx context.Context, namespace string) (*NamespaceStatsData, error)
}

// EventManager manages event type registrations and event publishing.
type EventManager interface {
	RegisterEvent(ctx context.Context, name string, description string, schema map[string]any, metadata map[string]string, active bool) (string, time.Time, error)
	ListEvents(ctx context.Context, activeOnly bool, limit, offset int32) ([]*store.EventRegistration, int32, error)
	UpdateEvent(ctx context.Context, name string, description string, schema map[string]any, metadata map[string]string, active bool) error
	DeleteEvent(ctx context.Context, name string) error
	GetEvent(ctx context.Context, name string) (*store.EventRegistration, error)
	// PushEvent returns (eventID, isDuplicate, schemaValid, warnings, err).
	// isDuplicate is true when idempotencyKey matched an existing event; the
	// other fields then describe that existing event, not a new one.
	PushEvent(ctx context.Context, namespace string, event string, payload map[string]any, ttlSeconds int64, metadata map[string]string, labels map[string]string, idempotencyKey *string) (eventID string, isDuplicate bool, schemaValid bool, warnings []string, err error)
	RePushEvent(ctx context.Context, eventID string) (string, []string, error)
	GetEventRecord(ctx context.Context, eventID string) (*store.EventRecord, int32, int32, int32, int32, error)
	ListEventReports(ctx context.Context, filter store.EventReportFilter) ([]*store.EventReportWithStats, int32, string, error)
}

// SubscriptionManager manages event subscriptions and payload-transform templates.
type SubscriptionManager interface {
	CreateSubscription(ctx context.Context, webhookID, eventName, namespace string, headers map[string]string, method string, timeout int, transformEnabled bool, transformTemplate string, labelFilters map[string]string) (string, time.Time, error)
	GetSubscription(ctx context.Context, subscriptionID string, namespace string) (*store.EventSubscription, error)
	ListSubscriptions(ctx context.Context, namespace string, webhookID string, eventName string, limit, offset int32) ([]*store.EventSubscription, int32, error)
	UpdateSubscription(ctx context.Context, subscriptionID string, namespace string, headers map[string]string, method string, timeout int, transformEnabled bool, transformTemplate string, labelFilters map[string]string) error
	DeleteSubscription(ctx context.Context, subscriptionID string, namespace string) error
	TestSubscriptionTemplate(ctx context.Context, eventName, transformTemplate, namespace string) (string, error)
	ListSubscriptionsByWebhookIDs(ctx context.Context, webhookIDs []uuid.UUID) ([]*store.EventSubscription, error)
	GetTemplateFunctions() []TemplateFunctionInfo
}

// DeliveryManager exposes delivery status, attempt history, and retries.
type DeliveryManager interface {
	GetDeliveryStatus(ctx context.Context, deliveryID string, namespace string) (*store.WebhookDelivery, error)
	GetDeliveryAttempts(ctx context.Context, deliveryID string) ([]*store.WebhookHealthEvent, error)
	ListDeliveries(ctx context.Context, filter store.DeliveryFilter) ([]*store.WebhookDelivery, int32, string, error)
	RetryDelivery(ctx context.Context, namespace string, deliveryID string, webhookID string, force bool) ([]string, int32, error)
}

// HealthManager exposes per-webhook and aggregate health.
type HealthManager interface {
	GetWebhookHealth(ctx context.Context, webhookID string, namespace string) (*WebhookHealthData, error)
	GetHealthSummary(ctx context.Context) (*HealthSummaryData, error)
}

// BatchManager runs prepared bulk re-push and bulk retry jobs.
type BatchManager interface {
	RePushEvents(ctx context.Context, repushID string) error
	GetRepushStatus(ctx context.Context, repushID string) (*store.BatchJob, error)
	CancelRepush(ctx context.Context, repushID string) error
	RetryDeliveries(ctx context.Context, retryID string) error
	GetRetryStatus(ctx context.Context, retryID string) (*store.BatchJob, error)
	CancelRetry(ctx context.Context, retryID string) error
}

// SecretRevealer decrypts stored webhook secrets for masking and exposes the
// Ed25519 signing public key. Used by response conversion, never for delivery.
type SecretRevealer interface {
	DecryptSecretHeaders(encrypted []byte) (map[string]string, error)
	DecryptWebhookSecret(encrypted []byte) (string, error)
	WebhookSigningPublicKeyHex(encryptedPrivKey []byte) string
}

//go:generate gowrap gen -i WebhookServiceInterface -t ../../templates/opentelemetry.tmpl -o WebhookServiceInterface_otel.go

// WebhookServiceInterface is the composite of every domain interface, satisfied
// by *WebhookService. Consumers should depend on the narrow per-domain
// interfaces; the composite exists for wiring and the OTel decorator.
type WebhookServiceInterface interface {
	WebhookManager
	EventManager
	SubscriptionManager
	DeliveryManager
	HealthManager
	BatchManager
	SecretRevealer
}

type TemplateFunctionInfo struct {
	Name        string
	Description string
}

var _ WebhookServiceInterface = (*WebhookService)(nil)

// NewWebhookService creates a new WebhookService instance
// WebhookServiceOption configures a WebhookService.
type WebhookServiceOption func(*WebhookService)

// WithAllowPrivateNetworks disables SSRF protection for webhook URL validation,
// permitting loopback and private-network addresses. Useful for self-hosted
// deployments where webhook targets live on the same network, and required for
// integration tests that use httptest.NewServer.
func WithAllowPrivateNetworks(allow bool) WebhookServiceOption {
	return func(s *WebhookService) {
		s.allowPrivateNetworks = allow
	}
}

func NewWebhookService(queueManager queue.JobInserter, webhookRepo store.RepositoryInterface, cryptoSvc *crypto.Service, opts ...WebhookServiceOption) *WebhookService {
	metrics, err := observability.NewSparrowMetrics()
	if err != nil {
		// Log error but continue without metrics
		slog.Default().With("component", "webhook-service").Error("Failed to initialize metrics", "error", err)
	}

	svc := &WebhookService{
		jobInserter: queueManager,
		webhookRepo: webhookRepo,
		crypto:      cryptoSvc,
		logger:      slog.Default().With("component", "webhook-service"),
		tracer:      observability.GetTracer("sparrow.service.webhook"),
		metrics:     metrics,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// --- Common Helpers ---

// parseUUID parses a UUID string and returns a typed error on failure.
// entityName is used in the error message (e.g. "webhook ID", "subscription ID").
func parseUUID(s string, entityName string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, svcerrors.Errorf(svcerrors.InvalidArgument, "invalid %s: %v", entityName, err)
	}
	return id, nil
}

// normalizePagination applies default limit (50) and ensures offset is non-negative.
func normalizePagination(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// Label validation constraints
const (
	maxLabelsPerMap  = 20  // max key-value pairs per labels/labelFilters map
	maxLabelKeyLen   = 64  // max characters per label key
	maxLabelValueLen = 256 // max characters per label value
)

// labelKeyPattern restricts label keys to alphanumeric, dot, underscore, and hyphen.
var labelKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// validateLabels checks that a labels/label_filters map meets size and format constraints.
func validateLabels(m map[string]string, fieldName string) error {
	if len(m) > maxLabelsPerMap {
		return svcerrors.Errorf(svcerrors.InvalidArgument, "%s: too many entries (%d), maximum is %d", fieldName, len(m), maxLabelsPerMap)
	}
	for k, v := range m {
		if k == "" {
			return svcerrors.Errorf(svcerrors.InvalidArgument, "%s: key must not be empty", fieldName)
		}
		if len(k) > maxLabelKeyLen {
			return svcerrors.Errorf(svcerrors.InvalidArgument, "%s: key %q exceeds maximum length of %d characters", fieldName, k, maxLabelKeyLen)
		}
		if !labelKeyPattern.MatchString(k) {
			return svcerrors.Errorf(svcerrors.InvalidArgument, "%s: key %q contains invalid characters (allowed: alphanumeric, '.', '_', '-')", fieldName, k)
		}
		if len(v) > maxLabelValueLen {
			return svcerrors.Errorf(svcerrors.InvalidArgument, "%s: value for key %q exceeds maximum length of %d characters", fieldName, k, maxLabelValueLen)
		}
	}
	return nil
}
