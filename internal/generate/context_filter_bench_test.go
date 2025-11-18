package generate

import (
	"testing"

	"github.com/dshills/gocreator/internal/models"
)

// BenchmarkContextFilter benchmarks the context filtering performance
func BenchmarkContextFilter(b *testing.B) {
	fcs := createTestFCS()
	plan := &models.GenerationPlan{}

	b.Run("NewContextFilter", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewContextFilter(fcs)
		}
	})

	cf := NewContextFilter(fcs)

	b.Run("FilterForFile_User", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = cf.FilterForFile("internal/user/user.go", plan, fcs)
		}
	})

	b.Run("FilterForFile_Product", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = cf.FilterForFile("internal/product/product.go", plan, fcs)
		}
	})

	b.Run("FilterForFile_Order", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = cf.FilterForFile("internal/order/service.go", plan, fcs)
		}
	})
}

// BenchmarkFormatFilteredFCS benchmarks the formatting performance
func BenchmarkFormatFilteredFCS(b *testing.B) {
	fcs := createTestFCS()
	cf := NewContextFilter(fcs)
	plan := &models.GenerationPlan{}

	filtered := cf.FilterForFile("internal/user/user.go", plan, fcs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cf.FormatFilteredFCS(filtered)
	}
}

// BenchmarkDependencyAnalysis benchmarks dependency graph operations
func BenchmarkDependencyAnalysis(b *testing.B) {
	fcs := createTestFCS()
	cf := NewContextFilter(fcs)

	b.Run("AddEntityWithDependencies", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			relevant := make(map[string]bool)
			cf.addEntityWithDependencies("Payment", relevant, 0)
		}
	})

	b.Run("ExtractEntityReference", func(b *testing.B) {
		types := []string{
			"User",
			"*User",
			"[]User",
			"[]*User",
			"map[string]User",
			"map[string]*User",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, t := range types {
				_ = cf.extractEntityReference(t)
			}
		}
	})
}

// BenchmarkContextReduction measures the actual context size reduction
func BenchmarkContextReduction(b *testing.B) {
	fcs := createLargeFCS() // Create a larger FCS for more realistic benchmarks
	cf := NewContextFilter(fcs)
	plan := &models.GenerationPlan{}

	files := []string{
		"internal/user/user.go",
		"internal/product/product.go",
		"internal/order/service.go",
		"internal/payment/payment.go",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, file := range files {
			filtered := cf.FilterForFile(file, plan, fcs)
			_ = cf.FormatFilteredFCS(filtered)
		}
	}

	// Report metrics
	if b.N > 0 {
		filtered := cf.FilterForFile(files[0], plan, fcs)
		b.ReportMetric(filtered.ReductionPercentage, "reduction_%")
		b.ReportMetric(float64(filtered.FilteredEntityCount), "filtered_entities")
		b.ReportMetric(float64(filtered.OriginalEntityCount), "original_entities")
	}
}

// createLargeFCS creates a larger test FCS for benchmarking
func createLargeFCS() *models.FinalClarifiedSpecification {
	fcs := createTestFCS()

	// Add more entities to simulate a realistic project
	additionalEntities := []models.Entity{
		{Name: "Invoice", Package: "billing", Attributes: map[string]string{"ID": "string", "OrderID": "string", "Order": "*Order"}},
		{Name: "Shipment", Package: "shipping", Attributes: map[string]string{"ID": "string", "OrderID": "string", "Order": "*Order"}},
		{Name: "Notification", Package: "notify", Attributes: map[string]string{"ID": "string", "UserID": "string", "User": "*User"}},
		{Name: "Audit", Package: "audit", Attributes: map[string]string{"ID": "string", "EntityID": "string"}},
		{Name: "Permission", Package: "auth", Attributes: map[string]string{"ID": "string", "UserID": "string", "User": "*User"}},
		{Name: "Role", Package: "auth", Attributes: map[string]string{"ID": "string", "Name": "string"}},
		{Name: "Session", Package: "auth", Attributes: map[string]string{"ID": "string", "UserID": "string", "User": "*User"}},
		{Name: "Settings", Package: "config", Attributes: map[string]string{"ID": "string", "UserID": "string", "User": "*User"}},
		{Name: "Review", Package: "product", Attributes: map[string]string{"ID": "string", "ProductID": "string", "Product": "*Product", "UserID": "string", "User": "*User"}},
		{Name: "Wishlist", Package: "product", Attributes: map[string]string{"ID": "string", "UserID": "string", "User": "*User", "Products": "[]*Product"}},
	}

	fcs.DataModel.Entities = append(fcs.DataModel.Entities, additionalEntities...)

	// Add corresponding relationships
	additionalRels := []models.Relationship{
		{From: "Invoice", To: "Order", Type: "belongs_to"},
		{From: "Shipment", To: "Order", Type: "belongs_to"},
		{From: "Notification", To: "User", Type: "belongs_to"},
		{From: "Permission", To: "User", Type: "belongs_to"},
		{From: "Session", To: "User", Type: "belongs_to"},
		{From: "Settings", To: "User", Type: "belongs_to"},
		{From: "Review", To: "Product", Type: "belongs_to"},
		{From: "Review", To: "User", Type: "belongs_to"},
		{From: "Wishlist", To: "User", Type: "belongs_to"},
		{From: "Wishlist", To: "Product", Type: "has_many"},
	}

	fcs.DataModel.Relationships = append(fcs.DataModel.Relationships, additionalRels...)

	return fcs
}
