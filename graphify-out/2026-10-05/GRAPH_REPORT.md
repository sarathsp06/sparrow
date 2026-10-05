# Graph Report - doc-repo-code-review-19116a  (2026-10-05)

## Corpus Check
- 489 files · ~457,630 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3945 nodes · 8420 edges · 268 communities (229 shown, 39 thin omitted)
- Extraction: 87% EXTRACTED · 13% INFERRED · 0% AMBIGUOUS · INFERRED: 1125 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `623a1f30`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- registerEventType
- event.go
- WebhookWorker
- event_filtering_test.go
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
- steps_surface.py
- sparrow-verify.ts
- system_events_test.go
- api-types.d.ts
- TemplateCache
- NewWebhookHandler
- api.astro
- .Work
- event_type_bundle.go
- newRootCmd
- Dual Protocol (gRPC + Connect-RPC)
- Client Libraries
- .CreateBatchJob
- Error Classification (Reference)
- Template Functions
- apiClient
- Context
- Sparrow Architecture
- event_bundle.go
- EventProcessingWorker
- ClassifyError
- envelope
- NewWithStore
- WebhookDelivery
- sparrow_verify.rs
- jobInserter
- mapError
- lib/utils.ts
- conversions.go
- newAuth
- dependencies
- NewWebhookClient
- TemplateEngine
- NewTemplateEngine
- apiConsole.svelte.ts
- testContext
- newEmailSink
- compose.ts
- store/models.go
- Wrapf
- Commands
- EventSubscription
- postSink
- setupEnv
- newOTLPSink
- devDependencies
- newPalette
- EventTypeSaveResult
- Client
- IsNotFound
- SparrowAPI
- BatchJobWorker
- recipes_test.go
- Handler
- ServiceError
- run
- newS3Sink
- sinks.mdx
- parseUUID
- payload-transformation.mdx
- startBodyRecorder
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
- clientFromCmd
- GetFunctionMap
- Sparrow Recipes
- The thought process
- runFunctions
- NewManager
- BenchmarkTransformPayload
- _Target
- Checker
- WebhookService
- Recipe
- newDrafter
- hooks.py
- SparrowEnvironment
- index.mdx
- 2. Split the `sparrow` CLI into its own module to isolate its `go install` graph
- SecurityHeaders
- 1. Payload transform engine: Go `text/template`, not embedded JavaScript
- webhooks/register/+page.svelte
- runTemplateTest
- services.ts
- release-submodules.sh
- Token
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
- Store
- Sparrow Webhook Delivery Platform
- scripts
- models_test.go
- steps_auth.py
- CI Build Job
- EventRegistration
- typescript
- PrepareDeliveryRequest
- web/package.json
- ai/ai.go
- buildVectors
- test-ui.sh
- manifest.json
- gen-og.mjs
- opencode.json
- Deliberately NOT covered by e2e
- vite-plugin-devtools-json
- RunAllMigrations
- Sparrow Web Dashboard
- @sveltejs/kit
- steps_access.py
- otel.go
- httpauth.go
- rest/access.go
- corsChain
- access.md
- parseDateFilter
- Errorf
- NewWebhookTemplateContext
- httpauth_test.go
- svelte-check
- WebhookClient
- events_bundle.go
- src/components/Footer.astro
- docs/tsconfig.json
- TestJobInserter_InsertOpts_Merge
- TestWebhookWorkerNextRetry
- Dual Webhook Signing (HMAC-SHA256 + Ed25519)
- +layout.ts
- WithConn Transaction Pattern
- SSRF Protection
- runUse
- ../../components/ThemeDiagram.astro
- content.config.ts
- proto2astro
- index.astro
- tailwindcss
- addOutputFlag
- fakeServer
- GetBuffer
- Request
- production.md
- svelte.config.js
- ParseNetworks
- deliveries/+page.svelte
- aiRouter
- separate-ui.md
- newTailCmd
- ValidateHeaders
- SparrowVerify
- WebhookClient
- webhooks/event_type_bundle_test.go
- TestModuleImportsOnlyStdlibAndItself
- dev-env.sh
- static-server.mjs
- Store
- Status
- schema_compat.go
- BatchCleanupWorker
- .VerifyEd25519
- SparrowVerify
- SparrowVerify
- SparrowVerify
- newOpenAIDrafter
- SparrowVerify
- newRESTClient
- verify-signatures.mdx
- Config
- schema-infer.ts
- TokenPurgeWorker
- Mount
- TestVectors
- runPush
- importBundle
- AlertConfig
- TestE2E_HappyPath
- newAlertConfigService
- @sveltejs/adapter-static
- alert_config.go
- TestE2E_EmailSink
- listenHandler
- posts.ts
- restClient
- TestE2E_SourcesGitHubWebhook
- What an import does
- NewDocFetcher
- access/+page.svelte
- event-type-versioning.mdx
- NewWithStore
- generator.ts
- Release Workflow
- webhook_service.go
- TestE2E_WebhookHealthAlerts
- TestListenHandlerRejectsUnsignedAndNeverForwardsThem
- subscription-pause.mdx
- RenderTemplatePreview
- resolveConfig
- runInit
- config/config_test.go
- JobInserterWithTracing
- .GetWebhookHealth
- newWorkerMetrics
- github.com/sarathsp06/sparrow
- sparrow-e2e
- Envelope Encryption at Rest (AES-256-GCM)
- PortalGateway
- TestCLI_EventTypeExportImport
- event_type_save_test.go
- Run
- SchemaFromSamples.svelte
- provider_openai.go
- newStatsCmd
- BatchJob
- sealedRouter
- parseRetryAfter
- palette
- mockJobInserter
- openapi-typescript
- @playwright/test
- ValidateHeaders
- signature_test.go
- TestE2E_SlackRecipeTransform
- typescript

## God Nodes (most connected - your core abstractions)
1. `Errorf()` - 212 edges
2. `RepositoryInterfaceWithTracing` - 87 edges
3. `setupEnv()` - 62 edges
4. `EventSubscription` - 56 edges
5. `WebhookServiceInterfaceWithTracing` - 49 edges
6. `newRESTClient()` - 46 edges
7. `testContext()` - 46 edges
8. `EventRegistration` - 42 edges
9. `NewWebhookService()` - 37 edges
10. `_base()` - 33 edges

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

## Communities (268 total, 39 thin omitted)

### Community 0 - "registerEventType"
Cohesion: 0.27
Nodes (28): deliveryItem, Context, Int32, Server, T, pollBatchJob(), pollDeliveryStatus(), pushTestEvent() (+20 more)

### Community 1 - "event.go"
Cohesion: 0.08
Nodes (42): API, Context, listEventOccurrencesImpl(), registerEventRoutes(), toBatchJobOutput(), toEventTypeChange(), toEventTypeItem(), toEventTypeOutput() (+34 more)

### Community 2 - "WebhookWorker"
Cohesion: 0.15
Nodes (17): Config, Int64Counter, JobInserter, WebhookWorker, Service, Tracer, WorkerDefaults, newTemplateErrorCounter() (+9 more)

