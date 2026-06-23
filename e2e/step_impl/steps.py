"""
Step implementations for all Sparrow e2e specs.
"""

import sys
import os
import time
import json

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "libs"))

from getgauge.python import step, data_store
from libs.signature_verifier import delivery_has_signature_headers


def _api():
    return data_store.suite["api"]


def _targets():
    return data_store.scenario["targets"]


# ---------------------------------------------------------------------------
# Namespace
# ---------------------------------------------------------------------------

@step("Create namespace <prefix>")
def create_namespace(prefix):
    ts = data_store.scenario["_ns_counter"]
    ns = f"{prefix}-{ts}"
    data_store.scenario["namespace"] = ns


# ---------------------------------------------------------------------------
# Event registration
# ---------------------------------------------------------------------------

@step("Register event type <event_name>")
def register_event_type(event_name):
    _api().register_event(event_name)


# ---------------------------------------------------------------------------
# Webhook target
# ---------------------------------------------------------------------------

@step("Start target <name> with behavior <behavior>")
def start_target(name, behavior):
    url = _targets().start_target(name, behavior)
    data_store.scenario[f"target_url_{name}"] = url


@step("Start target <name>")
def start_target_ok(name):
    url = _targets().start_target(name, "ok")
    data_store.scenario[f"target_url_{name}"] = url


# ---------------------------------------------------------------------------
# Webhook registration
# ---------------------------------------------------------------------------

@step("Register webhook <name> in current namespace subscribed to <events>")
def register_webhook_subscribed(name, events):
    ns = data_store.scenario["namespace"]
    url = data_store.scenario[f"target_url_{name}"]
    event_list = [e.strip() for e in events.split(",")]
    resp = _api().register_webhook(ns, url, *event_list)
    data_store.scenario[f"webhook_id_{name}"] = resp.get("webhookId")
    data_store.scenario[f"webhook_resp_{name}"] = resp


@step("Register webhook <name> in current namespace subscribed to <events> with max_retries <retries>")
def register_webhook_retries(name, events, retries):
    ns = data_store.scenario["namespace"]
    url = data_store.scenario[f"target_url_{name}"]
    event_list = [e.strip() for e in events.split(",")]
    resp = _api().register_webhook(ns, url, *event_list, max_retries=int(retries))
    data_store.scenario[f"webhook_id_{name}"] = resp.get("webhookId")


@step("Register webhook <name> in current namespace subscribed to <events> with max_retries <retries> and timeout <timeout>")
def register_webhook_retries_timeout(name, events, retries, timeout):
    ns = data_store.scenario["namespace"]
    url = data_store.scenario[f"target_url_{name}"]
    event_list = [e.strip() for e in events.split(",")]
    resp = _api().register_webhook(ns, url, *event_list, max_retries=int(retries), request_timeout=int(timeout))
    data_store.scenario[f"webhook_id_{name}"] = resp.get("webhookId")


@step("Register webhook <name> in current namespace with no subscriptions")
def register_webhook_no_sub(name):
    ns = data_store.scenario["namespace"]
    url = data_store.scenario[f"target_url_{name}"]
    resp = _api().register_webhook(ns, url)
    data_store.scenario[f"webhook_id_{name}"] = resp.get("webhookId")


# ---------------------------------------------------------------------------
# Subscription
# ---------------------------------------------------------------------------

@step("Subscribe webhook <name> to <event> with template <template>")
def subscribe_with_template(name, event, template):
    webhook_id = data_store.scenario[f"webhook_id_{name}"]
    ns = data_store.scenario["namespace"]
    _api().subscribe_to_event(webhook_id, event, ns, template=template)


@step("Subscribe webhook <name> to <event> with broken template <template>")
def subscribe_with_broken_template(name, event, template):
    subscribe_with_template(name, event, template)


# ---------------------------------------------------------------------------
# Push event
# ---------------------------------------------------------------------------

@step("Push event <event> with payload <payload_json>")
def push_event(event, payload_json):
    ns = data_store.scenario["namespace"]
    payload = json.loads(payload_json)
    resp = _api().push_event(event, ns, payload)
    data_store.scenario["last_push_resp"] = resp
    data_store.scenario["last_event_id"] = resp.get("eventId")


