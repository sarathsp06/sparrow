# Graph Report - sparrow  (2026-09-17)

## Corpus Check
- 331 files · ~299,507 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2480 nodes · 5021 edges · 185 communities (151 shown, 34 thin omitted)
- Extraction: 85% EXTRACTED · 15% INFERRED · 0% AMBIGUOUS · INFERRED: 731 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `7d28c8a8`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- setupEnv
- event.go
- Mount
- testContext
- subscription.go
- RepositoryInterfaceWithTracing
- delivery.go
- steps.py
- rest/webhook.go
- sparrow_verify.py
- webhooks/models.go
- NewService
- WebhookServiceInterfaceWithTracing
- Sparrow Detailed Flow Reference
- Context
- .Work
- sparrow-verify.ts
- system_events_test.go
- api-types.d.ts
- TemplateCache
- NewWebhookHandler
- api.astro
- Prune Journal
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
- EventRegistration
- ClassifyError
- envelope
- VerifyHMAC
- validConfig
- portal/+page.svelte
- Sparrow Implementation Plan
- webhook_service.go
- runInit
- registerWebhookRoutes
- scripts
- NewWebhookClient
- TemplateEngine
- NewTemplateEngine
- ValidateIP
- runPush
- newEmailSink
- [webhookId]/+page.svelte
- otel.go
- validConfig
- Commands
- Context
- postSink
- parseRetryAfter
- newOTLPSink
- devDependencies
- runEvents
- run
- Client
- apiConsole.svelte.ts
- SparrowAPI
- RepositoryInterface
- TestRecipes
- resolveConfig
- ServiceError
- config
- newS3Sink
- sinks.mdx
- .CreateWebhook
- Real-world examples
- pushEnv
- WebhookDelivery
- Included recipes
- sources.mdx
- ADDED Requirements
- Sparrow Deployment Template
- parseUUID
- runUse
- reference/security.mdx
- utils.ts
- ADDED Requirements
- compilerOptions
- WebhookTargetManager
- deployment/security.mdx
- runListen
- GetFunctionMap
- Sparrow Recipes
- newTailCmd
- runFunctions
- NewManager
- BenchmarkTransformPayload
- _Target
- Checker
- WebhookService
- Recipe
- jobInserter
- hooks.py
- SparrowEnvironment
- index.mdx
- 2. Split the `sparrow` CLI into its own module to isolate its `go install` graph
- EventProcessingWorker
- 1. Payload transform engine: Go `text/template`, not embedded JavaScript
- Context
- runTemplateTest
- RetentionWorker
- release-submodules.sh
- alert_config.go
- AlertConfig
- proposal.md
- Handler
- Feature: Webhook health/delivery-failure email alerts
- tasks.md
- Context
- webhook-health-alerts.mdx
- design.md
- consumer.svelte.ts
- Sparrow -- Condensed Reference
- WebhookWorker
- Sparrow Webhook Delivery Platform
- portal-embedding.mdx
- models_test.go
- steps_auth.py
- CI Build Job
- EventReportWithStats
- mapError
- NewWebhookTemplateContext
- services.ts
- palette
- scripts
- .Work
- manifest.json
- dependencies
- instructions
- Deliberately NOT covered by e2e
- run
- web/package.json
- Sparrow Web Dashboard
- newPagination
- svelte-check
- @sveltejs/adapter-static
- @sveltejs/kit
- tailwindcss
- @tailwindcss/typography
- @types/node
- vite
- Config
- vite-plugin-devtools-json
- svelte
- src/components/Footer.astro
- docs/tsconfig.json
- TestJobInserter_InsertOpts_Merge
- TestWebhookWorkerNextRetry
- Dual Webhook Signing (HMAC-SHA256 + Ed25519)
- +layout.ts
- WithConn Transaction Pattern
- ../../components/ThemeDiagram.astro
- content.config.ts
- proto2astro
- index.astro
- WebhookService
- svelte.config.js
- Sparrow Ingress Template
- Sparrow Chart NOTES
- Sparrow PodDisruptionBudget Template
- Release Workflow
- github.com/sarathsp06/sparrow
- sparrow-e2e
- Envelope Encryption at Rest (AES-256-GCM)

## God Nodes (most connected - your core abstractions)
1. `Errorf()` - 152 edges
2. `RepositoryInterfaceWithTracing` - 74 edges
3. `WebhookServiceInterfaceWithTracing` - 43 edges
4. `setupEnv()` - 40 edges
5. `EventSubscription` - 40 edges
6. `testContext()` - 35 edges
7. `WebhookDelivery` - 31 edges
8. `NewService()` - 30 edges
9. `EventRegistration` - 28 edges
10. `apiClient` - 27 edges

