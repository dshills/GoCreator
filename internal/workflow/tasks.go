package workflow

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/pkg/fsops"
	"github.com/rs/zerolog"
)

// Task defines the interface for executable workflow tasks
type Task interface {
	// Name returns the task name
	Name() string

	// Execute runs the task with given inputs and returns output
	Execute(ctx context.Context, inputs map[string]interface{}) (interface{}, error)
}

// ExecutionContext holds shared state for workflow execution
type ExecutionContext struct {
	WorkflowID  string
	ExecutionID string
	State       map[string]interface{}
	Config      models.WorkflowConfig
	mu          sync.RWMutex
}

// Get retrieves a value from the shared state
func (e *ExecutionContext) Get(key string) (interface{}, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	val, ok := e.State[key]
	return val, ok
}

// Set stores a value in the shared state
func (e *ExecutionContext) Set(key string, value interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.State[key] = value
}

// TaskRegistry manages registered task implementations
type TaskRegistry struct {
	tasks  map[string]Task
	fsops  fsops.FileOps
	logger zerolog.Logger
	mu     sync.RWMutex
}

// NewTaskRegistry creates a new task registry with default tasks
func NewTaskRegistry(fsops fsops.FileOps, logger zerolog.Logger) *TaskRegistry {
	registry := &TaskRegistry{
		tasks:  make(map[string]Task),
		fsops:  fsops,
		logger: logger,
	}

	// Register default task types
	registry.Register("file_op", NewFileOpTask(fsops, logger))
	registry.Register("patch", NewPatchTask(fsops, logger))
	registry.Register("shell_cmd", NewShellTask(logger))

	return registry
}

// Register registers a task implementation
func (r *TaskRegistry) Register(taskType string, task Task) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks[taskType] = task
}

// Get retrieves a task implementation by type
func (r *TaskRegistry) Get(taskType string) (Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.tasks[taskType]
	if !exists {
		return nil, fmt.Errorf("task type not registered: %s", taskType)
	}

	return task, nil
}

// FileOpTask handles file operations (create, read, delete)
type FileOpTask struct {
	fsops  fsops.FileOps
	logger zerolog.Logger
}

// NewFileOpTask creates a new file operation task
func NewFileOpTask(fsops fsops.FileOps, logger zerolog.Logger) *FileOpTask {
	return &FileOpTask{
		fsops:  fsops,
		logger: logger,
	}
}

// Name returns the task name
func (t *FileOpTask) Name() string {
	return "file_op"
}

// Execute runs the file operation task
func (t *FileOpTask) Execute(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
	// Extract operation type
	operation, ok := inputs["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation not specified or not a string")
	}

	// Extract path
	path, ok := inputs["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path not specified or not a string")
	}

	t.logger.Debug().
		Str("operation", operation).
		Str("path", path).
		Msg("Executing file operation")

	switch operation {
	case "write", "create":
		content, ok := inputs["content"].(string)
		if !ok {
			return nil, fmt.Errorf("content not specified or not a string")
		}
		if err := t.fsops.WriteFile(ctx, path, content); err != nil {
			return nil, fmt.Errorf("failed to write file: %w", err)
		}
		return map[string]interface{}{
			"operation": "write",
			"path":      path,
			"checksum":  t.fsops.GenerateChecksum(content),
		}, nil

	case "read":
		content, err := t.fsops.ReadFile(ctx, path)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
		return map[string]interface{}{
			"operation": "read",
			"path":      path,
			"content":   content,
		}, nil

	case "delete":
		if err := t.fsops.DeleteFile(ctx, path); err != nil {
			return nil, fmt.Errorf("failed to delete file: %w", err)
		}
		return map[string]interface{}{
			"operation": "delete",
			"path":      path,
		}, nil

	default:
		return nil, fmt.Errorf("unknown file operation: %s", operation)
	}
}

// PatchTask handles applying patches to files
type PatchTask struct {
	fsops  fsops.FileOps
	logger zerolog.Logger
}

