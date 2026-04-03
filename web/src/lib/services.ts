import { PUBLIC_API_URL } from "$env/static/public";
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
// The API key is always provided at runtime (never baked into the build).
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

const apiKey: string = runtimeConfig.apiKey || "";

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
