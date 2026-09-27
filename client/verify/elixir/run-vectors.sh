#!/usr/bin/env bash
# Run the Sparrow signature verification test vectors (Elixir).
# Works from any cwd. Uses the local elixir if available (>= 1.18),
# otherwise the official elixir:1.18 Docker image.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

if command -v elixir &>/dev/null; then
    echo "==> Using local elixir"
    cd "$SCRIPT_DIR"
    elixir test_vectors.exs
else
    echo "==> elixir not found; using Docker elixir:1.18"
    docker run --rm \
        -v "$REPO_ROOT:/repo:ro" \
        -w /repo/client/verify/elixir \
        elixir:1.18 \
        elixir test_vectors.exs
fi
