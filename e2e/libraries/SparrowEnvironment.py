"""
SparrowEnvironment -- Robot Framework library that manages
Postgres and Sparrow containers via testcontainers-python.

Call `Start Sparrow` in suite setup, `Stop Sparrow` in suite teardown.
"""

import os
import time
import subprocess
import urllib.request
import urllib.error

from testcontainers.postgres import PostgresContainer
from testcontainers.core.container import DockerContainer


# The Docker image is built before tests run (make docker-build)
SPARROW_IMAGE = os.environ.get("SPARROW_IMAGE", "sparrow:e2e")
ENCRYPTION_KEY = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"


class SparrowEnvironment:
    """Robot Framework library that provides the running Sparrow URL."""

    ROBOT_LIBRARY_SCOPE = "GLOBAL"

    def __init__(self):
        self._postgres = None
        self._sparrow = None
        self._sparrow_url = None

    def start_sparrow(self) -> str:
        """Start Postgres + Sparrow containers. Returns the Sparrow HTTP URL."""
        if self._sparrow_url:
            return self._sparrow_url

        # 1. Start Postgres
        self._postgres = PostgresContainer(
            image="postgres:15-alpine",
            username="sparrow",
            password="sparrow",
            dbname="sparrow",
        )
        self._postgres.start()

        pg_port = self._postgres.get_exposed_port(5432)
        db_url = f"postgres://sparrow:sparrow@host.docker.internal:{pg_port}/sparrow?sslmode=disable"

        # 2. Start Sparrow container
        self._sparrow = DockerContainer(SPARROW_IMAGE)
        self._sparrow.with_env("DATABASE_URL", db_url)
        self._sparrow.with_env("SPARROW_SERVE_UI", "false")
        self._sparrow.with_env("SPARROW_ENCRYPTION_KEY", ENCRYPTION_KEY)
        self._sparrow.with_env("SPARROW_ALLOW_PRIVATE_NETWORKS", "true")
        self._sparrow.with_env("SPARROW_HTTP_PORT", "8080")
        self._sparrow.with_env("SPARROW_GRPC_PORT", "50051")
        self._sparrow.with_exposed_ports(8080, 50051)
        self._sparrow.with_kwargs(extra_hosts={"host.docker.internal": "host-gateway"})
        self._sparrow.start()

        # 3. Wait for health
        http_port = self._sparrow.get_exposed_port(8080)
        self._sparrow_url = f"http://localhost:{http_port}"
        self._wait_for_health(self._sparrow_url, timeout=30)

        return self._sparrow_url

    def get_sparrow_url(self) -> str:
        """Return the Sparrow HTTP URL."""
        if not self._sparrow_url:
            return self.start_sparrow()
        return self._sparrow_url

    def stop_sparrow(self):
        """Stop all containers."""
        if self._sparrow:
            self._sparrow.stop()
            self._sparrow = None
        if self._postgres:
            self._postgres.stop()
            self._postgres = None
        self._sparrow_url = None

    def _wait_for_health(self, url: str, timeout: int = 30):
        """Poll /health endpoint until it returns 200."""
        deadline = time.time() + timeout
        while time.time() < deadline:
            try:
                resp = urllib.request.urlopen(f"{url}/health", timeout=2)
                if resp.status == 200:
                    return
            except (urllib.error.URLError, OSError):
                pass
            time.sleep(1)
        # Print container logs for debugging
        if self._sparrow:
            logs = self._sparrow.get_logs()
            print(f"Sparrow container logs:\n{logs}")
        raise TimeoutError(f"Sparrow did not become healthy at {url}/health within {timeout}s")