@step("Push event <event> with payload <payload_json> and idempotency key <key>")
def push_event_with_key(event, payload_json, key):
    ns = data_store.scenario["namespace"]
    payload = json.loads(payload_json)
    resp = _api().push_event(event, ns, payload, idempotency_key=key)
    data_store.scenario["last_push_resp"] = resp
    data_store.scenario["last_event_id"] = resp.get("eventId")


# ---------------------------------------------------------------------------
# Wait / Assert deliveries
# ---------------------------------------------------------------------------

@step("Wait for <name> to receive <count> deliveries")
def wait_for_deliveries(name, count):
    _targets().wait_for_deliveries(name, int(count), timeout=30.0)


@step("Wait for <name> to receive <count> deliveries within <timeout> seconds")
def wait_for_deliveries_timeout(name, count, timeout):
    _targets().wait_for_deliveries(name, int(count), timeout=float(timeout))


@step("Target <name> should have received <count> deliveries")
def assert_delivery_count(name, count):
    actual = _targets().get_delivery_count(name)
    assert actual == int(count), f"Expected {count} deliveries for '{name}', got {actual}"


@step("Target <name> should have received at least <count> deliveries")
def assert_delivery_count_at_least(name, count):
    actual = _targets().get_delivery_count(name)
    assert actual >= int(count), f"Expected at least {count} deliveries for '{name}', got {actual}"


@step("Target <name> should have received no deliveries")
def assert_no_deliveries(name):
    _targets().assert_no_deliveries(name, wait=5.0)


@step("Latest delivery to <name> has body field <field> equal to <value>")
def assert_body_field(name, field, value):
    d = _targets().get_latest_delivery(name)
    actual = d["body"].get(field)
    assert str(actual) == value, f"Expected body[{field}]={value}, got {actual}"


@step("Latest delivery to <name> has enveloped payload field <field> equal to <value>")
def assert_enveloped_payload_field(name, field, value):
    d = _targets().get_latest_delivery(name)
    payload = d["body"].get("payload", {})
    actual = payload.get(field)
    assert str(actual) == value, f"Expected body.payload[{field}]={value}, got {actual}"


@step("Latest delivery to <name> has signature headers")
def assert_signature_headers(name):
    d = _targets().get_latest_delivery(name)
    delivery_has_signature_headers(d)


@step("Latest delivery to <name> body contains key <key>")
def assert_body_has_key(name, key):
    d = _targets().get_latest_delivery(name)
    assert key in d["body"], f"Key '{key}' not in body: {list(d['body'].keys())}"


@step("Latest delivery to <name> body does not contain key <key>")
def assert_body_not_has_key(name, key):
    d = _targets().get_latest_delivery(name)
    assert key not in d["body"], f"Key '{key}' unexpectedly in body"


# ---------------------------------------------------------------------------
# API delivery status
# ---------------------------------------------------------------------------

@step("Wait for all deliveries in current namespace to be terminal with count <count>")
def wait_all_terminal(count):
    ns = data_store.scenario["namespace"]
    deliveries = _api().wait_for_all_deliveries_terminal(ns, expected_count=int(count), timeout=60.0)
    data_store.scenario["terminal_deliveries"] = deliveries


@step("Wait for all deliveries in current namespace to be terminal with count <count> within <timeout> seconds")
def wait_all_terminal_timeout(count, timeout):
    ns = data_store.scenario["namespace"]
    deliveries = _api().wait_for_all_deliveries_terminal(ns, expected_count=int(count), timeout=float(timeout))
    data_store.scenario["terminal_deliveries"] = deliveries


@step("Delivery <index> should have status <status>")
def assert_delivery_status(index, status):
    d = data_store.scenario["terminal_deliveries"][int(index)]
    assert d["status"] == status, f"Expected status {status}, got {d['status']}"


@step("Delivery <index> should have error category <category>")
def assert_delivery_error_category(index, category):
    d = data_store.scenario["terminal_deliveries"][int(index)]
    assert d.get("errorCategory") == category, f"Expected errorCategory {category}, got {d.get('errorCategory')}"


@step("Delivery <index> should have attempt count <count>")
def assert_delivery_attempt_count(index, count):
    d = data_store.scenario["terminal_deliveries"][int(index)]
    actual = int(d.get("attemptCount", 0))
    assert actual == int(count), f"Expected attemptCount {count}, got {actual}"


