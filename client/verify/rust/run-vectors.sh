#!/usr/bin/env bash
# Run the Sparrow signature verification test vectors (Rust).
# Works from any cwd. Uses the local cargo if available, otherwise
# the official rust:1 Docker image with a cached target dir.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

if command -v cargo &>/dev/null; then
    echo "==> Using local cargo"
    cd "$SCRIPT_DIR"
    export CARGO_TARGET_DIR="${CARGO_TARGET_DIR:-/tmp/sparrow-rust-verify-target}"
    cargo test -- --nocapture
else
    echo "==> cargo not found; using Docker rust:1"
    CACHE_DIR="/tmp/sparrow-rust-verify-cache"
    mkdir -p "$CACHE_DIR"
    docker run --rm \
        -v "$REPO_ROOT:/repo:ro" \
        -v "$CACHE_DIR:/target-cache" \
        -w /tmp/build \
        rust:1 \
        bash -c '
            cp -r /repo/client/verify/rust/. . &&
            mkdir -p pkg/signature/testdata &&
            cp /repo/pkg/signature/testdata/vectors.json pkg/signature/testdata/ &&
            export CARGO_TARGET_DIR=/target-cache &&
            cargo test -- --nocapture
        '
fi
