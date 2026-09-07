# xoscal Operations Runbook

## Deployment

### Kubernetes

```bash
kubectl apply -f k8s/server/rbac.yaml
kubectl apply -f k8s/server/configmap.yaml
kubectl apply -f k8s/server/deployment.yaml
kubectl apply -f k8s/server/service.yaml
kubectl apply -f k8s/server/hpa.yaml
kubectl apply -f k8s/server/pdb.yaml
kubectl apply -f k8s/server/networkpolicy.yaml
```

### Local Development

```bash
go run ./server/cmd/xoscal-server
# or with config
go run ./server/cmd/xoscal-server -config ./config.yaml
```

## Configuration

Configuration is loaded in this precedence order:
1. Command-line flags
2. Environment variables (`XOSCAL_*`)
3. Config file (YAML)
4. Built-in defaults

See `k8s/server/configmap.yaml` for the beta TLS/authentication baseline. The
default `auth_mode=none` is for local development only; network-reachable
deployments must provide TLS and authentication secrets.

### Key Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `XOSCAL_SERVER_ADDR` | `:50051` | gRPC listen address |
| `XOSCAL_STORE_DSN` | `oscal.db` | SQLite data source |
| `XOSCAL_OBSERVABILITY_LOG_LEVEL` | `info` | debug, info, warn, error |
| `XOSCAL_OBSERVABILITY_LOG_FORMAT` | `json` | json or text |
| `XOSCAL_OBSERVABILITY_METRICS_ENABLED` | `true` | Prometheus HTTP endpoint |
| `XOSCAL_OBSERVABILITY_METRICS_ADDR` | `:9090` | Metrics listen address |
| `XOSCAL_SECURITY_RATE_LIMIT_RPS` | `100` | Token bucket fill rate |

For token authentication, set `XOSCAL_SECURITY_AUTH_MODE=token` and provide
`XOSCAL_SECURITY_AUTH_TOKEN_FILE` or `XOSCAL_SECURITY_AUTH_TOKENS`. TLS uses
`XOSCAL_SERVER_TLS_CERT_PATH`, `XOSCAL_SERVER_TLS_KEY_PATH`, and
`XOSCAL_SERVER_REQUIRE_TLS=true`.

### HTTP gateway

For a network deployment, run the gateway with the same token file as the
server and configure the upstream gRPC TLS trust explicitly:

```bash
go run ./server/cmd/xoscal-gateway \
  -grpc xoscal-server:50051 \
  -token-file /etc/xoscal/auth/token \
  -grpc-tls-ca /etc/xoscal/tls/ca.crt \
  -grpc-tls-server-name xoscal-server
```

The gateway rejects a configured upstream token unless TLS is enabled, checks
the bearer token before serving HTTP, and uses the same protected transport
for health checks and generated gRPC-Gateway calls. Leaving `-token-file` and
the TLS flags empty is reserved for local `auth_mode=none` development.

## Health Checks

- **Liveness**: gRPC health probe on `:50051`
- **Readiness**: gRPC health probe on `:50051`
- **Startup**: gRPC health probe on `:50051` (up to 60s)
- **Metrics**: HTTP `/healthz` on `:9090`

## Graceful Shutdown

On SIGINT/SIGTERM the server:
1. Stops accepting new connections
2. Waits for in-flight RPCs to complete (up to `shutdown_timeout`)
3. Forces stop if timeout exceeded

## Monitoring

### Prometheus Metrics

Endpoint: `http://<pod>:9090/metrics`

| Metric | Type | Labels |
|--------|------|--------|
| `xoscal_rpc_total` | Counter | method, code |
| `xoscal_rpc_duration_seconds` | Histogram | method, code |

### Logs

Structured JSON logs are emitted to stderr. Fields include:
- `time`, `level`, `msg`
- `method`, `duration`, `code` (per-RPC)

### Tracing

OpenTelemetry tracing can be enabled via `XOSCAL_OBSERVABILITY_TRACING_ENABLED=true`.

### Verification audit and receipts

For an operator-facing audit history, request:

```text
GET /v1/transparency/claims/<claim-id>/verification-events
```

The endpoint validates the global append-only verification-event hash chain
before returning events. A failed precondition means the history must be
quarantined and investigated; do not treat a partial response as evidence.

To export a review receipt, request:

```text
GET /v1/transparency/claims/<claim-id>/receipt
```

