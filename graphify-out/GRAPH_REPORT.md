# Graph Report - nervous-hofstadter-f94811  (2026-10-12)

## Corpus Check
- 537 files · ~486,402 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 44 file(s) not represented in the graph (top: .spec 20, (none) 9, .ttf 6)

## Summary
- 5285 nodes · 15092 edges · 299 communities (192 shown, 107 thin omitted)
- Extraction: 94% EXTRACTED · 6% INFERRED · 0% AMBIGUOUS · INFERRED: 863 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `6fce0876`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- github.com/google/uuid.UUID
- context.Context
- testing.T
- go_pkg_context
- [webhookId]/+page.svelte
- store/models.go
- Token
- server/main.go
- log/slog.Logger
- Errorf
- setupEnv
- webhooks/register/+page.svelte
- clientTester
- testContext
- EventRegistration
- otel.go
- EventSubscription
- access/service.go
- steps.py
- TemplateEditor.svelte
- requests
- sources/config.go
- NewService
- webhooks/+page.svelte
- email_test.go
- io.Writer
- testhelpers_test.go
- posts.ts
- RecipeWorkbenchApp.svelte
- SubscriptionForm.svelte
- time.Time
- Repository
- use.go
- SparrowVerify.java
- Manager
- EventTypeDefinition
- deliveries/+page.svelte
- lib/utils.ts
- template_errors_test.go
- unwrap
- BuildRequest
- Service
- [eventId]/+page.svelte
- 000001_consolidated_schema.up.sql
- mcp_test.go
- events/+page.svelte
- generator.ts
- DeliveryTable.svelte
- EventTypeImport.svelte
- access/+page.svelte
- hooks.py
- reports/+page.svelte
- Sparrow — Detailed Flow Reference
- BatchJob
- newRootCmd
- E2ERunner
- WebhookHTTPConfig
- runInvite
- sparrow/client.go
- webhook_target.py
- listDeliveriesImpl
- services.ts
- access/service_test.go
- astro.config.mjs
- rest/access.go
- newDrafter
- mockRepo
- web/package.json
- update/+page.svelte
- event.go
- sparrow_verify.py
- webhook_registrations
- Checker
- prompt.go
- httpauth.go
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
- main
- SparrowAPI
- New
- architecture.mdx
- compose.ts
- steps_access.py
- api.ts
- Service
- NewTemplateEngine
- mcpListener
- net/http.Handler
- SecretRevealer
- Request
- Satellite ideas: landscape research and candidates
- Commands
- registerEventBundleRoutes
- ADDED Requirements
- ServiceError
- TemplateCache
- payload-transformation.mdx
- recipes.mdx
- middleware/portal_test.go
- NewWebhookTemplateContext
- devDependencies
- webhookHTTPConfig
- runListen
- event_records
- go-template-engine.ts
- access.md
- TemplateEngine
- sparrow/config.go
- Mount
- registerEventRoutes
- .Export
- AGENTS.md — Sparrow
- ReadingProgress.astro
- WebhookClient
- Config
- tenants
- dependencies
- benchmark-four-walls.md
- for-decision-makers.mdx
- entryToSimpleMarkdown.ts
- svelte.config.js
- compilerOptions
- sparrow/access_test.go
- production.md
- Wrapf
- go_pkg_embed
- event_registrations
- Recipe
- emitHealthChangedEvent
- .compare
- ShareBar.astro
- Sparrow Recipes
- events_bundle_test.go
- dependencies
- SparrowVerify
- SparrowVerify
- deployment/security.mdx
- reference/security.mdx
- sinks.mdx
- net/http.Header
- NetworkPolicy
- sparrow-verify.ts
- diagram-viewer.js
- recipe-workbench.astro
- separate-ui.md
- sources.mdx
- SparrowEnvironment
- runTemplateTest
- scripts
- SignatureVerificationException
- Core Concepts
- Feature Comparison
- portal-embedding.mdx
- .url
- docs/src/lib/recipes.ts
- security_headers_test.go
- ADDED Requirements
- palette
- sink_test.go
- seed.sql
- Sparrow Web Dashboard
- 2. Split the `sparrow` CLI into its own module to isolate its `go install` graph
- installation.mdx
- event-type-versioning.mdx
- NewTemplateEngineWithCacheSize
- time.Duration
- 3. New satellites
- @astrojs/starlight
- What an import does
- webhook-health-alerts.mdx
- WebhookService
- proposal.md
- test-ui.sh
- api-types.d.ts
- Feature: Webhook health/delivery-failure email alerts
- introducing-sparrow.md
- docker-compose.md
- subscription-pause.mdx
- registerAlertConfigRoutes
- tasks.md
- github.com/sarathsp06/sparrow
- writerWithBytes
- Sparrow Client Libraries
- 000027_access_tokens.up.sql
- webhook_health_summaries
- quickstart.mdx
- verify-signatures.mdx
- error-classification.md
- cli.mdx
- NewDocFetcher
- schema.sql
- install.sh
- release-submodules.sh
- 000037_health_evaluator.up.sql
- LayeredArchEmbed.astro
- `sparrow tokens`
- index.mdx
- timeoutError
- Requirement: Health transition event
- xorSealer
- anthropicClient
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
- webhookHealthOutput
- run.sh
- sparrow-e2e

