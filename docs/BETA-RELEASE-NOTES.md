# xOSCAL beta release notes

Status: review-only beta boundary
Date: 2026-08-10

## What this beta is

xOSCAL beta is a single-workspace operator surface for reviewing claims and
content-addressed evidence. A deployment has one SQLite writer and one
authenticated operator scope. The Live Review route is the supported first-use
journey for this beta.

The beta can:

- import a bounded batch of up to 10 claim/evidence records with preflight and
  all-or-nothing commit;
- re-hash stored evidence and expose claim lineage, verification history, and
  proof state;
- project claim relationships into a deterministic graph with an append-only,
  hash-chained projection audit history;
- run the configured digest, Ed25519 signature, and bounded policy checks; and
- export a receipt only when the required proof envelope, evidence bytes, and
  audit-chain head agree.

## Trust boundary

The following rules are release constraints, not UI guidance:

- `candidate`, `incomplete`, and `rejected` claims cannot be exported as
  attested receipts;
- an unavailable or failed verification provider remains visible as incomplete
  or rejected and never becomes an attestation by fallback;
- the deployment boundary is one workspace; token authentication is not
  tenant isolation;
- peer synchronization, multi-workspace sharing, external evidence fetching,
  multi-replica SQLite operation, and external transactional storage are out
  of scope; and
- production promotion requires separate evidence for target storage restore,
  published artifact signatures, and SLSA provenance.

## Evidence status

The local beta checkout has passing evidence for the Go build/test/race/vet
gates, Buf gates, generated-site checks, live gateway health, bounded batch
preflight/commit/replay, OSCAL schema and constraint validation, and the Dagger
security gates. These local results do not establish production deployment
acceptance or published artifact provenance.

Before promotion, attach the deployment-owner restore record, clean-environment
signature and provenance verification, and the full assistive-technology
acceptance record to the release bundle. Until those records exist, distribute
this build only as a clearly labelled review beta.

## Operator expectations

Configure TLS and token files through the Kubernetes Secrets referenced by the
server Deployment. Keep one replica while SQLite is the backing store. Use
`xoscal-backup` for snapshots, verify the digest, and stop the writer before a
restore. Do not treat a green local test run as evidence of a production
storage or identity boundary.
