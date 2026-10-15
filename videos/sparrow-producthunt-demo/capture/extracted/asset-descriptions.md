# Asset Descriptions

⚠️  No vision credentials — descriptions below are catalog-derived (alt text, headings, section context, filename) instead of Vision-generated. To get richer Vision descriptions on the next capture, set GEMINI_API_KEY (or GOOGLE_API_KEY), or HYPERFRAMES_VERTEX_PROJECT_ID plus HYPERFRAMES_VERTEX_SERVICE_ACCOUNT for Vertex service-account auth, and re-run.

The `logo-<hash>.svg` filename prefix is a structural hint (DOM said this SVG was inside a `<header>`, home-link `<a>`, or had an aria-label matching the page brand). To pick the actual brand logo without Vision, open the `logo-*` candidates in a previewer or rasterize them with `sharp` before referencing — composing a fake logo ships off-brand in the final video.

- svgs/logo-e5c21117.svg — logo e5c21117
- svgs/svg-3671717b.svg — svg 3671717b
- svgs/svg-4e46309c.svg — svg 4e46309c
- svgs/svg-56d09598.svg — svg 56d09598
- svgs/svg-639e2855.svg — svg 639e2855
- svgs/svg-af69762a.svg — svg af69762a
- svgs/svg-fd1fa7e6.svg — svg fd1fa7e6
- fonts/V8mDoQDjQSkFtoMM3T6r8E7mPb54C_k3HqUtEw.woff2 — font file
- fonts/V8mDoQDjQSkFtoMM3T6r8E7mPb94C_k3HqUtEw.woff2 — font file
- fonts/V8mDoQDjQSkFtoMM3T6r8E7mPbF4C_k3HqU.woff2 — font file
- fonts/uU9NCBsR6Z2vfE9aq3bh09SDqFGedCMX.woff2 — font file
- fonts/uU9NCBsR6Z2vfE9aq3bh0NSDqFGedCMX.woff2 — font file
- fonts/uU9NCBsR6Z2vfE9aq3bh0dSDqFGedCMX.woff2 — font file
- fonts/uU9NCBsR6Z2vfE9aq3bh2dSDqFGedCMX.woff2 — font file
- fonts/uU9NCBsR6Z2vfE9aq3bh3dSDqFGedA.woff2 — font file
- fonts/uU9NCBsR6Z2vfE9aq3bh3tSDqFGedCMX.woff2 — font file

## Dashboard page captures (localhost:8080, 1920×1080 @2x, light theme, cream canvas)

- svgs/sparrow-logo.svg — the Sparrow brand mark: a blue (#42ade2) flying sparrow with flapping wings, from the UI favicon
- ui-webhooks.png — Webhooks list page: stat strip (4 total / 4 active / 1 healthy / 0 unhealthy), filter row, table of four endpoints with Health badges (Idle, Healthy) and Active status pills; sidebar nav (Webhooks, Events, Deliveries, Health, Access) and amber "+ Push Event" button
- ui-deliveries.png — Deliveries list page: filter bar, table rows with event name (order.created, mcp.smoke), webhook, consumer, green "Success" badges with 200 responses, one red "Failed" 4xx row at top, attempts 1/4, Retry/Details buttons
- ui-delivery-detail.png — single delivery detail page for an order.created delivery to the sparrow listen webhook: status success, response 200, request/response bodies
- ui-health.png — Health Dashboard: overall health strip (1 healthy / 0 degraded / 0 unhealthy / 3 no data), statistics cards (Total 4, Active 4, Success Rate 94.7%, Successful 18, Failed 1, Pending 0), "No webhooks need attention" check panel
- ui-events.png — Event types list page (order.created, order.shipped, payment.succeeded, user.created …)
- ui-webhook-detail.png — Webhook detail page for the Slack recipe webhook with its order.created subscription and transform
- ui-subscription-edit.png — Subscription edit form (event, label filters, headers, "Edit template" entry)
- ui-template-editor.png — Template editor full-viewport modal: Slack Block Kit Go template on the left, RENDERED · LIVE JSON on the right with "✓ valid JSON"
- ui-template-editor-ai.png — Template editor with the AI drawer open: "PROMPT FOR AI" box containing "format this as a Slack message with the order ID and amount", sample payload {"order_id": "42", "amount": 99}, Copy prompt button, template + live render panes below
- ui-portal.png — Consumer portal view for consumer "default": stat strip (4 endpoints / 18 delivered / 1 failed / 95% success rate), "YOUR ENDPOINTS" list with health badges, "Powered by Sparrow" footer
- ui-schema-infer.png — Update Event Type page for `payment.refunded` (auto-created by `sparrow push`) with the JSON SCHEMA helper open: left pane shows the generated schema tree (type: object, properties: 6 props, required: 6 items); right pane "Recent events" lists the pushed payload ({"amount":99,"currency":"USD","order_id":"42","payment_id":"pay_8f31","reason":"requested_by_customer","refunded_at":…}) with an amber "Generate Schema" button and the note "Generated from 1 sample. Review it before saving."
- ui-register-recipes.png — Register webhook page with "START FROM A RECIPE" open: a 3×3 grid of recipe cards (Clickhouse, Cloudevents, Discord, Ntfy, Pagerduty, Sendgrid, Slack, Twilio), each with a one-line description such as "Send events as email via the SendGrid v3 Mail Send API" and "Send events as SMS via the Twilio Messages API"; below it the endpoint form (Target URL, Consumer, "Requires a transform") and the Payload Transform / Delivery panels
- ui-api-console.png — the UI's API Console drawer open on the right (dark panel over the blurred Webhooks page): header "API CONSOLE 4", subtitle "Every request this UI makes, as copy-ready curl. X-API-Key is redacted.", "API reference ↗" link, and four entries — GET /v1/stats, GET /v1/health-summary, GET /v1/webhooks?limit=25&offset=0, GET /v1/whoami — each with its status 200, latency and a `$ curl 'http://localhost:8080/v1/…'` block
- ui-events-list.png — Events list page (CATALOG / EVENTS): table of event types with NAME, DESCRIPTION, VERSION, STATUS; the `payment.refunded` row reads "created by sparrow push · v1 · Active" and, instead of a "Schema" link, shows an amber "Infer schema" link (no schema yet); other rows: mcp.smoke, order.cancelled, order.created (v3), order.shipped, payment.refund, payment.succeeded, user.created
- ui-deliveries-refund.png — Deliveries list with the newest row at top: `payment.refunded` → "sparrow listen (temporary)" · default · green Success · 200 · 1/4 · 22s ago; below it a run of acme order.created deliveries to recipe webhooks (slack, twilio, cloudevents, discord, sendgrid, ntfy) mostly Success with one amber "Retrying 5xx"
