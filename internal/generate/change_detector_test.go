package generate

import (
	"testing"

	"github.com/dshills/gocreator/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangeDetector_DetectEntityChanges(t *testing.T) {
	detector := NewChangeDetector()

	tests := []struct {
		name         string
		oldFCS       *models.FinalClarifiedSpecification
		newFCS       *models.FinalClarifiedSpecification
		wantAdded    []string
		wantModified []string
		wantDeleted  []string
	}{
		{
			name:   "nil old FCS (first generation)",
			oldFCS: nil,
			newFCS: &models.FinalClarifiedSpecification{
				DataModel: models.DataModel{
					Entities: []models.Entity{
						{Name: "User", Package: "models"},
						{Name: "Product", Package: "models"},
					},
				},
			},
			wantAdded:    []string{"User", "Product"},
			wantModified: []string{},
			wantDeleted:  []string{},
		},
		{
			name: "entity added",
			oldFCS: &models.FinalClarifiedSpecification{
				DataModel: models.DataModel{
					Entities: []models.Entity{
						{Name: "User", Package: "models"},
					},
				},
			},
			newFCS: &models.FinalClarifiedSpecification{
				DataModel: models.DataModel{
					Entities: []models.Entity{
						{Name: "User", Package: "models"},
						{Name: "Product", Package: "models"},
					},
				},
			},
			wantAdded:    []string{"Product"},
			wantModified: []string{},
			wantDeleted:  []string{},
		},
		{
			name: "entity modified (attribute added)",
			oldFCS: &models.FinalClarifiedSpecification{
				DataModel: models.DataModel{
					Entities: []models.Entity{
						{
							Name:    "User",
							Package: "models",
							Attributes: map[string]string{
								"ID":   "string",
								"Name": "string",
							},
						},
					},
				},
			},
			newFCS: &models.FinalClarifiedSpecification{
				DataModel: models.DataModel{
					Entities: []models.Entity{
						{
							Name:    "User",
							Package: "models",
							Attributes: map[string]string{
								"ID":    "string",
								"Name":  "string",
								"Email": "string",
							},
						},
					},
				},
			},
			wantAdded:    []string{},
			wantModified: []string{"User"},
			wantDeleted:  []string{},
		},
		{
			name: "entity modified (attribute type changed)",
			oldFCS: &models.FinalClarifiedSpecification{
				DataModel: models.DataModel{
					Entities: []models.Entity{
						{
							Name:    "User",
							Package: "models",
							Attributes: map[string]string{
								"ID":   "string",
								"Name": "string",
							},
						},
					},
				},
			},
			newFCS: &models.FinalClarifiedSpecification{
				DataModel: models.DataModel{
					Entities: []models.Entity{
						{
							Name:    "User",
							Package: "models",
							Attributes: map[string]string{
								"ID":   "int64",
								"Name": "string",
							},
						},
					},
				},
			},
			wantAdded:    []string{},
			wantModified: []string{"User"},
			wantDeleted:  []string{},
		},
		{
			name: "entity modified (package changed)",
			oldFCS: &models.FinalClarifiedSpecification{
				DataModel: models.DataModel{
					Entities: []models.Entity{
						{Name: "User", Package: "models"},
					},
				},
			},
			newFCS: &models.FinalClarifiedSpecification{
				DataModel: models.DataModel{
					Entities: []models.Entity{
						{Name: "User", Package: "domain"},
					},
				},
			},
			wantAdded:    []string{},
			wantModified: []string{"User"},
			wantDeleted:  []string{},
		},
		{
			name: "entity deleted",
			oldFCS: &models.FinalClarifiedSpecification{
				DataModel: models.DataModel{
					Entities: []models.Entity{
						{Name: "User", Package: "models"},
						{Name: "Product", Package: "models"},
					},
				},
			},
			newFCS: &models.FinalClarifiedSpecification{
				DataModel: models.DataModel{
					Entities: []models.Entity{
						{Name: "User", Package: "models"},
					},
				},
			},
			wantAdded:    []string{},
			wantModified: []string{},
			wantDeleted:  []string{"Product"},
		},
		{
			name: "multiple changes",
			oldFCS: &models.FinalClarifiedSpecification{
				DataModel: models.DataModel{
					Entities: []models.Entity{
						{Name: "User", Package: "models"},
						{Name: "Product", Package: "models", Attributes: map[string]string{"ID": "string"}},
					},
				},
			},
			newFCS: &models.FinalClarifiedSpecification{
				DataModel: models.DataModel{
					Entities: []models.Entity{
						{Name: "User", Package: "domain"},
						{Name: "Order", Package: "models"},
					},
				},
			},
			wantAdded:    []string{"Order"},
			wantModified: []string{"User"},
			wantDeleted:  []string{"Product"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes, err := detector.DetectChanges(tt.oldFCS, tt.newFCS)
			require.NoError(t, err)

			assert.ElementsMatch(t, tt.wantAdded, changes.AddedEntities)
			assert.ElementsMatch(t, tt.wantModified, changes.ModifiedEntities)
			assert.ElementsMatch(t, tt.wantDeleted, changes.DeletedEntities)

			// Verify HasChanges flag
			if len(tt.wantAdded) > 0 || len(tt.wantModified) > 0 || len(tt.wantDeleted) > 0 {
				assert.True(t, changes.HasChanges)
			}
		})
	}
}

