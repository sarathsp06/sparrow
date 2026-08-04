# Graph Report - .  (2026-08-04)

## Corpus Check
- 281 files · ~208,410 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3312 nodes · 6424 edges · 246 communities (174 shown, 72 thin omitted)
- Extraction: 92% EXTRACTED · 8% INFERRED · 0% AMBIGUOUS · INFERRED: 521 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- Client ·  Templates · Js · Connect
- Client ·  Templates · Python
- Client · Js · Connect
- Client · Python
- Migration
- Integration Tests & Middleware
- Db · Migrations
- Db · Migrations
- Db · Migrations
- Docs
- Docs · Src · Data · Api
- Docs · Src · Components
- Docs · Components
- Docs · Src
- Docs · Src · Pages
- Docs
- E2E Signature Verification
- E2E · Step Impl
- E2E · Libs
- E2E · Libs
- E2E · Libs
- E2E · Libs
- E2E
- E2E
- Examples
- Community 242
- Config
- Connect
- Connect
- Proto
- Connect
- Connect
- Connect
- Connect
- Connect
- Connect
- Connect
- Connect
- Connect
- Connect
- Connect
- Connect
- Connect
- Connect
- Connect
- Connect
- Connect
- Connect
- Connect
- Connect
- Connect
- Connect
- Grpc
- Grpc
- Grpc
- Grpc
- Grpc
- Grpc
- Grpc
- Proto↔Domain Conversions
- Grpc
- Health
- Observability
- Store Tenant & Repository
- Webhook Service OTel
- Webhooks · Client
- Webhooks · Client
- Webhooks · Client
- Webhooks · Client
- Webhooks · Client
- Webhooks
- Webhooks · Client
- Webhooks · Client
- Webhooks · Client
- Webhooks · Client
- Event Filtering Tests
- Webhooks · Queue
- Webhooks · Queue
- Webhooks · Store
- Webhooks · Queue
- Webhooks · Queue
- Webhooks · Queue
- Webhooks · Queue
- Webhooks · Queue
- Webhooks · Queue
- Webhooks · Store
- Webhooks Store Layer
- Webhooks · Store
- Webhooks · Store
- Webhooks · Store
- Webhooks · Store
- Delivery Repository
- Webhooks · Store
- Webhooks
- Webhooks
- Webhooks
- Errors
- Community 140
- Envelope Encryption (crypto)
- Errors
- Proto · Protoconnect
- Proto · Protoconnect
- Proto · Protoconnect
- Connect-RPC Clients
- Proto · Protoconnect
- Proto · Protoconnect
- Proto
- Proto Enums
- Proto Event Report Listing
- Proto
- Proto
- Proto
- Proto
- Proto EventSubscription
- Proto
- Proto
- Proto Subscription Messages
- Proto
- Proto
- Proto
- Proto
- Proto
- Proto
- Proto PushEvent
- Proto
- Proto
- Proto
- Proto EventReport
- Proto
- Proto
- Proto
- Proto
- Proto
- Proto
- Proto
- Proto
- Proto
- Proto
- Proto
- Proto
- Proto
- Proto
- Proto
- Proto
- Proto
- Proto Service Clients
- Proto
- Proto
- Proto
- Proto Delivery Service
- Proto Event Service
- Proto
- Proto
- Proto
- Proto
- Proto
- Proto
- Proto Webhook Types (TS)
- Proto Schemas (TS)
- Web · Src · Lib
- Web
- Web
- Web
- Web
- Web
- Web
- Web
- Web
- Web
- Web · Src
- Web · Src · Lib
- Web · Src · Routes
- Web
- Web
- .Github · Workflows
- .Github · Workflows
- Community 82
- Community 126
- Community 166
- Community 164
- Community 244
- Templates
- Templates
- Templates
- Docs & Deployment Guides
- Docs · Guides
- Docs · Reference

## God Nodes (most connected - your core abstractions)
1. `Errorf()` - 122 edges
2. `file_proto_webhook_proto_rawDescGZIP()` - 92 edges
3. `RepositoryInterfaceWithTracing` - 69 edges
4. `WebhookService` - 56 edges
5. `forwardUnary()` - 43 edges
6. `WebhookServiceInterfaceWithTracing` - 40 edges
7. `WebhookConnectServer` - 39 edges
8. `EventServiceClient` - 39 edges
9. `toGRPCError()` - 38 edges
10. `setupEnv()` - 37 edges

## Surprising Connections (you probably didn't know these)
- `Sparrow Webhook Delivery Platform` --conceptually_related_to--> `Svelte 5 Tutorial (PDF)`  [INFERRED]
  README.md → book/svelte5-tutorial.pdf
- `SSRF Protection` --semantically_similar_to--> `Sparrow NetworkPolicy Template`  [INFERRED] [semantically similar]
  README.md → charts/sparrow/templates/networkpolicy.yaml
