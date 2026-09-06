# OSCAL release updates

The project targets OSCAL `1.2.3`. The selected version lives in
`server/internal/oscalversion/VERSION`; the runtime serializer and embedded
validator both read that file, so they cannot drift through separate constants.

## Automated monitor

`.github/workflows/oscal-reconcile.yml` polls the official NIST release feed
every six hours and can also be run manually. It delegates update behavior to
`scripts/reconcile/update-oscal.sh`.

For a new release, the updater:

1. resolves the official GitHub release and tagged commit;
2. requires a non-draft, non-prerelease semantic-version tag;
3. verifies every JSON schema against the SHA-256 digest published with the
   release asset;
4. compares a normalized schema shape with the currently embedded release;
5. updates versioned schemas, the offline digest manifest, the version pin,
   the reconciliation lock, per-model spec registry, and current-version
   references on user-facing project surfaces;
6. runs `buf lint`, `buf breaking`, and the OSCAL-focused Go tests; and
7. opens a draft pull request. It never auto-merges.

Constraint, vocabulary, and documentation changes can be materialized by the
workflow. A change to object properties, required fields, cardinality, or type
shape exits with code `4`, writes `reconcile-report.json`, and requires an
additive protobuf reconciliation before the pin can advance.

## Local commands

```sh
make oscal-check

# Update to the latest release in a clean checkout.
make oscal-update

# Update to an explicit version.
OSCAL_VERSION=1.2.4 make oscal-update

# Verify an extracted official release without writing.
./scripts/reconcile/update-oscal.sh \
  --version 1.2.3 \
  --source-dir /path/to/oscal-1.2.3 \
  --check
```

The update command refuses tracked changes by default. `--allow-structural` is
an operator-only continuation after the protobuf delta has been reconciled and
reviewed; the scheduled workflow does not use it.

## Validation boundary

Tier 1 is blocking and uses the embedded OSCAL 1.2.3 JSON schema for all eight
released model types, including Mapping.

Tier 2 remains explicitly non-blocking because the latest released NIST
`oscal-cli` is `1.0.3`, which embeds OSCAL 1.1.2 and does not expose a Mapping
subcommand. It must not be described as authoritative 1.2.3 Metaschema
validation. The updater can advance Tier 1 independently while this external
tooling gap remains visible.
