package clarify

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/pkg/llm"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// QuestionGenerator generates clarification questions from ambiguities
type QuestionGenerator interface {
	// Generate creates clarification questions from identified ambiguities
	Generate(ctx context.Context, ambiguities []models.Ambiguity) ([]models.Question, error)
}

// LLMQuestionGenerator uses an LLM to generate clarification questions
type LLMQuestionGenerator struct {
	client llm.Client
}

// NewLLMQuestionGenerator creates a new LLM-based question generator
func NewLLMQuestionGenerator(client llm.Client) *LLMQuestionGenerator {
	return &LLMQuestionGenerator{
		client: client,
	}
}

// Generate creates targeted clarification questions from ambiguities
func (g *LLMQuestionGenerator) Generate(ctx context.Context, ambiguities []models.Ambiguity) ([]models.Question, error) {
	log.Info().
		Int("ambiguities", len(ambiguities)).
		Msg("Generating clarification questions")

	if len(ambiguities) == 0 {
		return []models.Question{}, nil
	}

	// Build the question generation prompt
	prompt := g.buildQuestionPrompt(ambiguities)

	// Call LLM with deterministic temperature (0.0)
	response, err := g.client.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM question generation failed: %w", err)
	}

	// Parse the LLM response
	questions, err := g.parseQuestionResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// Assign UUIDs to questions
	for i := range questions {
		questions[i].ID = uuid.New().String()
	}

	log.Info().
		Int("questions_generated", len(questions)).
		Msg("Clarification questions generated")

	return questions, nil
}

// buildQuestionPrompt constructs the prompt for question generation
func (g *LLMQuestionGenerator) buildQuestionPrompt(ambiguities []models.Ambiguity) string {
	var sb strings.Builder

	sb.WriteString("You are an expert at creating clarifying questions for technical specifications. ")
	sb.WriteString("Generate targeted, specific questions to resolve the following ambiguities.\n\n")

	sb.WriteString("# Identified Ambiguities\n\n")
	for i, amb := range ambiguities {
		sb.WriteString(fmt.Sprintf("%d. **Type**: %s | **Location**: %s | **Severity**: %s\n",
			i+1, amb.Type, amb.Location, amb.Severity))
		sb.WriteString(fmt.Sprintf("   **Description**: %s\n\n", amb.Description))
	}

	sb.WriteString("# Question Generation Guidelines\n\n")
	sb.WriteString("For each ambiguity, create ONE targeted question that:\n")
	sb.WriteString("1. Has a clear topic and context\n")
	sb.WriteString("2. Provides 2-4 specific options to choose from\n")
	sb.WriteString("3. Includes implications for each option\n")
	sb.WriteString("4. Is phrased clearly and concisely\n\n")

	sb.WriteString("# Output Format\n\n")
	sb.WriteString("Return your questions as a JSON array. Each question object must have:\n")
	sb.WriteString("- topic: brief topic (e.g., 'Concurrency Limits', 'Authentication Method')\n")
	sb.WriteString("- context: relevant context from the specification\n")
	sb.WriteString("- question: the clarifying question\n")
	sb.WriteString("- options: array of 2-4 option objects, each with:\n")
	sb.WriteString("  - label: short label (e.g., 'Option A: JWT Tokens')\n")
	sb.WriteString("  - description: what this option means\n")
	sb.WriteString("  - implications: what choosing this option implies for implementation\n\n")

	sb.WriteString("Example:\n")
	sb.WriteString("```json\n")
	sb.WriteString("[\n")
	sb.WriteString("  {\n")
	sb.WriteString("    \"topic\": \"User Concurrency\",\n")
	sb.WriteString("    \"context\": \"Requirement FR-003 mentions supporting concurrent users but doesn't specify limits\",\n")
	sb.WriteString("    \"question\": \"What is the maximum number of concurrent users the system should support?\",\n")
	sb.WriteString("    \"options\": [\n")
	sb.WriteString("      {\n")
	sb.WriteString("        \"label\": \"100 concurrent users\",\n")
	sb.WriteString("        \"description\": \"Small-scale deployment suitable for teams or departments\",\n")
	sb.WriteString("        \"implications\": \"Simpler architecture, lower resource requirements, faster development\"\n")
	sb.WriteString("      },\n")
	sb.WriteString("      {\n")
	sb.WriteString("        \"label\": \"1,000 concurrent users\",\n")
	sb.WriteString("        \"description\": \"Medium-scale deployment for organizations\",\n")
	sb.WriteString("        \"implications\": \"Requires connection pooling, caching, load balancing considerations\"\n")
	sb.WriteString("      },\n")
	sb.WriteString("      {\n")
	sb.WriteString("        \"label\": \"10,000+ concurrent users\",\n")
	sb.WriteString("        \"description\": \"Large-scale deployment for enterprises or public services\",\n")
	sb.WriteString("        \"implications\": \"Requires distributed architecture, horizontal scaling, advanced caching\"\n")
	sb.WriteString("      }\n")
	sb.WriteString("    ]\n")
	sb.WriteString("  }\n")
	sb.WriteString("]\n")
	sb.WriteString("```\n\n")

	sb.WriteString("Return ONLY the JSON array, no additional text.\n")

	return sb.String()
}

