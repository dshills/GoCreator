package integration

import (
	"context"
	"testing"

	"github.com/dshills/gocreator/internal/generate"
	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLLMClientUS4 implements llm.Client for integration testing
type mockLLMClientUS4 struct {
	responseFunc func(prompt string) string
}

func (m *mockLLMClientUS4) Generate(ctx context.Context, prompt string) (string, error) {
	if m.responseFunc != nil {
		return m.responseFunc(prompt), nil
	}
	// Default response
	return `{
		"path": "test/file.go",
		"content": "package test\n\nfunc TestFunc() {}"
	}`, nil
}

func (m *mockLLMClientUS4) GenerateStructured(ctx context.Context, prompt string, schema interface{}) (interface{}, error) {
	return nil, nil
}

func (m *mockLLMClientUS4) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	return "", nil
}

func (m *mockLLMClientUS4) Provider() string {
	return "mock"
}

func (m *mockLLMClientUS4) Model() string {
	return "mock-model"
}

// TestUS4_SpecModificationReflectedInOutput tests that modifying a spec results in changes being reflected in regenerated output
func TestUS4_SpecModificationReflectedInOutput(t *testing.T) {
	// Create original FCS
	originalFCS := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fcs-1",
		Version:        "1.0",
		OriginalSpecID: "spec-1",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{
					ID:          "FR-001",
					Description: "Implement user authentication",
					Priority:    "high",
				},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{
					Name:    "auth",
					Path:    "internal/auth",
					Purpose: "Authentication package",
				},
			},
		},
	}

	// Generate initial output
	mockClient := &mockLLMClientUS4{
		responseFunc: func(prompt string) string {
			return `{
				"path": "internal/auth/auth.go",
				"content": "package auth\n\nfunc Authenticate() bool {\n\treturn true\n}"
			}`
		},
	}

	generator, err := generate.NewIncrementalGenerator(generate.IncrementalConfig{
		LLMClient:      mockClient,
		ChangeDetector: generate.NewChangeDetector(),
	})
	require.NoError(t, err)

	// Simulate initial generation with empty old output
	initialOutput := &models.GenerationOutput{
		SchemaVersion: "1.0",
		ID:            "output-1",
		PlanID:        "plan-1",
		Files:         []models.GeneratedFile{},
		Status:        models.OutputStatusCompleted,
	}

	// First generation
	output1, err := generator.Regenerate(context.Background(), originalFCS, originalFCS, initialOutput)
	require.NoError(t, err)
	require.NotNil(t, output1)

	// Modify the spec - add a new requirement and package
	modifiedFCS := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fcs-1",
		Version:        "1.1", // Version bumped
		OriginalSpecID: "spec-1",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{
					ID:          "FR-001",
					Description: "Implement user authentication",
					Priority:    "high",
				},
				{
					ID:          "FR-002",
					Description: "Add password reset functionality",
					Priority:    "medium",
				},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{
					Name:    "auth",
					Path:    "internal/auth",
					Purpose: "Authentication package",
				},
				{
					Name:    "reset",
					Path:    "internal/reset",
					Purpose: "Password reset package",
				},
			},
		},
	}

	// Update mock to return different content for new package
	mockClient.responseFunc = func(prompt string) string {
		if contains(prompt, "reset") {
			return `{
				"path": "internal/reset/reset.go",
				"content": "package reset\n\nfunc ResetPassword(email string) error {\n\treturn nil\n}"
			}`
		}
		return `{
			"path": "internal/auth/auth.go",
			"content": "package auth\n\nfunc Authenticate() bool {\n\treturn true\n}"
		}`
	}

	// Regenerate with modified spec
	output2, err := generator.Regenerate(context.Background(), originalFCS, modifiedFCS, output1)
	require.NoError(t, err)
	require.NotNil(t, output2)

	// Verify changes are reflected in output
	assert.Greater(t, len(output2.Files), len(output1.Files), "modified spec should result in more files")

	// Check that new package is present
	hasResetPackage := false
	for _, file := range output2.Files {
		if contains(file.Path, "reset") {
			hasResetPackage = true
			assert.Contains(t, file.Content, "ResetPassword", "new package should have expected functionality")
		}
	}
	assert.True(t, hasResetPackage, "new reset package should be present in output")
}

