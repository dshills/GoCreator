package generate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dshills/gocreator/internal/models"
	"github.com/rs/zerolog/log"
)

// FCSChanges represents the detected changes between two FCS versions
type FCSChanges struct {
	HasChanges                        bool
	AddedRequirements                 []models.FunctionalRequirement
	ModifiedRequirements              []models.FunctionalRequirement
	DeletedRequirements               []string
	AddedNonFunctionalRequirements    []models.NonFunctionalRequirement
	ModifiedNonFunctionalRequirements []models.NonFunctionalRequirement
	DeletedNonFunctionalRequirements  []string
	AddedPackages                     []models.Package
	ModifiedPackages                  []models.Package
	DeletedPackages                   []string
	AddedEntities                     []string
	ModifiedEntities                  []string
	DeletedEntities                   []string
	AddedAPIContracts                 []string
	ModifiedAPIContracts              []string
	DeletedAPIContracts               []string
	ArchitectureChanged               bool
	BuildConfigChanged                bool
}

// ChangeDetector detects changes between FCS versions
type ChangeDetector struct{}

// NewChangeDetector creates a new change detector
func NewChangeDetector() *ChangeDetector {
	return &ChangeDetector{}
}

// DetectChanges compares two FCS versions and identifies changes
func (cd *ChangeDetector) DetectChanges(oldFCS, newFCS *models.FinalClarifiedSpecification) (*FCSChanges, error) {
	changes := &FCSChanges{
		HasChanges:           false,
		AddedEntities:        []string{},
		ModifiedEntities:     []string{},
		DeletedEntities:      []string{},
		AddedAPIContracts:    []string{},
		ModifiedAPIContracts: []string{},
		DeletedAPIContracts:  []string{},
	}

	// Handle nil oldFCS (first generation)
	if oldFCS == nil {
		changes.HasChanges = true
		changes.AddedEntities = cd.getAllEntityNames(newFCS)
		changes.AddedAPIContracts = cd.getAllAPIEndpoints(newFCS)
		return changes, nil
	}

	// Detect requirement changes
	cd.detectRequirementChanges(oldFCS, newFCS, changes)

	// Detect non-functional requirement changes
	cd.detectNonFunctionalRequirementChanges(oldFCS, newFCS, changes)

	// Detect package changes
	cd.detectPackageChanges(oldFCS, newFCS, changes)

	// Detect entity changes (NEW)
	cd.detectEntityChanges(oldFCS, newFCS, changes)

	// Detect API contract changes (NEW)
	cd.detectAPIChanges(oldFCS, newFCS, changes)

	// Detect architecture changes (NEW)
	changes.ArchitectureChanged = cd.hasArchitectureChanged(oldFCS, newFCS)

	// Detect build config changes (NEW)
	changes.BuildConfigChanged = cd.hasBuildConfigChanged(oldFCS, newFCS)

	// Set HasChanges flag
	changes.HasChanges = len(changes.AddedRequirements) > 0 ||
		len(changes.ModifiedRequirements) > 0 ||
		len(changes.DeletedRequirements) > 0 ||
		len(changes.AddedNonFunctionalRequirements) > 0 ||
		len(changes.ModifiedNonFunctionalRequirements) > 0 ||
		len(changes.DeletedNonFunctionalRequirements) > 0 ||
		len(changes.AddedPackages) > 0 ||
		len(changes.ModifiedPackages) > 0 ||
		len(changes.DeletedPackages) > 0 ||
		len(changes.AddedEntities) > 0 ||
		len(changes.ModifiedEntities) > 0 ||
		len(changes.DeletedEntities) > 0 ||
		len(changes.AddedAPIContracts) > 0 ||
		len(changes.ModifiedAPIContracts) > 0 ||
		changes.ArchitectureChanged ||
		changes.BuildConfigChanged

	log.Debug().
		Int("added_entities", len(changes.AddedEntities)).
		Int("modified_entities", len(changes.ModifiedEntities)).
		Int("deleted_entities", len(changes.DeletedEntities)).
		Bool("architecture_changed", changes.ArchitectureChanged).
		Bool("build_config_changed", changes.BuildConfigChanged).
		Msg("Detected FCS changes")

	return changes, nil
}

