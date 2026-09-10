# Graph Report - sparrow  (2026-09-10)

## Corpus Check
- 226 files · ~228,935 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1588 nodes · 3333 edges · 112 communities (85 shown, 27 thin omitted)
- Extraction: 85% EXTRACTED · 15% INFERRED · 0% AMBIGUOUS · INFERRED: 486 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `ae1f049e`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- setupEnv
- event.go
- Mount
- event_filtering_test.go
- subscription.go
- RepositoryInterfaceWithTracing
- delivery.go
- steps.py
- webhook.go
- sparrow_verify.py
- WebhookHTTPConfig
- NewService
- WebhookServiceInterfaceWithTracing
- Sparrow Detailed Flow Reference
- Context
- .Work
- sparrow-verify.ts
- Context
- api-types.d.ts
- TemplateCache
- TemplateEngine
- api.astro
- Prune Journal
- GetBuffer
- newPagination
- Dual Protocol (gRPC + Connect-RPC)
- Client Libraries
- BatchJob
- Error Classification (Reference)
- Template Functions
- WebhookClient
- Errorf
- Sparrow Architecture
- JobInserterWithTracing
- Context
- ClassifyError
- DefaultConfig
- PrepareDeliveryRequest
- Context
- .Work
- Sparrow Implementation Plan
- GetFunctionMap
- health.go
- WebhookServiceInterface
- scripts
- NewWebhookClient
- mapError
- ValidateIP
- utils.ts
- otel.go
- EventProcessingWorker
- parseRetryAfter
- devDependencies
- SparrowAPI
- RepositoryInterface
- ServiceError
- NewTemplateEngine
- .CreateWebhook
- mockRepo
- Sparrow Deployment Template
- compilerOptions
- WebhookTargetManager
- NewManager
- _Target
- Checker
- parseUUID
- jobInserter
- hooks.py
- SparrowEnvironment
- EventArgs
- Handler
- Sparrow -- Condensed Reference
- Sparrow Webhook Delivery Platform
- models_test.go
- CI Build Job
- manifest.json
- instructions
- Sparrow Web Dashboard
- Config
- src/components/Footer.astro
- docs/tsconfig.json
- TestJobInserter_InsertOpts_Merge
- TestWebhookWorkerDefaults
- Dual Webhook Signing (HMAC-SHA256 + Ed25519)
- +layout.ts
- WithConn Transaction Pattern
- ../../components/ThemeDiagram.astro
- content.config.ts
- proto2astro
- index.astro
- WebhookService
- app.d.ts
- svelte.config.js
- Sparrow Ingress Template
- Sparrow Chart NOTES
- Sparrow PodDisruptionBudget Template
- Release Workflow
- github.com/sarathsp06/sparrow
- sparrow-e2e
- Envelope Encryption at Rest (AES-256-GCM)

## God Nodes (most connected - your core abstractions)
1. `Errorf()` - 115 edges
2. `RepositoryInterfaceWithTracing` - 68 edges
3. `WebhookServiceInterfaceWithTracing` - 39 edges
4. `EventSubscription` - 38 edges
5. `NewService()` - 33 edges
6. `setupEnv()` - 32 edges
7. `WebhookDelivery` - 31 edges
8. `testContext()` - 30 edges
9. `NewWebhookService()` - 26 edges
10. `EventRegistration` - 24 edges

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

## Communities (112 total, 27 thin omitted)

### Community 0 - "setupEnv"
Cohesion: 0.08
Nodes (66): Container, HandlerFunc, Int32, capturedWebhook, deliveryItem, restClient, testEnv, Context (+58 more)

### Community 1 - "event.go"
Cohesion: 0.12
Nodes (25): API, registerEventRoutes(), toBatchJobOutput(), toEventTypeItem(), toEventTypeOutput(), batchJobOutput, eventIDOnlyInput, eventOccurrenceItem (+17 more)

### Community 2 - "Mount"
Cohesion: 0.22
Nodes (6): main(), API, Mount(), T, TestOpenAPISpecMatchesCommitted(), Router

