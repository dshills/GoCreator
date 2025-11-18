# Feature Specification: Performance & Efficiency Optimization

**Feature Branch**: `003-performance-optimization`
**Created**: 2025-11-18
**Status**: Draft
**Purpose**: Reduce token consumption, improve execution speed, and enhance usability

## Executive Summary

This specification addresses three critical optimization areas for GoCreator:

1. **Token Efficiency** - Reduce LLM token consumption by 60-80% through prompt caching, batching, and smarter context management
2. **Performance** - Improve generation speed by 40-60% through parallelization, template-based generation, and incremental processing
3. **Usability** - Enhance developer experience through better CLI, interactive modes, and real-time feedback

**Expected Impact**:
- **Cost Reduction**: 60-80% reduction in LLM API costs
- **Speed Improvement**: 40-60% faster generation times for medium projects
- **User Experience**: Significantly improved feedback, error handling, and workflow visibility

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Efficient Token Usage with Prompt Caching (Priority: P1)

Users generating code with GoCreator want to minimize LLM API costs while maintaining output quality. The system should leverage provider-native prompt caching (Claude's prompt caching, OpenAI's prompt caching) to reduce redundant token processing.

**Why this priority**: Token costs are the primary operational expense. This delivers immediate ROI and enables cost-effective scaling.

**Independent Test**: Execute the same generation workflow twice and verify that the second execution uses cached prompts, reducing input token consumption by 80%+ for repeated content.

**Acceptance Scenarios**:

1. **Given** an FCS specification, **When** generating code with Anthropic Claude, **Then** the system marks stable portions (FCS schema, examples) as cacheable and achieves 80%+ cache hit rate on subsequent calls
2. **Given** a multi-phase generation workflow, **When** each phase references the same FCS, **Then** the FCS is cached after the first phase and reused in subsequent phases with minimal token cost
3. **Given** an incremental regeneration request, **When** only specific files change, **Then** the system reuses cached context about unchanged portions of the specification
4. **Given** completion of a generation workflow, **When** reviewing metrics, **Then** users can see cache hit rates, tokens saved, and cost reduction per provider

**Token Reduction Target**: 60-80% reduction in input tokens across multi-phase workflows

---

### User Story 2 - Batch Code Generation (Priority: P1)

Users generating projects with many similar files (CRUD operations, entity models, REST endpoints) want the system to batch similar generation tasks into single LLM calls rather than making separate calls per file.

**Why this priority**: Many projects contain repetitive patterns. Batching reduces LLM round-trips and context switching overhead.

**Independent Test**: Generate a project with 10 similar entity files and verify that the system makes 1-2 batched calls instead of 10 individual calls.

**Acceptance Scenarios**:

1. **Given** a generation plan with 10 entity model files, **When** code generation executes, **Then** the system batches entities with similar structure into groups of 3-5 per LLM call
2. **Given** a REST API project with 8 CRUD endpoints, **When** generating handler code, **Then** the system uses a single batched call with template instructions instead of 8 separate calls
3. **Given** batched generation results, **When** parsing LLM responses, **Then** the system correctly splits multi-file responses into individual patches
4. **Given** a generation failure in a batch, **When** retry logic executes, **Then** the system can retry individual files within the batch without regenerating successful files

**Efficiency Target**: 70%+ reduction in LLM API calls for projects with repetitive structure

---

### User Story 3 - Template-Based Boilerplate Generation (Priority: P2)

Users generating standard Go project boilerplate (go.mod, .gitignore, Makefile, Dockerfile) want these files created instantly from templates rather than using expensive LLM calls for predictable content.

**Why this priority**: Boilerplate files are deterministic and don't require LLM creativity. This eliminates unnecessary LLM usage.

**Independent Test**: Generate a new project and verify that standard boilerplate files (go.mod, .gitignore, Dockerfile, Makefile, README template) are created without LLM calls.

**Acceptance Scenarios**:

