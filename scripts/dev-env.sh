#!/usr/bin/env sh
# Run a command with local-development defaults for `make run` / `make migrate`.
#
# A default applies only when the variable is set neither in the shell nor in
# ./.env, so an existing .env always wins (the server loads .env itself via
# godotenv, which never overrides variables that are already set). .env is
# read, never executed.
#
#   scripts/dev-env.sh go run ./cmd/server
set -eu

# dotenv NAME — value of NAME in ./.env (last assignment, surrounding quotes stripped).
dotenv() {
  [ -f .env ] || return 0
  sed -n "s/^$1=//p" .env | tail -n 1 | sed -e "s/^[\"']//" -e "s/[\"']\$//"
}

: "${DATABASE_URL:=$(dotenv DATABASE_URL)}"
: "${DATABASE_URL:=postgres://sparrow:sparrow@localhost:5432/sparrow?sslmode=disable}"
export DATABASE_URL

if [ -z "${SPARROW_ENCRYPTION_KEYS:-}" ] && [ -z "$(dotenv SPARROW_ENCRYPTION_KEYS)" ]; then
  echo "dev-env: SPARROW_ENCRYPTION_KEYS not set; using the dev-only all-zeros keyring (never use in production)" >&2
  export SPARROW_ENCRYPTION_KEYS=main=0000000000000000000000000000000000000000000000000000000000000000
  export SPARROW_ENCRYPTION_PRIMARY_KEY_ID=main
fi

# Local development pushes events without registering them first.
: "${SPARROW_AUTO_REGISTER_EVENTS:=$(dotenv SPARROW_AUTO_REGISTER_EVENTS)}"
: "${SPARROW_AUTO_REGISTER_EVENTS:=true}"
export SPARROW_AUTO_REGISTER_EVENTS

exec "$@"
