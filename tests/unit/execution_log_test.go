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

func TestExecutionLog_JSONMarshaling(t *testing.T) {
	errorMsg := "operation failed"

	tests := []struct {
		name string
		log  *models.ExecutionLog
	}{
		{
			name: "execution log with various entries",
			log: &models.ExecutionLog{
				SchemaVersion:       "1.0",
				ID:                  uuid.New().String(),
				WorkflowExecutionID: uuid.New().String(),
				Entries: []models.LogEntry{
					{
						Timestamp: time.Now().UTC(),
						Level:     "info",
						Component: "parser",
						Operation: "parse_spec",
						Context:   map[string]interface{}{"file": "spec.yaml"},
						Message:   "Parsing specification",
						Error:     nil,
					},
					{
						Timestamp: time.Now().UTC(),
						Level:     "error",
						Component: "generator",
						Operation: "generate_file",
						Context:   map[string]interface{}{"file": "main.go"},
						Message:   "Failed to generate file",
						Error:     &errorMsg,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.log)
			require.NoError(t, err)

			var unmarshaled models.ExecutionLog
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, tt.log.ID, unmarshaled.ID)
			assert.Equal(t, tt.log.WorkflowExecutionID, unmarshaled.WorkflowExecutionID)
			assert.Equal(t, len(tt.log.Entries), len(unmarshaled.Entries))
		})
	}
}

func TestDecisionLog_JSONMarshaling(t *testing.T) {
	decisionLog := &models.DecisionLog{
		LogEntry: models.LogEntry{
			Timestamp: time.Now().UTC(),
			Level:     "info",
			Component: "planner",
			Operation: "select_architecture",
			Message:   "Architecture decision made",
		},
		Decision:     "Use layered architecture",
		Rationale:    "Better separation of concerns for this application",
		Alternatives: []string{"Hexagonal", "Clean Architecture"},
	}

	data, err := json.Marshal(decisionLog)
	require.NoError(t, err)

	var unmarshaled models.DecisionLog
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, decisionLog.Decision, unmarshaled.Decision)
	assert.Equal(t, decisionLog.Rationale, unmarshaled.Rationale)
	assert.Equal(t, len(decisionLog.Alternatives), len(unmarshaled.Alternatives))
}

func TestFileOperationLog_JSONMarshaling(t *testing.T) {
	fileOpLog := &models.FileOperationLog{
		LogEntry: models.LogEntry{
			Timestamp: time.Now().UTC(),
			Level:     "info",
			Component: "file_manager",
			Operation: "create_file",
			Message:   "Creating new file",
		},
		OperationType: "create",
		Path:          "internal/models/user.go",
		Checksum:      "abc123def456",
	}

	data, err := json.Marshal(fileOpLog)
	require.NoError(t, err)

	var unmarshaled models.FileOperationLog
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, fileOpLog.OperationType, unmarshaled.OperationType)
	assert.Equal(t, fileOpLog.Path, unmarshaled.Path)
	assert.Equal(t, fileOpLog.Checksum, unmarshaled.Checksum)
}

func TestCommandLog_JSONMarshaling(t *testing.T) {
	commandLog := &models.CommandLog{
		LogEntry: models.LogEntry{
			Timestamp: time.Now().UTC(),
			Level:     "info",
			Component: "executor",
			Operation: "run_command",
			Message:   "Executing go build",
		},
		Command:  "go",
		Args:     []string{"build", "-o", "bin/app"},
		ExitCode: 0,
		Stdout:   "Build successful",
		Stderr:   "",
		Duration: 2 * time.Second,
	}

	data, err := json.Marshal(commandLog)
	require.NoError(t, err)

	var unmarshaled models.CommandLog
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, commandLog.Command, unmarshaled.Command)
	assert.Equal(t, commandLog.Args, unmarshaled.Args)
	assert.Equal(t, commandLog.ExitCode, unmarshaled.ExitCode)
	assert.Equal(t, commandLog.Duration, unmarshaled.Duration)
}

