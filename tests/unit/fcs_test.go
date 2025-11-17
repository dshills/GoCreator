package unit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFCS_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name string
		fcs  *models.FinalClarifiedSpecification
	}{
		{
			name: "complete FCS",
			fcs: &models.FinalClarifiedSpecification{
				SchemaVersion:  "1.0",
				ID:             uuid.New().String(),
				Version:        "1.0",
				OriginalSpecID: uuid.New().String(),
				Metadata: models.FCSMetadata{
					CreatedAt:    time.Now().UTC(),
					OriginalSpec: "test spec content",
					Clarifications: []models.AppliedClarification{
						{
							QuestionID: "q1",
							Answer:     "JWT",
							AppliedTo:  "architecture.authentication",
						},
					},
					Hash: "abc123",
				},
				Requirements: models.Requirements{
					Functional: []models.FunctionalRequirement{
						{
							ID:          "FR-001",
							Description: "User authentication",
							Priority:    "critical",
							Category:    "security",
						},
					},
					NonFunctional: []models.NonFunctionalRequirement{
						{
							ID:          "NFR-001",
							Description: "Response time < 200ms",
							Type:        "performance",
							Threshold:   "200ms",
						},
					},
				},
				Architecture: models.Architecture{
					Packages: []models.Package{
						{
							Name:         "auth",
							Path:         "internal/auth",
							Purpose:      "Authentication and authorization",
							Dependencies: []string{"domain"},
						},
					},
					Dependencies: []models.Dependency{
						{
							Name:    "github.com/golang-jwt/jwt/v5",
							Version: "v5.0.0",
							Purpose: "JWT token handling",
						},
					},
					Patterns: []models.DesignPattern{
						{
							Name:        "Repository",
							Description: "Data access abstraction",
							AppliesTo:   []string{"auth", "domain"},
						},
					},
				},
				DataModel: models.DataModel{
					Entities: []models.Entity{
						{
							Name:       "User",
							Package:    "domain",
							Attributes: map[string]string{"id": "string", "email": "string"},
						},
					},
					Relationships: []models.Relationship{
						{
							From:        "User",
							To:          "Role",
							Type:        "many-to-many",
							Description: "Users have multiple roles",
						},
					},
				},
				APIContracts: []models.APIContract{
					{
						Endpoint:    "/api/login",
						Method:      "POST",
						Description: "User login",
						Request: models.ContractSchema{
							Fields: map[string]string{"email": "string", "password": "string"},
						},
						Response: models.ContractSchema{
							Fields: map[string]string{"token": "string"},
						},
					},
				},
				TestingStrategy: models.TestingStrategy{
					CoverageTarget:   85.0,
					UnitTests:        true,
					IntegrationTests: true,
					Frameworks:       []string{"testify"},
				},
				BuildConfig: models.BuildConfig{
					GoVersion:  "1.23",
					OutputPath: "./bin/app",
					BuildFlags: []string{"-trimpath"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.fcs)
			require.NoError(t, err)

			var unmarshaled models.FinalClarifiedSpecification
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, tt.fcs.ID, unmarshaled.ID)
			assert.Equal(t, tt.fcs.Version, unmarshaled.Version)
			assert.Equal(t, len(tt.fcs.Requirements.Functional), len(unmarshaled.Requirements.Functional))
			assert.Equal(t, len(tt.fcs.Architecture.Packages), len(unmarshaled.Architecture.Packages))
		})
	}
}

