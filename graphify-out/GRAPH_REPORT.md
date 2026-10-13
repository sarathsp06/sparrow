# Graph Report - workbench-payload-template-input-7682ff  (2026-10-13)

## Corpus Check
- 544 files · ~488,817 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 46 file(s) not represented in the graph (top: .spec 20, (none) 10, .ttf 6)

## Summary
- 5283 nodes · 15254 edges · 306 communities (197 shown, 109 thin omitted)
- Extraction: 94% EXTRACTED · 6% INFERRED · 0% AMBIGUOUS · INFERRED: 863 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `a43ca019`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- github.com/google/uuid.UUID
- context.Context
- testing.T
- go_pkg_context
- [webhookId]/+page.svelte
- newWebhookHandler
- testhelpers_test.go
- newHandler
- log/slog.Logger
- Errorf
- setupEnv
- webhooks/register/+page.svelte
- .run
- testContext
- EventRegistration
- otel.go
- EventSubscription
- access/service.go
- api-types.d.ts
- TemplateEditor.svelte
- NewWebhookHandler
- encoding/json.RawMessage
- NewService
- webhooks/+page.svelte
- template_errors_test.go
- io.Writer
- webhook_service.go
- posts.ts
- RecipeWorkbenchApp.svelte
- SubscriptionForm.svelte
- time.Time
- Repository
- All
- Errorf
- Manager
- .compare
- deliveries/+page.svelte
- push/+page.svelte
- event.go
- unwrap
- EventRecord
- Service
- middleware/portal_test.go
- mapError
- lib/utils.ts
- events/+page.svelte
- generator.ts
- lib/utils.ts
- NewWebhookClient
- access/+page.svelte
- NewTemplateEngine
- reports/+page.svelte
- portal_gateway.go
- BatchJob
- newRootCmd
- E2ERunner
- server/main.go
- runInvite
- Context
- postSink
- listDeliveriesImpl
- services.ts
- access/service_test.go
- email_test.go
- rest/access.go
- New
- mockRepo
- web/package.json
- update/+page.svelte
- sinks_test.go
- Handler
- event_bundle.go
- run
- prompt.go
- auth_test.go
- parseUUID
- docs/package.json
- template-fields.ts
- mockRepo
- SchemaFromSamples.svelte
- sources.mdx
- ADDED Requirements
- Release GoReleaser Job
- portal/+page.svelte
- access/service_test.go
- Pusher
- RepositoryInterface
- schema-infer.ts
- compilerOptions
- WebhookTargetManager
- deployment/security.mdx
- clientFromCmd
- Service
- NewTemplateEngine
- emitHealthChangedEvent
- main
- registerWebhookRoutes
- BenchmarkTransformPayload
- _Target
- Checker
- Recipe
- New
- BatchJob
- Handle
- 2. Split the `sparrow` CLI into its own module to isolate its `go install` graph
- Config
- httpauth.go
- Handler
- devDependencies
- Security policy
- parseCron
- webhook_service.go
- steps_jobs.py
- proposal.md
- WebhookClient
- init_cmd.go
- Mount
- portal-embedding.mdx
- webhook-health-alerts.mdx
- design.md
- ReadingProgress.astro
- api.ts
- Config
- Wrapf
- Sparrow Webhook Delivery Platform
- scripts
- models_test.go
- entryToSimpleMarkdown.ts
- Envelope
- EventRegistration
- provider_openai.go
- PrepareDeliveryRequest
- Wrapf
- Documenting the REST API
- buildVectors
- Recipe
- Sparrow threat model
- NetworkPolicy
- Handler
- Sparrow Recipes
- vite-plugin-devtools-json
- Client
- Sparrow Web Dashboard
- @sveltejs/kit
- steps_access.py
- Setup
- httpauth.go
- net/http.Header
- Contributing to Sparrow
- .CheckIP
- playwright.config.ts
- parseDateFilter
- NetworkPolicy
- NewWebhookTemplateContext
- httpauth_test.go
- svelte-check
- Sparrow
- GetBuffer
- @astrojs/starlight
- docs/tsconfig.json
- TestJobInserter_InsertOpts_Merge
- TestWebhookWorkerNextRetry
- Dual Webhook Signing (HMAC-SHA256 + Ed25519)
- +layout.ts
- docs/src/lib/recipes.ts
- net/http.Client
- runUse
- time.Duration
- content.config.ts
- Deliberately NOT covered by e2e
- pages/index.astro
- tailwindcss
- NewDocFetcher
- fakeServer
- astro
- schema_compat.go
- production.md
- NewWithStore
- clientTester
- build.go
- svelte.config.js
- deliveries/+page.svelte
- aiRouter
- go_pkg_embed
- newTailCmd
- @sveltejs/vite-plugin-svelte
- SparrowVerify
- xorSealer
- runPush
- Code conventions
- dev-env.sh
- static-server.mjs
- Store
- Status
- config.js
- ShareBar.astro
- BatchCleanupWorker
- .VerifyEd25519
- ReadBody
- SparrowVerify
- SparrowVerify
- Store
- webhookHealthOutput
- newRESTClient
- verify-signatures.mdx
- Mount
- schema-infer.ts
- TestVectors
- runPush
- importBundle
- sealedRouter
- TestE2E_HappyPath
- All
- Context
- WebhookHealthData
- TestE2E_EmailSink
- listenHandler
- restClient
- TestE2E_SourcesGitHubWebhook
- What an import does
- for-decision-makers.mdx
- newAutoDisableWorker
- event-type-versioning.mdx
- TestCLI_EventTypeExportImport
- Release Workflow
- newOpenAIDrafter
- TestE2E_WebhookHealthAlerts
- svelte
- newRootCmd
- WebhookService
- provider_openai.go
- install.sh
- sparrow-e2e
- sparrow-e2e

## God Nodes (most connected - your core abstractions)
1. `unwrap()` - 95 edges
2. `formatAPIError()` - 82 edges
3. `RepositoryInterfaceWithTracing` - 79 edges
4. `EventSubscription` - 70 edges
5. `setupEnv()` - 68 edges
6. `newRESTClient()` - 54 edges
7. `WebhookServiceInterfaceWithTracing` - 51 edges
8. `EventRegistration` - 48 edges
9. `testContext()` - 42 edges
10. `NewWebhookService()` - 39 edges

## Surprising Connections (you probably didn't know these)
- `API surface deliberately not e2e-tested (removal / design candidates)` --references--> `retryDelivery()`  [INFERRED]
  e2e/NOT-TESTING.md → web/src/routes/portal/+page.svelte
- ``sparrow mcp`` --references--> `delete_webhook()`  [INFERRED]
  docs/src/content/docs/satellites/cli.mdx → e2e/step_impl/steps_surface.py
- `1. What the landscape ships as "ecosystem"` --references--> `webhooks`  [INFERRED]
  docs/design/satellite-ideas.md → web/src/lib/api-types.d.ts
- `2.4 CLI` --references--> `webhooks`  [INFERRED]
  docs/design/satellite-ideas.md → web/src/lib/api-types.d.ts
- `4.2 Typed event payloads from event-type schemas` --references--> `webhooks`  [INFERRED]
  docs/design/satellite-ideas.md → web/src/lib/api-types.d.ts

## Import Cycles
- None detected.

