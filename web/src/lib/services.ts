import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { WebhookService } from "../../../proto/webhook_pb.js";

const transport = createConnectTransport({
  baseUrl: "https://sparrow.sarathsadasivan.com",
});

export const client = createClient(WebhookService, transport);
