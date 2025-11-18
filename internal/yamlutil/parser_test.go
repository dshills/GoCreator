package yamlutil

import (
	"strings"
	"testing"
)

func TestUnmarshal_ValidYAML(t *testing.T) {
	yaml := `
name: test
version: 1.0
config:
  timeout: 30s
  retries: 3
`

	var result map[string]interface{}
	err := Unmarshal([]byte(yaml), &result)

	if err != nil {
		t.Fatalf("Expected no error for valid YAML, got: %v", err)
	}

	if result["name"] != "test" {
		t.Errorf("Expected name=test, got: %v", result["name"])
	}
}

func TestUnmarshal_InvalidIndentation(t *testing.T) {
	yaml := `
name: test
  version: 1.0
`

	var result map[string]interface{}
	err := Unmarshal([]byte(yaml), &result)

	if err == nil {
		t.Fatal("Expected error for invalid indentation")
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("Expected *ParseError, got: %T", err)
	}

	if parseErr.Line == 0 {
		t.Error("Expected line number in error")
	}

	if parseErr.Context == "" {
		t.Error("Expected context in error")
	}

	if !strings.Contains(parseErr.Message, "mapping") {
		t.Errorf("Expected 'mapping' in error message, got: %s", parseErr.Message)
	}

	if parseErr.Suggestion == "" {
		t.Error("Expected suggestion in error")
	}

	t.Logf("Error output:\n%s", err.Error())
}

func TestUnmarshal_TypeMismatch(t *testing.T) {
	yaml := `
schema_version: "1.0"
config:
  max_parallel: "not a number"
`

	type Config struct {
		SchemaVersion string `yaml:"schema_version"`
		Config        struct {
			MaxParallel int `yaml:"max_parallel"`
		} `yaml:"config"`
	}

	var result Config
	err := Unmarshal([]byte(yaml), &result)

	if err == nil {
		t.Fatal("Expected error for type mismatch")
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("Expected *ParseError, got: %T", err)
	}

	if parseErr.Line == 0 {
		t.Error("Expected line number in error")
	}

	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "line") {
		t.Error("Expected line number in formatted error")
	}

	if !strings.Contains(strings.ToLower(errorMsg), "suggestion") {
		t.Error("Expected suggestion in formatted error")
	}

	t.Logf("Error output:\n%s", errorMsg)
}

func TestUnmarshal_MissingColon(t *testing.T) {
	yaml := `
name test
version: 1.0
`

	var result map[string]interface{}
	err := Unmarshal([]byte(yaml), &result)

	if err == nil {
		t.Fatal("Expected error for missing colon")
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("Expected *ParseError, got: %T", err)
	}

	if parseErr.Line == 0 {
		t.Error("Expected line number in error")
	}

	t.Logf("Error output:\n%s", err.Error())
}

func TestUnmarshal_TabIndentation(t *testing.T) {
	yaml := "name: test\n\tversion: 1.0\n"

	var result map[string]interface{}
	err := Unmarshal([]byte(yaml), &result)

	if err == nil {
		t.Fatal("Expected error for tab indentation")
	}

	errorMsg := strings.ToLower(err.Error())
	if !strings.Contains(errorMsg, "tab") && !strings.Contains(errorMsg, "indent") {
		t.Errorf("Expected 'tab' or 'indent' in error message, got: %s", err.Error())
	}

	t.Logf("Error output:\n%s", err.Error())
}

func TestUnmarshal_DuplicateKey(t *testing.T) {
	yaml := `
name: test
version: 1.0
name: duplicate
`

	var result map[string]interface{}
	err := Unmarshal([]byte(yaml), &result)

	// Note: yaml.v3 may or may not error on duplicate keys depending on version
	// This test documents the behavior
	if err != nil {
		t.Logf("Duplicate key error (expected): %s", err.Error())
	} else {
		t.Logf("YAML library accepted duplicate key (last value wins)")
	}
}

