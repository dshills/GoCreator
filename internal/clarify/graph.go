package clarify

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/langgraph-go/graph"
	"github.com/dshills/langgraph-go/graph/emit"
	"github.com/dshills/langgraph-go/graph/store"
	"github.com/rs/zerolog/log"
)

// ClarificationState represents the state of the clarification workflow
type ClarificationState struct {
	Spec              *models.InputSpecification
	Ambiguities       []models.Ambiguity
	HasAmbiguities    bool
	Questions         []models.Question
	Answers           map[string]models.Answer
	FCS               *models.FinalClarifiedSpecification
	Error             error
	WorkflowStarted   bool
	WorkflowCompleted bool
}

// reduceClarificationState merges state updates
func reduceClarificationState(prev, delta ClarificationState) ClarificationState {
	if delta.Spec != nil {
		prev.Spec = delta.Spec
	}
	if delta.Ambiguities != nil {
		prev.Ambiguities = delta.Ambiguities
	}
	// Always apply boolean fields unconditionally to allow setting to false
	prev.HasAmbiguities = delta.HasAmbiguities
	if delta.Questions != nil {
		prev.Questions = delta.Questions
	}
	if delta.Answers != nil {
		prev.Answers = delta.Answers
	}
	if delta.FCS != nil {
		prev.FCS = delta.FCS
	}
	if delta.Error != nil {
		prev.Error = delta.Error
	}
	// Always apply boolean fields unconditionally to allow setting to false
	prev.WorkflowStarted = delta.WorkflowStarted
	prev.WorkflowCompleted = delta.WorkflowCompleted

	return prev
}

// ClarificationGraph builds and executes the clarification workflow
type ClarificationGraph struct {
	engine    *graph.Engine[ClarificationState]
	analyzer  Analyzer
	generator QuestionGenerator
}

// NewClarificationGraph creates a new clarification graph
func NewClarificationGraph(analyzer Analyzer, generator QuestionGenerator) (*ClarificationGraph, error) {
	if analyzer == nil {
		return nil, fmt.Errorf("analyzer is required")
	}
	if generator == nil {
		return nil, fmt.Errorf("generator is required")
	}

	cg := &ClarificationGraph{
		analyzer:  analyzer,
		generator: generator,
	}

	// Create store and emitter
	st := store.NewMemStore[ClarificationState]()
	emitter := emit.NewLogEmitter(os.Stdout, false)

	// Create engine with options
	engine := graph.New(
		reduceClarificationState,
		st,
		emitter,
		graph.WithMaxConcurrent(4),
		graph.WithDefaultNodeTimeout(5*time.Minute),
	)

	// Build the workflow nodes
	if err := cg.buildGraph(engine); err != nil {
		return nil, fmt.Errorf("failed to build clarification graph: %w", err)
	}

	cg.engine = engine

	return cg, nil
}

// buildGraph constructs the clarification workflow nodes
func (cg *ClarificationGraph) buildGraph(engine *graph.Engine[ClarificationState]) error {
	// Add nodes
	if err := engine.Add("start", graph.NodeFunc[ClarificationState](cg.startNode)); err != nil {
		return fmt.Errorf("failed to add start node: %w", err)
	}
	if err := engine.Add("analyze_spec", graph.NodeFunc[ClarificationState](cg.analyzeSpecNode)); err != nil {
		return fmt.Errorf("failed to add analyze_spec node: %w", err)
	}
	if err := engine.Add("check_ambiguities", graph.NodeFunc[ClarificationState](cg.checkAmbiguitiesNode)); err != nil {
		return fmt.Errorf("failed to add check_ambiguities node: %w", err)
	}
	if err := engine.Add("generate_questions", graph.NodeFunc[ClarificationState](cg.generateQuestionsNode)); err != nil {
		return fmt.Errorf("failed to add generate_questions node: %w", err)
	}
	if err := engine.Add("build_fcs", graph.NodeFunc[ClarificationState](cg.buildFCSNode)); err != nil {
		return fmt.Errorf("failed to add build_fcs node: %w", err)
	}
	if err := engine.Add("end", graph.NodeFunc[ClarificationState](cg.endNode)); err != nil {
		return fmt.Errorf("failed to add end node: %w", err)
	}

	// Set start node
	if err := engine.StartAt("start"); err != nil {
		return fmt.Errorf("failed to set start node: %w", err)
	}

	return nil
}

// Execute runs the clarification workflow
func (cg *ClarificationGraph) Execute(ctx context.Context, spec *models.InputSpecification) (*models.FinalClarifiedSpecification, error) {
	// Validate spec is not nil
	if spec == nil {
		return nil, fmt.Errorf("input specification is required")
	}

	// Create initial state
	initialState := ClarificationState{
		Spec:    spec,
		Answers: make(map[string]models.Answer),
	}

	log.Info().
		Str("spec_id", spec.ID).
		Msg("Starting clarification workflow execution")

	// Execute the graph
	executionID := fmt.Sprintf("clarify-%s", spec.ID)
	finalState, err := cg.engine.Run(ctx, executionID, initialState)
	if err != nil {
		return nil, fmt.Errorf("clarification workflow failed: %w", err)
	}

	// Check for errors in final state
	if finalState.Error != nil {
		return nil, finalState.Error
	}

	// Return the FCS
	if finalState.FCS == nil {
		return nil, fmt.Errorf("FCS not generated")
	}

	log.Info().
		Str("fcs_id", finalState.FCS.ID).
		Msg("Clarification workflow completed successfully")

	return finalState.FCS, nil
}

// Node implementations

