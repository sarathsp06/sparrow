# Graph Report - sparrow  (2026-09-13)

## Corpus Check
- 276 files · ~251,560 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1955 nodes · 4056 edges · 144 communities (118 shown, 26 thin omitted)
- Extraction: 84% EXTRACTED · 16% INFERRED · 0% AMBIGUOUS · INFERRED: 638 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `7c9492f7`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- newRESTClient
- event.go
- Mount
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
- WebhookWorker
- sparrow-verify.ts
- WebhookDelivery
- api-types.d.ts
- TemplateCache
- NewWebhookHandler
- api.astro
- Prune Journal
- store/models.go
- setupEnv
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
- PrepareDeliveryRequest
- Context
- .Work
- Sparrow Implementation Plan
- webhook_service.go
- mapError
- registerWebhookRoutes
- scripts
- NewWebhookClient
- TemplateEngine
- NewTemplateEngine
- ValidateIP
- parseWithArg
- newEmailSink
- utils.ts
- otel.go
- EventProcessingWorker
- Commands
- Context
- postSink
- parseRetryAfter
- newOTLPSink
- devDependencies
- HandlerFunc
- run
- Client
- startWebhookTarget
- SparrowAPI
- RepositoryInterface
- TestRecipes
- resolveConfig
- ServiceError
- run
- newS3Sink
- sinks.mdx
- .CreateWebhook
- newAPIClient
- runPush
- mockRepo
- Included recipes
- sources.mdx
- TestE2E_EmailSink
- Sparrow Deployment Template
- parseUUID
- runUse
- RunAllMigrations
- runTail
- restClient
- compilerOptions
- WebhookTargetManager
- TestE2E_SourcesGitHubWebhook
- runListen
- .Execute
- Sparrow Recipes
- TestCLI_E2E
- TestE2E_SlackRecipeTransform
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
- TestWebhookWorkerNextRetry
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
1. `Errorf()` - 149 edges
2. `RepositoryInterfaceWithTracing` - 68 edges
3. `WebhookServiceInterfaceWithTracing` - 40 edges
4. `EventSubscription` - 40 edges
5. `setupEnv()` - 39 edges
6. `NewService()` - 33 edges
7. `WebhookDelivery` - 31 edges
8. `testContext()` - 29 edges
9. `NewWebhookService()` - 25 edges
10. `newRESTClient()` - 24 edges

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

## Communities (144 total, 26 thin omitted)

### Community 0 - "newRESTClient"
Cohesion: 0.33
Nodes (27): Int32, deliveryItem, Context, Server, T, pollBatchJob(), pollDeliveryStatus(), pushTestEvent() (+19 more)

### Community 1 - "event.go"
Cohesion: 0.12
Nodes (25): API, registerEventRoutes(), toBatchJobOutput(), toEventTypeItem(), toEventTypeOutput(), batchJobOutput, eventIDOnlyInput, eventOccurrenceItem (+17 more)

### Community 2 - "Mount"
Cohesion: 0.22
Nodes (6): main(), API, Mount(), T, TestOpenAPISpecMatchesCommitted(), Router

### Community 3 - "event_filtering_test.go"
Cohesion: 0.10
Nodes (52): Context, T, UUID, TestCreateSubscription_CatchAllWithLabelFilters(), TestCreateSubscription_WithEmptyLabelFilters(), TestCreateSubscription_WithInvalidLabelFilters(), TestCreateSubscription_WithLabelFilters(), TestGetSubscriptionsByEvent_CatchAllReturned() (+44 more)

### Community 4 - "subscription.go"
Cohesion: 0.16
Nodes (18): newPagination(), API, registerSubscriptionRoutes(), toSubscriptionItem(), toSubscriptionOutput(), createSubscriptionBody, createSubscriptionInput, listSubscriptionsInput (+10 more)

### Community 5 - "RepositoryInterfaceWithTracing"
Cohesion: 0.10
Nodes (14): Context, Span, Time, UUID, WebhookRegistration, NewRepositoryInterfaceWithTracing(), BatchJobData, BatchJobStatus (+6 more)

