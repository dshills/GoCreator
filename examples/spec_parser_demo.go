// Package main demonstrates the specification parsing capabilities of GoCreator.
package main

import (
	"fmt"
	"log"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/internal/spec"
)

func main() {
	fmt.Println("=== GoCreator Specification Parser Demo ===")

	// Example 1: Parse YAML specification
	demonstrateYAMLParsing()

	// Example 2: Parse JSON specification
	demonstrateJSONParsing()

	// Example 3: Parse Markdown specification
	demonstrateMarkdownParsing()

	// Example 4: Security validation
	demonstrateSecurityValidation()

	// Example 5: Complete FCS workflow
	demonstrateCompleteFCSWorkflow()
}

func demonstrateYAMLParsing() {
	fmt.Println("1. YAML Parsing Example")
	fmt.Println("------------------------")

	yamlContent := `
name: MyGoProject
description: A REST API service with authentication
requirements:
  - id: FR-001
    description: Implement user authentication
    priority: critical
  - id: FR-002
    description: Create REST API endpoints
    priority: high
architecture:
  packages:
    - name: api
      path: internal/api
      purpose: HTTP handlers
    - name: auth
      path: internal/auth
      purpose: Authentication logic
`

	inputSpec, err := spec.ParseSpec(models.FormatYAML, yamlContent)
	if err != nil {
		log.Fatalf("YAML parsing failed: %v", err)
	}

	fmt.Printf("✓ Parsed YAML specification: %s\n", inputSpec.ParsedData["name"])
	fmt.Printf("✓ Format: %s\n", inputSpec.Format)
	fmt.Printf("✓ State: %s\n", inputSpec.State)
	fmt.Printf("✓ Spec ID: %s\n\n", inputSpec.ID)
}

func demonstrateJSONParsing() {
	fmt.Println("2. JSON Parsing Example")
	fmt.Println("-----------------------")

	jsonContent := `{
  "name": "DataProcessor",
  "description": "A data processing service",
  "requirements": [
    {
      "id": "FR-001",
      "description": "Process CSV files",
      "priority": "high"
    },
    {
      "id": "FR-002",
      "description": "Generate reports",
      "priority": "medium"
    }
  ]
}`

	inputSpec, err := spec.ParseSpec(models.FormatJSON, jsonContent)
	if err != nil {
		log.Fatalf("JSON parsing failed: %v", err)
	}

	fmt.Printf("✓ Parsed JSON specification: %s\n", inputSpec.ParsedData["name"])
	fmt.Printf("✓ Format: %s\n\n", inputSpec.Format)
}

func demonstrateMarkdownParsing() {
	fmt.Println("3. Markdown Parsing Example")
	fmt.Println("---------------------------")

	markdownContent := `---
name: DocumentationGenerator
description: Generate project documentation
requirements:
  - id: FR-001
    description: Parse source code
  - id: FR-002
    description: Generate HTML output
---

# DocumentationGenerator

This tool generates beautiful documentation from your Go code.

## Features
- Automatic API documentation
- Code examples extraction
- Customizable templates
`

	inputSpec, err := spec.ParseSpec(models.FormatMarkdown, markdownContent)
	if err != nil {
		log.Fatalf("Markdown parsing failed: %v", err)
	}

	fmt.Printf("✓ Parsed Markdown specification: %s\n", inputSpec.ParsedData["name"])
	fmt.Printf("✓ Format: %s\n\n", inputSpec.Format)
}

func demonstrateSecurityValidation() {
	fmt.Println("4. Security Validation Example")
	fmt.Println("-------------------------------")

	// Malicious spec with path traversal
	maliciousYAML := `
name: MaliciousProject
description: This has security issues
requirements:
  - id: FR-001
    description: Test
output_path: ../../../etc/passwd
build_command: make && rm -rf /
`

	inputSpec, err := spec.ParseSpec(models.FormatYAML, maliciousYAML)
	if err != nil {
		log.Fatalf("Parsing failed: %v", err)
	}

	validator := spec.NewValidator()
	err = validator.Validate(inputSpec)
	if err != nil {
		fmt.Printf("✓ Security validation correctly detected issues:\n")
		fmt.Printf("  Error: %v\n\n", err)
	} else {
		fmt.Println("✗ Security validation should have caught issues!")
	}
}