func (cg *ClarificationGraph) startNode(_ context.Context, s ClarificationState) graph.NodeResult[ClarificationState] {
	log.Info().Msg("Starting clarification workflow")

	// Validate that spec exists
	if s.Spec == nil {
		return graph.NodeResult[ClarificationState]{
			Delta: ClarificationState{
				Error: fmt.Errorf("input specification not found in state"),
			},
			Route: graph.Stop(),
		}
	}

	return graph.NodeResult[ClarificationState]{
		Delta: ClarificationState{
			WorkflowStarted: true,
		},
		Route: graph.Goto("analyze_spec"),
	}
}

func (cg *ClarificationGraph) analyzeSpecNode(ctx context.Context, s ClarificationState) graph.NodeResult[ClarificationState] {
	log.Info().Msg("Analyzing specification for ambiguities")

	// Analyze for ambiguities
	ambiguities, err := cg.analyzer.Analyze(ctx, s.Spec)
	if err != nil {
		return graph.NodeResult[ClarificationState]{
			Delta: ClarificationState{
				Error: fmt.Errorf("analysis failed: %w", err),
			},
			Route: graph.Stop(),
		}
	}

	log.Info().
		Int("ambiguities_found", len(ambiguities)).
		Msg("Specification analysis completed")

	return graph.NodeResult[ClarificationState]{
		Delta: ClarificationState{
			Ambiguities: ambiguities,
		},
		Route: graph.Goto("check_ambiguities"),
	}
}

func (cg *ClarificationGraph) checkAmbiguitiesNode(_ context.Context, s ClarificationState) graph.NodeResult[ClarificationState] {
	hasAmbiguities := len(s.Ambiguities) > 0

	if !hasAmbiguities {
		log.Info().Msg("No ambiguities found - specification is clear")

		// Skip question generation and go directly to building FCS
		return graph.NodeResult[ClarificationState]{
			Delta: ClarificationState{
				HasAmbiguities: false,
			},
			Route: graph.Goto("build_fcs"),
		}
	}

	log.Info().
		Int("count", len(s.Ambiguities)).
		Msg("Ambiguities detected - questions will be generated")

	return graph.NodeResult[ClarificationState]{
		Delta: ClarificationState{
			HasAmbiguities: true,
		},
		Route: graph.Goto("generate_questions"),
	}
}

func (cg *ClarificationGraph) generateQuestionsNode(ctx context.Context, s ClarificationState) graph.NodeResult[ClarificationState] {
	log.Info().Msg("Generating clarification questions")

	// Generate questions
	questions, err := cg.generator.Generate(ctx, s.Ambiguities)
	if err != nil {
		return graph.NodeResult[ClarificationState]{
			Delta: ClarificationState{
				Error: fmt.Errorf("question generation failed: %w", err),
			},
			Route: graph.Stop(),
		}
	}

	log.Info().
		Int("questions_generated", len(questions)).
		Msg("Clarification questions generated")

	return graph.NodeResult[ClarificationState]{
		Delta: ClarificationState{
			Questions: questions,
		},
		Route: graph.Goto("build_fcs"),
	}
}

func (cg *ClarificationGraph) buildFCSNode(_ context.Context, s ClarificationState) graph.NodeResult[ClarificationState] {
	log.Info().Msg("Building Final Clarified Specification")

	// Build FCS
	fcs := buildFCSFromSpec(s.Spec, s.Answers)

	log.Info().
		Str("fcs_id", fcs.ID).
		Msg("Final Clarified Specification built")

	return graph.NodeResult[ClarificationState]{
		Delta: ClarificationState{
			FCS: fcs,
		},
		Route: graph.Goto("end"),
	}
}

func (cg *ClarificationGraph) endNode(_ context.Context, _ ClarificationState) graph.NodeResult[ClarificationState] {
	log.Info().Msg("Clarification workflow completed")

	return graph.NodeResult[ClarificationState]{
		Delta: ClarificationState{
			WorkflowCompleted: true,
		},
		Route: graph.Stop(),
	}
}

// buildFCSFromSpec creates an FCS from the input spec and answers
func buildFCSFromSpec(spec *models.InputSpecification, answers map[string]models.Answer) *models.FinalClarifiedSpecification {
	// This is a simplified version. A full implementation would:
	// 1. Parse the spec content into structured requirements
	// 2. Apply answers to resolve ambiguities
	// 3. Extract architecture, data model, etc.
	// 4. Validate completeness

	fcs := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             fmt.Sprintf("fcs-%s", spec.ID),
		Version:        "1.0",
		OriginalSpecID: spec.ID,
		Metadata: models.FCSMetadata{
			OriginalSpec:   spec.Content,
			Clarifications: []models.AppliedClarification{},
		},
		Requirements: models.Requirements{
			Functional:    []models.FunctionalRequirement{},
			NonFunctional: []models.NonFunctionalRequirement{},
		},
		Architecture: models.Architecture{
			Packages:     []models.Package{},
			Dependencies: []models.Dependency{},
			Patterns:     []models.DesignPattern{},
		},
	}

	// Apply answers if provided
	for qID, answer := range answers {
		var answerText string
		if answer.SelectedOption != nil {
			answerText = *answer.SelectedOption
		} else if answer.CustomAnswer != nil {
			answerText = *answer.CustomAnswer
		}

		fcs.Metadata.Clarifications = append(fcs.Metadata.Clarifications, models.AppliedClarification{
			QuestionID: qID,
			Answer:     answerText,
			AppliedTo:  "specification",
		})
	}

	// Compute hash
	hash, _ := fcs.ComputeHash()
	fcs.Metadata.Hash = hash

	return fcs
}
