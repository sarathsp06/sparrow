---
title: Hosting the UI Separately
description: Serve the Sparrow dashboard from its own static host and point it at a Sparrow server on another origin.
---

The Sparrow server can serve the dashboard itself on the same origin as the API when `SPARROW_SERVE_UI=true` (the default is `false`). The local Compose file sets it; with the plain Docker image, set it yourself.

This page covers the other layout: the dashboard is a static site (nginx, a CDN, object storage) on one origin, e.g. `https://sparrow.example.com`, and the Sparrow server runs on another, e.g. `https://sparrow-api.example.com`.

## What changes when the UI is separate

| Concern | Embedded UI (`SPARROW_SERVE_UI=true`) | Separately hosted UI |
|---|---|---|
| Where the UI sends API calls | Same origin | `apiUrl` in `/config.js`, or `PUBLIC_API_URL` at build time |
| API key (`SPARROW_API_KEY`) | Sign-in prompt on first `401` (key never written into the page) | Sign-in prompt on first `401`, or set in `/config.js` |
| CORS | Not needed | `CORS_ALLOWED_ORIGINS` must list the UI origin |
| Security headers (CSP, framing) | Set by Sparrow | Set by your static host |

## 1. Build the UI

```bash
cd web
npm ci
npm run build          # output: ../internal/ui/dist
```

Upload the contents of `internal/ui/dist/` to your static host. The same build works for any server: the API URL is set at deploy time in `config.js` (next step), so you don't need to rebuild for each environment.

If you'd rather bake the URL in at build time, set `PUBLIC_API_URL` for the build:

```bash
PUBLIC_API_URL=https://sparrow-api.example.com npm run build
```

`PUBLIC_API_URL` is read only by `vite build`. Changing it later means rebuilding. An `apiUrl` in `config.js` overrides it.

## 2. Point the UI at the server: `config.js`

The build ships a `config.js` next to `index.html`. It is loaded before the app starts. Edit it on the static host:

```js
window.__SPARROW_CONFIG__ = window.__SPARROW_CONFIG__ || {
  apiUrl: "https://sparrow-api.example.com",
  // apiKey: "",   // optional, see "Authentication" below
};
```

- `apiUrl`: the absolute URL of the Sparrow server. A path prefix is fine (`https://gw.example.com/sparrow`) if a reverse proxy mounts Sparrow there. Trailing slashes are ignored.
- `apiKey`: optional. Leave it out and the UI asks for the key when it needs it.

Serve `config.js` and `index.html` with `Cache-Control: no-cache` so edits take effect straight away.

## 3. Serve it as a single-page app

Every unknown path must return `index.html`, because the UI routes on the client. The UI must be served at the root of its origin (`https://sparrow.example.com/`), not under a sub-path, since assets are referenced as `/_app/...` and `/config.js`.

nginx example:

```nginx
server {
  listen 443 ssl;
  server_name sparrow.example.com;
  root /srv/sparrow-ui;

  location /_app/immutable/ {
    add_header Cache-Control "public, max-age=31536000, immutable";
  }
  location = /config.js {
    add_header Cache-Control "no-cache";
  }
  location / {
    add_header Cache-Control "no-cache";
    try_files $uri /index.html;
  }
}
```

## 4. Configure the server

```bash
# Exact origin of the UI: scheme + host + port, no path.
CORS_ALLOWED_ORIGINS=https://sparrow.example.com
SPARROW_API_KEY=<long random string>
ENVIRONMENT=production
# Optional: turn off the server's own copy of the UI.
SPARROW_SERVE_UI=false
```

- `CORS_ALLOWED_ORIGINS` is required. With `ENVIRONMENT=production` and no allowlist, the server rejects every cross-origin request. Without `ENVIRONMENT=production` and no allowlist, it accepts any origin, which is only meant for local development. List more origins separated by commas. A trailing slash is ignored.
- The UI sends the key in the `X-API-Key` header, never as a cookie, so no credentialed CORS is involved.

## Authentication

When the server has `SPARROW_API_KEY` set, the UI needs a credential. Pick one of these options (1 and 3 combine well):

1. **Prompt (default, recommended).** Leave `apiKey` out of `config.js`. On the first `401` the UI shows a **Sign in to Sparrow** prompt. You can paste the master key or an access token. A pasted master key is exchanged for a browser token behind the scenes, so the master key is never stored in the browser. The sidebar shows "Signed in as &lt;name&gt;" with a **Sign out** button. If the stored credential stops working (revoked, expired, or key rotated), the prompt opens again with a clear message.
2. **`apiKey` in `config.js`.** Nobody has to type anything, but anyone who can load the UI can read the key (see [Security](/sparrow/deployment/security/)). `apiKey` may be the master key or a tenant-wide access token. Don't use it if you expose the consumer portal from the same host, because portal visitors can download `config.js` too.
3. **One-time invite.** Run `sparrow invite alice --ui-url https://sparrow.example.com` (or `POST /v1/invites`) and send the printed link. Opening it redeems the invite and creates a named access token for that browser -- the recipient never sees the master key. Each link works once, expires after its TTL (default 24 hours, max 7 days), and can be cancelled with `sparrow invites cancel`. Consumer invites (`--consumer acme`) open the portal instead of the console. See [Access: Tokens and Invites](/sparrow/deployment/access/) for the full guide.
4. **Authenticating proxy.** Put the UI and API behind a proxy that logs users in and adds `X-API-Key` itself (see [Security → auth proxy](/sparrow/deployment/security/)). Leave `apiKey` empty.

## Consumer portal

The portal (`/portal`) works from a separately hosted UI. Its API calls go to `<apiUrl>/portal/api/...` with the consumer's bearer token, never the admin key. When you mint a link with `POST /v1/tokens` and a `consumer`, the response's `portal_path` (`/portal#token=...`) is relative. Prepend the **UI's** base URL (`https://sparrow.example.com/portal#token=...`), not the API's.

## Local development

`make run` (server on `:8080`) plus `make run-web` (vite on `:5173`) is also a split deployment. It works without configuration because the server allows any origin when `ENVIRONMENT` isn't `production`, and the dev UI defaults to `http://localhost:8080`. If you run the server with `ENVIRONMENT=production`, also set `CORS_ALLOWED_ORIGINS=http://localhost:5173`.

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| Browser console shows `blocked by CORS policy` | UI origin missing from `CORS_ALLOWED_ORIGINS` (check scheme and port), or the server is in production mode with no allowlist |
| API calls go to the UI host and return HTML or 404 | No `apiUrl` in `config.js` and the build had no `PUBLIC_API_URL` |
| **API key required** dialog keeps coming back | The key you entered doesn't match the server's `SPARROW_API_KEY` |
| Reloading a deep link like `/webhooks/abc` returns 404 | Static host has no SPA fallback to `index.html` |
| Blank page, `/_app/...` requests return 404 | UI served under a sub-path; serve it at the origin root |
