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

// trackingLLMClient tracks which packages were generated
type trackingLLMClient struct {
	generatedPackages []string
}

func (t *trackingLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	// Track which package is being generated based on prompt
	if containsString(prompt, "Package Name: auth") {
		t.generatedPackages = append(t.generatedPackages, "auth")
		return `{
			"path": "internal/auth/auth.go",
			"content": "package auth\n\nfunc Authenticate() {}"
		}`, nil
	} else if containsString(prompt, "Package Name: logger") {
		t.generatedPackages = append(t.generatedPackages, "logger")
		return `{
			"path": "internal/logger/logger.go",
			"content": "package logger\n\nfunc Log(msg string) {}"
		}`, nil
	} else if containsString(prompt, "Package Name: api") {
		t.generatedPackages = append(t.generatedPackages, "api")
		return `{
			"path": "internal/api/api.go",
			"content": "package api\n\nfunc Handler() {}"
		}`, nil
	} else if containsString(prompt, "Package Name: database") {
		t.generatedPackages = append(t.generatedPackages, "database")
		return `{
			"path": "internal/database/db.go",
			"content": "package database\n\nfunc Connect() {}"
		}`, nil
	}

	return `{"path": "test.go", "content": "package test"}`, nil
}

func (t *trackingLLMClient) GenerateStructured(ctx context.Context, prompt string, schema interface{}) (interface{}, error) {
	return nil, nil
}

func (t *trackingLLMClient) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	return "", nil
}

func (t *trackingLLMClient) Provider() string {
	return "tracking"
}

func (t *trackingLLMClient) Model() string {
	return "tracking-model"
}

// TestUS4_OnlyAffectedFilesRegenerated tests that only affected files are regenerated
func TestUS4_OnlyAffectedFilesRegenerated(t *testing.T) {
	// Original FCS with multiple packages
	originalFCS := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fcs-incremental",
		Version:        "1.0",
		OriginalSpecID: "spec-incremental",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{ID: "FR-001", Description: "Authentication", Priority: "high"},
				{ID: "FR-002", Description: "Logging", Priority: "medium"},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "auth", Path: "internal/auth", Purpose: "Authentication"},
				{Name: "logger", Path: "internal/logger", Purpose: "Logging"},
			},
		},
	}

	trackingClient := &trackingLLMClient{
		generatedPackages: []string{},
	}

	generator, err := generate.NewIncrementalGenerator(generate.IncrementalConfig{
		LLMClient:      trackingClient,
		ChangeDetector: generate.NewChangeDetector(),
	})
	require.NoError(t, err)

	// Initial generation
	initialOutput := &models.GenerationOutput{
		SchemaVersion: "1.0",
		ID:            "output-1",
		PlanID:        "plan-1",
		Files:         []models.GeneratedFile{},
		Status:        models.OutputStatusCompleted,
	}

	output1, err := generator.Regenerate(context.Background(), originalFCS, originalFCS, initialOutput)
	require.NoError(t, err)

	// Reset tracking
	generatedInFirstRun := len(trackingClient.generatedPackages)
	trackingClient.generatedPackages = []string{}

	// Modify only one package (logger)
	modifiedFCS := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fcs-incremental",
		Version:        "1.1",
		OriginalSpecID: "spec-incremental",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{ID: "FR-001", Description: "Authentication", Priority: "high"},
				{ID: "FR-002", Description: "Enhanced logging with levels", Priority: "medium"}, // Modified
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "auth", Path: "internal/auth", Purpose: "Authentication"},       // Unchanged
				{Name: "logger", Path: "internal/logger", Purpose: "Enhanced logging"}, // Modified
			},
		},
	}

	// Regenerate with modified spec
	output2, err := generator.Regenerate(context.Background(), originalFCS, modifiedFCS, output1)
	require.NoError(t, err)
	require.NotNil(t, output2)

	// Verify only logger package was regenerated
	generatedInSecondRun := len(trackingClient.generatedPackages)
	assert.Less(t, generatedInSecondRun, generatedInFirstRun,
		"incremental regeneration should generate fewer packages than full regeneration")

	// Verify logger was regenerated but auth was not
	assert.Contains(t, trackingClient.generatedPackages, "logger",
		"modified package should be regenerated")
}

