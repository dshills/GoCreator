package models

import (
	"fmt"
	"time"
)

// SpecFormat represents the format of the input specification
type SpecFormat string

// SpecFormat constants define the supported specification file formats
const (
	FormatYAML     SpecFormat = "yaml"
	FormatJSON     SpecFormat = "json"
	FormatMarkdown SpecFormat = "markdown"
)

// SpecState represents the state of a specification in its lifecycle
type SpecState string

// SpecState constants define the lifecycle states of a specification
const (
	SpecStateUnparsed SpecState = "unparsed"
	SpecStateParsed   SpecState = "parsed"
	SpecStateValid    SpecState = "valid"
	SpecStateInvalid  SpecState = "invalid"
)

// SpecMetadata contains metadata about a specification
type SpecMetadata struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	Author    string    `json:"author,omitempty"`
	Version   string    `json:"version"`
	Tags      []string  `json:"tags,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// InputSpecification represents the initial specification provided by the user
type InputSpecification struct {
	SchemaVersion    string                 `json:"schema_version"`
	ID               string                 `json:"id"`
	Format           SpecFormat             `json:"format"`
	Content          string                 `json:"content"`
	ParsedData       map[string]interface{} `json:"parsed_data,omitempty"`
	Metadata         SpecMetadata           `json:"metadata"`
	ValidationErrors []ValidationError      `json:"validation_errors,omitempty"`
	State            SpecState              `json:"state"`
}

// Validate validates the input specification
func (s *InputSpecification) Validate() error {
	if s.ParsedData == nil {
		return fmt.Errorf("parsed data is nil")
	}

	// Check required fields
	if _, ok := s.ParsedData["name"]; !ok {
		return fmt.Errorf("required field 'name' is missing")
	}

	if _, ok := s.ParsedData["description"]; !ok {
		return fmt.Errorf("required field 'description' is missing")
	}

	if _, ok := s.ParsedData["requirements"]; !ok {
		return fmt.Errorf("required field 'requirements' is missing")
	}

	return nil
}

// TransitionTo attempts to transition the specification to a new state
func (s *InputSpecification) TransitionTo(newState SpecState) error {
	// Define valid state transitions
	validTransitions := map[SpecState][]SpecState{
		SpecStateUnparsed: {SpecStateParsed},
		SpecStateParsed:   {SpecStateValid, SpecStateInvalid},
		SpecStateValid:    {}, // Terminal state
		SpecStateInvalid:  {}, // Terminal state
	}

	allowed, ok := validTransitions[s.State]
	if !ok {
		return fmt.Errorf("unknown state: %s", s.State)
	}

	// Check if transition is allowed
	for _, allowedState := range allowed {
		if allowedState == newState {
			s.State = newState
			return nil
		}
	}

	return fmt.Errorf("invalid state transition from %s to %s", s.State, newState)
}

// IsValidFormat checks if the specification format is valid
func (s *InputSpecification) IsValidFormat() bool {
	switch s.Format {
	case FormatYAML, FormatJSON, FormatMarkdown:
		return true
	default:
		return false
	}
}