## Communities (306 total, 109 thin omitted)

### Community 0 - "github.com/google/uuid.UUID"
Cohesion: 0.03
Nodes (18): statusForFailure(), deliveryFilterArgs(), DeliveryRepository, Repository, AutoDisableResult, HealthRepository, Repository, cursorArgs() (+10 more)

### Community 1 - "context.Context"
Cohesion: 0.03
Nodes (20): AlertConfig, AlertConfigRepository, Repository, Repository, AlertConfigManager, BatchManager, DeliveryManager, EventManager (+12 more)

### Community 2 - "testing.T"
Cohesion: 0.02
Nodes (139): ErrorCategory, sinkSMTPMessage, TestAIConfig(), TestSlogLevel(), TestUIInjectKeyIsRetired(), TestValidate(), TestWarnings(), validConfig() (+131 more)

### Community 3 - "go_pkg_context"
Cohesion: 0.08
Nodes (5): consumerStats, subscriptionResp, applyRecipeIn, recipeNameIn, validateRecipeIn

### Community 4 - "[webhookId]/+page.svelte"
Cohesion: 0.03
Nodes (55): copied, Props, { secret, publicKey = '' }, VERIFY_SIGNATURES_GUIDE_URL, activeTab, applyDeliveryFilters(), changingSignature, clearDeliveryFilters() (+47 more)

### Community 5 - "newWebhookHandler"
Cohesion: 0.07
Nodes (35): loadConfig(), main(), newWebhookHandler(), discardLogger(), githubSig(), stripeSig(), TestLoadConfig(), TestLoadConfig_RequiresURL() (+27 more)

### Community 6 - "testhelpers_test.go"
Cohesion: 0.21
Nodes (20): eventTypeResp, eventTypeVersionResp, TestHealthLabel(), healthLabel(), alertConfigIDInput, attemptItem, attemptsOutput, createAlertConfigBody (+12 more)

### Community 7 - "newHandler"
Cohesion: 0.18
Nodes (14): inheritVerification(), loadConfig(), main(), newHandler(), run(), signEd25519Headers(), signHeaders(), TestLoadConfig_InlineLibraryConfig() (+6 more)

### Community 8 - "log/slog.Logger"
Cohesion: 0.04
Nodes (44): DeliveryRequest, main(), RunAllMigrations(), RunAppMigrations(), RunRiverMigrations(), RedactURL(), BuildEnvelopePayload(), PrepareDeliveryRequest() (+36 more)

### Community 9 - "Errorf"
Cohesion: 0.04
Nodes (54): BundleDigest(), checkStamp(), checkTemplates(), compileSchema(), generatePayload(), EventTypeImportResult, WebhookService, pauseFailing() (+46 more)

### Community 10 - "setupEnv"
Cohesion: 0.11
Nodes (67): deliveryItem, restClient, testEnv, buildCLI(), runCLI(), TestCLI_E2E(), TestCLI_EventTypeExportImport(), TestE2E_ConsumerDeliveryPaging() (+59 more)

### Community 11 - "webhooks/register/+page.svelte"
Cohesion: 0.03
Nodes (57): Schema (version 1), Recipe, RecipeParam, substituteParams(), active, alertDeliveryConfigured, alertEmail, alertEmailError (+49 more)

### Community 13 - "testContext"
Cohesion: 0.07
Nodes (55): Status, TestAlertDeliveryConfigured(), TestCreateSubscription_CatchAllWithLabelFilters(), TestCreateSubscription_WithEmptyLabelFilters(), TestCreateSubscription_WithInvalidLabelFilters(), TestCreateSubscription_WithLabelFilters(), TestGetSubscriptionsWithWebhooksByEvent_ActiveWebhooksOnly(), TestGetSubscriptionsWithWebhooksByEvent_InactiveExcluded() (+47 more)

### Community 14 - "EventRegistration"
Cohesion: 0.05
Nodes (16): registerSystemEventTypes(), SystemEventRegistrations(), EventTypeFilter, EventTypeRepository, Repository, RepositoryInterface, EventRegistration, EventRegistrationVersion (+8 more)

### Community 15 - "otel.go"
Cohesion: 0.07
Nodes (22): DefaultConfig(), GetMeter(), GetTracer(), SparrowMetrics, legacyEndpoint(), MetricsHandler(), newLoggerProvider(), NewSparrowMetrics() (+14 more)

### Community 16 - "EventSubscription"
Cohesion: 0.05
Nodes (10): EventSubscription, SubscriptionTemplateVersion, SubscriptionWithWebhook, Repository, SubscriptionRepository, insertSubscription(), SubscriptionManager, TemplateFunctionInfo (+2 more)

### Community 17 - "access/service.go"
Cohesion: 0.15
Nodes (15): TestMigrationsMatchPgstoreSchema(), cleanPortalPath(), portalTarget(), portalCtxKey, TestModuleImportsOnlyStdlibAndItself(), newSvc(), serve(), TestExtraVerifier() (+7 more)

### Community 18 - "api-types.d.ts"
Cohesion: 0.10
Nodes (48): delivery_has_signature_headers(), _api(), assert_api_delivery_count(), assert_body_field(), assert_body_has_key(), assert_body_not_has_key(), assert_delivery_attempt_count(), assert_delivery_count() (+40 more)

### Community 19 - "TemplateEditor.svelte"
Cohesion: 0.04
Nodes (52): aiAttempts, aiCapability, aiConfirmReplace, aiDocsUrl, aiError, aiInstructions, aiLoading, aiNotes (+44 more)

### Community 20 - "NewWebhookHandler"
Cohesion: 0.16
Nodes (45): alert_configs_count(), alert_configs_count_for_webhook(), also_use_consumer(), assert_delivery_page_meta(), _base(), create_alert_config_bad_webhook(), create_alert_config_scoped(), create_alert_config_wide() (+37 more)

### Community 21 - "encoding/json.RawMessage"
Cohesion: 0.17
Nodes (5): fakePusher, fakePusher, fakePusher, fakePusher, fakePusher

### Community 22 - "NewService"
Cohesion: 0.11
Nodes (41): Key, TestPrepareDeliveryRequest(), TestPrepareDeliveryRequestFailsClosedOnDecryptError(), GenerateKey(), Keyring, isSafeKeyID(), NewKeyring(), NewService() (+33 more)

### Community 23 - "webhooks/+page.svelte"
Cohesion: 0.04
Nodes (60): clear(), commit(), draft, options, search(), {
    value = $bindable(""),
    id,
    placeholder = "All consumers",
    required = false,
    live = false,
    label = "Consumer",
    class: className = "",
    onchange,
  }, c, config (+52 more)

### Community 24 - "template_errors_test.go"
Cohesion: 0.17
Nodes (24): bodyRecorder, bundle, importResult, requiresTransformSub, templateDelivery, exportBundle(), importBundle(), TestEventTypeBundle_AllOrNothingAndBreakingChanges() (+16 more)

### Community 25 - "io.Writer"
Cohesion: 0.09
Nodes (31): runTokensCreate(), newPalette(), importOutcomeError(), printImportResult(), runEventsExport(), runEventsImport(), runEventsVersions(), indentJSON() (+23 more)

