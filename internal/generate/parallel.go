package generate

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// ParallelGenerationConfig holds configuration for parallel generation
type ParallelGenerationConfig struct {
	// MaxParallel limits the number of concurrent file generation operations
	// Default: 4 (to avoid overwhelming the LLM API)
	MaxParallel int

	// EnableParallel controls whether parallel generation is enabled
	// If false, generation will be sequential
	EnableParallel bool
}

// DefaultParallelConfig returns default parallel generation configuration
func DefaultParallelConfig() ParallelGenerationConfig {
	return ParallelGenerationConfig{
		MaxParallel:    4,
		EnableParallel: true,
	}
}

// ParallelCoder wraps a Coder to enable parallel file generation with worker pool pattern
type ParallelCoder struct {
	coder  Coder
	config ParallelGenerationConfig
}

// NewParallelCoder creates a new parallel coder that wraps an existing coder
func NewParallelCoder(coder Coder, config ParallelGenerationConfig) *ParallelCoder {
	if config.MaxParallel <= 0 {
		config.MaxParallel = 4
	}

	return &ParallelCoder{
		coder:  coder,
		config: config,
	}
}

// Generate creates source code files in parallel based on the generation plan
// Uses worker pool pattern with bounded concurrency to avoid overwhelming LLM API
func (pc *ParallelCoder) Generate(ctx context.Context, plan *models.GenerationPlan) ([]models.Patch, error) {
	if !pc.config.EnableParallel {
		// Fall back to sequential generation
		return pc.coder.Generate(ctx, plan)
	}

	log.Info().
		Str("plan_id", plan.ID).
		Int("phases", len(plan.Phases)).
		Int("max_parallel", pc.config.MaxParallel).
		Msg("Starting parallel code generation")

	startTime := time.Now()

	// Build dependency graph from tasks
	taskGraph := pc.buildDependencyGraph(plan)

	// Generate files respecting dependencies
	patches, err := pc.generateWithDependencies(ctx, plan, taskGraph)
	if err != nil {
		return nil, fmt.Errorf("parallel generation failed: %w", err)
	}

	duration := time.Since(startTime)

	log.Info().
		Str("plan_id", plan.ID).
		Int("files_generated", len(patches)).
		Dur("duration", duration).
		Msg("Parallel code generation completed")

	return patches, nil
}

// GenerateFile delegates to the wrapped coder
func (pc *ParallelCoder) GenerateFile(ctx context.Context, task models.GenerationTask, plan *models.GenerationPlan) (models.Patch, error) {
	return pc.coder.GenerateFile(ctx, task, plan)
}

// taskNode represents a task in the dependency graph
type taskNode struct {
	task         models.GenerationTask
	dependencies []string
	level        int // Execution level (0 = no dependencies, 1 = depends on level 0, etc.)
}

// dependencyGraph represents tasks organized by execution level
type dependencyGraph struct {
	nodes  map[string]*taskNode
	levels [][]string // taskIDs organized by level
}

// buildDependencyGraph creates a dependency graph from the generation plan
func (pc *ParallelCoder) buildDependencyGraph(plan *models.GenerationPlan) *dependencyGraph {
	graph := &dependencyGraph{
		nodes: make(map[string]*taskNode),
	}

	// Build phase->tasks mapping
	phaseToTasks := make(map[string][]string)
	for _, phase := range plan.Phases {
		for _, task := range phase.Tasks {
			if task.Type == "generate_file" {
				phaseToTasks[phase.Name] = append(phaseToTasks[phase.Name], task.ID)
			}
		}
	}

	// Build nodes from phases, resolving phase dependencies to task dependencies
	for _, phase := range plan.Phases {
		for _, task := range phase.Tasks {
			if task.Type == "generate_file" {
				// Resolve phase dependencies to task dependencies
				var taskDeps []string
				for _, depPhaseName := range phase.Dependencies {
					// Add all tasks from the dependency phase as dependencies
					if depTasks, exists := phaseToTasks[depPhaseName]; exists {
						taskDeps = append(taskDeps, depTasks...)
					}
				}

				graph.nodes[task.ID] = &taskNode{
					task:         task,
					dependencies: taskDeps, // Resolved task-level dependencies
					level:        -1,       // Will be computed
				}
			}
		}
	}

	// Compute levels based on resolved task dependencies
	pc.computeLevels(graph)

	return graph
}

