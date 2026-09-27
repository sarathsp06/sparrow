#!/usr/bin/env bash
# Run the shared signature vectors against sparrow-verify.ts (Node >= 22.6
# for --experimental-strip-types; otherwise node:24 in Docker).
set -euo pipefail
ROOT=$(cd "$(dirname "$0")/../../.." && pwd)
if command -v node >/dev/null && node -e 'process.exit(+process.versions.node.split(".")[0] >= 23 ? 0 : 1)'; then
  exec node --test --experimental-strip-types "$ROOT/client/verify/js/vectors.test.ts"
fi
exec docker run --rm -v "$ROOT:/repo:ro" -w /repo node:24-slim \
  node --test --experimental-strip-types client/verify/js/vectors.test.ts
