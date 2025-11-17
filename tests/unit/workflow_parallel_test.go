package unit

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/internal/workflow"
	"github.com/dshills/gocreator/pkg/fsops"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDAG_AddNode(t *testing.T) {
	dag := workflow.NewDAG()

	t.Run("add node successfully", func(t *testing.T) {
		err := dag.AddNode("node1", "data1")
		assert.NoError(t, err)

		node, exists := dag.GetNode("node1")
		assert.True(t, exists)
		assert.Equal(t, "data1", node)
	})

	t.Run("add duplicate node", func(t *testing.T) {
		err := dag.AddNode("node1", "data2")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func TestDAG_AddEdge(t *testing.T) {
	dag := workflow.NewDAG()
	dag.AddNode("node1", "data1")
	dag.AddNode("node2", "data2")

	t.Run("add edge successfully", func(t *testing.T) {
		err := dag.AddEdge("node1", "node2")
		assert.NoError(t, err)
	})

	t.Run("add edge with nonexistent source", func(t *testing.T) {
		err := dag.AddEdge("nonexistent", "node2")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("add edge with nonexistent target", func(t *testing.T) {
		err := dag.AddEdge("node1", "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})
}

func TestDAG_HasCycle(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *workflow.DAG
		hasCycle bool
	}{
		{
			name: "no cycle - linear",
			setup: func() *workflow.DAG {
				dag := workflow.NewDAG()
				dag.AddNode("A", "dataA")
				dag.AddNode("B", "dataB")
				dag.AddNode("C", "dataC")
				dag.AddEdge("A", "B")
				dag.AddEdge("B", "C")
				return dag
			},
			hasCycle: false,
		},
		{
			name: "no cycle - diamond",
			setup: func() *workflow.DAG {
				dag := workflow.NewDAG()
				dag.AddNode("A", "dataA")
				dag.AddNode("B", "dataB")
				dag.AddNode("C", "dataC")
				dag.AddNode("D", "dataD")
				dag.AddEdge("A", "B")
				dag.AddEdge("A", "C")
				dag.AddEdge("B", "D")
				dag.AddEdge("C", "D")
				return dag
			},
			hasCycle: false,
		},
		{
			name: "has cycle - simple",
			setup: func() *workflow.DAG {
				dag := workflow.NewDAG()
				dag.AddNode("A", "dataA")
				dag.AddNode("B", "dataB")
				dag.AddEdge("A", "B")
				dag.AddEdge("B", "A")
				return dag
			},
			hasCycle: true,
		},
		{
			name: "has cycle - complex",
			setup: func() *workflow.DAG {
				dag := workflow.NewDAG()
				dag.AddNode("A", "dataA")
				dag.AddNode("B", "dataB")
				dag.AddNode("C", "dataC")
				dag.AddNode("D", "dataD")
				dag.AddEdge("A", "B")
				dag.AddEdge("B", "C")
				dag.AddEdge("C", "D")
				dag.AddEdge("D", "B")
				return dag
			},
			hasCycle: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dag := tt.setup()
			assert.Equal(t, tt.hasCycle, dag.HasCycle())
		})
	}
}

func TestDAG_TopologicalSort(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *workflow.DAG
		wantErr bool
		check   func(t *testing.T, order []string)
	}{
		{
			name: "simple linear order",
			setup: func() *workflow.DAG {
				dag := workflow.NewDAG()
				dag.AddNode("A", "dataA")
				dag.AddNode("B", "dataB")
				dag.AddNode("C", "dataC")
				dag.AddEdge("A", "B")
				dag.AddEdge("B", "C")
				return dag
			},
			wantErr: false,
			check: func(t *testing.T, order []string) {
				assert.Len(t, order, 3)
				// A must come before B, B must come before C
				aIdx := indexOf(order, "A")
				bIdx := indexOf(order, "B")
				cIdx := indexOf(order, "C")
				assert.True(t, aIdx < bIdx)
				assert.True(t, bIdx < cIdx)
			},
		},
		{
			name: "diamond dependency",
			setup: func() *workflow.DAG {
				dag := workflow.NewDAG()
				dag.AddNode("A", "dataA")
				dag.AddNode("B", "dataB")
				dag.AddNode("C", "dataC")
				dag.AddNode("D", "dataD")
				dag.AddEdge("A", "B")
				dag.AddEdge("A", "C")
				dag.AddEdge("B", "D")
				dag.AddEdge("C", "D")
				return dag
			},
			wantErr: false,
			check: func(t *testing.T, order []string) {
				assert.Len(t, order, 4)
				// A must come before B and C
				// B and C must come before D
				aIdx := indexOf(order, "A")
				bIdx := indexOf(order, "B")
				cIdx := indexOf(order, "C")
				dIdx := indexOf(order, "D")
				assert.True(t, aIdx < bIdx)
				assert.True(t, aIdx < cIdx)
				assert.True(t, bIdx < dIdx)
				assert.True(t, cIdx < dIdx)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dag := tt.setup()
			order, err := dag.TopologicalSort()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.check != nil {
					tt.check(t, order)
				}
			}
		})
	}
}

func TestParallelExecutor_Execute(t *testing.T) {
	tests := []struct {
		name    string
		tasks   []models.WorkflowTask
		wantErr bool
		check   func(t *testing.T, results map[string]*workflow.TaskResult)
	}{
		{
			name: "independent tasks execute in parallel",
			tasks: []models.WorkflowTask{
				{
					ID:   "task1",
					Type: "file_op",
					Inputs: map[string]interface{}{
						"operation": "write",
						"path":      "file1.txt",
						"content":   "Content 1",
					},
				},
				{
					ID:   "task2",
					Type: "file_op",
					Inputs: map[string]interface{}{
						"operation": "write",
						"path":      "file2.txt",
						"content":   "Content 2",
					},
				},
				{
					ID:   "task3",
					Type: "file_op",
					Inputs: map[string]interface{}{
						"operation": "write",
						"path":      "file3.txt",
						"content":   "Content 3",
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, results map[string]*workflow.TaskResult) {
				assert.Len(t, results, 3)
				assert.NoError(t, results["task1"].Error)
				assert.NoError(t, results["task2"].Error)
				assert.NoError(t, results["task3"].Error)
			},
		},
		{
			name: "dependent tasks execute in order",
			tasks: []models.WorkflowTask{
				{
					ID:   "task1",
					Type: "file_op",
					Inputs: map[string]interface{}{
						"operation": "write",
						"path":      "step1.txt",
						"content":   "Step 1",
					},
					Outputs: []string{"step1_result"},
				},
				{
					ID:           "task2",
					Type:         "file_op",
					Dependencies: []string{"task1"},
					Inputs: map[string]interface{}{
						"operation": "write",
						"path":      "step2.txt",
						"content":   "Step 2",
					},
					Outputs: []string{"step2_result"},
				},
				{
					ID:           "task3",
					Type:         "file_op",
					Dependencies: []string{"task2"},
					Inputs: map[string]interface{}{
						"operation": "write",
						"path":      "step3.txt",
						"content":   "Step 3",
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, results map[string]*workflow.TaskResult) {
				assert.Len(t, results, 3)
				// Verify execution order by timestamps
				t1 := results["task1"].CompletedAt
				t2 := results["task2"].CompletedAt
				t3 := results["task3"].CompletedAt
				assert.True(t, t1.Before(t2) || t1.Equal(t2))
				assert.True(t, t2.Before(t3) || t2.Equal(t3))
			},
		},
		{
			name: "circular dependency detected",
			tasks: []models.WorkflowTask{
				{
					ID:           "task1",
					Type:         "file_op",
					Dependencies: []string{"task2"},
					Inputs: map[string]interface{}{
						"operation": "write",
						"path":      "file1.txt",
						"content":   "Content 1",
					},
				},
				{
					ID:           "task2",
					Type:         "file_op",
					Dependencies: []string{"task1"},
					Inputs: map[string]interface{}{
						"operation": "write",
						"path":      "file2.txt",
						"content":   "Content 2",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			// Create fsops
			fsOps, err := fsops.New(fsops.Config{
				RootDir: tmpDir,
			})
			require.NoError(t, err)

			// Create task registry
			logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
			taskRegistry := workflow.NewTaskRegistry(fsOps, logger)

			// Create parallel executor
			executor := workflow.NewParallelExecutor(workflow.ParallelExecutorConfig{
				MaxWorkers:   4,
				TaskRegistry: taskRegistry,
				Logger:       logger,
			})

			// Create execution context
			execCtx := &workflow.ExecutionContext{
				WorkflowID:  "test-workflow",
				ExecutionID: "test-execution",
				State:       make(map[string]interface{}),
				Config: models.WorkflowConfig{
					MaxParallel: 4,
				},
			}

			// Execute tasks
			results, err := executor.Execute(context.Background(), tt.tasks, execCtx)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.check != nil {
					tt.check(t, results)
				}
			}
		})
	}
}

func TestParallelExecutor_Timeout(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create fsops
	fsOps, err := fsops.New(fsops.Config{
		RootDir: tmpDir,
	})
	require.NoError(t, err)

	// Create task registry
	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	taskRegistry := workflow.NewTaskRegistry(fsOps, logger)

	// Create parallel executor
	executor := workflow.NewParallelExecutor(workflow.ParallelExecutorConfig{
		MaxWorkers:   4,
		TaskRegistry: taskRegistry,
		Logger:       logger,
	})

	// Create task with timeout
	tasks := []models.WorkflowTask{
		{
			ID:      "timeout_task",
			Type:    "shell_cmd",
			Timeout: 1 * time.Second,
			Inputs: map[string]interface{}{
				"cmd":              "sleep",
				"args":             []interface{}{"5"},
				"allowed_commands": []string{"sleep"},
			},
		},
	}

	// Create execution context
	execCtx := &workflow.ExecutionContext{
		WorkflowID:  "test-workflow",
		ExecutionID: "test-execution",
		State:       make(map[string]interface{}),
		Config: models.WorkflowConfig{
			MaxParallel:     4,
			AllowedCommands: []string{"sleep"},
		},
	}

	// Execute tasks (should timeout)
	ctx := context.Background()
	_, err = executor.Execute(ctx, tasks, execCtx)

	// On timeout, the task should fail
	assert.Error(t, err)
}

func TestParallelExecutor_MaxWorkers(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create fsops
	fsOps, err := fsops.New(fsops.Config{
		RootDir: tmpDir,
	})
	require.NoError(t, err)

	// Create task registry
	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	taskRegistry := workflow.NewTaskRegistry(fsOps, logger)

	// Create parallel executor with limited workers
	executor := workflow.NewParallelExecutor(workflow.ParallelExecutorConfig{
		MaxWorkers:   2,
		TaskRegistry: taskRegistry,
		Logger:       logger,
	})

	// Create many independent tasks
	tasks := []models.WorkflowTask{}
	for i := 0; i < 10; i++ {
		tasks = append(tasks, models.WorkflowTask{
			ID:   "task_" + string(rune('0'+i)),
			Type: "file_op",
			Inputs: map[string]interface{}{
				"operation": "write",
				"path":      "file_" + string(rune('0'+i)) + ".txt",
				"content":   "Content",
			},
		})
	}

	// Create execution context
	execCtx := &workflow.ExecutionContext{
		WorkflowID:  "test-workflow",
		ExecutionID: "test-execution",
		State:       make(map[string]interface{}),
		Config: models.WorkflowConfig{
			MaxParallel: 2,
		},
	}

	// Execute tasks
	results, err := executor.Execute(context.Background(), tasks, execCtx)
	require.NoError(t, err)

	// All tasks should complete despite limited workers
	assert.Len(t, results, 10)
	for _, result := range results {
		assert.NoError(t, result.Error)
	}
}

// Helper function
func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}
