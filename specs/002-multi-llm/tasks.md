# Tasks: Multi-LLM Provider Support

**Input**: Design documents from `/specs/002-multi-llm/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/provider-registry.yaml

**Tests**: Per Constitution Principle IV (Test-First Discipline), comprehensive tests are included for all components.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Overall Status (as of commit 00d8162)

### Summary
- **Phase 1 (Setup)**: ✅ COMPLETE (4/4 tasks)
- **Phase 2 (Foundational)**: ✅ COMPLETE (12/12 tasks)
- **Phase 3 (User Story 1 - MVP)**: ✅ COMPLETE (19/19 tasks)
- **Phase 4 (User Story 2)**: ⚠️ PARTIAL (11/15 tasks) - 3 tasks blocked on LangGraph
- **Phase 5 (User Story 3)**: ⚠️ PARTIAL (4/20 tasks) - Tests complete, implementation blocked on metrics.go, CLI, and GoFlow
- **Phase 6 (Polish)**: ⚠️ IN PROGRESS (2/15 tasks)

### Completion Status
- **Total Tasks**: 85
- **Completed**: 48 (56%)
- **Blocked**: 9 (11%) - Waiting on LangGraph-Go, CLI, and GoFlow implementations
- **Remaining**: 28 (33%)

### What's Working
✅ **User Story 1 (MVP) is fully functional**:
- Multi-provider configuration with YAML support
- Provider registry with role-based selection
- OpenAI, Anthropic, and Google adapters implemented
- Credential validation and fallback chains
- Comprehensive test coverage (unit, integration, contract)

### What's Blocked
⏸️ **LangGraph Integration** (T042-T044): Requires LangGraph-Go implementation
⏸️ **Metrics Implementation** (T055-T064): Core metrics.go file needs implementation
⏸️ **CLI Integration** (T067-T070): Requires CLI framework implementation
⏸️ **GoFlow Integration** (T065-T066): Requires GoFlow implementation

### Next Steps
1. **Immediate**: Complete Phase 6 polish tasks (T074-T085) for User Story 1
2. **Short-term**: Implement metrics.go for User Story 3 (T055-T064)
3. **Blocked**: Wait for LangGraph-Go, CLI, and GoFlow before completing remaining integrations

---

## Phase 1: Setup (Shared Infrastructure) ✅ COMPLETE

**Purpose**: Project initialization and basic structure for multi-provider support

- [x] T001 Create directory structure for providers package: src/providers/ and src/providers/adapters/
- [x] T002 Create directory structure for provider tests: tests/unit/providers/, tests/integration/providers/, tests/contract/providers/
- [x] T003 [P] Create Go module dependencies file if needed, add gopkg.in/yaml.v3 for YAML parsing
- [x] T004 [P] Create type definitions file src/providers/types.go with enums (ProviderType, Role, TaskStatus, MetricStatus, ErrorCode)

---

## Phase 2: Foundational (Blocking Prerequisites) ✅ COMPLETE

**Purpose**: Core multi-provider infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T005 Implement RetryConfig struct and Execute method in src/providers/retry.go with exponential backoff (research.md section 5)
- [x] T006 Implement ProviderError type with error classification in src/providers/errors.go (contracts section "Error Contracts")
- [x] T007 Implement ConfigError type for configuration validation errors in src/providers/errors.go
- [x] T008 [P] Create Request and Response structs in src/providers/types.go (contracts section "Request/Response Structures")
- [x] T009 [P] Create ProviderConfig struct with validation methods in src/providers/config.go (data-model.md section 1)
- [x] T010 [P] Create RoleAssignment struct with validation methods in src/providers/config.go (data-model.md section 2)
- [x] T011 [P] Create MultiProviderConfig struct in src/providers/config.go (data-model.md "Configuration Object Model")
- [x] T012 Implement YAML configuration loader with environment variable expansion in src/providers/config.go LoadConfig method
- [x] T013 Implement configuration validation in src/providers/config.go ValidateConfig method (validates provider references, no circular fallbacks, parameter types)
- [x] T014 Implement backward compatibility for single-provider config in src/providers/config.go (contracts section "Backward Compatibility")
- [x] T015 [P] Create LLMProvider interface definition in src/providers/interface.go (contracts "LLMProvider Interface")
- [x] T016 [P] Create ProviderRegistry interface definition in src/providers/interface.go (contracts "ProviderRegistry Interface")

**Checkpoint**: ✅ Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Configure Specialized LLM Roles (Priority: P1) 🎯 MVP ✅ COMPLETE

**Goal**: Users can assign different LLM providers to specialized roles and verify that each role uses its assigned provider during execution

**Independent Test**: Configure multiple providers with different roles in YAML file, load configuration, validate all providers, verify role assignments are correct, and test provider selection for each role

### Tests for User Story 1 ✅

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T017 [P] [US1] Create unit test for configuration loading and validation in tests/unit/providers/config_test.go (test YAML parsing, env var expansion, validation rules)
- [x] T018 [P] [US1] Create unit test for provider registry initialization in tests/unit/providers/registry_test.go (test provider registration, role mapping, default provider)
- [x] T019 [P] [US1] Create unit test for provider selection logic in tests/unit/providers/registry_test.go (test primary selection, fallback chain, default provider)
- [x] T020 [P] [US1] Create unit test for credential validation in tests/unit/providers/validator_test.go (test parallel validation, timeout handling, error aggregation)
- [x] T021 [US1] Create contract test for provider interface compliance in tests/contract/providers/adapters_test.go (test all adapters implement LLMProvider correctly)
- [x] T022 [US1] Create integration test for multi-provider configuration and initialization in tests/integration/providers/routing_test.go (end-to-end config → registry → provider selection)

### Implementation for User Story 1 ✅

- [x] T023 [P] [US1] Implement ProviderRegistry struct with provider storage and role mappings in src/providers/registry.go
- [x] T024 [P] [US1] Implement NewRegistry constructor with configuration initialization in src/providers/registry.go (loads config, creates providers, validates credentials)
- [x] T025 [US1] Implement SelectProvider method with role-based routing in src/providers/registry.go (implements selection algorithm from contracts)
- [x] T026 [US1] Implement provider fallback logic in src/providers/registry.go SelectProvider method (try primary → fallbacks → default)
- [x] T027 [US1] Implement parameter override resolution in src/providers/registry.go (merge global + role-specific parameters per research.md section 2)
- [x] T028 [P] [US1] Create Validator struct with parallel validation in src/providers/validator.go (implements parallel credential validation from research.md section 3)
- [x] T029 [P] [US1] Implement ValidateAll method with timeout and error aggregation in src/providers/validator.go
- [x] T030 [US1] Implement OpenAI adapter in src/providers/adapters/openai.go (implements LLMProvider interface with Initialize, Execute, Name, Type, Shutdown)
- [x] T031 [US1] Implement Anthropic adapter in src/providers/adapters/anthropic.go (implements LLMProvider interface)
- [x] T032 [US1] Implement Google adapter in src/providers/adapters/google.go (implements LLMProvider interface)
- [x] T033 [US1] Implement adapter factory in src/providers/registry.go createProvider method (creates adapter based on ProviderType)
- [x] T034 [US1] Add error handling and logging for configuration errors with actionable messages in src/providers/config.go
- [x] T035 [US1] Add error handling for provider validation failures with clear failure reasons in src/providers/validator.go

**Checkpoint**: ✅ User Story 1 is fully functional - users can configure multiple providers with role assignments and verify correct provider selection

---

## Phase 4: User Story 2 - Dynamic Role Selection During Workflow (Priority: P2) ⚠️ PARTIAL

**Goal**: During workflow execution, tasks are automatically routed to the appropriate LLM based on task role, with execution logs showing which provider handled each task

**Independent Test**: Execute a multi-stage workflow with different task types, verify through execution logs that each task used the correct provider for its role, verify concurrent tasks don't conflict

**Status**: Tests and core types complete. LangGraph integration blocked until LangGraph-Go is implemented.

### Tests for User Story 2 ✅

- [x] T036 [P] [US2] Create unit test for TaskExecutionContext state management in tests/unit/providers/context_test.go (test state transitions, validation rules)
- [x] T037 [P] [US2] Create integration test for concurrent provider usage in tests/integration/providers/concurrent_test.go (test multiple tasks with different providers running in parallel)
- [x] T038 [US2] Create integration test for workflow-level provider routing in tests/integration/providers/routing_test.go (test clarification → planning → generation → review workflow)
- [x] T039 [US2] Create integration test for fallback behavior during failures in tests/integration/providers/fallback_test.go (test primary failure → fallback → default provider flow)

### Implementation for User Story 2 ⚠️ PARTIAL

- [x] T040 [P] [US2] Create TaskExecutionContext struct in src/providers/types.go (data-model.md section 3)
- [x] T041 [P] [US2] Implement task status state machine with validation in src/providers/types.go (enforce valid transitions from data-model.md)
- [ ] T042 [US2] ⏸️ BLOCKED: Integrate provider registry with LangGraph-Go agents in src/langgraph/agent.go (add provider selection based on task role) - **LangGraph-Go not yet implemented**
- [ ] T043 [US2] ⏸️ BLOCKED: Modify LangGraph-Go workflow to pass selected provider to LLM calls in src/langgraph/workflow.go - **LangGraph-Go not yet implemented**
- [ ] T044 [US2] ⏸️ BLOCKED: Add role parameter to LLM request context in src/langgraph/workflow.go - **LangGraph-Go not yet implemented**
- [x] T045 [US2] Implement provider execute with retry logic in provider adapters Execute methods (use RetryConfig from src/providers/retry.go)
- [x] T046 [US2] Implement error classification for retryable vs non-retryable errors in src/providers/errors.go (per contracts "Retryable Error Codes")
- [x] T047 [US2] Add execution logging for provider selection decisions in src/providers/registry.go SelectProvider method
- [x] T048 [US2] Add execution logging for provider request/response in provider adapters Execute methods (log provider ID, role, timestamps - NOT credentials)
- [x] T049 [US2] Implement thread-safe provider instance management with RWMutex in src/providers/registry.go (research.md section 6)
- [x] T050 [US2] Implement graceful shutdown with provider cleanup in src/providers/registry.go Shutdown method

**Checkpoint**: ⚠️ Provider infrastructure ready for workflow integration. LangGraph integration tasks (T042-T044) blocked until LangGraph-Go implementation begins.

---

## Phase 5: User Story 3 - Provider Performance Monitoring (Priority: P3) ⚠️ PARTIAL

**Goal**: Users can view provider performance metrics (response time, token usage, error rates) grouped by provider and role to identify performance issues and optimize configurations

**Independent Test**: Execute workflows with multiple providers, query metrics via CLI or API, verify metrics show correct response times, token counts, and success rates for each provider-role combination

**Status**: Tests complete. Core metrics implementation in progress. CLI and GoFlow integration blocked.

### Tests for User Story 3 ✅

- [x] T051 [P] [US3] Create unit test for metrics collection in tests/unit/providers/metrics_test.go (test ProviderMetrics creation, validation)
- [x] T052 [P] [US3] Create unit test for metrics aggregation in tests/unit/providers/metrics_test.go (test MetricsSummary computation, percentile calculations)
- [x] T053 [US3] Create unit test for metrics storage (SQLite and file-based) in tests/unit/providers/metrics_test.go (test write, read, query operations)
- [x] T054 [US3] Create integration test for end-to-end metrics flow in tests/integration/providers/metrics_test.go (execute task → record metrics → query metrics)

### Implementation for User Story 3 ⚠️ PARTIAL

- [ ] T055 [P] [US3] Create ProviderMetrics struct in src/providers/metrics.go (data-model.md section 4)
- [ ] T056 [P] [US3] Create MetricsSummary struct with aggregation fields in src/providers/metrics.go (contracts "MetricsSummary")
- [ ] T057 [P] [US3] Implement SQLite schema creation with indexes in src/providers/metrics.go (data-model.md "Storage Schema")
- [ ] T058 [US3] Implement MetricsCollector struct with thread-safe recording in src/providers/metrics.go (use sync.RWMutex per research.md section 6)
- [ ] T059 [US3] Implement RecordMetrics method with async write to SQLite in src/providers/metrics.go
- [ ] T060 [US3] Implement GetMetrics method with SQL aggregation queries in src/providers/metrics.go (avg response time, success rate, token totals)
- [ ] T061 [US3] Implement file-based metrics fallback (JSON Lines format) in src/providers/metrics.go
- [ ] T062 [US3] Implement metrics migration from file to SQLite in src/providers/metrics.go
- [ ] T063 [US3] Integrate metrics collection into provider Execute methods (record on completion - success or failure)
- [ ] T064 [US3] Integrate metrics collection into ProviderRegistry in src/providers/registry.go (add RecordMetrics and GetMetrics methods)
- [ ] T065 [US3] ⏸️ BLOCKED: Extend GoFlow configuration to load multi-provider config in src/goflow/config.go - **GoFlow not yet implemented**
- [ ] T066 [US3] ⏸️ BLOCKED: Integrate metrics storage with GoFlow storage layer in src/goflow/storage.go - **GoFlow not yet implemented**
- [ ] T067 [P] [US3] ⏸️ BLOCKED: Create CLI command for metrics summary in src/cli/metrics_cmd.go (implements gocreator metrics summary with filters) - **CLI not yet implemented**
- [ ] T068 [P] [US3] ⏸️ BLOCKED: Create CLI command for metrics export in src/cli/metrics_cmd.go (implements gocreator metrics export with formats: CSV, JSON) - **CLI not yet implemented**
- [ ] T069 [US3] ⏸️ BLOCKED: Add metrics query filters (provider ID, role, time range) to CLI commands in src/cli/metrics_cmd.go - **CLI not yet implemented**
- [ ] T070 [US3] ⏸️ BLOCKED: Implement metrics output formatting (table, CSV, JSON) in src/cli/metrics_cmd.go - **CLI not yet implemented**

**Checkpoint**: ⚠️ Metrics infrastructure tests complete. Core implementation and integrations (GoFlow, CLI) pending.

---

## Phase 6: Polish & Cross-Cutting Concerns ⚠️ IN PROGRESS

**Purpose**: Improvements that affect multiple user stories and final quality checks

- [ ] T071 [P] Add comprehensive GoDoc comments to all public interfaces and methods across src/providers/ package
- [x] T072 [P] Create example configuration files in examples/multi-provider-config.yaml with OpenAI, Anthropic, Google setups
- [x] T073 [P] Update CLAUDE.md with multi-provider architecture patterns (already done by update-agent-context.sh)
- [ ] T074 Run golangci-lint on all provider code and fix any linting issues
- [ ] T075 Run all tests (unit, integration, contract) and verify 100% pass rate with go test ./tests/...
- [ ] T076 Verify quickstart.md examples work end-to-end (test provider adapter creation, configuration validation, metrics queries)
- [ ] T077 Run performance benchmarks to verify < 10ms provider selection overhead (SC-002)
- [ ] T078 Run performance benchmarks to verify < 2s metrics query response (SC-004)
- [ ] T079 Verify concurrent execution shows no degradation (SC-005)
- [ ] T080 Test backward compatibility with existing single-provider configurations
- [ ] T081 Verify credentials are never logged or exposed in error messages (security requirement)
- [ ] T082 Create migration guide for users upgrading from single-provider to multi-provider configuration
- [ ] T083 Run mcp-pr code review using OpenAI provider on all new code in src/providers/
- [ ] T084 Address all code review findings or document justifications
- [ ] T085 Final validation: Run complete workflow with 3 providers across all 4 roles and verify all success criteria (SC-001 through SC-008)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup (Phase 1) completion - BLOCKS all user stories
- **User Stories (Phase 3, 4, 5)**: All depend on Foundational (Phase 2) completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1) - Configure Roles**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2) - Dynamic Routing**: Depends on US1 for provider registry and configuration - Builds on US1 by adding workflow integration
- **User Story 3 (P3) - Metrics**: Can start after Foundational (Phase 2) - Independent of US2, but integrates with provider execution from US1

### Within Each User Story

- Tests MUST be written and FAIL before implementation (Test-First Discipline)
- Type definitions and structs before implementation logic
- Core components (registry, adapters) before integrations (LangGraph, GoFlow, CLI)
- Error handling concurrent with or immediately after core implementation
- Logging added during implementation, not as separate task

### Parallel Opportunities

**Phase 1 (Setup)**: All tasks can run in parallel
- T001 (directories), T002 (test directories), T003 (dependencies), T004 (types) - all independent

**Phase 2 (Foundational)**: Tasks T008-T016 marked [P] can run in parallel
- Type definitions (T008, T009, T010, T011, T015, T016) can all be created concurrently
- Error types, retry logic, and interfaces are independent

**Phase 3 (User Story 1)**:
- Tests T017-T020 can all run in parallel (different test files)
- Implementation: T023, T024, T028, T029 can run in parallel
- Adapters T030, T031, T032 can run in parallel (different providers)

**Phase 4 (User Story 2)**:
- Tests T036, T037 can run in parallel
- T040, T041 can run in parallel

**Phase 5 (User Story 3)**:
- Tests T051, T052 can run in parallel
- T055, T056, T057 can run in parallel
- CLI commands T067, T068 can run in parallel

**Phase 6 (Polish)**: Tasks T071, T072, T073 can run in parallel

---

## Parallel Example: User Story 1 (Configure Specialized LLM Roles)

```bash
# Launch all tests for User Story 1 together:
Task: "Create unit test for configuration loading and validation in tests/unit/providers/config_test.go"
Task: "Create unit test for provider registry initialization in tests/unit/providers/registry_test.go"
Task: "Create unit test for provider selection logic in tests/unit/providers/registry_test.go"
Task: "Create unit test for credential validation in tests/unit/providers/validator_test.go"