## Surprising Connections (you probably didn't know these)
- `Svelte 5 Tutorial (PDF)` --conceptually_related_to--> `Sparrow Webhook Delivery Platform`  [INFERRED]
  book/svelte5-tutorial.pdf → README.md
- `Sparrow NetworkPolicy Template` --semantically_similar_to--> `SSRF Protection`  [INFERRED] [semantically similar]
  charts/sparrow/templates/networkpolicy.yaml → README.md
- `Development Docker Compose` --semantically_similar_to--> `Standalone Docker Compose`  [INFERRED] [semantically similar]
  docker-compose.dev.yml → deploy/docker-compose.yml
- `main()` --calls--> `RunAllMigrations()`  [INFERRED]
  cmd/migrate/main.go → internal/migration/migrate.go
- `main()` --calls--> `Mount()`  [INFERRED]
  cmd/openapi-export/main.go → internal/rest/app.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **CI Pipeline Stage Dependency Chain** — github_workflows_ci_lint_job, github_workflows_ci_test_job, github_workflows_ci_build_job, github_workflows_ci_integration_job [EXTRACTED 1.00]
- **Tag-triggered Release Automation** — github_workflows_release_goreleaser_job, github_workflows_release_docker_job, goreleaser_goreleaser_config, charts_sparrow_chart [EXTRACTED 1.00]
- **Security-hardened Kubernetes Deployment** — charts_sparrow_templates_deployment_deployment, charts_sparrow_templates_networkpolicy_networkpolicy, charts_sparrow_templates_configmap_configmap, charts_sparrow_templates_postgresql_service_service [INFERRED 0.85]
- **River-backed async delivery pipeline** — concept_river_queue, concept_event_processing_worker, concept_webhook_worker, concept_postgresql [EXTRACTED 1.00]
- **Sparrow RPC service surface** — concept_webhook_service, concept_event_service, concept_subscription_service, concept_delivery_service, concept_health_service [EXTRACTED 1.00]
- **Sparrow security feature set** — concept_envelope_encryption, concept_standard_webhooks_signing, concept_ssrf_protection [EXTRACTED 0.95]

## Communities (185 total, 34 thin omitted)

### Community 0 - "setupEnv"
Cohesion: 0.05
Nodes (99): Container, HandlerFunc, Int32, capturedWebhook, deliveryItem, restClient, sinkSMTPMessage, testEnv (+91 more)

### Community 1 - "event.go"
Cohesion: 0.10
Nodes (31): API, Context, listEventOccurrencesImpl(), registerEventRoutes(), toBatchJobOutput(), toEventTypeItem(), toEventTypeOutput(), batchJobOutput (+23 more)

### Community 2 - "Mount"
Cohesion: 0.22
Nodes (6): main(), API, Mount(), T, TestOpenAPISpecMatchesCommitted(), Router

### Community 3 - "testContext"
Cohesion: 0.08
Nodes (65): Context, T, UUID, TestCreateSubscription_CatchAllWithLabelFilters(), TestCreateSubscription_WithEmptyLabelFilters(), TestCreateSubscription_WithInvalidLabelFilters(), TestCreateSubscription_WithLabelFilters(), TestGetSubscriptionsByEvent_CatchAllReturned() (+57 more)

### Community 4 - "subscription.go"
Cohesion: 0.17
Nodes (20): API, Context, listSubscriptionsImpl(), registerSubscriptionRoutes(), toSubscriptionItem(), toSubscriptionOutput(), createSubscriptionBody, createSubscriptionInput (+12 more)

### Community 5 - "RepositoryInterfaceWithTracing"
Cohesion: 0.10
Nodes (12): Context, Span, Time, UUID, WebhookRegistration, NewRepositoryInterfaceWithTracing(), BatchJobData, BatchJobStatus (+4 more)

### Community 6 - "delivery.go"
Cohesion: 0.17
Nodes (19): API, Context, listDeliveriesImpl(), registerDeliveryRoutes(), toDeliveryItem(), toDeliveryOutput(), attemptItem, attemptsOutput (+11 more)

### Community 7 - "steps.py"
Cohesion: 0.08
Nodes (59): delivery_has_signature_headers(), SignatureVerifier -- Verifies HMAC-SHA256 (v1,) and Ed25519 (v1a,) signatures…, Return the exact body bytes (as text) that Sparrow signed. Standard Webhooks…, Verify HMAC-SHA256 signature (v1, prefix)., Verify Ed25519 signature (v1a, prefix)., Assert delivery has Standard Webhooks signature headers., _signed_body(), verify_ed25519_signature() (+51 more)

### Community 8 - "rest/webhook.go"
Cohesion: 0.20
Nodes (13): consumerOnlyInput, consumerStatsOutput, emptyOutput, listWebhooksInput, patchWebhookBody, patchWebhookInput, registerWebhookBody, registerWebhookInput (+5 more)