// TestUS4_ModifiedRequirementUpdatesCode tests that modifying a requirement updates the generated code
func TestUS4_ModifiedRequirementUpdatesCode(t *testing.T) {
	originalFCS := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fcs-2",
		Version:        "1.0",
		OriginalSpecID: "spec-2",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{
					ID:          "FR-001",
					Description: "Basic logging functionality",
					Priority:    "high",
				},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{
					Name:    "logger",
					Path:    "internal/logger",
					Purpose: "Logging package",
				},
			},
		},
	}

	mockClient := &mockLLMClientUS4{
		responseFunc: func(prompt string) string {
			if contains(prompt, "structured") {
				return `{
					"path": "internal/logger/logger.go",
					"content": "package logger\n\nimport \"encoding/json\"\n\nfunc LogJSON(v interface{}) {}"
				}`
			}
			return `{
				"path": "internal/logger/logger.go",
				"content": "package logger\n\nfunc Log(msg string) {}"
			}`
		},
	}

	generator, err := generate.NewIncrementalGenerator(generate.IncrementalConfig{
		LLMClient:      mockClient,
		ChangeDetector: generate.NewChangeDetector(),
	})
	require.NoError(t, err)

	initialOutput := &models.GenerationOutput{
		SchemaVersion: "1.0",
		ID:            "output-2",
		PlanID:        "plan-2",
		Files:         []models.GeneratedFile{},
		Status:        models.OutputStatusCompleted,
	}

	// First generation
	output1, err := generator.Regenerate(context.Background(), originalFCS, originalFCS, initialOutput)
	require.NoError(t, err)

	// Modify requirement to add structured logging
	modifiedFCS := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fcs-2",
		Version:        "1.1",
		OriginalSpecID: "spec-2",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{
					ID:          "FR-001",
					Description: "Structured logging with JSON support", // Modified
					Priority:    "high",
				},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{
					Name:    "logger",
					Path:    "internal/logger",
					Purpose: "Logging package with structured logging", // Modified
				},
			},
		},
	}

	// Regenerate
	output2, err := generator.Regenerate(context.Background(), originalFCS, modifiedFCS, output1)
	require.NoError(t, err)
	require.NotNil(t, output2)

	// Verify the modification is reflected
	found := false
	for _, file := range output2.Files {
		if contains(file.Path, "logger") {
			found = true
			// Check that new functionality is present
			assert.Contains(t, file.Content, "JSON", "modified requirement should add JSON logging")
		}
	}
	assert.True(t, found, "logger package should be present")
}

// TestUS4_DeletedPackageRemovedFromOutput tests that deleting a package removes it from output
func TestUS4_DeletedPackageRemovedFromOutput(t *testing.T) {
	originalFCS := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fcs-3",
		Version:        "1.0",
		OriginalSpecID: "spec-3",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{ID: "FR-001", Description: "Feature 1", Priority: "high"},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "core", Path: "internal/core", Purpose: "Core package"},
				{Name: "deprecated", Path: "internal/deprecated", Purpose: "Deprecated package"},
			},
		},
	}

	mockClient := &mockLLMClientUS4{}

	generator, err := generate.NewIncrementalGenerator(generate.IncrementalConfig{
		LLMClient:      mockClient,
		ChangeDetector: generate.NewChangeDetector(),
	})
	require.NoError(t, err)

	// Create initial output with both packages
	initialOutput := &models.GenerationOutput{
		SchemaVersion: "1.0",
		ID:            "output-3",
		PlanID:        "plan-3",
		Files: []models.GeneratedFile{
			{Path: "internal/core/core.go", Content: "package core"},
			{Path: "internal/deprecated/deprecated.go", Content: "package deprecated"},
		},
		Status: models.OutputStatusCompleted,
	}

	// Modify FCS to remove deprecated package
	modifiedFCS := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fcs-3",
		Version:        "1.1",
		OriginalSpecID: "spec-3",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{ID: "FR-001", Description: "Feature 1", Priority: "high"},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "core", Path: "internal/core", Purpose: "Core package"},
				// deprecated package removed
			},
		},
	}

	// Regenerate
	output2, err := generator.Regenerate(context.Background(), originalFCS, modifiedFCS, initialOutput)
	require.NoError(t, err)
	require.NotNil(t, output2)

	// Verify deprecated package is not in output
	for _, file := range output2.Files {
		assert.NotContains(t, file.Path, "deprecated", "deleted package should not be in output")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