### Community 3 - "event_filtering_test.go"
Cohesion: 0.14
Nodes (31): Context, T, UUID, TestCreateSubscription_WithInvalidLabelFilters(), TestGetSubscriptionsByEvent_CatchAllReturned(), TestGetSubscriptionsByEvent_ConsumerIsolation(), TestGetSubscriptionsByEvent_EmptyLabelFiltersMatchAll(), TestGetSubscriptionsByEvent_LabelFiltering() (+23 more)

### Community 4 - "subscription.go"
Cohesion: 0.12
Nodes (27): API, Context, Time, listSubscriptionsImpl(), registerSubscriptionRoutes(), toSubscriptionItem(), toSubscriptionOutput(), createSubscriptionBody (+19 more)

### Community 5 - "RepositoryInterfaceWithTracing"
Cohesion: 0.06
Nodes (8): Context, Duration, Span, Time, UUID, WebhookRegistration, NewRepositoryInterfaceWithTracing(), RepositoryInterfaceWithTracing

### Community 6 - "delivery.go"
Cohesion: 0.17
Nodes (19): API, Context, listDeliveriesImpl(), registerDeliveryRoutes(), toDeliveryItem(), toDeliveryOutput(), attemptItem, attemptsOutput (+11 more)

### Community 7 - "steps.py"
Cohesion: 0.08
Nodes (60): delivery_has_signature_headers(), SignatureVerifier -- Verifies HMAC-SHA256 (v1,) and Ed25519 (v1a,) signatures…, Return the exact body bytes (as text) that Sparrow signed. Standard Webhooks…, Verify HMAC-SHA256 signature (v1, prefix)., Verify Ed25519 signature (v1a, prefix)., Assert delivery has Standard Webhooks signature headers., _signed_body(), verify_ed25519_signature() (+52 more)

### Community 8 - "rest/webhook.go"
Cohesion: 0.15
Nodes (17): API, registerWebhookRoutes(), consumerOnlyInput, consumerStatsOutput, emptyOutput, listWebhooksInput, patchWebhookBody, patchWebhookInput (+9 more)

### Community 9 - "sparrow_verify.py"
Cohesion: 0.32
Nodes (11): _check_timestamp(), _decode_signatures(), Exception, Verify Sparrow webhook delivery signatures (Standard Webhooks format). Every…, Raised when a delivery signature cannot be verified., Verify the ``v1,`` (HMAC-SHA256) signature. Raises on failure. ``payload`` must…, Verify the ``v1a,`` (Ed25519) signature. Raises on failure. ``payload`` must be…, _required_headers() (+3 more)

### Community 10 - "webhooks/models.go"
Cohesion: 0.13
Nodes (14): toWebhookOutFromDomain(), DefaultWebhookHTTPConfig(), DerefBoolOr(), DerefIntOr(), Duration, Time, Value, HTTPConfigUpdate (+6 more)

### Community 11 - "NewService"
Cohesion: 0.07
Nodes (52): AEAD, Config, kek, Key, Keyring, Service, Duration, IPNet (+44 more)

### Community 12 - "WebhookServiceInterfaceWithTracing"
Cohesion: 0.08
Nodes (8): Context, Span, Time, UUID, WebhookRegistration, NewWebhookServiceInterfaceWithTracing(), WebhookResumeResult, WebhookServiceInterfaceWithTracing

### Community 13 - "Sparrow Detailed Flow Reference"
Cohesion: 0.09
Nodes (33): Sparrow Client Libraries README, Connect-RPC, DeliveryService, Envelope Encryption (AES-256-GCM), 10-Category Error Classification, EventProcessingWorker, EventService, gRPC (+25 more)

### Community 14 - "Context"
Cohesion: 0.14
Nodes (17): eventVersionOrDefault(), Context, Repository, Tx, UUID, WebhookRegistration, checkWebhookDuplicate(), Context (+9 more)

### Community 15 - "steps_surface.py"
Cohesion: 0.14
Nodes (45): alert_configs_count(), alert_configs_count_for_webhook(), also_use_consumer(), assert_delivery_page_meta(), _base(), create_alert_config_bad_webhook(), create_alert_config_scoped(), create_alert_config_wide() (+37 more)

### Community 16 - "sparrow-verify.ts"
Cohesion: 0.39
Nodes (8): decodeBase64Strict(), decodeSignatures(), DEFAULT_TOLERANCE_SECONDS, Headers, parseHeaders(), SignatureVerificationError, verifyEd25519(), verifyHmac()

### Community 17 - "system_events_test.go"
Cohesion: 0.06
Nodes (50): Context, Duration, T, UUID, WebhookWorker, newAutoDisableWorker(), TestHoldReason(), TestMaybeAutoDisable_DisablesAndEmits() (+42 more)

### Community 18 - "api-types.d.ts"
Cohesion: 0.29
Nodes (6): RFC-3339, components, $defs, operations, paths, webhooks

### Community 19 - "TemplateCache"
Cohesion: 0.24
Nodes (10): Cache, Template, hashTemplate(), NewTemplateCache(), T, TestHashTemplate(), TestTemplateCacheBasicOperations(), TestTemplateCacheConcurrency() (+2 more)

### Community 20 - "NewWebhookHandler"
Cohesion: 0.07
Nodes (49): main(), RawMessage, SparrowConfig, LoadConfig(), cronTick(), Context, Logger, Time (+41 more)

### Community 22 - ".Work"
Cohesion: 0.18
Nodes (10): Context, Duration, Job, Logger, Time, UUID, statusForFailure(), T (+2 more)

### Community 23 - "event_type_bundle.go"
Cohesion: 0.18
Nodes (21): BundleDigest(), checkStamp(), checkTemplates(), compileSchema(), generatePayload(), Context, Time, UUID (+13 more)

### Community 24 - "newRootCmd"
Cohesion: 0.20
Nodes (12): Command, Writer, newRootCmd(), newVersionCmd(), Context, Writer, main(), run() (+4 more)

### Community 27 - ".CreateBatchJob"
Cohesion: 0.17
Nodes (11): A deliberately small vocabulary, Back to the three listeners, Boring on purpose, Learn once, speak many times, One event, three listeners, Room on a small machine, Sources and further reading, Taking power away first (+3 more)

### Community 32 - "apiClient"
Cohesion: 0.15
Nodes (16): config, Context, apiClient, newAPIClient(), apiError, consumerStats, deliveryItem, eventTypeItem (+8 more)

### Community 33 - "Context"
Cohesion: 0.16
Nodes (8): generateSamplePayload(), Context, Time, UUID, WebhookService, ValidateJSONSchema(), normalizePagination(), SchemaValidationError

### Community 34 - "Sparrow Architecture"
Cohesion: 0.29
Nodes (7): Sparrow Architecture, Dual-Protocol API, Event Processing Pipeline, Health State Machine, Leaky Bucket Rate Limiting, Standard Webhooks Signing, Default Webhook Body Envelope

