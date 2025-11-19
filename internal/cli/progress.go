// Package cli provides command-line interface utilities for GoCreator,
// including progress tracking and display formatting.
package cli

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/fatih/color"
)

// ProgressConfig configures progress display behavior
type ProgressConfig struct {
	// Writer is where progress output is written (default: os.Stdout)
	Writer io.Writer

	// ShowTokens enables token usage display
	ShowTokens bool

	// ShowCost enables cost display
	ShowCost bool

	// ShowETA enables ETA calculation
	ShowETA bool

	// UpdateInterval is how often to refresh the display
	UpdateInterval time.Duration

	// Quiet disables all progress output
	Quiet bool
}

// ProgressTracker tracks and displays progress during generation
type ProgressTracker struct {
	config ProgressConfig
	mu     sync.RWMutex

	// State
	startTime       time.Time
	currentPhase    string
	currentFile     string
	fileStartTime   time.Time
	totalPhases     int
	completedPhases int
	filesCompleted  int

	// Metrics
	totalInputTokens  int64
	totalOutputTokens int64
	totalCachedTokens int64
	cacheHits         int
	cacheMisses       int
	totalCost         float64
	estimatedCost     float64

	// Phase tracking
	phaseStartTime map[string]time.Time
	phaseDurations map[string]time.Duration

	// Colors
	green  *color.Color
	yellow *color.Color
	red    *color.Color
	cyan   *color.Color
	blue   *color.Color
	gray   *color.Color
	bold   *color.Color

	// Spinner
	spinnerIndex int
	spinnerChars []string
	stopSpinner  chan struct{}
	spinnerDone  chan struct{}
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(config ProgressConfig) *ProgressTracker {
	if config.UpdateInterval == 0 {
		config.UpdateInterval = 500 * time.Millisecond
	}

	return &ProgressTracker{
		config:         config,
		startTime:      time.Now(),
		phaseStartTime: make(map[string]time.Time),
		phaseDurations: make(map[string]time.Duration),
		green:          color.New(color.FgGreen),
		yellow:         color.New(color.FgYellow),
		red:            color.New(color.FgRed),
		cyan:           color.New(color.FgCyan),
		blue:           color.New(color.FgBlue),
		gray:           color.New(color.FgHiBlack),
		bold:           color.New(color.Bold),
		spinnerChars:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		stopSpinner:    make(chan struct{}),
		spinnerDone:    make(chan struct{}),
	}
}

// Start begins progress tracking
func (pt *ProgressTracker) Start(totalPhases int) {
	if pt.config.Quiet {
		return
	}

	pt.mu.Lock()
	pt.totalPhases = totalPhases
	pt.startTime = time.Now()
	pt.mu.Unlock()

	pt.printHeader()
}

// HandleEvent processes a progress event
func (pt *ProgressTracker) HandleEvent(event models.ProgressEvent) {
	if pt.config.Quiet {
		return
	}

	pt.mu.Lock()
	defer pt.mu.Unlock()

	switch event.Type {
	case models.EventPhaseStarted:
		pt.handlePhaseStarted(event)
	case models.EventPhaseCompleted:
		pt.handlePhaseCompleted(event)
	case models.EventFileGenerating:
		pt.handleFileGenerating(event)
	case models.EventFileCompleted:
		pt.handleFileCompleted(event)
	case models.EventTokensUsed:
		pt.handleTokensUsed(event)
	case models.EventCostUpdate:
		pt.handleCostUpdate(event)
	case models.EventError:
		pt.handleError(event)
	}
}

// Complete finalizes progress tracking and displays summary
func (pt *ProgressTracker) Complete() {
	if pt.config.Quiet {
		return
	}

	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.printSummary()
}

// printHeader prints the initial header
func (pt *ProgressTracker) printHeader() {
	// Write errors are intentionally ignored for best-effort console output
	_, _ = fmt.Fprintln(pt.config.Writer)
	_, _ = pt.bold.Fprintln(pt.config.Writer, "GoCreator - Code Generation")
	_, _ = fmt.Fprintln(pt.config.Writer, strings.Repeat("=", 50))
	_, _ = fmt.Fprintln(pt.config.Writer)
}

// handlePhaseStarted handles phase started events
func (pt *ProgressTracker) handlePhaseStarted(event models.ProgressEvent) {
	phase := event.Data["phase"].(string)
	description := ""
	if desc, ok := event.Data["description"].(string); ok {
		description = desc
	}

	pt.currentPhase = phase
	pt.phaseStartTime[phase] = time.Now()
	pt.fileStartTime = time.Time{}
	pt.currentFile = ""

	// Print phase header
	pt.printPhaseHeader(phase, description)
}

// handlePhaseCompleted handles phase completed events
func (pt *ProgressTracker) handlePhaseCompleted(event models.ProgressEvent) {
	phase := event.Data["phase"].(string)
	duration := event.Data["duration"].(time.Duration)
	files := 0
	if f, ok := event.Data["files"].(int); ok {
		files = f
	}

	pt.phaseDurations[phase] = duration
	pt.completedPhases++

	// Print phase completion
	pt.printPhaseComplete(phase, duration, files)
	_, _ = fmt.Fprintln(pt.config.Writer)
}

// handleFileGenerating handles file generating events
func (pt *ProgressTracker) handleFileGenerating(event models.ProgressEvent) {
	path := event.Data["path"].(string)
	pt.currentFile = path
	pt.fileStartTime = time.Now()

	// Start spinner for this file
	go pt.runSpinner()
}

// handleFileCompleted handles file completed events
func (pt *ProgressTracker) handleFileCompleted(event models.ProgressEvent) {
	path := event.Data["path"].(string)
	lines := event.Data["lines"].(int)
	var duration time.Duration
	if d, ok := event.Data["duration"].(time.Duration); ok {
		duration = d
	}

	pt.filesCompleted++
	pt.currentFile = ""

	// Stop spinner
	pt.stopCurrentSpinner()

	// Print file completion
	pt.printFileComplete(path, lines, duration)
}

// handleTokensUsed handles token usage events
func (pt *ProgressTracker) handleTokensUsed(event models.ProgressEvent) {
	if !pt.config.ShowTokens {
		return
	}

	totalInput := event.Data["total_input"].(int64)
	totalOutput := event.Data["total_output"].(int64)
	totalCached := event.Data["total_cached"].(int64)

	pt.totalInputTokens = totalInput
	pt.totalOutputTokens = totalOutput
	pt.totalCachedTokens = totalCached

	if cachedTokens, ok := event.Data["cached_tokens"].(int64); ok && cachedTokens > 0 {
		pt.cacheHits++
	} else {
		pt.cacheMisses++
	}

	pt.printMetricsUpdate()
}

// handleCostUpdate handles cost update events
func (pt *ProgressTracker) handleCostUpdate(event models.ProgressEvent) {
	if !pt.config.ShowCost {
		return
	}

	totalCost := event.Data["total_cost"].(float64)
	pt.totalCost = totalCost

	if estimated, ok := event.Data["estimated_total"].(float64); ok {
		pt.estimatedCost = estimated
	}
}

// handleError handles error events
func (pt *ProgressTracker) handleError(event models.ProgressEvent) {
	phase := event.Data["phase"].(string)
	message := event.Data["message"].(string)
	file := ""
	if f, ok := event.Data["file"].(string); ok {
		file = f
	}

	// Stop any running spinner
	pt.stopCurrentSpinner()

	// Print error
	pt.printError(phase, message, file)
}

// runSpinner runs a spinner animation
func (pt *ProgressTracker) runSpinner() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-pt.stopSpinner:
			close(pt.spinnerDone)
			return
		case <-ticker.C:
			pt.mu.Lock()
			if pt.currentFile != "" {
				elapsed := time.Since(pt.fileStartTime)
				_, _ = fmt.Fprintf(pt.config.Writer, "\r%s Generating %s... (%s elapsed)",
					pt.cyan.Sprint(pt.spinnerChars[pt.spinnerIndex]),
					pt.currentFile,
					pt.gray.Sprint(formatDuration(elapsed)))
				pt.spinnerIndex = (pt.spinnerIndex + 1) % len(pt.spinnerChars)
			}
			pt.mu.Unlock()
		}
	}
}

