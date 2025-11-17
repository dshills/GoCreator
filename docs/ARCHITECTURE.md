# GoCreator Architecture

## Overview

GoCreator is a hybrid system combining **LangGraph-Go** (AI-powered reasoning layer) with **GoFlow** (deterministic workflow execution layer) to transform specifications into complete Go codebases.

The architecture enforces strict separation between:
- **Reasoning Operations**: LLM-driven analysis and decision-making (LangGraph-Go)
- **Action Operations**: Deterministic file operations and toolchain execution (GoFlow)

This separation ensures:
- **Deterministic Output**: Given the same inputs, produce identical outputs every time
- **Safety**: No arbitrary command execution, bounded file operations
- **Auditability**: Complete provenance tracking for all decisions and actions
- **Reliability**: Clear boundaries between cognitive work and mechanical work

## System Layers

```
┌─────────────────────────────────────────────────────────────────┐
│                       CLI Layer (cmd/gocreator)                 │
│  clarify | generate | validate | full | dump-fcs | version      │
└────────────────┬────────────────────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────────────────────┐
│              Workflow Orchestration Layer                        │
│  (internal/config, workflow execution, logging)                 │
└────────────────┬────────────────────────────────────────────────┘
                 │
        ┌────────┴────────┐
        │                 │
┌───────▼────────┐  ┌────▼──────────┐
│  LangGraph-Go  │  │    GoFlow      │
│  (Reasoning)   │  │  (Execution)   │
└───────────────┬┘  └────┬──────────┘
                │         │
        ┌───────┴─────────┘
        │
┌───────▼────────────────────────────────┐
│          Domain Models & Utilities       │
│  (models, config, logging, fsops, llm) │
└────────────────────────────────────────┘
```

## Detailed Component Architecture

### 1. CLI Layer

**Location**: `cmd/gocreator/`

**Responsibility**: User interaction, command routing, argument parsing

**Components**:
- `main.go` - Entry point, cobra root command setup
- `clarify.go` - Clarification command
- `generate.go` - Generation command
- `validate.go` - Validation command
- `full.go` - Full pipeline command
- `dump_fcs.go` - FCS output command
- `version.go` - Version information
- `exit_codes.go` - Standard exit code handling
- `interactive.go` - Interactive question prompting
- `batch.go` - Batch mode answer loading
- `progress.go` - Progress reporting

**Design**:
- All CLI commands delegate to internal packages
- No business logic in CLI layer
- Error handling with standard exit codes
- Structured logging integration

### 2. Workflow Orchestration Layer

**Location**: `internal/workflow/`, `internal/config/`, `pkg/logging/`

**Responsibility**: Execute workflows, manage configuration, log operations

**Key Classes**:

#### Config Management
- `internal/config/loader.go` - Load configuration from files/environment
- `internal/config/defaults.go` - Default settings
- Viper integration for flexible config sources

#### Logging Infrastructure
- `pkg/logging/logger.go` - Structured logging with zerolog
- `pkg/logging/execution_log.go` - JSONL execution audit log
- Captures all operations, decisions, and file modifications

#### Workflow Engine (GoFlow)
- `internal/workflow/engine.go` - Parse and execute YAML workflow definitions
- `internal/workflow/tasks.go` - Task type definitions (file_op, shell_cmd, langgraph)
- `internal/workflow/patcher.go` - Apply unified diffs to files
- `internal/workflow/parallel.go` - Parallel task execution with worker pools
- `internal/workflow/security.go` - Command whitelist enforcement

**Workflow YAML Format**:

```yaml
name: clarify-workflow
phases:
  - name: specification-analysis
    tasks:
      - id: parse-spec
        type: langgraph
        node: analyze
        input: spec
      - id: identify-ambiguities
        type: langgraph
        node: ambiguity_analyzer
        input: spec

  - name: question-generation
    depends_on: [specification-analysis]
    tasks:
      - id: generate-questions
        type: langgraph
        node: questions
        input: ambiguities
```

### 3. LangGraph-Go Layer (Reasoning)

**Location**: `pkg/langgraph/`, `internal/clarify/`, `internal/generate/`

**Responsibility**: AI-powered reasoning, planning, and artifact generation