### Community 35 - "event_bundle.go"
Cohesion: 0.21
Nodes (17): fromBundleItems(), API, registerEventBundleRoutes(), toBundleItems(), toImportOutput(), bundleItem, bundleStampBody, eventTypeBundle (+9 more)

### Community 36 - "EventProcessingWorker"
Cohesion: 0.15
Nodes (13): Context, Job, JobInserter, Logger, WorkerDefaults, NewEventProcessingWorker(), Time, WebhookRegistration (+5 more)

### Community 37 - "ClassifyError"
Cohesion: 0.16
Nodes (23): ErrorCategory, timeoutError, classifyByMessage(), ClassifyError(), ClassifyHTTPStatus(), classifySyscallError(), isDNSError(), IsRetryableCategory() (+15 more)

### Community 38 - "envelope"
Cohesion: 0.18
Nodes (12): Context, Template, config, Context, Logger, RawMessage, sinkHandler(), templateContext() (+4 more)

### Community 39 - "NewWithStore"
Cohesion: 0.22
Nodes (8): Back to Friday, Changing the locks, Friday afternoon, How one secret gets sealed, The seal that tells the truth, Two locks, kept apart, What a stranger would find, What the envelope cannot do

### Community 40 - "WebhookDelivery"
Cohesion: 0.29
Nodes (5): Context, Repository, UUID, WebhookDelivery, WebhookDeliveryStatus

### Community 41 - "sparrow_verify.rs"
Cohesion: 0.24
Nodes (16): hex_decode(), hex_nibble(), parse_headers(), SignatureError, signed_message(), verify_ed25519(), verify_ed25519_at(), verify_hmac() (+8 more)

### Community 42 - "jobInserter"
Cohesion: 0.19
Nodes (9): Context, InsertOpts, JobArgs, JobInsertResult, Logger, Tx, NewJobInserter(), BatchJobArgs (+1 more)

### Community 43 - "mapError"
Cohesion: 0.11
Nodes (23): buildDraftRequest(), findRecipe(), API, Context, Recipe, registerAIRoutes(), registerPromptRoute(), Context (+15 more)

### Community 44 - "lib/utils.ts"
Cohesion: 0.12
Nodes (9): $lib/api-types, preview(), ERROR_CATEGORIES, getCategoryBadge(), getCategoryDisplay(), JSONSchemaMetaSchema, RFC-9457, dismissRetry() (+1 more)

### Community 45 - "conversions.go"
Cohesion: 0.13
Nodes (23): derefString(), formatOptionalTime(), getWebhookEventsMap(), Context, Time, WebhookRegistration, maskEncryptedSecret(), maskSecret() (+15 more)

### Community 46 - "newAuth"
Cohesion: 0.27
Nodes (16): NewAuth(), apiKey(), bearer(), Service, T, TB, guarded(), newAuth() (+8 more)

### Community 47 - "dependencies"
Cohesion: 0.05
Nodes (36): @astrojs/sitemap, @astrojs/starlight, @astrojs/svelte, dependencies, astro, @astrojs/sitemap, @astrojs/starlight, @astrojs/svelte (+28 more)

### Community 48 - "NewWebhookClient"
Cohesion: 0.23
Nodes (19): NewWebhookClient(), ReadBody(), BenchmarkSend(), B, T, TestClientClose(), TestDeliverySpansDoNotExportSecretURLs(), TestNewWebhookClient() (+11 more)

### Community 49 - "TemplateEngine"
Cohesion: 0.16
Nodes (11): getBuffer(), Buffer, putBuffer(), FuncMap, Template, WebhookTemplateContext, NewTemplateEngineWithCacheSize(), ExecOptions (+3 more)

### Community 50 - "NewTemplateEngine"
Cohesion: 0.27
Nodes (17): NewTemplateEngine(), BenchmarkExecuteComplex(), BenchmarkExecuteSimple(), B, T, TestExecuteComplexTemplate(), TestExecuteEmptyTemplate(), TestExecuteInvalidTemplate() (+9 more)

### Community 51 - "apiConsole.svelte.ts"
Cohesion: 0.09
Nodes (14): apiConsole, ApiLogEntry, apiLogMiddleware, entries, started, toCurl(), copy(), consumerStore (+6 more)

### Community 52 - "testContext"
Cohesion: 0.18
Nodes (28): TestCreateSubscription_CatchAllWithLabelFilters(), TestCreateSubscription_WithEmptyLabelFilters(), TestCreateSubscription_WithLabelFilters(), TestPushEvent_WithInvalidLabels(), TestPushEvent_WithNilLabels(), TestPushEvent_WithValidLabels(), TestRegisterEvent_RejectsNamesOver255Characters(), JobInserter (+20 more)

### Community 53 - "newEmailSink"
Cohesion: 0.37
Nodes (12): Conn, newEmailSink(), T, serveSMTP(), startSMTPServer(), TestEmailRender_BadTemplate(), TestEmailRender_CustomTemplatesAndHeaderInjection(), TestEmailRender_Defaults() (+4 more)

### Community 54 - "compose.ts"
Cohesion: 0.06
Nodes (41): selectPreset(), switchRecipe(), DiscordCompose, emailArray(), EmailCompose, ENVELOPE_PATHS, generateDiscordTemplate(), generateNtfyTemplate() (+33 more)

### Community 55 - "store/models.go"
Cohesion: 0.17
Nodes (17): Int64Array, Time, UUID, Value, ConsumerStats, DeliveryFilter, JSONMap, JSONStringMap (+9 more)

### Community 56 - "Wrapf"
Cohesion: 0.15
Nodes (14): anthropicClient, completion, turn, ServiceError, mapStatus(), mapTransportError(), Config, Context (+6 more)

### Community 57 - "Commands"
Cohesion: 0.08
Nodes (25): 90-second quickstart, Commands, Configuration, How it's connected, Install, Real-World Use Cases, Scenario 1: Zero-Config Local Webhook Receiver & Debugging, Scenario 2: CI/CD Pipeline Automation & Deployment Notifications (+17 more)

### Community 58 - "EventSubscription"
Cohesion: 0.21
Nodes (5): Context, Repository, Time, UUID, EventSubscription

### Community 59 - "postSink"
Cohesion: 0.40
Nodes (12): Context, Header, ResponseRecorder, T, postSink(), signHeaders(), TestSinkHandler_DownstreamFailureIs502(), TestSinkHandler_MissingHeaders() (+4 more)

### Community 60 - "setupEnv"
Cohesion: 0.11
Nodes (24): Container, testEnv, T, TestE2E_EventDeliveryStats_NoDeliveries(), T, TestAcquireDeliverySlot_BusyBucketDoesNotBurnSlots(), Context, Pool (+16 more)

### Community 61 - "newOTLPSink"
Cohesion: 0.27
Nodes (10): Context, logsURL(), newOTLPSink(), otlpPayload(), T, TestLogsURL(), TestOTLPSinkDeliver(), TestOTLPSinkDeliverDownstreamFailure() (+2 more)

