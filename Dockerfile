
# Frontend build stage. Pinned to the build host: the output is static files,
# so there is no reason to run Node under QEMU for the arm64 image.
FROM --platform=$BUILDPLATFORM node:25-alpine@sha256:bdf2cca6fe3dabd014ea60163eca3f0f7015fbd5c7ee1b0e9ccb4ced6eb02ef4 AS frontend

ARG SEMVER=dev

WORKDIR /build/web

# Copy package files first for better layer caching
COPY web/package.json web/package-lock.json ./

# Install dependencies
RUN npm ci --ignore-scripts

# Copy frontend source
COPY web/ .

# Build static frontend
# adapter-static outputs to ../internal/ui/dist (i.e. /build/internal/ui/dist)
RUN VITE_APP_VERSION=${SEMVER} PUBLIC_API_URL=/ npm run build

# Go build stage. Also pinned to the build host; the go build below
# cross-compiles with GOOS/GOARCH from TARGETOS/TARGETARCH.
FROM --platform=$BUILDPLATFORM golang:1.27.2-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS builder

RUN apk add --no-cache git

WORKDIR /build

# Server is the root module; it resolves pkg/access, pkg/signature and pkg/template via
# `replace` directives in go.mod. GOWORK=off keeps the build off the workspace
# (go.work also wires the CLI module, which the server image does not need).
ENV GOWORK=off

# Copy module manifests first for better layer caching. The root replace targets
# must be present so `go mod download` can read their go.mod files.
COPY go.mod go.sum ./
COPY pkg/access/go.mod ./pkg/access/
COPY pkg/signature/go.mod ./pkg/signature/
COPY pkg/template/go.mod ./pkg/template/
COPY satellites/recipes/go.mod ./satellites/recipes/
COPY satellites/sparrow-sources/go.mod ./satellites/sparrow-sources/
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend into the embedded UI directory
COPY --from=frontend /build/internal/ui/dist/ /build/internal/ui/dist/

# Build server (includes embedded UI via go:embed, runs migrations on startup)
ARG VERSION=dev
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s -X github.com/sarathsp06/sparrow.Version=${VERSION}" \
    -trimpath \
    -o server ./cmd/server

# Final stage - distroless for minimal attack surface
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab

WORKDIR /app
# distroless/static already ships ca-certificates and tzdata — do not re-copy.


COPY --from=builder --chown=65532:65532 /build/server /app/server

EXPOSE 8080

ENTRYPOINT ["/app/server"]
