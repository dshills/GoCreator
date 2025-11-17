package models

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// AppliedClarification represents a clarification that was applied to the FCS
type AppliedClarification struct {
	QuestionID string `json:"question_id"`
	Answer     string `json:"answer"`
	AppliedTo  string `json:"applied_to"`
}

// FCSMetadata contains metadata about the FCS
type FCSMetadata struct {
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at,omitempty"`
	OriginalSpec   string                 `json:"original_spec"`
	Clarifications []AppliedClarification `json:"clarifications,omitempty"`
	Hash           string                 `json:"hash"`
}

// FunctionalRequirement represents a functional requirement
type FunctionalRequirement struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Priority    string `json:"priority,omitempty"`
	Category    string `json:"category,omitempty"`
}

// NonFunctionalRequirement represents a non-functional requirement
type NonFunctionalRequirement struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Threshold   string `json:"threshold,omitempty"`
}

// Requirements contains all requirements
type Requirements struct {
	Functional    []FunctionalRequirement    `json:"functional"`
	NonFunctional []NonFunctionalRequirement `json:"non_functional,omitempty"`
}

// Package represents a Go package in the architecture
type Package struct {
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	Purpose      string   `json:"purpose,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// Dependency represents an external dependency
type Dependency struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Purpose string `json:"purpose,omitempty"`
}

// DesignPattern represents a design pattern to be applied
type DesignPattern struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	AppliesTo   []string `json:"applies_to,omitempty"`
}

// Architecture describes the system architecture
type Architecture struct {
	Packages     []Package       `json:"packages"`
	Dependencies []Dependency    `json:"dependencies,omitempty"`
	Patterns     []DesignPattern `json:"patterns,omitempty"`
}

// Entity represents a domain entity
type Entity struct {
	Name       string            `json:"name"`
	Package    string            `json:"package"`
	Attributes map[string]string `json:"attributes"`
}

// Relationship represents a relationship between entities
type Relationship struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// DataModel describes the data model
type DataModel struct {
	Entities      []Entity       `json:"entities"`
	Relationships []Relationship `json:"relationships,omitempty"`
}

// ContractSchema represents a request or response schema
type ContractSchema struct {
	Fields map[string]string `json:"fields"`
}

// APIContract represents an API endpoint contract
type APIContract struct {
	Endpoint    string         `json:"endpoint"`
	Method      string         `json:"method"`
	Description string         `json:"description"`
	Request     ContractSchema `json:"request,omitempty"`
	Response    ContractSchema `json:"response"`
}

// TestingStrategy describes the testing approach
type TestingStrategy struct {
	CoverageTarget   float64  `json:"coverage_target"`
	UnitTests        bool     `json:"unit_tests"`
	IntegrationTests bool     `json:"integration_tests"`
	Frameworks       []string `json:"frameworks,omitempty"`
}

// BuildConfig contains build configuration
type BuildConfig struct {
	GoVersion  string   `json:"go_version"`
	OutputPath string   `json:"output_path"`
	BuildFlags []string `json:"build_flags,omitempty"`
}

// FinalClarifiedSpecification represents the complete, clarified specification
type FinalClarifiedSpecification struct {
	SchemaVersion   string          `json:"schema_version"`
	ID              string          `json:"id"`
	Version         string          `json:"version"`
	OriginalSpecID  string          `json:"original_spec_id"`
	Metadata        FCSMetadata     `json:"metadata"`
	Requirements    Requirements    `json:"requirements"`
	Architecture    Architecture    `json:"architecture"`
	DataModel       DataModel       `json:"data_model,omitempty"`
	APIContracts    []APIContract   `json:"api_contracts,omitempty"`
	TestingStrategy TestingStrategy `json:"testing_strategy,omitempty"`
	BuildConfig     BuildConfig     `json:"build_config,omitempty"`
}

// Validate validates the FCS
func (f *FinalClarifiedSpecification) Validate() error {
	// Check for cyclic dependencies
	if f.HasCyclicDependencies() {
		return fmt.Errorf("cyclic dependency detected in package dependencies")
	}

	// Verify hash if present
	if f.Metadata.Hash != "" {
		computedHash, err := f.ComputeHash()
		if err != nil {
			return fmt.Errorf("failed to compute hash: %w", err)
		}
		if computedHash != f.Metadata.Hash {
			return fmt.Errorf("hash mismatch: expected %s, got %s", f.Metadata.Hash, computedHash)
		}
	}

	return nil
}

// ComputeHash computes a SHA-256 hash of the FCS content
func (f *FinalClarifiedSpecification) ComputeHash() (string, error) {
	// Create a copy without the hash field to avoid circular dependency
	temp := *f
	temp.Metadata.Hash = ""

	data, err := json.Marshal(temp)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// HasCyclicDependencies detects cyclic dependencies in the package structure
func (f *FinalClarifiedSpecification) HasCyclicDependencies() bool {
	// Build adjacency list
	graph := make(map[string][]string)
	for _, pkg := range f.Architecture.Packages {
		graph[pkg.Name] = pkg.Dependencies
	}

	// Track visited nodes and recursion stack
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	// DFS to detect cycles
	var hasCycle func(node string) bool
	hasCycle = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, dep := range graph[node] {
			// Self-cycle
			if dep == node {
				return true
			}
			// If not visited, recurse
			if !visited[dep] {
				if hasCycle(dep) {
					return true
				}
			} else if recStack[dep] {
				// Back edge found (cycle)
				return true
			}
		}

		recStack[node] = false
		return false
	}

	// Check all nodes
	for pkg := range graph {
		if !visited[pkg] {
			if hasCycle(pkg) {
				return true
			}
		}
	}

	return false
}