### Community 26 - "webhook_service.go"
Cohesion: 0.14
Nodes (9): vector, vectorFile, WebhookEnvelope, TestValidateHeaders(), ValidateHeaders(), validHeaderName(), validHeaderValue(), TestIsSuccessStatusCode() (+1 more)

### Community 27 - "posts.ts"
Cohesion: 0.10
Nodes (28): links, Props, tags, Props, Props, allTags(), base, formatDate() (+20 more)

### Community 28 - "RecipeWorkbenchApp.svelte"
Cohesion: 0.05
Nodes (39): activeRecipe, attempt, b64(), chips, cliCommand, copiedToast, DEST, drafts (+31 more)

### Community 29 - "SubscriptionForm.svelte"
Cohesion: 0.03
Nodes (68): apiConsole, ApiLogEntry, apiLogMiddleware, entries, started, toCurl(), copiedId, copy() (+60 more)

### Community 30 - "time.Time"
Cohesion: 0.05
Nodes (26): countingStore, Status, downStore, encodeCursor(), newCursorPagination(), FailingWebhook, Repository, Store (+18 more)

### Community 31 - "Repository"
Cohesion: 0.08
Nodes (9): DeliveryAttempt, Repository, checkWebhookDuplicate(), Repository, WebhookRepository, insertWebhookRegistration(), DB, DBTX (+1 more)

### Community 32 - "All"
Cohesion: 0.08
Nodes (29): All(), TestRecipes_ParamGuidance(), TestSendgridRecipe_ConsumerHint(), addRecipeTools(), declaresJSON(), TestResolveParams(), TestValidateRecipe_BuiltinsAreCleanAndValid(), TestValidateRecipe_WholeURLParamCheckedOnceGiven() (+21 more)

### Community 33 - "Errorf"
Cohesion: 0.11
Nodes (5): SignatureVerificationException, SparrowVerify, ParsedHeaders, SignatureVerificationException, SparrowVerify

### Community 34 - "Manager"
Cohesion: 0.12
Nodes (7): NewJobInserterWithTracing(), jobInserter, NewJobInserter(), Manager, JobInserterWithTracing, noopJobInserter, mockJobInserter

### Community 35 - ".compare"
Cohesion: 0.21
Nodes (9): ClassifySchemaChange(), firstComposition(), joinPath(), requiredSet(), sortedKeys(), typeAllows(), typeChange(), typesOf() (+1 more)

### Community 36 - "deliveries/+page.svelte"
Cohesion: 0.04
Nodes (47): { pages, shown, onchange, itemLabel = "items" }, Props, CursorPages, allConsumers, applyFilters(), clearFilters(), createdAfterFilter, createdBeforeFilter (+39 more)

### Community 37 - "push/+page.svelte"
Cohesion: 0.06
Nodes (36): {
    expanded = $bindable(false),
    label = "Editor",
    children,
  }, isSystemEvent(), listAllEventTypes(), safeAjvValidator(), availableEvents, consumer, error, event (+28 more)

### Community 38 - "event.go"
Cohesion: 0.15
Nodes (15): eventIDOnlyInput, eventTypeBody, eventTypeNameInput, eventTypeVersionInput, jobIDInput, listEventTypesInput, patchEventTypeInput, pushEventBody (+7 more)

### Community 39 - "unwrap"
Cohesion: 0.04
Nodes (88): adding, AlertConfigItem, configs, { consumer, webhookId }, deliveryConfigured, email, error, eventLabel (+80 more)

### Community 40 - "EventRecord"
Cohesion: 0.10
Nodes (8): eventReportFilterArgs(), eventVersionOrDefault(), EventRepository, Repository, EventRecord, EventReportFilter, EventReportWithStats, EventReportPage

### Community 41 - "Service"
Cohesion: 0.14
Nodes (11): cacheEntry, Config, RootKey, Principal, invalidRequest(), clonePtr(), Service, hashSecret() (+3 more)

### Community 42 - "middleware/portal_test.go"
Cohesion: 0.36
Nodes (13): NewPortalVerifier(), PortalGateway(), doBearer(), mintConsumerToken(), newPortalGateway(), newPortalService(), TestPortalGatewayAcceptsConsumerAccessTokens(), TestPortalGatewayReDispatchesThroughRouter() (+5 more)

### Community 43 - "mapError"
Cohesion: 0.10
Nodes (29): idx_webhook_deliveries_created_at, idx_webhook_deliveries_event_id, idx_webhook_deliveries_expires_at, idx_webhook_deliveries_request_body_gin, idx_webhook_deliveries_status, idx_webhook_deliveries_subscription_id, idx_webhook_deliveries_webhook_id, idx_webhook_health_events_delivery_id (+21 more)

### Community 44 - "lib/utils.ts"
Cohesion: 0.12
Nodes (23): newAPIClient(), newMCPServer(), ptr(), call(), connectMCP(), newFakeSparrow(), TestMCPListenRoundTrip(), TestMCPPushAutoCreatesEventType() (+15 more)

### Community 45 - "events/+page.svelte"
Cohesion: 0.05
Nodes (33): { href, label, targetSelector }, visible, { currentPage, totalPages, totalCount, pageSize, onPageChange, itemLabel = "items" }, from, pages, Props, to, allVisibleSelected (+25 more)

### Community 46 - "generator.ts"
Cohesion: 0.10
Nodes (20): entryToSimpleMarkdown(), collator, generateLlmsTxt(), GET(), getStaticPaths, Params, prerender, Props (+12 more)

### Community 47 - "lib/utils.ts"
Cohesion: 0.03
Nodes (61): BatchKind, BatchProgressState, terminal, AttemptItem, attempts, { deliveryId }, error, loading (+53 more)

### Community 48 - "NewWebhookClient"
Cohesion: 0.07
Nodes (29): acknowledged, actionLabel, actionTone, allAcknowledged, breakingConfirmed, breakingItems, bundle, busy (+21 more)

### Community 49 - "access/+page.svelte"
Cohesion: 0.06
Nodes (26): { icon = "inbox", title, description, action }, paths, Props, cancelling, consumer, copied, createError, creating (+18 more)

### Community 50 - "NewTemplateEngine"
Cohesion: 0.09
Nodes (10): setup_environment(), setup_scenario(), teardown_environment(), teardown_scenario(), _get(), get_authed_correct_key(), get_authed_no_key(), get_authed_with_key() (+2 more)

### Community 51 - "reports/+page.svelte"
Cohesion: 0.07
Nodes (31): { batch, label = 'Batch', oncancel, ondone }, BatchStatus, dismissed, isTerminal, progressPercent, Props, statusColor, statusLabel (+23 more)

### Community 52 - "portal_gateway.go"
Cohesion: 0.06
Nodes (30): 1. Webhook Registration, 2. Event Registration, 3. Event Push (PushEvent), 4. Event Processing (EventProcessingWorker), 5. Webhook Delivery (WebhookWorker), 6. Subscription Creation, 7. Webhook Update (UpdateWebhookConfig), DB Writes (+22 more)

### Community 53 - "BatchJob"
Cohesion: 0.08
Nodes (10): NewBatchJobWorker(), BatchRepository, Repository, BatchJob, BatchJobStatus, BatchJobType, WebhookService, BatchJobArgs (+2 more)

