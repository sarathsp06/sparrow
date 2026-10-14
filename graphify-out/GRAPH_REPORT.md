# Graph Report - bold-euclid-d955a4  (2026-10-14)

## Corpus Check
- 555 files · ~497,930 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 46 file(s) not represented in the graph (top: .spec 20, (none) 10, .ttf 6)

## Summary
- 5387 nodes · 15920 edges · 308 communities (206 shown, 102 thin omitted)
- Extraction: 94% EXTRACTED · 6% INFERRED · 0% AMBIGUOUS · INFERRED: 898 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `9cc1659c`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- github.com/google/uuid.UUID
- context.Context
- testing.T
- testhelpers_test.go
- [webhookId]/+page.svelte
- go_pkg_context
- server/main.go
- newHandler
- log/slog.Logger
- EventTypeDefinition
- setupEnv
- webhooks/register/+page.svelte
- +layout.svelte
- testContext
- EventRegistration
- WebhookDelivery
- budgetFuncs
- webhook_service_registration.go
- steps.py
- TemplateEditor.svelte
- requests
- encoding/json.RawMessage
- crypto.go
- webhooks/+page.svelte
- template_errors_test.go
- io.Writer
- PrepareDeliveryRequest
- posts.ts
- RecipeWorkbenchApp.svelte
- SubscriptionForm.svelte
- WebhookRegistration
- Repository
- use.go
- SparrowVerify.java
- github.com/riverqueue/river.JobArgs
- otel.go
- [eventId]/+page.svelte
- push/+page.svelte
- update/+page.svelte
- unwrap
- EventRecord
- Service
- middleware/portal_test.go
- 000001_consolidated_schema.up.sql
- mcp_test.go
- events/+page.svelte
- generator.ts
- lib/utils.ts
- EventTypeImport.svelte
- access/+page.svelte
- hooks.py
- reports/+page.svelte
- Sparrow — Detailed Flow Reference
- BatchJob
- newRootCmd
- time.Time
- time.Duration
- parseDuration
- sparrow/client.go
- webhook_target.py
- listDeliveriesImpl
- astro.config.mjs
- access/service_test.go
- email_test.go
- registerAccessRoutes
- s3.go
- Errorf
- web/package.json
- services.ts
- New
- sparrow_verify.py
- webhook_registrations
- EventTypeSaveResult
- prompt.go
- auth_test.go
- sparrow_verify.rs
- docs/package.json
- template-fields.ts
- steps_jobs.py
- SchemaFromSamples.svelte
- template-functions.md
- Decision
- All variables
- listSubscriptionsImpl
- api.astro
- .compare
- SparrowAPI
- events/register/+page.svelte
- architecture.mdx
- compose.ts
- steps_access.py
- api.ts
- main
- NewTemplateEngine
- Mount
- registerWebhookRoutes
- Request
- Satellite ideas: landscape research and candidates
- Commands
- net/http.Header
- ADDED Requirements
- deliveries/+page.svelte
- registerEventBundleRoutes
- security_headers_test.go
- payload-transformation.mdx
- recipes.mdx
- WriteError
- @astrojs/starlight
- devDependencies
- Pusher
- runPush
- event_records
- go-template-engine.ts
- access.md
- go_pkg_strings
- init_cmd.go
- WebhookServiceInterface
- registerEventRoutes
- Handle
- AGENTS.md — Sparrow
- ReadingProgress.astro
- SecretRevealer
- TestAcquireDeliverySlot_BusyBucketDoesNotBurnSlots
- tenants
- dependencies
- benchmark-four-walls.md
- for-decision-makers.mdx
- entryToSimpleMarkdown.ts
- main
- compilerOptions
- sparrow/access_test.go
- production.md
- Wrapf
- Documenting the REST API
- event_registrations
- Handler
- Sparrow threat model
- Config
- Security policy
- Sparrow Recipes
- events_bundle_test.go
- parseCron
- SparrowVerify
- SparrowVerify
- deployment/security.mdx
- reference/security.mdx
- sinks.mdx
- sinks_test.go
- Envelope
- Handler
- sparrow-verify.ts
- diagram-viewer.js
- recipe-workbench.astro
- separate-ui.md
- sources.mdx
- SparrowEnvironment
- Sparrow
- scripts
- Recipe
- SignatureVerificationException
- Core Concepts
- Feature Comparison
- portal-embedding.mdx
- .url
- docs/src/lib/recipes.ts
- toImportOutput
- ADDED Requirements
- newWebhookHandler
- palette
- Client
- seed.sql
- Sparrow Web Dashboard
- Contributing to Sparrow
- 2. Split the `sparrow` CLI into its own module to isolate its `go install` graph
- installation.mdx
- event-type-versioning.mdx
- registerAlertConfigRoutes
- sealedRouter
- E2ERunner
- 3. New satellites
- checkTemplates
- What an import does
- webhook-health-alerts.mdx
- Handler
- WebhookService
- proposal.md
- test-ui.sh
- xorSealer
- net/http.Client
- 1. Payload transform engine: Go `text/template`, not embedded JavaScript
- api-types.d.ts
- Feature: Webhook health/delivery-failure email alerts
- introducing-sparrow.md
- docker-compose.md
- subscription-pause.mdx
- svelte.config.js
- tasks.md
- github.com/sarathsp06/sparrow
- client_test.go
- Sparrow Client Libraries
- 000027_access_tokens.up.sql
- webhook_health_summaries
- Code conventions
- quickstart.mdx
- verify-signatures.mdx
- error-classification.md
- cli.mdx
- migrations.go
- ShareBar.astro
- schema.sql
- install.sh
- release-submodules.sh
- 000037_health_evaluator.up.sql
- LayeredArchEmbed.astro
- `sparrow tokens`
- index.mdx
- timeoutError
- Requirement: Health transition event
- IsReservedEventName
- webhook_alert_configs
- idx_access_tokens_idempotency
- subscription_template_versions
- client-libraries.mdx
- docs/tsconfig.json
- manifest.json
- json
- opencode.json
- idx_access_tokens_idempotency
- dev-env.sh
- +layout.ts
- idx_webhook_deliveries_request_body_gin
- 000015_envelope_encryption.up.sql
- 000019_drop_system_settings.down.sql
- idx_webhook_deliveries_paused
- run.sh
- sparrow-e2e

