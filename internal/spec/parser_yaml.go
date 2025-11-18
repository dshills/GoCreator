package spec

import (
	"fmt"
	"strings"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/internal/yamlutil"
	"github.com/google/uuid"
)

// YAMLParser implements the Parser interface for YAML format
type YAMLParser struct{}

// Parse parses YAML content into an InputSpecification
func (p *YAMLParser) Parse(content string) (*models.InputSpecification, error) {
	// Validate content is not empty
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("empty content provided")
	}

	// Parse YAML into a map with enhanced error reporting
	var data map[string]interface{}
	if err := yamlutil.Unmarshal([]byte(content), &data); err != nil {
		return nil, fmt.Errorf("failed to parse specification: %w", err)
	}

	// Create InputSpecification
	spec := &models.InputSpecification{
		SchemaVersion: "1.0",
		ID:            uuid.New().String(),
		Format:        models.FormatYAML,
		Content:       content,
		ParsedData:    data,
		Metadata: models.SpecMetadata{
			CreatedAt: time.Now(),
			Version:   "1.0",
		},
		State: models.SpecStateUnparsed,
	}

	// Transition to parsed state
	if err := spec.TransitionTo(models.SpecStateParsed); err != nil {
		return nil, fmt.Errorf("failed to transition to parsed state: %w", err)
	}

	return spec, nil
}

// ParseYAMLFile is a helper for testing or direct file parsing
func ParseYAMLFile(content string) (*models.InputSpecification, error) {
	parser := &YAMLParser{}
	return parser.Parse(content)
}
