# Sparrow Client Libraries

This directory contains **generated** client SDKs derived from the committed OpenAPI contract
(`api/openapi.yaml`), which is itself generated from the Go REST API definitions in `internal/rest`
(via [Huma](https://github.com/danielgtaylor/huma)). Sparrow's interface is REST/OpenAPI only.

> **Note:** All files under `python/` (and any future `go/`, `js/` targets) are auto-generated.
> Do not edit them manually. Regenerate with: `make generate`.

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
    namespace="default",
    client=client,
    body=RegisterWebhookBody(events=["order.created"], url="https://example.com/hook"),
)
```

## Go / TypeScript

Not yet generated in this change — see the OpenSpec change
`rewamp-interface-to-rest-openapi` for the tracked follow-up. `api/openapi.yaml` is the source;
any OpenAPI 3.1-compatible generator (`openapi-generator`, `oapi-codegen`, `openapi-typescript`) works.