# After tests written, launch all adapter implementations together:
Task: "Implement OpenAI adapter in src/providers/adapters/openai.go"
Task: "Implement Anthropic adapter in src/providers/adapters/anthropic.go"
Task: "Implement Google adapter in src/providers/adapters/google.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T004)
2. Complete Phase 2: Foundational (T005-T016) - CRITICAL
3. Complete Phase 3: User Story 1 (T017-T035)
4. **STOP and VALIDATE**: Test configuration loading, provider selection, credential validation independently
5. Deploy/demo MVP: Multi-provider configuration with role-based selection

**MVP Deliverable**: Users can configure multiple LLM providers (OpenAI, Anthropic, Google) and assign them to roles (coder, reviewer, planner, clarifier). System validates credentials at startup and selects correct provider for each role.

### Incremental Delivery

1. **Foundation** (Phase 1 + 2): Project structure, types, configuration, interfaces → Foundation ready
2. **MVP** (Phase 3): User Story 1 → Test independently → Multi-provider configuration works
3. **Workflow Integration** (Phase 4): User Story 2 → Test independently → Workflows route tasks to providers
4. **Observability** (Phase 5): User Story 3 → Test independently → Performance metrics and monitoring
5. **Polish** (Phase 6): Final validation and quality checks
6. Each phase adds value without breaking previous functionality