### Community 54 - "newRootCmd"
Cohesion: 0.19
Nodes (29): accessLabel(), newInviteCmd(), newInvitesCmd(), newTokensCmd(), newTokensCreateCmd(), newTokensListCmd(), newTokensRevokeCmd(), timeOr() (+21 more)

### Community 56 - "server/main.go"
Cohesion: 0.08
Nodes (42): oaMessage, oaRequest, oaResponse, toImportOutput(), capabilitiesOutput, ConsumerItem, consumerOnlyInput, consumerStatsOutput (+34 more)

### Community 57 - "runInvite"
Cohesion: 0.11
Nodes (11): apiClient, parseDuration(), runInvite(), TestParseDuration(), createdInvite, createdToken, durationFlag, inviteOut (+3 more)

### Community 58 - "Context"
Cohesion: 0.14
Nodes (13): apiClient, apiError, consumerStats, deliveryItem, eventTypeItem, pushBody, pushResult, subscriptionItem (+5 more)

### Community 59 - "postSink"
Cohesion: 0.08
Nodes (4): CapturedDelivery, _Target, handle(), WebhookTargetManager

### Community 60 - "listDeliveriesImpl"
Cohesion: 0.09
Nodes (27): listDeliveriesImpl(), registerDeliveryRoutes(), toDeliveryItem(), toDeliveryOutput(), listEventOccurrencesImpl(), toBatchJobOutput(), parseDateFilter(), parseLabelFilter() (+19 more)

### Community 61 - "services.ts"
Cohesion: 0.05
Nodes (52): Use it as a Go library, onSignOut(), signingOut, who, auth, KEY_STORAGE, message, OWNED_TOKEN_STORAGE (+44 more)

### Community 62 - "access/service_test.go"
Cohesion: 0.27
Nodes (24): clock, New(), New(), newClock(), newSealedService(), newService(), ptr(), reason() (+16 more)

### Community 63 - "email_test.go"
Cohesion: 0.35
Nodes (10): smtpMessage, newEmailSink(), serveSMTP(), startSMTPServer(), TestEmailRender_BadTemplate(), TestEmailRender_CustomTemplatesAndHeaderInjection(), TestEmailRender_Defaults(), TestEmailSink_SendsViaSMTP() (+2 more)

### Community 64 - "rest/access.go"
Cohesion: 0.12
Nodes (25): TestValidateConsumer(), ValidateConsumer(), consumerScope(), fragmentEscape(), mapAccessError(), portalLinkPath(), principalName(), registerAccessRoutes() (+17 more)

### Community 65 - "New"
Cohesion: 0.11
Nodes (23): Config, Drafter, fakeModel, fakeOpenAI, Provider, Renderer, DocFetcher, New() (+15 more)

### Community 66 - "mockRepo"
Cohesion: 0.10
Nodes (6): getWebhookEventsMap(), mockRepo, WebhookHealth, WebhookRegistration, mockRepo, conversionService

### Community 67 - "web/package.json"
Cohesion: 0.10
Nodes (20): openapi-fetch, openapi-typescript, svelte-check, svelte-jsoneditor, @sveltejs/kit, tailwindcss, @tailwindcss/forms, @tailwindcss/vite (+12 more)

### Community 68 - "update/+page.svelte"
Cohesion: 0.10
Nodes (24): toJSONObject(), active, confirmPending(), description, error, hasSchema(), ImportItem, loading (+16 more)

### Community 69 - "sinks_test.go"
Cohesion: 0.38
Nodes (11): postSink(), signEd25519Headers(), signHeaders(), TestHandle_Ed25519(), TestSinkHandler_DownstreamFailureIs502(), TestSinkHandler_MissingHeaders(), TestSinkHandler_TamperedBody(), TestSinkHandler_TransformedBodyIs422() (+3 more)

### Community 70 - "Handler"
Cohesion: 0.15
Nodes (9): _check_timestamp(), _decode_signatures(), _required_headers(), SignatureVerificationError, verify_ed25519(), verify_hmac(), _signed_body(), verify_ed25519_signature() (+1 more)

### Community 71 - "event_bundle.go"
Cohesion: 0.10
Nodes (18): event_subscriptions, idx_event_subscriptions_event, idx_webhook_registrations_active, idx_webhook_registrations_consumer, idx_webhook_registrations_health, webhook_registrations, idx_event_records_tenant_consumer, idx_event_subscriptions_tenant (+10 more)

### Community 72 - "run"
Cohesion: 0.29
Nodes (4): Checker, HealthResponse, ReadyResponse, NewChecker()

### Community 73 - "prompt.go"
Cohesion: 0.15
Nodes (21): HelperFunc, outlineField, PromptBuilder, RecipeParam, fieldStep(), Recipe, mustJSON(), NewPromptBuilder() (+13 more)

### Community 74 - "auth_test.go"
Cohesion: 0.22
Nodes (14): redactSpanURL, NewAuth(), apiKey(), bearer(), guarded(), newAuth(), TestAPIKeyHTTPMiddlewareAcceptsHeader(), TestAPIKeyHTTPMiddlewareBypassesExcludedPath() (+6 more)

### Community 75 - "parseUUID"
Cohesion: 0.15
Nodes (17): DEFAULT_TOLERANCE_SECS, hex_decode(), hex_nibble(), parse_headers(), SignatureError, EmptySecret, InvalidPublicKey, InvalidSecretBase64 (+9 more)

### Community 76 - "docs/package.json"
Cohesion: 0.10
Nodes (19): blogLastmod, devDependencies, starlight-llms-txt, svelte, name, private, type, version (+11 more)

### Community 77 - "template-fields.ts"
Cohesion: 0.20
Nodes (22): editField(), actionBody(), actionEnd(), balanced(), decodeAction(), decodeJsonString(), decodeUrlValue(), encodeText() (+14 more)

### Community 78 - "mockRepo"
Cohesion: 0.22
Nodes (20): assert_all_terminal_status(), assert_bulk_retry_count(), assert_consumer_stats(), assert_consumer_stats_webhooks(), assert_global_stats(), assert_health_failed_count(), assert_health_summary(), assert_job_progress() (+12 more)

### Community 79 - "SchemaFromSamples.svelte"
Cohesion: 0.09
Nodes (19): coverage, error, { eventName, sampleId = '', hasSchema, ongenerate }, EventOccurrenceItem, eventsError, generate(), loadEvents(), loadingEvents (+11 more)

### Community 80 - "sources.mdx"
Cohesion: 0.09
Nodes (21): 1. Payload transform engine: Go `text/template`, not embedded JavaScript, Consequences, Context, Decision, Trigger to revisit, A complete template, Building JSON, default (+13 more)

### Community 81 - "ADDED Requirements"
Cohesion: 0.09
Nodes (22): 1. Reusable library: `pkg/access`, 2. Sparrow adapter: `internal/accessauth`, 3. Replace the single shared API key with revocable access tokens and one-time invites, 3. Two scopes only — no roles, 4. Stored invites instead of signed links, 5. No cascade revocation, 6. Tokens independent of master key, 7. 503, not 401, when the database is down (+14 more)

### Community 82 - "Release GoReleaser Job"
Cohesion: 0.09
Nodes (22): Access, AI drafting, AI template drafting, All variables, An OpenAI-compatible server, Anthropic, Database Pools, Default Tenant (+14 more)

