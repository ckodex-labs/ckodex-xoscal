# Canadian Sovereign Compliance Frameworks

Status: Authoritative  
Scope: Canadian Centre for Cyber Security (CCCS) and Treasury Board Secretariat (TBS) compliance suites in NIST OSCAL 1.2.3.

---

## 1. Overview

The Canadian Sovereign Frameworks suite in xOSCAL provides native, machine-verifiable NIST OSCAL 1.2.3 catalogs and profiles for all Government of Canada cybersecurity standards.

All frameworks are registered in the unified framework registry (`server/internal/frameworks`) and implemented through the Canadian frameworks provider (`server/internal/canadianframeworks`).

### Registered Frameworks

| Framework ID | Aliases | Authority | Standard ID | Model Kind | Description |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `cccs-itsp-10-033` | `itsp.10.033`, `itsp10033`, `itsp-10-033` | CCCS / CSE | ITSP.10.033 | Catalog | Cyber security and privacy risk management series: A lifecycle approach — Security and privacy controls and assurance activities catalogue (supersedes ITSG-33 Annex 3A, effective March 31, 2026). |
| `cccs-itsg-33` | `itsg-33`, `itsg33` | CCCS / CSE | ITSG-33 | Catalog | IT security risk management: A lifecycle approach — Annex 3A Security control catalogue. |
| `cybersecure-canada`| `cybersecure`, `csc` | ISED / CCCS | CyberSecure | Catalog | CyberSecure Canada certification baseline security controls for small and medium organizations. |
| `cccs-itsp-10-171` | `itsp.10.171`, `itsp10171` | CCCS / CSE | ITSP.10.171 | Catalog | Security requirements for non-federal systems and organizations processing sensitive government information. |
| `cccs-medium-cloud-pbmm` | `pbmm-cloud`, `pbmm` | TBS / CCCS | PBMM Cloud | Profile | Government of Canada Cloud Security Profile: Protected B, Medium Confidentiality, Medium Availability. |

---

## 2. Architecture: Framework Abstraction Layer

The framework subsystem follows the **Pure Kernel, Explicit Transport, Shared Validation** architecture defined in the **CKODEX Constitution**:

```
server/internal/frameworks/
  ├── framework.go    # Framework interface and Info metadata
  ├── base.go         # BaseFramework configurable implementation
  ├── registry.go     # Thread-safe framework registry with aliases and jurisdiction filtering
  ├── export.go       # Schema-validated export with SHA-256 sidecars
  ├── seeder.go       # Universal Knowledge Graph seeder (Protobuf and OSCAL JSON)
  └── default.go      # Thread-safe global registry singleton
```

### Dependency Direction

- `frameworks` does not import any specific regional framework package.
- `canadianframeworks` registers into `frameworks.DefaultRegistry()` via an explicit `RegisterAll()` hook.
- All exported JSON artifacts are validated against the embedded official NIST OSCAL 1.2.3 schema before emission, generating cryptographic `.sha256` sidecars.

---

## 3. Deep Dive: CCCS ITSP.10.033 (2026 Catalogue)

Published on March 31, 2026, **ITSP.10.033** represents the Government of Canada's modernization of ITSG-33 Annex 3A, harmonized with NIST SP 800-53 Rev. 5 while incorporating distinct sovereign requirements, Canadian legal mandates, and Canadian national security practices.

### Key Metrics

- **Families**: 20 (including Personal Information Handling and Transparency `PT`, and Supply Chain Risk Management `SR`)
- **Base Controls & Activities**: 330
- **Control Enhancements**: 918
- **Total Addressable Controls**: 1,248
- **Organization-Defined Parameters (ODP)**: 1,272
- **Schema Conformance**: NIST OSCAL 1.2.3 JSON Schema (0 errors)
- **Artifact Size**: 3.39 MB (`catalog.json`)

### Sovereign Canadian Delta

ITSP.10.033 introduces Canadian sovereign extensions that are explicitly modeled and tagged in OSCAL:

