package unit

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dshills/gocreator/pkg/langgraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapState_BasicOperations(t *testing.T) {
	state := langgraph.NewMapState()

	// Test Set and Get
	state.Set("key1", "value1")
	val, ok := state.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	// Test non-existent key
	_, ok = state.Get("nonexistent")
	assert.False(t, ok)

	// Test Delete
	state.Delete("key1")
	_, ok = state.Get("key1")
	assert.False(t, ok)
}

func TestMapState_TypedGetters(t *testing.T) {
	state := langgraph.NewMapState()

	// Test GetString
	state.Set("str", "hello")
	str, ok := state.GetString("str")
	assert.True(t, ok)
	assert.Equal(t, "hello", str)

	// Test GetInt
	state.Set("int", 42)
	intVal, ok := state.GetInt("int")
	assert.True(t, ok)
	assert.Equal(t, 42, intVal)

	// Test GetBool
	state.Set("bool", true)
	boolVal, ok := state.GetBool("bool")
	assert.True(t, ok)
	assert.True(t, boolVal)

	// Test GetSlice
	slice := []interface{}{"a", "b", "c"}
	state.Set("slice", slice)
	sliceVal, ok := state.GetSlice("slice")
	assert.True(t, ok)
	assert.Equal(t, slice, sliceVal)

	// Test GetMap
	m := map[string]interface{}{"nested": "value"}
	state.Set("map", m)
	mapVal, ok := state.GetMap("map")
	assert.True(t, ok)
	assert.Equal(t, m, mapVal)
}

func TestMapState_Keys(t *testing.T) {
	state := langgraph.NewMapState()

	state.Set("key1", "value1")
	state.Set("key2", "value2")
	state.Set("key3", "value3")

	keys := state.Keys()
	assert.Equal(t, 3, len(keys))
	assert.Contains(t, keys, "key1")
	assert.Contains(t, keys, "key2")
	assert.Contains(t, keys, "key3")
}

func TestMapState_Clone(t *testing.T) {
	original := langgraph.NewMapState()
	original.Set("key1", "value1")
	original.Set("key2", 42)

	cloned, err := original.Clone()
	require.NoError(t, err)

	// Verify cloned state has same values
	val, ok := cloned.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	// Modify cloned state
	cloned.Set("key3", "value3")

	// Verify original is unchanged
	_, ok = original.Get("key3")
	assert.False(t, ok)
}

func TestMapState_JSON(t *testing.T) {
	state := langgraph.NewMapState()
	state.Set("key1", "value1")
	state.Set("key2", 42)

	// Test ToJSON
	data, err := state.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Test FromJSON
	newState := langgraph.NewMapState()
	err = newState.FromJSON(data)
	require.NoError(t, err)

	val, ok := newState.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)
}

func TestBasicNode_Execute(t *testing.T) {
	executeCalled := false
	fn := func(ctx context.Context, state langgraph.State) (langgraph.State, error) {
		executeCalled = true
		state.Set("result", "success")
		return state, nil
	}

	node := langgraph.NewBasicNode("test_node", fn, []string{}, "Test node")

	assert.Equal(t, "test_node", node.ID())
	assert.Equal(t, "Test node", node.Description())
	assert.Empty(t, node.Dependencies())

	state := langgraph.NewMapState()
	ctx := context.Background()

	newState, err := node.Execute(ctx, state)
	require.NoError(t, err)
	assert.True(t, executeCalled)

	val, ok := newState.Get("result")
	assert.True(t, ok)
	assert.Equal(t, "success", val)
}

