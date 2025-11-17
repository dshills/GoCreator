package unit

import (
	"context"
	"testing"

	"github.com/dshills/gocreator/internal/clarify"
	"github.com/dshills/gocreator/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLLMQuestionGenerator_Generate(t *testing.T) {
	tests := []struct {
		name          string
		ambiguities   []models.Ambiguity
		llmResponse   string
		expectError   bool
		expectedCount int
	}{
		{
			name: "generates questions from ambiguities",
			ambiguities: []models.Ambiguity{
				{
					Type:        "missing_constraint",
					Location:    "requirements.FR-003",
					Description: "No concurrency limit specified",
					Severity:    "critical",
				},
			},
			llmResponse: `[
				{
					"topic": "Concurrency Limits",
					"context": "Requirement FR-003 mentions concurrent users but no limit specified",
					"question": "What is the maximum number of concurrent users?",
					"options": [
						{
							"label": "100 users",
							"description": "Small scale deployment",
							"implications": "Simpler architecture"
						},
						{
							"label": "1000 users",
							"description": "Medium scale deployment",
							"implications": "Requires load balancing"
						}
					]
				}
			]`,
			expectError:   false,
			expectedCount: 1,
		},
		{
			name:          "handles no ambiguities",
			ambiguities:   []models.Ambiguity{},
			llmResponse:   "",
			expectError:   false,
			expectedCount: 0,
		},
		{
			name: "generates multiple questions",
			ambiguities: []models.Ambiguity{
				{Type: "missing_constraint", Description: "Issue 1"},
				{Type: "unclear_requirement", Description: "Issue 2"},
			},
			llmResponse: `[
				{
					"topic": "Topic 1",
					"question": "Question 1?",
					"options": [
						{"label": "Option A", "description": "Desc A", "implications": "Impl A"},
						{"label": "Option B", "description": "Desc B", "implications": "Impl B"}
					]
				},
				{
					"topic": "Topic 2",
					"question": "Question 2?",
					"options": [
						{"label": "Option X", "description": "Desc X", "implications": "Impl X"},
						{"label": "Option Y", "description": "Desc Y", "implications": "Impl Y"}
					]
				}
			]`,
			expectError:   false,
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockLLMClient{
				GenerateFunc: func(ctx context.Context, prompt string) (string, error) {
					// If no ambiguities, shouldn't be called
					if len(tt.ambiguities) == 0 {
						t.Fatal("Generate should not be called for empty ambiguities")
					}
					return tt.llmResponse, nil
				},
			}

			generator := clarify.NewLLMQuestionGenerator(mockClient)
			ctx := context.Background()

			questions, err := generator.Generate(ctx, tt.ambiguities)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(questions))

				// Verify UUIDs were assigned
				for _, q := range questions {
					assert.NotEmpty(t, q.ID)
				}
			}
		})
	}
}

func TestValidateQuestions(t *testing.T) {
	tests := []struct {
		name        string
		questions   []models.Question
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid questions",
			questions: []models.Question{
				{
					Question: "What authentication method?",
					Options: []models.Option{
						{Label: "JWT"},
						{Label: "OAuth"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "empty question text",
			questions: []models.Question{
				{
					Question: "",
					Options:  []models.Option{{Label: "A"}, {Label: "B"}},
				},
			},
			expectError: true,
			errorMsg:    "empty question text",
		},
		{
			name: "too few options",
			questions: []models.Question{
				{
					Question: "Question?",
					Options:  []models.Option{{Label: "Only one"}},
				},
			},
			expectError: true,
			errorMsg:    "fewer than 2 options",
		},
		{
			name: "too many options",
			questions: []models.Question{
				{
					Question: "Question?",
					Options: []models.Option{
						{Label: "A"},
						{Label: "B"},
						{Label: "C"},
						{Label: "D"},
						{Label: "E"},
					},
				},
			},
			expectError: true,
			errorMsg:    "more than 4 options",
		},
		{
			name: "empty option label",
			questions: []models.Question{
				{
					Question: "Question?",
					Options: []models.Option{
						{Label: "A"},
						{Label: ""},
					},
				},
			},
			expectError: true,
			errorMsg:    "empty label",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := clarify.ValidateQuestions(tt.questions)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPrioritizeQuestions(t *testing.T) {
	questions := []models.Question{
		{ID: "q1", Question: "Minor question"},
		{ID: "q2", Question: "Critical question"},
		{ID: "q3", Question: "Important question"},
	}

	ambiguities := []models.Ambiguity{
		{Severity: "minor"},
		{Severity: "critical"},
		{Severity: "important"},
	}

	prioritized := clarify.PrioritizeQuestions(questions, ambiguities)

	// First should be critical (highest priority)
	assert.Equal(t, "q2", prioritized[0].ID)
	// Second should be important
	assert.Equal(t, "q3", prioritized[1].ID)
	// Third should be minor
	assert.Equal(t, "q1", prioritized[2].ID)
}
