package spec

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dshills/gocreator/internal/models"
)

// Validator validates input specifications
type Validator struct {
	// Security configuration
	AllowAbsolutePaths bool
	MaxPathDepth       int
}

// NewValidator creates a new validator with default settings
func NewValidator() *Validator {
	return &Validator{
		AllowAbsolutePaths: false,
		MaxPathDepth:       10,
	}
}

// Validate performs full validation on an InputSpecification
func (v *Validator) Validate(spec *models.InputSpecification) error {
	// Run all validation checks
	checks := []func(*models.InputSpecification) error{
		ValidateInputSpec,
		ValidateSecurityConstraints,
		ValidateSchemaStructure,
	}

	for _, check := range checks {
		if err := check(spec); err != nil {
			return err
		}
	}

	return nil
}

// ValidateInputSpec validates required fields in the specification
func ValidateInputSpec(spec *models.InputSpecification) error {
	if spec.ParsedData == nil {
		return fmt.Errorf("parsed data is nil")
	}

	// Check required fields
	requiredFields := []string{"name", "description", "requirements"}
	for _, field := range requiredFields {
		if _, ok := spec.ParsedData[field]; !ok {
			return fmt.Errorf("required field '%s' is missing", field)
		}
	}

	// Validate field types
	if name, ok := spec.ParsedData["name"].(string); !ok || name == "" {
		return fmt.Errorf("field 'name' must be a non-empty string")
	}

	if desc, ok := spec.ParsedData["description"].(string); !ok || desc == "" {
		return fmt.Errorf("field 'description' must be a non-empty string")
	}

	return nil
}

// ValidateSecurityConstraints validates security-related constraints
func ValidateSecurityConstraints(spec *models.InputSpecification) error {
	// Check for path traversal attempts
	pathFields := []string{"output_path", "input_path", "project_root"}
	for _, field := range pathFields {
		if path, ok := spec.ParsedData[field].(string); ok {
			if err := validatePath(path); err != nil {
				return fmt.Errorf("security violation in field '%s': %w", field, err)
			}
		}
	}

	// Check for command injection attempts
	commandFields := []string{"build_command", "test_command", "install_command", "pre_build", "post_build"}
	for _, field := range commandFields {
		if cmd, ok := spec.ParsedData[field].(string); ok {
			if err := validateCommand(cmd); err != nil {
				return fmt.Errorf("security violation in field '%s': %w", field, err)
			}
		}
	}

	// Check nested structures
	if arch, ok := spec.ParsedData["architecture"].(map[string]interface{}); ok {
		if err := validateArchitectureSecurity(arch); err != nil {
			return fmt.Errorf("security violation in architecture: %w", err)
		}
	}

	return nil
}

// ValidateSchemaStructure validates the structure of the specification
func ValidateSchemaStructure(spec *models.InputSpecification) error {
	// Validate requirements structure
	if reqs, ok := spec.ParsedData["requirements"]; ok {
		if _, ok := reqs.([]interface{}); !ok {
			return fmt.Errorf("requirements must be an array")
		}
	}

	// Validate architecture structure if present
	if arch, ok := spec.ParsedData["architecture"]; ok {
		if archMap, ok := arch.(map[string]interface{}); ok {
			if err := validateArchitectureStructure(archMap); err != nil {
				return fmt.Errorf("invalid architecture structure: %w", err)
			}
		} else {
			return fmt.Errorf("architecture must be an object")
		}
	}

	// Validate API contracts structure if present
	if contracts, ok := spec.ParsedData["api_contracts"]; ok {
		if _, ok := contracts.([]interface{}); !ok {
			return fmt.Errorf("api_contracts must be an array")
		}
	}

	// Validate data model structure if present
	if dataModel, ok := spec.ParsedData["data_model"]; ok {
		if dataModelMap, ok := dataModel.(map[string]interface{}); ok {
			if err := validateDataModelStructure(dataModelMap); err != nil {
				return fmt.Errorf("invalid data_model structure: %w", err)
			}
		} else {
			return fmt.Errorf("data_model must be an object")
		}
	}

	return nil
}

