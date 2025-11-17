package unit

import (
	"context"
	"os"
	"testing"

	"github.com/dshills/gocreator/internal/workflow"
	"github.com/dshills/gocreator/pkg/fsops"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileOpTask_Execute(t *testing.T) {
	tests := []struct {
		name    string
		inputs  map[string]interface{}
		wantErr bool
		check   func(t *testing.T, fsOps fsops.FileOps, result interface{})
	}{
		{
			name: "write file",
			inputs: map[string]interface{}{
				"operation": "write",
				"path":      "test.txt",
				"content":   "Hello, World!",
			},
			wantErr: false,
			check: func(t *testing.T, fsOps fsops.FileOps, result interface{}) {
				content, err := fsOps.ReadFile(context.Background(), "test.txt")
				require.NoError(t, err)
				assert.Equal(t, "Hello, World!", content)

				// Check result
				resultMap, ok := result.(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "write", resultMap["operation"])
				assert.Equal(t, "test.txt", resultMap["path"])
			},
		},
		{
			name: "read file",
			inputs: map[string]interface{}{
				"operation": "read",
				"path":      "existing.txt",
			},
			wantErr: false,
			check: func(t *testing.T, fsOps fsops.FileOps, result interface{}) {
				resultMap, ok := result.(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "read", resultMap["operation"])
				assert.Equal(t, "existing.txt", resultMap["path"])
				assert.Equal(t, "Existing content", resultMap["content"])
			},
		},
		{
			name: "delete file",
			inputs: map[string]interface{}{
				"operation": "delete",
				"path":      "to_delete.txt",
			},
			wantErr: false,
			check: func(t *testing.T, fsOps fsops.FileOps, result interface{}) {
				exists, err := fsOps.Exists(context.Background(), "to_delete.txt")
				require.NoError(t, err)
				assert.False(t, exists)
			},
		},
		{
			name: "missing operation",
			inputs: map[string]interface{}{
				"path":    "test.txt",
				"content": "Hello",
			},
			wantErr: true,
		},
		{
			name: "missing path",
			inputs: map[string]interface{}{
				"operation": "write",
				"content":   "Hello",
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

			// Setup: create files if needed
			if tt.name == "read file" {
				err := fsOps.WriteFile(context.Background(), "existing.txt", "Existing content")
				require.NoError(t, err)
			}
			if tt.name == "delete file" {
				err := fsOps.WriteFile(context.Background(), "to_delete.txt", "To be deleted")
				require.NoError(t, err)
			}

			// Create task
			logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
			task := workflow.NewFileOpTask(fsOps, logger)

			// Execute
			result, err := task.Execute(context.Background(), tt.inputs)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.check != nil {
					tt.check(t, fsOps, result)
				}
			}
		})
	}
}

func TestPatchTask_Execute(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(fsOps fsops.FileOps) error
		inputs  map[string]interface{}
		wantErr bool
		check   func(t *testing.T, fsOps fsops.FileOps, result interface{})
	}{
		{
			name: "apply patch to file",
			setup: func(fsOps fsops.FileOps) error {
				return fsOps.WriteFile(context.Background(), "target.txt", "Line 1\nLine 2\nLine 3\n")
			},
			inputs: map[string]interface{}{
				"target_file": "target.txt",
				"diff": `@@ -1,3 +1,3 @@
 Line 1
-Line 2
+Line 2 Modified
 Line 3
`,
				"reversible": true,
			},
			wantErr: false,
			check: func(t *testing.T, fsOps fsops.FileOps, result interface{}) {
				content, err := fsOps.ReadFile(context.Background(), "target.txt")
				require.NoError(t, err)
				assert.Contains(t, content, "Line 2 Modified")

				resultMap, ok := result.(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "patch", resultMap["operation"])
			},
		},
		{
			name: "missing target_file",
			inputs: map[string]interface{}{
				"diff": "test diff",
			},
			wantErr: true,
		},
		{
			name: "missing diff",
			inputs: map[string]interface{}{
				"target_file": "test.txt",
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

			// Setup
			if tt.setup != nil {
				err := tt.setup(fsOps)
				require.NoError(t, err)
			}

			// Create task
			logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
			task := workflow.NewPatchTask(fsOps, logger)

			// Execute
			result, err := task.Execute(context.Background(), tt.inputs)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.check != nil {
					tt.check(t, fsOps, result)
				}
			}
		})
	}
}

func TestShellTask_Execute(t *testing.T) {
	tests := []struct {
		name    string
		inputs  map[string]interface{}
		wantErr bool
		check   func(t *testing.T, result interface{})
	}{
		{
			name: "execute allowed command",
			inputs: map[string]interface{}{
				"cmd":              "echo",
				"args":             []interface{}{"Hello, World!"},
				"allowed_commands": []string{"echo", "go", "git"},
				"timeout":          5.0,
			},
			wantErr: false,
			check: func(t *testing.T, result interface{}) {
				resultMap, ok := result.(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "shell_cmd", resultMap["operation"])
				assert.Equal(t, 0, resultMap["exit_code"])
				assert.Contains(t, resultMap["output"], "Hello")
			},
		},
		{
			name: "disallowed command",
			inputs: map[string]interface{}{
				"cmd":              "rm",
				"args":             []interface{}{"-rf", "/"},
				"allowed_commands": []string{"echo", "go", "git"},
			},
			wantErr: true,
		},
		{
			name: "missing cmd",
			inputs: map[string]interface{}{
				"allowed_commands": []string{"echo"},
			},
			wantErr: true,
		},
		{
			name: "missing allowed_commands",
			inputs: map[string]interface{}{
				"cmd": "echo",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create task
			logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
			task := workflow.NewShellTask(logger)

			// Execute
			result, err := task.Execute(context.Background(), tt.inputs)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.check != nil {
					tt.check(t, result)
				}
			}
		})
	}
}

func TestTaskRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	fsOps, err := fsops.New(fsops.Config{
		RootDir: tmpDir,
	})
	require.NoError(t, err)

	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	registry := workflow.NewTaskRegistry(fsOps, logger)

	t.Run("get registered task", func(t *testing.T) {
		task, err := registry.Get("file_op")
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, "file_op", task.Name())
	})

	t.Run("get unregistered task", func(t *testing.T) {
		_, err := registry.Get("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not registered")
	})

	t.Run("register custom task", func(t *testing.T) {
		customTask := workflow.NewFileOpTask(fsOps, logger)
		registry.Register("custom", customTask)

		task, err := registry.Get("custom")
		assert.NoError(t, err)
		assert.NotNil(t, task)
	})
}

func TestExecutionContext(t *testing.T) {
	ctx := &workflow.ExecutionContext{
		WorkflowID:  "test-workflow",
		ExecutionID: "test-execution",
		State:       make(map[string]interface{}),
	}

	t.Run("set and get value", func(t *testing.T) {
		ctx.Set("key1", "value1")
		val, ok := ctx.Get("key1")
		assert.True(t, ok)
		assert.Equal(t, "value1", val)
	})

	t.Run("get nonexistent value", func(t *testing.T) {
		_, ok := ctx.Get("nonexistent")
		assert.False(t, ok)
	})

	t.Run("concurrent access", func(t *testing.T) {
		done := make(chan bool)

		// Writer goroutine
		go func() {
			for i := 0; i < 100; i++ {
				ctx.Set("counter", i)
			}
			done <- true
		}()

		// Reader goroutine
		go func() {
			for i := 0; i < 100; i++ {
				ctx.Get("counter")
			}
			done <- true
		}()

		<-done
		<-done
	})
}
