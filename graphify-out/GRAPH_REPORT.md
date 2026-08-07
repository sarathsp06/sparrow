# Graph Report - sparrow  (2026-08-07)

## Corpus Check
- 259 files · ~209,803 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3425 nodes · 6521 edges · 263 communities (188 shown, 75 thin omitted)
- Extraction: 92% EXTRACTED · 8% INFERRED · 0% AMBIGUOUS · INFERRED: 547 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `75d49c6c`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- setupEnv
- webhook_pb.js
- webhook_pb.d.ts
- event_filtering_test.go
- Timestamp
- RepositoryInterfaceWithTracing
- EventSubscription
- steps.py
- EventServiceClient
- PaginationRequest
- ConvertProtoHTTPConfig
- NewService
- WebhookServiceInterfaceWithTracing
- Sparrow Detailed Flow Reference
- Context
- webhook_grpc.pb.go
- RegisteredEvent
- UnaryServerInterceptor
- GetWebhookHealthResponse
- store/models.go
- EventReport
- UnimplementedEventServiceServer
- UnimplementedWebhookServiceServer
- RegisteredWebhook
- WebhookServiceClient
- RePushEventRequest
- types.ts
- BatchJob
- file_proto_webhook_proto_rawDescGZIP
- WebhookDelivery
- GetRepushStatusRequest
- client/js/connect/package.json
- _templates/js/connect/package.json
- Errorf
- Sparrow API Reference
- toGRPCError
- GetRetryStatusRequest
- ClassifyError
- BatchJobStatus
- PrepareDeliveryRequest
- EventRecord
- Context
- Sparrow Implementation Plan
- Response
- 000001_consolidated_schema.up.sql
- WebhookServiceInterface
- .GetRetryStatus
- scripts
- NewWebhookClient
- .RegisterWebhook
- RepositoryInterface
- ValidateIP
- .ListSubscriptions
- helpers.go
- utils.ts
- otel.go
- NewWebhookWorker
- DeliveryAttempt
- Context
- NamespaceStats
- WebhookWorker
- Context
- devDependencies
- dependencies
- Context
- NewMetrics
- SparrowAPI
- BatchJobWorker
- HealthSummary
- WebhookHTTPConfig
- ServiceError
- NewTemplateEngine
- UnimplementedHealthServiceServer
- services.ts
- parseUUID
- Message
- HealthServiceClient
- Context
- Request
- CreateSubscriptionRequest
- RegisterWebhookRequest
- Sparrow Deployment Template
- TemplateCache
- WebhookConnectServer
- .GetWebhookHealth
- UpdateSubscriptionRequest
- WebhookUpdateFields
- compilerOptions
- WebhookTargetManager
- GetBuffer
- TemplateEngine
- forwardUnary
- Request
- Context
- .ListEventReports
- NewManager
- Context
- _Target
- Checker
- WebhookService
- EventRegistration
- jobInserter
- hooks.py
- SparrowEnvironment
- .CancelRepush
- UpdateWebhookConfigRequest
- EventArgs
- JobInserterWithTracing
- RetryDeliveryRequest
- MessageState
- PushEventResponse
- ResumeWebhookRequest
- RetryDeliveriesResponse
- RePushEventsResponse
- DefaultConfig
- RunAllMigrations
- DeleteSubscriptionRequest
- webhook.pb.go
- ListSubscriptionsByEventRequest
- UnknownFields
- RetryDeliveryResponse
- UnregisterWebhookRequest
- APIKeyAuth
- Sparrow -- Condensed Reference
- scripts
- Sparrow Webhook Delivery Platform
- GetTemplateFunctions
- models_test.go
- WebhookServer
- CI Build Job
- GetSubscriptionRequest
- .PushEvent
- .RegisterEvent
- .UpdateEvent
- .CreateSubscription
- .RegisterWebhook
- web/package.json
- manifest.json
- grpc_client.go
- instructions
- .ListDeliveries
- .DeleteEvent
- SizeCache
- Sparrow Web Dashboard
- .GetHealthSummary
- .GetNamespaceStats
- Generic Table Component Usage Examples
- .GetWebhookHealth
- .ListEvents
- RetryDeliveriesRequest
- .ListWebhooks
- RePushEventsRequest
- Config
- .RePushEvents
- .RetryDeliveries
- Response
- .GetEventRecord
- .UpdateSubscription
- .UpdateWebhookConfig
- src/components/Footer.astro
- docs/tsconfig.json
- TestJobInserter_InsertOpts_Merge
- TestWebhookWorkerDefaults
- Dual Webhook Signing (HMAC-SHA256 + Ed25519)
- +layout.ts
- WithConn Transaction Pattern
- 000015_envelope_encryption.up.sql
- 000019_drop_system_settings.down.sql
- ../../components/ThemeDiagram.astro
- content.config.ts
- proto2astro
- index.astro
- .GetDeliveryAttempts
- @sveltejs/vite-plugin-svelte
- @tailwindcss/forms
- WebhookService
- typescript
- vite-plugin-devtools-json
- app.d.ts
- svelte.config.js
- Sparrow Ingress Template
- Sparrow Chart NOTES
- Sparrow PodDisruptionBudget Template
- sparrow-webhooks
- sparrow-webhooks
- Release Workflow
- github.com/sarathsp06/sparrow
- sparrow-e2e
- Envelope Encryption at Rest (AES-256-GCM)
- .ListEventReports
- .GetRetryStatus
- .ResumeWebhook
- .GetDeliveryStatus
- .GetRepushStatus
- .GetSubscription
- .ListDeliveries
- .ListWebhooksByHealth
- .RePushEvent
- .PushEvent
- vite

