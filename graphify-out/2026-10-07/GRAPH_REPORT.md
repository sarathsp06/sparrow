# Graph Report - workbench-payload-template-input-7682ff  (2026-10-07)

## Corpus Check
- 492 files · ~469,877 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3990 nodes · 8511 edges · 264 communities (224 shown, 40 thin omitted)
- Extraction: 87% EXTRACTED · 13% INFERRED · 0% AMBIGUOUS · INFERRED: 1136 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `0dc66833`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- registerEventType
- event.go
- RepositoryInterface
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
- steps_surface.py
- sparrow-verify.ts
- system_events_test.go
- api-types.d.ts
- TemplateCache
- NewWebhookHandler
- api.astro
- WebhookWorker
- event_type_bundle.go
- addOutputFlag
- Dual Protocol (gRPC + Connect-RPC)
- Client Libraries
- template-transformations.md
- Error Classification (Reference)
- Template Functions
- apiClient
- Context
- Sparrow Architecture
- Token
- EventProcessingWorker
- ClassifyError
- envelope
- envelope-encryption.md
- Context
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
- event_bundle.go
- newEmailSink
- template-fields.ts
- store/models.go
- Store
- Commands
- Context
- postSink
- setupEnv
- newOTLPSink
- devDependencies
- newPalette
- Mount
- Client
- IsNotFound
- SparrowAPI
- BatchJobWorker
- recipes_test.go
- Handler
- delivery-failures.md
- run
- newS3Sink
- sinks.mdx
- Errorf
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
- Config
- 1. Payload transform engine: Go `text/template`, not embedded JavaScript
- ai/ai.go
- runTemplateTest
- services.ts
- release-submodules.sh
- webhook_service.go
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
- Wrapf
- Sparrow Webhook Delivery Platform
- scripts
- models_test.go
- steps_auth.py
- CI Build Job
- EventRegistration
- recipes-as-config.md
- PrepareDeliveryRequest
- web/package.json
- webhooks/register/+page.svelte
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
- Setup
- httpauth.go
- rest/access.go
- corsChain
- access.md
- parseDateFilter
- NetworkPolicy
- NewWebhookTemplateContext
- httpauth_test.go
- svelte-check
- WebhookClient
- why-sparrow.md
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
- pages/index.astro
- tailwindcss
- EventTypeSaveResult
- fakeServer
- GetBuffer
- schema_compat.go
- production.md
- svelte.config.js
- pushSystemEvent
- deliveries/+page.svelte
- aiRouter
- separate-ui.md
- newTailCmd
- @sveltejs/vite-plugin-svelte
- SparrowVerify
- PortalGateway
- webhooks/event_type_bundle_test.go
- TestModuleImportsOnlyStdlibAndItself
- dev-env.sh
- static-server.mjs
- Store
- Status
- Context
- BatchCleanupWorker
- .VerifyEd25519
- SparrowVerify
- SparrowVerify
- SparrowVerify
- Request
- SparrowVerify
- newRESTClient
- verify-signatures.mdx
- .PushEvent
- schema-infer.ts
- TokenPurgeWorker
- fakeSystemEventRepo
- TestVectors
- runPush
- importBundle
- Config
- TestE2E_HappyPath
- config/config_test.go
- Context
- alert_config.go
- TestE2E_EmailSink
- listenHandler
- posts.ts
- restClient
- TestE2E_SourcesGitHubWebhook
- What an import does
- for-decision-makers.mdx
- AutoDisableResult
- event-type-versioning.mdx
- resolveConfig
- generator.ts
- Release Workflow
- newOpenAIDrafter
- TestE2E_WebhookHealthAlerts
- runInit
- subscription-pause.mdx
- RenderTemplatePreview
- newStatsCmd
- newRootCmd
- WebhookService
- provider_openai.go
- install.sh
- newWorkerMetrics
- github.com/sarathsp06/sparrow
- sparrow-e2e
- Envelope Encryption at Rest (AES-256-GCM)
- NewDocFetcher
- newAutoDisableWorker
- svelte
- WebhookService
- SchemaFromSamples.svelte
- TestCLI_EventTypeExportImport
- timeoutError
- BatchJob
- parseRetryAfter
- palette
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

## Communities (264 total, 40 thin omitted)

### Community 0 - "registerEventType"
Cohesion: 0.27
Nodes (28): deliveryItem, Context, Int32, Server, T, pollBatchJob(), pollDeliveryStatus(), pushTestEvent() (+20 more)

### Community 1 - "event.go"
Cohesion: 0.08
Nodes (42): API, Context, listEventOccurrencesImpl(), registerEventRoutes(), toBatchJobOutput(), toEventTypeChange(), toEventTypeItem(), toEventTypeOutput() (+34 more)

### Community 2 - "RepositoryInterface"
Cohesion: 0.25
Nodes (4): NewRepositoryInterfaceWithTracing(), RateLimitRepository, RepositoryInterface, Transactor

### Community 3 - "testContext"
Cohesion: 0.07
Nodes (84): Context, T, UUID, TestCreateSubscription_CatchAllWithLabelFilters(), TestCreateSubscription_WithEmptyLabelFilters(), TestCreateSubscription_WithInvalidLabelFilters(), TestCreateSubscription_WithLabelFilters(), TestGetSubscriptionsByEvent_CatchAllReturned() (+76 more)

### Community 4 - "subscription.go"
Cohesion: 0.12
Nodes (27): API, Context, Time, listSubscriptionsImpl(), registerSubscriptionRoutes(), toSubscriptionItem(), toSubscriptionOutput(), createSubscriptionBody (+19 more)