### Community 83 - "portal/+page.svelte"
Cohesion: 0.23
Nodes (11): listSubscriptionsImpl(), registerSubscriptionRoutes(), toSubscriptionItem(), toSubscriptionOutput(), listSubscriptionsGlobalInput, listSubscriptionsInput, listSubscriptionsOutput, resumeSubscriptionOutput (+3 more)

### Community 84 - "access/service_test.go"
Cohesion: 0.10
Nodes (10): out, items, Props, toc, spec, specPath, @scalar/astro, sharp (+2 more)

### Community 85 - "Pusher"
Cohesion: 0.27
Nodes (8): Emitter, entry, Job, New(), discardLogger(), TestCronTickPushesOnMatch(), TestNewCronValidates(), Pusher

### Community 86 - "RepositoryInterface"
Cohesion: 0.16
Nodes (3): `sparrow mcp`, SparrowAPI, _to_dict()

### Community 87 - "schema-infer.ts"
Cohesion: 0.22
Nodes (13): analyzeSamples(), collect(), DATE, detectStringFormat(), emit(), FieldCoverage, KIND_ORDER, newNode() (+5 more)

### Community 88 - "compilerOptions"
Cohesion: 0.10
Nodes (19): Default Webhook Body, Design Principles, Domain Packages, Encryption at Rest, Event Processing Pipeline, Health State Machine, How It Works, HTTP Client Design (+11 more)

### Community 89 - "WebhookTargetManager"
Cohesion: 0.17
Nodes (19): DiscordCompose, emailArray(), EmailCompose, ENVELOPE_PATHS, generateDiscordTemplate(), generateNtfyTemplate(), generatePagerdutyTemplate(), generateSendgridTemplate() (+11 more)

### Community 90 - "deployment/security.mdx"
Cohesion: 0.27
Nodes (17): _as_token(), cancel_invite(), create_consumer_token(), create_invite(), _create_token(), get_with_token(), get_with_token_rejected(), _invites() (+9 more)

### Community 91 - "clientFromCmd"
Cohesion: 0.22
Nodes (10): @playwright/test, ensureAlertDelivery(), ensureEventType(), mintPortalToken(), newConsumer(), newEventName(), pushEvent(), registerWebhook() (+2 more)

### Community 92 - "Service"
Cohesion: 0.18
Nodes (7): cryptoSealer, kek, buildKEK(), decryptDataWithDEK(), Service, newAEAD(), unwrapDEK()

### Community 93 - "NewTemplateEngine"
Cohesion: 0.08
Nodes (38): checkRendered(), TestCheckRendered(), TestTemplateReference_ExampleRenders(), BenchmarkSend(), GetHeaderMap(), PutHeaderMap(), BenchmarkHeaderMapPool(), TestHeaderMapPool() (+30 more)

### Community 94 - "emitHealthChangedEvent"
Cohesion: 0.35
Nodes (4): listenStatus, mcpListener, receivedDelivery, startListenIn

### Community 95 - "main"
Cohesion: 0.06
Nodes (55): logCORSMode(), main(), Load(), CORS(), CORSMode, NormalizeOrigins(), corsChain(), get() (+47 more)

### Community 96 - "registerWebhookRoutes"
Cohesion: 0.09
Nodes (22): derefString(), formatOptionalTime(), maskEncryptedSecret(), maskSecret(), maskSecretHeaders(), toWebhookOut(), toWebhookOutFromDomain(), registerWebhookRoutes() (+14 more)

### Community 97 - "BenchmarkTransformPayload"
Cohesion: 0.17
Nodes (7): draft, Request, Result, parseDraft(), TestParseDraft_ToleratesFencesAndPreamble(), slowDrafter, stubDrafter

### Community 98 - "_Target"
Cohesion: 0.13
Nodes (16): 1. What the landscape ships as "ecosystem", 2.1 Sources, 2.2 Sinks, 2.3 Recipes, 2.4 CLI, 2. Extend the existing satellites (lowest effort, highest leverage), 4.1 Helm chart and Grafana pack, 4.2 Typed event payloads from event-type schemas (+8 more)

### Community 99 - "Checker"
Cohesion: 0.12
Nodes (17): Commands, From a template to a recipe, `sparrow events`, `sparrow events export` and `sparrow events import`, `sparrow functions [name]`, `sparrow init`, `sparrow invite`, `sparrow invites` (+9 more)

### Community 101 - "Recipe"
Cohesion: 0.24
Nodes (9): fromBundleItems(), registerEventBundleRoutes(), toBundleItems(), bundleItem, bundleStampBody, eventRouteService, eventTypeBundle, exportEventTypesOutput (+1 more)

### Community 102 - "New"
Cohesion: 0.12
Nodes (16): ADDED Requirements, Purpose, Requirement: Alert-email configuration resource, Requirement: Email delivery through the core pipeline, Requirement: Human-readable alert content, Requirement: Recipient resolution at push time, Scenario: Consumer-level opt-in, Scenario: Degradation subject (+8 more)

### Community 104 - "Handle"
Cohesion: 0.31
Nodes (9): SMTPConfig, Config, Handler(), Ed25519(), Handle(), HMAC(), VerifierFor(), DeliverFunc (+1 more)

### Community 106 - "2. Split the `sparrow` CLI into its own module to isolate its `go install` graph"
Cohesion: 0.12
Nodes (15): 1. Send only the fields a partner needs, 2. Post a message to Slack, 3. Convert dollars to cents for a billing system, 4. Flatten a nested payload for a legacy endpoint, 5. Summarize a list of items, 6. Add a constant or computed field, 7. Provide safe defaults for optional fields, Good to know (+7 more)

### Community 107 - "Config"
Cohesion: 0.12
Nodes (15): Apply a recipe, clickhouse, cloudevents, discord, ntfy, pagerduty, Pick a recipe, sendgrid (+7 more)

### Community 108 - "httpauth.go"
Cohesion: 0.12
Nodes (19): AuthError, ctxKey, ErrorBody, RedeemRequest, RedeemResponse, Verifier, writeJSONError(), PortalAuthorized() (+11 more)

### Community 109 - "Handler"
Cohesion: 0.25
Nodes (7): s3Sink, Config, Handler(), newS3Sink(), objectKey(), TestObjectKey(), TestS3Sink_PutObject()

### Community 110 - "devDependencies"
Cohesion: 0.13
Nodes (15): devDependencies, openapi-typescript, @playwright/test, svelte, svelte-check, @sveltejs/adapter-static, @sveltejs/kit, @sveltejs/vite-plugin-svelte (+7 more)

### Community 111 - "Security policy"
Cohesion: 0.25
Nodes (4): Reporting a vulnerability, Scope, Security policy, Supported versions

### Community 112 - "parseCron"
Cohesion: 0.29
Nodes (7): cronSpec, parseCron(), parseCronField(), at(), mustParse(), TestCronMatch(), TestParseCronInvalid()

### Community 113 - "webhook_service.go"
Cohesion: 0.14
Nodes (13): event_records, idx_event_records_consumer, idx_event_records_created_at, idx_event_records_event, idx_event_records_expires_at, idx_event_records_tenant, idx_event_records_tenant_event_created, idx_event_records_tenant_ns_created (+5 more)

