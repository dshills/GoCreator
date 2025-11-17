package unit

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/dshills/gocreator/internal/generate"
	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/pkg/fsops"
	"github.com/dshills/gocreator/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEngineLLMClient implements llm.Client for testing
type mockEngineLLMClient struct {
	planResponse string
	codeResponse string
	testResponse string
}

func (m *mockEngineLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	// Determine response based on prompt content
	switch {
	case contains(prompt, "generation plan"):
		return m.planResponse, nil
	case contains(prompt, "test"):
		return m.testResponse, nil
	default:
		return m.codeResponse, nil
	}
}

func (m *mockEngineLLMClient) GenerateStructured(ctx context.Context, prompt string, schema interface{}) (interface{}, error) {
	return nil, nil
}

func (m *mockEngineLLMClient) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	return "", nil
}

func (m *mockEngineLLMClient) Provider() string {
	return "mock"
}

func (m *mockEngineLLMClient) Model() string {
	return "mock-model"
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr))
}

func TestNewGenerationEngine(t *testing.T) {
	tests := []struct {
		name    string
		config  generate.EngineConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: generate.EngineConfig{
				LLMClient: &mockEngineLLMClient{},
				FileOps:   createMockFileOps(t),
			},
			wantErr: false,
		},
		{
			name: "missing LLM client",
			config: generate.EngineConfig{
				LLMClient: nil,
				FileOps:   createMockFileOps(t),
			},
			wantErr: true,
		},
		{
			name: "missing file ops",
			config: generate.EngineConfig{
				LLMClient: &mockEngineLLMClient{},
				FileOps:   nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := generate.NewEngine(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, engine)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, engine)
			}
		})
	}
}

func TestEngine_Generate(t *testing.T) {
	// Create temporary directory for test output
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		fcs            *models.FinalClarifiedSpecification
		outputDir      string
		setupMocks     func() *mockEngineLLMClient
		wantErr        bool
		validateOutput func(t *testing.T, output *models.GenerationOutput)
	}{
		{
			name:      "successful end-to-end generation",
			fcs:       createCompleteTestFCS(),
			outputDir: filepath.Join(tmpDir, "test-project"),
			setupMocks: func() *mockEngineLLMClient {
				return &mockEngineLLMClient{
					planResponse: `{
						"file_tree": {
							"root": "` + filepath.Join(tmpDir, "test-project") + `",
							"directories": [{"path": "cmd", "purpose": "Main app"}],
							"files": [{"path": "main.go", "purpose": "Entry point", "generated_by": "gen_main"}]
						},
						"phases": [{
							"name": "setup",
							"order": 1,
							"dependencies": [],
							"tasks": [{
								"id": "gen_main",
								"type": "generate_file",
								"target_path": "main.go",
								"can_parallel": false
							}]
						}]
					}`,
					codeResponse: "package main\n\nfunc main() {}\n",
					testResponse: "package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {}\n",
				}
			},
			wantErr: false,
			validateOutput: func(t *testing.T, output *models.GenerationOutput) {
				assert.NotNil(t, output)
				assert.Equal(t, models.OutputStatusCompleted, output.Status)
				assert.Greater(t, len(output.Files), 0)
				assert.Greater(t, len(output.Patches), 0)
				assert.NotNil(t, output.Metadata.CompletedAt)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := tt.setupMocks()

			// Create file ops for test directory
			fileOps, err := fsops.New(fsops.Config{
				RootDir: tt.outputDir,
				Logger:  &noopFsLogger{},
			})
			require.NoError(t, err)

			engine, err := generate.NewEngine(generate.EngineConfig{
				LLMClient:    mockClient,
				FileOps:      fileOps,
				LogDecisions: true,
			})
			require.NoError(t, err)

			output, err := engine.Generate(context.Background(), tt.fcs, tt.outputDir)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validateOutput != nil {
					tt.validateOutput(t, output)
				}
			}
		})
	}
}

