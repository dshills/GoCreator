package clarify

import (
	"context"
	"fmt"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/pkg/langgraph"
	"github.com/rs/zerolog/log"
)

// ClarificationGraph builds and executes the clarification workflow
type ClarificationGraph struct {
	analyzer  Analyzer
	generator QuestionGenerator
}

// NewClarificationGraph creates a new clarification graph
func NewClarificationGraph(analyzer Analyzer, generator QuestionGenerator) *ClarificationGraph {
	return &ClarificationGraph{
		analyzer:  analyzer,
		generator: generator,
	}
}

// BuildGraph constructs the LangGraph for clarification workflow
func (cg *ClarificationGraph) BuildGraph() (*langgraph.Graph, error) {
	graph := langgraph.NewGraph(
		"clarification_workflow",
		"start",
		"end",
	)

	// Define nodes
	startNode := langgraph.NewBasicNode(
		"start",
		cg.startNode,
		[]string{},
		"Initialize clarification workflow",
	)

	analyzeNode := langgraph.NewBasicNode(
		"analyze_spec",
		cg.analyzeSpecNode,
		[]string{"start"},
		"Analyze specification for ambiguities",
	)

	checkAmbiguitiesNode := langgraph.NewConditionalNode(
		"check_ambiguities",
		cg.checkAmbiguitiesNode,
		[]string{"analyze_spec"},
		"Check if ambiguities were found",
		func(state langgraph.State) bool {
			// Check if ambiguities exist
			val, ok := state.Get("ambiguities")
			if !ok {
				return false
			}
			ambiguities, ok := val.([]models.Ambiguity)
			if !ok {
				return false
			}
			return len(ambiguities) > 0
		},
	)

	generateQuestionsNode := langgraph.NewBasicNode(
		"generate_questions",
		cg.generateQuestionsNode,
		[]string{"check_ambiguities"},
		"Generate clarification questions from ambiguities",
	)

	buildFCSNode := langgraph.NewBasicNode(
		"build_fcs",
		cg.buildFCSNode,
		[]string{"generate_questions"},
		"Build Final Clarified Specification",
	)

	endNode := langgraph.NewBasicNode(
		"end",
		cg.endNode,
		[]string{"build_fcs"},
		"Complete clarification workflow",
	)

	// Add nodes to graph
	if err := graph.AddNode(startNode); err != nil {
		return nil, err
	}
	if err := graph.AddNode(analyzeNode); err != nil {
		return nil, err
	}
	if err := graph.AddNode(checkAmbiguitiesNode); err != nil {
		return nil, err
	}
	if err := graph.AddNode(generateQuestionsNode); err != nil {
		return nil, err
	}
	if err := graph.AddNode(buildFCSNode); err != nil {
		return nil, err
	}
	if err := graph.AddNode(endNode); err != nil {
		return nil, err
	}

	return graph, nil
}

// startNode initializes the workflow state
func (cg *ClarificationGraph) startNode(ctx context.Context, state langgraph.State) (langgraph.State, error) {
	log.Info().Msg("Starting clarification workflow")

	// Validate that spec exists in state
	_, ok := state.Get("spec")
	if !ok {
		return nil, fmt.Errorf("input specification not found in state")
	}

	// Initialize workflow metadata
	state.Set("workflow_started", true)

	return state, nil
}