### Community 114 - "steps_jobs.py"
Cohesion: 0.26
Nodes (14): renderResult, applyFilterStage(), evaluateExpression(), evaluateToken(), EventContext, extractIfElseBlocks(), extractMatchingEnd(), goPrintf() (+6 more)

### Community 115 - "proposal.md"
Cohesion: 0.13
Nodes (14): API endpoints, CI and machine tokens, Consumer (portal) access, First run, From the API, From the CLI, From the web UI, Invite a teammate (+6 more)

### Community 116 - "WebhookClient"
Cohesion: 0.21
Nodes (5): WebhookClient, ExecOptions, TemplateEngine, hashTemplate(), sampleContext()

### Community 117 - "init_cmd.go"
Cohesion: 0.18
Nodes (12): configPath(), resolveConfig(), saveConfig(), TestResolveConfigDefaults(), TestResolveConfigPrecedence(), TestSaveConfigPermissions(), writeConfigFile(), isTerminal() (+4 more)

### Community 118 - "Mount"
Cohesion: 0.08
Nodes (26): main(), buildDraftRequest(), extendWriteDeadline(), TemplateDrafter, registerAIRoutes(), registerPromptRoute(), draftOverServer(), TestDraftTemplate_OutlivesServerWriteTimeout() (+18 more)

### Community 119 - "portal-embedding.mdx"
Cohesion: 0.11
Nodes (19): registerEventRoutes(), toEventTypeChange(), toEventTypeItem(), toEventTypeOutput(), toEventTypeVersionItem(), newPagination(), ValidateJSONSchema(), eventTypeChange (+11 more)

### Community 121 - "design.md"
Cohesion: 0.14
Nodes (14): AGENTS.md — Sparrow, Architecture, Authentication, DB / Migrations, Design Principles, First reads, Frontend (Svelte 5), graphify (+6 more)

### Community 124 - "Config"
Cohesion: 0.15
Nodes (8): Config, validatePort(), mustCIDRs(), ParseNetworks(), TestNetworkPolicy(), TestNetworkPolicyHostnames(), TestParseNetworks(), TestValidateWebhookURLPolicies()

### Community 125 - "Wrapf"
Cohesion: 0.19
Nodes (11): idx_tenants_slug, idx_tenants_status, tenants, update_tenants_updated_at, consumers, idx_consumers_tenant_id, consumers, idx_consumers_tenant_id (+3 more)

### Community 126 - "Sparrow Webhook Delivery Platform"
Cohesion: 0.14
Nodes (14): dependencies, astro, astro-mermaid, astro-og-canvas, @astrojs/sitemap, @astrojs/starlight, @astrojs/svelte, marked (+6 more)

### Community 127 - "scripts"
Cohesion: 0.14
Nodes (13): Day one: a tool that cannot fail loudly, Deriving health instead of maintaining it, Further reading, Measuring the thing that matters, Run it yourself, The numbers, side by side, Three things that went wrong on the way, Wall four: one busy webhook (+5 more)

### Community 128 - "models_test.go"
Cohesion: 0.14
Nodes (13): Does it depend on AI?, Does it tie us to Sparrow's way of doing things?, Does this help with HIPAA, GDPR, SOC 2, PCI?, How do our customers know a webhook really came from us?, How do we know it is reliable?, Is the stack open all the way down?, The short version, What does it cost? (+5 more)

### Community 129 - "entryToSimpleMarkdown.ts"
Cohesion: 0.14
Nodes (4): htmlToMarkdownPipeline, minify, minifyDefaults, selectors

### Community 130 - "Envelope"
Cohesion: 0.43
Nodes (4): emailSink, templateContext(), otlpPayload(), Envelope

### Community 131 - "EventRegistration"
Cohesion: 0.14
Nodes (13): ./.svelte-kit/tsconfig.json, compilerOptions, allowJs, checkJs, esModuleInterop, forceConsistentCasingInFileNames, moduleResolution, resolveJsonModule (+5 more)

### Community 132 - "provider_openai.go"
Cohesion: 0.37
Nodes (13): fakeServer(), runCLI(), TestAccessCommandsSurfaceServerErrors(), TestInviteBuildsLinkOnUIURL(), TestInviteDefaultsToServerURLAndJSON(), TestInvitesListAndCancel(), TestInviteTokenNeverExpires(), TestTokensCreate() (+5 more)

### Community 133 - "PrepareDeliveryRequest"
Cohesion: 0.15
Nodes (12): Access and authentication, Deployment, Example Ingress (internal), Kubernetes manifests, Network exposure, NetworkPolicy, Production checklist, Required configuration (+4 more)

### Community 134 - "Wrapf"
Cohesion: 0.12
Nodes (15): anthropicClient, completer, completion, turn, ServiceError, mapStatus(), mapTransportError(), newAnthropicClient() (+7 more)

### Community 135 - "Documenting the REST API"
Cohesion: 0.33
Nodes (4): Defaults that live in the service, Documenting the REST API, Where each part of the reference comes from, Writing descriptions

### Community 136 - "buildVectors"
Cohesion: 0.17
Nodes (10): event_registrations, idx_event_registrations_created_at, idx_event_registrations_name, update_event_registrations_updated_at, update_updated_at_column(), update_webhook_registrations_updated_at, idx_event_registrations_tenant, idx_event_registrations_tenant_name (+2 more)

### Community 137 - "Recipe"
Cohesion: 0.25
Nodes (8): findRecipe(), Subscription, Webhook, listRecipesOutput, ConsumerHint, Param, Recipe, recipeSummary

### Community 138 - "Sparrow threat model"
Cohesion: 0.25
Nodes (8): How to exercise it, Out of scope / leave alone, Principals and trust, Reports, Security properties, Severity, Sparrow threat model, Where untrusted input enters

### Community 140 - "Handler"
Cohesion: 0.32
Nodes (8): otlpSink, Config, Handler(), logsURL(), newOTLPSink(), TestLogsURL(), TestOTLPSinkDeliver(), TestOTLPSinkDeliverDownstreamFailure()

### Community 141 - "Sparrow Recipes"
Cohesion: 0.29
Nodes (6): Contributing a recipe, Guidance, How params work, Included recipes, Requiring the transform, Sparrow Recipes

### Community 142 - "vite-plugin-devtools-json"
Cohesion: 0.44
Nodes (11): runCLICmd(), startBundleServer(), TestEventsExport(), TestEventsExportNeedsOneSelector(), TestEventsImport_Applied(), TestEventsImport_BlockedExitsNonZero(), TestEventsImport_DryRun(), TestEventsImport_Stdin() (+3 more)

### Community 143 - "Client"
Cohesion: 0.48
Nodes (3): NewClient(), TestClientAutoCreatesEventType(), Client

### Community 144 - "Sparrow Web Dashboard"
Cohesion: 0.35
Nodes (10): SparrowVerify, decode_public_key(), decode_secret(), decode_signatures(), fetch_header(), parse_headers(), safe_hex_decode(), validate_timestamp() (+2 more)

### Community 146 - "steps_access.py"
Cohesion: 0.18
Nodes (10): Data retention and compliance posture, Hardening checklist, Not a fit, Option A — Authentik (username/password + Entra + Google in one tool), Option B — oauth2-proxy (single IdP, smallest footprint), Option C — Keycloak + oauth2-proxy (maximum boring), Recommended: SSO via an identity-aware proxy, The consumer portal (+2 more)