### Community 9 - "sparrow_verify.py"
Cohesion: 0.32
Nodes (11): _check_timestamp(), _decode_signatures(), Verify Sparrow webhook delivery signatures (Standard Webhooks format). Every…, Raised when a delivery signature cannot be verified., Verify the ``v1,`` (HMAC-SHA256) signature. Raises on failure. ``payload`` must…, Verify the ``v1a,`` (Ed25519) signature. Raises on failure. ``payload`` must be…, _required_headers(), SignatureVerificationError (+3 more)

### Community 10 - "webhooks/models.go"
Cohesion: 0.12
Nodes (13): DefaultWebhookHTTPConfig(), DerefBoolOr(), DerefIntOr(), Duration, Time, Value, HTTPConfigUpdate, IntArray (+5 more)

### Community 11 - "NewService"
Cohesion: 0.07
Nodes (54): AEAD, DeliveryRequest, WebhookEnvelope, Service, BuildEnvelopePayload(), BuildRequest(), generateEd25519Signature(), generateHMACSignature() (+46 more)

### Community 12 - "WebhookServiceInterfaceWithTracing"
Cohesion: 0.06
Nodes (16): Context, Repository, Time, UUID, Time, Context, Span, Time (+8 more)

### Community 13 - "Sparrow Detailed Flow Reference"
Cohesion: 0.08
Nodes (39): PostgreSQL StatefulSet, Sparrow Kubernetes Service, Sparrow Helm Chart Values, Sparrow Client Libraries README, Connect-RPC, DeliveryService, Envelope Encryption (AES-256-GCM), 10-Category Error Classification (+31 more)

### Community 14 - "Context"
Cohesion: 0.13
Nodes (19): T, TestE2E_EventDeliveryStats_NoDeliveries(), Context, Repository, Tx, UUID, WebhookRegistration, NewRepository() (+11 more)

### Community 15 - ".Work"
Cohesion: 0.24
Nodes (26): also_use_consumer(), assert_delivery_page_meta(), _base(), delete_event_type(), delete_saved_subscription(), delete_webhook(), get_consumer_path_status(), get_event_type_status() (+18 more)

### Community 16 - "sparrow-verify.ts"
Cohesion: 0.39
Nodes (7): decodeSignatures(), DEFAULT_TOLERANCE_SECONDS, Headers, parseHeaders(), SignatureVerificationError, verifyEd25519(), verifyHmac()

### Community 17 - "system_events_test.go"
Cohesion: 0.11
Nodes (29): Context, JobInserter, Logger, WebhookWorker, UUID, pushSystemEvent(), SystemEventRegistrations(), systemEventSamplePayload() (+21 more)

### Community 18 - "api-types.d.ts"
Cohesion: 0.29
Nodes (6): RFC-3339, components, $defs, operations, paths, webhooks

### Community 19 - "TemplateCache"
Cohesion: 0.24
Nodes (10): Cache, Template, hashTemplate(), NewTemplateCache(), T, TestHashTemplate(), TestTemplateCacheBasicOperations(), TestTemplateCacheConcurrency() (+2 more)

### Community 20 - "NewWebhookHandler"
Cohesion: 0.07
Nodes (50): main(), RawMessage, SparrowConfig, LoadConfig(), cronTick(), Context, Logger, Time (+42 more)

### Community 22 - "Prune Journal"
Cohesion: 0.50
Nodes (3): 2026-08-10 - Dead-code baseline established, 2026-09-09 - Ed25519 keygen duplication removed, Prune Journal

### Community 23 - "store/models.go"
Cohesion: 0.11
Nodes (15): Int64Array, Time, UUID, Value, DeliveryFilter, EventRecord, JSONMap, JSONStringMap (+7 more)

### Community 24 - "newRootCmd"
Cohesion: 0.15
Nodes (22): addOutputFlag(), clientFromCmd(), Command, config, Writer, newRootCmd(), newVersionCmd(), outputFmt() (+14 more)

### Community 27 - "BatchJob"
Cohesion: 0.19
Nodes (8): Context, Repository, UUID, RawMessage, Context, WebhookService, BatchJob, BatchJobType

### Community 32 - "apiClient"
Cohesion: 0.15
Nodes (15): config, Context, newAPIClient(), apiClient, apiError, consumerStats, deliveryItem, eventTypeItem (+7 more)

### Community 33 - "Errorf"
Cohesion: 0.22
Nodes (7): generateSamplePayload(), Context, Time, WebhookService, ValidateJSONSchema(), Errorf(), SchemaValidationError

### Community 34 - "Sparrow Architecture"
Cohesion: 0.29
Nodes (7): Sparrow Architecture, Dual-Protocol API, Event Processing Pipeline, Health State Machine, Leaky Bucket Rate Limiting, Standard Webhooks Signing, Default Webhook Body Envelope

