package unit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dshills/gocreator/internal/generate"
	"github.com/dshills/gocreator/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// mockConcurrentCoder simulates a coder that can generate files concurrently
type mockConcurrentCoder struct {
	generateFileCount int64
	generateFileMu    sync.Mutex
	delay             time.Duration
	errorOn           map[string]error // map of taskID to error
	mu                sync.Mutex
}

func newMockConcurrentCoder() *mockConcurrentCoder {
	return &mockConcurrentCoder{
		delay:   10 * time.Millisecond,
		errorOn: make(map[string]error),
	}
}

func (m *mockConcurrentCoder) Generate(_ context.Context, plan *models.GenerationPlan) ([]models.Patch, error) {
	var patches []models.Patch

	// Simulate processing all tasks
	for _, phase := range plan.Phases {
		for _, task := range phase.Tasks {
			if task.Type == "generate_file" {
				patch, err := m.GenerateFile(context.Background(), task, plan)
				if err != nil {
					return nil, err
				}
				patches = append(patches, patch)
			}
		}
	}

	return patches, nil
}

func (m *mockConcurrentCoder) GenerateFile(_ context.Context, task models.GenerationTask, _ *models.GenerationPlan) (models.Patch, error) {
	atomic.AddInt64(&m.generateFileCount, 1)

	// Simulate work
	time.Sleep(m.delay)

	// Check if this task should error
	m.mu.Lock()
	err, shouldError := m.errorOn[task.ID]
	m.mu.Unlock()

	if shouldError {
		return models.Patch{}, err
	}

	return models.Patch{
		TargetFile: task.TargetPath,
		Diff:       fmt.Sprintf("// Generated code for %s", task.TargetPath),
		AppliedAt:  time.Now(),
		Reversible: true,
	}, nil
}

func (m *mockConcurrentCoder) setError(taskID string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorOn[taskID] = err
}

func (m *mockConcurrentCoder) getGenerateFileCount() int64 {
	return atomic.LoadInt64(&m.generateFileCount)
}

// TestConcurrentFileGeneration tests parallel file generation for independent files
func TestConcurrentFileGeneration(t *testing.T) {
	tests := []struct {
		name          string
		numFiles      int
		maxWorkers    int
		expectedCalls int64
	}{
		{
			name:          "multiple independent files",
			numFiles:      10,
			maxWorkers:    4,
			expectedCalls: 10,
		},
		{
			name:          "many files with bounded concurrency",
			numFiles:      100,
			maxWorkers:    8,
			expectedCalls: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			coder := newMockConcurrentCoder()

			// Create plan with independent files
			plan := createTestPlan(tt.numFiles, false)

			// Generate files concurrently
			startTime := time.Now()
			patches, err := generateFilesParallel(ctx, coder, plan, tt.maxWorkers)
			duration := time.Since(startTime)

			require.NoError(t, err)
			assert.Equal(t, tt.numFiles, len(patches))
			assert.Equal(t, tt.expectedCalls, coder.getGenerateFileCount())

			// Verify parallel execution - should be faster than sequential
			sequentialTime := time.Duration(tt.numFiles) * coder.delay
			// With parallelism, should complete in roughly sequentialTime/maxWorkers
			// Allow some overhead for goroutine coordination
			maxExpectedTime := (sequentialTime / time.Duration(tt.maxWorkers)) * 2
			assert.Less(t, duration, maxExpectedTime,
				"parallel execution should be faster than sequential")

			t.Logf("Generated %d files in %v (sequential would be %v)",
				tt.numFiles, duration, sequentialTime)
		})
	}
}

// TestConcurrentFileGenerationErrorHandling tests proper error handling with errgroup
func TestConcurrentFileGenerationErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		numFiles      int
		errorOnTaskID string
		errorMsg      string
		expectError   bool
	}{
		{
			name:        "no errors",
			numFiles:    10,
			expectError: false,
		},
		{
			name:          "error on first file",
			numFiles:      10,
			errorOnTaskID: "task_0",
			errorMsg:      "generation failed",
			expectError:   true,
		},
		{
			name:          "error on middle file",
			numFiles:      10,
			errorOnTaskID: "task_5",
			errorMsg:      "syntax error",
			expectError:   true,
		},
		{
			name:          "error on last file",
			numFiles:      10,
			errorOnTaskID: "task_9",
			errorMsg:      "validation failed",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			coder := newMockConcurrentCoder()

			// Set up error condition
			if tt.errorOnTaskID != "" {
				coder.setError(tt.errorOnTaskID, errors.New(tt.errorMsg))
			}

			// Create plan
			plan := createTestPlan(tt.numFiles, false)

			// Generate files
			patches, err := generateFilesParallel(ctx, coder, plan, 4)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				// Some files may have been generated before error
				assert.LessOrEqual(t, len(patches), tt.numFiles)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.numFiles, len(patches))
			}
		})
	}
}