func TestChangeDetector_DetectAPIChanges(t *testing.T) {
	detector := NewChangeDetector()

	tests := []struct {
		name         string
		oldFCS       *models.FinalClarifiedSpecification
		newFCS       *models.FinalClarifiedSpecification
		wantAdded    []string
		wantModified []string
		wantDeleted  []string
	}{
		{
			name: "API contract added",
			oldFCS: &models.FinalClarifiedSpecification{
				APIContracts: []models.APIContract{
					{Method: "GET", Endpoint: "/users", Description: "Get users"},
				},
			},
			newFCS: &models.FinalClarifiedSpecification{
				APIContracts: []models.APIContract{
					{Method: "GET", Endpoint: "/users", Description: "Get users"},
					{Method: "POST", Endpoint: "/users", Description: "Create user"},
				},
			},
			wantAdded:    []string{"POST /users"},
			wantModified: []string{},
			wantDeleted:  []string{},
		},
		{
			name: "API contract modified",
			oldFCS: &models.FinalClarifiedSpecification{
				APIContracts: []models.APIContract{
					{Method: "GET", Endpoint: "/users", Description: "Get users"},
				},
			},
			newFCS: &models.FinalClarifiedSpecification{
				APIContracts: []models.APIContract{
					{Method: "GET", Endpoint: "/users", Description: "List all users"},
				},
			},
			wantAdded:    []string{},
			wantModified: []string{"GET /users"},
			wantDeleted:  []string{},
		},
		{
			name: "API contract deleted",
			oldFCS: &models.FinalClarifiedSpecification{
				APIContracts: []models.APIContract{
					{Method: "GET", Endpoint: "/users", Description: "Get users"},
					{Method: "POST", Endpoint: "/users", Description: "Create user"},
					{Method: "DELETE", Endpoint: "/users/{id}", Description: "Delete user"},
				},
			},
			newFCS: &models.FinalClarifiedSpecification{
				APIContracts: []models.APIContract{
					{Method: "GET", Endpoint: "/users", Description: "Get users"},
				},
			},
			wantAdded:    []string{},
			wantModified: []string{},
			wantDeleted:  []string{"POST /users", "DELETE /users/{id}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes, err := detector.DetectChanges(tt.oldFCS, tt.newFCS)
			require.NoError(t, err)

			assert.ElementsMatch(t, tt.wantAdded, changes.AddedAPIContracts)
			assert.ElementsMatch(t, tt.wantModified, changes.ModifiedAPIContracts)
			assert.ElementsMatch(t, tt.wantDeleted, changes.DeletedAPIContracts)
		})
	}
}

func TestChangeDetector_ArchitectureChanged(t *testing.T) {
	detector := NewChangeDetector()

	tests := []struct {
		name        string
		oldFCS      *models.FinalClarifiedSpecification
		newFCS      *models.FinalClarifiedSpecification
		wantChanged bool
	}{
		{
			name: "architecture unchanged",
			oldFCS: &models.FinalClarifiedSpecification{
				Architecture: models.Architecture{
					Packages: []models.Package{
						{Name: "models", Path: "internal/models"},
					},
				},
			},
			newFCS: &models.FinalClarifiedSpecification{
				Architecture: models.Architecture{
					Packages: []models.Package{
						{Name: "models", Path: "internal/models"},
					},
				},
			},
			wantChanged: false,
		},
		{
			name: "architecture changed",
			oldFCS: &models.FinalClarifiedSpecification{
				Architecture: models.Architecture{
					Packages: []models.Package{
						{Name: "models", Path: "internal/models"},
					},
				},
			},
			newFCS: &models.FinalClarifiedSpecification{
				Architecture: models.Architecture{
					Packages: []models.Package{
						{Name: "models", Path: "internal/models"},
						{Name: "services", Path: "internal/services"},
					},
				},
			},
			wantChanged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes, err := detector.DetectChanges(tt.oldFCS, tt.newFCS)
			require.NoError(t, err)

			assert.Equal(t, tt.wantChanged, changes.ArchitectureChanged)
		})
	}
}

