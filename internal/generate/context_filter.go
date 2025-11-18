package generate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dshills/gocreator/internal/models"
	"github.com/rs/zerolog/log"
)

// ContextFilter filters FCS content to include only relevant portions for a specific generation task
type ContextFilter struct {
	// depGraph maps entity names to their dependencies
	depGraph map[string][]string
	// entityPackages maps entity names to their packages
	entityPackages map[string]string
	// packageDeps maps package names to other packages they depend on
	packageDeps map[string][]string
}

// FilteredFCS represents a filtered subset of the FCS for a specific task
type FilteredFCS struct {
	// Original FCS metadata (always included)
	SchemaVersion string
	ID            string
	Version       string

	// Filtered requirements (only relevant ones)
	Requirements models.Requirements

	// Filtered architecture (only relevant packages)
	Architecture models.Architecture

	// Filtered data model (only relevant entities and relationships)
	DataModel models.DataModel

	// API contracts (only relevant ones)
	APIContracts []models.APIContract

	// Testing and build config (always included)
	TestingStrategy models.TestingStrategy
	BuildConfig     models.BuildConfig

	// Metrics
	OriginalEntityCount  int
	FilteredEntityCount  int
	OriginalPackageCount int
	FilteredPackageCount int
	ReductionPercentage  float64
}

// NewContextFilter creates a new ContextFilter from an FCS
func NewContextFilter(fcs *models.FinalClarifiedSpecification) *ContextFilter {
	cf := &ContextFilter{
		depGraph:       make(map[string][]string),
		entityPackages: make(map[string]string),
		packageDeps:    make(map[string][]string),
	}

	// Build dependency graph from FCS
	cf.buildDependencyGraph(fcs)

	return cf
}

// buildDependencyGraph constructs the dependency graph from FCS
func (cf *ContextFilter) buildDependencyGraph(fcs *models.FinalClarifiedSpecification) {
	// Map entity names to packages
	for _, entity := range fcs.DataModel.Entities {
		cf.entityPackages[entity.Name] = entity.Package
	}

	// Build entity dependencies from relationships
	for _, rel := range fcs.DataModel.Relationships {
		// Add 'To' entity as a dependency of 'From' entity
		cf.depGraph[rel.From] = append(cf.depGraph[rel.From], rel.To)

		log.Debug().
			Str("from", rel.From).
			Str("to", rel.To).
			Str("type", rel.Type).
			Msg("Added entity relationship to dependency graph")
	}

	// Build entity dependencies from attributes (detect references to other entities)
	for _, entity := range fcs.DataModel.Entities {
		for attrName, attrType := range entity.Attributes {
			// Check if attribute type references another entity
			// Common patterns: "User", "*User", "[]User", "[]*User", "map[string]User"
			referencedEntity := cf.extractEntityReference(attrType)
			if referencedEntity != "" && referencedEntity != entity.Name {
				// Check if it's a known entity
				if _, exists := cf.entityPackages[referencedEntity]; exists {
					cf.depGraph[entity.Name] = append(cf.depGraph[entity.Name], referencedEntity)

					log.Debug().
						Str("entity", entity.Name).
						Str("attribute", attrName).
						Str("references", referencedEntity).
						Msg("Detected entity reference in attribute")
				}
			}
		}
	}

	// Build package dependencies from architecture
	for _, pkg := range fcs.Architecture.Packages {
		cf.packageDeps[pkg.Name] = pkg.Dependencies
	}

	log.Info().
		Int("entities", len(cf.entityPackages)).
		Int("entity_dependencies", len(cf.depGraph)).
		Int("packages", len(cf.packageDeps)).
		Msg("Built dependency graph from FCS")
}

// extractEntityReference extracts entity name from a type string
func (cf *ContextFilter) extractEntityReference(typeStr string) string {
	cleaned := strings.TrimSpace(typeStr)

	// Handle map types first: map[string]User -> User, map[string]*User -> *User
	if strings.HasPrefix(cleaned, "map[") {
		parts := strings.Split(cleaned, "]")
		if len(parts) > 1 {
			cleaned = strings.TrimSpace(parts[1])
		}
	}

	// Remove type modifiers in a loop to handle combinations like []*User
	for {
		original := cleaned
		cleaned = strings.TrimPrefix(cleaned, "*")
		cleaned = strings.TrimPrefix(cleaned, "[]")
		if cleaned == original {
			break // No more prefixes to remove
		}
	}

	// Handle package qualifiers: models.User -> User
	if strings.Contains(cleaned, ".") {
		parts := strings.Split(cleaned, ".")
		if len(parts) > 1 {
			cleaned = parts[len(parts)-1]
		}
	}

	// Check if it's a capitalized identifier (likely an entity)
	if len(cleaned) > 0 && cleaned[0] >= 'A' && cleaned[0] <= 'Z' {
		return cleaned
	}

	return ""
}