1. **Canadian Base Controls (400-series)**:
   - `SA-400`: Sovereignty Requirements for Cloud and Managed Services
   - `PE-400`: Canadian Sovereign Physical Access
   - `PE-401`: Government of Canada Facility Accreditation
   - `SI-400`: National Threat Intelligence Sharing and Notification

2. **Canadian Control Enhancements**:
   - 43 enhancements carrying numbers $\ge 400$ (e.g. `AC-2(400)`, `IA-2(400)`), tagged with:
     ```json
     {"name": "jurisdiction", "value": "ca"},
     {"name": "sovereign-control", "value": "true"}
     ```

3. **Canadian Statements (`AA.`, `BB.`)**:
   - Sub-statements specific to Canadian jurisdictional and policy requirements are decomposed into distinct `item` parts with:
     ```json
     {"name": "jurisdiction", "value": "ca"},
     {"name": "sovereign-specification", "value": "true"}
     ```

4. **GC Discussion Parts**:
   - Government of Canada guidance sections (`GC discussion:`) are isolated as dedicated `guidance` parts titled `"GC Discussion"` with sovereign tagging:
     ```json
     {
       "id": "ca-ac-2_gc_gdn",
       "name": "guidance",
       "title": "GC Discussion",
       "props": [
         {"name": "jurisdiction", "value": "ca"},
         {"name": "sovereign-guidance", "value": "true"}
       ],
       "prose": "..."
     }
     ```

5. **Balanced ODP Parameter Interpolation**:
   - Parameters are extracted from `[Assignment: ...]` and `[Selection: ...]`.
   - Nested selections (e.g. a Selection containing nested Assignments) are resolved iteratively from the innermost bracket outward.
   - Text interpolation uses OSCAL standard syntax: `{{ insert: param, <param-id> }}`.
   - All `param.label` and `param.select.choice` elements satisfy `MarkupLineDatatype` (strictly single line without newline characters).

---

## 4. CLI Operations

### Listing Frameworks

```bash
# List all registered frameworks
xoscal-ctl frameworks list

# Filter by jurisdiction
xoscal-ctl frameworks list --jurisdiction ca
```

Sample output:
```
REGISTERED COMPLIANCE FRAMEWORKS:
ID                      JURISDICTION  VERSION       NAME
cccs-itsp-10-033        ca            1.0           CCCS ITSP.10.033 (2026)
cccs-itsg-33            ca            1.0           CCCS ITSG-33 Annex 3A
cybersecure-canada      ca            1.0           CyberSecure Canada
cccs-itsp-10-171        ca            1.0           CCCS ITSP.10.171
cccs-medium-cloud-pbmm  ca            1.0           CCCS Cloud Medium (PBMM) Profile
```

### Exporting Schema-Validated Artifacts

```bash
# Export ITSP.10.033 Catalog and Profile with automated schema validation and .sha256 sidecars
xoscal-ctl frameworks export \
  -framework cccs-itsp-10-033 \
  -out ./data/frameworks

# Export all registered frameworks
xoscal-ctl frameworks export -out ./exported-frameworks
```

### Seeding the Knowledge Graph

```bash
# Seed SQLite Knowledge Graph with all 5 Canadian frameworks
xoscal-ctl frameworks seed -db oscal.db
# or top-level: xoscal-ctl seed -dsn oscal.db
```
xoscal-ctl frameworks seed --db oscal.db
```

---

## 5. Verification and Conformance Evidence

1. **Schema Validation**:
   ```bash
   go run ./server/cmd/xoscal-validate-schema \
     -file data/frameworks/cccs-itsp-10-033/catalog.json \
     -kind catalog
   # Output: VALID: catalog passes OSCAL 1.2.3 schema validation
   ```

2. **Cryptographic Checksums**:
   Every exported framework artifact is accompanied by a SHA-256 sidecar file (`.sha256`) ensuring byte-for-byte immutability across air-gapped environments.
