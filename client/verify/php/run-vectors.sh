#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# PHP is not installed locally; use Docker.
if command -v php >/dev/null 2>&1; then
  exec php "$SCRIPT_DIR/test_vectors.php"
else
  exec docker run --rm \
    -v "$REPO_ROOT:/app:ro" \
    -w /app/client/verify/php \
    php:8.3-cli \
    php test_vectors.php
fi