// FilterForFile creates a filtered FCS containing only relevant context for a specific file
func (cf *ContextFilter) FilterForFile(filePath string, plan *models.GenerationPlan, fcs *models.FinalClarifiedSpecification) *FilteredFCS {
	log.Debug().
		Str("file_path", filePath).
		Msg("Filtering FCS for file")

	// Determine what entities/packages this file needs
	relevantEntities := cf.determineRelevantEntities(filePath, plan, fcs)
	relevantPackages := cf.determineRelevantPackages(filePath, plan, relevantEntities)

	// Build filtered FCS
	filtered := &FilteredFCS{
		SchemaVersion:        fcs.SchemaVersion,
		ID:                   fcs.ID,
		Version:              fcs.Version,
		TestingStrategy:      fcs.TestingStrategy,
		BuildConfig:          fcs.BuildConfig,
		OriginalEntityCount:  len(fcs.DataModel.Entities),
		OriginalPackageCount: len(fcs.Architecture.Packages),
	}

	// Filter entities
	filtered.DataModel.Entities = cf.filterEntities(fcs.DataModel.Entities, relevantEntities)
	filtered.DataModel.Relationships = cf.filterRelationships(fcs.DataModel.Relationships, relevantEntities)
	filtered.FilteredEntityCount = len(filtered.DataModel.Entities)

	// Filter packages
	filtered.Architecture.Packages = cf.filterPackages(fcs.Architecture.Packages, relevantPackages)
	filtered.Architecture.Dependencies = fcs.Architecture.Dependencies // Include all external deps
	filtered.Architecture.Patterns = fcs.Architecture.Patterns         // Include all patterns
	filtered.FilteredPackageCount = len(filtered.Architecture.Packages)

	// Filter requirements (include all for now - could be optimized further)
	filtered.Requirements = fcs.Requirements

	// Filter API contracts (only those relevant to this file's package)
	filtered.APIContracts = cf.filterAPIContracts(fcs.APIContracts, filePath, relevantPackages)

	// Calculate reduction percentage
	totalOriginal := filtered.OriginalEntityCount + filtered.OriginalPackageCount
	totalFiltered := filtered.FilteredEntityCount + filtered.FilteredPackageCount
	if totalOriginal > 0 {
		filtered.ReductionPercentage = float64(totalOriginal-totalFiltered) / float64(totalOriginal) * 100
	}

	log.Info().
		Str("file_path", filePath).
		Int("original_entities", filtered.OriginalEntityCount).
		Int("filtered_entities", filtered.FilteredEntityCount).
		Int("original_packages", filtered.OriginalPackageCount).
		Int("filtered_packages", filtered.FilteredPackageCount).
		Float64("reduction_pct", filtered.ReductionPercentage).
		Msg("FCS filtered for file")

	return filtered
}

// determineRelevantEntities identifies which entities are relevant for a file
func (cf *ContextFilter) determineRelevantEntities(filePath string, plan *models.GenerationPlan, fcs *models.FinalClarifiedSpecification) map[string]bool {
	relevant := make(map[string]bool)

	// Determine file type and primary entity
	fileName := filepath.Base(filePath)
	fileDir := filepath.Dir(filePath)

	// Extract package name from path (e.g., "internal/user/service.go" -> "user")
	pathParts := strings.Split(fileDir, string(filepath.Separator))
	var packageName string
	if len(pathParts) > 0 {
		packageName = pathParts[len(pathParts)-1]
	}

	log.Debug().
		Str("file", fileName).
		Str("package", packageName).
		Msg("Determining relevant entities")

	// Find primary entity based on file name or package
	var primaryEntity string

	// Check if filename contains an entity name
	for _, entity := range fcs.DataModel.Entities {
		entityLower := strings.ToLower(entity.Name)
		fileNameLower := strings.ToLower(fileName)

		if strings.Contains(fileNameLower, entityLower) {
			primaryEntity = entity.Name
			log.Debug().
				Str("entity", entity.Name).
				Str("file", fileName).
				Msg("Matched entity from filename")
			break
		}
	}

	// If not found, check package match
	if primaryEntity == "" {
		for _, entity := range fcs.DataModel.Entities {
			if strings.EqualFold(entity.Package, packageName) {
				primaryEntity = entity.Name
				log.Debug().
					Str("entity", entity.Name).
					Str("package", packageName).
					Msg("Matched entity from package")
				break
			}
		}
	}

	// If we found a primary entity, include it and its dependencies
	if primaryEntity != "" {
		cf.addEntityWithDependencies(primaryEntity, relevant, 0)
	} else {
		// For files without a clear entity (main.go, config.go, etc.)
		// Check task inputs for entity hints
		task := cf.findTaskForFile(filePath, plan)
		if task != nil && task.Inputs != nil {
			if entities, ok := task.Inputs["entities"].([]interface{}); ok {
				for _, e := range entities {
					if entityName, ok := e.(string); ok {
						cf.addEntityWithDependencies(entityName, relevant, 0)
					}
				}
			}
		}

		// For handler/service files without specific entity, include entities from the same package
		if strings.Contains(fileName, "handler") || strings.Contains(fileName, "service") ||
			strings.Contains(fileName, "repository") {
			for _, entity := range fcs.DataModel.Entities {
				if strings.EqualFold(entity.Package, packageName) {
					cf.addEntityWithDependencies(entity.Name, relevant, 0)
				}
			}
		}
	}

	// If no entities found, include all (fallback for safety)
	if len(relevant) == 0 {
		log.Warn().
			Str("file_path", filePath).
			Msg("No relevant entities identified, including all entities")
		for _, entity := range fcs.DataModel.Entities {
			relevant[entity.Name] = true
		}
	}

	return relevant
}

