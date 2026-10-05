# Blueprint pipeline and proof contract

The Pipeline view in `site/portal.html` and `portal-preview.html` describes a
release contract. It is explicitly illustrative and contains no signing run
evidence. Function and workflow cards display `requires hosted proof`; a
configuration object must never carry a `signed: true` result. Other illustrative
governance examples remain examples, not release admission inputs.

## Ordered release graph

1. **Pinned source**: an explicit release tag and full source revision enter the
   graph. Dagger owns deterministic production, tests, package consumer checks,
   scans, schema validation and admission policy.
2. **ReleaseCandidate**: stage all six SDK source bundles and supported tested
   package formats, 35 framework catalogs, the PBMM profile, Linux/macOS platform
   archives, a multi-architecture OCI archive, API/schema assets, installer and
   raw analysis reports. GoReleaser skips publication, announcement and signing.
   Dagger records producer, OSCAL touchpoint, source, size and digest for every
   output in the initial payload manifest. Failed mandatory analysis stops here.
3. **Hosted payload attestation**: the existing GitHub
   `actions/attest-build-provenance` identity in `.github/workflows/release.yml`
   attests the exact payload manifest. Workflow configuration is not evidence
   that this event has happened or that verification succeeded.
4. **FinalizeCandidate**: Dagger checks the exact payload manifest and its expected
   repository, workflow, tag and source identity, then checks its covered bytes.
   Only verified payload evidence drives the final build-time portal rendering.
   Missing data renders incomplete and blocks eligibility. No mutable latest
   release merge or JavaScript trust upgrade is permitted.
5. **Final release manifest**: after rendering and final site checks, Dagger
   creates `release-manifest.json` covering every final payload and site byte.
   That manifest does not hash itself or its detached proof envelope. The final
   site is frozen at this boundary.
6. **Hosted final attestation**: the existing release workflow identity attests
   the exact final manifest. Store the detached bundle and verification receipts
   outside their own digest coverage, under `proofs/`.
7. **VerifyCandidate**: Dagger verifies expected hosted identity and all exact
   final manifest/file digests. Missing proof, scanner failure, invalid required
   findings, changed bytes or an unexpected source/workflow must fail promotion.
8. **Promotion**: publish the unchanged verified GitHub assets and OCI bytes,
   and pass the same final static site to Pages. Pages retrieves the pinned
   promoted candidate and checks it before publication; it does not regenerate
   release files from main. No unsigned fallback or post-signing site mutation.

The hosted identity remains at its legitimate trust boundary. No new credential,
permission, persistent signer or production deployment is introduced by this
contract. Local tests, source configuration and a SHA-256 digest do not establish
hosted signing, SLSA level, or successful release promotion.

## Producer and OSCAL mapping

| Producer | Output | OSCAL touchpoint | Required admission |
| --- | --- | --- | --- |
| SDK generation/package stages | Six source ZIPs and precise tested package formats | Software component, back-matter resource | Real generation plus clean consumers; model-only formats are labeled accurately |
| Framework export | 35 catalogs and PBMM profile | Catalog, profile | 1.2.3 JSON Schema, byte-bound reviewed legacy-semantic scope and strict CLI checks for both kinds; see OSCAL-SEMANTIC-COVERAGE.md |
| Frontend assembly/analysis | Rebuilt Scalar JS/CSS/maps/notices, observed runtime SBOM and raw reports | Component-definition, assessment-results | Reviewed frozen package inputs, replayed emitted-source inventory, exact served bytes, vulnerability and strict SBOM admission |
| Four-service API producer | Complete deterministic OpenAPI JSON | Service interface, back-matter resource | Operation coverage, references and served Scalar contract |
| Release archive producer | Per-platform tar.gz files and SBOMs | Software component, back-matter resource | Build, analysis, signed manifest coverage and install checks |
| Image producer/analysis | OCI archive, image SBOM and raw findings | Component-definition, assessment-results | Enforced vulnerability/SBOM policy before consumer publication |
| Security/SBOM gates | Raw reports, findings and admission decisions | Observation, finding, risk | Operational errors fail; findings need policy-approved valid exceptions |
| FinalizeCandidate | Static HTML, local assets and pinned inventory | Evidence presentation, back-matter resource | Verified payload; no-JS eligibility and final site checks |
| Hosted Actions + VerifyCandidate | Detached attestations and exact-byte results | Attestation, checked requirement | Expected identity, source/tag and every covered digest |

The installer contract is in [INSTALL-CONTRACT.md](INSTALL-CONTRACT.md). The
comprehensive implementation phases and migration/rollback controls are in
[RELEASE-IMPLEMENTATION-PLAN.md](RELEASE-IMPLEMENTATION-PLAN.md).

## Blueprint acceptance

The two Pipeline views use identical declared flow, functions, workflows and
trace mappings. Assertions must inspect that bounded section: no stale signing
engine or self-reported signed boolean; GoReleaser stages without publication;
35 catalogs plus the profile; both hosted attestation steps; FinalizeCandidate
before final manifest signing; VerifyCandidate before promotion; Pages uses
the unchanged promoted bytes. Pipeline colors stay neutral while proof is absent.
Other explicitly illustrative governance cards are not rewritten into release
claims. Static design/accessibility lint and JavaScript syntax checks accompany
the diff; rendered desktop/mobile/keyboard checks use the assembled final site.

This change does not execute a release, deploy Pages, publish a container,
create credentials, or establish hosted cryptographic success. Those remain
required hosted gates over the actual future release bytes.
