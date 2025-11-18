# Implementation Guide: Performance Optimization

This guide provides implementation priorities and technical details for the performance optimization feature.

## Quick Wins (Implement First)

These optimizations provide maximum ROI with minimal complexity:

### 1. Template-Based Boilerplate (2-3 days)

**Impact**: Eliminate 15-25% of LLM calls immediately
**Complexity**: Low
**Dependencies**: None

**Implementation**:
```go
// internal/generate/templates/boilerplate.go
type TemplateGenerator struct {
    templates *template.Template
}

func (t *TemplateGenerator) GenerateGoMod(fcs *models.FCS) (string, error) {
    data := struct {
        ModuleName string
        GoVersion  string
        Dependencies []string
    }{
        ModuleName: fcs.ProjectName,
        GoVersion: fcs.BuildConfig.GoVersion,
        Dependencies: fcs.Architecture.Dependencies,
    }
    return t.execute("go.mod.tmpl", data)
}
```

**Templates to create**:
- `go.mod.tmpl` - Go module file
- `gitignore.tmpl` - Standard Go .gitignore
- `dockerfile.tmpl` - Multi-stage Go Docker build
- `makefile.tmpl` - Common Go tasks
- `readme.tmpl` - Project README skeleton

### 2. Anthropic Prompt Caching (3-5 days)

**Impact**: 60-80% reduction in input tokens across multi-phase workflows
**Complexity**: Medium
**Dependencies**: Anthropic SDK with caching support

**Implementation**:
```go
// pkg/llm/anthropic.go
type CacheableMessage struct {
    Role         string                 `json:"role"`
    Content      string                 `json:"content"`
    CacheControl *CacheControl          `json:"cache_control,omitempty"`
}

type CacheControl struct {
    Type string `json:"type"` // "ephemeral"
}

func (c *AnthropicClient) GenerateWithCache(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
    messages := []CacheableMessage{
        {
            Role: "user",
            Content: systemPrompt,  // FCS schema, examples
            CacheControl: &CacheControl{Type: "ephemeral"}, // Cache this!
        },
        {
            Role: "user",
            Content: userPrompt,  // Specific task
        },
    }
    // Make API call with cache_control
}
```

**What to cache**:
- ✅ FCS schema and structure (changes rarely)
- ✅ Code generation examples (static)
- ✅ Architecture patterns and guidelines (static)
- ❌ Task-specific details (changes per call)
- ❌ File-specific context (changes per file)

### 3. Smart Context Filtering (2-3 days)

**Impact**: 50%+ reduction in prompt tokens
**Complexity**: Low-Medium
**Dependencies**: None

**Implementation**:
```go
// internal/generate/context.go
type ContextFilter struct {
    fcs *models.FCS
}

func (f *ContextFilter) FilterForFile(filePath string, plan *models.GenerationPlan) *models.FilteredFCS {
    // Determine what entities this file needs
    requiredEntities := f.findDependencies(filePath, plan)

    return &models.FilteredFCS{
        Entities: f.filterEntities(requiredEntities),
        Packages: f.filterPackages(filePath),
        GlobalConfig: f.fcs.BuildConfig,  // Always include
    }
}

func (f *ContextFilter) findDependencies(filePath string, plan *models.GenerationPlan) []string {
    // Parse file path: "internal/user/repository.go" -> needs User entity
    // Check plan task metadata for explicit dependencies
    // Include transitive dependencies (User -> Address -> Country)
}
```

## Medium-Term Improvements (Implement Second)

### 4. Batch Generation (5-7 days)

**Impact**: 70%+ reduction in API calls for repetitive projects
**Complexity**: Medium-High
**Dependencies**: Enhanced prompt templates

**Strategy**:
```go
// internal/generate/batcher.go
type BatchGroup struct {
    Files      []string
    Pattern    string  // "entity" | "crud_handler" | "repository"
    Template   string
}

func (b *Batcher) GroupSimilarFiles(tasks []models.GenerationTask) []BatchGroup {
    // Group by similarity:
    // - All entity models together (User, Product, Order)
    // - All CRUD handlers together (UserHandler, ProductHandler)
    // - All repositories together
}

func (b *Batcher) GenerateBatch(ctx context.Context, group BatchGroup) ([]models.Patch, error) {
    prompt := fmt.Sprintf(`Generate %d similar files:

Pattern: %s

Files to generate:
%s

Return as JSON array:
[
  {"path": "file1.go", "content": "..."},
  {"path": "file2.go", "content": "..."}
]
`, len(group.Files), group.Pattern, formatFileList(group.Files))

    // Single LLM call for all files
    response := b.llm.Generate(ctx, prompt)
    return b.parseBatchResponse(response)
}
```

### 5. Parallel Execution (5-7 days)