### Community 5 - "RepositoryInterfaceWithTracing"
Cohesion: 0.08
Nodes (12): Context, Duration, Span, Time, UUID, WebhookRegistration, ConsumerStats, EventReportFilter (+4 more)

### Community 6 - "delivery.go"
Cohesion: 0.17
Nodes (19): API, Context, listDeliveriesImpl(), registerDeliveryRoutes(), toDeliveryItem(), toDeliveryOutput(), attemptItem, attemptsOutput (+11 more)

### Community 7 - "steps.py"
Cohesion: 0.08
Nodes (60): delivery_has_signature_headers(), SignatureVerifier -- Verifies HMAC-SHA256 (v1,) and Ed25519 (v1a,) signatures…, Return the exact body bytes (as text) that Sparrow signed. Standard Webhooks…, Verify HMAC-SHA256 signature (v1, prefix)., Verify Ed25519 signature (v1a, prefix)., Assert delivery has Standard Webhooks signature headers., _signed_body(), verify_ed25519_signature() (+52 more)

### Community 8 - "rest/webhook.go"
Cohesion: 0.18
Nodes (13): consumerOnlyInput, consumerStatsOutput, emptyOutput, listWebhooksInput, patchWebhookBody, patchWebhookInput, registerWebhookBody, registerWebhookInput (+5 more)

### Community 9 - "sparrow_verify.py"
Cohesion: 0.32
Nodes (11): _check_timestamp(), _decode_signatures(), Exception, Verify Sparrow webhook delivery signatures (Standard Webhooks format). Every…, Raised when a delivery signature cannot be verified., Verify the ``v1,`` (HMAC-SHA256) signature. Raises on failure. ``payload`` must…, Verify the ``v1a,`` (Ed25519) signature. Raises on failure. ``payload`` must be…, _required_headers() (+3 more)

### Community 10 - "webhooks/models.go"
Cohesion: 0.10
Nodes (18): toWebhookOutFromDomain(), DefaultWebhookHTTPConfig(), DerefBoolOr(), DerefIntOr(), Duration, Time, Value, T (+10 more)

### Community 11 - "NewService"
Cohesion: 0.07
Nodes (52): AEAD, Config, kek, Key, Keyring, Service, Duration, IPNet (+44 more)

### Community 12 - "WebhookServiceInterfaceWithTracing"
Cohesion: 0.06
Nodes (14): Context, Time, WebhookService, Context, Span, Time, UUID, WebhookRegistration (+6 more)

### Community 13 - "Sparrow Detailed Flow Reference"
Cohesion: 0.09
Nodes (33): Sparrow Client Libraries README, Connect-RPC, DeliveryService, Envelope Encryption (AES-256-GCM), 10-Category Error Classification, EventProcessingWorker, EventService, gRPC (+25 more)

### Community 14 - "Context"
Cohesion: 0.13
Nodes (19): T, TestE2E_EventDeliveryStats_NoDeliveries(), Context, Repository, Tx, UUID, WebhookRegistration, NewRepository() (+11 more)

### Community 15 - "steps_surface.py"
Cohesion: 0.14
Nodes (45): alert_configs_count(), alert_configs_count_for_webhook(), also_use_consumer(), assert_delivery_page_meta(), _base(), create_alert_config_bad_webhook(), create_alert_config_scoped(), create_alert_config_wide() (+37 more)

### Community 16 - "sparrow-verify.ts"
Cohesion: 0.39
Nodes (8): decodeBase64Strict(), decodeSignatures(), DEFAULT_TOLERANCE_SECONDS, Headers, parseHeaders(), SignatureVerificationError, verifyEd25519(), verifyHmac()

### Community 17 - "system_events_test.go"
Cohesion: 0.21
Nodes (18): T, WebhookWorker, newTestWorker(), TestEmitDeliveryFailedEvent_EmitsOnRecipientLookupError(), TestEmitDeliveryFailedEvent_EmitsWhenRecipientsOptedIn(), TestEmitDeliveryFailedEvent_EmitsWithZeroRecipients(), TestEmitDeliveryFailedEvent_SkipsSparrowConsumer(), TestEmitHealthChangedEvent_EmitsOnRealTransition() (+10 more)

### Community 18 - "api-types.d.ts"
Cohesion: 0.29
Nodes (6): RFC-3339, components, $defs, operations, paths, webhooks

### Community 19 - "TemplateCache"
Cohesion: 0.24
Nodes (10): Cache, Template, hashTemplate(), NewTemplateCache(), T, TestHashTemplate(), TestTemplateCacheBasicOperations(), TestTemplateCacheConcurrency() (+2 more)

### Community 20 - "NewWebhookHandler"
Cohesion: 0.07
Nodes (49): main(), RawMessage, SparrowConfig, LoadConfig(), cronTick(), Context, Logger, Time (+41 more)

### Community 22 - "WebhookWorker"
Cohesion: 0.12
Nodes (22): Config, Context, Duration, Int64Counter, Job, JobInserter, Logger, WebhookWorker (+14 more)

### Community 23 - "event_type_bundle.go"
Cohesion: 0.18
Nodes (21): BundleDigest(), checkStamp(), checkTemplates(), compileSchema(), generatePayload(), Context, Time, UUID (+13 more)

### Community 24 - "addOutputFlag"
Cohesion: 0.18
Nodes (22): Reader, addOutputFlag(), outputFmt(), apiClient, Command, Context, Writer, importOutcomeError() (+14 more)

### Community 27 - "template-transformations.md"
Cohesion: 0.17
Nodes (11): A deliberately small vocabulary, Back to the three listeners, Boring on purpose, Learn once, speak many times, One event, three listeners, Room on a small machine, Sources and further reading, Taking power away first (+3 more)