## God Nodes (most connected - your core abstractions)
1. `unwrap()` - 95 edges
2. `formatAPIError()` - 82 edges
3. `RepositoryInterfaceWithTracing` - 78 edges
4. `setupEnv()` - 70 edges
5. `EventSubscription` - 70 edges
6. `newRESTClient()` - 56 edges
7. `WebhookServiceInterfaceWithTracing` - 51 edges
8. `EventRegistration` - 48 edges
9. `testContext()` - 42 edges
10. `NewWebhookService()` - 39 edges

## Surprising Connections (you probably didn't know these)
- `Numbers` --references--> `add()`  [INFERRED]
  docs/src/content/docs/reference/template-functions.md → web/src/lib/components/AlertConfigs.svelte
- `Quick reference` --references--> `add()`  [INFERRED]
  docs/src/content/docs/reference/template-functions.md → web/src/lib/components/AlertConfigs.svelte
- ``sparrow mcp`` --references--> `delete_webhook()`  [INFERRED]
  docs/src/content/docs/satellites/cli.mdx → e2e/step_impl/steps_surface.py
- `1. What the landscape ships as "ecosystem"` --references--> `webhooks`  [INFERRED]
  docs/design/satellite-ideas.md → web/src/lib/api-types.d.ts
- `2.4 CLI` --references--> `webhooks`  [INFERRED]
  docs/design/satellite-ideas.md → web/src/lib/api-types.d.ts

## Import Cycles
- None detected.

## Communities (308 total, 102 thin omitted)

### Community 0 - "github.com/google/uuid.UUID"
Cohesion: 0.03
Nodes (16): mockRepo, AutoDisableResult, HealthRepository, Repository, EventSubscription, SubscriptionTemplateVersion, SubscriptionWithWebhook, Repository (+8 more)

### Community 1 - "context.Context"
Cohesion: 0.03
Nodes (19): AlertConfig, AlertConfigRepository, Repository, Repository, AlertConfigManager, BatchManager, EventManager, HealthManager (+11 more)

### Community 2 - "testing.T"
Cohesion: 0.02
Nodes (137): fakeModel, fakeOpenAI, ErrorCategory, sinkSMTPMessage, TestMigrationsMatchPgstoreSchema(), newDrafter(), TestDraftTemplate_DocsURL(), TestDraftTemplate_GivesUpAfterMaxAttempts() (+129 more)

### Community 3 - "testhelpers_test.go"
Cohesion: 0.25
Nodes (6): WebhookEnvelope, eventTypeResp, eventTypeVersionResp, ConsumerStats, HealthRules, WebhookUpdateFields

### Community 4 - "[webhookId]/+page.svelte"
Cohesion: 0.03
Nodes (51): activeTab, applyDeliveryFilters(), changingSignature, clearDeliveryFilters(), configForm, configHeaderKey, configHeaderValue, configSecretHeaderKey (+43 more)

### Community 5 - "go_pkg_context"
Cohesion: 0.09
Nodes (14): ctxKey, ErrorBody, RedeemRequest, consumerStats, subscriptionResp, portalCtxKey, str(), createInviteBody (+6 more)

### Community 6 - "server/main.go"
Cohesion: 0.07
Nodes (65): oaMessage, oaRequest, oaResponse, alertConfigIDInput, attemptItem, attemptsOutput, capabilitiesOutput, ConsumerItem (+57 more)

### Community 7 - "newHandler"
Cohesion: 0.18
Nodes (14): inheritVerification(), loadConfig(), main(), newHandler(), run(), signEd25519Headers(), signHeaders(), TestLoadConfig_InlineLibraryConfig() (+6 more)

### Community 8 - "log/slog.Logger"
Cohesion: 0.04
Nodes (50): main(), RunAllMigrations(), RunAppMigrations(), RunRiverMigrations(), BuildEnvelopePayload(), failing(), newAutoDisableWorker(), TestMaybeAutoDisable_DisablesAndEmits() (+42 more)

### Community 9 - "EventTypeDefinition"
Cohesion: 0.22
Nodes (13): BundleDigest(), checkStamp(), compileSchema(), EventTypeImportResult, WebhookService, TestCheckStamp(), validateBundle(), EventTypeDefinition (+5 more)

### Community 10 - "setupEnv"
Cohesion: 0.12
Nodes (64): deliveryItem, restClient, testEnv, buildCLI(), runCLI(), TestCLI_E2E(), TestCLI_EventTypeExportImport(), TestE2E_ConsumerDeliveryPaging() (+56 more)

### Community 11 - "webhooks/register/+page.svelte"
Cohesion: 0.03
Nodes (57): Schema (version 1), Recipe, RecipeParam, substituteParams(), active, alertDeliveryConfigured, alertEmail, alertEmailError (+49 more)

### Community 12 - "+layout.svelte"
Cohesion: 0.06
Nodes (30): apiConsole, ApiLogEntry, apiLogMiddleware, entries, shellQuote(), started, toCurl(), copiedId (+22 more)

### Community 13 - "testContext"
Cohesion: 0.07
Nodes (55): Status, TestAlertDeliveryConfigured(), TestCreateSubscription_CatchAllWithLabelFilters(), TestCreateSubscription_WithEmptyLabelFilters(), TestCreateSubscription_WithInvalidLabelFilters(), TestCreateSubscription_WithLabelFilters(), TestGetSubscriptionsWithWebhooksByEvent_ActiveWebhooksOnly(), TestGetSubscriptionsWithWebhooksByEvent_InactiveExcluded() (+47 more)

### Community 14 - "EventRegistration"
Cohesion: 0.05
Nodes (18): registerSystemEventTypes(), pauseFailing(), SystemEventRegistrations(), systemEventSamplePayload(), EventTypeFilter, EventTypeRepository, Repository, RepositoryInterface (+10 more)

### Community 15 - "WebhookDelivery"
Cohesion: 0.06
Nodes (15): deliveryFilterArgs(), DeliveryRepository, FanOutEnqueuer, Repository, DeliveryAttempt, cursorArgs(), DeliveryFilter, PageCursor (+7 more)

### Community 16 - "budgetFuncs"
Cohesion: 0.08
Nodes (20): atoi(), budgetFuncs(), checkPrintfFormat(), command(), instrument(), instrumentNode(), sizeOf(), stepAction() (+12 more)

### Community 17 - "webhook_service_registration.go"
Cohesion: 0.23
Nodes (6): vector, vectorFile, TestValidateHeaders(), ValidateHeaders(), validHeaderName(), validHeaderValue()

### Community 18 - "steps.py"
Cohesion: 0.10
Nodes (48): delivery_has_signature_headers(), _api(), assert_api_delivery_count(), assert_body_field(), assert_body_has_key(), assert_body_not_has_key(), assert_delivery_attempt_count(), assert_delivery_count() (+40 more)