// addEntityWithDependencies recursively adds an entity and its dependencies
func (cf *ContextFilter) addEntityWithDependencies(entityName string, relevant map[string]bool, depth int) {
	// Prevent infinite recursion
	if depth > 5 {
		return
	}

	// Already added
	if relevant[entityName] {
		return
	}

	relevant[entityName] = true
	log.Debug().
		Str("entity", entityName).
		Int("depth", depth).
		Msg("Added entity to relevant set")

	// Add direct dependencies
	if deps, exists := cf.depGraph[entityName]; exists {
		for _, dep := range deps {
			cf.addEntityWithDependencies(dep, relevant, depth+1)
		}
	}
}

// determineRelevantPackages identifies which packages are relevant
func (cf *ContextFilter) determineRelevantPackages(filePath string, _ *models.GenerationPlan, relevantEntities map[string]bool) map[string]bool {
	relevant := make(map[string]bool)

	// Always include the package this file belongs to
	fileDir := filepath.Dir(filePath)
	pathParts := strings.Split(fileDir, string(filepath.Separator))
	if len(pathParts) > 0 {
		packageName := pathParts[len(pathParts)-1]
		relevant[packageName] = true
	}

	// Include packages containing relevant entities
	for entityName := range relevantEntities {
		if pkg, exists := cf.entityPackages[entityName]; exists {
			relevant[pkg] = true

			// Include package dependencies
			if deps, exists := cf.packageDeps[pkg]; exists {
				for _, dep := range deps {
					relevant[dep] = true
				}
			}
		}
	}

	// Always include common packages
	commonPackages := []string{"main", "config", "common", "util"}
	for _, pkg := range commonPackages {
		relevant[pkg] = true
	}

	return relevant
}

// findTaskForFile finds the generation task for a given file path
func (cf *ContextFilter) findTaskForFile(filePath string, plan *models.GenerationPlan) *models.GenerationTask {
	if plan == nil {
		return nil
	}

	for _, phase := range plan.Phases {
		for i := range phase.Tasks {
			task := &phase.Tasks[i]
			if task.TargetPath == filePath || filepath.Base(task.TargetPath) == filepath.Base(filePath) {
				return task
			}
		}
	}

	return nil
}

// filterEntities returns only relevant entities
func (cf *ContextFilter) filterEntities(entities []models.Entity, relevant map[string]bool) []models.Entity {
	var filtered []models.Entity
	for _, entity := range entities {
		if relevant[entity.Name] {
			filtered = append(filtered, entity)
		}
	}
	return filtered
}

// filterRelationships returns only relationships between relevant entities
func (cf *ContextFilter) filterRelationships(relationships []models.Relationship, relevant map[string]bool) []models.Relationship {
	var filtered []models.Relationship
	for _, rel := range relationships {
		if relevant[rel.From] && relevant[rel.To] {
			filtered = append(filtered, rel)
		}
	}
	return filtered
}

// filterPackages returns only relevant packages
func (cf *ContextFilter) filterPackages(packages []models.Package, relevant map[string]bool) []models.Package {
	var filtered []models.Package
	for _, pkg := range packages {
		if relevant[pkg.Name] {
			filtered = append(filtered, pkg)
		}
	}
	return filtered
}

