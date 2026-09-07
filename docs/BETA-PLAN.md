# xOSCAL beta assessment and delivery plan

Status: repository-owned beta hardening complete; external acceptance pending
Assessment date: 2026-08-14
Target: reviewable, evidence-backed beta for one workspace per deployment

## Executive verdict

The product now has a real first-use loop: an operator can open the live review
surface, import content-addressed evidence, create a claim, inspect its lineage,
run verification, and see the resulting proof state. The service and UI fail
closed when a verification provider is unavailable.

The beta is not a general attestation platform yet. It is a single-workspace
review and evidence-intake product until external provider, tenancy, and
production provenance gates are closed. The beta must preserve that boundary in
its API, UI, deployment manifests, and release notes.

## Assessment matrix

| Area | Current state | Finding | Priority |
| --- | --- | --- | --- |
| First-use journey | Live Review route and browser-tested import flow | Core journey is usable; production identity and onboarding copy still need a real workspace bootstrap path | P1 |
| Claim intake | Preflight validates identity, predicate, evidence digest, size, references, and duplicates; all-or-nothing batch commit is transactional | Live Review composes bounded batches of up to 10 records; file/CSV bulk import remains out of scope | P1 |
| Evidence integrity | Blob is stored and re-hashed from SQLite | External URI fetch and retention policy are not implemented in beta | P1 |
| Verification | Digest, Ed25519 detached signature, and bounded `xoscal-json` policy checks are real | External provider types, key rotation, transparency inclusion, and witness checks remain out of scope | P1 |
| Trust state | `candidate`, `incomplete`, `verified`, `rejected`; no invented success; append-only chained verification events; operator audit and receipt endpoints | External provider types, key rotation, transparency inclusion, and witness checks remain out of scope | P1 |
| Graph | Claims project to deterministic nodes and edges; closure fails closed; projection and its hash-chained audit event are transactional | External graph federation and richer mutation history remain out of scope | P1 |
| Tenancy | Token auth and TLS are enforced for network deployment | Data is deployment-scoped, not tenant/workspace-scoped; beta must remain single-workspace | P0 |
| UI honesty | Sample portal is labelled and separate; Live Review has no sample fallback; receipt export is proof-gated | Bounded batch flow is live; file/CSV bulk workflows remain outside the live surface | P1 |
| Accessibility | Static design/a11y gates and live browser checks pass | Full assistive-technology matrix and human keyboard sign-off remain release evidence | P0 |
| Persistence | SQLite migrations and one-replica Kubernetes posture are explicit | External database is required before multi-replica operation | P0 |
| Backup/restore | `xoscal-backup` creates non-overwriting consistent snapshots; disposable kind PVC restore passed | Deployment-owner/target-storage restore evidence and release-bundle capture remain | P0 |
| Release evidence | Dagger site verification passes with pinned digest-verified SBOM tool | Release image signing, SLSA provenance, and published artifact verification remain open | P0 |

## Product contract to freeze

### In scope for beta

1. One authenticated operator workspace per deployment.
2. Import one claim plus one or more content-addressed evidence blobs.
3. Inspect claims, source evidence, proof state, current trust state, and graph
   lineage.
4. Run digest, Ed25519 signature, and bounded policy verification; display
   unavailable external provider checks as explicit incomplete results.
5. Export only records whose required proof envelope is complete. No claim may
   be exported as attested without a valid proof object and policy decision.
6. Operate on one SQLite writer with tested backup and restore procedures.

### Explicitly out of scope for this beta

- Multi-tenant or multi-workspace data sharing in one database.
- Peer claim synchronization; the RPC returns `Unimplemented` by design.
- Attestation, sealed export, or signed release claims without actual proof.
- Multi-replica SQLite operation.
- External evidence fetching without a bounded fetch policy and audit record.

## Delivery sequence

### Phase 0 — Contract and boundary freeze

Deliverables:

- Keep [BETA-CONTRACT.md](BETA-CONTRACT.md) authoritative for claim and trust
  vocabularies.
- Document the single-workspace deployment boundary in onboarding, manifests,
  manifests, and [release notes](BETA-RELEASE-NOTES.md).
- Assign stable QA IDs to every acceptance item below.
- Add a release-visible capability matrix: available, incomplete, or out of
  scope.

Exit evidence: contract review, API/UI vocabulary scan, and no UI state that
claims `verified`, `attested`, `sealed`, or `signed` without a proof object.

### Phase 1 — Trust and proof correctness

Deliverables:

- Define typed interfaces for signature verification, policy evaluation,
  transparency inclusion, and current-state evaluation.
- Provide explicit provider states: `available`, `unavailable`, `invalid`, and
  `stale`; map them deterministically to proof state and trust state.