### Community 19 - "TemplateEditor.svelte"
Cohesion: 0.03
Nodes (57): aiAttempts, aiCapability, aiConfirmReplace, aiDocsUrl, aiError, aiInstructions, aiLoading, aiNotes (+49 more)

### Community 20 - "requests"
Cohesion: 0.16
Nodes (45): alert_configs_count(), alert_configs_count_for_webhook(), also_use_consumer(), assert_delivery_page_meta(), _base(), create_alert_config_bad_webhook(), create_alert_config_scoped(), create_alert_config_wide() (+37 more)

### Community 21 - "encoding/json.RawMessage"
Cohesion: 0.17
Nodes (5): fakePusher, fakePusher, fakePusher, fakePusher, fakePusher

### Community 22 - "crypto.go"
Cohesion: 0.07
Nodes (46): cryptoSealer, kek, Key, TestPrepareDeliveryRequest(), TestPrepareDeliveryRequestFailsClosedOnDecryptError(), buildKEK(), decryptDataWithDEK(), GenerateKey() (+38 more)

### Community 23 - "webhooks/+page.svelte"
Cohesion: 0.03
Nodes (74): adding, AlertConfigItem, configs, { consumer, webhookId }, deliveryConfigured, email, error, eventLabel (+66 more)

### Community 24 - "template_errors_test.go"
Cohesion: 0.17
Nodes (24): bodyRecorder, bundle, importResult, requiresTransformSub, templateDelivery, exportBundle(), importBundle(), TestEventTypeBundle_AllOrNothingAndBreakingChanges() (+16 more)

### Community 25 - "io.Writer"
Cohesion: 0.08
Nodes (31): newPalette(), importOutcomeError(), printImportResult(), runEventsExport(), runEventsImport(), runEventsVersions(), indentJSON(), printEventType() (+23 more)

### Community 26 - "PrepareDeliveryRequest"
Cohesion: 0.08
Nodes (24): DeliveryRequest, redactSpanURL, RedactURL(), TestRedactURL(), isReservedHeader(), TestBuildRequestHeaderPrecedence(), GetHeaderMap(), PutHeaderMap() (+16 more)

### Community 27 - "posts.ts"
Cohesion: 0.10
Nodes (28): links, Props, tags, Props, Props, allTags(), base, formatDate() (+20 more)

### Community 28 - "RecipeWorkbenchApp.svelte"
Cohesion: 0.05
Nodes (39): activeRecipe, attempt, b64(), chips, cliCommand, copiedToast, DEST, drafts (+31 more)

### Community 29 - "SubscriptionForm.svelte"
Cohesion: 0.04
Nodes (50): onSignOut(), signingOut, who, {
    label,
    summary = "",
    open = $bindable(false),
    children,
  }, { icon = "inbox", title, description, action }, paths, Props, {
    expanded = $bindable(false),
    label = "Editor",
    children,
  } (+42 more)

### Community 30 - "WebhookRegistration"
Cohesion: 0.11
Nodes (6): getWebhookEventsMap(), WebhookHealth, WebhookRegistration, mockRepo, statusWebhookRepo, conversionService

### Community 31 - "Repository"
Cohesion: 0.08
Nodes (9): Repository, Repository, checkWebhookDuplicate(), Repository, WebhookRepository, insertWebhookRegistration(), DB, DBTX (+1 more)

### Community 32 - "use.go"
Cohesion: 0.09
Nodes (30): findRecipe(), All(), TestRecipes_ParamGuidance(), TestSendgridRecipe_ConsumerHint(), addRecipeTools(), declaresJSON(), TestResolveParams(), TestValidateRecipe_BuiltinsAreCleanAndValid() (+22 more)

### Community 33 - "SparrowVerify.java"
Cohesion: 0.11
Nodes (5): SignatureVerificationException, SparrowVerify, ParsedHeaders, SignatureVerificationException, SparrowVerify

### Community 34 - "github.com/riverqueue/river.JobArgs"
Cohesion: 0.07
Nodes (14): Checker, HealthResponse, ReadyResponse, NewChecker(), TestLivenessIsProcessOnlyAndReadinessDrains(), NewJobInserterWithTracing(), jobInserter, NewJobInserter() (+6 more)

### Community 35 - "otel.go"
Cohesion: 0.06
Nodes (24): DefaultConfig(), GetMeter(), GetTracer(), SparrowMetrics, legacyEndpoint(), MetricsHandler(), newLoggerProvider(), NewSparrowMetrics() (+16 more)

### Community 36 - "[eventId]/+page.svelte"
Cohesion: 0.07
Nodes (21): { pages, shown, onchange, itemLabel = "items" }, Props, CursorPages, consumer, deliveries, deliveriesError, deliveriesLoading, DeliveryItem (+13 more)

### Community 37 - "push/+page.svelte"
Cohesion: 0.10
Nodes (22): isSystemEvent(), listAllEventTypes(), safeAjvValidator(), availableEvents, consumer, error, event, EventTypeItem (+14 more)

### Community 38 - "update/+page.svelte"
Cohesion: 0.10
Nodes (24): toJSONObject(), active, confirmPending(), description, error, hasSchema(), ImportItem, loading (+16 more)

### Community 39 - "unwrap"
Cohesion: 0.04
Nodes (87): Decision, API surface deliberately not e2e-tested (removal / design candidates), Deliberately NOT covered by e2e, Known product-contract questions (asserted as-is, flagged), Out of e2e scope (belongs in unit/integration tests), message, portalInvite, status (+79 more)

### Community 40 - "EventRecord"
Cohesion: 0.09
Nodes (11): eventReportFilterArgs(), eventVersionOrDefault(), EventRepository, Repository, EventRecord, EventReportFilter, EventReportWithStats, cancellingEventRepo (+3 more)

### Community 41 - "Service"
Cohesion: 0.12
Nodes (13): cacheEntry, Config, CreateInviteRequest, CreateTokenRequest, RootKey, Principal, invalidRequest(), clonePtr() (+5 more)

### Community 42 - "middleware/portal_test.go"
Cohesion: 0.29
Nodes (15): NewPortalVerifier(), cleanPortalPath(), PortalGateway(), portalTarget(), doBearer(), mintConsumerToken(), newPortalGateway(), newPortalService() (+7 more)

### Community 43 - "000001_consolidated_schema.up.sql"
Cohesion: 0.10
Nodes (29): idx_webhook_deliveries_created_at, idx_webhook_deliveries_event_id, idx_webhook_deliveries_expires_at, idx_webhook_deliveries_request_body_gin, idx_webhook_deliveries_status, idx_webhook_deliveries_subscription_id, idx_webhook_deliveries_webhook_id, idx_webhook_health_events_delivery_id (+21 more)

