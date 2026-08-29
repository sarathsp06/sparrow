"""
WebhookTargetServer -- Programmable mock webhook endpoints for e2e tests.

Each target is a Flask app running in a background thread that captures all incoming
webhook deliveries. Targets can be configured with different behaviors (200, 404, 500,
slow response, fail-then-succeed, etc.)
"""

import json
import threading
import time
from dataclasses import dataclass, field
from typing import Optional

from flask import Flask, request as flask_request
from werkzeug.serving import make_server


@dataclass
class CapturedDelivery:
    body: dict | str
    headers: dict
    timestamp: float
    method: str
    path: str


class _Target:
    def __init__(self, name: str, behavior: str, port: int):
        self.name = name
        self.behavior = behavior
        self.port = port
        self.deliveries: list[CapturedDelivery] = []
        self._lock = threading.Lock()
        self._event = threading.Event()
        self._request_count = 0
        self._app = Flask(__name__)
        self._server: Optional[object] = None
        self._thread: Optional[threading.Thread] = None

        @self._app.route("/webhook", methods=["POST"])
        def handle():
            return self._handle_request()

    def _handle_request(self):
        with self._lock:
            self._request_count += 1
            count = self._request_count

        try:
            body = flask_request.get_json(force=True)
        except Exception:
            body = flask_request.get_data(as_text=True)

        delivery = CapturedDelivery(
            body=body,
            headers=dict(flask_request.headers),
            timestamp=time.time(),
            method=flask_request.method,
            path=flask_request.path,
        )
        with self._lock:
            self.deliveries.append(delivery)
            self._event.set()

        status_code = self._get_status_code(count)

        if self.behavior.startswith("slow_"):
            delay = int(self.behavior.split("_")[1].rstrip("s"))
            time.sleep(delay)
            status_code = 200

        if self.behavior.startswith("rate_limited_"):
            retry_after = self.behavior.split("_")[2]
            return "", 429, {"Retry-After": retry_after}

        return "", status_code

    def _get_status_code(self, request_number: int) -> int:
        if self.behavior == "ok":
            return 200
        elif self.behavior.startswith("status_"):
            return int(self.behavior.split("_")[1])
        elif self.behavior.startswith("fail_then_succeed_"):
            failures = int(self.behavior.split("_")[3])
            return 500 if request_number <= failures else 200
        elif self.behavior.startswith("slow_"):
            return 200
        elif self.behavior.startswith("rate_limited_"):
            return 429
        return 200

    def start(self):
        self._server = make_server("0.0.0.0", self.port, self._app)
        self._thread = threading.Thread(target=self._server.serve_forever, daemon=True)
        self._thread.start()

    def stop(self):
        if self._server:
            self._server.shutdown()

    def wait_for_deliveries(self, count: int, timeout: float = 30.0) -> list[CapturedDelivery]:
        deadline = time.time() + timeout
        while True:
            with self._lock:
                if len(self.deliveries) >= count:
                    return self.deliveries[:count]
            remaining = deadline - time.time()
            if remaining <= 0:
                with self._lock:
                    got = len(self.deliveries)
                raise TimeoutError(
                    f"Target '{self.name}': expected {count} deliveries, got {got} after {timeout}s"
                )
            self._event.wait(timeout=min(remaining, 0.5))
            self._event.clear()

    def switch_behavior(self, new_behavior: str):
        self.behavior = new_behavior


class WebhookTargetManager:
    """Manages mock webhook target servers."""

    def __init__(self):
        self._targets: dict[str, _Target] = {}
        self._next_port = 9100

    def start_target(self, name: str, behavior: str = "ok") -> str:
        """Start a mock webhook target. Returns the URL."""
        port = self._next_port
        self._next_port += 1
        target = _Target(name, behavior, port)
        target.start()
        self._targets[name] = target
        return f"http://host.docker.internal:{port}/webhook"

    def wait_for_deliveries(self, name: str, count: int, timeout: float = 30.0):
        self._targets[name].wait_for_deliveries(count, timeout)

    def get_delivery_count(self, name: str) -> int:
        target = self._targets[name]
        with target._lock:
            return len(target.deliveries)

    def get_latest_delivery(self, name: str) -> dict:
        target = self._targets[name]
        with target._lock:
            if not target.deliveries:
                raise ValueError(f"No deliveries for target '{name}'")
            d = target.deliveries[-1]
            return {"body": d.body, "headers": d.headers, "timestamp": d.timestamp}

    def switch_behavior(self, name: str, new_behavior: str):
        self._targets[name].switch_behavior(new_behavior)

    def assert_no_deliveries(self, name: str, wait: float = 3.0):
        time.sleep(wait)
        target = self._targets[name]
        with target._lock:
            count = len(target.deliveries)
        if count > 0:
            raise AssertionError(f"Expected 0 deliveries for '{name}', but got {count}")

    def stop_all(self):
        for target in self._targets.values():
            target.stop()
        self._targets.clear()
