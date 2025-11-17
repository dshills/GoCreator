# CLI Interface Contract

**Version**: 1.0
**Date**: 2025-11-17
**Purpose**: Define the command-line interface contract for GoCreator

## Overview

GoCreator provides a command-line interface for all operations. This document defines the contract for commands, flags, exit codes, and output formats.

---

## Commands

### `gocreator clarify <spec-file>`

**Purpose**: Analyze specification and run clarification phase only

**Arguments**:
- `<spec-file>` (required): Path to specification file (YAML, JSON, or Markdown)

**Flags**:
- `--output`, `-o` (string): Output directory for FCS (default: current directory)
- `--config`, `-c` (string): Path to configuration file
- `--interactive`, `-i` (bool): Interactive mode for answering questions (default: true)
- `--batch` (string): Path to JSON file with pre-answered questions

**Output**:
- **Success**: Writes FCS to `<output>/.gocreator/fcs.json`
- **Console**: Displays clarification questions (if interactive) or confirmation message
- **Exit Code**: 0 on success, non-zero on failure

**Example**:
```bash
gocreator clarify ./my-project-spec.yaml --output ./output
```

**Output Format** (interactive):
```
GoCreator v1.0.0 - Clarification Phase

Analyzing specification: ./my-project-spec.yaml

Found 3 ambiguities requiring clarification:

Question 1/3: Authentication Method
What authentication method should the system use?
  [A] OAuth2 with JWT tokens
  [B] Session-based authentication
  [C] API keys
  [D] Custom (provide your answer)

Your choice: _
```

**Output Format** (batch):
```
GoCreator v1.0.0 - Clarification Phase

Analyzing specification: ./my-project-spec.yaml
Loading batch answers from: ./clarifications.json
Applied 3 clarifications successfully

Final Clarified Specification written to: ./output/.gocreator/fcs.json
```

---

### `gocreator generate <spec-file>`

**Purpose**: Run clarification + generation phases (skip validation)

**Arguments**:
- `<spec-file>` (required): Path to specification file

**Flags**:
- `--output`, `-o` (string): Output directory for generated code (default: `./generated`)
- `--config`, `-c` (string): Path to configuration file
- `--resume` (bool): Resume from last checkpoint if available
- `--batch` (string): Path to JSON file with pre-answered questions
- `--dry-run` (bool): Show what would be generated without writing files

**Output**:
- **Success**: Writes complete project structure to `<output>/`
- **Console**: Progress updates during generation
- **Exit Code**: 0 on success, non-zero on failure

**Example**:
```bash
gocreator generate ./my-project-spec.yaml --output ./my-project
```

**Output Format**:
```
GoCreator v1.0.0 - Generation Phase

[1/4] Clarification
  ✓ Specification analyzed (0 ambiguities)
  ✓ FCS constructed

[2/4] Planning
  ✓ Architecture planned (12 packages)
  ✓ File tree generated (47 files)

[3/4] Code Generation
  ✓ Package internal/spec (4 files) [elapsed: 2.3s]
  ✓ Package internal/clarify (3 files) [elapsed: 3.1s]
  ✓ Package internal/generate (4 files) [elapsed: 4.2s]
  ...
  ✓ Tests generated (23 test files) [elapsed: 12.1s]

[4/4] Finalization
  ✓ go.mod created
  ✓ Makefile created
  ✓ README.md created

Generation complete! [total: 45.3s]
Output written to: ./my-project/

Next steps:
  cd ./my-project
  go mod tidy
  make test
```

---

### `gocreator validate <project-root>`

**Purpose**: Run build, lint, and test validation on existing project

**Arguments**:
- `<project-root>` (required): Path to project directory to validate

**Flags**:
- `--config`, `-c` (string): Path to configuration file
- `--skip-build` (bool): Skip build validation
- `--skip-lint` (bool): Skip lint validation
- `--skip-tests` (bool): Skip test validation
- `--report`, `-r` (string): Output validation report to file (JSON format)

**Output**:
- **Success**: Displays validation results
- **Console**: Detailed results for each validation phase
- **Exit Code**: 0 if all validations pass, non-zero if any fail

**Example**:
```bash
gocreator validate ./my-project --report ./validation.json
```

**Output Format**:
```
GoCreator v1.0.0 - Validation Phase

Validating project: ./my-project

[1/3] Build Validation
  Running: go build ./...
  ✓ Build successful [elapsed: 5.2s]

[2/3] Lint Validation
  Running: golangci-lint run ./...
  ✗ Found 3 issues:
    - internal/spec/parser.go:45: ineffassign: ineffectual assignment to err
    - internal/clarify/graph.go:120: errcheck: error return value not checked
    - cmd/gocreator/main.go:23: gosec: use of weak random number generator

[3/3] Test Validation
  Running: go test ./...
  ✓ All tests passed (47/47) [coverage: 85.3%] [elapsed: 15.7s]

Validation Result: FAILED (1/3 checks passed)

Detailed report written to: ./validation.json
```

---

### `gocreator full <spec-file>`

**Purpose**: Run complete pipeline (clarify + generate + validate)

**Arguments**:
- `<spec-file>` (required): Path to specification file

**Flags**:
- `--output`, `-o` (string): Output directory (default: `./generated`)
- `--config`, `-c` (string): Path to configuration file
- `--batch` (string): Path to JSON file with pre-answered questions
- `--resume` (bool): Resume from last checkpoint
- `--report`, `-r` (string): Output validation report to file

**Output**:
- **Success**: Complete project with validation results
- **Console**: Combined output from all phases
- **Exit Code**: 0 if generation and validation both succeed

