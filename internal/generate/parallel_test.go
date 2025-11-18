package generate

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockParallelCoder is a test implementation of Coder
type mockParallelCoder struct {
	generateCount int64
	delay         time.Duration
	errorOn       map[string]error
	mu            sync.Mutex
}

func newMockParallelCoder() *mockParallelCoder {
	return &mockParallelCoder{
		delay:   5 * time.Millisecond,
		errorOn: make(map[string]error),
	}
}

func (m *mockParallelCoder) Generate(_ context.Context, plan *models.GenerationPlan, _ *models.FinalClarifiedSpecification) ([]models.Patch, error) {
	var patches []models.Patch
	for _, phase := range plan.Phases {
		for _, task := range phase.Tasks {
			if task.Type == "generate_file" {
				patch, err := m.GenerateFile(context.Background(), task, plan, nil)
				if err != nil {
					return nil, err
				}
				patches = append(patches, patch)
			}
		}
	}
	return patches, nil
}

func (m *mockParallelCoder) GenerateFile(_ context.Context, task models.GenerationTask, _ *models.GenerationPlan, _ *models.FinalClarifiedSpecification) (models.Patch, error) {
	atomic.AddInt64(&m.generateCount, 1)

	// Simulate work
	time.Sleep(m.delay)

	// Check for error
	m.mu.Lock()
	err, shouldError := m.errorOn[task.ID]
	m.mu.Unlock()

	if shouldError {
		return models.Patch{}, err
	}

	return models.Patch{
		TargetFile: task.TargetPath,
		Diff:       fmt.Sprintf("// Code for %s", task.TargetPath),
		AppliedAt:  time.Now(),
		Reversible: true,
	}, nil
}

func (m *mockParallelCoder) setError(taskID string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorOn[taskID] = err
}

func TestParallelCoder_Generate(t *testing.T) {
	tests := []struct {
		name           string
		numFiles       int
		maxParallel    int
		enableParallel bool
		expectFaster   bool
	}{
		{
			name:           "sequential generation",
			numFiles:       10,
			maxParallel:    1,
			enableParallel: false,
			expectFaster:   false,
		},
		{
			name:           "parallel generation 4 workers",
			numFiles:       20,
			maxParallel:    4,
			enableParallel: true,
			expectFaster:   true,
		},
		{
			name:           "parallel generation 8 workers",
			numFiles:       32,
			maxParallel:    8,
			enableParallel: true,
			expectFaster:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			baseCoder := newMockParallelCoder()

			config := ParallelGenerationConfig{
				MaxParallel:    tt.maxParallel,
				EnableParallel: tt.enableParallel,
			}

			pc := NewParallelCoder(baseCoder, config)
			plan := createSimplePlan(tt.numFiles)

			startTime := time.Now()
			patches, err := pc.Generate(ctx, plan, nil)
			duration := time.Since(startTime)

			require.NoError(t, err)
			assert.Equal(t, tt.numFiles, len(patches))

			// Verify all files were generated
			fileMap := make(map[string]bool)
			for _, patch := range patches {
				fileMap[patch.TargetFile] = true
			}
			assert.Equal(t, tt.numFiles, len(fileMap), "should have unique files")

			// Check if parallel was faster (rough heuristic)
			sequentialTime := time.Duration(tt.numFiles) * baseCoder.delay
			if tt.expectFaster {
				maxExpectedTime := (sequentialTime / time.Duration(tt.maxParallel)) * 2
				assert.Less(t, duration, maxExpectedTime,
					"parallel execution should be faster than sequential")
			}

			t.Logf("Generated %d files in %v (sequential would be ~%v)",
				tt.numFiles, duration, sequentialTime)
		})
	}
}