### Parallel Team Strategy

With multiple developers:

1. **Team completes Setup + Foundational together** (Phase 1 + 2)
2. **Once Foundational is done**:
   - Developer A: User Story 1 (T017-T035) - Configuration and provider selection
   - Developer B: User Story 3 (T051-T070) - Metrics (can start in parallel with US1)
   - After US1 complete: Developer C: User Story 2 (T036-T050) - Workflow integration
3. Stories integrate and validate independently
4. Team collaborates on Phase 6 (Polish)

**Rationale**: US1 and US3 are mostly independent (both use foundational types). US2 depends on US1 for registry but can start once US1 core is done.

---

## Task Count Summary

- **Phase 1 (Setup)**: 4 tasks
- **Phase 2 (Foundational)**: 12 tasks
- **Phase 3 (User Story 1 - MVP)**: 19 tasks (6 tests + 13 implementation)
- **Phase 4 (User Story 2)**: 15 tasks (4 tests + 11 implementation)
- **Phase 5 (User Story 3)**: 20 tasks (4 tests + 16 implementation)
- **Phase 6 (Polish)**: 15 tasks
- **Total**: 85 tasks

### Parallel Opportunities Identified

- **Phase 1**: 4 parallel tasks
- **Phase 2**: 9 parallel tasks
- **Phase 3 (US1)**: 8 parallel tasks (4 tests + 4 implementations)
- **Phase 4 (US2)**: 4 parallel tasks
- **Phase 5 (US3)**: 6 parallel tasks
- **Phase 6**: 3 parallel tasks
- **Total Parallelizable**: 34 tasks (~40% of total)

