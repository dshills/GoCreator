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

func TestWorkflowDefinition_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name     string
		workflow *models.WorkflowDefinition
	}{
		{
			name: "complete workflow definition",
			workflow: &models.WorkflowDefinition{
				SchemaVersion: "1.0",
				ID:            uuid.New().String(),
				Name:          "generate",
				Version:       "1.0",
				Tasks: []models.WorkflowTask{
					{
						ID:           "task1",
						Name:         "Parse Spec",
						Type:         "langgraph",
						Inputs:       map[string]interface{}{"spec": "input.yaml"},
						Outputs:      []string{"parsed_spec"},
						Dependencies: []string{},
						Timeout:      30 * time.Second,
					},
					{
						ID:           "task2",
						Name:         "Generate Code",
						Type:         "langgraph",
						Inputs:       map[string]interface{}{"spec": "parsed_spec"},
						Outputs:      []string{"generated_files"},
						Dependencies: []string{"task1"},
						Timeout:      2 * time.Minute,
					},
				},
				Config: models.WorkflowConfig{
					MaxParallel:     4,
					Retries:         3,
					Timeout:         10 * time.Minute,
					AllowedCommands: []string{"go", "gofmt", "golangci-lint"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.workflow)
			require.NoError(t, err)

			var unmarshaled models.WorkflowDefinition
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, tt.workflow.ID, unmarshaled.ID)
			assert.Equal(t, tt.workflow.Name, unmarshaled.Name)
			assert.Equal(t, len(tt.workflow.Tasks), len(unmarshaled.Tasks))
		})
	}
}

