# inventory.Dockerfile — the InventoryService gRPC service.
# Multi-stage build: the heavy Go toolchain stays in the build stage and only the
# compiled binary lands in the final image.

# Image versions come from core.env through build args, so they are not duplicated in
# every Dockerfile. The defaults cover a manual `docker build`.
ARG GO_IMAGE=golang:1.26-alpine3.23
ARG ALPINE_IMAGE=alpine:3.23

FROM ${GO_IMAGE} AS builder

WORKDIR /app

# Copy the Go workspace files and every go.mod/go.sum.
# A separate layer: Docker caches it until the dependencies change.
COPY go.work go.work.sum ./
COPY platform/go.mod platform/go.sum ./platform/
COPY shared/go.mod shared/go.sum ./shared/
COPY order/go.mod order/go.sum ./order/
COPY inventory/go.mod inventory/go.sum ./inventory/
COPY payment/go.mod payment/go.sum ./payment/
COPY iam/go.mod iam/go.sum ./iam/
COPY assembly/go.mod assembly/go.sum ./assembly/

# The module cache is a cache mount rather than an image layer: it survives rebuilds and
# is shared by every service, so dependencies are downloaded once for all of them.
RUN --mount=type=cache,target=/go/pkg/mod \
    go work sync

# Copy the source code.
COPY . .

# Build the binary. CGO_ENABLED=0 links statically, so no libc or other system
# libraries are needed in the runtime image.
# The Go build cache (/root/.cache/go-build) is a cache mount as well. Without it every
# build recompiles the standard library and all dependencies from scratch, which is the
# main cost of rebuilding after a one-line code change.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o /bin/service ./inventory/cmd/main.go

# --- Runtime ---
# alpine is a minimal image (~5 MB), enough to run a Go binary.
FROM ${ALPINE_IMAGE}

COPY --from=builder /bin/service /bin/service

ENTRYPOINT ["/bin/service"]
