# AGENTS.md — ckodex-oscalify

## Build

```sh
buf generate                     # regenerate proto code after .proto changes
go build -buildvcs=false ./...   # build all packages (safe across isolated/sandboxed environments)
# Or using the Makefile:
make build
```

## Diagnostics (Rule 36 Deep Diagnostics)

```sh
go run -buildvcs=false ./server/cmd/xoscal doctor        # deep runtime, schema & registry check
go run -buildvcs=false ./server/cmd/xoscal doctor -json  # machine-verifiable Decision/Evidence trace
# Or using the Makefile:
make doctor
```

## Test

```sh
go test ./server/...                            # all server tests
go test ./server/cmd/xoscal/...                 # unified CLI binary
go test ./server/cmd/xoscal-ctl/...             # CLI command tests (incl. frameworks & seed)
go test ./server/internal/frameworks/...        # framework abstraction, registry, controls & compare
go test ./server/internal/canadianframeworks/...# Canadian sovereign frameworks tests
go test ./server/internal/ontology/...          # OWL/JSON-LD v1.0.0 regulatory & threat ontology tests
go test ./server/internal/graph/...             # property graph, 36-framework crosswalk, blast-radius & reconciler tests
go test ./server/internal/feeds/...             # CISA KEV, MITRE ATT&CK, crosswalk & flight-recorder feed tests
go test ./server/internal/resilience/...        # circuit breaker, retry w/ jitter, atomic fsync & HTTP backpressure tests
go test ./server/internal/utcp/...              # Universal Tool Calling Protocol (UTCP v1.0.1) & multi-language SDK tests
go test ./server/internal/webstudio/...         # embedded web studio & API tests
go test ./server/internal/completion/...        # shell completion tests
go test ./server/internal/integrity/...         # semantic reference integrity validator
go test ./server/internal/oscal/... -count=1    # OSCAL generator + serializer tests
go test ./server/internal/schemavalidate/...    # schema validation + conformance tests
go test ./server/cmd/xoscal-gateway/...         # gateway marshaler tests
go test ./server/cmd/xoscal-validate-constraints/... # constraint validator tests
```

## Lint

```sh
buf lint                  # proto lint
buf format -d --exit-code # proto format check (use -w to fix)
```

## Schema Validation

The project embeds the official OSCAL 1.2.3 JSON schema and validates all
artifact types against it. Key packages:

- `server/internal/schemavalidate` — schema validator (embeds `oscal_complete_schema.json`)
- `server/internal/oscal` — OSCAL-compliant JSON exporters and generator

### Conformance tests

- `schemavalidate/conformance_test.go` — validates proto-generated artifacts for all 8 model types
- `schemavalidate/validator_test.go` — schema loading and validation tests
- `oscal/e2e_conformance_test.go` — end-to-end: generator → exporter → schema validation
- `xoscal-gateway/oscal_marshaler_test.go` — gateway marshaler produces schema-valid JSON

### CLI schema validation

Two tiers:

```sh
# Tier 1 — JSON Schema (structural, offline, embedded schemas):
go run ./server/cmd/xoscal-generate -snapshot <name> -validate   # validate on generate
go run ./server/cmd/xoscal-validate-schema <file> <kind>         # standalone schema validator

# Tier 2 — Metaschema constraints (semantic, requires oscal-cli on PATH):
go run ./server/cmd/xoscal-generate -snapshot <name> -validate-constraints   # tier 1 + tier 2 on generate
go run ./server/cmd/xoscal-validate-constraints -file <file>                 # auto-detect kind, schema pre-check + oscal-cli
go run ./server/cmd/xoscal-validate-constraints -dir <dir> -oscal-cli <path> # batch validate a directory
go run ./server/cmd/xoscal-validate-constraints -file <file> -skip-schema    # skip tier 1, oscal-cli only
```

Tier 2 enforces Metaschema constraints that JSON Schema cannot express:
`<allowed-values>`, `<has-cardinality>`, `<index-has-key>`, `<is-unique>`, `<matches>`.

## CI

Dagger pipeline (`dagger call all --source=.`) runs in parallel: lint, test, race,
security, proto check, spec registry check, OSCAL schema validation (tier 1,
blocking), and OSCAL Metaschema constraint validation (tier 2, non-blocking via
oscal-cli).

## OSCAL Compliance Notes