### Community 32 - "apiClient"
Cohesion: 0.15
Nodes (16): config, Context, apiClient, newAPIClient(), apiError, consumerStats, deliveryItem, eventTypeItem (+8 more)

### Community 33 - "Context"
Cohesion: 0.29
Nodes (5): Context, Time, UUID, WebhookService, normalizePagination()

### Community 34 - "Sparrow Architecture"
Cohesion: 0.29
Nodes (7): Sparrow Architecture, Dual-Protocol API, Event Processing Pipeline, Health State Machine, Leaky Bucket Rate Limiting, Standard Webhooks Signing, Default Webhook Body Envelope

### Community 35 - "Token"
Cohesion: 0.11
Nodes (22): AuthError, Invite, Reason, Status, Token, Store, Duration, Time (+14 more)

### Community 36 - "EventProcessingWorker"
Cohesion: 0.15
Nodes (12): Context, Job, JobInserter, Logger, WorkerDefaults, NewEventProcessingWorker(), Time, eventVersionOrDefault() (+4 more)

### Community 37 - "ClassifyError"
Cohesion: 0.20
Nodes (22): ErrorCategory, classifyByMessage(), ClassifyError(), ClassifyHTTPStatus(), classifySyscallError(), isDNSError(), IsRetryableCategory(), isTLSError() (+14 more)

### Community 38 - "envelope"
Cohesion: 0.21
Nodes (10): Context, config, Context, Logger, RawMessage, sinkHandler(), templateContext(), deliverFunc (+2 more)

### Community 39 - "envelope-encryption.md"
Cohesion: 0.22
Nodes (8): Back to Friday, Changing the locks, Friday afternoon, How one secret gets sealed, The seal that tells the truth, Two locks, kept apart, What a stranger would find, What the envelope cannot do

### Community 40 - "Context"
Cohesion: 0.17
Nodes (5): Context, Repository, UUID, WebhookDelivery, WebhookDeliveryStatus

### Community 41 - "sparrow_verify.rs"
Cohesion: 0.24
Nodes (16): hex_decode(), hex_nibble(), parse_headers(), SignatureError, signed_message(), verify_ed25519(), verify_ed25519_at(), verify_hmac() (+8 more)

### Community 42 - "jobInserter"
Cohesion: 0.19
Nodes (9): Context, InsertOpts, JobArgs, JobInsertResult, Logger, Tx, NewJobInserter(), BatchJobArgs (+1 more)

### Community 43 - "mapError"
Cohesion: 0.18
Nodes (17): buildDraftRequest(), findRecipe(), API, Context, Recipe, registerAIRoutes(), registerPromptRoute(), Context (+9 more)

### Community 44 - "lib/utils.ts"
Cohesion: 0.12
Nodes (9): $lib/api-types, preview(), ERROR_CATEGORIES, getCategoryBadge(), getCategoryDisplay(), JSONSchemaMetaSchema, RFC-9457, dismissRetry() (+1 more)

### Community 45 - "conversions.go"
Cohesion: 0.11
Nodes (27): derefString(), formatOptionalTime(), getWebhookEventsMap(), Context, Time, WebhookRegistration, maskEncryptedSecret(), maskSecret() (+19 more)

### Community 46 - "newAuth"
Cohesion: 0.33
Nodes (14): apiKey(), bearer(), Service, T, TB, guarded(), newAuth(), TestAPIKeyHTTPMiddlewareAcceptsHeader() (+6 more)

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

### Community 52 - "event_bundle.go"
Cohesion: 0.21
Nodes (17): fromBundleItems(), API, registerEventBundleRoutes(), toBundleItems(), toImportOutput(), bundleItem, bundleStampBody, eventTypeBundle (+9 more)

### Community 53 - "newEmailSink"
Cohesion: 0.28
Nodes (14): Conn, Template, newEmailSink(), T, serveSMTP(), startSMTPServer(), TestEmailRender_BadTemplate(), TestEmailRender_CustomTemplatesAndHeaderInjection() (+6 more)

### Community 54 - "template-fields.ts"
Cohesion: 0.05
Nodes (61): DiscordCompose, emailArray(), EmailCompose, ENVELOPE_PATHS, generateDiscordTemplate(), generateNtfyTemplate(), generatePagerdutyTemplate(), generateSendgridTemplate() (+53 more)

### Community 55 - "store/models.go"
Cohesion: 0.10
Nodes (16): Int64Array, Time, UUID, Value, DeliveryFilter, EventRecord, JSONMap, JSONStringMap (+8 more)

### Community 56 - "Store"
Cohesion: 0.17
Nodes (14): NullString, NullTime, execer, scanner, Store, Context, Time, insertToken() (+6 more)

### Community 57 - "Commands"
Cohesion: 0.07
Nodes (28): 90-second quickstart, Commands, Configuration, How it's connected, Install, Real-World Use Cases, Scenario 1: Zero-Config Local Webhook Receiver & Debugging, Scenario 2: CI/CD Pipeline Automation & Deployment Notifications (+20 more)

### Community 58 - "Context"
Cohesion: 0.28
Nodes (4): Context, Repository, Time, UUID

### Community 59 - "postSink"
Cohesion: 0.40
Nodes (12): Context, Header, ResponseRecorder, T, postSink(), signHeaders(), TestSinkHandler_DownstreamFailureIs502(), TestSinkHandler_MissingHeaders() (+4 more)

### Community 60 - "setupEnv"
Cohesion: 0.13
Nodes (21): Container, testEnv, T, TestAcquireDeliverySlot_BusyBucketDoesNotBurnSlots(), Context, Pool, Server, T (+13 more)

