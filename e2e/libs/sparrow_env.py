"""
SparrowEnvironment -- Manages Postgres and Sparrow containers via testcontainers.
"""

import os
import time
import urllib.request
import urllib.error

from testcontainers.postgres import PostgresContainer
from testcontainers.core.container import DockerContainer


SPARROW_IMAGE = os.environ.get("SPARROW_IMAGE", os.environ.get("sparrow_image", "sparrow:e2e"))
ENCRYPTION_KEY = os.environ.get("encryption_key", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")


class SparrowEnvironment:
    """Manages the Sparrow test environment (Postgres + Sparrow containers)."""

    def __init__(self):
        self._postgres = None
        self._sparrow = None
        self._sparrow_url = None

    def start(self) -> str:
        """Start Postgres + Sparrow. Returns the Sparrow HTTP URL."""
        if self._sparrow_url:
            return self._sparrow_url

        self._postgres = PostgresContainer(
            image="postgres:15-alpine",
            username="sparrow",
            password="sparrow",
            dbname="sparrow",
        )
        self._postgres.start()

        pg_port = self._postgres.get_exposed_port(5432)
        db_url = f"postgres://sparrow:sparrow@host.docker.internal:{pg_port}/sparrow?sslmode=disable"

        self._sparrow = DockerContainer(SPARROW_IMAGE)
        self._sparrow.with_env("DATABASE_URL", db_url)
        self._sparrow.with_env("SPARROW_SERVE_UI", "false")
        self._sparrow.with_env("SPARROW_ENCRYPTION_KEY", ENCRYPTION_KEY)
        self._sparrow.with_env("SPARROW_ALLOW_PRIVATE_NETWORKS", "true")
        self._sparrow.with_env("SPARROW_HTTP_PORT", "8080")
        self._sparrow.with_exposed_ports(8080)
        self._sparrow.with_kwargs(extra_hosts={"host.docker.internal": "host-gateway"})
        self._sparrow.start()

        http_port = self._sparrow.get_exposed_port(8080)
        self._sparrow_url = f"http://localhost:{http_port}"
        self._wait_for_health(self._sparrow_url, timeout=30)

        return self._sparrow_url

    @property
    def url(self) -> str:
        if not self._sparrow_url:
            return self.start()
        return self._sparrow_url

    def stop(self):
        if self._sparrow:
            self._sparrow.stop()
            self._sparrow = None
        if self._postgres:
            self._postgres.stop()
            self._postgres = None
        self._sparrow_url = None

    def _wait_for_health(self, url: str, timeout: int = 30):
        deadline = time.time() + timeout
        while time.time() < deadline:
            try:
                resp = urllib.request.urlopen(f"{url}/health", timeout=2)
                if resp.status == 200:
                    return
            except (urllib.error.URLError, OSError):
                pass
            time.sleep(1)
        if self._sparrow:
            logs = self._sparrow.get_logs()
            print(f"Sparrow container logs:\n{logs}")
        raise TimeoutError(f"Sparrow did not become healthy at {url}/health within {timeout}s")