**Key Concepts**:

#### Typed State Management
- `pkg/langgraph/state.go` - Strongly-typed state (no dynamic maps)
- Ensures determinism through explicit type definitions
- JSON serialization for checkpointing

#### Node Interface
- `pkg/langgraph/node.go` - Node definition for graph execution
- Nodes receive typed state, return modifications
- Deterministic with temperature=0.0 LLM config

#### Graph Execution
- `pkg/langgraph/graph.go` - DAG-based graph execution engine
- Topological sort ensures deterministic ordering
- Checkpointing for recovery and replay

#### Checkpointing
- `pkg/langgraph/checkpoint.go` - JSON-based checkpoint serialization
- Enables resumption without re-running completed nodes
- Supports distributed execution recovery

### Clarification Workflow (LangGraph-Go)

**Location**: `internal/clarify/`

**Workflow**:

```
Input Specification
    ↓
[Analyze Ambiguities]
    ↓
Identified Issues
    ↓
[Generate Questions]
    ↓
Clarification Questions
    ↓
[User Provides Answers] (interactive or batch)
    ↓
[Construct FCS]
    ↓
Final Clarified Specification
```

**Components**:
- `analyzer.go` - Identify ambiguities, missing constraints, conflicts
- `questions.go` - Generate targeted clarification questions
- `graph.go` - LangGraph-Go state machine for workflow
- `engine.go` - Orchestration and checkpoint management

### Code Generation Workflow (LangGraph-Go + GoFlow)

**Location**: `internal/generate/`

**Workflow**:

```
Final Clarified Specification
    ↓
[Architectural Planning] (LangGraph-Go)
    ↓
Generation Plan
    ↓
[Code Synthesis] (LangGraph-Go)
    ↓
Generated Files & Patches
    ↓
[File Operations] (GoFlow)
    ↓
[Build & Format] (GoFlow)
    ↓
Generated Project
```

**Components**:
- `planner.go` - FCS → architecture plan and file structure (LangGraph-Go)
- `coder.go` - Generate Go code using templates and AST (LangGraph-Go)
- `tester.go` - Generate unit, integration, contract tests (LangGraph-Go)
- `graph.go` - LangGraph-Go state machine for generation
- `plan_builder.go` - Build structured GenerationPlan with phases/tasks

### 4. GoFlow Layer (Execution)

**Location**: `internal/workflow/`, `pkg/fsops/`

**Responsibility**: Deterministic file operations, build execution, validation

**Key Components**:

#### Safe File Operations
- `pkg/fsops/safe_fs.go` - Bounded file operations (read, write, delete within configured root)
- Path traversal prevention (no `..` escape sequences)
- File permission enforcement (0600 for secrets, 0644 for code, 0755 for dirs)

#### Patch Application
- `pkg/fsops/patch.go` - Apply unified diffs to existing files
- Supports incremental updates to files
- Atomic writes through temp file + move pattern

#### Validation Execution
- `internal/validate/build.go` - Run `go build`, capture errors
- `internal/validate/lint.go` - Run `golangci-lint`, parse output
- `internal/validate/test.go` - Run `go test`, capture results and coverage
- `internal/validate/report.go` - Aggregate results with per-file error mappings

### 5. Domain Model Layer

**Location**: `internal/models/`

**Core Models**:

```
InputSpecification
  ├── id: string
  ├── format: SpecFormat (YAML, JSON, Markdown)
  ├── content: string
  └── metadata: SpecMetadata

FinalClarifiedSpecification (immutable)
  ├── id: string
  ├── sourceID: string
  ├── requirements: Requirements
  ├── entities: Entity[]
  ├── architecture: Architecture
  └── metadata: FCSMetadata

GenerationPlan
  ├── id: string
  ├── fcsID: string
  ├── packages: Package[]
  ├── files: GeneratedFile[]
  └── phases: Phase[]

ValidationReport
  ├── projectPath: string
  ├── buildResult: BuildResult
  ├── lintResult: LintResult
  ├── testResult: TestResult
  └── summary: ValidationSummary
```

### 6. LLM Provider Layer

**Location**: `pkg/llm/`

