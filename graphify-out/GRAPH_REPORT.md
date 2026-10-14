# Graph Report - optimistic-dijkstra-2542b1  (2026-10-14)

## Corpus Check
- 545 files · ~490,806 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 46 file(s) not represented in the graph (top: .spec 20, (none) 10, .ttf 6)

## Summary
- 5331 nodes · 15053 edges · 305 communities (200 shown, 105 thin omitted)
- Extraction: 94% EXTRACTED · 6% INFERRED · 0% AMBIGUOUS · INFERRED: 867 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `08af9c37`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- github.com/google/uuid.UUID
- context.Context
- testing.T
- ref_lib
- [webhookId]/+page.svelte
- go_pkg_testing
- go_pkg_context
- sparrow-sinks/main_test.go
- log/slog.Logger
- EventTypeDefinition
- setupEnv
- webhooks/register/+page.svelte
- ref_app
- testContext
- EventRegistration
- otel.go
- EventSubscription
- newSvc
- steps.py
- lib/components/TemplateEditor.svelte
- requests
- encoding/json.RawMessage
- NewService
- webhooks/+page.svelte
- restClient
- io.Writer
- lib/components/AlertConfigs.svelte
- posts.ts
- RecipeWorkbenchApp.svelte
- lib/components/SubscriptionForm.svelte
- crypto.go
- Repository
- go_pkg_os
- SparrowVerify.java
- Manager
- otel_test.go
- [eventId]/+page.svelte
- push/+page.svelte
- update/+page.svelte
- portal/+page.svelte
- EventRecord
- Service
- PortalGateway
- 000001_consolidated_schema.up.sql
- mcp_test.go
- events/+page.svelte
- generator.ts
- $lib/api-types
- lib/components/EventTypeImport.svelte
- access/+page.svelte
- hooks.py
- reports/+page.svelte
- Sparrow — Detailed Flow Reference
- BatchJob
- newRootCmd
- time.Time
- lib/components/SubscriptionManager.svelte
- sparrow/access.go
- sparrow/client.go
- webhook_target.py
- listDeliveriesImpl
- services.ts
- New
- email.go
- go_pkg_strings
- ai/ai_test.go
- WebhookRegistration
- web/package.json
- client.ts
- ai/ai.go
- sparrow_verify.py
- webhook_registrations
- Checker
- prompt.go
- newAuth
- sparrow_verify.rs
- docs/package.json
- template-fields.ts
- steps_jobs.py
- lib/components/SchemaFromSamples.svelte
- template-functions.md
- Decision
- All variables
- registerHealthRoutes
- api.astro
- Setup
- SparrowAPI
- schema-infer.ts
- architecture.mdx
- compose.ts
- steps_access.py
- api.ts
- net/http.Handler
- NewTemplateEngine
- Mount
- migrate.go
- WebhookHTTPConfig
- Request
- Satellite ideas: landscape research and candidates
- Commands
- lib/utils.ts
- ADDED Requirements
- deliveries/+page.svelte
- listEventOccurrencesImpl
- security_headers_test.go
- payload-transformation.mdx
- recipes.mdx
- WriteError
- ref_astro_content
- devDependencies
- batch-job.svelte.ts
- createOnlyService
- event_records
- go-template-engine.ts
- access.md
- recipes_test.go
- runInit
- WebhookServiceInterface
- registerEventRoutes
- .Export
- AGENTS.md — Sparrow
- CursorPages
- registerWebhookRoutes
- NetworkPolicy
- tenants
- dependencies
- benchmark-four-walls.md
- for-decision-makers.mdx
- entryToSimpleMarkdown.ts
- compilerOptions
- sparrow/access_test.go
- production.md
- Wrapf
- Contributing to Sparrow
- event_registrations
- Sparrow threat model
- main
- Sparrow Recipes
- events_bundle_test.go
- SparrowVerify
- SparrowVerify
- deployment/security.mdx
- reference/security.mdx
- sinks.mdx
- sinks_test.go
- system.ts
- auth.svelte.ts
- sparrow-verify.ts
- diagram-viewer.js
- BlogLayout.astro
- separate-ui.md
- sources.mdx
- SparrowEnvironment
- Sparrow
- scripts
- SignatureVerificationException
- Core Concepts
- Feature Comparison
- portal-embedding.mdx
- .url
- docs/src/lib/recipes.ts
- ADDED Requirements
- time.Duration
- palette
- mcp.go
- seed.sql
- Sparrow Web Dashboard
- 2. Split the `sparrow` CLI into its own module to isolate its `go install` graph
- installation.mdx
- event-type-versioning.mdx
- lib/components/BatchProgress.svelte
- sealedRouter
- time.Duration
- 3. New satellites
- apiConsole.svelte.ts
- What an import does
- webhook-health-alerts.mdx
- cron.go
- WebhookService
- proposal.md
- test-ui.sh
- emitHealthChangedEvent
- api-types.d.ts
- Feature: Webhook health/delivery-failure email alerts
- introducing-sparrow.md
- docker-compose.md
- subscription-pause.mdx
- lib/components/TransformSettings.svelte
- tasks.md
- github.com/sarathsp06/sparrow
- client_test.go
- Sparrow Client Libraries
- 000027_access_tokens.up.sql
- webhook_health_summaries
- quickstart.mdx
- verify-signatures.mdx
- error-classification.md
- cli.mdx
- schema.sql
- install.sh
- release-submodules.sh
- 000037_health_evaluator.up.sql
- LayeredArchEmbed.astro
- `sparrow tokens`
- index.mdx
- timeoutError
- Requirement: Health transition event
- scripts
- webhook_alert_configs
- idx_access_tokens_idempotency
- subscription_template_versions
- client-libraries.mdx
- docs/tsconfig.json
- manifest.json
- json
- lib/components/HealthBadge.svelte
- opencode.json
- idx_access_tokens_idempotency
- dev-env.sh
- lib/components/FloatingAction.svelte
- +layout.ts
- idx_webhook_deliveries_request_body_gin
- 000015_envelope_encryption.up.sql
- 000019_drop_system_settings.down.sql
- idx_webhook_deliveries_paused
- aiRequestBody
- scheduleRender
- run.sh
- b64
- sparrow-e2e

## God Nodes (most connected - your core abstractions)
1. `RepositoryInterfaceWithTracing` - 79 edges
2. `EventSubscription` - 70 edges
3. `setupEnv()` - 68 edges
4. `newRESTClient()` - 54 edges
5. `WebhookServiceInterfaceWithTracing` - 51 edges
6. `EventRegistration` - 48 edges
7. `testContext()` - 42 edges
8. `NewWebhookService()` - 39 edges
9. `main()` - 38 edges
10. `Service` - 38 edges

## Surprising Connections (you probably didn't know these)
- ``sparrow mcp`` --references--> `delete_webhook()`  [INFERRED]
  docs/src/content/docs/satellites/cli.mdx → e2e/step_impl/steps_surface.py
- `1. What the landscape ships as "ecosystem"` --references--> `webhooks`  [INFERRED]
  docs/design/satellite-ideas.md → web/src/lib/api-types.d.ts
- `2.4 CLI` --references--> `webhooks`  [INFERRED]
  docs/design/satellite-ideas.md → web/src/lib/api-types.d.ts
- `4.2 Typed event payloads from event-type schemas` --references--> `webhooks`  [INFERRED]
  docs/design/satellite-ideas.md → web/src/lib/api-types.d.ts
- `Package Structure` --references--> `webhooks`  [INFERRED]
  docs/src/content/docs/reference/architecture.mdx → web/src/lib/api-types.d.ts

