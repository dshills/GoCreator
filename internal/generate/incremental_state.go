// Package generate provides code generation functionality for GoCreator.
package generate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/rs/zerolog/log"
)

// IncrementalState tracks generation state for incremental regeneration
type IncrementalState struct {
	// FCSChecksum is the SHA-256 checksum of the entire FCS
	FCSChecksum string `json:"fcs_checksum"`

	// PreviousFCS stores the complete FCS from the last generation
	// This enables fine-grained change detection
	PreviousFCS *models.FinalClarifiedSpecification `json:"previous_fcs,omitempty"`

	// GeneratedFiles maps file path to its state
	GeneratedFiles map[string]FileState `json:"generated_files"`

	// DependencyGraph maps file path to list of FCS entities it depends on
	DependencyGraph map[string][]string `json:"dependency_graph"`

	// LastGeneration is the timestamp of the last generation
	LastGeneration time.Time `json:"last_generation"`

	// Version is the state file format version
	Version string `json:"version"`
}

// FileState represents the state of a single generated file
type FileState struct {
	// Path is the relative path to the file
	Path string `json:"path"`

	// Checksum is the SHA-256 checksum of the file content
	Checksum string `json:"checksum"`

	// GeneratedAt is when the file was generated
	GeneratedAt time.Time `json:"generated_at"`

	// Dependencies lists FCS entity names this file depends on
	Dependencies []string `json:"dependencies"`

	// Template indicates if this file was generated from a template
	Template bool `json:"template"`

	// TaskID is the generation task ID that created this file
	TaskID string `json:"task_id"`
}

// IncrementalStateManager manages incremental state persistence
type IncrementalStateManager struct {
	mu            sync.RWMutex
	stateFilePath string
	state         *IncrementalState
}

// NewIncrementalStateManager creates a new state manager
func NewIncrementalStateManager(outputDir string) *IncrementalStateManager {
	stateDir := filepath.Join(outputDir, ".gocreator")
	stateFilePath := filepath.Join(stateDir, "state.json")

	return &IncrementalStateManager{
		stateFilePath: stateFilePath,
		state:         nil, // Will be loaded on first Load() call
	}
}