// computeLevels assigns each task to an execution level based on phase dependencies
func (pc *ParallelCoder) computeLevels(graph *dependencyGraph) {
	// Iteratively compute levels
	maxIterations := len(graph.nodes) + 1
	for iteration := 0; iteration < maxIterations; iteration++ {
		changed := false

		for _, node := range graph.nodes {
			if node.level >= 0 {
				continue // Already computed
			}

			// Check if all dependencies have assigned levels
			maxDepLevel := -1
			allDepsReady := true

			for _, depID := range node.dependencies {
				depNode, exists := graph.nodes[depID]
				if !exists {
					// Dependency not in graph (might be non-generate_file task or phase name)
					// Assume it's ready
					continue
				}

				if depNode.level < 0 {
					allDepsReady = false
					break
				}

				if depNode.level > maxDepLevel {
					maxDepLevel = depNode.level
				}
			}

			// If all dependencies ready (or no dependencies), assign level
			if allDepsReady {
				node.level = maxDepLevel + 1
				changed = true
			}
		}

		if !changed {
			break
		}
	}

	// Organize tasks by level
	levelMap := make(map[int][]string)
	for taskID, node := range graph.nodes {
		if node.level < 0 {
			node.level = 0 // Fallback for tasks with no computable level
		}
		levelMap[node.level] = append(levelMap[node.level], taskID)
	}

	// Convert to ordered slice
	maxLevel := 0
	for level := range levelMap {
		if level > maxLevel {
			maxLevel = level
		}
	}

	graph.levels = make([][]string, maxLevel+1)
	for level := 0; level <= maxLevel; level++ {
		graph.levels[level] = levelMap[level]
	}
}

// generateWithDependencies generates files in parallel while respecting dependencies
func (pc *ParallelCoder) generateWithDependencies(ctx context.Context, plan *models.GenerationPlan, graph *dependencyGraph) ([]models.Patch, error) {
	var allPatches []models.Patch
	var patchesMu sync.Mutex

	// Track completed tasks (dependencies are now task-level, not phase-level)
	completedTasks := make(map[string]bool)
	var tasksMu sync.RWMutex

	// Process each level sequentially, but parallelize within each level
	for levelIdx, levelTasks := range graph.levels {
		if len(levelTasks) == 0 {
			continue
		}

		log.Debug().
			Int("level", levelIdx).
			Int("tasks", len(levelTasks)).
			Msg("Processing generation level")

		// Use errgroup for bounded concurrency
		g, gCtx := errgroup.WithContext(ctx)
		g.SetLimit(pc.config.MaxParallel)

		// Process all tasks in this level in parallel
		for _, taskID := range levelTasks {
			taskID := taskID // Capture for goroutine

			g.Go(func() error {
				node := graph.nodes[taskID]

				// Verify task dependencies are completed
				for _, depTaskID := range node.dependencies {
					// Skip dependencies that are not in the graph (external/non-generate-file tasks)
					if graph.nodes[depTaskID] == nil {
						continue
					}

					tasksMu.RLock()
					isCompleted := completedTasks[depTaskID]
					tasksMu.RUnlock()

					if !isCompleted {
						return fmt.Errorf("dependency %s not completed for task %s", depTaskID, taskID)
					}
				}

				// Generate file - call pc.GenerateFile to respect method overrides
				patch, err := pc.GenerateFile(gCtx, node.task, plan)
				if err != nil {
					return fmt.Errorf("failed to generate file for task %s: %w", taskID, err)
				}

				// Store patch
				patchesMu.Lock()
				allPatches = append(allPatches, patch)
				patchesMu.Unlock()

				// Mark task as completed
				tasksMu.Lock()
				completedTasks[taskID] = true
				tasksMu.Unlock()

				log.Debug().
					Str("task_id", taskID).
					Str("file", node.task.TargetPath).
					Int("level", levelIdx).
					Msg("File generated successfully")

				return nil
			})
		}

		// Wait for all tasks in this level to complete
		if err := g.Wait(); err != nil {
			return allPatches, fmt.Errorf("level %d generation failed: %w", levelIdx, err)
		}

		log.Debug().
			Int("level", levelIdx).
			Int("completed", len(levelTasks)).
			Msg("Level completed successfully")
	}

	return allPatches, nil
}

