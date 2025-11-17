# Data Model: GoCreator Core Implementation

**Branch**: `001-core-implementation` | **Date**: 2025-11-17
**Purpose**: Define domain entities, relationships, and state transitions

## Overview

This document defines the core domain entities for GoCreator, their attributes, relationships, validation rules, and state transitions. All entities are designed to support deterministic, traceable code generation.

---

## Core Entities

### 1. InputSpecification

**Purpose**: Represents a user-authored specification before clarification

**Attributes**:
- `ID` (string, UUID): Unique identifier
- `Format` (enum: yaml, json, markdown): Source format
- `Content` (string): Raw specification content
- `ParsedData` (map[string]interface{}): Structured parsed data
- `Metadata` (SpecMetadata): Creation time, author, version
- `ValidationErrors` ([]ValidationError): Syntax/schema errors

**Relationships**:
- Produces → `ClarificationRequest` (1:1)
- Transforms into → `FinalClarifiedSpecification` (1:1)

**Validation Rules**:
- MUST be valid YAML, JSON, or Markdown with frontmatter
- MUST contain required fields: name, description, requirements
- MUST NOT contain malicious paths or command injection attempts

**States**:
1. `Unparsed` → Initial state
2. `Parsed` → Successfully parsed, awaiting validation
3. `Valid` → Passed all validation checks
4. `Invalid` → Failed validation (terminal state)

---

### 2. ClarificationRequest

**Purpose**: Contains questions generated from ambiguous specs

**Attributes**:
- `ID` (string, UUID): Unique identifier
- `SpecID` (string): Reference to InputSpecification
- `Questions` ([]Question): List of clarification questions
- `Ambiguities` ([]Ambiguity): Identified ambiguities
- `CreatedAt` (time.Time): When questions were generated

**Question Structure**:
```go
type Question struct {
    ID          string
    Topic       string
    Context     string      // Relevant spec section
    Question    string
    Options     []Option
    UserAnswer  *string     // nil until answered
}

type Option struct {
    Label        string
    Description  string
    Implications string
}
```

**Ambiguity Structure**:
```go
type Ambiguity struct {
    Type        string  // "missing_constraint", "conflict", "unclear_requirement"
    Location    string  // Path in spec (e.g., "requirements.FR-005")
    Description string
    Severity    string  // "critical", "important", "minor"
}
```

**Relationships**:
- Belongs to → `InputSpecification` (1:1)
- Produces → `ClarificationResponse` (1:1)

**Validation Rules**:
- MUST have at least 1 question or ambiguity
- Questions MUST have 2-4 options each
- Options MUST be mutually exclusive

---

### 3. ClarificationResponse

**Purpose**: Contains user answers to clarification questions

**Attributes**:
- `ID` (string, UUID): Unique identifier
- `RequestID` (string): Reference to ClarificationRequest
- `Answers` (map[string]Answer): Question ID → Answer
- `AnsweredAt` (time.Time): When responses were provided

**Answer Structure**:
```go
type Answer struct {
    QuestionID string
    SelectedOption *string    // If user chose from options
    CustomAnswer *string      // If user provided custom answer
}
```

**Relationships**:
- Responds to → `ClarificationRequest` (1:1)
- Contributes to → `FinalClarifiedSpecification` (1:1)

**Validation Rules**:
- MUST answer all questions
- Each answer MUST have either SelectedOption OR CustomAnswer (not both)
- SelectedOption MUST reference valid option from question

---

### 4. FinalClarifiedSpecification (FCS)

**Purpose**: Complete, unambiguous specification ready for code generation

**Attributes**:
- `ID` (string, UUID): Unique identifier
- `Version` (string): FCS schema version (e.g., "1.0")
- `OriginalSpecID` (string): Reference to InputSpecification
- `Metadata` (FCSMetadata): Creation time, clarifications applied
- `Requirements` (Requirements): Functional and non-functional
- `Architecture` (Architecture): Packages, dependencies, patterns
- `DataModel` (DataModel): Entities and relationships
- `APIContracts` ([]APIContract): API definitions (if applicable)
- `TestingStrategy` (TestingStrategy): Test coverage requirements
- `BuildConfig` (BuildConfig): Build and deployment settings