### Community 147 - "Setup"
Cohesion: 0.18
Nodes (10): Access tokens, API Authentication, Consumer Isolation, HTTP Hardening, Portal tokens, Production Config Checklist, Secret Masking in Responses, SSRF Protection (+2 more)

### Community 148 - "httpauth.go"
Cohesion: 0.17
Nodes (11): Before you start, Configuration, email, How a delivery is handled, and what Sparrow does with the answer, Install and run, otlp, Pick a sink, s3 (+3 more)

### Community 149 - "net/http.Header"
Cohesion: 0.36
Nodes (6): capturedWebhook, parseHeaders(), signedMessage(), VerifyEd25519(), VerifyHMAC(), Verifier

### Community 150 - "Contributing to Sparrow"
Cohesion: 0.33
Nodes (6): Commit Style, Contributing to Sparrow, Development Setup, Documenting the API, Notes, Release Process

### Community 151 - ".CheckIP"
Cohesion: 0.39
Nodes (4): embeddedIPv4(), isMetadataIP(), TestValidateIP(), ValidateIP()

### Community 152 - "playwright.config.ts"
Cohesion: 0.40
Nodes (8): decodeBase64Strict(), decodeSignatures(), DEFAULT_TOLERANCE_SECONDS, Headers, parseHeaders(), SignatureVerificationError, verifyEd25519(), verifyHmac()

### Community 153 - "parseDateFilter"
Cohesion: 0.44
Nodes (8): apply(), build(), button(), fit(), fitWidth(), open(), size(), zoom()

### Community 155 - "NewWebhookTemplateContext"
Cohesion: 0.20
Nodes (9): 1. Build the UI, 2. Point the UI at the server: `config.js`, 3. Serve it as a single-page app, 4. Configure the server, Authentication, Consumer portal, Local development, Troubleshooting (+1 more)

### Community 156 - "httpauth_test.go"
Cohesion: 0.20
Nodes (9): Before you start, Configuration, cron, github, Install and run, Pick a source, stripe, What the webhook server answers (+1 more)

### Community 158 - "Sparrow"
Cohesion: 0.29
Nodes (7): Contributing, Documentation, License, Quick start, Satellites, Sparrow, What you get

### Community 159 - "GetBuffer"
Cohesion: 0.20
Nodes (10): scripts, build, check, check:watch, dev, gen:api-types, prepare, preview (+2 more)

### Community 160 - "@astrojs/starlight"
Cohesion: 0.33
Nodes (3): buildDate, collections, @astrojs/starlight

### Community 162 - "TestJobInserter_InsertOpts_Merge"
Cohesion: 0.22
Nodes (8): Core Concepts, Deliveries, Events, Example: one event, two receivers, Key Design Decisions, Subscriptions, The Full Flow, Webhooks

### Community 163 - "TestWebhookWorkerNextRetry"
Cohesion: 0.22
Nodes (8): Deployment Flexibility, Detailed Comparison: Sparrow vs Svix, Feature Comparison, Important nuance, Sparrow vs the Landscape, What Sparrow Gives You, Where alternatives are stronger, Where Sparrow is stronger

### Community 164 - "Dual Webhook Signing (HMAC-SHA256 + Ed25519)"
Cohesion: 0.22
Nodes (8): Alternative: no Sparrow UI at all, Example: Caddy (simplest — start here), Example: nginx, How it works, end to end, Prerequisites, Security analysis, The route allowlist, What "forwarding" actually means