1. **Given** a new Go project generation request, **When** the planner identifies standard boilerplate files, **Then** the system marks them for template-based generation instead of LLM generation
2. **Given** template-based generation execution, **When** creating go.mod, **Then** the system uses Go's native template system with FCS metadata (module name, Go version) without LLM calls
3. **Given** a custom .gitignore template in project configuration, **When** generating .gitignore, **Then** the system merges custom rules with standard Go gitignore patterns
4. **Given** generation completion, **When** reviewing metrics, **Then** users can see which files used templates vs. LLM generation and the cost savings

**Cost Reduction Target**: Eliminate 15-25% of LLM calls by using templates for standard files

---

### User Story 4 - Parallel Phase Execution (Priority: P2)

Users generating large projects want independent generation tasks to execute in parallel (e.g., generating model files and configuration files simultaneously) to reduce total execution time.

**Why this priority**: Modern systems have multi-core CPUs. Sequential processing wastes resources and time.

**Independent Test**: Generate a project with 4 independent phases and verify that compatible phases execute concurrently, reducing total time by 40%+.

**Acceptance Scenarios**:

1. **Given** a generation plan with 4 phases where phase 2 and 3 have no dependencies on each other, **When** execution reaches phase 2, **Then** the system executes phases 2 and 3 in parallel
2. **Given** parallel task execution, **When** multiple LLM calls run concurrently, **Then** the system respects provider rate limits and connection pools
3. **Given** a dependency graph with 8 tasks where 4 can run in parallel, **When** executing the graph, **Then** the system achieves 50%+ time reduction compared to sequential execution
4. **Given** concurrent LLM provider usage (OpenAI for coder, Anthropic for reviewer), **When** parallel tasks execute, **Then** each provider connection pool is utilized independently

**Performance Target**: 40-60% reduction in wall-clock time for projects with parallelizable tasks

---

### User Story 5 - Incremental Regeneration Optimization (Priority: P2)

Users modifying a small portion of their specification want the system to regenerate only affected files rather than the entire project, minimizing both time and cost.

**Why this priority**: Real-world workflows involve iterative refinement. Smart incremental updates dramatically improve iteration speed.

**Independent Test**: Change a single entity definition and verify that only affected files (entity, tests, related services) are regenerated.

**Acceptance Scenarios**:

1. **Given** an existing generated project, **When** a user modifies one entity in the FCS, **Then** the system identifies only files that depend on that entity for regeneration
2. **Given** incremental regeneration execution, **When** the dependency graph is computed, **Then** the system skips files with unchanged inputs and valid checksums
3. **Given** a specification change affecting 3 out of 20 files, **When** regenerating, **Then** the system makes 3-5 LLM calls instead of 20+
4. **Given** incremental regeneration with caching, **When** generating, **Then** the system uses cached context for unchanged portions of the FCS

**Efficiency Target**: 80%+ reduction in regeneration time and cost for localized changes

---

### User Story 6 - Streaming Progress Feedback (Priority: P3)

Users executing long-running generation workflows want real-time visibility into progress (current phase, files generated, estimated time remaining) rather than waiting for completion.

**Why this priority**: Enhances user experience and builds confidence during long operations. Helps identify bottlenecks.

**Independent Test**: Start a generation workflow and verify that progress updates stream to the terminal every 2-3 seconds with phase status, file count, and time estimates.

**Acceptance Scenarios**:

1. **Given** a generation workflow execution, **When** processing begins, **Then** the CLI displays a progress bar with current phase, files completed, and ETA
2. **Given** multi-phase generation, **When** each phase completes, **Then** the system displays a summary (time taken, files generated, tokens used)
3. **Given** a long-running LLM call (>10 seconds), **When** waiting for response, **Then** the CLI displays an animated spinner with elapsed time
4. **Given** generation completion, **When** displaying results, **Then** the system shows a breakdown of time and cost per phase with optimization opportunities highlighted

**User Experience Target**: Real-time feedback for all operations longer than 3 seconds

---

### User Story 7 - Smart Context Window Management (Priority: P3)

Users generating complex projects with large specifications want the system to intelligently manage LLM context windows by including only relevant portions of the FCS for each task.

