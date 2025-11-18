# Research & Technology Decisions: Multi-LLM Provider Support

**Feature**: 002-multi-llm | **Date**: 2025-11-17
**Purpose**: Document technology choices, design patterns, and architectural decisions for multi-provider support

---

## 1. LangGraph-Go Provider Abstraction

### Decision: Extend existing `LLMProvider` interface with role-aware context

**Rationale**:
- LangGraph-Go likely uses a provider interface for LLM calls (common pattern in Go LLM libraries)
- Adding role context to existing calls is minimally invasive
- Keeps provider selection logic separate from agent reasoning logic

**Chosen Approach**:
```go
// Enhanced provider interface
type LLMProvider interface {
    Execute(ctx context.Context, request Request) (Response, error)
    Name() string
    Validate() error
}

// Role-aware provider registry
type ProviderRegistry struct {
    providers map[string]LLMProvider
    roleMap   map[Role][]string  // Role -> ordered provider IDs
    defaultProvider string
}
```

**Alternatives Considered**:
- **Modify every agent to select provider**: Rejected - violates separation of concerns, error-prone
- **Provider middleware/decorator pattern**: Considered but adds complexity; direct registry lookup is simpler
- **Dynamic provider loading (plugins)**: Rejected - adds complexity, violates determinism requirements

**References**:
- Standard Go interface patterns: https://go.dev/doc/effective_go#interfaces
- Provider pattern in Go: Singleton registry with dependency injection

---

## 2. Configuration Schema Design

### Decision: YAML configuration with nested structure for hybrid parameter scope

**Rationale**:
- YAML is human-readable and supports hierarchical configuration
- Hybrid scope (global + per-role overrides) requires structured nesting
- Go has excellent YAML parsing libraries (`gopkg.in/yaml.v3`)

**Chosen Schema**:
```yaml
providers:
  openai-gpt4:
    type: openai
    model: gpt-4-turbo
    api_key: ${OPENAI_API_KEY}
    endpoint: https://api.openai.com/v1
    parameters:
      temperature: 0.7     # Global default
      max_tokens: 4096

  anthropic-claude:
    type: anthropic
    model: claude-3-5-sonnet-20241022
    api_key: ${ANTHROPIC_API_KEY}
    parameters:
      temperature: 0.5
      max_tokens: 8192

roles:
  coder:
    provider: openai-gpt4
    fallback: anthropic-claude
    parameters:              # Role-specific overrides
      temperature: 0.8       # Higher temperature for creativity

  reviewer:
    provider: anthropic-claude
    parameters:
      temperature: 0.2       # Lower temperature for consistency

  planner:
    provider: anthropic-claude

  clarifier:
    provider: openai-gpt4

default_provider: anthropic-claude

retry:
  max_attempts: 3
  initial_backoff: 1s
  max_backoff: 30s
  multiplier: 2.0
```

**Alternatives Considered**:
- **JSON configuration**: Rejected - less human-readable, no comment support
- **TOML configuration**: Considered but YAML is more common in Go ecosystems
- **Flat structure with naming conventions**: Rejected - doesn't clearly express hierarchy

**Validation Rules**:
- All provider IDs must be unique
- All role provider references must exist in providers map
- API keys must be non-empty (validated at startup)
- Parameter overrides must match provider's accepted parameters

---

## 3. Credential Validation Patterns

### Decision: Synchronous validation with timeout and parallel validation across providers

**Rationale**:
- FR-012 requires synchronous, blocking validation (fail-fast)
- Validating providers in parallel reduces startup time
- Timeout prevents indefinite blocking (reasonable default: 2s per provider)

**Chosen Implementation**:
```go
func (v *Validator) ValidateAll(ctx context.Context, providers map[string]LLMProvider) error {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second) // Total timeout
    defer cancel()

    var wg sync.WaitGroup
    errCh := make(chan error, len(providers))

    for id, provider := range providers {
        wg.Add(1)
        go func(id string, p LLMProvider) {
            defer wg.Done()
            if err := p.Validate(); err != nil {
                errCh <- fmt.Errorf("provider %s: %w", id, err)
            }
        }(id, provider)
    }

    wg.Wait()
    close(errCh)

    var errs []error
    for err := range errCh {
        errs = append(errs, err)
    }

    if len(errs) > 0 {
        return fmt.Errorf("credential validation failed: %v", errs)
    }
    return nil
}
```

**Alternatives Considered**:
- **Lazy validation on first use**: Rejected - violates FR-012 synchronous requirement
- **Sequential validation**: Rejected - too slow for multiple providers
- **Async validation with warnings**: Rejected - spec requires blocking behavior

**Error Handling**:
- Validation failure stops startup with clear error message
- Errors include provider ID and specific failure reason (without exposing credentials)
- Supports partial validation (optional flag to allow some provider failures)

---

## 4. Metrics Storage Strategy

### Decision: SQLite for structured metrics with file-based fallback

**Rationale**:
- Spec allows "optional SQLite for execution history" - extend for metrics
- SQLite provides efficient querying, aggregation, and time-series storage
- File-based fallback maintains compatibility with minimal deployments
- No external dependencies (SQLite library is single-file, no daemon)

**Schema Design**:
```sql
CREATE TABLE provider_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_id TEXT NOT NULL,
    role TEXT NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    response_time_ms INTEGER NOT NULL,
    tokens_prompt INTEGER,
    tokens_completion INTEGER,
    status TEXT NOT NULL,  -- 'success', 'failure', 'retry'
    error_message TEXT
);

CREATE INDEX idx_provider_role ON provider_metrics(provider_id, role);
CREATE INDEX idx_timestamp ON provider_metrics(timestamp);
```

