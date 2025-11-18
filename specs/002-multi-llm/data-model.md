# Data Model: Multi-LLM Provider Support

**Feature**: 002-multi-llm | **Date**: 2025-11-17
**Purpose**: Define entities, relationships, and data structures for multi-provider support

---

## Core Entities

### 1. ProviderConfig

Represents the configuration for a single LLM provider.

**Attributes**:
- `ID` (string, required): Unique identifier for the provider (e.g., "openai-gpt4", "anthropic-claude")
- `Type` (ProviderType, required): Provider type enum (OpenAI, Anthropic, Google)
- `Model` (string, required): Model identifier (e.g., "gpt-4-turbo", "claude-3-5-sonnet-20241022")
- `APIKey` (string, required, sensitive): Authentication credential (never logged)
- `Endpoint` (string, optional): Custom API endpoint URL (defaults to provider's standard endpoint)
- `Parameters` (map[string]interface{}, optional): Provider-specific global parameters
  - `temperature` (float64, 0.0-2.0): Sampling temperature
  - `max_tokens` (int, > 0): Maximum response tokens
  - `top_p` (float64, 0.0-1.0): Nucleus sampling parameter
  - Additional provider-specific parameters

**Validation Rules**:
- ID must be unique across all providers
- ID must match pattern: `^[a-z0-9][a-z0-9-]*[a-z0-9]$`
- Type must be a valid ProviderType
- APIKey must be non-empty
- Endpoint must be valid URL if provided
- Parameters must be valid JSON-serializable types

**Example**:
```yaml
ID: openai-gpt4
Type: openai
Model: gpt-4-turbo
APIKey: sk-...
Endpoint: https://api.openai.com/v1
Parameters:
  temperature: 0.7
  max_tokens: 4096
```

---

### 2. RoleAssignment

Maps a specialized role to one or more providers with priority order for fallback.

**Attributes**:
- `Role` (Role, required): The specialized role enum (Coder, Reviewer, Planner, Clarifier)
- `PrimaryProvider` (string, required): Primary provider ID to use for this role
- `FallbackProviders` ([]string, optional): Ordered list of fallback provider IDs
- `ParameterOverrides` (map[string]interface{}, optional): Role-specific parameter overrides
  - Overrides global provider parameters for this role
  - Same schema as ProviderConfig.Parameters
  - Only tuning parameters allowed (temperature, max_tokens, etc.)
  - Critical parameters (APIKey, Endpoint, Model) cannot be overridden

**Validation Rules**:
- Role must be a valid Role enum value
- PrimaryProvider must exist in ProviderConfig collection
- All FallbackProviders must exist in ProviderConfig collection
- FallbackProviders must not contain duplicates
- ParameterOverrides must not include critical parameters (APIKey, Endpoint, Model)

**Example**:
```yaml
Role: coder
PrimaryProvider: openai-gpt4
FallbackProviders:
  - anthropic-claude
ParameterOverrides:
  temperature: 0.8  # Higher creativity for code generation
  max_tokens: 8192
```

---

### 3. TaskExecutionContext

Contains execution metadata for a single task within a workflow.

**Attributes**:
- `TaskID` (string, required): Unique identifier for the task
- `Role` (Role, required): The role assigned to this task
- `SelectedProvider` (string, required): The provider ID that executed or will execute this task
- `StartTime` (time.Time, required): Task execution start timestamp
- `EndTime` (time.Time, optional): Task execution completion timestamp (nil if in progress)
- `Status` (TaskStatus, required): Current task status (Pending, Running, Completed, Failed)
- `Attempt` (int, required): Current retry attempt number (1-based)
- `Error` (string, optional): Error message if status is Failed

**Validation Rules**:
- TaskID must be unique within a workflow execution
- SelectedProvider must exist in ProviderConfig collection
- StartTime must be set when Status is Running or later
- EndTime must be >= StartTime if set
- Attempt must be >= 1 and <= MaxRetryAttempts

**Relationships**:
- References ProviderConfig via SelectedProvider
- References RoleAssignment via Role
- Generates ProviderMetrics upon completion

**Example**:
```go
TaskExecutionContext{
    TaskID: "gen-user-model-001",
    Role: Coder,
    SelectedProvider: "openai-gpt4",
    StartTime: time.Now(),
    Status: Running,
    Attempt: 1,
}
```

---

### 4. ProviderMetrics

Tracks performance and usage statistics for a provider-role combination.

**Attributes**:
- `ID` (int64, auto-increment): Unique metric event ID
- `ProviderID` (string, required): Provider that handled the request
- `Role` (Role, required): Role context for the request
- `Timestamp` (time.Time, required): When the metric was recorded
- `ResponseTimeMs` (int64, required): Response time in milliseconds
- `TokensPrompt` (int, optional): Number of tokens in the prompt
- `TokensCompletion` (int, optional): Number of tokens in the completion
- `Status` (MetricStatus, required): Outcome (Success, Failure, Retry)
- `ErrorMessage` (string, optional): Error details if Status is Failure

**Validation Rules**:
- ProviderID must reference a valid ProviderConfig
- ResponseTimeMs must be >= 0
- TokensPrompt must be >= 0 if provided
- TokensCompletion must be >= 0 if provided
- ErrorMessage should be set if Status is Failure

**Indexes** (for query performance):
- `(ProviderID, Role)` - for provider-role aggregations
- `Timestamp` - for time-range queries
- `Status` - for success/failure filtering

**Aggregations** (computed on query):
- `AvgResponseTimeMs`: Average response time for provider-role
- `TotalRequests`: Count of all requests
- `SuccessRate`: Percentage of successful requests
- `TotalTokens`: Sum of prompt + completion tokens
- `ErrorRate`: Percentage of failed requests

**Example**:
```go
ProviderMetrics{
    ProviderID: "openai-gpt4",
    Role: Coder,
    Timestamp: time.Now(),
    ResponseTimeMs: 1250,
    TokensPrompt: 500,
    TokensCompletion: 1500,
    Status: Success,
}
```

---

## Supporting Types

### Enumerations

#### ProviderType
```go
type ProviderType string

const (
    ProviderTypeOpenAI    ProviderType = "openai"
    ProviderTypeAnthropic ProviderType = "anthropic"
    ProviderTypeGoogle    ProviderType = "google"
)
```

#### Role
```go
type Role string

const (
    RoleCoder     Role = "coder"
    RoleReviewer  Role = "reviewer"
    RolePlanner   Role = "planner"
    RoleClarifier Role = "clarifier"
)
```

#### TaskStatus
```go
type TaskStatus string

const (
    TaskStatusPending   TaskStatus = "pending"
    TaskStatusRunning   TaskStatus = "running"
    TaskStatusCompleted TaskStatus = "completed"
    TaskStatusFailed    TaskStatus = "failed"
)
```

#### MetricStatus
```go
type MetricStatus string

const (
    MetricStatusSuccess MetricStatus = "success"
    MetricStatusFailure MetricStatus = "failure"
    MetricStatusRetry   MetricStatus = "retry"
)
```

---

## Entity Relationships

```
┌─────────────────┐
│ ProviderConfig  │
│                 │
│ - ID            │◄───────────┐
│ - Type          │            │
│ - Model         │            │ References
│ - APIKey        │            │
│ - Parameters    │            │
└─────────────────┘            │
                               │
                               │
┌─────────────────┐            │
│ RoleAssignment  │            │
│                 │            │
│ - Role          │            │
│ - PrimaryProvider├───────────┤
│ - Fallbacks[]   ├───────────┘
│ - Overrides     │
└────────┬────────┘
         │
         │ Determines
         │ provider for
         │
         ▼
┌─────────────────┐      Generates     ┌─────────────────┐
│ TaskExecution   │─────────────────►  │ ProviderMetrics │
│ Context         │                    │                 │
│                 │                    │ - ProviderID    │
│ - TaskID        │                    │ - Role          │
│ - Role          │                    │ - ResponseTime  │
│ - SelectedProvider├───────────────┐  │ - Tokens        │
│ - StartTime     │                │  │ - Status        │
│ - Status        │                │  └─────────────────┘
│ - Attempt       │                │
└─────────────────┘                │
                                   │
                                   │ References
                                   │
                                   ▼
                            ┌─────────────────┐
                            │ ProviderConfig  │
                            └─────────────────┘
```

---

## Configuration Object Model

The complete configuration file structure:

```go
type MultiProviderConfig struct {
    Providers       map[string]ProviderConfig  // Provider ID -> Config
    Roles           map[Role]RoleAssignment    // Role -> Assignment
    DefaultProvider string                     // Fallback if role has no assignment
    Retry           RetryConfig                // Global retry configuration
}

type RetryConfig struct {
    MaxAttempts    int           // Maximum retry attempts (default: 3)
    InitialBackoff time.Duration // Initial backoff duration (default: 1s)
    MaxBackoff     time.Duration // Maximum backoff duration (default: 30s)
    Multiplier     float64       // Backoff multiplier (default: 2.0)
}
```

**Validation Rules** (at configuration load time):
1. At least one provider must be defined
2. DefaultProvider must reference a valid provider ID
3. All role assignments must reference valid provider IDs
4. No circular references in fallback chains
5. RetryConfig values must be positive and reasonable (MaxAttempts <= 10, etc.)

---

## Data Flow

### Provider Selection Flow
1. Task arrives with assigned Role
2. Look up RoleAssignment for Role
3. Return PrimaryProvider ID from RoleAssignment
4. If PrimaryProvider fails, iterate through FallbackProviders
5. If all fail, fall back to DefaultProvider
6. If DefaultProvider fails, return error

### Metrics Collection Flow
1. Task execution begins → Create TaskExecutionContext
2. Provider executes request
3. On completion (success or failure):
   - Record response time, tokens, status
   - Create ProviderMetrics entry
   - Update TaskExecutionContext with EndTime and Status
4. Metrics written to storage (SQLite or file)

### Parameter Resolution Flow
1. Start with ProviderConfig.Parameters (global defaults)
2. Apply RoleAssignment.ParameterOverrides if present
3. Resulting parameters passed to provider client
4. Critical parameters (APIKey, Endpoint, Model) always from ProviderConfig

---

## Storage Schema (SQLite)

```sql
-- Provider metrics table
CREATE TABLE provider_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_id TEXT NOT NULL,
    role TEXT NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    response_time_ms INTEGER NOT NULL,
    tokens_prompt INTEGER,
    tokens_completion INTEGER,
    status TEXT NOT NULL CHECK(status IN ('success', 'failure', 'retry')),
    error_message TEXT
);

CREATE INDEX idx_provider_role ON provider_metrics(provider_id, role);
CREATE INDEX idx_timestamp ON provider_metrics(timestamp);
CREATE INDEX idx_status ON provider_metrics(status);

-- Task execution context (optional, for debugging)
CREATE TABLE task_execution_log (
    task_id TEXT PRIMARY KEY,
    role TEXT NOT NULL,
    selected_provider TEXT NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME,
    status TEXT NOT NULL CHECK(status IN ('pending', 'running', 'completed', 'failed')),
    attempt INTEGER NOT NULL,
    error TEXT
);

CREATE INDEX idx_task_status ON task_execution_log(status);
CREATE INDEX idx_task_time ON task_execution_log(start_time);
```

---

## State Transitions

### TaskExecutionContext Status Transitions
```
Pending ──► Running ──► Completed
              │
              └────────► Failed
```

Valid transitions:
- `Pending → Running`: Task starts execution
- `Running → Completed`: Task completes successfully
- `Running → Failed`: Task fails after all retries exhausted
- `Running → Running`: Retry attempt (Attempt counter increments)

Invalid transitions (will error):
- `Completed → *`: Terminal state
- `Failed → *`: Terminal state
- `Pending → Completed`: Must go through Running
- `Pending → Failed`: Must go through Running

---

**Data Model Complete**: 2025-11-17
**Next**: Generate contracts/provider-registry.yaml
