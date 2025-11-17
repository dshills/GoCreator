package spec

import (
	"fmt"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/google/uuid"
)

// FCSBuilder constructs a Final Clarified Specification from an InputSpecification
type FCSBuilder struct {
	spec *models.InputSpecification
}

// NewFCSBuilder creates a new FCS builder
func NewFCSBuilder(spec *models.InputSpecification) *FCSBuilder {
	return &FCSBuilder{
		spec: spec,
	}
}

// Build constructs the FCS from the validated input specification
func (b *FCSBuilder) Build() (*models.FinalClarifiedSpecification, error) {
	// Validate the input specification is ready for FCS conversion
	if err := ValidateForFCS(b.spec); err != nil {
		return nil, fmt.Errorf("specification not ready for FCS conversion: %w", err)
	}

	fcs := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             uuid.New().String(),
		Version:        "1.0",
		OriginalSpecID: b.spec.ID,
		Metadata: models.FCSMetadata{
			CreatedAt:      time.Now(),
			OriginalSpec:   b.spec.ID,
			Clarifications: []models.AppliedClarification{},
		},
	}

	// Build requirements
	requirements, err := b.buildRequirements()
	if err != nil {
		return nil, fmt.Errorf("failed to build requirements: %w", err)
	}
	fcs.Requirements = requirements

	// Build architecture if present
	architecture, err := b.buildArchitecture()
	if err != nil {
		return nil, fmt.Errorf("failed to build architecture: %w", err)
	}
	fcs.Architecture = architecture

	// Build data model if present
	dataModel, err := b.buildDataModel()
	if err != nil {
		return nil, fmt.Errorf("failed to build data model: %w", err)
	}
	fcs.DataModel = dataModel

	// Build API contracts if present
	apiContracts, err := b.buildAPIContracts()
	if err != nil {
		return nil, fmt.Errorf("failed to build API contracts: %w", err)
	}
	fcs.APIContracts = apiContracts

	// Build testing strategy if present
	testingStrategy, err := b.buildTestingStrategy()
	if err != nil {
		return nil, fmt.Errorf("failed to build testing strategy: %w", err)
	}
	fcs.TestingStrategy = testingStrategy

	// Build build config if present
	buildConfig, err := b.buildBuildConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %w", err)
	}
	fcs.BuildConfig = buildConfig

	// Compute and set hash
	hash, err := fcs.ComputeHash()
	if err != nil {
		return nil, fmt.Errorf("failed to compute hash: %w", err)
	}
	fcs.Metadata.Hash = hash

	// Validate the constructed FCS
	if err := fcs.Validate(); err != nil {
		return nil, fmt.Errorf("constructed FCS failed validation: %w", err)
	}

	return fcs, nil
}

// buildRequirements extracts and builds the requirements section
func (b *FCSBuilder) buildRequirements() (models.Requirements, error) {
	reqs := models.Requirements{
		Functional:    []models.FunctionalRequirement{},
		NonFunctional: []models.NonFunctionalRequirement{},
	}

	// Extract requirements array
	reqsData, ok := b.spec.ParsedData["requirements"].([]interface{})
	if !ok {
		return reqs, nil // No requirements is valid (empty array)
	}

	// Process each requirement
	for _, reqItem := range reqsData {
		reqMap, ok := reqItem.(map[string]interface{})
		if !ok {
			continue // Skip invalid items
		}

		// Determine if functional or non-functional
		reqType, _ := reqMap["type"].(string)
		if reqType == "non-functional" || reqType == "nfr" {
			// Non-functional requirement
			nfr := models.NonFunctionalRequirement{
				ID:          getString(reqMap, "id"),
				Description: getString(reqMap, "description"),
				Type:        getString(reqMap, "nfr_type"),
				Threshold:   getString(reqMap, "threshold"),
			}
			reqs.NonFunctional = append(reqs.NonFunctional, nfr)
		} else {
			// Functional requirement (default)
			fr := models.FunctionalRequirement{
				ID:          getString(reqMap, "id"),
				Description: getString(reqMap, "description"),
				Priority:    getString(reqMap, "priority"),
				Category:    getString(reqMap, "category"),
			}
			reqs.Functional = append(reqs.Functional, fr)
		}
	}

	return reqs, nil
}

