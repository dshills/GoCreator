# Implementation Plan: GoCreator Core Implementation

**Branch**: `001-core-implementation` | **Date**: 2025-11-17 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-core-implementation/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement the GoCreator autonomous Go code generation system as a command-line tool that reads structured project specifications, resolves ambiguities through a controlled clarification process, and generates complete, functioning Go codebases deterministically. The system uses a hybrid architecture combining GoFlow (deterministic workflow execution) and LangGraph-Go (AI-powered reasoning and generation) to transform specifications into working code with comprehensive tests, validation, and quality assurance.

## Technical Context

**Language/Version**: Go 1.21+ (requires generics support)
**Primary Dependencies**: `langchaingo` (LLM provider abstractions), custom LangGraph-Go implementation, custom GoFlow workflow engine, `cobra` (CLI), `viper` (config), `zerolog` (logging)
**Storage**: File-based (specs, FCS, execution logs, generated code); Optional: SQLite for execution history
**Testing**: Go testing framework (`go test`), `testify` for assertions/mocks, table-driven tests, integration test harness
**Target Platform**: Linux, macOS, Windows (cross-platform CLI)
**Project Type**: Single CLI application with library components
**Performance Goals**: Medium project generation < 90 seconds; Clarification phase < 30 seconds; Support up to 100 files/50 packages
**Constraints**: Deterministic output (same inputs → identical results); No mid-execution dialogue; File operations bounded to configured root
**Scale/Scope**: CLI tool (~20-30 packages), ~50-100 source files, comprehensive test coverage (unit, integration, contract)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Specification as Source of Truth ✅
- Plan derives directly from spec.md requirements
- Implementation will trace all code generation to FCS
- No improvisation outside specification boundaries

### II. Deterministic Execution (NON-NEGOTIABLE) ✅
- Architecture designed for deterministic output (GoFlow for execution, LangGraph-Go for reasoning)
- Same FCS + model config + toolchain → identical output required
- Execution logging and provenance tracking planned

### III. Separation of Reasoning and Action ✅
- LangGraph-Go layer: reasoning, planning, artifact generation (no direct file writes)
- GoFlow layer: file operations, builds, tests (deterministic application)
- Clear architectural boundary enforced

### IV. Test-First Discipline ✅
- Comprehensive test generation required per spec (FR-009)
- Unit, integration, and contract test coverage mandatory
- Test validation as part of quality gates

### V. Concurrent Agent Execution ✅
- GoFlow designed for parallel execution of independent tasks (FR-032)
- Performance target requires concurrent workflows
- Multi-core parallelization for builds, tests, linting

### VI. Autonomous Operation After Clarification ✅
- No mid-execution dialogue after FCS established (FR-007)
- Validation failures reported, no automated repairs (FR-016)
- Specification update and regeneration workflow supported

### VII. Safety and Bounded Execution ✅
- All file operations bounded to configured root (FR-017)
- Patch-based, reversible operations with logging (FR-018, FR-019)
- Predefined, versioned workflow commands (FR-020, FR-021)

### Quality Standards Alignment ✅
- golangci-lint integration required
- mcp-pr code review before commits (constitution requirement)
- Build, vet, test validation gates (FR-011, FR-012, FR-014)

### Performance Standards Alignment ✅
- Medium project generation < 90 seconds (SC-001)
- Clarification phase < 30 seconds (SC-002)
- Multi-core parallelization planned

**Gate Result**: ✅ PASS - All constitution principles align with planned architecture

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/
└── gocreator/           # CLI entry point
    └── main.go

internal/
├── spec/                # Specification processing
│   ├── parser.go        # Parse YAML/JSON/MD specs
│   ├── validator.go     # Validate spec syntax
│   └── fcs.go          # FCS construction
├── clarify/             # Clarification engine (LangGraph-Go)
│   ├── analyzer.go      # Identify ambiguities
│   ├── questions.go     # Generate clarification questions
│   └── graph.go        # LangGraph-Go state machine
├── generate/            # Generation engine (LangGraph-Go)
│   ├── planner.go       # Architectural planning
│   ├── coder.go        # Code synthesis
│   ├── tester.go       # Test generation
│   └── graph.go        # LangGraph-Go state machine
├── workflow/            # Workflow execution (GoFlow)
│   ├── engine.go        # GoFlow execution engine
│   ├── tasks.go        # Task definitions
│   ├── patcher.go      # Patch application
│   └── parallel.go     # Concurrent task execution
├── validate/            # Validation engine
│   ├── build.go        # go build validation
│   ├── lint.go         # golangci-lint integration
│   ├── test.go         # go test execution
│   └── report.go       # Validation report generation
├── models/              # Domain models
│   ├── spec.go         # Input Specification
│   ├── fcs.go          # Final Clarified Specification
│   ├── output.go       # Generation Output Model
│   ├── validation.go   # Validation Report
│   └── workflow.go     # Workflow Definition
└── config/              # Configuration management
    ├── loader.go        # Load config from files/env
    └── defaults.go      # Default settings

