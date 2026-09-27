#!/usr/bin/env bash
# Boot a real Sparrow stack (ephemeral Postgres + server with the embedded UI)
# and run the Playwright browser tests against it, then tear everything down.
#
# Two deployments are exercised:
#   1. embedded: one server serves UI + API on the same origin (open API).
#   2. split:    the same UI build served by a plain static host on another
#                origin, calling a second server that runs with
#                ENVIRONMENT=production, SPARROW_API_KEY and CORS_ALLOWED_ORIGINS
#                (web/tests/split/).
#
#   scripts/test-ui.sh              # build UI, boot both stacks, run all UI tests
#   SPARROW_BASE_URL=... scripts/test-ui.sh --skip-server   # embedded tests against an existing server
#
# Requires: docker, go, node/npm.
set -euo pipefail
cd "$(dirname "$0")/.."

PG_NAME=sparrow-ui-pg
PG_PORT=${PG_PORT:-55432}
HTTP_PORT=${SPARROW_HTTP_PORT:-18080}
SPLIT_API_PORT=${SPLIT_API_PORT:-18081}
SPLIT_UI_PORT=${SPLIT_UI_PORT:-14173}
NOINJECT_PORT=${NOINJECT_PORT:-18082}
export SPARROW_BASE_URL=${SPARROW_BASE_URL:-http://localhost:${HTTP_PORT}}
SKIP_SERVER=${SKIP_SERVER:-0}
[ "${1:-}" = "--skip-server" ] && { SKIP_SERVER=1; shift; }

SERVER_PID=""
SPLIT_API_PID=""
SPLIT_UI_PID=""
NOINJECT_PID=""
cleanup() {
  for pid in $SERVER_PID $SPLIT_API_PID $SPLIT_UI_PID $NOINJECT_PID; do kill "$pid" 2>/dev/null || true; done
  [ "$SKIP_SERVER" = "1" ] || docker rm -f "$PG_NAME" >/dev/null 2>&1 || true
}
trap cleanup EXIT

SERVER_BIN=$(mktemp -d)/sparrow-server
DB_URL="postgres://sparrow:sparrow@localhost:$PG_PORT/sparrow?sslmode=disable"

# start_server <port> <log> [VAR=value ...] — runs the server in the background.
start_server() {
  local port=$1 log=$2
  shift 2
  env DATABASE_URL="$DB_URL" \
    SPARROW_HTTP_PORT="$port" \
    SPARROW_SERVE_UI=true \
    SPARROW_ALLOW_PRIVATE_NETWORKS=true \
    SPARROW_ENCRYPTION_KEYS=main=0000000000000000000000000000000000000000000000000000000000000000 \
    SPARROW_ENCRYPTION_PRIMARY_KEY_ID=main \
    "$@" \
    "$SERVER_BIN" >"$log" 2>&1 &
  SERVER_STARTED_PID=$!
}

# wait_healthy <base-url> <pid> <log>
wait_healthy() {
  echo "==> Waiting for $1/health"
  for _ in $(seq 1 60); do
    curl -fsS "$1/health" >/dev/null 2>&1 && return 0
    if ! kill -0 "$2" 2>/dev/null; then
      cat "$3"
      exit 1
    fi
    sleep 1
  done
  cat "$3"
  exit 1
}

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
  (cd web && PUBLIC_API_URL=/ npm run build)

  echo "==> Building server"
  go build -o "$SERVER_BIN" ./cmd/server

  echo "==> Starting Sparrow server on :$HTTP_PORT (migrations run on boot)"
  start_server "$HTTP_PORT" /tmp/sparrow-ui-server.log
  SERVER_PID=$SERVER_STARTED_PID
  wait_healthy "$SPARROW_BASE_URL" "$SERVER_PID" /tmp/sparrow-ui-server.log
fi

(cd web && { npx playwright install --with-deps chromium >/dev/null 2>&1 || npx playwright install chromium; })

echo "==> Running Playwright tests (embedded UI) against $SPARROW_BASE_URL"
(cd web && npx playwright test "$@")  # split specs self-skip here

[ "$SKIP_SERVER" = "1" ] && exit 0

SPLIT_UI_URL="http://localhost:$SPLIT_UI_PORT"
SPLIT_API_URL="http://localhost:$SPLIT_API_PORT"
SPLIT_API_KEY="pw-split-$(date +%s)"

echo "==> Starting production-mode Sparrow server on :$SPLIT_API_PORT (API key + CORS for $SPLIT_UI_URL)"
start_server "$SPLIT_API_PORT" /tmp/sparrow-ui-split-server.log \
  ENVIRONMENT=production \
  SPARROW_API_KEY="$SPLIT_API_KEY" \
  CORS_ALLOWED_ORIGINS="$SPLIT_UI_URL"
SPLIT_API_PID=$SERVER_STARTED_PID
wait_healthy "$SPLIT_API_URL" "$SPLIT_API_PID" /tmp/sparrow-ui-split-server.log

echo "==> Starting an embedded-UI server with SPARROW_UI_INJECT_KEY=false on :$NOINJECT_PORT"
NOINJECT_URL="http://localhost:$NOINJECT_PORT"
start_server "$NOINJECT_PORT" /tmp/sparrow-ui-noinject-server.log \
  SPARROW_API_KEY="$SPLIT_API_KEY" \
  SPARROW_UI_INJECT_KEY=false
NOINJECT_PID=$SERVER_STARTED_PID
wait_healthy "$NOINJECT_URL" "$NOINJECT_PID" /tmp/sparrow-ui-noinject-server.log

echo "==> Serving the UI build as a plain static site on :$SPLIT_UI_PORT"
node web/tests/split/static-server.mjs internal/ui/dist "$SPLIT_UI_PORT" >/tmp/sparrow-ui-static.log 2>&1 &
SPLIT_UI_PID=$!
wait_for_static() {
  for _ in $(seq 1 20); do curl -fsS "$SPLIT_UI_URL/config.js" >/dev/null 2>&1 && return 0; sleep 0.5; done
  cat /tmp/sparrow-ui-static.log
  exit 1
}
wait_for_static

echo "==> Running Playwright tests (split deployment)"
(cd web && SPARROW_SPLIT_UI_URL="$SPLIT_UI_URL" SPARROW_SPLIT_API_URL="$SPLIT_API_URL" SPARROW_SPLIT_API_KEY="$SPLIT_API_KEY" \
  SPARROW_NOINJECT_URL="$NOINJECT_URL" \
  npx playwright test tests/split/ "$@")