func TestUnmarshalStrict_UnknownField(t *testing.T) {
	yaml := `
schema_version: "1.0"
unknown_field: value
name: test
`

	type Config struct {
		SchemaVersion string `yaml:"schema_version"`
		Name          string `yaml:"name"`
	}

	var result Config
	err := UnmarshalStrict([]byte(yaml), &result)

	if err == nil {
		t.Fatal("Expected error for unknown field in strict mode")
	}

	errorMsg := strings.ToLower(err.Error())
	if !strings.Contains(errorMsg, "unknown") && !strings.Contains(errorMsg, "field") {
		t.Errorf("Expected 'unknown field' in error message, got: %s", err.Error())
	}

	t.Logf("Error output:\n%s", err.Error())
}

func TestUnmarshalStrict_ValidYAML(t *testing.T) {
	yaml := `
schema_version: "1.0"
name: test
`

	type Config struct {
		SchemaVersion string `yaml:"schema_version"`
		Name          string `yaml:"name"`
	}

	var result Config
	err := UnmarshalStrict([]byte(yaml), &result)

	if err != nil {
		t.Fatalf("Expected no error for valid YAML in strict mode, got: %v", err)
	}

	if result.Name != "test" {
		t.Errorf("Expected name=test, got: %v", result.Name)
	}
}

func TestValidate_ValidYAML(t *testing.T) {
	yaml := `
name: test
version: 1.0
list:
  - item1
  - item2
`

	err := Validate([]byte(yaml))

	if err != nil {
		t.Fatalf("Expected no error for valid YAML, got: %v", err)
	}
}

func TestValidate_InvalidYAML(t *testing.T) {
	yaml := `
name: test
  version: 1.0
    invalid: indentation
`

	err := Validate([]byte(yaml))

	if err == nil {
		t.Fatal("Expected error for invalid YAML")
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("Expected *ParseError, got: %T", err)
	}

	if parseErr.Line == 0 {
		t.Error("Expected line number in error")
	}
}

func TestExtractContext(t *testing.T) {
	content := `line 1
line 2
line 3
line 4
line 5
line 6
line 7`

	tests := []struct {
		name         string
		lineNum      int
		contextLines int
		wantContains string
	}{
		{
			name:         "middle line",
			lineNum:      4,
			contextLines: 2,
			wantContains: "→",
		},
		{
			name:         "first line",
			lineNum:      1,
			contextLines: 2,
			wantContains: "→    1",
		},
		{
			name:         "last line",
			lineNum:      7,
			contextLines: 2,
			wantContains: "→    7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context := extractContext(content, tt.lineNum, tt.contextLines)

			if context == "" {
				t.Error("Expected non-empty context")
			}

			if !strings.Contains(context, tt.wantContains) {
				t.Errorf("Expected context to contain %q, got:\n%s", tt.wantContains, context)
			}

			// Check that the marked line is present
			lines := strings.Split(context, "\n")
			foundMarker := false
			for _, line := range lines {
				if strings.HasPrefix(line, "→") {
					foundMarker = true
					break
				}
			}

			if !foundMarker {
				t.Error("Expected context to have marked error line with →")
			}

			t.Logf("Context:\n%s", context)
		})
	}
}

func TestGenerateSuggestion(t *testing.T) {
	tests := []struct {
		name            string
		message         string
		content         string
		line            int
		wantContains    string
		wantNotContains string
	}{
		{
			name:         "mapping values error",
			message:      "mapping values are not allowed in this context",
			content:      "name: test\n  version: 1.0",
			line:         2,
			wantContains: "indentation",
		},
		{
			name:         "unmarshal type error",
			message:      "cannot unmarshal string into int",
			content:      "retries: \"three\"",
			line:         1,
			wantContains: "type",
		},
		{
			name:         "duplicate key",
			message:      "duplicate key found",
			content:      "name: test\nname: duplicate",
			line:         2,
			wantContains: "duplicate",
		},
		{
			name:         "tab indentation",
			message:      "found character that cannot start any token",
			content:      "name: test\n\tversion: 1.0",
			line:         2,
			wantContains: "tab",
		},
		{
			name:         "unknown field",
			message:      "field unknown_field not found",
			content:      "unknown_field: value",
			line:         1,
			wantContains: "not recognized",
		},
		{
			name:         "generic error",
			message:      "some other error",
			content:      "name: test",
			line:         1,
			wantContains: "syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := generateSuggestion(tt.message, tt.content, tt.line)

			if suggestion == "" {
				t.Error("Expected non-empty suggestion")
			}

			suggestionLower := strings.ToLower(suggestion)
			if !strings.Contains(suggestionLower, tt.wantContains) {
				t.Errorf("Expected suggestion to contain %q, got: %s", tt.wantContains, suggestion)
			}

			if tt.wantNotContains != "" && strings.Contains(suggestionLower, tt.wantNotContains) {
				t.Errorf("Expected suggestion NOT to contain %q, got: %s", tt.wantNotContains, suggestion)
			}

			t.Logf("Suggestion: %s", suggestion)
		})
	}
}