// validatePath checks for path traversal and other path-based security issues
func validatePath(path string) error {
	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("null byte detected in path")
	}

	// Check for path traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal attempt detected (..)")
	}

	// Check for absolute paths outside project
	if filepath.IsAbs(path) {
		// Absolute paths could be dangerous
		if strings.HasPrefix(path, "/etc") || strings.HasPrefix(path, "/sys") ||
			strings.HasPrefix(path, "/proc") || strings.HasPrefix(path, "/dev") {
			return fmt.Errorf("absolute path to system directory not allowed")
		}
		return fmt.Errorf("absolute paths are not allowed")
	}

	// Check path depth
	parts := strings.Split(filepath.Clean(path), string(filepath.Separator))
	if len(parts) > 10 {
		return fmt.Errorf("path depth exceeds maximum allowed (10)")
	}

	return nil
}

// validateCommand checks for command injection attempts
func validateCommand(cmd string) error {
	// Check for null bytes
	if strings.Contains(cmd, "\x00") {
		return fmt.Errorf("null byte detected in command")
	}

	// Check for dangerous command injection patterns
	dangerousPatterns := []string{
		"&&", "||", ";", "|", "`", "$(",
		"rm -rf /", "rm -rf /*", "> /dev/", "curl", "wget",
	}

	cmdLower := strings.ToLower(cmd)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmdLower, pattern) {
			return fmt.Errorf("command injection pattern detected: %s", pattern)
		}
	}

	return nil
}

// validateArchitectureStructure validates the architecture object structure
func validateArchitectureStructure(arch map[string]interface{}) error {
	// If packages are present, validate structure
	if packages, ok := arch["packages"]; ok {
		if _, ok := packages.([]interface{}); !ok {
			return fmt.Errorf("architecture.packages must be an array")
		}
	}

	// If dependencies are present, validate structure
	if deps, ok := arch["dependencies"]; ok {
		if _, ok := deps.([]interface{}); !ok {
			return fmt.Errorf("architecture.dependencies must be an array")
		}
	}

	// If patterns are present, validate structure
	if patterns, ok := arch["patterns"]; ok {
		if _, ok := patterns.([]interface{}); !ok {
			return fmt.Errorf("architecture.patterns must be an array")
		}
	}

	return nil
}

// validateArchitectureSecurity validates security constraints in architecture
func validateArchitectureSecurity(arch map[string]interface{}) error {
	// Check package paths
	if packages, ok := arch["packages"].([]interface{}); ok {
		for i, pkg := range packages {
			if pkgMap, ok := pkg.(map[string]interface{}); ok {
				if path, ok := pkgMap["path"].(string); ok {
					if err := validatePath(path); err != nil {
						return fmt.Errorf("package[%d].path: %w", i, err)
					}
				}
			}
		}
	}

	return nil
}

// validateDataModelStructure validates the data model structure
func validateDataModelStructure(dataModel map[string]interface{}) error {
	// If entities are present, validate structure
	if entities, ok := dataModel["entities"]; ok {
		if _, ok := entities.([]interface{}); !ok {
			return fmt.Errorf("data_model.entities must be an array")
		}
	}

	// If relationships are present, validate structure
	if relationships, ok := dataModel["relationships"]; ok {
		if _, ok := relationships.([]interface{}); !ok {
			return fmt.Errorf("data_model.relationships must be an array")
		}
	}

	return nil
}

// ValidateForFCS validates that a specification is ready for FCS conversion
func ValidateForFCS(spec *models.InputSpecification) error {
	if spec.State != models.SpecStateValid {
		return fmt.Errorf("specification must be in valid state for FCS conversion")
	}

	// Ensure all critical fields are present
	requiredForFCS := []string{"name", "description", "requirements"}
	for _, field := range requiredForFCS {
		if _, ok := spec.ParsedData[field]; !ok {
			return fmt.Errorf("field '%s' is required for FCS conversion", field)
		}
	}

	return nil
}
