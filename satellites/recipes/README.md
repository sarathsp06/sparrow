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

Setup steps, what each recipe sends, and gotchas are in the
[recipes guide](https://sarathsp06.github.io/sparrow/satellites/recipes/).
Required params must be passed with `--param` when stdin is not a terminal.

| Recipe | Sends to | Required params | Optional params (default) |
|--------|----------|-----------------|---------------------------|
| `slack` | Slack incoming webhook (Block Kit) | `webhook_url` | |
| `discord` | Discord channel webhook (embed) | `webhook_url` | |
| `ntfy` | ntfy topic (JSON publish) | `topic` | `server_url` (`https://ntfy.sh`) |
| `pagerduty` | PagerDuty Events API v2 (trigger) | `routing_key` | `severity` (`error`) |
| `sendgrid` | SendGrid v3 Mail Send | `api_key`, `from_email` (placeholder default must be changed), `default_recipient` | `from_name` (`Sparrow`) |
| `clickhouse` | ClickHouse HTTP interface (JSONEachRow) | `base_url`, `table`, `user`, `password` | |
| `cloudevents` | CloudEvents 1.0, structured mode | `target_url` | `source` (`/sparrow`) |
| `twilio` | Twilio Messages API (SMS) | `account_sid`, `basic_auth`, `from_number`, `to_number` | |

For Sparrow's own health alert emails, apply `sendgrid` under the `_sparrow`
consumer with the `sparrow.webhook.*` events
(`sparrow --consumer _sparrow use sendgrid ...`). Sparrow only accepts
`sparrow.*` subscriptions under `_sparrow`, and `_sparrow` only accepts those.

## Schema (version 1)

```yaml
version: 1
name: slack                    # must match the filename
description: One-line human description
params:                        # values the user supplies at apply time
  - name: webhook_url          # substituted as {{param "webhook_url"}} in url/headers/template
    prompt: "Slack webhook URL" # short label
    help: "An Incoming Webhook URL from a Slack app." # one-line hint: what it is, where to find it
    example: https://hooks.slack.com/services/T000/B000/XXXX # shown as the input placeholder
    docs_url: https://api.slack.com/messaging/webhooks # destination docs for this value
    required: true
    default: ""                # optional fallback used by CLIs/UIs
    enum: []                   # optional: the only accepted values (UIs render a select)
    secret: false              # UIs may mask input for tokens/passwords
    activation_required: false # missing value blocks enabling an auto-provisioned recipe
    must_override_default: false # placeholder defaults must be replaced before activation
consumer:                      # optional: the consumer the recipe is meant for
  name: _sparrow
  note: "Why, and what happens under any other consumer."
webhook:
  requires_transform: true     # receiver only accepts the transformed payload (see below)
  url: '{{param "webhook_url"}}'
  headers: {Content-Type: application/json}
  secret_headers:              # envelope-encrypted at rest, masked in API responses
    Authorization: 'Bearer {{param "api_token"}}'  # for header-based auth tokens/passwords
subscription:
  transform_template: |
    <Go template producing destination-native JSON from .event_name/.payload/etc>
```

The Go struct for this schema lives in [`schema.go`](schema.go)
(`recipes.Recipe`) — dependency-free, importable by the CLI.

## Requiring the transform

`webhook.requires_transform: true` registers the webhook with
`requires_transform`, for destinations that cannot read Sparrow's default
envelope (every shipped recipe). Sparrow then refuses any of its subscriptions
without an enabled transform template, including ones added later, and a
delivery that somehow has none fails with `template_error` instead of sending
the envelope. A recipe that sets it must have a `subscription.transform_template`.

## How params work

`{{param "x"}}` tokens in `webhook.url`, `webhook.headers` values,
`webhook.secret_headers` values, and
`subscription.transform_template` are replaced by the CLI at apply time via
plain string substitution — write the token exactly as shown, with no extra
spaces or pipes. The substituted `transform_template` is then registered
verbatim on the subscription; Sparrow renders it per delivery with this
context:

- `.event_id` — event id.
- `.event_name` — event type name.
- `.timestamp` — delivery time, RFC 3339.
- `.attempt` — delivery attempt number (1-based).
- `.payload` — the event payload (`map[string]any`).

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
   JSON (twilio, being form-encoded, is exempt).
4. Iterate on templates with `sparrow template test` (renders locally, no
   server needed) or against a server with `POST /v1/subscriptions:testTemplate`.

Local tip: Sparrow rejects private-network delivery URLs by default (SSRF
guard). To deliver to a local receiver during development, start the server
with `SPARROW_ALLOW_PRIVATE_NETWORKS=true`.
