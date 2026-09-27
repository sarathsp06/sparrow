# Sparrow Client Libraries

This directory contains **generated** client SDKs derived from the committed OpenAPI contract
(`api/openapi.yaml`), which is itself generated from the Go REST API definitions in `internal/rest`
(via [Huma](https://github.com/danielgtaylor/huma)). Sparrow's interface is REST/OpenAPI only.

> **Note:** All files under `python/` (and any future `go/`, `js/` targets) are auto-generated.
> Do not edit them manually. Regenerate with: `make generate`.
> The exception is `verify/`, which is hand-written and NOT regenerated.

## Python

Generated with [`openapi-python-client`](https://github.com/openapi-generators/openapi-python-client)
(typed, `httpx`/`attrs`-based). Regenerate:

```bash
go run ./cmd/openapi-export api
uvx openapi-python-client generate --path api/openapi.yaml --output-path client/python --overwrite
```

Usage:

```python
from sparrow_client import AuthenticatedClient
from sparrow_client.api.webhooks import register_webhook
from sparrow_client.models.register_webhook_body import RegisterWebhookBody

client = AuthenticatedClient(base_url="http://localhost:8080", token="<api-key>",
                              prefix="", auth_header_name="X-API-Key")
resp = register_webhook.sync_detailed(
    consumer="default",
    client=client,
    body=RegisterWebhookBody(events=["order.created"], url="https://example.com/hook"),
)
```

## Go / TypeScript

Not yet generated in this change — see the OpenSpec change
`rewamp-interface-to-rest-openapi` for the tracked follow-up. `api/openapi.yaml` is the source;
any OpenAPI 3.1-compatible generator (`openapi-generator`, `oapi-codegen`, `openapi-typescript`) works.

## Signature verification helpers (`verify/`, hand-written)

Standalone, dependency-light verifiers for Sparrow's Standard Webhooks delivery signatures
(`v1,` HMAC-SHA256 and `v1a,` Ed25519), meant to be copied or vendored into consumer projects:

- `verify/python/sparrow_verify.py` -- HMAC is stdlib-only; Ed25519 needs `cryptography`.
- `verify/js/sparrow-verify.ts` -- Node >= 16 or Bun, `node:crypto` only.
- `verify/java/SparrowVerify.java` -- Java 15+, JDK only.
- `verify/kotlin/SparrowVerify.kt` -- JDK 15+, JDK only (independent of the Java file).
- `verify/ruby/sparrow_verify.rb` -- stdlib `openssl` only; accepts Rack-style header names.
- `verify/php/SparrowVerify.php` -- PHP 8.1+, no Composer; Ed25519 uses the bundled `sodium`.
- `verify/rust/sparrow_verify.rs` -- deps: `hmac`, `sha2`, `base64`, `ed25519-dalek`.
- `verify/elixir/sparrow_verify.ex` -- `:crypto` only (OTP 25+).
- Go consumers import `github.com/sarathsp06/sparrow/pkg/signature` instead (`verify/go/`
  only holds its vector runner).

Every directory has a `run-vectors.sh` that checks the helper against the shared vectors in
`pkg/signature/testdata/` (generated from the server's signer by
`go test ./internal/webhooks/client -run TestSignatureVectors -update-vectors`).
`make verify-conformance` runs them all; CI runs one job per language.

Framework examples: [Verifying Webhook Signatures](https://sarathsp06.github.io/sparrow/guides/verify-signatures/).
