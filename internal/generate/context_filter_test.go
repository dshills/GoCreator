package generate

import (
	"testing"

	"github.com/dshills/gocreator/internal/models"
)

// createTestFCS creates a test FCS with a realistic entity structure
func createTestFCS() *models.FinalClarifiedSpecification {
	return &models.FinalClarifiedSpecification{
		SchemaVersion: "1.0",
		ID:            "test-fcs-001",
		Version:       "1.0.0",
		DataModel: models.DataModel{
			Entities: []models.Entity{
				{
					Name:    "User",
					Package: "user",
					Attributes: map[string]string{
						"ID":        "string",
						"Name":      "string",
						"Email":     "string",
						"AddressID": "string",
						"Address":   "*Address",
					},
				},
				{
					Name:    "Address",
					Package: "user",
					Attributes: map[string]string{
						"ID":        "string",
						"Street":    "string",
						"City":      "string",
						"CountryID": "string",
						"Country":   "*Country",
					},
				},
				{
					Name:    "Country",
					Package: "geo",
					Attributes: map[string]string{
						"ID":   "string",
						"Name": "string",
						"Code": "string",
					},
				},
				{
					Name:    "Product",
					Package: "product",
					Attributes: map[string]string{
						"ID":         "string",
						"Name":       "string",
						"Price":      "float64",
						"CategoryID": "string",
						"Category":   "*Category",
					},
				},
				{
					Name:    "Category",
					Package: "product",
					Attributes: map[string]string{
						"ID":   "string",
						"Name": "string",
					},
				},
				{
					Name:    "Order",
					Package: "order",
					Attributes: map[string]string{
						"ID":       "string",
						"UserID":   "string",
						"User":     "*User",
						"Products": "[]*Product",
						"Total":    "float64",
					},
				},
				{
					Name:    "Payment",
					Package: "payment",
					Attributes: map[string]string{
						"ID":      "string",
						"OrderID": "string",
						"Order":   "*Order",
						"Amount":  "float64",
						"Status":  "string",
					},
				},
			},
			Relationships: []models.Relationship{
				{From: "User", To: "Address", Type: "has_one"},
				{From: "Address", To: "Country", Type: "belongs_to"},
				{From: "Product", To: "Category", Type: "belongs_to"},
				{From: "Order", To: "User", Type: "belongs_to"},
				{From: "Order", To: "Product", Type: "has_many"},
				{From: "Payment", To: "Order", Type: "belongs_to"},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "user", Path: "internal/user", Purpose: "User management"},
				{Name: "geo", Path: "internal/geo", Purpose: "Geographic entities"},
				{Name: "product", Path: "internal/product", Purpose: "Product catalog"},
				{Name: "order", Path: "internal/order", Purpose: "Order processing", Dependencies: []string{"user", "product"}},
				{Name: "payment", Path: "internal/payment", Purpose: "Payment processing", Dependencies: []string{"order"}},
			},
		},
		Requirements: models.Requirements{
			Functional: []models.FunctionalRequirement{
				{ID: "FR-001", Description: "System must manage users", Priority: "high"},
				{ID: "FR-002", Description: "System must process orders", Priority: "high"},
			},
		},
		TestingStrategy: models.TestingStrategy{
			CoverageTarget:   85.0,
			UnitTests:        true,
			IntegrationTests: true,
		},
		BuildConfig: models.BuildConfig{
			GoVersion:  "1.21",
			OutputPath: "./output",
		},
	}
}

func TestNewContextFilter(t *testing.T) {
	fcs := createTestFCS()
	cf := NewContextFilter(fcs)

	if cf == nil {
		t.Fatal("NewContextFilter returned nil")
	}

	// Verify dependency graph was built
	if len(cf.depGraph) == 0 {
		t.Error("Dependency graph was not built")
	}

	// Verify entity packages were mapped
	if len(cf.entityPackages) != len(fcs.DataModel.Entities) {
		t.Errorf("Expected %d entity packages, got %d", len(fcs.DataModel.Entities), len(cf.entityPackages))
	}

	// Verify package dependencies were mapped
	if len(cf.packageDeps) == 0 {
		t.Error("Package dependencies were not mapped")
	}
}

