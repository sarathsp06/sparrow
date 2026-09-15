"""
Step implementations for API-surface specs (isolation, CRUD lifecycle,
negative cases). Endpoints not wrapped by SparrowAPI are called directly
with requests.
"""

import sys
import os
import json

import requests

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "libs"))

from getgauge.python import step, data_store


def _base():
    return data_store.suite["sparrow_url"]


def _ns():
    return data_store.scenario["consumer"]


def _ns_for(alias):
    return data_store.scenario[f"consumer_{alias}"]


def _webhook_id(name):
    return data_store.scenario[f"webhook_id_{name}"]


# ---------------------------------------------------------------------------
# Multi-consumer (isolation)
# ---------------------------------------------------------------------------

@step("Also use consumer <prefix> as <alias>")
def also_use_consumer(prefix, alias):
    ts = data_store.scenario["_ns_counter"]
    data_store.scenario[f"consumer_{alias}"] = f"{prefix}-{ts}"


@step("Push event <event> with payload <payload_json> to consumer <alias>")
def push_event_to_consumer(event, payload_json, alias):
    ns = _ns_for(alias)
    resp = requests.post(
        f"{_base()}/v1/consumers/{ns}/events",
        params={"event": event},
        json={"payload": json.loads(payload_json)},
    )
    assert resp.status_code == 201, f"Push failed: {resp.status_code} {resp.text}"


@step("GET webhook <name> via consumer <alias> should return status <code>")
def get_webhook_via_consumer(name, alias, code):
    ns = _ns_for(alias)
    resp = requests.get(f"{_base()}/v1/consumers/{ns}/webhooks/{_webhook_id(name)}")
    assert resp.status_code == int(code), f"Expected {code}, got {resp.status_code}: {resp.text}"


# ---------------------------------------------------------------------------
# Webhook CRUD
# ---------------------------------------------------------------------------

@step("GET webhook <name> should match target <target> and event <event>")
def get_webhook_matches(name, target, event):
    resp = requests.get(f"{_base()}/v1/consumers/{_ns()}/webhooks/{_webhook_id(name)}")
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    body = resp.json()
    expected_url = data_store.scenario[f"target_url_{target}"]
    assert body["url"] == expected_url, f"Expected url {expected_url}, got {body['url']}"
    assert event in body.get("events", []), f"Expected event {event} in {body.get('events')}"


@step("Webhook <name> should appear in the webhook list")
def webhook_in_list(name):
    resp = requests.get(f"{_base()}/v1/consumers/{_ns()}/webhooks")
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}"
    ids = [w["webhook_id"] for w in resp.json()["items"]]
    assert _webhook_id(name) in ids, f"Webhook {_webhook_id(name)} not in list: {ids}"


@step("PATCH webhook <name> url to target <target>")
def patch_webhook_url(name, target):
    new_url = data_store.scenario[f"target_url_{target}"]
    resp = requests.patch(
        f"{_base()}/v1/consumers/{_ns()}/webhooks/{_webhook_id(name)}",
        json={"url": new_url},
    )
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    assert resp.json()["url"] == new_url, f"PATCH did not apply url: {resp.json()['url']}"


@step("DELETE webhook <name>")
def delete_webhook(name):
    resp = requests.delete(f"{_base()}/v1/consumers/{_ns()}/webhooks/{_webhook_id(name)}")
    assert resp.status_code == 204, f"Expected 204, got {resp.status_code}: {resp.text}"


@step("GET webhook <name> should return status <code>")
def get_webhook_status(name, code):
    resp = requests.get(f"{_base()}/v1/consumers/{_ns()}/webhooks/{_webhook_id(name)}")
    assert resp.status_code == int(code), f"Expected {code}, got {resp.status_code}: {resp.text}"


# ---------------------------------------------------------------------------
# Subscription CRUD
# ---------------------------------------------------------------------------

@step("List subscriptions for webhook <name> and save the first")
def save_first_subscription(name):
    resp = requests.get(
        f"{_base()}/v1/consumers/{_ns()}/subscriptions",
        params={"webhook_id": _webhook_id(name)},
    )
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    items = resp.json()["items"]
    assert items, f"No subscriptions for webhook {name}"
    data_store.scenario["saved_subscription_id"] = items[0]["subscription_id"]


