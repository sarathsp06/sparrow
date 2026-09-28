# Graph Report - svix-webhooks-security-review-fdc8e0  (2026-09-28)

## Corpus Check
- 415 files · ~367,136 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3301 nodes · 6988 edges · 235 communities (193 shown, 42 thin omitted)
- Extraction: 85% EXTRACTED · 15% INFERRED · 0% AMBIGUOUS · INFERRED: 1035 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `767fed4a`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- setupEnv
- event.go
- Token
- event_filtering_test.go
- subscription.go
- RepositoryInterfaceWithTracing
- delivery.go
- steps.py
- rest/webhook.go
- sparrow_verify.py
- WebhookHTTPConfig
- NewService
- WebhookServiceInterfaceWithTracing
- Sparrow Detailed Flow Reference
- Context
- steps_surface.py
- sparrow-verify.ts
- New
- api-types.d.ts
- TemplateCache
- NewWebhookHandler
- api.astro
- WebhookWorker
- store/models.go
- newRootCmd
- Dual Protocol (gRPC + Connect-RPC)
- Client Libraries
- BatchJob
- Error Classification (Reference)
- Template Functions
- apiClient
- Errorf
- Sparrow Architecture
- JobInserterWithTracing
- EventProcessingWorker
- ClassifyError
- envelope
- signature_test.go
- WebhookDelivery
- sparrow_verify.rs
- jobInserter
- webhook_service.go
- lib/utils.ts
- registerWebhookRoutes
- newAuth
- dependencies
- NewWebhookClient
- TemplateEngine
- NewTemplateEngine
- apiConsole.svelte.ts
- runPush
- newEmailSink
- compose.ts
- Service
- All
- Commands
- Context
- postSink
- Store
- newOTLPSink
- devDependencies
- newPalette
- run
- Client
- EventRecord
- SparrowAPI
- RepositoryInterface
- TestRecipes
- Handler
- ServiceError
- run
- newS3Sink
- sinks.mdx
- .CreateWebhook
- Real-world examples
- pushEnv
- mockRepo
- Included recipes
- sources.mdx
- ADDED Requirements
- Release GoReleaser Job
- portal/+page.svelte
- access/service_test.go
- reference/security.mdx
- RetentionWorker
- ADDED Requirements
- compilerOptions
- WebhookTargetManager
- deployment/security.mdx
- sparrow/access.go
- GetFunctionMap
- Sparrow Recipes
- palette
- runFunctions
- NewManager
- BenchmarkTransformPayload
- _Target
- Checker
- WebhookService
- Recipe
- testContext
- hooks.py
- SparrowEnvironment
- index.mdx
- 2. Split the `sparrow` CLI into its own module to isolate its `go install` graph
- parseUUID
- 1. Payload transform engine: Go `text/template`, not embedded JavaScript
- consumer.svelte.ts
- runTemplateTest
- services.ts
- release-submodules.sh
- mapError
- steps_jobs.py
- proposal.md
- newPortalGateway
- Feature: Webhook health/delivery-failure email alerts
- tasks.md
- portal-embedding.mdx
- webhook-health-alerts.mdx
- design.md
- api.ts
- Decision
- auth.svelte.ts
- Sparrow Webhook Delivery Platform
- scripts
- models_test.go
- steps_auth.py
- CI Build Job
- EventRegistration
- typescript
- ParseNetworks
- web/package.json
- svelte
- PrepareDeliveryRequest
- test-ui.sh
- manifest.json
- gen-og.mjs
- opencode.json
- Deliberately NOT covered by e2e
- vite-plugin-devtools-json
- .loadAndValidateBatch
- Sparrow Web Dashboard
- @sveltejs/kit
- steps_access.py
- otel.go
- httpauth.go
- rest/access.go
- corsChain
- access.md
- parseDateFilter
- NetworkPolicy
- NewWebhookTemplateContext
- FromContext
- svelte-check
- @sveltejs/vite-plugin-svelte
- HandlerFunc
- src/components/Footer.astro
- docs/tsconfig.json
- TestJobInserter_InsertOpts_Merge
- TestWebhookWorkerNextRetry
- Dual Webhook Signing (HMAC-SHA256 + Ed25519)
- +layout.ts
- WithConn Transaction Pattern
- SSRF Protection
- @playwright/test
- ../../components/ThemeDiagram.astro
- content.config.ts
- proto2astro
- index.astro
- newAlertConfigService
- Context
- fakeServer
- WebhookService
- Context
- production.md
- svelte.config.js
- validConfig
- access/+page.svelte
- pushSystemEvent
- separate-ui.md
- newTailCmd
- ValidateHeaders
- SparrowVerify
- WebhookClient
- PortalGateway
- TestModuleImportsOnlyStdlibAndItself
- dev-env.sh
- static-server.mjs
- Store
- Status
- Mount
- GetBuffer
- .VerifyEd25519
- SparrowVerify
- SparrowVerify
- SparrowVerify
- Config
- SparrowVerify
- buildVectors
- verify-signatures.mdx
- resolveConfig
- TestTemplateTestRendersFixtureRecipe
- TokenPurgeWorker
- Invite
- TestVectors
- listenHandler
- runUse
- runInit
- Run
- health.go
- @sveltejs/adapter-static
- parseRetryAfter
- use_test.go
- TestParseJSONArg
- listFlag
- LayeredArchEmbed.astro
- tailwindcss
- generator.ts
- Release Workflow
- github.com/sarathsp06/sparrow
- sparrow-e2e
- Envelope Encryption at Rest (AES-256-GCM)

## God Nodes (most connected - your core abstractions)
1. `Errorf()` - 186 edges
2. `New()` - 100 edges
3. `RepositoryInterfaceWithTracing` - 74 edges
4. `WebhookServiceInterfaceWithTracing` - 43 edges
5. `setupEnv()` - 40 edges
6. `EventSubscription` - 40 edges
7. `testContext()` - 39 edges
8. `_base()` - 33 edges
9. `NewService()` - 32 edges
10. `WebhookDelivery` - 31 edges

## Surprising Connections (you probably didn't know these)
- `Svelte 5 Tutorial (PDF)` --conceptually_related_to--> `Sparrow Webhook Delivery Platform`  [INFERRED]
  book/svelte5-tutorial.pdf → README.md
- `Development Docker Compose` --semantically_similar_to--> `Standalone Docker Compose`  [INFERRED] [semantically similar]
  docker-compose.dev.yml → deploy/docker-compose.yml
- `main()` --calls--> `RunAllMigrations()`  [INFERRED]
  cmd/migrate/main.go → internal/migration/migrate.go
- `main()` --calls--> `Mount()`  [INFERRED]
  cmd/openapi-export/main.go → internal/rest/app.go
