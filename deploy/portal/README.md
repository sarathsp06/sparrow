# Consumer portal behind a proxy

Sparrow on a private network, with only the consumer portal reachable from
the public side through Caddy. Use it to try the portal the way consumers
see it, and as the starting point for hosting it.

```bash
docker compose up -d
./try.sh          # demo webhooks, deliveries and a portal link for "acme"
```

| Address | Side | What answers |
|---|---|---|
| `http://localhost:8181/portal#token=...` | public (Caddy) | the portal only; `/`, `/v1`, `/docs`, `/health`, `/metrics` are 404 |
| `http://localhost:8180` | private ("VPN") | admin UI and `/v1`, key `local-admin-key` |

- `Caddyfile`: the public allowlist (`/portal`, `/portal/*`, `/_app/*`,
  `/favicon.png`, `POST /invite/redeem`); everything else is 404.
- `SPARROW_PORTAL_RETRY=false docker compose up -d sparrow`: consumers can
  no longer retry deliveries (Retry buttons disabled, retry calls 403).
- `docker build -t sparrow:dev ../.. && SPARROW_IMAGE=sparrow:dev docker compose up -d`:
  run a local build.
- `docker compose down -v`: remove everything.

The keys and `SPARROW_ALLOW_PRIVATE_NETWORKS` here are local-only values.
The full walkthrough, what each Caddyfile line does, the responses to
expect, and what must stay private:
https://sarathsp06.github.io/sparrow/guides/portal-embedding/
