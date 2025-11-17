# GoCreator

**Autonomous Go Software Generation System**

GoCreator is a command-line tool that transforms project specifications into complete, functioning Go codebases. It analyzes specifications, resolves ambiguities through interactive clarification, and generates deterministic, tested, validated code.

## Features

- **Specification Analysis**: Parse specifications in YAML, JSON, or Markdown format
- **Interactive Clarification**: Identify ambiguities and resolve them through targeted questions
- **Autonomous Generation**: Generate complete Go projects including source, tests, and configuration
- **Deterministic Output**: Same specification + configuration = identical output every time
- **Comprehensive Validation**: Build, lint, and test validation with detailed error reporting
- **Workflow Control**: Execute individual phases (clarify, generate, validate) or the complete pipeline

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/dshills/gocreator.git
cd gocreator

# Build from source
go build -o bin/gocreator ./cmd/gocreator

# Install to GOPATH
go install ./cmd/gocreator
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
- `--timeout DURATION` - Timeout for validation operations (default: 5m)

**Description:**

The validation phase:
1. **Build Validation**: Runs `go build` and captures compilation errors
2. **Lint Validation**: Runs `golangci-lint` and reports style issues
3. **Test Validation**: Runs `go test ./...` and captures test results and coverage
4. **Report Generation**: Aggregates results with per-file error mappings

Validation failures do not trigger automatic repairs. Use validation output to guide specification updates and regeneration.

**Examples:**

```bash
# Validate the default generated project
gocreator validate ./generated

# Validate a specific project directory
gocreator validate ./my-project

# Validate with custom timeout
gocreator validate ./my-project --timeout 10m
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
- `--format FORMAT` - Output format: json, yaml (default: json)

**Description:**

Produces a Final Clarified Specification (FCS) in machine-readable format. The FCS is the complete, deterministic specification used as the blueprint for code generation.

**Examples:**

```bash
# Output FCS as JSON (default)
gocreator dump-fcs ./my-spec.yaml

# Output FCS as YAML
gocreator dump-fcs ./my-spec.yaml --format yaml

# Output FCS with batch answers
gocreator dump-fcs ./my-spec.yaml --batch ./answers.json > fcs.json
```

#### `version`

Print version information.

### Global Options

- `-c, --config FILE` - Configuration file path (default: .gocreator.yaml)
- `--log-level LEVEL` - Log level: debug, info, warn, error (default: info)
- `--log-format FORMAT` - Log format: console, json (default: console)

## Specification Format

GoCreator accepts specifications in multiple formats. Create a spec file describing your desired system:

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

Create a `.gocreator.yaml` file in your project root:

```yaml
llm:
  provider: anthropic          # anthropic, openai, google
  model: claude-sonnet-4       # Model to use
  temperature: 0.0             # 0.0 for deterministic output
  api_key: ${ANTHROPIC_API_KEY} # Use environment variable

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

### Generation Timeouts

**Problem**: Generation takes longer than 90 seconds

**Solution**:
1. Check specification complexity (verify it's a reasonable scope)
2. Check LLM provider rate limits and availability
3. Increase timeout if needed (though 90s is the target for medium projects)

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

**Implementation Status**:
- ✅ Specification parsing (YAML, JSON, Markdown)
- ✅ Clarification engine (LangGraph-Go)
- ✅ Code generation (LangGraph-Go + GoFlow)
- ✅ Validation (build, lint, test)
- ✅ CLI commands (clarify, generate, validate, full, dump-fcs)
- ✅ Determinism verification
- ⏳ Performance optimizations
- ⏳ Incremental regeneration

See [tasks.md](specs/001-core-implementation/tasks.md) for detailed implementation progress.

## License

GoCreator is open source. See LICENSE file for details.

## Support

For issues, questions, or contributions:
1. Check existing documentation in `docs/` directory
2. Review specification and plan in `specs/001-core-implementation/`
3. Check troubleshooting section above
4. Report issues with full error context and `--log-level=debug` output
