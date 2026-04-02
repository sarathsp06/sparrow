# Kubernetes Deployment

Sparrow ships a Helm chart at [`charts/sparrow/`](charts/sparrow/) for production Kubernetes deployments.

**Prerequisites**: Helm 3, a running Kubernetes cluster, and a PostgreSQL database.

---

## Quick Install

### External database (recommended)

```bash
helm install sparrow charts/sparrow/ \
  --set secrets.databaseURL="postgres://user:pass@your-db:5432/sparrow?sslmode=require"
```

### Bundled PostgreSQL (evaluation only)

```bash
helm install sparrow charts/sparrow/ \
  --set postgresql.enabled=true
```

The bundled PostgreSQL runs as a single-replica StatefulSet with a 10Gi PVC. It is **not recommended for production** -- use a managed database (RDS, Cloud SQL, etc.) or your own HA PostgreSQL.

---

## Architecture

```
                    ┌─────────────────┐
                    │    Ingress       │  (optional)
                    │  ingress.enabled │
                    └────────┬────────┘
                             │
                    ┌────────v────────┐
                    │  Service        │
                    │  :8080 (HTTP)   │
                    │  :50051 (gRPC)  │
                    └────────┬────────┘
                             │
                    ┌────────v────────┐
                    │  Deployment     │
                    │  sparrow        │
                    │  (stateless)    │
                    └────────┬────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
     ┌────────v────────┐          ┌─────────v─────────┐
     │  PostgreSQL      │          │  OTel Collector    │
     │  (external or    │          │  (external, opt)   │
     │   bundled)       │          │  via extraEnv      │
     └─────────────────┘          └────────────────────┘
```

Sparrow is **stateless** -- all state lives in PostgreSQL and the River job queue. Safe to scale horizontally.

---

## Chart Values

### Image

| Value | Default | Description |
|-------|---------|-------------|
| `image.repository` | `ghcr.io/sarathsp06/sparrow` | Container image |
| `image.tag` | `latest` | Image tag |
| `image.pullPolicy` | `IfNotPresent` | Pull policy |

### Application

| Value | Default | Description |
|-------|---------|-------------|
| `sparrow.serveUI` | `"true"` | Serve the embedded web dashboard |
| `sparrow.environment` | `production` | `development` or `production` |
| `sparrow.corsAllowedOrigins` | `""` | CORS origins for Connect-RPC |
| `sparrow.extraEnv` | `[]` | Additional env vars (see examples below) |

### Secrets

| Value | Default | Description |
|-------|---------|-------------|
| `secrets.databaseURL` | `""` | PostgreSQL connection string |
| `secrets.encryptionKey` | `""` | 64-char hex key for HMAC webhook secrets |
| `secrets.existingSecret` | `""` | Use a pre-existing Secret instead of chart-managed one |

When `secrets.existingSecret` is set, the chart skips creating its own Secret. Your existing Secret must contain `DATABASE_URL` and `SPARROW_ENCRYPTION_KEY` keys.

### PostgreSQL (bundled, optional)

| Value | Default | Description |
|-------|---------|-------------|
| `postgresql.enabled` | `false` | Deploy a bundled PostgreSQL StatefulSet |
| `postgresql.image` | `postgres:15-alpine` | PostgreSQL image |
| `postgresql.auth.database` | `sparrow` | Database name |
| `postgresql.auth.username` | `sparrow` | Database user |
| `postgresql.auth.password` | `sparrow` | Database password |
| `postgresql.storage.size` | `10Gi` | PVC size |
| `postgresql.storage.storageClass` | `""` | Storage class (empty = cluster default) |

When `postgresql.enabled=true` and `secrets.databaseURL` is empty, the DATABASE_URL is auto-constructed from the auth values.

### Networking

| Value | Default | Description |
|-------|---------|-------------|
| `service.type` | `ClusterIP` | Service type |
| `service.httpPort` | `8080` | HTTP/Connect-RPC port |
| `service.grpcPort` | `50051` | gRPC port |
| `ingress.enabled` | `false` | Create an Ingress |
| `ingress.className` | `""` | Ingress class name |
| `ingress.annotations` | `{}` | Ingress annotations |
| `ingress.hosts` | `[]` | Ingress host rules |
| `ingress.tls` | `[]` | Ingress TLS config |

### Scaling

