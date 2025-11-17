# Development Guide

Welcome to GoCreator development! This guide covers everything you need to set up your development environment, understand the codebase, and contribute effectively.

## Table of Contents

1. [Development Setup](#development-setup)
2. [Project Structure](#project-structure)
3. [Development Workflow](#development-workflow)
4. [Testing](#testing)
5. [Code Quality](#code-quality)
6. [Common Tasks](#common-tasks)
7. [Contributing Guidelines](#contributing-guidelines)
8. [Troubleshooting](#troubleshooting)

## Development Setup

### Prerequisites

- **Go 1.21+** - [Download from go.dev](https://go.dev/dl/)
- **Git** - Version control
- **Make** - Build automation (pre-installed on macOS/Linux)
- **golangci-lint** - Linting tool
- **gosec** - Security scanner (optional but recommended)
- **goimports** - Import formatting (optional)

### Required Tools Installation

```bash
# Go (if not already installed)
# Visit https://go.dev/dl/ and follow instructions

# golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# gosec (optional but recommended)
go install github.com/securego/gosec/v2/cmd/gosec@latest

# goimports (optional)
go install golang.org/x/tools/cmd/goimports@latest
```

### LLM Provider Setup

You'll need access to an LLM provider. Choose one:

```bash
# Anthropic (recommended for determinism)
export ANTHROPIC_API_KEY=sk-ant-...

# OpenAI
export OPENAI_API_KEY=sk-...

# Google
export GOOGLE_API_KEY=...
```

### Clone and Initialize

```bash
# Clone repository
git clone https://github.com/dshills/gocreator.git
cd gocreator

# Ensure you're on the development branch
git checkout 001-core-implementation

# Download dependencies
go mod download
go mod tidy
```

## Project Structure

```
gocreator/
├── cmd/gocreator/              # CLI entry point
│   ├── main.go                 # Root command and setup
│   ├── clarify.go              # Clarify command
│   ├── generate.go             # Generate command
│   ├── validate.go             # Validate command
│   ├── full.go                 # Full pipeline command
│   ├── dump_fcs.go             # Dump FCS command
│   ├── version.go              # Version command
│   ├── exit_codes.go           # Exit code definitions
│   ├── interactive.go          # Interactive prompting
│   ├── batch.go                # Batch mode parsing
│   └── progress.go             # Progress reporting
│
├── internal/                   # Private packages (not importable by others)
│   ├── clarify/                # Clarification engine (LangGraph-Go)
│   │   ├── analyzer.go         # Identify ambiguities
│   │   ├── questions.go        # Generate questions
│   │   ├── graph.go            # LangGraph state machine
│   │   └── engine.go           # Orchestration
│   ├── generate/               # Generation engine (LangGraph-Go)
│   │   ├── planner.go          # Architecture planning
│   │   ├── coder.go            # Code synthesis
│   │   ├── tester.go           # Test generation
│   │   ├── graph.go            # LangGraph state machine
│   │   └── plan_builder.go     # Plan construction
│   ├── spec/                   # Specification processing
│   │   ├── parser.go           # Unified parser
│   │   ├── parser_yaml.go      # YAML parsing
│   │   ├── parser_json.go      # JSON parsing
│   │   ├── parser_md.go        # Markdown parsing
│   │   ├── validator.go        # Validation logic
│   │   ├── fcs_builder.go      # FCS construction
│   │   └── fcs_hash.go         # Checksum generation
│   ├── validate/               # Validation engine
│   │   ├── build.go            # Build validation
│   │   ├── lint.go             # Lint validation
│   │   ├── test.go             # Test validation
│   │   └── report.go           # Report generation
│   ├── workflow/               # Workflow execution (GoFlow)
│   │   ├── engine.go           # Workflow execution engine
│   │   ├── tasks.go            # Task definitions
│   │   ├── patcher.go          # Patch application
│   │   ├── parallel.go         # Parallel execution
│   │   ├── logger.go           # Execution logging
│   │   └── security.go         # Security enforcement
│   ├── models/                 # Domain models
│   │   ├── spec.go             # Input specification
│   │   ├── clarification.go    # Clarification models
│   │   ├── fcs.go              # FCS model
│   │   ├── generation.go       # Generation models
│   │   ├── validation.go       # Validation results
│   │   ├── workflow.go         # Workflow models
│   │   └── log.go              # Logging models
│   ├── config/                 # Configuration management
│   │   ├── loader.go           # Config loading
│   │   └── defaults.go         # Defaults
│   └── errors/                 # Error handling
│       ├── types.go            # Error type definitions
│       └── wrapping.go         # Error wrapping helpers
│
├── pkg/                        # Public packages (reusable libraries)
│   ├── langgraph/              # LangGraph-Go client
│   │   ├── node.go             # Node interface
│   │   ├── state.go            # Typed state management
│   │   ├── graph.go            # Graph execution
│   │   └── checkpoint.go       # Checkpointing
│   ├── llm/                    # LLM provider abstractions
│   │   ├── provider.go         # LLM client wrapper
│   │   └── config.go           # LLM configuration
│   ├── fsops/                  # File system operations
│   │   ├── safe_fs.go          # Bounded file ops
│   │   ├── patch.go            # Patch application
│   │   └── security.go         # Security checks
│   └── logging/                # Logging infrastructure
│       ├── logger.go           # Structured logging
│       └── execution_log.go    # Execution audit log
│
├── tests/                      # Test files
│   ├── unit/                   # Unit tests
│   ├── integration/            # Integration tests
│   └── contract/               # LLM provider contract tests
│
├── examples/                   # Example specifications
│   ├── simple-spec.yaml        # Minimal working example
│   └── medium-spec.yaml        # Realistic example
│
├── workflows/                  # Workflow definitions
│   ├── clarify.yaml            # Clarification workflow
│   ├── generate.yaml           # Generation workflow
│   └── validate.yaml           # Validation workflow
│
├── specs/001-core-implementation/  # Feature specification
│   ├── spec.md                 # User stories and requirements
│   ├── plan.md                 # Technical implementation plan
│   ├── tasks.md                # Actionable task list
│   └── quickstart.md           # Development quick start
│
├── docs/                       # Documentation
│   ├── ARCHITECTURE.md         # System architecture
│   └── DEVELOPMENT.md          # This file
│
├── go.mod                      # Module definition
├── go.sum                      # Dependency checksums
├── .golangci.yml               # Linter configuration
├── Makefile                    # Build automation
├── .gocreator.yaml             # Default configuration
├── .gitignore                  # Git ignore rules
└── README.md                   # Project overview
```

## Development Workflow

### Daily Development Cycle

```bash
# 1. Update from remote
git pull origin 001-core-implementation

# 2. Create feature branch (optional, for focused work)
git checkout -b feature/my-feature

# 3. Make changes to source files
# Edit files in internal/, pkg/, cmd/, etc.

# 4. Run tests (should pass)
make test

# 5. Run linter (should pass)
make lint

# 6. Run code review via mcp-pr
/review-unstaged
# or
/review-staged

# 7. Fix any review findings
# Edit files to address review comments

# 8. Commit changes
git add .
git commit -m "feat: description of changes"

# 9. Push to remote
git push origin feature/my-feature

# 10. Create pull request (if needed)
gh pr create --title "Title" --body "Description"
```

### Test-Driven Development (TDD)

GoCreator follows TDD principles. When adding features:

```bash
# 1. Write failing test
cat > internal/mypackage/myfeature_test.go <<'EOF'
package mypackage_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/dshills/gocreator/internal/mypackage"
)

func TestMyFeature(t *testing.T) {
    // Arrange
    input := "test input"

    // Act
    result, err := mypackage.MyFeature(input)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, "expected output", result)
}
EOF

# 2. Run tests (RED - should fail)
make test

# 3. Implement minimal code to pass
cat > internal/mypackage/myfeature.go <<'EOF'
package mypackage

func MyFeature(input string) (string, error) {
    return "expected output", nil
}
EOF

# 4. Run tests (GREEN - should pass)
make test

# 5. Refactor for quality
# Improve code without changing behavior

# 6. Run tests again (still GREEN)
make test

# 7. Commit with test
git add .
git commit -m "feat: implement MyFeature with tests"
```

### Table-Driven Tests Pattern

Always use table-driven tests for comprehensive coverage:

```go
func TestParserVariants(t *testing.T) {
    tests := []struct {
        name      string
        input     string
        wantErr   bool
        assertion func(t *testing.T, result *Type)
    }{
        {
            name:    "valid input",
            input:   "valid data",
            wantErr: false,
            assertion: func(t *testing.T, result *Type) {
                assert.Equal(t, "expected", result.Field)
            },
        },
        {
            name:    "empty input",
            input:   "",
            wantErr: true,
        },
        {
            name:    "invalid format",
            input:   "{{invalid}}",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Parse(tt.input)

            if tt.wantErr {
                assert.Error(t, err)
                return
            }

            assert.NoError(t, err)
            if tt.assertion != nil {
                tt.assertion(t, result)
            }
        })
    }
}
```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run specific package tests
go test -v ./internal/spec/...

# Run with coverage
make test-coverage

# Run integration tests
make test-integration

# Run contract tests (requires LLM provider)
go test -v -tags=contract ./tests/contract/...

# Run with race detector
go test -race ./...
```

### Test Coverage Requirements

- **Target**: 80% minimum coverage
- **Critical Paths**: 95% coverage for core packages (spec, clarify, generate, validate)
- **View HTML Report**:

```bash
make test-coverage
open coverage.html  # or use your browser
```

### Testing Principles

1. **Unit Tests**: Test individual functions in isolation
   - Mock external dependencies
   - Test both happy and error paths
   - Use table-driven tests

2. **Integration Tests**: Test components working together
   - Use temporary directories for file operations
   - Test complete workflows
   - Verify side effects (files created, etc.)

3. **Contract Tests**: Test external interface contracts
   - Test LLM provider interactions
   - Test file system operations
   - Use build tags: `//go:build contract`

## Code Quality

### Linting

```bash
# Run linter
make lint

# Run linter with auto-fix
make lint-fix

# Run security scanner
make sec

# Format code
make fmt

# All quality checks (linter + fmt)
make lint && make fmt
```

### Code Review (mcp-pr)

All code changes require review before committing:

```bash
# Review unstaged changes
/review-unstaged

# Review staged changes
/review-staged

# Review specific commit
/review-commit abc123def

# Address review findings, then commit
git add .
git commit -m "fix: address code review findings"
```

### Godoc Comments

All public types and functions must have godoc comments:

```go
// Package mypackage provides functionality for X.
package mypackage

// MyType represents a thing.
type MyType struct {
    Field string
}

// NewMyType creates a new MyType instance.
func NewMyType(value string) *MyType {
    return &MyType{Field: value}
}

// Process performs some operation on the type.
func (m *MyType) Process() error {
    return nil
}
```

### Code Style Guidelines

1. **Error Handling**: Always check and handle errors
   ```go
   // Good
   data, err := os.ReadFile(path)
   if err != nil {
       return fmt.Errorf("read file: %w", err)
   }

   // Bad
   data, _ := os.ReadFile(path)
   ```

2. **Interfaces**: Keep interfaces small and focused
   ```go
   // Good
   type Reader interface {
       Read(ctx context.Context) ([]byte, error)
   }

   // Avoid
   type AllTheThings interface {
       Read() ([]byte, error)
       Write([]byte) error
       Delete() error
       Process() error
       // ... lots more
   }
   ```

3. **Package Structure**: Organize by domain concept, not by type
   ```
   // Good
   pkg/clarify/
   ├── analyzer.go
   ├── questions.go
   └── engine.go

   // Avoid
   pkg/types/
   ├── analyzer.go
   pkg/questions/
   ├── generator.go
   ```

4. **Dependency Injection**: Pass dependencies as parameters
   ```go
   // Good
   func NewEngine(llmClient llm.Client, logger Logger) *Engine {
       return &Engine{llm: llmClient, log: logger}
   }

   // Avoid
   func NewEngine() *Engine {
       return &Engine{llm: createGlobalLLMClient(), log: globalLogger}
   }
   ```

5. **Constants and Enums**: Use typed constants for safety
   ```go
   // Good
   type SpecFormat string
   const (
       FormatYAML     SpecFormat = "yaml"
       FormatJSON     SpecFormat = "json"
       FormatMarkdown SpecFormat = "markdown"
   )

   // Avoid
   const (
       FormatYAML     = "yaml"
       FormatJSON     = "json"
       FormatMarkdown = "markdown"
   )
   ```

## Common Tasks

### Add a New Dependency

```bash
# Add to go.mod
go get github.com/org/package@version

# Verify and clean up
go mod tidy

# Verify integrity
go mod verify
```

### Generate Mocks for Testing

```bash
# Install mockgen
go install github.com/golang/mock/mockgen@latest

# Generate mock
mockgen -source=internal/pkg/interface.go \
    -destination=tests/mocks/interface_mock.go \
    -package=mocks

# Use in tests
func TestWithMock(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockClient := mocks.NewMockClient(ctrl)
    mockClient.EXPECT().
        SomeMethod().
        Return("expected", nil)

    // Test code using mockClient
}
```

### Profile Performance

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=. ./internal/generate/...

# Memory profiling
go test -memprofile=mem.prof -bench=. ./internal/generate/...

# Analyze profile
go tool pprof cpu.prof
# At pprof prompt: top10, list functionName, web, etc.
```

### Run Benchmark Tests

```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkParse ./internal/spec/...

# Run with memory stats
go test -bench=. -benchmem ./...
```

### Update Documentation

Documentation files are maintained in the `docs/` directory:

- `README.md` - User-facing overview and CLI reference
- `ARCHITECTURE.md` - System architecture and design
- `DEVELOPMENT.md` - This file, development guidelines
- `specs/001-core-implementation/` - Feature specifications and plans

To update documentation:

```bash
# Edit file
vim docs/ARCHITECTURE.md

# Verify it renders correctly
# Commit with documentation changes
git add docs/
git commit -m "docs: update architecture documentation"
```

## Contributing Guidelines

### Code Organization Principles

1. **Package Responsibility**: Each package has a single, clear responsibility
   - `internal/clarify/` - Only clarification logic
   - `internal/generate/` - Only generation logic
   - `pkg/fsops/` - Only file operations

2. **Separation of Concerns**: Keep reasoning (LangGraph) separate from execution (GoFlow)
   - Reasoning packages: `clarify/`, `generate/`
   - Execution packages: `workflow/`, `validate/`

3. **Error Handling**: Wrap errors with context
   ```go
   if err != nil {
       return fmt.Errorf("context about operation: %w", err)
   }
   ```

4. **Logging**: Use structured logging consistently
   ```go
   log.Info().
       Str("file", path).
       Int("lines", count).
       Msg("File processed")
   ```

### Commit Message Format

Follow conventional commits format:

```
feat: add new feature
fix: fix a bug
docs: update documentation
test: add or update tests
refactor: refactor without behavior changes
perf: performance improvements
chore: maintenance tasks
```

Example:
```
feat: implement FCS builder with validation

- Add FCS construction from spec + clarifications
- Add SHA-256 checksums for integrity verification
- Add comprehensive tests for builder
- Add documentation to ARCHITECTURE.md

Closes #123
```

### Pull Request Checklist

Before submitting a pull request:

- [ ] All tests pass (`make test`)
- [ ] Linting passes (`make lint`)
- [ ] Code review passed (`/review-staged`)
- [ ] New tests added for new functionality
- [ ] Documentation updated if needed
- [ ] Godoc comments added for public APIs
- [ ] Security considerations addressed
- [ ] No debug logging or temporary code left in
- [ ] Commits follow conventional format

### Adding a New Feature

1. **Plan**: Check existing design in `ARCHITECTURE.md`
2. **Spec**: Create specification in `specs/`
3. **Design**: Document in `plan.md`
4. **Implement**:
   - Write tests first (TDD)
   - Implement minimal code to pass tests
   - Refactor for quality
   - Add godoc comments
5. **Review**: Submit for code review
6. **Integrate**: Merge to main branch once approved

## Troubleshooting

### Common Issues

#### Build Failures

```bash
# Clear module cache
go clean -modcache

# Re-download dependencies
go mod download

# Verify module
go mod verify

# Rebuild
go build ./cmd/gocreator
```

#### Test Failures

```bash
# Run with verbose output
go test -v ./...

# Run with race detector
go test -race ./...

# Run specific test
go test -run TestName ./internal/package/...

# Check test output
go test -v 2>&1 | grep -A 10 FAIL
```

#### Linting Issues

```bash
# See what linter is complaining
make lint

# Auto-fix simple issues
make lint-fix

# Check specific file
golangci-lint run ./internal/myfile.go

# Ignore specific issue (last resort)
// nolint:errorlint
```

#### Import Issues

```bash
# Format imports
go fmt ./...
goimports -w .

# Verify imports
go mod tidy
```

### Performance Debugging

```bash
# Run with debug logging
gocreator generate ./spec.yaml --log-level=debug

# Check execution log
cat .gocreator/execution.jsonl | jq '.event,.ts'

# Profile memory usage
go test -memprofile=mem.prof ./...
go tool pprof mem.prof
```

### LLM Provider Issues

```bash
# Verify API key
echo $ANTHROPIC_API_KEY

# Test LLM connectivity
curl -X POST https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{
    "model": "claude-sonnet-4",
    "max_tokens": 10,
    "messages": [{"role": "user", "content": "test"}]
  }'
```

## Resources

### Documentation
- [Go Documentation](https://go.dev/doc/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Project Architecture](ARCHITECTURE.md)
- [Project Specification](../specs/001-core-implementation/spec.md)

### Libraries Used
- [langchaingo](https://github.com/tmc/langchaingo) - LLM abstractions
- [cobra](https://github.com/spf13/cobra) - CLI framework
- [viper](https://github.com/spf13/viper) - Configuration
- [zerolog](https://github.com/rs/zerolog) - Logging
- [testify](https://github.com/stretchr/testify) - Testing utilities
- [go-diff](https://github.com/sergi/go-diff) - Diff operations

### Tools
- [golangci-lint](https://golangci-lint.run/) - Linting
- [gosec](https://github.com/securego/gosec) - Security scanning
- [goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports) - Import formatting

## Getting Help

1. **Check Documentation**
   - [README.md](../README.md) - User guide
   - [ARCHITECTURE.md](ARCHITECTURE.md) - System design
   - [spec.md](../specs/001-core-implementation/spec.md) - Feature requirements

2. **Review Existing Code**
   - Look at similar implementations in existing packages
   - Check test files for usage examples

3. **Run with Debug Logging**
   ```bash
   gocreator --log-level=debug ...
   cat .gocreator/execution.jsonl | jq
   ```

4. **Check Latest Changes**
   ```bash
   git log --oneline -20
   git diff HEAD~1
   ```

## Future Development

### Planned Features (P4-P5)
- Incremental regeneration (Phase 6)
- Performance optimizations (Phase 8)
- Additional examples and documentation (Phase 8)
- Release automation (Phase 8)

See [tasks.md](../specs/001-core-implementation/tasks.md) for detailed roadmap.