**FCSMetadata Structure**:
```go
type FCSMetadata struct {
    CreatedAt      time.Time
    OriginalSpec   string
    Clarifications []AppliedClarification
    Hash           string  // SHA-256 of FCS content for determinism
}

type AppliedClarification struct {
    QuestionID string
    Answer     string
    AppliedTo  string  // Where in FCS this was applied
}
```

**Requirements Structure**:
```go
type Requirements struct {
    Functional    []FunctionalRequirement
    NonFunctional []NonFunctionalRequirement
}

type FunctionalRequirement struct {
    ID          string  // e.g., "FR-001"
    Description string
    Priority    string  // "critical", "high", "medium", "low"
    Category    string  // From spec categorization
}
```

**Architecture Structure**:
```go
type Architecture struct {
    Packages     []Package
    Dependencies []Dependency
    Patterns     []DesignPattern
}

type Package struct {
    Name         string
    Path         string
    Purpose      string
    Dependencies []string  // Other package names
}
```

**Relationships**:
- Derived from → `InputSpecification` + `ClarificationResponse` (N:1)
- Inputs to → `GenerationPlan` (1:1)

**Validation Rules**:
- MUST have zero ambiguities
- All requirements MUST be testable
- Package dependencies MUST be acyclic
- Hash MUST match content (for integrity)

**Immutability**:
- FCS is IMMUTABLE after creation
- Any changes require new FCS with incremented version

---

### 5. GenerationPlan

**Purpose**: Detailed plan for code generation derived from FCS

**Attributes**:
- `ID` (string, UUID): Unique identifier
- `FCSID` (string): Reference to FCS
- `Phases` ([]GenerationPhase): Ordered generation phases
- `FileTree` (FileTree): Target directory structure
- `CreatedAt` (time.Time): When plan was created

**GenerationPhase Structure**:
```go
type GenerationPhase struct {
    Name         string
    Order        int
    Tasks        []GenerationTask
    Dependencies []string  // Phase names that must complete first
}

type GenerationTask struct {
    ID           string
    Type         string  // "generate_file", "apply_patch", "run_command"
    TargetPath   string
    Inputs       map[string]interface{}
    CanParallel  bool
}
```

**FileTree Structure**:
```go
type FileTree struct {
    Root        string
    Directories []Directory
    Files       []File
}

type File struct {
    Path         string
    Purpose      string
    GeneratedBy  string  // Task ID
}
```

**Relationships**:
- Derived from → `FinalClarifiedSpecification` (1:1)
- Produces → `GenerationOutput` (1:1)

**Validation Rules**:
- Phase dependencies MUST be acyclic
- All target paths MUST be within root directory
- Parallel tasks MUST have no shared file writes

---

### 6. GenerationOutput

**Purpose**: Complete set of generated artifacts

**Attributes**:
- `ID` (string, UUID): Unique identifier
- `PlanID` (string): Reference to GenerationPlan
- `Files` ([]GeneratedFile): All generated files
- `Patches` ([]Patch): All patches applied
- `Metadata` (OutputMetadata): Timestamps, provenance
- `Status` (enum): pending, in_progress, completed, failed

**GeneratedFile Structure**:
```go
type GeneratedFile struct {
    Path        string
    Content     string
    Checksum    string  // SHA-256
    GeneratedAt time.Time
    Generator   string  // Which LangGraph node generated it
}
```

**Patch Structure**:
```go
type Patch struct {
    TargetFile string
    Diff       string  // Unified diff format
    AppliedAt  time.Time
    Reversible bool
}
```

**OutputMetadata Structure**:
```go
type OutputMetadata struct {
    StartedAt   time.Time
    CompletedAt *time.Time
    Duration    time.Duration
    FilesCount  int
    LinesCount  int
}
```