@step("GET saved subscription should return status <code>")
def get_saved_subscription(code):
    sub_id = data_store.scenario["saved_subscription_id"]
    resp = requests.get(f"{_base()}/v1/consumers/{_ns()}/subscriptions/{sub_id}")
    assert resp.status_code == int(code), f"Expected {code}, got {resp.status_code}: {resp.text}"


@step("PATCH saved subscription with template <template>")
def patch_saved_subscription(template):
    sub_id = data_store.scenario["saved_subscription_id"]
    resp = requests.patch(
        f"{_base()}/v1/consumers/{_ns()}/subscriptions/{sub_id}",
        json={"transform_enabled": True, "transform_template": template},
    )
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    body = resp.json()
    assert body["transform_template"] == template, f"Template not applied: {body}"


@step("DELETE saved subscription")
def delete_saved_subscription():
    sub_id = data_store.scenario["saved_subscription_id"]
    resp = requests.delete(f"{_base()}/v1/consumers/{_ns()}/subscriptions/{sub_id}")
    assert resp.status_code == 204, f"Expected 204, got {resp.status_code}: {resp.text}"


# ---------------------------------------------------------------------------
# Event type CRUD
# ---------------------------------------------------------------------------

@step("GET event type <name> should return status <code>")
def get_event_type_status(name, code):
    resp = requests.get(f"{_base()}/v1/event-types/{name}")
    assert resp.status_code == int(code), f"Expected {code}, got {resp.status_code}: {resp.text}"


@step("PATCH event type <name> description to <description>")
def patch_event_type(name, description):
    resp = requests.patch(
        f"{_base()}/v1/event-types/{name}", json={"description": description}
    )
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    body = resp.json()
    assert body["description"] == description, f"Description not applied: {body}"


@step("DELETE event type <name>")
def delete_event_type(name):
    resp = requests.delete(f"{_base()}/v1/event-types/{name}")
    assert resp.status_code == 204, f"Expected 204, got {resp.status_code}: {resp.text}"


# ---------------------------------------------------------------------------
# Negative / pagination
# ---------------------------------------------------------------------------

@step("Push event <event> with payload <payload_json> expecting status <code>")
def push_event_expect_status(event, payload_json, code):
    resp = requests.post(
        f"{_base()}/v1/consumers/{_ns()}/events",
        params={"event": event},
        json={"payload": json.loads(payload_json)},
    )
    assert resp.status_code == int(code), f"Expected {code}, got {resp.status_code}: {resp.text}"


@step("GET consumer path <suffix> should return status <code>")
def get_consumer_path_status(suffix, code):
    resp = requests.get(f"{_base()}/v1/consumers/{_ns()}{suffix}")
    assert resp.status_code == int(code), f"Expected {code}, got {resp.status_code}: {resp.text}"


@step("List deliveries with limit <limit> and offset <offset> should return <count> items")
def list_deliveries_page(limit, offset, count):
    resp = requests.get(
        f"{_base()}/v1/consumers/{_ns()}/deliveries",
        params={"limit": limit, "offset": offset},
    )
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    body = resp.json()
    assert len(body["items"]) == int(count), \
        f"Expected {count} items, got {len(body['items'])}"
    data_store.scenario["last_delivery_page"] = body


@step("Last delivery page should have total_count <total> and has_more <has_more>")
def assert_delivery_page_meta(total, has_more):
    p = data_store.scenario["last_delivery_page"]["pagination"]
    assert p["total_count"] == int(total), f"Expected total_count {total}, got {p['total_count']}"
    expected = has_more.lower() == "true"
    assert p["has_more"] is expected, f"Expected has_more {expected}, got {p['has_more']}"


@step("Paging through deliveries with limit <limit> should yield <count> unique deliveries")
def page_all_deliveries(limit, count):
    seen = set()
    offset = 0
    while True:
        resp = requests.get(
            f"{_base()}/v1/consumers/{_ns()}/deliveries",
            params={"limit": limit, "offset": offset},
        )
        assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
        body = resp.json()
        seen.update(d["delivery_id"] for d in body["items"])
        if not body["pagination"]["has_more"]:
            break
        offset += int(limit)
    assert len(seen) == int(count), f"Expected {count} unique deliveries, got {len(seen)}"