// stopCurrentSpinner stops the currently running spinner
func (pt *ProgressTracker) stopCurrentSpinner() {
	select {
	case pt.stopSpinner <- struct{}{}:
		<-pt.spinnerDone
	default:
		// No spinner running
	}

	// Clear the spinner line
	_, _ = fmt.Fprintf(pt.config.Writer, "\r%s\r", strings.Repeat(" ", 80))
}

// printPhaseHeader prints a phase header
func (pt *ProgressTracker) printPhaseHeader(phase, description string) {
	// Write errors are intentionally ignored for best-effort console output
	progress := fmt.Sprintf("[%d/%d]", pt.completedPhases+1, pt.totalPhases)
	_, _ = pt.cyan.Fprintf(pt.config.Writer, "%s Phase: %s\n", progress, phase)
	if description != "" {
		_, _ = pt.gray.Fprintf(pt.config.Writer, "      %s\n", description)
	}
	_, _ = fmt.Fprintln(pt.config.Writer)
}

// printPhaseComplete prints phase completion
func (pt *ProgressTracker) printPhaseComplete(phase string, duration time.Duration, files int) {
	// Write errors are intentionally ignored for best-effort console output
	_, _ = pt.green.Fprintf(pt.config.Writer, "  ✓ %s completed", phase)
	_, _ = fmt.Fprintf(pt.config.Writer, " (%s", formatDuration(duration))
	if files > 0 {
		_, _ = fmt.Fprintf(pt.config.Writer, ", %d files", files)
	}
	_, _ = fmt.Fprintln(pt.config.Writer, ")")
}

