
# Frontend build stage
FROM node:22-alpine AS frontend

ARG SEMVER=dev

WORKDIR /build/web

# Copy package files first for better layer caching
COPY web/package.json web/package-lock.json* web/yarn.lock* ./

# Install dependencies
RUN npm ci --ignore-scripts 2>/dev/null || npm install --ignore-scripts

# Copy frontend source
COPY web/ .

# Copy generated proto JS/TS files (frontend imports them via relative paths)
COPY proto/webhook_pb.js proto/webhook_pb.d.ts /build/proto/

# Build static frontend
# adapter-static outputs to ../internal/ui/dist (i.e. /build/internal/ui/dist)
RUN VITE_APP_VERSION=${SEMVER} PUBLIC_API_URL=/ npm run build

# Go build stage
FROM golang:1.26.1-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend into the embedded UI directory
COPY --from=frontend /build/internal/ui/dist/ /build/internal/ui/dist/

# Build server (includes embedded UI via go:embed, runs migrations on startup)
ARG VERSION=dev
ARG TARGETOS=linux
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s -X github.com/sarathsp06/sparrow.Version=${VERSION}" \
    -trimpath \
    -o server ./cmd/server

# Final stage - distroless for minimal attack surface
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

COPY --from=builder --chown=65532:65532 /build/server /app/server

EXPOSE 50051 8080

ENTRYPOINT ["/app/server"]
