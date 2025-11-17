# Changelog

All notable changes to the GoCreator project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned Features

- **Phase 6: Specification Update and Regeneration** - Incremental regeneration and caching for modified specifications
- **Performance Optimizations** - LLM response caching and parallel generation optimization
- **Additional Documentation** - Extended architecture and development guides
- **Example Specifications** - Complete sample projects demonstrating GoCreator capabilities
- **Security Scanner Integration** - Automated security scanning in validation pipeline
- **Test Coverage Analysis** - Enhanced test coverage reporting and enforcement

## [0.1.0] - 2025-01-17

### Initial Release

The initial release of GoCreator marks the completion of the core specification-driven code generation system. This version implements the foundational architecture supporting autonomous Go project generation from structured specifications.

### Features

#### Phase 1: Project Foundation
- **Go Module Initialization** - Properly configured Go 1.25.4 module with semantic versioning support
- **Project Structure** - Organized package layout with separation of concerns:
  - `cmd/gocreator/` - CLI application
  - `internal/` - Private domain logic (spec, clarify, generate, validate, workflow, models, config)
  - `pkg/` - Public library packages (langgraph, llm, fsops, logging)
  - `tests/` - Comprehensive test suite (unit, integration, contract)
- **Build Infrastructure** - Makefile with build, test, lint, clean, and coverage targets
- **Configuration Management** - YAML-based configuration with environment variable and CLI flag support
- **Dependency Management** - Complete Go module with pinned dependency versions

#### Phase 2: Core Domain Models
- **Specification Models** (`internal/models/spec.go`) - InputSpecification struct with validation
- **Clarification Models** (`internal/models/clarification.go`) - Request/response structures for ambiguity resolution
- **FCS Models** (`internal/models/fcs.go`) - Final Clarified Specification with immutability guarantees
- **Generation Models** (`internal/models/generation.go`) - GenerationPlan and GenerationOutput structures
- **Validation Models** (`internal/models/validation.go`) - ValidationReport, BuildResult, LintResult, TestResult
- **Workflow Models** (`internal/models/workflow.go`) - WorkflowDefinition and WorkflowExecution
- **Logging Models** (`internal/models/log.go`) - ExecutionLog and specialized log entries for decisions and operations

#### Phase 3: Specification Processing (User Story 1)
- **Specification Parsers** - Multi-format parser supporting:
  - YAML format parsing (`internal/spec/parser_yaml.go`)
  - JSON format parsing (`internal/spec/parser_json.go`)
  - Markdown with frontmatter parsing (`internal/spec/parser_md.go`)
  - Unified Parse() function with automatic format detection
- **Specification Validator** (`internal/spec/validator.go`) - Validates syntax, schema, and required fields
- **Specification Model Tests** - Table-driven unit tests for all model types
- **LLM Provider Integration** (`pkg/llm/provider.go`) - LangChain-Go integration with temperature control and token tracking
- **LangGraph-Go Engine** (`pkg/langgraph/`) - Custom Go implementation of LangGraph:
  - Node execution interface with typed state
  - State management without dynamic maps
  - Graph execution engine with DAG traversal
  - Checkpointing with JSON serialization
- **Ambiguity Analyzer** (`internal/clarify/analyzer.go`) - Identifies gaps, conflicts, and unclear requirements
- **Clarification Question Generator** (`internal/clarify/questions.go`) - Creates targeted clarification questions
- **Clarification Graph** (`internal/clarify/graph.go`) - LangGraph-based state machine for clarification workflow
- **FCS Builder** (`internal/spec/fcs_builder.go`) - Merges specification with clarification answers into FCS
- **FCS Hash Generation** (`internal/spec/fcs_hash.go`) - SHA-256 integrity verification
- **Full Integration Test Suite** - Tests covering ambiguous specs, well-formed specs, and conflict resolution

#### Phase 4: Code Generation (User Story 2)
- **Architectural Planner** (`internal/generate/planner.go`) - Transforms FCS into package structure and file tree
- **Generation Plan Builder** (`internal/generate/plan_builder.go`) - Creates GenerationPlan with phases and tasks
- **Code Synthesizer** (`internal/generate/coder.go`) - Generates Go code using templates and AST manipulation
- **Test Generator** (`internal/generate/tester.go`) - Creates unit, integration, and contract tests
- **Generation LangGraph** (`internal/generate/graph.go`) - State machine for generation workflow
- **GoFlow Workflow Engine** (`internal/workflow/`) - Deterministic execution engine:
  - Task definitions for file operations, shell commands, and LangGraph calls
  - YAML workflow parser and DAG executor
  - Unified diff patch application
  - Parallel task execution with worker pools
  - Execution logging with decision and operation tracking
- **Checksum Generation** (`internal/generate/checksum.go`) - SHA-256 for all generated files ensuring determinism
- **Workflow Definitions** - YAML specifications for clarify, generate, and validate workflows
- **Complete Integration Test Suite** - Tests covering full generation, byte-for-byte determinism, and performance

#### Phase 5: Validation (User Story 3)
- **Build Validator** (`internal/validate/build.go`) - Executes `go build` and captures errors
- **Lint Validator** (`internal/validate/lint.go`) - Runs `golangci-lint` and parses output
- **Test Validator** (`internal/validate/test.go`) - Executes `go test` with coverage capture
- **Validation Report Generator** (`internal/validate/report.go`) - Aggregates all validation results
- **Full Integration Test Suite** - Tests covering successful validation, build errors, lint failures, and test failures

