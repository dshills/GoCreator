# Specification Parser Package

**Package**: `github.com/dshills/gocreator/internal/spec`

## Overview

The `spec` package provides comprehensive parsing, validation, and processing of GoCreator input specifications. It supports YAML, JSON, and Markdown formats with security validation and Final Clarified Specification (FCS) construction.

## Quick Start

```go
import "github.com/dshills/gocreator/internal/spec"

// Parse and validate in one step
spec, err := spec.ParseAndValidate(models.FormatYAML, yamlContent)
if err != nil {
    log.Fatal(err)
}

// Build FCS
fcs, err := spec.BuildFCS(spec)
if err != nil {
    log.Fatal(err)
}
```

## API Reference

### Parser Functions

#### `NewParser(format SpecFormat) (Parser, error)`
Creates a new parser for the specified format.

**Supported Formats**:
- `models.FormatYAML` - YAML files
- `models.FormatJSON` - JSON files
- `models.FormatMarkdown` - Markdown with YAML frontmatter

**Example**:
```go
parser, err := spec.NewParser(models.FormatYAML)
if err != nil {
    return err
}
spec, err := parser.Parse(content)
```

#### `ParseSpec(format SpecFormat, content string) (*models.InputSpecification, error)`
Convenience function to parse specification content.

**Example**:
```go
spec, err := spec.ParseSpec(models.FormatYAML, yamlContent)
```

#### `ParseAndValidate(format SpecFormat, content string) (*models.InputSpecification, error)`
Parses and validates specification in a single call.

**Example**:
```go
spec, err := spec.ParseAndValidate(models.FormatYAML, yamlContent)
// spec.State will be SpecStateValid if successful
```

### Validation

#### `NewValidator() *Validator`
Creates a new validator with default settings.

**Example**:
```go
validator := spec.NewValidator()
err := validator.Validate(inputSpec)
```

#### Validation Functions

- `ValidateInputSpec(spec)` - Validates required fields
- `ValidateSecurityConstraints(spec)` - Security checks
- `ValidateSchemaStructure(spec)` - Schema validation
- `ValidateForFCS(spec)` - Readiness for FCS conversion

### FCS Construction

#### `BuildFCS(spec *models.InputSpecification) (*models.FinalClarifiedSpecification, error)`
Builds a Final Clarified Specification from a validated InputSpecification.

**Example**:
```go
fcs, err := spec.BuildFCS(validatedSpec)
if err != nil {
    return err
}

// Access FCS data
fmt.Println(fcs.Requirements.Functional)
fmt.Println(fcs.Architecture.Packages)
fmt.Println(fcs.Metadata.Hash)
```

#### `NewFCSBuilder(spec) *FCSBuilder`
Creates a new FCS builder for custom construction.

**Example**:
```go
builder := spec.NewFCSBuilder(inputSpec)
fcs, err := builder.Build()
```

## Input Specification Format

### Required Fields

All specifications must include:

```yaml
name: ProjectName           # Required: Project name
description: Description    # Required: Project description
requirements:               # Required: Array of requirements
  - id: FR-001
    description: Requirement description
```

### Optional Sections

```yaml
architecture:
  packages:
    - name: package_name
      path: internal/package
      purpose: Package purpose
  dependencies:
    - name: github.com/lib/pq
      version: v1.10.0
      purpose: PostgreSQL driver

data_model:
  entities:
    - name: Entity
      package: models
      attributes:
        field: type

testing_strategy:
  coverage_target: 85.0
  unit_tests: true
  integration_tests: true

build_config:
  go_version: "1.23"
  output_path: ./bin
```

## Supported Formats

### YAML Format

```yaml
name: MyProject
description: A Go project
requirements:
  - id: FR-001
    description: First requirement
```

### JSON Format

```json
{
  "name": "MyProject",
  "description": "A Go project",
  "requirements": [
    {
      "id": "FR-001",
      "description": "First requirement"
    }
  ]
}
```

### Markdown Format

```markdown
---
name: MyProject
description: A Go project
requirements:
  - id: FR-001
    description: First requirement
---

# MyProject

Additional documentation here...
```

## Security Features

The validator performs comprehensive security checks:

### Path Security
- Detects path traversal (`../`)
- Blocks absolute paths to system directories
- Validates path depth
- Detects null bytes

### Command Security
- Detects command injection patterns
- Blocks dangerous commands
- Validates shell command fields

**Example**:
```go
// This will be blocked:
spec.ParsedData["output_path"] = "../../../etc/passwd"
validator.Validate(spec) // Returns error

spec.ParsedData["build_command"] = "make && rm -rf /"
validator.Validate(spec) // Returns error
```

## State Machine

Specifications follow this state progression:

```
Unparsed → Parsed → Valid (or Invalid)
```

**State Transitions**:
```go
// After parsing
spec.State == models.SpecStateParsed

// After successful validation
spec.TransitionTo(models.SpecStateValid)

// FCS can only be built from Valid state
fcs, err := spec.BuildFCS(spec)
```

## Error Handling

All errors are wrapped with context:

```go
spec, err := spec.ParseSpec(format, content)
if err != nil {
    // Errors are descriptive:
    // "failed to parse specification: failed to parse yaml: ..."
}

err = validator.Validate(spec)
if err != nil {
    // Security errors:
    // "security violation in field 'output_path': path traversal attempt detected"
    // Validation errors:
    // "required field 'name' is missing"
}
```

## Testing

Run tests:
```bash
go test ./internal/spec/... -v
go test ./internal/spec/... -cover
```

## Performance

- **Parsing**: ~1ms for typical specs
- **Validation**: ~1ms
- **FCS Construction**: ~2ms for complex specs
- **Thread-safe**: All parsers support concurrent use

## Examples

See `example_test.go` for runnable examples:
- `ExampleParseSpec`
- `ExampleParseAndValidate`
- `ExampleBuildFCS`
- `ExampleCompleteWorkflow`

Run examples:
```bash
go test -run Example ./internal/spec/
```

## Package Structure

```
internal/spec/
├── parser.go              # Parser interface and factory
├── parser_yaml.go         # YAML parser
├── parser_json.go         # JSON parser
├── parser_markdown.go     # Markdown parser
├── validator.go           # Validation logic
├── fcs.go                # FCS construction
├── parser_test.go        # Parser tests
├── validator_test.go     # Validator tests
├── fcs_test.go           # FCS tests
├── example_test.go       # Example usage
└── README.md             # This file
```

## Dependencies

- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/google/uuid` - UUID generation
- `github.com/yuin/goldmark` - Markdown support (future)

## Contributing

When adding new features:
1. Write tests first (TDD)
2. Follow existing patterns
3. Add security validation if needed
4. Update examples
5. Document new APIs

## License

See project LICENSE file.
