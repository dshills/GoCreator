package providers_test

import (
	"testing"
	"time"

	"github.com/dshills/gocreator/src/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskExecutionContext_Validate(t *testing.T) {
	t.Run("valid_context", func(t *testing.T) {
		ctx := &providers.TaskExecutionContext{
			TaskID:           "task-001",
			Role:             providers.RoleCoder,
			SelectedProvider: "openai-fast",
			StartTime:        time.Now(),
			Status:           providers.TaskStatusPending,
			Attempt:          1,
		}

		err := ctx.Validate()
		assert.NoError(t, err)
	})

	t.Run("empty_task_id", func(t *testing.T) {
		ctx := &providers.TaskExecutionContext{
			TaskID:           "",
			SelectedProvider: "openai-fast",
			Attempt:          1,
		}

		err := ctx.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task ID must not be empty")
	})

	t.Run("empty_provider", func(t *testing.T) {
		ctx := &providers.TaskExecutionContext{
			TaskID:           "task-001",
			SelectedProvider: "",
			Attempt:          1,
		}

		err := ctx.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "selected provider must not be empty")
	})

	t.Run("invalid_attempt_number", func(t *testing.T) {
		ctx := &providers.TaskExecutionContext{
			TaskID:           "task-001",
			SelectedProvider: "openai-fast",
			Attempt:          0,
		}

		err := ctx.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "attempt must be >= 1")
	})

	t.Run("end_time_before_start_time", func(t *testing.T) {
		now := time.Now()
		ctx := &providers.TaskExecutionContext{
			TaskID:           "task-001",
			SelectedProvider: "openai-fast",
			StartTime:        now,
			EndTime:          now.Add(-1 * time.Hour),
			Attempt:          1,
		}

		err := ctx.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "end time must be after start time")
	})

	t.Run("valid_with_end_time", func(t *testing.T) {
		now := time.Now()
		ctx := &providers.TaskExecutionContext{
			TaskID:           "task-001",
			SelectedProvider: "openai-fast",
			StartTime:        now,
			EndTime:          now.Add(1 * time.Hour),
			Status:           providers.TaskStatusCompleted,
			Attempt:          1,
		}

		err := ctx.Validate()
		assert.NoError(t, err)
	})
}

func TestTaskExecutionContext_TransitionTo(t *testing.T) {
	t.Run("pending_to_running", func(t *testing.T) {
		ctx := &providers.TaskExecutionContext{
			TaskID:           "task-001",
			SelectedProvider: "openai-fast",
			Status:           providers.TaskStatusPending,
			Attempt:          1,
		}

		err := ctx.TransitionTo(providers.TaskStatusRunning)
		assert.NoError(t, err)
		assert.Equal(t, providers.TaskStatusRunning, ctx.Status)
	})

	t.Run("pending_to_completed_invalid", func(t *testing.T) {
		ctx := &providers.TaskExecutionContext{
			TaskID:           "task-001",
			SelectedProvider: "openai-fast",
			Status:           providers.TaskStatusPending,
			Attempt:          1,
		}

		err := ctx.TransitionTo(providers.TaskStatusCompleted)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid transition from Pending to completed")
		assert.Equal(t, providers.TaskStatusPending, ctx.Status) // Status unchanged
	})

	t.Run("pending_to_failed_invalid", func(t *testing.T) {
		ctx := &providers.TaskExecutionContext{
			TaskID:           "task-001",
			SelectedProvider: "openai-fast",
			Status:           providers.TaskStatusPending,
			Attempt:          1,
		}

		err := ctx.TransitionTo(providers.TaskStatusFailed)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid transition from Pending to failed")
	})

	t.Run("running_to_completed", func(t *testing.T) {
		ctx := &providers.TaskExecutionContext{
			TaskID:           "task-001",
			SelectedProvider: "openai-fast",
			Status:           providers.TaskStatusRunning,
			Attempt:          1,
		}

		err := ctx.TransitionTo(providers.TaskStatusCompleted)
		assert.NoError(t, err)
		assert.Equal(t, providers.TaskStatusCompleted, ctx.Status)
	})

	t.Run("running_to_failed", func(t *testing.T) {
		ctx := &providers.TaskExecutionContext{
			TaskID:           "task-001",
			SelectedProvider: "openai-fast",
			Status:           providers.TaskStatusRunning,
			Attempt:          1,
		}

		err := ctx.TransitionTo(providers.TaskStatusFailed)
		assert.NoError(t, err)
		assert.Equal(t, providers.TaskStatusFailed, ctx.Status)
	})

	t.Run("running_to_running_retry", func(t *testing.T) {
		ctx := &providers.TaskExecutionContext{
			TaskID:           "task-001",
			SelectedProvider: "openai-fast",
			Status:           providers.TaskStatusRunning,
			Attempt:          1,
		}

		// Allow retry: Running → Running (for retry scenarios)
		err := ctx.TransitionTo(providers.TaskStatusRunning)
		assert.NoError(t, err)
		assert.Equal(t, providers.TaskStatusRunning, ctx.Status)
	})

	t.Run("running_to_pending_invalid", func(t *testing.T) {
		ctx := &providers.TaskExecutionContext{
			TaskID:           "task-001",
			SelectedProvider: "openai-fast",
			Status:           providers.TaskStatusRunning,
			Attempt:          1,
		}

		err := ctx.TransitionTo(providers.TaskStatusPending)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid transition from Running to pending")
	})

	t.Run("completed_is_terminal", func(t *testing.T) {
		ctx := &providers.TaskExecutionContext{
			TaskID:           "task-001",
			SelectedProvider: "openai-fast",
			Status:           providers.TaskStatusCompleted,
			Attempt:          1,
		}

		// Cannot transition from completed to any state
		err := ctx.TransitionTo(providers.TaskStatusRunning)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot transition from terminal state completed")
		assert.Equal(t, providers.TaskStatusCompleted, ctx.Status) // Status unchanged
	})

	t.Run("failed_is_terminal", func(t *testing.T) {
		ctx := &providers.TaskExecutionContext{
			TaskID:           "task-001",
			SelectedProvider: "openai-fast",
			Status:           providers.TaskStatusFailed,
			Attempt:          1,
		}

		// Cannot transition from failed to any state
		err := ctx.TransitionTo(providers.TaskStatusRunning)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot transition from terminal state failed")
		assert.Equal(t, providers.TaskStatusFailed, ctx.Status) // Status unchanged
	})
}