func TestFCS_Validate(t *testing.T) {
	tests := []struct {
		name    string
		fcs     *models.FinalClarifiedSpecification
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid FCS with matching hash",
			fcs: func() *models.FinalClarifiedSpecification {
				fcs := &models.FinalClarifiedSpecification{
					ID:             uuid.New().String(),
					Version:        "1.0",
					OriginalSpecID: uuid.New().String(),
					Requirements: models.Requirements{
						Functional: []models.FunctionalRequirement{
							{ID: "FR-001", Description: "Test", Priority: "high"},
						},
					},
					Architecture: models.Architecture{
						Packages: []models.Package{
							{Name: "main", Path: "cmd/main"},
						},
					},
				}
				// Compute correct hash
				content, _ := json.Marshal(fcs)
				hash := sha256.Sum256(content)
				fcs.Metadata.Hash = hex.EncodeToString(hash[:])
				return fcs
			}(),
			wantErr: false,
		},
		{
			name: "invalid - cyclic package dependencies",
			fcs: &models.FinalClarifiedSpecification{
				ID:             uuid.New().String(),
				Version:        "1.0",
				OriginalSpecID: uuid.New().String(),
				Architecture: models.Architecture{
					Packages: []models.Package{
						{Name: "a", Path: "a", Dependencies: []string{"b"}},
						{Name: "b", Path: "b", Dependencies: []string{"c"}},
						{Name: "c", Path: "c", Dependencies: []string{"a"}},
					},
				},
			},
			wantErr: true,
			errMsg:  "cyclic dependency",
		},
		{
			name: "invalid - hash mismatch",
			fcs: &models.FinalClarifiedSpecification{
				ID:             uuid.New().String(),
				Version:        "1.0",
				OriginalSpecID: uuid.New().String(),
				Metadata: models.FCSMetadata{
					Hash: "invalid-hash",
				},
				Requirements: models.Requirements{
					Functional: []models.FunctionalRequirement{
						{ID: "FR-001", Description: "Test"},
					},
				},
			},
			wantErr: true,
			errMsg:  "hash mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fcs.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFCS_ComputeHash(t *testing.T) {
	fcs := &models.FinalClarifiedSpecification{
		ID:             uuid.New().String(),
		Version:        "1.0",
		OriginalSpecID: uuid.New().String(),
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{ID: "FR-001", Description: "Test requirement"},
			},
		},
	}

	hash1, err := fcs.ComputeHash()
	require.NoError(t, err)
	assert.NotEmpty(t, hash1)

	// Same FCS should produce same hash
	hash2, err := fcs.ComputeHash()
	require.NoError(t, err)
	assert.Equal(t, hash1, hash2)

	// Modifying FCS should change hash
	fcs.Requirements.Functional[0].Description = "Modified"
	hash3, err := fcs.ComputeHash()
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash3)
}

func TestFCS_DetectCyclicDependencies(t *testing.T) {
	tests := []struct {
		name      string
		packages  []models.Package
		hasCycles bool
	}{
		{
			name: "no cycles - linear dependency",
			packages: []models.Package{
				{Name: "a", Dependencies: []string{"b"}},
				{Name: "b", Dependencies: []string{"c"}},
				{Name: "c", Dependencies: []string{}},
			},
			hasCycles: false,
		},
		{
			name: "no cycles - tree structure",
			packages: []models.Package{
				{Name: "a", Dependencies: []string{"b", "c"}},
				{Name: "b", Dependencies: []string{"d"}},
				{Name: "c", Dependencies: []string{"d"}},
				{Name: "d", Dependencies: []string{}},
			},
			hasCycles: false,
		},
		{
			name: "simple cycle",
			packages: []models.Package{
				{Name: "a", Dependencies: []string{"b"}},
				{Name: "b", Dependencies: []string{"a"}},
			},
			hasCycles: true,
		},
		{
			name: "three-node cycle",
			packages: []models.Package{
				{Name: "a", Dependencies: []string{"b"}},
				{Name: "b", Dependencies: []string{"c"}},
				{Name: "c", Dependencies: []string{"a"}},
			},
			hasCycles: true,
		},
		{
			name: "self-cycle",
			packages: []models.Package{
				{Name: "a", Dependencies: []string{"a"}},
			},
			hasCycles: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fcs := &models.FinalClarifiedSpecification{
				Architecture: models.Architecture{
					Packages: tt.packages,
				},
			}

			hasCycles := fcs.HasCyclicDependencies()
			assert.Equal(t, tt.hasCycles, hasCycles)
		})
	}
}