### Community 3 - "event_filtering_test.go"
Cohesion: 0.09
Nodes (57): Context, T, UUID, TestCreateSubscription_CatchAllWithLabelFilters(), TestCreateSubscription_WithEmptyLabelFilters(), TestCreateSubscription_WithInvalidLabelFilters(), TestCreateSubscription_WithLabelFilters(), TestGetSubscriptionsByEvent_CatchAllReturned() (+49 more)

### Community 4 - "subscription.go"
Cohesion: 0.19
Nodes (16): API, registerSubscriptionRoutes(), toSubscriptionItem(), toSubscriptionOutput(), createSubscriptionBody, createSubscriptionInput, listSubscriptionsInput, listSubscriptionsOutput (+8 more)

### Community 5 - "RepositoryInterfaceWithTracing"
Cohesion: 0.05
Nodes (34): Int64Array, Time, UUID, Value, Context, Span, Time, UUID (+26 more)

### Community 6 - "delivery.go"
Cohesion: 0.19
Nodes (15): API, registerDeliveryRoutes(), toDeliveryItem(), toDeliveryOutput(), attemptItem, attemptsOutput, deliveryIDInput, deliveryIDOnlyInput (+7 more)

### Community 7 - "steps.py"
Cohesion: 0.08
Nodes (58): delivery_has_signature_headers(), SignatureVerifier -- Verifies HMAC-SHA256 (v1,) and Ed25519 (v1a,) signatures…, Return the exact body bytes (as text) that Sparrow signed. Standard Webhooks…, Verify HMAC-SHA256 signature (v1, prefix)., Verify Ed25519 signature (v1a, prefix)., Assert delivery has Standard Webhooks signature headers., _signed_body(), verify_ed25519_signature() (+50 more)

### Community 8 - "webhook.go"
Cohesion: 0.22
Nodes (12): emptyOutput, listWebhooksInput, namespaceOnlyInput, namespaceStatsOutput, patchWebhookBody, patchWebhookInput, registerWebhookBody, registerWebhookInput (+4 more)

### Community 9 - "sparrow_verify.py"
Cohesion: 0.32
Nodes (11): _check_timestamp(), _decode_signatures(), Verify Sparrow webhook delivery signatures (Standard Webhooks format). Every…, Raised when a delivery signature cannot be verified., Verify the ``v1,`` (HMAC-SHA256) signature. Raises on failure. ``payload`` must…, Verify the ``v1a,`` (Ed25519) signature. Raises on failure. ``payload`` must be…, _required_headers(), SignatureVerificationError (+3 more)

### Community 10 - "WebhookHTTPConfig"
Cohesion: 0.11
Nodes (16): DefaultWebhookHTTPConfig(), Duration, Time, Value, T, TestApplyConfig_RateLimitRPS(), TestDefaultWebhookHTTPConfig_RateLimitRPS(), TestToWebhookRegistration_RateLimitRPS() (+8 more)

### Community 11 - "NewService"
Cohesion: 0.12
Nodes (36): AEAD, Service, IsEnvelopeEncrypted(), newAEAD(), NewService(), ParseKey(), T, TestDecrypt_BackwardCompatibility() (+28 more)

### Community 12 - "WebhookServiceInterfaceWithTracing"
Cohesion: 0.07
Nodes (12): Context, Time, WebhookService, Context, Span, Time, WebhookRegistration, NewWebhookServiceInterfaceWithTracing() (+4 more)

### Community 13 - "Sparrow Detailed Flow Reference"
Cohesion: 0.08
Nodes (39): PostgreSQL StatefulSet, Sparrow Kubernetes Service, Sparrow Helm Chart Values, Sparrow Client Libraries README, Connect-RPC, DeliveryService, Envelope Encryption (AES-256-GCM), 10-Category Error Classification (+31 more)

