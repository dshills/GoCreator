package unit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClarificationRequest_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name    string
		request *models.ClarificationRequest
	}{
		{
			name: "complete clarification request",
			request: &models.ClarificationRequest{
				SchemaVersion: "1.0",
				ID:            uuid.New().String(),
				SpecID:        uuid.New().String(),
				Questions: []models.Question{
					{
						ID:       "q1",
						Topic:    "Authentication",
						Context:  "requirements.FR-005",
						Question: "Which authentication method should be used?",
						Options: []models.Option{
							{
								Label:        "JWT",
								Description:  "JSON Web Tokens for stateless auth",
								Implications: "Requires token management and expiry handling",
							},
							{
								Label:        "Session",
								Description:  "Server-side session storage",
								Implications: "Requires session store and scaling considerations",
							},
						},
						UserAnswer: nil,
					},
				},
				Ambiguities: []models.Ambiguity{
					{
						Type:        "missing_constraint",
						Location:    "requirements.FR-010",
						Description: "No performance requirements specified",
						Severity:    "important",
					},
				},
				CreatedAt: time.Now().UTC(),
			},
		},
		{
			name: "request with answered questions",
			request: &models.ClarificationRequest{
				SchemaVersion: "1.0",
				ID:            uuid.New().String(),
				SpecID:        uuid.New().String(),
				Questions: []models.Question{
					{
						ID:       "q1",
						Topic:    "Database",
						Context:  "architecture",
						Question: "Which database should be used?",
						Options: []models.Option{
							{Label: "PostgreSQL", Description: "Relational database"},
							{Label: "MongoDB", Description: "Document database"},
						},
						UserAnswer: stringPtr("PostgreSQL"),
					},
				},
				Ambiguities: []models.Ambiguity{},
				CreatedAt:   time.Now().UTC(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.request)
			require.NoError(t, err)

			var unmarshaled models.ClarificationRequest
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, tt.request.ID, unmarshaled.ID)
			assert.Equal(t, tt.request.SpecID, unmarshaled.SpecID)
			assert.Equal(t, len(tt.request.Questions), len(unmarshaled.Questions))
			assert.Equal(t, len(tt.request.Ambiguities), len(unmarshaled.Ambiguities))
		})
	}
}

func TestClarificationRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request *models.ClarificationRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request with questions",
			request: &models.ClarificationRequest{
				ID:     uuid.New().String(),
				SpecID: uuid.New().String(),
				Questions: []models.Question{
					{
						ID:       "q1",
						Question: "Test?",
						Options:  []models.Option{{Label: "A"}, {Label: "B"}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid request with ambiguities",
			request: &models.ClarificationRequest{
				ID:     uuid.New().String(),
				SpecID: uuid.New().String(),
				Ambiguities: []models.Ambiguity{
					{Type: "missing_constraint", Description: "Test"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid - no questions or ambiguities",
			request: &models.ClarificationRequest{
				ID:          uuid.New().String(),
				SpecID:      uuid.New().String(),
				Questions:   []models.Question{},
				Ambiguities: []models.Ambiguity{},
			},
			wantErr: true,
			errMsg:  "at least 1 question or ambiguity",
		},
		{
			name: "invalid - question with only 1 option",
			request: &models.ClarificationRequest{
				ID:     uuid.New().String(),
				SpecID: uuid.New().String(),
				Questions: []models.Question{
					{
						ID:       "q1",
						Question: "Test?",
						Options:  []models.Option{{Label: "A"}},
					},
				},
			},
			wantErr: true,
			errMsg:  "2-4 options",
		},
		{
			name: "invalid - question with more than 4 options",
			request: &models.ClarificationRequest{
				ID:     uuid.New().String(),
				SpecID: uuid.New().String(),
				Questions: []models.Question{
					{
						ID:       "q1",
						Question: "Test?",
						Options: []models.Option{
							{Label: "A"}, {Label: "B"}, {Label: "C"},
							{Label: "D"}, {Label: "E"},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "2-4 options",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClarificationResponse_JSONMarshaling(t *testing.T) {
	answer1 := "JWT"
	customAnswer := "Custom solution"

	tests := []struct {
		name     string
		response *models.ClarificationResponse
	}{
		{
			name: "response with selected options",
			response: &models.ClarificationResponse{
				SchemaVersion: "1.0",
				ID:            uuid.New().String(),
				RequestID:     uuid.New().String(),
				Answers: map[string]models.Answer{
					"q1": {
						QuestionID:     "q1",
						SelectedOption: &answer1,
					},
				},
				AnsweredAt: time.Now().UTC(),
			},
		},
		{
			name: "response with custom answers",
			response: &models.ClarificationResponse{
				SchemaVersion: "1.0",
				ID:            uuid.New().String(),
				RequestID:     uuid.New().String(),
				Answers: map[string]models.Answer{
					"q1": {
						QuestionID:   "q1",
						CustomAnswer: &customAnswer,
					},
				},
				AnsweredAt: time.Now().UTC(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.response)
			require.NoError(t, err)

			var unmarshaled models.ClarificationResponse
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, tt.response.ID, unmarshaled.ID)
			assert.Equal(t, tt.response.RequestID, unmarshaled.RequestID)
			assert.Equal(t, len(tt.response.Answers), len(unmarshaled.Answers))
		})
	}
}

func TestClarificationResponse_Validate(t *testing.T) {
	selectedOption := "Option A"
	customAnswer := "Custom answer"

	tests := []struct {
		name     string
		response *models.ClarificationResponse
		request  *models.ClarificationRequest
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid response with selected option",
			response: &models.ClarificationResponse{
				ID:        uuid.New().String(),
				RequestID: "req1",
				Answers: map[string]models.Answer{
					"q1": {
						QuestionID:     "q1",
						SelectedOption: &selectedOption,
					},
				},
			},
			request: &models.ClarificationRequest{
				ID: "req1",
				Questions: []models.Question{
					{ID: "q1", Question: "Test?"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid response with custom answer",
			response: &models.ClarificationResponse{
				ID:        uuid.New().String(),
				RequestID: "req1",
				Answers: map[string]models.Answer{
					"q1": {
						QuestionID:   "q1",
						CustomAnswer: &customAnswer,
					},
				},
			},
			request: &models.ClarificationRequest{
				ID: "req1",
				Questions: []models.Question{
					{ID: "q1", Question: "Test?"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid - answer has both selected and custom",
			response: &models.ClarificationResponse{
				ID:        uuid.New().String(),
				RequestID: "req1",
				Answers: map[string]models.Answer{
					"q1": {
						QuestionID:     "q1",
						SelectedOption: &selectedOption,
						CustomAnswer:   &customAnswer,
					},
				},
			},
			request: &models.ClarificationRequest{
				ID: "req1",
				Questions: []models.Question{
					{ID: "q1", Question: "Test?"},
				},
			},
			wantErr: true,
			errMsg:  "either SelectedOption OR CustomAnswer",
		},
		{
			name: "invalid - missing answer for question",
			response: &models.ClarificationResponse{
				ID:        uuid.New().String(),
				RequestID: "req1",
				Answers:   map[string]models.Answer{},
			},
			request: &models.ClarificationRequest{
				ID: "req1",
				Questions: []models.Question{
					{ID: "q1", Question: "Test?"},
				},
			},
			wantErr: true,
			errMsg:  "must answer all questions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.response.ValidateAgainst(tt.request)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