- Verify every proof reference against the claim digest and stored evidence.
- Persist an append-only verification event with request ID, check results,
  provider version, policy digest, and timestamp. Keep the latest projection as
  a query optimization, not the audit history.
- Expose chain-validated verification history and a deterministic receipt export
  blocked unless the latest verified projection, proof envelope, and referenced
  evidence bytes all agree.
- Make verification idempotent for the same claim, profile, and evidence set.
- Make graph projection atomic with claim/evidence eligibility checks and emit
  a projection audit event. The event is append-only, hash-chained, returned by
  `ProjectEdge`, and queryable through `ListProjectionEvents`.

Exit evidence: positive, invalid, unavailable, stale, replay, and duplicate
vectors; `verified` is unreachable when any required check is missing or
unavailable.

### Phase 2 — Operator experience

Deliverables:

- Add a workspace bootstrap/connection state with clear API, auth, and scope
  errors.
- [x] Add import preflight: parse, digest, duplicate, reference, and size
  results before mutation.
- [x] Add batch import with per-record results and all-or-nothing mode for a
  bounded batch; Live Review composes up to 10 records and file/CSV bulk import
  remains out of scope.
- [x] Add an explicit verification profile selector and explain why a claim is
  incomplete or rejected.
- [x] Add an export gate that names missing proof objects and provides the next
  action; never leave a dead or decorative beta action.
- Complete keyboard-only, screen-reader, reduced-motion, forced-colors, and
  mobile checks against the live route, not only the sample portal.

Exit evidence: browser recordings or test output for first load, empty state,
import success, duplicate, invalid digest, provider unavailable, rejection,
and blocked export.

### Phase 3 — Persistence and operations

Deliverables:

- Use `xoscal-backup` for consistent SQLite snapshots and verify backup hashes.
- Execute a restore drill on a disposable PVC: stop writer, restore through a
  maintenance pod, restore runtime ownership (`65532:65532`), start one
  replica, run schema checks, and complete the live review smoke journey.
- Add backup age, size, digest, and restore-result evidence to the release
  bundle.
- Add token rotation procedure, TLS secret rotation procedure, request IDs,
  rate-limit telemetry, and health/readiness checks to the runbook.
- Keep the single-replica guard; design the external transactional-store
  migration before any replica increase.

Exit evidence: successful restore drill, no data loss in the drill fixture,
and operator runbook signed off by the deployment owner.

### Phase 4 — Release and provenance

Deliverables:

- Build the server and backup binaries from the same immutable source revision.
- Generate SBOM, checksum manifest, SLSA provenance, and release metadata.
- Sign release archives and container image with the configured keyless or
  managed-key workflow; publish transparency references.
- Verify every published artifact from a clean environment, including the
  portal's downloaded assets and OSCAL schema/constraint outputs.
- Run Dagger `all`, generated-site verification, browser smoke, and exact-head
  CI checks before promotion.

Exit evidence: artifact digests, signature verification output, provenance
verification output, SBOM assessment, release asset inventory, and exact commit
SHA alignment.

## Prioritized issue register

### P0 — release blockers

- Deployment-scoped auth is not tenant isolation; keep beta single-workspace
  or implement persisted workspace boundaries before expanding scope.
- A disposable kind PVC restore passed with SQLite integrity, fixture
  preservation, runtime ownership, and server readiness evidence; the
  deployment owner's target storage and release-bundle capture remain open.
- Published image/archive signature and provenance verification are not yet
  evidenced.
- Full live assistive-technology acceptance is not yet evidenced.
- The oversized non-generated Go sources were split by responsibility. This
  includes the transparency exchange, OSCAL generator, graph service, primary
  store, governance service, and Dagger module. Every non-generated Go source
  file is now at or below the repository's 500-line hard limit; the largest is
  the existing exchange test at 498 lines.

### P1 — beta quality gaps

- Live Review has bounded multi-record composition (up to 10 records) plus
  CSV bulk import (100-row cap, 8 MiB file cap) with resumable pending
  batches persisted in the operator's browser; file/CSV resumable recovery
  is now implemented and browser-verified.
- External evidence fetching is implemented behind a bounded fetch policy
  (HTTPS-only, byte cap, timeout, redirect bound, optional host allowlist)
  with an append-only fetch audit trail queryable via `ListFetchEvents`;
  fetching stays disabled unless the deployment enables it.
- No peer-sync implementation; currently explicit and safe, but needs product
  decision before the beta API is advertised as exchange-capable.
- Graph projection atomicity, append-only audit history, chain verification, and
  fail-closed closure behavior are covered by `CV-GRAPH-001` through
  `CV-GRAPH-005` in [docs/GRAPH-CONFORMANCE.md](GRAPH-CONFORMANCE.md).
  Projection edge identity is content-stable when callers omit an edge ID, so
  retries cannot create duplicate graph edges or audit events.