### Community 61 - "newOTLPSink"
Cohesion: 0.27
Nodes (10): Context, logsURL(), newOTLPSink(), otlpPayload(), T, TestLogsURL(), TestOTLPSinkDeliver(), TestOTLPSinkDeliverDownstreamFailure() (+2 more)

### Community 62 - "devDependencies"
Cohesion: 0.15
Nodes (13): openapi-typescript, @tailwindcss/forms, @tailwindcss/typography, @tailwindcss/vite, @types/node, vite, devDependencies, openapi-typescript (+5 more)

### Community 63 - "newPalette"
Cohesion: 0.16
Nodes (18): Writer, newPalette(), apiClient, Context, Writer, indentJSON(), printEventType(), runEvents() (+10 more)

### Community 64 - "Mount"
Cohesion: 0.12
Nodes (12): main(), API, Mount(), T, TestOpenAPISpecMatchesCommitted(), API, Recipe, registerRecipeRoutes() (+4 more)

### Community 65 - "Client"
Cohesion: 0.22
Nodes (8): Duration, Context, RawMessage, SparrowConfig, NewClient(), T, TestClientAutoCreatesEventType(), Client

### Community 66 - "IsNotFound"
Cohesion: 0.10
Nodes (15): Context, Repository, Time, UUID, Context, Repository, Time, UUID (+7 more)

