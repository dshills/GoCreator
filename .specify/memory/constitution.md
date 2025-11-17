<!--
SYNC IMPACT REPORT
==================
Version Change: [initial] → 1.0.0
Change Type: MINOR (initial constitution establishment)
Modified Principles: N/A (new constitution)
Added Sections:
  - Core Principles (I-VII)
  - Quality Standards
  - Development Workflow
  - Governance
Templates Requiring Updates:
  ✅ plan-template.md (Constitution Check section references this file)
  ✅ tasks-template.md (aligned with parallel execution and testing principles)
  ✅ spec-template.md (aligned with determinism and specification principles)
Follow-up TODOs: None
-->

# GoCreator Constitution

## Core Principles

### I. Specification as Source of Truth

All code generation MUST trace back to a human-authored specification (spec.md).
The system MUST NOT improvise outside the clarified boundaries of the Final
Clarified Specification (FCS). Specifications are technology-agnostic,
describing WHAT and WHY, never HOW.

**Rationale**: Ensures determinism, reproducibility, and mitigates hallucination
risks in autonomous code generation.

### II. Deterministic Execution (NON-NEGOTIABLE)

Given the same input specification, clarifications, model configuration, and
toolchain versions, GoCreator MUST produce identical output. All operations MUST
be deterministic, reproducible, and traceable.

**Rationale**: Transforms GoCreator from "assistant" to "compiler," enabling
version-controlled specs, CI/CD integration, and predictable system behavior.

### III. Separation of Reasoning and Action

LangGraph-Go performs cognitive work (interpretation, design, planning,
generation) and produces structured artifacts. GoFlow applies those artifacts
deterministically through file operations, builds, and tests. LangGraph MUST
NEVER write files directly; all file operations flow through GoFlow.

**Rationale**: Provides safety, reproducibility, debuggability, and clear
separation of concerns between AI reasoning and mechanical execution.

### IV. Test-First Discipline

All generated code MUST include comprehensive tests. Test generation is
mandatory and MUST cover:
- Unit tests for all public APIs
- Integration tests for system boundaries
- Contract tests for external dependencies
- Validation tests that verify FCS compliance

**Rationale**: Ensures generated code quality, validates specification
adherence, and provides confidence in autonomous generation output.

### V. Concurrent Agent Execution

When multiple independent tasks can be executed in parallel (different files, no
dependencies), agents and workflows MUST execute concurrently to maximize
performance and minimize total execution time.

**Rationale**: Performance is a first-class requirement. Target: medium project
generation < 90 seconds. Concurrent execution reduces latency and improves
developer experience.

### VI. Autonomous Operation After Clarification

Once the FCS is established, GoCreator MUST execute without interruption until
completion. No mid-execution dialogue is permitted. Validation failures do NOT
trigger automated repairs; users modify specs and re-run.

**Rationale**: Maintains determinism and clear workflow. Forces specification
improvement rather than ad-hoc fixes, leading to better long-term outcomes.

### VII. Safety and Bounded Execution

All file operations MUST be:
- Bounded within a configured root directory
- Patch-based and reversible
- Logged with full provenance
- Subject to validation before application

Workflow commands MUST be predefined, static, versioned, and subject to
allowlists. No arbitrary command execution permitted.

**Rationale**: Prevents unintended modifications, supports rollback, enables
auditing, and maintains security boundaries.

## Quality Standards

### Linting and Code Review (REQUIRED)

Before ANY commit:
1. All linting MUST pass (golangci-lint for Go code)
2. Code review MUST be performed using `mcp-pr` with OpenAI provider
3. All review findings MUST be addressed or explicitly justified

**Enforcement**:
```bash
# Pre-commit validation sequence
golangci-lint run ./...
# Review using mcp-pr (via slash command or direct call)
# Address all findings before git commit
```

### Build and Test Validation

All generated code MUST pass:
- `go build` - compilation without errors
- `go vet` - static analysis
- `go test ./...` - all tests passing
- Optional: security scanners (gosec, etc.)

Validation produces machine-readable diagnostics with per-file error mappings.

### Performance Standards

- Medium project generation: < 90 seconds
- Minimal redundant LLM calls through batching
- Multi-core parallelization for builds, tests, linting
- Caching of unchanged spec fragments

## Development Workflow

### Specification-Driven Process

All features follow this mandatory sequence:

1. **Specify** (`/speckit.specify`) - Create technology-agnostic specification
2. **Clarify** (`/speckit.clarify`) - Resolve ambiguities (if needed)
3. **Plan** (`/speckit.plan`) - Generate technical implementation plan
4. **Tasks** (`/speckit.tasks`) - Create dependency-ordered task list
5. **Analyze** (`/speckit.analyze`) - Validate cross-artifact consistency
6. **Implement** (`/speckit.implement`) - Execute implementation plan

Each stage produces versioned artifacts in `specs/<number>-<short-name>/`.

### Constitution Compliance Gates

All plans MUST include a "Constitution Check" section (see plan-template.md)
validating adherence to:
- Specification-driven approach
- Deterministic execution requirements
- Test-first discipline
- Safety boundaries
- Performance targets

### Commit Protocol

1. Ensure all tests pass
2. Run linting and fix all issues
3. Execute code review via mcp-pr (OpenAI provider)
4. Address all review findings
5. Commit with descriptive message
6. Reference related spec/plan/task artifacts in commit body

## Governance

### Amendment Process

Constitution amendments require:
1. Documented rationale and impact analysis
2. Sync Impact Report covering all affected templates
3. Version increment following semantic versioning:
   - **MAJOR**: Backward-incompatible principle changes
   - **MINOR**: New principles or materially expanded guidance
   - **PATCH**: Clarifications, wording fixes, non-semantic refinements

### Compliance and Review

- All PRs MUST verify constitution compliance
- Plans and implementations MUST document alignment with principles
- Deviations MUST be explicitly justified and approved
- Complexity MUST be justified against simplicity principles

### Authoritative Documents

1. This Constitution (highest authority)
2. System Specification (`specs/gocreator_specification.md`)
3. Architecture Whitepaper (`specs/architecture_whitepaper.md`)
4. CLAUDE.md (operational guidance for AI agents)
5. Template files (`.specify/templates/*.md`)

**Version**: 1.0.0 | **Ratified**: 2025-11-17 | **Last Amended**: 2025-11-17
