package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/internal/workflow"
	"github.com/dshills/gocreator/pkg/fsops"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_Execute(t *testing.T) {
	tests := []struct {
		name        string
		workflow    *models.WorkflowDefinition
		wantErr     bool
		checkResult func(t *testing.T, execution *models.WorkflowExecution)
	}{
		{
			name: "simple file write workflow",
			workflow: &models.WorkflowDefinition{
				SchemaVersion: "1.0",
				ID:            "test-workflow-1",
				Name:          "Simple File Write",
				Version:       "1.0",
				Tasks: []models.WorkflowTask{
					{
						ID:   "task1",
						Type: "file_op",
						Inputs: map[string]interface{}{
							"operation": "write",
							"path":      "test.txt",
							"content":   "Hello, World!",
						},
					},
				},
				Config: models.WorkflowConfig{
					MaxParallel: 4,
				},
			},
			wantErr: false,
			checkResult: func(t *testing.T, execution *models.WorkflowExecution) {
				assert.Equal(t, models.WorkflowStatusCompleted, execution.Status)
				assert.Len(t, execution.TaskExecutions, 1)
				assert.Equal(t, "completed", execution.TaskExecutions[0].Status)
			},
		},
		{
			name: "workflow with dependencies",
			workflow: &models.WorkflowDefinition{
				SchemaVersion: "1.0",
				ID:            "test-workflow-2",
				Name:          "Workflow with Dependencies",
				Version:       "1.0",
				Tasks: []models.WorkflowTask{
					{
						ID:   "task1",
						Type: "file_op",
						Inputs: map[string]interface{}{
							"operation": "write",
							"path":      "file1.txt",
							"content":   "First file",
						},
					},
					{
						ID:           "task2",
						Type:         "file_op",
						Dependencies: []string{"task1"},
						Inputs: map[string]interface{}{
							"operation": "write",
							"path":      "file2.txt",
							"content":   "Second file",
						},
					},
				},
				Config: models.WorkflowConfig{
					MaxParallel: 4,
				},
			},
			wantErr: false,
			checkResult: func(t *testing.T, execution *models.WorkflowExecution) {
				assert.Equal(t, models.WorkflowStatusCompleted, execution.Status)
				assert.Len(t, execution.TaskExecutions, 2)
			},
		},
		{
			name: "workflow with invalid task type",
			workflow: &models.WorkflowDefinition{
				SchemaVersion: "1.0",
				ID:            "test-workflow-3",
				Name:          "Invalid Task Type",
				Version:       "1.0",
				Tasks: []models.WorkflowTask{
					{
						ID:   "task1",
						Type: "invalid_type",
						Inputs: map[string]interface{}{
							"operation": "test",
						},
					},
				},
				Config: models.WorkflowConfig{
					MaxParallel: 4,
				},
			},
			wantErr:     true,
			checkResult: nil, // No execution created when validation fails
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for test
			tmpDir := t.TempDir()

			// Create fsops
			fsOps, err := fsops.New(fsops.Config{
				RootDir: tmpDir,
			})
			require.NoError(t, err)

			// Create task registry
			logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
			taskRegistry := workflow.NewTaskRegistry(fsOps, logger)

			// Create engine
			checkpointDir := filepath.Join(tmpDir, ".checkpoints")
			engine, err := workflow.NewEngine(workflow.Config{
				FSops:            fsOps,
				TaskRegistry:     taskRegistry,
				CheckpointDir:    checkpointDir,
				CheckpointEveryN: 2,
				MaxParallel:      4,
				Logger:           logger,
			})
			require.NoError(t, err)

			// Execute workflow
			execution, err := engine.Execute(context.Background(), tt.workflow)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.checkResult != nil && execution != nil {
				tt.checkResult(t, execution)
			}
		})
	}
}