### Community 6 - "delivery.go"
Cohesion: 0.19
Nodes (15): API, registerDeliveryRoutes(), toDeliveryItem(), toDeliveryOutput(), attemptItem, attemptsOutput, deliveryIDInput, deliveryIDOnlyInput (+7 more)

### Community 7 - "steps.py"
Cohesion: 0.08
Nodes (58): delivery_has_signature_headers(), SignatureVerifier -- Verifies HMAC-SHA256 (v1,) and Ed25519 (v1a,) signatures…, Return the exact body bytes (as text) that Sparrow signed. Standard Webhooks…, Verify HMAC-SHA256 signature (v1, prefix)., Verify Ed25519 signature (v1a, prefix)., Assert delivery has Standard Webhooks signature headers., _signed_body(), verify_ed25519_signature() (+50 more)

### Community 8 - "rest/webhook.go"
Cohesion: 0.20
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
Nodes (13): Context, Time, WebhookService, Context, Span, Time, UUID, WebhookRegistration (+5 more)

### Community 13 - "Sparrow Detailed Flow Reference"
Cohesion: 0.08
Nodes (39): PostgreSQL StatefulSet, Sparrow Kubernetes Service, Sparrow Helm Chart Values, Sparrow Client Libraries README, Connect-RPC, DeliveryService, Envelope Encryption (AES-256-GCM), 10-Category Error Classification (+31 more)

### Community 14 - "Context"
Cohesion: 0.13
Nodes (19): T, TestE2E_EventDeliveryStats_NoDeliveries(), Context, Repository, Tx, UUID, WebhookRegistration, NewRepository() (+11 more)

### Community 15 - "WebhookWorker"
Cohesion: 0.22
Nodes (10): Context, Duration, Job, Logger, Time, Tracer, UUID, WorkerDefaults (+2 more)

### Community 16 - "sparrow-verify.ts"
Cohesion: 0.39
Nodes (7): decodeSignatures(), DEFAULT_TOLERANCE_SECONDS, Headers, parseHeaders(), SignatureVerificationError, verifyEd25519(), verifyHmac()

### Community 17 - "WebhookDelivery"
Cohesion: 0.18
Nodes (5): Context, Repository, UUID, WebhookDelivery, WebhookDeliveryStatus

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
Nodes (14): Int64Array, Time, UUID, Value, DeliveryFilter, EventRecord, JSONMap, JSONStringMap (+6 more)

### Community 24 - "setupEnv"
Cohesion: 0.15
Nodes (20): Container, testEnv, T, TestAcquireDeliverySlot_BusyBucketDoesNotBurnSlots(), Context, Pool, Server, T (+12 more)

### Community 27 - "BatchJob"
Cohesion: 0.12
Nodes (12): Context, Repository, UUID, Context, Repository, UUID, RawMessage, Context (+4 more)

### Community 32 - "apiClient"
Cohesion: 0.21
Nodes (10): Context, apiClient, apiError, deliveryItem, pushBody, pushResult, subscriptionItem, subscriptionPatch (+2 more)

### Community 33 - "Errorf"
Cohesion: 0.20
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

### Community 39 - "PrepareDeliveryRequest"
Cohesion: 0.10
Nodes (38): DeliveryRequest, WebhookEnvelope, BuildEnvelopePayload(), BuildRequest(), generateEd25519Signature(), generateHMACSignature(), Context, Duration (+30 more)

### Community 40 - "Context"
Cohesion: 0.38
Nodes (4): Context, Repository, Time, UUID

### Community 41 - ".Work"
Cohesion: 0.52
Nodes (3): Context, Job, UUID

### Community 42 - "Sparrow Implementation Plan"
Cohesion: 0.05
Nodes (43): API Changes, API Changes, Commands, Completed Parts (v0.8.0 -- v1.2.1), Configuration, Configuration, Current State (as of v1.2.1), Decisions Log (+35 more)