**Relationships**:
- Produced by → `GenerationPlan` (1:1)
- Inputs to → `ValidationReport` (1:1)

**Validation Rules**:
- All file paths MUST be unique
- All checksums MUST match content
- Status transitions MUST follow: pending → in_progress → (completed|failed)

---

### 7. ValidationReport

**Purpose**: Results from build, lint, and test validation

**Attributes**:
- `ID` (string, UUID): Unique identifier
- `OutputID` (string): Reference to GenerationOutput
- `BuildResult` (BuildResult): Compilation results
- `LintResult` (LintResult): Linting results
- `TestResult` (TestResult): Test execution results
- `OverallStatus` (enum): pass, fail
- `CreatedAt` (time.Time): When validation ran

**BuildResult Structure**:
```go
type BuildResult struct {
    Success  bool
    Errors   []CompilationError
    Warnings []CompilationWarning
    Duration time.Duration
}

type CompilationError struct {
    File    string
    Line    int
    Column  int
    Message string
}
```

**LintResult Structure**:
```go
type LintResult struct {
    Success bool
    Issues  []LintIssue
    Duration time.Duration
}

type LintIssue struct {
    File     string
    Line     int
    Severity string  // "error", "warning", "info"
    Rule     string
    Message  string
}
```

**TestResult Structure**:
```go
type TestResult struct {
    Success     bool
    TotalTests  int
    PassedTests int
    FailedTests int
    Failures    []TestFailure
    Coverage    float64  // Percentage
    Duration    time.Duration
}

type TestFailure struct {
    Package  string
    Test     string
    Message  string
    Location string
}
```

**Relationships**:
- Validates → `GenerationOutput` (1:1)

**Validation Rules**:
- BuildResult, LintResult, TestResult MUST all be present
- OverallStatus = pass only if all three succeed
- Duration MUST be > 0

---

### 8. WorkflowDefinition

**Purpose**: Static, versioned template describing task execution

**Attributes**:
- `ID` (string, UUID): Unique identifier
- `Name` (string): Workflow name (e.g., "clarify", "generate", "validate")
- `Version` (string): Workflow version
- `Tasks` ([]WorkflowTask): Task definitions
- `Config` (WorkflowConfig): Workflow configuration

**WorkflowTask Structure**:
```go
type WorkflowTask struct {
    ID           string
    Name         string
    Type         string  // "langgraph", "file_op", "shell_cmd"
    Inputs       map[string]interface{}
    Outputs      []string
    Dependencies []string  // Task IDs
    Timeout      time.Duration
}
```

**WorkflowConfig Structure**:
```go
type WorkflowConfig struct {
    MaxParallel int
    Retries     int
    Timeout     time.Duration
    AllowedCommands []string
}
```

**Relationships**:
- Executes → `WorkflowExecution` (1:N)

**Validation Rules**:
- Task dependencies MUST be acyclic (DAG)
- All task types MUST be in allowed list
- Shell commands MUST be in AllowedCommands

---

### 9. WorkflowExecution

**Purpose**: Runtime instance of workflow execution

**Attributes**:
- `ID` (string, UUID): Unique identifier
- `WorkflowID` (string): Reference to WorkflowDefinition
- `Status` (enum): pending, running, completed, failed
- `StartedAt` (time.Time): Execution start
- `CompletedAt` (*time.Time): Execution end (nil if running)
- `TaskExecutions` ([]TaskExecution): Individual task results
- `Checkpoints` ([]Checkpoint): State snapshots

**TaskExecution Structure**:
```go
type TaskExecution struct {
    TaskID      string
    Status      string
    StartedAt   time.Time
    CompletedAt *time.Time
    Result      interface{}
    Error       *string
}
```

**Checkpoint Structure**:
```go
type Checkpoint struct {
    ID          string
    TaskID      string  // Last completed task
    State       map[string]interface{}
    CreatedAt   time.Time
    Recoverable bool
}
```

**Relationships**:
- Instance of → `WorkflowDefinition` (N:1)

**State Transitions**:
```
pending → running → (completed | failed)
         ↓
    (can checkpoint at any point during running)
```