### Community 165 - "+layout.ts"
Cohesion: 0.22
Nodes (7): Example: Express (your app's backend as the proxy), Alternatives rejected, Context, Decisions, Requirement: Terminal delivery failure event, Scenario: Fired once when retries are exhausted, Scenario: Fired only on terminal failure

### Community 166 - "docs/src/lib/recipes.ts"
Cohesion: 0.22
Nodes (8): EventPreset, ORDER, parsed, rawFiles, RECIPE_META, RecipeDef, RecipeParam, RECIPES

### Community 168 - "runUse"
Cohesion: 0.22
Nodes (8): ADDED Requirements, Purpose, Requirement: Feedback-loop guard, Requirement: Internal `_sparrow` consumer, Requirement: Registered system event types, Scenario: Idempotent registration, Scenario: Internal channel failure does not recurse, Scenario: System events are invisible to tenants

### Community 169 - "time.Duration"
Cohesion: 0.18
Nodes (10): CreateInviteRequest, CreateTokenRequest, InviteTTL(), TestTTLRules(), TokenTTL(), NewTokenPurgeWorker(), ms(), TokenPurgeArgs (+2 more)

### Community 171 - "Deliberately NOT covered by e2e"
Cohesion: 0.40
Nodes (4): API surface deliberately not e2e-tested (removal / design candidates), Deliberately NOT covered by e2e, Known product-contract questions (asserted as-is, flagged), Out of e2e scope (belongs in unit/integration tests)

### Community 172 - "pages/index.astro"
Cohesion: 0.22
Nodes (8): consumer_pool, evt_ids, evt_names, hot_webhook, recent_delivery_ids, sub_ids, sub_lookup, wh_ids

### Community 173 - "tailwindcss"
Cohesion: 0.22
Nodes (8): Build, Configuration, Embedding in the Go Binary, Local Development, Prerequisites, Sparrow Web Dashboard, Standalone Deployment, Tech Stack

### Community 174 - "NewDocFetcher"
Cohesion: 0.40
Nodes (5): collapseSpace(), htmlToText(), looksLikeHTML(), NewDocFetcher(), TestDocFetcher_ExtractsTextAndRespectsPolicy()

### Community 175 - "fakeServer"
Cohesion: 0.22
Nodes (8): 2. Split the `sparrow` CLI into its own module to isolate its `go install` graph, Amendment (2026-10): sinks and sources are libraries, Consequences, Context, Decision, Releasing the split modules (tag scheme), Trigger to revisit, When this is worth it

### Community 176 - "astro"
Cohesion: 0.25
Nodes (7): Build, Build from Source, Docker Compose (Recommended), Docker Image, Pre-built Binaries, Prerequisites, Run

### Community 177 - "schema_compat.go"
Cohesion: 0.25
Nodes (7): Breaking changes, Inferring a schema from real events, Reading the history, Reserved names, Retiring an event type, Unregistered event names, What creates a new version

### Community 178 - "production.md"
Cohesion: 0.33
Nodes (5): GetFunctionMap(), GetTemplateFunctions(), TestTitleFunc(), toNumber(), TemplateFunc

### Community 179 - "NewWithStore"
Cohesion: 0.21
Nodes (9): New(), NewWithStore(), Realm(), Sealer(), TestMasterKeyIsRootInDefaultTenant(), TestPgstoreConformance(), TestTenantTokenLifetimeDefaultsAndNeverExpires(), SecretSealer (+1 more)

### Community 180 - "clientTester"
Cohesion: 0.14
Nodes (13): BacklogSample, ClientConfig, ClientResults, clientTester, E2EConfig, E2EResults, LatencyReservoir, LatencySummary (+5 more)

### Community 181 - "build.go"
Cohesion: 0.29
Nodes (7): 3.1 `sparrow-bus`: message-bus bridge, both directions, 3.2 `sparrow-cdc`: Postgres change capture as events, 3.3 `sparrow mcp`: MCP server for coding agents and operators, 3.4 `sparrow-inbox`: durable consumer-side receiver, 3.5 `sparrow-capture`: a self-hosted webhook.site, 3.6 Sparrow-to-Sparrow federation, 3. New satellites

### Community 182 - "svelte.config.js"
Cohesion: 0.40
Nodes (3): @sveltejs/adapter-static, @sveltejs/vite-plugin-svelte, config

### Community 183 - "deliveries/+page.svelte"
Cohesion: 0.29
Nodes (6): API, Subscriptions whose template fails, The bundle file, The preview, What an import does, When nothing is written

### Community 184 - "aiRouter"
Cohesion: 0.29
Nodes (6): Automatic disabling, Check that alerts are being sent, Good to know, How it works, Register an alert config, Set up the email delivery channel

### Community 187 - "@sveltejs/vite-plugin-svelte"
Cohesion: 0.29
Nodes (6): Capabilities, Impact, Modified Capabilities, New Capabilities, What Changes, Why

### Community 188 - "SparrowVerify"
Cohesion: 0.43
Nodes (5): test-ui.sh script, SPARROW_BASE_URL, start_server(), wait_for_static(), wait_healthy()

### Community 190 - "runPush"
Cohesion: 0.22
Nodes (6): parseJSONArg(), TestParseJSONArg(), pushWithAutoCreate(), runPush(), runTemplateTest(), kvFlag

### Community 191 - "Code conventions"
Cohesion: 0.67
Nodes (3): Code conventions, Handler pattern, Repository / Storage pattern

### Community 192 - "dev-env.sh"
Cohesion: 0.33
Nodes (5): RFC-3339, components, $defs, operations, paths

### Community 193 - "static-server.mjs"
Cohesion: 0.33
Nodes (5): Context, Design, Diagrams, Feature: Webhook health/delivery-failure email alerts, Open Questions

### Community 194 - "Store"
Cohesion: 0.33
Nodes (5): MIT, and meant that way, Simple to run does not mean simple-minded, The whole thing fits in your head, Try it in five minutes, Your data stays home

### Community 195 - "Status"
Cohesion: 0.33
Nodes (5): Development (Build from Source), Docker Image, Observability, Quick Start, Trying authentication locally

### Community 196 - "config.js"
Cohesion: 0.33
Nodes (5): Pausing from an import, Resuming, Seeing what was held, What a pause does, Where pause fits in a delivery

### Community 198 - "BatchCleanupWorker"
Cohesion: 0.33
Nodes (5): 1. Foundation, 2. Event emission, 3. Tenant-facing API, 4. Delivery channel, 5. Verification & docs

### Community 199 - ".VerifyEd25519"
Cohesion: 0.36
Nodes (8): github.com/sarathsp06/sparrow, github.com/sarathsp06/sparrow/pkg/access, github.com/sarathsp06/sparrow/pkg/signature, github.com/sarathsp06/sparrow/pkg/template, github.com/sarathsp06/sparrow/satellites/recipes, github.com/sarathsp06/sparrow/satellites/sparrow, github.com/sarathsp06/sparrow/satellites/sparrow-sinks, github.com/sarathsp06/sparrow/satellites/sparrow-sources

### Community 200 - "ReadBody"
Cohesion: 0.17
Nodes (10): ReadBody(), TestReadBody(), TestReadBodyNil(), TestReadBodyNilBody(), TestSendReusesConnections(), GetBuffer(), PutBuffer(), BenchmarkBufferPool() (+2 more)

### Community 201 - "SparrowVerify"
Cohesion: 0.40
Nodes (4): Go / TypeScript, Python, Signature verification helpers (`verify/`, hand-written), Sparrow Client Libraries

### Community 202 - "SparrowVerify"
Cohesion: 0.70
Nodes (4): access_invites, access_tokens, idx_access_invites_realm, idx_access_tokens_realm

### Community 203 - "Store"
Cohesion: 0.70
Nodes (4): idx_webhook_health_summaries_unique, idx_webhook_health_summaries_webhook_id, idx_webhook_health_summaries_window, webhook_health_summaries

### Community 205 - "newRESTClient"
Cohesion: 0.40
Nodes (4): Before you start, If you see something else, Next steps, What happened

### Community 206 - "verify-signatures.mdx"
Cohesion: 0.40
Nodes (4): Get the helper and verify, Testing your receiver, What gets signed, Where the secret or public key comes from

### Community 207 - "Mount"
Cohesion: 0.40
Nodes (4): Classification Logic, Error Categories, Monitoring Errors, Retry Behavior

### Community 208 - "schema-infer.ts"
Cohesion: 0.40
Nodes (4): 90-second quickstart, Configuration, Install, Worked example: develop a receiver on your laptop

### Community 211 - "TestVectors"
Cohesion: 0.70
Nodes (4): access_invites, access_tokens, idx_access_invites_realm, idx_access_tokens_realm

### Community 212 - "runPush"
Cohesion: 0.80
Nodes (4): fail(), need(), install.sh script, usage()

### Community 213 - "importBundle"
Cohesion: 0.80
Nodes (4): release_leaf(), release_module(), release-submodules.sh script, tag_exists()

### Community 214 - "sealedRouter"
Cohesion: 0.67
Nodes (3): idx_webhook_health_buckets_start, webhook_health_buckets, webhook_health_evaluator

### Community 216 - "All"
Cohesion: 0.50
Nodes (4): `sparrow tokens`, `sparrow tokens create`, `sparrow tokens list`, `sparrow tokens revoke`

### Community 217 - "Context"
Cohesion: 0.50
Nodes (3): Example: Stripe payments, end to end, The contract, The satellites

### Community 219 - "TestE2E_EmailSink"
Cohesion: 0.50
Nodes (4): Requirement: Health transition event, Scenario: Degradation emits one event, Scenario: No event without a transition, Scenario: Recovery emits one event

### Community 220 - "listenHandler"
Cohesion: 0.40
Nodes (5): scripts, astro, build, dev, preview

## Knowledge Gaps
- **31 isolated node(s):** `InvalidTimestamp`, `TimestampOutsideTolerance`, `EmptySecret`, `InvalidSecretBase64`, `InvalidPublicKey` (+26 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 1758 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **109 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Use it as a Go library` connect `services.ts` to `httpauth_test.go`?**
  _High betweenness centrality (0.049) - this node is a cross-community bridge._
- **What connects `InvalidTimestamp`, `TimestampOutsideTolerance`, `EmptySecret` to the rest of the system?**
  _31 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `github.com/google/uuid.UUID` be split into smaller, more focused modules?**
  _Cohesion score 0.032992930086410056 - nodes in this community are weakly interconnected._
- **Why does `Configuration` connect `httpauth_test.go` to `+layout.ts`?**
  _High betweenness centrality (0.048) - this node is a cross-community bridge._
- **Should `context.Context` be split into smaller, more focused modules?**
  _Cohesion score 0.03104970801880074 - nodes in this community are weakly interconnected._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.02156997931645819 - nodes in this community are weakly interconnected._
- **Should `go_pkg_context` be split into smaller, more focused modules?**
  _Cohesion score 0.08156988188976377 - nodes in this community are weakly interconnected._