- Verification audit history and fail-closed receipt export are available; the
  release bundle now has an operator retention and re-verification procedure
  in [docs/OPERATIONS.md](OPERATIONS.md) and a Dagger `ReleaseBundle` assembly.
- Remote policy documents (`xoscal-remote`) are fetched through the bounded
  fetch policy and evaluated as xoscal-policy-v1; a deployment key registry
  with rotation (retired keys verify but surface rotation) and a transparency
  inclusion provider (RFC 6962-style Merkle proof plus Ed25519-signed
  checkpoint) are implemented with fail-closed vectors. External signature
  algorithms beyond Ed25519 and registry-backed transparency (Rekor) remain
  out of scope.
- Gateway-to-gRPC TLS/auth configuration is now fail-closed and unit-tested;
  the Kubernetes manifest contract test verifies one SQLite writer, read-only
  TLS/token Secret mounts, and matching config paths; target-cluster
  verification remains.
- Dependency advisories identified during QA were upgraded to fixed module
  versions; the post-upgrade `govulncheck` scan reports no vulnerabilities.
- Release tooling is now reproducibility-bound: GoReleaser and Syft downloads
  are version- and SHA-256-pinned; Buf and sbom-tools are pinned the same way;
  runtime bases are digest-pinned, workflow actions are commit-pinned, and
  critical Trivy findings fail the image job. Cosign and slsa-verifier are
  digest-pinned in the Dagger module, release archives and checksums.txt are
  cosign-signed, and `VerifyPublishedRelease` verifies a published release
  from a clean environment.

### P2 — post-beta improvements

- External transactional database and horizontal scaling.
- Workspace/tenant administration and scoped authorization.
- Background verification queues and provider retries.
- Evidence retention, garbage collection, and legal-hold workflows.
- Search/indexing, saved review views, and richer lineage visualization.

## Beta acceptance checklist

### Build and correctness

- [x] `go test ./...`
- [x] `go test -race ./server/...`
- [x] `go build ./...`
- [x] `go vet ./...`
- [x] `buf lint`
- [x] `buf format -d --exit-code`
- [ ] Current exact-tree Dagger `all` export passes, including security and
      OSCAL gates. The last complete export was `final-15`; a post-fix rerun
      reached all application/security branches but its export stalled while
      warming the isolated scanner tool cache and was canceled. An exact
      repository-pinned `v0.21.7` engine run now reaches module type
      generation, but the engine container cannot resolve
      `proxy.golang.org` (`dial udp ...:53: connection refused`) while syncing
      Dagger's Go type-definition dependencies; no repository function
      executes. The Dagger Go module itself compiles, so this remains a local
      engine-network gate rather than a passing current-tree Dagger result.
- [x] Security advisory output is reviewed; the post-upgrade scan reports no
      vulnerabilities.
- [x] Non-generated Go sources comply with the 500-line hard limit; the
      repository-wide source audit confirms no non-generated `server/` or
      `dagger/` Go file exceeds 500 lines after the responsibility splits.

### Live product

- [x] Authenticated operator reaches the live review route: an ephemeral
      TLS-enabled token deployment rejected unauthenticated gateway health
      (`401`), served authenticated health (`200`, `{"status":"ok"}`),
      returned an authenticated claims response (`200`), passed CORS
      preflight (`204`), and served `/review.html` (`200`).
- [x] Valid evidence import stores and re-hashes exact bytes in the live API
      smoke; the browser import path also hashes the submitted bytes locally.
- [x] Invalid digest, size mismatch, duplicate, and missing evidence errors
      are actionable: the Live Review form now routes missing fields through
      its explicit status/receipt path; live duplicate import diagnostics pass;
      digest/size rejection diagnostics were exercised through a controlled
      local API responder using the real rendered route.
- [x] Provider-unavailable verification is visibly incomplete, exposes the
      missing provider diagnostics, and leaves receipt export disabled.
- [x] Graph explanation shows the same claim, evidence, proof, and trust state
      as the exchange service in the live review route.
- [x] Graph projections return a hash-chained audit event; projection history
      is exposed only after the global graph audit chain verifies.
- [x] Export is blocked until all required proof objects exist and a verified
      claim receipt is bound to the audit-chain head.

### Operations and release

- [x] `xoscal-backup` snapshot is copied off-PVC and hash-checked in a
      disposable kind PVC drill.
- [x] Restore drill succeeds on a disposable kind PVC, including integrity,
      fixture, ownership, and server readiness checks.
- [x] TLS and token secrets are mounted read-only from Kubernetes Secrets and
      checked by `go test ./server/internal/k8scontract`.
- [ ] Image, archives, SBOM, signatures, and provenance verify from clean
      tooling for the current proposed revision. The existing public `v0.1.3`
      release is verified separately below but predates the current dirty tree.