// buildArchitecture extracts and builds the architecture section
func (b *FCSBuilder) buildArchitecture() (models.Architecture, error) {
	arch := models.Architecture{
		Packages:     []models.Package{},
		Dependencies: []models.Dependency{},
		Patterns:     []models.DesignPattern{},
	}

	archData, ok := b.spec.ParsedData["architecture"].(map[string]interface{})
	if !ok {
		return arch, nil // No architecture is valid
	}

	// Build packages
	if pkgsData, ok := archData["packages"].([]interface{}); ok {
		for _, pkgItem := range pkgsData {
			pkgMap, ok := pkgItem.(map[string]interface{})
			if !ok {
				continue
			}

			pkg := models.Package{
				Name:         getString(pkgMap, "name"),
				Path:         getString(pkgMap, "path"),
				Purpose:      getString(pkgMap, "purpose"),
				Dependencies: getStringSlice(pkgMap, "dependencies"),
			}
			arch.Packages = append(arch.Packages, pkg)
		}
	}

	// Build dependencies
	if depsData, ok := archData["dependencies"].([]interface{}); ok {
		for _, depItem := range depsData {
			depMap, ok := depItem.(map[string]interface{})
			if !ok {
				continue
			}

			dep := models.Dependency{
				Name:    getString(depMap, "name"),
				Version: getString(depMap, "version"),
				Purpose: getString(depMap, "purpose"),
			}
			arch.Dependencies = append(arch.Dependencies, dep)
		}
	}

	// Build patterns
	if patternsData, ok := archData["patterns"].([]interface{}); ok {
		for _, patternItem := range patternsData {
			patternMap, ok := patternItem.(map[string]interface{})
			if !ok {
				continue
			}

			pattern := models.DesignPattern{
				Name:        getString(patternMap, "name"),
				Description: getString(patternMap, "description"),
				AppliesTo:   getStringSlice(patternMap, "applies_to"),
			}
			arch.Patterns = append(arch.Patterns, pattern)
		}
	}

	return arch, nil
}

// buildDataModel extracts and builds the data model section
func (b *FCSBuilder) buildDataModel() (models.DataModel, error) {
	dm := models.DataModel{
		Entities:      []models.Entity{},
		Relationships: []models.Relationship{},
	}

	dmData, ok := b.spec.ParsedData["data_model"].(map[string]interface{})
	if !ok {
		return dm, nil // No data model is valid
	}

	// Build entities
	if entitiesData, ok := dmData["entities"].([]interface{}); ok {
		for _, entityItem := range entitiesData {
			entityMap, ok := entityItem.(map[string]interface{})
			if !ok {
				continue
			}

			entity := models.Entity{
				Name:       getString(entityMap, "name"),
				Package:    getString(entityMap, "package"),
				Attributes: getStringMap(entityMap, "attributes"),
			}
			dm.Entities = append(dm.Entities, entity)
		}
	}

	// Build relationships
	if relsData, ok := dmData["relationships"].([]interface{}); ok {
		for _, relItem := range relsData {
			relMap, ok := relItem.(map[string]interface{})
			if !ok {
				continue
			}

			rel := models.Relationship{
				From:        getString(relMap, "from"),
				To:          getString(relMap, "to"),
				Type:        getString(relMap, "type"),
				Description: getString(relMap, "description"),
			}
			dm.Relationships = append(dm.Relationships, rel)
		}
	}

	return dm, nil
}

