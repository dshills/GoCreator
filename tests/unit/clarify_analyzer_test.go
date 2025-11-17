package unit

import (
	"context"
	"testing"

	"github.com/dshills/gocreator/internal/clarify"
	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLLMClient for testing
type MockLLMClient struct {
	GenerateFunc func(ctx context.Context, prompt string) (string, error)
}

func (m *MockLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, prompt)
	}
	return "", nil
}

func (m *MockLLMClient) GenerateStructured(ctx context.Context, prompt string, schema interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockLLMClient) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	return "", nil
}

func (m *MockLLMClient) Provider() string {
	return "mock"
}

func (m *MockLLMClient) Model() string {
	return "mock-model"
}

func TestLLMAnalyzer_Analyze(t *testing.T) {
	tests := []struct {
		name          string
		spec          *models.InputSpecification
		llmResponse   string
		expectError   bool
		expectedCount int
		expectedTypes []string
	}{
		{
			name: "finds multiple ambiguities",
			spec: &models.InputSpecification{
				ID:      "spec-1",
				Format:  models.FormatMarkdown,
				Content: "Build a user management system",
			},
			llmResponse: `[
				{
					"type": "missing_constraint",
					"location": "requirements",
					"description": "No authentication method specified",
					"severity": "critical"
				},
				{
					"type": "unclear_requirement",
					"location": "features",
					"description": "User roles are not defined",
					"severity": "important"
				}
			]`,
			expectError:   false,
			expectedCount: 2,
			expectedTypes: []string{"missing_constraint", "unclear_requirement"},
		},
		{
			name: "finds no ambiguities",
			spec: &models.InputSpecification{
				ID:      "spec-2",
				Format:  models.FormatYAML,
				Content: "Well-defined specification",
			},
			llmResponse:   `[]`,
			expectError:   false,
			expectedCount: 0,
		},
		{
			name: "handles JSON with markdown wrapper",
			spec: &models.InputSpecification{
				ID:      "spec-3",
				Format:  models.FormatJSON,
				Content: "Some spec",
			},
			llmResponse:   "```json\n[{\"type\": \"conflict\", \"location\": \"req-1\", \"description\": \"Conflicting requirements\", \"severity\": \"critical\"}]\n```",
			expectError:   false,
			expectedCount: 1,
			expectedTypes: []string{"conflict"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockLLMClient{
				GenerateFunc: func(ctx context.Context, prompt string) (string, error) {
					// Verify prompt contains the spec content
					assert.Contains(t, prompt, tt.spec.Content)
					return tt.llmResponse, nil
				},
			}

			analyzer := clarify.NewLLMAnalyzer(mockClient)
			ctx := context.Background()

			ambiguities, err := analyzer.Analyze(ctx, tt.spec)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(ambiguities))

				if len(tt.expectedTypes) > 0 {
					for i, expectedType := range tt.expectedTypes {
						assert.Equal(t, expectedType, ambiguities[i].Type)
					}
				}
			}
		})
	}
}

func TestFilterAmbiguities(t *testing.T) {
	ambiguities := []models.Ambiguity{
		{Type: "conflict", Severity: "critical", Description: "Critical issue"},
		{Type: "missing_constraint", Severity: "important", Description: "Important issue"},
		{Type: "unclear_requirement", Severity: "minor", Description: "Minor issue"},
	}

	tests := []struct {
		name          string
		minSeverity   string
		expectedCount int
	}{
		{"filter critical only", "critical", 1},
		{"filter important and above", "important", 2},
		{"filter all", "minor", 3},
		{"unknown severity defaults to all", "unknown", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := clarify.FilterAmbiguities(ambiguities, tt.minSeverity)
			assert.Equal(t, tt.expectedCount, len(filtered))
		})
	}
}

func TestGroupAmbiguities(t *testing.T) {
	ambiguities := []models.Ambiguity{
		{Type: "conflict", Description: "Conflict 1"},
		{Type: "missing_constraint", Description: "Missing 1"},
		{Type: "conflict", Description: "Conflict 2"},
		{Type: "unclear_requirement", Description: "Unclear 1"},
	}

	groups := clarify.GroupAmbiguities(ambiguities)

	assert.Equal(t, 3, len(groups))
	assert.Equal(t, 2, len(groups["conflict"]))
	assert.Equal(t, 1, len(groups["missing_constraint"]))
	assert.Equal(t, 1, len(groups["unclear_requirement"]))
}