// TestUS4_DependentPackagesRegenerated tests that dependent packages are also regenerated
func TestUS4_DependentPackagesRegenerated(t *testing.T) {
	// Create FCS with package dependencies
	originalFCS := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fcs-deps",
		Version:        "1.0",
		OriginalSpecID: "spec-deps",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{ID: "FR-001", Description: "Database access", Priority: "high"},
				{ID: "FR-002", Description: "API layer", Priority: "high"},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "database", Path: "internal/database", Purpose: "Database", Dependencies: []string{}},
				{Name: "api", Path: "internal/api", Purpose: "API", Dependencies: []string{"database"}}, // Depends on database
			},
		},
	}

	trackingClient := &trackingLLMClient{
		generatedPackages: []string{},
	}

	generator, err := generate.NewIncrementalGenerator(generate.IncrementalConfig{
		LLMClient:      trackingClient,
		ChangeDetector: generate.NewChangeDetector(),
	})
	require.NoError(t, err)

	// Initial generation
	initialOutput := &models.GenerationOutput{
		SchemaVersion: "1.0",
		ID:            "output-deps",
		PlanID:        "plan-deps",
		Files:         []models.GeneratedFile{},
		Status:        models.OutputStatusCompleted,
	}

	output1, err := generator.Regenerate(context.Background(), originalFCS, originalFCS, initialOutput)
	require.NoError(t, err)

	// Reset tracking
	trackingClient.generatedPackages = []string{}

	// Modify database package (the dependency)
	modifiedFCS := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fcs-deps",
		Version:        "1.1",
		OriginalSpecID: "spec-deps",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{ID: "FR-001", Description: "Enhanced database access", Priority: "high"}, // Modified
				{ID: "FR-002", Description: "API layer", Priority: "high"},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "database", Path: "internal/database", Purpose: "Enhanced database", Dependencies: []string{}}, // Modified
				{Name: "api", Path: "internal/api", Purpose: "API", Dependencies: []string{"database"}},
			},
		},
	}

	// Regenerate
	output2, err := generator.Regenerate(context.Background(), originalFCS, modifiedFCS, output1)
	require.NoError(t, err)
	require.NotNil(t, output2)

	// Verify both database and api were regenerated (api depends on database)
	assert.Contains(t, trackingClient.generatedPackages, "database",
		"modified package should be regenerated")
	assert.Contains(t, trackingClient.generatedPackages, "api",
		"dependent package should also be regenerated")
}

// TestUS4_UnaffectedPackagesPreserved tests that unaffected packages are not regenerated
func TestUS4_UnaffectedPackagesPreserved(t *testing.T) {
	originalFCS := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fcs-preserve",
		Version:        "1.0",
		OriginalSpecID: "spec-preserve",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{ID: "FR-001", Description: "Auth", Priority: "high"},
				{ID: "FR-002", Description: "Logging", Priority: "medium"},
				{ID: "FR-003", Description: "API", Priority: "high"},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "auth", Path: "internal/auth", Purpose: "Authentication"},
				{Name: "logger", Path: "internal/logger", Purpose: "Logging"},
				{Name: "api", Path: "internal/api", Purpose: "API"},
			},
		},
	}

	trackingClient := &trackingLLMClient{
		generatedPackages: []string{},
	}

	generator, err := generate.NewIncrementalGenerator(generate.IncrementalConfig{
		LLMClient:      trackingClient,
		ChangeDetector: generate.NewChangeDetector(),
	})
	require.NoError(t, err)

	// Initial generation
	initialOutput := &models.GenerationOutput{
		SchemaVersion: "1.0",
		ID:            "output-preserve",
		PlanID:        "plan-preserve",
		Files:         []models.GeneratedFile{},
		Status:        models.OutputStatusCompleted,
	}

	output1, err := generator.Regenerate(context.Background(), originalFCS, originalFCS, initialOutput)
	require.NoError(t, err)

	// Store original auth package content
	var originalAuthContent string
	for _, file := range output1.Files {
		if containsString(file.Path, "auth") {
			originalAuthContent = file.Content
			break
		}
	}

	// Reset tracking
	trackingClient.generatedPackages = []string{}

	// Modify only logger package
	modifiedFCS := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fcs-preserve",
		Version:        "1.1",
		OriginalSpecID: "spec-preserve",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{ID: "FR-001", Description: "Auth", Priority: "high"},               // Unchanged
				{ID: "FR-002", Description: "Enhanced logging", Priority: "medium"}, // Modified
				{ID: "FR-003", Description: "API", Priority: "high"},                // Unchanged
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "auth", Path: "internal/auth", Purpose: "Authentication"},       // Unchanged
				{Name: "logger", Path: "internal/logger", Purpose: "Enhanced logging"}, // Modified
				{Name: "api", Path: "internal/api", Purpose: "API"},                    // Unchanged
			},
		},
	}

	// Regenerate
	output2, err := generator.Regenerate(context.Background(), originalFCS, modifiedFCS, output1)
	require.NoError(t, err)

	// Verify auth package content is preserved
	var newAuthContent string
	for _, file := range output2.Files {
		if containsString(file.Path, "auth") {
			newAuthContent = file.Content
			break
		}
	}

	assert.Equal(t, originalAuthContent, newAuthContent,
		"unaffected package content should be preserved")

	// Verify only logger was regenerated
	assert.Contains(t, trackingClient.generatedPackages, "logger",
		"modified package should be regenerated")
	assert.NotContains(t, trackingClient.generatedPackages, "auth",
		"unaffected package should not be regenerated")
	assert.NotContains(t, trackingClient.generatedPackages, "api",
		"unaffected package should not be regenerated")
}

