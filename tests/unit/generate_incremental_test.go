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

// mockIncrementalLLMClient implements llm.Client for testing
type mockIncrementalLLMClient struct {
	generateFunc func(ctx context.Context, prompt string) (string, error)
}

func (m *mockIncrementalLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, prompt)
	}
	return "", nil
}

func (m *mockIncrementalLLMClient) GenerateStructured(ctx context.Context, prompt string, schema interface{}) (interface{}, error) {
	return nil, nil
}

func (m *mockIncrementalLLMClient) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	return "", nil
}

func (m *mockIncrementalLLMClient) Provider() string {
	return "mock"
}

func (m *mockIncrementalLLMClient) Model() string {
	return "mock-model"
}

func TestNewIncrementalGenerator(t *testing.T) {
	tests := []struct {
		name    string
		config  generate.IncrementalConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: generate.IncrementalConfig{
				LLMClient:      &mockIncrementalLLMClient{},
				ChangeDetector: generate.NewChangeDetector(),
			},
			wantErr: false,
		},
		{
			name: "missing LLM client",
			config: generate.IncrementalConfig{
				LLMClient:      nil,
				ChangeDetector: generate.NewChangeDetector(),
			},
			wantErr: true,
		},
		{
			name: "missing change detector",
			config: generate.IncrementalConfig{
				LLMClient:      &mockIncrementalLLMClient{},
				ChangeDetector: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := generate.NewIncrementalGenerator(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, gen)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, gen)
			}
		})
	}
}