### Community 35 - "JobInserterWithTracing"
Cohesion: 0.31
Nodes (7): Context, JobArgs, JobInserter, JobInsertResult, Span, NewJobInserterWithTracing(), JobInserterWithTracing

### Community 36 - "EventRegistration"
Cohesion: 0.28
Nodes (4): Context, Repository, UUID, EventRegistration

### Community 37 - "ClassifyError"
Cohesion: 0.16
Nodes (23): ErrorCategory, timeoutError, classifyByMessage(), ClassifyError(), ClassifyHTTPStatus(), classifySyscallError(), isDNSError(), IsRetryableCategory() (+15 more)

### Community 38 - "envelope"
Cohesion: 0.18
Nodes (12): Context, Template, config, Context, Logger, RawMessage, sinkHandler(), templateContext() (+4 more)

### Community 39 - "VerifyHMAC"
Cohesion: 0.24
Nodes (16): T, TestBuildRequestSignaturesVerifiable(), Header, Time, parseHeaders(), signedMessage(), T, signHMAC() (+8 more)

### Community 40 - "validConfig"
Cohesion: 0.18
Nodes (5): Context, Repository, UUID, WebhookDelivery, WebhookDeliveryStatus

### Community 41 - "portal/+page.svelte"
Cohesion: 0.19
Nodes (19): unwrap(), formatAPIError(), deliveriesLoading, error, executeDelete(), expired, fetchDeliveries(), fetchSubscriptions() (+11 more)

### Community 42 - "Sparrow Implementation Plan"
Cohesion: 0.05
Nodes (43): API Changes, API Changes, Commands, Completed Parts (v0.8.0 -- v1.2.1), Configuration, Configuration, Current State (as of v1.2.1), Decisions Log (+35 more)

### Community 43 - "webhook_service.go"
Cohesion: 0.21
Nodes (15): WithAllowPrivateNetworks(), deliveryRouteService, eventRouteService, healthRouteService, webhookRouteService, AlertConfigManager, BatchManager, DeliveryManager (+7 more)

### Community 44 - "runInit"
Cohesion: 0.31
Nodes (9): File, Command, config, Context, Writer, isTerminal(), newInitCmd(), probeServer() (+1 more)

### Community 45 - "registerWebhookRoutes"
Cohesion: 0.28
Nodes (14): getWebhookEventsMap(), Context, WebhookRegistration, maskEncryptedSecret(), maskSecret(), maskSecretHeaders(), toWebhookOut(), toWebhookOutFromDomain() (+6 more)

### Community 47 - "scripts"
Cohesion: 0.07
Nodes (28): @astrojs/starlight, dependencies, astro, @astrojs/starlight, marked, @scalar/astro, sharp, zod (+20 more)

### Community 48 - "NewWebhookClient"
Cohesion: 0.08
Nodes (37): Config, WebhookClient, Config, Context, Duration, Response, WebhookTemplateContext, NewWebhookClient() (+29 more)

### Community 49 - "TemplateEngine"
Cohesion: 0.16
Nodes (10): getBuffer(), Buffer, putBuffer(), FuncMap, Template, WebhookTemplateContext, NewTemplateEngineWithCacheSize(), limitedWriter (+2 more)

### Community 50 - "NewTemplateEngine"
Cohesion: 0.27
Nodes (17): NewTemplateEngine(), BenchmarkExecuteComplex(), BenchmarkExecuteSimple(), B, T, TestExecuteComplexTemplate(), TestExecuteEmptyTemplate(), TestExecuteInvalidTemplate() (+9 more)

### Community 51 - "ValidateIP"
Cohesion: 0.23
Nodes (10): Request, permissiveCheckRedirect(), ssrfDialControl(), ssrfSafeCheckRedirect(), T, TestValidateIP(), ValidateIP(), validateRedirectURL() (+2 more)

### Community 52 - "runPush"
Cohesion: 0.12
Nodes (11): parseJSONArg(), T, TestKVFlag(), TestParseJSONArg(), Command, Context, Writer, newPushCmd() (+3 more)

### Community 53 - "newEmailSink"
Cohesion: 0.37
Nodes (12): Conn, newEmailSink(), T, serveSMTP(), startSMTPServer(), TestEmailRender_BadTemplate(), TestEmailRender_CustomTemplatesAndHeaderInjection(), TestEmailRender_Defaults() (+4 more)

### Community 55 - "otel.go"
Cohesion: 0.18
Nodes (19): Int64Counter, Int64UpDownCounter, DefaultConfig(), GetMeter(), GetTracer(), Context, Tracer, newLoggerProvider() (+11 more)

### Community 56 - "validConfig"
Cohesion: 0.26
Nodes (10): Config, T, TestValidate(), TestWarnings(), validConfig(), T, TestApplyConfig_RateLimitRPS(), TestDefaultWebhookHTTPConfig_RateLimitRPS() (+2 more)

