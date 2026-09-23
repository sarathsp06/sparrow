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


# ---------------------------------------------------------------------------
# Alert configs
# ---------------------------------------------------------------------------

@step("Create alert config <name> with email <email> for events <events>")
def create_alert_config_wide(name, email, events):
    event_list = [e.strip() for e in events.split(",")]
    resp = requests.post(
        f"{_base()}/v1/consumers/{_ns()}/alert-configs",
        json={"email": email, "event_types": event_list},
    )
    assert resp.status_code == 201, f"Expected 201, got {resp.status_code}: {resp.text}"
    body = resp.json()
    assert body["email"] == email, f"email not echoed: {body}"
    assert body["event_types"] == event_list, f"event_types not echoed: {body}"
    assert not body.get("webhook_id"), f"expected consumer-wide (no webhook_id): {body}"
    data_store.scenario[f"alert_id_{name}"] = body["id"]


@step("Create alert config <name> for webhook <wh> with email <email> for events <events>")
def create_alert_config_scoped(name, wh, email, events):
    event_list = [e.strip() for e in events.split(",")]
    resp = requests.post(
        f"{_base()}/v1/consumers/{_ns()}/alert-configs",
        json={"webhook_id": _webhook_id(wh), "email": email, "event_types": event_list},
    )
    assert resp.status_code == 201, f"Expected 201, got {resp.status_code}: {resp.text}"
    body = resp.json()
    assert body["webhook_id"] == _webhook_id(wh), f"webhook scope not echoed: {body}"
    data_store.scenario[f"alert_id_{name}"] = body["id"]


@step("Alert configs list should have <count> items")
def alert_configs_count(count):
    resp = requests.get(f"{_base()}/v1/consumers/{_ns()}/alert-configs")
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    items = resp.json()["items"]
    assert len(items) == int(count), f"Expected {count} configs, got {len(items)}: {items}"


@step("Alert configs list for webhook <wh> should have <count> items")
def alert_configs_count_for_webhook(wh, count):
    resp = requests.get(
        f"{_base()}/v1/consumers/{_ns()}/alert-configs",
        params={"webhook_id": _webhook_id(wh)},
    )
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    items = resp.json()["items"]
    assert len(items) == int(count), f"Expected {count} configs, got {len(items)}: {items}"


@step("Delete alert config <name>")
def delete_alert_config(name):
    cfg_id = data_store.scenario[f"alert_id_{name}"]
    resp = requests.delete(f"{_base()}/v1/consumers/{_ns()}/alert-configs/{cfg_id}")
    assert resp.status_code == 204, f"Expected 204, got {resp.status_code}: {resp.text}"


@step("Delete alert config with id <cfg_id> should return status <code>")
def delete_alert_config_status(cfg_id, code):
    resp = requests.delete(f"{_base()}/v1/consumers/{_ns()}/alert-configs/{cfg_id}")
    assert resp.status_code == int(code), f"Expected {code}, got {resp.status_code}: {resp.text}"


@step("Create alert config for webhook id <wh_id> with email <email> for events <events> expecting status <code>")
def create_alert_config_bad_webhook(wh_id, email, events, code):
    event_list = [e.strip() for e in events.split(",")]
    resp = requests.post(
        f"{_base()}/v1/consumers/{_ns()}/alert-configs",
        json={"webhook_id": wh_id, "email": email, "event_types": event_list},
    )
    assert resp.status_code == int(code), f"Expected {code}, got {resp.status_code}: {resp.text}"


# ---------------------------------------------------------------------------
# Recipes
# ---------------------------------------------------------------------------

@step("GET recipes should return a non-empty catalog")
def get_recipes_catalog():
    resp = requests.get(f"{_base()}/v1/recipes")
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    items = resp.json()["items"]
    assert len(items) > 0, "recipe catalog is empty"
    assert all(i.get("name") for i in items), f"recipe missing name: {items}"


# ---------------------------------------------------------------------------
# Portal tokens
# ---------------------------------------------------------------------------

@step("Mint portal token for current consumer")
def mint_portal_token():
    resp = requests.post(f"{_base()}/v1/consumers/{_ns()}/portal-token")
    assert resp.status_code == 201, f"Expected 201, got {resp.status_code}: {resp.text}"
    body = resp.json()
    assert body.get("token"), f"no token in response: {body}"
    data_store.scenario["portal_token"] = body["token"]


@step("Portal GET <suffix> should return status <code>")
def portal_get(suffix, code):
    token = data_store.scenario["portal_token"]
    resp = requests.get(
        f"{_base()}/portal/api/{suffix}",
        headers={"Authorization": f"Bearer {token}"},
    )
    assert resp.status_code == int(code), f"Expected {code}, got {resp.status_code}: {resp.text}"


@step("Portal GET <suffix> without token should return status <code>")
def portal_get_no_token(suffix, code):
    resp = requests.get(f"{_base()}/portal/api/{suffix}")
    assert resp.status_code == int(code), f"Expected {code}, got {resp.status_code}: {resp.text}"


@step("Portal webhooks list should contain webhook <name>")
def portal_webhooks_contains(name):
    token = data_store.scenario["portal_token"]
    resp = requests.get(
        f"{_base()}/portal/api/webhooks",
        headers={"Authorization": f"Bearer {token}"},
    )
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    ids = [w["webhook_id"] for w in resp.json()["items"]]
    assert _webhook_id(name) in ids, f"webhook {_webhook_id(name)} not in portal list: {ids}"
