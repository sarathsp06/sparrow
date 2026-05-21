"""
Gauge hooks for suite/scenario setup and teardown.
"""

import sys
import os
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "libs"))

from getgauge.python import before_suite, after_suite, before_scenario, after_scenario, data_store

from sparrow_env import SparrowEnvironment
from sparrow_api import SparrowAPI
from webhook_target import WebhookTargetManager


@before_suite
def setup_environment():
    """Start Postgres + Sparrow containers."""
    env = SparrowEnvironment()
    url = env.start()
    data_store.suite["env"] = env
    data_store.suite["sparrow_url"] = url
    data_store.suite["api"] = SparrowAPI(url)


@after_suite
def teardown_environment():
    """Stop all containers."""
    env = data_store.suite.get("env")
    if env:
        env.stop()


@before_scenario
def setup_scenario():
    """Fresh target manager for each scenario."""
    data_store.scenario["targets"] = WebhookTargetManager()
    ts = str(int(time.time() * 1000) % 100000)
    data_store.scenario["_ns_counter"] = ts


@after_scenario
def teardown_scenario():
    """Stop all mock targets."""
    targets = data_store.scenario.get("targets")
    if targets:
        targets.stop_all()
