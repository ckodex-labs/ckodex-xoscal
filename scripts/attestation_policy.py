"""One exact hosted identity policy for local detached-bundle verification."""
from release_inventory import identity

REPOSITORY = "ckodex-labs/ckodex-xoscal"
WORKFLOW = REPOSITORY + "/.github/workflows/release.yml"


def verification_flags(bundle, tag, revision):
    identity(tag, revision)
    if tag == "dev":
        raise ValueError("development has no production signing identity")
    # gh makes --cert-identity and --signer-workflow mutually exclusive.
    # The exact certificate SAN includes both the workflow and the tag ref.
    return ["--bundle", str(bundle), "--hostname", "github.com", "--repo", REPOSITORY,
            "--cert-identity", "https://github.com/" + WORKFLOW + "@refs/tags/" + tag,
            "--cert-oidc-issuer", "https://token.actions.githubusercontent.com",
            "--signer-digest", revision, "--source-digest", revision,
            "--source-ref", "refs/tags/" + tag, "--predicate-type", "https://slsa.dev/provenance/v1",
            "--deny-self-hosted-runners", "--format", "json"]
