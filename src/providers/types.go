package providers

import (
	"fmt"
	"time"
)

// ProviderType identifies the LLM provider implementation
type ProviderType string

// Provider type constants
const (
	ProviderTypeOpenAI    ProviderType = "openai"
	ProviderTypeAnthropic ProviderType = "anthropic"
	ProviderTypeGoogle    ProviderType = "google"
)

// Role identifies specialized task roles in the system
type Role string

// Role constants
const (
	RoleCoder     Role = "coder"
	RoleReviewer  Role = "reviewer"
	RolePlanner   Role = "planner"
	RoleClarifier Role = "clarifier"
)

// TaskStatus represents the execution state of a task
type TaskStatus string

// Task status constants
const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

// MetricStatus represents the outcome of a provider request
type MetricStatus string

// Metric status constants
const (
	MetricStatusSuccess MetricStatus = "success"
	MetricStatusFailure MetricStatus = "failure"
	MetricStatusRetry   MetricStatus = "retry"
)

// ErrorCode identifies categories of provider errors
type ErrorCode string

// Error code constants
const (
	ErrorCodeAuth         ErrorCode = "AUTH_FAILED"
	ErrorCodeRateLimit    ErrorCode = "RATE_LIMIT"
	ErrorCodeNetwork      ErrorCode = "NETWORK_ERROR"
	ErrorCodeTimeout      ErrorCode = "TIMEOUT"
	ErrorCodeInvalidInput ErrorCode = "INVALID_INPUT"
	ErrorCodeServerError  ErrorCode = "SERVER_ERROR"
	ErrorCodeUnknown      ErrorCode = "UNKNOWN"
)

// Request represents a request to an LLM provider
type Request struct {
	Prompt      string            // The prompt to send to the LLM
	Role        Role              // The role context for this request
	Parameters  map[string]any    // Merged parameters (global + role overrides)
	MaxTokens   int               // Maximum tokens in response
	Temperature float64           // Sampling temperature (0.0-2.0)
	Metadata    map[string]string // Additional context (task ID, etc.)
}

// Response represents a response from an LLM provider
type Response struct {
	Content        string            // The LLM's response text
	TokensPrompt   int               // Tokens used in prompt
	TokensResponse int               // Tokens in response
	Model          string            // Actual model used
	Metadata       map[string]string // Provider-specific metadata
	Error          error             // Non-nil if request failed
}

// TaskExecutionContext contains execution metadata for a single task
type TaskExecutionContext struct {
	TaskID           string     // Unique identifier for the task
	Role             Role       // The role assigned to this task
	SelectedProvider string     // The provider ID that executed or will execute this task
	StartTime        time.Time  // Task execution start timestamp
	EndTime          time.Time  // Task execution completion timestamp (zero if in progress)
	Status           TaskStatus // Current task status
	Attempt          int        // Current retry attempt number (1-based)
	Error            string     // Error message if status is Failed
}

// Validate validates the task execution context
func (t *TaskExecutionContext) Validate() error {
	if t.TaskID == "" {
		return fmt.Errorf("task ID must not be empty")
	}
	if t.SelectedProvider == "" {
		return fmt.Errorf("selected provider must not be empty")
	}
	if t.Attempt < 1 {
		return fmt.Errorf("attempt must be >= 1")
	}
	if !t.EndTime.IsZero() && t.EndTime.Before(t.StartTime) {
		return fmt.Errorf("end time must be after start time")
	}
	return nil
}

// TransitionTo validates and performs a state transition
func (t *TaskExecutionContext) TransitionTo(newStatus TaskStatus) error {
	// Validate state transition
	switch t.Status {
	case TaskStatusPending:
		if newStatus != TaskStatusRunning {
			return fmt.Errorf("invalid transition from Pending to %s (must transition to Running)", newStatus)
		}
	case TaskStatusRunning:
		if newStatus != TaskStatusCompleted && newStatus != TaskStatusFailed && newStatus != TaskStatusRunning {
			return fmt.Errorf("invalid transition from Running to %s", newStatus)
		}
	case TaskStatusCompleted, TaskStatusFailed:
		return fmt.Errorf("cannot transition from terminal state %s", t.Status)
	}

	t.Status = newStatus
	return nil
}