## God Nodes (most connected - your core abstractions)
1. `unwrap()` - 97 edges
2. `formatAPIError()` - 85 edges
3. `RepositoryInterfaceWithTracing` - 81 edges
4. `setupEnv()` - 71 edges
5. `EventSubscription` - 70 edges
6. `newRESTClient()` - 54 edges
7. `WebhookServiceInterfaceWithTracing` - 52 edges
8. `EventRegistration` - 51 edges
9. `main()` - 42 edges
10. `testContext()` - 42 edges

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

## Communities (299 total, 107 thin omitted)

### Community 0 - "github.com/google/uuid.UUID"
Cohesion: 0.03
Nodes (18): statusForFailure(), deliveryFilterArgs(), DeliveryRepository, Repository, AutoDisableResult, HealthRepository, Repository, cursorArgs() (+10 more)

### Community 1 - "context.Context"
Cohesion: 0.03
Nodes (22): AlertConfig, AlertConfigRepository, Repository, Repository, AlertConfigManager, BatchManager, DeliveryManager, HealthManager (+14 more)

### Community 2 - "testing.T"
Cohesion: 0.03
Nodes (121): ErrorCategory, sinkSMTPMessage, TestAIConfig(), TestSlogLevel(), TestUIInjectKeyIsRetired(), TestValidate(), TestWarnings(), validConfig() (+113 more)

### Community 3 - "go_pkg_context"
Cohesion: 0.08
Nodes (7): consumerStats, subscriptionResp, applyRecipeIn, recipeNameIn, deliverFunc, sinkDef, validateRecipeIn

### Community 4 - "[webhookId]/+page.svelte"
Cohesion: 0.03
Nodes (57): activeTab, applyDeliveryFilters(), batchStatus, changingSignature, clearDeliveryFilters(), configForm, configHeaderKey, configHeaderValue (+49 more)

### Community 5 - "store/models.go"
Cohesion: 0.18
Nodes (32): eventTypeResp, eventTypeVersionResp, TestHealthLabel(), healthLabel(), alertConfigIDInput, attemptItem, attemptsOutput, createAlertConfigBody (+24 more)

### Community 6 - "Token"
Cohesion: 0.09
Nodes (18): Status, New(), Store, execer, scanner, Store, Invite, Token (+10 more)

### Community 7 - "server/main.go"
Cohesion: 0.11
Nodes (19): oaMessage, oaRequest, oaResponse, capabilitiesOutput, ConsumerItem, consumerOnlyInput, consumerStatsOutput, draftTemplateOutput (+11 more)

### Community 8 - "log/slog.Logger"
Cohesion: 0.03
Nodes (47): main(), RunAllMigrations(), RunAppMigrations(), RunRiverMigrations(), BuildEnvelopePayload(), failing(), newAutoDisableWorker(), TestMaybeAutoDisable_DisablesAndEmits() (+39 more)

### Community 9 - "Errorf"
Cohesion: 0.07
Nodes (18): checkEventForConsumer(), stringHeaders(), validateHeaders(), ValidateWebhookURL(), WebhookService, generateWebhookSecret(), WebhookService, WebhookService (+10 more)

### Community 10 - "setupEnv"
Cohesion: 0.13
Nodes (61): deliveryItem, restClient, testEnv, buildCLI(), runCLI(), TestCLI_E2E(), TestCLI_EventTypeExportImport(), TestE2E_ConsumerDeliveryPaging() (+53 more)

### Community 11 - "webhooks/register/+page.svelte"
Cohesion: 0.03
Nodes (57): Schema (version 1), Recipe, RecipeParam, substituteParams(), active, alertDeliveryConfigured, alertEmail, alertEmailError (+49 more)

### Community 12 - "clientTester"
Cohesion: 0.19
Nodes (8): clientTester, LatencyReservoir, RateLimiter, NewE2ERunner(), main(), newClientTester(), NewLatencyReservoir(), NewRateLimiter()

### Community 13 - "testContext"
Cohesion: 0.06
Nodes (56): Status, GetTracer(), TestAlertDeliveryConfigured(), TestCreateSubscription_CatchAllWithLabelFilters(), TestCreateSubscription_WithEmptyLabelFilters(), TestCreateSubscription_WithInvalidLabelFilters(), TestCreateSubscription_WithLabelFilters(), TestGetSubscriptionsWithWebhooksByEvent_ActiveWebhooksOnly() (+48 more)

### Community 14 - "EventRegistration"
Cohesion: 0.06
Nodes (14): registerSystemEventTypes(), SystemEventRegistrations(), systemEventSamplePayload(), EventTypeFilter, EventTypeRepository, Repository, EventRegistration, EventRegistrationVersion (+6 more)

### Community 15 - "otel.go"
Cohesion: 0.08
Nodes (18): DefaultConfig(), GetMeter(), SparrowMetrics, legacyEndpoint(), MetricsHandler(), newLoggerProvider(), NewSparrowMetrics(), Setup() (+10 more)

### Community 16 - "EventSubscription"
Cohesion: 0.05
Nodes (12): RepositoryInterface, EventSubscription, SubscriptionTemplateVersion, SubscriptionWithWebhook, NewRepositoryInterfaceWithTracing(), Repository, SubscriptionRepository, insertSubscription() (+4 more)