## Import Cycles
- None detected.

## Communities (305 total, 105 thin omitted)

### Community 0 - "github.com/google/uuid.UUID"
Cohesion: 0.03
Nodes (18): statusForFailure(), deliveryFilterArgs(), DeliveryRepository, Repository, AutoDisableResult, HealthRepository, Repository, cursorArgs() (+10 more)

### Community 1 - "context.Context"
Cohesion: 0.03
Nodes (19): AlertConfig, AlertConfigRepository, Repository, Repository, AlertConfigManager, BatchManager, DeliveryManager, EventManager (+11 more)

### Community 2 - "testing.T"
Cohesion: 0.02
Nodes (125): vector, vectorFile, ErrorCategory, sinkSMTPMessage, TestMigrationsMatchPgstoreSchema(), TestAIConfig(), TestSlogLevel(), TestUIInjectKeyIsRetired() (+117 more)

### Community 3 - "ref_lib"
Cohesion: 0.05
Nodes (28): signingOut, who, copiedId, METHOD_TONE, { open = $bindable(false) }, {
    consumer,
    count,
    filter,
    onclose,
  }, error, retrying (+20 more)

### Community 4 - "[webhookId]/+page.svelte"
Cohesion: 0.03
Nodes (58): activeTab, applyDeliveryFilters(), changingSignature, clearDeliveryFilters(), configForm, configHeaderKey, configHeaderValue, configSecretHeaderKey (+50 more)

### Community 5 - "go_pkg_testing"
Cohesion: 0.12
Nodes (4): subscriptionResp, firstLine(), TestFirstLineSkipsHeadings(), eventTypeVersion

### Community 6 - "go_pkg_context"
Cohesion: 0.06
Nodes (124): oaMessage, oaRequest, oaResponse, WebhookEnvelope, eventTypeResp, eventTypeVersionResp, collapseSpace(), htmlToText() (+116 more)

### Community 7 - "sparrow-sinks/main_test.go"
Cohesion: 0.10
Nodes (29): otlpSink, s3Sink, inheritVerification(), loadConfig(), main(), newHandler(), run(), signEd25519Headers() (+21 more)

### Community 8 - "log/slog.Logger"
Cohesion: 0.05
Nodes (37): DeliveryRequest, RunAllMigrations(), RunAppMigrations(), RunRiverMigrations(), RedactURL(), BuildEnvelopePayload(), PrepareDeliveryRequest(), failing() (+29 more)

### Community 9 - "EventTypeDefinition"
Cohesion: 0.07
Nodes (39): BundleDigest(), checkStamp(), checkTemplates(), compileSchema(), generatePayload(), EventTypeImportResult, WebhookService, TestBundleDigest_IgnoresFormattingAndKeyOrder() (+31 more)

### Community 10 - "setupEnv"
Cohesion: 0.13
Nodes (58): consumerStats, deliveryItem, testEnv, buildCLI(), runCLI(), TestCLI_E2E(), TestCLI_EventTypeExportImport(), TestE2E_ConsumerDeliveryPaging() (+50 more)

### Community 11 - "webhooks/register/+page.svelte"
Cohesion: 0.03
Nodes (53): active, alertDeliveryConfigured, alertEmail, alertEmailError, alertError, alertsApply, allEvents, appliedRecipe (+45 more)

### Community 12 - "ref_app"
Cohesion: 0.08
Nodes (17): active, Blip, blips, { height = 30 }, Props, scatter(), toneVar, beat (+9 more)

### Community 13 - "testContext"
Cohesion: 0.07
Nodes (54): Status, TestAlertDeliveryConfigured(), TestCreateSubscription_CatchAllWithLabelFilters(), TestCreateSubscription_WithEmptyLabelFilters(), TestCreateSubscription_WithInvalidLabelFilters(), TestCreateSubscription_WithLabelFilters(), TestGetSubscriptionsWithWebhooksByEvent_ActiveWebhooksOnly(), TestGetSubscriptionsWithWebhooksByEvent_InactiveExcluded() (+46 more)

### Community 14 - "EventRegistration"
Cohesion: 0.05
Nodes (17): registerSystemEventTypes(), SystemEventRegistrations(), systemEventSamplePayload(), EventTypeFilter, EventTypeRepository, Repository, RepositoryInterface, EventRegistration (+9 more)

### Community 16 - "EventSubscription"
Cohesion: 0.05
Nodes (10): EventSubscription, SubscriptionTemplateVersion, SubscriptionWithWebhook, Repository, SubscriptionRepository, insertSubscription(), SubscriptionManager, TemplateFunctionInfo (+2 more)

### Community 17 - "newSvc"
Cohesion: 0.38
Nodes (7): RedeemHandler(), newSvc(), serve(), TestExtraVerifier(), TestMiddleware(), TestMiddlewareOutageIs503(), TestRedeemHandler()

### Community 18 - "steps.py"
Cohesion: 0.10
Nodes (48): delivery_has_signature_headers(), _api(), assert_api_delivery_count(), assert_body_field(), assert_body_has_key(), assert_body_not_has_key(), assert_delivery_attempt_count(), assert_delivery_count() (+40 more)

### Community 19 - "lib/components/TemplateEditor.svelte"
Cohesion: 0.04
Nodes (44): aiAttempts, aiCapability, aiConfirmReplace, aiDocsUrl, aiError, aiInstructions, aiLoading, aiNotes (+36 more)

### Community 20 - "requests"
Cohesion: 0.16
Nodes (45): alert_configs_count(), alert_configs_count_for_webhook(), also_use_consumer(), assert_delivery_page_meta(), _base(), create_alert_config_bad_webhook(), create_alert_config_scoped(), create_alert_config_wide() (+37 more)

### Community 21 - "encoding/json.RawMessage"
Cohesion: 0.13
Nodes (8): fakePusher, fakePusher, NewClient(), TestClientAutoCreatesEventType(), Client, fakePusher, fakePusher, fakePusher

### Community 22 - "NewService"
Cohesion: 0.11
Nodes (40): Key, TestPrepareDeliveryRequest(), TestPrepareDeliveryRequestFailsClosedOnDecryptError(), GenerateKey(), Keyring, NewKeyring(), NewService(), NewServiceFromKeyring() (+32 more)

### Community 23 - "webhooks/+page.svelte"
Cohesion: 0.05
Nodes (34): ConsumerStats, degradedWebhooks, error, HealthSummary, loading, rules, ruleText, scope (+26 more)

### Community 24 - "restClient"
Cohesion: 0.14
Nodes (26): bodyRecorder, bundle, importResult, requiresTransformSub, restClient, templateDelivery, exportBundle(), importBundle() (+18 more)

### Community 25 - "io.Writer"
Cohesion: 0.10
Nodes (28): newPalette(), importOutcomeError(), printImportResult(), runEventsExport(), runEventsImport(), runEventsVersions(), indentJSON(), printEventType() (+20 more)

### Community 26 - "lib/components/AlertConfigs.svelte"
Cohesion: 0.08
Nodes (20): adding, AlertConfigItem, configs, { consumer, webhookId }, deliveryConfigured, email, error, eventLabel (+12 more)

### Community 27 - "posts.ts"
Cohesion: 0.09
Nodes (30): links, links, Props, Props, tags, Props, allTags(), base (+22 more)

### Community 28 - "RecipeWorkbenchApp.svelte"
Cohesion: 0.05
Nodes (36): activeRecipe, attempt, chips, cliCommand, copiedToast, DEST, drafts, emailAddresses (+28 more)