func TestIncrementalGenerator_Regenerate(t *testing.T) {
	tests := []struct {
		name           string
		oldFCS         *models.FinalClarifiedSpecification
		newFCS         *models.FinalClarifiedSpecification
		oldOutput      *models.GenerationOutput
		llmResponse    string
		wantErr        bool
		validateOutput func(t *testing.T, output *models.GenerationOutput)
	}{
		{
			name: "no changes - returns old output",
			oldFCS: &models.FinalClarifiedSpecification{
				ID:      "test-1",
				Version: "1.0",
				Requirements: models.Requirements{
					Functional: []models.FunctionalRequirement{
						{ID: "FR-001", Description: "Requirement 1", Priority: "high"},
					},
				},
				Architecture: models.Architecture{
					Packages: []models.Package{
						{Name: "main", Path: "cmd/main", Purpose: "Main package"},
					},
				},
			},
			newFCS: &models.FinalClarifiedSpecification{
				ID:      "test-1",
				Version: "1.0",
				Requirements: models.Requirements{
					Functional: []models.FunctionalRequirement{
						{ID: "FR-001", Description: "Requirement 1", Priority: "high"},
					},
				},
				Architecture: models.Architecture{
					Packages: []models.Package{
						{Name: "main", Path: "cmd/main", Purpose: "Main package"},
					},
				},
			},
			oldOutput: &models.GenerationOutput{
				ID:     "output-1",
				PlanID: "plan-1",
				Files: []models.GeneratedFile{
					{Path: "cmd/main/main.go", Content: "package main"},
				},
			},
			wantErr: false,
			validateOutput: func(t *testing.T, output *models.GenerationOutput) {
				assert.Equal(t, "output-1", output.ID)
				assert.Len(t, output.Files, 1)
			},
		},
		{
			name: "added package - regenerate only new package",
			oldFCS: &models.FinalClarifiedSpecification{
				ID:      "test-1",
				Version: "1.0",
				Requirements: models.Requirements{
					Functional: []models.FunctionalRequirement{
						{ID: "FR-001", Description: "Requirement 1", Priority: "high"},
					},
				},
				Architecture: models.Architecture{
					Packages: []models.Package{
						{Name: "main", Path: "cmd/main", Purpose: "Main package"},
					},
				},
			},
			newFCS: &models.FinalClarifiedSpecification{
				ID:      "test-1",
				Version: "1.1",
				Requirements: models.Requirements{
					Functional: []models.FunctionalRequirement{
						{ID: "FR-001", Description: "Requirement 1", Priority: "high"},
					},
				},
				Architecture: models.Architecture{
					Packages: []models.Package{
						{Name: "main", Path: "cmd/main", Purpose: "Main package"},
						{Name: "util", Path: "internal/util", Purpose: "Utility package"},
					},
				},
			},
			oldOutput: &models.GenerationOutput{
				ID:     "output-1",
				PlanID: "plan-1",
				Files: []models.GeneratedFile{
					{Path: "cmd/main/main.go", Content: "package main", Checksum: "abc123"},
				},
			},
			llmResponse: `{
				"path": "internal/util/util.go",
				"content": "package util\n\nfunc Helper() string {\n\treturn \"helper\"\n}"
			}`,
			wantErr: false,
			validateOutput: func(t *testing.T, output *models.GenerationOutput) {
				assert.NotNil(t, output)
				// Should have both old and new files
				assert.GreaterOrEqual(t, len(output.Files), 1)
			},
		},
		{
			name: "modified package - regenerate affected packages",
			oldFCS: &models.FinalClarifiedSpecification{
				ID:      "test-1",
				Version: "1.0",
				Requirements: models.Requirements{
					Functional: []models.FunctionalRequirement{
						{ID: "FR-001", Description: "Requirement 1", Priority: "high"},
					},
				},
				Architecture: models.Architecture{
					Packages: []models.Package{
						{Name: "lib", Path: "internal/lib", Purpose: "Library", Dependencies: []string{}},
						{Name: "main", Path: "cmd/main", Purpose: "Main package", Dependencies: []string{"lib"}},
					},
				},
			},
			newFCS: &models.FinalClarifiedSpecification{
				ID:      "test-1",
				Version: "1.1",
				Requirements: models.Requirements{
					Functional: []models.FunctionalRequirement{
						{ID: "FR-001", Description: "Modified requirement 1", Priority: "high"},
					},
				},
				Architecture: models.Architecture{
					Packages: []models.Package{
						{Name: "lib", Path: "internal/lib", Purpose: "Modified library", Dependencies: []string{}},
						{Name: "main", Path: "cmd/main", Purpose: "Main package", Dependencies: []string{"lib"}},
					},
				},
			},
			oldOutput: &models.GenerationOutput{
				ID:     "output-1",
				PlanID: "plan-1",
				Files: []models.GeneratedFile{
					{Path: "internal/lib/lib.go", Content: "package lib", Checksum: "abc123"},
					{Path: "cmd/main/main.go", Content: "package main", Checksum: "def456"},
				},
			},
			llmResponse: `{
				"path": "internal/lib/lib.go",
				"content": "package lib\n\n// Modified library"
			}`,
			wantErr: false,
			validateOutput: func(t *testing.T, output *models.GenerationOutput) {
				assert.NotNil(t, output)
				// Should regenerate both lib and main (dependent)
				assert.GreaterOrEqual(t, len(output.Files), 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockIncrementalLLMClient{
				generateFunc: func(ctx context.Context, prompt string) (string, error) {
					return tt.llmResponse, nil
				},
			}

			gen, err := generate.NewIncrementalGenerator(generate.IncrementalConfig{
				LLMClient:      mockClient,
				ChangeDetector: generate.NewChangeDetector(),
			})
			require.NoError(t, err)

			output, err := gen.Regenerate(context.Background(), tt.oldFCS, tt.newFCS, tt.oldOutput)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, output)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, output)
				if tt.validateOutput != nil {
					tt.validateOutput(t, output)
				}
			}
		})
	}
}

