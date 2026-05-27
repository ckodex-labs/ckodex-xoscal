# syntax=docker/dockerfile:1
ARG LANCEDB=false

# Build stage
FROM golang:1.25-bookworm AS builder
ARG LANCEDB
WORKDIR /app

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
    && go build -tags lancedb -ldflags="-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o /bin/xoscal-server ./server/cmd/xoscal-server; \
    else \
    CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o /bin/xoscal-server ./server/cmd/xoscal-server; \
    fi

# Runtime stage: base image includes glibc required by CGO-linked LanceDB binaries
FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /data
COPY --from=builder /bin/xoscal-server /xoscal-server
COPY --from=builder /app/k8s/server/configmap.yaml /etc/xoscal/config.yaml
EXPOSE 50051 9090
ENTRYPOINT ["/xoscal-server"]
CMD ["-config", "/etc/xoscal/config.yaml"]
