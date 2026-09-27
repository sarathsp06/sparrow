"""
Step implementations for access tokens and invites (21_access_tokens.spec).

Runs against the authed server started by steps_auth.py. Token and invite
secrets are kept in the scenario data store under their spec names.
"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "libs"))

import requests
from getgauge.python import data_store, step

# Same default as steps_auth.py (not imported: Gauge loads each step file once).
DEFAULT_MASTER_KEY = "e2e-secret-key"


def _url():
    return data_store.suite["authed_url"]


def _master():
    return {"X-API-Key": data_store.suite.get("authed_key", DEFAULT_MASTER_KEY)}


def _tokens():
    return data_store.scenario.setdefault("access_tokens", {})


def _invites():
    return data_store.scenario.setdefault("access_invites", {})


def _create_token(name, consumer=None):
    body = {"name": name}
    if consumer:
        body["consumer"] = consumer
    resp = requests.post(f"{_url()}/v1/tokens", json=body, headers=_master(), timeout=10)
    assert resp.status_code == 201, f"create token: {resp.status_code} {resp.text}"
    data = resp.json()
    assert data["secret"].startswith("sparrow_tk_"), data
    _tokens()[name] = {"secret": data["secret"], "id": data["token"]["id"]}


@step("Create access token <name> with the master key")
def create_token(name):
    _create_token(name)


@step("Create access token <name> for consumer <consumer> with the master key")
def create_consumer_token(name, consumer):
    _create_token(name, consumer)


def _as_token(method, path, name):
    secret = _tokens()[name]["secret"]
    return requests.request(method, f"{_url()}{path}", headers={"Authorization": f"Bearer {secret}"}, timeout=10)


@step("GET <path> with access token <name> should return status <status>")
def get_with_token(path, name, status):
    resp = _as_token("GET", path, name)
    assert resp.status_code == int(status), f"Expected {status}, got {resp.status_code}: {resp.text}"


@step("GET <path> with access token <name> should be rejected with reason <reason>")
def get_with_token_rejected(path, name, reason):
    resp = _as_token("GET", path, name)
    assert resp.status_code == 401, f"Expected 401, got {resp.status_code}: {resp.text}"
    assert resp.json().get("reason") == reason, resp.text


@step("Whoami with access token <name> should report name <expected>")
def whoami(name, expected):
    resp = _as_token("GET", "/v1/whoami", name)
    assert resp.status_code == 200, resp.text
    body = resp.json()
    assert body["name"] == expected and body["master_key"] is False and body["auth_enabled"] is True, body


@step("Revoke access token <name>")
def revoke(name):
    tid = _tokens()[name]["id"]
    resp = requests.delete(f"{_url()}/v1/tokens/{tid}", headers=_master(), timeout=10)
    assert resp.status_code == 204, f"revoke: {resp.status_code} {resp.text}"


@step("Create invite <name> with the master key")
def create_invite(name):
    resp = requests.post(f"{_url()}/v1/invites", json={"name": name}, headers=_master(), timeout=10)
    assert resp.status_code == 201, f"create invite: {resp.status_code} {resp.text}"
    data = resp.json()
    assert data["path"] == f"/#invite={data['secret']}", data
    _invites()[name] = {"secret": data["secret"], "id": data["invite"]["id"]}


@step("Cancel invite <name>")
def cancel_invite(name):
    iid = _invites()[name]["id"]
    resp = requests.delete(f"{_url()}/v1/invites/{iid}", headers=_master(), timeout=10)
    assert resp.status_code == 204, f"cancel invite: {resp.status_code} {resp.text}"


def _redeem(name):
    # No credential: the invite itself is the credential.
    return requests.post(f"{_url()}/invite/redeem", json={"invite": _invites()[name]["secret"]}, timeout=10)


@step("Redeem invite <name> should succeed as token <token_name>")
def redeem_ok(name, token_name):
    resp = _redeem(name)
    assert resp.status_code == 200, f"redeem: {resp.status_code} {resp.text}"
    assert resp.headers.get("Cache-Control") == "no-store", resp.headers
    data = resp.json()
    assert data["name"] == name and data["token"].startswith("sparrow_tk_"), data
    _tokens()[token_name] = {"secret": data["token"], "id": data["token_id"]}


@step("Redeem invite <name> should fail")
def redeem_fails(name):
    resp = _redeem(name)
    assert resp.status_code == 400, f"Expected 400, got {resp.status_code}: {resp.text}"
    assert "sparrow_tk_" not in resp.text, resp.text


@step("Restart authed sparrow server with master key <key>")
def restart_with_key(key):
    env = data_store.suite["env"]
    env.stop_authed()
    data_store.suite["authed_url"] = env.start_authed(key)
    data_store.suite["authed_key"] = key