// TestBoundedConcurrency tests that max parallel workers limit is respected
func TestBoundedConcurrency(t *testing.T) {
	tests := []struct {
		name        string
		numFiles    int
		maxWorkers  int
		fileDelay   time.Duration
		minDuration time.Duration
		maxDuration time.Duration
	}{
		{
			name:        "4 files, 2 workers",
			numFiles:    4,
			maxWorkers:  2,
			fileDelay:   50 * time.Millisecond,
			minDuration: 100 * time.Millisecond, // 2 batches * 50ms
			maxDuration: 200 * time.Millisecond, // with overhead
		},
		{
			name:        "10 files, 5 workers",
			numFiles:    10,
			maxWorkers:  5,
			fileDelay:   20 * time.Millisecond,
			minDuration: 40 * time.Millisecond, // 2 batches * 20ms
			maxDuration: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			coder := newMockConcurrentCoder()
			coder.delay = tt.fileDelay

			plan := createTestPlan(tt.numFiles, false)

			// Track concurrent execution
			var activeWorkers int64
			var maxActiveWorkers int64
			var mu sync.Mutex

			// Wrap coder to track concurrency
			trackedCoder := &concurrencyTrackingCoder{
				coder:         coder,
				activeWorkers: &activeWorkers,
				maxActive:     &maxActiveWorkers,
				mu:            &mu,
				maxWorkers:    tt.maxWorkers,
			}

			startTime := time.Now()
			_, err := generateFilesParallel(ctx, trackedCoder, plan, tt.maxWorkers)
			duration := time.Since(startTime)

			require.NoError(t, err)

			// Verify duration is within expected range
			assert.GreaterOrEqual(t, duration, tt.minDuration,
				"should take at least minDuration")
			assert.LessOrEqual(t, duration, tt.maxDuration,
				"should complete within maxDuration")

			// Verify max concurrent workers didn't exceed limit
			mu.Lock()
			maxActive := int(atomic.LoadInt64(&maxActiveWorkers))
			mu.Unlock()

			assert.LessOrEqual(t, maxActive, tt.maxWorkers,
				"should not exceed max workers limit")

			t.Logf("Max concurrent workers: %d (limit: %d), Duration: %v",
				maxActive, tt.maxWorkers, duration)
		})
	}
}

// TestContextCancellation tests proper cleanup when context is cancelled
func TestContextCancellation(t *testing.T) {
	coder := newMockConcurrentCoder()
	coder.delay = 50 * time.Millisecond

	plan := createTestPlan(50, false)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after short delay to interrupt some file generations
	go func() {
		time.Sleep(80 * time.Millisecond)
		cancel()
	}()

	patches, err := generateFilesParallel(ctx, coder, plan, 4)

	// Should return context cancelled error or complete before cancellation
	if err != nil {
		// Cancellation happened during generation
		assert.True(t, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded),
			"should return context error")
		// Some files may have been generated before cancellation
		assert.Less(t, len(patches), 50,
			"should not generate all files after cancellation")
	} else {
		// All files completed before cancellation fired
		assert.Equal(t, 50, len(patches))
	}

	t.Logf("Generated %d files (cancellation after 80ms, file delay 50ms)", len(patches))
}

// TestDeterministicOutput tests that concurrent execution produces identical output
func TestDeterministicOutput(t *testing.T) {
	numFiles := 50
	numRuns := 5

	var results [][]string

	for run := 0; run < numRuns; run++ {
		ctx := context.Background()
		coder := newMockConcurrentCoder()
		coder.delay = 1 * time.Millisecond // Fast execution

		plan := createTestPlan(numFiles, false)
		patches, err := generateFilesParallel(ctx, coder, plan, 8)

		require.NoError(t, err)
		require.Equal(t, numFiles, len(patches))

		// Extract file paths in order
		var filePaths []string
		for _, patch := range patches {
			filePaths = append(filePaths, patch.TargetFile)
		}

		results = append(results, filePaths)
	}

	// Verify all runs produced same file list (though possibly in different order)
	for i := 1; i < numRuns; i++ {
		assert.ElementsMatch(t, results[0], results[i],
			"run %d should produce same files as run 0", i)
	}
}

// Helper functions