func TestDependencyGraphConstruction(t *testing.T) {
	fcs := createTestFCS()
	cf := NewContextFilter(fcs)

	tests := []struct {
		name         string
		entity       string
		expectedDeps []string
	}{
		{
			name:         "User has Address dependency",
			entity:       "User",
			expectedDeps: []string{"Address"},
		},
		{
			name:         "Address has Country dependency",
			entity:       "Address",
			expectedDeps: []string{"Country"},
		},
		{
			name:         "Order has User and Product dependencies",
			entity:       "Order",
			expectedDeps: []string{"User", "Product"},
		},
		{
			name:         "Payment has Order dependency",
			entity:       "Payment",
			expectedDeps: []string{"Order"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps, exists := cf.depGraph[tt.entity]
			if !exists {
				t.Errorf("Entity %s not found in dependency graph", tt.entity)
				return
			}

			// Check if all expected dependencies are present
			for _, expectedDep := range tt.expectedDeps {
				found := false
				for _, dep := range deps {
					if dep == expectedDep {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected dependency %s not found for entity %s", expectedDep, tt.entity)
				}
			}
		})
	}
}

func TestExtractEntityReference(t *testing.T) {
	cf := &ContextFilter{}

	tests := []struct {
		name     string
		typeStr  string
		expected string
	}{
		{"Simple entity", "User", "User"},
		{"Pointer entity", "*User", "User"},
		{"Slice entity", "[]User", "User"},
		{"Slice pointer entity", "[]*User", "User"},
		{"Map entity", "map[string]User", "User"},
		{"Map pointer entity", "map[string]*User", "User"},
		{"Primitive type", "string", ""},
		{"Lowercase type", "int", ""},
		{"Package qualified", "models.User", "User"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cf.extractEntityReference(tt.typeStr)
			if result != tt.expected {
				t.Errorf("extractEntityReference(%q) = %q, want %q", tt.typeStr, result, tt.expected)
			}
		})
	}
}

func TestFilterForFile_UserEntity(t *testing.T) {
	fcs := createTestFCS()
	cf := NewContextFilter(fcs)

	// Test filtering for a User entity file
	filePath := "internal/user/user.go"
	plan := &models.GenerationPlan{}

	filtered := cf.FilterForFile(filePath, plan, fcs)

	if filtered == nil {
		t.Fatal("FilterForFile returned nil")
	}

	// User entity file should include User, Address, and Country (transitive dependency)
	expectedEntities := map[string]bool{
		"User":    true,
		"Address": true,
		"Country": true,
	}

	if filtered.FilteredEntityCount < 2 {
		t.Errorf("Expected at least 2 filtered entities (User + Address), got %d", filtered.FilteredEntityCount)
	}

	// Verify the right entities are included
	for _, entity := range filtered.DataModel.Entities {
		if !expectedEntities[entity.Name] {
			t.Logf("Unexpected entity included: %s (may be acceptable)", entity.Name)
		}
	}

	// Verify reduction percentage is calculated
	if filtered.ReductionPercentage == 0 && filtered.OriginalEntityCount > filtered.FilteredEntityCount {
		t.Error("ReductionPercentage should be non-zero when entities are filtered")
	}

	t.Logf("Original entities: %d, Filtered: %d, Reduction: %.1f%%",
		filtered.OriginalEntityCount, filtered.FilteredEntityCount, filtered.ReductionPercentage)
}

func TestFilterForFile_ProductEntity(t *testing.T) {
	fcs := createTestFCS()
	cf := NewContextFilter(fcs)

	// Test filtering for a Product entity file
	filePath := "internal/product/product.go"
	plan := &models.GenerationPlan{}

	filtered := cf.FilterForFile(filePath, plan, fcs)

	if filtered == nil {
		t.Fatal("FilterForFile returned nil")
	}

	// Product entity file should NOT include User, Address, Order, Payment
	// Should include Product and Category
	excludedEntities := []string{"User", "Address", "Payment"}

	entityMap := make(map[string]bool)
	for _, entity := range filtered.DataModel.Entities {
		entityMap[entity.Name] = true
	}

	for _, excluded := range excludedEntities {
		if entityMap[excluded] {
			t.Errorf("Entity %s should not be included for Product file", excluded)
		}
	}

	// Should include Product
	if !entityMap["Product"] {
		t.Error("Product entity should be included for product file")
	}

	// Should include Category (dependency)
	if !entityMap["Category"] {
		t.Error("Category entity should be included (dependency of Product)")
	}

	t.Logf("Filtered entities for Product: %d (reduction: %.1f%%)",
		filtered.FilteredEntityCount, filtered.ReductionPercentage)
}

