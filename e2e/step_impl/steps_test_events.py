"""
Steps for test events (22_test_events.spec).
"""

import requests

from getgauge.python import step, data_store


def _base():
    return data_store.suite["sparrow_url"]


def _ns():
    return data_store.scenario["consumer"]


def _webhook_id(name):
    return data_store.scenario[f"webhook_id_{name}"]


def _subscription(name):
    resp = requests.get(f"{_base()}/v1/consumers/{_ns()}/subscriptions", params={"webhook_id": _webhook_id(name)})
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    items = resp.json()["items"]
    assert items, f"webhook {name} has no subscriptions"
    return items[0]


def _send_test_event(name):
    sub = _subscription(name)
    return requests.post(
        f"{_base()}/v1/consumers/{_ns()}/subscriptions/{sub['subscription_id']}:sendTestEvent",
        json={"event_name": sub["event_name"]},
    )


@step("Send a test event to webhook <name>")
def send_test_event(name):
    resp = _send_test_event(name)
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    data_store.scenario["test_event_result"] = resp.json()


@step("Test event result should have status code <code>")
def test_event_status(code):
    result = data_store.scenario["test_event_result"]
    assert result["status_code"] == int(code), f"Expected {code}: {result}"
    assert not result.get("error_class"), f"Unexpected error class: {result}"


@step("Latest delivery to <name> has header <header> equal to <value>")
def latest_delivery_header(name, header, value):
    d = data_store.scenario["targets"].get_latest_delivery(name)
    headers = {k.lower(): v for k, v in d["headers"].items()}
    assert headers.get(header.lower()) == value, f"{header} = {headers.get(header.lower())!r}, want {value!r}"


@step("Webhook <name> should have no recorded deliveries")
def webhook_has_no_deliveries(name):
    resp = requests.get(f"{_base()}/v1/consumers/{_ns()}/deliveries", params={"webhook_id": _webhook_id(name)})
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    items = resp.json()["items"]
    assert items == [], f"A test event recorded deliveries: {items}"


@step("Sending another test event to webhook <name> right away should return status <code>")
def send_test_event_again(name, code):
    resp = _send_test_event(name)
    assert resp.status_code == int(code), f"Expected {code}, got {resp.status_code}: {resp.text}"