| Value | Default | Description |
|-------|---------|-------------|
| `replicaCount` | `1` | Number of replicas (when HPA is off) |
| `autoscaling.enabled` | `false` | Enable HPA |
| `autoscaling.minReplicas` | `2` | HPA minimum |
| `autoscaling.maxReplicas` | `10` | HPA maximum |
| `autoscaling.targetCPUUtilization` | `80` | CPU target % |

### Resources

| Value | Default | Description |
|-------|---------|-------------|
| `resources.requests.cpu` | `100m` | CPU request |
| `resources.requests.memory` | `128Mi` | Memory request |
| `resources.limits.cpu` | `500m` | CPU limit |
| `resources.limits.memory` | `512Mi` | Memory limit |

### Service Account

| Value | Default | Description |
|-------|---------|-------------|
| `serviceAccount.create` | `false` | Create a dedicated ServiceAccount |
| `serviceAccount.name` | `""` | SA name (auto-generated if empty) |
| `serviceAccount.annotations` | `{}` | SA annotations (e.g. for IAM roles) |

---

## Examples

### With Ingress and TLS

```bash
helm install sparrow charts/sparrow/ \
  --set secrets.databaseURL="postgres://..." \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set ingress.hosts[0].host=sparrow.example.com \
  --set ingress.hosts[0].paths[0].path=/ \
  --set ingress.hosts[0].paths[0].pathType=Prefix \
  --set ingress.tls[0].secretName=sparrow-tls \
  --set ingress.tls[0].hosts[0]=sparrow.example.com
```

### With OpenTelemetry

```bash
helm install sparrow charts/sparrow/ \
  --set secrets.databaseURL="postgres://..." \
  --set sparrow.extraEnv[0].name=OTEL_EXPORTER_OTLP_ENDPOINT \
  --set sparrow.extraEnv[0].value="http://otel-collector.observability:4318"
```

### With an existing Secret

```bash
# Create the secret first
kubectl create secret generic sparrow-creds \
  --from-literal=DATABASE_URL="postgres://..." \
  --from-literal=SPARROW_ENCRYPTION_KEY="$(openssl rand -hex 32)"

# Reference it in the chart
helm install sparrow charts/sparrow/ \
  --set secrets.existingSecret=sparrow-creds
```

### With HPA

```bash
helm install sparrow charts/sparrow/ \
  --set secrets.databaseURL="postgres://..." \
  --set autoscaling.enabled=true \
  --set autoscaling.minReplicas=2 \
  --set autoscaling.maxReplicas=10
```

### Using a values file

```yaml
# values-prod.yaml
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

---

## Health Probes

The chart configures liveness and readiness probes automatically:

| Probe | Path | Behavior |
|-------|------|----------|
| Liveness | `/health` | Checks DB + queue connectivity. Returns 503 on failure. |
| Readiness | `/ready` | Always returns 200. |

---

## Disruption Budget

A `PodDisruptionBudget` is automatically created when `replicaCount > 1` or `autoscaling.enabled=true`. It sets `maxUnavailable: 1` so at least N-1 pods remain available during voluntary disruptions (node drains, upgrades).

---

## Security

The chart applies a hardened security context by default:

**Sparrow containers:**
- `runAsNonRoot: true` (UID 65532, matching the distroless image)
- `readOnlyRootFilesystem: true`
- `allowPrivilegeEscalation: false`
- All capabilities dropped
- `RuntimeDefault` seccomp profile

**Bundled PostgreSQL** (when `postgresql.enabled=true`):
- Runs as UID/GID 999 (the `postgres` user in the alpine image)
- `RuntimeDefault` seccomp profile

**Init containers** (wait-for-postgresql):
- Resource-limited (10m CPU, 32Mi memory)
- Read-only root filesystem, all capabilities dropped

---

## Makefile Targets

```bash
make helm-lint         # Lint the chart
make helm-template     # Dry-run render (external DB)
make helm-template-pg  # Dry-run render (bundled PG)
make helm-package      # Package chart into build/sparrow-*.tgz
make docker-build      # Build Docker image locally
make docker-push       # Push image to ghcr.io
```

---

## Releases

On every git tag (`v*`), the [release workflow](.github/workflows/release.yml):

1. Builds cross-platform binaries
2. Packages the Helm chart (versioned from the tag)
3. Pushes a multi-arch Docker image to `ghcr.io/sarathsp06/sparrow`
4. Creates a GitHub Release with binaries, chart `.tgz`, and checksums
