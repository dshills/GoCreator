package models

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// GenerationTask represents a single generation task
type GenerationTask struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	TargetPath  string                 `json:"target_path,omitempty"`
	Inputs      map[string]interface{} `json:"inputs,omitempty"`
	CanParallel bool                   `json:"can_parallel"`
}

// Validate validates the generation task
func (t *GenerationTask) Validate() error {
	validTypes := map[string]bool{
		"generate_file": true,
		"apply_patch":   true,
		"run_command":   true,
	}

	if !validTypes[t.Type] {
		return fmt.Errorf("invalid task type: %s", t.Type)
	}

	return nil
}

// GenerationPhase represents a phase in the generation plan
type GenerationPhase struct {
	Name         string           `json:"name"`
	Order        int              `json:"order"`
	Tasks        []GenerationTask `json:"tasks"`
	Dependencies []string         `json:"dependencies,omitempty"`
}

// Directory represents a directory in the file tree
type Directory struct {
	Path    string `json:"path"`
	Purpose string `json:"purpose,omitempty"`
}

// File represents a file in the file tree
type File struct {
	Path        string `json:"path"`
	Purpose     string `json:"purpose,omitempty"`
	GeneratedBy string `json:"generated_by,omitempty"`
}

// FileTree represents the target directory structure
type FileTree struct {
	Root        string      `json:"root"`
	Directories []Directory `json:"directories,omitempty"`
	Files       []File      `json:"files,omitempty"`
}

// GenerationPlan represents a detailed plan for code generation
type GenerationPlan struct {
	SchemaVersion string            `json:"schema_version"`
	ID            string            `json:"id"`
	FCSID         string            `json:"fcs_id"`
	Phases        []GenerationPhase `json:"phases"`
	FileTree      FileTree          `json:"file_tree"`
	CreatedAt     time.Time         `json:"created_at"`
}

// Validate validates the generation plan
func (p *GenerationPlan) Validate() error {
	// Check for cyclic phase dependencies
	if p.HasCyclicDependencies() {
		return fmt.Errorf("cyclic dependency detected in generation phases")
	}

	// Check that all target paths are within root directory
	for _, phase := range p.Phases {
		for _, task := range phase.Tasks {
			if task.TargetPath != "" {
				if !p.isPathWithinRoot(task.TargetPath) {
					return fmt.Errorf("target path outside root: %s", task.TargetPath)
				}
			}
		}
	}

	// Check that parallel tasks don't write to the same file
	for _, phase := range p.Phases {
		if err := p.validateParallelTasks(phase.Tasks); err != nil {
			return err
		}
	}

	return nil
}

// isPathWithinRoot checks if a path is within the root directory
func (p *GenerationPlan) isPathWithinRoot(targetPath string) bool {
	// Clean the target path
	cleanTarget := filepath.Clean(targetPath)

	// If the path is absolute, it must start with the root
	if filepath.IsAbs(cleanTarget) {
		cleanRoot := filepath.Clean(p.FileTree.Root)
		return strings.HasPrefix(cleanTarget, cleanRoot)
	}

	// For relative paths, check that they don't try to escape with "../"
	// A relative path is considered safe if it doesn't start with ".."
	return !strings.HasPrefix(cleanTarget, ".."+string(filepath.Separator)) && cleanTarget != ".."
}

// validateParallelTasks checks that parallel tasks don't write to the same file
func (p *GenerationPlan) validateParallelTasks(tasks []GenerationTask) error {
	parallelPaths := make(map[string]bool)

	for _, task := range tasks {
		if task.CanParallel && task.TargetPath != "" {
			if parallelPaths[task.TargetPath] {
				return fmt.Errorf("parallel tasks cannot write to same file: %s", task.TargetPath)
			}
			parallelPaths[task.TargetPath] = true
		}
	}

	return nil
}

// HasCyclicDependencies detects cyclic dependencies in the phase graph
func (p *GenerationPlan) HasCyclicDependencies() bool {
	// Build adjacency list
	graph := make(map[string][]string)
	for _, phase := range p.Phases {
		graph[phase.Name] = phase.Dependencies
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
	for phase := range graph {
		if !visited[phase] {
			if hasCycle(phase) {
				return true
			}
		}
	}

	return false
}

// GetTaskByID finds a task by its ID across all phases
func (p *GenerationPlan) GetTaskByID(taskID string) *GenerationTask {
	for _, phase := range p.Phases {
		for i := range phase.Tasks {
			if phase.Tasks[i].ID == taskID {
				return &phase.Tasks[i]
			}
		}
	}
	return nil
}
