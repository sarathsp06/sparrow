"""
SparrowAPI -- wraps the generated REST client (sparrow_client, generated from
api/openapi.yaml via `openapi-python-client`) with the same method surface the
Gauge step implementations already use. Sparrow's interface is REST/OpenAPI
only (Connect-RPC and gRPC have been removed).
"""

import sys
import time
from pathlib import Path

# The generated client lives at client/python/sparrow_client (repo-relative).
_CLIENT_ROOT = Path(__file__).resolve().parents[2] / "client" / "python"
if str(_CLIENT_ROOT) not in sys.path:
    sys.path.insert(0, str(_CLIENT_ROOT))

from sparrow_client import AuthenticatedClient, Client  # noqa: E402
from sparrow_client.api.event_types import register_event_type  # noqa: E402
from sparrow_client.api.events import push_event, repush_event  # noqa: E402
from sparrow_client.api.webhooks import (  # noqa: E402
    register_webhook,
    pause_webhook,
    resume_webhook,
)
from sparrow_client.api.subscriptions import create_subscription  # noqa: E402
from sparrow_client.api.deliveries import (  # noqa: E402
    list_deliveries,
    get_delivery,
    get_delivery_attempts,
    retry_delivery as retry_delivery_op,
)
from sparrow_client.models.event_type_body import EventTypeBody  # noqa: E402
from sparrow_client.models.register_webhook_body import RegisterWebhookBody  # noqa: E402
from sparrow_client.models.webhook_http_config import WebhookHTTPConfig  # noqa: E402
from sparrow_client.models.create_subscription_body import CreateSubscriptionBody  # noqa: E402
from sparrow_client.models.push_event_body import PushEventBody  # noqa: E402
from sparrow_client.models.push_event_body_payload import PushEventBodyPayload  # noqa: E402
from sparrow_client.types import UNSET  # noqa: E402


def _to_dict(model) -> dict:
    """Convert a generated attrs model (or None) to a plain dict."""
    if model is None:
        return {}
    return model.to_dict()


class SparrowAPI:
    """Client for Sparrow's REST API, built on the generated sparrow_client SDK."""

    def __init__(self, base_url: str, api_key: str | None = None):
        self.base_url = base_url
        if api_key:
            self.client = AuthenticatedClient(
                base_url=base_url, token=api_key, prefix="", auth_header_name="X-API-Key"
            )
        else:
            self.client = Client(base_url=base_url)

    def register_event(self, name: str) -> dict:
        resp = register_event_type.sync_detailed(
            client=self.client, body=EventTypeBody(name=name, active=True)
        )
        return _to_dict(resp.parsed)

    def push_event(self, event: str, namespace: str, payload: dict, idempotency_key: str | None = None) -> dict:
        body = PushEventBody(payload=PushEventBodyPayload.from_dict(payload))
        if idempotency_key:
            body.idempotency_key = idempotency_key
        resp = push_event.sync_detailed(namespace=namespace, client=self.client, body=body, event=event)
        return _to_dict(resp.parsed)

    def register_webhook(self, namespace: str, url: str, *events: str,
                          max_retries: int = 3, request_timeout: int = 10) -> dict:
        http_config = WebhookHTTPConfig(
            max_retries=max_retries,
            request_timeout_seconds=request_timeout,
            capture_response_body=True,
        )
        body = RegisterWebhookBody(events=list(events), url=url, active=True, http_config=http_config)
        resp = register_webhook.sync_detailed(namespace=namespace, client=self.client, body=body)
        return _to_dict(resp.parsed)

    def subscribe_to_event(self, webhook_id: str, event_name: str, namespace: str,
                            template: str | None = None) -> dict:
        body = CreateSubscriptionBody(webhook_id=webhook_id, event_name=event_name)
        if template:
            body.transform_enabled = True
            body.transform_template = template
        resp = create_subscription.sync_detailed(namespace=namespace, client=self.client, body=body)
        return _to_dict(resp.parsed)

    def list_deliveries(self, namespace: str | None = None, webhook_id: str | None = None,
                         event_id: str | None = None, status: str | None = None) -> dict:
        resp = list_deliveries.sync_detailed(
            namespace=namespace or "",
            client=self.client,
            webhook_id=webhook_id if webhook_id else UNSET,
            event_id=event_id if event_id else UNSET,
            status=status if status else UNSET,
        )
        out = _to_dict(resp.parsed)
        # steps.py reads out["deliveries"]; the REST API returns "items".
        out["deliveries"] = out.get("items", [])
        return out
    def get_delivery_status(self, namespace: str, delivery_id: str) -> dict:
        resp = get_delivery.sync_detailed(namespace=namespace, delivery_id=delivery_id, client=self.client)
        d = _to_dict(resp.parsed)
        return {"delivery": d}

    def get_delivery_attempts(self, namespace: str, delivery_id: str) -> dict:
        resp = get_delivery_attempts.sync_detailed(namespace=namespace, delivery_id=delivery_id, client=self.client)
        return _to_dict(resp.parsed)

    def retry_delivery(self, namespace: str, delivery_id: str) -> dict:
        resp = retry_delivery_op.sync_detailed(namespace=namespace, delivery_id=delivery_id, client=self.client)
        return _to_dict(resp.parsed)

    def replay_event(self, event_id: str) -> dict:
        resp = repush_event.sync_detailed(event_id=event_id, client=self.client)
        return _to_dict(resp.parsed)

    def pause_webhook(self, webhook_id: str, namespace: str) -> dict:
        resp = pause_webhook.sync_detailed(namespace=namespace, webhook_id=webhook_id, client=self.client)
        return {"status_code": resp.status_code}

    def resume_webhook(self, webhook_id: str, namespace: str) -> dict:
        resp = resume_webhook.sync_detailed(namespace=namespace, webhook_id=webhook_id, client=self.client)
        return {"status_code": resp.status_code}

    def wait_for_all_deliveries_terminal(self, namespace: str, expected_count: int = 1,
                                          timeout: float = 60.0) -> list[dict]:
        """Poll until all deliveries reach terminal status."""
        non_retryable = {"client_error", "dns_error", "tls_error", "unexpected_status"}
        deadline = time.time() + timeout
        while True:
            resp = self.list_deliveries(namespace=namespace)
            deliveries = resp.get("deliveries", [])
            all_terminal = True
            for d in deliveries:
                status = d.get("status", "")
                if status in ("success", "expired"):
                    continue
                if status == "failed":
                    cat = d.get("error_category", "")
                    attempts = int(d.get("attempt_count", 0))
                    max_attempts = int(d.get("max_attempts", 1))
                    if cat in non_retryable or attempts >= max_attempts:
                        continue
                all_terminal = False
                break

            if all_terminal and len(deliveries) >= expected_count:
                return deliveries

            if time.time() >= deadline:
                raise TimeoutError(
                    f"Not all deliveries terminal within {timeout}s. "
                    f"Count: {len(deliveries)}/{expected_count}"
                )
            time.sleep(2)

    def wait_for_delivery_terminal(self, namespace: str, delivery_id: str, timeout: float = 60.0) -> dict:
        """Poll a single delivery until terminal."""
        deadline = time.time() + timeout
        while True:
            resp = self.get_delivery_status(namespace, delivery_id)
            delivery = resp["delivery"]
            status = delivery.get("status", "")
            if status in ("success", "failed", "expired"):
                return delivery
            if time.time() >= deadline:
                raise TimeoutError(f"Delivery {delivery_id} not terminal within {timeout}s. Status: {status}")
            time.sleep(1)
