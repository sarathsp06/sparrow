#!/bin/sh
# Behavioural contract for one generated receiver:
#
#   contract.sh TARGET INPUT     e.g. contract.sh go-nethttp testdata/hmac.json
#
# Generates the project, runs its own tests (targets/TARGET/setup.sh and
# test.sh), starts it in test mode (run.sh) with the throwaway credentials,
# replays every vector against it (TestContract) and stops it. WORK_DIR keeps
# the files (default: a temp dir); PORT picks the port (default 8000).
set -eu
here=$(cd "$(dirname "$0")" && pwd)
target=$1
input=$2
work=${WORK_DIR:-$(mktemp -d)}
proj=$work/project
port=${PORT:-8000}
export GOWORK=off
mkdir -p "$work"

cd "$here"
go run ./cmd/gen -in "$input" -target "$target" -o "$proj" -env "$work/receiver.env"
"targets/$target/setup.sh" "$proj"
"targets/$target/test.sh" "$proj"

. "$work/receiver.env"
SPARROW_RECEIVER_TEST_MODE=1 PORT=$port "targets/$target/run.sh" "$proj" >"$work/receiver.log" 2>"$work/receiver.err" &
pid=$!
trap 'kill $pid 2>/dev/null || true' EXIT

if ! go test -count=1 -run TestContract . -args -target "$target" -url "http://127.0.0.1:$port" \
  -log "$work/receiver.log" -spec "$proj/testdata/spec.json"; then
  echo "--- receiver stderr ---" >&2
  cat "$work/receiver.err" >&2
  exit 1
fi
