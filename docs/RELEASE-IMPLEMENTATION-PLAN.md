# Evidence-gated artifact and portal implementation

Baseline: `fcb55186083de069bc386757c5fd0d3f1709627d`, reconciled with remote main on 2026-10-05.
Branch: `fix/evidence-gated-release`. The original MNC-MBP checkout is preserved.
Status: implementation and complete unsigned candidate validation passed; hosted signing and promotion remain unrun.

The independent dependency remediation is commit
`27422c93158aaf19b05ae230414e8e0b67a18051`. The integrated production repair
`74f3876784e720f76b2be30202b4bf95bb769b34` passed the complete Dagger candidate:
555 inventoried files, 479 mandatory outputs, ten check markers and 28 analysis
receipts. All 247 Python contract tests passed. Exact-byte integrity and final
presentation replay passed, including 63 HTTP assets and 96 API operations.
Six SDK consumers and four platform archives passed; browser inspection passed
on desktop/mobile with keyboard interaction and no observed external resources.
Earlier environmental failures and blocked findings remain recorded. See
`docs/RELEASE-VALIDATION-HANDOFF.md` for precise evidence scope and remaining
hosted-only boundaries. Local validation does not establish release signing.

## Outcome and authority

Dagger produces and analyzes a complete candidate before publication. Hosted Actions supplies
its existing OIDC attestation identity, verifies the exact candidate bytes, then permits
promotion. A static portal presents one pinned inventory. Missing, unsupported, failed,
or unverified evidence is incomplete and disables download eligibility. The Design Blueprint
remains illustrative and cannot supply a live trust decision.

Authorized: code/document changes, local tests, isolated commits, and a draft PR where existing
access allows it. Excluded: merge, release execution, production deployment, credentials,
permissions, and new signing identities.

## Current and target execution graphs

Current: main site rebuild -> latest-release API merge -> static favorable fallback;
GoReleaser publish -> later analysis/attestation; image push -> later vulnerability scan.
SDK generation creates source ZIPs while the site claims compiled packages. OpenAPI copies
an arbitrary first file. Findings can exist without enforcing promotion policy.

Target dependencies:

1. pinned source -> proto generation -> complete API + package builds;
2. pinned source -> binaries/archives + image archive + catalogs/profiles;
3. exact outputs -> raw scans/SBOMs/schema checks/install smoke -> enforcing analysis gates;
4. complete candidate -> payload manifest -> existing hosted attestation;
5. payload attestation + exact-byte admission -> verified static rendering and presentation analysis;
6. frozen final payload/site/evidence -> final manifest + transport + OCI root -> hosted attestation;
7. verified final subjects -> immutable registry/release promotion -> same bytes delivered to Pages.

No post-signing asset rewriting. Detached final proofs do not hash themselves.
The payload manifest binds production outputs; final rendering occurs only after
its actual provenance verification. The final manifest then binds payload proof,
verification evidence and every presentation byte before hosted final signing.
The final manifest excludes itself and exactly two detached final proof paths.
Identity signing stays at the existing hosted trust boundary. Dagger owns
production, policy, validation, rendering and exact-byte verification. Actions
owns orchestration, existing OIDC identity and promotion. Read-only destination
preflight refuses existing release/registry tags; no automatic overwrite. Pages
is staged and deployed in the release workflow after promotion, with separate
manual verified retrieval retained for operators.

## Phase 0: isolation and contracts

Files: this plan, `docs/RELEASE-CONTRACT.md`, `ORCHESTRATOR.md`, `dagger/identity.go`,
`scripts/release-source.py` and source-isolation tests.
Read the beta contract, portal Pipeline/download manifesto and historical design. Record
current main, clean original checkout and tools. Define typed inventory: release version,
source revision, artifact kind/path/size/SHA-256, producer, OSCAL touchpoint, analysis refs,
proof refs, eligibility. Reject unsafe paths, duplicates, missing outputs and unknown states.
Acceptance: original checkout untouched; candidate/proof boundary documented and executable.
Production producer inputs must come from the validated tracked commit, excluding caller
ignored files, workspaces and injected site assets. Reconstruct safe Git metadata without
inheriting credential headers or hooks. Source identity tests cover both dirty tracked
inputs and ignored-file injection; Go builders disable workspace and implicit vendor use.

## Phase 1: independent dependency remediation

Files: root and `dagger/` `go.mod`/`go.sum`, `dagger.json`,
`dagger/secure-go-sdk`, `dagger/third_party/otel-go`, dependency provenance documentation.
Verify GO-2026-6505 / GHSA-8wmf-6v46-5gfg against the Go vulnerability database and upstream
OTel advisory. Upgrade compatible SDK/exporter families in both modules to fixed releases.
Do not describe the application as leaking endpoints: stdout logging, tracing defaults and
logger configuration must be assessed separately. Keep this change in a separate commit.
Tests: module tidy, root/nested compile and tests, dependency inventory, govulncheck.
Acceptance: affected SDK versions absent from both module graphs; record any other advisory.
Engine 0.21.7 injects obsolete log-family replacements into its builtin Go runtime. The
reviewed wrapper delegates static type generation, then admits the actual fixed runtime
graph, compiles with pinned Go 1.27.1 and enforces tests/vet/vulnerability results before
use. A narrow attributed compatibility fork preserves the fixed upstream family's API.
JSON-mode scanner exit success never substitutes for inspecting called findings.

