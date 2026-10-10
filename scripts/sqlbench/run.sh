#!/usr/bin/env sh
# SQL benchmark: build a large synthetic dataset and time Sparrow's heavy
# queries against it, optionally with the end-to-end pipeline benchmark.
#
#   scripts/sqlbench/run.sh                 # fresh DB, seed at scale 1 (~9 GB, ~20 min), query benchmark
#   scripts/sqlbench/run.sh --scale 0.1     # ~10x smaller and faster
#   scripts/sqlbench/run.sh --skip-seed     # reuse the last seeded DB (the evaluator case then has no backlog)
#   scripts/sqlbench/run.sh --e2e           # also run cmd/benchmark -mode e2e against a server on the seeded DB
#
# Needs a Postgres the script can create databases on (`make dev-db` works),
# psql, and Go. Nothing is committed: the data comes from seed.sql each run.
#
# Environment:
#   SQLBENCH_PG       server URL without a database (default postgres://sparrow:sparrow@localhost:5432)
#   SQLBENCH_DB       database to (re)create (default sparrow_sqlbench); never point it at real data
#   SQLBENCH_OUT_DIR  where results go (default ./sqlbench-results)
#   SQLBENCH_PORT     port for the --e2e server (default 8095)
#   SQLBENCH_RUNS     runs per query case (default 7)
set -eu

scale=1
seed=1
e2e=0
while [ $# -gt 0 ]; do
  case "$1" in
    --scale) scale="$2"; shift 2 ;;
    --skip-seed) seed=0; shift ;;
    --e2e) e2e=1; shift ;;
    -h|--help) sed -n '2,20p' "$0"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

cd "$(dirname "$0")/../.."
pg="${SQLBENCH_PG:-postgres://sparrow:sparrow@localhost:5432}"
db="${SQLBENCH_DB:-sparrow_sqlbench}"
out="${SQLBENCH_OUT_DIR:-sqlbench-results}"
port="${SQLBENCH_PORT:-8095}"
dsn="$pg/$db?sslmode=disable"
mkdir -p "$out"
stamp=$(date +%Y%m%d-%H%M%S)

case "$db" in
  sparrow|postgres) echo "refusing to recreate database '$db'; set SQLBENCH_DB to a scratch name" >&2; exit 2 ;;
esac

if [ "$seed" = 1 ]; then
  echo ">>> Recreating database $db"
  psql "$pg/postgres?sslmode=disable" -q -v ON_ERROR_STOP=1 \
    -c "DROP DATABASE IF EXISTS \"$db\"" -c "CREATE DATABASE \"$db\""
  echo ">>> Migrating"
  DATABASE_URL="$dsn" go run ./cmd/migrate
  echo ">>> Seeding at scale $scale (scale 1 takes ~20 minutes)"
  psql "$dsn" -q -v scale="$scale" -f scripts/sqlbench/seed.sql
  psql "$dsn" -At -c "SELECT 'deliveries: ' || count(*) FROM webhook_deliveries" \
                 -c "SELECT 'database size: ' || pg_size_pretty(pg_database_size(current_database()))"
fi

echo ">>> Query benchmark"
SQLBENCH_DSN="$dsn" SQLBENCH_OUT="$out/queries-$stamp.txt" \
  go test -tags sqlbench -run TestSQLBench -count=1 -v ./internal/webhooks/store/ >"$out/queries-$stamp.log" 2>&1 || {
    cat "$out/queries-$stamp.log"; exit 1; }
cat "$out/queries-$stamp.txt"

if [ "$e2e" = 1 ]; then
  echo ">>> End-to-end benchmark against a server on $db"
  go build -o "$out/server" ./cmd/server
  go build -o "$out/benchmark" ./cmd/benchmark
  DATABASE_URL="$dsn" SPARROW_HTTP_PORT="$port" SPARROW_ALLOW_PRIVATE_NETWORKS=true \
  SPARROW_AUTO_REGISTER_EVENTS=true SPARROW_SERVE_UI=false \
  SPARROW_ENCRYPTION_KEYS=main=0000000000000000000000000000000000000000000000000000000000000000 \
  SPARROW_ENCRYPTION_PRIMARY_KEY_ID=main \
    "$out/server" >"$out/server-$stamp.log" 2>&1 &
  server=$!
  trap 'kill $server 2>/dev/null || true' EXIT
  i=0
  until curl -sf "http://localhost:$port/health" >/dev/null; do
    i=$((i + 1)); [ "$i" -gt 60 ] && { echo "server did not start; see $out/server-$stamp.log" >&2; exit 1; }
    sleep 1
  done
  "$out/benchmark" -mode e2e -sparrow-url "http://localhost:$port" -duration 30s -rps 500 -concurrency 50 \
    -webhooks 300 -subscribers 2 -consumer bench -json "$out/e2e-sustained-$stamp.json"
  "$out/benchmark" -mode e2e -sparrow-url "http://localhost:$port" -burst 20000 -concurrency 50 \
    -webhooks 300 -subscribers 2 -consumer burst -drain-timeout 5m -json "$out/e2e-burst-$stamp.json"
fi

echo ">>> Results in $out/"