### Community 44 - "mcp_test.go"
Cohesion: 0.12
Nodes (23): newAPIClient(), newMCPServer(), ptr(), call(), connectMCP(), newFakeSparrow(), TestMCPListenRoundTrip(), TestMCPPushAutoCreatesEventType() (+15 more)

### Community 45 - "events/+page.svelte"
Cohesion: 0.06
Nodes (30): { href, label, targetSelector }, visible, { currentPage, totalPages, totalCount, pageSize, onPageChange, itemLabel = "items" }, from, pages, Props, to, allVisibleSelected (+22 more)

### Community 46 - "generator.ts"
Cohesion: 0.12
Nodes (16): entryToSimpleMarkdown(), collator, generateLlmsTxt(), GET(), getStaticPaths, Params, prerender, Props (+8 more)

### Community 47 - "lib/utils.ts"
Cohesion: 0.04
Nodes (46): AttemptItem, attempts, { deliveryId }, error, loading, Props, copied, display (+38 more)

### Community 48 - "EventTypeImport.svelte"
Cohesion: 0.07
Nodes (29): acknowledged, actionLabel, actionTone, allAcknowledged, breakingConfirmed, breakingItems, bundle, busy (+21 more)

### Community 49 - "access/+page.svelte"
Cohesion: 0.06
Nodes (28): cancelBtn, confirmBtn, {
    open,
    title = "Confirm",
    message,
    confirmLabel = "Confirm",
    cancelLabel = "Cancel",
    variant = "danger",
    onconfirm,
    oncancel,
  }, Props, tone, cancelling, consumer, copied (+20 more)

### Community 50 - "hooks.py"
Cohesion: 0.09
Nodes (10): setup_environment(), setup_scenario(), teardown_environment(), teardown_scenario(), _get(), get_authed_correct_key(), get_authed_no_key(), get_authed_with_key() (+2 more)

### Community 51 - "reports/+page.svelte"
Cohesion: 0.05
Nodes (36): BatchJob, BatchKind, BatchProgressState, progressOf(), terminal, { batch, label = 'Batch', oncancel, ondone }, BatchStatus, dismissed (+28 more)

### Community 52 - "Sparrow — Detailed Flow Reference"
Cohesion: 0.06
Nodes (30): 1. Webhook Registration, 2. Event Registration, 3. Event Push (PushEvent), 4. Event Processing (EventProcessingWorker), 5. Webhook Delivery (WebhookWorker), 6. Subscription Creation, 7. Webhook Update (UpdateWebhookConfig), DB Writes (+22 more)

### Community 53 - "BatchJob"
Cohesion: 0.10
Nodes (8): BatchRepository, Repository, BatchJob, BatchJobStatus, BatchJobType, WebhookService, fakeBatchRepo, BatchJobData

### Community 54 - "newRootCmd"
Cohesion: 0.11
Nodes (38): accessLabel(), apiClient, newInviteCmd(), newInvitesCmd(), newTokensCmd(), newTokensCreateCmd(), newTokensListCmd(), newTokensRevokeCmd() (+30 more)

### Community 55 - "time.Time"
Cohesion: 0.05
Nodes (25): countingStore, Status, downStore, RedeemResponse, encodeCursor(), newCursorPagination(), Store, downStore (+17 more)

### Community 56 - "time.Duration"
Cohesion: 0.14
Nodes (10): InviteTTL(), TestTTLRules(), TokenTTL(), extendWriteDeadline(), NewTokenPurgeWorker(), ms(), TokenPurgeArgs, TokenPurger (+2 more)

### Community 57 - "parseDuration"
Cohesion: 0.29
Nodes (3): parseDuration(), TestParseDuration(), durationFlag

### Community 58 - "sparrow/client.go"
Cohesion: 0.14
Nodes (13): apiClient, apiError, consumerStats, deliveryItem, eventTypeItem, pushBody, pushResult, subscriptionItem (+5 more)

### Community 59 - "webhook_target.py"
Cohesion: 0.08
Nodes (4): CapturedDelivery, _Target, handle(), WebhookTargetManager

### Community 60 - "listDeliveriesImpl"
Cohesion: 0.09
Nodes (27): listDeliveriesImpl(), registerDeliveryRoutes(), toDeliveryItem(), toDeliveryOutput(), listEventOccurrencesImpl(), toBatchJobOutput(), parseDateFilter(), parseLabelFilter() (+19 more)

### Community 61 - "astro.config.mjs"
Cohesion: 0.16
Nodes (10): blogLastmod, starlightLlmsTxt(), GET(), prerender, ensureTrailingSlash(), getSiteTitle(), astro, astro-mermaid (+2 more)

### Community 62 - "access/service_test.go"
Cohesion: 0.27
Nodes (24): clock, New(), New(), newClock(), newSealedService(), newService(), ptr(), reason() (+16 more)

### Community 63 - "email_test.go"
Cohesion: 0.35
Nodes (10): smtpMessage, newEmailSink(), serveSMTP(), startSMTPServer(), TestEmailRender_BadTemplate(), TestEmailRender_CustomTemplatesAndHeaderInjection(), TestEmailRender_Defaults(), TestEmailSink_SendsViaSMTP() (+2 more)

### Community 64 - "registerAccessRoutes"
Cohesion: 0.10
Nodes (23): TestValidateConsumer(), ValidateConsumer(), consumerScope(), fragmentEscape(), mapAccessError(), portalLinkPath(), principalName(), registerAccessRoutes() (+15 more)

### Community 65 - "s3.go"
Cohesion: 0.22
Nodes (7): s3Sink, Config, Handler(), newS3Sink(), objectKey(), TestObjectKey(), TestS3Sink_PutObject()

### Community 66 - "Errorf"
Cohesion: 0.05
Nodes (27): checkEventForConsumer(), reservedEventNameError(), DerefBoolOr(), DerefIntOr(), WebhookRegistrationRequest, validateSubscriptionTimeout(), stringHeaders(), validateHeaders() (+19 more)

### Community 67 - "web/package.json"
Cohesion: 0.10
Nodes (20): openapi-fetch, openapi-typescript, svelte-check, svelte-jsoneditor, @sveltejs/kit, tailwindcss, @tailwindcss/forms, @tailwindcss/vite (+12 more)