## Phase 2: deterministic API contract

Files: `dagger/openapi.go`, `server/cmd/xoscal-openapi/`,
`proto/oscal/services/v1/governance_service.proto` and generated outputs,
`site/openapi.json`, `site/docs.html`, gateway route/request contract tests.
Generate one merged JSON document containing operations from OSCAL, governance, transparency
exchange and transparency graph services; preserve references and operation identities.
Fail on duplicate or missing required operations and empty paths. Use service annotations as
authority. Fix component-definition, claim and receipt routes and gRPC/metrics port labels.
Tests: JSON parse, service/operation coverage, reference resolution, known route assertions,
generation drift, Scalar served with local assets. Acceptance: Scalar exposes operations
for all four services; never copy YAML bytes to a `.json` filename.

## Phase 3: package production and consumer tests

Files: `dagger/sdk.go`, `scripts/sdk-package.py`, `scripts/sdk-buf.gen.yaml`,
`scripts/sdk-consumers/`, `scripts/sdk-verify.py`, package/verifier negative tests,
`site/downloads.html`, `site/cli.html`, `scripts/install.sh`, `scripts/test-install.py`. Split existing SdkBundles ownership
out of main. Preserve source ZIP compatibility and include package metadata in each ZIP.
Produce Go module, Python wheel/sdist, Java Maven JAR/POM, C# NuGet, typed Node ESM/CJS,
and Swift Package Manager where the pinned generated clients support the promised API.
Generation, packaging and isolated consumer compilation/import/install run in Dagger.
Do not fabricate Pydantic, Jackson, Swift gRPC compatibility or universal macOS binaries.
If a promise is incompatible, record the concrete limitation and decision in the contract;
publish the precise supported format and keep unsupported formats ineligible.
Installer accepts a pinned candidate/manifest, checks exact archive and provenance before
install, handles supported OS/arch, and never defaults to mutable latest or unsigned fallback.
Tests: clean consumers of actual packages, ZIP traversal rejection, missing artifact failure,
installer dry-run/help, tampered archive/proof failure, documented commands.
Acceptance: advertised formats match tested outputs; six language source bundles remain.

## Phase 4: enforcing analysis and complete inventory

Files: `dagger/artifacts.go`, `dagger/analysis.go`, `dagger/candidate.go`,
`dagger/frontend.go`, `dagger/frontend_inventory.py`, frozen frontend package inputs,
`dagger/enrich_sbom.py`, `dagger/publisher_metadata.py`, `dagger/extract_archive.py`,
`scripts/analysis-gate.py`, `scripts/release_inventory.py`, `scripts/release_requirements.py`,
`scripts/inspect-image-archive.py`, `dagger/all.go`, framework schema/constraint validation,
`docs/release-policy.json`, `docs/analysis-exceptions.json`, `docs/SBOM-DISTRIBUTOR.json`
and their meaningful negative tests.
Preserve raw gosec and vulnerability/SBOM findings. Separate report collection from admission;
nonzero scanner operational errors fail; existing critical vulnerability thresholds stay.
SBOM violations require explicit versioned exceptions, never report-exists success.
Validate profiles as well as catalogs. Inventory all 35 catalogs, PBMM profile, API/schema,
six SDKs/packages, binaries, SBOMs, scans, install script, site and vendor assets.
Tests: failing and passing real gate inputs, malformed/missing reports, expired exceptions,
wrong digest, duplicate paths, missing profile, required producer/OSCAL references.
Acceptance: every file has inventory/manifest coverage or belongs to the detached proof
envelope; incomplete mandatory evidence prevents promotion.

The browser bundle is an analyzed release artifact. Rebuild the pinned Scalar published
ESM modules with an exact npm lock and fixed compatible dependency overrides. Keep the
actual build input manifests, bundler input graph, lock, output source map, and original
publisher metadata. Derive the runtime SBOM from modules included in the emitted bundle,
not unused peer dependency names or the whole build environment. Scan that inventory,
retain all findings, enforce the existing release threshold, and replay its binding to
the exact served JavaScript before final presentation admission. Do not claim the rebuilt
bundle has the publisher's original standalone digest. Tests include missing, altered,
unexplained, or substituted inputs and served bytes, plus actual Scalar rendering and
request configuration with optional external Agent/telemetry features disabled.

## Phase 5: static, honest portal rendering