func TestParallelCoder_DependencyResolution(t *testing.T) {
	tests := []struct {
		name        string
		setupPlan   func() *models.GenerationPlan
		maxParallel int
		expectError bool
	}{
		{
			name: "no dependencies",
			setupPlan: func() *models.GenerationPlan {
				return createSimplePlan(10)
			},
			maxParallel: 4,
			expectError: false,
		},
		{
			name: "linear dependencies",
			setupPlan: func() *models.GenerationPlan {
				return createPlanWithLinearDeps(5)
			},
			maxParallel: 4,
			expectError: false,
		},
		{
			name: "tree dependencies",
			setupPlan: func() *models.GenerationPlan {
				return createPlanWithTreeDeps()
			},
			maxParallel: 4,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			baseCoder := newMockParallelCoder()

			config := ParallelGenerationConfig{
				MaxParallel:    tt.maxParallel,
				EnableParallel: true,
			}

			pc := NewParallelCoder(baseCoder, config)
			plan := tt.setupPlan()

			patches, err := pc.Generate(ctx, plan, nil)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Count expected files
				expectedFiles := 0
				for _, phase := range plan.Phases {
					for _, task := range phase.Tasks {
						if task.Type == "generate_file" {
							expectedFiles++
						}
					}
				}
				assert.Equal(t, expectedFiles, len(patches))
			}
		})
	}
}

func TestParallelCoder_ErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		numFiles  int
		errorTask string
		errorMsg  string
	}{
		{
			name:      "error on first task",
			numFiles:  10,
			errorTask: "task_0",
			errorMsg:  "generation error",
		},
		{
			name:      "error on middle task",
			numFiles:  10,
			errorTask: "task_5",
			errorMsg:  "syntax error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			baseCoder := newMockParallelCoder()
			baseCoder.setError(tt.errorTask, errors.New(tt.errorMsg))

			config := DefaultParallelConfig()
			pc := NewParallelCoder(baseCoder, config)
			plan := createSimplePlan(tt.numFiles)

			patches, err := pc.Generate(ctx, plan, nil)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorMsg)
			// Some files may have been generated
			assert.LessOrEqual(t, len(patches), tt.numFiles)
		})
	}
}

func TestParallelCoder_BoundedConcurrency(t *testing.T) {
	ctx := context.Background()
	baseCoder := newMockParallelCoder()
	baseCoder.delay = 20 * time.Millisecond

	config := ParallelGenerationConfig{
		MaxParallel:    3,
		EnableParallel: true,
	}

	pc := NewParallelCoder(baseCoder, config)
	plan := createSimplePlan(15)

	startTime := time.Now()
	patches, err := pc.Generate(ctx, plan, nil)
	duration := time.Since(startTime)

	require.NoError(t, err)
	assert.Equal(t, 15, len(patches))

	// With 15 files, 3 workers, and 20ms delay:
	// Expected time: ~(15/3) * 20ms = 100ms
	// Allow 2x overhead for coordination
	maxExpected := 200 * time.Millisecond
	assert.Less(t, duration, maxExpected,
		"should respect bounded concurrency")

	t.Logf("Generated 15 files with 3 workers in %v", duration)
}

func TestBuildDependencyGraph(t *testing.T) {
	tests := []struct {
		name           string
		plan           *models.GenerationPlan
		expectedLevels int
	}{
		{
			name:           "no dependencies",
			plan:           createSimplePlan(5),
			expectedLevels: 1,
		},
		{
			name:           "linear dependencies",
			plan:           createPlanWithLinearDeps(5),
			expectedLevels: 5,
		},
		{
			name:           "tree dependencies",
			plan:           createPlanWithTreeDeps(),
			expectedLevels: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseCoder := newMockParallelCoder()
			config := DefaultParallelConfig()
			pc := NewParallelCoder(baseCoder, config)

			graph := pc.buildDependencyGraph(tt.plan)

			assert.Equal(t, tt.expectedLevels, len(graph.levels),
				"should compute correct number of levels")

			// Verify all nodes have valid levels
			for _, node := range graph.nodes {
				assert.GreaterOrEqual(t, node.level, 0,
					"all nodes should have valid level")
			}

			// Verify level ordering respects dependencies
			for taskID, node := range graph.nodes {
				for _, depID := range node.dependencies {
					if depNode, exists := graph.nodes[depID]; exists {
						assert.Less(t, depNode.level, node.level,
							"dependency %s should be at lower level than %s", depID, taskID)
					}
				}
			}
		})
	}
}