- `TokenTTL()` --calls--> `Errorf()`  [INFERRED]
  internal/accessauth/accessauth.go → pkg/errors/service.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **CI Pipeline Stage Dependency Chain** — github_workflows_ci_lint_job, github_workflows_ci_test_job, github_workflows_ci_build_job, github_workflows_ci_integration_job [EXTRACTED 1.00]
- **River-backed async delivery pipeline** — concept_river_queue, concept_event_processing_worker, concept_webhook_worker, concept_postgresql [EXTRACTED 1.00]
- **Sparrow RPC service surface** — concept_webhook_service, concept_event_service, concept_subscription_service, concept_delivery_service, concept_health_service [EXTRACTED 1.00]
- **Sparrow security feature set** — concept_envelope_encryption, concept_standard_webhooks_signing, concept_ssrf_protection [EXTRACTED 0.95]

## Communities (235 total, 42 thin omitted)

### Community 0 - "setupEnv"
Cohesion: 0.06
Nodes (89): Container, capturedWebhook, deliveryItem, restClient, sinkSMTPMessage, testEnv, buildCLI(), T (+81 more)

### Community 1 - "event.go"
Cohesion: 0.10
Nodes (33): API, Context, listEventOccurrencesImpl(), registerEventRoutes(), toBatchJobOutput(), toEventTypeItem(), toEventTypeOutput(), newPagination() (+25 more)

### Community 2 - "Token"
Cohesion: 0.17
Nodes (16): Token, NullString, NullTime, execer, scanner, Store, Context, Result (+8 more)

### Community 3 - "event_filtering_test.go"
Cohesion: 0.16
Nodes (29): Context, T, UUID, TestGetSubscriptionsByEvent_CatchAllReturned(), TestGetSubscriptionsByEvent_ConsumerIsolation(), TestGetSubscriptionsByEvent_EmptyLabelFiltersMatchAll(), TestGetSubscriptionsByEvent_LabelFiltering(), TestGetSubscriptionsByEvent_LabelMismatchFiltered() (+21 more)

### Community 4 - "subscription.go"
Cohesion: 0.17
Nodes (20): API, Context, listSubscriptionsImpl(), registerSubscriptionRoutes(), toSubscriptionItem(), toSubscriptionOutput(), createSubscriptionBody, createSubscriptionInput (+12 more)

### Community 5 - "RepositoryInterfaceWithTracing"
Cohesion: 0.11
Nodes (10): Context, Span, Time, UUID, WebhookRegistration, NewRepositoryInterfaceWithTracing(), ConsumerStats, EventSubscription (+2 more)

### Community 6 - "delivery.go"
Cohesion: 0.17
Nodes (20): API, Context, listDeliveriesImpl(), registerDeliveryRoutes(), toDeliveryItem(), toDeliveryOutput(), attemptItem, attemptsOutput (+12 more)

### Community 7 - "steps.py"
Cohesion: 0.08
Nodes (59): delivery_has_signature_headers(), SignatureVerifier -- Verifies HMAC-SHA256 (v1,) and Ed25519 (v1a,) signatures…, Return the exact body bytes (as text) that Sparrow signed. Standard Webhooks…, Verify HMAC-SHA256 signature (v1, prefix)., Verify Ed25519 signature (v1a, prefix)., Assert delivery has Standard Webhooks signature headers., _signed_body(), verify_ed25519_signature() (+51 more)

### Community 8 - "rest/webhook.go"
Cohesion: 0.20
Nodes (12): consumerOnlyInput, consumerStatsOutput, emptyOutput, listWebhooksInput, patchWebhookBody, patchWebhookInput, registerWebhookBody, registerWebhookInput (+4 more)

### Community 9 - "sparrow_verify.py"
Cohesion: 0.32
Nodes (11): _check_timestamp(), _decode_signatures(), Exception, Verify Sparrow webhook delivery signatures (Standard Webhooks format). Every…, Raised when a delivery signature cannot be verified., Verify the ``v1,`` (HMAC-SHA256) signature. Raises on failure. ``payload`` must…, Verify the ``v1a,`` (Ed25519) signature. Raises on failure. ``payload`` must be…, _required_headers() (+3 more)

### Community 10 - "WebhookHTTPConfig"
Cohesion: 0.11
Nodes (14): DefaultWebhookHTTPConfig(), Duration, Time, Value, T, TestApplyConfig_RateLimitRPS(), TestDefaultWebhookHTTPConfig_RateLimitRPS(), TestToWebhookRegistration_RateLimitRPS() (+6 more)

### Community 11 - "NewService"
Cohesion: 0.09
Nodes (47): AEAD, kek, Key, Keyring, Service, buildKEK(), decryptDataWithDEK(), isSafeKeyID() (+39 more)

### Community 12 - "WebhookServiceInterfaceWithTracing"
Cohesion: 0.06
Nodes (15): Context, Repository, Time, UUID, Time, Context, Span, Time (+7 more)

### Community 13 - "Sparrow Detailed Flow Reference"
Cohesion: 0.09
Nodes (33): Sparrow Client Libraries README, Connect-RPC, DeliveryService, Envelope Encryption (AES-256-GCM), 10-Category Error Classification, EventProcessingWorker, EventService, gRPC (+25 more)

### Community 14 - "Context"
Cohesion: 0.13
Nodes (19): T, TestE2E_EventDeliveryStats_NoDeliveries(), Context, Repository, Tx, UUID, WebhookRegistration, NewRepository() (+11 more)

### Community 15 - "steps_surface.py"
Cohesion: 0.16
Nodes (43): alert_configs_count(), alert_configs_count_for_webhook(), also_use_consumer(), assert_delivery_page_meta(), _base(), create_alert_config_bad_webhook(), create_alert_config_scoped(), create_alert_config_wide() (+35 more)

### Community 16 - "sparrow-verify.ts"
Cohesion: 0.39
Nodes (8): decodeBase64Strict(), decodeSignatures(), DEFAULT_TOLERANCE_SECONDS, Headers, parseHeaders(), SignatureVerificationError, verifyEd25519(), verifyHmac()

### Community 17 - "New"
Cohesion: 0.31
Nodes (17): New(), T, newTestWorker(), TestEmitDeliveryFailedEvent_EmitsOnRecipientLookupError(), TestEmitDeliveryFailedEvent_EmitsWhenRecipientsOptedIn(), TestEmitDeliveryFailedEvent_EmitsWithZeroRecipients(), TestEmitDeliveryFailedEvent_SkipsSparrowConsumer(), TestEmitHealthChangedEvent_EmitsOnRealTransition() (+9 more)

### Community 18 - "api-types.d.ts"
Cohesion: 0.29
Nodes (6): RFC-3339, components, $defs, operations, paths, webhooks

### Community 19 - "TemplateCache"
Cohesion: 0.24
Nodes (10): Cache, Template, hashTemplate(), NewTemplateCache(), T, TestHashTemplate(), TestTemplateCacheBasicOperations(), TestTemplateCacheConcurrency() (+2 more)