// Load loads the incremental state from disk
// Returns a new empty state if the file doesn't exist
func (ism *IncrementalStateManager) Load() (*IncrementalState, error) {
	// Check if state file exists
	if _, err := os.Stat(ism.stateFilePath); os.IsNotExist(err) {
		log.Debug().
			Str("path", ism.stateFilePath).
			Msg("No existing state file, creating new state")

		state := &IncrementalState{
			GeneratedFiles:  make(map[string]FileState),
			DependencyGraph: make(map[string][]string),
			Version:         "1.0",
		}
		ism.state = state
		return state, nil
	}

	// Read state file
	data, err := os.ReadFile(ism.stateFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	// Parse JSON
	var state IncrementalState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	// Ensure maps are initialized
	if state.GeneratedFiles == nil {
		state.GeneratedFiles = make(map[string]FileState)
	}
	if state.DependencyGraph == nil {
		state.DependencyGraph = make(map[string][]string)
	}

	log.Debug().
		Str("path", ism.stateFilePath).
		Int("files", len(state.GeneratedFiles)).
		Str("fcs_checksum", state.FCSChecksum).
		Msg("Loaded incremental state")

	ism.mu.Lock()
	ism.state = &state
	ism.mu.Unlock()
	return &state, nil
}

// Save persists the incremental state to disk
func (ism *IncrementalStateManager) Save(state *IncrementalState) error {
	// Ensure state directory exists
	stateDir := filepath.Dir(ism.stateFilePath)
	if err := os.MkdirAll(stateDir, 0750); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temporary file first
	tempPath := ism.stateFilePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, ism.stateFilePath); err != nil {
		_ = os.Remove(tempPath) // Clean up temp file, ignore error
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	log.Debug().
		Str("path", ism.stateFilePath).
		Int("files", len(state.GeneratedFiles)).
		Msg("Saved incremental state")

	ism.mu.Lock()
	ism.state = state
	ism.mu.Unlock()
	return nil
}

// ComputeFCSChecksum computes the SHA-256 checksum of an FCS
func ComputeFCSChecksum(fcs *models.FinalClarifiedSpecification) (string, error) {
	// Marshal FCS to canonical JSON for consistent hashing
	data, err := json.Marshal(fcs)
	if err != nil {
		return "", fmt.Errorf("failed to marshal FCS: %w", err)
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// ComputeFileChecksum computes the SHA-256 checksum of file content
func ComputeFileChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// normalizePath normalizes a file path for consistent storage in dependency graphs
// Uses filepath.Clean to remove redundant separators and resolve . and .. elements
func normalizePath(path string) string {
	if path == "" {
		return ""
	}
	// Clean the path to ensure consistent format
	return filepath.Clean(path)
}

// UpdateState updates the state after successful generation
func (ism *IncrementalStateManager) UpdateState(
	fcs *models.FinalClarifiedSpecification,
	patches []models.Patch,
	dependencyGraph map[string][]string,
) error {
	if ism.state == nil {
		state, err := ism.Load()
		if err != nil {
			return fmt.Errorf("failed to load state: %w", err)
		}
		ism.state = state
	}

	// Compute new FCS checksum
	fcsChecksum, err := ComputeFCSChecksum(fcs)
	if err != nil {
		return fmt.Errorf("failed to compute FCS checksum: %w", err)
	}

	// Update FCS checksum and store the complete FCS for next comparison
	ism.state.FCSChecksum = fcsChecksum
	ism.state.PreviousFCS = fcs
	ism.state.LastGeneration = time.Now()

	// Update dependency graph with normalized paths
	for path, deps := range dependencyGraph {
		normalizedPath := normalizePath(path)
		ism.state.DependencyGraph[normalizedPath] = deps
	}

	// Update file states from patches
	for _, patch := range patches {
		// Normalize the target file path
		normalizedPath := normalizePath(patch.TargetFile)

		// Extract file content from diff (simplified - assumes new file creation)
		content := extractContentFromDiff(patch.Diff)
		checksum := ComputeFileChecksum(content)

		fileState := FileState{
			Path:         normalizedPath,
			Checksum:     checksum,
			GeneratedAt:  patch.AppliedAt,
			Dependencies: dependencyGraph[patch.TargetFile], // Use original path to look up in incoming graph
			Template:     isTemplateFile(normalizedPath),
		}

		ism.state.GeneratedFiles[normalizedPath] = fileState
	}

	// Save updated state
	return ism.Save(ism.state)
}

// extractContentFromDiff extracts file content from a unified diff
// This is a simplified version that assumes new file creation (all lines start with +)
func extractContentFromDiff(diff string) string {
	lines := []string{}
	for _, line := range splitLines(diff) {
		// Skip diff header lines
		if len(line) > 0 && line[0] == '+' && !isHeaderLine(line) {
			// Remove the leading '+' prefix
			lines = append(lines, line[1:])
		}
	}
	result := joinLines(lines)
	// If original diff ended with newline and we have content, add trailing newline
	if len(diff) > 0 && diff[len(diff)-1] == '\n' && len(result) > 0 {
		result += "\n"
	}
	return result
}

func splitLines(s string) []string {
	result := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func joinLines(lines []string) string {
	result := ""
	for i, line := range lines {
		result += line
		if i < len(lines)-1 {
			result += "\n"
		}
	}
	return result
}

func isHeaderLine(line string) bool {
	if len(line) < 3 {
		return false
	}
	return line[:3] == "+@@" || line[:3] == "+++"
}

// GetState returns the current state (loads if not already loaded)
func (ism *IncrementalStateManager) GetState() (*IncrementalState, error) {
	if ism.state == nil {
		return ism.Load()
	}
	return ism.state, nil
}

// Clear removes the state file
func (ism *IncrementalStateManager) Clear() error {
	if err := os.Remove(ism.stateFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove state file: %w", err)
	}

	log.Debug().Str("path", ism.stateFilePath).Msg("Cleared incremental state")
	ism.state = nil
	return nil
}

// isTemplateFile determines if a file is generated from a template rather than AI-generated code
func isTemplateFile(path string) bool {
	// Get the base filename
	base := filepath.Base(path)

	// Common template-generated files
	templateFiles := map[string]bool{
		"go.mod":              true,
		"go.sum":              true,
		"go.work":             true,
		"Makefile":            true,
		"Dockerfile":          true,
		"docker-compose.yml":  true,
		"docker-compose.yaml": true,
		".gitignore":          true,
		".dockerignore":       true,
		"LICENSE":             true,
		"LICENSE.txt":         true,
		"LICENSE.md":          true,
		".golangci.yml":       true,
		".golangci.yaml":      true,
		".editorconfig":       true,
		".env.example":        true,
		".env.template":       true,
	}

	return templateFiles[base]
}