### Community 17 - "access/service.go"
Cohesion: 0.14
Nodes (17): TestMigrationsMatchPgstoreSchema(), TestPgstoreConformance(), cleanPortalPath(), PortalAuthorized(), portalTarget(), portalCtxKey, TestModuleImportsOnlyStdlibAndItself(), newSvc() (+9 more)

### Community 18 - "steps.py"
Cohesion: 0.10
Nodes (48): delivery_has_signature_headers(), _api(), assert_api_delivery_count(), assert_body_field(), assert_body_has_key(), assert_body_not_has_key(), assert_delivery_attempt_count(), assert_delivery_count() (+40 more)

### Community 19 - "TemplateEditor.svelte"
Cohesion: 0.03
Nodes (57): aiAttempts, aiCapability, aiConfirmReplace, aiDocsUrl, aiError, aiInstructions, aiLoading, aiNotes (+49 more)

### Community 20 - "requests"
Cohesion: 0.16
Nodes (45): alert_configs_count(), alert_configs_count_for_webhook(), also_use_consumer(), assert_delivery_page_meta(), _base(), create_alert_config_bad_webhook(), create_alert_config_scoped(), create_alert_config_wide() (+37 more)

### Community 21 - "sources/config.go"
Cohesion: 0.07
Nodes (37): main(), ProvidersConfig, LoadConfig(), cronTick(), parseCron(), parseCronField(), RunCron(), at() (+29 more)

### Community 22 - "NewService"
Cohesion: 0.12
Nodes (38): Key, Keyring, isSafeKeyID(), NewKeyring(), NewService(), NewServiceFromKeyring(), ParseKey(), otherTestKey() (+30 more)

### Community 23 - "webhooks/+page.svelte"
Cohesion: 0.03
Nodes (74): adding, AlertConfigItem, configs, { consumer, webhookId }, deliveryConfigured, email, error, eventLabel (+66 more)

### Community 24 - "email_test.go"
Cohesion: 0.07
Nodes (31): loadConfig(), newEmailSink(), serveSMTP(), startSMTPServer(), TestEmailRender_BadTemplate(), TestEmailRender_CustomTemplatesAndHeaderInjection(), TestEmailRender_Defaults(), TestEmailSink_SendsViaSMTP() (+23 more)

### Community 25 - "io.Writer"
Cohesion: 0.11
Nodes (26): runTokensCreate(), newPalette(), importOutcomeError(), printImportResult(), runEventsExport(), runEventsImport(), runEventsVersions(), indentJSON() (+18 more)

### Community 26 - "testhelpers_test.go"
Cohesion: 0.13
Nodes (9): vector, vectorFile, WebhookEnvelope, TestValidateHeaders(), ValidateHeaders(), validHeaderName(), validHeaderValue(), TestIsSuccessStatusCode() (+1 more)

### Community 27 - "posts.ts"
Cohesion: 0.10
Nodes (28): links, Props, tags, Props, Props, allTags(), base, formatDate() (+20 more)

### Community 28 - "RecipeWorkbenchApp.svelte"
Cohesion: 0.05
Nodes (39): activeRecipe, attempt, b64(), chips, cliCommand, copiedToast, DEST, drafts (+31 more)

### Community 29 - "SubscriptionForm.svelte"
Cohesion: 0.03
Nodes (74): openapi-fetch, apiConsole, ApiLogEntry, apiLogMiddleware, entries, started, toCurl(), copiedId (+66 more)

### Community 30 - "time.Time"
Cohesion: 0.08
Nodes (11): countingStore, downStore, encodeCursor(), newCursorPagination(), FailingWebhook, Repository, downStore, Store (+3 more)

### Community 31 - "Repository"
Cohesion: 0.07
Nodes (9): DeliveryAttempt, Repository, checkWebhookDuplicate(), Repository, WebhookRepository, insertWebhookRegistration(), DB, DBTX (+1 more)

### Community 32 - "use.go"
Cohesion: 0.10
Nodes (28): All(), TestRecipes_ParamGuidance(), TestSendgridRecipe_ConsumerHint(), addRecipeTools(), declaresJSON(), TestResolveParams(), TestValidateRecipe_BuiltinsAreCleanAndValid(), TestValidateRecipe_WholeURLParamCheckedOnceGiven() (+20 more)

### Community 33 - "SparrowVerify.java"
Cohesion: 0.11
Nodes (5): SignatureVerificationException, SparrowVerify, ParsedHeaders, SignatureVerificationException, SparrowVerify

### Community 34 - "Manager"
Cohesion: 0.12
Nodes (7): NewJobInserterWithTracing(), jobInserter, NewJobInserter(), Manager, JobInserterWithTracing, noopJobInserter, mockJobInserter

### Community 35 - "EventTypeDefinition"
Cohesion: 0.08
Nodes (34): BundleDigest(), checkStamp(), checkTemplates(), compileSchema(), generatePayload(), EventTypeImportResult, WebhookService, pauseFailing() (+26 more)

### Community 36 - "deliveries/+page.svelte"
Cohesion: 0.07
Nodes (35): isSystemConsumer(), alertsInScope, allConsumers, applyFilters(), batchStatus, cancelRetryBatch(), clearFilters(), confirmRetry (+27 more)