## God Nodes (most connected - your core abstractions)
1. `Errorf()` - 122 edges
2. `file_proto_webhook_proto_rawDescGZIP()` - 92 edges
3. `RepositoryInterfaceWithTracing` - 69 edges
4. `forwardUnary()` - 43 edges
5. `WebhookServiceInterfaceWithTracing` - 40 edges
6. `WebhookConnectServer` - 39 edges
7. `EventServiceClient` - 39 edges
8. `toGRPCError()` - 38 edges
9. `setupEnv()` - 37 edges
10. `NewService()` - 32 edges

## Surprising Connections (you probably didn't know these)
- `Svelte 5 Tutorial (PDF)` --conceptually_related_to--> `Sparrow Webhook Delivery Platform`  [INFERRED]
  book/svelte5-tutorial.pdf → README.md
- `Sparrow NetworkPolicy Template` --semantically_similar_to--> `SSRF Protection`  [INFERRED] [semantically similar]
  charts/sparrow/templates/networkpolicy.yaml → README.md
- `Development Docker Compose` --semantically_similar_to--> `Standalone Docker Compose`  [INFERRED] [semantically similar]
  docker-compose.dev.yml → deploy/docker-compose.yml
- `main()` --calls--> `RunAllMigrations()`  [INFERRED]
  cmd/migrate/main.go → internal/migration/migrate.go
- `RunAppMigrations()` --calls--> `GetMigrationsFS()`  [INFERRED]
  internal/migration/migrate.go → db/migrations.go

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

## Communities (263 total, 75 thin omitted)

### Community 0 - "setupEnv"
Cohesion: 0.05
Nodes (89): Container, GetMigrationsFS(), DeliveryServiceClient, EventServiceClient, FS, HandlerFunc, Header, HTTPClient (+81 more)

### Community 1 - "webhook_pb.js"
Cohesion: 0.02
Nodes (94): BatchJobStatusSchema, CancelRepushRequestSchema, CancelRepushResponseSchema, CancelRetryRequestSchema, CancelRetryResponseSchema, CreateSubscriptionRequestSchema, CreateSubscriptionResponseSchema, DeleteEventRequestSchema (+86 more)

### Community 2 - "webhook_pb.d.ts"
Cohesion: 0.02
Nodes (91): BatchJobStatus, CancelRepushRequest, CancelRepushResponse, CancelRetryRequest, CancelRetryResponse, CreateSubscriptionRequest, CreateSubscriptionResponse, DeleteEventRequest (+83 more)

### Community 3 - "event_filtering_test.go"
Cohesion: 0.07
Nodes (60): Context, EventSubscription, T, UUID, TestCreateSubscription_CatchAllWithLabelFilters(), TestCreateSubscription_WithEmptyLabelFilters(), TestCreateSubscription_WithInvalidLabelFilters(), TestCreateSubscription_WithLabelFilters() (+52 more)

### Community 4 - "Timestamp"
Cohesion: 0.04
Nodes (6): CreateSubscriptionResponse, ListEventReportsRequest, RegisterEventResponse, RegisterWebhookResponse, Timestamp, WebhookHealthMetrics

### Community 5 - "RepositoryInterfaceWithTracing"
Cohesion: 0.09
Nodes (14): BatchJobStatus, Context, EventSubscription, NamespaceStats, Span, Time, UUID, WebhookDelivery (+6 more)

### Community 6 - "EventSubscription"
Cohesion: 0.04
Nodes (6): EventSubscription, GetSubscriptionResponse, ListEventsResponse, ListSubscriptionsByEventResponse, ListSubscriptionsResponse, PaginationResponse

### Community 7 - "steps.py"
Cohesion: 0.09
Nodes (53): delivery_has_signature_headers(), SignatureVerifier -- Verifies HMAC-SHA256 (v1,) and Ed25519 (v1a,) signatures…, Verify HMAC-SHA256 signature (v1, prefix)., Verify Ed25519 signature (v1a, prefix)., Assert delivery has Standard Webhooks signature headers., verify_ed25519_signature(), verify_hmac_signature(), _api() (+45 more)

### Community 8 - "EventServiceClient"
Cohesion: 0.09
Nodes (22): CancelRepushRequest, CancelRepushResponse, DeleteEventRequest, DeleteEventResponse, GetEventRecordRequest, GetEventRecordResponse, GetEventRequest, GetEventResponse (+14 more)

### Community 9 - "PaginationRequest"
Cohesion: 0.04
Nodes (5): ListDeliveriesRequest, ListEventsRequest, ListSubscriptionsRequest, ListWebhooksRequest, PaginationRequest