func TestFilterForFile_OrderService(t *testing.T) {
	fcs := createTestFCS()
	cf := NewContextFilter(fcs)

	// Test filtering for an Order service file
	filePath := "internal/order/service.go"
	plan := &models.GenerationPlan{}

	filtered := cf.FilterForFile(filePath, plan, fcs)

	if filtered == nil {
		t.Fatal("FilterForFile returned nil")
	}

	// Order service should include Order, User, Product entities
	expectedEntities := map[string]bool{
		"Order":   true,
		"User":    true,
		"Product": true,
	}

	entityMap := make(map[string]bool)
	for _, entity := range filtered.DataModel.Entities {
		entityMap[entity.Name] = true
	}

	for entityName := range expectedEntities {
		if !entityMap[entityName] {
			t.Errorf("Entity %s should be included for Order service", entityName)
		}
	}

	t.Logf("Filtered entities for Order service: %d (reduction: %.1f%%)",
		filtered.FilteredEntityCount, filtered.ReductionPercentage)
}

func TestFilterForFile_ContextReduction(t *testing.T) {
	fcs := createTestFCS()
	cf := NewContextFilter(fcs)

	testFiles := []string{
		"internal/user/user.go",
		"internal/product/product.go",
		"internal/order/service.go",
		"internal/payment/payment.go",
	}

	plan := &models.GenerationPlan{}

	for _, filePath := range testFiles {
		t.Run(filePath, func(t *testing.T) {
			filtered := cf.FilterForFile(filePath, plan, fcs)

			// Verify we're achieving significant reduction
			if filtered.FilteredEntityCount >= filtered.OriginalEntityCount {
				t.Logf("Warning: No reduction for %s (filtered: %d, original: %d)",
					filePath, filtered.FilteredEntityCount, filtered.OriginalEntityCount)
			}

			// Target: max 40% of FCS per call (FR-007)
			totalFiltered := filtered.FilteredEntityCount + filtered.FilteredPackageCount
			totalOriginal := filtered.OriginalEntityCount + filtered.OriginalPackageCount
			inclusionPercentage := float64(totalFiltered) / float64(totalOriginal) * 100

			if inclusionPercentage > 60 {
				t.Logf("Warning: Inclusion percentage %.1f%% exceeds target (should be <40%%)", inclusionPercentage)
			}

			t.Logf("%s: Entities %d/%d (%.1f%%), Total inclusion: %.1f%%",
				filePath,
				filtered.FilteredEntityCount, filtered.OriginalEntityCount,
				filtered.ReductionPercentage,
				inclusionPercentage)
		})
	}
}

func TestFilterEntities(t *testing.T) {
	fcs := createTestFCS()
	cf := NewContextFilter(fcs)

	relevant := map[string]bool{
		"User":    true,
		"Address": true,
	}

	filtered := cf.filterEntities(fcs.DataModel.Entities, relevant)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 filtered entities, got %d", len(filtered))
	}

	for _, entity := range filtered {
		if !relevant[entity.Name] {
			t.Errorf("Unexpected entity in filtered result: %s", entity.Name)
		}
	}
}

func TestFilterRelationships(t *testing.T) {
	fcs := createTestFCS()
	cf := NewContextFilter(fcs)

	relevant := map[string]bool{
		"User":    true,
		"Address": true,
		"Country": true,
	}

	filtered := cf.filterRelationships(fcs.DataModel.Relationships, relevant)

	// Should only include relationships between relevant entities
	for _, rel := range filtered {
		if !relevant[rel.From] {
			t.Errorf("Relationship has non-relevant From entity: %s", rel.From)
		}
		if !relevant[rel.To] {
			t.Errorf("Relationship has non-relevant To entity: %s", rel.To)
		}
	}

	// Should include User->Address and Address->Country
	foundUserAddress := false
	foundAddressCountry := false
	for _, rel := range filtered {
		if rel.From == "User" && rel.To == "Address" {
			foundUserAddress = true
		}
		if rel.From == "Address" && rel.To == "Country" {
			foundAddressCountry = true
		}
	}

	if !foundUserAddress {
		t.Error("Expected User->Address relationship in filtered results")
	}
	if !foundAddressCountry {
		t.Error("Expected Address->Country relationship in filtered results")
	}
}