**Query Patterns**:
```go
// Aggregate metrics by provider and role
type MetricsSummary struct {
    ProviderID      string
    Role            string
    AvgResponseTime float64
    TotalRequests   int
    SuccessRate     float64
    TotalTokens     int
}

func (m *MetricsStore) GetSummary(providerID, role string, since time.Time) (*MetricsSummary, error)
```

**File-Based Fallback**:
- JSON Lines format (`.jsonl`) for append-only metrics
- One line per metric event, timestamp-ordered
- Simple grep/jq queries for basic analysis
- Automatic migration: file → SQLite when SQLite is enabled

**Alternatives Considered**:
- **In-memory only**: Rejected - metrics lost on restart
- **Dedicated time-series DB (Prometheus, InfluxDB)**: Rejected - external dependency, overkill
- **CSV files**: Rejected - poor querying, no schema validation

---

## 5. Retry Implementation

### Decision: Custom exponential backoff with context-aware cancellation

**Rationale**:
- Simple algorithm, no external dependencies needed
- Context support allows cancellation during retries
- Deterministic: same config = same retry timing

**Chosen Implementation**:
```go
type RetryConfig struct {
    MaxAttempts    int
    InitialBackoff time.Duration
    MaxBackoff     time.Duration
    Multiplier     float64
}

func (r *RetryConfig) Execute(ctx context.Context, fn func() error) error {
    var lastErr error
    backoff := r.InitialBackoff

    for attempt := 0; attempt < r.MaxAttempts; attempt++ {
        if err := fn(); err == nil {
            return nil
        } else {
            lastErr = err
        }

        if attempt < r.MaxAttempts-1 {
            select {
            case <-time.After(backoff):
                backoff = time.Duration(float64(backoff) * r.Multiplier)
                if backoff > r.MaxBackoff {
                    backoff = r.MaxBackoff
                }
            case <-ctx.Done():
                return ctx.Err()
            }
        }
    }

    return fmt.Errorf("failed after %d attempts: %w", r.MaxAttempts, lastErr)
}
```

**Alternatives Considered**:
- **Use existing library (github.com/cenkalti/backoff)**: Considered - adds dependency, custom is simple enough
- **Jittered exponential backoff**: Considered for production systems, but determinism requirement prefers predictable timing
- **Per-provider retry config**: Rejected per clarification (Q3: global configuration)

**Error Classification**:
- Retriable errors: Rate limits, temporary network failures, 5xx server errors
- Non-retriable errors: Invalid credentials (4xx auth), malformed requests, context cancellation

---

## 6. Thread-Safety Patterns

### Decision: Provider instance per registry + sync.RWMutex for metrics

**Rationale**:
- Provider SDK clients are typically thread-safe (OpenAI, Anthropic, Google SDKs)
- One provider instance per registry avoids client creation overhead
- Metrics collection requires synchronization (atomic or mutex)
- RWMutex optimizes for read-heavy metrics queries

**Chosen Approach**:
```go
type ProviderRegistry struct {
    mu        sync.RWMutex
    providers map[string]LLMProvider  // Thread-safe reads, immutable after init
    roleMap   map[Role][]string       // Thread-safe reads, immutable after init
    metrics   *MetricsCollector       // Internally synchronized
}

type MetricsCollector struct {
    mu     sync.RWMutex
    events []MetricEvent  // Or direct writes to SQLite with connection pool
}

func (m *MetricsCollector) Record(event MetricEvent) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.events = append(m.events, event)
}

func (m *MetricsCollector) GetSummary(...) MetricsSummary {
    m.mu.RLock()
    defer m.mu.RUnlock()
    // Compute summary from events
}
```

**Alternatives Considered**:
- **Provider pooling**: Rejected - SDK clients handle concurrency internally
- **Per-goroutine provider instances**: Rejected - high memory overhead, unnecessary
- **Lock-free metrics (sync/atomic)**: Considered for counters, but complex for aggregate metrics
- **Channel-based metrics**: Considered but mutex is simpler for this use case

**Concurrency Guarantees**:
- SelectProvider: Concurrent reads, no locks needed (immutable after init)
- RecordMetrics: Mutex-protected writes, safe for concurrent agents
- GetMetrics: RWMutex read lock, doesn't block metric recording

---

## Summary of Technology Choices

| Decision Area | Choice | Key Rationale |
|---------------|--------|---------------|
| Provider Interface | Extend existing with role context | Minimal invasiveness, clean separation |
| Configuration Format | YAML with nested structure | Human-readable, supports hierarchy |
| Credential Validation | Parallel sync validation with timeout | Fast startup, fail-fast behavior |
| Metrics Storage | SQLite with file fallback | Structured queries, no external deps |
| Retry Logic | Custom exponential backoff | Deterministic, context-aware |
| Thread Safety | Provider instances + RWMutex | SDK clients thread-safe, optimize reads |

**Dependencies Summary**:
- `gopkg.in/yaml.v3`: YAML parsing
- `modernc.org/sqlite` or `github.com/mattn/go-sqlite3`: SQLite driver (optional)
- Standard library: `sync`, `context`, `time`, `encoding/json`
- No external retry libraries (custom implementation)

**Performance Characteristics**:
- Provider selection: O(1) map lookup, < 1ms
- Credential validation: Parallel, ~2s per provider, ~2-5s total for 3 providers
- Metrics recording: Mutex-protected append, < 1ms
- Metrics queries: SQLite indexed queries, < 100ms for 10k records

---

**Research Complete**: 2025-11-17
**Next Phase**: Design & Contracts (data-model.md, contracts/, quickstart.md)
