import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { WebhookService } from "../../../proto/webhook_pb.js";

const transport = createConnectTransport({
  baseUrl: "http://localhost:8080",
});

export const client = createClient(WebhookService, transport);
