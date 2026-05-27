# Contributing to OSCALiFY

Thank you for your interest in contributing to OSCALiFY!

## Getting Started

### Prerequisites

- Go 1.25 or later
- Docker 20.10 or later
- Dagger CLI v0.21.0 or later
- Make
- buf (for protobuf)

### Setup

```bash
# Clone the repository
git clone https://github.com/ckodex/ckodex-oscalify.git
cd ckodex-oscalify

# Install dependencies
make deps

# Run tests
make test

# Run linting
make lint
```

## Development Workflow

We use Git Flow for branch management:

- **master**: Production releases
- **develop**: Integration branch for next release
- **feature/**: New features
- **bugfix/**: Bug fixes
- **release/**: Release preparation
- **hotfix/**: Critical production fixes

### Creating a Feature Branch

```bash
git flow feature start my-feature
# Make your changes
git flow feature finish my-feature
```

### Making Changes

1. Create a branch from `develop`
2. Make your changes with clear, focused commits
3. Add tests for new functionality
4. Ensure all tests pass: `make test`
5. Ensure linting passes: `make lint`
6. Submit a pull request to `develop`

### Commit Messages

Follow conventional commits format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

Example:
```
feat(oscal): add support for AssessmentPlan generation

- Added GenerateAssessmentPlan method
- Updated protobuf definitions
- Added unit tests

Closes #123
```

## Code Style

- Follow Go standard formatting: `gofmt -s -w .`
- Run linters: `make lint`
- Write tests for all new code
- Keep functions focused and small
- Add comments for exported functions and complex logic

## Testing

### Unit Tests

```bash
make test
```

### Integration Tests

```bash
make test-integration
```

### Dagger Pipeline Validation

```bash
dagger call test --source=.
```

## Documentation

- Update README.md for user-facing changes
- Add inline comments for complex code
- Update proto files with clear descriptions
- Keep OSCALiFY.md in sync with implementation

## Pull Request Process

1. Update documentation if needed
2. Ensure all tests pass
3. Update CHANGELOG.md
4. Request review from maintainers
5. Address review feedback
6. Squash commits if requested
7. Merge after approval

## Project Structure

```
├── dagger/          # Dagger CI/CD pipeline
├── proto/           # Protobuf definitions
├── server/          # Main application code
│   ├── cmd/         # CLI commands
│   ├── internal/    # Internal packages
│   └── pkg/         # Public packages
├── docs/            # Documentation
└── data/            # Sample data
```

## Questions?

- Open an issue for bugs or feature requests
- Check existing issues first
- Join discussions in GitHub Discussions
- Contact maintainers via security@ckodex.io for security issues