**Impact**: 40-60% faster execution for compatible workflows
**Complexity**: Medium-High
**Dependencies**: Fix langgraph-go bug or implement custom executor

**Option A: Fix langgraph-go**
```go
// Contribute fix to github.com/dshills/langgraph-go
// Issue: Concurrent execution doesn't merge deltas correctly
// Solution: Use sync.Mutex around state reducer
```

**Option B: Custom Executor**
```go
// internal/workflow/executor.go
type ParallelExecutor struct {
    maxConcurrent int
    semaphore     chan struct{}
}

func (e *ParallelExecutor) Execute(ctx context.Context, graph *DependencyGraph) error {
    // Topological sort of tasks
    levels := graph.TopologicalLevels()

    for _, level := range levels {
        // All tasks in a level can run in parallel
        g, ctx := errgroup.WithContext(ctx)
        g.SetLimit(e.maxConcurrent)

        for _, task := range level {
            task := task
            g.Go(func() error {
                return e.executeTask(ctx, task)
            })
        }

        if err := g.Wait(); err != nil {
            return err
        }
    }
    return nil
}
```

### 6. Incremental Regeneration (7-10 days)

**Impact**: 80%+ faster regeneration for localized changes
**Complexity**: High
**Dependencies**: Checksum tracking, dependency graph

**Implementation**:
```go
// internal/generate/incremental.go
type IncrementalEngine struct {
    state *IncrementalState
}

func (e *IncrementalEngine) ComputeDelta(oldFCS, newFCS *models.FCS) *RegenerationPlan {
    changes := e.detectChanges(oldFCS, newFCS)

    // Build dependency graph
    affected := make(map[string]bool)
    for _, change := range changes {
        // Add directly changed files
        affectedFiles := e.filesForEntity(change.Entity)

        // Add transitively affected files
        dependents := e.findDependents(change.Entity)

        for _, file := range append(affectedFiles, dependents...) {
            affected[file] = true
        }
    }

    return &RegenerationPlan{
        FilesToRegenerate: affected,
        FilesToPreserve:   e.computeUnchanged(affected),
    }
}

func (e *IncrementalEngine) Regenerate(ctx context.Context, plan *RegenerationPlan) error {
    // Only generate files in plan.FilesToRegenerate
    // Reuse existing files from plan.FilesToPreserve
    // Update checksums in state
}
```

## Long-Term Enhancements (Implement Third)

### 7. Streaming Progress (3-5 days)

**Impact**: Significantly better UX
**Complexity**: Medium
**Dependencies**: Terminal rendering library

```go
// internal/cli/progress.go
import "github.com/cheggaaa/pb/v3"

type ProgressTracker struct {
    bar *pb.ProgressBar
    events chan Event
}

func (p *ProgressTracker) Start(ctx context.Context, totalPhases int) {
    p.bar = pb.StartNew(totalPhases)
    p.bar.SetTemplate(`{{string . "phase"}} {{bar . }} {{percent . }} [{{counters .}}] {{string . "eta"}}`)

    go p.consumeEvents(ctx)
}

func (p *ProgressTracker) consumeEvents(ctx context.Context) {
    for {
        select {
        case event := <-p.events:
            p.handleEvent(event)
        case <-ctx.Done():
            p.bar.Finish()
            return
        }
    }
}
```

### 8. Dry-Run Mode (2-3 days)

**Impact**: Better user confidence, experimentation
**Complexity**: Low

```go
// cmd/gocreator/generate.go
var dryRun bool

func init() {
    generateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show plan without executing")
}

func runGenerate(cmd *cobra.Command, args []string) error {
    plan, cost := planner.Plan(ctx, fcs)

    if dryRun {
        printPlanSummary(plan, cost)
        fmt.Printf("\nEstimated cost: $%.4f\n", cost.TotalUSD)
        fmt.Printf("Estimated time: %s\n", cost.EstimatedDuration)
        return nil
    }

    // Normal execution
}
```

### 9. Resume Capability (5-7 days)

**Impact**: Reliability for long-running operations
**Complexity**: Medium-High

```go
// internal/generate/checkpoint.go
type Checkpoint struct {
    ID              string
    FCS             *models.FCS
    CompletedPhases []string
    CompletedFiles  map[string]string  // file -> checksum
    CreatedAt       time.Time
}

func (e *Engine) SaveCheckpoint(state *GenerationState) error {
    checkpoint := &Checkpoint{
        ID: state.ExecutionID,
        FCS: state.FCS,
        CompletedPhases: state.CompletedPhases,
        CompletedFiles: state.GeneratedFiles,
        CreatedAt: time.Now(),
    }

    return e.checkpointStore.Save(checkpoint)
}

func (e *Engine) Resume(checkpointID string) (*models.GenerationOutput, error) {
    checkpoint, err := e.checkpointStore.Load(checkpointID)
    if err != nil {
        return nil, err
    }

    // Continue from last completed phase
    startPhase := len(checkpoint.CompletedPhases)
    return e.executeFromPhase(checkpoint.FCS, startPhase)
}
```