- `Standalone Docker Compose` --semantically_similar_to--> `Development Docker Compose`  [INFERRED] [semantically similar]
  deploy/docker-compose.yml → docker-compose.dev.yml
- `main()` --calls--> `RunAllMigrations()`  [INFERRED]
  cmd/migrate/main.go → internal/migration/migrate.go
- `GetMigrationsFS()` --calls--> `RunAppMigrations()`  [INFERRED]
  db/migrations.go → internal/migration/migrate.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **CI Pipeline Stage Dependency Chain** — github_workflows_ci_lint_job, github_workflows_ci_test_job, github_workflows_ci_build_job, github_workflows_ci_integration_job [EXTRACTED 1.00]
- **Tag-triggered Release Automation** — github_workflows_release_goreleaser_job, github_workflows_release_docker_job, goreleaser_goreleaser_config, charts_sparrow_chart [EXTRACTED 1.00]
- **Security-hardened Kubernetes Deployment** — charts_sparrow_templates_deployment_deployment, charts_sparrow_templates_networkpolicy_networkpolicy, charts_sparrow_templates_configmap_configmap, charts_sparrow_templates_postgresql_service_service [INFERRED 0.85]
- **River-backed async delivery pipeline** — concept_river_queue, concept_event_processing_worker, concept_webhook_worker, concept_postgresql [EXTRACTED 1.00]
- **Sparrow RPC service surface** — concept_webhook_service, concept_event_service, concept_subscription_service, concept_delivery_service, concept_health_service [EXTRACTED 1.00]
- **Sparrow security feature set** — concept_envelope_encryption, concept_standard_webhooks_signing, concept_ssrf_protection [EXTRACTED 0.95]
- **Sparrow gRPC Services** — docs_src_content_docs_reference_api_webhook_service_webhookservice, docs_src_content_docs_reference_api_event_service_eventservice, docs_src_content_docs_reference_api_subscription_service_subscriptionservice, docs_src_content_docs_reference_api_delivery_service_deliveryservice, docs_src_content_docs_reference_api_health_service_healthservice [EXTRACTED 1.00]
- **Event Delivery Pipeline** — docs_src_content_docs_reference_architecture_event_processing_pipeline, okf_architecture_data_flow_event_pipeline_data_flow, okf_concepts_delivery_delivery, docs_src_content_docs_reference_architecture_health_state_machine [INFERRED 0.85]
- **Webhook Delivery Domain** — okf_concepts_webhook_registration_webhook_registration, okf_concepts_subscription_subscription, okf_concepts_event_event [INFERRED 0.85]
- **Webhook Security Controls** — okf_concepts_payload_signing_payload_signing, okf_concepts_rate_limiting_rate_limiting, okf_concepts_webhook_registration_webhook_registration [INFERRED 0.75]
- **Deployment Pipeline** — okf_devops_docker_docker_build, okf_devops_helm_chart_helm_chart, okf_devops_ci_cd_ci_cd_and_release [INFERRED 0.85]

## Communities (246 total, 72 thin omitted)

### Community 32 - "Client ·  Templates · Js · Connect"
Cohesion: 0.07
Nodes (29): name, version, description, type, license, repository, type, url (+21 more)

### Community 31 - "Client · Js · Connect"
Cohesion: 0.07
Nodes (29): name, version, description, type, license, repository, type, url (+21 more)

### Community 116 - "Migration"
Cohesion: 0.46
Nodes (6): main(), RunRiverMigrations(), Context, Logger, RunAppMigrations(), RunAllMigrations()

### Community 0 - "Integration Tests & Middleware"
Cohesion: 0.05
Nodes (84): GetMigrationsFS(), FS, HandlerFunc, startFailThenSucceedTarget(), T, Server, Int32, startAlwaysFailTarget() (+76 more)

### Community 44 - "Db · Migrations"
Cohesion: 0.11
Nodes (18): event_registrations, webhook_registrations, event_subscriptions, event_records, webhook_deliveries, webhook_health_events, webhook_health_summaries, webhook_health_state (+10 more)

### Community 47 - "Docs"
Cohesion: 0.08
Nodes (23): name, type, version, private, scripts, diagrams, diagrams:check, diagrams:auto (+15 more)

### Community 26 - "Docs · Src · Data · Api"
Cohesion: 0.09
Nodes (17): buildExampleObject(), generateCurl(), generateResponseJson(), ../../../data/api/webhook-service, service, enumData, enumData, service (+9 more)

### Community 7 - "E2E Signature Verification"
Cohesion: 0.09
Nodes (53): verify_hmac_signature(), verify_ed25519_signature(), delivery_has_signature_headers(), SignatureVerifier -- Verifies HMAC-SHA256 (v1,) and Ed25519 (v1a,) signatures…, Verify HMAC-SHA256 signature (v1, prefix)., Verify Ed25519 signature (v1a, prefix)., Assert delivery has Standard Webhooks signature headers., _api() (+45 more)