### Community 10 - "ConvertProtoHTTPConfig"
Cohesion: 0.08
Nodes (33): ConvertProtoHTTPConfig(), CreateWebhookRegistrationRequest(), float32PtrToFloat64Ptr(), float64PtrToFloat32Ptr(), RegisterWebhookRequest, WebhookHTTPConfig, float32Ptr(), float64Ptr() (+25 more)

### Community 11 - "NewService"
Cohesion: 0.12
Nodes (35): AEAD, Service, IsEnvelopeEncrypted(), newAEAD(), NewService(), ParseKey(), T, TestDecrypt_BackwardCompatibility() (+27 more)

### Community 12 - "WebhookServiceInterfaceWithTracing"
Cohesion: 0.06
Nodes (17): Context, Time, WebhookHealth, WebhookRegistration, WebhookService, Context, EventSubscription, Span (+9 more)

### Community 13 - "Sparrow Detailed Flow Reference"
Cohesion: 0.07
Nodes (44): PostgreSQL StatefulSet, Sparrow Kubernetes Service, Sparrow Helm Chart Values, Python Client README Template, Sparrow Python Client README, Sparrow Client Libraries README, Connect-RPC, DeliveryService (+36 more)

### Community 14 - "Context"
Cohesion: 0.12
Nodes (22): Bootstrap(), Context, Context, Repository, Tx, UUID, WebhookRegistration, NewRepository() (+14 more)

### Community 15 - "webhook_grpc.pb.go"
Cohesion: 0.08
Nodes (26): ClientConnInterface, HealthServiceClient, SubscriptionServiceClient, UnimplementedSubscriptionServiceServer, UnsafeDeliveryServiceServer, UnsafeEventServiceServer, UnsafeHealthServiceServer, UnsafeSubscriptionServiceServer (+18 more)

### Community 16 - "RegisteredEvent"
Cohesion: 0.04
Nodes (6): GetEventResponse, PushEventRequest, RegisteredEvent, RegisterEventRequest, UpdateEventRequest, Struct

### Community 17 - "UnaryServerInterceptor"
Cohesion: 0.08
Nodes (22): DeliveryServiceClient, UnimplementedDeliveryServiceServer, _DeliveryService_GetDeliveryAttempts_Handler(), _DeliveryService_GetDeliveryStatus_Handler(), _DeliveryService_GetRetryStatus_Handler(), _DeliveryService_ListDeliveries_Handler(), _DeliveryService_RetryDeliveries_Handler(), _DeliveryService_RetryDelivery_Handler() (+14 more)

### Community 18 - "GetWebhookHealthResponse"
Cohesion: 0.06
Nodes (8): EnumDescriptor, EnumNumber, EnumType, CancelRetryResponse, GetWebhookHealthResponse, ListWebhooksByHealthRequest, WebhookDeliveryStatus, WebhookHealth

### Community 19 - "store/models.go"
Cohesion: 0.10
Nodes (20): Int64Array, Time, UUID, Value, WebhookDeliveryStatus, BatchJobStatus, DeliveryFilter, EventSubscription (+12 more)

### Community 20 - "EventReport"
Cohesion: 0.06
Nodes (3): EventReport, GetEventRecordResponse, ListEventReportsResponse

### Community 21 - "UnimplementedEventServiceServer"
Cohesion: 0.08
Nodes (21): EventServiceClient, UnimplementedEventServiceServer, _EventService_CancelRepush_Handler(), _EventService_GetEvent_Handler(), _EventService_GetRepushStatus_Handler(), _EventService_ListEventReports_Handler(), _EventService_RePushEvent_Handler(), _EventService_RePushEvents_Handler() (+13 more)

### Community 22 - "UnimplementedWebhookServiceServer"
Cohesion: 0.08
Nodes (21): UnimplementedWebhookServiceServer, GetNamespaceStatsRequest, GetNamespaceStatsResponse, GetTemplateFunctionsRequest, GetTemplateFunctionsResponse, ListWebhooksRequest, ListWebhooksResponse, PauseWebhookRequest (+13 more)

### Community 23 - "RegisteredWebhook"
Cohesion: 0.06
Nodes (3): ListWebhooksByHealthResponse, ListWebhooksResponse, RegisteredWebhook

### Community 24 - "WebhookServiceClient"
Cohesion: 0.11
Nodes (18): GetNamespaceStatsRequest, GetNamespaceStatsResponse, GetTemplateFunctionsRequest, GetTemplateFunctionsResponse, ListWebhooksRequest, ListWebhooksResponse, PauseWebhookRequest, PauseWebhookResponse (+10 more)

### Community 26 - "types.ts"
Cohesion: 0.09
Nodes (17): ../../../data/api/webhook-service, buildExampleObject(), generateCurl(), generateResponseJson(), service, enumData, enumData, service (+9 more)

### Community 27 - "BatchJob"
Cohesion: 0.26
Nodes (6): Context, WebhookService, RawMessage, BatchJob, BatchJobData, BatchJobType

### Community 28 - "file_proto_webhook_proto_rawDescGZIP"
Cohesion: 0.06
Nodes (5): CancelRetryRequest, DeleteSubscriptionResponse, GetDeliveryAttemptsRequest, GetEventRequest, file_proto_webhook_proto_rawDescGZIP()

