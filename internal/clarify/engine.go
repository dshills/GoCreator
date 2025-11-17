package clarify

import (
	"context"
	"fmt"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/pkg/langgraph"
	"github.com/dshills/gocreator/pkg/llm"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Engine orchestrates the clarification process
type Engine interface {
	// Clarify processes a specification and returns an FCS
	// In interactive mode, it will prompt for user input
	// In batch mode, it will use default or provided answers
	Clarify(ctx context.Context, spec *models.InputSpecification, interactive bool) (*models.FinalClarifiedSpecification, error)

	// AnalyzeOnly identifies ambiguities without generating questions
	AnalyzeOnly(ctx context.Context, spec *models.InputSpecification) ([]models.Ambiguity, error)

	// GenerateRequest creates a clarification request from a spec
	GenerateRequest(ctx context.Context, spec *models.InputSpecification) (*models.ClarificationRequest, error)

	// ApplyAnswers applies user answers to build the FCS
	ApplyAnswers(ctx context.Context, spec *models.InputSpecification, request *models.ClarificationRequest, response *models.ClarificationResponse) (*models.FinalClarifiedSpecification, error)
}

// ClarificationEngine implements the Engine interface
type ClarificationEngine struct {
	llmClient        llm.Client
	analyzer         Analyzer
	generator        QuestionGenerator
	checkpointMgr    langgraph.CheckpointManager
	enableCheckpoint bool
}

// EngineConfig configures the clarification engine
type EngineConfig struct {
	LLMClient        llm.Client
	CheckpointDir    string
	EnableCheckpoint bool
}

// NewEngine creates a new clarification engine
func NewEngine(config EngineConfig) (*ClarificationEngine, error) {
	// Create analyzer and generator
	analyzer := NewLLMAnalyzer(config.LLMClient)
	generator := NewLLMQuestionGenerator(config.LLMClient)

	// Create checkpoint manager if enabled
	var checkpointMgr langgraph.CheckpointManager
	if config.EnableCheckpoint && config.CheckpointDir != "" {
		var err error
		checkpointMgr, err = langgraph.NewFileCheckpointManager(config.CheckpointDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create checkpoint manager: %w", err)
		}
	}

	return &ClarificationEngine{
		llmClient:        config.LLMClient,
		analyzer:         analyzer,
		generator:        generator,
		checkpointMgr:    checkpointMgr,
		enableCheckpoint: config.EnableCheckpoint,
	}, nil
}

// Clarify performs the full clarification workflow
func (e *ClarificationEngine) Clarify(
	ctx context.Context,
	spec *models.InputSpecification,
	interactive bool,
) (*models.FinalClarifiedSpecification, error) {
	log.Info().
		Str("spec_id", spec.ID).
		Bool("interactive", interactive).
		Msg("Starting clarification process")

	startTime := time.Now()

	// Create clarification graph
	clarifyGraph := NewClarificationGraph(e.analyzer, e.generator)

	// Build and execute the graph
	graph, err := clarifyGraph.BuildGraph()
	if err != nil {
		return nil, fmt.Errorf("failed to build clarification graph: %w", err)
	}

	// Add checkpointing if enabled
	if e.enableCheckpoint && e.checkpointMgr != nil {
		// Note: This would require modifying the graph to use checkpointing
		// For now, we'll execute without checkpointing
		log.Debug().Msg("Checkpointing configured but not yet integrated with graph")
	}

	// Create initial state
	initialState := langgraph.NewMapState()
	initialState.Set("spec", spec)
	initialState.Set("interactive", interactive)

	// Execute the graph
	finalState, err := graph.Execute(ctx, initialState)
	if err != nil {
		return nil, fmt.Errorf("clarification workflow failed: %w", err)
	}

	// Extract results
	fcsVal, ok := finalState.Get("fcs")
	if !ok {
		return nil, fmt.Errorf("FCS not found in final state")
	}

	fcs, ok := fcsVal.(*models.FinalClarifiedSpecification)
	if !ok {
		return nil, fmt.Errorf("invalid FCS type in final state")
	}

	duration := time.Since(startTime)
	log.Info().
		Str("spec_id", spec.ID).
		Str("fcs_id", fcs.ID).
		Dur("duration", duration).
		Msg("Clarification process completed")

	return fcs, nil
}

// AnalyzeOnly identifies ambiguities without generating questions
func (e *ClarificationEngine) AnalyzeOnly(
	ctx context.Context,
	spec *models.InputSpecification,
) ([]models.Ambiguity, error) {
	log.Info().
		Str("spec_id", spec.ID).
		Msg("Analyzing specification (analysis-only mode)")

	ambiguities, err := e.analyzer.Analyze(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	log.Info().
		Str("spec_id", spec.ID).
		Int("ambiguities_found", len(ambiguities)).
		Msg("Analysis completed")

	return ambiguities, nil
}

// GenerateRequest creates a clarification request from a spec
func (e *ClarificationEngine) GenerateRequest(
	ctx context.Context,
	spec *models.InputSpecification,
) (*models.ClarificationRequest, error) {
	log.Info().
		Str("spec_id", spec.ID).
		Msg("Generating clarification request")

	// Analyze for ambiguities
	ambiguities, err := e.analyzer.Analyze(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	// Generate questions
	questions, err := e.generator.Generate(ctx, ambiguities)
	if err != nil {
		return nil, fmt.Errorf("question generation failed: %w", err)
	}

	// Create clarification request
	request := &models.ClarificationRequest{
		SchemaVersion: "1.0",
		ID:            uuid.New().String(),
		SpecID:        spec.ID,
		Questions:     questions,
		Ambiguities:   ambiguities,
		CreatedAt:     time.Now(),
	}

	// Validate request
	if err := request.Validate(); err != nil {
		return nil, fmt.Errorf("invalid clarification request: %w", err)
	}

	log.Info().
		Str("spec_id", spec.ID).
		Str("request_id", request.ID).
		Int("questions", len(questions)).
		Int("ambiguities", len(ambiguities)).
		Msg("Clarification request generated")

	return request, nil
}

// ApplyAnswers applies user answers to build the FCS
func (e *ClarificationEngine) ApplyAnswers(
	_ context.Context,
	spec *models.InputSpecification,
	request *models.ClarificationRequest,
	response *models.ClarificationResponse,
) (*models.FinalClarifiedSpecification, error) {
	log.Info().
		Str("spec_id", spec.ID).
		Str("request_id", request.ID).
		Str("response_id", response.ID).
		Msg("Applying clarification answers")

	// Validate response against request
	if err := response.ValidateAgainst(request); err != nil {
		return nil, fmt.Errorf("invalid clarification response: %w", err)
	}

	// Build FCS from spec and answers
	fcs := buildFCSFromSpec(spec, response.Answers)

	// Validate FCS
	if err := fcs.Validate(); err != nil {
		return nil, fmt.Errorf("invalid FCS: %w", err)
	}

	log.Info().
		Str("spec_id", spec.ID).
		Str("fcs_id", fcs.ID).
		Int("clarifications_applied", len(fcs.Metadata.Clarifications)).
		Msg("FCS created from clarification answers")

	return fcs, nil
}

// GetQuestions extracts questions from a clarification request
func (e *ClarificationEngine) GetQuestions(request *models.ClarificationRequest) []models.Question {
	return request.Questions
}

// GetAmbiguities extracts ambiguities from a clarification request
func (e *ClarificationEngine) GetAmbiguities(request *models.ClarificationRequest) []models.Ambiguity {
	return request.Ambiguities
}

// ValidateSpec validates a specification before clarification
func (e *ClarificationEngine) ValidateSpec(spec *models.InputSpecification) error {
	if spec == nil {
		return fmt.Errorf("specification is nil")
	}

	if spec.ID == "" {
		return fmt.Errorf("specification ID is empty")
	}

	if spec.Content == "" {
		return fmt.Errorf("specification content is empty")
	}

	if !spec.IsValidFormat() {
		return fmt.Errorf("invalid specification format: %s", spec.Format)
	}

	if spec.State != models.SpecStateValid {
		return fmt.Errorf("specification must be in valid state, got: %s", spec.State)
	}

	return nil
}

// LogDecision logs a clarification decision with rationale
func (e *ClarificationEngine) LogDecision(
	_ context.Context,
	decision string,
	rationale string,
	alternatives []string,
) {
	log.Info().
		Str("component", "clarification_engine").
		Str("decision", decision).
		Str("rationale", rationale).
		Strs("alternatives", alternatives).
		Msg("Clarification decision made")
}
