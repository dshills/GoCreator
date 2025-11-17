package spec

import (
	"fmt"

	"github.com/dshills/gocreator/internal/models"
)

// Parser defines the interface for specification parsers
type Parser interface {
	// Parse parses the raw specification content and returns an InputSpecification
	Parse(content string) (*models.InputSpecification, error)
}

// NewParser creates a parser based on the specified format
func NewParser(format models.SpecFormat) (Parser, error) {
	switch format {
	case models.FormatYAML:
		return &YAMLParser{}, nil
	case models.FormatJSON:
		return &JSONParser{}, nil
	case models.FormatMarkdown:
		return &MarkdownParser{}, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// ParseSpec is a convenience function that creates a parser and parses content
func ParseSpec(format models.SpecFormat, content string) (*models.InputSpecification, error) {
	parser, err := NewParser(format)
	if err != nil {
		return nil, fmt.Errorf("failed to create parser: %w", err)
	}

	spec, err := parser.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse specification: %w", err)
	}

	return spec, nil
}

// ParseAndValidate parses and validates a specification in one step
func ParseAndValidate(format models.SpecFormat, content string) (*models.InputSpecification, error) {
	spec, err := ParseSpec(format, content)
	if err != nil {
		return nil, err
	}

	validator := NewValidator()
	if err := validator.Validate(spec); err != nil {
		spec.State = models.SpecStateInvalid
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := spec.TransitionTo(models.SpecStateValid); err != nil {
		return nil, fmt.Errorf("failed to transition to valid state: %w", err)
	}

	return spec, nil
}