// detectRequirementChanges identifies changes in functional requirements
func (cd *ChangeDetector) detectRequirementChanges(oldFCS, newFCS *models.FinalClarifiedSpecification, changes *FCSChanges) {
	// Build maps for quick lookup
	oldReqs := make(map[string]models.FunctionalRequirement)
	for _, req := range oldFCS.Requirements.Functional {
		oldReqs[req.ID] = req
	}

	newReqs := make(map[string]models.FunctionalRequirement)
	for _, req := range newFCS.Requirements.Functional {
		newReqs[req.ID] = req
	}

	// Find added and modified requirements
	for id, newReq := range newReqs {
		if oldReq, exists := oldReqs[id]; !exists {
			// Added requirement
			changes.AddedRequirements = append(changes.AddedRequirements, newReq)
		} else if !requirementEquals(oldReq, newReq) {
			// Modified requirement
			changes.ModifiedRequirements = append(changes.ModifiedRequirements, newReq)
		}
	}

	// Find deleted requirements
	for id := range oldReqs {
		if _, exists := newReqs[id]; !exists {
			changes.DeletedRequirements = append(changes.DeletedRequirements, id)
		}
	}
}

// detectNonFunctionalRequirementChanges identifies changes in non-functional requirements
func (cd *ChangeDetector) detectNonFunctionalRequirementChanges(oldFCS, newFCS *models.FinalClarifiedSpecification, changes *FCSChanges) {
	// Build maps for quick lookup
	oldReqs := make(map[string]models.NonFunctionalRequirement)
	for _, req := range oldFCS.Requirements.NonFunctional {
		oldReqs[req.ID] = req
	}

	newReqs := make(map[string]models.NonFunctionalRequirement)
	for _, req := range newFCS.Requirements.NonFunctional {
		newReqs[req.ID] = req
	}

	// Find added and modified requirements
	for id, newReq := range newReqs {
		if oldReq, exists := oldReqs[id]; !exists {
			// Added requirement
			changes.AddedNonFunctionalRequirements = append(changes.AddedNonFunctionalRequirements, newReq)
		} else if !nonFunctionalRequirementEquals(oldReq, newReq) {
			// Modified requirement
			changes.ModifiedNonFunctionalRequirements = append(changes.ModifiedNonFunctionalRequirements, newReq)
		}
	}

	// Find deleted requirements
	for id := range oldReqs {
		if _, exists := newReqs[id]; !exists {
			changes.DeletedNonFunctionalRequirements = append(changes.DeletedNonFunctionalRequirements, id)
		}
	}
}

// detectPackageChanges identifies changes in packages
func (cd *ChangeDetector) detectPackageChanges(oldFCS, newFCS *models.FinalClarifiedSpecification, changes *FCSChanges) {
	// Build maps for quick lookup
	oldPkgs := make(map[string]models.Package)
	for _, pkg := range oldFCS.Architecture.Packages {
		oldPkgs[pkg.Name] = pkg
	}

	newPkgs := make(map[string]models.Package)
	for _, pkg := range newFCS.Architecture.Packages {
		newPkgs[pkg.Name] = pkg
	}

	// Find added and modified packages
	for name, newPkg := range newPkgs {
		if oldPkg, exists := oldPkgs[name]; !exists {
			// Added package
			changes.AddedPackages = append(changes.AddedPackages, newPkg)
		} else if !packageEquals(oldPkg, newPkg) {
			// Modified package
			changes.ModifiedPackages = append(changes.ModifiedPackages, newPkg)
		}
	}

	// Find deleted packages
	for name := range oldPkgs {
		if _, exists := newPkgs[name]; !exists {
			changes.DeletedPackages = append(changes.DeletedPackages, name)
		}
	}
}

// IdentifyAffectedPackages determines which packages are affected by changes
func (cd *ChangeDetector) IdentifyAffectedPackages(changes *FCSChanges, architecture *models.Architecture) ([]string, error) {
	if !changes.HasChanges {
		return []string{}, nil
	}

	affected := make(map[string]bool)

	// Added packages affect only themselves
	for _, pkg := range changes.AddedPackages {
		affected[pkg.Name] = true
	}

	// Modified packages affect themselves and their dependents
	for _, pkg := range changes.ModifiedPackages {
		affected[pkg.Name] = true
		cd.addDependents(pkg.Name, architecture, affected)
	}

	// Deleted packages affect themselves and their dependents
	for _, pkgName := range changes.DeletedPackages {
		affected[pkgName] = true // Mark the deleted package itself
		cd.addDependents(pkgName, architecture, affected)
	}

	// Convert to slice
	result := make([]string, 0, len(affected))
	for pkgName := range affected {
		result = append(result, pkgName)
	}

	return result, nil
}