### Community 62 - "devDependencies"
Cohesion: 0.15
Nodes (13): @tailwindcss/forms, @tailwindcss/typography, @tailwindcss/vite, @types/node, vite, devDependencies, svelte, @tailwindcss/forms (+5 more)

### Community 63 - "newPalette"
Cohesion: 0.16
Nodes (18): Writer, newPalette(), apiClient, Context, Writer, indentJSON(), printEventType(), runEvents() (+10 more)

### Community 64 - "EventTypeSaveResult"
Cohesion: 0.23
Nodes (13): breakingChangeError(), Context, UUID, WebhookService, metadataEqual(), planEventTypeSave(), reservedEventNameError(), schemasEqual() (+5 more)

### Community 65 - "Client"
Cohesion: 0.22
Nodes (8): Duration, Context, RawMessage, SparrowConfig, NewClient(), T, TestClientAutoCreatesEventType(), Client

### Community 66 - "IsNotFound"
Cohesion: 0.18
Nodes (9): Context, Repository, Time, UUID, Context, WebhookService, IsNotFound(), EventRecord (+1 more)

### Community 67 - "SparrowAPI"
Cohesion: 0.14
Nodes (7): SparrowAPI -- wraps the generated REST client (sparrow_client, generated from…, Poll until all deliveries reach terminal status., Poll a single delivery until terminal., Convert a generated attrs model (or None) to a plain dict., Client for Sparrow's REST API, built on the generated sparrow_client SDK., SparrowAPI, _to_dict()

### Community 68 - "BatchJobWorker"
Cohesion: 0.23
Nodes (10): Context, Job, JobInserter, Logger, UUID, WorkerDefaults, NewBatchJobWorker(), BatchJobWorker (+2 more)

### Community 69 - "recipes_test.go"
Cohesion: 0.36
Nodes (9): T, WebhookTemplateContext, sampleContext(), substituteParams(), TestPagerdutyRecipe_Severity(), TestRecipes(), TestSendGridActivationParams(), TestValidate_RequiresTransformNeedsTemplate() (+1 more)

### Community 70 - "Handler"
Cohesion: 0.19
Nodes (15): MaxBodyBytes(), computeInlineScriptHashes(), FS, Logger, Handler(), InlineScriptHashes(), inlineScriptHashesFromFS(), newHandler() (+7 more)

### Community 71 - "ServiceError"
Cohesion: 0.29
Nodes (6): "Come back later", The friend in the shower, The wrong key, Three red rows, What the red rows say now, When the fault is ours

### Community 72 - "run"
Cohesion: 0.27
Nodes (8): loadConfig(), Context, Logger, main(), run(), config, emailConfig, smtpConfig

### Community 73 - "newS3Sink"
Cohesion: 0.27
Nodes (8): Context, newS3Sink(), objectKey(), T, TestObjectKey(), TestS3Sink_PutObject(), s3Config, s3Sink

### Community 74 - "sinks.mdx"
Cohesion: 0.14
Nodes (13): Configuration, How it's connected, Install and run, Real-World Use Cases, Retry semantics, Scenario 1: Long-Term Regulatory & Financial Audit Archiving (S3 / MinIO / R2), Scenario 2: Operational Stakeholder Notifications via SMTP Email, Scenario 3: Centralized OpenTelemetry Log & Event Pipeline (OTLP) (+5 more)

### Community 75 - "parseUUID"
Cohesion: 0.16
Nodes (12): stringHeaders(), validateHeaders(), ValidateWebhookURL(), generateWebhookSecret(), Context, WebhookService, UUID, parseUUID() (+4 more)

### Community 76 - "payload-transformation.mdx"
Cohesion: 0.12
Nodes (15): 1. Send only the fields a partner needs, 2. Post a message to Slack, 3. Convert dollars to cents for a billing system, 4. Flatten a nested payload for a legacy endpoint, 5. Summarize a list of items, 6. Add a constant or computed field, 7. Provide safe defaults for optional fields, Good to know (+7 more)

### Community 77 - "startBodyRecorder"
Cohesion: 0.17
Nodes (25): bodyRecorder, requiresTransformSub, subscriptionResp, templateDelivery, Context, T, listWebhookSubs(), TestRequiresTransform_DeliveryWithoutTransformFails() (+17 more)

### Community 78 - "mockRepo"
Cohesion: 0.23
Nodes (4): Context, UUID, WebhookRegistration, mockRepo

### Community 79 - "Included recipes"
Cohesion: 0.12
Nodes (15): clickhouse, cloudevents, discord, How it's connected, Included recipes, ntfy, pagerduty, Real-World Use Cases (+7 more)

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
Nodes (24): loadEvents(), unwrap(), formatAPIError(), cancelInvite(), create(), revoke(), deliveriesLoading, error (+16 more)

### Community 84 - "access/service_test.go"
Cohesion: 0.08
Nodes (51): cacheEntry, clock, Config, countingStore, CreateInviteRequest, CreateTokenRequest, RootKey, Service (+43 more)

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

### Community 91 - "clientFromCmd"
Cohesion: 0.13
Nodes (27): accessLabel(), Command, Context, Duration, apiClient, Time, Writer, newInviteCmd() (+19 more)

### Community 92 - "GetFunctionMap"
Cohesion: 0.31
Nodes (7): GetFunctionMap(), GetTemplateFunctions(), FuncMap, T, TestTitleFunc(), toNumber(), TemplateFunc

### Community 93 - "Sparrow Recipes"
Cohesion: 0.29
Nodes (6): Contributing a recipe, How params work, Included recipes, Requiring the transform, Schema (version 1), Sparrow Recipes

### Community 94 - "The thought process"
Cohesion: 0.10
Nodes (20): Configuration, Endpoints, How it works, Light model, strict renderer, No AI configured? You still get the prompt, Recipes for known destinations, examples and docs for everything else, Refine instead of replace, Sparrow already had everything a model needs (+12 more)

### Community 95 - "runFunctions"
Cohesion: 0.23
Nodes (10): firstLine(), apiClient, Command, Context, Writer, newFunctionsCmd(), runFunctions(), T (+2 more)

### Community 96 - "NewManager"
Cohesion: 0.20
Nodes (10): Config, Context, JobInserter, Logger, Pool, Manager, Service, Tx (+2 more)

### Community 97 - "BenchmarkTransformPayload"
Cohesion: 0.40
Nodes (4): BenchmarkTransformPayload(), B, T, TestTransformPayload()

### Community 98 - "_Target"
Cohesion: 0.23
Nodes (4): CapturedDelivery, WebhookTargetServer -- Programmable mock webhook endpoints for e2e tests. Each…, Start a mock webhook target. Returns the URL., _Target

### Community 99 - "Checker"
Cohesion: 0.29
Nodes (8): Checker, HealthResponse, ReadyResponse, Context, HandlerFunc, Pool, Time, NewChecker()

### Community 100 - "WebhookService"
Cohesion: 0.19
Nodes (10): checkRequiredTransform(), Context, Time, UUID, WebhookRegistration, WebhookService, paginateSubscriptions(), ResumeResult (+2 more)

### Community 101 - "Recipe"
Cohesion: 0.53
Nodes (4): Param, Recipe, Subscription, Webhook

### Community 102 - "newDrafter"
Cohesion: 0.26
Nodes (15): fakeModel, HandlerFunc, T, newDrafter(), TestCheckRendered(), TestDraftTemplate_DocsURL(), TestDraftTemplate_GivesUpAfterMaxAttempts(), TestDraftTemplate_RefusalIsFailedPrecondition() (+7 more)

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

### Community 107 - "SecurityHeaders"
Cohesion: 0.40
Nodes (8): buildCSP(), SecurityHeaders(), T, TestBuildCSP(), TestSecurityHeadersNoHashes(), TestSecurityHeadersPassthrough(), TestSecurityHeadersPortalAllowsFraming(), TestSecurityHeadersWithHashes()

### Community 108 - "1. Payload transform engine: Go `text/template`, not embedded JavaScript"
Cohesion: 0.33
Nodes (5): 1. Payload transform engine: Go `text/template`, not embedded JavaScript, Consequences, Context, Decision, Trigger to revisit

### Community 109 - "webhooks/register/+page.svelte"
Cohesion: 0.08
Nodes (15): src(), if(), changed, editorOpen, lines, on, Recipe, RecipeParam (+7 more)

### Community 110 - "runTemplateTest"
Cohesion: 0.50
Nodes (4): Command, Writer, newTemplateCmd(), runTemplateTest()

### Community 111 - "services.ts"
Cohesion: 0.06
Nodes (44): auth, KEY_STORAGE, message, OWNED_TOKEN_STORAGE, ownedTokenId, redeeming, required, stored (+36 more)

### Community 112 - "release-submodules.sh"
Cohesion: 0.80
Nodes (4): release_leaf(), release_module(), release-submodules.sh script, tag_exists()

### Community 113 - "Token"
Cohesion: 0.16
Nodes (14): AuthError, Invite, Reason, Status, Token, Store, Duration, Time (+6 more)

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
Cohesion: 0.33
Nodes (5): Automatic disabling, Good to know, How it works, Register an alert config, Set up the email delivery channel

### Community 121 - "design.md"
Cohesion: 0.50
Nodes (3): Alternatives rejected, Context, Decisions

### Community 123 - "api.ts"
Cohesion: 0.25
Nodes (8): ensureEventType(), mintPortalToken(), newConsumer(), newEventName(), pushEvent(), registerWebhook(), uniq(), seedWebhook()

### Community 124 - "Decision"
Cohesion: 0.09
Nodes (22): 1. Reusable library: `pkg/access`, 2. Sparrow adapter: `internal/accessauth`, 3. Replace the single shared API key with revocable access tokens and one-time invites, 3. Two scopes only — no roles, 4. Stored invites instead of signed links, 5. No cascade revocation, 6. Tokens independent of master key, 7. 503, not 401, when the database is down (+14 more)

### Community 125 - "Store"
Cohesion: 0.17
Nodes (14): NullString, NullTime, execer, scanner, Store, Context, Time, insertToken() (+6 more)

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
Cohesion: 0.22
Nodes (9): Context, UUID, Context, Repository, UUID, EventRegistration, EventRegistrationVersion, fakeEventTypeRepo (+1 more)

### Community 132 - "typescript"
Cohesion: 0.29
Nodes (6): Same care, every time, Tasting before serving, The card in the drawer, The first request, Then everyone else asked, When the recipe goes wrong

### Community 133 - "PrepareDeliveryRequest"
Cohesion: 0.13
Nodes (24): DeliveryRequest, WebhookEnvelope, BuildEnvelopePayload(), BuildRequest(), generateEd25519Signature(), generateHMACSignature(), Context, Duration (+16 more)

### Community 134 - "web/package.json"
Cohesion: 0.20
Nodes (9): openapi-fetch, dependencies, openapi-fetch, svelte-jsoneditor, svelte-jsoneditor, name, private, type (+1 more)

### Community 135 - "ai/ai.go"
Cohesion: 0.28
Nodes (13): completer, Config, DocFetcher, draft, Drafter, HelperFunc, PromptBuilder, Provider (+5 more)

### Community 136 - "buildVectors"
Cohesion: 0.60
Nodes (5): vector, vectorFile, buildVectors(), T, TestSignatureVectors()

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

### Community 143 - "RunAllMigrations"
Cohesion: 0.21
Nodes (9): cryptoSealer, main(), GetMigrationsFS(), FS, Context, Logger, RunAllMigrations(), RunAppMigrations() (+1 more)

### Community 144 - "Sparrow Web Dashboard"
Cohesion: 0.22
Nodes (8): Build, Configuration, Embedding in the Go Binary, Local Development, Prerequisites, Sparrow Web Dashboard, Standalone Deployment, Tech Stack

### Community 146 - "steps_access.py"
Cohesion: 0.27
Nodes (19): _as_token(), cancel_invite(), create_consumer_token(), create_invite(), _create_token(), get_with_token(), get_with_token_rejected(), _invites() (+11 more)

### Community 147 - "otel.go"
Cohesion: 0.10
Nodes (35): ExportTraceServiceRequest, ExportTraceServiceResponse, Int64UpDownCounter, DefaultConfig(), GetMeter(), GetTracer(), Context, Int64Counter (+27 more)

### Community 148 - "httpauth.go"
Cohesion: 0.16
Nodes (18): Principal, Authenticator, ctxKey, ErrorBody, RedeemRequest, RedeemResponse, Verifier, authMessage() (+10 more)

### Community 149 - "rest/access.go"
Cohesion: 0.13
Nodes (26): fragmentEscape(), API, Context, Duration, Service, Time, mapAccessError(), portalLinkPath() (+18 more)

### Community 150 - "corsChain"
Cohesion: 0.28
Nodes (14): CORS(), NormalizeOrigins(), corsChain(), get(), ResponseRecorder, T, TB, preflight() (+6 more)

### Community 151 - "access.md"
Cohesion: 0.13
Nodes (14): API endpoints, CI and machine tokens, Consumer (portal) access, First run, From the API, From the CLI, From the web UI, Invite a teammate (+6 more)

### Community 153 - "parseDateFilter"
Cohesion: 0.33
Nodes (7): Time, parseDateFilter(), parseLabelFilter(), T, TestParseDateFilter(), TestParseDateFilter_AcceptsRFC3339(), TestParseLabelFilter()

### Community 154 - "Errorf"
Cohesion: 0.16
Nodes (16): NetworkPolicy, embeddedIPv4(), IPNet, isMetadataIP(), mustCIDRs(), ValidateIP(), IP, Errorf() (+8 more)

### Community 155 - "NewWebhookTemplateContext"
Cohesion: 0.36
Nodes (10): T, TestTransformPayloadWith_CachesStrictAndLenientSeparately(), TestTransformPayloadWith_StrictMissingKeys(), NewWebhookTemplateContext(), T, loadSendgridTemplate(), TestSendgridRecipe_CustomEvent(), TestSendgridRecipe_DeliveryFailed() (+2 more)

### Community 156 - "httpauth_test.go"
Cohesion: 0.28
Nodes (13): downStore, Context, ResponseRecorder, Service, Store, T, newSvc(), serve() (+5 more)

### Community 158 - "WebhookClient"
Cohesion: 0.18
Nodes (9): redactSpanURL, WebhookClient, Config, Context, Duration, Response, WebhookTemplateContext, RedactURL() (+1 more)

### Community 159 - "events_bundle.go"
Cohesion: 0.40
Nodes (4): A promise you can't check is a hope, Keep it where the events live, Own only what you can carry, Some things should stay home

### Community 164 - "Dual Webhook Signing (HMAC-SHA256 + Ed25519)"
Cohesion: 0.67
Nodes (3): Dual Webhook Signing (HMAC-SHA256 + Ed25519), Timestamp Replay Protection, Standard Webhooks Format

### Community 168 - "runUse"
Cohesion: 0.20
Nodes (14): apiClient, Command, Context, recipe, Writer, loadRecipe(), newUseCmd(), resolveRecipe() (+6 more)

### Community 174 - "addOutputFlag"
Cohesion: 0.18
Nodes (22): Reader, addOutputFlag(), outputFmt(), apiClient, Command, Context, Writer, importOutcomeError() (+14 more)

### Community 175 - "fakeServer"
Cohesion: 0.13
Nodes (36): fakeServer(), Server, T, runCLI(), TestAccessCommandsSurfaceServerErrors(), TestInviteBuildsLinkOnUIURL(), TestInviteDefaultsToServerURLAndJSON(), TestInvitesListAndCancel() (+28 more)

### Community 176 - "GetBuffer"
Cohesion: 0.28
Nodes (11): GetBuffer(), GetHeaderMap(), Buffer, PutBuffer(), PutHeaderMap(), BenchmarkBufferPool(), BenchmarkHeaderMapPool(), B (+3 more)

### Community 177 - "Request"
Cohesion: 0.24
Nodes (6): Request, Result, Context, mustJSON(), ResolveDocs(), stubDrafter

### Community 178 - "production.md"
Cohesion: 0.15
Nodes (12): Access and authentication, Deployment, Example Ingress (internal), Kubernetes manifests, Network exposure, NetworkPolicy, Production checklist, Required configuration (+4 more)

### Community 182 - "ParseNetworks"
Cohesion: 0.29
Nodes (9): ParseNetworks(), T, TestDialControlBlocksMetadataWithAllowPrivate(), TestNetworkPolicy(), TestNetworkPolicyHostnames(), TestParseNetworks(), TestValidateIP(), T (+1 more)

### Community 183 - "deliveries/+page.svelte"
Cohesion: 0.25
Nodes (14): applyFilters(), cancelRetryBatch(), clearFilters(), executeRetry(), fetchDeliveries(), handlePageChange(), onBatchDone(), prepareRetryBatch() (+6 more)

### Community 184 - "aiRouter"
Cohesion: 0.53
Nodes (9): aiRouter(), do(), ResponseRecorder, T, TestCapabilities_ReflectsDrafter(), TestDraftTemplate_DisabledIs503(), TestDraftTemplate_GroundsRequestAndReturnsDraft(), TestDraftTemplate_NotFound() (+1 more)

### Community 185 - "separate-ui.md"
Cohesion: 0.20
Nodes (9): 1. Build the UI, 2. Point the UI at the server: `config.js`, 3. Serve it as a single-page app, 4. Configure the server, Authentication, Consumer portal, Local development, Troubleshooting (+1 more)

### Community 186 - "newTailCmd"
Cohesion: 0.39
Nodes (8): deliveryItem, apiClient, Command, Context, Writer, newTailCmd(), printDelivery(), runTail()

### Community 188 - "SparrowVerify"
Cohesion: 0.38
Nodes (5): ByteArray, Exception, ParsedHeaders, SignatureVerificationException, SparrowVerify

### Community 189 - "WebhookClient"
Cohesion: 0.31
Nodes (11): accessRouter(), Context, ResponseRecorder, Store, T, post(), TestAccessRejectsUnsafeConsumerNames(), TestAccessStoreErrorsAreNotLeaked() (+3 more)

### Community 190 - "webhooks/event_type_bundle_test.go"
Cohesion: 0.31
Nodes (9): T, TestBundleDigest_IgnoresFormattingAndKeyOrder(), TestBundleItemJSONShape(), TestCheckStamp(), TestCheckTemplates(), TestValidateBundle(), T, js() (+1 more)

### Community 197 - "schema_compat.go"
Cohesion: 0.29
Nodes (9): ClassifySchemaChange(), firstComposition(), joinPath(), requiredSet(), sortedKeys(), typeAllows(), typeChange(), typesOf() (+1 more)

### Community 198 - "BatchCleanupWorker"
Cohesion: 0.19
Nodes (9): Context, InsertOpts, Job, Logger, WorkerDefaults, NewBatchCleanupWorker(), BatchCleanupArgs, BatchCleanupWorker (+1 more)

### Community 199 - ".VerifyEd25519"
Cohesion: 0.38
Nodes (8): Duration, Header, Time, parseHeaders(), signedMessage(), VerifyEd25519(), VerifyHMAC(), Verifier

### Community 201 - "SparrowVerify"
Cohesion: 0.35
Nodes (10): SparrowVerify, decode_public_key(), decode_secret(), decode_signatures(), fetch_header(), parse_headers(), safe_hex_decode(), validate_timestamp() (+2 more)

### Community 202 - "SparrowVerify"
Cohesion: 0.35
Nodes (3): SparrowVerify, SparrowVerify::SignatureVerificationError, StandardError

### Community 203 - "newOpenAIDrafter"
Cohesion: 0.40
Nodes (8): fakeOpenAI, HandlerFunc, T, newOpenAIDrafter(), TestNew_ProviderValidation(), TestOpenAIProvider_AuthAndErrors(), TestOpenAIProvider_DraftsAndRepairs(), TestParseDraft_ToleratesFencesAndPreamble()

### Community 204 - "SparrowVerify"
Cohesion: 0.22
Nodes (3): SignatureVerificationException, SparrowVerify, RuntimeException

### Community 205 - "newRESTClient"
Cohesion: 0.38
Nodes (11): eventTypeResp, eventTypeVersionResp, T, TestEventTypeVersions_AutoRegisterThenFillIn(), TestEventTypeVersions_BreakingChangeNeedsOptIn(), TestEventTypeVersions_ConcurrentSchemaChanges(), TestEventTypeVersions_Lifecycle(), TestEventTypeVersions_NoDelete() (+3 more)

### Community 206 - "verify-signatures.mdx"
Cohesion: 0.50
Nodes (3): Get the helper and verify, Testing your receiver, What gets signed

### Community 207 - "Config"
Cohesion: 0.22
Nodes (6): Config, DefaultConfig(), Duration, IPNet, T, TestDefaultConfig()

### Community 208 - "schema-infer.ts"
Cohesion: 0.21
Nodes (15): analyzeSamples(), collect(), DATE, detectStringFormat(), emit(), FieldCoverage, inferSchema(), jsonToJsonSchema() (+7 more)

### Community 209 - "TokenPurgeWorker"
Cohesion: 0.18
Nodes (11): Duration, Context, Duration, InsertOpts, Job, Logger, WorkerDefaults, NewTokenPurgeWorker() (+3 more)

### Community 210 - "Mount"
Cohesion: 0.22
Nodes (6): main(), API, Mount(), T, TestOpenAPISpecMatchesCommitted(), Router

### Community 212 - "runPush"
Cohesion: 0.11
Nodes (12): parseJSONArg(), T, TestKVFlag(), TestParseJSONArg(), apiClient, Command, Context, Writer (+4 more)

### Community 213 - "importBundle"
Cohesion: 0.45
Nodes (10): bundle, importResult, exportBundle(), Context, T, importBundle(), TestEventTypeBundle_AllOrNothingAndBreakingChanges(), TestEventTypeBundle_InvalidInput() (+2 more)

### Community 214 - "AlertConfig"
Cohesion: 0.32
Nodes (5): Context, Repository, Time, UUID, AlertConfig

### Community 215 - "TestE2E_HappyPath"
Cohesion: 0.31
Nodes (10): capturedWebhook, Context, Duration, Header, Server, T, pollDeliverySuccess(), startWebhookTarget() (+2 more)

### Community 216 - "newAlertConfigService"
Cohesion: 0.35
Nodes (13): Context, Status, T, mockRepo, newAlertConfigService(), requireStatus(), TestCreateAlertConfig_ConsumerWide(), TestCreateAlertConfig_RejectsUnsupportedEventType() (+5 more)

### Community 218 - "alert_config.go"
Cohesion: 0.27
Nodes (10): API, registerAlertConfigRoutes(), toAlertConfigItem(), alertConfigIDInput, alertConfigItem, alertConfigOutput, createAlertConfigBody, createAlertConfigInput (+2 more)

### Community 219 - "TestE2E_EmailSink"
Cohesion: 0.50
Nodes (8): sinkSMTPMessage, freePort(), Context, T, startSinksBinary(), startSinkSMTPServer(), TestE2E_EmailSink(), TestE2E_OTLPSink()

### Community 220 - "listenHandler"
Cohesion: 0.23
Nodes (14): apiClient, Command, Context, ResponseWriter, Writer, T, signedRequest(), TestListenHandlerRejectsUnsignedAndNeverForwardsThem() (+6 more)

### Community 221 - "posts.ts"
Cohesion: 0.14
Nodes (22): tags, allTags(), base, formatDate(), getPosts(), loaders, Post, PostFrontmatter (+14 more)

### Community 222 - "restClient"
Cohesion: 0.46
Nodes (4): restClient, Context, Response, T

### Community 223 - "TestE2E_SourcesGitHubWebhook"
Cohesion: 0.50
Nodes (7): findEventOccurrence(), Context, Server, T, startWebhook(), TestE2E_SourcesGitHubWebhook(), TestE2E_SourcesStripeWebhook()

### Community 224 - "What an import does"
Cohesion: 0.29
Nodes (6): API, Subscriptions whose template fails, The bundle file, The preview, What an import does, When nothing is written

### Community 225 - "NewDocFetcher"
Cohesion: 0.36
Nodes (6): collapseSpace(), htmlToText(), looksLikeHTML(), NewDocFetcher(), T, TestDocFetcher_ExtractsTextAndRespectsPolicy()

### Community 226 - "access/+page.svelte"
Cohesion: 0.15
Nodes (10): consumer, copied, createError, creating, inviteTTL, name, string, tokenTTL (+2 more)

### Community 227 - "event-type-versioning.mdx"
Cohesion: 0.29
Nodes (6): Breaking changes, Reading the history, Reserved names, Retiring an event type, Unregistered event names, What creates a new version

### Community 228 - "NewWithStore"
Cohesion: 0.19
Nodes (18): SecretSealer, Duration, Service, Store, InviteTTL(), New(), NewWithStore(), Realm() (+10 more)

### Community 229 - "generator.ts"
Cohesion: 0.08
Nodes (25): entryToSimpleMarkdown(), htmlToMarkdownPipeline, minify, minifyDefaults, selectors, collator, generateLlmsTxt(), starlightLlmsTxt() (+17 more)

### Community 231 - "webhook_service.go"
Cohesion: 0.14
Nodes (14): Context, IPNet, WithAllowedNetworks(), WithAllowPrivateNetworks(), WithAutoRegisterEvents(), deliveryRouteService, eventOnlyService, eventRouteService (+6 more)

### Community 232 - "TestE2E_WebhookHealthAlerts"
Cohesion: 0.60
Nodes (5): assertHasRecipient(), Context, T, pollSystemEvent(), TestE2E_WebhookHealthAlerts()

### Community 234 - "subscription-pause.mdx"
Cohesion: 0.33
Nodes (5): Pausing from an import, Resuming, Seeing what was held, What a pause does, Where pause fits in a delivery

### Community 235 - "RenderTemplatePreview"
Cohesion: 0.50
Nodes (3): T, TestRenderTemplatePreview_StrictFlag(), RenderTemplatePreview()

### Community 236 - "resolveConfig"
Cohesion: 0.36
Nodes (9): configPath(), resolveConfig(), saveConfig(), T, TestResolveConfigDefaults(), TestResolveConfigPrecedence(), TestSaveConfigPermissions(), writeConfigFile() (+1 more)

### Community 237 - "runInit"
Cohesion: 0.31
Nodes (9): File, Command, config, Context, Writer, isTerminal(), newInitCmd(), probeServer() (+1 more)

### Community 238 - "config/config_test.go"
Cohesion: 0.23
Nodes (12): Config, T, TestAIConfig(), TestUIInjectKeyIsRetired(), TestValidate(), TestWarnings(), validConfig(), T (+4 more)

### Community 239 - "JobInserterWithTracing"
Cohesion: 0.31
Nodes (7): Context, JobArgs, JobInserter, JobInsertResult, Span, NewJobInserterWithTracing(), JobInserterWithTracing

### Community 240 - ".GetWebhookHealth"
Cohesion: 0.21
Nodes (6): Context, Time, WebhookService, ConsumerStatsData, HealthSummaryData, WebhookHealthData

### Community 241 - "newWorkerMetrics"
Cohesion: 0.43
Nodes (5): Float64Histogram, Context, Int64Counter, newWorkerMetrics(), workerMetrics

### Community 242 - "github.com/sarathsp06/sparrow"
Cohesion: 0.53
Nodes (6): github.com/sarathsp06/sparrow, github.com/sarathsp06/sparrow/pkg/access, github.com/sarathsp06/sparrow/pkg/signature, github.com/sarathsp06/sparrow/pkg/template, github.com/sarathsp06/sparrow/satellites/recipes, github.com/sarathsp06/sparrow/satellites/sparrow

### Community 248 - "PortalGateway"
Cohesion: 0.23
Nodes (11): ResponseWriter, writeJSONError(), cleanPortalPath(), Context, ResponseWriter, PortalAuthorized(), PortalGateway(), portalTarget() (+3 more)

### Community 249 - "TestCLI_EventTypeExportImport"
Cohesion: 0.80
Nodes (5): buildCLI(), T, runCLI(), TestCLI_E2E(), TestCLI_EventTypeExportImport()

### Community 250 - "event_type_save_test.go"
Cohesion: 0.30
Nodes (14): IsReservedEventName(), assertStatus(), Status, T, newFakeEventTypeRepo(), orderSchema(), TestIsReservedEventName(), TestPlanEventTypeSave() (+6 more)

### Community 251 - "Run"
Cohesion: 0.29
Nodes (8): T, TestConformance(), Store, T, ids(), mintFn(), must(), Run()

### Community 252 - "SchemaFromSamples.svelte"
Cohesion: 0.24
Nodes (6): coverage, generate(), picked, readSample(), setSelection(), toggle()

### Community 253 - "provider_openai.go"
Cohesion: 0.32
Nodes (7): oaMessage, oaRequest, oaResponse, openAIClient, Config, newOpenAIClient(), truncate()

### Community 254 - "newStatsCmd"
Cohesion: 0.38
Nodes (6): apiClient, Command, Context, Writer, newStatsCmd(), runStats()

### Community 255 - "BatchJob"
Cohesion: 0.16
Nodes (11): Context, Repository, UUID, RawMessage, Context, WebhookService, BatchJob, BatchJobData (+3 more)

### Community 256 - "sealedRouter"
Cohesion: 0.50
Nodes (8): Service, T, mintToken(), sealedRouter(), TestConsumerTokenComesWithAPortalLink(), TestExternalIDReturnsTheValidConsumerToken(), TestExternalIDRules(), mintedToken

### Community 257 - "parseRetryAfter"
Cohesion: 0.38
Nodes (9): T, TestDefaultAndMaxRetryAfterConstants(), TestIsSuccessStatusCode(), TestParseRetryAfter(), TestParseRetryAfter_HTTPDate(), TestParseRetryAfter_HTTPDate_FarFuture(), TestParseRetryAfter_HTTPDate_Past(), isSuccessStatusCode() (+1 more)

### Community 261 - "mockJobInserter"
Cohesion: 0.47
Nodes (4): JobArgs, JobInsertResult, Mock, mockJobInserter

### Community 270 - "ValidateHeaders"
Cohesion: 0.31
Nodes (7): isReservedHeader(), T, TestBuildRequestHeaderPrecedence(), TestValidateHeaders(), ValidateHeaders(), validHeaderName(), validHeaderValue()

### Community 271 - "signature_test.go"
Cohesion: 0.50
Nodes (8): T, signHMAC(), TestMissingHeaders(), TestTimestampTolerance(), TestVerifyEd25519(), TestVerifyHMAC(), TestVerifyHMACMultiSignatureHeader(), TestVerifyHMACRawSecret()

### Community 274 - "TestE2E_SlackRecipeTransform"
Cohesion: 0.60
Nodes (4): Server, T, startBodyCaptureTarget(), TestE2E_SlackRecipeTransform()

## Knowledge Gaps
- **576 isolated node(s):** `DEFAULT_TOLERANCE_SECONDS`, `SignatureVerificationError`, `Headers`, `name`, `type` (+571 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **39 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Errorf()` connect `Errorf` to `event_filtering_test.go`, `PrepareDeliveryRequest`, `ai/ai.go`, `webhooks/models.go`, `NewService`, `ValidateHeaders`, `RunAllMigrations`, `Context`, `system_events_test.go`, `otel.go`, `NewWebhookHandler`, `.Work`, `event_type_bundle.go`, `parseDateFilter`, `apiClient`, `Context`, `EventProcessingWorker`, `ClassifyError`, `envelope`, `runUse`, `WebhookDelivery`, `jobInserter`, `addOutputFlag`, `TemplateEngine`, `newEmailSink`, `ParseNetworks`, `Wrapf`, `newTailCmd`, `EventSubscription`, `setupEnv`, `newOTLPSink`, `newPalette`, `EventTypeSaveResult`, `Client`, `IsNotFound`, `BatchJobWorker`, `.VerifyEd25519`, `run`, `newS3Sink`, `parseUUID`, `access/service_test.go`, `runPush`, `clientFromCmd`, `GetFunctionMap`, `listenHandler`, `restClient`, `runFunctions`, `NewManager`, `NewDocFetcher`, `NewWithStore`, `WebhookService`, `Recipe`, `TestListenHandlerRejectsUnsignedAndNeverForwardsThem`, `resolveConfig`, `runInit`, `runTemplateTest`, `.GetWebhookHealth`, `Store`, `BatchJob`?**
  _High betweenness centrality (0.301) - this node is a cross-community bridge._
- **Why does `setupEnv()` connect `setupEnv` to `registerEventType`, `NewManager`, `PrepareDeliveryRequest`, `webhook_service.go`, `TestE2E_WebhookHealthAlerts`, `NewService`, `WebhookServiceInterfaceWithTracing`, `newRESTClient`, `startBodyRecorder`, `TestE2E_SlackRecipeTransform`, `Mount`, `access/service_test.go`, `importBundle`, `testContext`, `TestE2E_HappyPath`, `TestCLI_EventTypeExportImport`, `TestE2E_EmailSink`, `TestE2E_SourcesGitHubWebhook`?**
  _High betweenness centrality (0.051) - this node is a cross-community bridge._
- **Why does `NewManager()` connect `NewManager` to `Client`, `WebhookWorker`, `EventProcessingWorker`, `BatchJobWorker`, `BatchCleanupWorker`, `RetentionWorker`, `Errorf`, `setupEnv`?**
  _High betweenness centrality (0.037) - this node is a cross-community bridge._
- **Are the 209 inferred relationships involving `Errorf()` (e.g. with `.Authenticate()` and `.GetOrCreateToken()`) actually correct?**
  _`Errorf()` has 209 INFERRED edges - model-reasoned connections that need verification._
- **Are the 57 inferred relationships involving `setupEnv()` (e.g. with `TestCLI_E2E()` and `TestCLI_EventTypeExportImport()`) actually correct?**
  _`setupEnv()` has 57 INFERRED edges - model-reasoned connections that need verification._
- **What connects `DEFAULT_TOLERANCE_SECONDS`, `SignatureVerificationError`, `Headers` to the rest of the system?**
  _576 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `event.go` be split into smaller, more focused modules?**
  _Cohesion score 0.07928118393234672 - nodes in this community are weakly interconnected._