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

// mockCoderLLMClient implements llm.Client for testing
type mockCoderLLMClient struct {
	generateFunc func(ctx context.Context, prompt string) (string, error)
}

func (m *mockCoderLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, prompt)
	}
	return "package main\n\nfunc main() {}\n", nil
}

func (m *mockCoderLLMClient) GenerateStructured(ctx context.Context, prompt string, schema interface{}) (interface{}, error) {
	return nil, nil
}

func (m *mockCoderLLMClient) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	return "", nil
}

func (m *mockCoderLLMClient) Provider() string {
	return "mock"
}

func (m *mockCoderLLMClient) Model() string {
	return "mock-model"
}

func TestNewCoder(t *testing.T) {
	tests := []struct {
		name    string
		config  generate.CoderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: generate.CoderConfig{
				LLMClient: &mockCoderLLMClient{},
			},
			wantErr: false,
		},
		{
			name: "missing LLM client",
			config: generate.CoderConfig{
				LLMClient: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coder, err := generate.NewCoder(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, coder)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, coder)
			}
		})
	}
}

func TestCoder_GenerateFile(t *testing.T) {
	tests := []struct {
		name          string
		task          models.GenerationTask
		plan          *models.GenerationPlan
		llmResponse   string
		wantErr       bool
		validatePatch func(t *testing.T, patch models.Patch)
	}{
		{
			name: "generate main.go",
			task: models.GenerationTask{
				ID:         "generate_main",
				Type:       "generate_file",
				TargetPath: "./output/main.go",
			},
			plan:        createTestGenerationPlan(),
			llmResponse: "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n",
			wantErr:     false,
			validatePatch: func(t *testing.T, patch models.Patch) {
				assert.Equal(t, "./output/main.go", patch.TargetFile)
				assert.NotEmpty(t, patch.Diff)
				assert.True(t, patch.Reversible)
				assert.Contains(t, patch.Diff, "+package main")
			},
		},
		{
			name: "generate go.mod",
			task: models.GenerationTask{
				ID:         "generate_gomod",
				Type:       "generate_file",
				TargetPath: "./output/go.mod",
			},
			plan:        createTestGenerationPlan(),
			llmResponse: "module github.com/test/app\n\ngo 1.23\n",
			wantErr:     false,
			validatePatch: func(t *testing.T, patch models.Patch) {
				assert.Equal(t, "./output/go.mod", patch.TargetFile)
				assert.Contains(t, patch.Diff, "+module")
			},
		},
		{
			name: "generate with markdown code blocks",
			task: models.GenerationTask{
				ID:         "generate_file",
				Type:       "generate_file",
				TargetPath: "./output/test.go",
			},
			plan:        createTestGenerationPlan(),
			llmResponse: "```go\npackage test\n\nfunc Test() {}\n```",
			wantErr:     false,
			validatePatch: func(t *testing.T, patch models.Patch) {
				// Should strip markdown code blocks
				assert.Contains(t, patch.Diff, "+package test")
				assert.NotContains(t, patch.Diff, "```")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockCoderLLMClient{
				generateFunc: func(ctx context.Context, prompt string) (string, error) {
					return tt.llmResponse, nil
				},
			}

			coder, err := generate.NewCoder(generate.CoderConfig{
				LLMClient: mockClient,
			})
			require.NoError(t, err)

			patch, err := coder.GenerateFile(context.Background(), tt.task, tt.plan)

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

func TestCoder_Generate(t *testing.T) {
	tests := []struct {
		name          string
		plan          *models.GenerationPlan
		llmResponse   string
		wantErr       bool
		expectedFiles int
	}{
		{
			name:          "generate all files in plan",
			plan:          createTestGenerationPlan(),
			llmResponse:   "package main\n\nfunc main() {}\n",
			wantErr:       false,
			expectedFiles: 2, // go.mod and main.go
		},
		{
			name: "skip non-generate_file tasks",
			plan: &models.GenerationPlan{
				FileTree: models.FileTree{
					Root: "./output",
				},
				Phases: []models.GenerationPhase{
					{
						Name:  "test",
						Order: 1,
						Tasks: []models.GenerationTask{
							{
								ID:         "task1",
								Type:       "run_command",
								TargetPath: "",
							},
							{
								ID:         "task2",
								Type:       "generate_file",
								TargetPath: "./output/main.go",
							},
						},
					},
				},
			},
			llmResponse:   "package main\n",
			wantErr:       false,
			expectedFiles: 1, // Only main.go, skip run_command
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockCoderLLMClient{
				generateFunc: func(ctx context.Context, prompt string) (string, error) {
					return tt.llmResponse, nil
				},
			}

			coder, err := generate.NewCoder(generate.CoderConfig{
				LLMClient: mockClient,
			})
			require.NoError(t, err)

			patches, err := coder.Generate(context.Background(), tt.plan)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, patches, tt.expectedFiles)
			}
		})
	}
}

func TestCoder_PromptGeneration(t *testing.T) {
	tests := []struct {
		name             string
		task             models.GenerationTask
		plan             *models.GenerationPlan
		expectedInPrompt []string
	}{
		{
			name: "main.go prompt includes relevant context",
			task: models.GenerationTask{
				ID:         "generate_main",
				Type:       "generate_file",
				TargetPath: "./output/main.go",
			},
			plan: createTestGenerationPlan(),
			expectedInPrompt: []string{
				"main.go",
				"Go",
				"package main",
			},
		},
		{
			name: "go.mod prompt includes module information",
			task: models.GenerationTask{
				ID:         "generate_gomod",
				Type:       "generate_file",
				TargetPath: "./output/go.mod",
			},
			plan: createTestGenerationPlan(),
			expectedInPrompt: []string{
				"go.mod",
				"module",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPrompt string
			mockClient := &mockCoderLLMClient{
				generateFunc: func(ctx context.Context, prompt string) (string, error) {
					capturedPrompt = prompt
					return "package test\n", nil
				},
			}

			coder, err := generate.NewCoder(generate.CoderConfig{
				LLMClient: mockClient,
			})
			require.NoError(t, err)

			_, err = coder.GenerateFile(context.Background(), tt.task, tt.plan)
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

func createTestGenerationPlan() *models.GenerationPlan {
	return &models.GenerationPlan{
		SchemaVersion: "1.0",
		ID:            "test-plan-id",
		FCSID:         "test-fcs-id",
		FileTree: models.FileTree{
			Root: "./output",
			Directories: []models.Directory{
				{Path: "./output/cmd/app", Purpose: "Main application"},
			},
			Files: []models.File{
				{Path: "./output/go.mod", Purpose: "Go module definition"},
				{Path: "./output/main.go", Purpose: "Main entry point"},
			},
		},
		Phases: []models.GenerationPhase{
			{
				Name:  "setup",
				Order: 1,
				Tasks: []models.GenerationTask{
					{
						ID:          "generate_gomod",
						Type:        "generate_file",
						TargetPath:  "./output/go.mod",
						CanParallel: false,
					},
					{
						ID:          "generate_main",
						Type:        "generate_file",
						TargetPath:  "./output/main.go",
						CanParallel: false,
					},
				},
			},
		},
	}
}