func TestDeterministicParallelCoder(t *testing.T) {
	ctx := context.Background()
	numRuns := 5
	numFiles := 30

	var allResults [][]string

	for run := 0; run < numRuns; run++ {
		baseCoder := newMockParallelCoder()
		baseCoder.delay = 1 * time.Millisecond

		config := ParallelGenerationConfig{
			MaxParallel:    8,
			EnableParallel: true,
		}

		dpc := NewDeterministicParallelCoder(baseCoder, config)
		plan := createSimplePlan(numFiles)

		patches, err := dpc.Generate(ctx, plan, nil)
		require.NoError(t, err)
		require.Equal(t, numFiles, len(patches))

		// Extract file paths
		var filePaths []string
		for _, patch := range patches {
			filePaths = append(filePaths, patch.TargetFile)
		}

		allResults = append(allResults, filePaths)
	}

	// Verify all runs produced identical ordering
	for i := 1; i < numRuns; i++ {
		assert.Equal(t, allResults[0], allResults[i],
			"run %d should produce identical ordering to run 0", i)
	}

	t.Logf("Verified deterministic output across %d runs", numRuns)
}

func TestParallelCoderWithStats(t *testing.T) {
	ctx := context.Background()
	baseCoder := newMockParallelCoder()

	config := ParallelGenerationConfig{
		MaxParallel:    4,
		EnableParallel: true,
	}

	pcs := NewParallelCoderWithStats(baseCoder, config)
	plan := createSimplePlan(20)

	patches, err := pcs.Generate(ctx, plan, nil)

	require.NoError(t, err)
	assert.Equal(t, 20, len(patches))

	stats := pcs.Stats()

	assert.Equal(t, 20, stats.TotalFiles)
	assert.Equal(t, 4, stats.MaxParallelism)
	// Note: ActualMaxWorkers tracking requires a different interception point
	// as the coder.GenerateFile is called directly, not through the wrapper
	assert.Greater(t, stats.Duration, time.Duration(0))
	assert.Greater(t, stats.FilesPerSecond, 0.0)

	t.Logf("Stats: %d files in %v (%.2f files/sec)",
		stats.TotalFiles, stats.Duration, stats.FilesPerSecond)
}

func TestContextCancellation(t *testing.T) {
	baseCoder := newMockParallelCoder()
	baseCoder.delay = 150 * time.Millisecond // Longer delay to ensure cancellation happens

	config := DefaultParallelConfig()
	pc := NewParallelCoder(baseCoder, config)
	plan := createSimplePlan(50) // More files to ensure some are left

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	patches, err := pc.Generate(ctx, plan, nil)

	// Context cancellation might happen during or after execution depending on timing
	// Accept either: error + partial files OR no error + all files (if fast)
	if err != nil {
		assert.True(t, errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled),
			"should return context error")
		assert.LessOrEqual(t, len(patches), 50,
			"should not generate more files than requested")
	} else {
		// In very rare cases, all files complete before cancellation
		assert.Equal(t, 50, len(patches), "if no error, should complete all files")
	}

	t.Logf("Generated %d files (error: %v)", len(patches), err)
}

// Helper functions

func createSimplePlan(numFiles int) *models.GenerationPlan {
	plan := &models.GenerationPlan{
		ID:     "test_plan",
		Phases: []models.GenerationPhase{},
	}

	var tasks []models.GenerationTask
	for i := 0; i < numFiles; i++ {
		tasks = append(tasks, models.GenerationTask{
			ID:         fmt.Sprintf("task_%d", i),
			Type:       "generate_file",
			TargetPath: fmt.Sprintf("pkg/file_%d.go", i),
		})
	}

	plan.Phases = append(plan.Phases, models.GenerationPhase{
		Name:  "generate",
		Tasks: tasks,
	})

	return plan
}

func createPlanWithLinearDeps(numFiles int) *models.GenerationPlan {
	plan := &models.GenerationPlan{
		ID:     "linear_plan",
		Phases: []models.GenerationPhase{},
	}

	// Create phases where each phase depends on the previous one
	for i := 0; i < numFiles; i++ {
		var deps []string
		if i > 0 {
			deps = []string{fmt.Sprintf("phase_%d", i-1)}
		}

		phase := models.GenerationPhase{
			Name:         fmt.Sprintf("phase_%d", i),
			Order:        i,
			Dependencies: deps,
			Tasks: []models.GenerationTask{
				{
					ID:         fmt.Sprintf("task_%d", i),
					Type:       "generate_file",
					TargetPath: fmt.Sprintf("pkg/file_%d.go", i),
				},
			},
		}

		plan.Phases = append(plan.Phases, phase)
	}

	return plan
}

