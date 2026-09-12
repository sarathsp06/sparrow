# Sparrow Recipes

Adapters as config. Each `<name>.yaml` in this directory pairs a webhook
target with a `transform_template` that Sparrow renders server-side per
delivery, turning the generic event envelope into the destination's native
payload — retries, signing, and delivery tracking included, no glue service
required.

Apply one with the CLI:

```sh
sparrow use slack --param webhook_url=https://hooks.slack.com/services/T000/B000/XXX \
  --event user.created --label env=prod
```

## Included recipes

| Recipe | Destination | Params |
|---|---|---|
| `slack` | Slack incoming webhook (Block Kit message) | `webhook_url` |
| `discord` | Discord channel webhook (embed) | `webhook_url` |
| `ntfy` | ntfy topic (plain-text push notification) | `topic_url` |
| `pagerduty` | PagerDuty Events API v2 (trigger alert) | `routing_key` |
| `clickhouse` | ClickHouse HTTP interface (JSONEachRow insert) | `base_url`, `table`, `user`, `password` |

## Schema (version 1)

```yaml
version: 1
name: slack                    # must match the filename
description: One-line human description
params:                        # values the user supplies at apply time
  - name: webhook_url          # substituted as {{param "webhook_url"}} in url/headers/template
    prompt: "Slack incoming webhook URL"
    required: true
webhook:
  url: '{{param "webhook_url"}}'
  headers: {Content-Type: application/json}
subscription:
  transform_template: |
    <Go template producing destination-native JSON from .event_name/.payload/etc>
```

The Go struct for this schema lives in [`schema.go`](schema.go)
(`recipes.Recipe`) — dependency-free, importable by the CLI.

## How params work

`{{param "x"}}` tokens in `webhook.url`, `webhook.headers` values, and
`subscription.transform_template` are replaced by the CLI at apply time via
plain string substitution — write the token exactly as shown, with no extra
spaces or pipes. The substituted `transform_template` is then registered
verbatim on the subscription; Sparrow renders it per delivery with this
context:

| Key | Meaning |
|---|---|
| `.event_id` | Event id |
| `.event_name` | Event type name |
| `.timestamp` | Delivery time, RFC3339 |
| `.attempt` | Delivery attempt number (1-based) |
| `.payload` | The event payload (`map[string]any`) |

Template helpers include `json`, `upper`, `lower`, `printf`, `ellipsis`, and
more — `GET /v1/template-functions` lists them all. Events (`--event`,
repeatable) and label filters (`--label k=v`) are chosen at apply time, never
baked into a recipe.

## Contributing a recipe

1. Create `satellites/recipes/<name>.yaml` per the schema above; `name` must equal the
   filename.
2. Templates that produce JSON must quote every interpolated value through
   the `json` helper (e.g. `{{.event_name | json}}`) so quotes and newlines
   in payloads can't break the output.
3. Run `go test ./satellites/recipes/` — it validates every recipe: schema fields,
   param-token references, and that the rendered template output is valid
   JSON (ntfy, being plain text, is exempt).
4. Iterate on templates against a live server with
   `POST /v1/subscriptions:testTemplate` or `sparrow template test`.

Local tip: Sparrow rejects private-network delivery URLs by default (SSRF
guard). To deliver to a local receiver during development, start the server
with `SPARROW_ALLOW_PRIVATE_NETWORKS=true`.
