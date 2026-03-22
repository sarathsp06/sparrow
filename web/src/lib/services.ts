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
  NamespaceMembershipService,
  TeamService
} from "../../../proto/webhook_pb.js";
import { getSessionToken } from "./auth.js";

// Create a transport that automatically injects the auth session token
// as a Bearer token on every request. If no auth provider is active or
// the user is not signed in, requests are sent without an Authorization
// header (backwards compatible).
const transport = createConnectTransport({
  baseUrl: PUBLIC_API_URL || "/",
  interceptors: [
    (next) => async (req) => {
      const token = await getSessionToken();
      if (token) {
        req.header.set("Authorization", `Bearer ${token}`);
      }
      return next(req);
    },
  ],
});

export const webhookClient = createClient(WebhookService, transport);
export const eventClient = createClient(EventService, transport);
export const subscriptionClient = createClient(SubscriptionService, transport);
export const deliveryClient = createClient(DeliveryService, transport);
export const healthClient = createClient(HealthService, transport);
export const namespaceClient = createClient(NamespaceService, transport);
export const namespaceMembershipClient = createClient(NamespaceMembershipService, transport);
export const teamClient = createClient(TeamService, transport);

// Backward compatibility
export const client = webhookClient;