// addDependents recursively adds all packages that depend on the given package
func (cd *ChangeDetector) addDependents(pkgName string, architecture *models.Architecture, affected map[string]bool) {
	for _, pkg := range architecture.Packages {
		// Check if this package depends on pkgName
		for _, dep := range pkg.Dependencies {
			if dep == pkgName {
				// Only add if not already processed (avoid infinite loops)
				if !affected[pkg.Name] {
					affected[pkg.Name] = true
					// Recursively add dependents of this package
					cd.addDependents(pkg.Name, architecture, affected)
				}
				break
			}
		}
	}
}

// requirementEquals compares two functional requirements for equality
func requirementEquals(a, b models.FunctionalRequirement) bool {
	return a.ID == b.ID &&
		a.Description == b.Description &&
		a.Priority == b.Priority &&
		a.Category == b.Category
}

// nonFunctionalRequirementEquals compares two non-functional requirements for equality
func nonFunctionalRequirementEquals(a, b models.NonFunctionalRequirement) bool {
	return a.ID == b.ID &&
		a.Description == b.Description &&
		a.Type == b.Type &&
		a.Threshold == b.Threshold
}

// packageEquals compares two packages for equality
func packageEquals(a, b models.Package) bool {
	if a.Name != b.Name || a.Path != b.Path || a.Purpose != b.Purpose {
		return false
	}

	// Compare dependencies
	if len(a.Dependencies) != len(b.Dependencies) {
		return false
	}

	depMap := make(map[string]bool)
	for _, dep := range a.Dependencies {
		depMap[dep] = true
	}

	for _, dep := range b.Dependencies {
		if !depMap[dep] {
			return false
		}
	}

	return true
}

// detectEntityChanges identifies entity additions, modifications, and deletions
func (cd *ChangeDetector) detectEntityChanges(
	oldFCS, newFCS *models.FinalClarifiedSpecification,
	changes *FCSChanges,
) {
	// Build entity maps for efficient lookup
	oldEntities := make(map[string]*models.Entity)
	for i := range oldFCS.DataModel.Entities {
		entity := &oldFCS.DataModel.Entities[i]
		oldEntities[entity.Name] = entity
	}

	newEntities := make(map[string]*models.Entity)
	for i := range newFCS.DataModel.Entities {
		entity := &newFCS.DataModel.Entities[i]
		newEntities[entity.Name] = entity
	}

	// Find added and modified entities
	for name, newEntity := range newEntities {
		oldEntity, exists := oldEntities[name]
		if !exists {
			// Entity added
			changes.AddedEntities = append(changes.AddedEntities, name)
		} else {
			// Check if entity was modified
			if cd.hasEntityChanged(oldEntity, newEntity) {
				changes.ModifiedEntities = append(changes.ModifiedEntities, name)
			}
		}
	}

	// Find deleted entities
	for name := range oldEntities {
		if _, exists := newEntities[name]; !exists {
			changes.DeletedEntities = append(changes.DeletedEntities, name)
		}
	}
}

// hasEntityChanged checks if an entity was modified
func (cd *ChangeDetector) hasEntityChanged(old, updated *models.Entity) bool {
	// Check package change
	if old.Package != updated.Package {
		return true
	}

	// Check attribute count
	if len(old.Attributes) != len(updated.Attributes) {
		return true
	}

	// Check each attribute
	for attrName, oldType := range old.Attributes {
		newType, exists := updated.Attributes[attrName]
		if !exists || oldType != newType {
			return true
		}
	}

	return false
}

// detectAPIChanges identifies API contract changes
func (cd *ChangeDetector) detectAPIChanges(
	oldFCS, newFCS *models.FinalClarifiedSpecification,
	changes *FCSChanges,
) {
	oldAPIs := make(map[string]*models.APIContract)
	for i := range oldFCS.APIContracts {
		api := &oldFCS.APIContracts[i]
		key := fmt.Sprintf("%s %s", api.Method, api.Endpoint)
		oldAPIs[key] = api
	}

	newAPIs := make(map[string]*models.APIContract)
	for i := range newFCS.APIContracts {
		api := &newFCS.APIContracts[i]
		key := fmt.Sprintf("%s %s", api.Method, api.Endpoint)
		newAPIs[key] = api
	}

	// Find added and modified APIs
	for key, newAPI := range newAPIs {
		oldAPI, exists := oldAPIs[key]
		if !exists {
			changes.AddedAPIContracts = append(changes.AddedAPIContracts, key)
		} else {
			if cd.hasAPIChanged(oldAPI, newAPI) {
				changes.ModifiedAPIContracts = append(changes.ModifiedAPIContracts, key)
			}
		}
	}

	// Find deleted APIs
	for key := range oldAPIs {
		if _, exists := newAPIs[key]; !exists {
			changes.DeletedAPIContracts = append(changes.DeletedAPIContracts, key)
		}
	}
}

