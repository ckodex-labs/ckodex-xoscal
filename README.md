# xoscal

A Go gRPC server for managing OSCAL (Open Security Controls Assessment Language) artifacts, backed by SQLite. Part of the broader reg-to-OSCAL engine for converting regulatory texts into structured compliance documents.

## Portal

Proto docs, supply-chain transparency, SDK clients, and OSCAL framework
downloads are published to GitHub Pages (built by `dagger call site`).

- Live local preview (real data, needs the Dagger engine + network):
  `make site-serve` → http://localhost:8080
- Standalone preview (sample data, no server, no build): open
  `portal-preview.html` directly in a browser.

## Architecture

```plaintext
┌─────────────────────────────────────────────────────────────┐
│                     gRPC Clients                            │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│              OSCAL gRPC Service (port 50051)                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐   │
│  │   Catalog    │  │   Profile    │  │ Component Def.   │   │
│  │   CRUD +     │  │   CRUD +     │  │     CRUD +       │   │
│  │   Search     │  │   Search     │  │     Search       │   │
│  └──────────────┘  └──────────────┘  └──────────────────┘   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐   │
│  │     SSP      │  │  Assessment  │  │  Assessment      │   │
│  │   CRUD +     │  │    Plan      │  │    Results       │   │
│  │   Search     │  │   CRUD +     │  │   CRUD +         │   │
│  └──────────────┘  └──────────────┘  └──────────────────┘   │
│  ┌──────────────┐  ┌──────────────┐                         │
│  │    POAM      │  │   Mapping    │                         │
│  │   CRUD +     │  │   CRUD +     │                         │
│  │   Search     │  │   Search     │                         │
│  └──────────────┘  └──────────────┘                         │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│              SQLite Store (embedded, single-node)           │
│  One table per OSCAL model: catalogs, profiles, ssps, etc.  │
│  Protobuf messages serialized as BLOBs for storage.         │
└─────────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.22+
- [Buf](https://buf.build) (for protobuf code generation)
- Docker (optional, for containerized deployment)

### Build

```bash
# Build the server binary
make build

# Or directly:
go build -o bin/xoscal-server ./server/cmd/xoscal-server
```

### Run

```bash
# In-memory SQLite (ephemeral)
./bin/xoscal-server -dsn ":memory:"

# With persistent database
./bin/xoscal-server -dsn oscal.db

# Custom port
./bin/xoscal-server -addr :8080 -dsn oscal.db
```

### Test

```bash
make test
```

### Docker

```bash
make docker
docker run -p 50051:50051 xoscal-server:latest
```

### Kubernetes

```bash
kubectl apply -f k8s/server/
```

## Protobuf Code Generation

```bash
make proto
```

This regenerates Go (and other language) SDKs from the `.proto` definitions under `proto/oscal/`.

## gRPC Service

The `OscalService` exposes CRUD + Search for all major OSCAL models:

- **Catalog** — `GetCatalog`, `ListCatalogs`, `CreateCatalog`, `UpdateCatalog`, `DeleteCatalog`
- **Profile** — `GetProfile`, `ListProfiles`, `CreateProfile`, `UpdateProfile`, `DeleteProfile`
- **Component Definition** — same pattern
- **SSP** — same pattern
- **Assessment Plan** — same pattern
- **Assessment Results** — same pattern
- **POA&M** — same pattern
- **Mapping** — same pattern
- **Search** — cross-model full-text search over title/version

Health checks are available via the standard gRPC health protocol, and reflection is enabled for `grpcurl` / Postman discovery.

## Project Structure

```plaintext
.
├── buf.yaml                     # Buf module config
├── buf.gen.yaml                 # Code generation plugins
├── go.mod                       # Go module
├── Dockerfile                   # Multi-stage container build
├── Makefile                     # Build, test, proto, lint, docker
├── proto/                       # Protobuf definitions + generated SDKs
│   └── oscal/
│       ├── common/v1/           # Shared OSCAL types (UUID, Metadata, etc.)
│       ├── catalog/v1/
│       ├── profile/v1/
│       ├── component_definition/v1/
│       ├── ssp/v1/
│       ├── assessment_plan/v1/
│       ├── assessment_results/v1/
│       ├── poam/v1/
│       ├── mapping/v1/
│       └── services/v1/         # OscalService gRPC definition
├── server/
│   ├── cmd/xoscal-server/        # CLI entrypoint
│   └── internal/
│       ├── store/               # SQLite-backed Store interface
│       └── service/             # gRPC handler implementations
└── k8s/
    └── server/                  # K8s Deployment + Service + PVC
```

## CI/CD

GitHub Actions workflow (`.github/workflows/ci.yml`) runs on every push/PR:

- `buf lint`
- `buf generate`
- `go build ./...`
- `go test ./...`
- `go vet ./...`
- `docker build`
