// Package workflow provides deterministic workflow execution with checkpointing.
package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/pkg/fsops"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Engine defines the interface for workflow execution
type Engine interface {
	// Execute runs a workflow from start to completion
	Execute(ctx context.Context, workflow *models.WorkflowDefinition) (*models.WorkflowExecution, error)

	// Resume continues execution from the last checkpoint
	Resume(ctx context.Context, executionID string) (*models.WorkflowExecution, error)

	// GetExecution retrieves the current state of a workflow execution
	GetExecution(ctx context.Context, executionID string) (*models.WorkflowExecution, error)

	// SaveCheckpoint creates a checkpoint at the current execution state
	SaveCheckpoint(ctx context.Context, execution *models.WorkflowExecution, taskID string, state map[string]interface{}) error

	// LoadCheckpoint loads the most recent checkpoint for an execution
	LoadCheckpoint(ctx context.Context, executionID string) (*models.Checkpoint, error)
}

// engine implements the Engine interface
type engine struct {
	fsops            fsops.FileOps
	taskRegistry     *TaskRegistry
	executor         *ParallelExecutor
	checkpointDir    string
	checkpointEveryN int
	logger           zerolog.Logger
	mu               sync.RWMutex
	executions       map[string]*models.WorkflowExecution
}

// Config holds configuration for the workflow engine
type Config struct {
	FSops            fsops.FileOps
	TaskRegistry     *TaskRegistry
	CheckpointDir    string
	CheckpointEveryN int // Create checkpoint after every N tasks (0 = no checkpointing)
	MaxParallel      int // Maximum parallel task execution (default: 4)
	Logger           zerolog.Logger
}

// NewEngine creates a new workflow engine instance
func NewEngine(cfg Config) (Engine, error) {
	if cfg.FSops == nil {
		return nil, fmt.Errorf("fsops is required")
	}

	if cfg.TaskRegistry == nil {
		return nil, fmt.Errorf("task registry is required")
	}

	if cfg.CheckpointDir == "" {
		cfg.CheckpointDir = ".gocreator/checkpoints"
	}

	if cfg.CheckpointEveryN <= 0 {
		cfg.CheckpointEveryN = 5 // Default: checkpoint every 5 tasks
	}

	maxParallel := cfg.MaxParallel
	if maxParallel <= 0 {
		maxParallel = 4 // Default: 4-way parallelism
	}

	logger := cfg.Logger
	if logger.GetLevel() == zerolog.Disabled {
		logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	}

	// Create parallel executor
	executor := NewParallelExecutor(ParallelExecutorConfig{
		MaxWorkers:   maxParallel,
		TaskRegistry: cfg.TaskRegistry,
		Logger:       logger,
	})

	eng := &engine{
		fsops:            cfg.FSops,
		taskRegistry:     cfg.TaskRegistry,
		executor:         executor,
		checkpointDir:    cfg.CheckpointDir,
		checkpointEveryN: cfg.CheckpointEveryN,
		logger:           logger,
		executions:       make(map[string]*models.WorkflowExecution),
	}

	return eng, nil
}

// Execute runs a workflow from start to completion
func (e *engine) Execute(ctx context.Context, workflow *models.WorkflowDefinition) (*models.WorkflowExecution, error) {
	// Validate workflow
	if err := workflow.Validate(); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	// Create execution record
	execution := &models.WorkflowExecution{
		SchemaVersion:  "1.0",
		ID:             uuid.New().String(),
		WorkflowID:     workflow.ID,
		Status:         models.WorkflowStatusPending,
		StartedAt:      time.Now(),
		TaskExecutions: make([]models.TaskExecution, 0),
		Checkpoints:    make([]models.Checkpoint, 0),
	}

	// Store execution
	e.mu.Lock()
	e.executions[execution.ID] = execution
	e.mu.Unlock()

	e.logger.Info().
		Str("execution_id", execution.ID).
		Str("workflow_id", workflow.ID).
		Str("workflow_name", workflow.Name).
		Msg("Starting workflow execution")

	// Transition to running
	if err := execution.TransitionTo(models.WorkflowStatusRunning); err != nil {
		return execution, fmt.Errorf("failed to transition to running: %w", err)
	}

	// Execute workflow tasks
	if err := e.executeTasks(ctx, execution, workflow); err != nil {
		execution.Status = models.WorkflowStatusFailed
		now := time.Now()
		execution.CompletedAt = &now
		e.logger.Error().
			Err(err).
			Str("execution_id", execution.ID).
			Msg("Workflow execution failed")
		return execution, fmt.Errorf("workflow execution failed: %w", err)
	}

	// Mark as completed
	now := time.Now()
	execution.CompletedAt = &now
	if err := execution.TransitionTo(models.WorkflowStatusCompleted); err != nil {
		return execution, fmt.Errorf("failed to transition to completed: %w", err)
	}

	e.logger.Info().
		Str("execution_id", execution.ID).
		Int("tasks_completed", len(execution.TaskExecutions)).
		Dur("duration", time.Since(execution.StartedAt)).
		Msg("Workflow execution completed successfully")

	return execution, nil
}

