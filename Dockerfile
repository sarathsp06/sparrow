# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the migration tool
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o tools/migrate ./cmd/migrate

# Build the gRPC server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o grpc-server ./cmd/grpc-server

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 appgroup && \
    adduser -D -s /bin/sh -u 1001 -G appgroup appuser

# Set working directory
WORKDIR /app

# Create logs directory
RUN mkdir -p /app/logs && chown -R appuser:appgroup /app

# Copy the binaries from builder stage
COPY --from=builder /build/tools/migrate ./tools/migrate
COPY --from=builder /build/grpc-server ./grpc-server

# Copy migrations directory
COPY db/migrations ./db/migrations

# Change ownership
RUN chown appuser:appgroup grpc-server

# Switch to non-root user
USER appuser

# Expose gRPC port
EXPOSE 50051

# Health check for gRPC server
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD pgrep grpc-server || exit 1

# Run the gRPC server by default
CMD ["./grpc-server"]