### Community 68 - "services.ts"
Cohesion: 0.06
Nodes (43): Use it as a Go library, auth, KEY_STORAGE, message, OWNED_TOKEN_STORAGE, ownedTokenId, redeeming, required (+35 more)

### Community 69 - "New"
Cohesion: 0.16
Nodes (12): anthropicClient, completer, Config, Drafter, Provider, Renderer, DocFetcher, New() (+4 more)

### Community 70 - "sparrow_verify.py"
Cohesion: 0.15
Nodes (9): _check_timestamp(), _decode_signatures(), _required_headers(), SignatureVerificationError, verify_ed25519(), verify_hmac(), _signed_body(), verify_ed25519_signature() (+1 more)

### Community 71 - "webhook_registrations"
Cohesion: 0.10
Nodes (18): event_subscriptions, idx_event_subscriptions_event, idx_webhook_registrations_active, idx_webhook_registrations_consumer, idx_webhook_registrations_health, webhook_registrations, idx_event_records_tenant_consumer, idx_event_subscriptions_tenant (+10 more)

### Community 72 - "EventTypeSaveResult"
Cohesion: 0.27
Nodes (9): breakingChangeError(), EventTypeSaveResult, WebhookService, metadataEqual(), planEventTypeSave(), schemasEqual(), EventTypeSaveAction, eventTypeSaveMode (+1 more)

### Community 73 - "prompt.go"
Cohesion: 0.15
Nodes (21): HelperFunc, outlineField, PromptBuilder, RecipeParam, fieldStep(), Recipe, mustJSON(), NewPromptBuilder() (+13 more)

### Community 74 - "auth_test.go"
Cohesion: 0.18
Nodes (17): Verifier, NewAuth(), apiKey(), bearer(), guarded(), newAuth(), TestAPIKeyHTTPMiddlewareAcceptsHeader(), TestAPIKeyHTTPMiddlewareBypassesExcludedPath() (+9 more)

### Community 75 - "sparrow_verify.rs"
Cohesion: 0.15
Nodes (17): DEFAULT_TOLERANCE_SECS, hex_decode(), hex_nibble(), parse_headers(), SignatureError, EmptySecret, InvalidPublicKey, InvalidSecretBase64 (+9 more)

### Community 76 - "docs/package.json"
Cohesion: 0.10
Nodes (17): devDependencies, starlight-llms-txt, svelte, name, private, scripts, astro, build (+9 more)

### Community 77 - "template-fields.ts"
Cohesion: 0.20
Nodes (22): editField(), actionBody(), actionEnd(), balanced(), decodeAction(), decodeJsonString(), decodeUrlValue(), encodeText() (+14 more)

### Community 78 - "steps_jobs.py"
Cohesion: 0.22
Nodes (20): assert_all_terminal_status(), assert_bulk_retry_count(), assert_consumer_stats(), assert_consumer_stats_webhooks(), assert_global_stats(), assert_health_failed_count(), assert_health_summary(), assert_job_progress() (+12 more)

### Community 79 - "SchemaFromSamples.svelte"
Cohesion: 0.09
Nodes (19): coverage, error, { eventName, sampleId = '', hasSchema, ongenerate }, EventOccurrenceItem, eventsError, generate(), loadEvents(), loadingEvents (+11 more)

### Community 80 - "template-functions.md"
Cohesion: 0.12
Nodes (15): A complete template, Building JSON, default, dig, Encoding, Lists and splitting, Missing values, Numbers (+7 more)

### Community 81 - "Decision"
Cohesion: 0.09
Nodes (22): 1. Reusable library: `pkg/access`, 2. Sparrow adapter: `internal/accessauth`, 3. Replace the single shared API key with revocable access tokens and one-time invites, 3. Two scopes only — no roles, 4. Stored invites instead of signed links, 5. No cascade revocation, 6. Tokens independent of master key, 7. 503, not 401, when the database is down (+14 more)

### Community 82 - "All variables"
Cohesion: 0.09
Nodes (22): Access, AI drafting, AI template drafting, All variables, An OpenAI-compatible server, Anthropic, Database Pools, Default Tenant (+14 more)

### Community 83 - "listSubscriptionsImpl"
Cohesion: 0.23
Nodes (11): listSubscriptionsImpl(), registerSubscriptionRoutes(), toSubscriptionItem(), toSubscriptionOutput(), listSubscriptionsGlobalInput, listSubscriptionsInput, listSubscriptionsOutput, resumeSubscriptionOutput (+3 more)

### Community 84 - "api.astro"
Cohesion: 0.10
Nodes (10): out, items, Props, toc, spec, specPath, @scalar/astro, sharp (+2 more)

### Community 85 - ".compare"
Cohesion: 0.21
Nodes (9): ClassifySchemaChange(), firstComposition(), joinPath(), requiredSet(), sortedKeys(), typeAllows(), typeChange(), typesOf() (+1 more)

### Community 86 - "SparrowAPI"
Cohesion: 0.15
Nodes (4): Authentication, `sparrow mcp`, SparrowAPI, _to_dict()

### Community 87 - "events/register/+page.svelte"
Cohesion: 0.09
Nodes (28): analyzeSamples(), collect(), DATE, detectStringFormat(), emit(), FieldCoverage, inferSchema(), jsonToJsonSchema() (+20 more)

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

### Community 92 - "main"
Cohesion: 0.10
Nodes (30): logCORSMode(), main(), Load(), CORS(), CORSMode, NormalizeOrigins(), corsChain(), get() (+22 more)

### Community 93 - "NewTemplateEngine"
Cohesion: 0.08
Nodes (38): checkRendered(), TestCheckRendered(), TestTemplateReference_ExampleRenders(), BenchmarkSend(), TestRenderTemplatePreview_StrictFlag(), RenderTemplatePreview(), TestExecuteBoundsConcurrency(), TestExecuteBudgetKeepsNormalTemplates() (+30 more)

### Community 94 - "Mount"
Cohesion: 0.14
Nodes (16): main(), aiRouter(), do(), TestCapabilities_ReflectsDrafter(), TestCapabilities_ReportsAlertDelivery(), TestDraftTemplate_DisabledIs503(), TestDraftTemplate_GroundsRequestAndReturnsDraft(), TestDraftTemplate_NotFound() (+8 more)

### Community 96 - "registerWebhookRoutes"
Cohesion: 0.20
Nodes (9): registerWebhookRoutes(), DefaultWebhookHTTPConfig(), WebhookHTTPConfig, TestDefaultWebhookHTTPConfig_RateLimitRPS(), TestValidateConfig_RateLimitRPS(), IntArray, JSONBMap, StringArray (+1 more)

