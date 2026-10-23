// Package server builds a complete Sparrow server so another Go program can
// run Sparrow in its own process and serve its HTTP handler behind its own
// middleware.
//
// Configuration comes from the same SPARROW_* environment variables the
// sparrow-server binary reads; nothing here adds configuration. New sets
// process-wide state as the binary does (the default slog logger, the
// OpenTelemetry providers, metric callbacks), so call it once per process.
//
// A typical embedding:
//
//	srv, err := server.New(ctx)
//	if err != nil { ... }
//	if err := srv.Start(ctx); err != nil { ... }
//	httpServer := &http.Server{Addr: srv.Addr(), Handler: myMiddleware(srv.Handler())}
//	go httpServer.ListenAndServe()
//	...
//	srv.Drain()
//	_ = httpServer.Shutdown(shutdownCtx)
//	_ = srv.Shutdown(shutdownCtx)
package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx"
	slogotel "github.com/remychantenay/slog-otel"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/sarathsp06/sparrow/internal/accessauth"
	"github.com/sarathsp06/sparrow/internal/ai"
	"github.com/sarathsp06/sparrow/internal/config"
	"github.com/sarathsp06/sparrow/internal/crypto"
	"github.com/sarathsp06/sparrow/internal/health"
	"github.com/sarathsp06/sparrow/internal/middleware"
	"github.com/sarathsp06/sparrow/internal/migration"
	"github.com/sarathsp06/sparrow/internal/observability"
	"github.com/sarathsp06/sparrow/internal/rest"
	"github.com/sarathsp06/sparrow/internal/storage/postgres"
	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/tlsrevocation"
	"github.com/sarathsp06/sparrow/internal/ui"
	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"github.com/sarathsp06/sparrow/pkg/access"
	"github.com/sarathsp06/sparrow/pkg/access/httpauth"
	"github.com/sarathsp06/sparrow/pkg/safetransport"
)

// Option configures New.
type Option func(*options)

type options struct {
	queues map[string]int
	limits func(ctx context.Context, tenantID uuid.UUID) TenantLimits
}

// TenantLimits bound one tenant's share of the shared work queues. Zero
// values mean no limit and the default queues.
type TenantLimits struct {
	// MaxInFlight caps the tenant's event fan-out and delivery jobs running
	// at once in one process; a job over the cap waits a second or two and
	// tries again, without using a retry attempt. With N processes a tenant
	// can run up to N times MaxInFlight.
	MaxInFlight int
	// Queue, when set, names one of the queues declared in WithTenantLimits;
	// the tenant's event fan-out and delivery jobs run there.
	Queue string
}

// WithTenantLimits keeps one busy tenant from delaying the others. queues
// declares extra work queues by name with their worker counts (for example
// a small queue for free tenants); limits returns each tenant's limits and
// is called for every event fan-out and delivery job, so it must be fast.
// Without this option every tenant shares the default queues with no cap,
// as in a self-hosted Sparrow.
func WithTenantLimits(queues map[string]int, limits func(ctx context.Context, tenantID uuid.UUID) TenantLimits) Option {
	return func(o *options) {
		o.queues = queues
		o.limits = limits
	}
}

// Server is a fully wired Sparrow: database pools, queue workers, the webhook
// service and the HTTP router. Create it with New, then Start it.
type Server struct {
	cfg     *config.Config
	handler http.Handler

	queue          *queue.Manager
	repo           store.RepositoryInterface
	access         *access.Service
	db             *sqlx.DB
	health         *health.Checker
	revocation     *tlsrevocation.Refresher
	stopRevocation context.CancelFunc
	otelShutdown   func(context.Context) error

	// Shown in the startup summary printed by Start.
	authEnabled  bool
	aiDeps       rest.AIDeps
	otlpEndpoint string
}

// Migrate runs Sparrow's database migrations and returns, without starting
// anything else. It is what `sparrow-server -migrate` runs as a pre-deploy job.
func Migrate(ctx context.Context) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	otelShutdown, _ := setupObservability(ctx, cfg)
	defer shutdownObservability(otelShutdown)
	return runMigrations(ctx, cfg)
}

