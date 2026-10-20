"""
Steps for receiver code generation and test events (22_receiver_code.spec).
"""

import requests

from getgauge.python import step, data_store


def _base():
    return data_store.suite["sparrow_url"]


def _ns():
    return data_store.scenario["consumer"]


def _webhook_id(name):
    return data_store.scenario[f"webhook_id_{name}"]


def _subscription_id(name):
    resp = requests.get(f"{_base()}/v1/consumers/{_ns()}/subscriptions", params={"webhook_id": _webhook_id(name)})
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    items = resp.json()["items"]
    assert items, f"webhook {name} has no subscriptions"
    return items[0]["subscription_id"]


@step("Generate <language> <framework> receiver code for webhook <name>")
def generate_receiver_code(language, framework, name):
    resp = requests.post(
        f"{_base()}/v1/consumers/{_ns()}/webhooks/{_webhook_id(name)}:generateReceiverCode",
        json={"language": language, "framework": framework},
    )
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    data_store.scenario["receiver_code"] = resp.json()


@step("Generated receiver code should contain file <path>")
def receiver_code_has_file(path):
    paths = [f["path"] for f in data_store.scenario["receiver_code"]["files"]]
    assert path in paths, f"{path} not in {paths}"


@step("Generated receiver spec should list <count> subscriptions")
def receiver_spec_subscriptions(count):
    subs = data_store.scenario["receiver_code"]["spec"]["subscriptions"]
    assert len(subs) == int(count), f"Expected {count} subscriptions, got {subs}"


@step("Generated receiver code should not contain the webhook secret of <name>")
def receiver_code_has_no_secret(name):
    secret = data_store.scenario[f"webhook_resp_{name}"]["http_config"]["webhook_secret"]
    assert secret, "webhook has no secret to look for"
    for f in data_store.scenario["receiver_code"]["files"]:
        assert secret not in f["content"], f"{f['path']} contains the webhook secret"


@step("Portal generate <language> <framework> receiver code for webhook <name> should return status <code>")
def portal_generate_receiver_code(language, framework, name, code):
    resp = requests.post(
        f"{_base()}/portal/api/webhooks/{_webhook_id(name)}:generateReceiverCode",
        headers={"Authorization": f"Bearer {data_store.scenario['portal_token']}"},
        json={"language": language, "framework": framework},
    )
    assert resp.status_code == int(code), f"Expected {code}, got {resp.status_code}: {resp.text}"


def _send_test_event(name):
    return requests.post(
        f"{_base()}/v1/consumers/{_ns()}/webhooks/{_webhook_id(name)}:sendTestEvent",
        json={"subscription_id": _subscription_id(name)},
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
