# CISO Assistant Framework Ingestion

This document describes the ingestion pipeline for importing CISO Assistant community library YAML files into the xoscal knowledge graph, generating OSCAL artifacts, and managing cross-framework mappings.

## Architecture Overview

```
CISO Assistant GitHub Repo
         |
         v
  GitHub SDK Fetcher  ──→  Local YAML Cache
         |                         |
         v                         v
  CISO Assistant Parser  ←──  Raw YAML
         |
         v
  KG Normalizer  ──→  Knowledge Graph (SQLite)
         |                    |
         v                    v
  OSCAL Generator  ←──  Snapshots
         |
         v
  OSCAL Artifacts (Catalog, Profile, Mapping, SSP, POAM)
```

## Data Model

### Extended Requirement Entity

The `Requirement` entity in the KG now carries CISO Assistant-specific fields:

| Field | Type | Description |
|-------|------|-------------|
| `depth` | int | Hierarchy depth (1 = top-level, 2+ = nested) |
| `parent_urn` | string | URN of the parent requirement node |
| `assessable` | bool | Whether this node is a leaf control (assessable) or a group header |
| `implementation_groups` | []string | Maturity levels / tiers applicable (e.g., `["L1", "L2"]`) |
| `ref_id` | string | CISO Assistant node reference ID |
| `node_name` | string | Human-readable name of the node |

### Framework Entity

Stores metadata for each ingested framework:

| Field | Description |
|-------|-------------|
| `ref_id` | Short identifier (e.g., `cmmc-2.0`) |
| `name` | Human-readable name |
| `provider` | Publishing organization |
| `version` | Version string |
| `locale` | Language code |

### RequirementMapping Entity

Represents cross-framework mappings:

| Field | Description |
|-------|-------------|
| `source_urn` | Source requirement URN |
| `target_urn` | Target requirement URN |
| `relationship` | `equal`, `subset`, `superset`, `intersect` |
| `rationale` | Human-readable justification |
| `strength` | Confidence score (0–10) |

## API Usage

### Bulk Ingestion

```bash
grpcurl -plaintext localhost:50051 \
  oscal.services.v1.GovernanceService/BulkIngestFrameworks \
  -d '{
    "github_owner": "intuitem",
    "github_repo": "ciso-assistant-community",
    "github_path": "backend/library/libraries",
    "filter": "cmmc-2.0,nist-csf-2.0,cyber_essentials,bs-it-gs-2023-isms*"
  }'
```

### List Ingested Frameworks

```bash
grpcurl -plaintext localhost:50051 \
  oscal.services.v1.GovernanceService/ListFrameworks
```

### Generate OSCAL Catalog

```bash
grpcurl -plaintext localhost:50051 \
  oscal.services.v1.GovernanceService/GenerateCatalog \
  -d '{"snapshot_name": "cmmc-2.0", "framework": "cmmc-2.0"}'
```

### Generate Cross-Framework Mappings

```bash
grpcurl -plaintext localhost:50051 \
  oscal.services.v1.GovernanceService/GenerateCrossFrameworkMappings \
  -d '{
    "snapshot_name": "main",
    "source_framework": "nist-csf-2.0",
    "target_framework": "iso27001-2022"
  }'
```

## Supported Jurisdictions

| Jurisdiction | Example Frameworks |
|--------------|-------------------|
| USA | NIST CSF 2.0, CMMC 2.0, FedRAMP Rev 5, CISA CPG 2.0, CJIS, NYDFS 500 |
| Canada | ITSP.10.171 |
| UK | NCSC Cyber Essentials v3.1 |
| France | ANSSI guides, CNIL, 3CF |
| Germany | BSI IT-Grundschutz 2023, BSI C5, e-ITS 2024 |
| EU-wide | NIS2, DORA |

## Adding New Frameworks

1. Identify the CISO Assistant YAML filename in `backend/library/libraries/`.
2. Add an entry to `data/frameworks/manifest.yaml` with `ref_id`, `jurisdiction`, and `filename`.
3. Run bulk ingestion with the filter matching the new `ref_id`.

## URN Conventions

- **Requirements**: `urn:<framework>:req:<ref_id>`
- **Frameworks**: `urn:ciso:framework:<ref_id>`
- **Mappings**: `urn:ciso:mapping:<source>:<target>:<ref_id>`

## Authentication

The GitHub SDK fetcher respects the `GITHUB_TOKEN` environment variable for authenticated API requests. Unauthenticated requests work but are subject to lower rate limits (60 requests/hour).

```bash
export GITHUB_TOKEN=ghp_xxx
./bin/xoscal-server
```

For K8s deployments, the SPIRE/SPIRE sidecar can inject a JWT or mTLS identity that the fetcher can use as a bearer token.

## Caching

Downloaded YAML files are cached in `data/cache/` with a naming scheme that includes the repository path and Git SHA prefix. Cache invalidation is manual: delete `data/cache/` to force re-download.
