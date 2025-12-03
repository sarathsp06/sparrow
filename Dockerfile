
# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build with optimizations: stripped symbols, no debug info, smaller binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -trimpath \
    -o migrate ./cmd/migrate

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -trimpath \
    -o server ./cmd/server

# Final stage - use distroless for minimal attack surface and size
FROM gcr.io/distroless/static-debian12:nonroot

# Set working directory
WORKDIR /app

# Copy CA certificates and timezone data from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binaries with correct ownership (distroless uses uid/gid 65532)
COPY --from=builder --chown=65532:65532 /build/server /app/server
COPY --from=builder --chown=65532:65532 /build/migrate /app/tools/migrate

# Copy migrations directory
COPY --from=builder --chown=65532:65532 /build/db/migrations /app/db/migrations

# Expose gRPC and HTTP ports
EXPOSE 50051 8080

# distroless doesn't have curl/wget, but we can use a simple TCP check
# For production, consider using grpc-health-probe or adding a health binary
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/server"] || exit 1

# Run the server (distroless doesn't have shell, use exec form)
ENTRYPOINT ["/app/server"]