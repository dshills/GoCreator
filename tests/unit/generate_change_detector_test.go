package unit

import (
	"testing"

	"github.com/dshills/gocreator/internal/generate"
	"github.com/dshills/gocreator/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChangeDetector(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "creates new change detector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := generate.NewChangeDetector()
			assert.NotNil(t, detector)
		})
	}
}

func TestChangeDetector_DetectChanges(t *testing.T) {
	tests := []struct {
		name            string
		oldFCS          *models.FinalClarifiedSpecification
		newFCS          *models.FinalClarifiedSpecification
		wantErr         bool
		validateChanges func(t *testing.T, changes *generate.FCSChanges)
	}{
		{
			name: "no changes - identical FCS",
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
			wantErr: false,
			validateChanges: func(t *testing.T, changes *generate.FCSChanges) {
				assert.False(t, changes.HasChanges)
				assert.Empty(t, changes.AddedRequirements)
				assert.Empty(t, changes.ModifiedRequirements)
				assert.Empty(t, changes.DeletedRequirements)
				assert.Empty(t, changes.AddedPackages)
				assert.Empty(t, changes.ModifiedPackages)
				assert.Empty(t, changes.DeletedPackages)
			},
		},
		{
			name: "added requirement",
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
						{ID: "FR-002", Description: "New requirement", Priority: "medium"},
					},
				},
				Architecture: models.Architecture{
					Packages: []models.Package{
						{Name: "main", Path: "cmd/main", Purpose: "Main package"},
					},
				},
			},
			wantErr: false,
			validateChanges: func(t *testing.T, changes *generate.FCSChanges) {
				assert.True(t, changes.HasChanges)
				assert.Len(t, changes.AddedRequirements, 1)
				assert.Equal(t, "FR-002", changes.AddedRequirements[0].ID)
				assert.Empty(t, changes.ModifiedRequirements)
				assert.Empty(t, changes.DeletedRequirements)
			},
		},
		{
			name: "modified requirement",
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
						{ID: "FR-001", Description: "Modified requirement", Priority: "high"},
					},
				},
				Architecture: models.Architecture{
					Packages: []models.Package{
						{Name: "main", Path: "cmd/main", Purpose: "Main package"},
					},
				},
			},
			wantErr: false,
			validateChanges: func(t *testing.T, changes *generate.FCSChanges) {
				assert.True(t, changes.HasChanges)
				assert.Empty(t, changes.AddedRequirements)
				assert.Len(t, changes.ModifiedRequirements, 1)
				assert.Equal(t, "FR-001", changes.ModifiedRequirements[0].ID)
				assert.Empty(t, changes.DeletedRequirements)
			},
		},
		{
			name: "deleted requirement",
			oldFCS: &models.FinalClarifiedSpecification{
				ID:      "test-1",
				Version: "1.0",
				Requirements: models.Requirements{
					Functional: []models.FunctionalRequirement{
						{ID: "FR-001", Description: "Requirement 1", Priority: "high"},
						{ID: "FR-002", Description: "Requirement 2", Priority: "medium"},
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
					},
				},
			},
			wantErr: false,
			validateChanges: func(t *testing.T, changes *generate.FCSChanges) {
				assert.True(t, changes.HasChanges)
				assert.Empty(t, changes.AddedRequirements)
				assert.Empty(t, changes.ModifiedRequirements)
				assert.Len(t, changes.DeletedRequirements, 1)
				assert.Equal(t, "FR-002", changes.DeletedRequirements[0])
			},
		},
		{
			name: "added package",
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
			wantErr: false,
			validateChanges: func(t *testing.T, changes *generate.FCSChanges) {
				assert.True(t, changes.HasChanges)
				assert.Len(t, changes.AddedPackages, 1)
				assert.Equal(t, "util", changes.AddedPackages[0].Name)
				assert.Empty(t, changes.ModifiedPackages)
				assert.Empty(t, changes.DeletedPackages)
			},
		},
		{
			name: "modified package",
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
						{Name: "main", Path: "cmd/main", Purpose: "Modified main package"},
					},
				},
			},
			wantErr: false,
			validateChanges: func(t *testing.T, changes *generate.FCSChanges) {
				assert.True(t, changes.HasChanges)
				assert.Empty(t, changes.AddedPackages)
				assert.Len(t, changes.ModifiedPackages, 1)
				assert.Equal(t, "main", changes.ModifiedPackages[0].Name)
				assert.Empty(t, changes.DeletedPackages)
			},
		},
		{
			name: "deleted package",
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
						{Name: "util", Path: "internal/util", Purpose: "Utility package"},
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
					},
				},
			},
			wantErr: false,
			validateChanges: func(t *testing.T, changes *generate.FCSChanges) {
				assert.True(t, changes.HasChanges)
				assert.Empty(t, changes.AddedPackages)
				assert.Empty(t, changes.ModifiedPackages)
				assert.Len(t, changes.DeletedPackages, 1)
				assert.Equal(t, "util", changes.DeletedPackages[0])
			},
		},
		{
			name: "multiple changes",
			oldFCS: &models.FinalClarifiedSpecification{
				ID:      "test-1",
				Version: "1.0",
				Requirements: models.Requirements{
					Functional: []models.FunctionalRequirement{
						{ID: "FR-001", Description: "Requirement 1", Priority: "high"},
						{ID: "FR-002", Description: "Requirement 2", Priority: "medium"},
					},
				},
				Architecture: models.Architecture{
					Packages: []models.Package{
						{Name: "main", Path: "cmd/main", Purpose: "Main package"},
						{Name: "old", Path: "internal/old", Purpose: "Old package"},
					},
				},
			},
			newFCS: &models.FinalClarifiedSpecification{
				ID:      "test-1",
				Version: "1.1",
				Requirements: models.Requirements{
					Functional: []models.FunctionalRequirement{
						{ID: "FR-001", Description: "Modified requirement 1", Priority: "high"},
						{ID: "FR-003", Description: "New requirement 3", Priority: "low"},
					},
				},
				Architecture: models.Architecture{
					Packages: []models.Package{
						{Name: "main", Path: "cmd/main", Purpose: "Modified main package"},
						{Name: "new", Path: "internal/new", Purpose: "New package"},
					},
				},
			},
			wantErr: false,
			validateChanges: func(t *testing.T, changes *generate.FCSChanges) {
				assert.True(t, changes.HasChanges)
				// Requirements
				assert.Len(t, changes.AddedRequirements, 1)
				assert.Equal(t, "FR-003", changes.AddedRequirements[0].ID)
				assert.Len(t, changes.ModifiedRequirements, 1)
				assert.Equal(t, "FR-001", changes.ModifiedRequirements[0].ID)
				assert.Len(t, changes.DeletedRequirements, 1)
				assert.Equal(t, "FR-002", changes.DeletedRequirements[0])
				// Packages
				assert.Len(t, changes.AddedPackages, 1)
				assert.Equal(t, "new", changes.AddedPackages[0].Name)
				assert.Len(t, changes.ModifiedPackages, 1)
				assert.Equal(t, "main", changes.ModifiedPackages[0].Name)
				assert.Len(t, changes.DeletedPackages, 1)
				assert.Equal(t, "old", changes.DeletedPackages[0])
			},
		},
		{
			name: "non-functional requirement changes",
			oldFCS: &models.FinalClarifiedSpecification{
				ID:      "test-1",
				Version: "1.0",
				Requirements: models.Requirements{
					Functional: []models.FunctionalRequirement{
						{ID: "FR-001", Description: "Requirement 1", Priority: "high"},
					},
					NonFunctional: []models.NonFunctionalRequirement{
						{ID: "NFR-001", Description: "Performance", Type: "performance"},
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
					NonFunctional: []models.NonFunctionalRequirement{
						{ID: "NFR-001", Description: "Modified performance", Type: "performance"},
					},
				},
				Architecture: models.Architecture{
					Packages: []models.Package{
						{Name: "main", Path: "cmd/main", Purpose: "Main package"},
					},
				},
			},
			wantErr: false,
			validateChanges: func(t *testing.T, changes *generate.FCSChanges) {
				assert.True(t, changes.HasChanges)
				assert.Len(t, changes.ModifiedNonFunctionalRequirements, 1)
				assert.Equal(t, "NFR-001", changes.ModifiedNonFunctionalRequirements[0].ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := generate.NewChangeDetector()
			changes, err := detector.DetectChanges(tt.oldFCS, tt.newFCS)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, changes)
			} else {
				require.NoError(t, err)
				require.NotNil(t, changes)
				if tt.validateChanges != nil {
					tt.validateChanges(t, changes)
				}
			}
		})
	}
}

