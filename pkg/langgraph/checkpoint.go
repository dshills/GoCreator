package langgraph

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Checkpoint represents a snapshot of graph execution state
type Checkpoint struct {
	ID                string    `json:"id"`
	GraphID           string    `json:"graph_id"`
	LastCompletedNode string    `json:"last_completed_node"`
	State             []byte    `json:"state"` // Serialized state
	CompletedNodes    []string  `json:"completed_nodes"`
	CreatedAt         time.Time `json:"created_at"`
	Recoverable       bool      `json:"recoverable"`
}

// CheckpointManager handles checkpoint creation and recovery
type CheckpointManager interface {
	// Save saves a checkpoint to storage
	Save(ctx ExecutionContext, state State) error

	// Load loads the latest checkpoint for a graph
	Load(graphID string) (*Checkpoint, error)

	// List lists all checkpoints for a graph
	List(graphID string) ([]*Checkpoint, error)

	// Delete removes a checkpoint
	Delete(checkpointID string) error

	// DeleteAll removes all checkpoints for a graph
	DeleteAll(graphID string) error
}

// FileCheckpointManager implements CheckpointManager using the file system
type FileCheckpointManager struct {
	baseDir string
}

// NewFileCheckpointManager creates a new file-based checkpoint manager
func NewFileCheckpointManager(baseDir string) (*FileCheckpointManager, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	return &FileCheckpointManager{
		baseDir: baseDir,
	}, nil
}

// Save saves a checkpoint to the file system
func (m *FileCheckpointManager) Save(ctx ExecutionContext, state State) error {
	// Serialize state
	stateData, err := state.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}

	// Create checkpoint
	checkpoint := &Checkpoint{
		ID:                uuid.New().String(),
		GraphID:           ctx.GraphID,
		LastCompletedNode: ctx.CurrentNode,
		State:             stateData,
		CompletedNodes:    ctx.CompletedNodes,
		CreatedAt:         time.Now(),
		Recoverable:       true,
	}

	// Create graph-specific directory
	graphDir := filepath.Join(m.baseDir, ctx.GraphID)
	if err := os.MkdirAll(graphDir, 0755); err != nil {
		return fmt.Errorf("failed to create graph checkpoint directory: %w", err)
	}

	// Write checkpoint file
	filename := filepath.Join(graphDir, fmt.Sprintf("checkpoint_%s.json", checkpoint.ID))
	data, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write checkpoint file: %w", err)
	}

	log.Info().
		Str("checkpoint_id", checkpoint.ID).
		Str("graph_id", ctx.GraphID).
		Str("node", ctx.CurrentNode).
		Str("file", filename).
		Msg("Checkpoint saved")

	return nil
}

// Load loads the latest checkpoint for a graph
func (m *FileCheckpointManager) Load(graphID string) (*Checkpoint, error) {
	checkpoints, err := m.List(graphID)
	if err != nil {
		return nil, err
	}

	if len(checkpoints) == 0 {
		return nil, fmt.Errorf("no checkpoints found for graph %s", graphID)
	}

	// Return the most recent checkpoint
	latest := checkpoints[0]
	for _, cp := range checkpoints {
		if cp.CreatedAt.After(latest.CreatedAt) {
			latest = cp
		}
	}

	log.Info().
		Str("checkpoint_id", latest.ID).
		Str("graph_id", graphID).
		Str("last_node", latest.LastCompletedNode).
		Time("created_at", latest.CreatedAt).
		Msg("Checkpoint loaded")

	return latest, nil
}

// List lists all checkpoints for a graph
func (m *FileCheckpointManager) List(graphID string) ([]*Checkpoint, error) {
	graphDir := filepath.Join(m.baseDir, graphID)

	// Check if directory exists
	if _, err := os.Stat(graphDir); os.IsNotExist(err) {
		return []*Checkpoint{}, nil
	}

	// Read directory
	entries, err := os.ReadDir(graphDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint directory: %w", err)
	}

	var checkpoints []*Checkpoint
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Read checkpoint file
		filename := filepath.Join(graphDir, entry.Name())
		data, err := os.ReadFile(filename)
		if err != nil {
			log.Warn().
				Err(err).
				Str("file", filename).
				Msg("Failed to read checkpoint file, skipping")
			continue
		}

		var checkpoint Checkpoint
		if err := json.Unmarshal(data, &checkpoint); err != nil {
			log.Warn().
				Err(err).
				Str("file", filename).
				Msg("Failed to unmarshal checkpoint file, skipping")
			continue
		}

		checkpoints = append(checkpoints, &checkpoint)
	}

	return checkpoints, nil
}

// Delete removes a checkpoint
func (m *FileCheckpointManager) Delete(checkpointID string) error {
	// Find the checkpoint file across all graph directories
	graphDirs, err := os.ReadDir(m.baseDir)
	if err != nil {
		return fmt.Errorf("failed to read base directory: %w", err)
	}

	for _, graphDir := range graphDirs {
		if !graphDir.IsDir() {
			continue
		}

		filename := filepath.Join(m.baseDir, graphDir.Name(), fmt.Sprintf("checkpoint_%s.json", checkpointID))
		if _, err := os.Stat(filename); err == nil {
			// File exists, delete it
			if err := os.Remove(filename); err != nil {
				return fmt.Errorf("failed to delete checkpoint file: %w", err)
			}

			log.Info().
				Str("checkpoint_id", checkpointID).
				Str("file", filename).
				Msg("Checkpoint deleted")

			return nil
		}
	}

	return fmt.Errorf("checkpoint %s not found", checkpointID)
}

// DeleteAll removes all checkpoints for a graph
func (m *FileCheckpointManager) DeleteAll(graphID string) error {
	graphDir := filepath.Join(m.baseDir, graphID)

	// Check if directory exists
	if _, err := os.Stat(graphDir); os.IsNotExist(err) {
		return nil // Nothing to delete
	}

	// Remove entire directory
	if err := os.RemoveAll(graphDir); err != nil {
		return fmt.Errorf("failed to remove checkpoint directory: %w", err)
	}

	log.Info().
		Str("graph_id", graphID).
		Str("directory", graphDir).
		Msg("All checkpoints deleted")

	return nil
}

// RecoverState restores state from a checkpoint
func RecoverState(checkpoint *Checkpoint) (State, error) {
	state := NewMapState()
	if err := state.FromJSON(checkpoint.State); err != nil {
		return nil, fmt.Errorf("failed to deserialize state: %w", err)
	}
	return state, nil
}
