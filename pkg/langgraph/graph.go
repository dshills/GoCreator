package langgraph

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// ExecutionContext contains context about the current execution
type ExecutionContext struct {
	GraphID        string
	ExecutionID    string
	CurrentNode    string
	CompletedNodes []string
	StartTime      time.Time
}

// Graph represents a directed acyclic graph of nodes to execute
type Graph struct {
	id               string
	nodes            map[string]Node
	startNode        string
	endNode          string
	checkpointMgr    CheckpointManager
	enableCheckpoint bool
	maxParallel      int
}

// GraphOption is a functional option for Graph configuration
type GraphOption func(*Graph)

// WithCheckpointing enables checkpointing for the graph
func WithCheckpointing(mgr CheckpointManager) GraphOption {
	return func(g *Graph) {
		g.checkpointMgr = mgr
		g.enableCheckpoint = true
	}
}

// WithMaxParallel sets the maximum number of parallel node executions
func WithMaxParallel(max int) GraphOption {
	return func(g *Graph) {
		g.maxParallel = max
	}
}

// NewGraph creates a new execution graph
func NewGraph(id string, startNode string, endNode string, opts ...GraphOption) *Graph {
	g := &Graph{
		id:          id,
		nodes:       make(map[string]Node),
		startNode:   startNode,
		endNode:     endNode,
		maxParallel: 10, // Default
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// AddNode adds a node to the graph
func (g *Graph) AddNode(node Node) error {
	if _, exists := g.nodes[node.ID()]; exists {
		return fmt.Errorf("node %s already exists", node.ID())
	}

	g.nodes[node.ID()] = node
	return nil
}

// Validate validates the graph structure
func (g *Graph) Validate() error {
	// Check start node exists
	if _, ok := g.nodes[g.startNode]; !ok {
		return fmt.Errorf("start node %s not found", g.startNode)
	}

	// Check end node exists
	if _, ok := g.nodes[g.endNode]; !ok {
		return fmt.Errorf("end node %s not found", g.endNode)
	}

	// Validate dependencies exist
	for _, node := range g.nodes {
		for _, dep := range node.Dependencies() {
			if _, ok := g.nodes[dep]; !ok {
				return fmt.Errorf("node %s depends on non-existent node %s", node.ID(), dep)
			}
		}
	}

	// Check for cycles
	if g.hasCycles() {
		return fmt.Errorf("graph contains cycles")
	}

	return nil
}

// hasCycles detects cycles in the graph using DFS
func (g *Graph) hasCycles() bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(nodeID string) bool
	hasCycle = func(nodeID string) bool {
		visited[nodeID] = true
		recStack[nodeID] = true

		node := g.nodes[nodeID]
		for _, dep := range node.Dependencies() {
			if !visited[dep] {
				if hasCycle(dep) {
					return true
				}
			} else if recStack[dep] {
				return true
			}
		}

		recStack[nodeID] = false
		return false
	}

	for nodeID := range g.nodes {
		if !visited[nodeID] {
			if hasCycle(nodeID) {
				return true
			}
		}
	}

	return false
}

// Execute runs the graph with the given initial state
func (g *Graph) Execute(ctx context.Context, initialState State) (State, error) {
	// Validate graph
	if err := g.Validate(); err != nil {
		return nil, fmt.Errorf("graph validation failed: %w", err)
	}

	// Create execution context
	execCtx := ExecutionContext{
		GraphID:        g.id,
		ExecutionID:    fmt.Sprintf("%s_%d", g.id, time.Now().Unix()),
		CurrentNode:    g.startNode,
		CompletedNodes: []string{},
		StartTime:      time.Now(),
	}

	log.Info().
		Str("graph_id", g.id).
		Str("execution_id", execCtx.ExecutionID).
		Str("start_node", g.startNode).
		Str("end_node", g.endNode).
		Msg("Starting graph execution")

	// Execute nodes in topological order
	state := initialState
	executionOrder, err := g.topologicalSort()
	if err != nil {
		return nil, fmt.Errorf("failed to determine execution order: %w", err)
	}

	// Execute nodes
	for _, batch := range executionOrder {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("graph execution cancelled: %w", ctx.Err())
		default:
		}

		// Execute batch (potentially in parallel)
		newState, err := g.executeBatch(ctx, batch, state, &execCtx)
		if err != nil {
			return nil, fmt.Errorf("batch execution failed: %w", err)
		}
		state = newState

		// Save checkpoint after each batch if enabled
		if g.enableCheckpoint && g.checkpointMgr != nil {
			if err := g.checkpointMgr.Save(execCtx, state); err != nil {
				log.Warn().
					Err(err).
					Str("graph_id", g.id).
					Msg("Failed to save checkpoint")
			}
		}
	}

	duration := time.Since(execCtx.StartTime)
	log.Info().
		Str("graph_id", g.id).
		Str("execution_id", execCtx.ExecutionID).
		Dur("duration", duration).
		Int("nodes_executed", len(execCtx.CompletedNodes)).
		Msg("Graph execution completed")

	return state, nil
}

