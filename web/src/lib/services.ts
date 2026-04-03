import { PUBLIC_API_URL, PUBLIC_API_KEY } from "$env/static/public";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import type { Interceptor } from "@connectrpc/connect";
import {
  WebhookService,
  EventService,
  SubscriptionService,
  DeliveryService,
  HealthService,
} from "../../../proto/webhook_pb.js";

// Runtime config injected by the Go server into window.__SPARROW_CONFIG__.
// When running the UI standalone (npm run dev / npm run build), falls back
// to the PUBLIC_API_KEY env var set in web/.env or at build time.
interface SparrowConfig {
  apiKey?: string;
}

declare global {
  interface Window {
    __SPARROW_CONFIG__?: SparrowConfig;
  }
}

const runtimeConfig: SparrowConfig =
  (typeof window !== "undefined" && window.__SPARROW_CONFIG__) || {};

// Resolve the API key:
//   1. Runtime injection from Go server (embedded mode)
//   2. Build-time env var PUBLIC_API_KEY (standalone mode)
const apiKey: string = runtimeConfig.apiKey || PUBLIC_API_KEY || "";

// Interceptor that attaches the API key header to every request when configured.
const apiKeyInterceptor: Interceptor = (next) => async (req) => {
  req.header.set("X-API-Key", apiKey);
  return next(req);
};

const interceptors: Interceptor[] = [];
if (apiKey) {
  interceptors.push(apiKeyInterceptor);
}

const transport = createConnectTransport({
  baseUrl: PUBLIC_API_URL || "/",
  interceptors,
});

export const webhookClient = createClient(WebhookService, transport);
export const eventClient = createClient(EventService, transport);
export const subscriptionClient = createClient(SubscriptionService, transport);
export const deliveryClient = createClient(DeliveryService, transport);
export const healthClient = createClient(HealthService, transport);

// Backward compatibility
export const client = webhookClient;