### Community 97 - "Request"
Cohesion: 0.23
Nodes (6): draft, Request, Result, parseDraft(), TestParseDraft_ToleratesFencesAndPreamble(), stubDrafter

### Community 98 - "Satellite ideas: landscape research and candidates"
Cohesion: 0.13
Nodes (16): 1. What the landscape ships as "ecosystem", 2.1 Sources, 2.2 Sinks, 2.3 Recipes, 2.4 CLI, 2. Extend the existing satellites (lowest effort, highest leverage), 4.1 Helm chart and Grafana pack, 4.2 Typed event payloads from event-type schemas (+8 more)

### Community 99 - "Commands"
Cohesion: 0.12
Nodes (17): Commands, From a template to a recipe, `sparrow events`, `sparrow events export` and `sparrow events import`, `sparrow functions [name]`, `sparrow init`, `sparrow invite`, `sparrow invites` (+9 more)

### Community 101 - "net/http.Header"
Cohesion: 0.36
Nodes (6): capturedWebhook, parseHeaders(), signedMessage(), VerifyEd25519(), VerifyHMAC(), Verifier

### Community 102 - "ADDED Requirements"
Cohesion: 0.12
Nodes (16): ADDED Requirements, Purpose, Requirement: Alert-email configuration resource, Requirement: Email delivery through the core pipeline, Requirement: Human-readable alert content, Requirement: Recipient resolution at push time, Scenario: Consumer-level opt-in, Scenario: Degradation subject (+8 more)

### Community 103 - "deliveries/+page.svelte"
Cohesion: 0.09
Nodes (28): isSystemConsumer(), alertsInScope, allConsumers, applyFilters(), clearFilters(), createdAfterFilter, createdBeforeFilter, deliveries (+20 more)

### Community 104 - "registerEventBundleRoutes"
Cohesion: 0.24
Nodes (9): fromBundleItems(), registerEventBundleRoutes(), toBundleItems(), bundleItem, bundleStampBody, eventRouteService, eventTypeBundle, exportEventTypesOutput (+1 more)

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
Cohesion: 0.22
Nodes (9): AuthError, writeJSONError(), PortalAuthorized(), writePortalAuthError(), Reason, authMessage(), WithPrincipal(), WriteError() (+1 more)

### Community 109 - "@astrojs/starlight"
Cohesion: 0.33
Nodes (3): buildDate, collections, @astrojs/starlight

### Community 110 - "devDependencies"
Cohesion: 0.13
Nodes (15): devDependencies, openapi-typescript, @playwright/test, svelte, svelte-check, @sveltejs/adapter-static, @sveltejs/kit, @sveltejs/vite-plugin-svelte (+7 more)

### Community 111 - "Pusher"
Cohesion: 0.27
Nodes (8): Emitter, entry, Job, New(), discardLogger(), TestCronTickPushesOnMatch(), TestNewCronValidates(), Pusher

### Community 112 - "runPush"
Cohesion: 0.22
Nodes (6): parseJSONArg(), TestParseJSONArg(), pushWithAutoCreate(), runPush(), runTemplateTest(), kvFlag

### Community 113 - "event_records"
Cohesion: 0.14
Nodes (13): event_records, idx_event_records_consumer, idx_event_records_created_at, idx_event_records_event, idx_event_records_expires_at, idx_event_records_tenant, idx_event_records_tenant_event_created, idx_event_records_tenant_ns_created (+5 more)

### Community 114 - "go-template-engine.ts"
Cohesion: 0.26
Nodes (14): renderResult, applyFilterStage(), evaluateExpression(), evaluateToken(), EventContext, extractIfElseBlocks(), extractMatchingEnd(), goPrintf() (+6 more)

### Community 115 - "access.md"
Cohesion: 0.13
Nodes (14): API endpoints, CI and machine tokens, Consumer (portal) access, First run, From the API, From the CLI, From the web UI, Invite a teammate (+6 more)

### Community 116 - "go_pkg_strings"
Cohesion: 0.15
Nodes (3): applyRecipeIn, recipeNameIn, validateRecipeIn

### Community 117 - "init_cmd.go"
Cohesion: 0.19
Nodes (12): configPath(), resolveConfig(), saveConfig(), TestResolveConfigDefaults(), TestResolveConfigPrecedence(), TestSaveConfigPermissions(), writeConfigFile(), isTerminal() (+4 more)

### Community 118 - "WebhookServiceInterface"
Cohesion: 0.18
Nodes (10): buildDraftRequest(), TemplateDrafter, registerAIRoutes(), registerPromptRoute(), mapError(), WebhookServiceInterface, AIDeps, createOnlyService (+2 more)

### Community 119 - "registerEventRoutes"
Cohesion: 0.11
Nodes (19): registerEventRoutes(), toEventTypeChange(), toEventTypeItem(), toEventTypeOutput(), toEventTypeVersionItem(), newPagination(), ValidateJSONSchema(), eventTypeChange (+11 more)

### Community 120 - "Handle"
Cohesion: 0.31
Nodes (9): SMTPConfig, Config, Handler(), Ed25519(), Handle(), HMAC(), VerifierFor(), DeliverFunc (+1 more)

### Community 121 - "AGENTS.md — Sparrow"
Cohesion: 0.14
Nodes (14): AGENTS.md — Sparrow, Architecture, Authentication, DB / Migrations, Design Principles, First reads, Frontend (Svelte 5), graphify (+6 more)

### Community 123 - "SecretRevealer"
Cohesion: 0.12
Nodes (14): derefString(), formatOptionalTime(), maskEncryptedSecret(), maskSecret(), maskSecretHeaders(), toWebhookOut(), healthRules(), registerHealthRoutes() (+6 more)

### Community 124 - "TestAcquireDeliverySlot_BusyBucketDoesNotBurnSlots"
Cohesion: 0.33
Nodes (7): TestAcquireDeliverySlot_BusyBucketDoesNotBurnSlots(), TestAcquireDeliverySlot_NoLimitCreatesNoRow(), runMigrations(), setupTestDB(), Bootstrap(), TestSQLBench(), Open()

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

### Community 130 - "main"
Cohesion: 0.22
Nodes (8): loadConfig(), main(), TestLoadConfig(), TestLoadConfig_RequiresURL(), SparrowConfig, Config, fileConfig, providers

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
Cohesion: 0.14
Nodes (14): completion, turn, ServiceError, mapStatus(), mapTransportError(), truncate(), TestTemplateRenderError(), templateRenderError() (+6 more)