### Community 29 - "lib/components/SubscriptionForm.svelte"
Cohesion: 0.05
Nodes (29): {
    label,
    summary = "",
    open = $bindable(false),
    children,
  }, { icon = "inbox", title, description, action }, paths, Props, {
    expanded = $bindable(false),
    label = "Editor",
    children,
  }, availableEvents, backHref, catchAll (+21 more)

### Community 30 - "crypto.go"
Cohesion: 0.14
Nodes (8): cryptoSealer, kek, buildKEK(), decryptDataWithDEK(), Service, isSafeKeyID(), newAEAD(), unwrapDEK()

### Community 31 - "Repository"
Cohesion: 0.08
Nodes (9): DeliveryAttempt, Repository, checkWebhookDuplicate(), Repository, WebhookRepository, insertWebhookRegistration(), DB, DBTX (+1 more)

### Community 32 - "go_pkg_os"
Cohesion: 0.08
Nodes (29): configPath(), resolveConfig(), saveConfig(), TestResolveConfigDefaults(), TestResolveConfigPrecedence(), TestSaveConfigPermissions(), writeConfigFile(), parseJSONArg() (+21 more)

### Community 33 - "SparrowVerify.java"
Cohesion: 0.11
Nodes (5): SignatureVerificationException, SparrowVerify, ParsedHeaders, SignatureVerificationException, SparrowVerify

### Community 34 - "Manager"
Cohesion: 0.12
Nodes (7): NewJobInserterWithTracing(), jobInserter, NewJobInserter(), Manager, JobInserterWithTracing, noopJobInserter, mockJobInserter

### Community 35 - "otel_test.go"
Cohesion: 0.15
Nodes (14): DefaultConfig(), GetMeter(), GetTracer(), MetricsHandler(), exportOneSpan(), startGRPCCollector(), startHTTPCollector(), TestSetupExportsSpans() (+6 more)

### Community 36 - "[eventId]/+page.svelte"
Cohesion: 0.08
Nodes (17): { pages, shown, onchange, itemLabel = "items" }, Props, consumer, deliveries, deliveriesError, deliveriesLoading, DeliveryItem, error (+9 more)

### Community 37 - "push/+page.svelte"
Cohesion: 0.06
Nodes (29): availableEvents, consumer, error, event, EventTypeItem, hasSchema(), labels, loading (+21 more)

### Community 38 - "update/+page.svelte"
Cohesion: 0.09
Nodes (21): active, confirmPending(), description, error, ImportItem, loading, metadata, name (+13 more)

### Community 39 - "portal/+page.svelte"
Cohesion: 0.05
Nodes (39): API surface deliberately not e2e-tested (removal / design candidates), Deliberately NOT covered by e2e, Known product-contract questions (asserted as-is, flagged), Out of e2e scope (belongs in unit/integration tests), alertDeliveryConfigured, confirmDeleteId, deliveries, deliveriesLoading (+31 more)

### Community 40 - "EventRecord"
Cohesion: 0.08
Nodes (13): NewEventProcessingWorker(), NewRetentionWorker(), eventReportFilterArgs(), eventVersionOrDefault(), EventRepository, Repository, EventRecord, EventReportFilter (+5 more)

### Community 41 - "Service"
Cohesion: 0.12
Nodes (13): cacheEntry, Config, CreateInviteRequest, CreateTokenRequest, RootKey, Principal, invalidRequest(), clonePtr() (+5 more)

### Community 42 - "PortalGateway"
Cohesion: 0.21
Nodes (17): NewPortalVerifier(), PortalAuthorized(), PortalGateway(), doBearer(), mintConsumerToken(), newPortalGateway(), newPortalService(), TestPortalGatewayAcceptsConsumerAccessTokens() (+9 more)

### Community 43 - "000001_consolidated_schema.up.sql"
Cohesion: 0.10
Nodes (29): idx_webhook_deliveries_created_at, idx_webhook_deliveries_event_id, idx_webhook_deliveries_expires_at, idx_webhook_deliveries_request_body_gin, idx_webhook_deliveries_status, idx_webhook_deliveries_subscription_id, idx_webhook_deliveries_webhook_id, idx_webhook_health_events_delivery_id (+21 more)

### Community 44 - "mcp_test.go"
Cohesion: 0.23
Nodes (14): newAPIClient(), newMCPServer(), call(), connectMCP(), newFakeSparrow(), TestMCPListenRoundTrip(), TestMCPPushAutoCreatesEventType(), TestMCPReadOnlyHidesWriteTools() (+6 more)

### Community 45 - "events/+page.svelte"
Cohesion: 0.07
Nodes (26): allVisibleSelected, closeModal(), confirmToggle, content, currentPage, error, events, eventToToggle (+18 more)

### Community 46 - "generator.ts"
Cohesion: 0.10
Nodes (20): entryToSimpleMarkdown(), collator, generateLlmsTxt(), GET(), getStaticPaths, Params, prerender, Props (+12 more)

### Community 47 - "$lib/api-types"
Cohesion: 0.04
Nodes (34): AttemptItem, attempts, { deliveryId }, error, loading, Props, copied, display (+26 more)

### Community 48 - "lib/components/EventTypeImport.svelte"
Cohesion: 0.07
Nodes (29): acknowledged, actionLabel, actionTone, allAcknowledged, breakingConfirmed, breakingItems, bundle, busy (+21 more)

### Community 49 - "access/+page.svelte"
Cohesion: 0.07
Nodes (27): cancelInvite(), cancelling, consumer, copied, create(), createError, creating, dialog (+19 more)

### Community 50 - "hooks.py"
Cohesion: 0.09
Nodes (10): setup_environment(), setup_scenario(), teardown_environment(), teardown_scenario(), _get(), get_authed_correct_key(), get_authed_no_key(), get_authed_with_key() (+2 more)

### Community 51 - "reports/+page.svelte"
Cohesion: 0.10
Nodes (22): applyFilters(), clearFilters(), createdAfterFilter, createdBeforeFilter, currentEvent, error, eventName, EventOccurrenceItem (+14 more)

### Community 52 - "Sparrow — Detailed Flow Reference"
Cohesion: 0.06
Nodes (30): 1. Webhook Registration, 2. Event Registration, 3. Event Push (PushEvent), 4. Event Processing (EventProcessingWorker), 5. Webhook Delivery (WebhookWorker), 6. Subscription Creation, 7. Webhook Update (UpdateWebhookConfig), DB Writes (+22 more)

### Community 53 - "BatchJob"
Cohesion: 0.08
Nodes (10): NewBatchJobWorker(), BatchRepository, Repository, BatchJob, BatchJobStatus, BatchJobType, WebhookService, BatchJobArgs (+2 more)

### Community 54 - "newRootCmd"
Cohesion: 0.20
Nodes (26): newInviteCmd(), newTokensCmd(), newTokensCreateCmd(), newTokensRevokeCmd(), addOutputFlag(), clientFromCmd(), newRootCmd(), newVersionCmd() (+18 more)

### Community 55 - "time.Time"
Cohesion: 0.05
Nodes (29): countingStore, Status, downStore, encodeCursor(), newCursorPagination(), FailingWebhook, HealthEvaluationRepository, Repository (+21 more)

### Community 56 - "lib/components/SubscriptionManager.svelte"
Cohesion: 0.09
Nodes (18): cancelBtn, confirmBtn, {
    open,
    title = "Confirm",
    message,
    confirmLabel = "Confirm",
    cancelLabel = "Cancel",
    variant = "danger",
    onconfirm,
    oncancel,
  }, Props, tone, confirmDeleteOpen, error, executeDelete() (+10 more)