**Why this priority**: Modern LLMs have large context windows, but filling them completely is expensive and can degrade output quality.

**Independent Test**: Generate a file requiring entity A and verify that the LLM prompt includes entity A's definition but excludes unrelated entities B-Z.

**Acceptance Scenarios**:

1. **Given** a generation task for a specific file, **When** building the LLM prompt, **Then** the system includes only FCS sections relevant to that file (dependencies, related entities)
2. **Given** an FCS with 50 entities, **When** generating code for entity "User", **Then** the prompt includes User definition and its direct dependencies (5-10 entities) but excludes unrelated entities
3. **Given** a code review task, **When** building the review prompt, **Then** the system includes the file being reviewed plus its immediate dependencies but excludes the full FCS
4. **Given** context window optimization, **When** generating files, **Then** the system reduces average prompt size by 50%+ while maintaining output quality

**Efficiency Target**: 50%+ reduction in prompt tokens through smart context filtering

---

## Functional Requirements *(mandatory)*

### Token Efficiency Requirements

**FR-001**: System MUST support Anthropic's prompt caching API, marking stable FCS portions as cacheable with `cache_control` headers

**FR-002**: System MUST support OpenAI's prompt caching mechanism when available, reusing conversation context across related calls

**FR-003**: System MUST track cache hit rates per provider and display them in generation metrics

**FR-004**: System MUST batch similar file generation tasks (entities, CRUD handlers, tests) into single LLM calls when files share >70% structural similarity

**FR-005**: System MUST implement template-based generation for standard boilerplate files (go.mod, .gitignore, Dockerfile, Makefile, basic README) without LLM calls

**FR-006**: System MUST compute prompt token estimates before LLM calls and warn users when prompts exceed 80% of provider context windows

**FR-007**: System MUST implement smart context filtering, including only FCS sections relevant to the current generation task (max 40% of full FCS per call)

### Performance Requirements

**FR-008**: System MUST execute independent generation phases in parallel, utilizing up to 4 concurrent goroutines per workflow

**FR-009**: System MUST respect provider rate limits through token bucket rate limiting (configurable per provider)

**FR-010**: System MUST implement connection pooling for HTTP clients to LLM providers with keepalive enabled

**FR-011**: System MUST support incremental regeneration by tracking file checksums and dependencies in a state file

**FR-012**: System MUST identify unchanged portions of FCS across regeneration runs and reuse cached generation artifacts

**FR-013**: System MUST complete medium-sized project generation (15 files, 2000 LOC) in under 90 seconds on modern hardware

### Usability Requirements

**FR-014**: System MUST provide streaming progress updates during generation with phase name, files completed, and estimated time remaining

**FR-015**: System MUST display real-time token usage and estimated cost during execution

**FR-016**: System MUST provide a `--dry-run` mode that shows the generation plan and estimated cost without executing

**FR-017**: System MUST support interactive confirmation before expensive operations (>10,000 tokens estimated)

**FR-018**: System MUST provide detailed error messages with actionable recommendations when generation fails

**FR-019**: System MUST generate and save execution reports (JSON/HTML) with metrics breakdown (time per phase, tokens per task, cache hit rates)

**FR-020**: System MUST support `--resume` flag to continue interrupted generation workflows from the last successful checkpoint

## Success Criteria *(mandatory)*

### Quantitative Metrics

1. **Token Reduction**: Achieve 60-80% reduction in total input tokens for multi-phase workflows through caching and batching
2. **Cost Savings**: Reduce average generation cost by 65%+ for typical projects (vs. baseline without optimizations)
3. **Speed Improvement**: Complete medium project generation 40-60% faster through parallelization and templates
4. **Cache Hit Rate**: Achieve 75%+ cache hit rate for FCS content across multi-phase workflows
5. **Batch Efficiency**: Reduce LLM API calls by 70%+ for projects with repetitive structures (CRUD apps)
6. **Incremental Performance**: Complete incremental regeneration 80%+ faster when only 10-20% of files change

