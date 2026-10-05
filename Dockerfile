# syntax=docker/dockerfile:1
ARG LANCEDB=false

# Build stage
FROM golang:1.25-bookworm@sha256:e401dae1bf814e29204a8cb7915682e1780951e609ca0dd8865ee1937f510c48 AS builder
ARG LANCEDB
WORKDIR /app

RUN case "$LANCEDB" in true|false) ;; *) echo "LANCEDB must be true or false" >&2; exit 1;; esac

# Install base dependencies and optionally Rust for LanceDB builds
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates git \
    && if [ "$LANCEDB" = "true" ]; then \
    apt-get install -y --no-install-recommends curl \
    && curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y; \
    fi \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build: LanceDB requires CGO + pre-built Rust static library; default is a static binary
RUN if [ "$LANCEDB" = "true" ]; then \
    export PATH="/root/.cargo/bin:$PATH" \
    && git clone --depth 1 --branch v0.1.2 https://github.com/lancedb/lancedb-go.git /tmp/lancedb-go \
    && cd /tmp/lancedb-go/rust \
    && cargo fetch \
    && LANCE_SRC=$(find /root/.cargo/registry/src -path "*/lance-0.37.0/src/lib.rs" | head -1) \
    && if [ -n "$LANCE_SRC" ]; then sed -i '1s/^/#![recursion_limit = "256"]\\n/' "$LANCE_SRC"; fi \
    && cd /tmp/lancedb-go && bash ./scripts/build-native.sh \
    && ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') \
    && export CGO_CFLAGS="-I/tmp/lancedb-go/include" \
    && export CGO_LDFLAGS="/tmp/lancedb-go/lib/linux_${ARCH}/liblancedb_go.a" \
    && cd /app \
    && CGO_ENABLED=1 go build -tags lancedb -ldflags="-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o /bin/xoscal-server ./server/cmd/xoscal-server; \
    else \
    CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o /bin/xoscal-server ./server/cmd/xoscal-server; \
    fi

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/xoscal-backup ./server/cmd/xoscal-backup
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/xoscal-ctl ./server/cmd/xoscal-ctl

# The default release path is static. The optional native path needs glibc.
# LanceDB's Rust toolchain/native library are not release-validated here; this
# runtime selection preserves the development option without a release claim.
FROM gcr.io/distroless/static-debian13:nonroot@sha256:e2e927ec666bae08560abb3c55d0659eceabb657f56b6782ab500a9fc7f555e3 AS runtime-false
FROM gcr.io/distroless/base-debian13:nonroot@sha256:a0d70d6a97cd697d9362bc2aae4a6560dd65817e365d0043b07325a97975dc91 AS runtime-true
FROM runtime-${LANCEDB} AS runtime
WORKDIR /data
COPY --from=builder /bin/xoscal-server /xoscal-server
COPY --from=builder /bin/xoscal-backup /xoscal-backup
COPY --from=builder /bin/xoscal-ctl /xoscal-ctl
# Mount actual application YAML at /etc/xoscal/config.yaml, plus its referenced
# TLS/auth files. k8s/server/configmap.yaml is a Kubernetes wrapper, not that file.
EXPOSE 50051 9090
ENTRYPOINT ["/xoscal-server"]
CMD ["-config", "/etc/xoscal/config.yaml"]