### Community 57 - "sparrow/access.go"
Cohesion: 0.13
Nodes (16): accessLabel(), apiClient, newInvitesCmd(), newTokensListCmd(), parseDuration(), runInvite(), runTokensCreate(), TestParseDuration() (+8 more)

### Community 58 - "sparrow/client.go"
Cohesion: 0.12
Nodes (15): apiClient, pushWithAutoCreate(), runPush(), apiError, consumerStats, deliveryItem, eventTypeItem, pushBody (+7 more)

### Community 59 - "webhook_target.py"
Cohesion: 0.08
Nodes (4): CapturedDelivery, _Target, handle(), WebhookTargetManager

### Community 60 - "listDeliveriesImpl"
Cohesion: 0.27
Nodes (9): listDeliveriesImpl(), registerDeliveryRoutes(), toDeliveryItem(), toDeliveryOutput(), toBatchJobOutput(), batchJobOutput, deliveryItem, deliveryOutput (+1 more)

### Community 61 - "services.ts"
Cohesion: 0.11
Nodes (18): fragmentParam(), message, portalInvite, status, parsePortalLink(), PortalSession, apiHref(), basePath() (+10 more)

### Community 62 - "New"
Cohesion: 0.26
Nodes (21): clock, New(), newClock(), newSealedService(), newService(), ptr(), reason(), TestCacheAndTouchThrottling() (+13 more)

### Community 63 - "email.go"
Cohesion: 0.16
Nodes (17): emailSink, SMTPConfig, smtpMessage, Config, Handler(), newEmailSink(), templateContext(), serveSMTP() (+9 more)

### Community 64 - "go_pkg_strings"
Cohesion: 0.08
Nodes (43): AuthError, ctxKey, ErrorBody, RedeemRequest, RedeemResponse, InviteTTL(), TestTTLRules(), TestValidateConsumer() (+35 more)

### Community 65 - "ai/ai_test.go"
Cohesion: 0.35
Nodes (10): fakeModel, newDrafter(), TestDraftTemplate_DocsURL(), TestDraftTemplate_GivesUpAfterMaxAttempts(), TestDraftTemplate_RefusalIsFailedPrecondition(), TestDraftTemplate_RendersAgainstRequestPayload(), TestDraftTemplate_RepairsSilentlyWrongField(), TestDraftTemplate_RepairsUntilItRenders() (+2 more)

### Community 66 - "WebhookRegistration"
Cohesion: 0.04
Nodes (34): getWebhookEventsMap(), mockRepo, pauseFailing(), breakingChangeError(), checkEventForConsumer(), EventTypeSaveResult, reservedEventNameError(), DerefBoolOr() (+26 more)

### Community 67 - "web/package.json"
Cohesion: 0.09
Nodes (21): openapi-typescript, svelte-check, @sveltejs/adapter-static, @sveltejs/kit, @sveltejs/vite-plugin-svelte, tailwindcss, @tailwindcss/forms, @tailwindcss/vite (+13 more)

### Community 68 - "client.ts"
Cohesion: 0.13
Nodes (20): auth, browserTokenName(), exchangeKey(), ExchangeResult, InviteError, RedeemedInvite, redeemInvite(), rejectMessage() (+12 more)

### Community 69 - "ai/ai.go"
Cohesion: 0.21
Nodes (12): Config, draft, Drafter, Provider, Renderer, DocFetcher, New(), parseDraft() (+4 more)

### Community 70 - "sparrow_verify.py"
Cohesion: 0.15
Nodes (9): _check_timestamp(), _decode_signatures(), _required_headers(), SignatureVerificationError, verify_ed25519(), verify_hmac(), _signed_body(), verify_ed25519_signature() (+1 more)

### Community 71 - "webhook_registrations"
Cohesion: 0.10
Nodes (18): event_subscriptions, idx_event_subscriptions_event, idx_webhook_registrations_active, idx_webhook_registrations_consumer, idx_webhook_registrations_health, webhook_registrations, idx_event_records_tenant_consumer, idx_event_subscriptions_tenant (+10 more)

### Community 72 - "Checker"
Cohesion: 0.18
Nodes (8): fakeOpenAI, Checker, HealthResponse, ReadyResponse, newOpenAIDrafter(), TestOpenAIProvider_AuthAndErrors(), TestOpenAIProvider_DraftsAndRepairs(), NewChecker()

### Community 73 - "prompt.go"
Cohesion: 0.15
Nodes (21): HelperFunc, outlineField, PromptBuilder, RecipeParam, fieldStep(), Recipe, mustJSON(), NewPromptBuilder() (+13 more)

### Community 74 - "newAuth"
Cohesion: 0.12
Nodes (18): redactSpanURL, Verifier, NewAuth(), apiKey(), bearer(), guarded(), newAuth(), TestAPIKeyHTTPMiddlewareAcceptsHeader() (+10 more)

### Community 75 - "sparrow_verify.rs"
Cohesion: 0.15
Nodes (17): DEFAULT_TOLERANCE_SECS, hex_decode(), hex_nibble(), parse_headers(), SignatureError, EmptySecret, InvalidPublicKey, InvalidSecretBase64 (+9 more)

### Community 76 - "docs/package.json"
Cohesion: 0.11
Nodes (18): blogLastmod, devDependencies, starlight-llms-txt, svelte, name, private, type, version (+10 more)

### Community 77 - "template-fields.ts"
Cohesion: 0.19
Nodes (23): editField(), actionBody(), actionEnd(), balanced(), decodeAction(), decodeJsonString(), decodeUrlValue(), encodeText() (+15 more)

### Community 78 - "steps_jobs.py"
Cohesion: 0.22
Nodes (20): assert_all_terminal_status(), assert_bulk_retry_count(), assert_consumer_stats(), assert_consumer_stats_webhooks(), assert_global_stats(), assert_health_failed_count(), assert_health_summary(), assert_job_progress() (+12 more)

### Community 79 - "lib/components/SchemaFromSamples.svelte"
Cohesion: 0.09
Nodes (19): coverage, error, { eventName, sampleId = '', hasSchema, ongenerate }, EventOccurrenceItem, eventsError, generate(), loadEvents(), loadingEvents (+11 more)

### Community 80 - "template-functions.md"
Cohesion: 0.09
Nodes (21): 1. Payload transform engine: Go `text/template`, not embedded JavaScript, Consequences, Context, Decision, Trigger to revisit, A complete template, Building JSON, default (+13 more)

### Community 81 - "Decision"
Cohesion: 0.09
Nodes (22): 1. Reusable library: `pkg/access`, 2. Sparrow adapter: `internal/accessauth`, 3. Replace the single shared API key with revocable access tokens and one-time invites, 3. Two scopes only — no roles, 4. Stored invites instead of signed links, 5. No cascade revocation, 6. Tokens independent of master key, 7. 503, not 401, when the database is down (+14 more)

### Community 82 - "All variables"
Cohesion: 0.09
Nodes (22): Access, AI drafting, AI template drafting, All variables, An OpenAI-compatible server, Anthropic, Database Pools, Default Tenant (+14 more)

### Community 83 - "registerHealthRoutes"
Cohesion: 0.15
Nodes (14): registerHealthRoutes(), newPagination(), listSubscriptionsImpl(), registerSubscriptionRoutes(), toSubscriptionItem(), toSubscriptionOutput(), listEventTypesOutput, listSubscriptionsOutput (+6 more)

### Community 84 - "api.astro"
Cohesion: 0.10
Nodes (10): out, items, Props, toc, spec, specPath, @scalar/astro, sharp (+2 more)

### Community 85 - "Setup"
Cohesion: 0.33
Nodes (6): legacyEndpoint(), newLoggerProvider(), Setup(), setupMetrics(), setupTracing(), Config