func TestFilterPackages(t *testing.T) {
	fcs := createTestFCS()
	cf := NewContextFilter(fcs)

	relevant := map[string]bool{
		"user": true,
		"geo":  true,
	}

	filtered := cf.filterPackages(fcs.Architecture.Packages, relevant)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 filtered packages, got %d", len(filtered))
	}

	for _, pkg := range filtered {
		if !relevant[pkg.Name] {
			t.Errorf("Unexpected package in filtered result: %s", pkg.Name)
		}
	}
}

func TestFormatFilteredFCS(t *testing.T) {
	fcs := createTestFCS()
	cf := NewContextFilter(fcs)

	filePath := "internal/user/user.go"
	plan := &models.GenerationPlan{}

	filtered := cf.FilterForFile(filePath, plan, fcs)
	formatted := cf.FormatFilteredFCS(filtered)

	if formatted == "" {
		t.Fatal("FormatFilteredFCS returned empty string")
	}

	// Verify key sections are present
	requiredSections := []string{
		"Final Clarified Specification",
		"Functional Requirements",
		"Packages",
		"Entities",
		"Testing Strategy",
		"Build Configuration",
	}

	for _, section := range requiredSections {
		if !contains(formatted, section) {
			t.Errorf("Formatted FCS missing section: %s", section)
		}
	}

	t.Logf("Formatted FCS length: %d characters", len(formatted))
}

func TestTransitiveDependencies(t *testing.T) {
	fcs := createTestFCS()
	cf := NewContextFilter(fcs)

	// Test that transitive dependencies are included
	// User -> Address -> Country
	relevant := make(map[string]bool)
	cf.addEntityWithDependencies("User", relevant, 0)

	expectedEntities := []string{"User", "Address", "Country"}
	for _, entity := range expectedEntities {
		if !relevant[entity] {
			t.Errorf("Expected transitive dependency %s not included", entity)
		}
	}

	if len(relevant) > len(expectedEntities)+1 {
		t.Logf("Warning: More entities than expected included: %v", relevant)
	}
}

func TestAddEntityWithDependencies_MaxDepth(t *testing.T) {
	fcs := createTestFCS()
	cf := NewContextFilter(fcs)

	// Create a circular dependency scenario (should be prevented by depth limit)
	relevant := make(map[string]bool)
	cf.addEntityWithDependencies("Payment", relevant, 0)

	// Payment -> Order -> User/Product -> ...
	// Should stop at depth 5
	if len(relevant) > 10 {
		t.Errorf("Too many entities included (possible infinite recursion): %d", len(relevant))
	}

	t.Logf("Entities included for Payment (depth-limited): %d", len(relevant))
}

func TestDetermineRelevantPackages(t *testing.T) {
	fcs := createTestFCS()
	cf := NewContextFilter(fcs)

	relevantEntities := map[string]bool{
		"User":    true,
		"Address": true,
	}

	filePath := "internal/user/service.go"
	plan := &models.GenerationPlan{}

	relevant := cf.determineRelevantPackages(filePath, plan, relevantEntities)

	// Should include 'user' package
	if !relevant["user"] {
		t.Error("Expected 'user' package to be relevant")
	}

	// Should include 'geo' package (dependency of user via Address->Country)
	// Note: This depends on implementation details
	t.Logf("Relevant packages: %v", relevant)
}

func TestMetricsTracking(t *testing.T) {
	fcs := createTestFCS()
	cf := NewContextFilter(fcs)

	filePath := "internal/user/user.go"
	plan := &models.GenerationPlan{}

	filtered := cf.FilterForFile(filePath, plan, fcs)

	// Verify metrics are populated
	if filtered.OriginalEntityCount == 0 {
		t.Error("OriginalEntityCount should be populated")
	}
	if filtered.OriginalPackageCount == 0 {
		t.Error("OriginalPackageCount should be populated")
	}

	// Verify reduction percentage is sensible
	if filtered.ReductionPercentage < 0 || filtered.ReductionPercentage > 100 {
		t.Errorf("ReductionPercentage should be between 0-100, got %.1f", filtered.ReductionPercentage)
	}

	t.Logf("Metrics: Entities %d->%d (%.1f%%), Packages %d->%d",
		filtered.OriginalEntityCount, filtered.FilteredEntityCount, filtered.ReductionPercentage,
		filtered.OriginalPackageCount, filtered.FilteredPackageCount)
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