### Community 37 - "lib/utils.ts"
Cohesion: 0.05
Nodes (56): svelte-jsoneditor, analyzeSamples(), collect(), DATE, detectStringFormat(), emit(), FieldCoverage, inferSchema() (+48 more)

### Community 38 - "template_errors_test.go"
Cohesion: 0.17
Nodes (25): bodyRecorder, bundle, importResult, requiresTransformSub, templateDelivery, exportBundle(), importBundle(), TestEventTypeBundle_AllOrNothingAndBreakingChanges() (+17 more)

### Community 39 - "unwrap"
Cohesion: 0.03
Nodes (88): 1. Payload transform engine: Go `text/template`, not embedded JavaScript, Consequences, Context, Decision, Trigger to revisit, API surface deliberately not e2e-tested (removal / design candidates), Deliberately NOT covered by e2e, Known product-contract questions (asserted as-is, flagged) (+80 more)

### Community 40 - "BuildRequest"
Cohesion: 0.09
Nodes (23): DeliveryRequest, isReservedHeader(), TestBuildRequestHeaderPrecedence(), GetHeaderMap(), PutHeaderMap(), BenchmarkHeaderMapPool(), TestHeaderMapPool(), BuildRequest() (+15 more)

### Community 41 - "Service"
Cohesion: 0.12
Nodes (13): cacheEntry, Config, CreateInviteRequest, CreateTokenRequest, RootKey, Principal, invalidRequest(), clonePtr() (+5 more)

### Community 42 - "[eventId]/+page.svelte"
Cohesion: 0.07
Nodes (21): { pages, shown, onchange, itemLabel = "items" }, Props, CursorPages, consumer, deliveries, deliveriesError, deliveriesLoading, DeliveryItem (+13 more)

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

### Community 47 - "DeliveryTable.svelte"
Cohesion: 0.04
Nodes (45): AttemptItem, attempts, { deliveryId }, error, loading, Props, copied, display (+37 more)

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
Cohesion: 0.06
Nodes (38): { batch, label = 'Batch', oncancel, ondone }, BatchStatus, dismissed, isTerminal, progressPercent, Props, statusColor, statusLabel (+30 more)

### Community 52 - "Sparrow — Detailed Flow Reference"
Cohesion: 0.06
Nodes (30): 1. Webhook Registration, 2. Event Registration, 3. Event Push (PushEvent), 4. Event Processing (EventProcessingWorker), 5. Webhook Delivery (WebhookWorker), 6. Subscription Creation, 7. Webhook Update (UpdateWebhookConfig), DB Writes (+22 more)

### Community 53 - "BatchJob"
Cohesion: 0.07
Nodes (13): NewBatchCleanupWorker(), BatchRepository, Repository, eventVersionOrDefault(), Repository, BatchJob, BatchJobStatus, BatchJobType (+5 more)

### Community 54 - "newRootCmd"
Cohesion: 0.17
Nodes (31): accessLabel(), newInviteCmd(), newInvitesCmd(), newTokensCmd(), newTokensCreateCmd(), newTokensListCmd(), newTokensRevokeCmd(), timeOr() (+23 more)

### Community 56 - "WebhookHTTPConfig"
Cohesion: 0.12
Nodes (8): DefaultWebhookHTTPConfig(), WebhookHTTPConfig, WebhookRegistration, TestDefaultWebhookHTTPConfig_RateLimitRPS(), TestValidateConfig_RateLimitRPS(), IntArray, JSONBMap, StringArray

### Community 57 - "runInvite"
Cohesion: 0.11
Nodes (11): apiClient, parseDuration(), runInvite(), TestParseDuration(), createdInvite, createdToken, durationFlag, inviteOut (+3 more)

### Community 58 - "sparrow/client.go"
Cohesion: 0.12
Nodes (15): apiClient, pushWithAutoCreate(), runPush(), apiError, consumerStats, deliveryItem, eventTypeItem, pushBody (+7 more)

### Community 59 - "webhook_target.py"
Cohesion: 0.08
Nodes (4): CapturedDelivery, _Target, handle(), WebhookTargetManager

### Community 60 - "listDeliveriesImpl"
Cohesion: 0.10
Nodes (25): listDeliveriesImpl(), registerDeliveryRoutes(), toDeliveryItem(), toDeliveryOutput(), listEventOccurrencesImpl(), parseDateFilter(), parseLabelFilter(), TestParseDateFilter() (+17 more)

### Community 61 - "services.ts"
Cohesion: 0.06
Nodes (48): onSignOut(), signingOut, who, auth, KEY_STORAGE, message, OWNED_TOKEN_STORAGE, ownedTokenId (+40 more)

### Community 62 - "access/service_test.go"
Cohesion: 0.27
Nodes (24): clock, New(), New(), newClock(), newSealedService(), newService(), ptr(), reason() (+16 more)

### Community 63 - "astro.config.mjs"
Cohesion: 0.16
Nodes (10): blogLastmod, starlightLlmsTxt(), GET(), prerender, ensureTrailingSlash(), getSiteTitle(), astro, astro-mermaid (+2 more)

### Community 64 - "rest/access.go"
Cohesion: 0.10
Nodes (28): InviteTTL(), TestTTLRules(), TestValidateConsumer(), TokenTTL(), ValidateConsumer(), consumerScope(), fragmentEscape(), mapAccessError() (+20 more)