### Community 103 - "E2E · Step Impl"
Cohesion: 0.18
Nodes (8): SparrowAPI -- HTTP client for the Sparrow Connect-RPC API., SparrowEnvironment -- Manages Postgres and Sparrow containers via…, after_suite, teardown_environment(), after_scenario, teardown_scenario(), Gauge hooks for suite/scenario setup and teardown., Stop all mock targets.

### Community 67 - "E2E · Libs"
Cohesion: 0.12
Nodes (4): SparrowAPI, Client for Sparrow Connect-RPC endpoints., Poll until all deliveries reach terminal status., Poll a single delivery until terminal.

### Community 104 - "E2E · Libs"
Cohesion: 0.22
Nodes (6): SparrowEnvironment, Manages the Sparrow test environment (Postgres + Sparrow containers)., Start Postgres + Sparrow. Returns the Sparrow HTTP URL., before_suite, setup_environment(), Start Postgres + Sparrow containers.

### Community 98 - "E2E · Libs"
Cohesion: 0.23
Nodes (4): CapturedDelivery, _Target, WebhookTargetServer -- Programmable mock webhook endpoints for e2e tests. Each…, Start a mock webhook target. Returns the URL.

### Community 89 - "E2E · Libs"
Cohesion: 0.15
Nodes (5): WebhookTargetManager, Manages mock webhook target servers., before_scenario, setup_scenario(), Fresh target manager for each scenario.

### Community 138 - "E2E"
Cohesion: 0.50
Nodes (3): Language, Plugins, html-report

### Community 139 - "Examples"
Cohesion: 0.83
Nodes (3): MainGRPC(), extractFirstLine(), main()

### Community 128 - "Config"
Cohesion: 0.47
Nodes (3): Config, Load(), validatePort()

### Community 105 - "Connect"
Cohesion: 0.20
Nodes (8): grpcUnary, forwardUnary(), Req, Resp, GetRepushStatusRequest, GetRepushStatusResponse, RetryDeliveryRequest, RetryDeliveryResponse

### Community 107 - "Connect"
Cohesion: 0.20
Nodes (7): WebhookConnectServer, RegisterEventRequest, RegisterEventResponse, GetEventRecordRequest, GetEventRecordResponse, CreateSubscriptionRequest, CreateSubscriptionResponse

### Community 84 - "Proto"
Cohesion: 0.27
Nodes (13): grpcServerWrapper, NewWebhookConnectServer(), WebhookServiceServer, RegisterWebhookServiceServer(), ServiceRegistrar, EventServiceServer, RegisterEventServiceServer(), SubscriptionServiceServer (+5 more)

### Community 94 - "Connect"
Cohesion: 0.15
Nodes (9): Context, GetTemplateFunctionsRequest, GetTemplateFunctionsResponse, UpdateEventRequest, UpdateEventResponse, PushEventRequest, PushEventResponse, GetSubscriptionRequest (+1 more)

### Community 93 - "Connect"
Cohesion: 0.15
Nodes (9): Request, GetEventRequest, GetEventResponse, ListEventReportsRequest, ListEventReportsResponse, RePushEventRequest, RePushEventResponse, GetDeliveryStatusRequest (+1 more)

### Community 92 - "Connect"
Cohesion: 0.15
Nodes (9): Response, ResumeWebhookRequest, ResumeWebhookResponse, ListDeliveriesRequest, ListDeliveriesResponse, CancelRetryRequest, CancelRetryResponse, ListWebhooksByHealthRequest (+1 more)

### Community 46 - "Grpc"
Cohesion: 0.12
Nodes (18): WebhookServer, Context, GetDeliveryStatusRequest, GetDeliveryStatusResponse, ListDeliveriesRequest, ListDeliveriesResponse, RetryDeliveryRequest, RetryDeliveryResponse (+10 more)

### Community 35 - "Grpc"
Cohesion: 0.09
Nodes (20): WebhookServer, Context, PushEventRequest, PushEventResponse, RegisterEventRequest, RegisterEventResponse, UpdateEventRequest, UpdateEventResponse (+12 more)

### Community 95 - "Grpc"
Cohesion: 0.15
Nodes (10): ListEventsRequest, ListEventsResponse, GetEventRecordRequest, GetEventRecordResponse, ListEventReportsRequest, ListEventReportsResponse, convertMapToStruct(), Struct (+2 more)

### Community 85 - "Grpc"
Cohesion: 0.18
Nodes (10): WebhookServer, Context, GetWebhookHealthRequest, GetWebhookHealthResponse, GetHealthSummaryRequest, GetHealthSummaryResponse, ListWebhooksByHealthRequest, ListWebhooksByHealthResponse (+2 more)

### Community 53 - "Grpc"
Cohesion: 0.14
Nodes (22): convertDeliveryStatus(), WebhookDeliveryStatus, convertWebhookHealth(), WebhookHealth, convertExpectedStatusCodes(), convertTimeToProto(), Time, Timestamp (+14 more)