// NewPatchTask creates a new patch task
func NewPatchTask(fsops fsops.FileOps, logger zerolog.Logger) *PatchTask {
	return &PatchTask{
		fsops:  fsops,
		logger: logger,
	}
}

// Name returns the task name
func (t *PatchTask) Name() string {
	return "patch"
}

// Execute runs the patch task
func (t *PatchTask) Execute(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
	// Extract patch data
	targetFile, ok := inputs["target_file"].(string)
	if !ok {
		return nil, fmt.Errorf("target_file not specified or not a string")
	}

	diff, ok := inputs["diff"].(string)
	if !ok {
		return nil, fmt.Errorf("diff not specified or not a string")
	}

	reversible := true
	if rev, ok := inputs["reversible"].(bool); ok {
		reversible = rev
	}

	patch := models.Patch{
		TargetFile: targetFile,
		Diff:       diff,
		AppliedAt:  time.Now(),
		Reversible: reversible,
	}

	t.logger.Debug().
		Str("target_file", targetFile).
		Bool("reversible", reversible).
		Msg("Applying patch")

	// Apply the patch
	if err := t.fsops.ApplyPatch(ctx, patch); err != nil {
		return nil, fmt.Errorf("failed to apply patch: %w", err)
	}

	// Get patch statistics
	added, removed, modified, err := t.fsops.GetPatchStats(patch)
	if err != nil {
		t.logger.Warn().Err(err).Msg("Failed to get patch stats")
	}

	return map[string]interface{}{
		"operation":      "patch",
		"target_file":    targetFile,
		"lines_added":    added,
		"lines_removed":  removed,
		"lines_modified": modified,
	}, nil
}

// ShellTask handles executing whitelisted shell commands
type ShellTask struct {
	logger zerolog.Logger
}

// NewShellTask creates a new shell command task
func NewShellTask(logger zerolog.Logger) *ShellTask {
	return &ShellTask{
		logger: logger,
	}
}

// Name returns the task name
func (t *ShellTask) Name() string {
	return "shell_cmd"
}

// Execute runs the shell command task
func (t *ShellTask) Execute(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
	// Extract command
	cmd, ok := inputs["cmd"].(string)
	if !ok {
		return nil, fmt.Errorf("cmd not specified or not a string")
	}

	// Extract allowed commands list from context
	allowedCommands, ok := inputs["allowed_commands"].([]string)
	if !ok {
		// Try as []interface{} and convert
		if allowedIntf, ok := inputs["allowed_commands"].([]interface{}); ok {
			allowedCommands = make([]string, len(allowedIntf))
			for i, v := range allowedIntf {
				if s, ok := v.(string); ok {
					allowedCommands[i] = s
				}
			}
		} else {
			return nil, fmt.Errorf("allowed_commands not specified or not a string array")
		}
	}

	// Validate command is allowed
	if !isCommandAllowed(cmd, allowedCommands) {
		return nil, fmt.Errorf("command not in whitelist: %s", cmd)
	}

	// Extract optional arguments
	args := []string{}
	if argsIntf, ok := inputs["args"].([]interface{}); ok {
		args = make([]string, len(argsIntf))
		for i, v := range argsIntf {
			if s, ok := v.(string); ok {
				args[i] = s
			}
		}
	} else if argsStr, ok := inputs["args"].([]string); ok {
		args = argsStr
	}

	// Extract timeout (default: 30 seconds)
	timeout := 30 * time.Second
	if timeoutSec, ok := inputs["timeout"].(float64); ok {
		timeout = time.Duration(timeoutSec) * time.Second
	} else if timeoutSec, ok := inputs["timeout"].(int); ok {
		timeout = time.Duration(timeoutSec) * time.Second
	}

	t.logger.Debug().
		Str("cmd", cmd).
		Strs("args", args).
		Dur("timeout", timeout).
		Msg("Executing shell command")

	// Create command with timeout context
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute command
	startTime := time.Now()
	//nolint:gosec // G204: Subprocess execution required for workflow task execution
	command := exec.CommandContext(cmdCtx, cmd, args...)
	output, err := command.CombinedOutput()
	duration := time.Since(startTime)

	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("failed to execute command: %w", err)
		}
	}

	result := map[string]interface{}{
		"operation": "shell_cmd",
		"cmd":       cmd,
		"args":      args,
		"exit_code": exitCode,
		"output":    string(output),
		"duration":  duration.Seconds(),
	}

	if exitCode != 0 {
		return result, fmt.Errorf("command exited with code %d: %s", exitCode, string(output))
	}

	t.logger.Debug().
		Str("cmd", cmd).
		Int("exit_code", exitCode).
		Dur("duration", duration).
		Msg("Shell command completed")

	return result, nil
}