### Community 14 - "Context"
Cohesion: 0.13
Nodes (19): T, TestE2E_EventDeliveryStats_NoDeliveries(), Context, Repository, Tx, UUID, WebhookRegistration, NewRepository() (+11 more)

### Community 15 - ".Work"
Cohesion: 0.20
Nodes (11): Config, Context, Job, Logger, Tracer, UUID, WorkerDefaults, NewWebhookWorker() (+3 more)

### Community 16 - "sparrow-verify.ts"
Cohesion: 0.39
Nodes (7): decodeSignatures(), DEFAULT_TOLERANCE_SECONDS, Headers, parseHeaders(), SignatureVerificationError, verifyEd25519(), verifyHmac()

### Community 17 - "Context"
Cohesion: 0.35
Nodes (3): Context, Repository, UUID

### Community 18 - "api-types.d.ts"
Cohesion: 0.29
Nodes (6): RFC-3339, components, $defs, operations, paths, webhooks

### Community 19 - "TemplateCache"
Cohesion: 0.24
Nodes (10): Cache, TemplateCache, Template, hashTemplate(), NewTemplateCache(), T, TestHashTemplate(), TestTemplateCacheBasicOperations() (+2 more)

### Community 20 - "TemplateEngine"
Cohesion: 0.20
Nodes (8): limitedWriter, TemplateEngine, writerWithBytes, FuncMap, Template, WebhookTemplateContext, NewTemplateEngineWithCacheSize(), NewWebhookTemplateContext()

### Community 22 - "Prune Journal"
Cohesion: 0.50
Nodes (3): 2026-08-10 - Dead-code baseline established, 2026-09-09 - Ed25519 keygen duplication removed, Prune Journal

### Community 23 - "GetBuffer"
Cohesion: 0.28
Nodes (11): Buffer, GetBuffer(), GetHeaderMap(), PutBuffer(), PutHeaderMap(), BenchmarkBufferPool(), BenchmarkHeaderMapPool(), B (+3 more)

### Community 27 - "BatchJob"
Cohesion: 0.12
Nodes (12): Context, Repository, UUID, Context, Repository, UUID, Context, WebhookService (+4 more)

### Community 32 - "WebhookClient"
Cohesion: 0.20
Nodes (7): WebhookClient, Client, Config, Context, Duration, Response, WebhookTemplateContext

### Community 33 - "Errorf"
Cohesion: 0.20
Nodes (7): generateSamplePayload(), Context, Time, WebhookService, ValidateJSONSchema(), Errorf(), SchemaValidationError

### Community 34 - "Sparrow Architecture"
Cohesion: 0.29
Nodes (7): Sparrow Architecture, Dual-Protocol API, Event Processing Pipeline, Health State Machine, Leaky Bucket Rate Limiting, Standard Webhooks Signing, Default Webhook Body Envelope

### Community 35 - "JobInserterWithTracing"
Cohesion: 0.31
Nodes (7): Context, JobArgs, JobInserter, JobInsertResult, Span, NewJobInserterWithTracing(), JobInserterWithTracing

### Community 36 - "Context"
Cohesion: 0.53
Nodes (3): Context, Repository, UUID

### Community 37 - "ClassifyError"
Cohesion: 0.16
Nodes (23): ErrorCategory, timeoutError, classifyByMessage(), ClassifyError(), ClassifyHTTPStatus(), classifySyscallError(), isDNSError(), IsRetryableCategory() (+15 more)

### Community 38 - "DefaultConfig"
Cohesion: 0.33
Nodes (5): Config, DefaultConfig(), Duration, T, TestDefaultConfig()

### Community 39 - "PrepareDeliveryRequest"
Cohesion: 0.10
Nodes (38): DeliveryRequest, WebhookEnvelope, BuildEnvelopePayload(), BuildRequest(), generateEd25519Signature(), generateHMACSignature(), Context, Duration (+30 more)

### Community 40 - "Context"
Cohesion: 0.36
Nodes (4): Context, Repository, Time, UUID

