# OSCAL Spec Integrity Registry — Design

- Date: 2026-06-21
- Status: Approved (pending spec review)
- Depends on: the reconcile loop (`recipes/`, `data/oscal/upstream.lock.yaml`,
  `scripts/reconcile/poll-feeds.sh`) and the `fetcher` package, both on `main`.

## 1. Problem

The repo's OSCAL protos are hand-authored to shadow NIST's OSCAL schema, but
nothing pins *which exact upstream schema bytes* they correspond to. The
reconcile lock has a `schema_sha256` field that is empty, and "we are conformant
to OSCAL v1.2.2" is a label, not a verified claim. Two failure modes are
currently invisible:

1. **Upstream re-tag / silent change** — if NIST re-publishes a schema under the
   same version tag, or a model's schema changes, we have no way to detect it.
2. **Unanchored conformance** — there is no per-model record binding an upstream
   OSCAL model schema to the proto that implements it.

We must stay **provably lock-step with the OSCAL specs**: pin the exact upstream
per-model schema bytes and detect, per model, when upstream moves.

## 2. Goals / Non-goals

Goals:
- A committed, versioned registry mapping each OSCAL model → its upstream schema
  sha256 → the proto that implements it.
- Detect per-model upstream drift (re-tag, silent change) as a reconcile signal.
- Populate the registry deterministically from a pinned OSCAL release; fail loud
  on a missing/renamed upstream asset (no phantom URLs).

Non-goals (YAGNI):
- **Semantic conformance proof.** The registry proves *which bytes we pinned* and
  detects upstream change; it does NOT prove a proto semantically matches its
  schema. That remains the job of the `oscal-schema-reconciler` agent + `buf`.
  The registry is the trigger/anchor ("model X drifted → re-verify proto X").
- SDK-hash chaining (the "full conformance chain" option was declined).
- Signing the registry (committed + reviewed is the integrity boundary for now;
  cosign is a follow-up).

## 3. Architecture (Four Spaces)

Lives in **Proof** (integrity ledger) + a thin **Presentation** CLI. No domain
logic. Reuses the existing `fetcher` download path.

```
pinned OSCAL release ──fetch per-model schemas──► xoscal-spec-registry
                                                    ├─ populate → spec-registry.yaml (committed)
                                                    └─ verify   → recompute vs registry → drift?
poll-feeds.sh ──(version drift)──┐
spec-registry verify ─(hash drift per model)─┴─► reconcile draft PR / CI gate
```

## 4. The registry file

`data/oscal/spec-registry.yaml`:

```yaml
version: "1.0"
oscal_version: "1.2.2"          # the pinned release these hashes are taken from
generated_at: ""                # ISO-8601, filled by populate
models:
  - model: catalog
    schema_asset: oscal_catalog_schema.json
    schema_sha256: ""           # filled by populate
    proto: proto/oscal/catalog/v1/catalog.proto
  - model: profile
    schema_asset: oscal_profile_schema.json
    schema_sha256: ""
    proto: proto/oscal/profile/v1/profile.proto
  - model: ssp
    schema_asset: oscal_ssp_schema.json
    schema_sha256: ""
    proto: proto/oscal/ssp/v1/ssp.proto
  - model: component-definition
    schema_asset: oscal_component_schema.json
    schema_sha256: ""
    proto: proto/oscal/component_definition/v1/component.proto
  - model: assessment-plan
    schema_asset: oscal_assessment-plan_schema.json
    schema_sha256: ""
    proto: proto/oscal/assessment_plan/v1/assessment_plan.proto
  - model: assessment-results
    schema_asset: oscal_assessment-results_schema.json
    schema_sha256: ""
    proto: proto/oscal/assessment_results/v1/assessment_results.proto
  - model: poam
    schema_asset: oscal_poam_schema.json
    schema_sha256: ""
    proto: proto/oscal/poam/v1/poam.proto
  - model: mapping
    schema_asset: oscal_mapping_schema.json   # OSCAL >= 1.2.0
    schema_sha256: ""
    proto: proto/oscal/mapping/v1/mapping.proto
```