- [x] Generated site export and browser smoke pass.
- [x] `docs/BETA-RELEASE-NOTES.md` states the single-workspace and review-only
      trust boundary.

### Local QA evidence snapshot

- `go test ./...`, `go test -race ./server/...`, `go build ./...`, `go vet
  ./...`, `go mod verify`, Buf lint/format, design/a11y lint, and site smoke
  pass in the current checkout.
- `go test ./server/internal/k8scontract` verifies that the beta Deployment
  keeps one SQLite writer and that the TLS/token Secret mounts match the
  required fail-closed server configuration.
- `make site-verify` passes against the generated `_site` export, including the
  9-route smoke surface.
- `dagger call all --source=. export --path=/tmp/xoscal-dagger-all-beta-final-15`
  passed before the final deterministic-ID and scanner-pinning fixes. Its
  exported SARIF contained `gosec_results=0`; `govulncheck` reported no
  vulnerabilities; protobuf stability, spec-registry, OSCAL schema, and
  Metaschema checks passed.
- Current exact-toolchain local security evidence is green: `make security`
  uses Go `1.25.13`, `govulncheck` module `v1.7.0`, and `gosec` `v2.28.0`;
  govulncheck reports no vulnerabilities and the SARIF report contains
  `gosec_results=0`.
- Release reproducibility hardening is locally syntax-checked: the Dagger
  module compiles, both CI/release workflow files pass YAML parsing and
  `actionlint`, GoReleaser/Syft/runtime image inputs are pinned, and the
  release image scan is configured to fail on critical/high findings.
- The exact repository-pinned Dagger `v0.21.7` engine was exercised after the
  desktop `v0.21.8` mismatch was removed. It failed before function execution
  because the engine's type-definition sync could not resolve
  `proxy.golang.org`; this is recorded as an environment/network failure, not
  as Dagger gate evidence for the current tree.
- A non-publishing snapshot using the pinned GoReleaser `v2.17.1` and Syft
  `v1.51.0` completed twice from this checkout. All four archive checksums and
  four SPDX SBOMs validated, archive contents were checked, metadata matched
  commit `101203be2cea0c1d77e94ce5af593551aa01cba0`, and the Linux x86_64
  archive reproduced at
  `sha256:e6d8bb3590b153a5441115e9d4281671de767e6e234c0c913eb0fd56e5eafce2`.
- External baseline evidence exists for the public
  [v0.1.3 release](https://github.com/ckodex-labs/ckodex-xoscal/releases/tag/v0.1.3):
  selected archive and SBOM checksums verified, an SDK blob signature verified
  against its embedded Fulcio certificate, and the corresponding
  [release workflow](https://github.com/ckodex-labs/ckodex-xoscal/actions/runs/31348794697)
  completed its SLSA verification job successfully. This is evidence for the
  published `v0.1.3` commit, not for the current uncommitted revision; a new
  release must repeat the clean-environment verification.
- The graph retry regression is covered by a focused test: an omitted edge ID
  resolves to the same content-stable identifier on retry, which returns the
  expected idempotency conflict instead of creating a second edge.
- Live local gateway smoke returns health `ok`; the bounded batch API completed
  preflight, commit, and idempotent replay for a valid record. The browser
  surface is available at `/review.html?api=http://127.0.0.1:58081` during QA.
- An isolated TLS/token gateway smoke verified the network beta boundary: the
  unauthenticated health request returned `401`, the authenticated health and
  claims requests returned `200`, CORS preflight returned `204`, and the real
  `/review.html` route returned `200` against the same static site.
- Live Review browser evidence covers missing-field validation, duplicate
  import diagnostics, digest/size rejection rendering, 44px interaction
  targets, mobile no-overflow, forced-colors opacity, and proof-gated receipt
  export. The full assistive-technology matrix remains a human sign-off gate.
- Live browser QA on the review route verified an actionable duplicate-ID
  preflight failure, provider-unavailable diagnostics, disabled receipt export
  for an incomplete claim, unique IDs for added batch records, and the hard
  ten-record UI ceiling. The route had no captured warning/error console
  entries, no horizontal overflow at 390×844, and all visible buttons and
  fields measured at least 44 CSS pixels high. Ledger primary-action contrast
  measured 4.99:1 after the foreground correction.

The checked restore items are local QA evidence, not production-storage
acceptance. The deployment owner must repeat the drill against the intended
Kubernetes storage class and attach the backup digest, integrity result,
fixture result, runtime ownership, and readiness output to the release bundle.

## Release decision

The beta may be promoted only when every P0 item has direct evidence. Green
unit tests alone are insufficient. If the providers or production provenance
are not available, ship only a clearly labelled review preview with no
attestation or sealed-export claims.