### Community 49 - "Grpc"
Cohesion: 0.11
Nodes (17): convertStatusCodesToInt(), WebhookServer, Context, RegisterWebhookRequest, RegisterWebhookResponse, UnregisterWebhookRequest, UnregisterWebhookResponse, ListWebhooksRequest (+9 more)

### Community 52 - "Grpc"
Cohesion: 0.11
Nodes (16): WebhookServer, Context, CreateSubscriptionRequest, CreateSubscriptionResponse, GetSubscriptionRequest, GetSubscriptionResponse, ListSubscriptionsRequest, ListSubscriptionsResponse (+8 more)

### Community 10 - "Proto↔Domain Conversions"
Cohesion: 0.08
Nodes (33): ConvertProtoHTTPConfig(), WebhookHTTPConfig, CreateWebhookRegistrationRequest(), RegisterWebhookRequest, float32PtrToFloat64Ptr(), float64PtrToFloat32Ptr(), TestFloat32PtrToFloat64Ptr(), T (+25 more)

### Community 129 - "Grpc"
Cohesion: 0.40
Nodes (4): WebhookServer, NewWebhookServer(), Context, WebhookRegistration

### Community 99 - "Health"
Cohesion: 0.32
Nodes (7): Checker, Pool, Time, NewChecker(), HealthResponse, ReadyResponse, Context

### Community 55 - "Observability"
Cohesion: 0.16
Nodes (21): Config, Duration, DefaultConfig(), Setup(), Context, setupTracing(), Resource, TracerProvider (+13 more)

### Community 14 - "Store Tenant & Repository"
Cohesion: 0.12
Nodes (22): Bootstrap(), Context, Repository, NewRepository(), Context, Tx, UUID, WebhookRegistration (+14 more)

### Community 12 - "Webhook Service OTel"
Cohesion: 0.08
Nodes (9): WebhookServiceInterfaceWithTracing, Span, NewWebhookServiceInterfaceWithTracing(), Context, Time, WebhookRegistration, WebhookDelivery, EventSubscription (+1 more)

### Community 48 - "Webhooks · Client"
Cohesion: 0.16
Nodes (20): WebhookClient, Client, Config, NewWebhookClient(), WebhookTemplateContext, ReadBody(), TestNewWebhookClient(), T (+12 more)

### Community 39 - "Webhooks · Client"
Cohesion: 0.13
Nodes (24): Context, Response, Duration, DeliveryRequest, UUID, Duration, WebhookEnvelope, BuildEnvelopePayload() (+16 more)

### Community 115 - "Webhooks · Client"
Cohesion: 0.32
Nodes (6): Config, Duration, DefaultConfig(), TestDefaultConfig(), T, TestCustomConfig()

### Community 66 - "Webhooks · Client"
Cohesion: 0.20
Nodes (13): Metrics, Mutex, Duration, NewMetrics(), TestNewMetrics(), T, TestRecordRequest(), TestRecordSuccess() (+5 more)

### Community 90 - "Webhooks · Client"
Cohesion: 0.28
Nodes (11): GetBuffer(), Buffer, PutBuffer(), GetHeaderMap(), PutHeaderMap(), TestBufferPool(), T, TestHeaderMapPool() (+3 more)

### Community 51 - "Webhooks"
Cohesion: 0.15
Nodes (12): ValidateIP(), IP, validateRedirectURL(), ssrfSafeCheckRedirect(), Request, ssrfDialControl(), RawConn, parseUUID() (+4 more)

### Community 91 - "Webhooks · Client"
Cohesion: 0.21
Nodes (7): limitedWriter, writerWithBytes, TemplateEngine, FuncMap, Template, NewWebhookTemplateContext(), WebhookTemplateContext

### Community 72 - "Webhooks · Client"
Cohesion: 0.27
Nodes (16): NewTemplateEngine(), NewTemplateEngineWithCacheSize(), TestNewTemplateEngine(), T, TestExecuteSimpleTemplate(), TestExecuteEmptyTemplate(), TestExecuteWithTemplateFunctions(), TestExecuteInvalidTemplate() (+8 more)

### Community 83 - "Webhooks · Client"
Cohesion: 0.24
Nodes (10): TemplateCache, Cache, Template, NewTemplateCache(), hashTemplate(), TestTemplateCacheBasicOperations(), T, TestTemplateCacheLRUEviction() (+2 more)

### Community 127 - "Webhooks · Client"
Cohesion: 0.53
Nodes (5): TemplateFunc, GetTemplateFunctions(), GetFunctionMap(), FuncMap, GetFunctionDocumentation()

