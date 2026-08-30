---
type: DevOps Config
title: Helm Chart
description: Kubernetes deployment chart with security hardening — NetworkPolicy, PodSecurityContext, Secrets, HPA, PDB
tags: [helm, kubernetes, deployment, security]
timestamp: 2026-08-29T00:00:00Z
---

# Helm Chart

Located at `charts/sparrow/`. Production-ready deployment for Kubernetes.

## Templates (14 files)

| Template | Purpose |
|----------|---------|
| `deployment.yaml` | Main deployment with PodSecurityContext, container SecurityContext, config checksum annotations |
| `service.yaml` | ClusterIP Service (HTTP port only) |
| `ingress.yaml` | Optional Ingress |
| `hpa.yaml` | Optional HorizontalPodAutoscaler |
| `pdb.yaml` | Optional PodDisruptionBudget |
| `networkpolicy.yaml` | Ingress restrictions, isolate PostgreSQL |
| `secret.yaml` | Database URL, encryption key, API key |
| `configmap.yaml` | Non-secret env vars |
| `serviceaccount.yaml` | automountServiceAccountToken: false |
| `postgresql/statefulset.yaml` | Optional bundled PostgreSQL |
| `postgresql/service.yaml` | PostgreSQL service |
| `postgresql/secret.yaml` | PostgreSQL secrets |

## Security Hardening

| Control | Detail |
|---------|--------|
| Pod SecurityContext | `runAsNonRoot: true`, `runAsUser: 65532`, `seccompProfile: RuntimeDefault` |
| Container SecurityContext | `allowPrivilegeEscalation: false`, `readOnlyRootFilesystem: true`, `capabilities.drop: ALL` |
| automountServiceAccountToken | `false` on both Sparrow and PostgreSQL pods |
| NetworkPolicy | Restricts Sparrow ingress; restricts PG to Sparrow pods only |

## Options

| Feature | Configuration |
|---------|--------------|
| PostgreSQL | `postgresql.enabled` — bundled or external |
| AP Key | `secrets.apiKey` or `existingSecret` |
| Encryption Key | Required via `SPARROW_ENCRYPTION_KEY` |
| Ingress | `ingress.enabled` |
| Autoscaling | `hpa.enabled` |
| PDB | `pdb.enabled` |

## Citations

- `charts/sparrow/`