func TestEngine_GenerateWithInvalidFCS(t *testing.T) {
	tmpDir := t.TempDir()

	mockClient := &mockEngineLLMClient{
		planResponse: `{"file_tree": {"root": "./output"}, "phases": []}`,
		codeResponse: "package main\n",
		testResponse: "package main\n",
	}

	fileOps, err := fsops.New(fsops.Config{
		RootDir: tmpDir,
		Logger:  &noopFsLogger{},
	})
	require.NoError(t, err)

	engine, err := generate.NewEngine(generate.EngineConfig{
		LLMClient: mockClient,
		FileOps:   fileOps,
	})
	require.NoError(t, err)

	// Create FCS with cyclic dependencies
	fcs := createCompleteTestFCS()
	fcs.Architecture.Packages = []models.Package{
		{Name: "pkg1", Dependencies: []string{"pkg2"}},
		{Name: "pkg2", Dependencies: []string{"pkg1"}},
	}

	_, err = engine.Generate(context.Background(), fcs, tmpDir)
	assert.Error(t, err, "Should fail with cyclic dependencies")
}

func TestEngine_OutputValidation(t *testing.T) {
	tmpDir := t.TempDir()

	mockClient := &mockEngineLLMClient{
		planResponse: `{
			"file_tree": {
				"root": "` + tmpDir + `",
				"directories": [],
				"files": [{"path": "test.go", "purpose": "Test file", "generated_by": "gen_test"}]
			},
			"phases": [{
				"name": "phase1",
				"order": 1,
				"tasks": [{
					"id": "gen_test",
					"type": "generate_file",
					"target_path": "test.go",
					"can_parallel": false
				}]
			}]
		}`,
		codeResponse: "package test\n\nfunc Test() {}\n",
		testResponse: "",
	}

	fileOps, err := fsops.New(fsops.Config{
		RootDir: tmpDir,
		Logger:  &noopFsLogger{},
	})
	require.NoError(t, err)

	engine, err := generate.NewEngine(generate.EngineConfig{
		LLMClient: mockClient,
		FileOps:   fileOps,
	})
	require.NoError(t, err)

	output, err := engine.Generate(context.Background(), createCompleteTestFCS(), tmpDir)
	require.NoError(t, err)

	// Validate output structure
	assert.Equal(t, "1.0", output.SchemaVersion)
	assert.NotEmpty(t, output.ID)
	assert.Equal(t, models.OutputStatusCompleted, output.Status)

	// Validate metadata
	assert.NotZero(t, output.Metadata.StartedAt)
	assert.NotNil(t, output.Metadata.CompletedAt)
	assert.Greater(t, output.Metadata.Duration.Milliseconds(), int64(0))

	// Validate files
	for _, file := range output.Files {
		assert.NotEmpty(t, file.Path)
		assert.NotEmpty(t, file.Content)
		assert.NotEmpty(t, file.Checksum)
		assert.True(t, file.VerifyChecksum(), "Checksum should match content")
	}
}

// Helper functions

func createMockFileOps(t *testing.T) fsops.FileOps {
	tmpDir := t.TempDir()
	fileOps, err := fsops.New(fsops.Config{
		RootDir: tmpDir,
		Logger:  &noopFsLogger{},
	})
	require.NoError(t, err)
	return fileOps
}

func createCompleteTestFCS() *models.FinalClarifiedSpecification {
	return &models.FinalClarifiedSpecification{
		SchemaVersion: "1.0",
		ID:            "test-fcs-complete",
		Version:       "1.0",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{
					ID:          "FR-001",
					Description: "Application must start successfully",
					Priority:    "critical",
				},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{
					Name:    "main",
					Path:    "cmd/app",
					Purpose: "Main application entry point",
				},
			},
			Dependencies: []models.Dependency{
				{
					Name:    "github.com/stretchr/testify",
					Version: "v1.8.0",
					Purpose: "Testing framework",
				},
			},
		},
		DataModel: models.DataModel{
			Entities: []models.Entity{},
		},
		TestingStrategy: models.TestingStrategy{
			CoverageTarget:   85.0,
			UnitTests:        true,
			IntegrationTests: false,
		},
		BuildConfig: models.BuildConfig{
			GoVersion:  "1.23",
			OutputPath: "./bin",
		},
	}
}

// noopFsLogger is a no-op logger for fsops
type noopFsLogger struct{}

func (n *noopFsLogger) LogFileOperation(ctx context.Context, op models.FileOperationLog) error {
	return nil
}

func (n *noopFsLogger) LogDecision(ctx context.Context, log models.DecisionLog) error {
	return nil
}

func (n *noopFsLogger) LogError(ctx context.Context, component, operation, message string, err error) error {
	return nil
}

func (n *noopFsLogger) LogInfo(ctx context.Context, component, operation, message string) error {
	return nil
}

func (n *noopFsLogger) Close() error {
	return nil
}
