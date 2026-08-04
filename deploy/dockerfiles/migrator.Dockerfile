# migrator.Dockerfile — a goose container that applies the SQL migrations.
# It is used in deploy/compose/{iam,inventory,order}/docker-compose.yml so that
# `task up-<svc>` applies the migrations before the application starts.
# The container is one-shot: it runs goose up and exits with code 0.
#
# Versions come from core.env through build args (a single source of truth).
ARG GO_IMAGE=golang:1.26-alpine3.23
ARG ALPINE_IMAGE=alpine:3.23
ARG GOOSE_VERSION=v3.27.1

FROM ${GO_IMAGE} AS builder

ARG GOOSE_VERSION
# Cache mounts keep goose from being downloaded and rebuilt on every rebuild
# (the cache is shared with the other builds).
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install github.com/pressly/goose/v3/cmd/goose@${GOOSE_VERSION}

# --- Runtime ---
FROM ${ALPINE_IMAGE}

COPY --from=builder /go/bin/goose /usr/local/bin/goose

ENTRYPOINT ["goose"]
