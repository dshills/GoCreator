# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**GoCreator** is an autonomous Go software generation system that reads structured project specifications, resolves ambiguities through clarification, and produces complete, functioning Go codebases. The system is designed to be idempotent and deterministic—given the same specification and configuration, it produces the same output.

### Core Architecture

GoCreator consists of two cooperating subsystems:

1. **GoFlow (Workflow Execution Layer)**: Manages deterministic execution of build/test/lint actions, file operations, patching, and external tool calls
2. **LangGraph-Go (Reasoning & Generation Layer)**: Performs planning, decision-making, design analysis, and code synthesis within a controlled stepwise graph

Key architectural principles:
- **Specification as Source of Truth**: All generated code must trace back to a human-authored specification
- **Separation of Reasoning and Action**: LangGraph-Go handles cognitive work (planning, design), GoFlow handles mechanical work (file operations, builds)
- **Deterministic Execution**: Same spec + model config + toolchain versions = identical output

## Development Workflow

This project uses the **Specify** system for specification-driven development. The workflow follows a structured process managed through slash commands:

### Primary Workflow Commands

1. **`/speckit.specify <feature-description>`** - Create a new feature specification
   - Generates a branch numbered sequentially (e.g., `001-feature-name`)
   - Creates `specs/<number>-<short-name>/spec.md` with structured requirements
   - Validates specification quality with a requirements checklist
   - Use this as the FIRST step for any new feature

2. **`/speckit.clarify`** - Identify underspecified areas in the specification
   - Asks up to 5 targeted clarification questions
   - Encodes answers back into the spec
   - Run this AFTER `/speckit.specify` if requirements are unclear

3. **`/speckit.plan`** - Generate technical implementation plan
   - Creates `specs/<number>-<short-name>/plan.md`
   - Translates requirements into technical design
   - Run this AFTER spec is complete and clarified

4. **`/speckit.tasks`** - Generate actionable task list
   - Creates `specs/<number>-<short-name>/tasks.md`
   - Breaks plan into dependency-ordered tasks
   - Run this AFTER `/speckit.plan`

5. **`/speckit.implement`** - Execute the implementation plan
   - Processes and executes all tasks in `tasks.md`
   - Run this AFTER `/speckit.tasks`

6. **`/speckit.analyze`** - Cross-artifact consistency analysis
   - Validates consistency between `spec.md`, `plan.md`, and `tasks.md`
   - Non-destructive quality check
   - Run this AFTER task generation to ensure alignment

7. **`/speckit.taskstoissues`** - Convert tasks to GitHub issues
   - Creates dependency-ordered GitHub issues from `tasks.md`
   - Run this when ready to track work externally

8. **`/speckit.constitution`** - Create or update project constitution
   - Defines core development principles and constraints
   - Ensures consistency across all design artifacts

### Recommended Sequence

For a new feature:
```
/speckit.specify <description>
  → /speckit.clarify (if needed)
    → /speckit.plan
      → /speckit.tasks
        → /speckit.analyze
          → /speckit.implement
```

## Specification Philosophy

**Specifications are technology-agnostic**. They describe:
- WHAT users need (requirements, scenarios)
- WHY features exist (business value, success criteria)
- NOT HOW to implement (no tech stack, APIs, code structure)

**Plans are technology-specific**. They describe:
- Technical architecture and design decisions
- Technology choices and rationale
- Implementation approach and patterns

Keep this separation strict. Specifications should be readable by business stakeholders, not just developers.

## Project Structure

```
.claude/commands/        # Specify slash command definitions
.specify/
  memory/               # Project constitution and principles
  templates/            # Templates for specs, plans, tasks, checklists
  scripts/bash/         # Feature creation and management scripts
specs/                  # Feature specifications organized by number
  <number>-<short-name>/
    spec.md            # Feature specification (WHAT & WHY)
    plan.md            # Technical plan (HOW)
    tasks.md           # Actionable task list
    checklists/        # Quality validation checklists
```

## Key Concepts

### Final Clarified Specification (FCS)
A validated, deterministic representation of the system's design that becomes the authoritative blueprint for all code generation. The FCS is:
- Machine-readable
- Complete (no ambiguities)
- Immutable during generation phase

### Execution Stages
1. **Load Input Specification** - Parse user-authored spec
2. **Clarification Stage** (LangGraph) - Identify and resolve ambiguities
3. **FCS Construction** - Build authoritative specification
4. **Generation Stage** (LangGraph + GoFlow) - Create code artifacts
5. **File Application** (GoFlow) - Write files to disk
6. **Validation Stage** (GoFlow) - Build, lint, test
7. **Result Packaging** - Output complete project

### Design Principles from Specifications

When implementing GoCreator features:

