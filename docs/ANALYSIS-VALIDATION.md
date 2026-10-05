# Local analysis validation

These results were obtained on MNC-MBP with Dagger 0.21.7 in an isolated
analysis test module copied from the repository. They verify production methods
and actual exported artifact bytes; they do not substitute for hosted release
identity attestations or authorize publication. The final release candidate must
run these methods over its own immutable artifact graph and replay all receipts.

The final source gate exported `/tmp/xoscal-source-security-final-frozen` with
38 retained gosec findings, 25 recorded in-source suppressions, no blocking
HIGH/HIGH finding, and no called govulncheck vulnerability. Lower findings remain
in raw JSON/SARIF. The two reviewed CLI G703 suppressions are local operator
output paths with specific source rationales and negative path-flow tests.

All six final aggregate SDK ZIPs passed the release vulnerability threshold
(CRITICAL with a fixed version) and strict SBOM validation with zero violations
and zero exceptions. Each receipt below was independently replayed using the
final repository policy. Local directories are under `/tmp/`.

| SDK | Exact ZIP SHA-256 | Strict SBOM evidence directory |
| --- | --- | --- |
| go | `ceadcb2028e7cac004033a9481feb0cb3b18dd7cb47dac4675edec12058c92c9` | `xoscal-sdk-aggregate-go-sbom-final` |
| python | `c85b8524d31e3b1c6e97f768a41aef480f1b251881c2ee1728560dec5e330ea3` | `xoscal-sdk-aggregate-python-sbom-final` |
| java | `e7dc48bab9e1440595d1a97a6f7139fd18cb0d42881065f9d59abc6ce9615510` | `xoscal-sdk-aggregate-java-sbom-final` |
| csharp | `8a35f38d0065f084821335b0a206e9a41538eba55fbc299d55bdae35a6077e82` | `xoscal-sdk-aggregate-csharp-sbom-final` |
| ts | `fd89082341fb38679ab911a101a13c02df28cb46038bb470503621d7b713ae54` | `xoscal-sdk-aggregate-ts-sbom-final-v2` |
| swift | `39a805fd4cc11c5a1023d779842e9ec32a2554201a452da13f93d4e1f50bca1c` | `xoscal-sdk-aggregate-swift-sbom-final` |

The scan receipts and their exact current ZIP bindings are recorded in
`/tmp/xoscal-analysis-final-sca-replay.json`. Actual versioned dependency inventory
counts were Go 12, Python 4, Java 16, C# 7, TypeScript 1, Swift 1; all had zero
reported vulnerabilities. Scope is the delivered archive and its identifiable
source declarations/locks/compiled metadata, as documented in
[ANALYSIS-POLICY.md](ANALYSIS-POLICY.md). Consumer-dependent future resolutions
are not a claim of these source archive scans.

The updated static Distroless image tar
`c40683ae3c46e862e5ac7075d6752be184773ca854d1ba331e9b06eabb9afb5a`
passed both release and stricter HIGH/CRITICAL image thresholds with zero reported
vulnerabilities. Actual image evidence is `/tmp/xoscal-static-image-analysis` and
`/tmp/xoscal-static-image-ci-admitted`; strict final SBOM evidence is
`/tmp/xoscal-static-image-sbom-final-frozen`. The official base signature was
read-only verified with Cosign against the existing public Distroless identity;
`/tmp/xoscal-static-base-cosign.json` and its log retain that result. The image's
actual help/startup/config checks are retained in
`/tmp/xoscal-static-runtime-{help.log,server.log,state.json}`. This local image was
never pushed. The original image's real HIGH OpenSSL finding remains preserved
in `/tmp/xoscal-image-analysis-real` rather than waived.

The actual Linux x86_64 GoReleaser archive containing three binaries passed strict
SBOM admission at SHA-256
`1355f239ab12c1c74798979317899ea1af243e5870e059de52ae496320bb2210`:
`/tmp/xoscal-snapshot-linux-x86_64-sbom-final-frozen`. Other platform candidate
outputs must retain their own subject proofs; this check is not a claim that
nominally similar archive bytes share its result.

The original incomplete binary SBOM had 48 supplier errors plus one relationship
error. Producers now preserve the raw inventory and enrich only evidenced
supplier/author roles, source versions, and actual relationships. Unknown original
authors remain unknown where a source mirror or explicit registry publishing
account is identified. No exceptions were added. Python's dependency warning was
fixed from the exact SHA-256-backed delivered requirements file. The validator's
warning-only satisfied NTIA roll-up is interpreted accurately while every
unsatisfied warning remains blocking under strict admission.

Final regression checks passed: 24 admission tests, 8 safe archive extraction
tests, 8 metadata enrichment tests, and 13 publisher evidence tests (53 total),
plus Python compilation and scoped Git whitespace checks. Logs are
`/tmp/xoscal-{analysis,extraction,sbom-enrichment,publisher-metadata}-tests.log`.
Negative cases cover tool errors, unsupported/empty inventory, missing proof,
wrong/tampered digests, duplicate JSON properties, exact-version mismatches,
misleading publisher substitutions, unsupported XML entities, traversal/links,
warning-only admission, and original-file dependency graph tampering.