### Community 43 - "webhook_service.go"
Cohesion: 0.23
Nodes (14): WithAllowPrivateNetworks(), deliveryRouteService, eventRouteService, healthRouteService, webhookRouteService, BatchManager, DeliveryManager, EventManager (+6 more)

### Community 44 - "mapError"
Cohesion: 0.22
Nodes (7): Context, mapError(), API, registerHealthRoutes(), healthSummaryOutput, listWebhooksGlobalInput, webhookHealthOutput

### Community 45 - "registerWebhookRoutes"
Cohesion: 0.23
Nodes (16): getWebhookEventsMap(), Context, WebhookRegistration, maskEncryptedSecret(), maskSecret(), maskSecretHeaders(), toWebhookOut(), toWebhookOutFromDomain() (+8 more)

### Community 47 - "scripts"
Cohesion: 0.08
Nodes (25): @astrojs/starlight, dependencies, astro, @astrojs/starlight, marked, @scalar/astro, sharp, zod (+17 more)

### Community 48 - "NewWebhookClient"
Cohesion: 0.08
Nodes (36): Config, WebhookClient, Config, Context, Duration, Response, WebhookTemplateContext, NewWebhookClient() (+28 more)

### Community 49 - "TemplateEngine"
Cohesion: 0.17
Nodes (11): GetFunctionMap(), GetTemplateFunctions(), FuncMap, FuncMap, WebhookTemplateContext, NewTemplateEngineWithCacheSize(), NewWebhookTemplateContext(), limitedWriter (+3 more)

### Community 50 - "NewTemplateEngine"
Cohesion: 0.29
Nodes (15): NewTemplateEngine(), BenchmarkExecuteComplex(), BenchmarkExecuteSimple(), B, T, TestExecuteComplexTemplate(), TestExecuteEmptyTemplate(), TestExecuteInvalidTemplate() (+7 more)

### Community 51 - "ValidateIP"
Cohesion: 0.24
Nodes (9): Request, ssrfDialControl(), ssrfSafeCheckRedirect(), T, TestValidateIP(), ValidateIP(), validateRedirectURL(), IP (+1 more)

### Community 52 - "parseWithArg"
Cohesion: 0.15
Nodes (8): FlagSet, parseJSONArg(), parseWithArg(), T, TestKVFlag(), TestParseJSONArg(), kvFlag, listFlag

### Community 53 - "newEmailSink"
Cohesion: 0.37
Nodes (12): Conn, newEmailSink(), T, serveSMTP(), startSMTPServer(), TestEmailRender_BadTemplate(), TestEmailRender_CustomTemplatesAndHeaderInjection(), TestEmailRender_Defaults() (+4 more)

### Community 54 - "utils.ts"
Cohesion: 0.08
Nodes (21): $lib/api-types, current, known, namespaceStore, beat, data, pulseStore, Telemetry (+13 more)

### Community 55 - "otel.go"
Cohesion: 0.18
Nodes (19): Int64Counter, Int64UpDownCounter, DefaultConfig(), GetMeter(), GetTracer(), Context, Tracer, newLoggerProvider() (+11 more)

### Community 56 - "EventProcessingWorker"
Cohesion: 0.19
Nodes (10): Context, Job, JobInserter, Logger, WorkerDefaults, NewEventProcessingWorker(), Time, EventArgs (+2 more)

### Community 57 - "Commands"
Cohesion: 0.15
Nodes (12): 90-second quickstart, Commands, Configuration, How it's connected, Install, `sparrow init`, `sparrow listen`, `sparrow push <event-name>` (+4 more)

### Community 58 - "Context"
Cohesion: 0.38
Nodes (3): Context, Repository, UUID

### Community 59 - "postSink"
Cohesion: 0.40
Nodes (12): ResponseRecorder, Context, Header, T, postSink(), signHeaders(), TestSinkHandler_DownstreamFailureIs502(), TestSinkHandler_MissingHeaders() (+4 more)

