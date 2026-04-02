/**
 * API Reference Data Types
 *
 * These types define the structure for API service documentation.
 * Each service is a single .ts file exporting an ApiService object.
 *
 * To add a new endpoint:
 *   1. Open the service's .ts file (e.g., webhook-service.ts)
 *   2. Add a new object to the `rpcs` array with request/response fields
 *   3. Add `example` values to fields — curl and response JSON are auto-generated
 *   4. Rebuild docs: make docs-build
 */

export interface Field {
  /** Field name (e.g., "namespace", "webhook_id") */
  name: string;
  /** Type (e.g., "string", "int32", "bool", "Timestamp") */
  type: string;
  /** Whether the field is required (request fields only) */
  required?: boolean;
  /** Human-readable description */
  description: string;
  /** Example value — used to auto-generate curl commands and response JSON */
  example?: string | number | boolean | object;
}

export interface ErrorCode {
  /** gRPC error code (e.g., "NOT_FOUND", "ALREADY_EXISTS") */
  code: string;
  /** When this error is returned */
  description: string;
}

export interface Rpc {
  /** RPC name (e.g., "RegisterWebhook") — used as heading and to auto-generate paths */
  name: string;
  /** What this RPC does */
  description: string;
  /** Request fields. Empty array for no-input RPCs. */
  request: Field[];
  /** Response fields. Omit or empty array if response is empty. */
  response?: Field[];
  /** gRPC error codes. Omit if none specific. */
  errors?: ErrorCode[];
}

export interface ApiService {
  /** Proto service name (e.g., "WebhookService") */
  service: string;
  /** One-paragraph description shown at top of page */
  description: string;
  /** Optional markdown content rendered after description (e.g., tables, notes) */
  notes?: string;
  /** All RPCs in this service */
  rpcs: Rpc[];
  /** Optional markdown content rendered at the bottom (e.g., shared types, lifecycle diagrams) */
  footer?: string;
}
