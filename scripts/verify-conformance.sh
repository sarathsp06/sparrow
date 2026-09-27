#!/usr/bin/env bash
# Run the shared signature vectors (pkg/signature/testdata) against every
# verify helper in client/verify/<lang>. Each helper's run-vectors.sh uses a
# local toolchain when present and falls back to Docker.
#
#   scripts/verify-conformance.sh            # all languages
#   scripts/verify-conformance.sh java ruby  # selected languages
set -uo pipefail
ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

# The vectors must match the production signer before anything else.
go test ./internal/webhooks/client -run TestSignatureVectors >/dev/null || {
  echo "vectors are stale: go test ./internal/webhooks/client -run TestSignatureVectors -update-vectors" >&2
  exit 1
}

if [ $# -gt 0 ]; then langs=("$@"); else
  langs=()
  for d in client/verify/*/run-vectors.sh; do langs+=("$(basename "$(dirname "$d")")"); done
fi

failed=()
for lang in "${langs[@]}"; do
  echo "==> $lang"
  if ! "client/verify/$lang/run-vectors.sh" >"/tmp/sparrow-verify-$lang.log" 2>&1; then
    failed+=("$lang")
    tail -40 "/tmp/sparrow-verify-$lang.log"
  else
    echo "    ok"
  fi
done

if [ ${#failed[@]} -gt 0 ]; then
  echo "FAILED: ${failed[*]}" >&2
  exit 1
fi
echo "all verify helpers pass the shared vectors (${langs[*]})"
