# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| 0.1.x   | :white_check_mark: |

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

For users of OSCALiFY:

- Keep dependencies updated
- Review the [SBOM](https://github.com/ckodex/ckodex-oscalify/releases) for each release
- Follow the principle of least privilege when deploying
- Enable authentication and rate limiting in production deployments
- Regularly audit generated OSCAL artifacts

## Dependency Scanning

This project uses automated security scanning:
- **Static Analysis**: Gosec SAST scanner
- **Container Scanning**: Trivy vulnerability scanner
- **SBOM Generation**: Syft for all releases
- **Supply Chain**: SLSA provenance with Cosign signing

Results are published in each GitHub release.

## Security-Related Features

- **Authentication**: Token-based and SPIRE-based authentication support
- **Rate Limiting**: Configurable rate limiting per client
- **Input Validation**: Strict protobuf schema validation
- **Audit Logging**: Comprehensive request/response logging
- **Secure Storage**: SQLite with optional encryption support