### Community 57 - "Commands"
Cohesion: 0.12
Nodes (16): 90-second quickstart, Commands, Configuration, How it's connected, Install, Real-World Use Cases, Scenario 1: Zero-Config Local Webhook Receiver & Debugging, Scenario 2: CI/CD Pipeline Automation & Deployment Notifications (+8 more)

### Community 58 - "Context"
Cohesion: 0.38
Nodes (3): Context, Repository, UUID

### Community 59 - "postSink"
Cohesion: 0.40
Nodes (12): ResponseRecorder, Context, Header, T, postSink(), signHeaders(), TestSinkHandler_DownstreamFailureIs502(), TestSinkHandler_MissingHeaders() (+4 more)

### Community 60 - "parseRetryAfter"
Cohesion: 0.22
Nodes (12): T, TestDefaultAndMaxRetryAfterConstants(), TestIsSuccessStatusCode(), TestParseRetryAfter(), TestParseRetryAfter_HTTPDate(), TestParseRetryAfter_HTTPDate_FarFuture(), TestParseRetryAfter_HTTPDate_Past(), isSuccessStatusCode() (+4 more)

### Community 61 - "newOTLPSink"
Cohesion: 0.27
Nodes (10): Context, logsURL(), newOTLPSink(), otlpPayload(), T, TestLogsURL(), TestOTLPSinkDeliver(), TestOTLPSinkDeliverDownstreamFailure() (+2 more)

### Community 62 - "devDependencies"
Cohesion: 0.15
Nodes (13): openapi-typescript, @sveltejs/vite-plugin-svelte, @tailwindcss/forms, @tailwindcss/vite, @types/js-yaml, typescript, devDependencies, openapi-typescript (+5 more)

### Community 63 - "runEvents"
Cohesion: 0.16
Nodes (17): eventTypeItem, Writer, newPalette(), Context, Writer, indentJSON(), printEventType(), runEvents() (+9 more)

### Community 64 - "run"
Cohesion: 0.50
Nodes (4): Context, Writer, main(), run()

### Community 65 - "Client"
Cohesion: 0.27
Nodes (7): Context, RawMessage, SparrowConfig, NewClient(), T, TestClientAutoCreatesEventType(), Client

### Community 66 - "apiConsole.svelte.ts"
Cohesion: 0.12
Nodes (9): apiConsole, ApiLogEntry, apiLogMiddleware, entries, started, beat, data, pulseStore (+1 more)