// parseQuestionResponse parses the LLM's JSON response into questions
func (g *LLMQuestionGenerator) parseQuestionResponse(response string) ([]models.Question, error) {
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

	var questions []models.Question
	if err := json.Unmarshal([]byte(cleaned), &questions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response: %w (response: %s)", err, cleaned)
	}

	// Validate questions
	for i, q := range questions {
		if len(q.Options) < 2 || len(q.Options) > 4 {
			return nil, fmt.Errorf("question %d has %d options, must have 2-4", i, len(q.Options))
		}
	}

	return questions, nil
}

// ValidateQuestions ensures questions meet quality criteria
func ValidateQuestions(questions []models.Question) error {
	for i, q := range questions {
		if q.Question == "" {
			return fmt.Errorf("question %d has empty question text", i)
		}
		if len(q.Options) < 2 {
			return fmt.Errorf("question %d has fewer than 2 options", i)
		}
		if len(q.Options) > 4 {
			return fmt.Errorf("question %d has more than 4 options", i)
		}
		for j, opt := range q.Options {
			if opt.Label == "" {
				return fmt.Errorf("question %d option %d has empty label", i, j)
			}
		}
	}
	return nil
}

// PrioritizeQuestions sorts questions by importance (based on severity)
func PrioritizeQuestions(questions []models.Question, ambiguities []models.Ambiguity) []models.Question {
	// Create a map of severity by index
	severityMap := make(map[int]string)
	for i, amb := range ambiguities {
		severityMap[i] = amb.Severity
	}

	// Assign priority scores
	type questionWithPriority struct {
		question models.Question
		priority int
	}

	priorityMap := map[string]int{
		"critical":  3,
		"important": 2,
		"minor":     1,
	}

	var qWithPriority []questionWithPriority
	for i, q := range questions {
		severity := severityMap[i]
		priority := priorityMap[severity]
		qWithPriority = append(qWithPriority, questionWithPriority{
			question: q,
			priority: priority,
		})
	}

	// Sort by priority (highest first)
	// Using a simple bubble sort for clarity
	for i := 0; i < len(qWithPriority)-1; i++ {
		for j := 0; j < len(qWithPriority)-i-1; j++ {
			if qWithPriority[j].priority < qWithPriority[j+1].priority {
				qWithPriority[j], qWithPriority[j+1] = qWithPriority[j+1], qWithPriority[j]
			}
		}
	}

	// Extract questions
	result := make([]models.Question, len(questions))
	for i, qp := range qWithPriority {
		result[i] = qp.question
	}

	return result
}