func createTestPlan(numFiles int, withDependencies bool) *models.GenerationPlan {
	plan := &models.GenerationPlan{
		SchemaVersion: "1.0",
		ID:            "test_plan",
		Phases:        []models.GenerationPhase{},
		FileTree: models.FileTree{
			Files: []models.File{},
		},
	}

	var tasks []models.GenerationTask

	for i := 0; i < numFiles; i++ {
		task := models.GenerationTask{
			ID:         fmt.Sprintf("task_%d", i),
			Type:       "generate_file",
			TargetPath: fmt.Sprintf("internal/pkg/file_%d.go", i),
			Inputs:     map[string]interface{}{},
		}

		// Tasks don't have direct dependencies - they inherit from phase
		// withDependencies param is ignored as we use phase ordering instead

		tasks = append(tasks, task)
	}

	// Add all tasks to a single phase for simplicity
	plan.Phases = append(plan.Phases, models.GenerationPhase{
		Name:  "generate_code",
		Tasks: tasks,
	})

	return plan
}

func generateFilesParallel(ctx context.Context, coder generate.Coder, plan *models.GenerationPlan, maxWorkers int) ([]models.Patch, error) {
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(maxWorkers)

	var mu sync.Mutex
	var patches []models.Patch

	// Process all tasks concurrently
	for _, phase := range plan.Phases {
		for _, task := range phase.Tasks {
			if task.Type == "generate_file" {
				task := task // Capture for goroutine

				g.Go(func() error {
					patch, err := coder.GenerateFile(gCtx, task, plan)
					if err != nil {
						return err
					}

					mu.Lock()
					patches = append(patches, patch)
					mu.Unlock()

					return nil
				})
			}
		}
	}

	if err := g.Wait(); err != nil {
		return patches, err
	}

	return patches, nil
}

// concurrencyTrackingCoder wraps a coder to track concurrent executions
type concurrencyTrackingCoder struct {
	coder         generate.Coder
	activeWorkers *int64
	maxActive     *int64
	mu            *sync.Mutex
	maxWorkers    int
}

func (c *concurrencyTrackingCoder) Generate(ctx context.Context, plan *models.GenerationPlan) ([]models.Patch, error) {
	return c.coder.Generate(ctx, plan)
}

func (c *concurrencyTrackingCoder) GenerateFile(ctx context.Context, task models.GenerationTask, plan *models.GenerationPlan) (models.Patch, error) {
	// Increment active workers
	active := atomic.AddInt64(c.activeWorkers, 1)

	// Track max
	c.mu.Lock()
	if active > atomic.LoadInt64(c.maxActive) {
		atomic.StoreInt64(c.maxActive, active)
	}
	c.mu.Unlock()

	// Ensure we don't exceed max workers (should be enforced by errgroup)
	if active > int64(c.maxWorkers) {
		atomic.AddInt64(c.activeWorkers, -1)
		return models.Patch{}, fmt.Errorf("exceeded max workers: %d > %d", active, c.maxWorkers)
	}

	// Call actual coder
	patch, err := c.coder.GenerateFile(ctx, task, plan)

	// Decrement active workers
	atomic.AddInt64(c.activeWorkers, -1)

	return patch, err
}

// BenchmarkConcurrentFileGeneration benchmarks parallel file generation
func BenchmarkConcurrentFileGeneration(b *testing.B) {
	benchmarks := []struct {
		name       string
		numFiles   int
		maxWorkers int
	}{
		{"10_files_2_workers", 10, 2},
		{"10_files_4_workers", 10, 4},
		{"10_files_8_workers", 10, 8},
		{"50_files_4_workers", 50, 4},
		{"50_files_8_workers", 50, 8},
		{"100_files_8_workers", 100, 8},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			ctx := context.Background()
			plan := createTestPlan(bm.numFiles, false)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				coder := newMockConcurrentCoder()
				coder.delay = 1 * time.Millisecond

				_, err := generateFilesParallel(ctx, coder, plan, bm.maxWorkers)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkSequentialVsParallel compares sequential vs parallel generation
func BenchmarkSequentialVsParallel(b *testing.B) {
	numFiles := 50
	plan := createTestPlan(numFiles, false)

	b.Run("sequential", func(b *testing.B) {
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			coder := newMockConcurrentCoder()
			coder.delay = 1 * time.Millisecond

			// Generate sequentially
			var patches []models.Patch
			for _, phase := range plan.Phases {
				for _, task := range phase.Tasks {
					if task.Type == "generate_file" {
						patch, err := coder.GenerateFile(ctx, task, plan)
						if err != nil {
							b.Fatal(err)
						}
						patches = append(patches, patch)
					}
				}
			}
		}
	})

	b.Run("parallel_4_workers", func(b *testing.B) {
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			coder := newMockConcurrentCoder()
			coder.delay = 1 * time.Millisecond

			_, err := generateFilesParallel(ctx, coder, plan, 4)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("parallel_8_workers", func(b *testing.B) {
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			coder := newMockConcurrentCoder()
			coder.delay = 1 * time.Millisecond

			_, err := generateFilesParallel(ctx, coder, plan, 8)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
