---
title: Embedding the Consumer Portal
description: Give external consumers a self-service webhook portal while Sparrow stays on a private network — full walkthrough of the proxy-the-slice topology with nginx and Express examples, plus the security analysis.
---

Sparrow's consumer portal (`/portal`) lets each consumer manage their own
webhooks, subscriptions, and deliveries with a scoped, expiring **portal
token** instead of the admin API key. This guide shows the recommended way to
serve it to **external users while Sparrow itself stays VPN-only**: your
product's public app acts as a forwarding proxy for exactly the routes the
portal needs — nothing else.

The result: the visitor's browser only ever talks to *your* domain, Sparrow
is never directly reachable from the internet, and the only credential in the
browser is a token that can touch one consumer and nothing more.

## How it works, end to end

```text
 ┌────────────── public internet ──────────────┐  ┌───────── private network / VPN ─────────┐
 │                                             │  │                                          │
 │  consumer's browser          yourapp.com    │  │              sparrow:8080                │
 │        │                        │           │  │                   │                      │
 │  (1)   │── log in ─────────────▶│           │  │                   │                      │
 │        │                        │──(2) POST /v1/consumers/acme/portal-token               │
 │        │                        │      X-API-Key: <admin key>      │                      │
 │        │                        │◀─── { token, path: "/portal#token=spt_v1..." }          │
 │  (3)   │◀─ link or <iframe> ────│           │  │                   │                      │
 │        │                        │           │  │                   │                      │
 │  (4)   │── GET /portal ────────▶│── forward ────────────────────▶ │  (HTML, no admin key)│
 │        │── GET /_app/*.js ─────▶│── forward ────────────────────▶ │  (SPA bundle)        │
 │  (5)   │── GET /v1/consumers/acme/webhooks ─│──▶                   │                      │
 │        │   Authorization: Bearer spt_v1...  │  │   (bearer verified, scoped to "acme")    │
 └─────────────────────────────────────────────┘  └──────────────────────────────────────────┘
```

1. **Your app authenticates the user** with whatever login it already has.
   Sparrow plays no part in identity — it never needs user accounts.
2. **Your backend mints a token** over the private network:

   ```bash
   curl -s -X POST \
     -H "X-API-Key: $SPARROW_API_KEY" \
     "http://sparrow:8080/v1/consumers/acme/portal-token?ttl_seconds=3600"
   # → { "token": "spt_v1....", "expires_at": "...", "path": "/portal#token=spt_v1...." }
   ```

   This is the **only** call that uses the admin key, and it happens
   server-to-server — the key never crosses the network boundary.
