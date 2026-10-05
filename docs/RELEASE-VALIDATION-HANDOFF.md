# Release and portal implementation handoff

Validated implementation revision: `74f3876784e720f76b2be30202b4bf95bb769b34`.
Baseline and reconciled main: `fcb55186083de069bc386757c5fd0d3f1709627d`.
Branch: `fix/evidence-gated-release`. Host: MNC-MBP. Date: 2026-10-05 UTC.
The final delivery commit updates documentation only after these checks.
The original checkout was preserved and remains clean.

## Implemented behavior

Dagger owns complete four-service OpenAPI generation, six SDK distribution
families and clean consumers, four platform CLI archives, both static image
platforms, 35 catalogs and the PBMM profile, analysis and exact-byte admission.
The portal receives one pinned inventory with producer and OSCAL touchpoints.
Initial HTML and no-JavaScript downloads derive from real evidence. Missing or
unverified proof keeps download eligibility disabled; favorable fetch fallbacks
and historical hardcoded signing/SBOM claims are removed.

The existing hosted GitHub identity signs the payload manifest, final manifest,
transport and actual OCI root bytes. Exact verification precedes immutable
release/registry/Pages promotion. Dagger never invents a signing identity.
No publication is performed before admission and no final bytes change after
signing. Detached final proofs avoid self-reference. Failure diagnostics retain
independent raw exports and invocation statuses rather than concealing findings.
Failed verifier output is retained with credential redaction.

The OpenTelemetry fix is independent commit
`27422c93158aaf19b05ae230414e8e0b67a18051`. Both module graphs and the custom
engine runtime admit the fixed 1.45 family. The logger advisory is not evidence
that the default server actually leaked endpoints.

## Actual passed evidence

| Check | Observed result |
| --- | --- |
| Complete unsigned Dagger candidate, engine 0.21.7 | Exit 0; clean committed input; `/tmp/xoscal-preview-candidate-final-v7` |
| Required release-subject replay | 555 inventoried files, 479 mandatory outputs, 12 byte-bound SDK consumer subjects |
| Dagger All within candidate | lint, tests, race, proto drift, spec registry, schema, constraints, OpenAPI, contracts and module-tidy markers passed |
| Python contracts | 247 passed on host and in Dagger; installer and package tests also passed |
| SDK consumers | Go, Python wheel/sdist, Java JAR/POM, C# NuGet, typed ESM/CJS npm and Swift SPM models passed |
| Exact artifact analysis | Six SDK bundles and four platform archives passed; both raw Trivy modes retained and merge replayed; all three binaries identified per archive |
| Image platforms and strict SBOMs | Both image platforms passed; all 28 analysis receipts passed; 14 strict SBOM receipts have zero violations and exceptions |
| Source security | No called govulncheck vulnerabilities or blocking HIGH/HIGH gosec findings; 38 raw gosec findings and 25 source suppressions remain visible |
| Frameworks | 35 catalogs plus PBMM profile passed structural and restricted semantic admission with exact-byte scope receipts |
| Frontend | All 185 observed runtime packages scanned; zero findings; exact matching strict SBOM passed |
| Final presentation | Exact-byte replay, capacity, design, accessibility and HTTP smoke passed; 63 served assets and 96 operations |
| Browser | Desktop 1280/mobile 390, keyboard dialog and real local GET 200, no observed external resources, unsigned no-JS downloads disabled |
| Native local CLI smoke | Actual final Darwin arm64 and Linux arm64 ctl/backup version and server help commands passed; no service started |
| Invalid proof | Real pinned gh verifier rejected malformed media type with exit 1; no verified candidate exported |
| Workflow checks | actionlint passed for CI/release/Pages; gofmt and whitespace checks passed |

The browser-tested five frontend assets and OpenAPI are byte-identical to the
final candidate. Its JS SHA-256 is
`5158274d074de2230d0b22622d6c6d0afbe71d8949fa0c78f82210768f813798`;
map SHA-256 is `99cb4bbed466b897fef347370bd0b6b13b694b58e5fe28b765afef8f30884c43`.
Frontend BOM and scan receipts are taken from the same candidate, never mixed
with a nominally equivalent earlier build. Artifact/SBOM timestamps can differ
across builds; exact subject hashes are authoritative.

Full logs: `/tmp/xoscal-preview-candidate-final-v7.log`,
`/tmp/xoscal-preview-candidate-final-v7-presentation.log`,
`/tmp/xoscal-final-invalid-proof-probe-v3.log`,
`/tmp/xoscal-python-final-verifier-authorized.log`, and
`/tmp/xoscal-site-browser-final-v5.md`.
Portable summaries, the unsigned payload manifest, consumer receipts and checksums
are retained in the task's `verification/` directory. Earlier actual failures,
including the original no-target artifact scan and frontend/base-image findings,
remain recorded in `docs/ANALYSIS-VALIDATION.md` and local raw reports.
A host run without loopback permission failed two HTTP tests; the authorized run
passed all 247. No check was skipped or weakened to obtain that result.

## Unrun boundaries and scope decisions

This candidate is explicitly unsigned and ineligible. Successful hosted OIDC
signing and final cryptographic verification, registry/release promotion and
Pages deployment were not run. No tag/release was cut, no merge/deployment was
performed, and no credentials, permissions or identities were changed. Those
future hosted checks remain mandatory; local verification is not a signing claim.

Python supplies protobuf models rather than Pydantic models. Swift supplies a
portable tested SPM model package without an Apple transport claim. The exact
compatibility decisions and requirements for expanding those promises are in
`docs/SDK-CONTRACT.md`. No package registry publication was performed.

The released NIST CLI's semantic bindings are older than the structural schema;
`docs/OSCAL-SEMANTIC-COVERAGE.md` records the tested restricted catalog/profile
scope and fail-closed guard. This does not claim unrestricted OSCAL 1.2.3 semantic
validation. The optional LanceDB/Rust native image was not fully compiled or
release-analyzed; the supported release path remains static. Docker structural
checks and native-base verification do not substitute for that build.

## Review and rollback

Review the implementation plan, release and SDK contracts, analysis policy,
semantic coverage and workflow identity predicates before allowing a hosted run.
Revert repair commits before execution if needed. A future rollback must select
previous immutable verified bytes, without rebuilding or overwriting identities.
GitHub Releases, the registry and Pages are not one atomic provider transaction;
operator recovery resumes only the missing promotion from the retained verified
candidate. The task does not perform any promotion or production cluster action.