// printFileComplete prints file completion
func (pt *ProgressTracker) printFileComplete(path string, lines int, duration time.Duration) {
	// Write errors are intentionally ignored for best-effort console output
	_, _ = fmt.Fprintf(pt.config.Writer, "  ")
	_, _ = pt.green.Fprintf(pt.config.Writer, "✓")
	_, _ = fmt.Fprintf(pt.config.Writer, " %s", path)

	if lines > 0 {
		_, _ = pt.gray.Fprintf(pt.config.Writer, " (%d lines", lines)
		if duration > 0 {
			_, _ = pt.gray.Fprintf(pt.config.Writer, ", %s", formatDuration(duration))
		}
		_, _ = pt.gray.Fprintf(pt.config.Writer, ")")
	}

	_, _ = fmt.Fprintln(pt.config.Writer)
}

// printMetricsUpdate prints current metrics
func (pt *ProgressTracker) printMetricsUpdate() {
	// Write errors are intentionally ignored for best-effort console output
	_, _ = fmt.Fprintln(pt.config.Writer)
	_, _ = pt.bold.Fprintln(pt.config.Writer, "Progress Metrics:")

	// Files
	_, _ = fmt.Fprintf(pt.config.Writer, "  Files: %d completed\n", pt.filesCompleted)

	// Tokens
	if pt.config.ShowTokens {
		_, _ = fmt.Fprintf(pt.config.Writer, "  Tokens: %s input, %s output",
			formatNumber(pt.totalInputTokens),
			formatNumber(pt.totalOutputTokens))

		if pt.totalCachedTokens > 0 {
			cacheHitRate := float64(pt.totalCachedTokens) / float64(pt.totalInputTokens) * 100
			_, _ = pt.green.Fprintf(pt.config.Writer, " (%s cached - %.1f%% hit rate)",
				formatNumber(pt.totalCachedTokens),
				cacheHitRate)
		}
		_, _ = fmt.Fprintln(pt.config.Writer)
	}

	// Cost
	if pt.config.ShowCost && pt.totalCost > 0 {
		_, _ = fmt.Fprintf(pt.config.Writer, "  Cost: $%.4f", pt.totalCost)
		if pt.estimatedCost > 0 {
			_, _ = fmt.Fprintf(pt.config.Writer, " (estimated total: $%.4f)", pt.estimatedCost)
		}
		_, _ = fmt.Fprintln(pt.config.Writer)
	}

	// ETA
	if pt.config.ShowETA && pt.completedPhases > 0 && pt.completedPhases < pt.totalPhases {
		elapsed := time.Since(pt.startTime)
		avgPerPhase := elapsed / time.Duration(pt.completedPhases)
		remaining := avgPerPhase * time.Duration(pt.totalPhases-pt.completedPhases)
		_, _ = fmt.Fprintf(pt.config.Writer, "  ETA: ~%s remaining\n", formatDuration(remaining))
	}

	_, _ = fmt.Fprintln(pt.config.Writer)
}