### Community 3 - "Event Filtering Tests"
Cohesion: 0.07
Nodes (61): TestValidateLabels_EmptyMap(), T, TestValidateLabels_ValidEntries(), TestValidateLabels_EmptyKey(), TestValidateLabels_KeyTooLong(), TestValidateLabels_KeyExactlyAtLimit(), TestValidateLabels_InvalidKeyCharacters(), TestValidateLabels_ValidKeyCharacters() (+53 more)

### Community 108 - "Webhooks · Queue"
Cohesion: 0.31
Nodes (7): JobInserterWithTracing, JobInserter, Span, NewJobInserterWithTracing(), Context, JobArgs, JobInsertResult

### Community 68 - "Webhooks · Queue"
Cohesion: 0.18
Nodes (11): BatchJobWorker, WorkerDefaults, Logger, JobInserter, NewBatchJobWorker(), Context, Job, UUID (+3 more)

### Community 56 - "Webhooks · Store"
Cohesion: 0.13
Nodes (14): EventProcessingWorker, WorkerDefaults, Logger, JobInserter, NewEventProcessingWorker(), NewWebhookWorker(), Config, EventRepository (+6 more)

### Community 101 - "Webhooks · Queue"
Cohesion: 0.18
Nodes (6): Context, Job, EventArgs, Time, InsertOpts, WebhookArgs

### Community 102 - "Webhooks · Queue"
Cohesion: 0.32
Nodes (8): jobInserter, Client, Tx, Logger, NewJobInserter(), Context, JobArgs, JobInsertResult

### Community 96 - "Webhooks · Queue"
Cohesion: 0.22
Nodes (9): Manager, Client, Tx, Pool, Logger, NewManager(), Context, Config (+1 more)

### Community 60 - "Webhooks · Queue"
Cohesion: 0.18
Nodes (16): TestParseRetryAfter(), T, TestParseRetryAfter_HTTPDate(), TestParseRetryAfter_HTTPDate_Past(), TestParseRetryAfter_HTTPDate_FarFuture(), TestDefaultAndMaxRetryAfterConstants(), TestIsSuccessStatusCode(), WebhookWorker (+8 more)

### Community 5 - "Webhooks Store Layer"
Cohesion: 0.10
Nodes (13): RepositoryInterfaceWithTracing, Span, Context, UUID, Time, WebhookDelivery, EventSubscription, WebhookHealth (+5 more)

### Community 64 - "Webhooks · Store"
Cohesion: 0.20
Nodes (9): Repository, Context, UUID, BatchJobStatus, BatchJobStatus, BatchJobType, BatchJobData, BatchJob (+1 more)

### Community 50 - "Webhooks · Store"
Cohesion: 0.14
Nodes (17): WebhookHealth, SignatureType, WebhookRegistration, UUID, Int64Array, Time, WebhookDelivery, WebhookDeliveryStatus (+9 more)

### Community 61 - "Webhooks · Store"
Cohesion: 0.21
Nodes (6): Repository, Context, UUID, EventRecord, EventReportWithStats, EventReportFilter

### Community 78 - "Webhooks · Store"
Cohesion: 0.28
Nodes (4): Repository, Context, UUID, EventRegistration

### Community 19 - "Delivery Repository"
Cohesion: 0.13
Nodes (12): Repository, Context, UUID, WebhookDelivery, WebhookDeliveryStatus, Repository, Context, UUID (+4 more)

### Community 97 - "Webhooks · Store"
Cohesion: 0.44
Nodes (4): Repository, Context, UUID, EventSubscription

### Community 75 - "Webhooks"
Cohesion: 0.19
Nodes (7): ValidateWebhookURL(), Time, WebhookRegistration, WebhookHealthData, WebhookHealth, generateWebhookSecret(), GenerateKey()

### Community 45 - "Webhooks"
Cohesion: 0.10
Nodes (20): WebhookServiceInterface, EventService, SubscriptionService, DeliveryService, HealthService, WebhookRegistrationService, EventService, SubscriptionService (+12 more)

### Community 27 - "Webhooks"
Cohesion: 0.13
Nodes (7): WebhookService, Logger, Tracer, Context, EventSubscription, HealthSummaryData, NamespaceStatsData

### Community 71 - "Errors"
Cohesion: 0.21
Nodes (11): ServiceError, Code, Error(), Wrap(), Wrapf(), TestServiceError_Error(), T, TestServiceError_ClientMessage() (+3 more)

### Community 140 - "Community 140"
Cohesion: 0.50
Nodes (3): $schema, instructions, plan.md

### Community 11 - "Envelope Encryption (crypto)"
Cohesion: 0.12
Nodes (35): Service, NewService(), ParseKey(), IsEnvelopeEncrypted(), newAEAD(), AEAD, testKey(), TestNewService_NilKey() (+27 more)

### Community 37 - "Errors"
Cohesion: 0.16
Nodes (23): ErrorCategory, IsRetryableCategory(), ClassifyHTTPStatus(), ClassifyError(), isDNSError(), isTLSError(), classifySyscallError(), classifyByMessage() (+15 more)