### Community 67 - "SparrowAPI"
Cohesion: 0.14
Nodes (7): SparrowAPI -- wraps the generated REST client (sparrow_client, generated from…, Poll until all deliveries reach terminal status., Poll a single delivery until terminal., Convert a generated attrs model (or None) to a plain dict., Client for Sparrow's REST API, built on the generated sparrow_client SDK., SparrowAPI, _to_dict()

### Community 68 - "BatchJobWorker"
Cohesion: 0.20
Nodes (11): Context, Job, JobInserter, Logger, UUID, WorkerDefaults, NewBatchJobWorker(), BatchJobWorker (+3 more)

### Community 69 - "recipes_test.go"
Cohesion: 0.36
Nodes (9): T, WebhookTemplateContext, sampleContext(), substituteParams(), TestPagerdutyRecipe_Severity(), TestRecipes(), TestSendGridActivationParams(), TestValidate_RequiresTransformNeedsTemplate() (+1 more)

### Community 70 - "Handler"
Cohesion: 0.19
Nodes (15): MaxBodyBytes(), computeInlineScriptHashes(), FS, Logger, Handler(), InlineScriptHashes(), inlineScriptHashesFromFS(), newHandler() (+7 more)

### Community 71 - "delivery-failures.md"
Cohesion: 0.29
Nodes (6): "Come back later", The friend in the shower, The wrong key, Three red rows, What the red rows say now, When the fault is ours

### Community 72 - "run"
Cohesion: 0.26
Nodes (9): loadConfig(), Context, Logger, main(), run(), config, emailConfig, s3Config (+1 more)

### Community 73 - "newS3Sink"
Cohesion: 0.31
Nodes (7): Context, newS3Sink(), objectKey(), T, TestObjectKey(), TestS3Sink_PutObject(), s3Sink

### Community 74 - "sinks.mdx"
Cohesion: 0.14
Nodes (13): Configuration, How it's connected, Install and run, Real-World Use Cases, Retry semantics, Scenario 1: Long-Term Regulatory & Financial Audit Archiving (S3 / MinIO / R2), Scenario 2: Operational Stakeholder Notifications via SMTP Email, Scenario 3: Centralized OpenTelemetry Log & Event Pipeline (OTLP) (+5 more)

### Community 75 - "Errorf"
Cohesion: 0.12
Nodes (19): stringHeaders(), validateHeaders(), ValidateWebhookURL(), generateWebhookSecret(), Context, WebhookService, UUID, parseUUID() (+11 more)

### Community 76 - "payload-transformation.mdx"
Cohesion: 0.12
Nodes (15): 1. Send only the fields a partner needs, 2. Post a message to Slack, 3. Convert dollars to cents for a billing system, 4. Flatten a nested payload for a legacy endpoint, 5. Summarize a list of items, 6. Add a constant or computed field, 7. Provide safe defaults for optional fields, Good to know (+7 more)

### Community 77 - "startBodyRecorder"
Cohesion: 0.17
Nodes (25): bodyRecorder, requiresTransformSub, subscriptionResp, templateDelivery, Context, T, listWebhookSubs(), TestRequiresTransform_DeliveryWithoutTransformFails() (+17 more)

### Community 78 - "mockRepo"
Cohesion: 0.17
Nodes (8): Context, JobArgs, JobInsertResult, UUID, WebhookRegistration, mockRepo, Mock, mockJobInserter

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
Cohesion: 0.08
Nodes (34): unwrap(), formatAPIError(), cancelInvite(), consumer, copied, create(), createError, creating (+26 more)

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
Cohesion: 0.18
Nodes (11): checkRequiredTransform(), Context, Time, UUID, WebhookRegistration, WebhookService, paginateSubscriptions(), ResumeResult (+3 more)

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

### Community 107 - "Config"
Cohesion: 0.40
Nodes (8): buildCSP(), SecurityHeaders(), T, TestBuildCSP(), TestSecurityHeadersNoHashes(), TestSecurityHeadersPassthrough(), TestSecurityHeadersPortalAllowsFraming(), TestSecurityHeadersWithHashes()

### Community 108 - "1. Payload transform engine: Go `text/template`, not embedded JavaScript"
Cohesion: 0.33
Nodes (5): 1. Payload transform engine: Go `text/template`, not embedded JavaScript, Consequences, Context, Decision, Trigger to revisit

### Community 109 - "ai/ai.go"
Cohesion: 0.28
Nodes (13): completer, Config, DocFetcher, draft, Drafter, HelperFunc, PromptBuilder, Provider (+5 more)

### Community 110 - "runTemplateTest"
Cohesion: 0.50
Nodes (4): Command, Writer, newTemplateCmd(), runTemplateTest()

### Community 111 - "services.ts"
Cohesion: 0.06
Nodes (44): auth, KEY_STORAGE, message, OWNED_TOKEN_STORAGE, ownedTokenId, redeeming, required, stored (+36 more)

### Community 112 - "release-submodules.sh"
Cohesion: 0.80
Nodes (4): release_leaf(), release_module(), release-submodules.sh script, tag_exists()

### Community 113 - "webhook_service.go"
Cohesion: 0.15
Nodes (13): Context, IPNet, WithAllowedNetworks(), WithAllowPrivateNetworks(), WithAutoRegisterEvents(), deliveryRouteService, eventOnlyService, eventRouteService (+5 more)

### Community 114 - "steps_jobs.py"
Cohesion: 0.23
Nodes (22): assert_all_terminal_status(), assert_bulk_retry_count(), assert_consumer_stats(), assert_consumer_stats_webhooks(), assert_global_stats(), assert_health_failed_count(), assert_health_summary(), assert_job_progress() (+14 more)

### Community 115 - "proposal.md"
Cohesion: 0.29
Nodes (6): Capabilities, Impact, Modified Capabilities, New Capabilities, What Changes, Why

### Community 116 - "newPortalGateway"
Cohesion: 0.29
Nodes (16): NewPortalVerifier(), doBearer(), Context, Service, Store, T, mintConsumerToken(), newPortalGateway() (+8 more)

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

### Community 125 - "Wrapf"
Cohesion: 0.15
Nodes (14): anthropicClient, completion, turn, ServiceError, mapStatus(), mapTransportError(), Config, Context (+6 more)

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
Cohesion: 0.14
Nodes (9): Context, UUID, Context, Repository, UUID, EventRegistration, EventRegistrationVersion, fakeEventTypeRepo (+1 more)

### Community 132 - "recipes-as-config.md"
Cohesion: 0.29
Nodes (6): Same care, every time, Tasting before serving, The card in the drawer, The first request, Then everyone else asked, When the recipe goes wrong

### Community 133 - "PrepareDeliveryRequest"
Cohesion: 0.13
Nodes (24): DeliveryRequest, WebhookEnvelope, BuildEnvelopePayload(), BuildRequest(), generateEd25519Signature(), generateHMACSignature(), Context, Duration (+16 more)

### Community 134 - "web/package.json"
Cohesion: 0.20
Nodes (9): openapi-fetch, dependencies, openapi-fetch, svelte-jsoneditor, svelte-jsoneditor, name, private, type (+1 more)

### Community 135 - "webhooks/register/+page.svelte"
Cohesion: 0.08
Nodes (16): src(), return(), if(), changed, editorOpen, lines, on, Recipe (+8 more)

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
Cohesion: 0.29
Nodes (8): main(), GetMigrationsFS(), FS, Context, Logger, RunAllMigrations(), RunAppMigrations(), RunRiverMigrations()

### Community 144 - "Sparrow Web Dashboard"
Cohesion: 0.22
Nodes (8): Build, Configuration, Embedding in the Go Binary, Local Development, Prerequisites, Sparrow Web Dashboard, Standalone Deployment, Tech Stack

### Community 146 - "steps_access.py"
Cohesion: 0.27
Nodes (19): _as_token(), cancel_invite(), create_consumer_token(), create_invite(), _create_token(), get_with_token(), get_with_token_rejected(), _invites() (+11 more)

### Community 147 - "Setup"
Cohesion: 0.11
Nodes (36): ExportTraceServiceRequest, ExportTraceServiceResponse, Int64UpDownCounter, DefaultConfig(), GetMeter(), GetTracer(), Context, Int64Counter (+28 more)

### Community 148 - "httpauth.go"
Cohesion: 0.16
Nodes (18): Principal, Authenticator, ctxKey, ErrorBody, RedeemRequest, RedeemResponse, Verifier, authMessage() (+10 more)

### Community 149 - "rest/access.go"
Cohesion: 0.05
Nodes (64): SecretSealer, cryptoSealer, Duration, Service, Store, InviteTTL(), New(), NewWithStore() (+56 more)

### Community 150 - "corsChain"
Cohesion: 0.28
Nodes (14): CORS(), NormalizeOrigins(), corsChain(), get(), ResponseRecorder, T, TB, preflight() (+6 more)

### Community 151 - "access.md"
Cohesion: 0.13
Nodes (14): API endpoints, CI and machine tokens, Consumer (portal) access, First run, From the API, From the CLI, From the web UI, Invite a teammate (+6 more)

### Community 153 - "parseDateFilter"
Cohesion: 0.33
Nodes (7): Time, parseDateFilter(), parseLabelFilter(), T, TestParseDateFilter(), TestParseDateFilter_AcceptsRFC3339(), TestParseLabelFilter()

### Community 154 - "NetworkPolicy"
Cohesion: 0.13
Nodes (18): NetworkPolicy, embeddedIPv4(), IPNet, isMetadataIP(), mustCIDRs(), ParseNetworks(), T, TestDialControlBlocksMetadataWithAllowPrivate() (+10 more)

### Community 155 - "NewWebhookTemplateContext"
Cohesion: 0.36
Nodes (10): T, TestTransformPayloadWith_CachesStrictAndLenientSeparately(), TestTransformPayloadWith_StrictMissingKeys(), NewWebhookTemplateContext(), T, loadSendgridTemplate(), TestSendgridRecipe_CustomEvent(), TestSendgridRecipe_DeliveryFailed() (+2 more)

### Community 156 - "httpauth_test.go"
Cohesion: 0.28
Nodes (13): downStore, Context, ResponseRecorder, Service, Store, T, newSvc(), serve() (+5 more)

### Community 158 - "WebhookClient"
Cohesion: 0.18
Nodes (9): redactSpanURL, WebhookClient, Config, Context, Duration, Response, WebhookTemplateContext, RedactURL() (+1 more)

### Community 159 - "why-sparrow.md"
Cohesion: 0.40
Nodes (4): A promise you can't check is a hope, Keep it where the events live, Own only what you can carry, Some things should stay home

### Community 164 - "Dual Webhook Signing (HMAC-SHA256 + Ed25519)"
Cohesion: 0.67
Nodes (3): Dual Webhook Signing (HMAC-SHA256 + Ed25519), Timestamp Replay Protection, Standard Webhooks Format

### Community 168 - "runUse"
Cohesion: 0.20
Nodes (14): apiClient, Command, Context, recipe, Writer, loadRecipe(), newUseCmd(), resolveRecipe() (+6 more)

### Community 174 - "EventTypeSaveResult"
Cohesion: 0.29
Nodes (11): breakingChangeError(), Context, UUID, WebhookService, metadataEqual(), planEventTypeSave(), schemasEqual(), EventTypeSaveAction (+3 more)

### Community 175 - "fakeServer"
Cohesion: 0.13
Nodes (36): fakeServer(), Server, T, runCLI(), TestAccessCommandsSurfaceServerErrors(), TestInviteBuildsLinkOnUIURL(), TestInviteDefaultsToServerURLAndJSON(), TestInvitesListAndCancel() (+28 more)

### Community 176 - "GetBuffer"
Cohesion: 0.28
Nodes (11): GetBuffer(), GetHeaderMap(), Buffer, PutBuffer(), PutHeaderMap(), BenchmarkBufferPool(), BenchmarkHeaderMapPool(), B (+3 more)

### Community 177 - "schema_compat.go"
Cohesion: 0.29
Nodes (9): ClassifySchemaChange(), firstComposition(), joinPath(), requiredSet(), sortedKeys(), typeAllows(), typeChange(), typesOf() (+1 more)

### Community 178 - "production.md"
Cohesion: 0.15
Nodes (12): Access and authentication, Deployment, Example Ingress (internal), Kubernetes manifests, Network exposure, NetworkPolicy, Production checklist, Required configuration (+4 more)

### Community 182 - "pushSystemEvent"
Cohesion: 0.27
Nodes (11): Context, JobInserter, Logger, WebhookWorker, UUID, pushSystemEvent(), SystemEventRegistrations(), systemEventSamplePayload() (+3 more)

### Community 183 - "deliveries/+page.svelte"
Cohesion: 0.27
Nodes (13): applyFilters(), clearFilters(), executeRetry(), fetchDeliveries(), handlePageChange(), onBatchDone(), prepareRetryBatch(), retrySingleDelivery() (+5 more)

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

### Community 189 - "PortalGateway"
Cohesion: 0.18
Nodes (14): ResponseWriter, Service, NewAuth(), writeJSONError(), cleanPortalPath(), Context, ResponseWriter, PortalAuthorized() (+6 more)

### Community 190 - "webhooks/event_type_bundle_test.go"
Cohesion: 0.31
Nodes (9): T, TestBundleDigest_IgnoresFormattingAndKeyOrder(), TestBundleItemJSONShape(), TestCheckStamp(), TestCheckTemplates(), TestValidateBundle(), T, js() (+1 more)

### Community 197 - "Context"
Cohesion: 0.34
Nodes (4): Context, Duration, Repository, UUID

### Community 198 - "BatchCleanupWorker"
Cohesion: 0.21
Nodes (8): Context, InsertOpts, Job, Logger, WorkerDefaults, NewBatchCleanupWorker(), BatchCleanupArgs, BatchCleanupWorker

### Community 199 - ".VerifyEd25519"
Cohesion: 0.38
Nodes (8): Duration, Header, Time, parseHeaders(), signedMessage(), VerifyEd25519(), VerifyHMAC(), Verifier

### Community 201 - "SparrowVerify"
Cohesion: 0.35
Nodes (10): SparrowVerify, decode_public_key(), decode_secret(), decode_signatures(), fetch_header(), parse_headers(), safe_hex_decode(), validate_timestamp() (+2 more)

### Community 202 - "SparrowVerify"
Cohesion: 0.35
Nodes (3): SparrowVerify, SparrowVerify::SignatureVerificationError, StandardError

### Community 203 - "Request"
Cohesion: 0.24
Nodes (6): Request, Result, Context, mustJSON(), ResolveDocs(), stubDrafter

### Community 204 - "SparrowVerify"
Cohesion: 0.22
Nodes (3): SignatureVerificationException, SparrowVerify, RuntimeException

### Community 205 - "newRESTClient"
Cohesion: 0.38
Nodes (11): eventTypeResp, eventTypeVersionResp, T, TestEventTypeVersions_AutoRegisterThenFillIn(), TestEventTypeVersions_BreakingChangeNeedsOptIn(), TestEventTypeVersions_ConcurrentSchemaChanges(), TestEventTypeVersions_Lifecycle(), TestEventTypeVersions_NoDelete() (+3 more)

### Community 206 - "verify-signatures.mdx"
Cohesion: 0.50
Nodes (3): Get the helper and verify, Testing your receiver, What gets signed

### Community 207 - ".PushEvent"
Cohesion: 0.22
Nodes (7): IsReservedEventName(), reservedEventNameError(), TestIsReservedEventName(), validateEventTypeName(), generateSamplePayload(), ValidateJSONSchema(), SchemaValidationError

### Community 208 - "schema-infer.ts"
Cohesion: 0.21
Nodes (15): analyzeSamples(), collect(), DATE, detectStringFormat(), emit(), FieldCoverage, inferSchema(), jsonToJsonSchema() (+7 more)

### Community 209 - "TokenPurgeWorker"
Cohesion: 0.11
Nodes (18): Context, JobArgs, JobInserter, JobInsertResult, Span, NewJobInserterWithTracing(), Duration, Context (+10 more)

### Community 210 - "fakeSystemEventRepo"
Cohesion: 0.27
Nodes (6): Context, JobArgs, JobInsertResult, UUID, fakeSystemEventRepo, noopJobInserter

### Community 212 - "runPush"
Cohesion: 0.11
Nodes (12): parseJSONArg(), T, TestKVFlag(), TestParseJSONArg(), apiClient, Command, Context, Writer (+4 more)

### Community 213 - "importBundle"
Cohesion: 0.45
Nodes (10): bundle, importResult, exportBundle(), Context, T, importBundle(), TestEventTypeBundle_AllOrNothingAndBreakingChanges(), TestEventTypeBundle_InvalidInput() (+2 more)

### Community 214 - "Config"
Cohesion: 0.22
Nodes (6): Config, DefaultConfig(), Duration, IPNet, T, TestDefaultConfig()

### Community 215 - "TestE2E_HappyPath"
Cohesion: 0.31
Nodes (10): capturedWebhook, Context, Duration, Header, Server, T, pollDeliverySuccess(), startWebhookTarget() (+2 more)

### Community 216 - "config/config_test.go"
Cohesion: 0.43
Nodes (7): Config, T, TestAIConfig(), TestUIInjectKeyIsRetired(), TestValidate(), TestWarnings(), validConfig()

### Community 218 - "alert_config.go"
Cohesion: 0.24
Nodes (11): API, registerAlertConfigRoutes(), toAlertConfigItem(), alertConfigIDInput, alertConfigItem, alertConfigOutput, createAlertConfigBody, createAlertConfigInput (+3 more)

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

### Community 225 - "for-decision-makers.mdx"
Cohesion: 0.14
Nodes (13): Does it depend on AI?, Does it tie us to Sparrow's way of doing things?, Does this help with HIPAA, GDPR, SOC 2, PCI?, How do our customers know a webhook really came from us?, How do we know it is reliable?, Is the stack open all the way down?, The short version, What does it cost? (+5 more)

### Community 226 - "AutoDisableResult"
Cohesion: 0.21
Nodes (9): Context, Duration, UUID, Manager, Time, fakeAutoDisableRepo, AutoDisableResult, HealthRepository (+1 more)

### Community 227 - "event-type-versioning.mdx"
Cohesion: 0.25
Nodes (7): Breaking changes, Inferring a schema from real events, Reading the history, Reserved names, Retiring an event type, Unregistered event names, What creates a new version

### Community 228 - "resolveConfig"
Cohesion: 0.36
Nodes (9): configPath(), resolveConfig(), saveConfig(), T, TestResolveConfigDefaults(), TestResolveConfigPrecedence(), TestSaveConfigPermissions(), writeConfigFile() (+1 more)

### Community 229 - "generator.ts"
Cohesion: 0.08
Nodes (25): entryToSimpleMarkdown(), htmlToMarkdownPipeline, minify, minifyDefaults, selectors, collator, generateLlmsTxt(), starlightLlmsTxt() (+17 more)

### Community 231 - "newOpenAIDrafter"
Cohesion: 0.40
Nodes (8): fakeOpenAI, HandlerFunc, T, newOpenAIDrafter(), TestNew_ProviderValidation(), TestOpenAIProvider_AuthAndErrors(), TestOpenAIProvider_DraftsAndRepairs(), TestParseDraft_ToleratesFencesAndPreamble()

### Community 232 - "TestE2E_WebhookHealthAlerts"
Cohesion: 0.60
Nodes (5): assertHasRecipient(), Context, T, pollSystemEvent(), TestE2E_WebhookHealthAlerts()

### Community 233 - "runInit"
Cohesion: 0.31
Nodes (9): File, Command, config, Context, Writer, isTerminal(), newInitCmd(), probeServer() (+1 more)

### Community 234 - "subscription-pause.mdx"
Cohesion: 0.33
Nodes (5): Pausing from an import, Resuming, Seeing what was held, What a pause does, Where pause fits in a delivery

### Community 235 - "RenderTemplatePreview"
Cohesion: 0.50
Nodes (3): T, TestRenderTemplatePreview_StrictFlag(), RenderTemplatePreview()

### Community 236 - "newStatsCmd"
Cohesion: 0.38
Nodes (6): apiClient, Command, Context, Writer, newStatsCmd(), runStats()

### Community 237 - "newRootCmd"
Cohesion: 0.20
Nodes (12): Command, Writer, newRootCmd(), newVersionCmd(), Context, Writer, main(), run() (+4 more)

### Community 239 - "provider_openai.go"
Cohesion: 0.32
Nodes (7): oaMessage, oaRequest, oaResponse, openAIClient, Config, newOpenAIClient(), truncate()

### Community 240 - "install.sh"
Cohesion: 0.80
Nodes (4): fail(), need(), install.sh script, usage()

### Community 241 - "newWorkerMetrics"
Cohesion: 0.43
Nodes (5): Float64Histogram, Context, Int64Counter, newWorkerMetrics(), workerMetrics

### Community 242 - "github.com/sarathsp06/sparrow"
Cohesion: 0.53
Nodes (6): github.com/sarathsp06/sparrow, github.com/sarathsp06/sparrow/pkg/access, github.com/sarathsp06/sparrow/pkg/signature, github.com/sarathsp06/sparrow/pkg/template, github.com/sarathsp06/sparrow/satellites/recipes, github.com/sarathsp06/sparrow/satellites/sparrow

### Community 248 - "NewDocFetcher"
Cohesion: 0.36
Nodes (6): collapseSpace(), htmlToText(), looksLikeHTML(), NewDocFetcher(), T, TestDocFetcher_ExtractsTextAndRespectsPolicy()

### Community 249 - "newAutoDisableWorker"
Cohesion: 0.46
Nodes (7): T, WebhookWorker, newAutoDisableWorker(), TestHoldReason(), TestMaybeAutoDisable_DisablesAndEmits(), TestMaybeAutoDisable_NothingDisabledEmitsNothing(), TestMaybeAutoDisable_Skips()

### Community 252 - "SchemaFromSamples.svelte"
Cohesion: 0.24
Nodes (7): coverage, generate(), loadEvents(), picked, readSample(), setSelection(), toggle()

### Community 253 - "TestCLI_EventTypeExportImport"
Cohesion: 0.80
Nodes (5): buildCLI(), T, runCLI(), TestCLI_E2E(), TestCLI_EventTypeExportImport()

### Community 255 - "BatchJob"
Cohesion: 0.21
Nodes (7): Context, Repository, UUID, RawMessage, BatchJob, BatchJobData, BatchJobStatus

### Community 257 - "parseRetryAfter"
Cohesion: 0.38
Nodes (9): T, TestDefaultAndMaxRetryAfterConstants(), TestIsSuccessStatusCode(), TestParseRetryAfter(), TestParseRetryAfter_HTTPDate(), TestParseRetryAfter_HTTPDate_FarFuture(), TestParseRetryAfter_HTTPDate_Past(), isSuccessStatusCode() (+1 more)

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
- **596 isolated node(s):** `DEFAULT_TOLERANCE_SECONDS`, `SignatureVerificationError`, `Headers`, `name`, `type` (+591 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **40 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Errorf()` connect `Errorf` to `testContext`, `PrepareDeliveryRequest`, `RepositoryInterfaceWithTracing`, `webhooks/models.go`, `NewService`, `WebhookServiceInterfaceWithTracing`, `ValidateHeaders`, `RunAllMigrations`, `Context`, `Setup`, `NewWebhookHandler`, `rest/access.go`, `WebhookWorker`, `event_type_bundle.go`, `addOutputFlag`, `parseDateFilter`, `NetworkPolicy`, `apiClient`, `Context`, `EventProcessingWorker`, `ClassifyError`, `envelope`, `runUse`, `Context`, `jobInserter`, `EventTypeSaveResult`, `TemplateEngine`, `newEmailSink`, `Store`, `newTailCmd`, `Context`, `setupEnv`, `newOTLPSink`, `newPalette`, `Client`, `IsNotFound`, `BatchJobWorker`, `Context`, `.VerifyEd25519`, `run`, `newS3Sink`, `.PushEvent`, `fakeSystemEventRepo`, `access/service_test.go`, `runPush`, `clientFromCmd`, `GetFunctionMap`, `listenHandler`, `restClient`, `runFunctions`, `NewManager`, `AutoDisableResult`, `WebhookService`, `Recipe`, `resolveConfig`, `runInit`, `ai/ai.go`, `runTemplateTest`, `NewDocFetcher`, `WebhookService`, `Wrapf`, `BatchJob`?**
  _High betweenness centrality (0.286) - this node is a cross-community bridge._
- **Why does `setupEnv()` connect `setupEnv` to `registerEventType`, `testContext`, `PrepareDeliveryRequest`, `NewService`, `WebhookServiceInterfaceWithTracing`, `Context`, `TestE2E_SlackRecipeTransform`, `Mount`, `newRESTClient`, `startBodyRecorder`, `access/service_test.go`, `importBundle`, `TestE2E_HappyPath`, `TestE2E_EmailSink`, `TestE2E_SourcesGitHubWebhook`, `NewManager`, `TestE2E_WebhookHealthAlerts`, `webhook_service.go`, `TestCLI_EventTypeExportImport`?**
  _High betweenness centrality (0.046) - this node is a cross-community bridge._
- **Why does `Handler()` connect `Handler` to `envelope`, `Config`, `Setup`, `newPortalGateway`, `httpauth.go`, `corsChain`, `rest/access.go`, `aiRouter`, `listenHandler`, `NewWebhookHandler`, `httpauth_test.go`, `PortalGateway`?**
  _High betweenness centrality (0.037) - this node is a cross-community bridge._
- **Are the 209 inferred relationships involving `Errorf()` (e.g. with `.Authenticate()` and `.GetOrCreateToken()`) actually correct?**
  _`Errorf()` has 209 INFERRED edges - model-reasoned connections that need verification._
- **Are the 57 inferred relationships involving `setupEnv()` (e.g. with `TestCLI_E2E()` and `TestCLI_EventTypeExportImport()`) actually correct?**
  _`setupEnv()` has 57 INFERRED edges - model-reasoned connections that need verification._
- **What connects `DEFAULT_TOLERANCE_SECONDS`, `SignatureVerificationError`, `Headers` to the rest of the system?**
  _596 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `event.go` be split into smaller, more focused modules?**
  _Cohesion score 0.07928118393234672 - nodes in this community are weakly interconnected._