// printError prints an error
func (pt *ProgressTracker) printError(phase, message, file string) {
	// Write errors are intentionally ignored for best-effort console output
	_, _ = fmt.Fprintln(pt.config.Writer)
	_, _ = pt.red.Fprintf(pt.config.Writer, "✗ Error in phase %s\n", phase)
	if file != "" {
		_, _ = fmt.Fprintf(pt.config.Writer, "  File: %s\n", file)
	}
	_, _ = fmt.Fprintf(pt.config.Writer, "  %s\n", message)
	_, _ = fmt.Fprintln(pt.config.Writer)
}

// printSummary prints final summary
func (pt *ProgressTracker) printSummary() {
	// Write errors are intentionally ignored for best-effort console output
	totalDuration := time.Since(pt.startTime)

	_, _ = fmt.Fprintln(pt.config.Writer)
	_, _ = fmt.Fprintln(pt.config.Writer, strings.Repeat("=", 50))
	_, _ = pt.bold.Fprintln(pt.config.Writer, "Generation Complete!")
	_, _ = fmt.Fprintln(pt.config.Writer)

	// Overall stats
	_, _ = pt.green.Fprintf(pt.config.Writer, "✓ Total Duration: %s\n", formatDuration(totalDuration))
	_, _ = fmt.Fprintf(pt.config.Writer, "✓ Files Generated: %d\n", pt.filesCompleted)

	// Phase breakdown
	if len(pt.phaseDurations) > 0 {
		_, _ = fmt.Fprintln(pt.config.Writer)
		_, _ = pt.bold.Fprintln(pt.config.Writer, "Phase Breakdown:")
		for phase, duration := range pt.phaseDurations {
			percentage := float64(duration) / float64(totalDuration) * 100
			_, _ = fmt.Fprintf(pt.config.Writer, "  %s: %s (%.1f%%)\n",
				phase,
				formatDuration(duration),
				percentage)
		}
	}

	// Token stats
	if pt.config.ShowTokens && pt.totalInputTokens > 0 {
		_, _ = fmt.Fprintln(pt.config.Writer)
		_, _ = pt.bold.Fprintln(pt.config.Writer, "Token Usage:")
		_, _ = fmt.Fprintf(pt.config.Writer, "  Input: %s tokens\n", formatNumber(pt.totalInputTokens))
		_, _ = fmt.Fprintf(pt.config.Writer, "  Output: %s tokens\n", formatNumber(pt.totalOutputTokens))

		if pt.totalCachedTokens > 0 {
			cacheHitRate := float64(pt.totalCachedTokens) / float64(pt.totalInputTokens) * 100
			_, _ = pt.green.Fprintf(pt.config.Writer, "  Cached: %s tokens (%.1f%% hit rate)\n",
				formatNumber(pt.totalCachedTokens),
				cacheHitRate)
		}
	}

	// Cost stats
	if pt.config.ShowCost && pt.totalCost > 0 {
		_, _ = fmt.Fprintln(pt.config.Writer)
		_, _ = pt.bold.Fprintln(pt.config.Writer, "Cost:")
		_, _ = fmt.Fprintf(pt.config.Writer, "  Total: $%.4f\n", pt.totalCost)
	}

	_, _ = fmt.Fprintln(pt.config.Writer)
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Milliseconds()))
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}

// formatNumber formats a number with thousand separators
func formatNumber(n int64) string {
	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}

	var result strings.Builder
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}