func TestWorkflowDefinition_Validate(t *testing.T) {
	tests := []struct {
		name     string
		workflow *models.WorkflowDefinition
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid workflow with DAG",
			workflow: &models.WorkflowDefinition{
				ID:      uuid.New().String(),
				Name:    "test",
				Version: "1.0",
				Tasks: []models.WorkflowTask{
					{ID: "t1", Type: "langgraph", Dependencies: []string{}},
					{ID: "t2", Type: "langgraph", Dependencies: []string{"t1"}},
					{ID: "t3", Type: "langgraph", Dependencies: []string{"t1", "t2"}},
				},
				Config: models.WorkflowConfig{
					AllowedCommands: []string{"go"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid - cyclic dependencies",
			workflow: &models.WorkflowDefinition{
				ID:      uuid.New().String(),
				Name:    "test",
				Version: "1.0",
				Tasks: []models.WorkflowTask{
					{ID: "t1", Type: "langgraph", Dependencies: []string{"t2"}},
					{ID: "t2", Type: "langgraph", Dependencies: []string{"t1"}},
				},
			},
			wantErr: true,
			errMsg:  "cyclic dependency",
		},
		{
			name: "invalid - shell command not in allowed list",
			workflow: &models.WorkflowDefinition{
				ID:      uuid.New().String(),
				Name:    "test",
				Version: "1.0",
				Tasks: []models.WorkflowTask{
					{ID: "t1", Type: "shell_cmd", Inputs: map[string]interface{}{"cmd": "rm"}},
				},
				Config: models.WorkflowConfig{
					AllowedCommands: []string{"go", "gofmt"},
				},
			},
			wantErr: true,
			errMsg:  "command not allowed",
		},
		{
			name: "invalid - unknown task type",
			workflow: &models.WorkflowDefinition{
				ID:      uuid.New().String(),
				Name:    "test",
				Version: "1.0",
				Tasks: []models.WorkflowTask{
					{ID: "t1", Type: "unknown_type"},
				},
			},
			wantErr: true,
			errMsg:  "invalid task type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.workflow.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWorkflowDefinition_DetectCycles(t *testing.T) {
	tests := []struct {
		name      string
		tasks     []models.WorkflowTask
		hasCycles bool
	}{
		{
			name: "no cycles - linear",
			tasks: []models.WorkflowTask{
				{ID: "t1", Dependencies: []string{}},
				{ID: "t2", Dependencies: []string{"t1"}},
				{ID: "t3", Dependencies: []string{"t2"}},
			},
			hasCycles: false,
		},
		{
			name: "no cycles - parallel",
			tasks: []models.WorkflowTask{
				{ID: "t1", Dependencies: []string{}},
				{ID: "t2", Dependencies: []string{}},
				{ID: "t3", Dependencies: []string{"t1", "t2"}},
			},
			hasCycles: false,
		},
		{
			name: "simple cycle",
			tasks: []models.WorkflowTask{
				{ID: "t1", Dependencies: []string{"t2"}},
				{ID: "t2", Dependencies: []string{"t1"}},
			},
			hasCycles: true,
		},
		{
			name: "three-node cycle",
			tasks: []models.WorkflowTask{
				{ID: "t1", Dependencies: []string{"t2"}},
				{ID: "t2", Dependencies: []string{"t3"}},
				{ID: "t3", Dependencies: []string{"t1"}},
			},
			hasCycles: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow := &models.WorkflowDefinition{Tasks: tt.tasks}
			hasCycles := workflow.HasCyclicDependencies()
			assert.Equal(t, tt.hasCycles, hasCycles)
		})
	}
}

func TestWorkflowExecution_JSONMarshaling(t *testing.T) {
	completedAt := time.Now().UTC()
	errorMsg := "task failed"

	tests := []struct {
		name      string
		execution *models.WorkflowExecution
	}{
		{
			name: "complete execution",
			execution: &models.WorkflowExecution{
				SchemaVersion: "1.0",
				ID:            uuid.New().String(),
				WorkflowID:    uuid.New().String(),
				Status:        models.WorkflowStatusCompleted,
				StartedAt:     time.Now().UTC().Add(-5 * time.Minute),
				CompletedAt:   &completedAt,
				TaskExecutions: []models.TaskExecution{
					{
						TaskID:      "t1",
						Status:      "completed",
						StartedAt:   time.Now().UTC().Add(-5 * time.Minute),
						CompletedAt: &completedAt,
						Result:      map[string]interface{}{"output": "success"},
						Error:       nil,
					},
				},
				Checkpoints: []models.Checkpoint{
					{
						ID:          uuid.New().String(),
						TaskID:      "t1",
						State:       map[string]interface{}{"step": 1},
						CreatedAt:   time.Now().UTC(),
						Recoverable: true,
					},
				},
			},
		},
		{
			name: "failed execution",
			execution: &models.WorkflowExecution{
				SchemaVersion: "1.0",
				ID:            uuid.New().String(),
				WorkflowID:    uuid.New().String(),
				Status:        models.WorkflowStatusFailed,
				StartedAt:     time.Now().UTC().Add(-2 * time.Minute),
				CompletedAt:   &completedAt,
				TaskExecutions: []models.TaskExecution{
					{
						TaskID:      "t1",
						Status:      "failed",
						StartedAt:   time.Now().UTC().Add(-2 * time.Minute),
						CompletedAt: &completedAt,
						Error:       &errorMsg,
					},
				},
				Checkpoints: []models.Checkpoint{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.execution)
			require.NoError(t, err)

			var unmarshaled models.WorkflowExecution
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, tt.execution.ID, unmarshaled.ID)
			assert.Equal(t, tt.execution.WorkflowID, unmarshaled.WorkflowID)
			assert.Equal(t, tt.execution.Status, unmarshaled.Status)
			assert.Equal(t, len(tt.execution.TaskExecutions), len(unmarshaled.TaskExecutions))
		})
	}
}

func TestWorkflowExecution_StateTransitions(t *testing.T) {
	tests := []struct {
		name          string
		fromState     models.WorkflowStatus
		toState       models.WorkflowStatus
		shouldSucceed bool
	}{
		{
			name:          "pending to running",
			fromState:     models.WorkflowStatusPending,
			toState:       models.WorkflowStatusRunning,
			shouldSucceed: true,
		},
		{
			name:          "running to completed",
			fromState:     models.WorkflowStatusRunning,
			toState:       models.WorkflowStatusCompleted,
			shouldSucceed: true,
		},
		{
			name:          "running to failed",
			fromState:     models.WorkflowStatusRunning,
			toState:       models.WorkflowStatusFailed,
			shouldSucceed: true,
		},
		{
			name:          "invalid - pending to completed",
			fromState:     models.WorkflowStatusPending,
			toState:       models.WorkflowStatusCompleted,
			shouldSucceed: false,
		},
		{
			name:          "invalid - completed to running (terminal)",
			fromState:     models.WorkflowStatusCompleted,
			toState:       models.WorkflowStatusRunning,
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execution := &models.WorkflowExecution{
				ID:         uuid.New().String(),
				WorkflowID: uuid.New().String(),
				Status:     tt.fromState,
			}

			err := execution.TransitionTo(tt.toState)
			if tt.shouldSucceed {
				require.NoError(t, err)
				assert.Equal(t, tt.toState, execution.Status)
			} else {
				require.Error(t, err)
				assert.Equal(t, tt.fromState, execution.Status)
			}
		})
	}
}

func TestWorkflowExecution_Validate(t *testing.T) {
	completedAt := time.Now().UTC()

	tests := []struct {
		name      string
		execution *models.WorkflowExecution
		wantErr   bool
		errMsg    string
	}{
		{
			name: "valid completed execution",
			execution: &models.WorkflowExecution{
				ID:          uuid.New().String(),
				WorkflowID:  uuid.New().String(),
				Status:      models.WorkflowStatusCompleted,
				StartedAt:   time.Now().UTC().Add(-5 * time.Minute),
				CompletedAt: &completedAt,
			},
			wantErr: false,
		},
		{
			name: "valid running execution",
			execution: &models.WorkflowExecution{
				ID:          uuid.New().String(),
				WorkflowID:  uuid.New().String(),
				Status:      models.WorkflowStatusRunning,
				StartedAt:   time.Now().UTC(),
				CompletedAt: nil,
			},
			wantErr: false,
		},
		{
			name: "invalid - running with CompletedAt set",
			execution: &models.WorkflowExecution{
				ID:          uuid.New().String(),
				WorkflowID:  uuid.New().String(),
				Status:      models.WorkflowStatusRunning,
				StartedAt:   time.Now().UTC(),
				CompletedAt: &completedAt,
			},
			wantErr: true,
			errMsg:  "CompletedAt must be nil",
		},
		{
			name: "invalid - completed without CompletedAt",
			execution: &models.WorkflowExecution{
				ID:          uuid.New().String(),
				WorkflowID:  uuid.New().String(),
				Status:      models.WorkflowStatusCompleted,
				StartedAt:   time.Now().UTC(),
				CompletedAt: nil,
			},
			wantErr: true,
			errMsg:  "CompletedAt must be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.execution.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
