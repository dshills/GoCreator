package unit

import (
	"testing"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/internal/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParserFactory tests the parser factory function
func TestParserFactory(t *testing.T) {
	tests := []struct {
		name        string
		format      models.SpecFormat
		wantErr     bool
		errContains string
	}{
		{
			name:    "YAML format returns YAML parser",
			format:  models.FormatYAML,
			wantErr: false,
		},
		{
			name:    "JSON format returns JSON parser",
			format:  models.FormatJSON,
			wantErr: false,
		},
		{
			name:    "Markdown format returns Markdown parser",
			format:  models.FormatMarkdown,
			wantErr: false,
		},
		{
			name:        "Invalid format returns error",
			format:      models.SpecFormat("invalid"),
			wantErr:     true,
			errContains: "unsupported format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := spec.NewParser(tt.format)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, parser)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, parser)
			}
		})
	}
}

// TestYAMLParser tests the YAML parser implementation
func TestYAMLParser(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *models.InputSpecification)
	}{
		{
			name: "Valid YAML specification",
			content: `
name: TestProject
description: A test project specification
requirements:
  - id: FR-001
    description: First requirement
  - id: FR-002
    description: Second requirement
architecture:
  packages:
    - name: main
      path: cmd/main
      purpose: Entry point
`,
			wantErr: false,
			validate: func(t *testing.T, spec *models.InputSpecification) {
				assert.Equal(t, models.FormatYAML, spec.Format)
				assert.Equal(t, models.SpecStateParsed, spec.State)
				assert.NotNil(t, spec.ParsedData)
				assert.Equal(t, "TestProject", spec.ParsedData["name"])
				assert.Equal(t, "A test project specification", spec.ParsedData["description"])
			},
		},
		{
			name: "YAML with missing required fields",
			content: `
name: TestProject
architecture:
  packages: []
`,
			wantErr: false, // Parsing succeeds, validation should catch this
			validate: func(t *testing.T, spec *models.InputSpecification) {
				assert.NotNil(t, spec.ParsedData)
				// Validation will be tested separately
			},
		},
		{
			name:        "Invalid YAML syntax",
			content:     `name: TestProject\n\tinvalid: [unclosed`,
			wantErr:     true,
			errContains: "yaml",
		},
		{
			name:        "Empty content",
			content:     "",
			wantErr:     true,
			errContains: "empty",
		},
		{
			name: "YAML with malicious path attempt",
			content: `
name: TestProject
description: Test
requirements:
  - id: FR-001
    description: Test
output_path: ../../../etc/passwd
`,
			wantErr: false, // Parsing succeeds, validator should catch security issues
			validate: func(t *testing.T, spec *models.InputSpecification) {
				assert.NotNil(t, spec.ParsedData)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := spec.NewParser(models.FormatYAML)
			require.NoError(t, err)

			result, err := parser.Parse(tt.content)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

// TestJSONParser tests the JSON parser implementation
func TestJSONParser(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *models.InputSpecification)
	}{
		{
			name: "Valid JSON specification",
			content: `{
  "name": "TestProject",
  "description": "A test project specification",
  "requirements": [
    {
      "id": "FR-001",
      "description": "First requirement"
    },
    {
      "id": "FR-002",
      "description": "Second requirement"
    }
  ],
  "architecture": {
    "packages": [
      {
        "name": "main",
        "path": "cmd/main",
        "purpose": "Entry point"
      }
    ]
  }
}`,
			wantErr: false,
			validate: func(t *testing.T, spec *models.InputSpecification) {
				assert.Equal(t, models.FormatJSON, spec.Format)
				assert.Equal(t, models.SpecStateParsed, spec.State)
				assert.NotNil(t, spec.ParsedData)
				assert.Equal(t, "TestProject", spec.ParsedData["name"])
				assert.Equal(t, "A test project specification", spec.ParsedData["description"])
			},
		},
		{
			name:        "Invalid JSON syntax",
			content:     `{"name": "TestProject", "invalid": }`,
			wantErr:     true,
			errContains: "json",
		},
		{
			name:        "Empty JSON",
			content:     "",
			wantErr:     true,
			errContains: "empty",
		},
		{
			name: "JSON with command injection attempt",
			content: `{
  "name": "TestProject",
  "description": "Test",
  "requirements": [
    {
      "id": "FR-001",
      "description": "Test"
    }
  ],
  "build_command": "rm -rf / && make"
}`,
			wantErr: false, // Parsing succeeds, validator should catch security issues
			validate: func(t *testing.T, spec *models.InputSpecification) {
				assert.NotNil(t, spec.ParsedData)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := spec.NewParser(models.FormatJSON)
			require.NoError(t, err)

			result, err := parser.Parse(tt.content)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

// TestMarkdownParser tests the Markdown parser with frontmatter
func TestMarkdownParser(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *models.InputSpecification)
	}{
		{
			name: "Valid Markdown with YAML frontmatter",
			content: `---
name: TestProject
description: A test project specification
requirements:
  - id: FR-001
    description: First requirement
  - id: FR-002
    description: Second requirement
---

# TestProject

This is the main content of the specification.

## Requirements

The requirements are defined in the frontmatter above.
`,
			wantErr: false,
			validate: func(t *testing.T, spec *models.InputSpecification) {
				assert.Equal(t, models.FormatMarkdown, spec.Format)
				assert.Equal(t, models.SpecStateParsed, spec.State)
				assert.NotNil(t, spec.ParsedData)
				assert.Equal(t, "TestProject", spec.ParsedData["name"])
				assert.Equal(t, "A test project specification", spec.ParsedData["description"])
			},
		},
		{
			name: "Markdown without frontmatter",
			content: `# TestProject

This is a specification without frontmatter.
`,
			wantErr:     true,
			errContains: "frontmatter",
		},
		{
			name: "Markdown with invalid frontmatter",
			content: `---
name: TestProject
invalid: [unclosed
---

# Content
`,
			wantErr:     true,
			errContains: "frontmatter",
		},
		{
			name:        "Empty markdown",
			content:     "",
			wantErr:     true,
			errContains: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := spec.NewParser(models.FormatMarkdown)
			require.NoError(t, err)

			result, err := parser.Parse(tt.content)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

// TestParserConcurrency tests that parsers are safe for concurrent use
func TestParserConcurrency(t *testing.T) {
	parser, err := spec.NewParser(models.FormatYAML)
	require.NoError(t, err)

	content := `
name: TestProject
description: Concurrent test
requirements:
  - id: FR-001
    description: Test requirement
`

	// Run multiple goroutines concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := parser.Parse(content)
			assert.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
