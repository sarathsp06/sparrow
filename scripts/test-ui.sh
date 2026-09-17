#!/usr/bin/env bash
# Boot a real Sparrow stack (ephemeral Postgres + server with the embedded UI)
# and run the Playwright browser tests against it, then tear everything down.
#
#   scripts/test-ui.sh              # build UI, boot stack, run all UI tests
#   SPARROW_BASE_URL=... scripts/test-ui.sh --skip-server   # test an existing server
#
# Requires: docker, go, node/npm.
set -euo pipefail
cd "$(dirname "$0")/.."

PG_NAME=sparrow-ui-pg
PG_PORT=${PG_PORT:-55432}
HTTP_PORT=${SPARROW_HTTP_PORT:-18080}
export SPARROW_BASE_URL=${SPARROW_BASE_URL:-http://localhost:${HTTP_PORT}}
SKIP_SERVER=${SKIP_SERVER:-0}
[ "${1:-}" = "--skip-server" ] && { SKIP_SERVER=1; shift; }

SERVER_PID=""
cleanup() {
  [ -n "$SERVER_PID" ] && kill "$SERVER_PID" 2>/dev/null || true
  [ "$SKIP_SERVER" = "1" ] || docker rm -f "$PG_NAME" >/dev/null 2>&1 || true
}
trap cleanup EXIT

if [ "$SKIP_SERVER" != "1" ]; then
  echo "==> Starting Postgres ($PG_NAME) on :$PG_PORT"
  docker rm -f "$PG_NAME" >/dev/null 2>&1 || true
  docker run -d --name "$PG_NAME" \
    -e POSTGRES_USER=sparrow -e POSTGRES_PASSWORD=sparrow -e POSTGRES_DB=sparrow \
    -p "$PG_PORT:5432" postgres:15-alpine >/dev/null
  echo "==> Waiting for Postgres"
  for _ in $(seq 1 30); do
    docker exec "$PG_NAME" pg_isready -U sparrow -d sparrow >/dev/null 2>&1 && break
    sleep 1
  done

  echo "==> Building embedded UI"
  (cd web && npm run build)

  echo "==> Starting Sparrow server on :$HTTP_PORT (migrations run on boot)"
  DATABASE_URL="postgres://sparrow:sparrow@localhost:$PG_PORT/sparrow?sslmode=disable" \
  SPARROW_HTTP_PORT="$HTTP_PORT" \
  SPARROW_SERVE_UI=true \
  SPARROW_ALLOW_PRIVATE_NETWORKS=true \
  SPARROW_ENCRYPTION_KEY=0000000000000000000000000000000000000000000000000000000000000000 \
    go run ./cmd/server >/tmp/sparrow-ui-server.log 2>&1 &
  SERVER_PID=$!

  echo "==> Waiting for $SPARROW_BASE_URL/health"
  for _ in $(seq 1 60); do
    curl -fsS "$SPARROW_BASE_URL/health" >/dev/null 2>&1 && break
    sleep 1
  done
fi

echo "==> Running Playwright tests against $SPARROW_BASE_URL"
cd web
npx playwright install --with-deps chromium >/dev/null 2>&1 || npx playwright install chromium
npx playwright test "$@"