### Community 41 - ".Work"
Cohesion: 0.52
Nodes (3): Context, Job, UUID

### Community 42 - "Sparrow Implementation Plan"
Cohesion: 0.05
Nodes (43): API Changes, API Changes, Commands, Completed Parts (v0.8.0 -- v1.2.1), Configuration, Configuration, Current State (as of v1.2.1), Decisions Log (+35 more)

### Community 43 - "GetFunctionMap"
Cohesion: 0.60
Nodes (4): TemplateFunc, GetFunctionMap(), GetTemplateFunctions(), FuncMap

### Community 44 - "health.go"
Cohesion: 0.50
Nodes (3): healthSummaryOutput, listWebhooksGlobalInput, webhookHealthOutput

### Community 45 - "WebhookServiceInterface"
Cohesion: 0.22
Nodes (17): getWebhookEventsMap(), Context, WebhookRegistration, maskEncryptedSecret(), maskSecret(), maskSecretHeaders(), toWebhookOut(), toWebhookOutFromDomain() (+9 more)

### Community 47 - "scripts"
Cohesion: 0.08
Nodes (25): @astrojs/starlight, dependencies, astro, @astrojs/starlight, marked, @scalar/astro, sharp, zod (+17 more)

### Community 48 - "NewWebhookClient"
Cohesion: 0.24
Nodes (16): NewWebhookClient(), ReadBody(), BenchmarkSend(), BenchmarkTransformPayload(), B, T, TestClientClose(), TestNewWebhookClient() (+8 more)

### Community 51 - "ValidateIP"
Cohesion: 0.24
Nodes (9): Request, ssrfDialControl(), ssrfSafeCheckRedirect(), T, TestValidateIP(), ValidateIP(), validateRedirectURL(), IP (+1 more)

### Community 54 - "utils.ts"
Cohesion: 0.08
Nodes (21): $lib/api-types, current, known, namespaceStore, beat, data, pulseStore, Telemetry (+13 more)

### Community 55 - "otel.go"
Cohesion: 0.18
Nodes (19): Int64Counter, Int64UpDownCounter, DefaultConfig(), GetMeter(), GetTracer(), Context, Tracer, newLoggerProvider() (+11 more)

### Community 56 - "EventProcessingWorker"
Cohesion: 0.24
Nodes (8): Context, Job, JobInserter, Logger, WorkerDefaults, NewEventProcessingWorker(), EventProcessingWorker, SubscriptionRepository

### Community 60 - "parseRetryAfter"
Cohesion: 0.30
Nodes (10): T, TestDefaultAndMaxRetryAfterConstants(), TestIsSuccessStatusCode(), TestParseRetryAfter(), TestParseRetryAfter_HTTPDate(), TestParseRetryAfter_HTTPDate_FarFuture(), TestParseRetryAfter_HTTPDate_Past(), Duration (+2 more)

### Community 62 - "devDependencies"
Cohesion: 0.04
Nodes (46): openapi-fetch, openapi-typescript, svelte-check, @sveltejs/adapter-static, @sveltejs/kit, @sveltejs/vite-plugin-svelte, tailwindcss, @tailwindcss/forms (+38 more)

### Community 67 - "SparrowAPI"
Cohesion: 0.15
Nodes (6): Poll until all deliveries reach terminal status., Poll a single delivery until terminal., Convert a generated attrs model (or None) to a plain dict., Client for Sparrow's REST API, built on the generated sparrow_client SDK., SparrowAPI, _to_dict()

### Community 68 - "RepositoryInterface"
Cohesion: 0.19
Nodes (11): JobInserter, Logger, WorkerDefaults, NewBatchJobWorker(), BatchJobWorker, BatchRepository, EventRepository, EventTypeRepository (+3 more)

### Community 71 - "ServiceError"
Cohesion: 0.21
Nodes (11): ServiceError, Status, Error(), T, TestConstructors(), TestServiceError_ClientMessage(), TestServiceError_Error(), TestServiceError_ErrorsAs() (+3 more)

