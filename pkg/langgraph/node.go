package langgraph

import (
	"context"
	"fmt"
)

// NodeFunc is the function signature for a graph node
// It takes a context and state, performs operations, and returns updated state or error
type NodeFunc func(ctx context.Context, state State) (State, error)

// Node represents a node in the execution graph
type Node interface {
	// ID returns the unique identifier for this node
	ID() string

	// Execute runs the node's logic with the given state
	Execute(ctx context.Context, state State) (State, error)

	// Dependencies returns the IDs of nodes that must complete before this node
	Dependencies() []string

	// Description returns a human-readable description of what this node does
	Description() string
}

// BasicNode is a simple implementation of Node using a function
type BasicNode struct {
	id           string
	fn           NodeFunc
	dependencies []string
	description  string
}

// NewBasicNode creates a new BasicNode
func NewBasicNode(id string, fn NodeFunc, dependencies []string, description string) *BasicNode {
	return &BasicNode{
		id:           id,
		fn:           fn,
		dependencies: dependencies,
		description:  description,
	}
}

// ID returns the node's ID
func (n *BasicNode) ID() string {
	return n.id
}

// Execute runs the node's function
func (n *BasicNode) Execute(ctx context.Context, state State) (State, error) {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("node %s cancelled: %w", n.id, ctx.Err())
	default:
	}

	// Execute the node function
	return n.fn(ctx, state)
}

// Dependencies returns the node's dependencies
func (n *BasicNode) Dependencies() []string {
	return n.dependencies
}

// Description returns the node's description
func (n *BasicNode) Description() string {
	return n.description
}

// ConditionalNode represents a node with conditional routing
type ConditionalNode struct {
	id           string
	fn           NodeFunc
	dependencies []string
	description  string
	condition    func(state State) bool
}

// NewConditionalNode creates a new ConditionalNode
func NewConditionalNode(
	id string,
	fn NodeFunc,
	dependencies []string,
	description string,
	condition func(state State) bool,
) *ConditionalNode {
	return &ConditionalNode{
		id:           id,
		fn:           fn,
		dependencies: dependencies,
		description:  description,
		condition:    condition,
	}
}

// ID returns the node's ID
func (n *ConditionalNode) ID() string {
	return n.id
}

// Execute runs the node's function if condition is met
func (n *ConditionalNode) Execute(ctx context.Context, state State) (State, error) {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("node %s cancelled: %w", n.id, ctx.Err())
	default:
	}

	// Check condition
	if n.condition != nil && !n.condition(state) {
		// Condition not met, return state unchanged
		return state, nil
	}

	// Execute the node function
	return n.fn(ctx, state)
}

// Dependencies returns the node's dependencies
func (n *ConditionalNode) Dependencies() []string {
	return n.dependencies
}

// Description returns the node's description
func (n *ConditionalNode) Description() string {
	return n.description
}

// ShouldExecute checks if the condition is met
func (n *ConditionalNode) ShouldExecute(state State) bool {
	if n.condition == nil {
		return true
	}
	return n.condition(state)
}

// ParallelNode represents a node that can execute in parallel with others
type ParallelNode struct {
	id           string
	fn           NodeFunc
	dependencies []string
	description  string
	parallel     bool
}

// NewParallelNode creates a new ParallelNode
func NewParallelNode(
	id string,
	fn NodeFunc,
	dependencies []string,
	description string,
	parallel bool,
) *ParallelNode {
	return &ParallelNode{
		id:           id,
		fn:           fn,
		dependencies: dependencies,
		description:  description,
		parallel:     parallel,
	}
}

// ID returns the node's ID
func (n *ParallelNode) ID() string {
	return n.id
}

// Execute runs the node's function
func (n *ParallelNode) Execute(ctx context.Context, state State) (State, error) {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("node %s cancelled: %w", n.id, ctx.Err())
	default:
	}

	// Execute the node function
	return n.fn(ctx, state)
}

// Dependencies returns the node's dependencies
func (n *ParallelNode) Dependencies() []string {
	return n.dependencies
}

// Description returns the node's description
func (n *ParallelNode) Description() string {
	return n.description
}

// CanRunInParallel returns whether this node can run in parallel
func (n *ParallelNode) CanRunInParallel() bool {
	return n.parallel
}

// NodeResult captures the result of a node execution
type NodeResult struct {
	NodeID    string
	State     State
	Error     error
	Skipped   bool
	StartTime int64
	EndTime   int64
}
