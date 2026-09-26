# SPIRE/SPIFFE Workload Authentication Architecture

**Status**: Architectural Target & Roadmap Specification  
**Current Active Authentication**: Bearer Token (`security.auth_mode: token`)

This document specifies the planned SPIRE/SPIFFE workload identity architecture for the xOSCAL gRPC server and Kubernetes deployments.

---

## 1. Overview

SPIRE (the SPIFFE Runtime Environment) provides cryptographically verifiable workload identity without long-lived API secrets. In the target architecture, SPIRE issues X.509 SVIDs or JWT-SVIDs (SPIFFE Verifiable Identity Documents) to authenticate gRPC and HTTP callers directly to `xoscal-server`.

```text
┌─────────────────┐
│   gRPC Client   │
│  (with SPIRE)   │
└────────┬────────┘
         │ JWT-SVID / mTLS X.509 SVID
         ↓
┌─────────────────┐
│  xOSCAL Server  │
│  (validates     │
│   SVID token)   │
└─────────────────┘
```

---

## 2. Current Implementation vs. Roadmap

| Capability | Current Status | Notes |
| ---------- | -------------- | ----- |
| Token Authentication | **Active / Production** | Bearer tokens validated via `UnaryAuth` interceptor (`security.auth_mode: token`). |
| No-Auth (Local Dev) | **Active** | `security.auth_mode: none` for local development and CI testing. |
| SPIRE mTLS / SVID | **Planned** | Interceptor stubbed in `server/internal/interceptors/auth.go` returning `codes.Unimplemented`. |

---

## 3. Server Configuration (Target)

When enabled in a future release, the server configuration in `config.yaml` will declare:

```yaml
security:
  auth_mode: spire
  auth_spire_socket: /run/spire/sockets/agent.sock
  spire_trust_domain: example.org
  spire_audience: xoscal.internal
```

The Go server utilizes the SPIFFE Go SDK (`github.com/spiffe/go-spiffe/v2`):

```go
// Planned interceptor integration:
source, err := workloadapi.NewX509Source(ctx, workloadapi.WithClientOptions(
    workloadapi.WithAddr("unix:///run/spire/sockets/agent.sock"),
))
// Validates caller SPIFFE ID against authorized trust domains.
```

---

## 4. Kubernetes Sidecar Architecture

In a Kubernetes deployment, a SPIRE Agent runs as a DaemonSet on each node, exposing its Unix Domain Socket. Workloads communicate with the socket to receive attested SVIDs:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: xoscal-server
  labels:
    app.kubernetes.io/name: xoscal-server
spec:
  containers:
    - name: xoscal-server
      image: ghcr.io/mchorfa/xoscal:latest
      volumeMounts:
        - name: spire-agent-socket
          mountPath: /run/spire/sockets
          readOnly: true
  volumes:
    - name: spire-agent-socket
      hostPath:
        path: /run/spire/sockets
        type: Directory
```

---

## 5. Verification Commands

Operators running a SPIRE Agent locally can verify token minting using the SPIRE CLI:

```bash
# Fetch a JWT-SVID for the xoscal audience
spire-agent api fetch jwt \
    -audience xoscal.internal \
    -socketPath /run/spire/sockets/agent.sock

# Validate SPIFFE registration entry
spire-server entry show -spiffeID spiffe://example.org/ns/oscal/sa/xoscal-server
```
