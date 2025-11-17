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

func TestInputSpecification_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name string
		spec *models.InputSpecification
	}{
		{
			name: "complete specification",
			spec: &models.InputSpecification{
				SchemaVersion: "1.0",
				ID:            uuid.New().String(),
				Format:        models.FormatYAML,
				Content:       "name: test\ndescription: test spec\nrequirements:\n  - FR-001: test requirement",
				ParsedData: map[string]interface{}{
					"name":        "test",
					"description": "test spec",
				},
				Metadata: models.SpecMetadata{
					CreatedAt: time.Now().UTC(),
					Author:    "test-user",
					Version:   "1.0",
				},
				ValidationErrors: []models.ValidationError{},
				State:            models.SpecStateValid,
			},
		},
		{
			name: "specification with validation errors",
			spec: &models.InputSpecification{
				SchemaVersion: "1.0",
				ID:            uuid.New().String(),
				Format:        models.FormatJSON,
				Content:       `{"name": "test"}`,
				ParsedData:    map[string]interface{}{"name": "test"},
				Metadata: models.SpecMetadata{
					CreatedAt: time.Now().UTC(),
					Author:    "test-user",
					Version:   "1.0",
				},
				ValidationErrors: []models.ValidationError{
					{
						Field:   "description",
						Message: "required field missing",
					},
				},
				State: models.SpecStateInvalid,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.spec)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// Unmarshal back
			var unmarshaled models.InputSpecification
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			// Verify key fields
			assert.Equal(t, tt.spec.ID, unmarshaled.ID)
			assert.Equal(t, tt.spec.Format, unmarshaled.Format)
			assert.Equal(t, tt.spec.Content, unmarshaled.Content)
			assert.Equal(t, tt.spec.State, unmarshaled.State)
			assert.Equal(t, tt.spec.SchemaVersion, unmarshaled.SchemaVersion)
		})
	}
}

func TestInputSpecification_Validate(t *testing.T) {
	tests := []struct {
		name    string
		spec    *models.InputSpecification
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid specification",
			spec: &models.InputSpecification{
				ID:      uuid.New().String(),
				Format:  models.FormatYAML,
				Content: "name: test\ndescription: test spec\nrequirements:\n  - FR-001: test",
				ParsedData: map[string]interface{}{
					"name":         "test",
					"description":  "test spec",
					"requirements": []interface{}{"FR-001: test"},
				},
				State: models.SpecStateValid,
			},
			wantErr: false,
		},
		{
			name: "missing required field - name",
			spec: &models.InputSpecification{
				ID:      uuid.New().String(),
				Format:  models.FormatYAML,
				Content: "description: test spec\nrequirements:\n  - FR-001: test",
				ParsedData: map[string]interface{}{
					"description":  "test spec",
					"requirements": []interface{}{"FR-001: test"},
				},
				State: models.SpecStateParsed,
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "missing required field - description",
			spec: &models.InputSpecification{
				ID:      uuid.New().String(),
				Format:  models.FormatYAML,
				Content: "name: test\nrequirements:\n  - FR-001: test",
				ParsedData: map[string]interface{}{
					"name":         "test",
					"requirements": []interface{}{"FR-001: test"},
				},
				State: models.SpecStateParsed,
			},
			wantErr: true,
			errMsg:  "description",
		},
		{
			name: "missing required field - requirements",
			spec: &models.InputSpecification{
				ID:      uuid.New().String(),
				Format:  models.FormatYAML,
				Content: "name: test\ndescription: test spec",
				ParsedData: map[string]interface{}{
					"name":        "test",
					"description": "test spec",
				},
				State: models.SpecStateParsed,
			},
			wantErr: true,
			errMsg:  "requirements",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInputSpecification_StateTransitions(t *testing.T) {
	tests := []struct {
		name          string
		fromState     models.SpecState
		toState       models.SpecState
		shouldSucceed bool
	}{
		{
			name:          "unparsed to parsed",
			fromState:     models.SpecStateUnparsed,
			toState:       models.SpecStateParsed,
			shouldSucceed: true,
		},
		{
			name:          "parsed to valid",
			fromState:     models.SpecStateParsed,
			toState:       models.SpecStateValid,
			shouldSucceed: true,
		},
		{
			name:          "parsed to invalid",
			fromState:     models.SpecStateParsed,
			toState:       models.SpecStateInvalid,
			shouldSucceed: true,
		},
		{
			name:          "invalid is terminal - cannot transition",
			fromState:     models.SpecStateInvalid,
			toState:       models.SpecStateValid,
			shouldSucceed: false,
		},
		{
			name:          "cannot skip parsed state",
			fromState:     models.SpecStateUnparsed,
			toState:       models.SpecStateValid,
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &models.InputSpecification{
				ID:    uuid.New().String(),
				State: tt.fromState,
			}

			err := spec.TransitionTo(tt.toState)
			if tt.shouldSucceed {
				require.NoError(t, err)
				assert.Equal(t, tt.toState, spec.State)
			} else {
				require.Error(t, err)
				assert.Equal(t, tt.fromState, spec.State) // State should not change
			}
		})
	}
}

func TestInputSpecification_Format(t *testing.T) {
	tests := []struct {
		name    string
		format  models.SpecFormat
		isValid bool
	}{
		{"yaml format", models.FormatYAML, true},
		{"json format", models.FormatJSON, true},
		{"markdown format", models.FormatMarkdown, true},
		{"invalid format", models.SpecFormat("xml"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &models.InputSpecification{
				ID:     uuid.New().String(),
				Format: tt.format,
			}

			isValid := spec.IsValidFormat()
			assert.Equal(t, tt.isValid, isValid)
		})
	}
}
