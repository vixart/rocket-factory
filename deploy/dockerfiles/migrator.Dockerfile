# migrator.Dockerfile — контейнер goose для накатки SQL-миграций.
# Используется в deploy/compose/{iam,inventory,order}/docker-compose.yml,
# чтобы при `task up-<svc>` миграции применялись до старта приложения.
# Сам контейнер одноразовый — отрабатывает goose up и завершается с exit 0.
#
# Версии передаются через build args из core.env (один источник истины).
ARG GO_IMAGE=golang:1.26-alpine3.23
ARG ALPINE_IMAGE=alpine:3.23
ARG GOOSE_VERSION=v3.27.1

FROM ${GO_IMAGE} AS builder

ARG GOOSE_VERSION
# Cache mount'ы — чтобы goose не качался и не компилировался заново
# при каждой пересборке (кеш общий с остальными сборками).
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install github.com/pressly/goose/v3/cmd/goose@${GOOSE_VERSION}

# --- Runtime ---
FROM ${ALPINE_IMAGE}

COPY --from=builder /go/bin/goose /usr/local/bin/goose

ENTRYPOINT ["goose"]
