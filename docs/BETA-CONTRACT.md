# xOSCAL beta contract

Status: implementation baseline
Schema contract: OSCAL 1.2.3

## Beta outcome

The beta is a live, evidence-backed review loop:

1. An authenticated operator opens a workspace.
2. A claim and its content-addressed evidence are preflighted, then imported as
   one all-or-nothing record.
3. The operator runs verification and sees every check result.
4. The operator reviews the claim lineage, current trust state, and append-only verification history.
5. The operator exports a deterministic verification receipt only when the evidence envelope is complete.

The portal may expose illustrative design views, but illustrative data is never mixed
with live workspace data and never supplies a live trust decision.

## Trust-state contract

The closed claim-state vocabulary is:

`observed` · `inferred` · `claimed` · `attested` · `contradicted` · `quarantined`

The service-level trust vocabulary is:

`candidate` · `incomplete` · `verified` · `rejected`

`verified` is fail-closed. It requires all checks selected for the verification profile
to have run and passed. An unavailable, unsupported, missing, or inconclusive check is
`incomplete`; it is never silently treated as a pass.

The audit surface validates the global verification-event hash chain before returning
history. A broken chain is a failed precondition, not a partial success. Receipt export
also requires a verified current projection, a matching latest audit event, valid
source/proof/policy evidence bytes, and a valid content digest. The receipt contains
claim, proof, event, and evidence metadata but never raw evidence bytes.

Batch intake uses `POST /v1/transparency/import/preflight` followed by
`POST /v1/transparency/import` with `all_or_nothing=true`. A failed preflight does
not mutate the workspace; a failed commit is rolled back. Equivalent replays are
reported as `already_present`.

## Beta verification profile

The default profile requires:

- at least one source evidence reference;
- a matching stored content digest;
- a signature check that is implemented and passed;
- a policy check that is implemented and passed;
- a current-state check;
- a persisted proof-state record and audit event.

The beta includes these bounded verification providers:

- `ed25519` detached signatures: `issuer.key_json` contains the base64 public key and
  a `signature` proof reference resolves to raw signature bytes in content-addressed
  evidence. The signed payload is the canonical `xoscal-claim-signature-v1` claim
  envelope, excluding proof references themselves.
- `ecdsa-p256` and `ecdsa-p384` detached signatures: the issuer key carries the raw
  uncompressed point coordinates (`x || y`, base64) and the signature proof reference
  resolves to the raw `r || s` signature (64 bytes for P-256, 96 for P-384) over the
  SHA-256 digest of the same canonical claim envelope.
- `xoscal-json` policy profiles: a policy reference resolves to a content-addressed
  JSON document with `version: xoscal-policy-v1` and optional relation, BOM-kind,
  issuer-kind, and validity constraints.
- `xoscal-remote` policy profiles: fetched over HTTPS through the deployment's
  bounded fetch policy and evaluated as `xoscal-policy-v1` documents.
- `transparency` proof references resolve to an RFC 6962-style inclusion proof with
  an Ed25519-signed checkpoint from the configured log.

External evidence fetching is available only when the deployment enables a
bounded fetch policy (`fetch_policy.enabled`). A fetch is admitted only over
HTTPS, within the configured byte cap, timeout, redirect bound, and optional
host allowlist; fetched bytes are stored as content-addressed evidence with
the source URI recorded, and every attempt — success, policy denial, or
failure — appends a fetch audit event queryable via
`GET /v1/transparency/evidence/fetch/events`. Fetching stays disabled unless
explicitly enabled; an absent policy is a fail-closed default.

Missing inputs remain `incomplete`; malformed or cryptographically invalid inputs are
`rejected`. Unsupported external provider types remain explicitly `unavailable` and
cannot produce a verified state.

## Beta security boundary

- `auth_mode=none` is local-development-only.
- Network-reachable beta deployments require TLS and authentication.
- A beta deployment is single-workspace and single-tenant. Its configured
  authentication tokens grant access to that deployment's workspace; tenant
  and workspace multiplexing is not part of this beta boundary.
- SQLite beta deployments run as one replica with tested backup and restore.
- Multi-replica operation requires an external transactional database.
- Peer claim synchronization and external transparency/witness inclusion are not
  advertised as available capabilities in this beta.

## Beta release gates

- `go test ./...`, `go vet ./...`, and `buf lint` pass.
- Generated site assets exist and dynamic browser smoke passes.
- Core review and verification journeys complete with keyboard-only input.
- No user-visible beta action is an inert placeholder.
- No `verified`, `attested`, `sealed`, or `signed` state is emitted without its proof object.
- Verification history is chain-valid and receipt export is blocked for incomplete or
  mismatched proof projections.
- OSCAL version identity is consistent across schema, API, portal, and exported artifacts.