**Responsibility**: Abstract LLM provider interactions

**Components**:
- `provider.go` - LLM client wrapper using langchaingo
- `config.go` - Temperature control (default 0.0), token tracking
- Support for Anthropic, OpenAI, Google providers

**Determinism Configuration**:
```go
// Temperature must be 0.0 for deterministic output
config := LLMConfig{
    Temperature: 0.0,  // No randomness
    Model: "claude-sonnet-4",
}
```

### 7. Specification Processing Layer

**Location**: `internal/spec/`

**Responsibility**: Parse and validate specifications, build FCS

**Components**:
- `parser.go` - Unified Parse() function
- `parser_yaml.go` - YAML specification parsing
- `parser_json.go` - JSON specification parsing
- `parser_md.go` - Markdown + YAML frontmatter parsing
- `validator.go` - Schema validation, required field checks
- `fcs_builder.go` - Merge spec + clarifications → FCS
- `fcs_hash.go` - SHA-256 checksums for integrity

## Data Flow

### Clarification Flow

```
User provides spec file
        ↓
[CLI: clarify.go]
        ↓
Parse spec (format detection)
        ↓
Load config
        ↓
Create LLM client
        ↓
[Clarification Engine]
  ├─[Analyze Ambiguities] (LangGraph)
  ├─[Generate Questions] (LangGraph)
  ├─[Prompt User] (Interactive or Batch)
  └─[Build FCS] (Merge spec + answers)
        ↓
Write FCS to .gocreator/fcs.json
        ↓
Log all operations
        ↓
Exit with success
```

### Generation Flow

```
User provides spec file
        ↓
[CLI: generate.go]
        ↓
[Clarification Phase]
  └─(produces FCS)
        ↓
Load config & FCS
        ↓
[Planning Phase] (LangGraph-Go)
  ├─Generate architecture
  └─Create file structure
        ↓
[Code Generation Phase] (LangGraph-Go + GoFlow)
  ├─Generate Go source files
  ├─Generate tests
  ├─Generate configuration
  └─Apply patches to files
        ↓
[Finalization Phase] (GoFlow)
  ├─Create build files
  ├─Format code
  └─Create documentation
        ↓
Generated project ready
        ↓
Exit with success
```

### Validation Flow

```
User provides project path
        ↓
[CLI: validate.go]
        ↓
[Build Validation] (GoFlow)
  └─Execute: go build
  └─Capture: compilation errors
        ↓
[Lint Validation] (GoFlow)
  └─Execute: golangci-lint run
  └─Parse: style issues by file
        ↓
[Test Validation] (GoFlow)
  └─Execute: go test ./...
  └─Capture: test results, coverage
        ↓
[Report Generation]
  └─Aggregate results
  └─Create per-file error mappings
        ↓
Print report to console
        ↓
Write detailed report to file
        ↓
Exit with appropriate code
```

## Determinism Guarantees

GoCreator achieves deterministic output through:

### 1. **Deterministic LLM Configuration**
```
temperature = 0.0  (no randomness)
same model + version = identical outputs
```

### 2. **Typed State Management**
```
No dynamic maps or untyped interfaces
Strong typing enables reproducible serialization
```

### 3. **Deterministic Ordering**
```
DAG topological sort for graph execution
Explicit ordering in workflow definitions
JSON serialization for deterministic marshaling
```

### 4. **Reproducible File Operations**
```
Atomic writes (temp file + move pattern)
Deterministic file permissions (0600, 0644, 0755)
Checksums verify output integrity
```

### 5. **Execution Logging**
```
All decisions logged with timestamps
Execution log enables replay and audit
Checksums for generated files
```

## Separation of Concerns

### LangGraph-Go Responsibilities (Reasoning)
- ✅ Analyze specifications
- ✅ Generate clarification questions
- ✅ Plan architecture
- ✅ Generate code and tests
- ✅ Make design decisions
- ✅ Output structured artifacts (patches, definitions)

### GoFlow Responsibilities (Execution)
- ✅ Apply file operations
- ✅ Execute shell commands (whitelisted)
- ✅ Run build/lint/test tools
- ✅ Manage file permissions
- ✅ Write to disk
- ✅ Log all operations

