// Package clarify provides interactive clarification of ambiguous specifications.
package clarify

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/pkg/llm"
	"github.com/rs/zerolog/log"
)

// Analyzer identifies ambiguities in specifications
type Analyzer interface {
	// Analyze examines a specification and identifies ambiguities
	Analyze(ctx context.Context, spec *models.InputSpecification) ([]models.Ambiguity, error)
}

// LLMAnalyzer uses an LLM to identify ambiguities
type LLMAnalyzer struct {
	client llm.Client
}

// NewLLMAnalyzer creates a new LLM-based analyzer
func NewLLMAnalyzer(client llm.Client) *LLMAnalyzer {
	return &LLMAnalyzer{
		client: client,
	}
}

// Analyze uses an LLM to identify ambiguities in the specification
func (a *LLMAnalyzer) Analyze(ctx context.Context, spec *models.InputSpecification) ([]models.Ambiguity, error) {
	log.Info().
		Str("spec_id", spec.ID).
		Str("format", string(spec.Format)).
		Msg("Analyzing specification for ambiguities")

	// Build the analysis prompt
	prompt := a.buildAnalysisPrompt(spec)

	// Call LLM with deterministic temperature (0.0)
	response, err := a.client.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	// Parse the LLM response
	ambiguities, err := a.parseAnalysisResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	log.Info().
		Str("spec_id", spec.ID).
		Int("ambiguities_found", len(ambiguities)).
		Msg("Specification analysis completed")

	return ambiguities, nil
}

// buildAnalysisPrompt constructs the prompt for ambiguity detection
func (a *LLMAnalyzer) buildAnalysisPrompt(spec *models.InputSpecification) string {
	var sb strings.Builder

	sb.WriteString("You are an expert technical specification analyzer. ")
	sb.WriteString("Analyze the following specification and identify ALL ambiguities, missing constraints, ")
	sb.WriteString("conflicting requirements, unclear specifications, and underspecified features.\n\n")

	sb.WriteString("# Specification Content\n\n")
	sb.WriteString(spec.Content)
	sb.WriteString("\n\n")

	sb.WriteString("# Analysis Guidelines\n\n")
	sb.WriteString("Identify the following types of ambiguities:\n\n")
	sb.WriteString("1. **Missing Constraints**: Requirements that lack necessary constraints or bounds\n")
	sb.WriteString("2. **Conflicting Requirements**: Requirements that contradict each other\n")
	sb.WriteString("3. **Unclear Specifications**: Vague or imprecise requirement descriptions\n")
	sb.WriteString("4. **Ambiguous Terminology**: Terms used inconsistently or without clear definition\n")
	sb.WriteString("5. **Underspecified Features**: Features described at too high a level without implementation details\n\n")

	sb.WriteString("# Output Format\n\n")
	sb.WriteString("Return your analysis as a JSON array of ambiguity objects. Each object must have:\n")
	sb.WriteString("- type: one of 'missing_constraint', 'conflict', 'unclear_requirement', 'ambiguous_terminology', 'underspecified_feature'\n")
	sb.WriteString("- location: the section or requirement ID where the ambiguity occurs\n")
	sb.WriteString("- description: a clear description of the ambiguity\n")
	sb.WriteString("- severity: one of 'critical', 'important', 'minor'\n\n")

	sb.WriteString("Example:\n")
	sb.WriteString("```json\n")
	sb.WriteString("[\n")
	sb.WriteString("  {\n")
	sb.WriteString("    \"type\": \"missing_constraint\",\n")
	sb.WriteString("    \"location\": \"requirements.FR-003\",\n")
	sb.WriteString("    \"description\": \"No upper bound specified for number of concurrent users\",\n")
	sb.WriteString("    \"severity\": \"important\"\n")
	sb.WriteString("  }\n")
	sb.WriteString("]\n")
	sb.WriteString("```\n\n")

	sb.WriteString("Return ONLY the JSON array, no additional text.\n")

	return sb.String()
}

// parseAnalysisResponse parses the LLM's JSON response into ambiguities
func (a *LLMAnalyzer) parseAnalysisResponse(response string) ([]models.Ambiguity, error) {
	// Clean the response - extract JSON if it's wrapped in markdown
	cleaned := strings.TrimSpace(response)
	if strings.HasPrefix(cleaned, "```json") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
		cleaned = strings.TrimSuffix(cleaned, "```")
		cleaned = strings.TrimSpace(cleaned)
	} else if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```")
		cleaned = strings.TrimSuffix(cleaned, "```")
		cleaned = strings.TrimSpace(cleaned)
	}

	var ambiguities []models.Ambiguity
	if err := json.Unmarshal([]byte(cleaned), &ambiguities); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response: %w (response: %s)", err, cleaned)
	}

	// Validate and normalize ambiguities
	for i := range ambiguities {
		if ambiguities[i].Type == "" {
			ambiguities[i].Type = "unclear_requirement"
		}
		if ambiguities[i].Severity == "" {
			ambiguities[i].Severity = "important"
		}
	}

	return ambiguities, nil
}

// categorizeAmbiguity determines the category of an ambiguity
func categorizeAmbiguity(ambiguity models.Ambiguity) string {
	// Map severity to priority for clarification
	switch ambiguity.Severity {
	case "critical":
		return "Must address before generation"
	case "important":
		return "Should address for quality"
	case "minor":
		return "Optional improvement"
	default:
		return "Unknown priority"
	}
}

// FilterAmbiguities filters ambiguities by severity
func FilterAmbiguities(ambiguities []models.Ambiguity, minSeverity string) []models.Ambiguity {
	severityOrder := map[string]int{
		"critical":  3,
		"important": 2,
		"minor":     1,
	}

	minLevel, ok := severityOrder[minSeverity]
	if !ok {
		minLevel = 1 // Default to showing everything
	}

	var filtered []models.Ambiguity
	for _, amb := range ambiguities {
		level, ok := severityOrder[amb.Severity]
		if !ok {
			level = 2 // Default to important
		}
		if level >= minLevel {
			filtered = append(filtered, amb)
		}
	}

	return filtered
}

// GroupAmbiguities groups ambiguities by type
func GroupAmbiguities(ambiguities []models.Ambiguity) map[string][]models.Ambiguity {
	groups := make(map[string][]models.Ambiguity)
	for _, amb := range ambiguities {
		groups[amb.Type] = append(groups[amb.Type], amb)
	}
	return groups
}