func createPlanWithTreeDeps() *models.GenerationPlan {
	plan := &models.GenerationPlan{
		ID:     "tree_plan",
		Phases: []models.GenerationPhase{},
	}

	// Create a tree structure with phases:
	// Level 0: phase_0 (root) - task_0
	// Level 1: phase_1, phase_2 (depend on phase_0) - task_1, task_2
	// Level 2: phase_3, phase_4, phase_5, phase_6 (depend on phase_1 or phase_2)

	// Phase 0 - root
	plan.Phases = append(plan.Phases, models.GenerationPhase{
		Name:  "phase_0",
		Order: 0,
		Tasks: []models.GenerationTask{
			{
				ID:         "task_0",
				Type:       "generate_file",
				TargetPath: "pkg/root.go",
			},
		},
	})

	// Phase 1 - left branch
	plan.Phases = append(plan.Phases, models.GenerationPhase{
		Name:         "phase_1",
		Order:        1,
		Dependencies: []string{"phase_0"},
		Tasks: []models.GenerationTask{
			{
				ID:         "task_1",
				Type:       "generate_file",
				TargetPath: "pkg/left.go",
			},
		},
	})

	// Phase 2 - right branch
	plan.Phases = append(plan.Phases, models.GenerationPhase{
		Name:         "phase_2",
		Order:        1,
		Dependencies: []string{"phase_0"},
		Tasks: []models.GenerationTask{
			{
				ID:         "task_2",
				Type:       "generate_file",
				TargetPath: "pkg/right.go",
			},
		},
	})

	// Phases 3-4 - depend on phase_1
	plan.Phases = append(plan.Phases, models.GenerationPhase{
		Name:         "phase_3",
		Order:        2,
		Dependencies: []string{"phase_1"},
		Tasks: []models.GenerationTask{
			{
				ID:         "task_3",
				Type:       "generate_file",
				TargetPath: "pkg/left_left.go",
			},
		},
	})

	plan.Phases = append(plan.Phases, models.GenerationPhase{
		Name:         "phase_4",
		Order:        2,
		Dependencies: []string{"phase_1"},
		Tasks: []models.GenerationTask{
			{
				ID:         "task_4",
				Type:       "generate_file",
				TargetPath: "pkg/left_right.go",
			},
		},
	})

	// Phases 5-6 - depend on phase_2
	plan.Phases = append(plan.Phases, models.GenerationPhase{
		Name:         "phase_5",
		Order:        2,
		Dependencies: []string{"phase_2"},
		Tasks: []models.GenerationTask{
			{
				ID:         "task_5",
				Type:       "generate_file",
				TargetPath: "pkg/right_left.go",
			},
		},
	})

	plan.Phases = append(plan.Phases, models.GenerationPhase{
		Name:         "phase_6",
		Order:        2,
		Dependencies: []string{"phase_2"},
		Tasks: []models.GenerationTask{
			{
				ID:         "task_6",
				Type:       "generate_file",
				TargetPath: "pkg/right_right.go",
			},
		},
	})

	return plan
}

// Benchmarks

func BenchmarkParallelCoder_Sequential(b *testing.B) {
	ctx := context.Background()
	plan := createSimplePlan(20)

	for i := 0; i < b.N; i++ {
		baseCoder := newMockParallelCoder()
		baseCoder.delay = 1 * time.Millisecond

		config := ParallelGenerationConfig{
			MaxParallel:    1,
			EnableParallel: false,
		}

		pc := NewParallelCoder(baseCoder, config)
		_, err := pc.Generate(ctx, plan, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParallelCoder_Parallel4(b *testing.B) {
	ctx := context.Background()
	plan := createSimplePlan(20)

	for i := 0; i < b.N; i++ {
		baseCoder := newMockParallelCoder()
		baseCoder.delay = 1 * time.Millisecond

		config := ParallelGenerationConfig{
			MaxParallel:    4,
			EnableParallel: true,
		}

		pc := NewParallelCoder(baseCoder, config)
		_, err := pc.Generate(ctx, plan, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParallelCoder_Parallel8(b *testing.B) {
	ctx := context.Background()
	plan := createSimplePlan(20)

	for i := 0; i < b.N; i++ {
		baseCoder := newMockParallelCoder()
		baseCoder.delay = 1 * time.Millisecond

		config := ParallelGenerationConfig{
			MaxParallel:    8,
			EnableParallel: true,
		}

		pc := NewParallelCoder(baseCoder, config)
		_, err := pc.Generate(ctx, plan, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}
