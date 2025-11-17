package unit

import (
	"os"
	"testing"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowLoader_LoadFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		check   func(t *testing.T, wf *models.WorkflowDefinition)
	}{
		{
			name: "valid simple workflow",
			yaml: `
schema_version: "1.0"
name: "Simple Workflow"
version: "1.0"
config:
  max_parallel: 4
  retries: 3
  timeout: "30s"
  allowed_commands:
    - go
    - git
tasks:
  - id: task1
    name: "Write File"
    type: file_op
    inputs:
      operation: write
      path: test.txt
      content: "Hello, World!"
`,
			wantErr: false,
			check: func(t *testing.T, wf *models.WorkflowDefinition) {
				assert.Equal(t, "Simple Workflow", wf.Name)
				assert.Equal(t, "1.0", wf.Version)
				assert.Len(t, wf.Tasks, 1)
				assert.Equal(t, "task1", wf.Tasks[0].ID)
				assert.Equal(t, "file_op", wf.Tasks[0].Type)
				assert.Equal(t, 4, wf.Config.MaxParallel)
				assert.Equal(t, 3, wf.Config.Retries)
				assert.Equal(t, 30*time.Second, wf.Config.Timeout)
				assert.Contains(t, wf.Config.AllowedCommands, "go")
				assert.Contains(t, wf.Config.AllowedCommands, "git")
			},
		},
		{
			name: "workflow with dependencies",
			yaml: `
schema_version: "1.0"
name: "Dependent Workflow"
version: "1.0"
config:
  max_parallel: 2
  allowed_commands:
    - echo
tasks:
  - id: task1
    type: file_op
    inputs:
      operation: write
      path: file1.txt
      content: "First"
    outputs:
      - task1_result
  - id: task2
    type: file_op
    dependencies:
      - task1
    inputs:
      operation: write
      path: file2.txt
      content: "Second"
`,
			wantErr: false,
			check: func(t *testing.T, wf *models.WorkflowDefinition) {
				assert.Len(t, wf.Tasks, 2)
				assert.Empty(t, wf.Tasks[0].Dependencies)
				assert.Contains(t, wf.Tasks[1].Dependencies, "task1")
				assert.Contains(t, wf.Tasks[0].Outputs, "task1_result")
			},
		},
		{
			name: "workflow with timeouts",
			yaml: `
schema_version: "1.0"
name: "Timeout Workflow"
version: "1.0"
config:
  max_parallel: 4
  timeout: "5m"
  allowed_commands: []
tasks:
  - id: task1
    type: file_op
    timeout: "30s"
    inputs:
      operation: write
      path: test.txt
      content: "Test"
`,
			wantErr: false,
			check: func(t *testing.T, wf *models.WorkflowDefinition) {
				assert.Equal(t, 5*time.Minute, wf.Config.Timeout)
				assert.Equal(t, 30*time.Second, wf.Tasks[0].Timeout)
			},
		},
		{
			name: "invalid YAML syntax",
			yaml: `
name: "Invalid"
tasks:
  - id: task1
    type: file_op
    inputs:
      operation: write
    [invalid yaml
`,
			wantErr: true,
		},
		{
			name: "workflow with cyclic dependencies",
			yaml: `
schema_version: "1.0"
name: "Cyclic Workflow"
version: "1.0"
config:
  max_parallel: 4
  allowed_commands: []
tasks:
  - id: task1
    type: file_op
    dependencies:
      - task2
    inputs:
      operation: write
      path: file1.txt
      content: "First"
  - id: task2
    type: file_op
    dependencies:
      - task1
    inputs:
      operation: write
      path: file2.txt
      content: "Second"
`,
			wantErr: true,
		},
		{
			name: "workflow with invalid task type",
			yaml: `
schema_version: "1.0"
name: "Invalid Task Type"
version: "1.0"
config:
  max_parallel: 4
  allowed_commands: []
tasks:
  - id: task1
    type: invalid_type
    inputs:
      operation: test
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := workflow.NewWorkflowLoader()
			wf, err := loader.LoadFromBytes([]byte(tt.yaml))

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, wf)
				if tt.check != nil {
					tt.check(t, wf)
				}
			}
		})
	}
}

func TestWorkflowLoader_SaveToFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workflow definition
	wfDef := &models.WorkflowDefinition{
		SchemaVersion: "1.0",
		ID:            "test-workflow",
		Name:          "Test Workflow",
		Version:       "1.0",
		Tasks: []models.WorkflowTask{
			{
				ID:   "task1",
				Name: "Write File",
				Type: "file_op",
				Inputs: map[string]interface{}{
					"operation": "write",
					"path":      "test.txt",
					"content":   "Hello",
				},
				Timeout: 30 * time.Second,
			},
		},
		Config: models.WorkflowConfig{
			MaxParallel:     4,
			Retries:         3,
			Timeout:         5 * time.Minute,
			AllowedCommands: []string{"go", "git"},
		},
	}

	// Save to file
	loader := workflow.NewWorkflowLoader()
	filePath := tmpDir + "/workflow.yaml"
	err := loader.SaveToFile(wfDef, filePath)
	require.NoError(t, err)

	// Load back and verify
	loadedWf, err := loader.LoadFromFile(filePath)
	require.NoError(t, err)

	assert.Equal(t, wfDef.Name, loadedWf.Name)
	assert.Equal(t, wfDef.Version, loadedWf.Version)
	assert.Len(t, loadedWf.Tasks, 1)
	assert.Equal(t, wfDef.Tasks[0].ID, loadedWf.Tasks[0].ID)
	assert.Equal(t, wfDef.Config.MaxParallel, loadedWf.Config.MaxParallel)
}

func TestWorkflowLoader_LoadFromFile(t *testing.T) {
	tmpDir := t.TempDir()

	yamlContent := `
schema_version: "1.0"
name: "File Load Test"
version: "1.0"
config:
  max_parallel: 4
  allowed_commands:
    - go
tasks:
  - id: task1
    type: file_op
    inputs:
      operation: write
      path: test.txt
      content: "Test"
`

	// Write YAML to file
	filePath := tmpDir + "/workflow.yaml"
	err := os.WriteFile(filePath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Load from file
	loader := workflow.NewWorkflowLoader()
	wf, err := loader.LoadFromFile(filePath)
	require.NoError(t, err)

	assert.Equal(t, "File Load Test", wf.Name)
	assert.Len(t, wf.Tasks, 1)
}

func TestWorkflowLoader_ValidateYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "valid YAML",
			yaml: `
schema_version: "1.0"
name: "Test"
version: "1.0"
config:
  max_parallel: 4
  allowed_commands: []
tasks:
  - id: task1
    type: file_op
    inputs:
      operation: write
`,
			wantErr: false,
		},
		{
			name: "invalid YAML",
			yaml: `
name: "Test"
tasks:
  - id: task1
    [invalid
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := workflow.NewWorkflowLoader()
			err := loader.ValidateYAML([]byte(tt.yaml))

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWorkflowLoader_RoundTrip(t *testing.T) {
	// Original workflow
	original := &models.WorkflowDefinition{
		SchemaVersion: "1.0",
		ID:            "round-trip-test",
		Name:          "Round Trip Test",
		Version:       "2.0",
		Tasks: []models.WorkflowTask{
			{
				ID:   "task1",
				Name: "First Task",
				Type: "file_op",
				Inputs: map[string]interface{}{
					"operation": "write",
					"path":      "test.txt",
					"content":   "Hello, World!",
				},
				Outputs: []string{"result1"},
				Timeout: 30 * time.Second,
			},
			{
				ID:           "task2",
				Name:         "Second Task",
				Type:         "file_op",
				Dependencies: []string{"task1"},
				Inputs: map[string]interface{}{
					"operation": "read",
					"path":      "test.txt",
				},
				Timeout: 15 * time.Second,
			},
		},
		Config: models.WorkflowConfig{
			MaxParallel:     4,
			Retries:         3,
			Timeout:         5 * time.Minute,
			AllowedCommands: []string{"go", "git", "golangci-lint"},
		},
	}

	loader := workflow.NewWorkflowLoader()

	// Convert to YAML
	tmpDir := t.TempDir()
	filePath := tmpDir + "/roundtrip.yaml"
	err := loader.SaveToFile(original, filePath)
	require.NoError(t, err)

	// Load back
	loaded, err := loader.LoadFromFile(filePath)
	require.NoError(t, err)

	// Verify all fields match
	assert.Equal(t, original.Name, loaded.Name)
	assert.Equal(t, original.Version, loaded.Version)
	assert.Len(t, loaded.Tasks, len(original.Tasks))

	for i := range original.Tasks {
		assert.Equal(t, original.Tasks[i].ID, loaded.Tasks[i].ID)
		assert.Equal(t, original.Tasks[i].Name, loaded.Tasks[i].Name)
		assert.Equal(t, original.Tasks[i].Type, loaded.Tasks[i].Type)
		assert.Equal(t, original.Tasks[i].Dependencies, loaded.Tasks[i].Dependencies)
		assert.Equal(t, original.Tasks[i].Outputs, loaded.Tasks[i].Outputs)
		assert.Equal(t, original.Tasks[i].Timeout, loaded.Tasks[i].Timeout)
	}

	assert.Equal(t, original.Config.MaxParallel, loaded.Config.MaxParallel)
	assert.Equal(t, original.Config.Retries, loaded.Config.Retries)
	assert.Equal(t, original.Config.Timeout, loaded.Config.Timeout)
	assert.Equal(t, original.Config.AllowedCommands, loaded.Config.AllowedCommands)
}
