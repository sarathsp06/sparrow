---
title: Kubernetes Deployment
description: Deploy Sparrow to Kubernetes using the bundled Helm chart.
---

Sparrow ships a Helm chart at [`charts/sparrow/`](https://github.com/sarathsp06/sparrow/tree/main/charts/sparrow) for Kubernetes deployments.

**Prerequisites:** Helm 3, kubectl, and a running Kubernetes cluster.

## What You Get

With default values, `helm install` creates:

- A **Deployment** running 1 Sparrow pod (stateless, safe to scale)
- A **ClusterIP Service** exposing HTTP (`:8080`) and gRPC (`:50051`)
- A **ConfigMap** with application settings
- A **Secret** with `DATABASE_URL` and `SPARROW_ENCRYPTION_KEY`
- Liveness (`/health`) and readiness (`/ready`) probes
- Hardened security context (nonroot, read-only fs, all capabilities dropped)

Optional resources created when enabled: Ingress, HPA, PodDisruptionBudget, ServiceAccount, and a bundled PostgreSQL StatefulSet.

## Install with External Database

The expected production setup. You provide your own PostgreSQL:

```bash
helm install sparrow charts/sparrow/ \
  --set secrets.databaseURL="postgres://user:pass@your-db:5432/sparrow?sslmode=require"
```

Sparrow runs migrations on startup automatically -- no separate migration step needed.

## Install with Bundled PostgreSQL

For evaluation and development. Deploys a single-replica PostgreSQL StatefulSet with a 10Gi PVC:

```bash
helm install sparrow charts/sparrow/ \
  --set postgresql.enabled=true
```

When `postgresql.enabled=true` and `secrets.databaseURL` is empty, the connection string is auto-constructed from `postgresql.auth.*` values.

:::note
The bundled PostgreSQL is not recommended for production. Use a managed database (RDS, Cloud SQL, etc.) or your own HA deployment.
:::

## Local End-to-End with k3d

A complete walkthrough: create a local cluster, install Sparrow with bundled PostgreSQL, and verify the full event-to-delivery flow.

### 1. Create cluster

```bash
# Install k3d if needed: https://k3d.io
k3d cluster create sparrow-dev -p "8080:80@loadbalancer"
```

This maps `localhost:8080` to the k3d Traefik ingress on port 80.

### 2. Build and load the image

```bash
make docker-build
k3d image import ghcr.io/sarathsp06/sparrow:latest -c sparrow-dev
```

### 3. Install the chart

```bash
helm install sparrow charts/sparrow/ \
  --set postgresql.enabled=true \
  --set image.repository=ghcr.io/sarathsp06/sparrow \
  --set image.tag=latest \
  --set image.pullPolicy=Never \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host="" \
  --set ingress.hosts[0].paths[0].path=/ \
  --set ingress.hosts[0].paths[0].pathType=Prefix
```

Setting `ingress.hosts[0].host=""` creates an Ingress that matches all hostnames, so `localhost:8080` routes through Traefik to the Sparrow service.

### 4. Wait for pods

```bash
kubectl get pods -w
```

Wait until the sparrow pod is `Running` and `1/1` ready. The PostgreSQL StatefulSet starts first, then Sparrow's init container waits for it before the main container starts.

### 5. Verify

```bash
# Health check
curl -s http://localhost:8080/health | jq .

# Register an event type
curl -s -X POST http://localhost:8080/webhook.EventService/RegisterEvent \
  -H "Content-Type: application/json" \
  -d '{"name":"order.created","description":"New order","active":true}'

# Register a webhook
curl -s -X POST http://localhost:8080/webhook.WebhookService/RegisterWebhook \
  -H "Content-Type: application/json" \
  -d '{"namespace":"default","url":"https://testhooks.sarathsadasivan.com/hooks","events":["order.created"],"active":true}'

# Push an event
curl -s -X POST http://localhost:8080/webhook.EventService/PushEvent \
  -H "Content-Type: application/json" \
  -d '{"namespace":"default","event":"order.created","payload":{"order_id":"ord_k8s_001","amount":42.00}}'

# Check deliveries
curl -s -X POST http://localhost:8080/webhook.DeliveryService/ListDeliveries \
  -H "Content-Type: application/json" -d '{}' | jq '.deliveries[0].status'
```

### 6. Open the web UI

The embedded dashboard is served at `http://localhost:8080/` by default (`sparrow.serveUI: "true"`).

### 7. Cleanup

```bash
helm uninstall sparrow
k3d cluster delete sparrow-dev
```

## Configuration

All chart values are documented with comments in [`values.yaml`](https://github.com/sarathsp06/sparrow/blob/main/charts/sparrow/values.yaml). Key areas:

**`image.*`** -- Container image, tag, pull policy.

**`secrets.databaseURL`** -- PostgreSQL connection string. Required when `postgresql.enabled=false`. When using an existing Kubernetes Secret, set `secrets.existingSecret` instead -- it must contain `DATABASE_URL` and `SPARROW_ENCRYPTION_KEY` keys.

**`secrets.encryptionKey`** -- 64-char hex key for HMAC webhook secret encryption. Generate with `openssl rand -hex 32`. Optional -- if unset, the secret headers feature is disabled.

**`sparrow.extraEnv`** -- Inject arbitrary environment variables. Use this for OpenTelemetry, feature flags, or any env var Sparrow supports:

```yaml
sparrow:
  extraEnv:
    - name: OTEL_EXPORTER_OTLP_ENDPOINT
      value: "http://otel-collector.observability:4318"
```

**`postgresql.*`** -- Bundled PostgreSQL. Off by default. Enable with `postgresql.enabled: true`.

**`ingress.*`** -- Ingress resource. Off by default. Set `ingress.enabled: true` and configure `hosts` and optionally `tls`.

**`autoscaling.*`** -- HPA. Off by default. Set `autoscaling.enabled: true`. A PodDisruptionBudget is auto-created when multiple replicas are running.

**`resources.*`** -- CPU/memory requests and limits for the Sparrow container. Defaults: 100m/128Mi requests, 500m/512Mi limits.

## Using a Values File

For anything beyond a quick test, use a values file:

```yaml title="values-prod.yaml"
image:
  tag: "1.0.0"

secrets:
  existingSecret: sparrow-prod-creds

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: sparrow.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: sparrow-tls
      hosts:
        - sparrow.example.com

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 20

resources:
  requests:
    cpu: 250m
    memory: 256Mi
  limits:
    cpu: "1"
    memory: 1Gi

sparrow:
  extraEnv:
    - name: OTEL_EXPORTER_OTLP_ENDPOINT
      value: "http://otel-collector.observability:4318"
```

```bash
helm install sparrow charts/sparrow/ -f values-prod.yaml
```

## Security

The chart applies a hardened security context by default:

- Sparrow runs as nonroot (UID 65532, matching the distroless image), read-only root filesystem, all capabilities dropped, `RuntimeDefault` seccomp profile.
- Bundled PostgreSQL runs as UID 999 (the `postgres` user in the alpine image) with `RuntimeDefault` seccomp.
- Init containers are resource-limited and run with the same restrictions.

## Health and Disruption Budget

Probes are configured automatically. Liveness hits `/health` (checks DB + queue, returns 503 on failure). Readiness hits `/ready` (always 200).

A `PodDisruptionBudget` (`maxUnavailable: 1`) is created automatically when `replicaCount > 1` or `autoscaling.enabled=true`.

## Observability

OTel export is **off by default**. Set `OTEL_EXPORTER_OTLP_ENDPOINT` via `sparrow.extraEnv` to enable traces, metrics, and logs export. The chart does not deploy an OTel Collector -- point the endpoint at your existing collector.

## Makefile Targets

```bash
make docker-build      # Build Docker image locally
make docker-push       # Push to ghcr.io
make helm-lint         # Lint the chart
make helm-template     # Dry-run render (external DB)
make helm-template-pg  # Dry-run render (bundled PG)
make helm-package      # Package into build/sparrow-*.tgz
```

## Releases

On every git tag (`v*`), the release workflow builds cross-platform binaries, packages the Helm chart, pushes a multi-arch Docker image to `ghcr.io/sarathsp06/sparrow`, and creates a GitHub Release with all artifacts.
