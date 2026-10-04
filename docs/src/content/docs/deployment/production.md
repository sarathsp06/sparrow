---
title: Production Deployment
description: Deploy Sparrow on Kubernetes or any container platform with proper secrets, hardening, and network exposure.
---

Sparrow is a single Go binary backed by PostgreSQL. It runs on any container platform -- this page uses Kubernetes as the worked example, but the same configuration applies to ECS, Cloud Run, Fly.io, or a plain Docker host behind a reverse proxy.

> The [Docker Compose (local)](/sparrow/deployment/docker-compose/) page covers a
> zero-config setup for trying Sparrow on your own machine. Everything below
> assumes you are deploying for real use.

## Required configuration

Set these environment variables on the Sparrow container. Keep them in a secret manager or Kubernetes Secret -- never bake them into images or check them into source control.

| Variable | How to generate | Notes |
|----------|-----------------|-------|
| `ENVIRONMENT` | `production` | The server refuses to start without `SPARROW_API_KEY` and blocks cross-origin browser requests by default. |
| `SPARROW_API_KEY` | `openssl rand -hex 32` | Master key for `/v1` and the UI sign-in prompt. Must be at least 32 characters in production (the server refuses to start otherwise). |
| `SPARROW_ENCRYPTION_KEYS` | `main=$(openssl rand -hex 32)` | Keyring for envelope encryption; each key is 32 random bytes as 64 hex chars. See [key rotation](/sparrow/getting-started/configuration/#key-management). |
| `SPARROW_ENCRYPTION_PRIMARY_KEY_ID` | `main` | Which key ID in the keyring is used for new encryption. |
| `DATABASE_URL` | — | PostgreSQL connection string. Use `sslmode=require` or `sslmode=verify-full` when the database is not on localhost. |
| `SPARROW_SERVE_UI` | `true` or `false` | Serve the embedded web dashboard. |

Other variables (`SPARROW_ALLOWED_NETWORKS`, `CORS_ALLOWED_ORIGINS`, `SPARROW_EVENT_RETENTION_DAYS`, `SPARROW_AUTO_DISABLE_AFTER`, `SPARROW_METRICS_ENABLED`, `SPARROW_AI_API_KEY`, `OTEL_EXPORTER_OTLP_ENDPOINT`, etc.) are documented in [Configuration](/sparrow/getting-started/configuration/).

## Kubernetes manifests

### Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: sparrow
type: Opaque
stringData:
  ENVIRONMENT: production
  SPARROW_API_KEY: "<openssl rand -hex 32>"
  SPARROW_ENCRYPTION_KEYS: "main=<openssl rand -hex 32>"
  SPARROW_ENCRYPTION_PRIMARY_KEY_ID: main
  DATABASE_URL: "postgres://sparrow:<password>@db.internal:5432/sparrow?sslmode=require"
  SPARROW_SERVE_UI: "true"
```

Replace the placeholder values with real secrets. In a managed cluster, use an external secrets operator (AWS Secrets Manager, Vault, etc.) instead of inline `stringData`.

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sparrow
spec:
  replicas: 2          # safe: migrations use Postgres advisory locks; River queue is Postgres-backed
  selector:
    matchLabels:
      app: sparrow
  template:
    metadata:
      labels:
        app: sparrow
    spec:
      containers:
        - name: sparrow
          image: ghcr.io/sarathsp06/sparrow:vX.Y.Z   # pin a release tag, not latest
          ports:
            - containerPort: 8080
          envFrom:
            - secretRef:
                name: sparrow
          securityContext:
            runAsNonRoot: true
            runAsUser: 65532        # distroless nonroot
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
            capabilities:
              drop: [ALL]
            seccompProfile:
              type: RuntimeDefault
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 15
          readinessProbe:
            httpGet:
              path: /ready
              port: 8080
            initialDelaySeconds: 3
            periodSeconds: 10
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: "1"
              memory: 512Mi
```

### Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: sparrow
spec:
  type: ClusterIP
  selector:
    app: sparrow
  ports:
    - port: 8080
      targetPort: 8080
```

## Network exposure

Sparrow is designed to run behind a VPN. Expose it through an **internal** Ingress or LoadBalancer on the VPN network, with TLS terminated at the ingress controller. Never use a public-facing LoadBalancer unless you also put an authenticating proxy in front (see [Securing Sparrow](/sparrow/deployment/security/)).

### Example Ingress (internal)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: sparrow
  annotations:
    # Cloud-specific: mark as internal-only
    # AWS: alb.ingress.kubernetes.io/scheme: internal
    # GCP: networking.gke.io/internal: "true"
spec:
  rules:
    - host: sparrow.internal.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: sparrow
                port:
                  number: 8080
  tls:
    - hosts: [sparrow.internal.example.com]
      secretName: sparrow-tls
```

### NetworkPolicy

Restrict inbound traffic to the ingress controller and deny sensitive egress destinations:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: sparrow
spec:
  podSelector:
    matchLabels:
      app: sparrow
  policyTypes: [Ingress, Egress]
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: ingress-nginx   # your ingress controller's namespace
      ports:
        - port: 8080
  egress:
    # Allow DNS
    - to:
        - namespaceSelector: {}
      ports:
        - port: 53
          protocol: UDP
        - port: 53
          protocol: TCP
    # Allow PostgreSQL
    - to:
        - ipBlock:
            cidr: 10.0.0.0/8            # adjust to your DB subnet
      ports:
        - port: 5432
    # Webhook delivery: any destination and port (receivers often listen on
    # 8080, 8443, ...), except the cloud metadata endpoint. Sparrow blocks it
    # too; this is defence in depth. Tighten to your receivers' CIDRs/ports
    # if you know them.
    - to:
        - ipBlock:
            cidr: 0.0.0.0/0
            except:
              - 169.254.169.254/32
      ports:
        - protocol: TCP
```

Adjust the namespace label and database CIDR to match your cluster. Which internal webhook targets are reachable is controlled by `SPARROW_ALLOWED_NETWORKS` (below), so the policy does not need to repeat it.

## SSRF protection

Sparrow's SSRF check runs at connect time in every mode -- it is never skipped. By default, private, loopback, and link-local addresses are blocked as webhook targets.

**Cloud metadata endpoints are always blocked**, even with `SPARROW_ALLOW_PRIVATE_NETWORKS=true`: `169.254.169.254`, `169.254.170.2`, `169.254.170.23`, `100.100.100.200`, `fd00:ec2::254` and `fd00:ec2::23`, including their IPv4-mapped, 6to4 and NAT64 forms. They hand out cloud credentials; list one in `SPARROW_ALLOWED_NETWORKS` only if you truly mean to deliver there.

The default blocklist also covers `0.0.0.0/8`, local-use NAT64 (`64:ff9b:1::/48`), and 6to4 (`2002::/16`) or NAT64 (`64:ff9b::/96`) addresses that embed a restricted IPv4 address -- public 6to4/NAT64 destinations still work.

To deliver webhooks to internal services (the common case on a VPN), use `SPARROW_ALLOWED_NETWORKS`:

```bash
# Allow webhook deliveries to your VPN subnet
SPARROW_ALLOWED_NETWORKS=10.20.0.0/16,fd12::/48
```

Only the listed CIDRs are opened; loopback, cloud metadata, and other private ranges stay blocked. Invalid entries fail startup. When an allowlist is set, `.internal` and `.local` hostnames are no longer blocked by name; their resolved addresses are checked instead.

`SPARROW_ALLOW_PRIVATE_NETWORKS=true` opens **all** private IPs (except cloud metadata) and is meant for local development and testing, not production. A Kubernetes egress NetworkPolicy (above) is still good defence in depth.

## Access and authentication

The server never writes `SPARROW_API_KEY` into the UI. When `SPARROW_API_KEY` is set, the embedded dashboard shows a **Sign in to Sparrow** prompt on the first `401`. Operators can:

- Paste the master key (exchanged for a named browser token, never stored).
- Use an [access token or invite link](/sparrow/deployment/access/) so the master key is never shared.
- Put an [authenticating proxy](/sparrow/deployment/security/) in front for SSO.

Give each person and CI job their own access token instead of sharing `SPARROW_API_KEY`. Use invites for people (`sparrow invite alice`), direct token creation for machines (`sparrow tokens create --name ci-deploy`).

## Production checklist

- [ ] `ENVIRONMENT=production` and `SPARROW_API_KEY` set (at least 32 characters; `openssl rand -hex 32`).
- [ ] Encryption keyring (`SPARROW_ENCRYPTION_KEYS` + `SPARROW_ENCRYPTION_PRIMARY_KEY_ID`) generated with `openssl rand -hex 32` and stored in a secret manager. Back it up -- lose it and encrypted webhook secrets are unrecoverable.
- [ ] `DATABASE_URL` points to a managed Postgres with `sslmode=require` or stronger.
- [ ] Image tag pinned to a release (not `latest`).
- [ ] Container runs as non-root (UID 65532), read-only filesystem, no privilege escalation, all capabilities dropped.
- [ ] Liveness (`/health`) and readiness (`/ready`) probes configured.
- [ ] Prometheus scraping `/metrics` (or OTLP export configured), with alerts on delivery success rate, `sparrow_queue_jobs` backlog, and `sparrow_webhooks_auto_disabled_total`. `/metrics` needs no API key: keep it off public networks, or set `SPARROW_METRICS_ENABLED=false`.
- [ ] `SPARROW_AUTO_DISABLE_AFTER` reviewed (default 5 days of failures pauses a webhook; deliveries are held, not dropped) and an alert config opted into `sparrow.webhook.disabled`.
- [ ] Exposed only through an internal ingress or VPN -- never a public LoadBalancer.
- [ ] TLS terminated at the ingress controller or reverse proxy.
- [ ] `CORS_ALLOWED_ORIGINS` set if the UI is hosted on a different origin.
- [ ] Use `SPARROW_ALLOWED_NETWORKS` (not `SPARROW_ALLOW_PRIVATE_NETWORKS`) for internal webhook targets. Cloud metadata endpoints are always blocked.
- [ ] `SPARROW_EVENT_RETENTION_DAYS` set if payloads carry regulated data.
- [ ] Each person and CI job uses their own [access token](/sparrow/deployment/access/), not the master key. Tokens expire after `SPARROW_TOKEN_DEFAULT_TTL` (90 days) unless created with `--ttl never`.
- [ ] Revoke tokens when someone leaves or a machine credential is retired.

## Upgrade notes

- **`SPARROW_API_KEY` minimum length.** With `ENVIRONMENT=production`, the server now requires `SPARROW_API_KEY` to be at least 32 characters. If your existing key is shorter, generate a new one with `openssl rand -hex 32` before upgrading. Outside production mode, a short key logs a warning but still works.
- **Cloud metadata always blocked.** Cloud metadata endpoints (169.254.169.254, 169.254.170.2, etc.) are now blocked even when `SPARROW_ALLOW_PRIVATE_NETWORKS=true`. If you need to reach one deliberately, list it in `SPARROW_ALLOWED_NETWORKS`.
- **Tenant-wide tokens expire by default.** New tenant-wide tokens, browser sign-ins and invite-created tokens now expire after `SPARROW_TOKEN_DEFAULT_TTL` (90 days) unless created with an explicit TTL or `never_expires` (`sparrow tokens create --ttl never`). Existing tokens keep their lifetime. Set `SPARROW_TOKEN_DEFAULT_TTL=0` to keep the old never-expiring default.
- **Custom header validation.** Webhook headers are now validated on save. Invalid HTTP token names, CR/LF or control characters in values, values over 8 KiB, and reserved framing headers (`Host`, `Content-Length`, `Transfer-Encoding`, `Connection`, `Keep-Alive`, `Proxy-Connection`, `TE`, `Trailer`, `Upgrade`) are rejected with `400`. Existing webhooks keep delivering: a reserved framing header stored before this check is skipped at delivery (it never had an effect), and the webhook only needs fixing the next time its headers are updated.
