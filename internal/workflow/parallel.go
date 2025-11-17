package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

// TaskResult holds the result of a task execution
type TaskResult struct {
	TaskID      string
	Output      interface{}
	Error       error
	StartedAt   time.Time
	CompletedAt time.Time
}

// ParallelExecutor handles parallel execution of tasks with dependency resolution
type ParallelExecutor struct {
	maxWorkers   int
	taskRegistry *TaskRegistry
	logger       zerolog.Logger
}

// ParallelExecutorConfig holds configuration for parallel executor
type ParallelExecutorConfig struct {
	MaxWorkers   int
	TaskRegistry *TaskRegistry
	Logger       zerolog.Logger
}

// NewParallelExecutor creates a new parallel executor
func NewParallelExecutor(cfg ParallelExecutorConfig) *ParallelExecutor {
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = 4
	}

	return &ParallelExecutor{
		maxWorkers:   cfg.MaxWorkers,
		taskRegistry: cfg.TaskRegistry,
		logger:       cfg.Logger,
	}
}

// Execute runs tasks in parallel respecting dependencies
func (e *ParallelExecutor) Execute(ctx context.Context, tasks []models.WorkflowTask, execCtx *ExecutionContext) (map[string]*TaskResult, error) {
	// Build dependency graph
	graph := NewDAG()
	for _, task := range tasks {
		if err := graph.AddNode(task.ID, task); err != nil {
			return nil, fmt.Errorf("failed to add task to graph: %w", err)
		}
	}

	// Add edges for dependencies
	for _, task := range tasks {
		for _, depID := range task.Dependencies {
			if err := graph.AddEdge(depID, task.ID); err != nil {
				return nil, fmt.Errorf("failed to add dependency edge: %w", err)
			}
		}
	}

	// Validate graph (check for cycles)
	if graph.HasCycle() {
		return nil, fmt.Errorf("circular dependency detected in task graph")
	}

	// Get topological ordering
	ordering, err := graph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("failed to compute task ordering: %w", err)
	}

	e.logger.Debug().
		Int("total_tasks", len(tasks)).
		Int("max_workers", e.maxWorkers).
		Msg("Starting parallel task execution")

	// Execute tasks in topological order with parallelism
	results := make(map[string]*TaskResult)
	var resultsMu sync.Mutex

	// Track completed tasks
	completed := make(map[string]bool)
	var completedMu sync.RWMutex

	// Process tasks level by level (tasks at same level can run in parallel)
	levels := e.computeLevels(graph, ordering)

	for levelNum, levelTasks := range levels {
		e.logger.Debug().
			Int("level", levelNum).
			Int("tasks_in_level", len(levelTasks)).
			Msg("Executing task level")

		// Execute all tasks in this level in parallel
		g, gCtx := errgroup.WithContext(ctx)
		g.SetLimit(e.maxWorkers)

		for _, taskID := range levelTasks {
		taskID := taskID // Capture for goroutine (required for Go < 1.22)
			g.Go(func() error {
				// Get task definition
				taskDef, exists := graph.GetNode(taskID)
				if !exists {
					return fmt.Errorf("task not found in graph: %s", taskID)
				}

				task := taskDef.(models.WorkflowTask)

				// Check dependencies are completed
				for _, depID := range task.Dependencies {
					completedMu.RLock()
					isCompleted := completed[depID]
					completedMu.RUnlock()

					if !isCompleted {
						return fmt.Errorf("dependency not completed: %s", depID)
					}

					// Check if dependency failed
					resultsMu.Lock()
					depResult := results[depID]
					resultsMu.Unlock()

					if depResult.Error != nil {
						return fmt.Errorf("dependency failed: %s", depID)
					}
				}

				// Execute task
				result, err := e.executeTask(gCtx, task, execCtx)

				// Store result
				resultsMu.Lock()
				results[taskID] = result
				resultsMu.Unlock()

				// Mark as completed
				completedMu.Lock()
				completed[taskID] = true
				completedMu.Unlock()

				return err
			})
		}

		// Wait for all tasks in this level to complete
		if err := g.Wait(); err != nil {
			return results, fmt.Errorf("task execution failed at level %d: %w", levelNum, err)
		}
	}

	e.logger.Info().
		Int("tasks_completed", len(results)).
		Msg("Parallel task execution completed")

	return results, nil
}

// executeTask runs a single task
func (e *ParallelExecutor) executeTask(ctx context.Context, task models.WorkflowTask, execCtx *ExecutionContext) (*TaskResult, error) {
	result := &TaskResult{
		TaskID:    task.ID,
		StartedAt: time.Now(),
	}

	e.logger.Debug().
		Str("task_id", task.ID).
		Str("task_type", task.Type).
		Msg("Executing task")

	// Apply timeout if specified
	taskCtx := ctx
	if task.Timeout > 0 {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithTimeout(ctx, task.Timeout)
		defer cancel()
	}

	// Get task implementation
	taskImpl, err := e.taskRegistry.Get(task.Type)
	if err != nil {
		result.Error = err
		result.CompletedAt = time.Now()
		return result, err
	}

	// Prepare inputs with execution context
	inputs := make(map[string]interface{})
	for k, v := range task.Inputs {
		inputs[k] = v
	}

	// Add allowed commands to inputs for shell tasks
	if task.Type == "shell_cmd" {
		inputs["allowed_commands"] = execCtx.Config.AllowedCommands
	}

	// Substitute references to previous task outputs
	inputs = e.resolveInputReferences(inputs, execCtx)

	// Execute task
	output, err := taskImpl.Execute(taskCtx, inputs)
	result.Output = output
	result.Error = err
	result.CompletedAt = time.Now()

	if err != nil {
		e.logger.Error().
			Err(err).
			Str("task_id", task.ID).
			Str("task_type", task.Type).
			Msg("Task execution failed")
		return result, err
	}

	// Store output in execution context for dependent tasks
	if len(task.Outputs) > 0 {
		for _, outputKey := range task.Outputs {
			execCtx.Set(outputKey, output)
		}
	}

	e.logger.Debug().
		Str("task_id", task.ID).
		Dur("duration", result.CompletedAt.Sub(result.StartedAt)).
		Msg("Task completed successfully")

	return result, nil
}