// New loads configuration from the environment, runs migrations, opens the
// database pools and builds the queue, service and HTTP router. It does not
// start background work; call Start for that.
func New(ctx context.Context, opts ...Option) (*Server, error) {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}

	s := &Server{cfg: cfg}
	startTime := time.Now() // Track service start time for uptime calculation

	otelShutdown, otelConfig := setupObservability(ctx, cfg)
	s.otelShutdown = otelShutdown
	if otelShutdown != nil {
		s.otlpEndpoint = otelConfig.OTLPEndpoint
	}

	if err := runMigrations(ctx, cfg); err != nil {
		return nil, err
	}

	// Create database connection pool for River queue workers.
	// River runs 45 concurrent workers (20 events + 20 webhooks + 5 default)
	// so we need enough connections to avoid starvation. The default pgxpool
	// MaxConns is max(4, NumCPU) which is far too low.
	pgxConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL for pgxpool: %w", err)
	}
	pgxConfig.MaxConns = cfg.QueueDBMaxConns         // SPARROW_QUEUE_DB_MAX_CONNS, default 50
	pgxConfig.MinConns = min(10, pgxConfig.MaxConns) // keep warm connections ready
	pgxConfig.MaxConnLifetime = 30 * time.Minute     // recycle connections periodically
	pgxConfig.MaxConnIdleTime = 5 * time.Minute      // release idle connections
	pgxConfig.HealthCheckPeriod = 30 * time.Second   // detect stale connections

	dbPool, err := pgxpool.NewWithConfig(ctx, pgxConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}
	if err := dbPool.Ping(ctx); err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlxDB, err := postgres.Open(cfg.DatabaseURL, 3)
	if err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("failed to open sqlx database: %w", err)
	}
	s.db = sqlxDB
	sqlxDB.SetMaxOpenConns(cfg.DBMaxConns)
	sqlxDB.SetMaxIdleConns(cfg.DBMaxConns)
	sqlxDB.SetConnMaxLifetime(5 * time.Minute)
	sqlxDB.SetConnMaxIdleTime(5 * time.Minute)

	// Anything that fails from here on releases what was opened above.
	built := false
	defer func() {
		if !built {
			dbPool.Close()
			_ = sqlxDB.Close()
		}
	}()

	// Bootstrap default tenant (must exist from migrations)
	if err := tenant.Bootstrap(ctx, sqlxDB); err != nil {
		return nil, fmt.Errorf("failed to bootstrap: %w", err)
	}

	// Initialize encryption service.
	// SPARROW_ENCRYPTION_KEYS and SPARROW_ENCRYPTION_PRIMARY_KEY_ID must be
	// configured. The server will not start without a valid keyring.
	encKeyring, err := cfg.EncryptionKeyring()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve encryption keyring: %w", err)
	}
	cryptoSvc, err := crypto.NewServiceFromKeyring(encKeyring)
	if err != nil {
		return nil, fmt.Errorf("failed to create crypto service: %w", err)
	}
	fmt.Println("🔐 Encryption enabled (envelope encryption with per-record DEK)")

	// Access tokens and invites (pkg/access, adapted in internal/accessauth).
	// SPARROW_API_KEY is the root key; named tokens created from it can each
	// be revoked. Consumer-scoped tokens only work through the portal gateway.
	accessSvc, err := accessauth.New(sqlxDB.DB, cfg.APIKey, accessauth.Sealer(cryptoSvc))
	if err != nil {
		return nil, fmt.Errorf("failed to create access service: %w", err)
	}
	s.access = accessSvc

	// Authentication for /v1. When SPARROW_API_KEY is set, every request needs
	// the master key or a tenant-wide access token (X-API-Key header or
	// Authorization: Bearer). Health/ready and the OpenAPI docs/spec are
	// excluded. Portal traffic arrives pre-authorized through the /portal/api
	// gateway (see below). When unset, all requests are open.
	auth := middleware.NewAuth(cfg.APIKey, accessSvc, accessauth.Realm(), "/health", "/ready", "/docs", "/openapi")
	s.authEnabled = auth.Enabled
	if auth.Enabled {
		fmt.Println("🔑 Authentication enabled (SPARROW_API_KEY or an access token required)")
	} else {
		fmt.Println("⚠️  SPARROW_API_KEY not set — all endpoints are open (no authentication)")
	}

	// Private network access for webhook URLs
	// When true, localhost/private IPs are allowed as webhook targets.
	// Useful for local dev or self-hosted deployments where targets are on the same network.
	if cfg.AllowPrivateNetworks {
		fmt.Println("⚠️  SPARROW_ALLOW_PRIVATE_NETWORKS=true — SSRF protection relaxed (loopback/private IPs allowed; cloud metadata still blocked). Prefer SPARROW_ALLOWED_NETWORKS in production")
	}
	if len(cfg.AllowedNetworks) > 0 {
		fmt.Printf("🌐 Webhook deliveries may also reach SPARROW_ALLOWED_NETWORKS: %s\n", strings.Join(cfg.AllowedNetworks, ", "))
	}

	// Event retention: purge events (and their deliveries) older than N days.
	if cfg.EventRetentionDays > 0 {
		fmt.Printf("🧹 Event retention enabled — purging data older than %d days (SPARROW_EVENT_RETENTION_DAYS)\n", cfg.EventRetentionDays)
	}

	// Create webhook repository
	webhookRepo := store.NewRepositoryInterfaceWithTracing(store.NewRepository(sqlxDB), "")
	s.repo = webhookRepo

	if err := registerSystemEventTypes(ctx, webhookRepo); err != nil {
		log.Printf("⚠️  %v", err)
	}

	// Initialize webhook HTTP client config
	clientConfig := client.DefaultConfig()
	clientConfig.AllowPrivateNetworks = cfg.AllowPrivateNetworks
	clientConfig.AllowedNetworks = cfg.AllowedNetworkList()
	clientConfig.MaxCapturedResponseBytes = cfg.MaxCapturedResponseBytes
	// Size the connection pool from the worker pool: each delivery worker can
	// hold one connection, and many webhooks often share one receiver host.
	// The library default of 10 per host silently capped in-flight deliveries
	// to a slow host at 10 however many workers were configured; per-webhook
	// rate limits (rate_limit_rps) are the intended receiver protection.
	if workers := cfg.WebhookWorkers; workers > clientConfig.MaxConnsPerHost {
		clientConfig.MaxConnsPerHost = workers
		clientConfig.MaxIdleConns = 2 * workers
	}

	// Check intermediates in receivers' chains against their CRLs: Go checks
	// no revocation itself. CRLs are cached per instance, downloaded on first
	// use and refreshed by use or POST /v1/tls/crls:refresh (NOTIFY to every
	// instance). Downloads obey the delivery network policy: CRL addresses
	// come from certificates remote servers send.
	revocationMode, _ := cfg.TLSRevocationMode() // checked by Validate
	if revocationMode != "off" {
		mode := safetransport.RevocationSoft
		if revocationMode == "hard" {
			mode = safetransport.RevocationHard
		}
		checker := safetransport.NewRevocationChecker(safetransport.RevocationOptions{
			Client: clientConfig.Policy().NewHTTPClient(10 * time.Second),
			Mode:   mode,
			MaxAge: cfg.CRLMaxAge,
		})
		clientConfig.Revocation = checker
		s.revocation = tlsrevocation.New(dbPool, checker, revocationMode)
		fmt.Printf("🔐 TLS revocation checking: %s (CRLs refreshed after %s)\n", revocationMode, cfg.CRLMaxAge)
	}

	// Initialize queue manager
	autoDisable := queue.AutoDisablePolicy{After: cfg.AutoDisableAfter, MinFailures: cfg.AutoDisableMinFailuresOrDefault()}
	pool := queue.WorkerPoolConfig{EventWorkers: cfg.EventWorkers, WebhookWorkers: cfg.WebhookWorkers, FetchCooldown: cfg.QueueFetchCooldown}
	if o.limits != nil {
		pool.ExtraQueues = o.queues
		pool.TenantLimits = func(ctx context.Context, tenantID uuid.UUID) queue.TenantLimits {
			l := o.limits(ctx, tenantID)
			return queue.TenantLimits{MaxInFlight: l.MaxInFlight, Queue: l.Queue}
		}
	}
	healthEval := queue.HealthEvaluatorConfig{Interval: cfg.HealthEvalInterval}
	queueManager, err := queue.NewManager(ctx, webhookRepo, cryptoSvc, dbPool, clientConfig, cfg.EventRetentionDays, autoDisable, pool, healthEval)
	if err != nil {
		return nil, fmt.Errorf("failed to create queue manager: %w", err)
	}
	s.queue = queueManager
	if cfg.AutoDisableAfter > 0 {
		fmt.Printf("⏸️  Webhooks auto-disable after %s of failures (min %d in a row)\n", cfg.AutoDisableAfter, autoDisable.MinFailures)
	}
	if err := queueManager.RegisterStateMetrics(webhookRepo); err != nil {
		log.Printf("⚠️  Failed to register queue/webhook state metrics: %v", err)
	}

	// Portal links are access tokens minted per visit; purge dead ones daily.
	queueManager.EnableTokenPurge(accessSvc, accessauth.TokenRetention)

	webhookService := webhooks.NewWebhookService(queueManager.GetJobInserter(), webhookRepo, cryptoSvc, webhooks.WithAllowPrivateNetworks(cfg.AllowPrivateNetworks), webhooks.WithAllowedNetworks(cfg.AllowedNetworkList()), webhooks.WithAutoRegisterEvents(cfg.AutoRegisterEvents), webhooks.WithDeliveryClient(client.NewWebhookClient(clientConfig)))
	tracedWebhookService := webhooks.NewWebhookServiceInterfaceWithTracing(webhookService, "")

	// AI-assisted template drafting is opt-in (SPARROW_AI_ENABLED). Drafts are
	// verified through the same dry-run the editor's preview uses. The
	// prompt endpoint and its docs_url fetch work without a provider, so an
	// install with no AI still gets a copy-and-paste prompt.
	// docs_url fetches obey the delivery network policy (SSRF guard).
	fetch := ai.NewDocFetcher(client.NetworkPolicy{AllowPrivate: cfg.AllowPrivateNetworks, AllowedNetworks: cfg.AllowedNetworkList()})
	aiDeps := rest.AIDeps{Fetch: fetch, Timeout: cfg.AITimeout}
	if cfg.AIEnabled() {
		aiDeps.ConfigErr = cfg.AIConfigError()
	}
	if cfg.AIEnabled() && aiDeps.ConfigErr == nil {
		helpers := make([]ai.HelperFunc, 0)
		for _, f := range tracedWebhookService.GetTemplateFunctions() {
			helpers = append(helpers, ai.HelperFunc{Name: f.Name, Description: f.Description})
		}
		// Drafts are verified strictly: a field the sample lacks is an error
		// naming the key, which the repair loop feeds back to the model.
		render := func(_ context.Context, eventName, tmpl string, payload map[string]any) (string, error) {
			return webhooks.RenderTemplatePreview(eventName, tmpl, payload, true)
		}
		drafter, err := ai.New(ai.Config{Provider: ai.Provider(cfg.AIProviderName()), APIKey: cfg.AIAPIKey, Model: cfg.AIModelOrDefault(), BaseURL: cfg.AIBaseURL}, render, helpers, fetch)
		if err != nil {
			aiDeps.ConfigErr = err
		} else {
			aiDeps.Drafter = drafter
		}
	}
	s.aiDeps = aiDeps

	// Create chi router for the REST API, health endpoints, and embedded UI.
	// Chi provides clean route grouping: API routes get auth middleware,
	// health endpoints and UI are open.
	r := chi.NewRouter()

	// Global middleware: body size cap, security headers, then CORS
	r.Use(middleware.MaxBodyBytes(cfg.MaxBodyBytes))
	r.Use(middleware.SecurityHeaders(ui.InlineScriptHashes()))
	corsMiddleware, corsMode := middleware.CORS(cfg.CORSAllowedOrigins, cfg.IsProduction())
	logCORSMode(corsMode, cfg.CORSAllowedOrigins)
	r.Use(corsMiddleware)
	r.Use(otelhttp.NewMiddleware("sparrow"))

	// REST API — protected by API key auth. Huma registers every /v1
	// operation plus /openapi.{json,yaml} and the Scalar reference at /docs.
	r.Group(func(r chi.Router) {
		r.Use(auth.HTTPMiddleware)
		rest.Mount(r, tracedWebhookService, rest.AccessDeps{Service: accessSvc, AuthEnabled: auth.Enabled, TokenDefaultTTL: cfg.TokenDefaultTTL, PortalRetry: cfg.PortalRetry}, aiDeps, s.revocation)
	})

	// Invite redemption: the invite in the request is the credential, so it
	// sits outside the authenticated /v1 group.
	r.Post("/invite/redeem", httpauth.RedeemHandler(accessSvc).ServeHTTP)

	// Consumer portal API — one static public prefix. The gateway verifies the
	// consumer-scoped bearer token, maps /portal/api/<rest> to its real /v1
	// path, and re-dispatches into the router so the same handlers run. The
	// consumer is carried by the token, never the URL, so an operator exposing
	// the portal allowlists just /portal, /_app, and /portal/api.
	// SPARROW_PORTAL_RETRY=false refuses delivery retries there.
	r.Handle("/portal/api/*", middleware.PortalGateway(middleware.NewPortalVerifier(accessSvc, accessauth.Realm()), middleware.PortalOptions{Retry: cfg.PortalRetry}, r))

	// Health and readiness endpoints bypass API key auth.
	s.health = health.NewChecker(dbPool, startTime)
	r.Get("/health", s.health.HealthHandler())
	r.Get("/ready", s.health.ReadyHandler())

	// Prometheus scrape endpoint for every OTel metric. Like /health it sits
	// outside API key auth: scrapers rarely carry credentials, and it exposes
	// only aggregate counts.
	if h := observability.MetricsHandler(); h != nil {
		r.Method(http.MethodGet, "/metrics", h)
	}

	// Serve embedded web UI if enabled.
	// The UI handler is registered as the NotFound handler so it acts as
	// a catch-all for paths that don't match any API or health route.
	// Chi's explicit routes always take precedence — API requests can
	// never accidentally be served HTML by the SPA.
	if cfg.ServeUI {
		if ui.Available() {
			// The UI is served as built — the API key is never written into
			// its pages. Operators sign in with the key (swapped for a
			// browser token), an access token, or an invite link.
			uiHandler := ui.Handler(slog.Default())
			r.NotFound(func(w http.ResponseWriter, r *http.Request) {
				// Serve SPA only for GET/HEAD requests. Non-GET to unknown
				// paths returns a JSON 404 so API clients never get HTML.
				if r.Method != http.MethodGet && r.Method != http.MethodHead {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"error":"not found"}`))
					return
				}
				uiHandler.ServeHTTP(w, r)
			})
			fmt.Println("🖥️  Embedded web UI enabled at http://localhost:" + cfg.HTTPPort + "/")
		} else {
			fmt.Println("⚠️  SPARROW_SERVE_UI=true but no frontend build found. Build with: cd web && npm run build")
		}
	}

	s.handler = r
	built = true
	return s, nil
}

// WithTenant returns a context whose requests Sparrow serves for tenantID.
// An embedding program sets it in its own middleware, before calling
// Handler, once it has decided which tenant a request belongs to. Requests
// without it run against the default tenant, exactly as a self-hosted
// Sparrow does.
func WithTenant(ctx context.Context, tenantID uuid.UUID) context.Context {
	return tenant.WithTenant(ctx, tenantID)
}

// CreateTenant creates a tenant, if it does not exist yet, with Sparrow's
// own sparrow.* system event types, so requests run under WithTenant for it
// work like those of a fresh self-hosted Sparrow. It is idempotent.
func (s *Server) CreateTenant(ctx context.Context, tenantID uuid.UUID, name string) error {
	if tenantID == uuid.Nil {
		return errors.New("create tenant: tenant id is required")
	}
	if err := s.repo.CreateTenant(ctx, tenantID, name); err != nil {
		return fmt.Errorf("create tenant: %w", err)
	}
	return registerSystemEventTypes(tenant.WithTenant(ctx, tenantID), s.repo)
}

// PurgeTenant erases a tenant and everything it owns: its queued jobs, its
// webhooks, subscriptions, events, deliveries, health history, event types,
// alert configs, batch jobs, access tokens and invites. Stop routing the
// tenant's requests to Sparrow first. It refuses the default tenant, and it
// is safe to run again if it fails part way. Rows already in database
// backups age out with those backups.
func (s *Server) PurgeTenant(ctx context.Context, tenantID uuid.UUID) error {
	if tenantID == uuid.Nil || tenantID == tenant.DefaultTenantID {
		return errors.New("purge tenant: refusing to purge the default tenant")
	}
	if _, err := s.queue.CancelTenantJobs(ctx, tenantID); err != nil {
		return fmt.Errorf("purge tenant: %w", err)
	}
	if err := s.repo.PurgeTenant(ctx, tenantID); err != nil {
		return err
	}
	if _, _, err := s.access.PurgeRealm(ctx, tenantID.String()); err != nil {
		return fmt.Errorf("purge tenant access tokens: %w", err)
	}
	return nil
}

// Addr is the listen address Sparrow is configured for (SPARROW_HTTP_PORT),
// in the form ":8080".
func (s *Server) Addr() string { return ":" + s.cfg.HTTPPort }

// ShutdownTimeout is the configured budget for a graceful shutdown
// (SPARROW_SHUTDOWN_TIMEOUT).
func (s *Server) ShutdownTimeout() time.Duration { return s.cfg.ShutdownTimeoutOrDefault() }

// Handler returns Sparrow's full HTTP handler: the REST API, docs, portal
// gateway, invite redemption, health, metrics and, when enabled, the UI.
func (s *Server) Handler() http.Handler { return s.handler }

// Start starts the queue workers, periodic jobs and the CRL refresh listener.
func (s *Server) Start(ctx context.Context) error {
	if err := s.queue.Start(ctx); err != nil {
		return fmt.Errorf("failed to start queue manager: %w", err)
	}
	fmt.Println("🚀 River queue started successfully")

	if s.revocation != nil {
		listenCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
		s.stopRevocation = cancel
		go s.revocation.Listen(listenCtx)
	}

	s.printSummary()
	return nil
}

// Drain marks the server as draining: /ready starts failing so load
// balancers stop sending traffic while in-flight work finishes. Call it
// before shutting down the HTTP server.
func (s *Server) Drain() { s.health.SetDraining() }

// Shutdown stops the CRL listener and the queue (which closes its database
// pool), then closes the remaining database pool and flushes telemetry. Shut
// the HTTP server down first so no request is still using Sparrow.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.stopRevocation != nil {
		// Release the CRL refresh listener's dedicated database connection.
		s.stopRevocation()
	}
	var errs []error
	if err := s.queue.Stop(ctx); err != nil {
		errs = append(errs, fmt.Errorf("queue shutdown: %w", err))
	}
	if err := s.db.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close database: %w", err))
	}
	shutdownObservability(s.otelShutdown)
	return errors.Join(errs...)
}

// printSummary prints the startup summary the sparrow-server binary has
// always shown.
func (s *Server) printSummary() {
	cfg := s.cfg
	fmt.Println("🎯 HTTP Queue Server is running...")
	fmt.Printf("   REST API: localhost:%s\n", cfg.HTTPPort)
	fmt.Printf("   API docs: http://localhost:%s/docs\n", cfg.HTTPPort)
	fmt.Printf("   Health check: http://localhost:%s/health\n", cfg.HTTPPort)
	fmt.Printf("   Readiness check: http://localhost:%s/ready\n", cfg.HTTPPort)
	if cfg.ServeUI && ui.Available() {
		fmt.Printf("   Web UI: http://localhost:%s/\n", cfg.HTTPPort)
	}
	if s.authEnabled {
		fmt.Println("   Auth: API key or access token required (X-API-Key or Authorization: Bearer)")
	} else {
		fmt.Println("   Auth: disabled (set SPARROW_API_KEY to enable)")
	}
	switch {
	case s.aiDeps.Drafter != nil:
		fmt.Printf("   AI drafting: enabled (%s via %s)\n", cfg.AIModelOrDefault(), cfg.AIProviderName())
	case s.aiDeps.ConfigErr != nil:
		fmt.Printf("   AI drafting: enabled but not usable: %v\n", s.aiDeps.ConfigErr)
	default:
		fmt.Println("   AI drafting: off (set SPARROW_AI_ENABLED=true to draft in place; the copy-and-paste prompt works either way)")
	}
	if s.otelShutdown != nil {
		fmt.Printf("   OTLP endpoint: %s\n", s.otlpEndpoint)
	}
}

// loadConfig reads and validates configuration from the environment.
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	for _, warn := range cfg.Warnings() {
		log.Printf("⚠️  %s", warn)
	}
	return cfg, nil
}

// setupObservability initializes OpenTelemetry and the default slog logger.
// It returns a nil shutdown function when OpenTelemetry could not be set up.
func setupObservability(ctx context.Context, cfg *config.Config) (func(context.Context) error, *observability.Config) {
	otelConfig := observability.DefaultConfig()
	if cfg.Environment != "" {
		otelConfig.Environment = cfg.Environment
	}
	if cfg.OTLPEndpoint != "" {
		otelConfig.OTLPEndpoint = cfg.OTLPEndpoint
	}
	otelConfig.OTLPProtocol = cfg.OTLPProtocol
	otelConfig.OTLPSignals = cfg.OTLPSignals
	otelConfig.Prometheus = cfg.MetricsEnabled

	// Initialize OpenTelemetry (no-op when OTEL_EXPORTER_OTLP_ENDPOINT is unset)
	otelShutdown, err := observability.Setup(ctx, otelConfig)
	if err != nil {
		log.Printf("⚠️  Failed to setup OpenTelemetry: %v", err)
		fmt.Println("🚀 Continuing without OpenTelemetry...")
		otelShutdown = nil
	} else {
		if otelConfig.OTLPEndpoint != "" {
			fmt.Printf("🔭 OpenTelemetry enabled (endpoint: %s, protocol: %s, signals: %s, env: %s)\n",
				otelConfig.OTLPEndpoint, otelConfig.OTLPProtocol, strings.Join(otelConfig.OTLPSignals, ","), otelConfig.Environment)
		} else {
			fmt.Println("🔭 OpenTelemetry export disabled (set OTEL_EXPORTER_OTLP_ENDPOINT to enable)")
		}
		if otelConfig.Prometheus {
			fmt.Println("📈 Prometheus metrics at /metrics (set SPARROW_METRICS_ENABLED=false to disable)")
		}
	}

	// Initialize structured logging with OTel bridge. The bridge defers to
	// the JSON handler's level, so records below SPARROW_LOG_LEVEL are
	// dropped before any formatting or export.
	logLevel, _ := cfg.SlogLevel() // checked by cfg.Validate
	slog.SetDefault(slog.New(slogotel.OtelHandler{
		Next: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel, AddSource: true}),
	}))
	return otelShutdown, otelConfig
}

func shutdownObservability(otelShutdown func(context.Context) error) {
	if otelShutdown == nil {
		return
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := otelShutdown(shutdownCtx); err != nil {
		log.Printf("Failed to shutdown OpenTelemetry: %v", err)
	}
}

// runMigrations runs database migrations before anything else touches the DB.
// This covers both River queue schema and application schema migrations.
// golang-migrate uses PostgreSQL advisory locks, so concurrent server
// instances won't conflict.
func runMigrations(ctx context.Context, cfg *config.Config) error {
	fmt.Println("📦 Running database migrations...")
	if err := migration.RunAllMigrations(ctx, cfg.DatabaseURL, "up", 0, 0, slog.Default()); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}
	fmt.Println("✅ Database migrations completed")
	return nil
}

// logCORSMode prints the cross-origin policy chosen by middleware.CORS.
func logCORSMode(mode middleware.CORSMode, origins []string) {
	switch mode {
	case middleware.CORSAllowList:
		fmt.Printf("🔒 CORS allowed origins: %v\n", middleware.NormalizeOrigins(origins))
	case middleware.CORSBlockAll:
		fmt.Println("🔒 CORS: production mode — cross-origin requests blocked")
		fmt.Println("   If the UI is hosted separately, set SPARROW_CORS_ALLOWED_ORIGINS to the UI origin")
	case middleware.CORSAllowAll:
		fmt.Println("⚠️  SPARROW_CORS_ALLOWED_ORIGINS not set — allowing all origins (development mode, not for production)")
	}
}

// registerSystemEventTypes idempotently registers the event types Sparrow
// generates about its own webhooks, with their JSON schemas and sample
// payloads (see queue.SystemEventRegistrations). Tenants opt an email address
// into them via the webhook_alert_configs API; they are never delivered
// directly, only fanned out through the normal event/subscription/delivery
// pipeline under the "_sparrow" consumer.
//
// The tenant comes from ctx (the default tenant unless set with WithTenant).
func registerSystemEventTypes(ctx context.Context, repo store.EventTypeRepository) error {
	tenantID := tenant.FromContext(ctx)
	var errs []error
	for _, t := range queue.SystemEventRegistrations() {
		existing, err := repo.GetEventByName(ctx, tenantID, t.Name)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to look up system event type %s: %w", t.Name, err))
			continue
		}
		if existing != nil {
			continue
		}
		reg := t
		if err := repo.RegisterEvent(ctx, tenantID, &reg); err != nil {
			errs = append(errs, fmt.Errorf("failed to register system event type %s: %w", t.Name, err))
		}
	}
	return errors.Join(errs...)
}