3. **Your app hands the link to the browser** — redirect to
   `https://yourapp.com/portal#token=...`, or render it in an `<iframe>`
   (the portal's headers permit framing; the rest of Sparrow forbids it).
   The token rides in the URL *fragment*, which browsers never send in HTTP
   requests — it stays out of your proxy's and Sparrow's access logs.
4. **The browser loads the portal through your proxy.** The HTML and JS
   bundle come from Sparrow via the forwarded routes. Portal HTML is served
   *without* the injected admin key (unlike the admin dashboard).
5. **The portal calls the API through the same proxy**, sending
   `Authorization: Bearer <token>` on every request. Sparrow verifies the
   HMAC signature and expiry statelessly, then authorizes the path: a token
   for `acme` reaches `/v1/consumers/acme/...` (minus event injection and
   token minting) plus three read-only helpers — every other path is 401.

## What "forwarding" actually means

The proxy is not copying files or caching pages — it is an ordinary program
(nginx, Caddy, or your own backend) that sits on a machine with a foot in
**both** networks: the internet can reach it, and it can reach Sparrow over
the VPN/private network. For every incoming request it does four things:

1. **Match the path** against the allowlist below. No match → respond 404
   itself; Sparrow is never contacted.
2. **Open its own connection** to `sparrow:8080` over the private network.
3. **Replay the request** — same method, same path, same headers (including
   the `Authorization: Bearer` token), same body.
4. **Stream Sparrow's response back** to the browser unchanged.

Trace one real request through it:

```text
browser:  GET https://hooks.yourapp.com/v1/consumers/acme/webhooks
          Authorization: Bearer spt_v1...
                │
proxy:    path starts with /v1/consumers/  →  allowed
          GET http://sparrow:8080/v1/consumers/acme/webhooks   (over the VPN)
          Authorization: Bearer spt_v1...
                │
sparrow:  verifies token signature + expiry, checks the path belongs
          to "acme"  →  200 + JSON
                │
proxy:    streams the 200 back to the browser
```

The browser never learns Sparrow's address, never joins the VPN, and never
holds anything but the scoped token. To the browser the portal simply *is*
`hooks.yourapp.com`. And because HTML, JS, and API all arrive from that one
origin, no CORS configuration is needed anywhere.

### Prerequisites

- A machine (or container) that is publicly reachable **and** can open TCP
  connections to Sparrow on the private network — typically your existing
  app server or ingress, since it already sits in both worlds.
- A DNS name for it (`hooks.yourapp.com`) with a TLS certificate — Caddy
  below provisions one automatically.
- Sparrow reachable from that machine as `sparrow:8080` (substitute your
  real host/IP), with `SPARROW_API_KEY` and `SPARROW_ENCRYPTION_KEY` set.

## The route allowlist

Forward exactly these, deny everything else:

| Route | Why the portal needs it |
|---|---|
| `GET /portal` | the portal page itself |
| `GET /_app/*` | SPA JS/CSS bundle |
| `GET /favicon.png` | tab icon (optional) |
| `* /v1/consumers/*` | webhooks, subscriptions, deliveries, retries — bearer-scoped per consumer |
| `GET /v1/event-types`, `GET /v1/event-types/*` | event catalog for the subscribe form |
| `GET /v1/template-functions` | template helper list for the transform editor |
| `POST /v1/subscriptions:testTemplate` | stateless template dry-run |

Not forwarded — and therefore unreachable from the internet: the admin
dashboard (`/`), token minting (`POST .../portal-token`), event injection
(`POST .../events`), the global list/admin routes, `/docs`, `/openapi.*`.

Paths must be preserved **verbatim**: the SPA's asset and API paths are
root-absolute (`/_app`, `/v1`), so the portal cannot be remounted under
`/integrations/webhooks/`. If those roots collide with your app's own routes,
put the proxy on a dedicated subdomain (`hooks.yourapp.com`).

## Example: Caddy (simplest — start here)

[Caddy](https://caddyserver.com) provisions the TLS certificate itself, so
the entire public vhost is one `Caddyfile` block:

```text
hooks.yourapp.com {
	# Admin-only routes stay unreachable even inside the slice.
	@blocked path_regexp ^/v1/consumers/[^/]+/(portal-token|events)$
	respond @blocked 403

	@portal {
		path /portal /_app/* /favicon.png
		path /v1/consumers/* /v1/event-types /v1/event-types/*
		path /v1/template-functions /v1/subscriptions:testTemplate
	}
	reverse_proxy @portal sparrow:8080

	respond 404   # everything else
}
```

Run `caddy run --config Caddyfile` on the dual-homed machine and the portal
is live at `https://hooks.yourapp.com/portal#token=...`.

## Example: nginx

```nginx
# hooks.yourapp.com — public vhost, forwards only the portal slice.
server {
    listen 443 ssl;
    server_name hooks.yourapp.com;
    # ... ssl_certificate, etc.

    # Rate-limit the API slice (Sparrow does not rate-limit itself).
    limit_req_zone $binary_remote_addr zone=portal:10m rate=20r/s;

    location = /portal          { proxy_pass http://sparrow:8080; }
    location /_app/             { proxy_pass http://sparrow:8080; }
    location = /favicon.png     { proxy_pass http://sparrow:8080; }

    location /v1/consumers/ {
        limit_req zone=portal burst=40 nodelay;
        # Belt and braces: these must stay admin-only even if Sparrow's
        # own checks change. The portal UI never calls them.
        location ~ ^/v1/consumers/[^/]+/portal-token$ { return 403; }
        location ~ ^/v1/consumers/[^/]+/events$       { return 403; }
        proxy_pass http://sparrow:8080;
    }
    location = /v1/event-types            { proxy_pass http://sparrow:8080; }
    location /v1/event-types/             { proxy_pass http://sparrow:8080; }
    location = /v1/template-functions     { proxy_pass http://sparrow:8080; }
    location = /v1/subscriptions:testTemplate { proxy_pass http://sparrow:8080; }

    location / { return 404; }
}
```

Never add `proxy_set_header X-API-Key ...` in this vhost — a valid admin key
outranks portal-token scoping, so injecting it would silently make every
portal visitor an admin.

## Example: Express (your app's backend as the proxy)

If you'd rather not run a separate vhost, ~20 lines in your existing Node
backend do the same job:

```js
import { createProxyMiddleware } from "http-proxy-middleware";

const SPARROW = "http://sparrow:8080"; // reachable over the VPN only

const portalSlice = [
  "/portal",
  "/_app",
  "/favicon.png",
  "/v1/consumers",
  "/v1/event-types",
  "/v1/template-functions",
  "/v1/subscriptions:testTemplate",
];

app.use(
  portalSlice,
  (req, res, next) => {
    // Admin-only routes stay unreachable even through the slice.
    if (/^\/v1\/consumers\/[^/]+\/(portal-token|events)$/.test(req.path)) {
      return res.sendStatus(403);
    }
    next();
  },
  createProxyMiddleware({ target: SPARROW, changeOrigin: true }),
);

// Minting stays server-side, behind YOUR auth:
app.post("/api/webhook-portal-link", requireLogin, async (req, res) => {
  const consumer = req.user.tenantId; // however you map users → consumers
  const r = await fetch(
    `${SPARROW}/v1/consumers/${consumer}/portal-token?ttl_seconds=3600`,
    { method: "POST", headers: { "X-API-Key": process.env.SPARROW_API_KEY } },
  );
  const { path } = await r.json();
  res.json({ url: path }); // same-origin: /portal#token=...
});
```

Then link or iframe `url` from your UI. Because everything is same-origin,
no `CORS_ALLOWED_ORIGINS` configuration is needed.

## Security analysis

What each party can and cannot do:

| Actor | Can | Cannot |
|---|---|---|
| Portal visitor with a valid token for `acme` | Manage `acme`'s webhooks/subscriptions, view and retry `acme`'s deliveries | Read any other consumer, inject events, mint tokens, reach the admin UI/API |
| Anyone hitting the public vhost without a token | Load the portal shell (renders "missing access link") | Call any API route — everything under `/v1` returns 401 |
| Someone who steals a portal link | Everything the legitimate holder can, until expiry | Escalate beyond that consumer |
| Your backend | Mint tokens for any consumer (it holds the admin key) | — (it is fully trusted; keep the key server-side) |

Why the pieces hold:

- **Token integrity** — tokens are HMAC-SHA256 signed with
  `SPARROW_ENCRYPTION_KEY` and verified in constant time. Tampering with the
  consumer name or expiry invalidates the signature; forging one requires
  the server's key.
- **Scope is enforced by Sparrow, not the proxy.** The proxy allowlist is
  defense-in-depth; even if you forwarded all of `/v1`, a bearer token still
  only authorizes its own consumer's paths. Conversely, the proxy's `403`s
  on minting/injection protect you even if a future admin key leaks into the
  slice.
- **No admin key in the browser, ever.** Portal HTML omits the config
  injection the admin dashboard relies on, and the mint call is
  server-to-server. The single fatal misconfiguration is a proxy that
  injects `X-API-Key` on forwarded portal routes — never do that.
- **Fragment tokens don't leak into logs.** `#token=...` is never sent in
  HTTP requests; the portal moves it to `sessionStorage` and sends it only
  as an `Authorization` header over TLS.
- **Blast radius of a leaked link is bounded** by consumer + TTL. Mint short
  (`ttl_seconds=3600`-ish) tokens on demand — your users are already logged
  in, so a fresh link per visit costs nothing. Revocation is expiry-only;
  there is no denylist.

Checklist before going live:

- [ ] `SPARROW_API_KEY` set — with an empty key Sparrow's auth middleware is
      disabled and the forwarded slice would be wide open.
- [ ] `SPARROW_ENCRYPTION_KEY` set (it signs the tokens) and stored in a
      secret manager.
- [ ] Proxy forwards **only** the allowlist; default route denies.
- [ ] No `X-API-Key` injection anywhere on the public vhost.
- [ ] TLS terminated at the proxy; rate limiting on `/v1/consumers/`.
- [ ] Short `ttl_seconds`, minted per visit behind your login.

## Alternative: no Sparrow UI at all

If you want full brand control, skip the portal UI and build your own
screens: your backend keeps the token (or admin key) server-side, calls the
consumer-scoped REST API over the VPN, and renders native components. The
[OpenAPI spec](/sparrow/reference/api) and generated clients make this
mechanical — the portal is a convenience, the API is the contract.
