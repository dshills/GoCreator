package spec

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/google/uuid"
)

// JSONParser implements the Parser interface for JSON format
type JSONParser struct{}

// Parse parses JSON content into an InputSpecification
func (p *JSONParser) Parse(content string) (*models.InputSpecification, error) {
	// Validate content is not empty
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("empty content provided")
	}

	// Parse JSON into a map
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil, fmt.Errorf("failed to parse json: %w", err)
	}

	// Create InputSpecification
	spec := &models.InputSpecification{
		SchemaVersion: "1.0",
		ID:            uuid.New().String(),
		Format:        models.FormatJSON,
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

// ParseJSONFile is a helper for testing or direct file parsing
func ParseJSONFile(content string) (*models.InputSpecification, error) {
	parser := &JSONParser{}
	return parser.Parse(content)
}