### Community 20 - "NewWebhookHandler"
Cohesion: 0.07
Nodes (50): main(), RawMessage, SparrowConfig, LoadConfig(), cronTick(), Context, Logger, Time (+42 more)

### Community 22 - "WebhookWorker"
Cohesion: 0.11
Nodes (22): Config, Context, Duration, Job, JobInserter, Logger, WebhookWorker, Service (+14 more)

### Community 23 - "store/models.go"
Cohesion: 0.13
Nodes (13): Int64Array, Time, UUID, Value, DeliveryFilter, JSONMap, JSONStringMap, SignatureType (+5 more)

### Community 24 - "newRootCmd"
Cohesion: 0.14
Nodes (24): newInviteCmd(), addOutputFlag(), clientFromCmd(), apiClient, Command, config, Writer, newRootCmd() (+16 more)

### Community 27 - "BatchJob"
Cohesion: 0.20
Nodes (8): Context, Repository, UUID, RawMessage, BatchJob, BatchJobData, BatchJobStatus, BatchJobType

### Community 32 - "apiClient"
Cohesion: 0.15
Nodes (15): config, Context, apiClient, newAPIClient(), apiError, consumerStats, deliveryItem, eventTypeItem (+7 more)

### Community 33 - "Errorf"
Cohesion: 0.19
Nodes (8): generateSamplePayload(), Context, Time, WebhookService, ValidateJSONSchema(), normalizePagination(), Errorf(), SchemaValidationError

### Community 34 - "Sparrow Architecture"
Cohesion: 0.29
Nodes (7): Sparrow Architecture, Dual-Protocol API, Event Processing Pipeline, Health State Machine, Leaky Bucket Rate Limiting, Standard Webhooks Signing, Default Webhook Body Envelope

### Community 35 - "JobInserterWithTracing"
Cohesion: 0.31
Nodes (7): Context, JobArgs, JobInserter, JobInsertResult, Span, NewJobInserterWithTracing(), JobInserterWithTracing

### Community 36 - "EventProcessingWorker"
Cohesion: 0.19
Nodes (10): Context, Job, JobInserter, Logger, WorkerDefaults, NewEventProcessingWorker(), Time, EventArgs (+2 more)

### Community 37 - "ClassifyError"
Cohesion: 0.16
Nodes (23): ErrorCategory, timeoutError, classifyByMessage(), ClassifyError(), ClassifyHTTPStatus(), classifySyscallError(), isDNSError(), IsRetryableCategory() (+15 more)

### Community 38 - "envelope"
Cohesion: 0.18
Nodes (12): Context, Template, config, Context, Logger, RawMessage, sinkHandler(), templateContext() (+4 more)

### Community 39 - "signature_test.go"
Cohesion: 0.50
Nodes (8): T, signHMAC(), TestMissingHeaders(), TestTimestampTolerance(), TestVerifyEd25519(), TestVerifyHMAC(), TestVerifyHMACMultiSignatureHeader(), TestVerifyHMACRawSecret()

### Community 40 - "WebhookDelivery"
Cohesion: 0.18
Nodes (5): Context, Repository, UUID, WebhookDelivery, WebhookDeliveryStatus

### Community 41 - "sparrow_verify.rs"
Cohesion: 0.25
Nodes (17): hex_decode(), hex_nibble(), parse_headers(), Result, SignatureError, signed_message(), verify_ed25519(), verify_ed25519_at() (+9 more)

### Community 42 - "jobInserter"
Cohesion: 0.19
Nodes (9): Context, InsertOpts, JobArgs, JobInsertResult, Logger, Tx, NewJobInserter(), BatchJobArgs (+1 more)

### Community 43 - "webhook_service.go"
Cohesion: 0.20
Nodes (17): IPNet, WithAllowedNetworks(), WithAllowPrivateNetworks(), eventRouteService, healthRouteService, webhookRouteService, AlertConfigManager, BatchManager (+9 more)

### Community 44 - "lib/utils.ts"
Cohesion: 0.13
Nodes (8): $lib/api-types, ERROR_CATEGORIES, getCategoryBadge(), getCategoryDisplay(), inferType(), JSONSchemaMetaSchema, jsonToJsonSchema(), RFC-9457

### Community 45 - "registerWebhookRoutes"
Cohesion: 0.19
Nodes (17): getWebhookEventsMap(), Context, WebhookRegistration, maskEncryptedSecret(), maskSecret(), maskSecretHeaders(), toWebhookOut(), toWebhookOutFromDomain() (+9 more)

### Community 46 - "newAuth"
Cohesion: 0.27
Nodes (17): NewAuth(), apiKey(), bearer(), Request, Service, T, TB, guarded() (+9 more)

### Community 47 - "dependencies"
Cohesion: 0.05
Nodes (36): @astrojs/sitemap, @astrojs/starlight, @astrojs/svelte, dependencies, astro, @astrojs/sitemap, @astrojs/starlight, @astrojs/svelte (+28 more)

### Community 48 - "NewWebhookClient"
Cohesion: 0.23
Nodes (19): NewWebhookClient(), ReadBody(), BenchmarkSend(), B, T, TestClientClose(), TestDeliverySpansDoNotExportSecretURLs(), TestNewWebhookClient() (+11 more)

### Community 49 - "TemplateEngine"
Cohesion: 0.16
Nodes (10): getBuffer(), Buffer, putBuffer(), FuncMap, Template, WebhookTemplateContext, NewTemplateEngineWithCacheSize(), limitedWriter (+2 more)

### Community 50 - "NewTemplateEngine"
Cohesion: 0.27
Nodes (17): NewTemplateEngine(), BenchmarkExecuteComplex(), BenchmarkExecuteSimple(), B, T, TestExecuteComplexTemplate(), TestExecuteEmptyTemplate(), TestExecuteInvalidTemplate() (+9 more)

### Community 51 - "apiConsole.svelte.ts"
Cohesion: 0.10
Nodes (11): apiConsole, ApiLogEntry, apiLogMiddleware, entries, started, toCurl(), copy(), beat (+3 more)

### Community 52 - "runPush"
Cohesion: 0.19
Nodes (8): parseJSONArg(), apiClient, Command, Context, Writer, newPushCmd(), runPush(), kvFlag

### Community 53 - "newEmailSink"
Cohesion: 0.37
Nodes (12): Conn, newEmailSink(), T, serveSMTP(), startSMTPServer(), TestEmailRender_BadTemplate(), TestEmailRender_CustomTemplatesAndHeaderInjection(), TestEmailRender_Defaults() (+4 more)

### Community 54 - "compose.ts"
Cohesion: 0.06
Nodes (41): selectPreset(), switchRecipe(), DiscordCompose, emailArray(), EmailCompose, ENVELOPE_PATHS, generateDiscordTemplate(), generateNtfyTemplate() (+33 more)

