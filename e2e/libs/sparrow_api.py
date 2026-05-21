"""
SparrowAPI -- HTTP client for the Sparrow Connect-RPC API.
"""

import time
import requests


class SparrowAPI:
    """Client for Sparrow Connect-RPC endpoints."""

    def __init__(self, base_url: str, api_key: str | None = None):
        self.base_url = base_url
        self.session = requests.Session()
        self.session.headers["Content-Type"] = "application/json"
        if api_key:
            self.session.headers["X-API-Key"] = api_key

    def register_event(self, name: str) -> dict:
        resp = self.session.post(f"{self.base_url}/webhook.EventService/RegisterEvent",
                                 json={"name": name, "active": True})
        resp.raise_for_status()
        return resp.json()

    def push_event(self, event: str, namespace: str, payload: dict, idempotency_key: str | None = None) -> dict:
        body = {"event": event, "namespace": namespace, "payload": payload}
        if idempotency_key:
            body["id"] = idempotency_key
        resp = self.session.post(f"{self.base_url}/webhook.EventService/PushEvent", json=body)
        resp.raise_for_status()
        return resp.json()

    def register_webhook(self, namespace: str, url: str, *events: str,
                         max_retries: int = 3, request_timeout: int = 10,
                         webhook_secret: str = "e2e-test-secret") -> dict:
        http_config = {
            "max_retries": max_retries,
            "request_timeout_seconds": request_timeout,
            "webhook_secret": webhook_secret,
        }
        body = {
            "namespace": namespace,
            "url": url,
            "active": True,
            "events": list(events),
            "http_config": http_config,
        }
        resp = self.session.post(f"{self.base_url}/webhook.WebhookService/RegisterWebhook", json=body)
        resp.raise_for_status()
        return resp.json()

    def subscribe_to_event(self, webhook_id: str, event_name: str, namespace: str,
                           template: str | None = None) -> dict:
        body = {"webhook_id": webhook_id, "event_name": event_name, "namespace": namespace}
        if template:
            body["transform_enabled"] = True
            body["transform_template"] = template
        resp = self.session.post(f"{self.base_url}/webhook.SubscriptionService/CreateSubscription", json=body)
        resp.raise_for_status()
        return resp.json()

    def list_deliveries(self, namespace: str | None = None, webhook_id: str | None = None,
                        event_id: str | None = None, status: str | None = None) -> dict:
        body = {}
        if namespace:
            body["namespace"] = namespace
        if webhook_id:
            body["webhook_id"] = webhook_id
        if event_id:
            body["event_id"] = event_id
        if status:
            body["status"] = status
        resp = self.session.post(f"{self.base_url}/webhook.DeliveryService/ListDeliveries", json=body)
        resp.raise_for_status()
        return resp.json()

    def get_delivery_status(self, delivery_id: str) -> dict:
        resp = self.session.post(f"{self.base_url}/webhook.DeliveryService/GetDeliveryStatus",
                                 json={"delivery_id": delivery_id})
        resp.raise_for_status()
        return resp.json()

    def get_delivery_attempts(self, delivery_id: str) -> dict:
        resp = self.session.post(f"{self.base_url}/webhook.DeliveryService/GetDeliveryAttempts",
                                 json={"delivery_id": delivery_id})
        resp.raise_for_status()
        return resp.json()

    def retry_delivery(self, delivery_id: str) -> dict:
        resp = self.session.post(f"{self.base_url}/webhook.DeliveryService/RetryDelivery",
                                 json={"delivery_id": delivery_id})
        resp.raise_for_status()
        return resp.json()

    def replay_event(self, event_id: str) -> dict:
        resp = self.session.post(f"{self.base_url}/webhook.EventService/RePushEvent",
                                 json={"event_id": event_id})
        resp.raise_for_status()
        return resp.json()

    def pause_webhook(self, webhook_id: str, namespace: str) -> dict:
        resp = self.session.post(f"{self.base_url}/webhook.WebhookService/PauseWebhook",
                                 json={"webhook_id": webhook_id, "namespace": namespace})
        resp.raise_for_status()
        return resp.json()

    def resume_webhook(self, webhook_id: str, namespace: str) -> dict:
        resp = self.session.post(f"{self.base_url}/webhook.WebhookService/ResumeWebhook",
                                 json={"webhook_id": webhook_id, "namespace": namespace})
        resp.raise_for_status()
        return resp.json()

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
                if status in ("DELIVERY_SUCCESS", "DELIVERY_EXPIRED"):
                    continue
                if status == "DELIVERY_FAILED":
                    cat = d.get("errorCategory", "")
                    attempts = int(d.get("attemptCount", 0))
                    max_attempts = int(d.get("maxAttempts", 1))
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

    def wait_for_delivery_terminal(self, delivery_id: str, timeout: float = 60.0) -> dict:
        """Poll a single delivery until terminal."""
        deadline = time.time() + timeout
        while True:
            resp = self.get_delivery_status(delivery_id)
            delivery = resp["delivery"]
            status = delivery.get("status", "")
            if status in ("DELIVERY_SUCCESS", "DELIVERY_FAILED", "DELIVERY_EXPIRED"):
                return delivery
            if time.time() >= deadline:
                raise TimeoutError(f"Delivery {delivery_id} not terminal within {timeout}s. Status: {status}")
            time.sleep(1)
