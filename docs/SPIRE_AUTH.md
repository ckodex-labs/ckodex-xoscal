# SPIRE/SPIFFE Authentication

This guide explains how to configure SPIRE/SPIFFE authentication for the OSCAL gRPC server.

## Overview

SPIRE (the SPIFFE Runtime Environment) provides workload identity and issues JWT-SVIDs (JWT SPIFFE Verifiable Identity Documents) that can be used to authenticate gRPC clients to the OSCAL server.

## Architecture

```text
┌─────────────────┐
│   gRPC Client   │
│  (with SPIRE)   │
└────────┬────────┘
         │ JWT-SVID
         ↓
┌─────────────────┐
│  OSCAL gRPC     │
│   Server        │
│  (validates     │
│   JWT-SVID)     │
└─────────────────┘
```

## Prerequisites

- SPIRE Server and SPIRE Agent deployed
- SPIRE OIDC Discovery Provider running
- Workloads registered in SPIRE
- SPIRE Server configured with `jwt_issuer` matching OIDC Discovery Provider URL

## Server Configuration

### 1. Add SPIRE Auth Interceptor

```python
from src.mcp_server_for_oscal.grpc.spire_auth import SpireAuthInterceptor, get_spire_config

# Get SPIRE configuration from environment
config = get_spire_config()

# Create auth interceptor
auth_interceptor = SpireAuthInterceptor(
    trust_domain=config['trust_domain'],
    oidc_discovery_url=config['oidc_discovery_url'],
    audience=config['audience']
)

# Add to gRPC server
server = grpc.aio.server(
    interceptors=[auth_interceptor]
)
```

### 2. Environment Variables

Set the following environment variables:

```bash
export SPIRE_TRUST_DOMAIN="spiffe://prod.example.com"
export SPIRE_OIDC_DISCOVERY_URL="https://oidc-discovery.prod.example.com"
export SPIRE_AUDIENCE="https://api.anthropic.com"
export SPIRE_AGENT_SOCKET="/run/spire/sockets/agent.sock"
```

## Client Configuration

### 1. Using the Client Interceptor

```python
from src.mcp_server_for_oscal.grpc.spire_auth import SpireAuthClientInterceptor

# Create authenticated channel
interceptor = SpireAuthClientInterceptor(
    spire_agent_socket="/run/spire/sockets/agent.sock",
    audience="https://api.anthropic.com"
)

channel = grpc.aio.insecure_channel('localhost:50051')
# Apply interceptor (implementation depends on gRPC version)
```

### 2. Using spiffe-helper

Configure spiffe-helper to fetch JWT-SVIDs:

```text helper.conf
agent_address = "/run/spire/sockets/agent.sock"
cert_dir      = "/var/run/secrets/anthropic.com"
daemon_mode   = true

jwt_svids = [{
    jwt_audience       = "https://api.anthropic.com"
    jwt_svid_file_name = "token"
}]
```

The JWT-SVID will be written to `/var/run/secrets/anthropic.com/token`.

## Kubernetes Deployment

### 1. Register Workload in SPIRE

```bash
spire-server entry create \
    -spiffeID spiffe://prod.example.com/ns/oscal/sa/xoscal-server \
    -parentID spiffe://prod.example.com/spire/agent/k8s_psat/prod-cluster/NODE_UID \
    -selector k8s:ns:oscal \
    -selector k8s:sa:xoscal-server
```

### 2. Deploy with Sidecar

```bash
kubectl apply -f k8s/spire-auth-configmap.yaml
kubectl apply -f k8s/spire-sidecar.yaml
```

The deployment includes:
- OSCAL gRPC server container
- SPIRE helper sidecar
- Shared memory volume for JWT-SVID
- SPIRE Agent socket mount

## Verification

### 1. Verify JWT-SVID Fetch

```bash
spire-agent api fetch jwt \
    -audience https://api.anthropic.com \
    -socketPath /run/spire/sockets/agent.sock
```

### 2. Verify gRPC Authentication

```python
import grpc
from oscal.services.v1 import oscal_service_pb2_grpc

# Try without token (should fail)
channel = grpc.insecure_channel('localhost:50051')
stub = oscal_service_pb2_grpc.OscalServiceStub(channel)
try:
    stub.GetCatalog(request)
except grpc.RpcError as e:
    print(f"Expected failure: {e.code()}")  # UNAUTHENTICATED

# Try with valid token (should succeed)
# ... add JWT-SVID to metadata
```

## Security Considerations

1. **Trust Domain Scope**: Ensure `subject_prefix` in federation rules is scoped to specific workloads, not the entire trust domain.

2. **Audience Matching**: Always set and validate the `audience` claim to prevent token replay across services.

3. **Token Lifetime**: Keep JWT-SVID TTL at 5 minutes or less to limit token exposure.

4. **Key Rotation**: If using inline JWKS (not discovery mode), update keys on every SPIRE rotation.

5. **Socket Permissions**: Ensure SPIRE Agent socket has proper permissions (typically root:spire).

## Troubleshooting

### JWT-SVID Fetch Fails

```bash
# Check SPIRE Agent is running
ps aux | grep spire-agent

# Check socket exists
ls -la /run/spire/sockets/agent.sock

# Check workload registration
spire-server entry show -spiffeID spiffe://prod.example.com/ns/oscal/sa/xoscal-server
```

### gRPC Authentication Fails

```bash
# Verify JWT-SVID claims
spire-agent api fetch jwt -audience https://api.anthropic.com | jq

# Check `iss` matches registered OIDC Discovery Provider URL
# Check `sub` matches workload SPIFFE ID
# Check `aud` contains expected audience
```

### OIDC Discovery Provider Unreachable

```bash
# Test OIDC Discovery Provider
curl https://oidc-discovery.prod.example.com/.well-known/openid-configuration
curl https://oidc-discovery.prod.example.com/.well-known/jwks.json

# Check SPIRE Server jwt_issuer configuration
cat /etc/spire/server.conf | grep jwt_issuer
```

## Integration with Anthropic WIF

The SPIRE authentication can be integrated with Anthropic's Workload Identity Federation:

1. Register OIDC Discovery Provider URL as federation issuer
2. Create federation rule matching workload SPIFFE ID
3. Configure spiffe-helper with Anthropic audience
4. Set Anthropic environment variables

See Anthropic's [WIF with SPIRE documentation](https://docs.anthropic.com/en/docs/manage-claude/wif-providers/spire) for details.