### Community 86 - "SparrowAPI"
Cohesion: 0.15
Nodes (4): Authentication, `sparrow mcp`, SparrowAPI, _to_dict()

### Community 87 - "schema-infer.ts"
Cohesion: 0.20
Nodes (15): analyzeSamples(), collect(), DATE, detectStringFormat(), emit(), FieldCoverage, inferSchema(), jsonToJsonSchema() (+7 more)

### Community 88 - "architecture.mdx"
Cohesion: 0.10
Nodes (19): Default Webhook Body, Design Principles, Domain Packages, Encryption at Rest, Event Processing Pipeline, Health State Machine, How It Works, HTTP Client Design (+11 more)

### Community 89 - "compose.ts"
Cohesion: 0.17
Nodes (19): DiscordCompose, emailArray(), EmailCompose, ENVELOPE_PATHS, generateDiscordTemplate(), generateNtfyTemplate(), generatePagerdutyTemplate(), generateSendgridTemplate() (+11 more)

### Community 90 - "steps_access.py"
Cohesion: 0.27
Nodes (17): _as_token(), cancel_invite(), create_consumer_token(), create_invite(), _create_token(), get_with_token(), get_with_token_rejected(), _invites() (+9 more)

### Community 91 - "api.ts"
Cohesion: 0.22
Nodes (10): @playwright/test, ensureAlertDelivery(), ensureEventType(), mintPortalToken(), newConsumer(), newEventName(), pushEvent(), registerWebhook() (+2 more)

### Community 92 - "net/http.Handler"
Cohesion: 0.21
Nodes (15): logCORSMode(), CORS(), CORSMode, NormalizeOrigins(), corsChain(), get(), preflight(), TestCORSAllowListSplitDeploymentWithAPIKey() (+7 more)

### Community 93 - "NewTemplateEngine"
Cohesion: 0.10
Nodes (31): BenchmarkSend(), GetHeaderMap(), PutHeaderMap(), BenchmarkHeaderMapPool(), TestHeaderMapPool(), TestRenderTemplatePreview_StrictFlag(), RenderTemplatePreview(), TestTransformPayloadWith_CachesStrictAndLenientSeparately() (+23 more)

### Community 94 - "Mount"
Cohesion: 0.12
Nodes (18): main(), aiRouter(), do(), TestCapabilities_ReflectsDrafter(), TestCapabilities_ReportsAlertDelivery(), TestDraftTemplate_DisabledIs503(), TestDraftTemplate_GroundsRequestAndReturnsDraft(), TestDraftTemplate_NotFound() (+10 more)

### Community 95 - "migrate.go"
Cohesion: 0.10
Nodes (7): main(), GetMigrationsFS(), computeInlineScriptHashes(), InlineScriptHashes(), inlineScriptHashesFromFS(), TestComputeInlineScriptHashes(), TestInlineScriptHashesFromFS()

### Community 96 - "WebhookHTTPConfig"
Cohesion: 0.26
Nodes (8): DefaultWebhookHTTPConfig(), WebhookHTTPConfig, TestDefaultWebhookHTTPConfig_RateLimitRPS(), TestValidateConfig_RateLimitRPS(), IntArray, JSONBMap, StringArray, WebhookRegistration

### Community 97 - "Request"
Cohesion: 0.17
Nodes (7): checkRendered(), Request, Result, TestCheckRendered(), TestTemplateReference_ExampleRenders(), slowDrafter, stubDrafter

### Community 98 - "Satellite ideas: landscape research and candidates"
Cohesion: 0.13
Nodes (16): 1. What the landscape ships as "ecosystem", 2.1 Sources, 2.2 Sinks, 2.3 Recipes, 2.4 CLI, 2. Extend the existing satellites (lowest effort, highest leverage), 4.1 Helm chart and Grafana pack, 4.2 Typed event payloads from event-type schemas (+8 more)

### Community 99 - "Commands"
Cohesion: 0.12
Nodes (17): Commands, From a template to a recipe, `sparrow events`, `sparrow events export` and `sparrow events import`, `sparrow functions [name]`, `sparrow init`, `sparrow invite`, `sparrow invites` (+9 more)

### Community 101 - "lib/utils.ts"
Cohesion: 0.13
Nodes (11): svelte-jsoneditor, { currentPage, totalPages, totalCount, pageSize, onPageChange, itemLabel = "items" }, from, pages, Props, to, ERROR_CATEGORIES, getCategoryBadge() (+3 more)

### Community 102 - "ADDED Requirements"
Cohesion: 0.12
Nodes (16): ADDED Requirements, Purpose, Requirement: Alert-email configuration resource, Requirement: Email delivery through the core pipeline, Requirement: Human-readable alert content, Requirement: Recipient resolution at push time, Scenario: Consumer-level opt-in, Scenario: Degradation subject (+8 more)

### Community 103 - "deliveries/+page.svelte"
Cohesion: 0.10
Nodes (27): alertsInScope, allConsumers, applyFilters(), clearFilters(), createdAfterFilter, createdBeforeFilter, deliveries, DeliveryItem (+19 more)

### Community 104 - "listEventOccurrencesImpl"
Cohesion: 0.20
Nodes (9): registerEventBundleRoutes(), listEventOccurrencesImpl(), parseDateFilter(), parseLabelFilter(), TestParseDateFilter(), TestParseDateFilter_AcceptsRFC3339(), TestParseLabelFilter(), decodeCursor() (+1 more)

### Community 105 - "security_headers_test.go"
Cohesion: 0.39
Nodes (7): buildCSP(), SecurityHeaders(), TestBuildCSP(), TestSecurityHeadersNoHashes(), TestSecurityHeadersPassthrough(), TestSecurityHeadersPortalAllowsFraming(), TestSecurityHeadersWithHashes()

### Community 106 - "payload-transformation.mdx"
Cohesion: 0.12
Nodes (15): 1. Send only the fields a partner needs, 2. Post a message to Slack, 3. Convert dollars to cents for a billing system, 4. Flatten a nested payload for a legacy endpoint, 5. Summarize a list of items, 6. Add a constant or computed field, 7. Provide safe defaults for optional fields, Good to know (+7 more)

### Community 107 - "recipes.mdx"
Cohesion: 0.12
Nodes (15): Apply a recipe, clickhouse, cloudevents, discord, ntfy, pagerduty, Pick a recipe, sendgrid (+7 more)

### Community 108 - "WriteError"
Cohesion: 0.43
Nodes (5): writeJSONError(), writePortalAuthError(), WithPrincipal(), WriteError(), writeJSON()

### Community 109 - "ref_astro_content"
Cohesion: 0.29
Nodes (3): buildDate, collections, @astrojs/starlight

### Community 110 - "devDependencies"
Cohesion: 0.13
Nodes (15): devDependencies, openapi-typescript, @playwright/test, svelte, svelte-check, @sveltejs/adapter-static, @sveltejs/kit, @sveltejs/vite-plugin-svelte (+7 more)

### Community 111 - "batch-job.svelte.ts"
Cohesion: 0.19
Nodes (8): BatchJob, BatchKind, BatchProgressState, progressOf(), terminal, api, unwrap(), formatAPIError()

### Community 113 - "event_records"
Cohesion: 0.14
Nodes (13): event_records, idx_event_records_consumer, idx_event_records_created_at, idx_event_records_event, idx_event_records_expires_at, idx_event_records_tenant, idx_event_records_tenant_event_created, idx_event_records_tenant_ns_created (+5 more)

