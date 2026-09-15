"""
Step implementations for API key enforcement (12_auth_enforcement.spec).
"""

import sys
import os

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "libs"))

import requests
from getgauge.python import step, data_store

API_KEY = "e2e-secret-key"


@step("Start authed sparrow server")
def start_authed_server():
    env = data_store.suite["env"]
    url = env.start_authed(API_KEY)
    data_store.suite["authed_url"] = url


def _get(path, key):
    url = data_store.suite["authed_url"]
    headers = {"X-API-Key": key} if key else {}
    return requests.get(f"{url}{path}", headers=headers, timeout=10)


@step("GET <path> on authed server without key should return status <status>")
def get_authed_no_key(path, status):
    resp = _get(path, None)
    assert resp.status_code == int(status), f"Expected {status}, got {resp.status_code}: {resp.text}"


@step("GET <path> on authed server with key <key> should return status <status>")
def get_authed_with_key(path, key, status):
    resp = _get(path, key)
    assert resp.status_code == int(status), f"Expected {status}, got {resp.status_code}: {resp.text}"


@step("GET <path> on authed server with the correct key should return status <status>")
def get_authed_correct_key(path, status):
    resp = _get(path, API_KEY)
    assert resp.status_code == int(status), f"Expected {status}, got {resp.status_code}: {resp.text}"


@step("Stop authed sparrow server")
def stop_authed_server():
    data_store.suite["env"].stop_authed()
