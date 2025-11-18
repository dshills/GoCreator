# Contributing to GoCreator

Thank you for your interest in contributing to GoCreator! This document provides guidelines and information to help you contribute effectively.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Contribution Process](#contribution-process)
- [Code Standards](#code-standards)
- [Testing Requirements](#testing-requirements)
- [Pull Request Guidelines](#pull-request-guidelines)
- [Reporting Issues](#reporting-issues)
- [License](#license)

## Code of Conduct

This project adheres to a Code of Conduct that all contributors are expected to follow. Please read [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) before contributing.

## Getting Started

### Prerequisites

- Go 1.21+ (requires generics support)
- Git
- golangci-lint for linting
- LLM provider API key (Anthropic, OpenAI, or Google)

### Development Setup

For detailed development environment setup, see [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).

Quick setup:

```bash
# Clone the repository
git clone https://github.com/dshills/gocreator.git
cd gocreator

# Install dependencies
go mod download

# Set up your LLM provider
export ANTHROPIC_API_KEY=sk-ant-your-key-here

# Build the project
make build

# Run tests
make test
```

## Development Workflow

GoCreator uses the **Specify** system for specification-driven development. For new features:

1. **Create a specification** using `/speckit.specify <feature-description>`
2. **Clarify requirements** with `/speckit.clarify` if needed
3. **Generate implementation plan** with `/speckit.plan`
4. **Generate tasks** with `/speckit.tasks`
5. **Implement the feature** following the plan
6. **Validate** with tests, linting, and code review

See [CLAUDE.md](CLAUDE.md) for detailed workflow documentation.

## Contribution Process

### 1. Find or Create an Issue

- Check existing [issues](../../issues) for work in progress
- For bugs: Search for duplicates before creating new issues
- For features: Discuss in an issue before starting work
- Reference the issue number in your commits and PR

### 2. Fork and Branch

```bash
# Fork the repository on GitHub, then:
git clone https://github.com/YOUR_USERNAME/gocreator.git
cd gocreator

# Create a feature branch
git checkout -b feature/your-feature-name

# Or for bug fixes
git checkout -b fix/issue-123-description
```

### 3. Make Your Changes

- Follow the [Code Standards](#code-standards) below
- Write tests for all new functionality
- Update documentation as needed
- Keep commits focused and atomic
- Write clear commit messages

### 4. Test Your Changes

```bash
# Run all tests
make test

# Run linting
make lint

# Run security checks (if available)
make security

# Build to ensure no compilation errors
make build
```

### 5. Submit a Pull Request

- Push your branch to your fork
- Create a Pull Request against the `main` branch
- Fill out the PR template completely
- Link to related issues
- Wait for review and address feedback

## Code Standards

### Go Code Style

- Follow standard Go conventions and idioms
- Use `gofmt` for formatting (automatic with most editors)
- Use `goimports` for import organization
- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines

### Naming Conventions

- Use descriptive variable and function names
- Avoid abbreviations unless widely understood
- Use camelCase for unexported names, PascalCase for exported
- Package names should be lowercase, single-word when possible

### Documentation

- All exported functions, types, and packages must have godoc comments
- Comments should explain *why*, not just *what*
- Include examples in godoc when helpful
- Keep README.md and other docs updated

### Project Principles

GoCreator follows strict design principles documented in `.specify/memory/constitution.md`:

1. **Deterministic Execution** - Same inputs produce identical outputs (NON-NEGOTIABLE)
2. **Specification as Source of Truth** - All code traces back to specifications
3. **Separation of Reasoning and Action** - LangGraph reasons, GoFlow executes
4. **Test-First Discipline** - Comprehensive tests required
5. **Safety and Bounded Execution** - All operations logged, reversible, bounded

## Testing Requirements

### Test Coverage

- Aim for 80%+ test coverage on new code
- All public APIs must have unit tests
- Integration tests for end-to-end workflows
- Use table-driven tests for multiple test cases

### Test Structure

```go
func TestFeatureName(t *testing.T) {
    tests := []struct {
        name     string
        input    InputType
        expected OutputType
        wantErr  bool
    }{
        {
            name:     "descriptive test case name",
            input:    /* test data */,
            expected: /* expected result */,
            wantErr:  false,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with race detection
go test -race ./...

# Run specific package tests
go test ./pkg/llm/...

# Run with verbose output
go test -v ./...
```

## Pull Request Guidelines

### Before Submitting

- [ ] All tests pass (`make test`)
- [ ] Linting passes (`make lint`)
- [ ] Code is properly formatted (`gofmt`)
- [ ] Documentation is updated
- [ ] Commit messages are clear and descriptive
- [ ] Branch is up to date with main

### PR Description

Your PR should include:

- **Summary**: Brief description of the change
- **Motivation**: Why is this change needed?
- **Changes**: What was changed?
- **Testing**: How was it tested?
- **Screenshots**: If UI changes (N/A for GoCreator)
- **Breaking Changes**: Any backwards incompatible changes?
- **Related Issues**: Links to related issues

### Review Process

- All PRs require at least one approval
- Code review feedback should be addressed
- CI checks must pass
- Maintainers may request changes or additional tests
- Be patient and responsive to feedback

### Commit Message Format

```
<type>: <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Test additions or changes
- `chore`: Build process or auxiliary tool changes

**Example:**
```
feat: Add prompt caching support for Anthropic

Implement provider-native caching to reduce token costs by 60-80%.
Caching is automatic for Anthropic clients with graceful fallback
for other providers.

Closes #123
```

## Reporting Issues

### Bug Reports

Include:
- GoCreator version (`gocreator version`)
- Go version (`go version`)
- Operating system
- Steps to reproduce
- Expected behavior
- Actual behavior
- Error messages and logs (`--log-level=debug`)
- Specification file (if applicable)

### Feature Requests

Include:
- Clear description of the feature
- Use cases and motivation
- Proposed implementation (if you have ideas)
- Willingness to contribute implementation

### Security Issues

**Do not report security vulnerabilities in public issues.**

See [SECURITY.md](SECURITY.md) for how to report security issues responsibly.

## Development Resources

- [Architecture Documentation](docs/ARCHITECTURE.md) - System design and architecture
- [Development Guide](docs/DEVELOPMENT.md) - Detailed development setup
- [Project Workflow](CLAUDE.md) - Specify system and development workflow
- [Project Constitution](.specify/memory/constitution.md) - Core principles

## Getting Help

- Check the [documentation](docs/)
- Search [existing issues](../../issues)
- Ask questions in [discussions](../../discussions)
- Review example specifications in `examples/`

## Recognition

Contributors will be recognized in:
- CHANGELOG.md for significant contributions
- GitHub contributors page
- Release notes

## License

By contributing to GoCreator, you agree that your contributions will be licensed under the [MIT License](LICENSE).

---

Thank you for contributing to GoCreator! Your efforts help make autonomous Go code generation better for everyone.
