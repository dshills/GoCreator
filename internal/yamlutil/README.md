# Enhanced YAML Parser

A wrapper around `gopkg.in/yaml.v3` that provides detailed error messages with line numbers, context, and actionable suggestions.

## Features

- **Line Numbers**: Pinpoint exactly where errors occur
- **Context Display**: Shows surrounding lines with an arrow (→) marking the error location
- **Helpful Suggestions**: Provides specific guidance on how to fix common YAML errors
- **Strict Mode**: Optionally reject unknown fields for stricter validation

## Usage

### Basic Unmarshaling

```go
import "github.com/dshills/gocreator/internal/yamlutil"

var config map[string]interface{}
err := yamlutil.Unmarshal(data, &config)
if err != nil {
    // Error contains line number, context, and suggestion
    fmt.Println(err)
}
```

### Strict Mode (Reject Unknown Fields)

```go
type Config struct {
    Name    string `yaml:"name"`
    Version string `yaml:"version"`
}

var config Config
err := yamlutil.UnmarshalStrict(data, &config)
// Will error if YAML contains fields not in struct
```

### Validation Only

```go
// Just check if YAML is valid without unmarshaling
err := yamlutil.Validate(data)
```

## Error Output Examples

### Indentation Error

```
YAML parse error at line 3, column 0: mapping values are not allowed in this context

Context:
     1 |
     2 | name: TestProject
→    3 |   version: 1.0
     4 | config:
     5 |   timeout: 30s

Suggestion: Check for incorrect indentation or missing colon. YAML requires
consistent indentation (use spaces, not tabs) and colons for key-value pairs.
```

### Type Mismatch Error

```
YAML parse error at line 5, column 0: cannot unmarshal !!str `three` into int

Context:
     3 | config:
     4 |   max_parallel: 4
→    5 |   retries: "three"
     6 |   timeout: 30s

Suggestion: Value type doesn't match expected type. Check the documentation
for the correct data type (string, number, boolean, array, or object).
```

### Tab Indentation Error

```
YAML parse error at line 2, column 0: found character that cannot start any token

Context:
     1 | name: test
→    2 | 	version: 1.0
     3 | config:

Suggestion: Line contains tabs. Replace tabs with spaces for proper YAML indentation.
```

### Unknown Field (Strict Mode)

```
YAML parse error at line 4, column 0: field unknown_field not found in type Config

Context:
     2 | schema_version: "1.0"
     3 | name: "test"
→    4 | unknown_field: "value"
     5 |

Suggestion: Field name not recognized. Check for typos or refer to the
documentation for valid field names.
```

## Integration

This package has been integrated into:

- **Specification Parser** (`internal/spec/parser_yaml.go`) - Project specification parsing
- **Markdown Parser** (`internal/spec/parser_markdown.go`) - YAML frontmatter parsing
- **Workflow Loader** (`internal/workflow/loader.go`) - Workflow definition loading
- **Provider Config** (`internal/providers/config.go`) - Multi-provider configuration

All YAML parsing in the codebase now provides enhanced error messages automatically.

## Testing

Run the test suite:

```bash
go test ./internal/yamlutil/...
```

See examples:

```bash
go run ./cmd/yaml-demo/main.go
```

## Error Suggestions

The parser provides contextual suggestions for common errors:

| Error Pattern | Suggestion |
|---------------|------------|
| Mapping values not allowed | Check indentation and colons |
| Cannot unmarshal | Type mismatch - check expected type |
| Duplicate key | Remove or rename duplicate entry |
| Tab character | Replace tabs with spaces |
| Unknown field | Check for typos in field names |
| Required field missing | Add the missing field |
| Invalid value | Check documentation for valid values |

## API Reference

### `Unmarshal(data []byte, v interface{}) error`

Parses YAML content with enhanced error reporting. Equivalent to `yaml.Unmarshal` but with better errors.

### `UnmarshalStrict(data []byte, v interface{}) error`

Parses YAML in strict mode, rejecting unknown fields. Useful for catching configuration errors.

### `Validate(data []byte) error`

Checks if YAML is valid without unmarshaling into a specific type.

### `ParseError` Type

```go
type ParseError struct {
    Line       int    // Line number (1-indexed)
    Column     int    // Column number (1-indexed)
    Message    string // Error message
    Context    string // Surrounding lines with arrow
    Suggestion string // How to fix the error
}
```

Implements the `error` interface with a formatted multi-line output.
