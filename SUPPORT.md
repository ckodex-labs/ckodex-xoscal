# Support

## Getting Help

### Documentation

- [README.md](README.md) - Project overview and quick start
- [OSCALiFY.md](OSCALiFY.md) - Detailed OSCAL mapping documentation
- [docs/](docs/) - Additional documentation

### Community

- **GitHub Issues**: Report bugs and request features
- **GitHub Discussions**: Ask questions and share ideas
- **Slack**: Join our community at [ckodex.slack.com](https://ckodex.slack.com)

### Enterprise Support

For enterprise support, SLAs, and priority issue resolution:
- Email: support@ckodex.io
- Website: https://ckodex.io/support

## Common Issues

### Build Failures

```bash
# Clean and rebuild
make clean
make build
```

### Test Failures

```bash
# Run tests with verbose output
go test -v ./server/...
```

### Dagger Pipeline Issues

```bash
# Clear Dagger cache
dagger cache prune

# Re-run with debug output
dagger call test --source=. --log-level=debug
```

### Database Issues

```bash
# Reset SQLite database
rm -f data/*.db
make test
```

## Reporting Bugs

When reporting bugs, include:

1. **Version**: OSCALiFY version
2. **OS**: Operating system and version
3. **Go Version**: `go version`
4. **Steps to Reproduce**: Detailed steps
5. **Expected Behavior**: What you expected
6. **Actual Behavior**: What happened instead
7. **Logs**: Relevant log output

Use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.md) if available.

## Feature Requests

When requesting features, include:

1. **Use Case**: What problem does this solve?
2. **Proposed Solution**: How should it work?
3. **Alternatives**: What alternatives did you consider?
4. **Additional Context**: Any other relevant information

Use the [feature request template](.github/ISSUE_TEMPLATE/feature_request.md) if available.

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Security

For security vulnerabilities, see [SECURITY.md](SECURITY.md) for reporting guidelines.

## Release Notes

Release notes are published in [CHANGELOG.md](CHANGELOG.md) and in GitHub Releases.

## Training

- **Documentation**: See [docs/](docs/)
- **Examples**: See [data/](data/) for sample configurations
- **Tutorials**: Coming soon

## Professional Services

For custom development, consulting, or training:
- Email: services@ckodex.io
- Website: https://ckodex.io/services