### Community 72 - "NewTemplateEngine"
Cohesion: 0.29
Nodes (15): NewTemplateEngine(), BenchmarkExecuteComplex(), BenchmarkExecuteSimple(), B, T, TestExecuteComplexTemplate(), TestExecuteEmptyTemplate(), TestExecuteInvalidTemplate() (+7 more)

### Community 75 - ".CreateWebhook"
Cohesion: 0.22
Nodes (7): ValidateWebhookURL(), generateWebhookSecret(), Context, Time, WebhookRegistration, WebhookService, SignatureType

### Community 78 - "mockRepo"
Cohesion: 0.18
Nodes (8): Context, JobArgs, JobInsertResult, UUID, WebhookRegistration, Mock, mockJobInserter, mockRepo

### Community 82 - "Sparrow Deployment Template"
Cohesion: 0.18
Nodes (11): Sparrow Helm Chart, Sparrow ConfigMap Template, Sparrow Deployment Template, Sparrow HPA Template, Sparrow NetworkPolicy Template, Sparrow PostgreSQL Service Template, Conventional Commits Convention, Release Docker Image Job (+3 more)

### Community 88 - "compilerOptions"
Cohesion: 0.14
Nodes (13): ./.svelte-kit/tsconfig.json, compilerOptions, allowJs, checkJs, esModuleInterop, forceConsistentCasingInFileNames, moduleResolution, resolveJsonModule (+5 more)

### Community 89 - "WebhookTargetManager"
Cohesion: 0.15
Nodes (5): before_scenario, Manages mock webhook target servers., WebhookTargetManager, Fresh target manager for each scenario., setup_scenario()

### Community 96 - "NewManager"
Cohesion: 0.22
Nodes (9): Client, Config, Context, JobInserter, Logger, Pool, Tx, NewManager() (+1 more)

### Community 98 - "_Target"
Cohesion: 0.23
Nodes (4): CapturedDelivery, WebhookTargetServer -- Programmable mock webhook endpoints for e2e tests. Each…, Start a mock webhook target. Returns the URL., _Target

### Community 99 - "Checker"
Cohesion: 0.32
Nodes (7): Checker, HealthResponse, ReadyResponse, Context, Pool, Time, NewChecker()

### Community 100 - "parseUUID"
Cohesion: 0.18
Nodes (8): Context, WebhookService, UUID, parseUUID(), Context, Time, WebhookService, paginateSubscriptions()

### Community 102 - "jobInserter"
Cohesion: 0.32
Nodes (8): Client, Context, JobArgs, JobInsertResult, Logger, Tx, NewJobInserter(), jobInserter

