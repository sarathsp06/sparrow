---
type: Go Package
title: internal/webhooks (service)
description: Core business logic — webhook registration, event processing, delivery orchestration
tags: [webhooks, core, service-layer]
timestamp: 2026-07-06T16:46:48Z
---

# internal/webhooks (service)

The central service layer defining `WebhookServiceInterface` and its implementation `WebhookService`.

## WebhookServiceInterface

`WebhookServiceInterface` is a composite interface that embeds smaller domain interfaces. This preserves the existing service seam used by `internal/rest`, while making the domain-specific seams explicit for tests and future refactors.

Embedded domain interfaces:

- **WebhookRegistrationService**: RegisterWebhook, CreateWebhook, UnregisterWebhook, ListWebhooks, UpdateWebhookConfig, PauseWebhook, ResumeWebhook, GetNamespaceStats
- **EventService**: RegisterEvent, ListEvents, UpdateEvent, DeleteEvent, GetEvent, PushEvent, RePushEvent, GetEventRecord, ListEventReports
- **SubscriptionService**: CreateSubscription, GetSubscription, ListSubscriptions, UpdateSubscription, DeleteSubscription, TestSubscriptionTemplate
- **DeliveryService**: GetDeliveryStatus, GetDeliveryAttempts, ListDeliveries, RetryDelivery
- **HealthService**: GetWebhookHealth, ListWebhooksByHealth, GetHealthSummary
- **BatchService**: RePushEvents, GetRepushStatus, CancelRepush, RetryDeliveries, GetRetryStatus, CancelRetry
- **TemplateMetadataService**: GetTemplateFunctions
- **WebhookRepositoryAccessor**: GetWebhookRepo
- **WebhookSecretPresenter**: DecryptWebhookSecret, DecryptSecretHeaders, WebhookSigningPublicKeyHex

`WebhookSecretPresenter` keeps encrypted webhook secret presentation and Ed25519 public-key derivation behind the service seam, so transport modules do not decrypt private-key material directly.

## Key Types

- `WebhookRegistration` — domain model with URL, headers, health, HTTPConfig, secrets
- `WebhookHTTPConfig` — HTTP delivery configuration (retries, timeout, SSL, rate limit)
- `WebhookRegistrationRequest` — creation input
- `WebhookHealthData` — health metrics with consecutive failures, success rate, response time

## Citations

- `internal/webhooks/webhook_service.go`
- `internal/rest/conversions.go` — uses `WebhookSigningPublicKeyHex` for safe signing-key presentation