func TestChangeDetector_IdentifyAffectedPackages(t *testing.T) {
	tests := []struct {
		name             string
		changes          *generate.FCSChanges
		architecture     *models.Architecture
		wantErr          bool
		validatePackages func(t *testing.T, packages []string)
	}{
		{
			name: "no changes",
			changes: &generate.FCSChanges{
				HasChanges: false,
			},
			architecture: &models.Architecture{
				Packages: []models.Package{
					{Name: "main", Path: "cmd/main"},
				},
			},
			wantErr: false,
			validatePackages: func(t *testing.T, packages []string) {
				assert.Empty(t, packages)
			},
		},
		{
			name: "added package affects only itself",
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
			wantErr: false,
			validatePackages: func(t *testing.T, packages []string) {
				assert.Len(t, packages, 1)
				assert.Contains(t, packages, "new")
			},
		},
		{
			name: "modified package affects dependents",
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
			wantErr: false,
			validatePackages: func(t *testing.T, packages []string) {
				assert.Len(t, packages, 2)
				assert.Contains(t, packages, "lib")
				assert.Contains(t, packages, "main")
			},
		},
		{
			name: "deleted package affects itself and dependents",
			changes: &generate.FCSChanges{
				HasChanges:      true,
				DeletedPackages: []string{"lib"},
			},
			architecture: &models.Architecture{
				Packages: []models.Package{
					{Name: "main", Path: "cmd/main", Dependencies: []string{"lib"}},
					{Name: "other", Path: "cmd/other", Dependencies: []string{}},
				},
			},
			wantErr: false,
			validatePackages: func(t *testing.T, packages []string) {
				assert.Len(t, packages, 2)
				assert.Contains(t, packages, "lib")  // Deleted package itself
				assert.Contains(t, packages, "main") // Dependent
			},
		},
		{
			name: "transitive dependencies",
			changes: &generate.FCSChanges{
				HasChanges: true,
				ModifiedPackages: []models.Package{
					{Name: "base", Path: "internal/base"},
				},
			},
			architecture: &models.Architecture{
				Packages: []models.Package{
					{Name: "base", Path: "internal/base", Dependencies: []string{}},
					{Name: "mid", Path: "internal/mid", Dependencies: []string{"base"}},
					{Name: "top", Path: "cmd/top", Dependencies: []string{"mid"}},
					{Name: "other", Path: "cmd/other", Dependencies: []string{}},
				},
			},
			wantErr: false,
			validatePackages: func(t *testing.T, packages []string) {
				assert.Len(t, packages, 3)
				assert.Contains(t, packages, "base")
				assert.Contains(t, packages, "mid")
				assert.Contains(t, packages, "top")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := generate.NewChangeDetector()
			packages, err := detector.IdentifyAffectedPackages(tt.changes, tt.architecture)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, packages)
			} else {
				require.NoError(t, err)
				require.NotNil(t, packages)
				if tt.validatePackages != nil {
					tt.validatePackages(t, packages)
				}
			}
		})
	}
}