### Community 100 - "Proto · Protoconnect"
Cohesion: 0.23
Nodes (11): NewWebhookServiceClient(), HTTPClient, WebhookServiceHandler, NewEventServiceClient(), EventServiceHandler, NewSubscriptionServiceClient(), SubscriptionServiceHandler, NewDeliveryServiceClient() (+3 more)

### Community 24 - "Proto · Protoconnect"
Cohesion: 0.11
Nodes (18): WebhookServiceClient, RegisterWebhookRequest, RegisterWebhookResponse, UnregisterWebhookRequest, UnregisterWebhookResponse, ListWebhooksRequest, ListWebhooksResponse, UpdateWebhookConfigRequest (+10 more)

### Community 77 - "Proto · Protoconnect"
Cohesion: 0.26
Nodes (10): Client, Context, HealthServiceClient, GetWebhookHealthRequest, GetWebhookHealthResponse, ListWebhooksByHealthRequest, ListWebhooksByHealthResponse, GetHealthSummaryRequest (+2 more)

### Community 8 - "Connect-RPC Clients"
Cohesion: 0.08
Nodes (29): Request, EventServiceClient, RegisterEventRequest, RegisterEventResponse, ListEventsRequest, ListEventsResponse, UpdateEventRequest, UpdateEventResponse (+21 more)

### Community 41 - "Proto · Protoconnect"
Cohesion: 0.17
Nodes (15): Response, DeliveryServiceClient, GetDeliveryStatusRequest, GetDeliveryStatusResponse, ListDeliveriesRequest, ListDeliveriesResponse, RetryDeliveryRequest, RetryDeliveryResponse (+7 more)

### Community 43 - "Proto · Protoconnect"
Cohesion: 0.15
Nodes (14): SubscriptionServiceClient, CreateSubscriptionRequest, CreateSubscriptionResponse, GetSubscriptionRequest, GetSubscriptionResponse, ListSubscriptionsRequest, ListSubscriptionsResponse, UpdateSubscriptionRequest (+6 more)

### Community 36 - "Proto"
Cohesion: 0.07
Nodes (6): GetEventRequest, TestSubscriptionTemplateResponse, GetHealthSummaryRequest, GetTemplateFunctionsRequest, init(), file_proto_webhook_proto_init()

### Community 18 - "Proto Enums"
Cohesion: 0.07
Nodes (7): WebhookDeliveryStatus, EnumDescriptor, EnumType, EnumNumber, WebhookHealth, ListWebhooksByHealthRequest, CancelRetryResponse

### Community 9 - "Proto Event Report Listing"
Cohesion: 0.04
Nodes (5): PaginationRequest, ListWebhooksRequest, ListEventsRequest, ListEventReportsRequest, ListSubscriptionsRequest

### Community 28 - "Proto"
Cohesion: 0.06
Nodes (6): MessageState, UpdateWebhookConfigResponse, DeleteEventRequest, DeleteSubscriptionResponse, GetDeliveryAttemptsRequest, GetRetryStatusRequest

### Community 38 - "Proto"
Cohesion: 0.07
Nodes (5): UnknownFields, GetDeliveryStatusRequest, GetNamespaceStatsRequest, RePushEventsRequest, GetRepushStatusRequest

### Community 25 - "Proto"
Cohesion: 0.06
Nodes (6): SizeCache, ResumeWebhookResponse, DeleteEventResponse, UpdateSubscriptionResponse, RePushEventsResponse, RetryDeliveriesRequest

### Community 30 - "Proto"
Cohesion: 0.06
Nodes (5): Message, UnregisterWebhookResponse, PauseWebhookResponse, RePushEventRequest, CancelRetryRequest

### Community 6 - "Proto EventSubscription"
Cohesion: 0.04
Nodes (6): PaginationResponse, EventSubscription, GetSubscriptionResponse, ListSubscriptionsResponse, ListSubscriptionsByEventResponse, ListDeliveriesResponse

### Community 4 - "Proto Subscription Messages"
Cohesion: 0.03
Nodes (7): RegisterWebhookResponse, Timestamp, RegisterEventResponse, CreateSubscriptionResponse, WebhookDelivery, GetDeliveryStatusResponse, ListDeliveriesRequest

### Community 23 - "Proto"
Cohesion: 0.06
Nodes (3): RegisteredWebhook, ListWebhooksResponse, ListWebhooksByHealthResponse

### Community 16 - "Proto PushEvent"
Cohesion: 0.06
Nodes (4): RegisterEventRequest, Struct, UpdateEventRequest, PushEventRequest

### Community 40 - "Proto"
Cohesion: 0.08
Nodes (3): RegisteredEvent, ListEventsResponse, GetEventResponse

### Community 29 - "Proto"
Cohesion: 0.06
Nodes (5): UpdateEventResponse, GetEventRecordRequest, GetSubscriptionRequest, CancelRepushRequest, file_proto_webhook_proto_rawDescGZIP()