func TestChangeDetector_BuildConfigChanged(t *testing.T) {
	detector := NewChangeDetector()

	tests := []struct {
		name        string
		oldFCS      *models.FinalClarifiedSpecification
		newFCS      *models.FinalClarifiedSpecification
		wantChanged bool
	}{
		{
			name: "build config unchanged",
			oldFCS: &models.FinalClarifiedSpecification{
				BuildConfig: models.BuildConfig{
					GoVersion: "1.21",
				},
			},
			newFCS: &models.FinalClarifiedSpecification{
				BuildConfig: models.BuildConfig{
					GoVersion: "1.21",
				},
			},
			wantChanged: false,
		},
		{
			name: "build config changed",
			oldFCS: &models.FinalClarifiedSpecification{
				BuildConfig: models.BuildConfig{
					GoVersion: "1.21",
				},
			},
			newFCS: &models.FinalClarifiedSpecification{
				BuildConfig: models.BuildConfig{
					GoVersion: "1.22",
				},
			},
			wantChanged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes, err := detector.DetectChanges(tt.oldFCS, tt.newFCS)
			require.NoError(t, err)

			assert.Equal(t, tt.wantChanged, changes.BuildConfigChanged)
		})
	}
}

func TestAffectedFilesCalculator_CalculateAffectedFiles(t *testing.T) {
	tests := []struct {
		name            string
		dependencyGraph map[string][]string
		changes         *FCSChanges
		allFiles        []string
		wantAffected    []string
	}{
		{
			name: "entity modified - affected files",
			dependencyGraph: map[string][]string{
				"models/user.go":           {"User"},
				"models/product.go":        {"Product"},
				"services/user_service.go": {"User"},
				"handlers/user_handler.go": {"User"},
			},
			changes: &FCSChanges{
				ModifiedEntities: []string{"User"},
			},
			allFiles: []string{
				"models/user.go",
				"models/product.go",
				"services/user_service.go",
				"handlers/user_handler.go",
			},
			wantAffected: []string{
				"models/user.go",
				"services/user_service.go",
				"handlers/user_handler.go",
			},
		},
		{
			name: "entity added - no existing dependencies",
			dependencyGraph: map[string][]string{
				"models/user.go": {"User"},
			},
			changes: &FCSChanges{
				AddedEntities: []string{"Product"},
			},
			allFiles: []string{
				"models/user.go",
				"models/product.go",
			},
			wantAffected: []string{},
		},
		{
			name: "entity deleted - find files by name",
			dependencyGraph: map[string][]string{
				"models/user.go":    {"User"},
				"models/product.go": {"Product"},
			},
			changes: &FCSChanges{
				DeletedEntities: []string{"Product"},
			},
			allFiles: []string{
				"models/user.go",
				"models/product.go",
			},
			wantAffected: []string{"models/product.go"},
		},
		{
			name: "architecture changed - regenerate all",
			dependencyGraph: map[string][]string{
				"models/user.go": {"User"},
			},
			changes: &FCSChanges{
				ArchitectureChanged: true,
			},
			allFiles: []string{
				"models/user.go",
				"models/product.go",
				"services/user_service.go",
			},
			wantAffected: []string{
				"models/user.go",
				"models/product.go",
				"services/user_service.go",
			},
		},
		{
			name: "build config changed - regenerate all",
			dependencyGraph: map[string][]string{
				"models/user.go": {"User"},
			},
			changes: &FCSChanges{
				BuildConfigChanged: true,
			},
			allFiles: []string{
				"models/user.go",
				"models/product.go",
			},
			wantAffected: []string{
				"models/user.go",
				"models/product.go",
			},
		},
		{
			name: "multiple entity changes",
			dependencyGraph: map[string][]string{
				"models/user.go":    {"User"},
				"models/product.go": {"Product"},
				"models/order.go":   {"Order", "User", "Product"},
			},
			changes: &FCSChanges{
				ModifiedEntities: []string{"User", "Product"},
			},
			allFiles: []string{
				"models/user.go",
				"models/product.go",
				"models/order.go",
			},
			wantAffected: []string{
				"models/user.go",
				"models/product.go",
				"models/order.go",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calculator := NewAffectedFilesCalculator(tt.dependencyGraph)
			affected := calculator.CalculateAffectedFiles(tt.changes, tt.allFiles)

			assert.ElementsMatch(t, tt.wantAffected, affected)
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "User", want: "user"},
		{input: "UserProfile", want: "user_profile"},
		{input: "HTTPRequest", want: "h_t_t_p_request"},
		{input: "APIKey", want: "a_p_i_key"},
		{input: "ID", want: "i_d"},
		{input: "", want: ""},
		{input: "lowercase", want: "lowercase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toSnakeCase(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