// GenerationStats tracks statistics about parallel generation
type GenerationStats struct {
	TotalFiles       int
	Levels           int
	MaxParallelism   int
	ActualMaxWorkers int
	Duration         time.Duration
	FilesPerSecond   float64
}

// ParallelCoderWithStats wraps ParallelCoder to collect generation statistics
type ParallelCoderWithStats struct {
	*ParallelCoder
	stats      GenerationStats
	statsMu    sync.Mutex
	startTime  time.Time
	maxWorkers int64
	curWorkers int64
	workersMu  sync.Mutex
}

// NewParallelCoderWithStats creates a parallel coder that tracks statistics
func NewParallelCoderWithStats(coder Coder, config ParallelGenerationConfig) *ParallelCoderWithStats {
	pc := NewParallelCoder(coder, config)

	return &ParallelCoderWithStats{
		ParallelCoder: pc,
		stats: GenerationStats{
			MaxParallelism: config.MaxParallel,
		},
	}
}

// Generate wraps the parent Generate to collect statistics
func (pcs *ParallelCoderWithStats) Generate(ctx context.Context, plan *models.GenerationPlan) ([]models.Patch, error) {
	pcs.startTime = time.Now()

	patches, err := pcs.ParallelCoder.Generate(ctx, plan)

	pcs.statsMu.Lock()
	pcs.stats.Duration = time.Since(pcs.startTime)
	pcs.stats.TotalFiles = len(patches)
	if pcs.stats.Duration > 0 {
		pcs.stats.FilesPerSecond = float64(len(patches)) / pcs.stats.Duration.Seconds()
	}
	pcs.stats.ActualMaxWorkers = int(pcs.maxWorkers)
	pcs.statsMu.Unlock()

	return patches, err
}

// GenerateFile wraps the parent GenerateFile to track worker count
func (pcs *ParallelCoderWithStats) GenerateFile(ctx context.Context, task models.GenerationTask, plan *models.GenerationPlan) (models.Patch, error) {
	// Track concurrent workers
	pcs.workersMu.Lock()
	pcs.curWorkers++
	if pcs.curWorkers > pcs.maxWorkers {
		pcs.maxWorkers = pcs.curWorkers
	}
	pcs.workersMu.Unlock()

	// Generate file
	patch, err := pcs.ParallelCoder.GenerateFile(ctx, task, plan)

	// Decrement worker count
	pcs.workersMu.Lock()
	pcs.curWorkers--
	pcs.workersMu.Unlock()

	return patch, err
}

// Stats returns the collected generation statistics
func (pcs *ParallelCoderWithStats) Stats() GenerationStats {
	pcs.statsMu.Lock()
	defer pcs.statsMu.Unlock()
	return pcs.stats
}

// DeterministicParallelCoder ensures deterministic output despite parallel execution
// by sorting patches before returning them
type DeterministicParallelCoder struct {
	*ParallelCoder
}

// NewDeterministicParallelCoder creates a parallel coder with deterministic output
func NewDeterministicParallelCoder(coder Coder, config ParallelGenerationConfig) *DeterministicParallelCoder {
	pc := NewParallelCoder(coder, config)

	return &DeterministicParallelCoder{
		ParallelCoder: pc,
	}
}

// Generate wraps the parent Generate to ensure deterministic patch ordering
func (dpc *DeterministicParallelCoder) Generate(ctx context.Context, plan *models.GenerationPlan) ([]models.Patch, error) {
	patches, err := dpc.ParallelCoder.Generate(ctx, plan)
	if err != nil {
		return patches, err
	}

	// Sort patches by target file path to ensure deterministic output
	// This is crucial for reproducible builds
	sortPatches(patches)

	return patches, nil
}

// sortPatches sorts patches by target file path for deterministic output
func sortPatches(patches []models.Patch) {
	sort.Slice(patches, func(i, j int) bool {
		return patches[i].TargetFile < patches[j].TargetFile
	})
}
