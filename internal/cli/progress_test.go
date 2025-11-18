package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/dshills/gocreator/internal/models"
)

func TestProgressTracker_Basic(t *testing.T) {
	var buf bytes.Buffer

	config := ProgressConfig{
		Writer:         &buf,
		ShowTokens:     true,
		ShowCost:       true,
		ShowETA:        true,
		UpdateInterval: 100 * time.Millisecond,
		Quiet:          false,
	}

	tracker := NewProgressTracker(config)
	if tracker == nil {
		t.Fatal("NewProgressTracker returned nil")
	}

	// Start tracking
	tracker.Start(3)

	// Simulate phase lifecycle
	tracker.HandleEvent(models.NewPhaseStartedEvent("test_phase", "Testing phase tracking"))
	time.Sleep(50 * time.Millisecond)
	tracker.HandleEvent(models.NewPhaseCompletedEvent("test_phase", 50*time.Millisecond, 5))

	// Complete tracking
	tracker.Complete()

	output := buf.String()

	// Verify output contains expected elements
	if !strings.Contains(output, "GoCreator") {
		t.Error("Output should contain 'GoCreator' header")
	}

	if !strings.Contains(output, "test_phase") {
		t.Error("Output should contain phase name")
	}

	if !strings.Contains(output, "Generation Complete") {
		t.Error("Output should contain completion message")
	}
}

func TestProgressTracker_FileTracking(t *testing.T) {
	var buf bytes.Buffer

	config := ProgressConfig{
		Writer:         &buf,
		ShowTokens:     false,
		ShowCost:       false,
		ShowETA:        false,
		UpdateInterval: 100 * time.Millisecond,
		Quiet:          false,
	}

	tracker := NewProgressTracker(config)
	tracker.Start(1)

	// Simulate file generation
	tracker.HandleEvent(models.NewPhaseStartedEvent("generate", "Generating files"))
	tracker.HandleEvent(models.NewFileGeneratingEvent("/path/to/file.go", "generate"))
	time.Sleep(50 * time.Millisecond)
	tracker.HandleEvent(models.NewFileCompletedEvent("/path/to/file.go", "generate", 150, 50*time.Millisecond))
	tracker.HandleEvent(models.NewPhaseCompletedEvent("generate", 100*time.Millisecond, 1))

	tracker.Complete()

	output := buf.String()

	if !strings.Contains(output, "file.go") {
		t.Error("Output should contain filename")
	}

	if !strings.Contains(output, "150 lines") {
		t.Error("Output should contain line count")
	}
}

func TestProgressTracker_TokenTracking(t *testing.T) {
	var buf bytes.Buffer

	config := ProgressConfig{
		Writer:         &buf,
		ShowTokens:     true,
		ShowCost:       false,
		ShowETA:        false,
		UpdateInterval: 100 * time.Millisecond,
		Quiet:          false,
	}

	tracker := NewProgressTracker(config)
	tracker.Start(1)

	// Simulate token usage
	tracker.HandleEvent(models.NewPhaseStartedEvent("generate", "Generating code"))
	tracker.HandleEvent(models.NewTokensUsedEvent(
		"anthropic",
		1000, // input tokens
		500,  // output tokens
		800,  // cached tokens
		1000, // total input
		500,  // total output
		800,  // total cached
		0.8,  // cache hit rate
	))
	tracker.HandleEvent(models.NewPhaseCompletedEvent("generate", 100*time.Millisecond, 1))

	tracker.Complete()

	output := buf.String()

	if !strings.Contains(output, "Tokens") {
		t.Error("Output should contain token information")
	}

	if !strings.Contains(output, "cached") {
		t.Error("Output should mention cached tokens")
	}
}

func TestProgressTracker_CostTracking(t *testing.T) {
	var buf bytes.Buffer

	config := ProgressConfig{
		Writer:         &buf,
		ShowTokens:     false,
		ShowCost:       true,
		ShowETA:        false,
		UpdateInterval: 100 * time.Millisecond,
		Quiet:          false,
	}

	tracker := NewProgressTracker(config)
	tracker.Start(1)

	// Simulate cost tracking
	tracker.HandleEvent(models.NewPhaseStartedEvent("generate", "Generating code"))
	tracker.HandleEvent(models.NewCostUpdateEvent("anthropic", 0.05, 0.15, 0.20))
	tracker.HandleEvent(models.NewPhaseCompletedEvent("generate", 100*time.Millisecond, 1))

	tracker.Complete()

	output := buf.String()

	if !strings.Contains(output, "Cost") || !strings.Contains(output, "$") {
		t.Error("Output should contain cost information")
	}
}

func TestProgressTracker_ErrorHandling(t *testing.T) {
	var buf bytes.Buffer

	config := ProgressConfig{
		Writer:         &buf,
		ShowTokens:     false,
		ShowCost:       false,
		ShowETA:        false,
		UpdateInterval: 100 * time.Millisecond,
		Quiet:          false,
	}

	tracker := NewProgressTracker(config)
	tracker.Start(1)

	// Simulate error
	tracker.HandleEvent(models.NewPhaseStartedEvent("generate", "Generating code"))
	tracker.HandleEvent(models.NewErrorEvent("generate", "Something went wrong", "/path/to/file.go"))

	tracker.Complete()

	output := buf.String()

	if !strings.Contains(output, "Error") {
		t.Error("Output should contain error information")
	}

	if !strings.Contains(output, "Something went wrong") {
		t.Error("Output should contain error message")
	}
}

func TestProgressTracker_QuietMode(t *testing.T) {
	var buf bytes.Buffer

	config := ProgressConfig{
		Writer:         &buf,
		ShowTokens:     true,
		ShowCost:       true,
		ShowETA:        true,
		UpdateInterval: 100 * time.Millisecond,
		Quiet:          true, // Quiet mode enabled
	}

	tracker := NewProgressTracker(config)
	tracker.Start(1)

	// These should not produce output
	tracker.HandleEvent(models.NewPhaseStartedEvent("generate", "Generating code"))
	tracker.HandleEvent(models.NewPhaseCompletedEvent("generate", 100*time.Millisecond, 1))
	tracker.Complete()

	output := buf.String()

	if output != "" {
		t.Error("Quiet mode should produce no output")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{500 * time.Millisecond, "500ms"},
		{1500 * time.Millisecond, "1.5s"},
		{65 * time.Second, "1m5s"},
		{125 * time.Second, "2m5s"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.duration)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
		}
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		num  int64
		want string
	}{
		{123, "123"},
		{1234, "1,234"},
		{1234567, "1,234,567"},
		{1000000000, "1,000,000,000"},
	}

	for _, tt := range tests {
		got := formatNumber(tt.num)
		if got != tt.want {
			t.Errorf("formatNumber(%d) = %q, want %q", tt.num, got, tt.want)
		}
	}
}
