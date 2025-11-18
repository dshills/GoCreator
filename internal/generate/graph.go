package generate

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/langgraph-go/graph"
	"github.com/dshills/langgraph-go/graph/emit"
	"github.com/dshills/langgraph-go/graph/store"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// GenerationState represents the state of the generation workflow
type GenerationState struct {
	FCS             *models.FinalClarifiedSpecification
	Plan            *models.GenerationPlan
	CodePatches     []models.Patch
	TestPatches     []models.Patch
	ConfigPatches   []models.Patch
	AllPatches      []models.Patch
	Output          *models.GenerationOutput
	Error           error
	OutputDir       string
	PackageList     []string
	CurrentPhase    string
	CompletedPhases []string
}

// reduceGenerationState merges state updates
func reduceGenerationState(prev, delta GenerationState) GenerationState {
	// Apply delta fields to prev
	if delta.FCS != nil {
		prev.FCS = delta.FCS
	}
	if delta.Plan != nil {
		prev.Plan = delta.Plan
	}
	if delta.CodePatches != nil {
		prev.CodePatches = delta.CodePatches
	}
	if delta.TestPatches != nil {
		prev.TestPatches = delta.TestPatches
	}
	if delta.ConfigPatches != nil {
		prev.ConfigPatches = delta.ConfigPatches
	}
	if delta.AllPatches != nil {
		prev.AllPatches = delta.AllPatches
	}
	if delta.Output != nil {
		prev.Output = delta.Output
	}
	if delta.Error != nil {
		prev.Error = delta.Error
	}
	if delta.OutputDir != "" {
		prev.OutputDir = delta.OutputDir
	}
	if delta.PackageList != nil {
		prev.PackageList = delta.PackageList
	}
	if delta.CurrentPhase != "" {
		prev.CurrentPhase = delta.CurrentPhase
	}
	// Append CompletedPhases with deduplication
	if delta.CompletedPhases != nil {
		// Create a map to track existing phases
		existing := make(map[string]bool)
		for _, phase := range prev.CompletedPhases {
			existing[phase] = true
		}
		// Add only new phases
		for _, phase := range delta.CompletedPhases {
			if !existing[phase] {
				prev.CompletedPhases = append(prev.CompletedPhases, phase)
			}
		}
	}

	return prev
}

// GenerationGraph creates the LangGraph-Go workflow for code generation
type GenerationGraph struct {
	engine  *graph.Engine[GenerationState]
	planner Planner
	coder   Coder
	tester  Tester
}

// GenerationGraphConfig contains configuration for the generation graph
type GenerationGraphConfig struct {
	Planner             Planner
	Coder               Coder
	Tester              Tester
	EnableCheckpointing bool
}

// NewGenerationGraph creates a new generation workflow graph
func NewGenerationGraph(cfg GenerationGraphConfig) (*GenerationGraph, error) {
	if cfg.Planner == nil {
		return nil, fmt.Errorf("planner is required")
	}
	if cfg.Coder == nil {
		return nil, fmt.Errorf("coder is required")
	}
	if cfg.Tester == nil {
		return nil, fmt.Errorf("tester is required")
	}

	gg := &GenerationGraph{
		planner: cfg.Planner,
		coder:   cfg.Coder,
		tester:  cfg.Tester,
	}

	// Create store and emitter
	st := store.NewMemStore[GenerationState]()
	emitter := emit.NewLogEmitter(os.Stdout, false)

	// Create engine with options
	// NOTE: Using sequential execution (no WithMaxConcurrent) because concurrent execution
	// in langgraph-go v0.3.0-alpha has a bug where deltas are not merged between nodes
	engine := graph.New(
		reduceGenerationState,
		st,
		emitter,
		graph.WithDefaultNodeTimeout(10*time.Minute),
	)

	// Build the workflow nodes
	if err := gg.buildGraph(engine); err != nil {
		return nil, fmt.Errorf("failed to build generation graph: %w", err)
	}

	gg.engine = engine

	return gg, nil
}

