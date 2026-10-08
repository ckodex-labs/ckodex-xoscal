# Security Policy

## Supported Versions

| Version | Supported |
| ------- | --------- |
| 0.3.x   | Yes       |
| < 0.3   | No        |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly.

### How to Report

1. **Email**: Send details to security@ckodex.io
2. **Include**: 
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if known)

### What to Expect

- We will acknowledge receipt within 48 hours
- We will provide a detailed response within 7 days
- We will work with you to validate and patch the vulnerability
- We will coordinate disclosure on a mutually agreed timeline

### Private Disclosure

We request that you do not publicly disclose the vulnerability until we have had a chance to address it and issue a fix.

## Security Best Practices

For users of xOSCAL:

- Keep dependencies updated
- Review the [SBOM](https://github.com/ckodex-labs/ckodex-xoscal/releases) for each release
- Follow the principle of least privilege when deploying
- Enable authentication and rate limiting in production deployments
- Regularly audit generated OSCAL artifacts

See [BETA-CONTRACT.md](docs/BETA-CONTRACT.md) for the single-workspace
deployment boundary and [RELEASE-CONTRACT.md](docs/RELEASE-CONTRACT.md) for
exact-byte release verification. Checksums alone do not establish provenance.

### Local CLI filesystem boundary

Explicit artifact and output paths are selected by the operator. Repository
scaffolding and evidence collection confine derived paths to the selected
directory using handle-backed filesystem roots. Relative child symlinks that
resolve inside the directory are supported; absolute child symlinks, including
absolute links to an in-directory file, are rejected. The selected directory
itself may be a symlink. These pathname controls do not establish hardlink
ownership or prevent an operator from selecting a mounted filesystem.

Repository inspection fails on unreadable or broken child paths in the scanned
tree, including paths outside the conventional metadata filenames. It does not
silently report a complete scan after a filesystem failure.

An explicitly supplied evidence directory must be readable. Missing blobs or
checksums, malformed checksums and inaccessible files cannot establish verified
evidence. A checksum mismatch remains tampered evidence. Local derogations are
operator-maintained risk declarations, not hosted release signatures.

Audit-bundle creation stages a private file and replaces the selected output
name only after successful completion. The resulting archive has mode `0600`;
an existing output symlink is replaced rather than followed. Producer and
verifier share a 256 MiB limit on the total decompressed TAR stream, including
headers and padding. Member names must be canonical slash-separated NFC names,
portable to the supported native filesystems; aliases, case-fold collisions,
reserved device names and undeclared members are rejected. Artifact/evidence
digests and sidecars establish internal byte consistency, not signer identity.

## Dependency Scanning

This project uses automated security scanning:
- **Static Analysis**: Gosec SAST scanner
- **Container Scanning**: Trivy vulnerability scanner
- **SBOM Generation**: Syft for all releases
- **Supply Chain**: GitHub artifact attestations from the existing hosted
  release workflow identity, verified against exact manifest, transport and OCI
  subjects. Signed manifests cover every required payload and presentation file.

Results are published in each GitHub release.

## Security-Related Features

- **Authentication**: Token-based authentication (SPIRE/SPIFFE workload identity mTLS planned on roadmap)
- **Rate Limiting**: Configurable rate limiting per client
- **Input Validation**: Protobuf request validation and OSCAL schema checks
- **Audit Logging**: Verification/fetch audit events and gRPC method, duration,
  status and request-ID logging
- **Storage**: SQLite persistence with tested backup/restore. This beta does not
  provide application-managed SQLite encryption; protect host files and backups
  through deployment controls.
