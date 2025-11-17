# Research: GoCreator Core Implementation

**Branch**: `001-core-implementation` | **Date**: 2025-11-17
**Purpose**: Resolve technology choices and clarify implementation approaches

## Overview

This document resolves the "NEEDS CLARIFICATION" items from the Technical Context, researches best practices for the chosen technologies, and provides rationale for key decisions.

## Technology Decisions

### 1. LangGraph-Go Library Selection

**Decision**: Use `github.com/tmc/langchaingo` with custom LangGraph-Go implementation

**Rationale**:
- langchaingo provides LLM provider abstractions (OpenAI, Anthropic, etc.)
- No mature LangGraph-Go library exists yet; we'll implement graph execution ourselves
- Allows full control over state management, checkpointing, and deterministic execution
- Simpler than porting Python LangGraph to Go
- Aligns with GoCreator's determinism requirements

**Alternatives Considered**:
- **Port Python LangGraph**: Rejected due to complexity and maintenance burden
- **Use Python LangGraph via RPC**: Rejected due to performance overhead and deployment complexity
- **Wait for official LangGraph-Go**: Rejected due to uncertain timeline

**Implementation Approach**:
- Build graph execution engine in `internal/clarify/graph.go` and `internal/generate/graph.go`
- Use typed Go structs for state (no dynamic maps)
- Implement checkpointing with JSON serialization
- Support concurrent node execution where dependencies allow

---

### 2. GoFlow Workflow Engine

**Decision**: Build custom workflow engine tailored to GoCreator needs

**Rationale**:
- Existing Go workflow engines (Temporal, Cadence) are too heavyweight for CLI use
- Need tight control over determinism and provenance logging
- Simple YAML-based workflow definitions sufficient for our use case
- Can optimize for file operations and shell command execution

**Alternatives Considered**:
- **Temporal**: Rejected due to server dependency and complexity
- **Cadence**: Rejected for same reasons as Temporal
- **Argo Workflows**: Rejected as it's designed for Kubernetes, not CLI
- **go-task (Task runner)**: Considered but lacks checkpointing and state management

**Implementation Approach**:
- YAML workflow definitions in `internal/workflow/tasks.go`
- DAG-based execution with dependency resolution
- Parallel execution using goroutines with worker pools
- Execution log with structured JSON output
- Support for retries, timeouts, and error handling

---

### 3. LLM Client Library

**Decision**: Use `github.com/tmc/langchaingo` LLM abstractions

**Rationale**:
- Provides unified interface for multiple LLM providers (OpenAI, Anthropic, Google, etc.)
- Handles authentication, retries, and rate limiting
- Active development and community support
- Type-safe Go API

**Alternatives Considered**:
- **Direct OpenAI SDK**: Rejected due to vendor lock-in
- **Direct Anthropic SDK**: Same issue as OpenAI
- **Custom HTTP clients**: Rejected due to maintenance burden

**Implementation Approach**:
- Wrap langchaingo in `pkg/llm/` for GoCreator-specific config
- Support configurable temperature for determinism (default: 0.0)
- Implement token counting and cost tracking
- Add request/response logging for debugging

---

### 4. Specification Format Support

**Decision**: Support YAML, JSON, and Markdown with structured frontmatter

**Rationale**:
- YAML: Human-friendly, widely used for config files
- JSON: Machine-readable, easy to generate programmatically
- Markdown with frontmatter: Combines documentation with structured data

**Implementation Approach**:
- Use `gopkg.in/yaml.v3` for YAML parsing
- Use `encoding/json` for JSON parsing
- Use `github.com/yuin/goldmark` with frontmatter extension for Markdown
- Normalize all formats to common `models.InputSpecification` struct

---

### 5. Patch Generation and Application

**Decision**: Use unified diff format with `github.com/sergi/go-diff`

**Rationale**:
- Unified diff is human-readable and widely supported
- Can be versioned and reviewed
- Supports reversibility (applying diffs in reverse)
- Standard format understood by git

**Implementation Approach**:
- LangGraph-Go outputs generate patch strings
- GoFlow applies patches using `go-diff/diffmatchpatch`
- Log all patches before application
- Support dry-run mode for validation

---

### 6. Execution Logging and Provenance

**Decision**: Structured JSON logging with `github.com/rs/zerolog`

**Rationale**:
- High-performance structured logging
- Zero allocations in hot paths
- Human-readable console output with color
- Machine-parseable JSON for analysis

**Implementation Approach**:
- Log all LangGraph-Go decisions (what/why)
- Log all GoFlow operations (file writes, commands)
- Include timestamps, operation IDs, and context
- Support log levels (debug, info, warn, error)
- Write execution log to `<output-dir>/.gocreator/execution.jsonl`

---

### 7. Configuration Management

**Decision**: Layered configuration (file → env → CLI flags)

**Rationale**:
- Flexibility for different use cases (local dev, CI, production)
- Industry standard approach (12-factor app principles)
- Easy to override for testing

**Configuration Sources** (priority order):
1. CLI flags (highest priority)
2. Environment variables (`GOCREATOR_*` prefix)
3. Config file (`.gocreator.yaml` in project root or `~/.config/gocreator/config.yaml`)
4. Built-in defaults

**Configuration Structure**:
```yaml
llm:
  provider: anthropic  # or openai, google, etc.
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
  format: console  # or json
  output: stderr
  execution_log: .gocreator/execution.jsonl
```

**Implementation**:
- Use `github.com/spf13/viper` for configuration management
- Use `github.com/spf13/cobra` for CLI framework

---