## Performance Testing Framework

Create benchmarks to validate improvements:

```go
// internal/generate/benchmark_test.go
func BenchmarkGenerationWithoutCache(b *testing.B) {
    engine := setupEngine(false)  // caching disabled

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := engine.Generate(context.Background(), sampleFCS, "/tmp/output")
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkGenerationWithCache(b *testing.B) {
    engine := setupEngine(true)  // caching enabled

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := engine.Generate(context.Background(), sampleFCS, "/tmp/output")
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkTokenUsage(b *testing.B) {
    recorder := &TokenRecorder{}
    engine := setupEngineWithRecorder(recorder)

    _, _ = engine.Generate(context.Background(), sampleFCS, "/tmp/output")

    b.ReportMetric(float64(recorder.TotalInputTokens), "input_tokens")
    b.ReportMetric(float64(recorder.TotalOutputTokens), "output_tokens")
    b.ReportMetric(recorder.CacheHitRate, "cache_hit_rate")
}
```

## Metrics Collection

Track optimization effectiveness:

```go
// internal/metrics/collector.go
type OptimizationMetrics struct {
    // Token Efficiency
    BaselineTokens    int64
    OptimizedTokens   int64
    TokenReduction    float64  // %

    // Caching
    CacheHitRate      float64
    TokensCached      int64
    CostSavings       float64  // USD

    // Batching
    TotalFiles        int
    BatchedFiles      int
    BatchReduction    float64  // %

    // Templates
    TemplateFiles     int
    LLMFiles          int
    TemplateRatio     float64  // %

    // Performance
    BaselineDuration  time.Duration
    OptimizedDuration time.Duration
    SpeedImprovement  float64  // %
}

func (m *MetricsCollector) Report() string {
    return fmt.Sprintf(`
Performance Optimization Report
================================

Token Efficiency:
  Baseline: %d tokens
  Optimized: %d tokens
  Reduction: %.1f%%

Caching:
  Hit Rate: %.1f%%
  Tokens Cached: %d
  Cost Savings: $%.4f

Batching:
  Total Files: %d
  Batched: %d
  API Call Reduction: %.1f%%

Templates:
  Template-Generated: %d
  LLM-Generated: %d
  Template Usage: %.1f%%

Performance:
  Baseline Duration: %s
  Optimized Duration: %s
  Speed Improvement: %.1f%%
`, m.BaselineTokens, m.OptimizedTokens, m.TokenReduction,
   m.CacheHitRate, m.TokensCached, m.CostSavings,
   m.TotalFiles, m.BatchedFiles, m.BatchReduction,
   m.TemplateFiles, m.LLMFiles, m.TemplateRatio,
   m.BaselineDuration, m.OptimizedDuration, m.SpeedImprovement)
}
```

## Configuration

Add optimization controls to config:

```yaml
# config/gocreator.yaml
optimization:
  # Token efficiency
  prompt_caching:
    enabled: true
    providers:
      anthropic: true
      openai: false  # Not yet available

  batching:
    enabled: true
    max_batch_size: 5
    min_similarity: 0.7

  templates:
    enabled: true
    directory: "./templates"

  context_filtering:
    enabled: true
    max_entities_per_call: 10

  # Performance
  parallelization:
    enabled: true
    max_workers: 4

  incremental:
    enabled: true
    state_file: ".gocreator/state.json"

  # UX
  progress:
    enabled: true
    update_interval: 2s

  dry_run:
    show_token_estimates: true
    show_cost_estimates: true

  resume:
    enabled: true
    checkpoint_dir: ".gocreator/checkpoints"
    auto_checkpoint_interval: 30s
```

## Migration Path

1. **Week 1**: Templates + Context Filtering (Quick wins)
2. **Week 2**: Anthropic Caching (High impact)
3. **Week 3**: Batching (Medium complexity)
4. **Week 4**: Parallel Execution (Fix dependencies)
5. **Week 5**: Incremental + Progress (Polish)
6. **Week 6**: Dry-run + Resume (Reliability)

## Success Validation

After each phase, validate with real projects:

```bash
# Baseline measurement
gocreator generate --spec sample-crud.yaml --no-optimization --metrics baseline.json

# Optimized measurement
gocreator generate --spec sample-crud.yaml --metrics optimized.json

# Compare
gocreator compare-metrics baseline.json optimized.json
# Should show:
# ✓ Token reduction: 72%
# ✓ Cost reduction: 68%
# ✓ Speed improvement: 45%
# ✓ Cache hit rate: 81%
```