### Community 29 - "WebhookDelivery"
Cohesion: 0.06
Nodes (3): GetDeliveryStatusResponse, ListDeliveriesResponse, WebhookDelivery

### Community 31 - "client/js/connect/package.json"
Cohesion: 0.07
Nodes (29): description, exports, ./proto/*, files, homepage, @bufbuild/protobuf, @connectrpc/connect, @connectrpc/connect-web (+21 more)

### Community 32 - "_templates/js/connect/package.json"
Cohesion: 0.07
Nodes (29): description, exports, ./proto/*, files, homepage, @bufbuild/protobuf, @connectrpc/connect, @connectrpc/connect-web (+21 more)

### Community 33 - "Errorf"
Cohesion: 0.24
Nodes (7): generateSamplePayload(), Context, Time, WebhookService, ValidateJSONSchema(), Errorf(), SchemaValidationError

### Community 34 - "Sparrow API Reference"
Cohesion: 0.12
Nodes (18): DeliveryService, WebhookDeliveryStatus, WebhookHealth, EventService, HealthService, Sparrow API Reference, SubscriptionService, WebhookService (+10 more)

### Community 35 - "toGRPCError"
Cohesion: 0.09
Nodes (26): CancelRepushRequest, CancelRepushResponse, Context, DeleteEventRequest, DeleteEventResponse, GetEventRecordRequest, GetEventRecordResponse, GetEventRequest (+18 more)

### Community 37 - "ClassifyError"
Cohesion: 0.16
Nodes (23): ErrorCategory, timeoutError, classifyByMessage(), ClassifyError(), ClassifyHTTPStatus(), classifySyscallError(), isDNSError(), IsRetryableCategory() (+15 more)

### Community 38 - "BatchJobStatus"
Cohesion: 0.08
Nodes (3): BatchJobStatus, GetRepushStatusResponse, GetRetryStatusResponse

### Community 39 - "PrepareDeliveryRequest"
Cohesion: 0.12
Nodes (25): DeliveryRequest, WebhookEnvelope, Context, Duration, Response, BuildEnvelopePayload(), BuildRequest(), generateEd25519Signature() (+17 more)

### Community 40 - "EventRecord"
Cohesion: 0.20
Nodes (7): Context, Repository, UUID, IsNotFound(), EventRecord, EventReportFilter, EventReportWithStats

### Community 41 - "Context"
Cohesion: 0.17
Nodes (15): Context, GetDeliveryAttemptsRequest, GetDeliveryAttemptsResponse, GetDeliveryStatusRequest, GetDeliveryStatusResponse, GetRetryStatusRequest, GetRetryStatusResponse, ListDeliveriesRequest (+7 more)

### Community 42 - "Sparrow Implementation Plan"
Cohesion: 0.05
Nodes (42): Commands, Completed Parts (v0.8.0 -- v1.2.1), Configuration, Configuration, Current State (as of v1.2.1), Decisions Log, Design, Design (+34 more)

### Community 43 - "Response"
Cohesion: 0.17
Nodes (15): CreateSubscriptionRequest, CreateSubscriptionResponse, DeleteSubscriptionRequest, DeleteSubscriptionResponse, GetSubscriptionRequest, GetSubscriptionResponse, ListSubscriptionsRequest, ListSubscriptionsResponse (+7 more)

### Community 44 - "000001_consolidated_schema.up.sql"
Cohesion: 0.11
Nodes (18): event_records, event_registrations, event_subscriptions, update_event_registrations_updated_at, update_updated_at_column(), update_webhook_registrations_updated_at, webhook_deliveries, webhook_health_events (+10 more)

### Community 45 - "WebhookServiceInterface"
Cohesion: 0.13
Nodes (18): DeliveryService, EventService, HealthService, normalizePagination(), WithAllowPrivateNetworks(), SubscriptionService, BatchService, DeliveryService (+10 more)

### Community 46 - ".GetRetryStatus"
Cohesion: 0.11
Nodes (16): CancelRetryRequest, CancelRetryResponse, Context, GetDeliveryAttemptsRequest, GetDeliveryAttemptsResponse, GetDeliveryStatusRequest, GetDeliveryStatusResponse, GetRetryStatusRequest (+8 more)

### Community 47 - "scripts"
Cohesion: 0.08
Nodes (23): astro, @astrojs/starlight, dependencies, astro, @astrojs/starlight, marked, sharp, zod (+15 more)

### Community 48 - "NewWebhookClient"
Cohesion: 0.16
Nodes (20): WebhookClient, Client, Config, WebhookTemplateContext, NewWebhookClient(), ReadBody(), BenchmarkSend(), BenchmarkTransformPayload() (+12 more)

### Community 49 - ".RegisterWebhook"
Cohesion: 0.11
Nodes (17): convertStatusCodesToInt(), Context, GetTemplateFunctionsRequest, GetTemplateFunctionsResponse, WebhookServer, ListWebhooksRequest, ListWebhooksResponse, PauseWebhookRequest (+9 more)

### Community 50 - "RepositoryInterface"
Cohesion: 0.16
Nodes (9): JobInserter, Logger, Tracer, WebhookService, EventTypeRepository, HealthRepository, RateLimitRepository, RepositoryInterface (+1 more)

### Community 51 - "ValidateIP"
Cohesion: 0.36
Nodes (7): Request, ssrfDialControl(), ssrfSafeCheckRedirect(), ValidateIP(), validateRedirectURL(), IP, RawConn

### Community 52 - ".ListSubscriptions"
Cohesion: 0.11
Nodes (16): convertSubscriptionToProto(), Context, CreateSubscriptionRequest, CreateSubscriptionResponse, DeleteSubscriptionRequest, DeleteSubscriptionResponse, EventSubscription, GetSubscriptionRequest (+8 more)

### Community 53 - "helpers.go"
Cohesion: 0.13
Nodes (24): batchJobToProto(), convertDeliveryStatus(), convertDeliveryToProto(), convertEventToProto(), convertExpectedStatusCodes(), convertMapToStruct(), convertPtrTimeToProto(), convertTimeToProto() (+16 more)

### Community 54 - "utils.ts"
Cohesion: 0.14
Nodes (6): ERROR_CATEGORIES, getCategoryBadge(), getCategoryDisplay(), inferType(), JSONSchemaMetaSchema, jsonToJsonSchema()

### Community 55 - "otel.go"
Cohesion: 0.19
Nodes (19): Float64Histogram, Int64Counter, Int64UpDownCounter, DefaultConfig(), GetMeter(), Context, Duration, newLoggerProvider() (+11 more)

### Community 56 - "NewWebhookWorker"
Cohesion: 0.20
Nodes (11): GetTracer(), Tracer, JobInserter, Logger, WorkerDefaults, NewEventProcessingWorker(), Config, NewWebhookWorker() (+3 more)

### Community 58 - "Context"
Cohesion: 0.15
Nodes (13): _DeliveryService_CancelRetry_Handler(), _EventService_DeleteEvent_Handler(), _EventService_ListEvents_Handler(), CancelRetryRequest, CancelRetryResponse, Context, DeleteEventRequest, DeleteEventResponse (+5 more)

### Community 60 - "WebhookWorker"
Cohesion: 0.16
Nodes (17): T, TestDefaultAndMaxRetryAfterConstants(), TestIsSuccessStatusCode(), TestParseRetryAfter(), TestParseRetryAfter_HTTPDate(), TestParseRetryAfter_HTTPDate_FarFuture(), TestParseRetryAfter_HTTPDate_Past(), Context (+9 more)

### Community 61 - "Context"
Cohesion: 0.44
Nodes (4): Context, EventSubscription, Repository, UUID

### Community 62 - "devDependencies"
Cohesion: 0.11
Nodes (19): @bufbuild/protoc-gen-es, svelte-check, @sveltejs/adapter-static, @sveltejs/kit, tailwindcss, @tailwindcss/typography, @tailwindcss/vite, @types/node (+11 more)

### Community 63 - "dependencies"
Cohesion: 0.11
Nodes (19): flowbite, flowbite-svelte, flowbite-svelte-icons, @kaifronsdal/svelte-json-viewer, dependencies, @bufbuild/protobuf, @connectrpc/connect, @connectrpc/connect-web (+11 more)

### Community 64 - "Context"
Cohesion: 0.38
Nodes (4): BatchJobStatus, Context, Repository, UUID

### Community 66 - "NewMetrics"
Cohesion: 0.20
Nodes (13): Metrics, Duration, NewMetrics(), BenchmarkRecordRequest(), BenchmarkRecordSuccess(), B, T, TestMetricsConcurrency() (+5 more)

### Community 67 - "SparrowAPI"
Cohesion: 0.12
Nodes (4): Client for Sparrow Connect-RPC endpoints., Poll until all deliveries reach terminal status., Poll a single delivery until terminal., SparrowAPI

### Community 68 - "BatchJobWorker"
Cohesion: 0.16
Nodes (11): Context, Job, JobInserter, Logger, UUID, WorkerDefaults, NewBatchJobWorker(), BatchJobArgs (+3 more)

### Community 71 - "ServiceError"
Cohesion: 0.23
Nodes (11): Code, ServiceError, Error(), T, TestConstructors(), TestServiceError_ClientMessage(), TestServiceError_Error(), TestServiceError_ErrorsAs() (+3 more)

### Community 72 - "NewTemplateEngine"
Cohesion: 0.27
Nodes (16): NewTemplateEngine(), NewTemplateEngineWithCacheSize(), BenchmarkExecuteComplex(), BenchmarkExecuteSimple(), B, T, TestExecuteComplexTemplate(), TestExecuteEmptyTemplate() (+8 more)

### Community 73 - "UnimplementedHealthServiceServer"
Cohesion: 0.14
Nodes (10): UnimplementedHealthServiceServer, GetHealthSummaryRequest, GetHealthSummaryResponse, GetWebhookHealthRequest, GetWebhookHealthResponse, ListWebhooksByHealthRequest, ListWebhooksByHealthResponse, _HealthService_GetHealthSummary_Handler() (+2 more)

### Community 74 - "services.ts"
Cohesion: 0.12
Nodes (15): DeliveryService, EventService, HealthService, SubscriptionService, WebhookService, client, deliveryClient, eventClient (+7 more)

### Community 75 - "parseUUID"
Cohesion: 0.20
Nodes (10): ValidateWebhookURL(), generateWebhookSecret(), UUID, parseUUID(), Context, Time, WebhookRegistration, WebhookService (+2 more)

### Community 76 - "Message"
Cohesion: 0.06
Nodes (5): Message, GetTemplateFunctionsResponse, ResumeWebhookResponse, TemplateFunction, UpdateWebhookConfigResponse

### Community 77 - "HealthServiceClient"
Cohesion: 0.24
Nodes (9): Client, GetHealthSummaryRequest, GetHealthSummaryResponse, GetWebhookHealthRequest, GetWebhookHealthResponse, ListWebhooksByHealthRequest, ListWebhooksByHealthResponse, HealthServiceClient (+1 more)

### Community 78 - "Context"
Cohesion: 0.34
Nodes (5): Context, Repository, UUID, WebhookDelivery, WebhookDeliveryStatus

### Community 79 - "Request"
Cohesion: 0.23
Nodes (7): CancelRetryRequest, CancelRetryResponse, GetRepushStatusRequest, GetRepushStatusResponse, RePushEventRequest, RePushEventResponse, Request

### Community 82 - "Sparrow Deployment Template"
Cohesion: 0.14
Nodes (14): Dual Protocol (gRPC + Connect-RPC), buf Codegen Plugins, buf Lint/Breaking Config, Sparrow Helm Chart, Sparrow ConfigMap Template, Sparrow Deployment Template, Sparrow HPA Template, Sparrow NetworkPolicy Template (+6 more)

### Community 83 - "TemplateCache"
Cohesion: 0.24
Nodes (10): Cache, TemplateCache, Template, hashTemplate(), NewTemplateCache(), T, TestHashTemplate(), TestTemplateCacheBasicOperations() (+2 more)

### Community 84 - "WebhookConnectServer"
Cohesion: 0.25
Nodes (14): grpcServerWrapper, WebhookConnectServer, NewWebhookConnectServer(), DeliveryServiceServer, EventServiceServer, HealthServiceServer, SubscriptionServiceServer, RegisterDeliveryServiceServer() (+6 more)

### Community 85 - ".GetWebhookHealth"
Cohesion: 0.18
Nodes (10): Context, GetHealthSummaryRequest, GetHealthSummaryResponse, GetNamespaceStatsRequest, GetNamespaceStatsResponse, GetWebhookHealthRequest, GetWebhookHealthResponse, WebhookServer (+2 more)

### Community 88 - "compilerOptions"
Cohesion: 0.14
Nodes (13): ./.svelte-kit/tsconfig.json, compilerOptions, allowJs, checkJs, esModuleInterop, forceConsistentCasingInFileNames, moduleResolution, resolveJsonModule (+5 more)

### Community 89 - "WebhookTargetManager"
Cohesion: 0.15
Nodes (5): before_scenario, Manages mock webhook target servers., WebhookTargetManager, Fresh target manager for each scenario., setup_scenario()

### Community 90 - "GetBuffer"
Cohesion: 0.28
Nodes (11): Buffer, GetBuffer(), GetHeaderMap(), PutBuffer(), PutHeaderMap(), BenchmarkBufferPool(), BenchmarkHeaderMapPool(), B (+3 more)

### Community 91 - "TemplateEngine"
Cohesion: 0.21
Nodes (7): limitedWriter, TemplateEngine, writerWithBytes, FuncMap, Template, WebhookTemplateContext, NewWebhookTemplateContext()

### Community 92 - "forwardUnary"
Cohesion: 0.20
Nodes (8): grpcUnary, forwardUnary(), CreateSubscriptionRequest, CreateSubscriptionResponse, RegisterEventRequest, RegisterEventResponse, Req, Resp

### Community 93 - "Request"
Cohesion: 0.15
Nodes (9): GetEventRecordRequest, GetEventRecordResponse, GetEventRequest, GetEventResponse, ListSubscriptionsRequest, ListSubscriptionsResponse, PauseWebhookRequest, PauseWebhookResponse (+1 more)

### Community 94 - "Context"
Cohesion: 0.15
Nodes (9): Context, GetTemplateFunctionsRequest, GetTemplateFunctionsResponse, RegisterWebhookRequest, RegisterWebhookResponse, RetryDeliveryRequest, RetryDeliveryResponse, UpdateEventRequest (+1 more)

### Community 95 - ".ListEventReports"
Cohesion: 0.40
Nodes (4): ListEventReportsRequest, ListEventReportsResponse, extractPagination(), PaginationRequest

### Community 96 - "NewManager"
Cohesion: 0.22
Nodes (9): Client, Config, Context, JobInserter, Logger, Pool, Tx, NewManager() (+1 more)

### Community 97 - "Context"
Cohesion: 0.29
Nodes (6): Context, Repository, Time, UUID, WebhookHealth, WebhookHealthMetrics

### Community 98 - "_Target"
Cohesion: 0.23
Nodes (4): CapturedDelivery, WebhookTargetServer -- Programmable mock webhook endpoints for e2e tests. Each…, Start a mock webhook target. Returns the URL., _Target

### Community 99 - "Checker"
Cohesion: 0.32
Nodes (7): Checker, HealthResponse, ReadyResponse, Context, Pool, Time, NewChecker()

### Community 100 - "WebhookService"
Cohesion: 0.38
Nodes (4): Context, EventSubscription, Time, WebhookService

### Community 101 - "EventRegistration"
Cohesion: 0.28
Nodes (4): Context, Repository, UUID, EventRegistration

### Community 102 - "jobInserter"
Cohesion: 0.32
Nodes (8): Client, Context, JobArgs, JobInsertResult, Logger, Tx, NewJobInserter(), jobInserter

### Community 103 - "hooks.py"
Cohesion: 0.18
Nodes (8): after_scenario, after_suite, SparrowAPI -- HTTP client for the Sparrow Connect-RPC API., SparrowEnvironment -- Manages Postgres and Sparrow containers via…, Gauge hooks for suite/scenario setup and teardown., Stop all mock targets., teardown_environment(), teardown_scenario()

### Community 104 - "SparrowEnvironment"
Cohesion: 0.22
Nodes (6): before_suite, Manages the Sparrow test environment (Postgres + Sparrow containers)., Start Postgres + Sparrow. Returns the Sparrow HTTP URL., SparrowEnvironment, Start Postgres + Sparrow containers., setup_environment()

### Community 107 - "EventArgs"
Cohesion: 0.22
Nodes (5): InsertOpts, Context, Job, Time, EventArgs

### Community 108 - "JobInserterWithTracing"
Cohesion: 0.31
Nodes (7): Context, JobArgs, JobInserter, JobInsertResult, Span, NewJobInserterWithTracing(), JobInserterWithTracing

### Community 110 - "MessageState"
Cohesion: 0.07
Nodes (5): MessageState, DeleteEventRequest, PauseWebhookRequest, PauseWebhookResponse, TestSubscriptionTemplateResponse

### Community 115 - "DefaultConfig"
Cohesion: 0.32
Nodes (6): Config, DefaultConfig(), Duration, T, TestCustomConfig(), TestDefaultConfig()

### Community 116 - "RunAllMigrations"
Cohesion: 0.46
Nodes (6): main(), Context, Logger, RunAllMigrations(), RunAppMigrations(), RunRiverMigrations()

### Community 118 - "webhook.pb.go"
Cohesion: 0.07
Nodes (6): GetTemplateFunctionsRequest, GetWebhookHealthRequest, RePushEventResponse, UpdateSubscriptionResponse, file_proto_webhook_proto_init(), init()

### Community 120 - "UnknownFields"
Cohesion: 0.07
Nodes (5): CancelRepushRequest, GetDeliveryStatusRequest, GetHealthSummaryRequest, GetNamespaceStatsRequest, UnknownFields

### Community 123 - "APIKeyAuth"
Cohesion: 0.28
Nodes (6): apiKeysFromIncomingContext(), Context, Request, UnaryServerInterceptor, APIKeyAuth, StreamServerInterceptor

### Community 124 - "Sparrow -- Condensed Reference"
Cohesion: 0.17
Nodes (11): API Key Authentication, Architecture, Code Patterns & Conventions, Design Principles, Development History, Handler Pattern, HTTP Routing (chi), Known Gaps (+3 more)

### Community 125 - "scripts"
Cohesion: 0.29
Nodes (7): scripts, build, check, check:watch, dev, prepare, preview

### Community 126 - "Sparrow Webhook Delivery Platform"
Cohesion: 0.33
Nodes (6): Sparrow Agent/Repo Conventions, River Queue (Postgres-backed workers), Svelte 5 Tutorial (PDF), Deploy Docs Workflow, Event-driven Fan-out Pipeline, Sparrow Webhook Delivery Platform

### Community 127 - "GetTemplateFunctions"
Cohesion: 0.53
Nodes (5): TemplateFunc, GetFunctionDocumentation(), GetFunctionMap(), GetTemplateFunctions(), FuncMap

### Community 128 - "models_test.go"
Cohesion: 0.39
Nodes (8): T, TestEventRegistration_NilJSONFieldsAreNullSafe(), TestJSONMap_RoundTrip(), TestJSONMap_Scan(), TestJSONMap_Value(), TestJSONStringMap_RoundTrip(), TestJSONStringMap_Scan(), TestJSONStringMap_Value()

### Community 129 - "WebhookServer"
Cohesion: 0.40
Nodes (4): Context, WebhookServer, WebhookRegistration, NewWebhookServer()

### Community 130 - "CI Build Job"
Cohesion: 0.50
Nodes (5): CI Build Job, CI Workflow, CI Integration Test Job, CI Lint Job, CI Test Job (Postgres service)

### Community 132 - ".PushEvent"
Cohesion: 0.50
Nodes (3): _EventService_PushEvent_Handler(), PushEventRequest, PushEventResponse

### Community 133 - ".RegisterEvent"
Cohesion: 0.50
Nodes (3): _EventService_RegisterEvent_Handler(), RegisterEventRequest, RegisterEventResponse

### Community 134 - ".UpdateEvent"
Cohesion: 0.50
Nodes (3): _EventService_UpdateEvent_Handler(), UpdateEventRequest, UpdateEventResponse

### Community 135 - ".CreateSubscription"
Cohesion: 0.50
Nodes (3): CreateSubscriptionRequest, CreateSubscriptionResponse, _SubscriptionService_CreateSubscription_Handler()

### Community 136 - ".RegisterWebhook"
Cohesion: 0.50
Nodes (3): RegisterWebhookRequest, RegisterWebhookResponse, _WebhookService_RegisterWebhook_Handler()

### Community 137 - "web/package.json"
Cohesion: 0.40
Nodes (4): name, private, type, version

### Community 138 - "manifest.json"
Cohesion: 0.50
Nodes (3): Language, Plugins, html-report

### Community 139 - "grpc_client.go"
Cohesion: 0.83
Nodes (3): extractFirstLine(), main(), MainGRPC()

### Community 140 - "instructions"
Cohesion: 0.50
Nodes (3): instructions, $schema, plan.md

### Community 141 - ".ListDeliveries"
Cohesion: 0.48
Nodes (3): Context, WebhookDelivery, WebhookService

### Community 143 - "SizeCache"
Cohesion: 0.06
Nodes (6): CancelRepushResponse, DeleteEventResponse, GetEventRecordRequest, UnregisterWebhookResponse, UpdateEventResponse, SizeCache

### Community 144 - "Sparrow Web Dashboard"
Cohesion: 0.22
Nodes (8): Build, Embedding in the Go Binary, Environment Variables, Local Development, Prerequisites, Sparrow Web Dashboard, Standalone Deployment, Tech Stack

### Community 147 - "Generic Table Component Usage Examples"
Cohesion: 0.22
Nodes (8): Advanced Usage with Column Formatters, Basic Usage, Benefits, ColumnFormatter Interface, Component Props, Generic Table Component Usage Examples, Optional Props, Required Props

### Community 153 - "Config"
Cohesion: 0.47
Nodes (3): Config, Load(), validatePort()

### Community 156 - "Response"
Cohesion: 0.15
Nodes (9): CancelRetryRequest, CancelRetryResponse, DeleteSubscriptionRequest, DeleteSubscriptionResponse, Response, TestSubscriptionTemplateRequest, TestSubscriptionTemplateResponse, UnregisterWebhookRequest (+1 more)

### Community 157 - ".GetEventRecord"
Cohesion: 0.50
Nodes (3): _EventService_GetEventRecord_Handler(), GetEventRecordRequest, GetEventRecordResponse

### Community 164 - "Dual Webhook Signing (HMAC-SHA256 + Ed25519)"
Cohesion: 0.67
Nodes (3): Dual Webhook Signing (HMAC-SHA256 + Ed25519), Timestamp Replay Protection, Standard Webhooks Format

## Knowledge Gaps
- **441 isolated node(s):** `name`, `version`, `description`, `type`, `license` (+436 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **75 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `setupEnv()` connect `setupEnv` to `NewManager`, `WebhookServer`, `event_filtering_test.go`, `parseUUID`, `WebhookServiceInterfaceWithTracing`, `WebhookServiceInterface`, `Context`, `NewService`, `WebhookConnectServer`?**
  _High betweenness centrality (0.115) - this node is a cross-community bridge._
- **Why does `Errorf()` connect `Errorf` to `event_filtering_test.go`, `ConvertProtoHTTPConfig`, `NewService`, `WebhookServiceInterfaceWithTracing`, `.ListDeliveries`, `Context`, `store/models.go`, `Config`, `BatchJob`, `toGRPCError`, `ClassifyError`, `PrepareDeliveryRequest`, `EventRecord`, `.GetRetryStatus`, `WebhookService`, `ValidateIP`, `otel.go`, `WebhookWorker`, `Context`, `Context`, `BatchJobWorker`, `ServiceError`, `parseUUID`, `TemplateEngine`, `NewManager`, `Context`, `WebhookService`, `jobInserter`, `EventArgs`, `RunAllMigrations`, `APIKeyAuth`, `GetTemplateFunctions`?**
  _High betweenness centrality (0.105) - this node is a cross-community bridge._
- **Why does `NewWebhookConnectServer()` connect `WebhookConnectServer` to `setupEnv`?**
  _High betweenness centrality (0.068) - this node is a cross-community bridge._
- **Are the 119 inferred relationships involving `Errorf()` (e.g. with `.Write()` and `.Execute()`) actually correct?**
  _`Errorf()` has 119 INFERRED edges - model-reasoned connections that need verification._
- **What connects `name`, `version`, `description` to the rest of the system?**
  _441 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `setupEnv` be split into smaller, more focused modules?**
  _Cohesion score 0.05171717171717172 - nodes in this community are weakly interconnected._
- **Should `webhook_pb.js` be split into smaller, more focused modules?**
  _Cohesion score 0.021052631578947368 - nodes in this community are weakly interconnected._