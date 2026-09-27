#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Use Docker (ruby:3.3) for a modern Ruby with Ed25519/OpenSSL support.
# Falls back to local ruby only if Docker is unavailable and ruby >= 3.0.
if command -v docker >/dev/null 2>&1; then
  exec docker run --rm \
    -v "$REPO_ROOT:/app:ro" \
    -w /app/client/verify/ruby \
    ruby:3.3 \
    ruby test_vectors.rb
elif command -v ruby >/dev/null 2>&1; then
  exec ruby "$SCRIPT_DIR/test_vectors.rb"
else
  echo "ERROR: neither docker nor ruby found" >&2
  exit 1
fi