### Community 114 - "go-template-engine.ts"
Cohesion: 0.26
Nodes (14): renderResult, applyFilterStage(), evaluateExpression(), evaluateToken(), EventContext, extractIfElseBlocks(), extractMatchingEnd(), goPrintf() (+6 more)

### Community 115 - "access.md"
Cohesion: 0.13
Nodes (14): API endpoints, CI and machine tokens, Consumer (portal) access, First run, From the API, From the CLI, From the web UI, Invite a teammate (+6 more)

### Community 116 - "recipes_test.go"
Cohesion: 0.05
Nodes (39): findRecipe(), GetFunctionMap(), GetTemplateFunctions(), TestTitleFunc(), toNumber(), checkLoopNode(), checkTemplateLoops(), ExecOptions (+31 more)

### Community 117 - "runInit"
Cohesion: 0.40
Nodes (3): isTerminal(), probeServer(), runInit()

### Community 118 - "WebhookServiceInterface"
Cohesion: 0.25
Nodes (8): buildDraftRequest(), extendWriteDeadline(), TemplateDrafter, registerAIRoutes(), registerPromptRoute(), mapError(), WebhookServiceInterface, AIDeps

### Community 119 - "registerEventRoutes"
Cohesion: 0.24
Nodes (10): registerEventRoutes(), toEventTypeChange(), toEventTypeItem(), toEventTypeOutput(), ValidateJSONSchema(), eventTypeChange, EventTypeItem, eventTypeOutput (+2 more)

### Community 121 - "AGENTS.md — Sparrow"
Cohesion: 0.14
Nodes (14): AGENTS.md — Sparrow, Architecture, Authentication, DB / Migrations, Design Principles, First reads, Frontend (Svelte 5), graphify (+6 more)

### Community 122 - "CursorPages"
Cohesion: 0.14
Nodes (6): Code conventions, Handler pattern, Repository / Storage pattern, article, bar, CursorPages

### Community 123 - "registerWebhookRoutes"
Cohesion: 0.16
Nodes (11): derefString(), formatOptionalTime(), maskEncryptedSecret(), maskSecret(), maskSecretHeaders(), toWebhookOut(), registerWebhookRoutes(), SecretRevealer (+3 more)

### Community 124 - "NetworkPolicy"
Cohesion: 0.20
Nodes (6): embeddedIPv4(), NetworkPolicy, isMetadataIP(), TestValidateIP(), ValidateIP(), WebhookService

### Community 125 - "tenants"
Cohesion: 0.19
Nodes (11): idx_tenants_slug, idx_tenants_status, tenants, update_tenants_updated_at, consumers, idx_consumers_tenant_id, consumers, idx_consumers_tenant_id (+3 more)

### Community 126 - "dependencies"
Cohesion: 0.14
Nodes (14): dependencies, astro, astro-mermaid, astro-og-canvas, @astrojs/sitemap, @astrojs/starlight, @astrojs/svelte, marked (+6 more)

### Community 127 - "benchmark-four-walls.md"
Cohesion: 0.14
Nodes (13): Day one: a tool that cannot fail loudly, Deriving health instead of maintaining it, Further reading, Measuring the thing that matters, Run it yourself, The numbers, side by side, Three things that went wrong on the way, Wall four: one busy webhook (+5 more)

### Community 128 - "for-decision-makers.mdx"
Cohesion: 0.14
Nodes (13): Does it depend on AI?, Does it tie us to Sparrow's way of doing things?, Does this help with HIPAA, GDPR, SOC 2, PCI?, How do our customers know a webhook really came from us?, How do we know it is reliable?, Is the stack open all the way down?, The short version, What does it cost? (+5 more)

### Community 129 - "entryToSimpleMarkdown.ts"
Cohesion: 0.14
Nodes (4): htmlToMarkdownPipeline, minify, minifyDefaults, selectors

### Community 131 - "compilerOptions"
Cohesion: 0.14
Nodes (13): ./.svelte-kit/tsconfig.json, compilerOptions, allowJs, checkJs, esModuleInterop, forceConsistentCasingInFileNames, moduleResolution, resolveJsonModule (+5 more)

### Community 132 - "sparrow/access_test.go"
Cohesion: 0.37
Nodes (13): fakeServer(), runCLI(), TestAccessCommandsSurfaceServerErrors(), TestInviteBuildsLinkOnUIURL(), TestInviteDefaultsToServerURLAndJSON(), TestInvitesListAndCancel(), TestInviteTokenNeverExpires(), TestTokensCreate() (+5 more)

### Community 133 - "production.md"
Cohesion: 0.15
Nodes (12): Access and authentication, Deployment, Example Ingress (internal), Kubernetes manifests, Network exposure, NetworkPolicy, Production checklist, Required configuration (+4 more)

### Community 134 - "Wrapf"
Cohesion: 0.16
Nodes (12): anthropicClient, completer, completion, openAIClient, turn, mapStatus(), mapTransportError(), newAnthropicClient() (+4 more)

### Community 135 - "Contributing to Sparrow"
Cohesion: 0.17
Nodes (10): Commit Style, Contributing to Sparrow, Development Setup, Documenting the API, Notes, Release Process, Defaults that live in the service, Documenting the REST API (+2 more)

### Community 136 - "event_registrations"
Cohesion: 0.17
Nodes (10): event_registrations, idx_event_registrations_created_at, idx_event_registrations_name, update_event_registrations_updated_at, update_updated_at_column(), update_webhook_registrations_updated_at, idx_event_registrations_tenant, idx_event_registrations_tenant_name (+2 more)

### Community 138 - "Sparrow threat model"
Cohesion: 0.14
Nodes (12): How to exercise it, Out of scope / leave alone, Principals and trust, Reports, Security properties, Severity, Sparrow threat model, Where untrusted input enters (+4 more)

### Community 139 - "main"
Cohesion: 0.07
Nodes (28): main(), Config, Load(), validatePort(), TestE2E_EventDeliveryStats_NoDeliveries(), TestEventTypeVersions_ConcurrentSchemaChanges(), TestAcquireDeliverySlot_BusyBucketDoesNotBurnSlots(), TestAcquireDeliverySlot_NoLimitCreatesNoRow() (+20 more)

### Community 141 - "Sparrow Recipes"
Cohesion: 0.17
Nodes (9): Contributing a recipe, Guidance, How params work, Included recipes, Requiring the transform, Schema (version 1), Sparrow Recipes, Recipe (+1 more)

### Community 142 - "events_bundle_test.go"
Cohesion: 0.44
Nodes (11): runCLICmd(), startBundleServer(), TestEventsExport(), TestEventsExportNeedsOneSelector(), TestEventsImport_Applied(), TestEventsImport_BlockedExitsNonZero(), TestEventsImport_DryRun(), TestEventsImport_Stdin() (+3 more)

### Community 144 - "SparrowVerify"
Cohesion: 0.35
Nodes (10): SparrowVerify, decode_public_key(), decode_secret(), decode_signatures(), fetch_header(), parse_headers(), safe_hex_decode(), validate_timestamp() (+2 more)

### Community 146 - "deployment/security.mdx"
Cohesion: 0.18
Nodes (10): Data retention and compliance posture, Hardening checklist, Not a fit, Option A — Authentik (username/password + Entra + Google in one tool), Option B — oauth2-proxy (single IdP, smallest footprint), Option C — Keycloak + oauth2-proxy (maximum boring), Recommended: SSO via an identity-aware proxy, The consumer portal (+2 more)

### Community 147 - "reference/security.mdx"
Cohesion: 0.18
Nodes (10): Access tokens, API Authentication, Consumer Isolation, HTTP Hardening, Portal tokens, Production Config Checklist, Secret Masking in Responses, SSRF Protection (+2 more)