### Community 65 - "newDrafter"
Cohesion: 0.25
Nodes (10): fakeModel, newDrafter(), TestDraftTemplate_DocsURL(), TestDraftTemplate_GivesUpAfterMaxAttempts(), TestDraftTemplate_RefusalIsFailedPrecondition(), TestDraftTemplate_RendersAgainstRequestPayload(), TestDraftTemplate_RepairsSilentlyWrongField(), TestDraftTemplate_RepairsUntilItRenders() (+2 more)

### Community 66 - "mockRepo"
Cohesion: 0.07
Nodes (11): getWebhookEventsMap(), mockRepo, eventReportFilterArgs(), EventReportFilter, EventReportWithStats, WebhookHealth, WebhookRegistration, normalizePagination() (+3 more)

### Community 67 - "web/package.json"
Cohesion: 0.13
Nodes (16): openapi-typescript, svelte-check, @sveltejs/kit, tailwindcss, @tailwindcss/forms, @tailwindcss/typography, @tailwindcss/vite, @types/node (+8 more)

### Community 68 - "update/+page.svelte"
Cohesion: 0.10
Nodes (24): toJSONObject(), active, confirmPending(), description, error, hasSchema(), ImportItem, loading (+16 more)

### Community 69 - "event.go"
Cohesion: 0.15
Nodes (15): eventIDOnlyInput, eventTypeBody, eventTypeNameInput, eventTypeVersionInput, jobIDInput, listEventTypesInput, patchEventTypeInput, pushEventBody (+7 more)

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
Cohesion: 0.17
Nodes (19): HelperFunc, outlineField, PromptBuilder, RecipeParam, fieldStep(), mustJSON(), NewPromptBuilder(), payloadOutline() (+11 more)

### Community 74 - "httpauth.go"
Cohesion: 0.08
Nodes (37): AuthError, ctxKey, ErrorBody, RedeemRequest, RedeemResponse, Verifier, NewAuth(), apiKey() (+29 more)

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
Cohesion: 0.15
Nodes (15): listSubscriptionsImpl(), registerSubscriptionRoutes(), toSubscriptionItem(), toSubscriptionOutput(), listEventTypesOutput, listSubscriptionsGlobalInput, listSubscriptionsInput, listSubscriptionsOutput (+7 more)

### Community 84 - "api.astro"
Cohesion: 0.10
Nodes (10): out, items, Props, toc, spec, specPath, @scalar/astro, sharp (+2 more)

### Community 85 - "main"
Cohesion: 0.09
Nodes (25): main(), TestE2E_EventDeliveryStats_NoDeliveries(), TestEventTypeVersions_ConcurrentSchemaChanges(), TestAcquireDeliverySlot_BusyBucketDoesNotBurnSlots(), runMigrations(), setupTestDB(), Bootstrap(), Available() (+17 more)

### Community 86 - "SparrowAPI"
Cohesion: 0.16
Nodes (3): `sparrow mcp`, SparrowAPI, _to_dict()

### Community 87 - "New"
Cohesion: 0.18
Nodes (11): Config, Drafter, openAIClient, Provider, Renderer, DocFetcher, New(), ResolveDocs() (+3 more)

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

### Community 92 - "Service"
Cohesion: 0.18
Nodes (7): cryptoSealer, kek, buildKEK(), decryptDataWithDEK(), Service, newAEAD(), unwrapDEK()

### Community 93 - "NewTemplateEngine"
Cohesion: 0.20
Nodes (16): NewTemplateEngine(), BenchmarkExecuteComplex(), BenchmarkExecuteSimple(), TestExecuteComplexTemplate(), TestExecuteEmptyTemplate(), TestExecuteInvalidTemplate(), TestExecuteReshapeFunctions(), TestExecuteSimpleTemplate() (+8 more)

### Community 94 - "mcpListener"
Cohesion: 0.30
Nodes (5): main(), listenStatus, mcpListener, receivedDelivery, startListenIn

### Community 95 - "net/http.Handler"
Cohesion: 0.10
Nodes (32): logCORSMode(), NewWithStore(), Realm(), Sealer(), TestMasterKeyIsRootInDefaultTenant(), CORS(), CORSMode, NormalizeOrigins() (+24 more)

### Community 96 - "SecretRevealer"
Cohesion: 0.11
Nodes (19): derefString(), formatOptionalTime(), maskEncryptedSecret(), maskSecret(), maskSecretHeaders(), toWebhookOut(), toWebhookOutFromDomain(), healthRules() (+11 more)

### Community 97 - "Request"
Cohesion: 0.23
Nodes (4): Request, Result, slowDrafter, stubDrafter

### Community 98 - "Satellite ideas: landscape research and candidates"
Cohesion: 0.13
Nodes (16): 1. What the landscape ships as "ecosystem", 2.1 Sources, 2.2 Sinks, 2.3 Recipes, 2.4 CLI, 2. Extend the existing satellites (lowest effort, highest leverage), 4.1 Helm chart and Grafana pack, 4.2 Typed event payloads from event-type schemas (+8 more)

### Community 99 - "Commands"
Cohesion: 0.12
Nodes (17): Commands, From a template to a recipe, `sparrow events`, `sparrow events export` and `sparrow events import`, `sparrow functions [name]`, `sparrow init`, `sparrow invite`, `sparrow invites` (+9 more)

