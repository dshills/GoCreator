package models

import (
	"fmt"
	"time"
)

// LogEntry represents a single log entry in the execution log
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Component string                 `json:"component"`
	Operation string                 `json:"operation"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Message   string                 `json:"message"`
	Error     *string                `json:"error,omitempty"`
}

// DecisionLog represents a log of a decision made during execution
type DecisionLog struct {
	LogEntry     LogEntry `json:"log_entry"`
	Decision     string   `json:"decision"`
	Rationale    string   `json:"rationale"`
	Alternatives []string `json:"alternatives,omitempty"`
}

// Validate validates the decision log
func (d *DecisionLog) Validate() error {
	if d.Decision == "" {
		return fmt.Errorf("decision cannot be empty")
	}
	if d.Rationale == "" {
		return fmt.Errorf("rationale is required")
	}
	return nil
}

// FileOperationLog represents a log of a file operation
type FileOperationLog struct {
	LogEntry      LogEntry `json:"log_entry"`
	OperationType string   `json:"operation_type"`
	Path          string   `json:"path"`
	Checksum      string   `json:"checksum,omitempty"`
}

// Validate validates the file operation log
func (f *FileOperationLog) Validate() error {
	if f.OperationType == "" {
		return fmt.Errorf("operation type cannot be empty")
	}
	if f.Path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	validOps := map[string]bool{"create": true, "update": true, "delete": true, "patch": true, "read": true}
	if !validOps[f.OperationType] {
		return fmt.Errorf("invalid operation type: %s", f.OperationType)
	}
	return nil
}

// CommandLog represents a log of a command execution
type CommandLog struct {
	LogEntry LogEntry      `json:"log_entry"`
	Command  string        `json:"command"`
	Args     []string      `json:"args,omitempty"`
	ExitCode int           `json:"exit_code"`
	Stdout   string        `json:"stdout,omitempty"`
	Stderr   string        `json:"stderr,omitempty"`
	Duration time.Duration `json:"duration"`
}

// ExecutionLog represents the complete execution log
type ExecutionLog struct {
	SchemaVersion       string     `json:"schema_version"`
	ID                  string     `json:"id"`
	WorkflowExecutionID string     `json:"workflow_execution_id"`
	Entries             []LogEntry `json:"entries"`
}

// Validate validates the execution log
func (e *ExecutionLog) Validate() error {
	// Check that entries are in chronological order
	for i := 1; i < len(e.Entries); i++ {
		if e.Entries[i].Timestamp.Before(e.Entries[i-1].Timestamp) {
			return fmt.Errorf("entries must be chronologically ordered")
		}
	}

	return nil
}

// AddEntry adds a new entry to the execution log
func (e *ExecutionLog) AddEntry(entry LogEntry) error {
	// Check chronological order
	if len(e.Entries) > 0 {
		lastEntry := e.Entries[len(e.Entries)-1]
		if entry.Timestamp.Before(lastEntry.Timestamp) || entry.Timestamp.Equal(lastEntry.Timestamp) {
			return fmt.Errorf("timestamp must be after last entry")
		}
	}

	e.Entries = append(e.Entries, entry)
	return nil
}
