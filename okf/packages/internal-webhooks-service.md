---
type: Go Package
title: internal/webhooks (service)
description: Core business logic — webhook registration, event processing, delivery orchestration
tags: [webhooks, core, service-layer]
timestamp: 2026-07-06T16:46:48Z
---

# internal/webhooks (service)

The central service layer. `WebhookService` implements seven narrow domain
interfaces; consumers in `internal/rest` depend on the single slice they need,
not on the whole service.

## Domain interfaces

`WebhookServiceInterface` is a composite that embeds the seven interfaces below.
It exists only for DI wiring and the generated OTel decorator (`//go:generate
gowrap -i WebhookServiceInterface`); new consumers should depend on a single
domain interface instead.

- **WebhookManager**: RegisterWebhook, CreateWebhook, UnregisterWebhook, ListWebhooks, UpdateWebhookConfig, PauseWebhook, ResumeWebhook, GetConsumerStats
- **EventManager**: RegisterEvent, ListEvents, UpdateEvent, DeleteEvent, GetEvent, PushEvent, RePushEvent, GetEventRecord, ListEventReports
- **SubscriptionManager**: CreateSubscription, GetSubscription, ListSubscriptions, UpdateSubscription, DeleteSubscription, TestSubscriptionTemplate, ListSubscriptionsByWebhookIDs, GetTemplateFunctions
- **DeliveryManager**: GetDeliveryStatus, GetDeliveryAttempts, ListDeliveries, RetryDelivery
- **HealthManager**: GetWebhookHealth, GetHealthSummary
- **BatchManager**: RePushEvents, GetRepushStatus, CancelRepush, RetryDeliveries, GetRetryStatus, CancelRetry
- **SecretRevealer**: DecryptSecretHeaders, DecryptWebhookSecret, WebhookSigningPublicKeyHex

`SecretRevealer` keeps encrypted webhook secret presentation and Ed25519
public-key derivation behind the service seam, so transport modules never
decrypt private-key material directly. Subscription payload transforms run
through `pkg/template` (extracted from the webhook client).

## Key Types

- `WebhookRegistration` — domain model with URL, headers, health, HTTPConfig, secrets
- `WebhookHTTPConfig` — HTTP delivery configuration (retries, timeout, SSL, rate limit)
- `WebhookRegistrationRequest` — creation input
- `WebhookHealthData` — health metrics with consecutive failures, success rate, response time

## Citations

- `internal/webhooks/webhook_service.go`
- `internal/rest/conversions.go` — uses `WebhookSigningPublicKeyHex` for safe signing-key presentation