// executeBatch executes a batch of nodes, potentially in parallel
func (g *Graph) executeBatch(
	ctx context.Context,
	nodeIDs []string,
	state State,
	execCtx *ExecutionContext,
) (State, error) {
	// If only one node, execute directly
	if len(nodeIDs) == 1 {
		return g.executeNode(ctx, nodeIDs[0], state, execCtx)
	}

	// Check if nodes can run in parallel
	canParallel := true
	for _, nodeID := range nodeIDs {
		node := g.nodes[nodeID]
		if pNode, ok := node.(*ParallelNode); ok {
			if !pNode.CanRunInParallel() {
				canParallel = false
				break
			}
		}
	}

	// If can't run in parallel, execute sequentially
	if !canParallel {
		currentState := state
		for _, nodeID := range nodeIDs {
			newState, err := g.executeNode(ctx, nodeID, currentState, execCtx)
			if err != nil {
				return nil, err
			}
			currentState = newState
		}
		return currentState, nil
	}

	// Execute in parallel with errgroup
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(g.maxParallel)

	// Collect results
	var mu sync.Mutex
	results := make(map[string]State)

	for _, nodeID := range nodeIDs {
		nodeID := nodeID // Capture for goroutine
		eg.Go(func() error {
			// Clone state for parallel execution
			clonedState, err := state.Clone()
			if err != nil {
				return fmt.Errorf("failed to clone state for node %s: %w", nodeID, err)
			}

			newState, err := g.executeNode(egCtx, nodeID, clonedState, execCtx)
			if err != nil {
				return err
			}

			mu.Lock()
			results[nodeID] = newState
			mu.Unlock()

			return nil
		})
	}

	// Wait for all nodes to complete
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	// Merge results back into state
	finalState := state
	for _, nodeState := range results {
		if err := finalState.(*MapState).Merge(nodeState); err != nil {
			return nil, fmt.Errorf("failed to merge parallel results: %w", err)
		}
	}

	return finalState, nil
}

// executeNode executes a single node
func (g *Graph) executeNode(
	ctx context.Context,
	nodeID string,
	state State,
	execCtx *ExecutionContext,
) (State, error) {
	node, ok := g.nodes[nodeID]
	if !ok {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}

	execCtx.CurrentNode = nodeID

	log.Debug().
		Str("graph_id", g.id).
		Str("node_id", nodeID).
		Str("description", node.Description()).
		Msg("Executing node")

	startTime := time.Now()

	// Execute node
	newState, err := node.Execute(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("node %s execution failed: %w", nodeID, err)
	}

	duration := time.Since(startTime)

	log.Debug().
		Str("graph_id", g.id).
		Str("node_id", nodeID).
		Dur("duration", duration).
		Msg("Node execution completed")

	// Mark node as completed
	execCtx.CompletedNodes = append(execCtx.CompletedNodes, nodeID)

	return newState, nil
}

// topologicalSort returns nodes grouped by execution level (batch)
func (g *Graph) topologicalSort() ([][]string, error) {
	// Calculate in-degrees
	inDegree := make(map[string]int)
	for nodeID := range g.nodes {
		inDegree[nodeID] = 0
	}
	for _, node := range g.nodes {
		for range node.Dependencies() {
			inDegree[node.ID()]++
		}
	}

	// Find nodes with no dependencies (level 0)
	var result [][]string
	queue := []string{}
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	visited := make(map[string]bool)

	for len(queue) > 0 {
		// Current batch
		currentBatch := make([]string, len(queue))
		copy(currentBatch, queue)
		result = append(result, currentBatch)

		// Clear queue for next level
		queue = []string{}

		// Process current batch
		for _, nodeID := range currentBatch {
			visited[nodeID] = true

			// Find nodes that depend on this one
			for _, node := range g.nodes {
				// Check if this node depends on the current node
				hasDep := false
				for _, dep := range node.Dependencies() {
					if dep == nodeID {
						hasDep = true
						break
					}
				}

				if hasDep {
					inDegree[node.ID()]--
					// If all dependencies satisfied, add to next batch
					if inDegree[node.ID()] == 0 && !visited[node.ID()] {
						queue = append(queue, node.ID())
					}
				}
			}
		}
	}

	// Check if all nodes were visited
	if len(visited) != len(g.nodes) {
		return nil, fmt.Errorf("topological sort failed: not all nodes visited (possible cycle)")
	}

	return result, nil
}

// Resume resumes execution from a checkpoint
func (g *Graph) Resume(ctx context.Context, checkpointID string) (State, error) {
	if !g.enableCheckpoint || g.checkpointMgr == nil {
		return nil, fmt.Errorf("checkpointing not enabled")
	}

	// Load checkpoint
	checkpoint, err := g.checkpointMgr.Load(g.id)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// Recover state
	state, err := RecoverState(checkpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to recover state: %w", err)
	}

	log.Info().
		Str("graph_id", g.id).
		Str("checkpoint_id", checkpoint.ID).
		Str("last_node", checkpoint.LastCompletedNode).
		Msg("Resuming from checkpoint")

	// Continue execution from last completed node
	// This is a simplified implementation - a full implementation would
	// need to carefully track which nodes to skip
	return g.Execute(ctx, state)
}
