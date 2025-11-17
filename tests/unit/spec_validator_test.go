package unit

import (
	"testing"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/internal/spec"
	"github.com/stretchr/testify/assert"
)

// TestValidateInputSpecification tests the validation of parsed specifications
func TestValidateInputSpecification(t *testing.T) {
	tests := []struct {
		name        string
		spec        *models.InputSpecification
		wantErr     bool
		errContains string
	}{
		{
			name: "Valid specification with all required fields",
			spec: &models.InputSpecification{
				Format: models.FormatYAML,
				State:  models.SpecStateParsed,
				ParsedData: map[string]interface{}{
					"name":        "TestProject",
					"description": "A valid test specification",
					"requirements": []interface{}{
						map[string]interface{}{
							"id":          "FR-001",
							"description": "Test requirement",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Missing name field",
			spec: &models.InputSpecification{
				Format: models.FormatYAML,
				State:  models.SpecStateParsed,
				ParsedData: map[string]interface{}{
					"description":  "Missing name",
					"requirements": []interface{}{},
				},
			},
			wantErr:     true,
			errContains: "name",
		},
		{
			name: "Missing description field",
			spec: &models.InputSpecification{
				Format: models.FormatYAML,
				State:  models.SpecStateParsed,
				ParsedData: map[string]interface{}{
					"name":         "TestProject",
					"requirements": []interface{}{},
				},
			},
			wantErr:     true,
			errContains: "description",
		},
		{
			name: "Missing requirements field",
			spec: &models.InputSpecification{
				Format: models.FormatYAML,
				State:  models.SpecStateParsed,
				ParsedData: map[string]interface{}{
					"name":        "TestProject",
					"description": "Missing requirements",
				},
			},
			wantErr:     true,
			errContains: "requirements",
		},
		{
			name: "Nil parsed data",
			spec: &models.InputSpecification{
				Format:     models.FormatYAML,
				State:      models.SpecStateParsed,
				ParsedData: nil,
			},
			wantErr:     true,
			errContains: "parsed data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := spec.ValidateInputSpec(tt.spec)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateSecurityConstraints tests security validation
func TestValidateSecurityConstraints(t *testing.T) {
	tests := []struct {
		name        string
		spec        *models.InputSpecification
		wantErr     bool
		errContains string
	}{
		{
			name: "Safe output path",
			spec: &models.InputSpecification{
				Format: models.FormatYAML,
				ParsedData: map[string]interface{}{
					"name":         "TestProject",
					"description":  "Test",
					"requirements": []interface{}{},
					"output_path":  "./output",
				},
			},
			wantErr: false,
		},
		{
			name: "Path traversal attempt with ../",
			spec: &models.InputSpecification{
				Format: models.FormatYAML,
				ParsedData: map[string]interface{}{
					"name":         "TestProject",
					"description":  "Test",
					"requirements": []interface{}{},
					"output_path":  "../../../etc/passwd",
				},
			},
			wantErr:     true,
			errContains: "path traversal",
		},
		{
			name: "Absolute path outside project",
			spec: &models.InputSpecification{
				Format: models.FormatYAML,
				ParsedData: map[string]interface{}{
					"name":         "TestProject",
					"description":  "Test",
					"requirements": []interface{}{},
					"output_path":  "/etc/passwd",
				},
			},
			wantErr:     true,
			errContains: "absolute path",
		},
		{
			name: "Command injection in build command",
			spec: &models.InputSpecification{
				Format: models.FormatYAML,
				ParsedData: map[string]interface{}{
					"name":          "TestProject",
					"description":   "Test",
					"requirements":  []interface{}{},
					"build_command": "make && rm -rf /",
				},
			},
			wantErr:     true,
			errContains: "command injection",
		},
		{
			name: "Safe build command",
			spec: &models.InputSpecification{
				Format: models.FormatYAML,
				ParsedData: map[string]interface{}{
					"name":          "TestProject",
					"description":   "Test",
					"requirements":  []interface{}{},
					"build_command": "go build",
				},
			},
			wantErr: false,
		},
		{
			name: "Null byte in path",
			spec: &models.InputSpecification{
				Format: models.FormatYAML,
				ParsedData: map[string]interface{}{
					"name":         "TestProject",
					"description":  "Test",
					"requirements": []interface{}{},
					"output_path":  "test\x00file",
				},
			},
			wantErr:     true,
			errContains: "null byte",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := spec.ValidateSecurityConstraints(tt.spec)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateSchemaStructure tests structural validation
func TestValidateSchemaStructure(t *testing.T) {
	tests := []struct {
		name        string
		spec        *models.InputSpecification
		wantErr     bool
		errContains string
	}{
		{
			name: "Valid requirements array structure",
			spec: &models.InputSpecification{
				Format: models.FormatYAML,
				ParsedData: map[string]interface{}{
					"name":        "TestProject",
					"description": "Test",
					"requirements": []interface{}{
						map[string]interface{}{
							"id":          "FR-001",
							"description": "Requirement one",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Requirements is not an array",
			spec: &models.InputSpecification{
				Format: models.FormatYAML,
				ParsedData: map[string]interface{}{
					"name":         "TestProject",
					"description":  "Test",
					"requirements": "invalid string",
				},
			},
			wantErr:     true,
			errContains: "requirements must be an array",
		},
		{
			name: "Valid architecture structure",
			spec: &models.InputSpecification{
				Format: models.FormatYAML,
				ParsedData: map[string]interface{}{
					"name":         "TestProject",
					"description":  "Test",
					"requirements": []interface{}{},
					"architecture": map[string]interface{}{
						"packages": []interface{}{
							map[string]interface{}{
								"name": "main",
								"path": "cmd/main",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid architecture structure (not a map)",
			spec: &models.InputSpecification{
				Format: models.FormatYAML,
				ParsedData: map[string]interface{}{
					"name":         "TestProject",
					"description":  "Test",
					"requirements": []interface{}{},
					"architecture": "invalid",
				},
			},
			wantErr:     true,
			errContains: "architecture must be an object",
		},
		{
			name: "Empty requirements array is valid",
			spec: &models.InputSpecification{
				Format: models.FormatYAML,
				ParsedData: map[string]interface{}{
					"name":         "TestProject",
					"description":  "Test",
					"requirements": []interface{}{},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := spec.ValidateSchemaStructure(tt.spec)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestFullValidationPipeline tests the complete validation pipeline
func TestFullValidationPipeline(t *testing.T) {
	tests := []struct {
		name    string
		spec    *models.InputSpecification
		wantErr bool
	}{
		{
			name: "Fully valid specification",
			spec: &models.InputSpecification{
				Format: models.FormatYAML,
				State:  models.SpecStateParsed,
				ParsedData: map[string]interface{}{
					"name":        "ValidProject",
					"description": "A complete valid specification",
					"requirements": []interface{}{
						map[string]interface{}{
							"id":          "FR-001",
							"description": "First requirement",
							"priority":    "high",
						},
					},
					"architecture": map[string]interface{}{
						"packages": []interface{}{
							map[string]interface{}{
								"name": "main",
								"path": "cmd/main",
							},
						},
					},
					"output_path": "./output",
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid - missing required field",
			spec: &models.InputSpecification{
				Format: models.FormatYAML,
				State:  models.SpecStateParsed,
				ParsedData: map[string]interface{}{
					"name":         "InvalidProject",
					"requirements": []interface{}{},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid - security violation",
			spec: &models.InputSpecification{
				Format: models.FormatYAML,
				State:  models.SpecStateParsed,
				ParsedData: map[string]interface{}{
					"name":         "InvalidProject",
					"description":  "Has security issue",
					"requirements": []interface{}{},
					"output_path":  "../../etc/passwd",
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid - bad schema structure",
			spec: &models.InputSpecification{
				Format: models.FormatYAML,
				State:  models.SpecStateParsed,
				ParsedData: map[string]interface{}{
					"name":         "InvalidProject",
					"description":  "Has schema issue",
					"requirements": "not an array",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := spec.NewValidator()
			err := validator.Validate(tt.spec)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