// executeTasks executes all tasks in the workflow using parallel execution
func (e *engine) executeTasks(ctx context.Context, execution *models.WorkflowExecution, workflow *models.WorkflowDefinition) error {
	// Build execution context with shared state
	execCtx := &ExecutionContext{
		WorkflowID:  workflow.ID,
		ExecutionID: execution.ID,
		State:       make(map[string]interface{}),
		Config:      workflow.Config,
	}

	// Execute tasks with parallel executor
	results, err := e.executor.Execute(ctx, workflow.Tasks, execCtx)
	if err != nil {
		return err
	}

	// Process results and update execution
	tasksCompleted := 0
	for taskID, result := range results {
		taskExec := models.TaskExecution{
			TaskID:      taskID,
			Status:      "completed",
			StartedAt:   result.StartedAt,
			CompletedAt: &result.CompletedAt,
			Result:      result.Output,
		}

		if result.Error != nil {
			taskExec.Status = "failed"
			errMsg := result.Error.Error()
			taskExec.Error = &errMsg
		}

		execution.TaskExecutions = append(execution.TaskExecutions, taskExec)

		// Update shared state with task result
		if result.Error == nil && result.Output != nil {
			execCtx.State[taskID] = result.Output
		}

		tasksCompleted++

		// Create checkpoint if configured
		if e.checkpointEveryN > 0 && tasksCompleted%e.checkpointEveryN == 0 {
			if err := e.SaveCheckpoint(ctx, execution, taskID, execCtx.State); err != nil {
				e.logger.Warn().
					Err(err).
					Str("execution_id", execution.ID).
					Str("task_id", taskID).
					Msg("Failed to save checkpoint")
			}
		}
	}

	// Check if any tasks failed
	for _, taskExec := range execution.TaskExecutions {
		if taskExec.Status == "failed" {
			return fmt.Errorf("task %s failed: %s", taskExec.TaskID, *taskExec.Error)
		}
	}

	return nil
}

// Resume continues execution from the last checkpoint
func (e *engine) Resume(ctx context.Context, executionID string) (*models.WorkflowExecution, error) {
	e.logger.Info().
		Str("execution_id", executionID).
		Msg("Resuming workflow execution from checkpoint")

	// Load checkpoint
	checkpoint, err := e.LoadCheckpoint(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	if !checkpoint.Recoverable {
		return nil, fmt.Errorf("checkpoint is not recoverable")
	}

	// Load execution state
	e.mu.RLock()
	execution, exists := e.executions[executionID]
	e.mu.RUnlock()

	if !exists {
		// Try to load from disk
		var err error
		execution, err = e.loadExecutionFromDisk(ctx, executionID)
		if err != nil {
			return nil, fmt.Errorf("failed to load execution: %w", err)
		}

		e.mu.Lock()
		e.executions[executionID] = execution
		e.mu.Unlock()
	}

	// Load workflow definition
	// In a real implementation, this would load from storage
	// For now, we return an error indicating the workflow definition is needed
	return nil, fmt.Errorf("resume not fully implemented: workflow definition needed")
}

// GetExecution retrieves the current state of a workflow execution
func (e *engine) GetExecution(ctx context.Context, executionID string) (*models.WorkflowExecution, error) {
	e.mu.RLock()
	execution, exists := e.executions[executionID]
	e.mu.RUnlock()

	if !exists {
		return e.loadExecutionFromDisk(ctx, executionID)
	}

	return execution, nil
}

// SaveCheckpoint creates a checkpoint at the current execution state
func (e *engine) SaveCheckpoint(ctx context.Context, execution *models.WorkflowExecution, taskID string, state map[string]interface{}) error {
	checkpoint := models.Checkpoint{
		ID:          uuid.New().String(),
		TaskID:      taskID,
		State:       state,
		CreatedAt:   time.Now(),
		Recoverable: true,
	}

	execution.Checkpoints = append(execution.Checkpoints, checkpoint)

	// Ensure checkpoint directory exists
	checkpointDir := filepath.Join(e.checkpointDir, execution.ID)
	if err := os.MkdirAll(checkpointDir, 0750); err != nil {
		return fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	// Save checkpoint to file
	checkpointPath := filepath.Join(checkpointDir, fmt.Sprintf("checkpoint_%s.json", checkpoint.ID))
	data, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	if err := os.WriteFile(checkpointPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write checkpoint: %w", err)
	}

	e.logger.Info().
		Str("execution_id", execution.ID).
		Str("checkpoint_id", checkpoint.ID).
		Str("task_id", taskID).
		Msg("Checkpoint saved")

	return nil
}

// LoadCheckpoint loads the most recent checkpoint for an execution
func (e *engine) LoadCheckpoint(ctx context.Context, executionID string) (*models.Checkpoint, error) {
	checkpointDir := filepath.Join(e.checkpointDir, executionID)

	// Check if checkpoint directory exists
	if _, err := os.Stat(checkpointDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("no checkpoints found for execution %s", executionID)
	}

	// Read all checkpoint files
	files, err := os.ReadDir(checkpointDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint directory: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no checkpoint files found")
	}

	// Get the most recent checkpoint (last file alphabetically due to UUID ordering)
	var latestCheckpoint *models.Checkpoint
	var latestTime time.Time

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		checkpointPath := filepath.Join(checkpointDir, file.Name())
		//nolint:gosec // G304: Reading checkpoint file - required for workflow recovery
		data, err := os.ReadFile(checkpointPath)
		if err != nil {
			continue
		}

		var checkpoint models.Checkpoint
		if err := json.Unmarshal(data, &checkpoint); err != nil {
			continue
		}

		if latestCheckpoint == nil || checkpoint.CreatedAt.After(latestTime) {
			latestCheckpoint = &checkpoint
			latestTime = checkpoint.CreatedAt
		}
	}

	if latestCheckpoint == nil {
		return nil, fmt.Errorf("failed to load any valid checkpoints")
	}

	return latestCheckpoint, nil
}

// loadExecutionFromDisk loads execution state from checkpoint directory
func (e *engine) loadExecutionFromDisk(ctx context.Context, executionID string) (*models.WorkflowExecution, error) {
	// In a real implementation, this would load the full execution state
	// For now, return an error
	return nil, fmt.Errorf("loading execution from disk not fully implemented")
}