### Community 55 - "Service"
Cohesion: 0.14
Nodes (19): cacheEntry, Config, CreateInviteRequest, CreateTokenRequest, Principal, RootKey, Service, invalidRequest() (+11 more)

### Community 56 - "All"
Cohesion: 0.25
Nodes (6): API, Recipe, registerRecipeRoutes(), listRecipesOutput, All(), Recipe

### Community 57 - "Commands"
Cohesion: 0.08
Nodes (23): 90-second quickstart, Commands, Configuration, How it's connected, Install, Real-World Use Cases, Scenario 1: Zero-Config Local Webhook Receiver & Debugging, Scenario 2: CI/CD Pipeline Automation & Deployment Notifications (+15 more)

### Community 58 - "Context"
Cohesion: 0.30
Nodes (4): Context, Repository, UUID, SubscriptionWithWebhook

### Community 59 - "postSink"
Cohesion: 0.40
Nodes (12): Context, Header, ResponseRecorder, T, postSink(), signHeaders(), TestSinkHandler_DownstreamFailureIs502(), TestSinkHandler_MissingHeaders() (+4 more)

### Community 60 - "Store"
Cohesion: 0.26
Nodes (7): Store, Context, Mutex, Time, New(), newer(), sameScope()

### Community 61 - "newOTLPSink"
Cohesion: 0.27
Nodes (10): Context, logsURL(), newOTLPSink(), otlpPayload(), T, TestLogsURL(), TestOTLPSinkDeliver(), TestOTLPSinkDeliverDownstreamFailure() (+2 more)

### Community 62 - "devDependencies"
Cohesion: 0.15
Nodes (13): openapi-typescript, @tailwindcss/forms, @tailwindcss/typography, @tailwindcss/vite, @types/node, vite, devDependencies, openapi-typescript (+5 more)

### Community 63 - "newPalette"
Cohesion: 0.14
Nodes (19): eventTypeItem, Writer, newPalette(), apiClient, Context, Writer, indentJSON(), printEventType() (+11 more)

### Community 64 - "run"
Cohesion: 0.50
Nodes (4): Context, Writer, main(), run()

### Community 65 - "Client"
Cohesion: 0.27
Nodes (7): Context, RawMessage, SparrowConfig, NewClient(), T, TestClientAutoCreatesEventType(), Client

### Community 66 - "EventRecord"
Cohesion: 0.18
Nodes (7): Context, Repository, Time, UUID, EventRecord, EventReportFilter, EventReportWithStats

