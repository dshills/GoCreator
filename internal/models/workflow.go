package models

import (
	"fmt"
	"strings"
	"time"
)

// WorkflowStatus represents the status of a workflow execution
type WorkflowStatus string

const (
	WorkflowStatusPending   WorkflowStatus = "pending"
	WorkflowStatusRunning   WorkflowStatus = "running"
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusFailed    WorkflowStatus = "failed"
)

// WorkflowTask represents a single task in a workflow
type WorkflowTask struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name,omitempty"`
	Type         string                 `json:"type"`
	Inputs       map[string]interface{} `json:"inputs,omitempty"`
	Outputs      []string               `json:"outputs,omitempty"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Timeout      time.Duration          `json:"timeout,omitempty"`
}

// WorkflowConfig represents configuration for a workflow
type WorkflowConfig struct {
	MaxParallel     int           `json:"max_parallel,omitempty"`
	Retries         int           `json:"retries,omitempty"`
	Timeout         time.Duration `json:"timeout,omitempty"`
	AllowedCommands []string      `json:"allowed_commands,omitempty"`
}

// WorkflowDefinition represents a complete workflow definition
type WorkflowDefinition struct {
	SchemaVersion string         `json:"schema_version"`
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Version       string         `json:"version"`
	Tasks         []WorkflowTask `json:"tasks"`
	Config        WorkflowConfig `json:"config"`
}

// Validate validates the workflow definition
func (w *WorkflowDefinition) Validate() error {
	// Check for cyclic dependencies
	if w.HasCyclicDependencies() {
		return fmt.Errorf("cyclic dependency detected in workflow tasks")
	}

	// Validate task types
	validTypes := map[string]bool{
		"langgraph": true,
		"shell_cmd": true,
		"file_op":   true,
	}

	for _, task := range w.Tasks {
		if !validTypes[task.Type] {
			return fmt.Errorf("invalid task type: %s", task.Type)
		}

		// Validate shell commands if task is shell_cmd
		if task.Type == "shell_cmd" {
			if cmd, ok := task.Inputs["cmd"].(string); ok {
				if !w.isCommandAllowed(cmd) {
					return fmt.Errorf("command not allowed: %s", cmd)
				}
			}
		}
	}

	return nil
}

// HasCyclicDependencies detects cyclic dependencies in the task graph
func (w *WorkflowDefinition) HasCyclicDependencies() bool {
	// Build adjacency list
	graph := make(map[string][]string)
	for _, task := range w.Tasks {
		graph[task.ID] = task.Dependencies
	}

	// Track visited nodes and recursion stack
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	// DFS to detect cycles
	var hasCycle func(node string) bool
	hasCycle = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, dep := range graph[node] {
			// Self-cycle
			if dep == node {
				return true
			}
			// If not visited, recurse
			if !visited[dep] {
				if hasCycle(dep) {
					return true
				}
			} else if recStack[dep] {
				// Back edge found (cycle)
				return true
			}
		}

		recStack[node] = false
		return false
	}

	// Check all nodes
	for task := range graph {
		if !visited[task] {
			if hasCycle(task) {
				return true
			}
		}
	}

	return false
}

// isCommandAllowed checks if a command is in the allowed commands list
func (w *WorkflowDefinition) isCommandAllowed(cmd string) bool {
	// Extract the base command (first word)
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}
	baseCmd := parts[0]

	for _, allowed := range w.Config.AllowedCommands {
		if allowed == baseCmd {
			return true
		}
	}

	return false
}

// TaskExecution represents the execution of a single task
type TaskExecution struct {
	TaskID      string      `json:"task_id"`
	Status      string      `json:"status"`
	StartedAt   time.Time   `json:"started_at"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	Result      interface{} `json:"result,omitempty"`
	Error       *string     `json:"error,omitempty"`
}

// Checkpoint represents a recoverable checkpoint during workflow execution
type Checkpoint struct {
	ID          string                 `json:"id"`
	TaskID      string                 `json:"task_id"`
	State       map[string]interface{} `json:"state"`
	CreatedAt   time.Time              `json:"created_at"`
	Recoverable bool                   `json:"recoverable"`
}

// WorkflowExecution represents the execution of a workflow
type WorkflowExecution struct {
	SchemaVersion  string          `json:"schema_version"`
	ID             string          `json:"id"`
	WorkflowID     string          `json:"workflow_id"`
	Status         WorkflowStatus  `json:"status"`
	StartedAt      time.Time       `json:"started_at"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
	TaskExecutions []TaskExecution `json:"task_executions"`
	Checkpoints    []Checkpoint    `json:"checkpoints,omitempty"`
}

// TransitionTo attempts to transition the workflow execution to a new status
func (w *WorkflowExecution) TransitionTo(newStatus WorkflowStatus) error {
	// Define valid status transitions
	validTransitions := map[WorkflowStatus][]WorkflowStatus{
		WorkflowStatusPending:   {WorkflowStatusRunning},
		WorkflowStatusRunning:   {WorkflowStatusCompleted, WorkflowStatusFailed},
		WorkflowStatusCompleted: {}, // Terminal state
		WorkflowStatusFailed:    {}, // Terminal state
	}

	allowed, ok := validTransitions[w.Status]
	if !ok {
		return fmt.Errorf("unknown status: %s", w.Status)
	}

	// Check if transition is allowed
	for _, allowedStatus := range allowed {
		if allowedStatus == newStatus {
			w.Status = newStatus
			return nil
		}
	}

	return fmt.Errorf("invalid status transition from %s to %s", w.Status, newStatus)
}

// Validate validates the workflow execution
func (w *WorkflowExecution) Validate() error {
	// Check CompletedAt based on status
	switch w.Status {
	case WorkflowStatusPending, WorkflowStatusRunning:
		if w.CompletedAt != nil {
			return fmt.Errorf("CompletedAt must be nil for status %s", w.Status)
		}
	case WorkflowStatusCompleted, WorkflowStatusFailed:
		if w.CompletedAt == nil {
			return fmt.Errorf("CompletedAt must be set for status %s", w.Status)
		}
	}

	return nil
}
