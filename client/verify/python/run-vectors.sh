#!/usr/bin/env bash
# Run the shared signature vectors against sparrow_verify.py.
# Uses local python3 when it has the `cryptography` package (needed for
# Ed25519), otherwise python:3.12-slim in Docker.
set -euo pipefail
ROOT=$(cd "$(dirname "$0")/../../.." && pwd)
if command -v python3 >/dev/null && python3 -c 'import cryptography' 2>/dev/null; then
  exec python3 "$ROOT/client/verify/python/test_vectors.py" -v
fi
exec docker run --rm -v "$ROOT:/repo:ro" -w /repo python:3.12-slim \
  sh -c 'pip install -q --disable-pip-version-check cryptography >/dev/null && python client/verify/python/test_vectors.py -v'