@step("API should show <count> deliveries in current namespace")
def assert_api_delivery_count(count):
    ns = data_store.scenario["namespace"]
    resp = _api().list_deliveries(namespace=ns)
    deliveries = resp.get("deliveries", [])
    assert len(deliveries) == int(count), f"Expected {count} deliveries, got {len(deliveries)}"


@step("Get delivery attempts for delivery <index> and verify count is <count>")
def verify_attempts_count(index, count):
    d = data_store.scenario["terminal_deliveries"][int(index)]
    resp = _api().get_delivery_attempts(d["deliveryId"])
    attempts = resp.get("attempts", [])
    assert len(attempts) == int(count), f"Expected {count} attempts, got {len(attempts)}"


# ---------------------------------------------------------------------------
# Pause / Resume
# ---------------------------------------------------------------------------

@step("Pause webhook <name>")
def pause_webhook(name):
    webhook_id = data_store.scenario[f"webhook_id_{name}"]
    ns = data_store.scenario["namespace"]
    _api().pause_webhook(webhook_id, ns)


@step("Resume webhook <name>")
def resume_webhook(name):
    webhook_id = data_store.scenario[f"webhook_id_{name}"]
    ns = data_store.scenario["namespace"]
    _api().resume_webhook(webhook_id, ns)


@step("Wait <seconds> seconds")
def wait_seconds(seconds):
    time.sleep(float(seconds))


# ---------------------------------------------------------------------------
# Idempotency
# ---------------------------------------------------------------------------

@step("Last push response should not be duplicate")
def assert_not_duplicate():
    resp = data_store.scenario["last_push_resp"]
    assert not resp.get("duplicate", False), "Expected non-duplicate but got duplicate=true"


@step("Last push response should be duplicate")
def assert_is_duplicate():
    resp = data_store.scenario["last_push_resp"]
    assert resp.get("duplicate", False), "Expected duplicate=true but got false"


@step("Last push response event ID should match previous")
def assert_same_event_id():
    resp = data_store.scenario["last_push_resp"]
    prev_id = data_store.scenario.get("first_event_id")
    assert resp["eventId"] == prev_id, f"Event IDs differ: {resp['eventId']} != {prev_id}"


@step("Save event ID as first")
def save_first_event_id():
    data_store.scenario["first_event_id"] = data_store.scenario["last_event_id"]


# ---------------------------------------------------------------------------
# Retry / Replay
# ---------------------------------------------------------------------------

@step("Retry the failed delivery")
def retry_failed_delivery():
    deliveries = data_store.scenario["terminal_deliveries"]
    failed = [d for d in deliveries if d["status"] == "DELIVERY_FAILED"]
    assert failed, "No failed delivery found to retry"
    delivery_id = failed[0]["deliveryId"]
    _api().retry_delivery(delivery_id)
    # Wait for retry to complete
    result = _api().wait_for_delivery_terminal(delivery_id, timeout=30.0)
    data_store.scenario["retried_delivery"] = result


@step("Retried delivery should have status <status>")
def assert_retried_status(status):
    d = data_store.scenario["retried_delivery"]
    assert d["status"] == status, f"Expected {status}, got {d['status']}"


@step("Switch target <name> to behavior <behavior>")
def switch_behavior(name, behavior):
    _targets().switch_behavior(name, behavior)


@step("Replay the last pushed event")
def replay_event():
    event_id = data_store.scenario["last_event_id"]
    _api().replay_event(event_id)


# ---------------------------------------------------------------------------
# Auth
# ---------------------------------------------------------------------------

@step("GET <path> should return status <code>")
def get_path_status(path, code):
    import requests
    url = data_store.suite["sparrow_url"]
    resp = requests.get(f"{url}{path}")
    assert resp.status_code == int(code), f"Expected {code}, got {resp.status_code}"


@step("POST <path> with empty body should return status <code>")
def post_path_status(path, code):
    import requests
    url = data_store.suite["sparrow_url"]
    resp = requests.post(f"{url}{path}", json={}, headers={"Content-Type": "application/json"})
    assert resp.status_code == int(code), f"Expected {code}, got {resp.status_code}"
