# Exact-byte release contract

Status: implemented contract; hosted signatures and promotion are not exercised by local tests.
Plan: [RELEASE-IMPLEMENTATION-PLAN.md](RELEASE-IMPLEMENTATION-PLAN.md).

Dagger owns generation, package consumers, analysis, admission, manifests and rendering.
Actions retains the repository's existing hosted OIDC identity for GitHub artifact attestations
and the existing GitHub/registry/Pages publication boundaries. No new signing identity,
credential or registry publication is introduced by local execution.

## Immutable subjects and stages

1. `ReleaseCandidate` produces all payloads and enforces analysis. It creates
   `payload-manifest.json` with exact SHA-256, size, kind, producer, source revision,
   platform, OSCAL touchpoint and attachment identity. It is unsigned and ineligible.
2. Hosted `actions/attest-build-provenance` signs that manifest. This covers each payload
   transitively through its exact digest; a SHA sidecar alone is not a signature.
3. `FinalizeCandidate` validates every covered file, then runs `gh attestation verify`
   against the manifest and supplied detached bundle. Verification enforces repository,
   `.github/workflows/release.yml`, source SHA, `refs/tags/<tag>` and hosted runners.
   Only that successful cryptographic verification allows download links in generated HTML.
4. `PresentationAnalysis` enforces static contracts and the pinned Dagger
   browser producer against the final rendered candidate. All eight pages and
   the home alias run in fresh desktop/mobile JavaScript and no-JavaScript
   contexts. Mandatory checks include full prebuilt API content, repeated
   dialog/keyboard lifecycles, unique landmarks/IDs, themes, native content,
   geometry/occlusion, actual keyboard view-entry focus and fourteen enhancement
   controls. The controls cover missing/tampered contracts, failed scripts,
   returning focus to Plain during loading, cancelling loading and forward
   Tab to the raw OpenAPI host link during loading and failure after that
   forward focus movement on desktop and mobile.
   Cold controls hold an actual exact-byte contract response until the observed
   key. Returning to Plain/cancelling permits only that contract's intentional
   abort; forward Tab requires the exact successful GET and real readiness
   while preserving the newly focused host link. A controlled missing-contract
   response after host focus movement must return visible focus to Plain;
   only that response's exact correlated abort is allowed. Independent replay
   enforces finite in-viewport focus/heading
   rectangles and actual loading key observations. Browser
   tooling stays in the analysis container. Raw observations and exact
   producer/toolchain sources are staged as six final-only evidence files;
   independent admission validates coverage and rehashes their presentation
   subjects. A saved success boolean alone is insufficient, and this receipt
   is not signing authority. The upgraded consumer refuses historical receipts
   missing the fourteen controls. Historical evidence can be replayed with its
   reviewed version-bound consumer, but cannot authorize promotion under this
   stronger contract; new releases require newly produced evidence.
   Rendering finishes before `release-manifest.json` inventories all final payload,
   evidence, HTML, CSS, JavaScript, fonts and other assets. `PackageCandidate` creates a
   deterministic transport archive of those frozen bytes. Hosted provenance signs the
   final manifest, transport and exact OCI root manifest bytes.
5. `VerifyCandidate` checks final provenance and rehashes every file. `VerifyTransport`
   verifies the signed transport, runs the actual public consumer's strict extraction,
   checks the embedded final manifest identity and every file, then verifies that manifest's
   signature and required raw-evidence policy. Both gates run before staging and again
   after transfer, before any registry/release/Pages promotion. A signature on an archive
   never substitutes for successful consumer admission. The producer emits regular file
   headers even when Dagger materializes identical files with shared inodes; filesystem
   links remain forbidden by the consumer. Promotion requires both gates to pass.
   It reconstructs flat attachments from that admitted candidate, rather than publishing
   unchecked transferred attachments; only the separately verified transport is added.
   Registry copy preserves OCI digests and compares the published root bytes. Read-only preflight refuses any existing release or image tag and fails when absence cannot be established; automatic recovery never overwrites them.
6. The release workflow stages admitted Pages bytes after promotion and records
   an explicit deployment handoff. The protected Pages environment accepts the
   main ref; a release-tag job must not request that environment. Releases
   created with `GITHUB_TOKEN` do not trigger a second workflow. An authorized
   operator dispatches `pages.yml` on `main`, supplying the exact release tag and
   full expected source SHA. This separate Pages run must pass both verification
   and deployment before publication is described as complete. Pages retrieval
   selects one explicit published release and expected source SHA, verifies the
   transport before extraction and verifies the final manifest. It serves those bytes.
   It never merges main-generated assets with a mutable latest release.

