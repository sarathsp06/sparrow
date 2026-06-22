---
type: DevOps Config
title: Docker Build
description: 3-stage Dockerfile — Node frontend build, Go compilation, distroless runtime
tags: [docker, build, deployment]
timestamp: 2026-06-22T00:00:00Z
---

# Docker Build

## Dockerfile (3-stage)

| Stage | Base Image | Purpose |
|-------|-----------|---------|
| `frontend` | `node:22-alpine` | Build SvelteKit SPA (adapter-static → `internal/ui/dist`) |
| `builder` | `golang:1.26.1-alpine` | Compile `server` binary with `CGO_ENABLED=0`, embed UI |
| **final** | `gcr.io/distroless/static-debian12:nonroot` | Minimal runtime, USER 65532, ports 50051+8080 |

## docker-compose

Two variants:
- `docker-compose.dev.yml` — local dev with hot-reload, default dev encryption key
- `deploy/docker-compose.yml` — standalone deployment, uses published image `ghcr.io/sarathsp06/sparrow:latest`

## Citations

- `Dockerfile`
- `docker-compose.dev.yml`
- `deploy/docker-compose.yml`
