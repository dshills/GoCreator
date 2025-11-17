# Quick Start Guide: GoCreator Core Implementation

**Branch**: `001-core-implementation` | **Date**: 2025-11-17
**Purpose**: Get development environment set up and start implementing

## Prerequisites

### Required Tools

- **Go 1.21+**: Download from [go.dev](https://go.dev/dl/)
- **Git**: Version control
- **Make**: Build automation (usually pre-installed on macOS/Linux)
- **golangci-lint**: Linting tool
  ```bash
  # Install golangci-lint
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  ```

### Optional Tools

- **gosec**: Security scanner
  ```bash
  go install github.com/securego/gosec/v2/cmd/gosec@latest
  ```
- **goimports**: Import management
  ```bash
  go install golang.org/x/tools/cmd/goimports@latest
  ```

### LLM Provider Access

You'll need access to an LLM provider (Anthropic, OpenAI, or Google):

```bash
# Set your API key (choose one)
export ANTHROPIC_API_KEY=sk-ant-...
# or
export OPENAI_API_KEY=sk-...
# or
export GOOGLE_API_KEY=...
```

---

## Initial Setup

### 1. Clone and Initialize

```bash
# You're already in the repo from the feature branch setup
cd /Users/dshills/Development/projects/GoCreator

# Ensure we're on the right branch
git checkout 001-core-implementation

# Initialize Go module (if not already done)
go mod init github.com/dshills/gocreator

# Install dependencies (once we have go.mod populated)
go mod tidy
```

### 2. Project Structure Setup

Create the initial directory structure:

```bash
# Create main directories
mkdir -p cmd/gocreator
mkdir -p internal/{spec,clarify,generate,workflow,validate,models,config}
mkdir -p pkg/{langgraph,llm,fsops}
mkdir -p tests/{unit,integration,contract}

# Create placeholder files to preserve structure
touch cmd/gocreator/main.go
touch internal/spec/.gitkeep
touch internal/clarify/.gitkeep
touch internal/generate/.gitkeep
touch internal/workflow/.gitkeep
touch internal/validate/.gitkeep
touch internal/models/.gitkeep
touch internal/config/.gitkeep
touch pkg/langgraph/.gitkeep
touch pkg/llm/.gitkeep
touch pkg/fsops/.gitkeep
```

### 3. Configuration Files

Create `.golangci.yml`:

```yaml
run:
  timeout: 5m
  tests: true
  modules-download-mode: readonly

linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gofmt
    - goimports
    - gosec
    - goconst
    - misspell
    - unparam

linters-settings:
  errcheck:
    check-blank: true
  govet:
    check-shadowing: true
  gofmt:
    simplify: true

issues:
  exclude-use-default: false
  max-issues-per-linter: 0
  max-same-issues: 0
```

Create `Makefile`:

```makefile
.PHONY: help build test lint clean install

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	go build -o bin/gocreator ./cmd/gocreator

install: ## Install the binary
	go install ./cmd/gocreator

test: ## Run tests
	go test -v -race -cover ./...

test-integration: ## Run integration tests
	go test -v -race -tags=integration ./tests/integration/...

test-coverage: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint: ## Run linter
	golangci-lint run ./...

lint-fix: ## Run linter with auto-fix
	golangci-lint run --fix ./...

sec: ## Run security scanner
	gosec ./...

fmt: ## Format code
	gofmt -s -w .
	goimports -w .

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

deps: ## Download dependencies
	go mod download
	go mod tidy

.DEFAULT_GOAL := help
```

Create `.gocreator.yaml` (for testing GoCreator itself):

```yaml
llm:
  provider: anthropic
  model: claude-sonnet-4
  temperature: 0.0
  api_key: ${ANTHROPIC_API_KEY}

workflow:
  root_dir: ./generated
  allow_commands:
    - go
    - git
    - golangci-lint
  max_parallel: 4

validation:
  enable_linting: true
  linter_config: .golangci.yml
  enable_tests: true
  test_timeout: 5m

logging:
  level: info
  format: console
  output: stderr
  execution_log: .gocreator/execution.jsonl
```

---

## Development Workflow

### Daily Workflow

```bash
# 1. Pull latest changes
git pull origin 001-core-implementation

# 2. Create a feature branch (optional, for sub-features)
git checkout -b feature/spec-parser

# 3. Make changes...

# 4. Run tests
make test

# 5. Run linter
make lint

# 6. Run mcp-pr code review (constitution requirement)
# Use the /review-unstaged or /review-staged slash commands

# 7. Commit changes
git add .
git commit -m "feat: implement spec parser with YAML support"

# 8. Push changes
git push origin feature/spec-parser
```

### Test-Driven Development (TDD)

Following constitution principle IV (Test-First Discipline):

```bash
# 1. Write tests first
cat > internal/spec/parser_test.go <<'EOF'
package spec_test

import (
    "testing"
    "github.com/dshills/gocreator/internal/spec"
)

func TestParseYAMLSpec(t *testing.T) {
    tests := []struct{
        name string
        input string
        want *spec.InputSpecification
        wantErr bool
    }{
        {
            name: "valid YAML spec",
            input: `
name: test-project
description: A test project
requirements:
  functional: []
`,
            want: &spec.InputSpecification{
                Format: "yaml",
                // ...
            },
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := spec.Parse(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            // assertions...
        })
    }
}
EOF

# 2. Run tests (they should fail - RED)
make test

# 3. Implement minimal code to pass tests - GREEN
# Edit internal/spec/parser.go

# 4. Run tests again (they should pass)
make test

# 5. Refactor - improve code without changing behavior
# Edit internal/spec/parser.go

# 6. Run tests again (ensure still passing)
make test
```

---

## Implementation Order

Follow user story priorities from spec.md:

### Phase 1: P1 - Specification Clarification and FCS Generation

**Packages to implement**:
1. `internal/models` - Define all domain models from data-model.md
2. `internal/spec` - Specification parsing and validation
3. `internal/clarify` - Clarification engine with LangGraph-Go
4. `pkg/llm` - LLM provider wrapper
5. `pkg/langgraph` - LangGraph-Go execution engine

**Milestone**: Can load a spec, identify ambiguities, ask questions, generate FCS

```bash
# Test it
gocreator clarify ./examples/sample-spec.yaml
```

### Phase 2: P2 - Autonomous Code Generation from FCS

**Packages to implement**:
1. `internal/generate` - Generation engine with LangGraph-Go
2. `internal/workflow` - GoFlow workflow engine
3. `pkg/fsops` - Safe file system operations

**Milestone**: Can generate a complete Go project from FCS

```bash
# Test it
gocreator generate ./examples/sample-spec.yaml --output ./test-output
```

### Phase 3: P3 - Validation and Quality Assurance

**Packages to implement**:
1. `internal/validate` - Build, lint, test validation

**Milestone**: Can validate generated projects

```bash
# Test it
gocreator validate ./test-output
```

### Phase 4: P4 & P5 - CLI and Workflow

**Packages to implement**:
1. `cmd/gocreator` - CLI with cobra
2. `internal/config` - Configuration management

**Milestone**: Full CLI with all commands working

```bash
# Test it
gocreator full ./examples/sample-spec.yaml
```

---

## Testing Strategy

### Unit Tests

```bash
# Run all unit tests
make test

# Run specific package tests
go test -v ./internal/spec/...

# Run with coverage
make test-coverage
```

### Integration Tests

```bash
# Run integration tests
make test-integration
```

### Contract Tests

For LLM provider interactions:

```bash
# Run contract tests (requires LLM provider access)
go test -v -tags=contract ./tests/contract/...
```

---

## Debugging

### Enable Debug Logging

```bash
# Run with debug logs
gocreator generate ./spec.yaml --log-level=debug
```

### Inspect Execution Logs

```bash
# View execution log
cat ./generated/.gocreator/execution.jsonl | jq
```

### Resume from Checkpoint

```bash
# If generation fails, resume from last checkpoint
gocreator generate ./spec.yaml --resume
```

---

## Code Quality Gates

Before committing, ensure all gates pass:

### 1. Tests Pass

```bash
make test
# Should see: PASS
```

### 2. Linting Passes

```bash
make lint
# Should see: no issues found
```

### 3. Code Review (mcp-pr with OpenAI)

```bash
# Use Claude Code slash command
/review-unstaged
# or
/review-staged
```

### 4. Security Scan (Optional but Recommended)

```bash
make sec
# Review any security findings
```

---

## Common Tasks

### Add a New Dependency

```bash
# Add dependency
go get github.com/some/package@version

# Tidy dependencies
go mod tidy

# Verify
go mod verify
```

### Generate Mocks (for testing)

```bash
# Install mockgen
go install github.com/golang/mock/mockgen@latest

# Generate mock for interface
mockgen -source=internal/spec/parser.go -destination=tests/mocks/parser_mock.go
```

### Profile Performance

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=. ./internal/generate/...

# View profile
go tool pprof cpu.prof
```

---

## Troubleshooting

### Go Build Issues

```bash
# Clear module cache
go clean -modcache

# Re-download dependencies
go mod download
```

### Linter False Positives

Add to `.golangci.yml` under `issues.exclude-rules`:

```yaml
issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck
```

### LLM Provider Issues

```bash
# Verify API key is set
echo $ANTHROPIC_API_KEY

# Test API access
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "content-type: application/json" \
  -d '{"model":"claude-sonnet-4","messages":[{"role":"user","content":"test"}],"max_tokens":10}'
```

---

## Resources

### Documentation
- [Go Documentation](https://go.dev/doc/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### Libraries
- [langchaingo](https://github.com/tmc/langchaingo)
- [cobra](https://github.com/spf13/cobra)
- [viper](https://github.com/spf13/viper)
- [zerolog](https://github.com/rs/zerolog)

### Tools
- [golangci-lint](https://golangci-lint.run/)
- [gosec](https://github.com/securego/gosec)
- [goreleaser](https://goreleaser.com/)

---

## Next Steps

1. Review `spec.md` for user stories and requirements
2. Review `data-model.md` for domain entities
3. Review `contracts/cli-interface.md` for CLI contract
4. Start implementing P1 (Specification Clarification) following TDD
5. Run `/speckit.tasks` to generate detailed task list