Receipt export is allowed only for a `verified` claim whose current projection
matches the latest audit event and whose source, signature, and policy evidence
still passes content-address verification. The response includes a stable
`receipt_digest` and the audit `verification_event_hash`; raw evidence blobs are
not included. Keep the JSON response with the release or review bundle and
verify the digest before transfer.

For graph projection history, request:

```text
GET /v1/graph/projection-events?claim_id=<claim-id>
GET /v1/graph/projection-events?edge_id=<edge-id>
```

The endpoint validates the global append-only graph projection hash chain before
returning events. A failed precondition means the graph projection history must
be quarantined and investigated; do not treat the edge as promotion evidence
until the chain verifies.

For intake, preflight one or more records first:

```text
POST /v1/transparency/import/preflight
POST /v1/transparency/import   {"all_or_nothing": true, ...}
```

The preflight response reports record indexes, actionable diagnostics, and
`ready`/`already_present` status without writing data. The import endpoint
rejects partial mode in beta and rolls back the entire batch if any record
cannot be committed.

## Troubleshooting

### High Memory Usage

Check HPA status and consider increasing `max_replicas`:
```bash
kubectl get hpa xoscal-server
```

### Rate Limiting

If clients receive `ResourceExhausted`, increase:
- `XOSCAL_SECURITY_RATE_LIMIT_RPS`
- `XOSCAL_SECURITY_RATE_LIMIT_BURST`

### Database Locking (SQLite)

SQLite file locking can occur under high concurrency. Consider:
- WAL mode (enabled automatically by modernc.org/sqlite)
- Reducing `max_concurrent_streams`
- Using the beta single-replica Deployment; SQLite is not a multi-replica
  production topology

The Kubernetes manifest intentionally runs one server replica because the
SQLite PVC is read-write-once. Move to an external transactional database
before enabling multiple server replicas.

## Backup

The SQLite database is stored in the PVC mounted at `/data`. Back up:
```bash
kubectl exec <pod> -- /xoscal-backup \
  -dsn /data/oscal.db \
  -out /data/backups/oscal-backup-YYYYMMDD-HHMMSS.db
kubectl cp <pod>:/data/backups/oscal-backup-YYYYMMDD-HHMMSS.db ./oscal-backup.db
```

The backup command uses SQLite's `VACUUM INTO` and refuses to overwrite an
existing destination. Keep the copied artifact outside the PVC and verify its
size and hash before retention or transfer.

Restore only during a maintenance window:
```bash
kubectl scale deployment/xoscal-server --replicas=0
# Start a maintenance pod that mounts the same PVC, then copy into it.
kubectl cp ./oscal-backup.db <maintenance-pod>:/data/oscal.db
# The runtime image is non-root (UID/GID 65532). kubectl cp commonly writes
# root-owned files, so repair ownership before restarting the server.
kubectl exec <maintenance-pod> -- chown 65532:65532 /data/oscal.db
kubectl exec <maintenance-pod> -- chmod 0600 /data/oscal.db
# Remove the maintenance pod after the copy completes.
kubectl scale deployment/xoscal-server --replicas=1
```

After restore, run `PRAGMA integrity_check`, verify the drill fixture and
schema, wait for the server readiness probe, and run the review-flow smoke
checks before reopening traffic. Keep the original PVC snapshot until the
restored database has been verified. Repeat this procedure against the
deployment's target storage class before production promotion.

## Release bundle retention and export

Every published release has a verifiable evidence bundle. Assemble it with:

```bash
dagger call release-bundle --github-token=env:GITHUB_TOKEN export --path=./release-bundle
```

The bundle contains every release asset, `provenance.intoto.jsonl`,
`image-signature.json`, and `inventory.json` (name, SHA-256, size, and
verification status per asset). Assembly aborts unless checksums, cosign
signatures, SLSA provenance, and the image signature all verify; a partial
bundle is never produced.

Retention procedure:

1. Assemble the bundle for the deployed tag and store it beside the backup
   snapshot for that deployment.
2. Record the bundle inventory digest alongside the backup hash:
   `sha256sum bundle/inventory.json` in the operations log.
3. Keep the bundle for the same retention period as the matching database
   backup; the receipt export on the Live Review surface references the
   release tag, so the bundle must outlive the evidence it attests.
4. On restore or promotion, re-run the bundle verification against the tag
   before trusting the deployed artifact:

```bash
dagger call verify-published-release --github-token=env:GITHUB_TOKEN export --path=./bundle-verify
```

The verification is fail-closed: checksum mismatch, missing signature,
broken provenance, or an unverifiable image signature aborts the bundle
with a non-zero exit and no partial output.