// resolveInputReferences replaces references to previous task outputs
func (e *ParallelExecutor) resolveInputReferences(inputs map[string]interface{}, execCtx *ExecutionContext) map[string]interface{} {
	resolved := make(map[string]interface{})

	for key, value := range inputs {
		// Check if value is a reference to execution context (e.g., "${task_id}")
		if strVal, ok := value.(string); ok {
			if len(strVal) > 3 && strVal[:2] == "${" && strVal[len(strVal)-1:] == "}" {
				refKey := strVal[2 : len(strVal)-1]
				if refValue, exists := execCtx.Get(refKey); exists {
					resolved[key] = refValue
					continue
				}
			}
		}

		resolved[key] = value
	}

	return resolved
}

// computeLevels computes execution levels for parallel execution
func (e *ParallelExecutor) computeLevels(graph *DAG, ordering []string) [][]string {
	// Compute the level of each task based on its dependencies
	levels := make(map[string]int)

	for _, taskID := range ordering {
		maxDepLevel := -1

		// Get task dependencies
		taskDef, _ := graph.GetNode(taskID)
		task := taskDef.(models.WorkflowTask)

		for _, depID := range task.Dependencies {
			if depLevel, exists := levels[depID]; exists {
				if depLevel > maxDepLevel {
					maxDepLevel = depLevel
				}
			}
		}

		levels[taskID] = maxDepLevel + 1
	}

	// Group tasks by level
	maxLevel := 0
	for _, level := range levels {
		if level > maxLevel {
			maxLevel = level
		}
	}

	levelGroups := make([][]string, maxLevel+1)
	for taskID, level := range levels {
		levelGroups[level] = append(levelGroups[level], taskID)
	}

	return levelGroups
}

// DAG represents a Directed Acyclic Graph for task dependencies
type DAG struct {
	nodes map[string]interface{} // taskID -> task definition
	edges map[string][]string    // taskID -> list of dependent task IDs
	mu    sync.RWMutex
}

// NewDAG creates a new DAG
func NewDAG() *DAG {
	return &DAG{
		nodes: make(map[string]interface{}),
		edges: make(map[string][]string),
	}
}

// AddNode adds a node to the graph
func (d *DAG) AddNode(id string, data interface{}) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.nodes[id]; exists {
		return fmt.Errorf("node already exists: %s", id)
	}

	d.nodes[id] = data
	d.edges[id] = []string{}
	return nil
}

// AddEdge adds a directed edge from source to target
func (d *DAG) AddEdge(from, to string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.nodes[from]; !exists {
		return fmt.Errorf("source node does not exist: %s", from)
	}

	if _, exists := d.nodes[to]; !exists {
		return fmt.Errorf("target node does not exist: %s", to)
	}

	// Add edge
	d.edges[from] = append(d.edges[from], to)
	return nil
}

// GetNode retrieves a node by ID
func (d *DAG) GetNode(id string) (interface{}, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	node, exists := d.nodes[id]
	return node, exists
}

// HasCycle detects if the graph has a cycle using DFS
func (d *DAG) HasCycle() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(node string) bool
	hasCycle = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range d.edges[node] {
			if !visited[neighbor] {
				if hasCycle(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}

		recStack[node] = false
		return false
	}

	for node := range d.nodes {
		if !visited[node] {
			if hasCycle(node) {
				return true
			}
		}
	}

	return false
}

// TopologicalSort returns a topological ordering of nodes
func (d *DAG) TopologicalSort() ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.HasCycle() {
		return nil, fmt.Errorf("graph contains a cycle")
	}

	visited := make(map[string]bool)
	stack := []string{}

	var visit func(node string)
	visit = func(node string) {
		if visited[node] {
			return
		}

		visited[node] = true

		for _, neighbor := range d.edges[node] {
			visit(neighbor)
		}

		stack = append([]string{node}, stack...)
	}

	for node := range d.nodes {
		visit(node)
	}

	return stack, nil
}

// GetDependencies returns the dependencies of a node
func (d *DAG) GetDependencies(nodeID string) []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Find all nodes that have an edge to this node
	deps := []string{}
	for from, targets := range d.edges {
		for _, to := range targets {
			if to == nodeID {
				deps = append(deps, from)
			}
		}
	}

	return deps
}

// GetDependents returns the dependents of a node
func (d *DAG) GetDependents(nodeID string) []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.edges[nodeID]
}