### Community 60 - "parseRetryAfter"
Cohesion: 0.35
Nodes (9): T, TestDefaultAndMaxRetryAfterConstants(), TestIsSuccessStatusCode(), TestParseRetryAfter(), TestParseRetryAfter_HTTPDate(), TestParseRetryAfter_HTTPDate_FarFuture(), TestParseRetryAfter_HTTPDate_Past(), isSuccessStatusCode() (+1 more)

### Community 61 - "newOTLPSink"
Cohesion: 0.27
Nodes (10): Context, logsURL(), newOTLPSink(), otlpPayload(), T, TestLogsURL(), TestOTLPSinkDeliver(), TestOTLPSinkDeliverDownstreamFailure() (+2 more)

### Community 62 - "devDependencies"
Cohesion: 0.04
Nodes (46): openapi-fetch, openapi-typescript, svelte-check, @sveltejs/adapter-static, @sveltejs/kit, @sveltejs/vite-plugin-svelte, tailwindcss, @tailwindcss/forms (+38 more)

### Community 63 - "HandlerFunc"
Cohesion: 0.29
Nodes (9): HandlerFunc, T, TestAPIKeyHTTPMiddlewareAcceptsHeader(), TestAPIKeyHTTPMiddlewareBypassesExcludedPath(), TestAPIKeyHTTPMiddlewareRejectsQueryParameter(), SecurityHeaders(), T, TestSecurityHeaders() (+1 more)

### Community 64 - "run"
Cohesion: 0.21
Nodes (9): Context, Writer, main(), run(), Writer, runTemplateTest(), T, TestTemplateTestRendersFixtureRecipe() (+1 more)

### Community 65 - "Client"
Cohesion: 0.27
Nodes (7): Context, RawMessage, SparrowConfig, NewClient(), T, TestClientAutoCreatesEventType(), Client

### Community 66 - "startWebhookTarget"
Cohesion: 0.31
Nodes (10): capturedWebhook, Context, Duration, Header, Server, T, pollDeliverySuccess(), startWebhookTarget() (+2 more)

### Community 67 - "SparrowAPI"
Cohesion: 0.15
Nodes (6): Poll until all deliveries reach terminal status., Poll a single delivery until terminal., Convert a generated attrs model (or None) to a plain dict., Client for Sparrow's REST API, built on the generated sparrow_client SDK., SparrowAPI, _to_dict()

### Community 68 - "RepositoryInterface"
Cohesion: 0.14
Nodes (16): JobInserter, Logger, WorkerDefaults, NewBatchJobWorker(), Config, NewWebhookWorker(), BatchJobWorker, BatchRepository (+8 more)

### Community 69 - "TestRecipes"
Cohesion: 0.25
Nodes (9): T, WebhookTemplateContext, sampleContext(), substituteParams(), TestRecipes(), tokenRefs(), T, TestLoadRecipeFixture() (+1 more)

### Community 70 - "resolveConfig"
Cohesion: 0.36
Nodes (9): configPath(), resolveConfig(), saveConfig(), T, TestResolveConfigDefaults(), TestResolveConfigPrecedence(), TestSaveConfigPermissions(), writeConfigFile() (+1 more)

### Community 71 - "ServiceError"
Cohesion: 0.18
Nodes (12): ServiceError, Status, Classify(), Error(), T, TestConstructors(), TestServiceError_ClientMessage(), TestServiceError_Error() (+4 more)

### Community 72 - "run"
Cohesion: 0.27
Nodes (8): loadConfig(), Context, Logger, main(), run(), config, emailConfig, smtpConfig

### Community 73 - "newS3Sink"
Cohesion: 0.27
Nodes (8): Context, newS3Sink(), objectKey(), T, TestObjectKey(), TestS3Sink_PutObject(), s3Config, s3Sink

### Community 74 - "sinks.mdx"
Cohesion: 0.20
Nodes (9): Configuration, How it's connected, Install and run, Retry semantics, Sink or recipe?, Untransformed envelope required, Walkthrough: email sink, Walkthrough: OTLP export (+1 more)