### Qualitative Metrics

1. **User Confidence**: Real-time progress feedback shows current status, eliminating "black box" execution concerns
2. **Error Clarity**: Error messages provide clear actionable steps, reducing user troubleshooting time
3. **Cost Transparency**: Users can see estimated costs before execution and actual costs after completion
4. **Developer Experience**: Dry-run mode allows experimentation without cost, improving workflow iteration
5. **Operational Reliability**: Resume capability prevents data loss from interrupted executions

## Key Entities *(if data is involved)*

### OptimizationConfig

Configuration for optimization features:

```yaml
optimization:
  prompt_caching:
    enabled: true
    provider_support:
      anthropic: true
      openai: true  # When available
    cache_stable_content: true

  batching:
    enabled: true
    max_batch_size: 5
    similarity_threshold: 0.7  # 70% structural similarity

  templates:
    enabled: true
    template_dir: "./templates"
    standard_files:
      - go.mod
      - .gitignore
      - Dockerfile
      - Makefile

  parallelization:
    enabled: true
    max_concurrent_tasks: 4
    respect_dependencies: true

  incremental:
    enabled: true
    state_file: ".gocreator/state.json"
    checksum_algorithm: "sha256"

  context_window:
    smart_filtering: true
    max_fcs_percentage: 40  # Include max 40% of FCS per call
    include_dependencies: true
```

### GenerationMetrics

Enhanced metrics tracking:

```go
type GenerationMetrics struct {
    // Timing
    TotalDuration      time.Duration
    PhaseTimings       map[string]time.Duration
    LLMCallDuration    time.Duration
    TemplateGeneration time.Duration

    // Token Usage
    TotalInputTokens   int64
    TotalOutputTokens  int64
    CachedTokens       int64  // Tokens served from cache
    TokensSaved        int64  // Tokens saved by caching

    // Cache Performance
    CacheHitRate       float64
    CacheHits          int
    CacheMisses        int

    // API Efficiency
    TotalLLMCalls      int
    BatchedCalls       int
    TemplateFiles      int
    LLMGeneratedFiles  int

    // Cost
    EstimatedCostUSD   float64
    CostBreakdown      map[string]float64  // Per provider

    // Parallelization
    ParallelPhases     int
    TimesSaved         time.Duration  // Time saved by parallelization
}
```

### CacheEntry

Prompt cache metadata:

```go
type CacheEntry struct {
    Key            string
    ContentHash    string
    ProviderType   ProviderType
    Prompt         string
    CacheControl   CacheControl  // Provider-specific cache metadata
    CreatedAt      time.Time
    LastAccessedAt time.Time
    HitCount       int
    ExpiresAt      time.Time
    TokensSaved    int64
}

type CacheControl struct {
    Type     string  // "ephemeral" for Anthropic
    Breakpoint bool  // Mark cache breakpoint for Anthropic
}
```

### IncrementalState

State for incremental regeneration:

```go
type IncrementalState struct {
    FCSChecksum       string
    GeneratedFiles    map[string]FileState
    DependencyGraph   map[string][]string  // file -> dependencies
    LastGeneration    time.Time
    Checksums         map[string]string    // file -> content checksum
}

type FileState struct {
    Path            string
    Checksum        string
    GeneratedAt     time.Time
    Dependencies    []string  // FCS elements this file depends on
    Template        bool      // True if template-generated
}
```

## Technical Approach *(overview)*

### Architecture Changes

1. **Caching Layer**:
   - Implement provider-specific caching adapters (Anthropic, OpenAI)
   - Add cache key generation based on prompt structure
   - Track cache metrics for reporting

2. **Batch Processor**:
   - Add similarity detector for grouping files
   - Implement multi-file prompt templates
   - Parse and split batched LLM responses

3. **Template Engine**:
   - Integrate Go's `text/template` for boilerplate
   - Create template library for common Go patterns
   - Add template customization support

4. **Parallel Executor**:
   - Fix langgraph-go concurrency bug or work around it
   - Implement goroutine pool with rate limiting
   - Add dependency-aware task scheduler

