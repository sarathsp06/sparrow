---
type: Go Package
title: internal/webhooks (service)
description: Core business logic — webhook registration, event processing, delivery orchestration
tags: [webhooks, core, service-layer]
timestamp: 2026-06-22T00:00:00Z
---

# internal/webhooks (service)

The central service layer defining `WebhookServiceInterface` (~35 methods) and its implementation `WebhookService`.

## WebhookServiceInterface

The master interface covering all domains:

- **Webhooks**: RegisterWebhook, UnregisterWebhook, ListWebhooks, UpdateWebhookConfig, PauseWebhook, ResumeWebhook
- **Events**: RegisterEvent, ListEvents, UpdateEvent, DeleteEvent, GetEvent, PushEvent, ListEventReports, GetEventRecord, RePushEvent
- **Subscriptions**: CreateSubscription, GetSubscription, ListSubscriptions, UpdateSubscription, DeleteSubscription, TestSubscriptionTemplate
- **Deliveries**: GetDeliveryStatus, ListDeliveries, RetryDelivery, GetDeliveryAttempts
- **Health**: GetWebhookHealth, ListWebhooksByHealth, GetHealthSummary, GetNamespaceStats
- **Batch**: RePushEvents, GetRepushStatus, CancelRepush, RetryDeliveries, GetRetryStatus, CancelRetry
- **Crypto**: DecryptWebhookSecret, DecryptSecretHeaders, GetEd25519PrivateKey
- **Metadata**: GetTemplateFunctions

## Key Types

- `WebhookRegistration` — domain model with URL, headers, health, HTTPConfig, secrets
- `WebhookHTTPConfig` — HTTP delivery configuration (retries, timeout, SSL, rate limit)
- `WebhookRegistrationRequest` — creation input
- `WebhookHealthData` — health metrics with consecutive failures, success rate, response time

## Citations

- `internal/webhooks/service.go`
- `internal/webhooks/webhook_service_interface.go`