Notes:
- `common.proto` has no upstream OSCAL model schema (it holds shared types) and is
  intentionally excluded; this is documented in the file header.
- `schema_asset` names follow OSCAL's `oscal_<model>_schema.json` release-asset
  convention. The exact names are confirmed at `populate` time against the actual
  release; a missing asset is a hard failure (P-VW-003), never silently skipped.
- Digest format follows the repo convention: `sha256:<hex>`.

## 5. The CLI — `xoscal-spec-registry`

`server/cmd/xoscal-spec-registry/main.go`, flags:
- `-mode populate|verify` (required)
- `-registry data/oscal/spec-registry.yaml`
- `-version` (default: read `oscal_version` from the registry; for populate, the
  target version)

Schema fetch URL: `https://github.com/usnistgov/OSCAL/releases/download/v{version}/{schema_asset}`
(matches the reconcile recipe's `schema_url_template` host pattern).

**populate:** for each model, download `schema_asset` for `version`, compute
sha256, write `schema_sha256` + `generated_at` + `oscal_version` into the
registry. A missing/renamed asset → non-zero exit with the model + URL named.

**verify:** for each model, re-download and recompute; compare to the registry.
Output a JSON report `{oscal_version, drift: bool, models:[{model, expected,
actual, status}]}`; exit 0 if all match, 3 if any model drifted, 1 on
operational error. Mirrors `poll-feeds.sh`'s exit-code contract.

Reuse: a new `server/internal/specregistry` package holds the pure logic
(parse/serialize the registry, compare a fetched hash to the recorded one,
aggregate per-model status into a drift verdict + exit code). populate/verify in
the CLI are thin I/O wrappers over it. The pure functions are unit-tested (TDD);
the network fetch is not (deferred to CI/engine).

## 6. Reconcile + CI wiring

- **Recipe:** `recipes/oscal.yaml` gains `spec_registry: data/oscal/spec-registry.yaml`
  so the reconcile flow knows where the ledger is.
- **Poller:** after the existing version-drift check, when in sync on version,
  run `xoscal-spec-registry -mode verify`; a per-model hash mismatch is drift
  (exit 3) and names the model in the report — catches re-tags even when the
  version string is unchanged.
- **Reconcile PR:** on a version bump, the reconcile workflow runs
  `-mode populate` to refresh the registry as part of the draft PR.
- **Gate:** a dagger `SpecRegistryCheck(source)` runs `verify` in-container
  (network available) and is wired into `All`, alongside `ProtoCheck`.
- **Lock tie-in:** `upstream.lock.yaml`'s `schema_sha256` is removed in favor of
  the per-model registry (single source of truth), or populated with a digest of
  the registry file itself. Decision: drop the bundle field; the registry is the
  anchor.

## 7. Error handling / honesty

- Missing/renamed upstream asset at populate → hard fail naming model + URL.
- `verify` distinguishes drift (exit 3) from operational failure (exit 1).
- The registry header states the boundary: it pins bytes and detects upstream
  change; it is NOT a semantic proto↔schema conformance proof.
- Network is required for populate/verify (deferred to CI/engine, like the
  existing dagger network steps); the local sandbox cannot run them.

## 8. Verification

- Unit tests (TDD) for the pure helpers: registry parse, hash-compare producing
  correct per-model status, drift aggregation, exit-code mapping.
- `go vet` / `gofmt` clean; `xoscal-spec-registry` builds.
- `dagger call spec-registry-check --source=.` on the engine (manual/CI) for the
  end-to-end fetch+verify.
- Honesty test: a tampered expected hash in the registry → `verify` reports that
  model as drift, exit 3.

## 9. Open follow-ups (out of scope)

- cosign-signing the registry.
- Chaining proto-hash and SDK-hash into the registry (full conformance chain).
- Auto-running the `oscal-schema-reconciler` agent when verify reports per-model
  drift (currently a manual handoff).