The final manifest excludes itself and exactly two detached envelope paths:
`proofs/release-manifest.sigstore.json` and `proofs/verification.json`. Its own detached
signature covers the manifest's digest; the verifier receipt is diagnostic evidence and
is never itself accepted as signing authority. Payload proof bundles and verification
results used for rendering are covered by the final manifest. The transport is a separately
attested subject and cannot include its own signature recursively.

## Required assurance and honest presentation

Missing/unknown/unsupported/failed evidence remains incomplete. Neither `signed: true`,
a favorable source template, a successful fetch, nor workflow configuration establishes
signature validity. Production admission reruns policy against raw evidence and checks
consumer subjects and sidecars. Failures retain raw reports for review. Exceptions must be
explicit, versioned, digest/finding-bound, attributed and unexpired; the initial exception
policy is empty. Existing release critical/fixed and CI high/critical thresholds are retained.

The local `Site` preview contains real packages, catalogs/profiles, API docs and local assets,
but has no hosted signature and offers no eligible download links. Source templates display
incomplete states. SBOM findings are counts from actual raw assessment output, never sample
42/0 pass summaries. `portal.html` and `portal-preview.html` remain Design Blueprints whose
illustrative data and pipeline contract do not establish a live trust decision.

Blueprint HTML is generated from tracked templates and fixtures before
inventory. The API-reference HTML is generated from the actual four-service
OpenAPI contract. JavaScript enhances existing content; it does not create
release artifacts, required reference content, or download eligibility. Pinned
Scalar component transformations happen at the source build boundary, with
upstream preimages, a versioned recipe and replay-checked receipts retained as
mandatory frontend evidence.

The frozen payload may gain only the exact admitted presentation marker and
browser evidence files after payload signing. Their bytes are covered by the
final manifest. Browser reports exclude themselves, detached proofs and
manifests from their presentation subjects, avoiding recursive hashes. Local
previews enforce the same browser gate while keeping the raw report available
through `PresentationBrowserReports`; they remain unsigned and ineligible.

Package formats and explicit unsupported promises are described in [SDK-CONTRACT.md](SDK-CONTRACT.md).
Installer verification and platform support are in [INSTALL-CONTRACT.md](INSTALL-CONTRACT.md).
The semantic-search route migration is in [API-CONTRACT.md](API-CONTRACT.md).

## Promotion, migration and recovery

`Release` now stages GoReleaser output without credentials, publishing or signing. `Snapshot`
is a development producer. Only the hosted release workflow promotes after mandatory gates.
No unsigned fallback release is allowed. SDK compatibility ZIP names are retained, with
native packages and digested clean-consumer evidence alongside them. Profiles are first-class
inventory entries. Flat GitHub attachments use names declared in the signed manifest; the
static site retains typed local paths.

Native packages may retain build timestamps or generated package metadata identifiers.
Repeated builds do not promise bit-identical package archives. Consumer and analysis
receipts bind the exact emitted bytes within one candidate; matching versions never
authorize substituting a prior build or its reports.

Do not overwrite an existing release during automatic recovery. Failed candidates remain
staged with raw findings. Roll back by choosing a previous independently verified immutable
release/site subject; do not regenerate rollback artifacts from current main. Historical
releases lacking complete proof remain historical, not silently upgraded to verified state.

Hosted OIDC signing, registry publication and Pages deployment require actual
successful runs over the exact reviewed release. Local checks cannot establish
that those operations have run or supply their hosted signing authority.

Versioned admission policy, exceptions and the framework source manifest are
staged as exact signed evidence. Historical release verification replays those
bound policy inputs rather than substituting mutable main policy. Production
source identity requires a clean checkout, matching Git tag/commit and matching
producer version. Production tags must also fit OCI's ASCII tag grammar
`[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}`: standard SemVer prereleases are supported,
but SemVer build metadata (`+...`) and tags longer than 128 characters are rejected
before source derivation or artifact production; tags are never rewritten.
`PreviewCandidate` uses snapshot archives for full local graph
validation without creating a tag or establishing production/signature authority.
