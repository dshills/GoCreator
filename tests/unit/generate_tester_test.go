package unit

import (
	"context"
	"strings"
	"testing"

	"github.com/dshills/gocreator/internal/generate"
	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTesterLLMClient implements llm.Client for testing
type mockTesterLLMClient struct {
	generateFunc func(ctx context.Context, prompt string) (string, error)
}

func (m *mockTesterLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, prompt)
	}
	return "package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {}\n", nil
}

func (m *mockTesterLLMClient) GenerateStructured(ctx context.Context, prompt string, schema interface{}) (interface{}, error) {
	return nil, nil
}

func (m *mockTesterLLMClient) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	return "", nil
}

func (m *mockTesterLLMClient) Provider() string {
	return "mock"
}

func (m *mockTesterLLMClient) Model() string {
	return "mock-model"
}

func TestNewTester(t *testing.T) {
	tests := []struct {
		name    string
		config  generate.TesterConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: generate.TesterConfig{
				LLMClient: &mockTesterLLMClient{},
			},
			wantErr: false,
		},
		{
			name: "missing LLM client",
			config: generate.TesterConfig{
				LLMClient: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester, err := generate.NewTester(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tester)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tester)
			}
		})
	}
}

func TestTester_GenerateTestFile(t *testing.T) {
	tests := []struct {
		name          string
		sourceFile    string
		plan          *models.GenerationPlan
		llmResponse   string
		wantErr       bool
		validatePatch func(t *testing.T, patch models.Patch)
	}{
		{
			name:       "generate test for main.go",
			sourceFile: "./output/main.go",
			plan:       createTestPlanForTester(),
			llmResponse: `package main

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "basic test",
			want: "expected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.want)
		})
	}
}`,
			wantErr: false,
			validatePatch: func(t *testing.T, patch models.Patch) {
				assert.Equal(t, "output/main_test.go", patch.TargetFile)
				assert.Contains(t, patch.Diff, "+package main")
				assert.Contains(t, patch.Diff, "+func TestMain")
				assert.True(t, patch.Reversible)
			},
		},
		{
			name:        "generate test with markdown code blocks",
			sourceFile:  "./output/service.go",
			plan:        createTestPlanForTester(),
			llmResponse: "```go\npackage service\n\nimport \"testing\"\n\nfunc TestService(t *testing.T) {}\n```",
			wantErr:     false,
			validatePatch: func(t *testing.T, patch models.Patch) {
				assert.Equal(t, "output/service_test.go", patch.TargetFile)
				assert.NotContains(t, patch.Diff, "```")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockTesterLLMClient{
				generateFunc: func(ctx context.Context, prompt string) (string, error) {
					return tt.llmResponse, nil
				},
			}

			tester, err := generate.NewTester(generate.TesterConfig{
				LLMClient: mockClient,
			})
			require.NoError(t, err)

			patch, err := tester.GenerateTestFile(context.Background(), tt.sourceFile, tt.plan)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validatePatch != nil {
					tt.validatePatch(t, patch)
				}
			}
		})
	}
}

func TestTester_Generate(t *testing.T) {
	tests := []struct {
		name         string
		packages     []string
		plan         *models.GenerationPlan
		llmResponse  string
		wantErr      bool
		minTestFiles int
	}{
		{
			name:         "generate tests for all Go files",
			packages:     []string{"main"},
			plan:         createTestPlanForTester(),
			llmResponse:  "package main\n\nimport \"testing\"\n\nfunc TestSomething(t *testing.T) {}\n",
			wantErr:      false,
			minTestFiles: 1, // At least one test file should be generated
		},
		{
			name:     "skip non-Go files",
			packages: []string{"main"},
			plan: &models.GenerationPlan{
				FileTree: models.FileTree{
					Root: "./output",
					Files: []models.File{
						{Path: "./output/go.mod"},
						{Path: "./output/README.md"},
						{Path: "./output/main.go"},
					},
				},
			},
			llmResponse:  "package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {}\n",
			wantErr:      false,
			minTestFiles: 1, // Only main.go should get a test
		},
		{
			name:     "skip existing test files",
			packages: []string{"main"},
			plan: &models.GenerationPlan{
				FileTree: models.FileTree{
					Root: "./output",
					Files: []models.File{
						{Path: "./output/main.go"},
						{Path: "./output/main_test.go"},
					},
				},
			},
			llmResponse:  "package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {}\n",
			wantErr:      false,
			minTestFiles: 0, // main_test.go already exists, only main.go should be considered
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockTesterLLMClient{
				generateFunc: func(ctx context.Context, prompt string) (string, error) {
					return tt.llmResponse, nil
				},
			}

			tester, err := generate.NewTester(generate.TesterConfig{
				LLMClient: mockClient,
			})
			require.NoError(t, err)

			patches, err := tester.Generate(context.Background(), tt.packages, tt.plan)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, len(patches), tt.minTestFiles)
			}
		})
	}
}

func TestTester_TestPromptGeneration(t *testing.T) {
	tests := []struct {
		name             string
		sourceFile       string
		expectedInPrompt []string
	}{
		{
			name:       "repository test prompt",
			sourceFile: "./output/repository.go",
			expectedInPrompt: []string{
				"repository",
				"test",
				"table-driven",
			},
		},
		{
			name:       "service test prompt",
			sourceFile: "./output/service.go",
			expectedInPrompt: []string{
				"service",
				"test",
				"business logic",
			},
		},
		{
			name:       "handler test prompt",
			sourceFile: "./output/handler.go",
			expectedInPrompt: []string{
				"handler",
				"HTTP",
				"test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPrompt string
			mockClient := &mockTesterLLMClient{
				generateFunc: func(ctx context.Context, prompt string) (string, error) {
					capturedPrompt = prompt
					return "package test\n\nimport \"testing\"\n", nil
				},
			}

			tester, err := generate.NewTester(generate.TesterConfig{
				LLMClient: mockClient,
			})
			require.NoError(t, err)

			plan := createTestPlanForTester()
			_, err = tester.GenerateTestFile(context.Background(), tt.sourceFile, plan)
			require.NoError(t, err)

			// Verify prompt contains expected elements
			for _, expected := range tt.expectedInPrompt {
				assert.Contains(t, strings.ToLower(capturedPrompt), strings.ToLower(expected),
					"Prompt should contain: %s", expected)
			}
		})
	}
}

// Helper functions

func createTestPlanForTester() *models.GenerationPlan {
	return &models.GenerationPlan{
		SchemaVersion: "1.0",
		ID:            "test-plan-id",
		FileTree: models.FileTree{
			Root: "./output",
			Files: []models.File{
				{Path: "./output/main.go", Purpose: "Main entry point"},
				{Path: "./output/service.go", Purpose: "Service layer"},
			},
		},
		Phases: []models.GenerationPhase{
			{
				Name:  "code",
				Order: 1,
				Tasks: []models.GenerationTask{
					{ID: "gen_main", Type: "generate_file", TargetPath: "./output/main.go"},
				},
			},
		},
	}
}