5. **Incremental Engine**:
   - Build dependency graph from FCS changes
   - Implement file checksum tracking
   - Add smart delta detection

6. **Progress Streaming**:
   - Implement event-based progress tracking
   - Add CLI progress renderer (bars, spinners)
   - Stream metrics to stdout/file

### Migration Strategy

**Phase 1**: Token Efficiency (Weeks 1-2)
- Implement prompt caching for Anthropic
- Add batching for entity generation
- Create template library for boilerplate

**Phase 2**: Performance (Weeks 3-4)
- Fix/workaround langgraph-go concurrency
- Implement parallel phase execution
- Add incremental regeneration

**Phase 3**: Usability (Weeks 5-6)
- Add streaming progress
- Implement dry-run mode
- Add resume capability
- Enhanced error messages

## Dependencies

- **langgraph-go v0.3.1+**: Fix for concurrent execution bug (or implement workaround)
- **Anthropic SDK v0.10+**: Prompt caching support
- **OpenAI SDK v1.20+**: Future prompt caching support
- **golang.org/x/sync/errgroup**: Parallel execution coordination
- **cheggaaa/pb/v3**: Progress bar library
- **spf13/cobra**: Enhanced CLI with progress support

## Non-Functional Requirements

### Performance

- **Latency**: 95th percentile LLM response time < 5 seconds
- **Throughput**: Support 10+ concurrent file generation tasks
- **Memory**: Keep memory usage under 500MB for medium projects

### Reliability

- **Fault Tolerance**: Graceful degradation when caching unavailable
- **Resume**: Support resuming from any failed phase
- **Idempotency**: Identical FCS + config = identical output

### Observability

- **Logging**: Structured logs (JSON) for all optimization decisions
- **Metrics**: Export Prometheus metrics for cache hit rates, latency, cost
- **Tracing**: Distributed tracing for multi-phase workflows

## Security Considerations

- **API Keys**: Never cache prompts containing API keys or secrets
- **Sensitive Data**: Detect and exclude sensitive patterns from cache
- **Rate Limits**: Respect provider limits to avoid account suspension
- **Cost Controls**: Hard limits on maximum tokens per execution

## Testing Strategy

### Unit Tests

- Cache key generation logic
- Batch grouping algorithm
- Template rendering
- Dependency graph calculation
- Token estimation accuracy

### Integration Tests

- End-to-end caching with Anthropic
- Batched file generation
- Parallel phase execution
- Incremental regeneration scenarios
- Progress streaming

### Performance Tests

- Benchmark token reduction (60-80% target)
- Benchmark speed improvement (40-60% target)
- Cache hit rate verification (75%+ target)
- Memory usage profiling

### Cost Tests

- Compare costs: baseline vs. optimized
- Verify cache savings in production workloads
- Test batching efficiency on CRUD projects

## Open Questions

1. Should we implement our own LangGraph-like execution engine to avoid concurrency bugs?
2. What's the optimal batch size for different file types?
3. Should caching be opt-in or opt-out by default?
4. How do we handle breaking changes in cached prompts?
5. Should we support distributed caching (Redis) for team environments?

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Provider caching API changes | High | Abstract caching behind interfaces, support multiple versions |
| Cache invalidation bugs | Medium | Conservative expiration, checksum validation |
| Parallel execution race conditions | High | Thorough testing, file lock mechanisms |
| Template rigidity | Low | Support custom templates, template overrides |
| Cost estimation inaccuracy | Medium | Regular calibration against actual usage |

## References

- [Anthropic Prompt Caching Documentation](https://docs.anthropic.com/claude/docs/prompt-caching)
- [OpenAI Prompt Caching (Beta)](https://platform.openai.com/docs/guides/prompt-caching)
- [Go text/template Package](https://pkg.go.dev/text/template)
- [GoCreator Architecture Whitepaper](../architecture_whitepaper.md)
- [002 Multi-LLM Provider Spec](../002-multi-llm/spec.md)