// TestUS4_AddedPackageDoesNotAffectExisting tests that adding a new package doesn't regenerate existing ones
func TestUS4_AddedPackageDoesNotAffectExisting(t *testing.T) {
	originalFCS := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fcs-add",
		Version:        "1.0",
		OriginalSpecID: "spec-add",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{ID: "FR-001", Description: "Core functionality", Priority: "high"},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "auth", Path: "internal/auth", Purpose: "Authentication"},
			},
		},
	}

	trackingClient := &trackingLLMClient{
		generatedPackages: []string{},
	}

	generator, err := generate.NewIncrementalGenerator(generate.IncrementalConfig{
		LLMClient:      trackingClient,
		ChangeDetector: generate.NewChangeDetector(),
	})
	require.NoError(t, err)

	// Initial generation
	initialOutput := &models.GenerationOutput{
		SchemaVersion: "1.0",
		ID:            "output-add",
		PlanID:        "plan-add",
		Files:         []models.GeneratedFile{},
		Status:        models.OutputStatusCompleted,
	}

	output1, err := generator.Regenerate(context.Background(), originalFCS, originalFCS, initialOutput)
	require.NoError(t, err)

	// Reset tracking
	trackingClient.generatedPackages = []string{}

	// Add new logger package
	modifiedFCS := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fcs-add",
		Version:        "1.1",
		OriginalSpecID: "spec-add",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{ID: "FR-001", Description: "Core functionality", Priority: "high"},
				{ID: "FR-002", Description: "Logging", Priority: "medium"}, // Added
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "auth", Path: "internal/auth", Purpose: "Authentication"}, // Unchanged
				{Name: "logger", Path: "internal/logger", Purpose: "Logging"},    // Added
			},
		},
	}

	// Regenerate
	output2, err := generator.Regenerate(context.Background(), originalFCS, modifiedFCS, output1)
	require.NoError(t, err)

	// Verify only the new logger package was generated
	assert.Contains(t, trackingClient.generatedPackages, "logger",
		"new package should be generated")
	assert.NotContains(t, trackingClient.generatedPackages, "auth",
		"existing package should not be regenerated when adding unrelated package")

	// Verify both packages exist in output
	hasAuth := false
	hasLogger := false
	for _, file := range output2.Files {
		if containsString(file.Path, "auth") {
			hasAuth = true
		}
		if containsString(file.Path, "logger") {
			hasLogger = true
		}
	}
	assert.True(t, hasAuth, "original package should still exist")
	assert.True(t, hasLogger, "new package should be added")
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findStringMatch(s, substr)
}

func findStringMatch(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