func demonstrateCompleteFCSWorkflow() {
	fmt.Println("5. Complete FCS Workflow Example")
	fmt.Println("--------------------------------")

	completeSpec := `
name: MicroserviceTemplate
description: A complete microservice with all best practices
requirements:
  - id: FR-001
    description: HTTP API with versioning
    priority: critical
    category: api
  - id: FR-002
    description: PostgreSQL data persistence
    priority: high
    category: data
  - id: NFR-001
    description: API response time under 100ms
    type: non-functional
    nfr_type: performance
    threshold: "< 100ms"
architecture:
  packages:
    - name: api
      path: internal/api
      purpose: HTTP handlers and routing
    - name: service
      path: internal/service
      purpose: Business logic
    - name: repository
      path: internal/repository
      purpose: Data access layer
  dependencies:
    - name: github.com/lib/pq
      version: v1.10.0
      purpose: PostgreSQL driver
    - name: github.com/gorilla/mux
      version: v1.8.0
      purpose: HTTP router
data_model:
  entities:
    - name: User
      package: models
      attributes:
        id: uuid
        username: string
        email: string
        created_at: timestamp
testing_strategy:
  coverage_target: 85.0
  unit_tests: true
  integration_tests: true
  frameworks:
    - testify
    - gomock
    - testcontainers
build_config:
  go_version: "1.23"
  output_path: ./bin
  build_flags:
    - "-ldflags=-s -w"
    - "-trimpath"
`

	// Step 1: Parse
	fmt.Println("Step 1: Parsing specification...")
	inputSpec, err := spec.ParseSpec(models.FormatYAML, completeSpec)
	if err != nil {
		log.Fatalf("Parse failed: %v", err)
	}
	fmt.Printf("  ✓ Parsed: %s\n", inputSpec.ParsedData["name"])

	// Step 2: Validate
	fmt.Println("Step 2: Validating...")
	validator := spec.NewValidator()
	if err := validator.Validate(inputSpec); err != nil {
		log.Fatalf("Validation failed: %v", err)
	}
	fmt.Println("  ✓ Validation passed")

	// Step 3: Transition to valid state
	if err := inputSpec.TransitionTo(models.SpecStateValid); err != nil {
		log.Fatalf("State transition failed: %v", err)
	}
	fmt.Println("  ✓ State transitioned to: valid")

	// Step 4: Build FCS
	fmt.Println("Step 3: Building Final Clarified Specification...")
	fcs, err := spec.BuildFCS(inputSpec)
	if err != nil {
		log.Fatalf("FCS build failed: %v", err)
	}

	fmt.Println("  ✓ FCS created successfully!")
	fmt.Println("\n  FCS Summary:")
	fmt.Printf("  - Functional Requirements: %d\n", len(fcs.Requirements.Functional))
	fmt.Printf("  - Non-Functional Requirements: %d\n", len(fcs.Requirements.NonFunctional))
	fmt.Printf("  - Architecture Packages: %d\n", len(fcs.Architecture.Packages))
	fmt.Printf("  - External Dependencies: %d\n", len(fcs.Architecture.Dependencies))
	fmt.Printf("  - Data Model Entities: %d\n", len(fcs.DataModel.Entities))
	fmt.Printf("  - Coverage Target: %.0f%%\n", fcs.TestingStrategy.CoverageTarget)
	fmt.Printf("  - Go Version: %s\n", fcs.BuildConfig.GoVersion)
	fmt.Printf("  - FCS Hash: %s\n", fcs.Metadata.Hash[:32]+"...")
	fmt.Printf("  - Original Spec ID: %s\n", fcs.OriginalSpecID)

	fmt.Println("\n✓ Complete workflow successful!")
	fmt.Println("\n=== Demo Complete ===")
}
