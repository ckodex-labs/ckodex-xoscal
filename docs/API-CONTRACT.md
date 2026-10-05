# Generated REST and gRPC contract

All four `oscal.services.v1` services are exposed by the REST gateway. Their
`google.api.http` protobuf annotations are the route authority. Default ports are
gRPC `50051`, REST `8080`, and metrics `9090`; deployments may override them.
Gateway request and response envelopes use protobuf JSON, including camelCase
field names. OSCAL model payloads are validated against the OSCAL 1.2.3 contract
where the service supports structural validation.

The pinned `buf.build/community/sudorandom-connect-openapi:v0.25.8` plugin emits
one OpenAPI 3.1 YAML file per service. `xoscal-openapi` reads the four explicitly
named service outputs and merges their paths, schemas and tags into canonical
JSON. Shared schemas must match exactly. Missing RPCs, route collisions, changed
operation identities, missing path parameters and unresolved or external schema
references block the build. Schema-only files cannot replace a service document.

```sh
buf generate
go run ./server/cmd/xoscal-openapi
go run ./server/cmd/xoscal-openapi -check
go test ./server/cmd/xoscal-openapi ./server/cmd/xoscal-gateway
dagger call openapi --source=. export --path=/tmp/openapi.json
dagger call openapi-check --source=.
```

Dagger runs pinned generation before merging; SDK bundles and the site use that
same complete document. `site/openapi.json` is the checked-in no-network preview
and generation drift oracle. Scalar loads local `openapi.json` and `scalar.js`;
the site never renames YAML content to JSON or selects an arbitrary first file.
This API document describes the service contract, not hosted availability or
cryptographic evidence that a particular release passed verification.

## Search route compatibility

`OscalService.Search` retains `GET /v1/search`. The governance
`SemanticSearch` HTTP annotation is `GET /v1/search/semantic`. Previously both
RPCs annotated `GET /v1/search`, so registration order made one unreachable.
Clients intending governance semantic search must use `/v1/search/semantic`;
an alias at `/v1/search` cannot safely preserve both request/response contracts.
The protobuf package, gRPC method names and messages remain compatible. The
generated gateway route test exercises both searches together in application
registration order to prevent this ambiguity returning.

Canonical component-definition and transparency routes are
`/v1/component-definitions`, `/v1/transparency/claims`, and
`/v1/transparency/claims/{claim_id}/receipt`. Path parameters such as
`{uuid.value}` preserve the annotated nested UUID field; callers substitute the
UUID value in the URL.