- The serializer (`oscal_json.go`) transforms protojson to OSCAL-compliant JSON:
  - camelCase → kebab-case
  - `{"value":"x"}` → `"x"` (unwrap wrapper types)
  - Deprecated fields are stripped (see `oscalDeprecatedFields` map)
  - Renamed fields are mapped (see `oscalFieldRenames` map)
- Proto fields that don't match the OSCAL schema are marked `[deprecated = true]`
  and replaced with schema-compliant fields
- All 7 OSCAL model types have proto-to-schema conformance tests

## Compliance Framework Registry

The `server/internal/frameworks` package provides a jurisdiction-agnostic framework abstraction:
- `frameworks.Framework`: Core interface for catalog and profile providers, metadata, and Knowledge Graph seeders.
- `frameworks.Registry`: Thread-safe registry supporting canonical registration, alias resolution, and jurisdiction filtering (`FindByJurisdiction`).
- `frameworks.ProjectAllLayers`: Projects all 7 OSCAL 1.2.3 layers (Catalog, Profile, Component Definition, System Security Plan, Assessment Plan, Assessment Results, POA&M) with deterministic RFC 4122 v5 UUID cross-layer linkages and full control/parameter implementation mappings.
- `frameworks.Export`: Exports all 7 schema-validated OSCAL JSON layers with cryptographic SHA-256 sidecars.
- `frameworks.VerifyArtifactParity`: Verbatim parity verification engine (`xoscal verify-parity`) comparing framework models against OSCAL artifacts across all 7 model types with hard anti-invariant checks (placeholder titles, empty statements, unextracted ODPs, missing sovereign tags).
- `frameworks.SeedKnowledgeGraph`: Ingests framework requirements into SQLite Knowledge Graph store.
- `frameworks/manifest_provider.go`: Registers all 31 non-Canadian sovereign & international frameworks from `data/frameworks/manifest.yaml` (USA: 7, UK: 1, France: 6, Germany: 12, EU: 3, Estonia: 1, International: 1; 14,058 controls total) alongside the 5 Canadian frameworks (2,179 controls total), for 36 registered frameworks and 16,237 controls across 252 7-layer OSCAL 1.2.3 artifacts.

### Verbatim Parity & Framework Export CLI

```sh
go run -buildvcs=false ./server/cmd/xoscal-export-frameworks -manifest data/frameworks/manifest.yaml -out data/frameworks
go run -buildvcs=false ./server/cmd/xoscal frameworks export -out data/frameworks
go run -buildvcs=false ./server/cmd/xoscal verify-parity -dir data/frameworks -strict
```

### Canadian Sovereign Frameworks Provider

`server/internal/canadianframeworks` registers the 5 Canadian national standards:
1. `cccs-itsp-10-033`: CCCS ITSP.10.033 (Effective March 31, 2026, superseding ITSG-33 Annex 3A; 1,248 controls + 384-control Medium Impact profile).
2. `cccs-itsg-33`: CCCS ITSG-33 Annex 3A Security Control Catalogue (234 active base controls across 18 families).
3. `cybersecure-canada`: ISED CyberSecure Canada / CAN/DGSI 104 baseline controls (46 controls across 14 families).
4. `cccs-itsp-10-171`: CCCS ITSP.10.171 Security Requirements for Non-Federal Systems (267 assessable controls across 17 families).
5. `cccs-medium-cloud-pbmm`: TBS/CCCS Protected B, Medium Integrity, Medium Availability Cloud Profile (384 selected controls and enhancements).

### ITSP.10.033 Generation Pipeline

The generation pipeline (`scripts/generate_itsp10033_oscal.py`) parses the official Canadian Centre for Cyber Security PDF (`itsp.10.033-2026-eng.pdf`) and executes the 7 Subagent Strategies:
- Extracts all 20 families, 330 base controls, and 918 enhancements (1,248 total controls).
- Resolves 1,273 Organization-Defined Parameters (ODP) using iterative balanced-bracket substitution (plus unclosed EOL ODP recovery) with `{{ insert: param, ... }}` interpolation.
- Decomposes control and enhancement statements into structured AST items (`A.`, `B.`, `1.`, `a.`, and Canadian `AA.`, `BB.`).
- Extracts `guidance` (Discussion) and Canadian-specific `gc-guidance` (`GC discussion`) parts bounded strictly before `Enhancements:` and `References:`, tagged `jurisdiction: "ca"` and `sovereign-guidance: "true"`.
- Validates the resulting 3.44 MB catalog against the embedded NIST OSCAL 1.2.3 JSON schema (`xoscal-validate-schema`) with 0 errors.