### Community 67 - "SparrowAPI"
Cohesion: 0.14
Nodes (7): SparrowAPI -- wraps the generated REST client (sparrow_client, generated from…, Poll until all deliveries reach terminal status., Poll a single delivery until terminal., Convert a generated attrs model (or None) to a plain dict., Client for Sparrow's REST API, built on the generated sparrow_client SDK., SparrowAPI, _to_dict()

### Community 68 - "RepositoryInterface"
Cohesion: 0.18
Nodes (12): JobInserter, Logger, WorkerDefaults, NewBatchJobWorker(), BatchJobWorker, systemEventRepo, BatchRepository, EventRepository (+4 more)

### Community 69 - "TestRecipes"
Cohesion: 0.43
Nodes (6): T, WebhookTemplateContext, sampleContext(), substituteParams(), TestRecipes(), tokenRefs()

### Community 70 - "resolveConfig"
Cohesion: 0.36
Nodes (9): configPath(), resolveConfig(), saveConfig(), T, TestResolveConfigDefaults(), TestResolveConfigPrecedence(), TestSaveConfigPermissions(), writeConfigFile() (+1 more)

### Community 71 - "ServiceError"
Cohesion: 0.18
Nodes (11): ServiceError, Status, Classify(), Error(), T, TestConstructors(), TestServiceError_ClientMessage(), TestServiceError_Error() (+3 more)

### Community 72 - "config"
Cohesion: 0.52
Nodes (5): loadConfig(), config, emailConfig, s3Config, smtpConfig

### Community 73 - "newS3Sink"
Cohesion: 0.31
Nodes (7): Context, newS3Sink(), objectKey(), T, TestObjectKey(), TestS3Sink_PutObject(), s3Sink

### Community 74 - "sinks.mdx"
Cohesion: 0.14
Nodes (13): Configuration, How it's connected, Install and run, Real-World Use Cases, Retry semantics, Scenario 1: Long-Term Regulatory & Financial Audit Archiving (S3 / MinIO / R2), Scenario 2: Operational Stakeholder Notifications via SMTP Email, Scenario 3: Centralized OpenTelemetry Log & Event Pipeline (OTLP) (+5 more)

### Community 75 - ".CreateWebhook"
Cohesion: 0.22
Nodes (7): ValidateWebhookURL(), generateWebhookSecret(), Context, Time, WebhookRegistration, WebhookService, Wrapf()

### Community 76 - "Real-world examples"
Cohesion: 0.13
Nodes (14): 1. Send only the fields a partner needs, 2. Post a message to Slack, 3. Convert dollars to cents for a billing system, 4. Flatten a nested payload for a legacy endpoint, 5. Summarize a list of items, 6. Add a constant or computed field, 7. Provide safe defaults for optional fields, Good to know (+6 more)

### Community 77 - "pushEnv"
Cohesion: 0.33
Nodes (8): T, TestEventsDetailShowsSchema(), TestEventsList(), Server, T, pushEnv(), TestPushAutoCreatesEventType(), TestPushRequestShape()

### Community 78 - "WebhookDelivery"
Cohesion: 0.18
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

### Community 82 - "Sparrow Deployment Template"
Cohesion: 0.18
Nodes (11): Sparrow Helm Chart, Sparrow ConfigMap Template, Sparrow Deployment Template, Sparrow HPA Template, Sparrow NetworkPolicy Template, Sparrow PostgreSQL Service Template, Conventional Commits Convention, Release Docker Image Job (+3 more)

### Community 83 - "parseUUID"
Cohesion: 0.16
Nodes (9): Context, WebhookService, Context, WebhookService, UUID, Context, WebhookService, parseUUID() (+1 more)

### Community 84 - "runUse"
Cohesion: 0.21
Nodes (13): findRecipe(), Command, Context, Writer, loadRecipe(), newUseCmd(), runUse(), substituteParams() (+5 more)

### Community 85 - "reference/security.mdx"
Cohesion: 0.25
Nodes (7): API Authentication, Consumer Isolation, HTTP Hardening, Portal tokens, Production Config Checklist, Secret Masking in Responses, SSRF Protection

### Community 86 - "utils.ts"
Cohesion: 0.18
Nodes (7): ERROR_CATEGORIES, getCategoryBadge(), getCategoryDisplay(), inferType(), JSONSchemaMetaSchema, jsonToJsonSchema(), RFC-9457

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

### Community 91 - "runListen"
Cohesion: 0.38
Nodes (9): Command, Context, Request, ResponseWriter, Writer, mirrorForward(), newListenCmd(), printReceived() (+1 more)

### Community 92 - "GetFunctionMap"
Cohesion: 0.31
Nodes (7): GetFunctionMap(), GetTemplateFunctions(), FuncMap, T, TestTitleFunc(), toNumber(), TemplateFunc

### Community 93 - "Sparrow Recipes"
Cohesion: 0.33
Nodes (5): Contributing a recipe, How params work, Included recipes, Schema (version 1), Sparrow Recipes

### Community 94 - "newTailCmd"
Cohesion: 0.43
Nodes (7): deliveryItem, Command, Context, Writer, newTailCmd(), printDelivery(), runTail()

### Community 95 - "runFunctions"
Cohesion: 0.28
Nodes (7): firstLine(), Context, Writer, runFunctions(), T, TestFirstLineSkipsHeadings(), TestRenderStructured()

### Community 96 - "NewManager"
Cohesion: 0.24
Nodes (8): Config, Context, JobInserter, Logger, Pool, Tx, NewManager(), Manager

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
Cohesion: 0.25
Nodes (5): Context, Time, UUID, WebhookService, paginateSubscriptions()

### Community 101 - "Recipe"
Cohesion: 0.70
Nodes (4): Param, Recipe, Subscription, Webhook

### Community 102 - "jobInserter"
Cohesion: 0.19
Nodes (9): Context, InsertOpts, JobArgs, JobInsertResult, Logger, Tx, NewJobInserter(), BatchJobArgs (+1 more)

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

### Community 107 - "EventProcessingWorker"
Cohesion: 0.17
Nodes (11): Context, Job, JobInserter, Logger, WorkerDefaults, NewEventProcessingWorker(), Time, EventArgs (+3 more)

### Community 108 - "1. Payload transform engine: Go `text/template`, not embedded JavaScript"
Cohesion: 0.33
Nodes (5): 1. Payload transform engine: Go `text/template`, not embedded JavaScript, Consequences, Context, Decision, Trigger to revisit

### Community 109 - "Context"
Cohesion: 0.35
Nodes (4): Context, Repository, Time, UUID

### Community 110 - "runTemplateTest"
Cohesion: 0.50
Nodes (4): Command, Writer, newTemplateCmd(), runTemplateTest()

### Community 111 - "RetentionWorker"
Cohesion: 0.21
Nodes (8): Context, InsertOpts, Job, Logger, WorkerDefaults, NewRetentionWorker(), RetentionArgs, RetentionWorker

### Community 112 - "release-submodules.sh"
Cohesion: 0.80
Nodes (4): release_leaf(), release_module(), release-submodules.sh script, tag_exists()

### Community 113 - "alert_config.go"
Cohesion: 0.27
Nodes (10): API, registerAlertConfigRoutes(), toAlertConfigItem(), alertConfigIDInput, alertConfigItem, alertConfigOutput, createAlertConfigBody, createAlertConfigInput (+2 more)

### Community 114 - "AlertConfig"
Cohesion: 0.23
Nodes (22): assert_all_terminal_status(), assert_bulk_retry_count(), assert_consumer_stats(), assert_consumer_stats_webhooks(), assert_global_stats(), assert_health_failed_count(), assert_health_summary(), assert_job_progress() (+14 more)

### Community 115 - "proposal.md"
Cohesion: 0.29
Nodes (6): Capabilities, Impact, Modified Capabilities, New Capabilities, What Changes, Why

### Community 116 - "Handler"
Cohesion: 0.06
Nodes (37): main(), GetMigrationsFS(), FS, Request, MaxBodyBytes(), Context, PortalAuthorized(), PortalGateway() (+29 more)

### Community 117 - "Feature: Webhook health/delivery-failure email alerts"
Cohesion: 0.33
Nodes (5): Context, Design, Diagrams, Feature: Webhook health/delivery-failure email alerts, Open Questions

### Community 118 - "tasks.md"
Cohesion: 0.33
Nodes (5): 1. Foundation, 2. Event emission, 3. Tenant-facing API, 4. Delivery channel, 5. Verification & docs

### Community 119 - "Context"
Cohesion: 0.38
Nodes (4): Context, Repository, Time, UUID

### Community 120 - "webhook-health-alerts.mdx"
Cohesion: 0.40
Nodes (4): Good to know, How it works, Register an alert config, Set up the email delivery channel

### Community 121 - "design.md"
Cohesion: 0.50
Nodes (3): Alternatives rejected, Context, Decisions

### Community 123 - "consumer.svelte.ts"
Cohesion: 0.17
Nodes (7): consumerStore, current, known, files, Recipe, RecipeParam, recipes

### Community 124 - "Sparrow -- Condensed Reference"
Cohesion: 0.17
Nodes (11): API Key Authentication, Architecture, Code Patterns & Conventions, Design Principles, Development History, Handler Pattern, HTTP Routing (chi), Known Gaps (+3 more)

### Community 125 - "WebhookWorker"
Cohesion: 0.14
Nodes (16): Config, Context, Duration, Job, JobInserter, Logger, WebhookWorker, Time (+8 more)

### Community 126 - "Sparrow Webhook Delivery Platform"
Cohesion: 0.33
Nodes (6): Sparrow Agent/Repo Conventions, River Queue (Postgres-backed workers), Svelte 5 Tutorial (PDF), Deploy Docs Workflow, Event-driven Fan-out Pipeline, Sparrow Webhook Delivery Platform

### Community 127 - "portal-embedding.mdx"
Cohesion: 0.20
Nodes (9): Alternative: no Sparrow UI at all, Example: Caddy (simplest — start here), Example: Express (your app's backend as the proxy), Example: nginx, How it works, end to end, Prerequisites, Security analysis, The route allowlist (+1 more)

### Community 128 - "models_test.go"
Cohesion: 0.39
Nodes (8): T, TestEventRegistration_NilJSONFieldsAreNullSafe(), TestJSONMap_RoundTrip(), TestJSONMap_Scan(), TestJSONMap_Value(), TestJSONStringMap_RoundTrip(), TestJSONStringMap_Scan(), TestJSONStringMap_Value()

### Community 129 - "steps_auth.py"
Cohesion: 0.42
Nodes (8): _get(), get_authed_correct_key(), get_authed_no_key(), get_authed_with_key(), step, Step implementations for API key enforcement (12_auth_enforcement.spec)., start_authed_server(), stop_authed_server()

### Community 130 - "CI Build Job"
Cohesion: 0.50
Nodes (5): CI Build Job, CI Workflow, CI Integration Test Job, CI Lint Job, CI Test Job (Postgres service)

### Community 131 - "EventReportWithStats"
Cohesion: 0.22
Nodes (3): normalizePagination(), EventReportFilter, EventReportWithStats

### Community 132 - "mapError"
Cohesion: 0.22
Nodes (7): Context, mapError(), API, registerHealthRoutes(), healthSummaryOutput, listWebhooksGlobalInput, webhookHealthOutput

### Community 133 - "NewWebhookTemplateContext"
Cohesion: 0.67
Nodes (5): NewWebhookTemplateContext(), T, loadSendgridTemplate(), TestSendgridRecipe_DeliveryFailed(), TestSendgridRecipe_HealthChanged()

### Community 134 - "services.ts"
Cohesion: 0.22
Nodes (6): api, portal, PortalSession, SparrowConfig, RFC-9457, Window

### Community 136 - "scripts"
Cohesion: 0.25
Nodes (8): scripts, build, check, check:watch, dev, gen:api-types, prepare, preview

### Community 137 - ".Work"
Cohesion: 0.52
Nodes (3): Context, Job, UUID

### Community 138 - "manifest.json"
Cohesion: 0.50
Nodes (3): Language, Plugins, html-report

### Community 139 - "dependencies"
Cohesion: 0.29
Nodes (7): js-yaml, openapi-fetch, dependencies, js-yaml, openapi-fetch, svelte-jsoneditor, svelte-jsoneditor

### Community 140 - "instructions"
Cohesion: 0.50
Nodes (3): instructions, $schema, plan.md

### Community 141 - "Deliberately NOT covered by e2e"
Cohesion: 0.40
Nodes (4): API surface deliberately not e2e-tested (removal / design candidates), Deliberately NOT covered by e2e, Known product-contract questions (asserted as-is, flagged), Out of e2e scope (belongs in unit/integration tests)

### Community 142 - "run"
Cohesion: 0.50
Nodes (4): Context, Logger, main(), run()

### Community 143 - "web/package.json"
Cohesion: 0.40
Nodes (4): name, private, type, version

### Community 144 - "Sparrow Web Dashboard"
Cohesion: 0.22
Nodes (8): Build, Embedding in the Go Binary, Environment Variables, Local Development, Prerequisites, Sparrow Web Dashboard, Standalone Deployment, Tech Stack

### Community 145 - "newPagination"
Cohesion: 0.67
Nodes (3): newPagination(), listWebhooksOutput, PaginationOutput

### Community 153 - "Config"
Cohesion: 0.43
Nodes (3): Config, Load(), validatePort()

### Community 164 - "Dual Webhook Signing (HMAC-SHA256 + Ed25519)"
Cohesion: 0.67
Nodes (3): Dual Webhook Signing (HMAC-SHA256 + Ed25519), Timestamp Replay Protection, Standard Webhooks Format

### Community 242 - "github.com/sarathsp06/sparrow"
Cohesion: 0.67
Nodes (4): github.com/sarathsp06/sparrow, github.com/sarathsp06/sparrow/pkg/signature, github.com/sarathsp06/sparrow/pkg/template, github.com/sarathsp06/sparrow/satellites/sparrow

## Knowledge Gaps
- **385 isolated node(s):** `DEFAULT_TOLERANCE_SECONDS`, `SignatureVerificationError`, `Headers`, `name`, `type` (+380 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **34 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Errorf()` connect `Errorf` to `setupEnv`, `testContext`, `EventReportWithStats`, `.Work`, `webhooks/models.go`, `NewService`, `Context`, `NewWebhookHandler`, `Config`, `BatchJob`, `apiClient`, `ClassifyError`, `envelope`, `VerifyHMAC`, `validConfig`, `runInit`, `WebhookService`, `TemplateEngine`, `ValidateIP`, `runPush`, `newEmailSink`, `otel.go`, `Context`, `newOTLPSink`, `runEvents`, `Client`, `resolveConfig`, `ServiceError`, `config`, `newS3Sink`, `.CreateWebhook`, `parseUUID`, `runUse`, `runListen`, `GetFunctionMap`, `newTailCmd`, `runFunctions`, `NewManager`, `WebhookService`, `jobInserter`, `EventProcessingWorker`, `Context`, `runTemplateTest`, `Handler`, `Context`, `WebhookWorker`?**
  _High betweenness centrality (0.288) - this node is a cross-community bridge._
- **Why does `NewManager()` connect `NewManager` to `setupEnv`, `Errorf`, `Client`, `RepositoryInterface`, `EventProcessingWorker`, `NewService`, `RetentionWorker`, `WebhookWorker`?**
  _High betweenness centrality (0.069) - this node is a cross-community bridge._
- **Why does `setupEnv()` connect `setupEnv` to `NewManager`, `Mount`, `testContext`, `webhook_service.go`, `WebhookServiceInterfaceWithTracing`, `NewService`, `Context`?**
  _High betweenness centrality (0.063) - this node is a cross-community bridge._
- **Are the 149 inferred relationships involving `Errorf()` (e.g. with `.Validate()` and `.EncryptJSON()`) actually correct?**
  _`Errorf()` has 149 INFERRED edges - model-reasoned connections that need verification._
- **What connects `DEFAULT_TOLERANCE_SECONDS`, `SignatureVerificationError`, `Headers` to the rest of the system?**
  _385 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `setupEnv` be split into smaller, more focused modules?**
  _Cohesion score 0.05069260241674035 - nodes in this community are weakly interconnected._
- **Should `event.go` be split into smaller, more focused modules?**
  _Cohesion score 0.10080645161290322 - nodes in this community are weakly interconnected._