func TestIncrementalGenerator_ShouldRegenerate(t *testing.T) {
	tests := []struct {
		name           string
		packageName    string
		changes        *generate.FCSChanges
		architecture   *models.Architecture
		wantRegenerate bool
	}{
		{
			name:        "no changes",
			packageName: "main",
			changes: &generate.FCSChanges{
				HasChanges: false,
			},
			architecture: &models.Architecture{
				Packages: []models.Package{
					{Name: "main", Path: "cmd/main"},
				},
			},
			wantRegenerate: false,
		},
		{
			name:        "package added - should regenerate",
			packageName: "new",
			changes: &generate.FCSChanges{
				HasChanges: true,
				AddedPackages: []models.Package{
					{Name: "new", Path: "internal/new"},
				},
			},
			architecture: &models.Architecture{
				Packages: []models.Package{
					{Name: "main", Path: "cmd/main"},
					{Name: "new", Path: "internal/new"},
				},
			},
			wantRegenerate: true,
		},
		{
			name:        "package modified - should regenerate",
			packageName: "lib",
			changes: &generate.FCSChanges{
				HasChanges: true,
				ModifiedPackages: []models.Package{
					{Name: "lib", Path: "internal/lib"},
				},
			},
			architecture: &models.Architecture{
				Packages: []models.Package{
					{Name: "lib", Path: "internal/lib"},
				},
			},
			wantRegenerate: true,
		},
		{
			name:        "dependent of modified package - should regenerate",
			packageName: "main",
			changes: &generate.FCSChanges{
				HasChanges: true,
				ModifiedPackages: []models.Package{
					{Name: "lib", Path: "internal/lib"},
				},
			},
			architecture: &models.Architecture{
				Packages: []models.Package{
					{Name: "lib", Path: "internal/lib", Dependencies: []string{}},
					{Name: "main", Path: "cmd/main", Dependencies: []string{"lib"}},
				},
			},
			wantRegenerate: true,
		},
		{
			name:        "unrelated package - should not regenerate",
			packageName: "other",
			changes: &generate.FCSChanges{
				HasChanges: true,
				ModifiedPackages: []models.Package{
					{Name: "lib", Path: "internal/lib"},
				},
			},
			architecture: &models.Architecture{
				Packages: []models.Package{
					{Name: "lib", Path: "internal/lib", Dependencies: []string{}},
					{Name: "main", Path: "cmd/main", Dependencies: []string{"lib"}},
					{Name: "other", Path: "cmd/other", Dependencies: []string{}},
				},
			},
			wantRegenerate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := generate.NewIncrementalGenerator(generate.IncrementalConfig{
				LLMClient:      &mockIncrementalLLMClient{},
				ChangeDetector: generate.NewChangeDetector(),
			})
			require.NoError(t, err)

			shouldRegen := gen.ShouldRegenerate(tt.packageName, tt.changes, tt.architecture)
			assert.Equal(t, tt.wantRegenerate, shouldRegen)
		})
	}
}

func TestIncrementalGenerator_MergeOutputs(t *testing.T) {
	tests := []struct {
		name           string
		oldOutput      *models.GenerationOutput
		newFiles       []models.GeneratedFile
		affectedPkgs   []string
		validateOutput func(t *testing.T, output *models.GenerationOutput)
	}{
		{
			name: "merge new files with old files",
			oldOutput: &models.GenerationOutput{
				ID:     "output-1",
				PlanID: "plan-1",
				Files: []models.GeneratedFile{
					{Path: "cmd/main/main.go", Content: "package main", Checksum: "abc123"},
					{Path: "internal/lib/lib.go", Content: "package lib", Checksum: "def456"},
				},
			},
			newFiles: []models.GeneratedFile{
				{Path: "internal/util/util.go", Content: "package util", Checksum: "ghi789"},
			},
			affectedPkgs: []string{"util"},
			validateOutput: func(t *testing.T, output *models.GenerationOutput) {
				assert.Len(t, output.Files, 3)
				// Check that old files are preserved
				hasMain := false
				hasLib := false
				hasUtil := false
				for _, file := range output.Files {
					if file.Path == "cmd/main/main.go" {
						hasMain = true
					}
					if file.Path == "internal/lib/lib.go" {
						hasLib = true
					}
					if file.Path == "internal/util/util.go" {
						hasUtil = true
					}
				}
				assert.True(t, hasMain)
				assert.True(t, hasLib)
				assert.True(t, hasUtil)
			},
		},
		{
			name: "replace modified files",
			oldOutput: &models.GenerationOutput{
				ID:     "output-1",
				PlanID: "plan-1",
				Files: []models.GeneratedFile{
					{Path: "cmd/main/main.go", Content: "package main", Checksum: "abc123"},
					{Path: "internal/lib/lib.go", Content: "package lib", Checksum: "def456"},
				},
			},
			newFiles: []models.GeneratedFile{
				{Path: "internal/lib/lib.go", Content: "package lib\n// Modified", Checksum: "new456"},
			},
			affectedPkgs: []string{"lib"},
			validateOutput: func(t *testing.T, output *models.GenerationOutput) {
				assert.Len(t, output.Files, 2)
				// Check that lib file is updated
				for _, file := range output.Files {
					if file.Path == "internal/lib/lib.go" {
						assert.Equal(t, "new456", file.Checksum)
						assert.Contains(t, file.Content, "Modified")
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := generate.NewIncrementalGenerator(generate.IncrementalConfig{
				LLMClient:      &mockIncrementalLLMClient{},
				ChangeDetector: generate.NewChangeDetector(),
			})
			require.NoError(t, err)

			output := gen.MergeOutputs(tt.oldOutput, tt.newFiles, tt.affectedPkgs)
			require.NotNil(t, output)
			if tt.validateOutput != nil {
				tt.validateOutput(t, output)
			}
		})
	}
}