### 8. CLI Framework

**Decision**: Use `github.com/spf13/cobra` for CLI structure

**Rationale**:
- Industry standard for Go CLIs (used by kubectl, hugo, etc.)
- Excellent subcommand support
- Auto-generated help and documentation
- Flag binding with viper

**CLI Command Structure**:
```
gocreator
├── clarify <spec-file>      # Run clarification only
├── generate <spec-file>     # Clarification + generation (skip validation)
├── validate <project-root>  # Validation only
├── full <spec-file>         # Complete pipeline
├── dump-fcs <spec-file>     # Output FCS as JSON
└── version                  # Show version info
```

---

### 9. FCS Storage Format

**Decision**: JSON with schema validation

**Rationale**:
- Machine-readable and parseable
- Schema validation ensures completeness
- Easy to version and diff
- Can be used as input for subsequent runs

**FCS Schema**:
```json
{
  "version": "1.0",
  "metadata": {
    "original_spec": "path/to/spec.yaml",
    "created_at": "2025-11-17T10:30:00Z",
    "clarifications": [...]
  },
  "requirements": {
    "functional": [...],
    "nonfunctional": [...]
  },
  "architecture": {
    "packages": [...],
    "dependencies": [...],
    "patterns": [...]
  },
  "data_model": {...},
  "api_contracts": {...},
  "testing_strategy": {...},
  "build_config": {...}
}
```

**Implementation**:
- Use `encoding/json` for serialization
- Use JSON Schema for validation (`github.com/xeipuuv/gojsonschema`)
- Store FCS in `<output-dir>/.gocreator/fcs.json`

---

### 10. Checkpoint and Recovery

**Decision**: Checkpoint after each major phase

**Rationale**:
- Allows resuming long-running generation
- Supports debugging and inspection
- Enables incremental regeneration

**Checkpoint Points**:
1. After clarification (FCS constructed)
2. After architectural planning
3. After each package generation
4. After test generation
5. Before validation

**Implementation**:
- Store checkpoints in `<output-dir>/.gocreator/checkpoints/`
- Each checkpoint is a JSON file with full state snapshot
- Support `--resume` flag to continue from last checkpoint

---

## Best Practices

### Go Project Layout
- Follow [golang-standards/project-layout](https://github.com/golang-standards/project-layout)
- Use `internal/` for private code
- Use `pkg/` for public libraries
- Use `cmd/` for CLI entry points

### Concurrency Patterns
- Use goroutines with `errgroup` for parallel task execution
- Implement worker pools for bounded concurrency
- Use channels for coordination
- Prefer sync.Map for concurrent read/write to shared state

### Error Handling
- Wrap errors with context using `fmt.Errorf` with `%w`
- Create custom error types for domain errors
- Use sentinel errors for expected error conditions
- Log errors before returning them up the stack

### Testing Strategy
- Table-driven tests for unit tests
- Use `testify/assert` for assertions
- Use `testify/mock` for mocking interfaces
- Integration tests use temporary directories
- Contract tests validate LLM provider interactions

### Code Generation Patterns
- Use `text/template` for simple templates
- Use AST manipulation (`go/ast`, `go/parser`) for complex code
- Validate generated code before writing files
- Run `gofmt` on all generated code

---

## Integration Patterns

### LLM Provider Integration
- Abstract provider behind interface
- Support multiple providers via configuration
- Implement circuit breaker for failures
- Cache responses for determinism (optional)

### File System Operations
- All operations go through `pkg/fsops` abstraction
- Enforce root directory boundaries
- Create atomic writes using temp files + rename
- Log all file operations

### Shell Command Execution
- Whitelist allowed commands
- Capture stdout/stderr
- Set timeouts
- Run in bounded goroutines

---

## Performance Optimizations

### Concurrency
- Parallel package generation (no dependencies)
- Parallel test execution
- Concurrent linting of multiple files

### Caching
- Cache LLM responses (optional, for development)
- Cache parsed specs between runs
- Reuse unchanged portions of previous generation

### Incremental Generation
- Detect changed requirements in spec
- Only regenerate affected packages
- Preserve manual modifications outside generated blocks

---

## Security Considerations

### Input Validation
- Validate all spec inputs against schema
- Sanitize paths to prevent directory traversal
- Reject specs requesting unauthorized operations

### Command Execution
- Strict command whitelist
- No shell interpretation (use exec directly)
- Validate all command arguments

### File Operations
- Enforce root directory boundaries
- Reject absolute paths outside root
- Validate file permissions before writes

---

## Dependencies Summary

### Core Dependencies
- `github.com/tmc/langchaingo` - LLM provider abstractions
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration management
- `github.com/rs/zerolog` - Structured logging
- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/yuin/goldmark` - Markdown parsing
- `github.com/sergi/go-diff` - Patch generation/application
- `github.com/xeipuuv/gojsonschema` - JSON schema validation

### Testing Dependencies
- `github.com/stretchr/testify` - Test assertions and mocks
- `github.com/google/go-cmp` - Deep equality comparison
- `github.com/matryer/is` - Minimal test assertions

### Development Dependencies
- `golang.org/x/tools/cmd/goimports` - Import management
- `github.com/golangci/golangci-lint` - Linting
- `github.com/goreleaser/goreleaser` - Release automation

---

## Next Steps

With all technical decisions resolved, proceed to Phase 1:
1. Generate `data-model.md` defining all domain entities
2. Generate API contracts (if applicable - may be N/A for CLI)
3. Generate `quickstart.md` for development setup
4. Update agent context with chosen technologies