### Community 67 - "SparrowAPI"
Cohesion: 0.14
Nodes (7): SparrowAPI -- wraps the generated REST client (sparrow_client, generated from…, Poll until all deliveries reach terminal status., Poll a single delivery until terminal., Convert a generated attrs model (or None) to a plain dict., Client for Sparrow's REST API, built on the generated sparrow_client SDK., SparrowAPI, _to_dict()

### Community 68 - "RepositoryInterface"
Cohesion: 0.13
Nodes (15): Context, Job, JobInserter, Logger, UUID, WorkerDefaults, NewBatchJobWorker(), BatchJobWorker (+7 more)

### Community 69 - "TestRecipes"
Cohesion: 0.39
Nodes (8): T, WebhookTemplateContext, sampleContext(), substituteParams(), TestPagerdutyRecipe_Severity(), TestRecipes(), TestSendGridActivationParams(), tokenRefs()

### Community 70 - "Handler"
Cohesion: 0.22
Nodes (14): computeInlineScriptHashes(), FS, Logger, Handler(), InlineScriptHashes(), inlineScriptHashesFromFS(), newHandler(), Response (+6 more)

### Community 71 - "ServiceError"
Cohesion: 0.20
Nodes (12): ServiceError, Classify(), Error(), Status, T, TestConstructors(), TestServiceError_ClientMessage(), TestServiceError_Error() (+4 more)

### Community 72 - "run"
Cohesion: 0.27
Nodes (8): loadConfig(), Context, Logger, main(), run(), config, emailConfig, smtpConfig

### Community 73 - "newS3Sink"
Cohesion: 0.27
Nodes (8): Context, newS3Sink(), objectKey(), T, TestObjectKey(), TestS3Sink_PutObject(), s3Config, s3Sink

### Community 74 - "sinks.mdx"
Cohesion: 0.14
Nodes (13): Configuration, How it's connected, Install and run, Real-World Use Cases, Retry semantics, Scenario 1: Long-Term Regulatory & Financial Audit Archiving (S3 / MinIO / R2), Scenario 2: Operational Stakeholder Notifications via SMTP Email, Scenario 3: Centralized OpenTelemetry Log & Event Pipeline (OTLP) (+5 more)

### Community 75 - ".CreateWebhook"
Cohesion: 0.14
Nodes (12): DerefBoolOr(), DerefIntOr(), stringHeaders(), ValidateWebhookURL(), generateWebhookSecret(), Context, Time, WebhookRegistration (+4 more)

### Community 76 - "Real-world examples"
Cohesion: 0.13
Nodes (14): 1. Send only the fields a partner needs, 2. Post a message to Slack, 3. Convert dollars to cents for a billing system, 4. Flatten a nested payload for a legacy endpoint, 5. Summarize a list of items, 6. Add a constant or computed field, 7. Provide safe defaults for optional fields, Good to know (+6 more)

### Community 77 - "pushEnv"
Cohesion: 0.33
Nodes (8): T, TestEventsDetailShowsSchema(), TestEventsList(), Server, T, pushEnv(), TestPushAutoCreatesEventType(), TestPushRequestShape()

### Community 78 - "mockRepo"
Cohesion: 0.17
Nodes (8): Context, JobArgs, JobInsertResult, UUID, WebhookRegistration, mockRepo, Mock, mockJobInserter

### Community 79 - "Included recipes"
Cohesion: 0.13
Nodes (14): clickhouse, discord, How it's connected, Included recipes, ntfy, pagerduty, Real-World Use Cases, Scenario 1: Real-Time Incident Alerting with PagerDuty (+6 more)

### Community 80 - "sources.mdx"
Cohesion: 0.17
Nodes (11): Configuration, Delivery semantics, Event naming, GitHub, How it's connected, Install and run, Provider setup, Real-World Use Cases (+3 more)

### Community 81 - "ADDED Requirements"
Cohesion: 0.12
Nodes (16): ADDED Requirements, Purpose, Requirement: Alert-email configuration resource, Requirement: Email delivery through the core pipeline, Requirement: Human-readable alert content, Requirement: Recipient resolution at push time, Scenario: Consumer-level opt-in, Scenario: Degradation subject (+8 more)

### Community 82 - "Release GoReleaser Job"
Cohesion: 0.50
Nodes (4): Conventional Commits Convention, Release Docker Image Job, Release GoReleaser Job, GoReleaser Config

### Community 83 - "portal/+page.svelte"
Cohesion: 0.15
Nodes (24): unwrap(), formatAPIError(), cancelInvite(), create(), revoke(), deliveriesLoading, error, executeDelete() (+16 more)

### Community 84 - "access/service_test.go"
Cohesion: 0.17
Nodes (33): clock, countingStore, xorSealer, Bool, New(), Context, Duration, Int32 (+25 more)

### Community 85 - "reference/security.mdx"
Cohesion: 0.18
Nodes (10): Access tokens, API Authentication, Consumer Isolation, HTTP Hardening, Portal tokens, Production Config Checklist, Secret Masking in Responses, SSRF Protection (+2 more)

### Community 86 - "RetentionWorker"
Cohesion: 0.21
Nodes (8): Context, InsertOpts, Job, Logger, WorkerDefaults, NewRetentionWorker(), RetentionArgs, RetentionWorker

### Community 87 - "ADDED Requirements"
Cohesion: 0.12
Nodes (15): ADDED Requirements, Purpose, Requirement: Feedback-loop guard, Requirement: Health transition event, Requirement: Internal `_sparrow` consumer, Requirement: Registered system event types, Requirement: Terminal delivery failure event, Scenario: Degradation emits one event (+7 more)

### Community 88 - "compilerOptions"
Cohesion: 0.14
Nodes (13): ./.svelte-kit/tsconfig.json, compilerOptions, allowJs, checkJs, esModuleInterop, forceConsistentCasingInFileNames, moduleResolution, resolveJsonModule (+5 more)

### Community 89 - "WebhookTargetManager"
Cohesion: 0.15
Nodes (5): before_scenario, Manages mock webhook target servers., WebhookTargetManager, Fresh target manager for each scenario., setup_scenario()

### Community 90 - "deployment/security.mdx"
Cohesion: 0.18
Nodes (10): Data retention and compliance posture, Hardening checklist, Not a fit, Option A — Authentik (username/password + Entra + Google in one tool), Option B — oauth2-proxy (single IdP, smallest footprint), Option C — Keycloak + oauth2-proxy (maximum boring), Recommended: SSO via an identity-aware proxy, The consumer portal (+2 more)

### Community 91 - "sparrow/access.go"
Cohesion: 0.14
Nodes (23): accessLabel(), Command, Context, Duration, apiClient, Time, Writer, newInvitesCmd() (+15 more)

### Community 92 - "GetFunctionMap"
Cohesion: 0.31
Nodes (7): GetFunctionMap(), GetTemplateFunctions(), FuncMap, T, TestTitleFunc(), toNumber(), TemplateFunc

### Community 93 - "Sparrow Recipes"
Cohesion: 0.33
Nodes (5): Contributing a recipe, How params work, Included recipes, Schema (version 1), Sparrow Recipes

### Community 95 - "runFunctions"
Cohesion: 0.24
Nodes (8): firstLine(), apiClient, Context, Writer, runFunctions(), T, TestFirstLineSkipsHeadings(), TestRenderStructured()

### Community 96 - "NewManager"
Cohesion: 0.20
Nodes (10): Config, Context, JobInserter, Logger, Pool, Service, Tx, NewManager() (+2 more)

### Community 97 - "BenchmarkTransformPayload"
Cohesion: 0.40
Nodes (4): BenchmarkTransformPayload(), B, T, TestTransformPayload()

### Community 98 - "_Target"
Cohesion: 0.23
Nodes (4): CapturedDelivery, WebhookTargetServer -- Programmable mock webhook endpoints for e2e tests. Each…, Start a mock webhook target. Returns the URL., _Target

### Community 99 - "Checker"
Cohesion: 0.32
Nodes (7): Checker, HealthResponse, ReadyResponse, Context, Pool, Time, NewChecker()

### Community 100 - "WebhookService"
Cohesion: 0.24
Nodes (6): validateHeaders(), Context, Time, UUID, WebhookService, paginateSubscriptions()

### Community 101 - "Recipe"
Cohesion: 0.53
Nodes (4): Param, Recipe, Subscription, Webhook

### Community 102 - "testContext"
Cohesion: 0.22
Nodes (24): TestCreateSubscription_CatchAllWithLabelFilters(), TestCreateSubscription_WithEmptyLabelFilters(), TestCreateSubscription_WithInvalidLabelFilters(), TestCreateSubscription_WithLabelFilters(), TestPushEvent_EventNameLengthCountsCharacters(), TestPushEvent_WithInvalidLabels(), TestPushEvent_WithNilLabels(), TestPushEvent_WithValidLabels() (+16 more)

### Community 103 - "hooks.py"
Cohesion: 0.17
Nodes (10): after_scenario, after_suite, before_suite, SparrowEnvironment -- Manages Postgres and Sparrow containers via…, Gauge hooks for suite/scenario setup and teardown., Start Postgres + Sparrow containers., Stop all mock targets., setup_environment() (+2 more)

### Community 104 - "SparrowEnvironment"
Cohesion: 0.27
Nodes (4): Manages the Sparrow test environment (Postgres + Sparrow containers)., Start Postgres + Sparrow. Returns the Sparrow HTTP URL., Start a second Sparrow container with API key auth enabled, reusing Postgres., SparrowEnvironment

### Community 105 - "index.mdx"
Cohesion: 0.33
Nodes (5): Real-World Architecture & Use Cases, Scenario 1: E-Commerce Payment & Order Fulfillment Pipeline, Scenario 2: Developer Operations & Infrastructure Alerting, The contract, The satellites

### Community 106 - "2. Split the `sparrow` CLI into its own module to isolate its `go install` graph"
Cohesion: 0.25
Nodes (7): 2. Split the `sparrow` CLI into its own module to isolate its `go install` graph, Consequences, Context, Decision, Releasing the split modules (tag scheme), Trigger to revisit, When this is worth it

### Community 107 - "parseUUID"
Cohesion: 0.16
Nodes (9): Context, WebhookService, Context, WebhookService, UUID, Context, WebhookService, parseUUID() (+1 more)

### Community 108 - "1. Payload transform engine: Go `text/template`, not embedded JavaScript"
Cohesion: 0.33
Nodes (5): 1. Payload transform engine: Go `text/template`, not embedded JavaScript, Consequences, Context, Decision, Trigger to revisit

### Community 109 - "consumer.svelte.ts"
Cohesion: 0.22
Nodes (5): consumerStore, current, known, Recipe, RecipeParam

### Community 110 - "runTemplateTest"
Cohesion: 0.50
Nodes (4): Command, Writer, newTemplateCmd(), runTemplateTest()

### Community 111 - "services.ts"
Cohesion: 0.09
Nodes (27): browserTokenName(), exchangeKey(), ExchangeResult, fragmentParam(), InviteError, RedeemedInvite, redeemInvite(), rejectMessage() (+19 more)

### Community 112 - "release-submodules.sh"
Cohesion: 0.80
Nodes (4): release_leaf(), release_module(), release-submodules.sh script, tag_exists()

### Community 113 - "mapError"
Cohesion: 0.20
Nodes (12): API, registerAlertConfigRoutes(), toAlertConfigItem(), Context, mapError(), alertConfigIDInput, alertConfigItem, alertConfigOutput (+4 more)

### Community 114 - "steps_jobs.py"
Cohesion: 0.23
Nodes (22): assert_all_terminal_status(), assert_bulk_retry_count(), assert_consumer_stats(), assert_consumer_stats_webhooks(), assert_global_stats(), assert_health_failed_count(), assert_health_summary(), assert_job_progress() (+14 more)

### Community 115 - "proposal.md"
Cohesion: 0.29
Nodes (6): Capabilities, Impact, Modified Capabilities, New Capabilities, What Changes, Why

### Community 116 - "newPortalGateway"
Cohesion: 0.26
Nodes (17): Service, NewPortalVerifier(), doBearer(), Context, Service, Store, T, mintConsumerToken() (+9 more)

### Community 117 - "Feature: Webhook health/delivery-failure email alerts"
Cohesion: 0.33
Nodes (5): Context, Design, Diagrams, Feature: Webhook health/delivery-failure email alerts, Open Questions

### Community 118 - "tasks.md"
Cohesion: 0.33
Nodes (5): 1. Foundation, 2. Event emission, 3. Tenant-facing API, 4. Delivery channel, 5. Verification & docs

### Community 119 - "portal-embedding.mdx"
Cohesion: 0.20
Nodes (9): Alternative: no Sparrow UI at all, Example: Caddy (simplest — start here), Example: Express (your app's backend as the proxy), Example: nginx, How it works, end to end, Prerequisites, Security analysis, The route allowlist (+1 more)

### Community 120 - "webhook-health-alerts.mdx"
Cohesion: 0.40
Nodes (4): Good to know, How it works, Register an alert config, Set up the email delivery channel

### Community 121 - "design.md"
Cohesion: 0.50
Nodes (3): Alternatives rejected, Context, Decisions

### Community 123 - "api.ts"
Cohesion: 0.25
Nodes (8): ensureEventType(), mintPortalToken(), newConsumer(), newEventName(), pushEvent(), registerWebhook(), uniq(), seedWebhook()

### Community 124 - "Decision"
Cohesion: 0.09
Nodes (22): 1. Reusable library: `pkg/access`, 2. Sparrow adapter: `internal/accessauth`, 3. Replace the single shared API key with revocable access tokens and one-time invites, 3. Two scopes only — no roles, 4. Stored invites instead of signed links, 5. No cascade revocation, 6. Tokens independent of master key, 7. 503, not 401, when the database is down (+14 more)

### Community 125 - "auth.svelte.ts"
Cohesion: 0.12
Nodes (17): auth, KEY_STORAGE, message, OWNED_TOKEN_STORAGE, ownedTokenId, redeeming, required, stored (+9 more)

### Community 126 - "Sparrow Webhook Delivery Platform"
Cohesion: 0.33
Nodes (6): Sparrow Agent/Repo Conventions, River Queue (Postgres-backed workers), Svelte 5 Tutorial (PDF), Deploy Docs Workflow, Event-driven Fan-out Pipeline, Sparrow Webhook Delivery Platform

### Community 127 - "scripts"
Cohesion: 0.20
Nodes (10): scripts, build, check, check:watch, dev, gen:api-types, prepare, preview (+2 more)

### Community 128 - "models_test.go"
Cohesion: 0.39
Nodes (8): T, TestEventRegistration_NilJSONFieldsAreNullSafe(), TestJSONMap_RoundTrip(), TestJSONMap_Scan(), TestJSONMap_Value(), TestJSONStringMap_RoundTrip(), TestJSONStringMap_Scan(), TestJSONStringMap_Value()

### Community 129 - "steps_auth.py"
Cohesion: 0.42
Nodes (8): _get(), get_authed_correct_key(), get_authed_no_key(), get_authed_with_key(), step, Step implementations for API key enforcement (12_auth_enforcement.spec)., start_authed_server(), stop_authed_server()

### Community 130 - "CI Build Job"
Cohesion: 0.50
Nodes (5): CI Build Job, CI Workflow, CI Integration Test Job, CI Lint Job, CI Test Job (Postgres service)

### Community 131 - "EventRegistration"
Cohesion: 0.28
Nodes (4): Context, Repository, UUID, EventRegistration

### Community 133 - "ParseNetworks"
Cohesion: 0.14
Nodes (14): Config, Duration, IPNet, Load(), validatePort(), ParseNetworks(), T, TestDialControlBlocksMetadataWithAllowPrivate() (+6 more)

### Community 134 - "web/package.json"
Cohesion: 0.20
Nodes (9): openapi-fetch, dependencies, openapi-fetch, svelte-jsoneditor, svelte-jsoneditor, name, private, type (+1 more)

### Community 136 - "PrepareDeliveryRequest"
Cohesion: 0.13
Nodes (25): DeliveryRequest, WebhookEnvelope, BuildEnvelopePayload(), BuildRequest(), generateEd25519Signature(), generateHMACSignature(), Context, Duration (+17 more)

### Community 137 - "test-ui.sh"
Cohesion: 0.43
Nodes (5): test-ui.sh script, SPARROW_BASE_URL, start_server(), wait_for_static(), wait_healthy()

### Community 138 - "manifest.json"
Cohesion: 0.50
Nodes (3): Language, Plugins, html-report

### Community 140 - "opencode.json"
Cohesion: 0.50
Nodes (3): instructions, $schema, AGENTS.md

### Community 141 - "Deliberately NOT covered by e2e"
Cohesion: 0.40
Nodes (4): API surface deliberately not e2e-tested (removal / design candidates), Deliberately NOT covered by e2e, Known product-contract questions (asserted as-is, flagged), Out of e2e scope (belongs in unit/integration tests)

### Community 144 - "Sparrow Web Dashboard"
Cohesion: 0.22
Nodes (8): Build, Configuration, Embedding in the Go Binary, Local Development, Prerequisites, Sparrow Web Dashboard, Standalone Deployment, Tech Stack

### Community 146 - "steps_access.py"
Cohesion: 0.27
Nodes (19): _as_token(), cancel_invite(), create_consumer_token(), create_invite(), _create_token(), get_with_token(), get_with_token_rejected(), _invites() (+11 more)

### Community 147 - "otel.go"
Cohesion: 0.13
Nodes (24): Int64Counter, Int64UpDownCounter, DefaultConfig(), GetMeter(), GetTracer(), Context, Tracer, newLoggerProvider() (+16 more)

### Community 148 - "httpauth.go"
Cohesion: 0.17
Nodes (17): Authenticator, ctxKey, ErrorBody, RedeemRequest, RedeemResponse, Verifier, authMessage(), Credential() (+9 more)

### Community 149 - "rest/access.go"
Cohesion: 0.05
Nodes (58): SecretSealer, cryptoSealer, main(), GetMigrationsFS(), FS, Duration, Service, Store (+50 more)

### Community 150 - "corsChain"
Cohesion: 0.28
Nodes (14): CORS(), NormalizeOrigins(), corsChain(), get(), ResponseRecorder, T, TB, preflight() (+6 more)

### Community 151 - "access.md"
Cohesion: 0.13
Nodes (14): API endpoints, CI and machine tokens, Consumer (portal) access, First run, From the API, From the CLI, From the web UI, Invite a teammate (+6 more)

### Community 153 - "parseDateFilter"
Cohesion: 0.32
Nodes (6): Time, parseDateFilter(), parseLabelFilter(), T, TestParseDateFilter(), TestParseLabelFilter()

### Community 154 - "NetworkPolicy"
Cohesion: 0.21
Nodes (10): NetworkPolicy, embeddedIPv4(), IPNet, Request, isMetadataIP(), mustCIDRs(), ValidateIP(), IP (+2 more)

### Community 155 - "NewWebhookTemplateContext"
Cohesion: 0.67
Nodes (6): NewWebhookTemplateContext(), T, loadSendgridTemplate(), TestSendgridRecipe_CustomEvent(), TestSendgridRecipe_DeliveryFailed(), TestSendgridRecipe_HealthChanged()

### Community 156 - "FromContext"
Cohesion: 0.26
Nodes (14): downStore, FromContext(), Context, ResponseRecorder, Service, Store, T, newSvc() (+6 more)

### Community 159 - "HandlerFunc"
Cohesion: 0.32
Nodes (10): HandlerFunc, MaxBodyBytes(), buildCSP(), SecurityHeaders(), T, TestBuildCSP(), TestSecurityHeadersNoHashes(), TestSecurityHeadersPassthrough() (+2 more)

### Community 164 - "Dual Webhook Signing (HMAC-SHA256 + Ed25519)"
Cohesion: 0.67
Nodes (3): Dual Webhook Signing (HMAC-SHA256 + Ed25519), Timestamp Replay Protection, Standard Webhooks Format

### Community 173 - "newAlertConfigService"
Cohesion: 0.35
Nodes (13): Context, Status, T, mockRepo, newAlertConfigService(), requireStatus(), TestCreateAlertConfig_ConsumerWide(), TestCreateAlertConfig_RejectsUnsupportedEventType() (+5 more)

### Community 174 - "Context"
Cohesion: 0.38
Nodes (4): Context, Repository, Time, UUID

### Community 175 - "fakeServer"
Cohesion: 0.36
Nodes (16): fakeServer(), Server, T, runCLI(), TestAccessCommandsSurfaceServerErrors(), TestInviteBuildsLinkOnUIURL(), TestInviteDefaultsToServerURLAndJSON(), TestInvitesListAndCancel() (+8 more)

### Community 177 - "Context"
Cohesion: 0.24
Nodes (7): Context, JobArgs, JobInsertResult, UUID, fakeAlertConfigRepoErr, fakeSystemEventRepo, noopJobInserter

### Community 178 - "production.md"
Cohesion: 0.15
Nodes (12): Access and authentication, Deployment, Example Ingress (internal), Kubernetes manifests, Network exposure, NetworkPolicy, Production checklist, Required configuration (+4 more)

### Community 182 - "validConfig"
Cohesion: 0.48
Nodes (6): Config, T, TestUIInjectKeyIsRetired(), TestValidate(), TestWarnings(), validConfig()

### Community 183 - "access/+page.svelte"
Cohesion: 0.15
Nodes (10): consumer, copied, createError, creating, inviteTTL, name, string, tokenTTL (+2 more)

### Community 184 - "pushSystemEvent"
Cohesion: 0.33
Nodes (9): Context, JobInserter, Logger, WebhookWorker, UUID, pushSystemEvent(), SystemEventRegistrations(), systemEventSamplePayload() (+1 more)

### Community 185 - "separate-ui.md"
Cohesion: 0.20
Nodes (9): 1. Build the UI, 2. Point the UI at the server: `config.js`, 3. Serve it as a single-page app, 4. Configure the server, Authentication, Consumer portal, Local development, Troubleshooting (+1 more)

### Community 186 - "newTailCmd"
Cohesion: 0.39
Nodes (8): deliveryItem, apiClient, Command, Context, Writer, newTailCmd(), printDelivery(), runTail()

### Community 187 - "ValidateHeaders"
Cohesion: 0.31
Nodes (7): isReservedHeader(), T, TestBuildRequestHeaderPrecedence(), TestValidateHeaders(), ValidateHeaders(), validHeaderName(), validHeaderValue()

### Community 188 - "SparrowVerify"
Cohesion: 0.38
Nodes (5): ByteArray, Exception, ParsedHeaders, SignatureVerificationException, SparrowVerify

### Community 189 - "WebhookClient"
Cohesion: 0.17
Nodes (10): redactSpanURL, WebhookClient, Config, Context, Duration, Request, Response, WebhookTemplateContext (+2 more)

### Community 190 - "PortalGateway"
Cohesion: 0.22
Nodes (11): ResponseWriter, writeJSONError(), cleanPortalPath(), Context, ResponseWriter, PortalAuthorized(), PortalGateway(), portalTarget() (+3 more)

### Community 197 - "Mount"
Cohesion: 0.13
Nodes (25): main(), accessRouter(), Context, ResponseRecorder, Store, T, post(), TestAccessRejectsUnsafeConsumerNames() (+17 more)

### Community 198 - "GetBuffer"
Cohesion: 0.28
Nodes (11): GetBuffer(), GetHeaderMap(), Buffer, PutBuffer(), PutHeaderMap(), BenchmarkBufferPool(), BenchmarkHeaderMapPool(), B (+3 more)

### Community 199 - ".VerifyEd25519"
Cohesion: 0.38
Nodes (8): Duration, Header, Time, parseHeaders(), signedMessage(), VerifyEd25519(), VerifyHMAC(), Verifier

### Community 201 - "SparrowVerify"
Cohesion: 0.35
Nodes (10): SparrowVerify, decode_public_key(), decode_secret(), decode_signatures(), fetch_header(), parse_headers(), safe_hex_decode(), validate_timestamp() (+2 more)

### Community 202 - "SparrowVerify"
Cohesion: 0.35
Nodes (3): SparrowVerify, SparrowVerify::SignatureVerificationError, StandardError

### Community 203 - "Config"
Cohesion: 0.22
Nodes (6): Config, DefaultConfig(), Duration, IPNet, T, TestDefaultConfig()

### Community 204 - "SparrowVerify"
Cohesion: 0.22
Nodes (3): SignatureVerificationException, SparrowVerify, RuntimeException

### Community 205 - "buildVectors"
Cohesion: 0.60
Nodes (5): vector, vectorFile, buildVectors(), T, TestSignatureVectors()

### Community 206 - "verify-signatures.mdx"
Cohesion: 0.50
Nodes (3): Get the helper and verify, Testing your receiver, What gets signed

### Community 207 - "resolveConfig"
Cohesion: 0.36
Nodes (9): configPath(), resolveConfig(), saveConfig(), T, TestResolveConfigDefaults(), TestResolveConfigPrecedence(), TestSaveConfigPermissions(), writeConfigFile() (+1 more)

### Community 208 - "TestTemplateTestRendersFixtureRecipe"
Cohesion: 0.67
Nodes (3): T, TestTemplateTestRendersFixtureRecipe(), TestTemplateTestReportsParseError()

### Community 209 - "TokenPurgeWorker"
Cohesion: 0.18
Nodes (11): Duration, Context, Duration, InsertOpts, Job, Logger, WorkerDefaults, NewTokenPurgeWorker() (+3 more)

### Community 210 - "Invite"
Cohesion: 0.29
Nodes (6): AuthError, Invite, Reason, Status, Duration, Time

### Community 212 - "listenHandler"
Cohesion: 0.20
Nodes (16): apiClient, Command, Context, Request, ResponseWriter, Writer, Request, T (+8 more)

### Community 213 - "runUse"
Cohesion: 0.29
Nodes (10): apiClient, Command, Context, recipe, Writer, loadRecipe(), newUseCmd(), resolveRecipe() (+2 more)

### Community 214 - "runInit"
Cohesion: 0.31
Nodes (9): File, Command, config, Context, Writer, isTerminal(), newInitCmd(), probeServer() (+1 more)

### Community 215 - "Run"
Cohesion: 0.29
Nodes (8): T, TestConformance(), Store, T, ids(), mintFn(), must(), Run()

### Community 216 - "health.go"
Cohesion: 0.50
Nodes (3): healthSummaryOutput, listWebhooksGlobalInput, webhookHealthOutput

### Community 218 - "parseRetryAfter"
Cohesion: 0.44
Nodes (8): T, TestDefaultAndMaxRetryAfterConstants(), TestIsSuccessStatusCode(), TestParseRetryAfter(), TestParseRetryAfter_HTTPDate(), TestParseRetryAfter_HTTPDate_FarFuture(), TestParseRetryAfter_HTTPDate_Past(), parseRetryAfter()

### Community 222 - "use_test.go"
Cohesion: 0.60
Nodes (4): T, TestLoadRecipeFixture(), TestResolveRecipeBuiltin(), TestSubstituteParams()

### Community 223 - "TestParseJSONArg"
Cohesion: 0.67
Nodes (3): T, TestKVFlag(), TestParseJSONArg()

### Community 229 - "generator.ts"
Cohesion: 0.08
Nodes (25): entryToSimpleMarkdown(), htmlToMarkdownPipeline, minify, minifyDefaults, selectors, collator, generateLlmsTxt(), starlightLlmsTxt() (+17 more)

### Community 242 - "github.com/sarathsp06/sparrow"
Cohesion: 0.53
Nodes (6): github.com/sarathsp06/sparrow, github.com/sarathsp06/sparrow/pkg/access, github.com/sarathsp06/sparrow/pkg/signature, github.com/sarathsp06/sparrow/pkg/template, github.com/sarathsp06/sparrow/satellites/recipes, github.com/sarathsp06/sparrow/satellites/sparrow

## Knowledge Gaps
- **466 isolated node(s):** `DEFAULT_TOLERANCE_SECONDS`, `SignatureVerificationError`, `Headers`, `name`, `type` (+461 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **42 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Errorf()` connect `Errorf` to `setupEnv`, `Token`, `event_filtering_test.go`, `ParseNetworks`, `PrepareDeliveryRequest`, `WebhookHTTPConfig`, `NewService`, `Context`, `.loadAndValidateBatch`, `otel.go`, `NewWebhookHandler`, `rest/access.go`, `WebhookWorker`, `parseDateFilter`, `NetworkPolicy`, `BatchJob`, `apiClient`, `EventProcessingWorker`, `ClassifyError`, `envelope`, `WebhookDelivery`, `jobInserter`, `Context`, `WebhookService`, `Context`, `TemplateEngine`, `runPush`, `newEmailSink`, `Service`, `newTailCmd`, `ValidateHeaders`, `Context`, `newOTLPSink`, `newPalette`, `Client`, `EventRecord`, `RepositoryInterface`, `ServiceError`, `.VerifyEd25519`, `run`, `newS3Sink`, `.CreateWebhook`, `resolveConfig`, `listenHandler`, `runUse`, `runInit`, `sparrow/access.go`, `GetFunctionMap`, `runFunctions`, `NewManager`, `listFlag`, `WebhookService`, `Recipe`, `parseUUID`, `runTemplateTest`?**
  _High betweenness centrality (0.239) - this node is a cross-community bridge._
- **Why does `New()` connect `New` to `setupEnv`, `event_filtering_test.go`, `PrepareDeliveryRequest`, `WebhookServiceInterfaceWithTracing`, `Context`, `otel.go`, `rest/access.go`, `corsChain`, `BatchJob`, `Errorf`, `EventProcessingWorker`, `newAlertConfigService`, `newAuth`, `Context`, `NewWebhookClient`, `pushSystemEvent`, `ValidateHeaders`, `EventRecord`, `RepositoryInterface`, `Mount`, `Handler`, `.CreateWebhook`, `mockRepo`, `NewManager`, `testContext`, `newPortalGateway`?**
  _High betweenness centrality (0.080) - this node is a cross-community bridge._
- **Why does `Mount()` connect `Mount` to `setupEnv`, `event.go`, `subscription.go`, `delivery.go`, `webhook_service.go`, `registerWebhookRoutes`, `New`, `mapError`, `rest/access.go`, `All`?**
  _High betweenness centrality (0.032) - this node is a cross-community bridge._
- **Are the 183 inferred relationships involving `Errorf()` (e.g. with `.Authenticate()` and `.CreateTokenIdempotent()`) actually correct?**
  _`Errorf()` has 183 INFERRED edges - model-reasoned connections that need verification._
- **Are the 94 inferred relationships involving `New()` (e.g. with `TestMasterKeyIsRootInDefaultTenant()` and `TestPgstoreConformance()`) actually correct?**
  _`New()` has 94 INFERRED edges - model-reasoned connections that need verification._
- **What connects `DEFAULT_TOLERANCE_SECONDS`, `SignatureVerificationError`, `Headers` to the rest of the system?**
  _466 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `setupEnv` be split into smaller, more focused modules?**
  _Cohesion score 0.057692307692307696 - nodes in this community are weakly interconnected._