---
title: Railway Deployment
description: Deploy Sparrow to Railway with a managed PostgreSQL database.
---

Deploy Sparrow to [Railway](https://railway.com) with a managed PostgreSQL database. Railway handles the build, deploy, and TLS -- no Docker or Kubernetes needed.

## Setup

### 1. Create a new project

Go to [railway.com/new](https://railway.com/new) and create a new project.

### 2. Add PostgreSQL

Click **+ New** and add a **PostgreSQL** database. Railway provisions it instantly and exposes the connection string as `DATABASE_URL`.

### 3. Deploy Sparrow

Click **+ New** > **GitHub Repo** and select your fork of `sarathsp06/sparrow` (or use the Docker image `ghcr.io/sarathsp06/sparrow:latest`).

### 4. Set environment variables

Add these variables to the Sparrow service:

| Variable | Value |
|----------|-------|
| `DATABASE_URL` | `${{Postgres.DATABASE_URL}}` (Railway reference variable) |
| `SPARROW_SERVE_UI` | `true` |
| `SPARROW_API_KEY` | *(optional)* use `${{secret(32)}}` to auto-generate |

Railway automatically injects `DATABASE_URL` when you link the PostgreSQL service using a reference variable.

### 5. Expose the service

Go to **Settings** > **Networking** and enable **Public Networking**. Railway assigns an HTTPS domain like `sparrow-production-xxxx.up.railway.app`.

## How It Works

The repository includes a `railway.toml` that configures the build and deploy:

```toml title="railway.toml"
[build]
builder = "DOCKERFILE"
dockerfilePath = "Dockerfile"

[deploy]
startCommand = "/app/server"
healthcheckPath = "/health"
healthcheckTimeout = 120
restartPolicyType = "ON_FAILURE"
restartPolicyMaxRetries = 5
```

- **Builder** -- Uses the multi-stage Dockerfile (Node frontend build, Go binary build, distroless final image)
- **Health check** -- Railway monitors `/health` and restarts the service if it becomes unhealthy
- **Restart policy** -- Automatically restarts on crash, up to 5 retries

Sparrow runs migrations on startup automatically -- no separate migration step or pre-deploy command needed.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | -- | PostgreSQL connection string (auto-set via Railway reference variable) |
| `SPARROW_SERVE_UI` | No | `false` | Serve the embedded web dashboard |
| `SPARROW_API_KEY` | No | -- | Require this key in `X-API-Key` header for all API requests |
| `SPARROW_ENCRYPTION_KEY` | No | auto-generated | 64-char hex key for envelope encryption of secrets (auto-generated if not set) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | -- | OTLP endpoint for traces, metrics, and logs |

## Using the Docker Image Directly

Instead of building from source, you can deploy the pre-built image:

1. Add a new service, choose **Docker Image**
2. Enter `ghcr.io/sarathsp06/sparrow:latest`
3. Set the environment variables above

This skips the build step entirely and deploys in seconds.

## Ports

Sparrow exposes two ports:

| Port | Protocol | Purpose |
|------|----------|---------|
| `8080` | HTTP | Web UI + Connect-RPC API |
| `50051` | gRPC | gRPC API (requires TCP proxy) |

Railway's public networking exposes HTTP by default. If you need gRPC access, enable a TCP proxy on port `50051` in the service's networking settings.

## Observability

Railway provides built-in logging. For traces and metrics, set `OTEL_EXPORTER_OTLP_ENDPOINT` to your collector:

```
OTEL_EXPORTER_OTLP_ENDPOINT=https://your-otel-collector:4318
```
