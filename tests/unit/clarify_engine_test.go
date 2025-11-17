package unit

import (
	"context"
	"testing"

	"github.com/dshills/gocreator/internal/clarify"
	"github.com/dshills/gocreator/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEngine(t *testing.T) {
	mockClient := &MockLLMClient{}

	config := clarify.EngineConfig{
		LLMClient:        mockClient,
		CheckpointDir:    t.TempDir(),
		EnableCheckpoint: true,
	}

	engine, err := clarify.NewEngine(config)
	require.NoError(t, err)
	assert.NotNil(t, engine)
}

func TestEngine_AnalyzeOnly(t *testing.T) {
	mockClient := &MockLLMClient{
		GenerateFunc: func(ctx context.Context, prompt string) (string, error) {
			return `[
				{
					"type": "missing_constraint",
					"location": "requirements",
					"description": "No timeout specified",
					"severity": "important"
				}
			]`, nil
		},
	}

	config := clarify.EngineConfig{
		LLMClient:        mockClient,
		EnableCheckpoint: false,
	}

	engine, err := clarify.NewEngine(config)
	require.NoError(t, err)

	spec := &models.InputSpecification{
		ID:      "spec-1",
		Format:  models.FormatYAML,
		Content: "Build a REST API",
		State:   models.SpecStateValid,
	}

	ctx := context.Background()
	ambiguities, err := engine.AnalyzeOnly(ctx, spec)

	require.NoError(t, err)
	assert.Equal(t, 1, len(ambiguities))
	assert.Equal(t, "missing_constraint", ambiguities[0].Type)
}

func TestEngine_GenerateRequest(t *testing.T) {
	callCount := 0
	mockClient := &MockLLMClient{
		GenerateFunc: func(ctx context.Context, prompt string) (string, error) {
			callCount++
			// First call: analysis
			if callCount == 1 {
				return `[{
					"type": "unclear_requirement",
					"location": "FR-001",
					"description": "Authentication method unclear",
					"severity": "critical"
				}]`, nil
			}
			// Second call: question generation
			return `[{
				"topic": "Authentication",
				"context": "FR-001 mentions authentication but doesn't specify method",
				"question": "Which authentication method should be used?",
				"options": [
					{
						"label": "JWT Tokens",
						"description": "JSON Web Tokens for stateless auth",
						"implications": "Requires token validation on each request"
					},
					{
						"label": "Session Cookies",
						"description": "Traditional session-based authentication",
						"implications": "Requires server-side session storage"
					}
				]
			}]`, nil
		},
	}

	config := clarify.EngineConfig{
		LLMClient:        mockClient,
		EnableCheckpoint: false,
	}

	engine, err := clarify.NewEngine(config)
	require.NoError(t, err)

	spec := &models.InputSpecification{
		ID:      "spec-1",
		Format:  models.FormatJSON,
		Content: "Build authentication system",
		State:   models.SpecStateValid,
	}

	ctx := context.Background()
	request, err := engine.GenerateRequest(ctx, spec)

	require.NoError(t, err)
	assert.NotNil(t, request)
	assert.Equal(t, spec.ID, request.SpecID)
	assert.Equal(t, 1, len(request.Questions))
	assert.Equal(t, 1, len(request.Ambiguities))
	assert.Equal(t, "Authentication", request.Questions[0].Topic)
	assert.Equal(t, 2, len(request.Questions[0].Options))
}

func TestEngine_ApplyAnswers(t *testing.T) {
	mockClient := &MockLLMClient{}

	config := clarify.EngineConfig{
		LLMClient:        mockClient,
		EnableCheckpoint: false,
	}

	engine, err := clarify.NewEngine(config)
	require.NoError(t, err)

	spec := &models.InputSpecification{
		ID:      "spec-1",
		Format:  models.FormatYAML,
		Content: "Specification content",
		State:   models.SpecStateValid,
	}

	selectedOption := "JWT Tokens"
	request := &models.ClarificationRequest{
		ID:     "req-1",
		SpecID: spec.ID,
		Questions: []models.Question{
			{
				ID:       "q1",
				Question: "Auth method?",
				Options: []models.Option{
					{Label: "JWT Tokens"},
					{Label: "OAuth"},
				},
			},
		},
	}

	response := &models.ClarificationResponse{
		ID:        "resp-1",
		RequestID: request.ID,
		Answers: map[string]models.Answer{
			"q1": {
				QuestionID:     "q1",
				SelectedOption: &selectedOption,
			},
		},
	}

	ctx := context.Background()
	fcs, err := engine.ApplyAnswers(ctx, spec, request, response)

	require.NoError(t, err)
	assert.NotNil(t, fcs)
	assert.Equal(t, spec.ID, fcs.OriginalSpecID)
	assert.Equal(t, 1, len(fcs.Metadata.Clarifications))
	assert.Equal(t, "q1", fcs.Metadata.Clarifications[0].QuestionID)
	assert.Equal(t, "JWT Tokens", fcs.Metadata.Clarifications[0].Answer)
}

