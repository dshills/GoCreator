# Performance & Efficiency Optimization

**Status**: Specification Complete
**Created**: 2025-11-18
**Branch**: `003-performance-optimization`

## Executive Summary

This feature addresses the three most critical optimization needs for GoCreator:

1. **Token Efficiency** → 60-80% cost reduction through caching and smart prompting
2. **Performance** → 40-60% faster execution through parallelization and templates
3. **Usability** → Significantly better developer experience with real-time feedback

## Expected Impact

### Cost Reduction
- **Baseline**: Medium project (15 files) costs ~$0.80 per generation
- **Optimized**: Same project costs ~$0.25 with prompt caching
- **Annual Savings**: For 100 generations/month → Save $660/year

### Speed Improvement
- **Baseline**: Medium project takes ~150 seconds
- **Optimized**: Same project takes ~60-75 seconds
- **Time Saved**: 50%+ faster iterations

### Developer Experience
- Real-time progress instead of black-box execution
- Cost estimates before committing
- Resume interrupted generations
- Clear error messages with actionable steps

## Quick Start

Read the documents in this order:

1. **[spec.md](./spec.md)** - Full specification with user stories and requirements
2. **[implementation-guide.md](./implementation-guide.md)** - Technical implementation details
3. This README - High-level overview

## Key Optimizations

### 1. Prompt Caching (Highest ROI)

**Problem**: Each LLM call sends the full FCS schema and examples, wasting tokens

**Solution**: Use Anthropic's prompt caching to mark stable content as cacheable

**Impact**:
- 60-80% reduction in input tokens
- 65%+ cost reduction
- Cache hit rate: 75%+

**Example**:
```
Before caching:
- Phase 1: 8,000 input tokens
- Phase 2: 8,000 input tokens
- Phase 3: 8,000 input tokens
- Total: 24,000 tokens

After caching:
- Phase 1: 8,000 input tokens (initial)
- Phase 2: 1,600 tokens (80% cached)
- Phase 3: 1,600 tokens (80% cached)
- Total: 11,200 tokens (53% reduction)
```

### 2. Template-Based Boilerplate

**Problem**: Using expensive LLM calls for predictable files like go.mod, Dockerfile

**Solution**: Generate boilerplate from Go templates without LLM calls

**Impact**:
- Eliminate 15-25% of LLM calls
- Instant generation for standard files
- 100% consistent formatting

**Files to template**:
- `go.mod` - Module definition
- `.gitignore` - Standard Go patterns
- `Dockerfile` - Multi-stage Go builds
- `Makefile` - Common tasks
- `README.md` - Project skeleton

### 3. Batch Generation

**Problem**: Making 10 separate LLM calls for 10 similar entity files

**Solution**: Group similar files and generate multiple per call

**Impact**:
- 70%+ reduction in API calls for CRUD apps
- Consistent structure across similar files
- Faster execution

**Example**:
```
Before batching:
- Call 1: Generate User entity (1 file)
- Call 2: Generate Product entity (1 file)
- ...
- Call 10: Generate Order entity (1 file)
- Total: 10 API calls

After batching:
- Call 1: Generate User, Product, Order entities (3 files)
- Call 2: Generate Payment, Shipping, Invoice entities (3 files)
- Call 3: Generate Customer, Supplier, Category entities (3 files)
- Call 4: Generate Inventory entity (1 file)
- Total: 4 API calls (60% reduction)
```

### 4. Smart Context Filtering

**Problem**: Sending entire FCS (50 entities) when only 3 are relevant

**Solution**: Include only relevant entities and their dependencies per file

**Impact**:
- 50%+ reduction in prompt size
- Better LLM focus and quality
- Reduced hallucination risk

### 5. Parallel Execution

**Problem**: Generating 4 independent phases sequentially

**Solution**: Execute compatible phases concurrently with goroutines

**Impact**:
- 40-60% faster for projects with parallelizable tasks
- Better CPU utilization
- Reduced wall-clock time

### 6. Incremental Regeneration

**Problem**: Changing 1 entity requires regenerating entire project

**Solution**: Track dependencies and regenerate only affected files

**Impact**:
- 80%+ faster for localized changes
- Iteration speed dramatically improved
- Lower cost for spec refinement

### 7. Streaming Progress

**Problem**: Black-box execution with no visibility

**Solution**: Real-time progress bars, metrics, and ETA

**Impact**:
- User confidence during long operations
- Early detection of issues
- Better understanding of costs

## Implementation Priority

### Phase 1: Quick Wins (Weeks 1-2)
Focus on highest ROI, lowest complexity:

✅ **Week 1**: Template-based boilerplate + Context filtering
✅ **Week 2**: Anthropic prompt caching

**Expected Results**:
- 50-60% token reduction
- 15-25% faster execution
- Minimal code changes

### Phase 2: Core Performance (Weeks 3-4)
Medium complexity, high impact:

🔄 **Week 3**: Batch generation
🔄 **Week 4**: Parallel execution (fix langgraph-go or workaround)

**Expected Results**:
- 70% fewer API calls for CRUD apps
- 40-50% faster execution
- Better resource utilization

