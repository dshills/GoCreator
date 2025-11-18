package spec

import (
	"testing"

	"github.com/dshills/gocreator/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			parser, err := NewParser(tt.format)
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
			name:        "Invalid YAML syntax",
			content:     `name: TestProject\n\tinvalid: [unclosed`,
			wantErr:     true,
			errContains: "parse",
		},
		{
			name:        "Empty content",
			content:     "",
			wantErr:     true,
			errContains: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(models.FormatYAML)
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
    }
  ]
}`,
			wantErr: false,
			validate: func(t *testing.T, spec *models.InputSpecification) {
				assert.Equal(t, models.FormatJSON, spec.Format)
				assert.Equal(t, models.SpecStateParsed, spec.State)
				assert.NotNil(t, spec.ParsedData)
				assert.Equal(t, "TestProject", spec.ParsedData["name"])
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(models.FormatJSON)
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
---

# TestProject

This is the main content.
`,
			wantErr: false,
			validate: func(t *testing.T, spec *models.InputSpecification) {
				assert.Equal(t, models.FormatMarkdown, spec.Format)
				assert.Equal(t, models.SpecStateParsed, spec.State)
				assert.NotNil(t, spec.ParsedData)
				assert.Equal(t, "TestProject", spec.ParsedData["name"])
			},
		},
		{
			name:        "Markdown without frontmatter",
			content:     `# TestProject\n\nNo frontmatter here.`,
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
			parser, err := NewParser(models.FormatMarkdown)
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

func TestParseAndValidate(t *testing.T) {
	tests := []struct {
		name    string
		format  models.SpecFormat
		content string
		wantErr bool
	}{
		{
			name:   "Valid complete specification",
			format: models.FormatYAML,
			content: `
name: ValidProject
description: A complete valid specification
requirements:
  - id: FR-001
    description: First requirement
`,
			wantErr: false,
		},
		{
			name:   "Missing required field",
			format: models.FormatYAML,
			content: `
name: InvalidProject
requirements: []
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := ParseAndValidate(tt.format, tt.content)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, spec)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, spec)
				assert.Equal(t, models.SpecStateValid, spec.State)
			}
		})
	}
}