### Community 148 - "sinks.mdx"
Cohesion: 0.17
Nodes (11): Before you start, Configuration, email, How a delivery is handled, and what Sparrow does with the answer, Install and run, otlp, Pick a sink, s3 (+3 more)

### Community 149 - "sinks_test.go"
Cohesion: 0.14
Nodes (23): capturedWebhook, parseHeaders(), signedMessage(), VerifyEd25519(), VerifyHMAC(), Ed25519(), Handle(), HMAC() (+15 more)

### Community 150 - "system.ts"
Cohesion: 0.13
Nodes (10): ALERT_EVENT_TYPES, ALERT_SETUP_HREF, ALERTS_GUIDE_URL, describeHealthRules(), EventTypeItem, HealthRules, pct(), SYSTEM_CONSUMER (+2 more)

### Community 151 - "auth.svelte.ts"
Cohesion: 0.17
Nodes (10): Use it as a Go library, KEY_STORAGE, message, OWNED_TOKEN_STORAGE, ownedTokenId, redeeming, required, stored (+2 more)

### Community 152 - "sparrow-verify.ts"
Cohesion: 0.40
Nodes (8): decodeBase64Strict(), decodeSignatures(), DEFAULT_TOLERANCE_SECONDS, Headers, parseHeaders(), SignatureVerificationError, verifyEd25519(), verifyHmac()

### Community 153 - "diagram-viewer.js"
Cohesion: 0.44
Nodes (8): apply(), build(), button(), fit(), fitWidth(), open(), size(), zoom()

### Community 154 - "BlogLayout.astro"
Cohesion: 0.24
Nodes (3): Props, base, base

### Community 155 - "separate-ui.md"
Cohesion: 0.20
Nodes (9): 1. Build the UI, 2. Point the UI at the server: `config.js`, 3. Serve it as a single-page app, 4. Configure the server, Authentication, Consumer portal, Local development, Troubleshooting (+1 more)

### Community 156 - "sources.mdx"
Cohesion: 0.20
Nodes (9): Before you start, Configuration, cron, github, Install and run, Pick a source, stripe, What the webhook server answers (+1 more)

### Community 158 - "Sparrow"
Cohesion: 0.22
Nodes (7): Contributing, Documentation, License, Quick start, Satellites, Sparrow, What you get

### Community 159 - "scripts"
Cohesion: 0.20
Nodes (10): scripts, build, check, check:watch, dev, gen:api-types, prepare, preview (+2 more)

### Community 162 - "Core Concepts"
Cohesion: 0.22
Nodes (8): Core Concepts, Deliveries, Events, Example: one event, two receivers, Key Design Decisions, Subscriptions, The Full Flow, Webhooks

### Community 163 - "Feature Comparison"
Cohesion: 0.22
Nodes (8): Deployment Flexibility, Detailed Comparison: Sparrow vs Svix, Feature Comparison, Important nuance, Sparrow vs the Landscape, What Sparrow Gives You, Where alternatives are stronger, Where Sparrow is stronger

### Community 164 - "portal-embedding.mdx"
Cohesion: 0.22
Nodes (8): Alternative: no Sparrow UI at all, Example: Caddy (simplest — start here), Example: nginx, How it works, end to end, Prerequisites, Security analysis, The route allowlist, What "forwarding" actually means

