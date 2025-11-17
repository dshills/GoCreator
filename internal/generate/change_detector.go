package generate

import (
	"github.com/dshills/gocreator/internal/models"
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
		HasChanges: false,
	}

	// Detect requirement changes
	cd.detectRequirementChanges(oldFCS, newFCS, changes)

	// Detect non-functional requirement changes
	cd.detectNonFunctionalRequirementChanges(oldFCS, newFCS, changes)

	// Detect package changes
	cd.detectPackageChanges(oldFCS, newFCS, changes)

	// Set HasChanges flag
	changes.HasChanges = len(changes.AddedRequirements) > 0 ||
		len(changes.ModifiedRequirements) > 0 ||
		len(changes.DeletedRequirements) > 0 ||
		len(changes.AddedNonFunctionalRequirements) > 0 ||
		len(changes.ModifiedNonFunctionalRequirements) > 0 ||
		len(changes.DeletedNonFunctionalRequirements) > 0 ||
		len(changes.AddedPackages) > 0 ||
		len(changes.ModifiedPackages) > 0 ||
		len(changes.DeletedPackages) > 0

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
