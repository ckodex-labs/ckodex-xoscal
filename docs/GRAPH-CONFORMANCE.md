# Transparency graph conformance

The machine-readable graph conformance contract is [conformance/graph.yaml](../conformance/graph.yaml).
The vectors are executable evidence, not a documentation-only checklist; the
referenced Go tests run under `go test ./...` and are included in the Dagger
`all` gate.

| Vector | Contract |
| --- | --- |
| `CV-GRAPH-001` | Projection derives deterministic endpoint identity and preserves claim/evidence bindings. |
| `CV-GRAPH-002` | Projection writes nodes, edge, and audit event atomically. |
| `CV-GRAPH-003` | Successful projections emit append-only hash-chained events. |
| `CV-GRAPH-004` | Audit history is fail-closed when the global chain is invalid. |
| `CV-GRAPH-005` | Closure never returns verified when a reachable edge is not verified. |

The projection audit event uses the `graph-projection-v1` hash domain and is
committed in the same SQLite transaction as the graph edge. The API returns the
event with `ProjectEdge` and exposes verified history through
`ListProjectionEvents`.
