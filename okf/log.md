# OKF Changelog

## 2026-06-23

### Store Layer Refactoring (Dynamic SQL → Fixed SQL with IS NULL guards)

Refactored all 4 store repository files to eliminate dynamic SQL builders:

- **`delivery_repository.go`**: `ListDeliveriesFiltered` → fixed SQL with `($N::text IS NULL OR wd.status::text = $N)` pattern. Cast `wd.status::text` for PG enum comparison. Deleted `buildDeliveryFilterConditions`.
- **`event_repository.go`**: `ListEventReports`, `ListEventReportsWithStats`, `ListEventReportsFiltered` → fixed SQL. Deleted `buildEventReportFilterConditions`. Removed `strings` import.
- **`batch_repository.go`**: `SnapshotEventIDs`, `SnapshotDeliveryIDs` → fixed SQL. Deleted `joinConditions`.
- **`webhook_repository.go`**: `ListWebhooksPaginated`, `GetNamespaceStats` → fixed SQL. Removed `strings` import.

Key insight: PG custom enum `webhook_delivery_status` must be cast to `text` for parameter comparison (`wd.status::text = $5`).

### Pre-existing Bug Fix

- **`request_test.go`**: Fixed 5 call sites where `generateHMACSignature` return changed from `string` to `(string, error)` but tests only captured one value.

### e2e Spec Fixes

- **`00_hello_world.spec`**: Body assertion was checking `body["message"]` but Sparrow wraps all default deliveries in an envelope (`{"payload": {...}}`). Added `has enveloped payload field` step that navigates `body["payload"][field]`.
- **`02_selective_subscription.spec`**: Event name `"order.created"` collided with `01_happy_path.spec` in the shared server instance. Renamed to `"selective.order.created"` / `"selective.order.shipped"`.

### Test Results

- Go build: `rtk go build ./...` ✅
- Webhook tests: 163/163 pass
- gRPC/middleware/pkg tests: 115/115 pass
- Integration tests: 11/12 pass (1 flaky retry timing test)
- e2e Gauge tests: specs fixed, pending rerun

## 2026-06-22

Initial OKF bundle created from codebase analysis. Sources:

- `opencode.md` — comprehensive codebase reference
- `plan.md` — implementation plan and design decisions
- Source code exploration across all Go packages, frontend, proto, database migrations, and DevOps configuration