### Boundary Enforcement
```
LangGraph-Go can ONLY output:
  ├─ Structured JSON/YAML artifacts
  ├─ Patch definitions (unified diff format)
  └─ Execution plans (workflow YAML)

GoFlow applies these outputs:
  ├─ Parses artifacts
  ├─ Executes workflow definitions
  └─ Applies patches to filesystem
```

## Error Handling

### Error Categories

| Category | Handled By | Recoverable |
|----------|-----------|-------------|
| Spec parsing errors | spec/ | No - update spec |
| Clarification errors | clarify/ | No - update spec |
| Generation errors | generate/ | No - update spec |
| File operation errors | fsops/ | No - check permissions |
| Command execution errors | workflow/ | No - check configuration |
| Validation failures | validate/ | No - update spec |

### Error Propagation
```
Internal package → Returns error with context
Workflow layer → Logs and wraps error
CLI layer → Maps to exit code and displays message
```

## Configuration Loading

```
Priority Order (highest to lowest):
  1. Command-line flags
  2. Environment variables
  3. Config file (.gocreator.yaml)
  4. Built-in defaults
```

## Logging Architecture

### Console Logging
```
INFO   [2025-11-17T10:30:45Z] Starting clarification phase
DEBUG  [2025-11-17T10:30:46Z] Spec format detected: yaml
...
```

### Structured Execution Log (JSONL)
```json
{"ts":"2025-11-17T10:30:45Z","level":"info","event":"clarification.start","spec_id":"sp-001"}
{"ts":"2025-11-17T10:30:46Z","level":"debug","event":"spec.parsed","format":"yaml","spec_id":"sp-001"}
{"ts":"2025-11-17T10:30:47Z","level":"info","event":"clarification.questions.generated","count":3,"spec_id":"sp-001"}
...
```

## Performance Optimization Strategies

### Parallel Execution
- Worker pools for independent tasks (default: 4 workers)
- DAG-based dependency resolution
- Goroutine coordination with `errgroup`

### Caching (Optional)
- LLM response caching for deterministic outputs
- Checkpointing for resume capability
- File operation deduplication

### Batching
- Multiple LLM calls in single batch when possible
- Reduce API latency
- Lower token usage

## Security Boundaries

### File Access
- All operations bounded to configured `root_dir`
- Path traversal prevention (reject `..` sequences)
- File permission enforcement

### Command Execution
- Whitelist of allowed commands (`go`, `git`, `golangci-lint`)
- No shell injection (use `exec.Command`, not shell)
- Command arguments validated

### Information Disclosure
- Secrets not logged to public execution logs
- Sensitive data marked in logging

## Extension Points

### Adding New Specification Formats
1. Create `internal/spec/parser_newformat.go`
2. Implement `ParseNewFormat() (*InputSpecification, error)`
3. Register in `Parse()` dispatcher

### Adding New Validators
1. Create `internal/validate/newvalidator.go`
2. Implement `Validate() (*ValidationResult, error)`
3. Register in validation report aggregation

### Adding New Workflow Task Types
1. Create task type in `internal/workflow/tasks.go`
2. Implement task executor in workflow engine
3. Update workflow YAML schema

### Adding New Generation Phases
1. Create phase in `internal/generate/`
2. Implement LangGraph nodes and edges
3. Register in generation workflow

## Testing Strategy

### Unit Tests
- Test each component in isolation
- Mock dependencies (LLM, filesystem)
- Table-driven tests for comprehensive coverage

### Integration Tests
- Test complete workflows end-to-end
- Use temporary directories for file operations
- Verify all phases work together

### Contract Tests
- Test LLM provider interfaces
- Verify response formats
- Test error handling

## Future Enhancements

### Phase 6: Incremental Regeneration
- Detect changes between spec versions
- Only regenerate affected packages
- Cache unchanged portions

### Performance Optimizations
- LLM response caching
- Parallel package generation
- Concurrent file operations

### Additional Specification Formats
- Protocol Buffer definitions
- AsyncAPI specifications
- Additional domain-specific formats

### Code Generation Targets
- REST API scaffolding
- gRPC service templates
- GraphQL schema generation