### Community 20 - "Proto EventReport"
Cohesion: 0.06
Nodes (3): EventReport, ListEventReportsResponse, GetEventRecordResponse

### Community 42 - "Proto"
Cohesion: 0.08
Nodes (3): BatchJobStatus, GetRepushStatusResponse, GetRetryStatusResponse

### Community 15 - "Proto Service Clients"
Cohesion: 0.08
Nodes (26): ClientConnInterface, UnsafeWebhookServiceServer, UnsafeEventServiceServer, SubscriptionServiceClient, NewSubscriptionServiceClient(), GetSubscriptionRequest, GetSubscriptionResponse, ListSubscriptionsRequest (+18 more)

### Community 22 - "Proto"
Cohesion: 0.08
Nodes (21): WebhookServiceClient, NewWebhookServiceClient(), UnregisterWebhookRequest, UnregisterWebhookResponse, ListWebhooksRequest, ListWebhooksResponse, UpdateWebhookConfigRequest, UpdateWebhookConfigResponse (+13 more)

### Community 136 - "Proto"
Cohesion: 0.50
Nodes (3): RegisterWebhookRequest, RegisterWebhookResponse, _WebhookService_RegisterWebhook_Handler()

### Community 58 - "Proto"
Cohesion: 0.15
Nodes (13): Context, ResumeWebhookRequest, ResumeWebhookResponse, _WebhookService_ResumeWebhook_Handler(), ListEventsRequest, ListEventsResponse, DeleteEventRequest, DeleteEventResponse (+5 more)

### Community 17 - "Proto Delivery Service"
Cohesion: 0.08
Nodes (22): UnaryServerInterceptor, DeliveryServiceClient, NewDeliveryServiceClient(), GetDeliveryStatusRequest, GetDeliveryStatusResponse, ListDeliveriesRequest, ListDeliveriesResponse, RetryDeliveryRequest (+14 more)

### Community 21 - "Proto Event Service"
Cohesion: 0.08
Nodes (21): EventServiceClient, NewEventServiceClient(), GetEventRequest, GetEventResponse, ListEventReportsRequest, ListEventReportsResponse, RePushEventRequest, RePushEventResponse (+13 more)

### Community 133 - "Proto"
Cohesion: 0.50
Nodes (3): RegisterEventRequest, RegisterEventResponse, _EventService_RegisterEvent_Handler()

### Community 134 - "Proto"
Cohesion: 0.50
Nodes (3): UpdateEventRequest, UpdateEventResponse, _EventService_UpdateEvent_Handler()

### Community 132 - "Proto"
Cohesion: 0.50
Nodes (3): PushEventRequest, PushEventResponse, _EventService_PushEvent_Handler()

### Community 131 - "Proto"
Cohesion: 0.50
Nodes (3): GetEventRecordRequest, GetEventRecordResponse, _EventService_GetEventRecord_Handler()

### Community 135 - "Proto"
Cohesion: 0.50
Nodes (3): CreateSubscriptionRequest, CreateSubscriptionResponse, _SubscriptionService_CreateSubscription_Handler()

### Community 73 - "Proto"
Cohesion: 0.14
Nodes (10): GetWebhookHealthRequest, GetWebhookHealthResponse, ListWebhooksByHealthRequest, ListWebhooksByHealthResponse, GetHealthSummaryRequest, GetHealthSummaryResponse, UnimplementedHealthServiceServer, _HealthService_GetWebhookHealth_Handler() (+2 more)

### Community 2 - "Proto Webhook Types (TS)"
Cohesion: 0.02
Nodes (91): PaginationRequest, PaginationResponse, WebhookHTTPConfig, RegisterWebhookRequest, RegisterWebhookResponse, UnregisterWebhookRequest, UnregisterWebhookResponse, ListWebhooksRequest (+83 more)

### Community 1 - "Proto Schemas (TS)"
Cohesion: 0.02
Nodes (94): file_proto_webhook, PaginationRequestSchema, PaginationResponseSchema, WebhookHTTPConfigSchema, RegisterWebhookRequestSchema, RegisterWebhookResponseSchema, UnregisterWebhookRequestSchema, UnregisterWebhookResponseSchema (+86 more)

### Community 74 - "Web · Src · Lib"
Cohesion: 0.12
Nodes (15): WebhookService, EventService, SubscriptionService, DeliveryService, HealthService, SparrowConfig, Window, interceptors (+7 more)

### Community 137 - "Web"
Cohesion: 0.40
Nodes (4): name, private, version, type

### Community 125 - "Web"
Cohesion: 0.29
Nodes (7): scripts, dev, build, preview, prepare, check, check:watch

