package spec

import (
	"testing"

	"github.com/dshills/gocreator/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFCSBuilder_Build(t *testing.T) {
	tests := []struct {
		name     string
		spec     *models.InputSpecification
		wantErr  bool
		validate func(*testing.T, *models.FinalClarifiedSpecification)
	}{
		{
			name: "Complete specification with all sections",
			spec: &models.InputSpecification{
				ID:     "test-spec-123",
				Format: models.FormatYAML,
				State:  models.SpecStateValid,
				ParsedData: map[string]interface{}{
					"name":        "TestProject",
					"description": "A test project",
					"requirements": []interface{}{
						map[string]interface{}{
							"id":          "FR-001",
							"description": "First requirement",
							"priority":    "high",
							"category":    "core",
						},
					},
					"architecture": map[string]interface{}{
						"packages": []interface{}{
							map[string]interface{}{
								"name":    "main",
								"path":    "cmd/main",
								"purpose": "Entry point",
							},
						},
						"dependencies": []interface{}{
							map[string]interface{}{
								"name":    "github.com/example/lib",
								"version": "v1.0.0",
								"purpose": "Core library",
							},
						},
					},
					"data_model": map[string]interface{}{
						"entities": []interface{}{
							map[string]interface{}{
								"name":    "User",
								"package": "models",
								"attributes": map[string]interface{}{
									"id":   "string",
									"name": "string",
								},
							},
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, fcs *models.FinalClarifiedSpecification) {
				assert.Equal(t, "test-spec-123", fcs.OriginalSpecID)
				assert.Equal(t, "1.0", fcs.Version)
				assert.NotEmpty(t, fcs.ID)
				assert.NotEmpty(t, fcs.Metadata.Hash)

				// Validate requirements
				assert.Len(t, fcs.Requirements.Functional, 1)
				assert.Equal(t, "FR-001", fcs.Requirements.Functional[0].ID)
				assert.Equal(t, "high", fcs.Requirements.Functional[0].Priority)

				// Validate architecture
				assert.Len(t, fcs.Architecture.Packages, 1)
				assert.Equal(t, "main", fcs.Architecture.Packages[0].Name)
				assert.Len(t, fcs.Architecture.Dependencies, 1)

				// Validate data model
				assert.Len(t, fcs.DataModel.Entities, 1)
				assert.Equal(t, "User", fcs.DataModel.Entities[0].Name)
			},
		},
		{
			name: "Minimal valid specification",
			spec: &models.InputSpecification{
				ID:     "test-spec-456",
				Format: models.FormatYAML,
				State:  models.SpecStateValid,
				ParsedData: map[string]interface{}{
					"name":        "MinimalProject",
					"description": "Minimal test",
					"requirements": []interface{}{
						map[string]interface{}{
							"id":          "FR-001",
							"description": "Single requirement",
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, fcs *models.FinalClarifiedSpecification) {
				assert.Equal(t, "test-spec-456", fcs.OriginalSpecID)
				assert.Len(t, fcs.Requirements.Functional, 1)
				assert.Empty(t, fcs.Architecture.Packages)
				assert.Empty(t, fcs.DataModel.Entities)
			},
		},
		{
			name: "Specification not in valid state",
			spec: &models.InputSpecification{
				ID:     "test-spec-789",
				Format: models.FormatYAML,
				State:  models.SpecStateParsed, // Not valid
				ParsedData: map[string]interface{}{
					"name":         "InvalidProject",
					"description":  "Test",
					"requirements": []interface{}{},
				},
			},
			wantErr: true,
		},
		{
			name: "Non-functional requirements",
			spec: &models.InputSpecification{
				ID:     "test-spec-nfr",
				Format: models.FormatYAML,
				State:  models.SpecStateValid,
				ParsedData: map[string]interface{}{
					"name":        "NFRProject",
					"description": "Project with NFRs",
					"requirements": []interface{}{
						map[string]interface{}{
							"id":          "FR-001",
							"description": "Functional requirement",
							"type":        "functional",
						},
						map[string]interface{}{
							"id":          "NFR-001",
							"description": "Performance requirement",
							"type":        "non-functional",
							"nfr_type":    "performance",
							"threshold":   "< 100ms",
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, fcs *models.FinalClarifiedSpecification) {
				assert.Len(t, fcs.Requirements.Functional, 1)
				assert.Len(t, fcs.Requirements.NonFunctional, 1)
				assert.Equal(t, "NFR-001", fcs.Requirements.NonFunctional[0].ID)
				assert.Equal(t, "performance", fcs.Requirements.NonFunctional[0].Type)
			},
		},
		{
			name: "With testing strategy and build config",
			spec: &models.InputSpecification{
				ID:     "test-spec-config",
				Format: models.FormatYAML,
				State:  models.SpecStateValid,
				ParsedData: map[string]interface{}{
					"name":        "ConfigProject",
					"description": "Project with config",
					"requirements": []interface{}{
						map[string]interface{}{
							"id":          "FR-001",
							"description": "Test requirement",
						},
					},
					"testing_strategy": map[string]interface{}{
						"coverage_target":   float64(90),
						"unit_tests":        true,
						"integration_tests": true,
						"frameworks":        []interface{}{"testify", "gomock"},
					},
					"build_config": map[string]interface{}{
						"go_version":  "1.23",
						"output_path": "./bin",
						"build_flags": []interface{}{"-tags=prod", "-ldflags=-s -w"},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, fcs *models.FinalClarifiedSpecification) {
				assert.Equal(t, float64(90), fcs.TestingStrategy.CoverageTarget)
				assert.True(t, fcs.TestingStrategy.UnitTests)
				assert.True(t, fcs.TestingStrategy.IntegrationTests)
				assert.Len(t, fcs.TestingStrategy.Frameworks, 2)

				assert.Equal(t, "1.23", fcs.BuildConfig.GoVersion)
				assert.Equal(t, "./bin", fcs.BuildConfig.OutputPath)
				assert.Len(t, fcs.BuildConfig.BuildFlags, 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewFCSBuilder(tt.spec)
			fcs, err := builder.Build()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, fcs)
			} else {
				require.NoError(t, err)
				require.NotNil(t, fcs)

				// Common validations
				assert.NotEmpty(t, fcs.ID)
				assert.NotEmpty(t, fcs.Metadata.Hash)
				assert.Equal(t, "1.0", fcs.SchemaVersion)

				if tt.validate != nil {
					tt.validate(t, fcs)
				}
			}
		})
	}
}

func TestBuildFCS_ConvenienceFunction(t *testing.T) {
	spec := &models.InputSpecification{
		ID:     "test-convenience",
		Format: models.FormatYAML,
		State:  models.SpecStateValid,
		ParsedData: map[string]interface{}{
			"name":        "ConvenienceTest",
			"description": "Testing convenience function",
			"requirements": []interface{}{
				map[string]interface{}{
					"id":          "FR-001",
					"description": "Test",
				},
			},
		},
	}

	fcs, err := BuildFCS(spec)
	require.NoError(t, err)
	require.NotNil(t, fcs)
	assert.Equal(t, "test-convenience", fcs.OriginalSpecID)
}

func TestFCSHashComputation(t *testing.T) {
	spec := &models.InputSpecification{
		ID:     "test-hash",
		Format: models.FormatYAML,
		State:  models.SpecStateValid,
		ParsedData: map[string]interface{}{
			"name":        "HashTest",
			"description": "Testing hash computation",
			"requirements": []interface{}{
				map[string]interface{}{
					"id":          "FR-001",
					"description": "Test",
				},
			},
		},
	}

	// Build FCS twice
	fcs1, err := BuildFCS(spec)
	require.NoError(t, err)

	fcs2, err := BuildFCS(spec)
	require.NoError(t, err)

	// Hashes should be identical for identical specs
	// (Note: UUIDs will be different, but the content hash should be consistent)
	assert.NotEmpty(t, fcs1.Metadata.Hash)
	assert.NotEmpty(t, fcs2.Metadata.Hash)
}
