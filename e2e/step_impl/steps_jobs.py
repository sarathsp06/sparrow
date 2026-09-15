"""
Step implementations for batch job, stats, and health specs.

These endpoints are not wrapped by SparrowAPI, so we call the REST API
directly with requests.
"""

import sys
import os
import time

import requests

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "libs"))

from getgauge.python import step, data_store


def _base_url():
    return data_store.suite["sparrow_url"]


def _consumer():
    return data_store.scenario["consumer"]


# ---------------------------------------------------------------------------
# Bulk retry by webhook
# ---------------------------------------------------------------------------

@step("Bulk retry deliveries for webhook <name>")
def bulk_retry_webhook(name):
    webhook_id = data_store.scenario[f"webhook_id_{name}"]
    resp = requests.post(
        f"{_base_url()}/v1/consumers/{_consumer()}/deliveries:retry",
        json={"webhook_id": webhook_id},
    )
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    data_store.scenario["bulk_retry_resp"] = resp.json()


@step("Bulk retry should have retried <count> deliveries")
def assert_bulk_retry_count(count):
    actual = data_store.scenario["bulk_retry_resp"]["count"]
    assert actual == int(count), f"Expected count {count}, got {actual}"


@step("All terminal deliveries should have status <status>")
def assert_all_terminal_status(status):
    statuses = [d["status"] for d in data_store.scenario["terminal_deliveries"]]
    assert statuses and all(s == status for s in statuses), \
        f"Expected all deliveries to have status {status}, got {statuses}"


# ---------------------------------------------------------------------------
# Batch jobs (delivery retry / event re-push)
# ---------------------------------------------------------------------------

@step("Prepare retry snapshot of failed deliveries")
def prepare_retry_snapshot():
    resp = requests.get(
        f"{_base_url()}/v1/consumers/{_consumer()}/deliveries",
        params={"status": "failed", "prepare_retry": "true"},
    )
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    snapshot_id = resp.json().get("retry_id")
    assert snapshot_id, f"No retry_id in response: {resp.text}"
    data_store.scenario["snapshot_id"] = snapshot_id


@step("Prepare repush snapshot of pushed events")
def prepare_repush_snapshot():
    resp = requests.get(
        f"{_base_url()}/v1/consumers/{_consumer()}/events",
        params={"prepare_repush": "true"},
    )
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    snapshot_id = resp.json().get("repush_id")
    assert snapshot_id, f"No repush_id in response: {resp.text}"
    data_store.scenario["snapshot_id"] = snapshot_id


def _start_job(action_path, jobs_segment):
    resp = requests.post(
        f"{_base_url()}/v1/consumers/{_consumer()}/{action_path}",
        json={"repush_id": data_store.scenario["snapshot_id"]},
    )
    assert resp.status_code == 202, f"Expected 202, got {resp.status_code}: {resp.text}"
    job = resp.json()
    data_store.scenario["job"] = job
    data_store.scenario["job_path"] = f"/v1/consumers/{_consumer()}/{jobs_segment}/{job['id']}"


@step("Start batch retry job from snapshot")
def start_retry_job():
    _start_job("deliveries:retryBatch", "retry-jobs")


@step("Start batch repush job from snapshot")
def start_repush_job():
    _start_job("events:rePush", "repush-jobs")


@step("Wait for batch job to complete within <timeout> seconds")
def wait_job_complete(timeout):
    deadline = time.time() + float(timeout)
    while True:
        resp = requests.get(f"{_base_url()}{data_store.scenario['job_path']}")
        assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
        job = resp.json()
        data_store.scenario["job"] = job
        if job["status"] in ("completed", "failed", "cancelled"):
            break
        if time.time() >= deadline:
            raise TimeoutError(f"Batch job not terminal within {timeout}s: {job}")
        time.sleep(1)
    assert job["status"] == "completed", f"Expected job status completed, got {job}"


@step("Batch job should show total <total> processed <processed> failed <failed>")
def assert_job_progress(total, processed, failed):
    job = data_store.scenario["job"]
    actual = (job["total"], job["processed"], job["failed"])
    expected = (int(total), int(processed), int(failed))
    assert actual == expected, \
        f"Expected (total, processed, failed)={expected}, got {actual}"