### Community 75 - ".CreateWebhook"
Cohesion: 0.22
Nodes (7): ValidateWebhookURL(), generateWebhookSecret(), Context, Time, WebhookRegistration, WebhookService, SignatureType

### Community 76 - "newAPIClient"
Cohesion: 0.27
Nodes (9): File, config, newAPIClient(), config, Context, Writer, isTerminal(), probeServer() (+1 more)

### Community 77 - "runPush"
Cohesion: 0.31
Nodes (8): Context, Writer, runPush(), Server, T, pushEnv(), TestPushAutoCreatesEventType(), TestPushRequestShape()

### Community 78 - "mockRepo"
Cohesion: 0.18
Nodes (8): Context, JobArgs, JobInsertResult, UUID, WebhookRegistration, Mock, mockJobInserter, mockRepo

### Community 79 - "Included recipes"
Cohesion: 0.22
Nodes (8): clickhouse, discord, How it's connected, Included recipes, ntfy, pagerduty, slack, Writing your own

### Community 80 - "sources.mdx"
Cohesion: 0.22
Nodes (8): Configuration, Delivery semantics, Event naming, GitHub, How it's connected, Install and run, Provider setup, Stripe

### Community 81 - "TestE2E_EmailSink"
Cohesion: 0.50
Nodes (8): sinkSMTPMessage, freePort(), Context, T, startSinksBinary(), startSinkSMTPServer(), TestE2E_EmailSink(), TestE2E_OTLPSink()

### Community 82 - "Sparrow Deployment Template"
Cohesion: 0.18
Nodes (11): Sparrow Helm Chart, Sparrow ConfigMap Template, Sparrow Deployment Template, Sparrow HPA Template, Sparrow NetworkPolicy Template, Sparrow PostgreSQL Service Template, Conventional Commits Convention, Release Docker Image Job (+3 more)

### Community 83 - "parseUUID"
Cohesion: 0.36
Nodes (5): Context, WebhookService, UUID, normalizePagination(), parseUUID()

### Community 84 - "runUse"
Cohesion: 0.36
Nodes (8): findRecipe(), Context, Writer, loadRecipe(), runUse(), substituteParams(), recipe, recipeParam

### Community 85 - "RunAllMigrations"
Cohesion: 0.46
Nodes (6): main(), Context, Logger, RunAllMigrations(), RunAppMigrations(), RunRiverMigrations()

### Community 86 - "runTail"
Cohesion: 0.36
Nodes (7): deliveryItem, configFlags(), FlagSet, Context, Writer, printDelivery(), runTail()

### Community 87 - "restClient"
Cohesion: 0.46
Nodes (4): restClient, Context, Response, T

### Community 88 - "compilerOptions"
Cohesion: 0.14
Nodes (13): ./.svelte-kit/tsconfig.json, compilerOptions, allowJs, checkJs, esModuleInterop, forceConsistentCasingInFileNames, moduleResolution, resolveJsonModule (+5 more)

### Community 89 - "WebhookTargetManager"
Cohesion: 0.15
Nodes (5): before_scenario, Manages mock webhook target servers., WebhookTargetManager, Fresh target manager for each scenario., setup_scenario()

### Community 90 - "TestE2E_SourcesGitHubWebhook"
Cohesion: 0.50
Nodes (7): findEventOccurrence(), Context, Server, T, startWebhook(), TestE2E_SourcesGitHubWebhook(), TestE2E_SourcesStripeWebhook()

### Community 91 - "runListen"
Cohesion: 0.46
Nodes (7): Context, Request, ResponseWriter, Writer, mirrorForward(), printReceived(), runListen()

### Community 92 - ".Execute"
Cohesion: 0.38
Nodes (4): getBuffer(), Buffer, putBuffer(), Template

### Community 93 - "Sparrow Recipes"
Cohesion: 0.33
Nodes (5): Contributing a recipe, How params work, Included recipes, Schema (version 1), Sparrow Recipes

### Community 94 - "TestCLI_E2E"
Cohesion: 0.80
Nodes (4): buildCLI(), T, runCLI(), TestCLI_E2E()