// analyzeSpecNode analyzes the specification for ambiguities
func (cg *ClarificationGraph) analyzeSpecNode(ctx context.Context, state langgraph.State) (langgraph.State, error) {
	log.Info().Msg("Analyzing specification for ambiguities")

	// Get spec from state
	specVal, ok := state.Get("spec")
	if !ok {
		return nil, fmt.Errorf("input specification not found in state")
	}

	spec, ok := specVal.(*models.InputSpecification)
	if !ok {
		return nil, fmt.Errorf("invalid spec type in state")
	}

	// Analyze for ambiguities
	ambiguities, err := cg.analyzer.Analyze(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	// Store ambiguities in state
	state.Set("ambiguities", ambiguities)
	state.Set("ambiguity_count", len(ambiguities))

	log.Info().
		Int("ambiguities_found", len(ambiguities)).
		Msg("Specification analysis completed")

	return state, nil
}

// checkAmbiguitiesNode checks if ambiguities were found
func (cg *ClarificationGraph) checkAmbiguitiesNode(ctx context.Context, state langgraph.State) (langgraph.State, error) {
	val, ok := state.Get("ambiguities")
	if !ok {
		state.Set("has_ambiguities", false)
		return state, nil
	}

	ambiguities, ok := val.([]models.Ambiguity)
	if !ok {
		state.Set("has_ambiguities", false)
		return state, nil
	}

	hasAmbiguities := len(ambiguities) > 0
	state.Set("has_ambiguities", hasAmbiguities)

	if !hasAmbiguities {
		log.Info().Msg("No ambiguities found - specification is clear")
	} else {
		log.Info().
			Int("count", len(ambiguities)).
			Msg("Ambiguities detected - questions will be generated")
	}

	return state, nil
}

// generateQuestionsNode generates clarification questions
func (cg *ClarificationGraph) generateQuestionsNode(ctx context.Context, state langgraph.State) (langgraph.State, error) {
	log.Info().Msg("Generating clarification questions")

	// Get ambiguities from state
	val, ok := state.Get("ambiguities")
	if !ok {
		return nil, fmt.Errorf("ambiguities not found in state")
	}

	ambiguities, ok := val.([]models.Ambiguity)
	if !ok {
		return nil, fmt.Errorf("invalid ambiguities type in state")
	}

	// Generate questions
	questions, err := cg.generator.Generate(ctx, ambiguities)
	if err != nil {
		return nil, fmt.Errorf("question generation failed: %w", err)
	}

	// Store questions in state
	state.Set("questions", questions)
	state.Set("question_count", len(questions))

	log.Info().
		Int("questions_generated", len(questions)).
		Msg("Clarification questions generated")

	return state, nil
}

// buildFCSNode builds the Final Clarified Specification
func (cg *ClarificationGraph) buildFCSNode(ctx context.Context, state langgraph.State) (langgraph.State, error) {
	log.Info().Msg("Building Final Clarified Specification")

	// Get spec from state
	specVal, ok := state.Get("spec")
	if !ok {
		return nil, fmt.Errorf("input specification not found in state")
	}

	spec, ok := specVal.(*models.InputSpecification)
	if !ok {
		return nil, fmt.Errorf("invalid spec type in state")
	}

	// Check if we have answers
	answersVal, hasAnswers := state.Get("answers")
	var answers map[string]models.Answer
	if hasAnswers {
		answers, _ = answersVal.(map[string]models.Answer)
	}

	// Build FCS (this is a simplified version - full implementation would
	// integrate answers into the FCS)
	fcs := buildFCSFromSpec(spec, answers)

	// Store FCS in state
	state.Set("fcs", fcs)
	state.Set("fcs_complete", true)

	log.Info().
		Str("fcs_id", fcs.ID).
		Msg("Final Clarified Specification built")

	return state, nil
}

// endNode completes the workflow
func (cg *ClarificationGraph) endNode(ctx context.Context, state langgraph.State) (langgraph.State, error) {
	log.Info().Msg("Clarification workflow completed")

	state.Set("workflow_completed", true)

	return state, nil
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
	if answers != nil {
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
	}

	// Compute hash
	hash, _ := fcs.ComputeHash()
	fcs.Metadata.Hash = hash

	return fcs
}

// Execute runs the clarification graph
func (cg *ClarificationGraph) Execute(ctx context.Context, spec *models.InputSpecification) (langgraph.State, error) {
	// Build the graph
	graph, err := cg.BuildGraph()
	if err != nil {
		return nil, fmt.Errorf("failed to build graph: %w", err)
	}

	// Create initial state
	initialState := langgraph.NewMapState()
	initialState.Set("spec", spec)

	// Execute the graph
	finalState, err := graph.Execute(ctx, initialState)
	if err != nil {
		return nil, fmt.Errorf("graph execution failed: %w", err)
	}

	return finalState, nil
}