## Patched frontend validation (2026-10-05)

The original vendored Scalar 1.25.0 browser JS was confirmed to contain Axios
1.7.2. Direct advisories for the outer Scalar package alone did not cover that
bundled runtime. The verified latest published Scalar 1.72.4 tar removed Axios,
but its linked source map identified two HIGH nanoid advisories, one MEDIUM
unhead advisory and one LOW provider-utils advisory. The rebuild fixes these
constituents with exact nanoid 5.1.16, unhead 2.1.13 and provider-utils 4.0.33
pins. Original metadata, tar SRI and source maps were preserved; no applicability
waivers or lowered security thresholds were used.

The final published-module rebuild ran through the task-owned isolated Dagger
0.21.7 runner, using pinned Node 22.23.3, npm 10.9.9 and esbuild 0.25.12.
`npm ci` used only registry.npmjs.org with empty user/global npm configurations
and dependency install scripts disabled. The final browser assembly has 185
unique observed runtime name/version pairs, including actual nested installations
and emitted CSS contributors. Its 2,813 mapped sources replay against preserved
raw compiled inputs under the documented upstream-map-comment transformation.
All 181 present runtime notices plus the exact Scalar upstream license are
preserved in the input archive and assembled notices. Actual core build logs and
outputs are `/tmp/xoscal-frontend-rebuild-frozen-v2{,.log}`; producer replay is
`/tmp/xoscal-frontend-production-replayed`.

Exact final served bytes:

- `scalar.js`: SHA-256
  `0c4128e03a25a1657f2de4a5ac682c17bf3547dd1c78a50b1e2025407fbc40e3`.
- `scalar.js.map`: SHA-256
  `68290172fa565a3b6b2d3cb6b3090e27c0a8d0f92a45822cd2f5d5e98b91421f`.

Formal `FrontendAssets` passed on the actual repository SDK/runtime and pinned
Dagger engine: Trivy 0.72.0 scanned all 185 observed versions and reported zero
findings at every severity. Raw scanner status/log/JSON and producer evidence are
`/tmp/xoscal-frontend-final-evidence-v2`; its raw SBOM SHA-256 was
`c05b16046e17f60b14f932ba6e7e02b3a0ddf0add53a55273a89341936f83723`.
An independent public official GitHub advisory API query for all 185 exact
runtime versions returned zero active reviewed advisories; all five primary
responses and URL/header/hash receipts are `/tmp/xoscal-frontend-advisories-final`.
This does not claim coverage of undisclosed vulnerabilities or a complete
upstream transitive graph.

Strict NTIA assessment correctly rejected that initial SBOM for its one real
missing unique identifier on the repository-produced assembly root. The raw
failure is `/tmp/xoscal-frontend-sbom-raw`, and the failed invocation is
`/tmp/xoscal-frontend-sbom-final.log`. The producer now identifies this actual
unregistered browser assembly with a generic PURL containing the exact JS
checksum, consistently used by its genuine assembly dependency relationships.
The primary [generic PURL definition](https://github.com/package-url/purl-spec/blob/main/types/generic-definition.json)
permits this unregistered-package identity and checksum qualifier. It does not
invent a published npm coordinate or unknown upstream author.

The repaired strict SBOM passed with zero violations and zero exceptions:
`/tmp/xoscal-frontend-sbom-repaired{,.log}`, exact input SBOM SHA-256
`5428b1c630adefc9b587f25150141dc9649526cbfa4eb8a0689751c01f3ce84f`.
The same JS/map bytes were used in both assessments. These are different SBOM
snapshots: the earlier Trivy receipt must not be paired with the repaired strict
SBOM receipt. The final Site invocation reruns both from its same producer graph
and binds the served five files; its outcome is owned by the parent integration
validation, not asserted by these earlier standalone snapshots.

The frontend's 24 source-only tests passed, including exact tar/SRI rejection,
peer-context exclusion, unexplained paths, incomplete source text, reviewed lock
and toolchain enforcement, emitted-runtime coverage, CSS/banner binding, actual
notices, failed build/scanner statuses, tampered served bytes and fixed CRITICAL
admission. Test-only fixture pins are isolated Python module mocks; the production
CLI has no pin override or test mode. Existing 24 analysis, 8 safe extraction,
8 metadata enrichment and 13 publisher tests also passed (77 total). Logs are
`/tmp/xoscal-frontend-contract-tests.log` and
`/tmp/xoscal-{analysis,extraction,sbom-enrichment,publisher}-tests-frontend.log`. Python
compilation and scoped whitespace checks passed. No hosted signing, publication,
release, merge, access/identity change or production cluster action was performed.