// filterAPIContracts returns only API contracts relevant to the file
func (cf *ContextFilter) filterAPIContracts(contracts []models.APIContract, filePath string, relevantPackages map[string]bool) []models.APIContract {
	// For handler files, include all contracts
	if strings.Contains(filePath, "handler") || strings.Contains(filePath, "api") {
		return contracts
	}

	// For other files, only include contracts if the file's package is in relevant packages
	fileDir := filepath.Dir(filePath)
	pathParts := strings.Split(fileDir, string(filepath.Separator))
	if len(pathParts) > 0 {
		packageName := pathParts[len(pathParts)-1]
		if relevantPackages[packageName] {
			return contracts
		}
	}

	// Otherwise, return empty (API contracts not needed)
	return nil
}

// FormatFilteredFCS formats a filtered FCS as a string for LLM prompts
func (cf *ContextFilter) FormatFilteredFCS(filtered *FilteredFCS) string {
	var sb strings.Builder

	sb.WriteString("# Final Clarified Specification (Filtered)\n\n")
	sb.WriteString(fmt.Sprintf("**Version**: %s | **Schema**: %s\n\n", filtered.Version, filtered.SchemaVersion))

	// Requirements
	if len(filtered.Requirements.Functional) > 0 {
		sb.WriteString("## Functional Requirements\n\n")
		for _, req := range filtered.Requirements.Functional {
			sb.WriteString(fmt.Sprintf("- **%s**: %s", req.ID, req.Description))
			if req.Priority != "" {
				sb.WriteString(fmt.Sprintf(" (Priority: %s)", req.Priority))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Architecture - Packages
	if len(filtered.Architecture.Packages) > 0 {
		sb.WriteString("## Packages\n\n")
		for _, pkg := range filtered.Architecture.Packages {
			sb.WriteString(fmt.Sprintf("- **%s** (`%s`): %s\n", pkg.Name, pkg.Path, pkg.Purpose))
			if len(pkg.Dependencies) > 0 {
				sb.WriteString(fmt.Sprintf("  - Dependencies: %s\n", strings.Join(pkg.Dependencies, ", ")))
			}
		}
		sb.WriteString("\n")
	}

	// Data Model - Entities
	if len(filtered.DataModel.Entities) > 0 {
		sb.WriteString("## Entities\n\n")
		for _, entity := range filtered.DataModel.Entities {
			sb.WriteString(fmt.Sprintf("### %s\n", entity.Name))
			sb.WriteString(fmt.Sprintf("**Package**: %s\n\n", entity.Package))
			sb.WriteString("**Attributes**:\n")
			for name, typeStr := range entity.Attributes {
				sb.WriteString(fmt.Sprintf("- `%s`: %s\n", name, typeStr))
			}
			sb.WriteString("\n")
		}
	}

	// Relationships
	if len(filtered.DataModel.Relationships) > 0 {
		sb.WriteString("## Entity Relationships\n\n")
		for _, rel := range filtered.DataModel.Relationships {
			sb.WriteString(fmt.Sprintf("- **%s** → **%s** (%s)", rel.From, rel.To, rel.Type))
			if rel.Description != "" {
				sb.WriteString(fmt.Sprintf(": %s", rel.Description))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// API Contracts
	if len(filtered.APIContracts) > 0 {
		sb.WriteString("## API Contracts\n\n")
		for _, contract := range filtered.APIContracts {
			sb.WriteString(fmt.Sprintf("- **%s %s**: %s\n", contract.Method, contract.Endpoint, contract.Description))
		}
		sb.WriteString("\n")
	}

	// Testing Strategy
	sb.WriteString("## Testing Strategy\n\n")
	sb.WriteString(fmt.Sprintf("- Coverage Target: %.1f%%\n", filtered.TestingStrategy.CoverageTarget))
	sb.WriteString(fmt.Sprintf("- Unit Tests: %t\n", filtered.TestingStrategy.UnitTests))
	sb.WriteString(fmt.Sprintf("- Integration Tests: %t\n", filtered.TestingStrategy.IntegrationTests))
	sb.WriteString("\n")

	// Build Config
	sb.WriteString("## Build Configuration\n\n")
	sb.WriteString(fmt.Sprintf("- Go Version: %s\n", filtered.BuildConfig.GoVersion))
	sb.WriteString(fmt.Sprintf("- Output Path: %s\n", filtered.BuildConfig.OutputPath))
	sb.WriteString("\n")

	return sb.String()
}