### Community 135 - "Documenting the REST API"
Cohesion: 0.33
Nodes (4): Defaults that live in the service, Documenting the REST API, Where each part of the reference comes from, Writing descriptions

### Community 136 - "event_registrations"
Cohesion: 0.17
Nodes (10): event_registrations, idx_event_registrations_created_at, idx_event_registrations_name, update_event_registrations_updated_at, update_updated_at_column(), update_webhook_registrations_updated_at, idx_event_registrations_tenant, idx_event_registrations_tenant_name (+2 more)

### Community 137 - "Handler"
Cohesion: 0.25
Nodes (9): Config, Handler(), mapEvent(), discardLogger(), githubSig(), TestHandler(), TestMapGitHubEvent(), TestVerifyGitHubSignature() (+1 more)

### Community 138 - "Sparrow threat model"
Cohesion: 0.25
Nodes (8): How to exercise it, Out of scope / leave alone, Principals and trust, Reports, Security properties, Severity, Sparrow threat model, Where untrusted input enters

### Community 139 - "Config"
Cohesion: 0.06
Nodes (21): Config, collapseSpace(), htmlToText(), looksLikeHTML(), NewDocFetcher(), TestDocFetcher_ExtractsTextAndRespectsPolicy(), validatePort(), Config (+13 more)

### Community 140 - "Security policy"
Cohesion: 0.25
Nodes (4): Reporting a vulnerability, Scope, Security policy, Supported versions

### Community 141 - "Sparrow Recipes"
Cohesion: 0.29
Nodes (6): Contributing a recipe, Guidance, How params work, Included recipes, Requiring the transform, Sparrow Recipes

### Community 142 - "events_bundle_test.go"
Cohesion: 0.44
Nodes (11): runCLICmd(), startBundleServer(), TestEventsExport(), TestEventsExportNeedsOneSelector(), TestEventsImport_Applied(), TestEventsImport_BlockedExitsNonZero(), TestEventsImport_DryRun(), TestEventsImport_Stdin() (+3 more)

### Community 143 - "parseCron"
Cohesion: 0.29
Nodes (7): cronSpec, parseCron(), parseCronField(), at(), mustParse(), TestCronMatch(), TestParseCronInvalid()

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
Cohesion: 0.38
Nodes (11): postSink(), signEd25519Headers(), signHeaders(), TestHandle_Ed25519(), TestSinkHandler_DownstreamFailureIs502(), TestSinkHandler_MissingHeaders(), TestSinkHandler_TamperedBody(), TestSinkHandler_TransformedBodyIs422() (+3 more)

### Community 150 - "Envelope"
Cohesion: 0.43
Nodes (4): emailSink, templateContext(), otlpPayload(), Envelope

### Community 151 - "Handler"
Cohesion: 0.32
Nodes (8): otlpSink, Config, Handler(), logsURL(), newOTLPSink(), TestLogsURL(), TestOTLPSinkDeliver(), TestOTLPSinkDeliverDownstreamFailure()

### Community 152 - "sparrow-verify.ts"
Cohesion: 0.40
Nodes (8): decodeBase64Strict(), decodeSignatures(), DEFAULT_TOLERANCE_SECONDS, Headers, parseHeaders(), SignatureVerificationError, verifyEd25519(), verifyHmac()

### Community 153 - "diagram-viewer.js"
Cohesion: 0.44
Nodes (8): apply(), build(), button(), fit(), fitWidth(), open(), size(), zoom()

### Community 155 - "separate-ui.md"
Cohesion: 0.20
Nodes (9): 1. Build the UI, 2. Point the UI at the server: `config.js`, 3. Serve it as a single-page app, 4. Configure the server, Authentication, Consumer portal, Local development, Troubleshooting (+1 more)

### Community 156 - "sources.mdx"
Cohesion: 0.20
Nodes (9): Before you start, Configuration, cron, github, Install and run, Pick a source, stripe, What the webhook server answers (+1 more)

### Community 158 - "Sparrow"
Cohesion: 0.29
Nodes (7): Contributing, Documentation, License, Quick start, Satellites, Sparrow, What you get

### Community 159 - "scripts"
Cohesion: 0.20
Nodes (10): scripts, build, check, check:watch, dev, gen:api-types, prepare, preview (+2 more)

### Community 160 - "Recipe"
Cohesion: 0.29
Nodes (7): Subscription, Webhook, listRecipesOutput, ConsumerHint, Param, Recipe, recipeSummary

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
Cohesion: 0.20
Nodes (9): EventPreset, ORDER, parsed, rawFiles, RECIPE_META, RecipeDef, RecipeParam, RECIPES (+1 more)

### Community 167 - "toImportOutput"
Cohesion: 0.29
Nodes (6): toImportOutput(), importEventTypesOutput, importItemResult, stampCheckItem, templateCheckItem, templateFailureItem

### Community 168 - "ADDED Requirements"
Cohesion: 0.22
Nodes (8): ADDED Requirements, Purpose, Requirement: Feedback-loop guard, Requirement: Internal `_sparrow` consumer, Requirement: Registered system event types, Scenario: Idempotent registration, Scenario: Internal channel failure does not recurse, Scenario: System events are invisible to tenants

### Community 169 - "newWebhookHandler"
Cohesion: 0.43
Nodes (7): newWebhookHandler(), discardLogger(), githubSig(), stripeSig(), TestWebhookHandler(), TestWebhookHandlerConfigErrors(), TestWebhookHandlerDefaultPath()

### Community 171 - "Client"
Cohesion: 0.48
Nodes (3): NewClient(), TestClientAutoCreatesEventType(), Client

### Community 172 - "seed.sql"
Cohesion: 0.22
Nodes (8): consumer_pool, evt_ids, evt_names, hot_webhook, recent_delivery_ids, sub_ids, sub_lookup, wh_ids

### Community 173 - "Sparrow Web Dashboard"
Cohesion: 0.22
Nodes (8): Build, Configuration, Embedding in the Go Binary, Local Development, Prerequisites, Sparrow Web Dashboard, Standalone Deployment, Tech Stack

### Community 174 - "Contributing to Sparrow"
Cohesion: 0.33
Nodes (6): Commit Style, Contributing to Sparrow, Development Setup, Documenting the API, Notes, Release Process

### Community 175 - "2. Split the `sparrow` CLI into its own module to isolate its `go install` graph"
Cohesion: 0.22
Nodes (8): 2. Split the `sparrow` CLI into its own module to isolate its `go install` graph, Amendment (2026-10): sinks and sources are libraries, Consequences, Context, Decision, Releasing the split modules (tag scheme), Trigger to revisit, When this is worth it