// isCommandAllowed checks if a command is in the whitelist
func isCommandAllowed(cmd string, allowedCommands []string) bool {
	// Extract base command (first word)
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}
	baseCmd := parts[0]

	for _, allowed := range allowedCommands {
		if allowed == baseCmd {
			return true
		}
	}

	return false
}

// LangGraphTask handles executing LangGraph-Go nodes
// This task allows executing arbitrary LangGraph graphs as workflow tasks.
// The graph must be provided via inputs along with the node to execute and initial state.
type LangGraphTask struct {
	logger zerolog.Logger
}

// NewLangGraphTask creates a new LangGraph task
func NewLangGraphTask(logger zerolog.Logger) *LangGraphTask {
	return &LangGraphTask{
		logger: logger,
	}
}

// Name returns the task name
func (t *LangGraphTask) Name() string {
	return "langgraph"
}

// Execute runs the LangGraph task
// Expected inputs:
//   - "node" (string): Name of the node to execute
//   - "state" (map[string]interface{}): Initial state for the graph
//   - "graph_func" (func): Optional function that returns a graph to execute
//
// For embedded LangGraph execution (clarify/generate), use those specific engines directly.
// This task is primarily for custom workflow-defined graphs.
func (t *LangGraphTask) Execute(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
	// Extract node name
	node, ok := inputs["node"].(string)
	if !ok {
		return nil, fmt.Errorf("node not specified or not a string")
	}

	t.logger.Debug().
		Str("node", node).
		Msg("Executing LangGraph node")

	// Extract state
	state, ok := inputs["state"].(map[string]interface{})
	if !ok {
		// Default to empty state if not provided
		state = make(map[string]interface{})
	}

	// Check for graph function
	graphFunc, hasGraphFunc := inputs["graph_func"]

	if !hasGraphFunc {
		// No graph provided - this is expected for the current implementation
		// The clarification and generation engines use their own embedded graphs
		// This task would be used for custom user-defined workflow graphs
		t.logger.Info().
			Str("node", node).
			Msg("LangGraph task executed with basic state passthrough (no graph function provided)")

		// Return the state with execution metadata
		return map[string]interface{}{
			"operation": "langgraph",
			"node":      node,
			"status":    "executed",
			"state":     state,
			"note":      "Executed as passthrough task - provide 'graph_func' for custom graph execution",
			"timestamp": time.Now(),
		}, nil
	}

	// If a graph function is provided, execute it
	// This allows for custom graph definitions in workflows
	t.logger.Info().
		Str("node", node).
		Msg("Executing custom graph function")

	// The graph function should have signature: func(context.Context, map[string]interface{}) (interface{}, error)
	execFunc, ok := graphFunc.(func(context.Context, map[string]interface{}) (interface{}, error))
	if !ok {
		return nil, fmt.Errorf("graph_func must have signature: func(context.Context, map[string]interface{}) (interface{}, error)")
	}

	// Execute the custom graph function
	result, err := execFunc(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("graph execution failed: %w", err)
	}

	t.logger.Info().
		Str("node", node).
		Msg("Custom graph execution completed successfully")

	return map[string]interface{}{
		"operation": "langgraph",
		"node":      node,
		"status":    "completed",
		"result":    result,
		"timestamp": time.Now(),
	}, nil
}