// buildAPIContracts extracts and builds the API contracts section
func (b *FCSBuilder) buildAPIContracts() ([]models.APIContract, error) {
	contracts := []models.APIContract{}

	contractsData, ok := b.spec.ParsedData["api_contracts"].([]interface{})
	if !ok {
		return contracts, nil // No API contracts is valid
	}

	for _, contractItem := range contractsData {
		contractMap, ok := contractItem.(map[string]interface{})
		if !ok {
			continue
		}

		contract := models.APIContract{
			Endpoint:    getString(contractMap, "endpoint"),
			Method:      getString(contractMap, "method"),
			Description: getString(contractMap, "description"),
		}

		// Build request schema
		if reqData, ok := contractMap["request"].(map[string]interface{}); ok {
			contract.Request = models.ContractSchema{
				Fields: getStringMap(reqData, "fields"),
			}
		}

		// Build response schema
		if respData, ok := contractMap["response"].(map[string]interface{}); ok {
			contract.Response = models.ContractSchema{
				Fields: getStringMap(respData, "fields"),
			}
		}

		contracts = append(contracts, contract)
	}

	return contracts, nil
}

// buildTestingStrategy extracts and builds the testing strategy section
func (b *FCSBuilder) buildTestingStrategy() (models.TestingStrategy, error) {
	ts := models.TestingStrategy{
		CoverageTarget:   80.0, // Default
		UnitTests:        true,
		IntegrationTests: false,
		Frameworks:       []string{},
	}

	tsData, ok := b.spec.ParsedData["testing_strategy"].(map[string]interface{})
	if !ok {
		return ts, nil // Use defaults
	}

	if coverage, ok := tsData["coverage_target"].(float64); ok {
		ts.CoverageTarget = coverage
	}

	if unitTests, ok := tsData["unit_tests"].(bool); ok {
		ts.UnitTests = unitTests
	}

	if integrationTests, ok := tsData["integration_tests"].(bool); ok {
		ts.IntegrationTests = integrationTests
	}

	if frameworks, ok := tsData["frameworks"].([]interface{}); ok {
		for _, fw := range frameworks {
			if fwStr, ok := fw.(string); ok {
				ts.Frameworks = append(ts.Frameworks, fwStr)
			}
		}
	}

	return ts, nil
}

// buildBuildConfig extracts and builds the build configuration section
func (b *FCSBuilder) buildBuildConfig() (models.BuildConfig, error) {
	bc := models.BuildConfig{
		GoVersion:  "1.23",
		OutputPath: "./bin",
		BuildFlags: []string{},
	}

	bcData, ok := b.spec.ParsedData["build_config"].(map[string]interface{})
	if !ok {
		return bc, nil // Use defaults
	}

	if goVersion, ok := bcData["go_version"].(string); ok {
		bc.GoVersion = goVersion
	}

	if outputPath, ok := bcData["output_path"].(string); ok {
		bc.OutputPath = outputPath
	}

	if buildFlags, ok := bcData["build_flags"].([]interface{}); ok {
		for _, flag := range buildFlags {
			if flagStr, ok := flag.(string); ok {
				bc.BuildFlags = append(bc.BuildFlags, flagStr)
			}
		}
	}

	return bc, nil
}

// Helper functions for type conversion

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getStringSlice(m map[string]interface{}, key string) []string {
	result := []string{}
	if arr, ok := m[key].([]interface{}); ok {
		for _, item := range arr {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
	}
	return result
}

func getStringMap(m map[string]interface{}, key string) map[string]string {
	result := make(map[string]string)
	if subMap, ok := m[key].(map[string]interface{}); ok {
		for k, v := range subMap {
			if str, ok := v.(string); ok {
				result[k] = str
			}
		}
	}
	return result
}

// BuildFCS is a convenience function that builds an FCS from a validated specification
func BuildFCS(spec *models.InputSpecification) (*models.FinalClarifiedSpecification, error) {
	builder := NewFCSBuilder(spec)
	return builder.Build()
}