### Community 176 - "installation.mdx"
Cohesion: 0.25
Nodes (7): Build, Build from Source, Docker Compose (Recommended), Docker Image, Pre-built Binaries, Prerequisites, Run

### Community 177 - "event-type-versioning.mdx"
Cohesion: 0.25
Nodes (7): Breaking changes, Inferring a schema from real events, Reading the history, Reserved names, Retiring an event type, Unregistered event names, What creates a new version

### Community 178 - "registerAlertConfigRoutes"
Cohesion: 0.33
Nodes (5): registerAlertConfigRoutes(), toAlertConfigItem(), alertConfigItem, alertConfigOutput, listAlertConfigsOutput

### Community 179 - "sealedRouter"
Cohesion: 0.11
Nodes (24): New(), NewWithStore(), Realm(), Sealer(), TestMasterKeyIsRootInDefaultTenant(), TestPgstoreConformance(), accessRouter(), post() (+16 more)

### Community 180 - "E2ERunner"
Cohesion: 0.06
Nodes (22): BacklogSample, ClientConfig, ClientResults, clientTester, E2EConfig, E2EResults, E2ERunner, eventRecord (+14 more)

### Community 181 - "3. New satellites"
Cohesion: 0.29
Nodes (7): 3.1 `sparrow-bus`: message-bus bridge, both directions, 3.2 `sparrow-cdc`: Postgres change capture as events, 3.3 `sparrow mcp`: MCP server for coding agents and operators, 3.4 `sparrow-inbox`: durable consumer-side receiver, 3.5 `sparrow-capture`: a self-hosted webhook.site, 3.6 Sparrow-to-Sparrow federation, 3. New satellites

### Community 182 - "checkTemplates"
Cohesion: 0.33
Nodes (6): checkTemplates(), generatePayload(), SamplePayload(), EventTypeImportItem, TemplateCheck, TemplateCheckFailure

### Community 183 - "What an import does"
Cohesion: 0.29
Nodes (6): API, Subscriptions whose template fails, The bundle file, The preview, What an import does, When nothing is written

### Community 184 - "webhook-health-alerts.mdx"
Cohesion: 0.29
Nodes (6): Automatic disabling, Check that alerts are being sent, Good to know, How it works, Register an alert config, Set up the email delivery channel

### Community 185 - "Handler"
Cohesion: 0.20
Nodes (11): Handle(), TestHandle(), Handler(), mapEvent(), discardLogger(), stripeSig(), TestHandler(), TestMapStripeEvent() (+3 more)

### Community 187 - "proposal.md"
Cohesion: 0.29
Nodes (6): Capabilities, Impact, Modified Capabilities, New Capabilities, What Changes, Why

### Community 188 - "test-ui.sh"
Cohesion: 0.43
Nodes (5): test-ui.sh script, SPARROW_BASE_URL, start_server(), wait_for_static(), wait_healthy()

### Community 191 - "1. Payload transform engine: Go `text/template`, not embedded JavaScript"
Cohesion: 0.40
Nodes (4): 1. Payload transform engine: Go `text/template`, not embedded JavaScript, Consequences, Context, Trigger to revisit

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

### Community 197 - "svelte.config.js"
Cohesion: 0.40
Nodes (3): @sveltejs/adapter-static, @sveltejs/vite-plugin-svelte, config

### Community 198 - "tasks.md"
Cohesion: 0.33
Nodes (5): 1. Foundation, 2. Event emission, 3. Tenant-facing API, 4. Delivery channel, 5. Verification & docs

### Community 199 - "github.com/sarathsp06/sparrow"
Cohesion: 0.36
Nodes (8): github.com/sarathsp06/sparrow, github.com/sarathsp06/sparrow/pkg/access, github.com/sarathsp06/sparrow/pkg/signature, github.com/sarathsp06/sparrow/pkg/template, github.com/sarathsp06/sparrow/satellites/recipes, github.com/sarathsp06/sparrow/satellites/sparrow, github.com/sarathsp06/sparrow/satellites/sparrow-sinks, github.com/sarathsp06/sparrow/satellites/sparrow-sources

### Community 200 - "client_test.go"
Cohesion: 0.08
Nodes (31): WebhookClient, NewWebhookClient(), ReadBody(), TestClientClose(), TestDeliverySpansDoNotExportSecretURLs(), TestNewWebhookClient(), TestNewWebhookClientWithNilConfig(), TestReadBody() (+23 more)

### Community 201 - "Sparrow Client Libraries"
Cohesion: 0.40
Nodes (4): Go / TypeScript, Python, Signature verification helpers (`verify/`, hand-written), Sparrow Client Libraries

### Community 202 - "000027_access_tokens.up.sql"
Cohesion: 0.70
Nodes (4): access_invites, access_tokens, idx_access_invites_realm, idx_access_tokens_realm

### Community 203 - "webhook_health_summaries"
Cohesion: 0.70
Nodes (4): idx_webhook_health_summaries_unique, idx_webhook_health_summaries_webhook_id, idx_webhook_health_summaries_window, webhook_health_summaries

### Community 204 - "Code conventions"
Cohesion: 0.67
Nodes (3): Code conventions, Handler pattern, Repository / Storage pattern

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

### Community 220 - "IsReservedEventName"
Cohesion: 0.67
Nodes (3): IsReservedEventName(), TestIsReservedEventName(), validateEventTypeName()

## Knowledge Gaps
- **31 isolated node(s):** `InvalidTimestamp`, `TimestampOutsideTolerance`, `EmptySecret`, `InvalidSecretBase64`, `InvalidPublicKey` (+26 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 1763 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **102 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Use it as a Go library` connect `services.ts` to `sources.mdx`?**
  _High betweenness centrality (0.046) - this node is a cross-community bridge._
- **What connects `InvalidTimestamp`, `TimestampOutsideTolerance`, `EmptySecret` to the rest of the system?**
  _31 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `github.com/google/uuid.UUID` be split into smaller, more focused modules?**
  _Cohesion score 0.030729833546734954 - nodes in this community are weakly interconnected._
- **Why does `Configuration` connect `sources.mdx` to `.url`?**
  _High betweenness centrality (0.046) - this node is a cross-community bridge._
- **Should `context.Context` be split into smaller, more focused modules?**
  _Cohesion score 0.03085399449035813 - nodes in this community are weakly interconnected._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.02235792376637447 - nodes in this community are weakly interconnected._
- **Should `[webhookId]/+page.svelte` be split into smaller, more focused modules?**
  _Cohesion score 0.03278688524590164 - nodes in this community are weakly interconnected._