- **Determinism is mandatory**: Given the same inputs, produce identical outputs
- **No mid-execution dialogue**: Once generation starts, it runs autonomously until completion
- **Validation failures don't trigger repairs**: Users modify specs and re-run
- **LangGraph never writes files directly**: It outputs structured patches/definitions that GoFlow applies
- **All decisions must be traceable**: Execution logs show all LangGraph reasoning
- **Safety boundaries**: File operations bounded to configured root, no self-modifying workflows

## Validation & Quality

GoCreator enforces strict validation as defined in the project constitution
(`.specify/memory/constitution.md`).

**Build validation**:
- `go build` - Compilation check
- `go vet` - Static analysis
- `golangci-lint` - Linting (REQUIRED before commits)
- `go test ./...` - Test execution

**Code review (REQUIRED before commits)**:
- Must run `mcp-pr` code review using OpenAI provider
- All review findings must be addressed or explicitly justified
- Use `/review-staged` or `/review-unstaged` slash commands

**Security**:
- Restricted file access (bounded to project root)
- No arbitrary command execution
- Patch-based, logged, reversible operations
- LangGraph has no direct filesystem authority

**Performance targets**:
- Medium project generation < 90 seconds
- Minimal redundant LLM calls through batching
- Multi-core parallelization for builds/tests/linting
- Concurrent agent execution when tasks are independent

## Working with Specifications

### Creating Quality Specs

Specifications must include:

1. **User Scenarios & Testing** (mandatory)
   - Prioritized user journeys (P1, P2, P3...)
   - Each story independently testable
   - Acceptance criteria in Given/When/Then format

2. **Functional Requirements** (mandatory)
   - Testable, unambiguous requirements
   - Format: `FR-001: System MUST [capability]`
   - Mark unclear items: `[NEEDS CLARIFICATION: question]` (max 3)

3. **Success Criteria** (mandatory)
   - Measurable, technology-agnostic outcomes
   - Both quantitative and qualitative metrics
   - Verifiable without implementation knowledge

4. **Key Entities** (if data is involved)
   - Domain concepts and relationships
   - No implementation details (tables, schemas)

### Validation Checklist Items

Every spec is validated against:
- No implementation details (languages, frameworks, APIs)
- Focused on user value and business needs
- All mandatory sections completed
- No [NEEDS CLARIFICATION] markers remain
- Requirements are testable and unambiguous
- Success criteria are measurable and technology-agnostic

## CLI Specification (Future)

The GoCreator CLI will support:

```bash
gocreator clarify <spec-file>     # Run clarification only
gocreator generate <spec-file>    # Clarification + generation
gocreator validate <project-root> # Validation only
gocreator full <spec-file>        # End-to-end pipeline
gocreator dump-fcs <spec-file>    # Output FCS representation
```

## Project Constitution

The project constitution (`.specify/memory/constitution.md`) defines the core principles
and governance for GoCreator development. Key principles include:

1. **Specification as Source of Truth** - All code traces back to human-authored specs
2. **Deterministic Execution** - Same inputs produce identical outputs (NON-NEGOTIABLE)
3. **Separation of Reasoning and Action** - LangGraph reasons, GoFlow executes
4. **Test-First Discipline** - Comprehensive tests required for all generated code
5. **Concurrent Agent Execution** - Use parallelism for independent tasks
6. **Autonomous Operation After Clarification** - No mid-execution dialogue
7. **Safety and Bounded Execution** - All operations logged, reversible, bounded

**Before every commit**:
1. All tests must pass
2. Linting must pass (`golangci-lint`)
3. Code review via `mcp-pr` (OpenAI) must be run
4. All findings must be addressed or justified

All plans must include a "Constitution Check" section validating adherence.

## Important Context

- **Current State**: This is a greenfield project. No Go code exists yet—only specifications and workflow definitions
- **Spec Documents**: See `specs/gocreator_specification.md` for system specification and `specs/architecture_whitepaper.md` for architectural philosophy
- **Constitution**: See `.specify/memory/constitution.md` for core principles and governance
- **Template System**: All spec/plan/task artifacts follow templates in `.specify/templates/`
- **Branch Naming**: Feature branches follow pattern `<number>-<short-name>` where number is sequential

## Extensibility

The system is designed to support:
- Additional LangGraph agent types
- Custom GoFlow workflow templates
- Domain-specific generators (healthcare, fintech, etc.)
- Alternate validation tools
- Multiple generation targets (gRPC, GraphQL, REST)

All extensions must operate through the FCS, not through freeform LLM prompting.

## Active Technologies
- Go 1.21+ (requires generics support) (001-core-implementation)
- File-based (specs, FCS, execution logs, generated code); Optional: SQLite for execution history (001-core-implementation)

## Recent Changes
- 001-core-implementation: Added Go 1.21+ (requires generics support)
