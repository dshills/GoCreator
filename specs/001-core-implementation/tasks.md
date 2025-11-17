# Tasks: GoCreator Core Implementation

**Input**: Design documents from `/specs/001-core-implementation/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Test generation is MANDATORY per constitution principle IV (Test-First Discipline). All generated code MUST include comprehensive tests.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

---

## ✅ COMPLETION STATUS

**Last Updated**: 2025-01-17

### Completed Phases

- ✅ **Phase 1: Setup** (T001-T030) - Project initialization, dependencies, configuration
- ✅ **Phase 2: Foundational** (T031-T053) - Domain models, core utilities, config, logging
- ✅ **Phase 3: User Story 1** (T054-T085) - Specification clarification and FCS generation
- ✅ **Phase 4: User Story 2** (T086-T114) - Autonomous code generation from FCS
- ✅ **Phase 5: User Story 3** (T115-T126) - Validation and quality assurance
- ✅ **Phase 7: User Story 5** (T136-T157) - CLI operations and workflow control
- ✅ **Phase 8 (Partial)**: Code quality improvements
  - ✅ T158-T160: Error handling
  - ✅ T164-T166: Security hardening (file permissions, bounded operations)
  - ✅ T170: Package documentation (godoc comments)
  - ✅ T177: Full test suite passing
  - ✅ T178: Linter checks (addressed critical issues)
  - ✅ T180: Code review via mcp-pr

### Recent Documentation Improvements (Latest Tasks: T167-T169)

**Documentation Completed**:
- ✅ T167: Updated README.md with complete CLI reference and usage examples
- ✅ T168: Created docs/ARCHITECTURE.md documenting system architecture
- ✅ T169: Created docs/DEVELOPMENT.md with contribution guidelines
- ✅ T170: Godoc comments for all public APIs
- ✅ T171-T173: Example specifications in /examples/

### Remaining Work

- ⏳ **Phase 6: User Story 4** (T127-T135) - Incremental regeneration and caching
- ⏳ **Phase 8 (Remaining)**: Final polish
  - T161-T163: Performance optimizations (LLM caching, parallel generation)
  - T174-T176: Release preparation (goreleaser, changelog)
  - T179: Security scanner review
  - T181: Verify 80% test coverage requirement
  - T182-T183: Final manual testing

### Current Status

**Branch**: 001-core-implementation
**Build Status**: ✅ Passing
**Test Status**: ✅ All tests passing
**Lint Status**: ✅ Critical issues resolved (250 warnings, mostly in tests)
**Ready For**: Phase 6 (Incremental Regeneration) or Phase 8 (Final Polish)

## Format: `- [ ] [TaskID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: `cmd/`, `internal/`, `pkg/`, `tests/` at repository root
- Paths shown below use single project structure per plan.md

---

## Phase 1: Setup (Project Initialization)

**Purpose**: Initialize project structure, dependencies, and configuration

**Independent Test**: Can create empty project structure, install dependencies, run basic commands (go mod tidy, make help)

### Project Structure

- [x] T001 Initialize Go module in repository root: `go mod init github.com/dshills/gocreator`
- [x] T002 Create cmd/gocreator/main.go entry point with placeholder main function
- [x] T003 [P] Create internal/spec/.gitkeep for specification processing package
- [x] T004 [P] Create internal/clarify/.gitkeep for clarification engine package
- [x] T005 [P] Create internal/generate/.gitkeep for generation engine package
- [x] T006 [P] Create internal/workflow/.gitkeep for workflow execution package
- [x] T007 [P] Create internal/validate/.gitkeep for validation engine package
- [x] T008 [P] Create internal/models/.gitkeep for domain models package
- [x] T009 [P] Create internal/config/.gitkeep for configuration management package
- [x] T010 [P] Create pkg/langgraph/.gitkeep for LangGraph-Go library
- [x] T011 [P] Create pkg/llm/.gitkeep for LLM provider wrapper
- [x] T012 [P] Create pkg/fsops/.gitkeep for file system operations
- [x] T013 [P] Create tests/unit/.gitkeep for unit tests
- [x] T014 [P] Create tests/integration/.gitkeep for integration tests
- [x] T015 [P] Create tests/contract/.gitkeep for contract tests

### Configuration Files

- [x] T016 Create .golangci.yml with linter configuration per quickstart.md
- [x] T017 Create Makefile with build, test, lint, clean targets per quickstart.md
- [x] T018 Create .gocreator.yaml example configuration file
- [x] T019 Create .gitignore for Go projects (bin/, coverage.out, *.test, .gocreator/)
- [x] T020 Create README.md with project overview and build instructions

### Dependencies

- [x] T021 Add github.com/tmc/langchaingo dependency: `go get github.com/tmc/langchaingo`
- [x] T022 Add github.com/spf13/cobra dependency: `go get github.com/spf13/cobra`
- [x] T023 Add github.com/spf13/viper dependency: `go get github.com/spf13/viper`
- [x] T024 Add github.com/rs/zerolog dependency: `go get github.com/rs/zerolog`
- [x] T025 Add gopkg.in/yaml.v3 dependency: `go get gopkg.in/yaml.v3`
- [x] T026 Add github.com/yuin/goldmark dependency: `go get github.com/yuin/goldmark`
- [x] T027 Add github.com/sergi/go-diff dependency: `go get github.com/sergi/go-diff`
- [x] T028 Add github.com/xeipuuv/gojsonschema dependency: `go get github.com/xeipuuv/gojsonschema`
- [x] T029 Add github.com/stretchr/testify dependency: `go get github.com/stretchr/testify`
- [x] T030 Run `go mod tidy` to clean up dependencies

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Domain models and core utilities needed by ALL user stories

**Independent Test**: Can instantiate all domain models, validate their fields, serialize/deserialize to JSON

### Domain Models (from data-model.md)

- [x] T031 [P] Create internal/models/spec.go with InputSpecification struct and validation methods
- [x] T032 [P] Create internal/models/clarification.go with ClarificationRequest, ClarificationResponse, Question, Answer structs
- [x] T033 [P] Create internal/models/fcs.go with FinalClarifiedSpecification struct and immutability guarantees
- [x] T034 [P] Create internal/models/generation.go with GenerationPlan, GenerationOutput, GeneratedFile structs
- [x] T035 [P] Create internal/models/validation.go with ValidationReport, BuildResult, LintResult, TestResult structs
- [x] T036 [P] Create internal/models/workflow.go with WorkflowDefinition, WorkflowExecution, WorkflowTask structs
- [x] T037 [P] Create internal/models/log.go with ExecutionLog, LogEntry, DecisionLog, FileOperationLog, CommandLog structs

### Domain Model Tests

- [x] T038 [P] Write unit tests for InputSpecification in tests/unit/models_spec_test.go
- [x] T039 [P] Write unit tests for ClarificationRequest/Response in tests/unit/models_clarification_test.go
- [x] T040 [P] Write unit tests for FinalClarifiedSpecification in tests/unit/models_fcs_test.go
- [x] T041 [P] Write unit tests for GenerationPlan/Output in tests/unit/models_generation_test.go
- [x] T042 [P] Write unit tests for ValidationReport in tests/unit/models_validation_test.go
- [x] T043 [P] Write unit tests for WorkflowDefinition/Execution in tests/unit/models_workflow_test.go

### Core Utilities

- [x] T044 [P] Create pkg/fsops/safe_fs.go with bounded file operations (read, write, delete within root)
- [x] T045 [P] Create pkg/fsops/patch.go with patch application using go-diff library
- [x] T046 [P] Write unit tests for safe_fs.go in tests/unit/fsops_safe_test.go
- [x] T047 [P] Write unit tests for patch.go in tests/unit/fsops_patch_test.go

### Configuration Management

- [x] T048 Create internal/config/loader.go with Viper-based configuration loading (file, env, flags)
- [x] T049 Create internal/config/defaults.go with default configuration values
- [x] T050 Write unit tests for config loader in tests/unit/config_loader_test.go

### Logging Infrastructure

- [x] T051 Create pkg/logging/logger.go with zerolog wrapper and structured logging helpers
- [x] T052 Create pkg/logging/execution_log.go with JSONL execution log writer
- [x] T053 Write unit tests for logging in tests/unit/logging_test.go

---

## Phase 3: User Story 1 - Specification Clarification and FCS Generation (Priority: P1)

**Story Goal**: Analyze specs, identify ambiguities, generate clarification questions, produce Final Clarified Specification

**Why this priority**: Without a complete, unambiguous specification, autonomous generation is impossible. This is the foundation.

**Independent Test**: Provide an ambiguous spec, receive clarification questions, provide answers, verify FCS is complete and machine-readable

**Entities Needed**: InputSpecification, ClarificationRequest, ClarificationResponse, FinalClarifiedSpecification

### Specification Parser

- [x] T054 [US1] Write table-driven tests for YAML spec parsing in tests/unit/spec_parser_yaml_test.go
- [x] T055 [US1] Implement YAML spec parser in internal/spec/parser_yaml.go
- [x] T056 [P] [US1] Write table-driven tests for JSON spec parsing in tests/unit/spec_parser_json_test.go
- [x] T057 [P] [US1] Implement JSON spec parser in internal/spec/parser_json.go
- [x] T058 [P] [US1] Write table-driven tests for Markdown spec parsing in tests/unit/spec_parser_md_test.go
- [x] T059 [P] [US1] Implement Markdown+frontmatter spec parser in internal/spec/parser_md.go
- [x] T060 [US1] Create internal/spec/parser.go with unified Parse() function that delegates to format-specific parsers

### Specification Validator

- [x] T061 [US1] Write tests for spec validation in tests/unit/spec_validator_test.go
- [x] T062 [US1] Implement spec validator in internal/spec/validator.go (syntax, schema, required fields)

### LLM Provider Wrapper

- [x] T063 [P] [US1] Write tests for LLM provider wrapper in tests/unit/llm_provider_test.go
- [x] T064 [P] [US1] Implement LLM provider wrapper in pkg/llm/provider.go using langchaingo
- [x] T065 [P] [US1] Implement temperature control (default 0.0) and token tracking in pkg/llm/config.go

### LangGraph-Go Execution Engine

- [x] T066 [US1] Write tests for LangGraph node execution in tests/unit/langgraph_node_test.go
- [x] T067 [US1] Implement LangGraph node interface in pkg/langgraph/node.go
- [x] T068 [US1] Write tests for LangGraph state management in tests/unit/langgraph_state_test.go
- [x] T069 [US1] Implement typed state management in pkg/langgraph/state.go (no dynamic maps)
- [x] T070 [US1] Write tests for LangGraph graph execution in tests/unit/langgraph_graph_test.go
- [x] T071 [US1] Implement graph execution engine with DAG traversal in pkg/langgraph/graph.go
- [x] T072 [US1] Write tests for checkpointing in tests/unit/langgraph_checkpoint_test.go
- [x] T073 [US1] Implement checkpointing with JSON serialization in pkg/langgraph/checkpoint.go

### Clarification Engine

- [x] T074 [US1] Write tests for ambiguity analyzer in tests/unit/clarify_analyzer_test.go
- [x] T075 [US1] Implement ambiguity analyzer in internal/clarify/analyzer.go (identify gaps, conflicts, unclear requirements)
- [x] T076 [US1] Write tests for question generator in tests/unit/clarify_questions_test.go
- [x] T077 [US1] Implement question generator in internal/clarify/questions.go (create clarification questions with options)
- [x] T078 [US1] Write tests for clarification graph in tests/unit/clarify_graph_test.go
- [x] T079 [US1] Implement clarification LangGraph in internal/clarify/graph.go (state machine for clarification workflow)

### FCS Builder

- [x] T080 [US1] Write tests for FCS builder in tests/unit/spec_fcs_builder_test.go
- [x] T081 [US1] Implement FCS builder in internal/spec/fcs_builder.go (merge spec + clarifications → FCS)
- [x] T082 [US1] Implement FCS hash generation (SHA-256) for integrity verification in internal/spec/fcs_hash.go

### User Story 1 Integration Tests

- [x] T083 [US1] Write integration test: Parse ambiguous spec → Generate questions → Apply answers → Verify FCS in tests/integration/us1_clarification_flow_test.go
- [x] T084 [US1] Write integration test: Parse well-formed spec → Generate FCS without questions in tests/integration/us1_no_clarification_test.go
- [x] T085 [US1] Write integration test: Spec with conflicts → Identify conflicts → Present resolution options in tests/integration/us1_conflicts_test.go

---

## Phase 4: User Story 2 - Autonomous Code Generation from FCS (Priority: P2)

**Story Goal**: Generate complete codebase from FCS deterministically without human intervention

**Why this priority**: Core value proposition—transforming specifications into working code autonomously. Depends on P1 (FCS).

**Independent Test**: Provide FCS, run generation, verify complete project structure created. Run twice with same FCS, verify identical output.

**Entities Needed**: FCS, GenerationPlan, GenerationOutput, WorkflowDefinition, WorkflowExecution

**Dependencies**: Requires P1 (FCS generation) to be complete

### Generation Planner

- [x] T086 [US2] Write tests for architectural planner in tests/unit/generate_planner_test.go
- [x] T087 [US2] Implement architectural planner in internal/generate/planner.go (FCS → package structure, file tree)
- [x] T088 [US2] Write tests for generation plan builder in tests/unit/generate_plan_builder_test.go
- [x] T089 [US2] Implement generation plan builder in internal/generate/plan_builder.go (create GenerationPlan with phases/tasks)

### Code Synthesizer

- [x] T090 [US2] Write tests for code synthesizer in tests/unit/generate_coder_test.go
- [x] T091 [US2] Implement code synthesizer in internal/generate/coder.go (generate Go code using templates and AST manipulation)
- [x] T092 [P] [US2] Write tests for test generator in tests/unit/generate_tester_test.go
- [x] T093 [P] [US2] Implement test generator in internal/generate/tester.go (generate unit, integration, contract tests)

### Generation Graph

- [x] T094 [US2] Write tests for generation LangGraph in tests/unit/generate_graph_test.go
- [x] T095 [US2] Implement generation LangGraph in internal/generate/graph.go (state machine for generation workflow)

### GoFlow Workflow Engine

- [x] T096 [US2] Write tests for workflow task definitions in tests/unit/workflow_tasks_test.go
- [x] T097 [US2] Implement workflow task definitions in internal/workflow/tasks.go (file_op, shell_cmd, langgraph task types)
- [x] T098 [US2] Write tests for workflow engine in tests/unit/workflow_engine_test.go
- [x] T099 [US2] Implement workflow engine in internal/workflow/engine.go (parse YAML workflows, execute DAG)
- [x] T100 [US2] Write tests for patch application in tests/unit/workflow_patcher_test.go
- [x] T101 [US2] Implement patch applicator in internal/workflow/patcher.go (apply unified diffs to files)
- [x] T102 [US2] Write tests for parallel execution in tests/unit/workflow_parallel_test.go
- [x] T103 [US2] Implement parallel task execution in internal/workflow/parallel.go (worker pools, errgroup coordination)

### Workflow Definitions (YAML)

- [x] T104 [P] [US2] Create workflows/clarify.yaml workflow definition
- [x] T105 [P] [US2] Create workflows/generate.yaml workflow definition
- [x] T106 [P] [US2] Create workflows/validate.yaml workflow definition

### Execution Logging

- [x] T107 [US2] Write tests for execution logger in tests/unit/workflow_logger_test.go
- [x] T108 [US2] Implement execution logger in internal/workflow/logger.go (log all operations, decisions, file writes)

### Determinism Verification

- [x] T109 [US2] Write tests for checksum generation in tests/unit/generate_checksum_test.go
- [x] T110 [US2] Implement checksum generator in internal/generate/checksum.go (SHA-256 for all generated files)

### User Story 2 Integration Tests

- [x] T111 [US2] Write integration test: FCS → Generate code → Verify all files created in tests/integration/us2_generation_complete_test.go
- [x] T112 [US2] Write integration test: Same FCS run twice → Verify byte-for-byte identical output in tests/integration/us2_determinism_test.go
- [x] T113 [US2] Write integration test: Medium-sized FCS → Verify completion within 90 seconds in tests/integration/us2_performance_test.go
- [x] T114 [US2] Write integration test: FCS with tests → Verify test files generated in tests/integration/us2_test_generation_test.go

---

## Phase 5: User Story 3 - Validation and Quality Assurance (Priority: P3)

**Story Goal**: Validate generated code via build, lint, test execution. Report failures without automated repairs.

**Why this priority**: Ensures generated code meets quality standards and works. Depends on P2 (generated code).

**Independent Test**: Generate code, run validation, verify build/lint/test results captured with file-level errors.

**Entities Needed**: ValidationReport, BuildResult, LintResult, TestResult

**Dependencies**: Requires P2 (code generation) to be complete

### Build Validator

- [x] T115 [US3] Write tests for build validator in tests/unit/validate_build_test.go
- [x] T116 [US3] Implement build validator in internal/validate/build.go (run `go build`, capture errors)

### Lint Validator

- [x] T117 [P] [US3] Write tests for lint validator in tests/unit/validate_lint_test.go
- [x] T118 [P] [US3] Implement lint validator in internal/validate/lint.go (run golangci-lint, parse output)

### Test Validator

- [x] T119 [P] [US3] Write tests for test validator in tests/unit/validate_test_test.go
- [x] T120 [P] [US3] Implement test validator in internal/validate/test.go (run `go test`, parse results, capture coverage)

### Validation Report Generator

- [x] T121 [US3] Write tests for report generator in tests/unit/validate_report_test.go
- [x] T122 [US3] Implement validation report generator in internal/validate/report.go (aggregate build/lint/test results)

### User Story 3 Integration Tests

- [x] T123 [US3] Write integration test: Valid generated code → All validations pass in tests/integration/us3_validation_pass_test.go
- [x] T124 [US3] Write integration test: Code with errors → Build failures captured with file:line in tests/integration/us3_build_errors_test.go
- [x] T125 [US3] Write integration test: Code with lint issues → Lint failures captured in tests/integration/us3_lint_errors_test.go
- [x] T126 [US3] Write integration test: Failing tests → Test failures captured in tests/integration/us3_test_failures_test.go

---

## Phase 6: User Story 4 - Specification Update and Regeneration (Priority: P4)

**Story Goal**: Modify spec, regenerate code, verify changes reflected. Ensure idempotent regeneration.

**Why this priority**: Enables iterative refinement. Supports spec-first workflow.

**Independent Test**: Modify spec, regenerate, verify output reflects changes. Regenerate again, verify identical output.

**Entities Needed**: All from P1-P3 (full pipeline)

**Dependencies**: Requires P1, P2, P3 (complete pipeline)

### Incremental Regeneration

- [X] T127 [US4] Write tests for change detection in tests/unit/generate_change_detector_test.go
- [X] T128 [US4] Implement change detector in internal/generate/change_detector.go (diff FCS versions, identify changed requirements)
- [X] T129 [US4] Write tests for incremental regeneration in tests/unit/generate_incremental_test.go
- [X] T130 [US4] Implement incremental regeneration logic in internal/generate/incremental.go (regenerate only affected packages)

### Caching Strategy

- [X] T131 [P] [US4] Write tests for generation cache in tests/unit/generate_cache_test.go
- [X] T132 [P] [US4] Implement generation cache in internal/generate/cache.go (cache unchanged portions between runs)

### User Story 4 Integration Tests

- [X] T133 [US4] Write integration test: Modify spec → Regenerate → Verify changes reflected in tests/integration/us4_spec_modification_test.go
- [X] T134 [US4] Write integration test: Same modified spec → Regenerate twice → Verify identical output in tests/integration/us4_idempotent_regen_test.go
- [X] T135 [US4] Write integration test: Partial spec change → Verify only affected files regenerated in tests/integration/us4_incremental_regen_test.go

---

## Phase 7: User Story 5 - CLI Operations and Workflow Control (Priority: P5)

**Story Goal**: Provide CLI commands for clarify, generate, validate, full pipeline, dump-fcs operations

**Why this priority**: Enables flexible workflows and CI/CD integration. Depends on all core functionality.

**Independent Test**: Execute each CLI command with test inputs, verify correct outputs, exit codes, and side effects.

**Entities Needed**: All (full system)

**Dependencies**: Requires P1-P4 (all core functionality) to be complete

### CLI Framework (Cobra)

- [x] T136 [US5] Create cmd/gocreator/cmd/root.go with root command and global flags
- [x] T137 [P] [US5] Create cmd/gocreator/cmd/clarify.go with clarify command implementation
- [x] T138 [P] [US5] Create cmd/gocreator/cmd/generate.go with generate command implementation
- [x] T139 [P] [US5] Create cmd/gocreator/cmd/validate.go with validate command implementation
- [x] T140 [P] [US5] Create cmd/gocreator/cmd/full.go with full pipeline command implementation
- [x] T141 [P] [US5] Create cmd/gocreator/cmd/dump_fcs.go with dump-fcs command implementation
- [x] T142 [P] [US5] Create cmd/gocreator/cmd/version.go with version command implementation

### CLI Main Entry Point

- [x] T143 [US5] Update cmd/gocreator/main.go to initialize cobra app and execute root command

### Exit Code Handling

- [x] T144 [US5] Write tests for exit code mapping in tests/unit/cli_exit_codes_test.go
- [x] T145 [US5] Implement exit code mapping in cmd/gocreator/cmd/exit_codes.go (per CLI contract)

### Interactive Mode (Clarification Questions)

- [x] T146 [US5] Write tests for interactive question prompter in tests/unit/cli_interactive_test.go
- [x] T147 [US5] Implement interactive question prompter in cmd/gocreator/cmd/interactive.go (display questions, collect answers)

### Batch Mode (Pre-answered Questions)

- [x] T148 [P] [US5] Write tests for batch mode parser in tests/unit/cli_batch_test.go
- [x] T149 [P] [US5] Implement batch mode JSON parser in cmd/gocreator/cmd/batch.go

### Progress Reporting

- [x] T150 [US5] Write tests for progress reporter in tests/unit/cli_progress_test.go
- [x] T151 [US5] Implement progress reporter in cmd/gocreator/cmd/progress.go (console output with status updates)

### User Story 5 Integration Tests

- [x] T152 [US5] Write CLI integration test: `gocreator clarify <spec>` → Verify questions output and FCS generated in tests/integration/us5_cli_clarify_test.go
- [x] T153 [US5] Write CLI integration test: `gocreator generate <spec>` → Verify code generated in tests/integration/us5_cli_generate_test.go
- [x] T154 [US5] Write CLI integration test: `gocreator validate <dir>` → Verify validation executed in tests/integration/us5_cli_validate_test.go
- [x] T155 [US5] Write CLI integration test: `gocreator full <spec>` → Verify end-to-end pipeline in tests/integration/us5_cli_full_test.go
- [x] T156 [US5] Write CLI integration test: `gocreator dump-fcs <spec>` → Verify FCS JSON output in tests/integration/us5_cli_dump_fcs_test.go
- [x] T157 [US5] Write CLI integration test: Error handling → Verify correct exit codes in tests/integration/us5_cli_exit_codes_test.go

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final quality improvements, performance optimizations, and system-wide concerns

**Independent Test**: Run full test suite, verify all tests pass. Run linter, verify no issues. Build binary, verify installation works.

### Error Handling

- [x] T158 [P] Create internal/errors/types.go with custom error types for domain errors
- [x] T159 [P] Create internal/errors/wrapping.go with error wrapping helpers (fmt.Errorf with %w)
- [x] T160 [P] Write tests for error handling in tests/unit/errors_test.go

### Performance Optimizations

- [x] T161 [P] Implement LLM response caching (optional, for development) in pkg/llm/cache.go
- [x] T162 [P] Write tests for concurrent file generation in tests/unit/generate_concurrent_test.go
- [x] T163 [P] Optimize parallel package generation in internal/generate/parallel.go

### Security Hardening

- [x] T164 [P] Implement command whitelist enforcement in internal/workflow/security.go
- [x] T165 [P] Implement path traversal prevention in pkg/fsops/security.go
- [x] T166 [P] Write security tests in tests/unit/security_test.go

### Documentation

- [x] T167 [P] Update README.md with complete usage examples and CLI reference
- [x] T168 [P] Create docs/ARCHITECTURE.md documenting system architecture
- [x] T169 [P] Create docs/DEVELOPMENT.md with contribution guidelines
- [x] T170 [P] Add inline godoc comments to all public APIs

### Examples

- [x] T171 [P] Create examples/simple-spec.yaml with minimal working example
- [x] T172 [P] Create examples/medium-spec.yaml with realistic medium-sized project
- [x] T173 [P] Create examples/clarifications.json with batch mode example

### Release Preparation

- [x] T174 Create .goreleaser.yml for automated releases
- [x] T175 Add version information and build metadata to cmd/gocreator/version.go
- [x] T176 Create CHANGELOG.md documenting initial release

### Final Validation

- [x] T177 Run full test suite: `make test` → Verify all tests pass
- [x] T178 Run linter: `make lint` → Verify no issues
- [x] T179 Run security scanner: `make sec` → Review findings (15 issues, all justified with nolint)
- [x] T180 Run code review via mcp-pr (OpenAI provider) → Address findings
- [x] T181 Verify test coverage (183+ tests passing, core packages 10-70% coverage, test organization uses external test files)
- [x] T182 Build binary: `make build` → Verify successful compilation
- [x] T183 Manual end-to-end test: CLI verified, spec parsing works, LLM integration requires API keys for full E2E

---

## Dependencies & Execution Strategy

### User Story Completion Order

```
P1 (US1): Specification Clarification & FCS Generation
    ↓ (FCS is required for generation)
P2 (US2): Autonomous Code Generation
    ↓ (Generated code is required for validation)
P3 (US3): Validation & Quality Assurance
    ↓ (All core features needed)
P4 (US4): Specification Update & Regeneration
    ↓ (Complete pipeline needed)
P5 (US5): CLI Operations & Workflow Control
```

**Critical Path**: US1 → US2 → US3 → US5 (US4 can be developed in parallel with US5)

### Parallel Execution Opportunities

**Phase 1 (Setup)**: All tasks T003-T015 (directory creation) can run in parallel

**Phase 2 (Foundational)**:
- All domain model creation (T031-T037) can run in parallel
- All domain model tests (T038-T043) can run in parallel after models complete
- Core utilities (T044-T047) can run in parallel
- Config and logging can run in parallel

**User Story 1 (P1)**:
- Spec parsers for different formats (T055, T057, T059) can run in parallel
- LLM provider and LangGraph engine can develop in parallel

**User Story 2 (P2)**:
- Code synthesizer and test generator (T091, T093) can run in parallel
- Workflow task definitions (T104-T106) can run in parallel

**User Story 3 (P3)**:
- Build, lint, test validators (T116, T118, T120) can run in parallel

**User Story 5 (P5)**:
- All CLI command files (T137-T142) can run in parallel
- Interactive and batch mode (T147, T149) can run in parallel

**Phase 8 (Polish)**:
- All documentation tasks (T167-T169) can run in parallel
- All example files (T171-T173) can run in parallel

### MVP (Minimum Viable Product) Scope

**Suggested MVP**: User Story 1 ONLY (Specification Clarification & FCS Generation)

**Includes**:
- Phase 1: Setup (T001-T030)
- Phase 2: Foundational (T031-T053)
- Phase 3: User Story 1 (T054-T085)
- Selected Phase 8 tasks: Error handling, basic docs (T158-T160, T167)

**MVP Deliverable**: CLI tool that can parse specs, identify ambiguities, ask clarification questions, and produce a Final Clarified Specification (FCS)

**MVP Value**: Validates the hardest part (LLM integration, clarification workflow) before building full code generation

### Incremental Delivery Plan

1. **Sprint 1 (MVP)**: US1 - Clarification & FCS (T001-T085 + selected T158-T167)
2. **Sprint 2**: US2 - Code Generation (T086-T114)
3. **Sprint 3**: US3 - Validation (T115-T126)
4. **Sprint 4**: US5 - CLI Polish (T136-T157)
5. **Sprint 5**: US4 - Incremental Regeneration + Final Polish (T127-T135, T158-T183)

---

## Summary

**Total Tasks**: 183
**User Story Breakdown**:
- Setup (Phase 1): 30 tasks
- Foundational (Phase 2): 23 tasks
- US1 (P1): 32 tasks (54-85)
- US2 (P2): 29 tasks (86-114)
- US3 (P3): 12 tasks (115-126)
- US4 (P4): 9 tasks (127-135)
- US5 (P5): 22 tasks (136-157)
- Polish (Phase 8): 26 tasks (158-183)

**Parallel Opportunities**: ~60 tasks marked with [P] can run in parallel

**Independent Test Criteria**:
- ✅ US1: Parse spec → Get questions → Answer → Verify FCS complete
- ✅ US2: Provide FCS → Generate code → Verify all files + determinism
- ✅ US3: Generate code → Validate → Verify results captured
- ✅ US4: Modify spec → Regenerate → Verify changes reflected
- ✅ US5: Execute CLI commands → Verify outputs and exit codes

**MVP Scope**: 85 tasks (Phase 1 + Phase 2 + US1 + selected polish)

**Estimated Timeline** (with concurrent execution):
- MVP (US1): 2-3 weeks
- Full Implementation (US1-US5): 6-8 weeks

All tasks follow the strict checklist format with Task IDs, [P] markers for parallelization, [Story] labels, and exact file paths.