// buildGraph constructs the generation workflow nodes
func (gg *GenerationGraph) buildGraph(engine *graph.Engine[GenerationState]) error {
	// Node 1: Start - Initialize state
	if err := engine.Add("start", graph.NodeFunc[GenerationState](gg.startNode)); err != nil {
		return fmt.Errorf("failed to add start node: %w", err)
	}

	// Node 2: Analyze FCS - Validate and prepare FCS
	if err := engine.Add("analyze_fcs", graph.NodeFunc[GenerationState](gg.analyzeFCSNode)); err != nil {
		return fmt.Errorf("failed to add analyze_fcs node: %w", err)
	}

	// Node 3: Create Plan - Generate architectural plan
	if err := engine.Add("create_plan", graph.NodeFunc[GenerationState](gg.createPlanNode)); err != nil {
		return fmt.Errorf("failed to add create_plan node: %w", err)
	}

	// Node 4: Generate Packages - Generate source code
	if err := engine.Add("generate_packages", graph.NodeFunc[GenerationState](gg.generatePackagesNode)); err != nil {
		return fmt.Errorf("failed to add generate_packages node: %w", err)
	}

	// Node 5: Generate Tests - Generate test files
	if err := engine.Add("generate_tests", graph.NodeFunc[GenerationState](gg.generateTestsNode)); err != nil {
		return fmt.Errorf("failed to add generate_tests node: %w", err)
	}

	// Node 6: Generate Config - Generate configuration files
	if err := engine.Add("generate_config", graph.NodeFunc[GenerationState](gg.generateConfigNode)); err != nil {
		return fmt.Errorf("failed to add generate_config node: %w", err)
	}

	// Node 7: Apply Patches - Collect and prepare patches
	if err := engine.Add("apply_patches", graph.NodeFunc[GenerationState](gg.applyPatchesNode)); err != nil {
		return fmt.Errorf("failed to add apply_patches node: %w", err)
	}

	// Node 8: End - Finalize output
	if err := engine.Add("end", graph.NodeFunc[GenerationState](gg.endNode)); err != nil {
		return fmt.Errorf("failed to add end node: %w", err)
	}

	// Set start node
	if err := engine.StartAt("start"); err != nil {
		return fmt.Errorf("failed to set start node: %w", err)
	}

	return nil
}

// Execute runs the generation workflow
func (gg *GenerationGraph) Execute(ctx context.Context, fcs *models.FinalClarifiedSpecification, outputDir string) (*models.GenerationOutput, error) {
	// Create initial state
	// NOTE: All fields must be explicitly initialized for proper state tracking
	initialState := GenerationState{
		FCS:             fcs,
		Plan:            nil,
		CodePatches:     nil,
		TestPatches:     nil,
		ConfigPatches:   nil,
		AllPatches:      nil,
		Output:          nil,
		Error:           nil,
		OutputDir:       outputDir,
		PackageList:     nil,
		CurrentPhase:    "",
		CompletedPhases: nil,
	}

	log.Info().
		Str("fcs_id", fcs.ID).
		Str("output_dir", outputDir).
		Msg("Starting generation workflow execution")

	// Execute the graph
	executionID := fmt.Sprintf("gen-%s", uuid.New().String())
	finalState, err := gg.engine.Run(ctx, executionID, initialState)
	if err != nil {
		return nil, fmt.Errorf("generation workflow failed: %w", err)
	}

	// Check for errors in final state
	if finalState.Error != nil {
		return nil, finalState.Error
	}

	// Validate required state
	if finalState.Plan == nil {
		return nil, fmt.Errorf("generation plan not created")
	}
	if finalState.AllPatches == nil {
		finalState.AllPatches = []models.Patch{}
	}

	// Create output structure with patches
	output := &models.GenerationOutput{
		SchemaVersion: "1.0",
		ID:            uuid.New().String(),
		PlanID:        finalState.Plan.ID,
		Patches:       finalState.AllPatches,
		Status:        models.OutputStatusInProgress,
	}

	log.Info().
		Str("output_id", output.ID).
		Int("patches", len(finalState.AllPatches)).
		Msg("Generation workflow completed successfully")

	return output, nil
}

// Node implementations

func (gg *GenerationGraph) startNode(_ context.Context, _ GenerationState) graph.NodeResult[GenerationState] {
	log.Debug().Msg("Starting generation workflow")

	return graph.NodeResult[GenerationState]{
		Delta: GenerationState{
			CurrentPhase:    "start",
			CompletedPhases: []string{},
		},
		Route: graph.Goto("analyze_fcs"),
	}
}

func (gg *GenerationGraph) analyzeFCSNode(_ context.Context, s GenerationState) graph.NodeResult[GenerationState] {
	log.Debug().Msg("Analyzing FCS")

	// Validate FCS
	if s.FCS == nil {
		return graph.NodeResult[GenerationState]{
			Delta: GenerationState{
				Error: fmt.Errorf("FCS not found in state"),
			},
			Route: graph.Stop(),
		}
	}

	if err := s.FCS.Validate(); err != nil {
		return graph.NodeResult[GenerationState]{
			Delta: GenerationState{
				Error: fmt.Errorf("FCS validation failed: %w", err),
			},
			Route: graph.Stop(),
		}
	}

	log.Debug().
		Str("fcs_id", s.FCS.ID).
		Int("packages", len(s.FCS.Architecture.Packages)).
		Msg("FCS validated successfully")

	return graph.NodeResult[GenerationState]{
		Delta: GenerationState{
			CurrentPhase:    "analyze_fcs",
			CompletedPhases: []string{"analyze_fcs"},
		},
		Route: graph.Goto("create_plan"),
	}
}

