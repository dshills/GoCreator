package models

import "time"

// EventType represents the type of progress event
type EventType string

const (
	// EventPhaseStarted indicates a generation phase has started
	EventPhaseStarted EventType = "phase_started"

	// EventPhaseCompleted indicates a generation phase has completed
	EventPhaseCompleted EventType = "phase_completed"

	// EventFileGenerating indicates a file is being generated
	EventFileGenerating EventType = "file_generating"

	// EventFileCompleted indicates a file has been generated
	EventFileCompleted EventType = "file_completed"

	// EventTokensUsed indicates tokens were consumed
	EventTokensUsed EventType = "tokens_used"

	// EventCostUpdate indicates a cost update
	EventCostUpdate EventType = "cost_update"

	// EventError indicates an error occurred
	EventError EventType = "error"
)

// ProgressEvent represents a progress event during generation
type ProgressEvent struct {
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// PhaseStartedData contains data for phase started events
type PhaseStartedData struct {
	Phase       string `json:"phase"`
	Description string `json:"description,omitempty"`
}

// PhaseCompletedData contains data for phase completed events
type PhaseCompletedData struct {
	Phase    string        `json:"phase"`
	Duration time.Duration `json:"duration"`
	Files    int           `json:"files,omitempty"`
}

// FileGeneratingData contains data for file generating events
type FileGeneratingData struct {
	Path  string `json:"path"`
	Phase string `json:"phase"`
}

// FileCompletedData contains data for file completed events
type FileCompletedData struct {
	Path     string        `json:"path"`
	Phase    string        `json:"phase"`
	Lines    int           `json:"lines"`
	Duration time.Duration `json:"duration,omitempty"`
}

// TokensUsedData contains data for token usage events
type TokensUsedData struct {
	Provider     string  `json:"provider"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	CachedTokens int64   `json:"cached_tokens,omitempty"`
	TotalInput   int64   `json:"total_input"`
	TotalOutput  int64   `json:"total_output"`
	TotalCached  int64   `json:"total_cached"`
	CacheHitRate float64 `json:"cache_hit_rate,omitempty"`
}

// CostUpdateData contains data for cost update events
type CostUpdateData struct {
	Provider        string  `json:"provider"`
	IncrementalCost float64 `json:"incremental_cost"`
	TotalCost       float64 `json:"total_cost"`
	EstimatedTotal  float64 `json:"estimated_total,omitempty"`
}

// ErrorData contains data for error events
type ErrorData struct {
	Phase   string `json:"phase"`
	Message string `json:"message"`
	File    string `json:"file,omitempty"`
}

// NewPhaseStartedEvent creates a phase started event
func NewPhaseStartedEvent(phase, description string) ProgressEvent {
	return ProgressEvent{
		Type:      EventPhaseStarted,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"phase":       phase,
			"description": description,
		},
	}
}

// NewPhaseCompletedEvent creates a phase completed event
func NewPhaseCompletedEvent(phase string, duration time.Duration, files int) ProgressEvent {
	return ProgressEvent{
		Type:      EventPhaseCompleted,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"phase":    phase,
			"duration": duration,
			"files":    files,
		},
	}
}

// NewFileGeneratingEvent creates a file generating event
func NewFileGeneratingEvent(path, phase string) ProgressEvent {
	return ProgressEvent{
		Type:      EventFileGenerating,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"path":  path,
			"phase": phase,
		},
	}
}

// NewFileCompletedEvent creates a file completed event
func NewFileCompletedEvent(path, phase string, lines int, duration time.Duration) ProgressEvent {
	return ProgressEvent{
		Type:      EventFileCompleted,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"path":     path,
			"phase":    phase,
			"lines":    lines,
			"duration": duration,
		},
	}
}

// NewTokensUsedEvent creates a tokens used event
func NewTokensUsedEvent(provider string, inputTokens, outputTokens, cachedTokens, totalInput, totalOutput, totalCached int64, cacheHitRate float64) ProgressEvent {
	return ProgressEvent{
		Type:      EventTokensUsed,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"provider":       provider,
			"input_tokens":   inputTokens,
			"output_tokens":  outputTokens,
			"cached_tokens":  cachedTokens,
			"total_input":    totalInput,
			"total_output":   totalOutput,
			"total_cached":   totalCached,
			"cache_hit_rate": cacheHitRate,
		},
	}
}

// NewCostUpdateEvent creates a cost update event
func NewCostUpdateEvent(provider string, incrementalCost, totalCost, estimatedTotal float64) ProgressEvent {
	return ProgressEvent{
		Type:      EventCostUpdate,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"provider":         provider,
			"incremental_cost": incrementalCost,
			"total_cost":       totalCost,
			"estimated_total":  estimatedTotal,
		},
	}
}

// NewErrorEvent creates an error event
func NewErrorEvent(phase, message, file string) ProgressEvent {
	return ProgressEvent{
		Type:      EventError,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"phase":   phase,
			"message": message,
			"file":    file,
		},
	}
}