@step("Cancelling the completed batch job should return status <code>")
def cancel_completed_job(code):
    resp = requests.post(f"{_base_url()}{data_store.scenario['job_path']}:cancel", json={})
    assert resp.status_code == int(code), \
        f"Expected {code}, got {resp.status_code}: {resp.text}"


# ---------------------------------------------------------------------------
# Stats
# ---------------------------------------------------------------------------

@step("Consumer stats should show <success> successful and <failed> failed deliveries")
def assert_consumer_stats(success, failed):
    resp = requests.get(f"{_base_url()}/v1/consumers/{_consumer()}/stats")
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    stats = resp.json()
    data_store.scenario["consumer_stats"] = stats
    assert stats["successful_deliveries"] == int(success), \
        f"Expected {success} successful, got {stats['successful_deliveries']}"
    assert stats["failed_deliveries"] == int(failed), \
        f"Expected {failed} failed, got {stats['failed_deliveries']}"


@step("Consumer stats should show <count> total webhooks")
def assert_consumer_stats_webhooks(count):
    stats = data_store.scenario["consumer_stats"]
    assert stats["total_webhooks"] == int(count), \
        f"Expected {count} webhooks, got {stats['total_webhooks']}"


@step("Global stats should include at least the consumer's counts")
def assert_global_stats():
    local = data_store.scenario["consumer_stats"]
    resp = requests.get(f"{_base_url()}/v1/stats")
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    stats = resp.json()
    for field in ("total_webhooks", "total_deliveries", "successful_deliveries", "failed_deliveries"):
        assert stats[field] >= local[field], \
            f"Expected global {field} >= {local[field]}, got {stats[field]}"


# ---------------------------------------------------------------------------
# Health
# ---------------------------------------------------------------------------

@step("Wait for webhook <name> health to become <status> within <timeout> seconds")
def wait_webhook_health(name, status, timeout):
    webhook_id = data_store.scenario[f"webhook_id_{name}"]
    url = f"{_base_url()}/v1/consumers/{_consumer()}/webhooks/{webhook_id}/health"
    deadline = time.time() + float(timeout)
    while True:
        resp = requests.get(url)
        assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
        health = resp.json()
        if health["health"] == status:
            data_store.scenario["webhook_health"] = health
            return
        if time.time() >= deadline:
            raise TimeoutError(f"Webhook health not {status} within {timeout}s: {health}")
        time.sleep(1)


@step("Webhook health should report <count> failed deliveries")
def assert_health_failed_count(count):
    health = data_store.scenario["webhook_health"]
    assert health["failed_deliveries"] == int(count), \
        f"Expected {count} failed deliveries, got {health['failed_deliveries']}"
    assert health["consecutive_failures"] >= int(count), \
        f"Expected consecutive_failures >= {count}, got {health['consecutive_failures']}"


@step("Health summary should count at least <count> <status> webhooks")
def assert_health_summary(count, status):
    resp = requests.get(f"{_base_url()}/v1/health-summary")
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    actual = resp.json()[f"{status}_count"]
    assert actual >= int(count), f"Expected at least {count} {status} webhooks, got {actual}"


@step("Webhook <name> should be listed under health <status>")
def assert_webhook_health_filter(name, status):
    webhook_id = data_store.scenario[f"webhook_id_{name}"]
    # Consumer-scoped listing filtered by computed health.
    resp = requests.get(
        f"{_base_url()}/v1/consumers/{_consumer()}/webhooks",
        params={"health": status},
    )
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    ids = [w["webhook_id"] for w in resp.json()["items"]]
    assert webhook_id in ids, f"Webhook {webhook_id} not in consumer {status} list: {ids}"
    # Cross-consumer listing, narrowed to this webhook's id.
    resp = requests.get(
        f"{_base_url()}/v1/webhooks",
        params={"health": status, "webhook_id": webhook_id},
    )
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    ids = [w["webhook_id"] for w in resp.json()["items"]]
    assert webhook_id in ids, f"Webhook {webhook_id} not in global {status} list: {ids}"
