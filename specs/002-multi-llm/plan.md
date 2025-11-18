# Implementation Plan: Multi-LLM Provider Support

**Branch**: `002-multi-llm` | **Date**: 2025-11-17 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-multi-llm/spec.md`

## Summary

This feature adds support for configuring and using multiple LLM providers (OpenAI, Anthropic, Google) with role-based assignment in GoCreator. Users can assign different providers to specialized roles (coder, reviewer, planner, clarifier) to optimize for cost, performance, and quality. The system will route tasks to appropriate providers during workflow execution, handle failures with fallback mechanisms, and track performance metrics for each provider-role combination.

**Technical Approach**: Extend LangGraph-Go's existing provider abstraction to support multi-provider configuration, add a provider registry with role-based routing, implement synchronous credential validation at startup, and integrate metrics collection throughout the workflow execution pipeline.

## Technical Context

**Language/Version**: Go 1.21+ (requires generics for provider type parameters and flexible configuration handling)
**Primary Dependencies**:
- LangGraph-Go (existing - workflow orchestration and LLM integration)
- Existing OpenAI, Anthropic, Google SDK libraries (used by LangGraph-Go)
- YAML/JSON parsing libraries (configuration loading)
**Storage**: File-based configuration (YAML/JSON), optional SQLite for metrics persistence (consistent with existing execution history storage)
**Testing**: Go testing framework (`go test`), table-driven tests for provider routing logic, integration tests with mock providers
**Target Platform**: Linux/macOS/Windows (Go cross-platform support)
**Project Type**: Single project (library/framework enhancement to GoCreator core)
**Performance Goals**:
- Provider selection overhead < 10ms (FR/SC-002)
- Credential validation < 2 seconds per provider at startup
- Metrics query response < 2 seconds (SC-004)
- No degradation in concurrent execution with multiple providers (SC-005)
**Constraints**:
- Synchronous credential validation at startup (fail-fast requirement from FR-012)
- Global retry configuration (3 attempts, exponential backoff - from FR-013)
- Backward compatibility with existing single-provider configurations
**Scale/Scope**:
- Support minimum 3 concurrent providers (SC-001)
- 4 role types initially (coder, reviewer, planner, clarifier)
- Extensible to additional roles and providers without core changes

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I: Specification as Source of Truth ✓

- All functionality traces to spec.md requirements (FR-001 through FR-013)
- No improvisation beyond specified role-based routing and metrics tracking
- Specification is technology-agnostic (describes provider configuration, not implementation)

### Principle II: Deterministic Execution ✓

- Provider selection is deterministic based on configuration (role → provider mapping)
- Same configuration + same workflow = same provider assignments
- Retry logic is deterministic (global configuration with fixed backoff strategy)
- Metrics are append-only, deterministic based on execution events

### Principle III: Separation of Reasoning and Action ✓

- LangGraph-Go agents select providers based on task role (reasoning)
- GoFlow manages configuration loading, credential validation, metrics persistence (action)
- Provider routing layer sits between LangGraph and GoFlow (pure function: role + config → provider)

### Principle IV: Test-First Discipline ✓

- Unit tests: Provider registry, role-based routing, fallback logic, metrics aggregation
- Integration tests: Multi-provider workflow execution, credential validation, error handling
- Contract tests: Provider interface compliance for OpenAI/Anthropic/Google adapters
- Validation tests: Configuration schema validation, metrics accuracy

### Principle V: Concurrent Agent Execution ✓

- Multiple agents can use different providers concurrently without conflicts (FR-008)
- Provider client instances are thread-safe or pooled per provider
- Metrics collection uses concurrent-safe data structures (atomic operations or mutexes)

### Principle VI: Autonomous Operation After Clarification ✓

- All configuration decisions resolved during clarification (FR-011, FR-012, FR-013)
- No mid-execution prompts for provider selection
- Failure modes are defined: fallback to default or fail with actionable error (FR-007)

### Principle VII: Safety and Bounded Execution ✓

- Configuration file access is read-only after startup
- Provider credentials never logged or exposed in errors
- Metrics storage bounded to configured directory
- No arbitrary command execution; only predefined provider API calls

### Quality Standards ✓

- Linting: golangci-lint with standard Go conventions
- Testing: All tests passing before commit
- Code review: mcp-pr review required
- Performance: Meets all performance targets (< 10ms routing, < 2s metrics queries)

**Gate Result**: PASS - No violations. All principles satisfied.

## Project Structure

### Documentation (this feature)

```text
specs/002-multi-llm/
├── plan.md              # This file (/speckit.plan command output)
├── spec.md              # Feature specification (completed)
├── checklists/
│   └── requirements.md  # Specification validation checklist (completed)
├── research.md          # Phase 0 output (to be generated)
├── data-model.md        # Phase 1 output (to be generated)
├── quickstart.md        # Phase 1 output (to be generated)
├── contracts/           # Phase 1 output (to be generated)
│   └── provider-registry.yaml  # Provider interface contracts
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
src/
├── providers/           # NEW: Multi-provider support
│   ├── registry.go      # Provider registry with role-based routing
│   ├── config.go        # Configuration loading and validation
│   ├── metrics.go       # Performance metrics collection and aggregation
│   ├── validator.go     # Credential validation at startup
│   ├── adapters/        # Provider-specific adapters
│   │   ├── openai.go
│   │   ├── anthropic.go
│   │   └── google.go
│   └── retry.go         # Retry logic with backoff
├── langgraph/           # MODIFY: Integrate provider registry
│   ├── agent.go         # Add provider selection based on task role
│   └── workflow.go      # Pass provider to LLM calls
├── goflow/              # MODIFY: Configuration and metrics persistence
│   ├── config.go        # Extend to load multi-provider config
│   └── storage.go       # Metrics storage integration
└── cli/                 # MODIFY: Add metrics query commands
    └── metrics_cmd.go   # New command for viewing provider metrics

