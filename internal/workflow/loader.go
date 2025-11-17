package workflow

import (
	"fmt"
	"os"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// WorkflowLoader handles loading workflow definitions from YAML files
type WorkflowLoader struct{}

// NewWorkflowLoader creates a new workflow loader
func NewWorkflowLoader() *WorkflowLoader {
	return &WorkflowLoader{}
}

// yamlWorkflow represents the YAML structure for workflow definitions
type yamlWorkflow struct {
	SchemaVersion string             `yaml:"schema_version"`
	Name          string             `yaml:"name"`
	Version       string             `yaml:"version"`
	Config        yamlWorkflowConfig `yaml:"config"`
	Tasks         []yamlTask         `yaml:"tasks"`
}

type yamlWorkflowConfig struct {
	MaxParallel     int      `yaml:"max_parallel"`
	Retries         int      `yaml:"retries"`
	Timeout         string   `yaml:"timeout"` // Duration string (e.g., "30s", "5m")
	AllowedCommands []string `yaml:"allowed_commands"`
}

type yamlTask struct {
	ID           string                 `yaml:"id"`
	Name         string                 `yaml:"name"`
	Type         string                 `yaml:"type"`
	Inputs       map[string]interface{} `yaml:"inputs"`
	Outputs      []string               `yaml:"outputs"`
	Dependencies []string               `yaml:"dependencies"`
	Timeout      string                 `yaml:"timeout"` // Duration string
}

// LoadFromFile loads a workflow definition from a YAML file
func (l *WorkflowLoader) LoadFromFile(path string) (*models.WorkflowDefinition, error) {
	// Read file
	//nolint:gosec // G304: Reading workflow file - required for workflow loading
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	return l.LoadFromBytes(data)
}

// LoadFromBytes loads a workflow definition from YAML bytes
func (l *WorkflowLoader) LoadFromBytes(data []byte) (*models.WorkflowDefinition, error) {
	var yamlWf yamlWorkflow

	// Unmarshal YAML
	if err := yaml.Unmarshal(data, &yamlWf); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Convert to models.WorkflowDefinition
	workflow, err := l.convertToWorkflowDefinition(yamlWf)
	if err != nil {
		return nil, fmt.Errorf("failed to convert workflow: %w", err)
	}

	// Validate workflow
	if err := workflow.Validate(); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	return workflow, nil
}

// convertToWorkflowDefinition converts YAML structure to workflow definition
func (l *WorkflowLoader) convertToWorkflowDefinition(yamlWf yamlWorkflow) (*models.WorkflowDefinition, error) {
	// Parse config timeout
	var configTimeout time.Duration
	if yamlWf.Config.Timeout != "" {
		var err error
		configTimeout, err = time.ParseDuration(yamlWf.Config.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid config timeout: %w", err)
		}
	}

	// Convert tasks
	tasks := make([]models.WorkflowTask, len(yamlWf.Tasks))
	for i, yamlTask := range yamlWf.Tasks {
		// Parse task timeout
		var taskTimeout time.Duration
		if yamlTask.Timeout != "" {
			var err error
			taskTimeout, err = time.ParseDuration(yamlTask.Timeout)
			if err != nil {
				return nil, fmt.Errorf("invalid timeout for task %s: %w", yamlTask.ID, err)
			}
		}

		tasks[i] = models.WorkflowTask{
			ID:           yamlTask.ID,
			Name:         yamlTask.Name,
			Type:         yamlTask.Type,
			Inputs:       yamlTask.Inputs,
			Outputs:      yamlTask.Outputs,
			Dependencies: yamlTask.Dependencies,
			Timeout:      taskTimeout,
		}
	}

	// Generate ID if not provided
	id := uuid.New().String()

	workflow := &models.WorkflowDefinition{
		SchemaVersion: yamlWf.SchemaVersion,
		ID:            id,
		Name:          yamlWf.Name,
		Version:       yamlWf.Version,
		Tasks:         tasks,
		Config: models.WorkflowConfig{
			MaxParallel:     yamlWf.Config.MaxParallel,
			Retries:         yamlWf.Config.Retries,
			Timeout:         configTimeout,
			AllowedCommands: yamlWf.Config.AllowedCommands,
		},
	}

	return workflow, nil
}

// SaveToFile saves a workflow definition to a YAML file
func (l *WorkflowLoader) SaveToFile(workflow *models.WorkflowDefinition, path string) error {
	// Convert to YAML structure
	yamlWf := l.convertFromWorkflowDefinition(workflow)

	// Marshal to YAML
	data, err := yaml.Marshal(yamlWf)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	return nil
}

// convertFromWorkflowDefinition converts workflow definition to YAML structure
func (l *WorkflowLoader) convertFromWorkflowDefinition(workflow *models.WorkflowDefinition) yamlWorkflow {
	// Convert tasks
	tasks := make([]yamlTask, len(workflow.Tasks))
	for i, task := range workflow.Tasks {
		timeoutStr := ""
		if task.Timeout > 0 {
			timeoutStr = task.Timeout.String()
		}

		tasks[i] = yamlTask{
			ID:           task.ID,
			Name:         task.Name,
			Type:         task.Type,
			Inputs:       task.Inputs,
			Outputs:      task.Outputs,
			Dependencies: task.Dependencies,
			Timeout:      timeoutStr,
		}
	}

	configTimeoutStr := ""
	if workflow.Config.Timeout > 0 {
		configTimeoutStr = workflow.Config.Timeout.String()
	}

	return yamlWorkflow{
		SchemaVersion: workflow.SchemaVersion,
		Name:          workflow.Name,
		Version:       workflow.Version,
		Config: yamlWorkflowConfig{
			MaxParallel:     workflow.Config.MaxParallel,
			Retries:         workflow.Config.Retries,
			Timeout:         configTimeoutStr,
			AllowedCommands: workflow.Config.AllowedCommands,
		},
		Tasks: tasks,
	}
}

// ValidateYAML validates YAML syntax without full conversion
func (l *WorkflowLoader) ValidateYAML(data []byte) error {
	var yamlWf yamlWorkflow
	return yaml.Unmarshal(data, &yamlWf)
}
