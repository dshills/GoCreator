package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/dshills/gocreator/internal/generate"
	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deterministicLLMClient returns consistent responses for testing idempotency
type deterministicLLMClient struct {
	responses map[string]string
}

func (d *deterministicLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	// Hash the prompt to get a consistent response
	hash := sha256.Sum256([]byte(prompt))
	key := hex.EncodeToString(hash[:8]) // Use first 8 bytes for key

	if resp, exists := d.responses[key]; exists {
		return resp, nil
	}

	// Default deterministic response based on prompt content
	if containsSubstr(prompt, "auth") {
		return `{
			"path": "internal/auth/auth.go",
			"content": "package auth\n\nfunc Authenticate() bool {\n\treturn true\n}"
		}`, nil
	}

	return `{
		"path": "test/default.go",
		"content": "package test\n\nfunc Default() {}"
	}`, nil
}

func (d *deterministicLLMClient) GenerateStructured(ctx context.Context, prompt string, schema interface{}) (interface{}, error) {
	return nil, nil
}

func (d *deterministicLLMClient) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	return "", nil
}

func (d *deterministicLLMClient) Provider() string {
	return "deterministic"
}

func (d *deterministicLLMClient) Model() string {
	return "deterministic-model"
}

// TestUS4_IdempotentRegeneration tests that regenerating with same spec produces identical output
func TestUS4_IdempotentRegeneration(t *testing.T) {
	fcs := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fcs-idempotent",
		Version:        "1.0",
		OriginalSpecID: "spec-idempotent",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{
					ID:          "FR-001",
					Description: "Implement authentication",
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

	mockClient := &deterministicLLMClient{
		responses: make(map[string]string),
	}

	generator, err := generate.NewIncrementalGenerator(generate.IncrementalConfig{
		LLMClient:      mockClient,
		ChangeDetector: generate.NewChangeDetector(),
	})
	require.NoError(t, err)

	initialOutput := &models.GenerationOutput{
		SchemaVersion: "1.0",
		ID:            "output-initial",
		PlanID:        "plan-1",
		Files:         []models.GeneratedFile{},
		Status:        models.OutputStatusCompleted,
	}

	// First generation
	output1, err := generator.Regenerate(context.Background(), fcs, fcs, initialOutput)
	require.NoError(t, err)
	require.NotNil(t, output1)

	// Second generation with same spec
	output2, err := generator.Regenerate(context.Background(), fcs, fcs, output1)
	require.NoError(t, err)
	require.NotNil(t, output2)

	// Verify outputs are identical
	assert.Equal(t, len(output1.Files), len(output2.Files), "number of files should be identical")

	// Compare files
	for i := range output1.Files {
		if i < len(output2.Files) {
			assert.Equal(t, output1.Files[i].Path, output2.Files[i].Path, "file paths should match")
			assert.Equal(t, output1.Files[i].Content, output2.Files[i].Content, "file contents should be identical")
			assert.Equal(t, output1.Files[i].Checksum, output2.Files[i].Checksum, "checksums should match")
		}
	}
}

// TestUS4_IdempotentWithModifiedSpec tests that two regenerations of the same modified spec produce identical results
func TestUS4_IdempotentWithModifiedSpec(t *testing.T) {
	originalFCS := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fcs-mod",
		Version:        "1.0",
		OriginalSpecID: "spec-mod",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{ID: "FR-001", Description: "Feature 1", Priority: "high"},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "core", Path: "internal/core", Purpose: "Core"},
			},
		},
	}

	modifiedFCS := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fcs-mod",
		Version:        "1.1", // Modified version
		OriginalSpecID: "spec-mod",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{ID: "FR-001", Description: "Feature 1", Priority: "high"},
				{ID: "FR-002", Description: "Feature 2", Priority: "medium"}, // Added
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "core", Path: "internal/core", Purpose: "Core"},
				{Name: "feature", Path: "internal/feature", Purpose: "Features"}, // Added
			},
		},
	}

	mockClient := &deterministicLLMClient{
		responses: make(map[string]string),
	}

	generator, err := generate.NewIncrementalGenerator(generate.IncrementalConfig{
		LLMClient:      mockClient,
		ChangeDetector: generate.NewChangeDetector(),
	})
	require.NoError(t, err)

	initialOutput := &models.GenerationOutput{
		SchemaVersion: "1.0",
		ID:            "output-base",
		PlanID:        "plan-base",
		Files:         []models.GeneratedFile{},
		Status:        models.OutputStatusCompleted,
	}

	// Generate from original
	outputOrig, err := generator.Regenerate(context.Background(), originalFCS, originalFCS, initialOutput)
	require.NoError(t, err)

	// First regeneration with modified spec
	outputMod1, err := generator.Regenerate(context.Background(), originalFCS, modifiedFCS, outputOrig)
	require.NoError(t, err)
	require.NotNil(t, outputMod1)

	// Second regeneration with same modified spec
	outputMod2, err := generator.Regenerate(context.Background(), originalFCS, modifiedFCS, outputOrig)
	require.NoError(t, err)
	require.NotNil(t, outputMod2)

	// Verify both regenerations produce identical output
	assert.Equal(t, len(outputMod1.Files), len(outputMod2.Files), "regenerations should produce same number of files")

	// Build maps for comparison (order might differ)
	files1 := make(map[string]models.GeneratedFile)
	for _, f := range outputMod1.Files {
		files1[f.Path] = f
	}

	files2 := make(map[string]models.GeneratedFile)
	for _, f := range outputMod2.Files {
		files2[f.Path] = f
	}

	// Compare contents
	for path, file1 := range files1 {
		file2, exists := files2[path]
		assert.True(t, exists, "file %s should exist in both outputs", path)
		if exists {
			assert.Equal(t, file1.Content, file2.Content, "file contents should be identical for %s", path)
			assert.Equal(t, file1.Checksum, file2.Checksum, "checksums should match for %s", path)
		}
	}
}