### Community 62 - "Web"
Cohesion: 0.11
Nodes (19): devDependencies, @bufbuild/protoc-gen-es, @bufbuild/protoc-gen-es, @sveltejs/adapter-static, @sveltejs/adapter-static, @sveltejs/kit, @sveltejs/kit, @tailwindcss/typography (+11 more)

### Community 63 - "Web"
Cohesion: 0.11
Nodes (19): dependencies, @bufbuild/protobuf, @bufbuild/protobuf, @connectrpc/connect, @connectrpc/connect, @connectrpc/connect-web, @connectrpc/connect-web, @kaifronsdal/svelte-json-viewer (+11 more)

### Community 54 - "Web · Src · Lib"
Cohesion: 0.14
Nodes (6): JSONSchemaMetaSchema, inferType(), jsonToJsonSchema(), ERROR_CATEGORIES, getCategoryBadge(), getCategoryDisplay()

### Community 88 - "Web"
Cohesion: 0.14
Nodes (13): extends, ./.svelte-kit/tsconfig.json, compilerOptions, allowJs, checkJs, esModuleInterop, forceConsistentCasingInFileNames, resolveJsonModule (+5 more)

### Community 130 - ".Github · Workflows"
Cohesion: 0.50
Nodes (5): CI Workflow, CI Lint Job, CI Test Job (Postgres service), CI Build Job, CI Integration Test Job

### Community 82 - "Community 82"
Cohesion: 0.14
Nodes (14): Release GoReleaser Job, Release Docker Image Job, GoReleaser Config, Dual Protocol (gRPC + Connect-RPC), SSRF Protection, Conventional Commits Convention, buf Codegen Plugins, buf Lint/Breaking Config (+6 more)

### Community 126 - "Community 126"
Cohesion: 0.33
Nodes (6): Deploy Docs Workflow, Sparrow Agent/Repo Conventions, River Queue (Postgres-backed workers), Sparrow Webhook Delivery Platform, Event-driven Fan-out Pipeline, Svelte 5 Tutorial (PDF)

### Community 164 - "Community 164"
Cohesion: 0.67
Nodes (3): Dual Webhook Signing (HMAC-SHA256 + Ed25519), Standard Webhooks Format, Timestamp Replay Protection

### Community 13 - "Docs & Deployment Guides"
Cohesion: 0.07
Nodes (44): PostgreSQL StatefulSet, Sparrow Kubernetes Service, Sparrow Helm Chart Values, Sparrow Client Libraries README, Python Client README Template, Sparrow Python Client README, Standalone Docker Compose, Development Docker Compose (+36 more)

### Community 34 - "Docs · Reference"
Cohesion: 0.12
Nodes (18): DeliveryService, WebhookDeliveryStatus, WebhookHealth, EventService, HealthService, Sparrow API Reference, SubscriptionService, WebhookService (+10 more)

## Knowledge Gaps
- **386 isolated node(s):** `name`, `version`, `description`, `type`, `license` (+381 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **72 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `setupEnv()` connect `Integration Tests & Middleware` to `Webhooks · Queue`, `Grpc`, `Event Filtering Tests`, `Webhooks`, `Webhook Service OTel`, `Webhooks`, `Store Tenant & Repository`, `Envelope Encryption (crypto)`, `Proto`?**
  _High betweenness centrality (0.124) - this node is a cross-community bridge._
- **Why does `Errorf()` connect `Webhooks` to `Config`, `Integration Tests & Middleware`, `Event Filtering Tests`, `Proto↔Domain Conversions`, `Envelope Encryption (crypto)`, `Store Tenant & Repository`, `Delivery Repository`, `Webhooks`, `Grpc`, `Errors`, `Webhooks · Client`, `Webhooks`, `Grpc`, `Observability`, `Webhooks · Queue`, `Webhooks · Store`, `Webhooks · Store`, `Webhooks · Queue`, `Errors`, `Webhooks`, `Webhooks · Client`, `Grpc`, `Webhooks · Queue`, `Webhooks · Store`, `Webhooks · Queue`, `Webhooks · Queue`, `Migration`, `Webhooks · Client`?**
  _High betweenness centrality (0.090) - this node is a cross-community bridge._
- **Why does `NewWebhookConnectServer()` connect `Proto` to `Integration Tests & Middleware`, `Connect`, `Connect`?**
  _High betweenness centrality (0.075) - this node is a cross-community bridge._
- **Are the 119 inferred relationships involving `Errorf()` (e.g. with `Load()` and `.Validate()`) actually correct?**
  _`Errorf()` has 119 INFERRED edges - model-reasoned connections that need verification._
- **What connects `name`, `version`, `description` to the rest of the system?**
  _386 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Client ·  Templates · Js · Connect` be split into smaller, more focused modules?**
  _Cohesion score 0.06666666666666667 - nodes in this community are weakly interconnected._
- **Should `Client · Js · Connect` be split into smaller, more focused modules?**
  _Cohesion score 0.06666666666666667 - nodes in this community are weakly interconnected._