### Community 101 - "registerEventBundleRoutes"
Cohesion: 0.13
Nodes (15): fromBundleItems(), registerEventBundleRoutes(), toBundleItems(), toImportOutput(), bundleItem, bundleStampBody, eventRouteService, eventTypeBundle (+7 more)

### Community 102 - "ADDED Requirements"
Cohesion: 0.12
Nodes (16): ADDED Requirements, Purpose, Requirement: Alert-email configuration resource, Requirement: Email delivery through the core pipeline, Requirement: Human-readable alert content, Requirement: Recipient resolution at push time, Scenario: Consumer-level opt-in, Scenario: Degradation subject (+8 more)

### Community 103 - "ServiceError"
Cohesion: 0.28
Nodes (5): ServiceError, Error(), TestServiceError_Error(), TestServiceError_Unwrap(), Wrap()

### Community 104 - "TemplateCache"
Cohesion: 0.28
Nodes (5): NewTemplateCache(), TestTemplateCacheBasicOperations(), TestTemplateCacheConcurrency(), TestTemplateCacheLRUEviction(), TemplateCache

### Community 106 - "payload-transformation.mdx"
Cohesion: 0.12
Nodes (15): 1. Send only the fields a partner needs, 2. Post a message to Slack, 3. Convert dollars to cents for a billing system, 4. Flatten a nested payload for a legacy endpoint, 5. Summarize a list of items, 6. Add a constant or computed field, 7. Provide safe defaults for optional fields, Good to know (+7 more)

### Community 107 - "recipes.mdx"
Cohesion: 0.12
Nodes (15): Apply a recipe, clickhouse, cloudevents, discord, ntfy, pagerduty, Pick a recipe, sendgrid (+7 more)

### Community 108 - "middleware/portal_test.go"
Cohesion: 0.36
Nodes (13): NewPortalVerifier(), PortalGateway(), doBearer(), mintConsumerToken(), newPortalGateway(), newPortalService(), TestPortalGatewayAcceptsConsumerAccessTokens(), TestPortalGatewayReDispatchesThroughRouter() (+5 more)

### Community 109 - "NewWebhookTemplateContext"
Cohesion: 0.16
Nodes (16): checkRendered(), TestCheckRendered(), TestTemplateReference_ExampleRenders(), TestRenderTemplatePreview_StrictFlag(), RenderTemplatePreview(), TestTransformPayloadWith_CachesStrictAndLenientSeparately(), TestTransformPayloadWith_StrictMissingKeys(), NewWebhookTemplateContext() (+8 more)

### Community 110 - "devDependencies"
Cohesion: 0.12
Nodes (16): devDependencies, openapi-typescript, @playwright/test, svelte, svelte-check, @sveltejs/adapter-static, @sveltejs/kit, @sveltejs/vite-plugin-svelte (+8 more)

### Community 111 - "webhookHTTPConfig"
Cohesion: 0.33
Nodes (5): patchWebhookBody, patchWebhookInput, registerWebhookBody, registerWebhookInput, webhookHTTPConfig

### Community 113 - "event_records"
Cohesion: 0.14
Nodes (13): event_records, idx_event_records_consumer, idx_event_records_created_at, idx_event_records_event, idx_event_records_expires_at, idx_event_records_tenant, idx_event_records_tenant_event_created, idx_event_records_tenant_ns_created (+5 more)

### Community 114 - "go-template-engine.ts"
Cohesion: 0.26
Nodes (14): renderResult, applyFilterStage(), evaluateExpression(), evaluateToken(), EventContext, extractIfElseBlocks(), extractMatchingEnd(), goPrintf() (+6 more)

### Community 115 - "access.md"
Cohesion: 0.13
Nodes (14): API endpoints, CI and machine tokens, Consumer (portal) access, First run, From the API, From the CLI, From the web UI, Invite a teammate (+6 more)

### Community 116 - "TemplateEngine"
Cohesion: 0.18
Nodes (9): GetBuffer(), PutBuffer(), BenchmarkBufferPool(), TestBufferPool(), hashTemplate(), TestHashTemplate(), getBuffer(), putBuffer() (+1 more)

### Community 117 - "sparrow/config.go"
Cohesion: 0.20
Nodes (11): configPath(), resolveConfig(), saveConfig(), TestResolveConfigDefaults(), TestResolveConfigPrecedence(), TestSaveConfigPermissions(), writeConfigFile(), isTerminal() (+3 more)

### Community 118 - "Mount"
Cohesion: 0.12
Nodes (18): main(), buildDraftRequest(), extendWriteDeadline(), TemplateDrafter, registerAIRoutes(), registerPromptRoute(), draftOverServer(), TestDraftTemplate_OutlivesServerWriteTimeout() (+10 more)

### Community 119 - "registerEventRoutes"
Cohesion: 0.14
Nodes (16): registerEventRoutes(), toBatchJobOutput(), toEventTypeChange(), toEventTypeItem(), toEventTypeOutput(), toEventTypeVersionItem(), ValidateJSONSchema(), batchJobOutput (+8 more)

### Community 121 - "AGENTS.md — Sparrow"
Cohesion: 0.05
Nodes (34): AGENTS.md — Sparrow, Architecture, Authentication, Code conventions, DB / Migrations, Design Principles, First reads, Frontend (Svelte 5) (+26 more)

