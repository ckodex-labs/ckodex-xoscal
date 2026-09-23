# xOSCAL

A high-assurance, developer-friendly engine and CLI for NIST OSCAL (Open Security Controls Assessment Language 1.2.3). 

xOSCAL bridges the chasm between raw compliance standards and modern software engineering, providing instant Day-0 scaffolding, continuous Vector State verification, GitOps pull-request diffing, spreadsheet synchronization for GRC analysts, and air-gapped verifiable audit exports.

---

## ⚡ 60-Second Developer Quickstart (`xoscal-ctl`)

Install or build the unified standalone CLI binary:

```bash
# Build binaries locally
make build

# Inspect the CLI suite
./bin/xoscal-ctl help
```

### 1. Day-0 Onboarding: Scaffold Component Definition
Auto-detect repository languages (Go, Python, TypeScript, Rust), container definitions, and Kubernetes manifests. Synthesize human-friendly URNs deterministically into RFC-4122 UUIDv5 identifiers and emit a schema-valid OSCAL 1.2.3 `ComponentDefinition`:

```bash
./bin/xoscal-ctl init --framework nist-sp-800-53-rev5
```

Output:
```text
🔍 Scanning repository at: . ...
[OK] Detected Project: my-service (Languages: [Go], Docker: true, K8s: true, TF: false)
[OK] Mapped 2 component(s) using virtual URNs -> deterministic UUIDv5
[OK] Verified compliance with official NIST OSCAL 1.2.3 JSON schema
[OK] Created workspace config: .xoscal/xoscal.yaml
[OK] Emitted component definition: component-definition.json (3907 bytes)
```

### 2. Continuous Verification & Vector Posture
Replace binary "pass/fail" check-the-box theater with the formal vector state product:
$$S(e,t) = \langle P, V, A, C, E, L, \tau \rangle$$
*(Presence, Valence, Anti-conflict, Coherence, Evidence status, Lifecycle mode, Temporal epoch)*:

```bash
./bin/xoscal-ctl verify --file component-definition.json
```

Output:
```text
=== xOSCAL GOVERNANCE VECTOR POSTURE ===
Artifact:  component-definition.json (component-definition)
Timestamp: 2026-09-23T21:51:13Z

Vector Product: S(e,t) = <P=PRESENT, V=POSITIVE, A=0, C=COHERENT, E=VERIFIED, L=NORMAL>

Operational Posture: NORMAL (Coherence: COHERENT)
Controls Summary:    7 Total | 7 Passing | 0 Degraded | 0 Derogated | 0 Anti-Conflicts

CONTROL        COMPONENT            POSTURE          DESCRIPTION / REMEDIATION
-------------------------------------------------------------------------------------
ac-2           my-service Core      PASS             Account management implemented v...
sc-8           my-service Core      PASS             Transmission confidentiality and...
sc-13          my-service Core      PASS             Cryptographic protection impleme...
```

### 3. GitOps PR Compliance Linter (`xoscal-ctl diff`)
Run in GitHub Actions or GitLab CI to detect control drift, deletions, and posture regressions between git branches. Emits a clean Markdown summary table ready for PR comments (`$GITHUB_STEP_SUMMARY`):

```bash
# Markdown output for Pull Request comments
./bin/xoscal-ctl diff --base main.component-definition.json --head component-definition.json --markdown >> $GITHUB_STEP_SUMMARY

# Enforce a strict build gate that fails on posture regressions
./bin/xoscal-ctl diff --base main.json --head head.json --fail-on-regression
```

### 4. Mathematical Risk Acceptance (`xoscal-ctl derogate`)
Enforces **Constitutional Rule 23**: *"Accepted risk does not rewrite history."* Teams no longer fabricate fake passes or delete controls to pass CI:

```bash
# Authorize a time-bounded risk exception
./bin/xoscal-ctl derogate add \
  --control sc-8 \
  --scope service:internal-worker \
  --authority urn:xoscal:authority:ciso-alex \
  --reason "Internal worker in isolated VPC without external egress" \
  --ttl-days 30 \
  --compensating "sc-7 boundary enforcement, au-2 audit logging"

# Active exceptions grant SAFE_HOLD without failing CI; expired exceptions immediately block builds
./bin/xoscal-ctl verify
```

### 5. Non-Technical GRC Bridge (`xoscal-ctl tabular`)
Compliance analysts live in Excel. Seamlessly round-trip between OSCAL 1.2.3 JSON and CSV/Excel tables:

```bash
# Export controls to CSV for spreadsheet editing
./bin/xoscal-ctl tabular export --in component-definition.json --out controls.csv

# Import edited responses back into schema-valid OSCAL JSON
./bin/xoscal-ctl tabular import --in controls.csv --base component-definition.json --out component-definition.updated.json
```

### 6. Auditor "Audit-in-a-Box" Offboarding (`xoscal-ctl bundle-export`)
Generate an immutable, air-gapped `.tar.gz` bundle for external regulators and 3PAO auditors with zero vendor lock-in. Includes canonical OSCAL JSON files, evidence blobs with SHA-256 sidecars, a cryptographic `manifest.json`, and an embedded zero-dependency standalone HTML viewer (`audit-viewer.html`):

```bash
./bin/xoscal-ctl bundle-export --in component-definition.json --evidence-dir ./evidence --out audit-bundle.tar.gz
```
An auditor in a disconnected SCIF can open `audit-viewer.html` directly via `file://` in any browser, inspect controls, and verify evidence SHA-256 digests offline via the browser-native W3C WebCrypto API.

---

## 🏛️ Architecture & gRPC Service

```plaintext
┌─────────────────────────────────────────────────────────────┐
│                 Clients: xoscal-ctl / gRPC                  │
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

The gRPC server (`xoscal-server`) exposes CRUD, cross-model full-text search, and knowledge graph mappings for enterprise service environments:

```bash
# Run server with local persistent SQLite database
./bin/xoscal-server -dsn oscal.db

# Standard gRPC health checks and reflection are enabled
grpcurl -plaintext localhost:50051 list
```

---

## 🔒 Trust Boundary & Beta Status

Per [docs/BETA-RELEASE-NOTES.md](docs/BETA-RELEASE-NOTES.md):
- **Current Scope**: Single-workspace operator review tool backed by embedded SQLite.
- **Evidence Verification**: Deterministic SHA-256 digest sidecars, Ed25519 detached signatures, and official NIST OSCAL 1.2.3 JSON schema validation.
- **Zero-Trust**: Unverified or candidate claims are never promoted to attestations without verifiable proof.

---

## 🛠️ Verification Ladder

Every build passes the full constitutional verification ladder:

```bash
make build        # Compiles bin/xoscal-server and bin/xoscal-ctl
make test         # Runs all unit & conformance tests
make test-race    # Executes Go race detector across all packages
make lint         # Runs buf lint, go vet, and gofmt
python3 scripts/design-lint.py  # Enforces CKODEX-DS-3 editorial constraints
python3 scripts/a11y-lint.py    # Enforces WCAG 3.0 static accessibility checks
```