func TestTaskExecutionContext_StateMachine_FullLifecycle(t *testing.T) {
	t.Run("successful_execution_path", func(t *testing.T) {
		ctx := &providers.TaskExecutionContext{
			TaskID:           "task-001",
			Role:             providers.RoleCoder,
			SelectedProvider: "openai-fast",
			StartTime:        time.Now(),
			Status:           providers.TaskStatusPending,
			Attempt:          1,
		}

		// Validate initial state
		require.NoError(t, ctx.Validate())

		// Pending → Running
		require.NoError(t, ctx.TransitionTo(providers.TaskStatusRunning))
		assert.Equal(t, providers.TaskStatusRunning, ctx.Status)

		// Running → Completed
		ctx.EndTime = time.Now()
		require.NoError(t, ctx.TransitionTo(providers.TaskStatusCompleted))
		assert.Equal(t, providers.TaskStatusCompleted, ctx.Status)

		// Validate final state
		require.NoError(t, ctx.Validate())

		// Cannot transition from completed
		err := ctx.TransitionTo(providers.TaskStatusRunning)
		require.Error(t, err)
	})

	t.Run("failed_execution_path", func(t *testing.T) {
		ctx := &providers.TaskExecutionContext{
			TaskID:           "task-001",
			Role:             providers.RoleReviewer,
			SelectedProvider: "anthropic-precise",
			StartTime:        time.Now(),
			Status:           providers.TaskStatusPending,
			Attempt:          1,
		}

		// Pending → Running
		require.NoError(t, ctx.TransitionTo(providers.TaskStatusRunning))

		// Running → Failed
		ctx.EndTime = time.Now()
		ctx.Error = "rate limit exceeded"
		require.NoError(t, ctx.TransitionTo(providers.TaskStatusFailed))
		assert.Equal(t, providers.TaskStatusFailed, ctx.Status)
		assert.Equal(t, "rate limit exceeded", ctx.Error)

		// Validate final state
		require.NoError(t, ctx.Validate())

		// Cannot transition from failed
		err := ctx.TransitionTo(providers.TaskStatusRunning)
		require.Error(t, err)
	})

	t.Run("retry_execution_path", func(t *testing.T) {
		ctx := &providers.TaskExecutionContext{
			TaskID:           "task-001",
			Role:             providers.RolePlanner,
			SelectedProvider: "google-fast",
			StartTime:        time.Now(),
			Status:           providers.TaskStatusPending,
			Attempt:          1,
		}

		// Pending → Running
		require.NoError(t, ctx.TransitionTo(providers.TaskStatusRunning))

		// Simulate retry: Running → Running (attempt incremented externally)
		ctx.Attempt = 2
		require.NoError(t, ctx.TransitionTo(providers.TaskStatusRunning))
		assert.Equal(t, providers.TaskStatusRunning, ctx.Status)
		assert.Equal(t, 2, ctx.Attempt)

		// Eventually succeeds
		ctx.EndTime = time.Now()
		require.NoError(t, ctx.TransitionTo(providers.TaskStatusCompleted))
		assert.Equal(t, providers.TaskStatusCompleted, ctx.Status)
	})
}
