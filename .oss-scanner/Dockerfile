# Build image for Anthropic's OSS Scanner (https://github.com/anthropics/oss-scanner).
# The scanner builds this with the repository root as the context, then audits the
# image with no network access: everything the build, tests and reproducers need
# is fetched here. See threat_model.md next to this file.
FROM node:22-bookworm AS node
FROM golang:1.26.1-bookworm

ENV DEBIAN_FRONTEND=noninteractive \
    GOTOOLCHAIN=local \
    GOFLAGS=-buildvcs=false

# PostgreSQL is Sparrow's only dependency (state, queue, cache). Install it so
# reproducers can run the real server and the DATABASE_URL-gated tests offline.
RUN apt-get update && apt-get install -y --no-install-recommends \
      build-essential ca-certificates curl git jq procps sudo \
      postgresql postgresql-contrib \
    && rm -rf /var/lib/apt/lists/*

COPY --from=node /usr/local/bin/node /usr/local/bin/node
COPY --from=node /usr/local/lib/node_modules/npm /usr/local/lib/node_modules/npm
RUN ln -s ../lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm \
    && ln -s ../lib/node_modules/npm/bin/npx-cli.js /usr/local/bin/npx

# scanner contract: the checkout lives inside the image, at /src
COPY . /src
WORKDIR /src

# Frontend: install dependencies and build the SPA that the server embeds
# (adapter-static writes to internal/ui/dist, picked up by go:embed).
RUN cd web && npm ci && PUBLIC_API_URL=/ npm run build

# All Go modules of the workspace (go.work), then the binaries.
RUN go mod download \
    && for m in pkg/access pkg/signature pkg/template satellites/recipes \
                satellites/sparrow satellites/sparrow-sinks satellites/sparrow-sources; do \
         (cd "$m" && go mod download) || exit 1; \
       done \
    && go build -o /usr/local/bin/sparrow-server ./cmd/server \
    && go build -o /usr/local/bin/sparrow-migrate ./cmd/migrate \
    && go build -o /usr/local/bin/sparrow ./satellites/sparrow \
    && go build -o /usr/local/bin/sparrow-sinks ./satellites/sparrow-sinks \
    && go build -o /usr/local/bin/sparrow-sources ./satellites/sparrow-sources

# A local database for tests and reproducers. Start it with:
#   service postgresql start
ENV DATABASE_URL=postgres://sparrow:sparrow@localhost:5432/sparrow?sslmode=disable \
    SPARROW_ENCRYPTION_KEYS=main=0000000000000000000000000000000000000000000000000000000000000000 \
    SPARROW_ENCRYPTION_PRIMARY_KEY_ID=main
RUN service postgresql start \
    && su postgres -c "psql -c \"CREATE ROLE sparrow LOGIN SUPERUSER PASSWORD 'sparrow'\"" \
    && su postgres -c "createdb -O sparrow sparrow" \
    && service postgresql stop

# Baseline tests (the same module list as `make test`). Compiles every test
# package into the build cache; a failing test does not abort the build.
RUN service postgresql start \
    && (make test || echo "WARNING: go tests reported failures") \
    && (cd web && npm run test:unit || echo "WARNING: frontend unit tests reported failures") \
    && service postgresql stop

# Reproducers and patches must build with the network disabled.
ENV GOPROXY=off \
    GOSUMDB=off

CMD ["bash"]
