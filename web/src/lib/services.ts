import { PUBLIC_API_URL } from "$env/static/public";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  WebhookService,
  EventService,
  SubscriptionService,
  DeliveryService,
  HealthService,
  NamespaceService,
} from "../../../proto/webhook_pb.js";

const transport = createConnectTransport({
  baseUrl: PUBLIC_API_URL || "/",
});

export const webhookClient = createClient(WebhookService, transport);
export const eventClient = createClient(EventService, transport);
export const subscriptionClient = createClient(SubscriptionService, transport);
export const deliveryClient = createClient(DeliveryService, transport);
export const healthClient = createClient(HealthService, transport);
export const namespaceClient = createClient(NamespaceService, transport);

// Backward compatibility
export const client = webhookClient;
