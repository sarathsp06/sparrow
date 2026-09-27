#!/usr/bin/env bash
# Run the shared signature vectors against pkg/signature (the Go helper).
set -euo pipefail
ROOT=$(cd "$(dirname "$0")/../../.." && pwd)
cd "$ROOT/pkg/signature"
exec go test -run 'TestVectors' -v .