### Independent Test Criteria by Story

**User Story 1 (Configure Specialized LLM Roles)**:
- Load multi-provider YAML configuration with 3 providers (OpenAI, Anthropic, Google)
- Verify all provider credentials validate successfully at startup
- Verify role-based provider selection returns correct provider for each of 4 roles
- Verify fallback chain works (primary fails → fallback 1 → fallback 2 → default)
- Verify parameter overrides merge correctly (global + role-specific)

**User Story 2 (Dynamic Role Selection During Workflow)**:
- Execute multi-stage workflow (clarification → planning → generation → review)
- Verify each stage's execution log shows correct provider used
- Execute 10 concurrent tasks with different roles
- Verify no provider conflicts, all tasks complete successfully
- Verify fallback to default provider when primary fails

**User Story 3 (Provider Performance Monitoring)**:
- Execute 100 tasks across 3 providers and 4 roles
- Query metrics via CLI: `gocreator metrics summary`
- Verify metrics show correct counts, average response times, success rates
- Query metrics for specific provider-role: `gocreator metrics summary --provider openai-gpt4 --role coder`
- Verify metrics query completes in < 2 seconds (SC-004)
- Export metrics to CSV and JSON formats

### Suggested MVP Scope

**Minimum Viable Product**: User Story 1 only (Phase 1 + 2 + 3)

**Delivers**:
- Multi-provider configuration (YAML)
- Provider credential validation at startup
- Role-based provider selection
- Fallback chains (primary → fallback → default)
- Support for OpenAI, Anthropic, Google providers
- 4 role types (coder, reviewer, planner, clarifier)

**Value**: Users can immediately optimize LLM costs by assigning cheaper models to non-critical roles (e.g., fast model for review, powerful model for code generation).

**Time Estimate**: ~35 tasks (Setup + Foundational + US1) = ~60% faster than full implementation

---

## Notes

- [P] tasks = different files, no dependencies - can execute in parallel
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable
- Tests written first, must fail before implementation (Test-First Discipline)
- Constitution Principle IV enforced: comprehensive test coverage at unit, integration, and contract levels
- All file paths are absolute from repository root
- Commit after each task or logical group of related tasks
- Stop at any checkpoint to validate story independently
- Performance targets validated in Phase 6 before final completion
- Security validation (credentials never logged) required in Phase 6
- mcp-pr code review required before final completion (Constitution requirement)