**Example**:
```bash
gocreator full ./my-project-spec.yaml --output ./my-project
```

**Output Format**: Combined output from `generate` + `validate`

---

### `gocreator dump-fcs <spec-file>`

**Purpose**: Output Final Clarified Specification as JSON

**Arguments**:
- `<spec-file>` (required): Path to specification file

**Flags**:
- `--output`, `-o` (string): Output file path (default: stdout)
- `--batch` (string): Path to JSON file with pre-answered questions
- `--pretty` (bool): Pretty-print JSON (default: true)

**Output**:
- **Success**: Outputs FCS JSON
- **Console**: FCS JSON or confirmation message
- **Exit Code**: 0 on success

**Example**:
```bash
gocreator dump-fcs ./my-project-spec.yaml --output ./fcs.json
```

---

### `gocreator version`

**Purpose**: Display version information

**Flags**: None

**Output**:
```
GoCreator v1.0.0
Commit: abc123def456
Built: 2025-11-17T10:30:00Z
Go version: go1.21.0
```

**Exit Code**: Always 0

---

## Global Flags

Available for all commands:

- `--config`, `-c` (string): Path to configuration file (default: `./.gocreator.yaml` or `~/.config/gocreator/config.yaml`)
- `--log-level` (string): Log level (debug, info, warn, error) (default: info)
- `--log-format` (string): Log format (console, json) (default: console)
- `--help`, `-h`: Display help for command
- `--version`, `-v`: Display version (same as `gocreator version`)

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (invalid arguments, config errors) |
| 2 | Specification parsing/validation error |
| 3 | Clarification phase error (LLM provider failure, etc.) |
| 4 | Generation phase error |
| 5 | Validation phase error (build/lint/test failures) |
| 6 | File system error (permission denied, disk full) |
| 7 | Network error (LLM provider unreachable) |
| 8 | Internal error (unexpected panic, etc.) |

---

## Configuration File Format

**File**: `.gocreator.yaml` (YAML format)

```yaml
# LLM Provider Configuration
llm:
  provider: anthropic  # or: openai, google, etc.
  model: claude-sonnet-4
  temperature: 0.0
  api_key: ${ANTHROPIC_API_KEY}  # Environment variable reference
  timeout: 60s
  max_tokens: 4096

# Workflow Configuration
workflow:
  root_dir: ./generated
  allow_commands:
    - go
    - git
    - golangci-lint
  max_parallel: 4
  checkpoint_interval: 10  # Checkpoint every N tasks

# Validation Configuration
validation:
  enable_linting: true
  linter_config: .golangci.yml
  enable_tests: true
  test_timeout: 5m
  required_coverage: 80.0  # Minimum test coverage percentage

# Logging Configuration
logging:
  level: info
  format: console
  output: stderr
  execution_log: .gocreator/execution.jsonl
```

**Environment Variables**:

All config values can be overridden with environment variables using `GOCREATOR_` prefix:

```bash
export GOCREATOR_LLM_PROVIDER=anthropic
export GOCREATOR_LLM_API_KEY=sk-ant-...
export GOCREATOR_WORKFLOW_MAX_PARALLEL=8
```

---

## Batch Clarification Format

**File**: JSON format with question answers

```json
{
  "version": "1.0",
  "answers": [
    {
      "question_id": "q1-auth-method",
      "selected_option": "A"
    },
    {
      "question_id": "q2-database",
      "custom_answer": "PostgreSQL with pgvector extension"
    },
    {
      "question_id": "q3-deployment",
      "selected_option": "C"
    }
  ]
}
```

---

## Output Directory Structure

After successful generation, the output directory contains:

```
<output-dir>/
├── .gocreator/                     # GoCreator metadata
│   ├── fcs.json                    # Final Clarified Specification
│   ├── generation_plan.json        # Generation plan
│   ├── execution.jsonl            # Execution log
│   ├── validation_report.json     # Validation results (if validated)
│   └── checkpoints/               # Execution checkpoints
│       ├── checkpoint_001.json
│       └── checkpoint_002.json
├── cmd/                           # Generated CLI (if applicable)
├── internal/                      # Generated internal packages
├── pkg/                           # Generated public packages
├── tests/                         # Generated tests
├── go.mod                         # Go module definition
├── go.sum                         # Go dependencies
├── Makefile                       # Build automation
├── README.md                      # Generated documentation
└── .golangci.yml                  # Linter configuration
```

---

## Error Output Format

All errors are output in a consistent format:

```
Error: <error-type>

<detailed-message>

<context-information>

For help, run: gocreator <command> --help
```

**Example**:
```
Error: Specification Validation Failed

The input specification contains 2 validation errors:

1. Missing required field: requirements.functional
   Location: /spec

2. Invalid value for field: metadata.version
   Location: /metadata/version
   Expected: semver format (e.g., "1.0.0")
   Got: "v1"

For help, run: gocreator clarify --help
```

---

## JSON Output Format

When `--json` flag is used (future enhancement), all output is in JSON:

```json
{
  "command": "generate",
  "status": "success",
  "duration_ms": 45300,
  "result": {
    "files_generated": 47,
    "packages_created": 12,
    "tests_generated": 23
  },
  "output_dir": "./my-project"
}
```

---

## Contract Versioning

CLI contract follows semantic versioning:

- **Major**: Breaking changes to command structure or output formats
- **Minor**: New commands or flags (backward compatible)
- **Patch**: Bug fixes, documentation updates

Current Version: **1.0.0**

---

## Next Steps

With contracts defined, proceed to:
1. Generate quickstart.md for development setup
2. Update agent context with chosen technologies