func TestExecutionLog_Validate(t *testing.T) {
	tests := []struct {
		name    string
		log     *models.ExecutionLog
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid log with chronological entries",
			log: &models.ExecutionLog{
				ID:                  uuid.New().String(),
				WorkflowExecutionID: uuid.New().String(),
				Entries: []models.LogEntry{
					{
						Timestamp: time.Now().UTC().Add(-2 * time.Minute),
						Level:     "info",
						Component: "test",
						Operation: "op1",
						Message:   "First entry",
					},
					{
						Timestamp: time.Now().UTC().Add(-1 * time.Minute),
						Level:     "info",
						Component: "test",
						Operation: "op2",
						Message:   "Second entry",
					},
					{
						Timestamp: time.Now().UTC(),
						Level:     "info",
						Component: "test",
						Operation: "op3",
						Message:   "Third entry",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid - entries not chronological",
			log: &models.ExecutionLog{
				ID:                  uuid.New().String(),
				WorkflowExecutionID: uuid.New().String(),
				Entries: []models.LogEntry{
					{
						Timestamp: time.Now().UTC(),
						Level:     "info",
						Component: "test",
						Operation: "op1",
						Message:   "First entry",
					},
					{
						Timestamp: time.Now().UTC().Add(-1 * time.Minute),
						Level:     "info",
						Component: "test",
						Operation: "op2",
						Message:   "Out of order",
					},
				},
			},
			wantErr: true,
			errMsg:  "entries must be chronologically ordered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.log.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestExecutionLog_AddEntry(t *testing.T) {
	log := &models.ExecutionLog{
		ID:                  uuid.New().String(),
		WorkflowExecutionID: uuid.New().String(),
		Entries:             []models.LogEntry{},
	}

	// Add first entry
	entry1 := models.LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     "info",
		Component: "test",
		Operation: "op1",
		Message:   "First",
	}
	err := log.AddEntry(entry1)
	require.NoError(t, err)
	assert.Equal(t, 1, len(log.Entries))

	// Add second entry (later timestamp - should succeed)
	time.Sleep(10 * time.Millisecond)
	entry2 := models.LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     "info",
		Component: "test",
		Operation: "op2",
		Message:   "Second",
	}
	err = log.AddEntry(entry2)
	require.NoError(t, err)
	assert.Equal(t, 2, len(log.Entries))

	// Try to add entry with earlier timestamp (should fail)
	entry3 := models.LogEntry{
		Timestamp: time.Now().UTC().Add(-5 * time.Minute),
		Level:     "info",
		Component: "test",
		Operation: "op3",
		Message:   "Out of order",
	}
	err = log.AddEntry(entry3)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timestamp must be after last entry")
	assert.Equal(t, 2, len(log.Entries)) // Should not add
}

func TestDecisionLog_Validate(t *testing.T) {
	tests := []struct {
		name    string
		log     *models.DecisionLog
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid decision log",
			log: &models.DecisionLog{
				LogEntry: models.LogEntry{
					Timestamp: time.Now().UTC(),
					Level:     "info",
					Component: "planner",
					Operation: "decide",
					Message:   "Decision made",
				},
				Decision:     "Use PostgreSQL",
				Rationale:    "Better support for transactions",
				Alternatives: []string{"MySQL", "MongoDB"},
			},
			wantErr: false,
		},
		{
			name: "invalid - missing rationale",
			log: &models.DecisionLog{
				LogEntry: models.LogEntry{
					Timestamp: time.Now().UTC(),
					Level:     "info",
					Component: "planner",
					Operation: "decide",
					Message:   "Decision made",
				},
				Decision:  "Use PostgreSQL",
				Rationale: "",
			},
			wantErr: true,
			errMsg:  "rationale is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.log.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFileOperationLog_Validate(t *testing.T) {
	tests := []struct {
		name    string
		log     *models.FileOperationLog
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid file operation log",
			log: &models.FileOperationLog{
				LogEntry: models.LogEntry{
					Timestamp: time.Now().UTC(),
					Level:     "info",
					Component: "file_manager",
					Operation: "file_op",
					Message:   "File created",
				},
				OperationType: "create",
				Path:          "internal/models/user.go",
				Checksum:      "abc123",
			},
			wantErr: false,
		},
		{
			name: "invalid - invalid operation type",
			log: &models.FileOperationLog{
				LogEntry: models.LogEntry{
					Timestamp: time.Now().UTC(),
					Level:     "info",
					Component: "file_manager",
					Operation: "file_op",
					Message:   "File operation",
				},
				OperationType: "invalid_op",
				Path:          "file.go",
			},
			wantErr: true,
			errMsg:  "invalid operation type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.log.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
