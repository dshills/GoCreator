# GoCreator

**Autonomous Go Software Generation System**

GoCreator is a command-line tool that transforms project specifications into complete, functioning Go codebases. It analyzes specifications, resolves ambiguities through interactive clarification, and generates deterministic, tested, validated code.

> **⚠️ Development Status**: GoCreator is currently in active development (v0.1.0-dev). The core functionality is implemented and tested, but the project is not yet production-ready. See [Project Status](#project-status) for details.

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
  - [Installation](#installation)
  - [Quick Start Guide](#quick-start-guide)
  - [Basic Usage](#basic-usage)
- [CLI Reference](#cli-reference)
  - [Commands](#commands)
  - [Global Options](#global-options)
- [Specification Format](#specification-format)
  - [Specification Best Practices](#specification-best-practices)
  - [Supported Formats](#supported-formats)
- [Configuration](#configuration)
- [Example Specifications](#example-specifications)
- [Workflow Examples](#workflow-examples)
- [Exit Codes](#exit-codes)
- [Troubleshooting](#troubleshooting)
- [Development & Contributing](#development--contributing)
- [Architecture](#architecture)
- [Project Status](#project-status)
- [Contributing](#contributing)
- [Support & Feedback](#support--feedback)
- [License](#license)

## Features

- **Specification Analysis**: Parse specifications in YAML, JSON, or Markdown format
- **Interactive Clarification**: Identify ambiguities and resolve them through targeted questions
- **Autonomous Generation**: Generate complete Go projects including source, tests, and configuration
- **Deterministic Output**: Same specification + configuration = identical output every time
- **Comprehensive Validation**: Build, lint, and test validation with detailed error reporting
- **Workflow Control**: Execute individual phases (clarify, generate, validate) or the complete pipeline
- **Prompt Caching**: Provider-native caching for 60-80% token cost reduction (Anthropic)
- **Incremental Regeneration**: Fine-grained change detection regenerates only modified files
- **Context Filtering**: Smart FCS filtering reduces prompt size by including only relevant context
- **Template-Based Generation**: Fast boilerplate generation without LLM calls

## Quick Start

### Installation

**Prerequisites**: Go 1.21+ (requires generics support)

```bash
# Clone the repository
git clone https://github.com/dshills/gocreator.git
cd gocreator

# Build from source
go build -o bin/gocreator ./cmd/gocreator

# Or use the Makefile
make build

# Optionally, install to GOPATH
go install ./cmd/gocreator
```

**Verify installation**:
```bash
./bin/gocreator version
```

### Quick Start Guide

Try GoCreator with the included example specification:

```bash
# 1. Ensure your API key is set
export ANTHROPIC_API_KEY=sk-ant-your-key-here

# 2. Generate a simple TODO CLI application
./bin/gocreator full ./examples/simple-spec.yaml --output ./todo-app

# 3. Check the generated code
ls -la ./todo-app

# 4. Review the validation results
# (The 'full' command automatically validates the generated code)
```

### Basic Usage

```bash
# Clarify a specification (interactive)
gocreator clarify ./my-spec.yaml

# Generate code from a specification
gocreator generate ./my-spec.yaml --output ./my-project

# Validate generated code
gocreator validate ./my-project

# Run complete pipeline (clarify → generate → validate)
gocreator full ./my-spec.yaml --output ./my-project

# Output the Final Clarified Specification
gocreator dump-fcs ./my-spec.yaml
```

## CLI Reference

### Commands

#### `clarify <spec-file>`

Analyze a specification and run the clarification phase.

**Options:**
- `-o, --output DIR` - Output directory for FCS (default: current directory)
- `--batch FILE` - Use pre-answered questions from JSON file

**Description:**

The clarification phase:
1. Parses and validates the input specification
2. Identifies ambiguities, missing constraints, and unclear requirements
3. Generates targeted questions for resolution
4. Produces a Final Clarified Specification (FCS)

**Examples:**

```bash
# Interactive mode (prompts for answers)
gocreator clarify ./my-spec.yaml

# Batch mode (uses pre-answered questions)
gocreator clarify ./my-spec.yaml --batch ./answers.json

# Specify output directory
gocreator clarify ./my-spec.yaml --output ./output
```

#### `generate <spec-file>`

Run clarification and generation phases.

**Options:**
- `-o, --output DIR` - Output directory for generated code (default: ./generated)
- `--batch FILE` - Use pre-answered questions from JSON file
- `--resume` - Resume from last checkpoint if available
- `--dry-run` - Show what would be generated without writing files

**Description:**

The generation phase:
1. **Clarification**: Analyzes specification and resolves ambiguities
2. **Planning**: Creates architecture plan and file structure
3. **Code Generation**: Generates complete project structure with source files, tests, and configuration
4. **Finalization**: Creates build files, documentation, and metadata

Validation is skipped (use `full` to include validation).

**Examples:**

```bash
# Basic generation
gocreator generate ./my-spec.yaml

# Specify output directory
gocreator generate ./my-spec.yaml --output ./my-project

# Resume from checkpoint
gocreator generate ./my-spec.yaml --resume

# Dry run (see what would be generated)
gocreator generate ./my-spec.yaml --dry-run

# Batch mode
gocreator generate ./my-spec.yaml --batch ./answers.json
```

#### `validate <path>`

Validate an existing project.

**Options:**
- `-r, --report FILE` - Output validation report to JSON file
- `--skip-build` - Skip build validation
- `--skip-lint` - Skip lint validation
- `--skip-tests` - Skip test validation

**Description:**

The validation phase:
1. **Build Validation**: Runs `go build` and captures compilation errors
2. **Lint Validation**: Runs `golangci-lint` and reports style issues
3. **Test Validation**: Runs `go test ./...` and captures test results and coverage
4. **Report Generation**: Aggregates results with per-file error mappings

All checks run by default. Use `--skip-*` flags to disable specific checks.

Validation failures do not trigger automatic repairs. Use validation output to guide specification updates and regeneration.

**Exit codes:**
- `0` - All validations passed
- `5` - One or more validations failed

**Examples:**

```bash
# Validate all checks
gocreator validate ./generated

# Validate a specific project directory
gocreator validate ./my-project

# Skip linting
gocreator validate ./my-project --skip-lint

# Save validation report to file
gocreator validate ./my-project --report ./validation.json
```

#### `full <spec-file>`

Execute the complete pipeline.

**Options:**
- `-o, --output DIR` - Output directory for generated code (default: ./generated)
- `--batch FILE` - Use pre-answered questions from JSON file
- `--resume` - Resume from last checkpoint if available

**Description:**

Executes the entire workflow in sequence:
1. **Clarification**: Analyzes and resolves ambiguities
2. **Generation**: Creates complete project structure
3. **Validation**: Builds, lints, and tests generated code

This is the recommended command for end-to-end project generation.

**Examples:**

```bash
# Complete pipeline with interactive clarification
gocreator full ./my-spec.yaml --output ./my-project

# Complete pipeline with batch mode
gocreator full ./my-spec.yaml --output ./my-project --batch ./answers.json
```

#### `dump-fcs <spec-file>`

Output the Final Clarified Specification.

**Options:**
- `--batch FILE` - Use pre-answered questions from JSON file
- `-o, --output FILE` - Output file path (default: stdout)
- `--pretty` - Pretty-print JSON (default: true)

**Description:**

Produces a Final Clarified Specification (FCS) in JSON format. The FCS is the complete, deterministic specification used as the blueprint for code generation.

The FCS contains:
- Fully resolved requirements
- Clarification decisions
- Architectural constraints
- Implementation details

**Examples:**

```bash
# Output FCS to stdout (pretty-printed JSON)
gocreator dump-fcs ./my-spec.yaml

# Save to file
gocreator dump-fcs ./my-spec.yaml --output ./fcs.json

# Compact JSON (no pretty-printing)
gocreator dump-fcs ./my-spec.yaml --pretty=false

# Batch mode with output file
gocreator dump-fcs ./my-spec.yaml --batch ./answers.json --output ./fcs.json
```

#### `version`

Print version information.

### Global Options

All commands support these global flags:

- `-c, --config FILE` - Configuration file path (default: `.gocreator.yaml`)
- `--log-level LEVEL` - Log level: `debug`, `info`, `warn`, `error` (default: `info`)
- `--log-format FORMAT` - Log format: `console`, `json` (default: `console`)
- `-h, --help` - Help for any command
- `-v, --version` - Display version information

**Examples:**

```bash
# Run with debug logging
gocreator generate ./spec.yaml --log-level=debug

# Use custom config file
gocreator full ./spec.yaml --config ./custom-config.yaml

# JSON logging for CI/CD
gocreator generate ./spec.yaml --log-format=json
```

## Specification Format

GoCreator accepts specifications in multiple formats (YAML, JSON, or Markdown). Create a spec file describing your desired system.

### Specification Best Practices

**Key principles:**
- ✅ **Focus on WHAT, not HOW** - Describe requirements and user needs, not implementation details
- ✅ **Technology-agnostic** - Avoid mentioning specific libraries, frameworks, or APIs
- ✅ **Testable requirements** - Each requirement should be independently verifiable
- ✅ **User scenarios** - Describe how users interact with the system (Given/When/Then format)
- ❌ **Avoid implementation details** - Don't specify database schemas, API endpoints, or code structure

**Required sections:**
1. **User Scenarios** - Prioritized user journeys with acceptance criteria
2. **Functional Requirements** - What the system must do
3. **Success Criteria** - Measurable outcomes that define success

See `examples/simple-spec.yaml` for a complete example.

### Supported Formats

### YAML Format

```yaml
name: my-project
description: A sample Go project
version: "1.0.0"

requirements:
  functional:
    - System MUST provide REST API for user management
    - System MUST store data in PostgreSQL
  non_functional:
    - Response time < 100ms for 95th percentile
    - Support 1000 concurrent users

entities:
  - name: User
    fields:
      - name: ID
        type: UUID
      - name: Email
        type: string
  - name: Post
    fields:
      - name: ID
        type: UUID
      - name: Title
        type: string
```

### JSON Format

```json
{
  "name": "my-project",
  "description": "A sample Go project",
  "version": "1.0.0",
  "requirements": {
    "functional": [
      "System MUST provide REST API for user management"
    ]
  }
}
```

### Markdown Format

```markdown
---
name: my-project
description: A sample Go project
---

# Requirements

## Functional
- System MUST provide REST API
- System MUST support user authentication
```

## Configuration

### Environment Variables

GoCreator requires an API key for the LLM provider. Set the environment variable before running:

```bash
# For Anthropic (Claude)
export ANTHROPIC_API_KEY=sk-ant-...

# For OpenAI
export OPENAI_API_KEY=sk-...

# For Google
export GOOGLE_API_KEY=...
```

### Configuration File

Create a `.gocreator.yaml` file in your project root (optional - uses defaults if not present):

```yaml
llm:
  provider: anthropic          # anthropic, openai, google
  model: claude-sonnet-4       # Model to use
  temperature: 0.0             # 0.0 for deterministic output
  api_key: ${ANTHROPIC_API_KEY} # Use environment variable
  enable_caching: true         # Enable prompt caching (Anthropic only)
  cache_ttl: 5m                # Cache TTL: 5m or 1h (default: 5m)

workflow:
  root_dir: ./generated        # Where to generate code
  allow_commands:              # Allowed shell commands
    - go
    - git
    - golangci-lint
  max_parallel: 4              # Parallel execution limit

validation:
  enable_linting: true         # Run golangci-lint
  linter_config: .golangci.yml # Linter configuration
  enable_tests: true           # Run tests
  test_timeout: 5m             # Test timeout

logging:
  level: info                  # Log level
  format: console              # console or json
  output: stderr               # Output destination
  execution_log: .gocreator/execution.jsonl  # Execution audit log
```

## Example Specifications

The repository includes example specifications in the `examples/` directory:

- **`simple-spec.yaml`** - A minimal TODO CLI application (beginner-level, ~10 second generation)
- **`medium-spec.yaml`** - A more complex REST API with database integration (medium complexity)
- **`clarifications.json`** - Example batch clarification answers file

Try them out:

```bash
# Quick test with the simple example
./bin/gocreator generate ./examples/simple-spec.yaml --output ./test-output

# Full pipeline with validation
./bin/gocreator full ./examples/simple-spec.yaml --output ./test-output
```

## Workflow Examples

### Example 1: Simple REST API

```bash
# Create specification
cat > api-spec.yaml <<EOF
name: user-api
description: Simple user management API
version: "1.0.0"

requirements:
  functional:
    - System MUST provide REST API endpoints for CRUD operations on users
    - System MUST store users in PostgreSQL
    - System MUST include unit and integration tests
EOF

# Generate code
gocreator generate ./api-spec.yaml --output ./user-api

# Validate
gocreator validate ./user-api
```

### Example 2: Batch Mode (CI/CD)

```bash
# Create answers file
cat > answers.json <<EOF
{
  "questions": [
    {"id": "q1", "answer": "REST"},
    {"id": "q2", "answer": "PostgreSQL"}
  ]
}
EOF

# Run without interactive prompts (suitable for CI/CD)
gocreator full ./my-spec.yaml --output ./project --batch ./answers.json
```

### Example 3: Iterative Development

```bash
# Initial generation
gocreator generate ./spec.yaml --output ./project

# Validate
gocreator validate ./project

# Update specification based on feedback
vim ./spec.yaml

# Regenerate (deterministic - same output structure)
gocreator generate ./spec.yaml --output ./project

# Validate again
gocreator validate ./project
```

## Exit Codes

| Code | Meaning | Solution |
|------|---------|----------|
| 0 | Success | - |
| 1 | General error | Check error message |
| 2 | Specification error | Validate spec format and content |
| 3 | Clarification failed | Review specification and try again |
| 4 | Generation failed | Check FCS and configuration |
| 5 | Validation failed | Review error report and update spec |
| 6 | File system error | Check permissions and disk space |
| 7 | Network error | Check LLM provider connectivity |
| 8 | Configuration error | Verify .gocreator.yaml format |
| 9 | Internal error | Report issue with full log (--log-level=debug) |

## Troubleshooting

### LLM Provider Issues

**Problem**: "Failed to create LLM client"

```bash
# Verify API key is set
echo $ANTHROPIC_API_KEY

# Test API access
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "content-type: application/json" \
  -d '{"model":"claude-sonnet-4","messages":[{"role":"user","content":"test"}],"max_tokens":10}'
```

**Solution**: Ensure API key is set and valid:
```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

### File System Issues

**Problem**: "Permission denied" or "Directory not found"

**Solution**: Ensure output directory exists and is writable:
```bash
mkdir -p ./generated
chmod 755 ./generated
gocreator generate ./spec.yaml --output ./generated
```

### Generation Performance

**Note**: Generation time varies based on specification complexity and LLM provider response times.

**Performance targets**:
- Simple projects (< 5 files): < 30 seconds
- Medium projects (5-20 files): < 90 seconds
- Complex projects (> 20 files): Variable (depends on scope)

**If generation is slower than expected**:
1. Check specification complexity (ensure reasonable scope)
2. Verify LLM provider rate limits and availability
3. Check network connectivity to LLM provider
4. Review execution logs with `--log-level=debug` for bottlenecks

### Validation Failures

**Problem**: Generated code fails validation

**Action**:
1. Review the validation report
2. Identify specific errors (build errors, lint issues, test failures)
3. Update specification to address issues
4. Regenerate and validate again

## Development & Contributing

See [DEVELOPMENT.md](docs/DEVELOPMENT.md) for:
- Development setup and environment
- Running tests and linting
- Code organization and patterns
- How to add new features
- Testing and code review requirements

## Architecture

See [ARCHITECTURE.md](docs/ARCHITECTURE.md) for:
- System architecture overview
- Component responsibilities
- Data flow through the system
- Separation of reasoning and action
- Determinism guarantees

## Project Status

**Current Version**: 0.1.0-dev

**Recent Updates**: Performance optimizations merged (prompt caching, incremental regeneration)

**Implementation Status**:
- ✅ **Completed**:
  - Specification parsing (YAML, JSON, Markdown)
  - Clarification engine with LangGraph-Go
  - Autonomous code generation engine
  - Comprehensive validation (build, lint, test)
  - Full CLI suite (clarify, generate, validate, full, dump-fcs)
  - Determinism verification
  - Comprehensive test coverage (unit + integration tests)
  - Complete documentation suite
  - Security hardening (bounded file operations, permission checks)
  - Multi-LLM provider support (Anthropic, OpenAI, Google)
  - Prompt caching for 60-80% token cost reduction (Anthropic)
  - Incremental regeneration with fine-grained change detection
  - Context filtering to reduce prompt size
  - Template-based boilerplate generation

- ⏳ **In Progress**:
  - Batch code generation for parallel file processing
  - Release preparation (goreleaser, automated releases)
  - Additional performance optimizations

**Build Status**: ✅ All tests passing, builds successfully
**Lint Status**: ✅ All linting checks pass
**Test Coverage**: Target 80% (comprehensive unit and integration tests)

See [specs/003-performance-optimization/tasks.md](specs/003-performance-optimization/tasks.md) for performance optimization progress and task tracking.

## Contributing

Contributions are welcome! Please see [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for:
- Development environment setup
- Code style and testing requirements
- How to run tests and linting
- Pull request guidelines

Before contributing, familiarize yourself with the project's architecture and design principles documented in:
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) - System architecture and design
- [CLAUDE.md](CLAUDE.md) - Project overview and development workflow
- [specs/001-core-implementation/](specs/001-core-implementation/) - Current implementation specification

## Support & Feedback

For issues, questions, or contributions:

1. **Check documentation first**:
   - README (this file) for usage and troubleshooting
   - [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for development setup
   - [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for system design
   - [Troubleshooting](#troubleshooting) section above

2. **Review existing resources**:
   - Example specifications in `examples/`
   - Implementation tasks in `specs/001-core-implementation/tasks.md`
   - Architecture documentation

3. **Report issues** with:
   - Full error context
   - Command used and flags
   - Specification file (if applicable)
   - Debug output (`--log-level=debug`)
   - GoCreator version (`gocreator version`)
   - Go version (`go version`)

4. **Feature requests**: Check the project roadmap in `specs/001-core-implementation/tasks.md` to see if your feature is already planned.

## License

GoCreator is open source. See LICENSE file for details.