### Community 123 - "WebhookClient"
Cohesion: 0.15
Nodes (6): cancelOnClose, WebhookClient, ExecOptions, sampleContext(), TestRecipes(), tokenRefs()

### Community 124 - "Config"
Cohesion: 0.11
Nodes (12): Config, Load(), validatePort(), DefaultConfig(), Config, TestDefaultConfig(), mustCIDRs(), ParseNetworks() (+4 more)

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

### Community 130 - "svelte.config.js"
Cohesion: 0.40
Nodes (3): @sveltejs/adapter-static, @sveltejs/vite-plugin-svelte, config

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
Cohesion: 0.18
Nodes (11): completer, completion, draft, turn, mapStatus(), mapTransportError(), parseDraft(), TestParseDraft_ToleratesFencesAndPreamble() (+3 more)

### Community 136 - "event_registrations"
Cohesion: 0.17
Nodes (10): event_registrations, idx_event_registrations_created_at, idx_event_registrations_name, update_event_registrations_updated_at, update_updated_at_column(), update_webhook_registrations_updated_at, idx_event_registrations_tenant, idx_event_registrations_tenant_name (+2 more)

### Community 137 - "Recipe"
Cohesion: 0.18
Nodes (10): Recipe, findRecipe(), toAIRecipe(), Subscription, Webhook, listRecipesOutput, ConsumerHint, Param (+2 more)

### Community 138 - "emitHealthChangedEvent"
Cohesion: 0.23
Nodes (11): emitHealthChangedEvent(), newTestWorker(), TestEmitDeliveryFailedEvent_EmitsWhenRecipientsOptedIn(), TestEmitDeliveryFailedEvent_EmitsWithZeroRecipients(), TestEmitDeliveryFailedEvent_SkipsSparrowConsumer(), TestEmitHealthChangedEvent_EmitsOnRealTransition(), TestEmitHealthChangedEvent_EmitsOnRecipientLookupError(), TestEmitHealthChangedEvent_EmitsWithZeroRecipients() (+3 more)

### Community 139 - ".compare"
Cohesion: 0.21
Nodes (9): ClassifySchemaChange(), firstComposition(), joinPath(), requiredSet(), sortedKeys(), typeAllows(), typeChange(), typesOf() (+1 more)

### Community 141 - "Sparrow Recipes"
Cohesion: 0.29
Nodes (6): Contributing a recipe, Guidance, How params work, Included recipes, Requiring the transform, Sparrow Recipes

### Community 142 - "events_bundle_test.go"
Cohesion: 0.44
Nodes (11): runCLICmd(), startBundleServer(), TestEventsExport(), TestEventsExportNeedsOneSelector(), TestEventsImport_Applied(), TestEventsImport_BlockedExitsNonZero(), TestEventsImport_DryRun(), TestEventsImport_Stdin() (+3 more)

### Community 143 - "dependencies"
Cohesion: 0.67
Nodes (3): dependencies, openapi-fetch, svelte-jsoneditor

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
Cohesion: 0.18
Nodes (10): Before you start, Configuration, email, How a delivery is handled, and what Sparrow does with the answer, Install and run, otlp, Pick a sink, s3 (+2 more)

### Community 149 - "net/http.Header"
Cohesion: 0.36
Nodes (6): capturedWebhook, parseHeaders(), signedMessage(), VerifyEd25519(), VerifyHMAC(), Verifier

### Community 151 - "NetworkPolicy"
Cohesion: 0.24
Nodes (5): embeddedIPv4(), NetworkPolicy, isMetadataIP(), TestValidateIP(), ValidateIP()

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

### Community 158 - "runTemplateTest"
Cohesion: 0.67
Nodes (3): parseJSONArg(), TestParseJSONArg(), runTemplateTest()

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
Cohesion: 0.20
Nodes (9): EventPreset, ORDER, parsed, rawFiles, RECIPE_META, RecipeDef, RecipeParam, RECIPES (+1 more)

### Community 167 - "security_headers_test.go"
Cohesion: 0.39
Nodes (7): buildCSP(), SecurityHeaders(), TestBuildCSP(), TestSecurityHeadersNoHashes(), TestSecurityHeadersPassthrough(), TestSecurityHeadersPortalAllowsFraming(), TestSecurityHeadersWithHashes()

### Community 168 - "ADDED Requirements"
Cohesion: 0.22
Nodes (8): ADDED Requirements, Purpose, Requirement: Feedback-loop guard, Requirement: Internal `_sparrow` consumer, Requirement: Registered system event types, Scenario: Idempotent registration, Scenario: Internal channel failure does not recurse, Scenario: System events are invisible to tenants

### Community 171 - "sink_test.go"
Cohesion: 0.53
Nodes (8): postSink(), signHeaders(), TestSinkHandler_DownstreamFailureIs502(), TestSinkHandler_MissingHeaders(), TestSinkHandler_TamperedBody(), TestSinkHandler_TransformedBodyIs422(), TestSinkHandler_ValidSignature(), TestSinkHandler_WrongSecret()

### Community 172 - "seed.sql"
Cohesion: 0.22
Nodes (8): consumer_pool, evt_ids, evt_names, hot_webhook, recent_delivery_ids, sub_ids, sub_lookup, wh_ids

