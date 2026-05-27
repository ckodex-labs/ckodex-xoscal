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

See `k8s/server/configmap.yaml` for a production-ready example.

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
- Using a single-replica Deployment for write-heavy workloads

## Backup

The SQLite database is stored in the PVC mounted at `/data`. Back up:
```bash
kubectl cp <pod>:/data/oscal.db ./oscal-backup.db
```
