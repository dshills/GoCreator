package unit

import (
	"context"
	"testing"

	"github.com/dshills/gocreator/internal/generate"
	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLLMClient implements llm.Client for testing
type mockPlannerLLMClient struct {
	generateFunc func(ctx context.Context, prompt string) (string, error)
}

func (m *mockPlannerLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, prompt)
	}
	return "", nil
}

func (m *mockPlannerLLMClient) GenerateStructured(ctx context.Context, prompt string, schema interface{}) (interface{}, error) {
	return nil, nil
}

func (m *mockPlannerLLMClient) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	return "", nil
}

func (m *mockPlannerLLMClient) Provider() string {
	return "mock"
}

func (m *mockPlannerLLMClient) Model() string {
	return "mock-model"
}

func TestNewPlanner(t *testing.T) {
	tests := []struct {
		name    string
		config  generate.PlannerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: generate.PlannerConfig{
				LLMClient: &mockPlannerLLMClient{},
			},
			wantErr: false,
		},
		{
			name: "missing LLM client",
			config: generate.PlannerConfig{
				LLMClient: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planner, err := generate.NewPlanner(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, planner)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, planner)
			}
		})
	}
}

func TestPlanner_Plan(t *testing.T) {
	tests := []struct {
		name         string
		fcs          *models.FinalClarifiedSpecification
		llmResponse  string
		wantErr      bool
		validatePlan func(t *testing.T, plan *models.GenerationPlan)
	}{
		{
			name: "successful plan generation",
			fcs:  createTestFCS(),
			llmResponse: `{
				"file_tree": {
					"root": "./output",
					"directories": [
						{"path": "cmd/app", "purpose": "Main application"}
					],
					"files": [
						{"path": "cmd/app/main.go", "purpose": "Entry point", "generated_by": "generate_main"}
					]
				},
				"phases": [
					{
						"name": "setup",
						"order": 1,
						"dependencies": [],
						"tasks": [
							{
								"id": "create_gomod",
								"type": "generate_file",
								"target_path": "go.mod",
								"can_parallel": false
							}
						]
					}
				]
			}`,
			wantErr: false,
			validatePlan: func(t *testing.T, plan *models.GenerationPlan) {
				assert.NotEmpty(t, plan.ID)
				assert.Equal(t, "1.0", plan.SchemaVersion)
				assert.Equal(t, "./output", plan.FileTree.Root)
				assert.Len(t, plan.Phases, 1)
				assert.Len(t, plan.FileTree.Files, 1)
				assert.Equal(t, "setup", plan.Phases[0].Name)
			},
		},
		{
			name: "plan with multiple phases",
			fcs:  createTestFCS(),
			llmResponse: `{
				"file_tree": {
					"root": "./output",
					"directories": [],
					"files": []
				},
				"phases": [
					{
						"name": "setup",
						"order": 1,
						"dependencies": [],
						"tasks": []
					},
					{
						"name": "models",
						"order": 2,
						"dependencies": ["setup"],
						"tasks": []
					}
				]
			}`,
			wantErr: false,
			validatePlan: func(t *testing.T, plan *models.GenerationPlan) {
				assert.Len(t, plan.Phases, 2)
				assert.Equal(t, "setup", plan.Phases[0].Name)
				assert.Equal(t, "models", plan.Phases[1].Name)
				assert.Contains(t, plan.Phases[1].Dependencies, "setup")
			},
		},
		{
			name:        "invalid JSON response",
			fcs:         createTestFCS(),
			llmResponse: "not valid json",
			wantErr:     true,
		},
		{
			name: "plan with cyclic dependencies",
			fcs:  createTestFCS(),
			llmResponse: `{
				"file_tree": {
					"root": "./output",
					"directories": [],
					"files": []
				},
				"phases": [
					{
						"name": "phase1",
						"order": 1,
						"dependencies": ["phase2"],
						"tasks": []
					},
					{
						"name": "phase2",
						"order": 2,
						"dependencies": ["phase1"],
						"tasks": []
					}
				]
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockPlannerLLMClient{
				generateFunc: func(ctx context.Context, prompt string) (string, error) {
					return tt.llmResponse, nil
				},
			}

			planner, err := generate.NewPlanner(generate.PlannerConfig{
				LLMClient: mockClient,
			})
			require.NoError(t, err)

			plan, err := planner.Plan(context.Background(), tt.fcs)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, plan)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, plan)
				if tt.validatePlan != nil {
					tt.validatePlan(t, plan)
				}
			}
		})
	}
}

func TestPlanner_PlanValidation(t *testing.T) {
	tests := []struct {
		name        string
		modifyPlan  func(*models.GenerationPlan)
		wantErr     bool
		errContains string
	}{
		{
			name: "valid plan passes validation",
			modifyPlan: func(p *models.GenerationPlan) {
				// No modifications needed
			},
			wantErr: false,
		},
		{
			name: "cyclic dependencies detected",
			modifyPlan: func(p *models.GenerationPlan) {
				p.Phases = []models.GenerationPhase{
					{
						Name:         "phase1",
						Order:        1,
						Dependencies: []string{"phase2"},
					},
					{
						Name:         "phase2",
						Order:        2,
						Dependencies: []string{"phase1"},
					},
				}
			},
			wantErr:     true,
			errContains: "cyclic dependency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &models.GenerationPlan{
				FileTree: models.FileTree{
					Root: "./output",
				},
				Phases: []models.GenerationPhase{
					{
						Name:  "setup",
						Order: 1,
						Tasks: []models.GenerationTask{
							{
								ID:          "task1",
								Type:        "generate_file",
								TargetPath:  "./output/go.mod",
								CanParallel: false,
							},
						},
					},
				},
			}

			tt.modifyPlan(plan)
			err := plan.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper functions

func createTestFCS() *models.FinalClarifiedSpecification {
	return &models.FinalClarifiedSpecification{
		SchemaVersion: "1.0",
		ID:            "test-fcs-id",
		Version:       "1.0",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{
					ID:          "FR-001",
					Description: "Test requirement",
					Priority:    "high",
				},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{
					Name:    "main",
					Path:    "cmd/app",
					Purpose: "Main application package",
				},
			},
		},
		BuildConfig: models.BuildConfig{
			GoVersion:  "1.23",
			OutputPath: "./output",
		},
		TestingStrategy: models.TestingStrategy{
			CoverageTarget:   80.0,
			UnitTests:        true,
			IntegrationTests: false,
		},
	}
}