### Community 173 - "Sparrow Web Dashboard"
Cohesion: 0.22
Nodes (8): Build, Configuration, Embedding in the Go Binary, Local Development, Prerequisites, Sparrow Web Dashboard, Standalone Deployment, Tech Stack

### Community 175 - "2. Split the `sparrow` CLI into its own module to isolate its `go install` graph"
Cohesion: 0.25
Nodes (7): 2. Split the `sparrow` CLI into its own module to isolate its `go install` graph, Consequences, Context, Decision, Releasing the split modules (tag scheme), Trigger to revisit, When this is worth it

### Community 176 - "installation.mdx"
Cohesion: 0.25
Nodes (7): Build, Build from Source, Docker Compose (Recommended), Docker Image, Pre-built Binaries, Prerequisites, Run

### Community 177 - "event-type-versioning.mdx"
Cohesion: 0.25
Nodes (7): Breaking changes, Inferring a schema from real events, Reading the history, Reserved names, Retiring an event type, Unregistered event names, What creates a new version

### Community 178 - "NewTemplateEngineWithCacheSize"
Cohesion: 0.25
Nodes (7): GetFunctionMap(), GetTemplateFunctions(), TestTitleFunc(), toNumber(), NewTemplateEngineWithCacheSize(), TestNewTemplateEngineWithCacheSize(), TemplateFunc

### Community 180 - "time.Duration"
Cohesion: 0.21
Nodes (10): BacklogSample, ClientConfig, ClientResults, E2EConfig, E2EResults, LatencySummary, printLatency(), NewTokenPurgeWorker() (+2 more)

### Community 181 - "3. New satellites"
Cohesion: 0.29
Nodes (7): 3.1 `sparrow-bus`: message-bus bridge, both directions, 3.2 `sparrow-cdc`: Postgres change capture as events, 3.3 `sparrow mcp`: MCP server for coding agents and operators, 3.4 `sparrow-inbox`: durable consumer-side receiver, 3.5 `sparrow-capture`: a self-hosted webhook.site, 3.6 Sparrow-to-Sparrow federation, 3. New satellites

### Community 182 - "@astrojs/starlight"
Cohesion: 0.33
Nodes (3): buildDate, collections, @astrojs/starlight

### Community 183 - "What an import does"
Cohesion: 0.29
Nodes (6): API, Subscriptions whose template fails, The bundle file, The preview, What an import does, When nothing is written

### Community 184 - "webhook-health-alerts.mdx"
Cohesion: 0.29
Nodes (6): Automatic disabling, Check that alerts are being sent, Good to know, How it works, Register an alert config, Set up the email delivery channel

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

### Community 197 - "registerAlertConfigRoutes"
Cohesion: 0.33
Nodes (5): registerAlertConfigRoutes(), toAlertConfigItem(), alertConfigItem, alertConfigOutput, listAlertConfigsOutput

### Community 198 - "tasks.md"
Cohesion: 0.33
Nodes (5): 1. Foundation, 2. Event emission, 3. Tenant-facing API, 4. Delivery channel, 5. Verification & docs

### Community 199 - "github.com/sarathsp06/sparrow"
Cohesion: 0.53
Nodes (6): github.com/sarathsp06/sparrow, github.com/sarathsp06/sparrow/pkg/access, github.com/sarathsp06/sparrow/pkg/signature, github.com/sarathsp06/sparrow/pkg/template, github.com/sarathsp06/sparrow/satellites/recipes, github.com/sarathsp06/sparrow/satellites/sparrow

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

### Community 210 - "NewDocFetcher"
Cohesion: 0.20
Nodes (8): redactSpanURL, collapseSpace(), htmlToText(), looksLikeHTML(), NewDocFetcher(), TestDocFetcher_ExtractsTextAndRespectsPolicy(), RedactURL(), TestRedactURL()

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

## Knowledge Gaps
- **30 isolated node(s):** `InvalidTimestamp`, `TimestampOutsideTolerance`, `EmptySecret`, `InvalidSecretBase64`, `InvalidPublicKey` (+25 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 1765 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **107 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `yaml` connect `docs/src/lib/recipes.ts` to `docs/package.json`, `astro.config.mjs`?**
  _High betweenness centrality (0.010) - this node is a cross-community bridge._
- **What connects `InvalidTimestamp`, `TimestampOutsideTolerance`, `EmptySecret` to the rest of the system?**
  _30 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `github.com/google/uuid.UUID` be split into smaller, more focused modules?**
  _Cohesion score 0.03235294117647059 - nodes in this community are weakly interconnected._
- **Why does `lifetimeFlag` connect `runInvite` to `io.Writer`, `time.Duration`, `newRootCmd`?**
  _High betweenness centrality (0.009) - this node is a cross-community bridge._
- **Should `context.Context` be split into smaller, more focused modules?**
  _Cohesion score 0.02882751937984496 - nodes in this community are weakly interconnected._
- **Why does `CursorPages` connect `[eventId]/+page.svelte` to `deliveries/+page.svelte`, `[webhookId]/+page.svelte`, `reports/+page.svelte`, `AGENTS.md — Sparrow`, `ReadingProgress.astro`, `api.ts`?**
  _High betweenness centrality (0.009) - this node is a cross-community bridge._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.025701547338054027 - nodes in this community are weakly interconnected._