**Validation Rules**:
- Status transitions MUST follow state machine
- CompletedAt MUST be nil if status is pending or running
- Each task MUST execute after its dependencies complete

---

### 10. ExecutionLog

**Purpose**: Comprehensive record of all operations and decisions

**Attributes**:
- `ID` (string, UUID): Unique identifier
- `WorkflowExecutionID` (string): Reference to WorkflowExecution
- `Entries` ([]LogEntry): Chronological log entries

**LogEntry Structure**:
```go
type LogEntry struct {
    Timestamp  time.Time
    Level      string  // "debug", "info", "warn", "error"
    Component  string  // Which part of system logged this
    Operation  string  // What operation was performed
    Context    map[string]interface{}
    Message    string
    Error      *string
}
```

**Specialized Entry Types**:

**DecisionLog** (LangGraph-Go decisions):
```go
type DecisionLog struct {
    LogEntry
    Decision  string
    Rationale string
    Alternatives []string
}
```

**FileOperationLog** (GoFlow file ops):
```go
type FileOperationLog struct {
    LogEntry
    Operation string  // "create", "update", "delete", "patch"
    Path      string
    Checksum  string
}
```

**CommandLog** (Shell command execution):
```go
type CommandLog struct {
    LogEntry
    Command   string
    Args      []string
    ExitCode  int
    Stdout    string
    Stderr    string
    Duration  time.Duration
}
```

**Relationships**:
- Logs → `WorkflowExecution` (1:1)

**Validation Rules**:
- Entries MUST be chronologically ordered
- All file operations MUST be logged before execution
- All LangGraph decisions MUST include rationale

**Storage**:
- Written to `<output-dir>/.gocreator/execution.jsonl`
- One JSON object per line (JSONL format)
- Supports streaming writes during execution

---

## Entity Relationships Diagram

```
InputSpecification
    ↓
ClarificationRequest
    ↓ (+ user input)
ClarificationResponse
    ↓
FinalClarifiedSpecification (FCS)
    ↓
GenerationPlan
    ↓ (executed by)
WorkflowExecution
    ↓ (produces)
GenerationOutput
    ↓
ValidationReport
    ↓ (logged in)
ExecutionLog
```

---

## Persistence Strategy

### Storage Locations

```
<project-root>/.gocreator/
├── input_spec.{yaml|json|md}      # Original spec
├── fcs.json                        # Final Clarified Specification
├── generation_plan.json            # Generation plan
├── checkpoints/                    # Execution checkpoints
│   ├── checkpoint_001.json
│   └── checkpoint_002.json
├── execution.jsonl                 # Execution log (JSONL)
├── validation_report.json          # Validation results
└── metadata.json                   # Project metadata
```

### Serialization Format

- **Primary Format**: JSON (human-readable, versionable)
- **Log Format**: JSONL (streamable, append-only)
- **Checkpoint Format**: JSON with compressed state for large data

### Versioning

All serialized entities include a `schemaVersion` field for forward compatibility:

```json
{
  "schemaVersion": "1.0",
  "entity": "FinalClarifiedSpecification",
  "data": { ... }
}
```

---

## Validation Summary

### Cross-Entity Validation

1. **Referential Integrity**: All ID references MUST resolve to existing entities
2. **State Consistency**: Entity states MUST be consistent across relationships
3. **Temporal Consistency**: Timestamps MUST be logically ordered
4. **Content Integrity**: All checksums MUST match content

### Invariants

1. **Determinism**: Same FCS + config → Same GenerationOutput (verified by checksums)
2. **Traceability**: Every GeneratedFile MUST trace back to FCS requirement
3. **Immutability**: FCS MUST NOT change after creation
4. **Completeness**: ValidationReport MUST validate entire GenerationOutput

---

## Next Steps

With the data model defined, proceed to:
1. Generate contracts (if applicable - may be N/A for CLI-only tool)
2. Generate quickstart.md for development setup
3. Update agent context with chosen technologies
