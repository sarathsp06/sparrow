import { PUBLIC_API_URL } from "$env/static/public";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { WebhookService } from "../../../proto/webhook_pb.js";

const transport = createConnectTransport({
  baseUrl: PUBLIC_API_URL as string || "http://localhost:8080",
});

export const client = createClient(WebhookService, transport);