func TestParseError_Error(t *testing.T) {
	err := &ParseError{
		Line:       5,
		Column:     12,
		Message:    "mapping values are not allowed",
		Context:    "  3 | name: test\n  4 | config:\n→ 5 |   version 1.0\n  6 | ",
		Suggestion: "Check for missing colon",
	}

	errorMsg := err.Error()

	// Check all components are present
	if !strings.Contains(errorMsg, "line 5") {
		t.Error("Expected line number in error message")
	}

	if !strings.Contains(errorMsg, "column 12") {
		t.Error("Expected column number in error message")
	}

	if !strings.Contains(errorMsg, "mapping values") {
		t.Error("Expected error message in output")
	}

	if !strings.Contains(errorMsg, "Context:") {
		t.Error("Expected context section in output")
	}

	if !strings.Contains(errorMsg, "Suggestion:") {
		t.Error("Expected suggestion section in output")
	}

	if !strings.Contains(errorMsg, "→") {
		t.Error("Expected error line marker in context")
	}

	t.Logf("Error output:\n%s", errorMsg)
}

func TestRealWorldExample_WorkflowConfig(t *testing.T) {
	yaml := `
schema_version: "1.0"
name: "Test Workflow"
version: "1.0"
config:
  max_parallel: 4
  retries: three
  timeout: "30s"
`

	type WorkflowConfig struct {
		SchemaVersion string `yaml:"schema_version"`
		Name          string `yaml:"name"`
		Version       string `yaml:"version"`
		Config        struct {
			MaxParallel int    `yaml:"max_parallel"`
			Retries     int    `yaml:"retries"`
			Timeout     string `yaml:"timeout"`
		} `yaml:"config"`
	}

	var result WorkflowConfig
	err := Unmarshal([]byte(yaml), &result)

	if err == nil {
		t.Fatal("Expected error for invalid retries value")
	}

	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "line") {
		t.Error("Expected line number in error")
	}

	if !strings.Contains(errorMsg, "Suggestion") {
		t.Error("Expected suggestion in error")
	}

	t.Logf("Error output:\n%s", errorMsg)
}

func TestRealWorldExample_ProviderConfig(t *testing.T) {
	yaml := `
schema_version: "1.0"

providers:
  openai-fast:
    type: openai
    model: gpt-4-turbo
    api_key: ${OPENAI_API_KEY}
  missing-type-provider:
    model: gpt-4
    api_key: secret

default_provider: openai-fast
`

	type ProviderConfig struct {
		Type   string `yaml:"type"`
		Model  string `yaml:"model"`
		APIKey string `yaml:"api_key"`
	}

	type MultiProviderConfig struct {
		SchemaVersion   string                    `yaml:"schema_version"`
		Providers       map[string]ProviderConfig `yaml:"providers"`
		DefaultProvider string                    `yaml:"default_provider"`
	}

	var result MultiProviderConfig
	err := UnmarshalStrict([]byte(yaml), &result)

	// In non-strict mode, missing fields are just zero values
	// This test demonstrates the difference between Unmarshal and UnmarshalStrict
	if err != nil {
		t.Logf("Strict mode caught issue: %s", err.Error())
	}

	// Now try with Unmarshal (non-strict)
	var result2 MultiProviderConfig
	err2 := Unmarshal([]byte(yaml), &result2)

	if err2 != nil {
		t.Fatalf("Non-strict unmarshal should succeed, got: %v", err2)
	}

	// The missing type field will be an empty string
	if result2.Providers["missing-type-provider"].Type != "" {
		t.Error("Expected empty type for provider with missing type field")
	}
}