### Community 165 - ".url"
Cohesion: 0.22
Nodes (7): Example: Express (your app's backend as the proxy), Alternatives rejected, Context, Decisions, Requirement: Terminal delivery failure event, Scenario: Fired once when retries are exhausted, Scenario: Fired only on terminal failure

### Community 166 - "docs/src/lib/recipes.ts"
Cohesion: 0.22
Nodes (8): EventPreset, ORDER, parsed, rawFiles, RECIPE_META, RecipeDef, RecipeParam, RECIPES

### Community 168 - "ADDED Requirements"
Cohesion: 0.22
Nodes (8): ADDED Requirements, Purpose, Requirement: Feedback-loop guard, Requirement: Internal `_sparrow` consumer, Requirement: Registered system event types, Scenario: Idempotent registration, Scenario: Internal channel failure does not recurse, Scenario: System events are invisible to tenants

### Community 169 - "time.Duration"
Cohesion: 0.28
Nodes (5): ServiceError, Error(), TestServiceError_Error(), TestServiceError_Unwrap(), Wrap()

### Community 171 - "mcp.go"
Cohesion: 0.18
Nodes (9): ptr(), createSubscriptionIn, deliveryIDIn, listDeliveriesIn, listWebhooksIn, pushEventIn, registerWebhookIn, testTemplateIn (+1 more)

### Community 172 - "seed.sql"
Cohesion: 0.22
Nodes (8): consumer_pool, evt_ids, evt_names, hot_webhook, recent_delivery_ids, sub_ids, sub_lookup, wh_ids

### Community 173 - "Sparrow Web Dashboard"
Cohesion: 0.22
Nodes (8): Build, Configuration, Embedding in the Go Binary, Local Development, Prerequisites, Sparrow Web Dashboard, Standalone Deployment, Tech Stack

### Community 175 - "2. Split the `sparrow` CLI into its own module to isolate its `go install` graph"
Cohesion: 0.22
Nodes (8): 2. Split the `sparrow` CLI into its own module to isolate its `go install` graph, Amendment (2026-10): sinks and sources are libraries, Consequences, Context, Decision, Releasing the split modules (tag scheme), Trigger to revisit, When this is worth it

### Community 176 - "installation.mdx"
Cohesion: 0.25
Nodes (7): Build, Build from Source, Docker Compose (Recommended), Docker Image, Pre-built Binaries, Prerequisites, Run

### Community 177 - "event-type-versioning.mdx"
Cohesion: 0.25
Nodes (7): Breaking changes, Inferring a schema from real events, Reading the history, Reserved names, Retiring an event type, Unregistered event names, What creates a new version

### Community 178 - "lib/components/BatchProgress.svelte"
Cohesion: 0.18
Nodes (9): { batch, label = 'Batch', oncancel, ondone }, BatchStatus, dismissed, isTerminal, progressPercent, Props, statusColor, statusLabel (+1 more)

### Community 179 - "sealedRouter"
Cohesion: 0.13
Nodes (20): New(), NewWithStore(), Realm(), Sealer(), TestMasterKeyIsRootInDefaultTenant(), TestPgstoreConformance(), accessRouter(), post() (+12 more)

### Community 180 - "time.Duration"
Cohesion: 0.05
Nodes (28): BacklogSample, ClientConfig, ClientResults, clientTester, E2EConfig, E2EResults, E2ERunner, eventRecord (+20 more)

### Community 181 - "3. New satellites"
Cohesion: 0.29
Nodes (7): 3.1 `sparrow-bus`: message-bus bridge, both directions, 3.2 `sparrow-cdc`: Postgres change capture as events, 3.3 `sparrow mcp`: MCP server for coding agents and operators, 3.4 `sparrow-inbox`: durable consumer-side receiver, 3.5 `sparrow-capture`: a self-hosted webhook.site, 3.6 Sparrow-to-Sparrow federation, 3. New satellites

### Community 182 - "apiConsole.svelte.ts"
Cohesion: 0.22
Nodes (8): openapi-fetch, apiConsole, ApiLogEntry, apiLogMiddleware, entries, shellQuote(), started, toCurl()

### Community 183 - "What an import does"
Cohesion: 0.29
Nodes (6): API, Subscriptions whose template fails, The bundle file, The preview, What an import does, When nothing is written

### Community 184 - "webhook-health-alerts.mdx"
Cohesion: 0.29
Nodes (6): Automatic disabling, Check that alerts are being sent, Good to know, How it works, Register an alert config, Set up the email delivery channel

### Community 185 - "cron.go"
Cohesion: 0.06
Nodes (50): cronSpec, Emitter, entry, loadConfig(), main(), newWebhookHandler(), discardLogger(), githubSig() (+42 more)

### Community 187 - "proposal.md"
Cohesion: 0.29
Nodes (6): Capabilities, Impact, Modified Capabilities, New Capabilities, What Changes, Why

### Community 188 - "test-ui.sh"
Cohesion: 0.43
Nodes (5): test-ui.sh script, SPARROW_BASE_URL, start_server(), wait_for_static(), wait_healthy()

### Community 192 - "api-types.d.ts"
Cohesion: 0.33
Nodes (5): RFC-3339, components, $defs, operations, paths

### Community 193 - "Feature: Webhook health/delivery-failure email alerts"
Cohesion: 0.33
Nodes (5): Context, Design, Diagrams, Feature: Webhook health/delivery-failure email alerts, Open Questions

### Community 194 - "introducing-sparrow.md"
Cohesion: 0.33
Nodes (5): MIT, and meant that way, Simple to run does not mean simple-minded, The whole thing fits in your head, Try it in five minutes, Your data stays home

### Community 195 - "docker-compose.md"
Cohesion: 0.33
Nodes (5): Development (Build from Source), Docker Image, Observability, Quick Start, Trying authentication locally

### Community 196 - "subscription-pause.mdx"
Cohesion: 0.33
Nodes (5): Pausing from an import, Resuming, Seeing what was held, What a pause does, Where pause fits in a delivery

### Community 197 - "lib/components/TransformSettings.svelte"
Cohesion: 0.22
Nodes (8): save(), TemplateSaveMeta, changed, editorOpen, {
    enabled = $bindable(false),
    template = $bindable(""),
    onError = $bindable("fail"),
    missingKey = $bindable("error"),
    meta = $bindable(null),
    eventName,
    consumer,
    subscriptionId = "",
    disabledReason = "",
    lockedReason = "",
    required = false,
  }, lines, on, onSave()

### Community 198 - "tasks.md"
Cohesion: 0.33
Nodes (5): 1. Foundation, 2. Event emission, 3. Tenant-facing API, 4. Delivery channel, 5. Verification & docs

### Community 199 - "github.com/sarathsp06/sparrow"
Cohesion: 0.36
Nodes (8): github.com/sarathsp06/sparrow, github.com/sarathsp06/sparrow/pkg/access, github.com/sarathsp06/sparrow/pkg/signature, github.com/sarathsp06/sparrow/pkg/template, github.com/sarathsp06/sparrow/satellites/recipes, github.com/sarathsp06/sparrow/satellites/sparrow, github.com/sarathsp06/sparrow/satellites/sparrow-sinks, github.com/sarathsp06/sparrow/satellites/sparrow-sources

### Community 200 - "client_test.go"
Cohesion: 0.12
Nodes (23): NewWebhookClient(), ReadBody(), TestClientClose(), TestDeliverySpansDoNotExportSecretURLs(), TestNewWebhookClient(), TestNewWebhookClientWithNilConfig(), TestReadBody(), TestReadBodyNil() (+15 more)

### Community 201 - "Sparrow Client Libraries"
Cohesion: 0.40
Nodes (4): Go / TypeScript, Python, Signature verification helpers (`verify/`, hand-written), Sparrow Client Libraries

### Community 202 - "000027_access_tokens.up.sql"
Cohesion: 0.70
Nodes (4): access_invites, access_tokens, idx_access_invites_realm, idx_access_tokens_realm

### Community 203 - "webhook_health_summaries"
Cohesion: 0.70
Nodes (4): idx_webhook_health_summaries_unique, idx_webhook_health_summaries_webhook_id, idx_webhook_health_summaries_window, webhook_health_summaries

### Community 205 - "quickstart.mdx"
Cohesion: 0.40
Nodes (4): Before you start, If you see something else, Next steps, What happened

### Community 206 - "verify-signatures.mdx"
Cohesion: 0.40
Nodes (4): Get the helper and verify, Testing your receiver, What gets signed, Where the secret or public key comes from

### Community 207 - "error-classification.md"
Cohesion: 0.40
Nodes (4): Classification Logic, Error Categories, Monitoring Errors, Retry Behavior

### Community 208 - "cli.mdx"
Cohesion: 0.40
Nodes (4): 90-second quickstart, Configuration, Install, Worked example: develop a receiver on your laptop

### Community 211 - "schema.sql"
Cohesion: 0.70
Nodes (4): access_invites, access_tokens, idx_access_invites_realm, idx_access_tokens_realm

### Community 212 - "install.sh"
Cohesion: 0.80
Nodes (4): fail(), need(), install.sh script, usage()

### Community 213 - "release-submodules.sh"
Cohesion: 0.80
Nodes (4): release_leaf(), release_module(), release-submodules.sh script, tag_exists()

### Community 214 - "000037_health_evaluator.up.sql"
Cohesion: 0.67
Nodes (3): idx_webhook_health_buckets_start, webhook_health_buckets, webhook_health_evaluator

### Community 216 - "`sparrow tokens`"
Cohesion: 0.50
Nodes (4): `sparrow tokens`, `sparrow tokens create`, `sparrow tokens list`, `sparrow tokens revoke`

### Community 217 - "index.mdx"
Cohesion: 0.50
Nodes (3): Example: Stripe payments, end to end, The contract, The satellites

### Community 219 - "Requirement: Health transition event"
Cohesion: 0.50
Nodes (4): Requirement: Health transition event, Scenario: Degradation emits one event, Scenario: No event without a transition, Scenario: Recovery emits one event

### Community 220 - "scripts"
Cohesion: 0.40
Nodes (5): scripts, astro, build, dev, preview

### Community 229 - "lib/components/HealthBadge.svelte"
Cohesion: 0.29
Nodes (6): c, config, dotCls, { health, size = "sm" }, Props, sizeCls

### Community 241 - "aiRequestBody"
Cohesion: 0.67
Nodes (3): aiRequestBody(), copyPrompt(), draftWithAI()

### Community 242 - "scheduleRender"
Cohesion: 0.67
Nodes (3): loadContext(), renderNow(), scheduleRender()

## Knowledge Gaps
- **31 isolated node(s):** `InvalidTimestamp`, `TimestampOutsideTolerance`, `EmptySecret`, `InvalidSecretBase64`, `InvalidPublicKey` (+26 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 1823 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **105 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `SparrowConfig` connect `auth.svelte.ts` to `services.ts`?**
  _High betweenness centrality (0.053) - this node is a cross-community bridge._
- **What connects `InvalidTimestamp`, `TimestampOutsideTolerance`, `EmptySecret` to the rest of the system?**
  _31 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `github.com/google/uuid.UUID` be split into smaller, more focused modules?**
  _Cohesion score 0.03283582089552239 - nodes in this community are weakly interconnected._
- **Why does `Use it as a Go library` connect `auth.svelte.ts` to `sources.mdx`?**
  _High betweenness centrality (0.053) - this node is a cross-community bridge._
- **Should `context.Context` be split into smaller, more focused modules?**
  _Cohesion score 0.03129074315514994 - nodes in this community are weakly interconnected._
- **Why does `Configuration` connect `sources.mdx` to `.url`?**
  _High betweenness centrality (0.053) - this node is a cross-community bridge._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.023861434108527133 - nodes in this community are weakly interconnected._