// hasAPIChanged checks if an API contract was modified
func (cd *ChangeDetector) hasAPIChanged(old, updated *models.APIContract) bool {
	// Compare using JSON serialization for deep equality
	oldJSON, err := json.Marshal(old)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to marshal old API contract, assuming changed")
		return true
	}
	newJSON, err := json.Marshal(updated)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to marshal new API contract, assuming changed")
		return true
	}
	return string(oldJSON) != string(newJSON)
}

// hasArchitectureChanged checks if architecture section changed
func (cd *ChangeDetector) hasArchitectureChanged(old, updated *models.FinalClarifiedSpecification) bool {
	oldJSON, err := json.Marshal(old.Architecture)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to marshal old architecture, assuming changed")
		return true
	}
	newJSON, err := json.Marshal(updated.Architecture)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to marshal new architecture, assuming changed")
		return true
	}
	return string(oldJSON) != string(newJSON)
}

// hasBuildConfigChanged checks if build config changed
func (cd *ChangeDetector) hasBuildConfigChanged(old, updated *models.FinalClarifiedSpecification) bool {
	oldJSON, err := json.Marshal(old.BuildConfig)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to marshal old build config, assuming changed")
		return true
	}
	newJSON, err := json.Marshal(updated.BuildConfig)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to marshal new build config, assuming changed")
		return true
	}
	return string(oldJSON) != string(newJSON)
}

// getAllEntityNames returns all entity names from FCS
func (cd *ChangeDetector) getAllEntityNames(fcs *models.FinalClarifiedSpecification) []string {
	names := make([]string, len(fcs.DataModel.Entities))
	for i, entity := range fcs.DataModel.Entities {
		names[i] = entity.Name
	}
	return names
}

// getAllAPIEndpoints returns all API endpoint keys from FCS
func (cd *ChangeDetector) getAllAPIEndpoints(fcs *models.FinalClarifiedSpecification) []string {
	endpoints := make([]string, len(fcs.APIContracts))
	for i, api := range fcs.APIContracts {
		endpoints[i] = fmt.Sprintf("%s %s", api.Method, api.Endpoint)
	}
	return endpoints
}

// AffectedFilesCalculator determines which files need regeneration based on changes
type AffectedFilesCalculator struct {
	dependencyGraph map[string][]string // file -> entity dependencies
}

// NewAffectedFilesCalculator creates a new calculator
func NewAffectedFilesCalculator(dependencyGraph map[string][]string) *AffectedFilesCalculator {
	return &AffectedFilesCalculator{
		dependencyGraph: dependencyGraph,
	}
}

// CalculateAffectedFiles determines which files need regeneration
func (afc *AffectedFilesCalculator) CalculateAffectedFiles(
	changes *FCSChanges,
	allFiles []string,
) []string {
	// If architecture or build config changed, regenerate everything
	if changes.ArchitectureChanged || changes.BuildConfigChanged {
		log.Debug().Msg("Architecture or build config changed, regenerating all files")
		return allFiles
	}

	affectedSet := make(map[string]bool)
	changedEntities := make(map[string]bool)

	// Build set of changed entities
	for _, entity := range changes.AddedEntities {
		changedEntities[entity] = true
	}
	for _, entity := range changes.ModifiedEntities {
		changedEntities[entity] = true
	}
	for _, entity := range changes.DeletedEntities {
		changedEntities[entity] = true
	}

	// Find files that depend on changed entities
	for filePath, dependencies := range afc.dependencyGraph {
		for _, dep := range dependencies {
			if changedEntities[dep] {
				affectedSet[filePath] = true
				break
			}
		}
	}

	// Special handling for deleted entities - mark their primary files as affected
	for _, deletedEntity := range changes.DeletedEntities {
		// Find files that implement this entity
		entityFileName := toSnakeCase(deletedEntity) + ".go"
		for _, filePath := range allFiles {
			if strings.Contains(filePath, entityFileName) {
				affectedSet[filePath] = true
			}
		}
	}

	// Convert set to slice
	affectedFiles := make([]string, 0, len(affectedSet))
	for filePath := range affectedSet {
		affectedFiles = append(affectedFiles, filePath)
	}

	log.Debug().
		Int("total_files", len(allFiles)).
		Int("affected_files", len(affectedFiles)).
		Int("changed_entities", len(changedEntities)).
		Msg("Calculated affected files for incremental regeneration")

	return affectedFiles
}

// toSnakeCase converts PascalCase to snake_case
func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	result := make([]rune, 0, len(s)+5)
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		if r >= 'A' && r <= 'Z' {
			result = append(result, r+('a'-'A'))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}