### Phase 3: Polish (Weeks 5-6)
Usability and reliability:

📋 **Week 5**: Incremental regeneration + Streaming progress
📋 **Week 6**: Dry-run mode + Resume capability

**Expected Results**:
- 80%+ faster iterations
- Excellent developer experience
- Production-ready reliability

## Validation Strategy

### Benchmark Projects

Create 3 test projects representing common use cases:

1. **CRUD API** (15 entities, REST handlers)
   - Validates batching efficiency
   - Target: 70%+ batch reduction

2. **Microservice** (10 services, gRPC)
   - Validates parallel execution
   - Target: 50%+ speed improvement

3. **Library** (20 packages, comprehensive tests)
   - Validates caching effectiveness
   - Target: 75%+ cache hit rate

### Metrics to Track

```yaml
baseline:
  tokens_input: 85,000
  tokens_output: 12,000
  api_calls: 45
  duration: 150s
  cost: $0.82

optimized:
  tokens_input: 28,000      # 67% reduction ✓
  tokens_cached: 57,000     # 67% of baseline
  tokens_output: 12,000     # Same (no change in output)
  api_calls: 12             # 73% reduction ✓
  duration: 68s             # 55% improvement ✓
  cost: $0.24               # 71% reduction ✓

  cache_hit_rate: 78%       # >75% ✓
  template_files: 6         # 40% of files
  batched_files: 9          # 60% of LLM files
```

## Architecture Changes

### New Components

```
internal/
  generate/
    templates/          # NEW: Template-based generation
      boilerplate.go
      templates/
        go.mod.tmpl
        dockerfile.tmpl
    batcher.go          # NEW: Batch similar files
    context_filter.go   # NEW: Smart context filtering
    incremental.go      # Enhanced: Better delta detection

  workflow/
    parallel.go         # NEW: Parallel task executor

  cli/
    progress.go         # NEW: Streaming progress

pkg/
  llm/
    cache.go            # Enhanced: Provider-specific caching
    anthropic.go        # Enhanced: Prompt caching support
```

### Configuration Changes

New `optimization` section in config:

```yaml
optimization:
  prompt_caching:
    enabled: true
    provider_support:
      anthropic: true

  batching:
    enabled: true
    max_batch_size: 5

  templates:
    enabled: true
    template_dir: "./templates"

  parallelization:
    enabled: true
    max_workers: 4

  incremental:
    enabled: true

  progress:
    enabled: true
```

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Provider API changes | Medium | High | Abstract behind interfaces |
| Cache invalidation bugs | Low | Medium | Conservative TTL, checksums |
| Parallel race conditions | Low | High | Thorough testing, file locks |
| Batch parsing errors | Medium | Medium | Robust JSON parsing, fallback to individual |
| Template limitations | Low | Low | Support custom templates |

## Success Criteria

### Must Achieve (Phase 1-2)

- ✅ 60%+ reduction in token costs
- ✅ 40%+ reduction in execution time
- ✅ 75%+ cache hit rate
- ✅ Zero regressions in output quality

### Should Achieve (Phase 3)

- ✅ Real-time progress for all operations
- ✅ Resume capability for long operations
- ✅ Dry-run mode with accurate estimates
- ✅ Comprehensive metrics reporting

### Nice to Have (Future)

- 📋 Distributed caching (Redis) for teams
- 📋 A/B testing different prompt strategies
- 📋 Auto-tuning batch sizes and parallelism
- 📋 Cost budgets and alerts

## Dependencies

### Required

- `langgraph-go` v0.3.1+ (fix concurrency bug)
- Anthropic SDK with prompt caching
- `golang.org/x/sync/errgroup` (parallel execution)

### Optional

- `cheggaaa/pb/v3` (progress bars)
- `fatih/color` (colored output)
- `olekukonko/tablewriter` (metrics tables)

## Migration Guide

### For Users

No breaking changes. Optimizations are opt-in via config:

```yaml
# Gradually enable optimizations
optimization:
  templates: true       # Start here (no risk)
  prompt_caching: true  # Then this (high value)
  batching: false       # Test carefully
  parallelization: false # Last (most complex)
```

### For Contributors

New interfaces to implement:

```go
type TemplateGenerator interface {
    Generate(name string, data interface{}) (string, error)
}

type BatchGenerator interface {
    Batch(tasks []Task) ([]Patch, error)
}

type ProgressReporter interface {
    Report(event Event)
}
```

## Next Steps

1. Review and approve specification
2. Create implementation branch: `003-performance-optimization`
3. Implement Phase 1 (weeks 1-2): Templates + Caching
4. Measure baseline vs. optimized performance
5. Iterate on remaining phases based on results

## Questions?

- See [spec.md](./spec.md) for detailed requirements
- See [implementation-guide.md](./implementation-guide.md) for code examples
- Open an issue for clarifications

---

**Estimated Effort**: 6 weeks (1 developer)
**Expected ROI**: 65% cost reduction, 50% speed improvement
**Risk Level**: Low-Medium (incremental rollout)
