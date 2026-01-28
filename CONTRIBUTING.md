# Contributing to ingress-to-gateway

Thank you for your interest in contributing to ingress-to-gateway! We welcome contributions from the community.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/mayens/ingress-to-gateway.git`
3. Create a feature branch: `git checkout -b my-feature`
4. Make your changes
5. Run tests: `make test`
6. Commit your changes: `git commit -am 'Add new feature'`
7. Push to your fork: `git push origin my-feature`
8. Create a Pull Request

## Development Setup

### Prerequisites

- Go 1.23 or later
- Access to a Kubernetes cluster (for testing)
- kubectl configured

### Building

```bash
make build
```

### Running Tests

```bash
make test
```

### Code Style

This project follows standard Go coding conventions. Please ensure your code:

- Passes `go fmt`
- Passes `go vet`
- Includes appropriate tests
- Includes comments for exported functions

Run verification checks:

```bash
make verify
```

## Project Structure

```
ingress-to-gateway/
├── cmd/               # Command implementations
├── pkg/               # Public libraries
│   ├── analyzer/      # Ingress analysis logic
│   ├── converter/     # HTTPRoute conversion
│   ├── reporter/      # Report generation
│   ├── validator/     # HTTPRoute validation
│   └── k8s/          # Kubernetes client wrapper
├── internal/          # Private libraries
├── test/             # Test fixtures and integration tests
├── examples/         # Example resources
└── docs/             # Documentation
```

## Adding New Annotation Support

To add support for a new NGINX Ingress annotation:

1. Update [pkg/analyzer/analyzer.go](pkg/analyzer/analyzer.go:1) to detect the annotation
2. Update [pkg/converter/converter.go](pkg/converter/converter.go:1) to convert the annotation
3. Add tests in appropriate `_test.go` files
4. Update documentation in [README.md](README.md:1)

## Testing Guidelines

- Write unit tests for all new functionality
- Use table-driven tests where appropriate
- Aim for >80% code coverage
- Include integration tests for complex features

## Pull Request Process

1. Update the README.md with details of changes if applicable
2. Update documentation for new features
3. Add tests for new functionality
4. Ensure all tests pass
5. Update CHANGELOG.md (if applicable)
6. Request review from maintainers

## Code of Conduct

This project follows the [Kubernetes Code of Conduct](https://github.com/kubernetes/community/blob/master/code-of-conduct.md).

## Questions?

Feel free to open an issue for any questions or concerns.

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