Files: `dagger/site.go`, build renderer, `site/downloads.html`, `site/transparency.html`,
`site/sbom-validation.html`, `site/portal.html`, `portal-preview.html`, README navigation.
Remove latest-release fetch/merge and favorable fallback. Build HTML from the pinned
candidate inventory and supplied verified evidence; local preview is explicitly unsigned
and ineligible. Correct initial HTML filenames and profile visibility. Render actual SBOM
counts or incomplete state. No JavaScript needed to repair links or trust state. Checked-in data indices start empty
and incomplete; stale historical release URLs and unbound assessment data are removed. Proof
objects and successful validation are required for attested coloring. Blueprint pipeline
cards describe current producer/signing boundaries and remain marked illustrative.
Tests: no-JS DOM, fetch failure, missing proof, tampered proof, pinned links, digest integrity,
HTML escaping, local Scalar/fonts/CSS/CLI assets; desktop/mobile and keyboard navigation.
Acceptance: source defaults and generated HTML never assert favorable unverified evidence.

## Phase 6: stage, attest, verify, promote

Files: `dagger/release.go`, `dagger/candidate.go`, `dagger/bundle.go`,
`scripts/release-contract.py`, `scripts/attestation_policy.py`, `scripts/verify-transport.py`,
`scripts/package-candidate.py`, `scripts/download-release.py`,
`scripts/promotion-assets.py`, `scripts/promotion-preflight.py`, `.goreleaser.yaml`,
`.github/workflows/release.yml`, `.github/workflows/pages.yml`, CI contract gates.
Use GoReleaser without publication; image export/scan before registry consumer publication.
One candidate contains all outputs and final site. Reuse current GitHub attestation action
and verify expected repo/workflow/source identities and subjects with GitHub CLI. Cover
profiles, package outputs, evidence and static site; do not substitute configured signing
for verification. Artifact upload is staging, public release/registry tags/Pages are promotion.
Pages consumes the promoted pinned site bundle instead of regenerating from main. Avoid
requiring new persistent identities. Never run this workflow during this task.
Tests: workflow YAML/static analysis, dry candidate build, all contract/negative tests,
unchanged hash verification after proof generation and before promotion.
CI also invokes Dagger's complete Site producer and retains raw frontend analysis;
this exercises the local browser assembly and package consumers before release work.
Acceptance: any required gate failure leaves no public release/image/site update; no
unsigned fallback; hosted exact-byte verification remains a required unexercised boundary.

## Phase 7: integration and handoff

Run repository ladder: gofmt, buf lint/format/drift, Go tests/race/vet/build, design/a11y lint,
Dagger All, full Site export with engine 0.21.7, candidate export, package consumer tests,
served site smoke and browser inspection. Keep actual logs under task evidence, distinguishing
passed/failed/not-run/hosted-only. Independent review focuses on proof forgery, self-reference,
post-sign mutation, hidden fallback, path handling and workflow promotion sequencing.
Prepare separate dependency commit and coherent repair commits; draft PR only if existing
authorized access works. Return exact head, diff, plan, evidence and remaining gates.

## Migration, rollback and Day-2

Migration: existing source ZIP paths retained; downloads index adds typed fields; old releases
remain historical and cannot be silently marked verified. New workflow publishes the first
complete candidate only after hosted gates pass. Pages follows the exact promoted release.
Local preview remains possible with incomplete evidence clearly labeled.
Rollback: revert repair commits before execution. After an approved future promotion, select
the previous immutable, independently verified site/release digest; never rebuild rollback
bytes from current main. Failed candidates remain staged with raw findings for diagnosis.
Do not replace, overwrite or delete prior release assets in an automatic recovery path.

## Risks and unresolved boundaries

Network/plugin/tool availability can block generation; record exact failure and keep outputs
ineligible. Strict analysis may expose existing failures; remediate or document policy-approved
exceptions without lowering gates. Six SDK runtime ecosystems may require metadata/import
repairs and pinned compatible runtimes. Large catalogs and redundant attestations can increase
CI time. No cryptographic success can be asserted until the existing hosted identity workflow
runs and verifies the exact final subjects. GitHub access may block draft PR delivery.
Native package timestamps and metadata identifiers can vary across repeated builds. All
consumer and scan receipts must bind one candidate's actual emitted bytes; matching versions
do not authorize reusing another build's hashes or reports.

The released NIST CLI implements OSCAL 1.1.2 semantic bindings while generated artifacts
declare the separately validated 1.2.3 structural contract. The reviewed catalog/profile
scope gate rejects changed semantic constructs, records actual subject hashes and replays
at admission. This is an explicit restriction of these 36 framework outputs, not a claim
of general 1.2.3 semantic coverage. Details and concrete residual-error probes are in
`docs/OSCAL-SEMANTIC-COVERAGE.md`; unsupported future constructs fail closed.

Registry, GitHub Releases and Pages do not share an atomic transaction. All analysis and
verification precede the first promotion, but a later provider failure can leave an already
verified image or release visible while Pages is still pending. Retain the exact admitted
candidate, inspect provider state and resume only the missing promotion under operator
control; the automatic preflight deliberately refuses overwriting an existing identity.
This task exercises local production and admission without invoking any promotion.