### Community 103 - "hooks.py"
Cohesion: 0.17
Nodes (10): after_scenario, after_suite, before_suite, SparrowAPI -- wraps the generated REST client (sparrow_client, generated from…, Gauge hooks for suite/scenario setup and teardown., Start Postgres + Sparrow containers., Stop all mock targets., setup_environment() (+2 more)

### Community 104 - "SparrowEnvironment"
Cohesion: 0.24
Nodes (4): SparrowEnvironment -- Manages Postgres and Sparrow containers via…, Manages the Sparrow test environment (Postgres + Sparrow containers)., Start Postgres + Sparrow. Returns the Sparrow HTTP URL., SparrowEnvironment

### Community 107 - "EventArgs"
Cohesion: 0.20
Nodes (5): InsertOpts, Time, BatchJobArgs, EventArgs, WebhookArgs

### Community 116 - "Handler"
Cohesion: 0.10
Nodes (18): main(), GetMigrationsFS(), FS, Request, SecurityHeaders(), T, TestSecurityHeaders(), TestSecurityHeadersPassthrough() (+10 more)

### Community 124 - "Sparrow -- Condensed Reference"
Cohesion: 0.17
Nodes (11): API Key Authentication, Architecture, Code Patterns & Conventions, Design Principles, Development History, Handler Pattern, HTTP Routing (chi), Known Gaps (+3 more)

### Community 126 - "Sparrow Webhook Delivery Platform"
Cohesion: 0.33
Nodes (6): Sparrow Agent/Repo Conventions, River Queue (Postgres-backed workers), Svelte 5 Tutorial (PDF), Deploy Docs Workflow, Event-driven Fan-out Pipeline, Sparrow Webhook Delivery Platform

### Community 128 - "models_test.go"
Cohesion: 0.39
Nodes (8): T, TestEventRegistration_NilJSONFieldsAreNullSafe(), TestJSONMap_RoundTrip(), TestJSONMap_Scan(), TestJSONMap_Value(), TestJSONStringMap_RoundTrip(), TestJSONStringMap_Scan(), TestJSONStringMap_Value()

### Community 130 - "CI Build Job"
Cohesion: 0.50
Nodes (5): CI Build Job, CI Workflow, CI Integration Test Job, CI Lint Job, CI Test Job (Postgres service)

### Community 138 - "manifest.json"
Cohesion: 0.50
Nodes (3): Language, Plugins, html-report

### Community 140 - "instructions"
Cohesion: 0.50
Nodes (3): instructions, $schema, plan.md

### Community 144 - "Sparrow Web Dashboard"
Cohesion: 0.22
Nodes (8): Build, Embedding in the Go Binary, Environment Variables, Local Development, Prerequisites, Sparrow Web Dashboard, Standalone Deployment, Tech Stack

### Community 153 - "Config"
Cohesion: 0.47
Nodes (3): Config, Load(), validatePort()

### Community 164 - "Dual Webhook Signing (HMAC-SHA256 + Ed25519)"
Cohesion: 0.67
Nodes (3): Dual Webhook Signing (HMAC-SHA256 + Ed25519), Timestamp Replay Protection, Standard Webhooks Format

## Knowledge Gaps
- **217 isolated node(s):** `DEFAULT_TOLERANCE_SECONDS`, `SignatureVerificationError`, `Headers`, `name`, `type` (+212 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **27 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Errorf()` connect `Errorf` to `setupEnv`, `event_filtering_test.go`, `RepositoryInterfaceWithTracing`, `WebhookHTTPConfig`, `NewService`, `WebhookServiceInterfaceWithTracing`, `Context`, `.Work`, `TemplateEngine`, `Config`, `BatchJob`, `ClassifyError`, `PrepareDeliveryRequest`, `Context`, `.Work`, `GetFunctionMap`, `WebhookService`, `ValidateIP`, `otel.go`, `EventProcessingWorker`, `ServiceError`, `.CreateWebhook`, `NewManager`, `parseUUID`, `jobInserter`, `Handler`?**
  _High betweenness centrality (0.209) - this node is a cross-community bridge._
- **Why does `setupEnv()` connect `setupEnv` to `NewManager`, `Mount`, `event_filtering_test.go`, `PrepareDeliveryRequest`, `NewService`, `WebhookServiceInterfaceWithTracing`, `Context`?**
  _High betweenness centrality (0.073) - this node is a cross-community bridge._
- **Why does `NewManager()` connect `NewManager` to `setupEnv`, `Errorf`, `RepositoryInterface`, `NewService`, `.Work`, `EventProcessingWorker`?**
  _High betweenness centrality (0.062) - this node is a cross-community bridge._
- **Are the 112 inferred relationships involving `Errorf()` (e.g. with `.Write()` and `.Execute()`) actually correct?**
  _`Errorf()` has 112 INFERRED edges - model-reasoned connections that need verification._
- **What connects `DEFAULT_TOLERANCE_SECONDS`, `SignatureVerificationError`, `Headers` to the rest of the system?**
  _217 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `setupEnv` be split into smaller, more focused modules?**
  _Cohesion score 0.07826384142173616 - nodes in this community are weakly interconnected._
- **Should `event.go` be split into smaller, more focused modules?**
  _Cohesion score 0.12 - nodes in this community are weakly interconnected._