tests/
├── unit/
│   └── providers/       # NEW: Provider registry tests
│       ├── registry_test.go
│       ├── config_test.go
│       ├── metrics_test.go
│       └── validator_test.go
├── integration/
│   └── providers/       # NEW: Multi-provider workflow tests
│       ├── routing_test.go
│       ├── fallback_test.go
│       └── concurrent_test.go
└── contract/
    └── providers/       # NEW: Provider interface compliance tests
        └── adapters_test.go
```

**Structure Decision**: Single project structure (Option 1) - This is a core library enhancement to GoCreator, not a separate web/mobile application. New `providers/` package added to `src/`, with tests in corresponding `tests/` subdirectories. This maintains existing GoCreator architecture while adding multi-provider capabilities.

## Complexity Tracking

*No violations to justify - all Constitution Check items passed.*

## Phase 0: Research & Technology Decisions

**Status**: To be generated in research.md

Research tasks to resolve:
1. **LangGraph-Go provider abstraction**: Current interface and extension points for multi-provider support
2. **Configuration schema design**: YAML/JSON structure for hybrid parameter scope (global + per-role overrides)
3. **Credential validation patterns**: Best practices for synchronous validation without blocking indefinitely
4. **Metrics storage strategy**: File-based vs SQLite trade-offs for provider performance metrics
5. **Retry implementation**: Exponential backoff libraries or custom implementation in Go
6. **Thread-safety patterns**: Concurrent-safe provider client management (pooling vs instance-per-call)

**Output**: research.md with decisions, rationale, and alternatives for each research area.

## Phase 1: Design & Contracts

**Prerequisites**: research.md complete

### Design Artifacts

1. **data-model.md**: Entity definitions for:
   - ProviderConfig (identifier, type, credentials, parameters)
   - RoleAssignment (role name, provider IDs, priority order)
   - TaskExecutionContext (task ID, role, selected provider, timestamps)
   - ProviderMetrics (provider ID, role, response times, token usage, success/failure counts)

2. **contracts/provider-registry.yaml**: OpenAPI-style contract for:
   - Provider interface (Initialize, Execute, Shutdown methods)
   - Registry interface (SelectProvider, RecordMetrics, GetMetrics methods)
   - Configuration schema (validation rules, required/optional fields)

3. **quickstart.md**: Developer guide for:
   - Adding a new provider adapter
   - Configuring multi-provider setup
   - Querying metrics
   - Troubleshooting credential validation failures

### Agent Context Update

Run `.specify/scripts/bash/update-agent-context.sh claude` to add:
- Go 1.21+ generics usage
- Multi-provider architecture pattern
- Metrics collection approach

**Output**: data-model.md, contracts/, quickstart.md, updated CLAUDE.md

## Phase 2: Task Generation

**Note**: This phase is executed by `/speckit.tasks` command, NOT `/speckit.plan`.

The tasks.md file will be generated based on this plan and will include:
- Dependency-ordered implementation tasks
- Test creation tasks for each component
- Integration and validation tasks
- Documentation update tasks

---

**Plan Generation Complete**: 2025-11-17

**Next Steps**:
1. Execute Phase 0 research (within this command execution)
2. Execute Phase 1 design (within this command execution)
3. User runs `/speckit.tasks` to generate tasks.md
4. User runs `/speckit.implement` to execute implementation