#### Phase 7: CLI Operations (User Story 5)
- **Cobra-Based CLI Framework** (`cmd/gocreator/`) - Professional command-line interface with:
  - `gocreator clarify` - Specification analysis and clarification
  - `gocreator generate` - Code generation with clarification
  - `gocreator validate` - Codebase validation
  - `gocreator full` - Complete end-to-end pipeline
  - `gocreator dump-fcs` - FCS output in JSON format
  - `gocreator version` - Version and build information
- **Global Configuration Flags** - Config file, log level, and log format controls
- **Exit Code Mapping** - Proper exit codes for all error conditions
- **Interactive Mode** (`cmd/gocreator/interactive.go`) - Display clarification questions and collect answers
- **Batch Mode** (`cmd/gocreator/batch.go`) - Pre-answered question JSON parser
- **Progress Reporting** (`cmd/gocreator/progress.go`) - Console output with status updates
- **Full Integration Test Suite** - Tests for each CLI command and error handling

### Architecture

- **Specification as Source of Truth** - All generated code traces back to human-authored specifications
- **Separation of Reasoning and Action** - LangGraph-Go handles planning/design, GoFlow handles file operations
- **Deterministic Execution** - Identical specifications produce identical output
- **Immutable FCS** - Final Clarified Specification cannot be modified during generation
- **Bounded File Operations** - All file I/O restricted to configured project root with symlink protection
- **Comprehensive Logging** - All decisions, operations, and file writes logged for auditability
- **Error Isolation** - Validation failures do not trigger automatic repairs; users modify specs and re-run

### Quality & Security

#### Code Quality
- **Comprehensive Test Suite** - 183 unit and integration tests across all packages
- **Linting** - golangci-lint configuration with strict settings for code quality
- **Code Review** - All code reviewed via mcp-pr with findings addressed
- **Package Documentation** - Full godoc comments on all public APIs
- **Error Handling** - Custom error types with proper wrapping and context

#### Security Hardening
- **File Permission Controls** - Sensitive files stored with restricted permissions (0600/0750)
- **Path Traversal Prevention** - Validation prevents directory traversal attacks
- **Command Whitelist** - Shell command execution limited to approved commands
- **Symlink Protection** - File operations detect and reject symlink traversal
- **Bounded Operations** - All file operations cannot escape configured project root

### Technologies

- **Language**: Go 1.25.4+ (requires generics support)
- **CLI Framework**: Cobra v1.10.1
- **Configuration**: Viper v1.21.0
- **Logging**: Zerolog v1.34.0
- **LLM Integration**: LangChain-Go v0.1.14
- **Testing**: Testify v1.11.1
- **Build System**: GNU Make with multi-target support

### Project Management

- **Specification-Driven Development** - Features defined through technology-agnostic specifications before implementation
- **Phase-Based Planning** - 8 phases with clear completion milestones
- **Task Organization** - 183 tasks organized by user story and dependency
- **Parallel Execution** - ~60 tasks identified for concurrent development
- **Test-First Discipline** - All generated code includes comprehensive tests

### Documentation

- **CLAUDE.md** - Project guidance for AI-assisted development
- **CLARIFICATION_IMPLEMENTATION.md** - Detailed clarification engine design and implementation notes
- **CLARIFICATION_GRAPH.md** - LangGraph workflow documentation
- **Code Review Summary** - Complete mcp-pr review findings and resolutions
- **Specification Documents** - Complete technical specification and architecture whitepaper

### Known Limitations

- **Phase 6 Not Implemented** - Incremental regeneration and caching for spec updates not yet implemented
- **Performance Optimizations Pending** - LLM caching and advanced parallelization planned for future releases
- **Limited Example Specifications** - Only minimal examples provided; comprehensive examples planned
- **Security Scanner Integration** - Automated security scanning planned for future release
- **80% Test Coverage Verification** - Coverage measurement framework in place; formal verification pending

### What's Next

**Phase 6: Specification Update and Regeneration** (Future Release)
- Change detection between FCS versions
- Incremental regeneration of only affected packages
- Caching strategy for unchanged portions
- Idempotent multi-run verification

**Performance & Optimization** (Future Release)
- LLM response caching for development workflows
- Parallel package generation with better worker pool management
- Build artifact caching
- Incremental compilation support

**Extended Documentation** (Future Release)
- Complete architecture guide with design decisions
- Development workflow documentation
- Extension points for custom generators
- Domain-specific generator examples (healthcare, fintech, etc.)

**Example Specifications** (Future Release)
- Simple 3-tier web application
- Microservice system with multiple services
- Batch processing system
- Real-time data pipeline

**Enhanced Integration** (Future Release)
- GitHub Actions workflow examples
- GitLab CI/CD integration
- Pre-commit hooks for specification validation
- IDE plugins for spec authoring

**Advanced Features** (Future Release)
- Multiple generation targets (gRPC, GraphQL, REST)
- Alternative validation tools beyond golangci-lint
- Custom LLM provider support
- Generation telemetry and analytics

---

### Repository

- **GitHub**: https://github.com/dshills/gocreator
- **License**: MIT
- **Initial Commit**: Implementation of core specification-driven code generation system

### Getting Started

```bash
# Build the binary
make build

# Run tests
make test

# Execute full pipeline on a specification
./bin/gocreator full specs/example-spec.yaml

# View version and build information
./bin/gocreator version
./bin/gocreator version --json
```

For detailed usage instructions, see the [project documentation](https://github.com/dshills/gocreator/blob/main/CLAUDE.md).

---

**Note**: This is the initial release of GoCreator focused on core specification processing and code generation. Subsequent releases will add incremental regeneration, advanced caching, and extended documentation.