pkg/                     # Public libraries (reusable)
├── langgraph/          # LangGraph-Go client wrapper
├── llm/                # LLM provider integrations
└── fsops/              # Safe file system operations

tests/
├── unit/               # Unit tests
├── integration/        # Integration tests
└── contract/           # Contract tests

go.mod
go.sum
.golangci.yml           # Linter configuration
Makefile                # Build automation
```

**Structure Decision**: Single CLI application using Go's standard layout. The `internal/` directory contains the core implementation organized by functional domains (spec, clarify, generate, workflow, validate). The `cmd/gocreator/` directory contains the CLI entry point. Public libraries in `pkg/` can be reused by external tools if needed. This structure supports the separation of reasoning (clarify, generate with LangGraph-Go) from action (workflow with GoFlow).

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

**Status**: No constitution violations - all complexity is justified by requirements

---

## Post-Design Constitution Check

*Re-evaluation after Phase 1 design complete*

### I. Specification as Source of Truth ✅
- All design artifacts (research.md, data-model.md, contracts/) derive from spec.md
- Implementation will follow planned architecture without deviation
- No improvisation outside specification boundaries

### II. Deterministic Execution (NON-NEGOTIABLE) ✅
- Technology choices support determinism:
  - LangGraph-Go with typed state (no dynamic maps)
  - JSON checkpointing for recovery
  - SHA-256 checksums for FCS integrity verification
- LLM temperature=0.0 configured by default
- All file operations logged with provenance

### III. Separation of Reasoning and Action ✅
- Architecture enforces separation:
  - `internal/clarify` and `internal/generate` contain LangGraph-Go reasoning (produce patches/artifacts)
  - `internal/workflow` contains GoFlow execution (applies patches, runs commands)
  - Clear package boundaries prevent violations

### IV. Test-First Discipline ✅
- TDD workflow documented in quickstart.md
- `testify` for assertions and mocks
- Table-driven tests pattern mandated
- Test coverage requirement: 80% minimum

### V. Concurrent Agent Execution ✅
- GoFlow designed for parallel task execution:
  - Worker pools with bounded concurrency
  - DAG-based dependency resolution
  - Goroutines with `errgroup` for coordination
- `max_parallel` configuration option (default: 4)

### VI. Autonomous Operation After Clarification ✅
- Clarification phase is only interactive point
- After FCS construction, execution runs to completion
- No mid-execution user prompts in design
- Checkpointing supports resumption without interaction

### VII. Safety and Bounded Execution ✅
- `pkg/fsops` enforces root directory boundaries
- Command whitelist in configuration (`allow_commands`)
- All operations reversible (patch-based)
- Execution log provides full audit trail

### Quality Standards Alignment ✅
- golangci-lint configured in .golangci.yml
- Makefile includes `make lint` target
- mcp-pr code review workflow documented in quickstart.md
- Validation engine implements all required checks

### Performance Standards Alignment ✅
- Concurrent execution design supports <90s target
- Checkpointing reduces retry cost
- Batch LLM calls minimize redundant API requests
- Caching strategy planned (optional, for development)

**Gate Result**: ✅ PASS - Architecture aligns with all constitution principles. Design is approved for implementation.

---

## Phase 2: Task Generation

This plan is complete through Phase 1 (design). Next steps:

1. Run `/speckit.tasks` to generate detailed implementation task list
2. Tasks will be organized by user story (P1-P5) for independent implementation
3. Each task will include exact file paths and acceptance criteria

**Artifacts Generated**:
- ✅ `plan.md` - This file (implementation plan)
- ✅ `research.md` - Technology decisions and rationale
- ✅ `data-model.md` - Domain entities and relationships
- ✅ `contracts/cli-interface.md` - CLI command contracts
- ✅ `quickstart.md` - Development setup guide
- ✅ Agent context updated (CLAUDE.md)

**Ready for**: `/speckit.tasks` to generate `tasks.md`
