# Kubernetes Deployment

Sparrow ships a Helm chart at [`charts/sparrow/`](charts/sparrow/) for Kubernetes deployments.

**Prerequisites**: Helm 3, kubectl, a running Kubernetes cluster, and a PostgreSQL database.

## Quick Install

```bash
# External database (recommended)
helm install sparrow charts/sparrow/ \
  --set secrets.databaseURL="postgres://user:pass@your-db:5432/sparrow?sslmode=require"

# Bundled PostgreSQL (evaluation only)
helm install sparrow charts/sparrow/ \
  --set postgresql.enabled=true
```

Sparrow runs migrations on startup automatically.

## Chart Values

All values are documented with comments in [`charts/sparrow/values.yaml`](charts/sparrow/values.yaml). Key areas: image config, database secrets, optional bundled PostgreSQL, ingress, autoscaling, resource limits, and `sparrow.extraEnv` for injecting arbitrary environment variables (OTel, feature flags, etc.).

## Full Guide

The complete deployment guide -- including a local end-to-end walkthrough with k3d, production values file examples, security details, and configuration reference -- is on the docs site:

**[Kubernetes Deployment Guide](https://sarathsp06.github.io/sparrow/deployment/kubernetes/)**