func TestConditionalNode_Execute(t *testing.T) {
	tests := []struct {
		name          string
		condition     func(state langgraph.State) bool
		expectExecute bool
	}{
		{
			name: "condition true",
			condition: func(state langgraph.State) bool {
				return true
			},
			expectExecute: true,
		},
		{
			name: "condition false",
			condition: func(state langgraph.State) bool {
				return false
			},
			expectExecute: false,
		},
		{
			name:          "nil condition",
			condition:     nil,
			expectExecute: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executeCalled := false
			fn := func(ctx context.Context, state langgraph.State) (langgraph.State, error) {
				executeCalled = true
				state.Set("executed", true)
				return state, nil
			}

			node := langgraph.NewConditionalNode("cond_node", fn, []string{}, "Conditional node", tt.condition)

			state := langgraph.NewMapState()
			ctx := context.Background()

			newState, err := node.Execute(ctx, state)
			require.NoError(t, err)
			assert.Equal(t, tt.expectExecute, executeCalled)

			if tt.expectExecute {
				val, ok := newState.Get("executed")
				assert.True(t, ok)
				assert.True(t, val.(bool))
			}
		})
	}
}

func TestGraph_AddNode(t *testing.T) {
	graph := langgraph.NewGraph("test_graph", "start", "end")

	fn := func(ctx context.Context, state langgraph.State) (langgraph.State, error) {
		return state, nil
	}

	node1 := langgraph.NewBasicNode("node1", fn, []string{}, "Node 1")
	err := graph.AddNode(node1)
	require.NoError(t, err)

	// Try adding duplicate
	err = graph.AddNode(node1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestGraph_Validate(t *testing.T) {
	tests := []struct {
		name        string
		setupGraph  func() *langgraph.Graph
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid graph",
			setupGraph: func() *langgraph.Graph {
				graph := langgraph.NewGraph("test", "start", "end")
				fn := func(ctx context.Context, state langgraph.State) (langgraph.State, error) {
					return state, nil
				}
				graph.AddNode(langgraph.NewBasicNode("start", fn, []string{}, "Start"))
				graph.AddNode(langgraph.NewBasicNode("middle", fn, []string{"start"}, "Middle"))
				graph.AddNode(langgraph.NewBasicNode("end", fn, []string{"middle"}, "End"))
				return graph
			},
			expectError: false,
		},
		{
			name: "missing start node",
			setupGraph: func() *langgraph.Graph {
				graph := langgraph.NewGraph("test", "start", "end")
				fn := func(ctx context.Context, state langgraph.State) (langgraph.State, error) {
					return state, nil
				}
				graph.AddNode(langgraph.NewBasicNode("end", fn, []string{}, "End"))
				return graph
			},
			expectError: true,
			errorMsg:    "start node",
		},
		{
			name: "missing end node",
			setupGraph: func() *langgraph.Graph {
				graph := langgraph.NewGraph("test", "start", "end")
				fn := func(ctx context.Context, state langgraph.State) (langgraph.State, error) {
					return state, nil
				}
				graph.AddNode(langgraph.NewBasicNode("start", fn, []string{}, "Start"))
				return graph
			},
			expectError: true,
			errorMsg:    "end node",
		},
		{
			name: "missing dependency",
			setupGraph: func() *langgraph.Graph {
				graph := langgraph.NewGraph("test", "start", "end")
				fn := func(ctx context.Context, state langgraph.State) (langgraph.State, error) {
					return state, nil
				}
				graph.AddNode(langgraph.NewBasicNode("start", fn, []string{}, "Start"))
				graph.AddNode(langgraph.NewBasicNode("end", fn, []string{"nonexistent"}, "End"))
				return graph
			},
			expectError: true,
			errorMsg:    "non-existent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := tt.setupGraph()
			err := graph.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGraph_Execute(t *testing.T) {
	graph := langgraph.NewGraph("test_graph", "start", "end")

	// Create nodes
	startFn := func(ctx context.Context, state langgraph.State) (langgraph.State, error) {
		state.Set("step", "start")
		return state, nil
	}

	middleFn := func(ctx context.Context, state langgraph.State) (langgraph.State, error) {
		state.Set("step", "middle")
		return state, nil
	}

	endFn := func(ctx context.Context, state langgraph.State) (langgraph.State, error) {
		state.Set("step", "end")
		state.Set("completed", true)
		return state, nil
	}

	// Add nodes
	graph.AddNode(langgraph.NewBasicNode("start", startFn, []string{}, "Start"))
	graph.AddNode(langgraph.NewBasicNode("middle", middleFn, []string{"start"}, "Middle"))
	graph.AddNode(langgraph.NewBasicNode("end", endFn, []string{"middle"}, "End"))

	// Execute
	initialState := langgraph.NewMapState()
	ctx := context.Background()

	finalState, err := graph.Execute(ctx, initialState)
	require.NoError(t, err)

	// Verify execution
	step, ok := finalState.Get("step")
	assert.True(t, ok)
	assert.Equal(t, "end", step)

	completed, ok := finalState.Get("completed")
	assert.True(t, ok)
	assert.True(t, completed.(bool))
}

func TestGraph_ExecuteWithCancellation(t *testing.T) {
	graph := langgraph.NewGraph("test_graph", "start", "end")

	// Create a long-running node
	slowFn := func(ctx context.Context, state langgraph.State) (langgraph.State, error) {
		select {
		case <-time.After(5 * time.Second):
			return state, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	graph.AddNode(langgraph.NewBasicNode("start", slowFn, []string{}, "Start"))
	graph.AddNode(langgraph.NewBasicNode("end", slowFn, []string{"start"}, "End"))

	// Create cancellable context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	initialState := langgraph.NewMapState()

	_, err := graph.Execute(ctx, initialState)
	assert.Error(t, err)
	// Accept both "cancelled" and "deadline exceeded" as valid cancellation errors
	errStr := err.Error()
	assert.True(t,
		strings.Contains(errStr, "cancelled") || strings.Contains(errStr, "deadline exceeded"),
		"expected cancellation error, got: %s", errStr,
	)
}

func TestCheckpoint_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := langgraph.NewFileCheckpointManager(tmpDir)
	require.NoError(t, err)

	// Create execution context
	ctx := langgraph.ExecutionContext{
		GraphID:        "test_graph",
		ExecutionID:    "exec_1",
		CurrentNode:    "node_2",
		CompletedNodes: []string{"node_1", "node_2"},
	}

	// Create state
	state := langgraph.NewMapState()
	state.Set("key1", "value1")
	state.Set("key2", 42)

	// Save checkpoint
	err = mgr.Save(ctx, state)
	require.NoError(t, err)

	// Load checkpoint
	checkpoint, err := mgr.Load("test_graph")
	require.NoError(t, err)
	assert.NotNil(t, checkpoint)
	assert.Equal(t, "test_graph", checkpoint.GraphID)
	assert.Equal(t, "node_2", checkpoint.LastCompletedNode)
	assert.Equal(t, 2, len(checkpoint.CompletedNodes))

	// Recover state
	recoveredState, err := langgraph.RecoverState(checkpoint)
	require.NoError(t, err)

	val, ok := recoveredState.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)
}

func TestCheckpoint_DeleteAll(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := langgraph.NewFileCheckpointManager(tmpDir)
	require.NoError(t, err)

	// Save multiple checkpoints
	ctx1 := langgraph.ExecutionContext{
		GraphID:     "test_graph",
		CurrentNode: "node_1",
	}
	state1 := langgraph.NewMapState()
	mgr.Save(ctx1, state1)

	ctx2 := langgraph.ExecutionContext{
		GraphID:     "test_graph",
		CurrentNode: "node_2",
	}
	state2 := langgraph.NewMapState()
	mgr.Save(ctx2, state2)

	// List checkpoints
	checkpoints, err := mgr.List("test_graph")
	require.NoError(t, err)
	assert.Equal(t, 2, len(checkpoints))

	// Delete all
	err = mgr.DeleteAll("test_graph")
	require.NoError(t, err)

	// Verify deleted
	checkpoints, err = mgr.List("test_graph")
	require.NoError(t, err)
	assert.Equal(t, 0, len(checkpoints))
}