### Community 95 - "TestE2E_SlackRecipeTransform"
Cohesion: 0.60
Nodes (4): Server, T, startBodyCaptureTarget(), TestE2E_SlackRecipeTransform()

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
Nodes (9): InsertOpts, Context, JobArgs, JobInsertResult, Logger, Tx, NewJobInserter(), BatchJobArgs (+1 more)

### Community 103 - "hooks.py"
Cohesion: 0.17
Nodes (10): after_scenario, after_suite, before_suite, SparrowAPI -- wraps the generated REST client (sparrow_client, generated from…, Gauge hooks for suite/scenario setup and teardown., Start Postgres + Sparrow containers., Stop all mock targets., setup_environment() (+2 more)

### Community 104 - "SparrowEnvironment"
Cohesion: 0.24
Nodes (4): SparrowEnvironment -- Manages Postgres and Sparrow containers via…, Manages the Sparrow test environment (Postgres + Sparrow containers)., Start Postgres + Sparrow. Returns the Sparrow HTTP URL., SparrowEnvironment

### Community 116 - "Handler"
Cohesion: 0.18
Nodes (8): GetMigrationsFS(), FS, Request, buildConfigScript(), Logger, Handler(), APIKeyAuth, Config

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
- **257 isolated node(s):** `DEFAULT_TOLERANCE_SECONDS`, `SignatureVerificationError`, `Headers`, `name`, `type` (+252 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **26 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Errorf()` connect `Errorf` to `event_filtering_test.go`, `WebhookHTTPConfig`, `NewService`, `WebhookServiceInterfaceWithTracing`, `Context`, `WebhookWorker`, `WebhookDelivery`, `NewWebhookHandler`, `setupEnv`, `Config`, `BatchJob`, `apiClient`, `ClassifyError`, `envelope`, `PrepareDeliveryRequest`, `Context`, `.Work`, `WebhookService`, `TemplateEngine`, `ValidateIP`, `parseWithArg`, `newEmailSink`, `otel.go`, `EventProcessingWorker`, `Context`, `newOTLPSink`, `run`, `Client`, `resolveConfig`, `ServiceError`, `run`, `newS3Sink`, `.CreateWebhook`, `newAPIClient`, `runPush`, `parseUUID`, `runUse`, `RunAllMigrations`, `runTail`, `restClient`, `runListen`, `.Execute`, `NewManager`, `WebhookService`, `jobInserter`?**
  _High betweenness centrality (0.312) - this node is a cross-community bridge._
- **Why does `setupEnv()` connect `setupEnv` to `newRESTClient`, `NewManager`, `startWebhookTarget`, `Mount`, `event_filtering_test.go`, `PrepareDeliveryRequest`, `webhook_service.go`, `WebhookServiceInterfaceWithTracing`, `NewService`, `Context`, `TestE2E_EmailSink`, `TestE2E_SourcesGitHubWebhook`, `TestCLI_E2E`, `TestE2E_SlackRecipeTransform`?**
  _High betweenness centrality (0.060) - this node is a cross-community bridge._
- **Why does `NewManager()` connect `NewManager` to `Errorf`, `Client`, `RepositoryInterface`, `NewService`, `setupEnv`, `EventProcessingWorker`?**
  _High betweenness centrality (0.040) - this node is a cross-community bridge._
- **Are the 146 inferred relationships involving `Errorf()` (e.g. with `.Validate()` and `.Decrypt()`) actually correct?**
  _`Errorf()` has 146 INFERRED edges - model-reasoned connections that need verification._
- **What connects `DEFAULT_TOLERANCE_SECONDS`, `SignatureVerificationError`, `Headers` to the rest of the system?**
  _257 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `event.go` be split into smaller, more focused modules?**
  _Cohesion score 0.12 - nodes in this community are weakly interconnected._
- **Should `event_filtering_test.go` be split into smaller, more focused modules?**
  _Cohesion score 0.1038961038961039 - nodes in this community are weakly interconnected._