func TestEngine_SaveAndLoadCheckpoint(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create fsops
	fsOps, err := fsops.New(fsops.Config{
		RootDir: tmpDir,
	})
	require.NoError(t, err)

	// Create task registry
	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	taskRegistry := workflow.NewTaskRegistry(fsOps, logger)

	// Create engine
	checkpointDir := filepath.Join(tmpDir, ".checkpoints")
	engine, err := workflow.NewEngine(workflow.Config{
		FSops:            fsOps,
		TaskRegistry:     taskRegistry,
		CheckpointDir:    checkpointDir,
		CheckpointEveryN: 1,
		MaxParallel:      4,
		Logger:           logger,
	})
	require.NoError(t, err)

	// Create execution
	execution := &models.WorkflowExecution{
		ID:        "test-execution-1",
		Status:    models.WorkflowStatusRunning,
		StartedAt: time.Now(),
	}

	// Save checkpoint
	state := map[string]interface{}{
		"test_key": "test_value",
		"counter":  42,
	}
	err = engine.SaveCheckpoint(context.Background(), execution, "task1", state)
	require.NoError(t, err)

	// Load checkpoint
	checkpoint, err := engine.LoadCheckpoint(context.Background(), execution.ID)
	require.NoError(t, err)

	assert.Equal(t, "task1", checkpoint.TaskID)
	assert.Equal(t, "test_value", checkpoint.State["test_key"])
	assert.Equal(t, float64(42), checkpoint.State["counter"]) // JSON unmarshals numbers as float64
	assert.True(t, checkpoint.Recoverable)
}

func TestEngine_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  workflow.Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing fsops",
			config: workflow.Config{
				TaskRegistry: workflow.NewTaskRegistry(nil, zerolog.Logger{}),
			},
			wantErr: true,
			errMsg:  "fsops is required",
		},
		{
			name: "missing task registry",
			config: workflow.Config{
				FSops: nil,
			},
			wantErr: true,
			errMsg:  "fsops is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := workflow.NewEngine(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEngine_ParallelExecution(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create fsops
	fsOps, err := fsops.New(fsops.Config{
		RootDir: tmpDir,
	})
	require.NoError(t, err)

	// Create task registry
	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	taskRegistry := workflow.NewTaskRegistry(fsOps, logger)

	// Create engine
	engine, err := workflow.NewEngine(workflow.Config{
		FSops:            fsOps,
		TaskRegistry:     taskRegistry,
		CheckpointDir:    filepath.Join(tmpDir, ".checkpoints"),
		CheckpointEveryN: 5,
		MaxParallel:      4,
		Logger:           logger,
	})
	require.NoError(t, err)

	// Create workflow with multiple independent tasks
	workflow := &models.WorkflowDefinition{
		SchemaVersion: "1.0",
		ID:            "parallel-workflow",
		Name:          "Parallel Execution Test",
		Version:       "1.0",
		Tasks: []models.WorkflowTask{
			{
				ID:   "task1",
				Type: "file_op",
				Inputs: map[string]interface{}{
					"operation": "write",
					"path":      "file1.txt",
					"content":   "File 1",
				},
			},
			{
				ID:   "task2",
				Type: "file_op",
				Inputs: map[string]interface{}{
					"operation": "write",
					"path":      "file2.txt",
					"content":   "File 2",
				},
			},
			{
				ID:   "task3",
				Type: "file_op",
				Inputs: map[string]interface{}{
					"operation": "write",
					"path":      "file3.txt",
					"content":   "File 3",
				},
			},
			{
				ID:   "task4",
				Type: "file_op",
				Inputs: map[string]interface{}{
					"operation": "write",
					"path":      "file4.txt",
					"content":   "File 4",
				},
			},
		},
		Config: models.WorkflowConfig{
			MaxParallel: 4,
		},
	}

	// Execute workflow
	execution, err := engine.Execute(context.Background(), workflow)
	require.NoError(t, err)

	// Verify all tasks completed
	assert.Equal(t, models.WorkflowStatusCompleted, execution.Status)
	assert.Len(t, execution.TaskExecutions, 4)

	// Verify all files were created
	for i := 1; i <= 4; i++ {
		path := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		exists, err := fsOps.Exists(context.Background(), "file"+string(rune('0'+i))+".txt")
		require.NoError(t, err)
		assert.True(t, exists, "File %s should exist", path)
	}
}
