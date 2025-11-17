package unit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerationOutput_JSONMarshaling(t *testing.T) {
	completedAt := time.Now().UTC()

	tests := []struct {
		name   string
		output *models.GenerationOutput
	}{
		{
			name: "complete generation output",
			output: &models.GenerationOutput{
				SchemaVersion: "1.0",
				ID:            uuid.New().String(),
				PlanID:        uuid.New().String(),
				Files: []models.GeneratedFile{
					{
						Path:        "main.go",
						Content:     "package main\n\nfunc main() {}",
						Checksum:    computeChecksum("package main\n\nfunc main() {}"),
						GeneratedAt: time.Now().UTC(),
						Generator:   "code-generator",
					},
				},
				Patches: []models.Patch{
					{
						TargetFile: "existing.go",
						Diff:       "--- a/existing.go\n+++ b/existing.go\n@@ -1,1 +1,2 @@",
						AppliedAt:  time.Now().UTC(),
						Reversible: true,
					},
				},
				Metadata: models.OutputMetadata{
					StartedAt:   time.Now().UTC().Add(-5 * time.Minute),
					CompletedAt: &completedAt,
					Duration:    5 * time.Minute,
					FilesCount:  1,
					LinesCount:  3,
				},
				Status: models.OutputStatusCompleted,
			},
		},
		{
			name: "in-progress output",
			output: &models.GenerationOutput{
				SchemaVersion: "1.0",
				ID:            uuid.New().String(),
				PlanID:        uuid.New().String(),
				Files:         []models.GeneratedFile{},
				Patches:       []models.Patch{},
				Metadata: models.OutputMetadata{
					StartedAt:   time.Now().UTC(),
					CompletedAt: nil,
					FilesCount:  0,
					LinesCount:  0,
				},
				Status: models.OutputStatusInProgress,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.output)
			require.NoError(t, err)

			var unmarshaled models.GenerationOutput
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, tt.output.ID, unmarshaled.ID)
			assert.Equal(t, tt.output.PlanID, unmarshaled.PlanID)
			assert.Equal(t, tt.output.Status, unmarshaled.Status)
			assert.Equal(t, len(tt.output.Files), len(unmarshaled.Files))
		})
	}
}

func TestGenerationOutput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		output  *models.GenerationOutput
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid output with unique paths",
			output: &models.GenerationOutput{
				ID:     uuid.New().String(),
				PlanID: uuid.New().String(),
				Files: []models.GeneratedFile{
					{
						Path:     "main.go",
						Content:  "package main",
						Checksum: computeChecksum("package main"),
					},
					{
						Path:     "utils.go",
						Content:  "package main",
						Checksum: computeChecksum("package main"),
					},
				},
				Status: models.OutputStatusCompleted,
			},
			wantErr: false,
		},
		{
			name: "invalid - duplicate file paths",
			output: &models.GenerationOutput{
				ID:     uuid.New().String(),
				PlanID: uuid.New().String(),
				Files: []models.GeneratedFile{
					{Path: "main.go", Content: "package main", Checksum: "abc"},
					{Path: "main.go", Content: "package test", Checksum: "def"},
				},
				Status: models.OutputStatusCompleted,
			},
			wantErr: true,
			errMsg:  "duplicate file path",
		},
		{
			name: "invalid - checksum mismatch",
			output: &models.GenerationOutput{
				ID:     uuid.New().String(),
				PlanID: uuid.New().String(),
				Files: []models.GeneratedFile{
					{
						Path:     "main.go",
						Content:  "package main",
						Checksum: "wrong-checksum",
					},
				},
				Status: models.OutputStatusCompleted,
			},
			wantErr: true,
			errMsg:  "checksum mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.output.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGenerationOutput_StateTransitions(t *testing.T) {
	tests := []struct {
		name          string
		fromState     models.OutputStatus
		toState       models.OutputStatus
		shouldSucceed bool
	}{
		{
			name:          "pending to in_progress",
			fromState:     models.OutputStatusPending,
			toState:       models.OutputStatusInProgress,
			shouldSucceed: true,
		},
		{
			name:          "in_progress to completed",
			fromState:     models.OutputStatusInProgress,
			toState:       models.OutputStatusCompleted,
			shouldSucceed: true,
		},
		{
			name:          "in_progress to failed",
			fromState:     models.OutputStatusInProgress,
			toState:       models.OutputStatusFailed,
			shouldSucceed: true,
		},
		{
			name:          "invalid - pending to completed (skip in_progress)",
			fromState:     models.OutputStatusPending,
			toState:       models.OutputStatusCompleted,
			shouldSucceed: false,
		},
		{
			name:          "invalid - completed to in_progress (backwards)",
			fromState:     models.OutputStatusCompleted,
			toState:       models.OutputStatusInProgress,
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &models.GenerationOutput{
				ID:     uuid.New().String(),
				PlanID: uuid.New().String(),
				Status: tt.fromState,
			}

			err := output.TransitionTo(tt.toState)
			if tt.shouldSucceed {
				require.NoError(t, err)
				assert.Equal(t, tt.toState, output.Status)
			} else {
				require.Error(t, err)
				assert.Equal(t, tt.fromState, output.Status)
			}
		})
	}
}

func TestGeneratedFile_VerifyChecksum(t *testing.T) {
	content := "package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}"
	correctChecksum := computeChecksum(content)

	tests := []struct {
		name    string
		file    models.GeneratedFile
		isValid bool
	}{
		{
			name: "valid checksum",
			file: models.GeneratedFile{
				Path:     "main.go",
				Content:  content,
				Checksum: correctChecksum,
			},
			isValid: true,
		},
		{
			name: "invalid checksum",
			file: models.GeneratedFile{
				Path:     "main.go",
				Content:  content,
				Checksum: "wrong-checksum",
			},
			isValid: false,
		},
		{
			name: "empty checksum",
			file: models.GeneratedFile{
				Path:     "main.go",
				Content:  content,
				Checksum: "",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.file.VerifyChecksum()
			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

func TestPatch_JSONMarshaling(t *testing.T) {
	patch := &models.Patch{
		TargetFile: "existing.go",
		Diff: `--- a/existing.go
+++ b/existing.go
@@ -1,3 +1,4 @@
 package main

+// New comment
 func main() {}`,
		AppliedAt:  time.Now().UTC(),
		Reversible: true,
	}

	data, err := json.Marshal(patch)
	require.NoError(t, err)

	var unmarshaled models.Patch
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, patch.TargetFile, unmarshaled.TargetFile)
	assert.Equal(t, patch.Diff, unmarshaled.Diff)
	assert.Equal(t, patch.Reversible, unmarshaled.Reversible)
}

func computeChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}
