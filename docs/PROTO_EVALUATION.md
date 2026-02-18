# Evaluation of Sparrow Protobuf Definitions

This document provides a detailed evaluation of the Protobuf definitions in `proto/webhook.proto` from both Developer Experience (DX) and User Experience (UX) perspectives, along with recommendations for improvement.

## 1. Developer Experience (DX)

### Strengths
*   **Strong Typing:** Correct use of `google.protobuf.Timestamp` and `google.protobuf.Struct` for core logic and dynamic payloads.
*   **Discovery RPCs:** `GetTemplateFunctions` allows clients to discover template capabilities dynamically.
*   **Health Model:** Good balance between high-level health status and detailed metrics.

### Areas for Improvement
*   **Service Bloat ("The God Service"):** `WebhookService` contains 30+ RPCs, making it hard to navigate and leading to bulky generated code.
*   **RPC Overlap and Redundancy:**
    *   `GetWebhookStatus` vs `GetWebhookDeliveryHistory`: One is paginated, the other is not; both return delivery lists.
    *   `ResubmitWebhook` vs `ResendWebhook`: Different signatures for the same underlying retry action.
    *   `ListWebhooks` vs `GetRegisteredWebhooks`: Redundant listing/retrieval logic.
*   **Inconsistent Namespace Requirements:** Namespace is required for some resource actions (Create/Update) but missing in others (Delete/Unregister), creating potential security or ownership confusion.
*   **Non-Standard Error Handling:** Persistent use of `bool success` and `string message` in response bodies instead of relying on standard gRPC/Connect status codes.
*   **Missing Pagination:** Inconsistent application of `limit` and `offset` across `List` operations.

## 2. User Experience (UX)

### Strengths
*   **Multi-tenancy Support:** Consistent use of `namespace` (where present) enables logical isolation.
*   **Observability:** The API supports detailed auditing and health tracking out of the box.

### Areas for Improvement
*   **Subscription Model Confusion:** The dual approach where `RegisterWebhook` takes events while `CreateSubscription` exists independently is confusing.
*   **Configuration Complexity:** `WebhookHTTPConfig` is a single large message. Grouping these settings would improve clarity in both the API and UI.

---

## Recommendations

### 1. Service Decomposition
Split `WebhookService` into logical modules:
*   **WebhookService:** Core lifecycle (Create, Update, Delete, List).
*   **EventService:** Event definitions and reports.
*   **SubscriptionService:** Dedicated mapping of webhooks to events.
*   **DeliveryService:** Tracking, history, and retries.
*   **HealthService:** Metrics and summaries.

### 2. RPC Consolidation
*   **Consolidate Retries:** Replace `ResubmitWebhook` and `ResendWebhook` with a single `RetryDelivery` RPC.
*   **Standardize Listing:** Merge various listing calls into a single `ListWebhooks` RPC with flexible filters.
*   **Unified History:** Use a paginated `ListDeliveries` RPC for all delivery-related queries.

### 3. Pattern Standardization
*   **Common Pagination Message:** Implement a reusable `PageRequest` message.
*   **Idiomatic Errors:** Transition from custom `success`/`message` fields to transport-level status codes.
*   **Consistent Namespacing:** Require `namespace` for every RPC acting on a namespaced resource.

### 4. Model Refinement
*   **Rich Subscription Views:** Return `SubscriptionSummary` objects instead of simple `repeated string events` in webhook responses to provide more context (IDs, transformation status).