func TestEngine_ValidateSpec(t *testing.T) {
	mockClient := &MockLLMClient{}

	config := clarify.EngineConfig{
		LLMClient:        mockClient,
		EnableCheckpoint: false,
	}

	engine, err := clarify.NewEngine(config)
	require.NoError(t, err)

	tests := []struct {
		name        string
		spec        *models.InputSpecification
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid spec",
			spec: &models.InputSpecification{
				ID:      "spec-1",
				Format:  models.FormatYAML,
				Content: "Content here",
				State:   models.SpecStateValid,
			},
			expectError: false,
		},
		{
			name:        "nil spec",
			spec:        nil,
			expectError: true,
			errorMsg:    "nil",
		},
		{
			name: "empty ID",
			spec: &models.InputSpecification{
				ID:      "",
				Format:  models.FormatYAML,
				Content: "Content",
				State:   models.SpecStateValid,
			},
			expectError: true,
			errorMsg:    "ID is empty",
		},
		{
			name: "empty content",
			spec: &models.InputSpecification{
				ID:      "spec-1",
				Format:  models.FormatYAML,
				Content: "",
				State:   models.SpecStateValid,
			},
			expectError: true,
			errorMsg:    "content is empty",
		},
		{
			name: "invalid state",
			spec: &models.InputSpecification{
				ID:      "spec-1",
				Format:  models.FormatYAML,
				Content: "Content",
				State:   models.SpecStateParsed,
			},
			expectError: true,
			errorMsg:    "valid state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateSpec(tt.spec)

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

func TestClarificationEngineRequest_Validate(t *testing.T) {
	tests := []struct {
		name        string
		request     *models.ClarificationRequest
		expectError bool
	}{
		{
			name: "valid request with questions",
			request: &models.ClarificationRequest{
				Questions: []models.Question{
					{
						Question: "Test?",
						Options:  []models.Option{{Label: "A"}, {Label: "B"}},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid request with ambiguities",
			request: &models.ClarificationRequest{
				Ambiguities: []models.Ambiguity{
					{Type: "missing_constraint", Description: "Test"},
				},
			},
			expectError: false,
		},
		{
			name: "invalid - no questions or ambiguities",
			request: &models.ClarificationRequest{
				Questions:   []models.Question{},
				Ambiguities: []models.Ambiguity{},
			},
			expectError: true,
		},
		{
			name: "invalid - question with too few options",
			request: &models.ClarificationRequest{
				Questions: []models.Question{
					{
						Question: "Test?",
						Options:  []models.Option{{Label: "Only one"}},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClarificationResponse_ValidateAgainst(t *testing.T) {
	request := &models.ClarificationRequest{
		Questions: []models.Question{
			{
				ID:       "q1",
				Question: "Question 1?",
				Options:  []models.Option{{Label: "A"}, {Label: "B"}},
			},
			{
				ID:       "q2",
				Question: "Question 2?",
				Options:  []models.Option{{Label: "X"}, {Label: "Y"}},
			},
		},
	}

	tests := []struct {
		name        string
		response    *models.ClarificationResponse
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid response with all questions answered",
			response: &models.ClarificationResponse{
				Answers: map[string]models.Answer{
					"q1": {QuestionID: "q1", SelectedOption: ptrString("A")},
					"q2": {QuestionID: "q2", SelectedOption: ptrString("X")},
				},
			},
			expectError: false,
		},
		{
			name: "invalid - missing answer",
			response: &models.ClarificationResponse{
				Answers: map[string]models.Answer{
					"q1": {QuestionID: "q1", SelectedOption: ptrString("A")},
				},
			},
			expectError: true,
			errorMsg:    "missing answer",
		},
		{
			name: "invalid - both selected and custom answer",
			response: &models.ClarificationResponse{
				Answers: map[string]models.Answer{
					"q1": {
						QuestionID:     "q1",
						SelectedOption: ptrString("A"),
						CustomAnswer:   ptrString("Custom"),
					},
					"q2": {QuestionID: "q2", SelectedOption: ptrString("X")},
				},
			},
			expectError: true,
			errorMsg:    "not both",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.response.ValidateAgainst(request)

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

func ptrString(s string) *string {
	return &s
}