// TestUS4_DeterministicChecksums verifies that checksums are consistent across regenerations
func TestUS4_DeterministicChecksums(t *testing.T) {
	fcs := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fcs-checksum",
		Version:        "1.0",
		OriginalSpecID: "spec-checksum",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{ID: "FR-001", Description: "Data processing", Priority: "high"},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "processor", Path: "internal/processor", Purpose: "Data processor"},
			},
		},
	}

	// Use deterministic client with fixed response
	mockClient := &deterministicLLMClient{
		responses: map[string]string{
			// Pre-compute hash for this specific prompt pattern
			// In real scenario, LLM would return consistent output
		},
	}

	generator, err := generate.NewIncrementalGenerator(generate.IncrementalConfig{
		LLMClient:      mockClient,
		ChangeDetector: generate.NewChangeDetector(),
	})
	require.NoError(t, err)

	initialOutput := &models.GenerationOutput{
		SchemaVersion: "1.0",
		ID:            "output-checksum",
		PlanID:        "plan-checksum",
		Files:         []models.GeneratedFile{},
		Status:        models.OutputStatusCompleted,
	}

	// Generate multiple times
	var checksums []map[string]string

	for i := 0; i < 3; i++ {
		output, err := generator.Regenerate(context.Background(), fcs, fcs, initialOutput)
		require.NoError(t, err)

		fileChecksums := make(map[string]string)
		for _, file := range output.Files {
			fileChecksums[file.Path] = file.Checksum
		}
		checksums = append(checksums, fileChecksums)

		// Use output as base for next iteration
		initialOutput = output
	}

	// Verify all checksums are identical across runs
	if len(checksums) > 1 {
		for i := 1; i < len(checksums); i++ {
			for path, checksum := range checksums[0] {
				assert.Equal(t, checksum, checksums[i][path],
					"checksum for %s should be identical across regenerations", path)
			}
		}
	}
}

// TestUS4_NoChangesReturnsUnmodifiedOutput tests that regenerating without changes returns the same output
func TestUS4_NoChangesReturnsUnmodifiedOutput(t *testing.T) {
	fcs := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fcs-nochange",
		Version:        "1.0",
		OriginalSpecID: "spec-nochange",
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{ID: "FR-001", Description: "Unchanged feature", Priority: "high"},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "stable", Path: "internal/stable", Purpose: "Stable package"},
			},
		},
	}

	mockClient := &deterministicLLMClient{
		responses: make(map[string]string),
	}

	generator, err := generate.NewIncrementalGenerator(generate.IncrementalConfig{
		LLMClient:      mockClient,
		ChangeDetector: generate.NewChangeDetector(),
	})
	require.NoError(t, err)

	baseOutput := &models.GenerationOutput{
		SchemaVersion: "1.0",
		ID:            "output-stable",
		PlanID:        "plan-stable",
		Files: []models.GeneratedFile{
			{
				Path:     "internal/stable/stable.go",
				Content:  "package stable\n\nfunc Stable() {}",
				Checksum: computeChecksum("package stable\n\nfunc Stable() {}"),
			},
		},
		Status: models.OutputStatusCompleted,
	}

	// Regenerate with same FCS (no changes)
	output, err := generator.Regenerate(context.Background(), fcs, fcs, baseOutput)
	require.NoError(t, err)

	// Verify output is exactly the same
	assert.Equal(t, baseOutput.ID, output.ID, "output ID should remain the same when no changes")
	assert.Equal(t, len(baseOutput.Files), len(output.Files), "file count should not change")

	for i, baseFile := range baseOutput.Files {
		if i < len(output.Files) {
			assert.Equal(t, baseFile.Path, output.Files[i].Path, "file path should not change")
			assert.Equal(t, baseFile.Content, output.Files[i].Content, "file content should not change")
			assert.Equal(t, baseFile.Checksum, output.Files[i].Checksum, "checksum should not change")
		}
	}
}

// Helper functions
func computeChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

func containsSubstr(s, substr string) bool {
	return len(s) >= len(substr) && findSubstr(s, substr)
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