func (gg *GenerationGraph) createPlanNode(ctx context.Context, s GenerationState) graph.NodeResult[GenerationState] {
	log.Debug().Msg("Creating generation plan")

	// Create generation plan using planner
	plan, err := gg.planner.Plan(ctx, s.FCS)
	if err != nil {
		return graph.NodeResult[GenerationState]{
			Delta: GenerationState{
				Error: fmt.Errorf("failed to create plan: %w", err),
			},
			Route: graph.Stop(),
		}
	}

	log.Debug().
		Str("plan_id", plan.ID).
		Int("phases", len(plan.Phases)).
		Msg("Generation plan created")

	// Extract package list
	packageList := make([]string, len(s.FCS.Architecture.Packages))
	for i, pkg := range s.FCS.Architecture.Packages {
		packageList[i] = pkg.Name
	}

	return graph.NodeResult[GenerationState]{
		Delta: GenerationState{
			Plan:            plan,
			PackageList:     packageList,
			CurrentPhase:    "create_plan",
			CompletedPhases: []string{"create_plan"},
		},
		Route: graph.Goto("generate_packages"),
	}
}

func (gg *GenerationGraph) generatePackagesNode(ctx context.Context, s GenerationState) graph.NodeResult[GenerationState] {
	log.Debug().
		Bool("plan_is_nil", s.Plan == nil).
		Str("current_phase", s.CurrentPhase).
		Int("completed_phases", len(s.CompletedPhases)).
		Msg("Generating source code packages")

	// Validate plan exists
	if s.Plan == nil {
		log.Error().
			Str("current_phase", s.CurrentPhase).
			Strs("completed_phases", s.CompletedPhases).
			Msg("Plan is nil in generatePackagesNode - state was not properly accumulated")
		return graph.NodeResult[GenerationState]{
			Delta: GenerationState{
				Error: fmt.Errorf("generation plan not found in state"),
			},
			Route: graph.Stop(),
		}
	}

	// Generate code using coder
	patches, err := gg.coder.Generate(ctx, s.Plan)
	if err != nil {
		return graph.NodeResult[GenerationState]{
			Delta: GenerationState{
				Error: fmt.Errorf("failed to generate code: %w", err),
			},
			Route: graph.Stop(),
		}
	}

	log.Debug().
		Int("patches", len(patches)).
		Msg("Code generation completed")

	return graph.NodeResult[GenerationState]{
		Delta: GenerationState{
			CodePatches:     patches,
			CurrentPhase:    "generate_packages",
			CompletedPhases: []string{"generate_packages"},
		},
		Route: graph.Goto("generate_tests"),
	}
}

func (gg *GenerationGraph) generateTestsNode(ctx context.Context, s GenerationState) graph.NodeResult[GenerationState] {
	log.Debug().Msg("Generating test files")

	var patches []models.Patch
	// Validate plan exists before generating tests
	if s.Plan == nil {
		log.Warn().Msg("Generation plan not found, skipping test generation")
		patches = []models.Patch{}
	} else {
		// Generate tests using tester
		var err error
		patches, err = gg.tester.Generate(ctx, s.PackageList, s.Plan)
		if err != nil {
			// Log error but don't fail - tests are important but not critical
			log.Warn().
				Err(err).
				Msg("Failed to generate some test files")
			patches = []models.Patch{}
		}
	}

	log.Debug().
		Int("patches", len(patches)).
		Msg("Test generation completed")

	return graph.NodeResult[GenerationState]{
		Delta: GenerationState{
			TestPatches:     patches,
			CurrentPhase:    "generate_tests",
			CompletedPhases: []string{"generate_tests"},
		},
		Route: graph.Goto("generate_config"),
	}
}

func (gg *GenerationGraph) generateConfigNode(_ context.Context, _ GenerationState) graph.NodeResult[GenerationState] {
	log.Debug().Msg("Generating configuration files")

	// For now, config generation is handled in the code generation phase
	// This node is a placeholder for future config file generation

	log.Debug().Msg("Configuration generation completed")

	return graph.NodeResult[GenerationState]{
		Delta: GenerationState{
			ConfigPatches:   []models.Patch{},
			CurrentPhase:    "generate_config",
			CompletedPhases: []string{"generate_config"},
		},
		Route: graph.Goto("apply_patches"),
	}
}

func (gg *GenerationGraph) applyPatchesNode(_ context.Context, s GenerationState) graph.NodeResult[GenerationState] {
	log.Debug().Msg("Collecting patches for application")

	// Collect all patches
	allPatches := append([]models.Patch{}, s.CodePatches...)
	allPatches = append(allPatches, s.TestPatches...)
	allPatches = append(allPatches, s.ConfigPatches...)

	log.Debug().
		Int("code_patches", len(s.CodePatches)).
		Int("test_patches", len(s.TestPatches)).
		Int("config_patches", len(s.ConfigPatches)).
		Int("total_patches", len(allPatches)).
		Msg("Patches collected for application")

	return graph.NodeResult[GenerationState]{
		Delta: GenerationState{
			AllPatches:      allPatches,
			CurrentPhase:    "apply_patches",
			CompletedPhases: []string{"apply_patches"},
		},
		Route: graph.Goto("end"),
	}
}

func (gg *GenerationGraph) endNode(_ context.Context, _ GenerationState) graph.NodeResult[GenerationState] {
	log.Debug().Msg("Finalizing generation output")

	return graph.NodeResult[GenerationState]{
		Delta: GenerationState{
			CurrentPhase:    "end",
			CompletedPhases: []string{"end"},
		},
		Route: graph.Stop(),
	}
}
