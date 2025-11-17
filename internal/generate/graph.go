package generate

import (
	"context"
	"fmt"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/pkg/langgraph"
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

// GenerationGraph creates the LangGraph-Go workflow for code generation
type GenerationGraph struct {
	graph   *langgraph.Graph
	planner Planner
	coder   Coder
	tester  Tester
}

// GenerationGraphConfig contains configuration for the generation graph
type GenerationGraphConfig struct {
	Planner             Planner
	Coder               Coder
	Tester              Tester
	CheckpointManager   langgraph.CheckpointManager
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

	// Create the graph with optional checkpointing
	var opts []langgraph.GraphOption
	if cfg.EnableCheckpointing && cfg.CheckpointManager != nil {
		opts = append(opts, langgraph.WithCheckpointing(cfg.CheckpointManager))
	}

	graph := langgraph.NewGraph("generation", "start", "end", opts...)

	// Build the workflow nodes
	if err := gg.buildGraph(graph); err != nil {
		return nil, fmt.Errorf("failed to build generation graph: %w", err)
	}

	gg.graph = graph

	return gg, nil
}

// buildGraph constructs the generation workflow nodes
func (gg *GenerationGraph) buildGraph(graph *langgraph.Graph) error {
	// Node 1: Start - Initialize state
	startNode := langgraph.NewBasicNode(
		"start",
		gg.startNode,
		[]string{},
		"Initialize generation state",
	)
	if err := graph.AddNode(startNode); err != nil {
		return err
	}

	// Node 2: Analyze FCS - Validate and prepare FCS
	analyzeFCSNode := langgraph.NewBasicNode(
		"analyze_fcs",
		gg.analyzeFCSNode,
		[]string{"start"},
		"Analyze and validate FCS",
	)
	if err := graph.AddNode(analyzeFCSNode); err != nil {
		return err
	}

	// Node 3: Create Plan - Generate architectural plan
	createPlanNode := langgraph.NewBasicNode(
		"create_plan",
		gg.createPlanNode,
		[]string{"analyze_fcs"},
		"Create generation plan from FCS",
	)
	if err := graph.AddNode(createPlanNode); err != nil {
		return err
	}

	// Node 4: Generate Packages - Generate source code
	generatePackagesNode := langgraph.NewBasicNode(
		"generate_packages",
		gg.generatePackagesNode,
		[]string{"create_plan"},
		"Generate source code for all packages",
	)
	if err := graph.AddNode(generatePackagesNode); err != nil {
		return err
	}

	// Node 5: Generate Tests - Generate test files
	generateTestsNode := langgraph.NewBasicNode(
		"generate_tests",
		gg.generateTestsNode,
		[]string{"generate_packages"},
		"Generate test files for all packages",
	)
	if err := graph.AddNode(generateTestsNode); err != nil {
		return err
	}

	// Node 6: Generate Config - Generate configuration files
	generateConfigNode := langgraph.NewBasicNode(
		"generate_config",
		gg.generateConfigNode,
		[]string{"generate_tests"},
		"Generate configuration and build files",
	)
	if err := graph.AddNode(generateConfigNode); err != nil {
		return err
	}

	// Node 7: Apply Patches - Collect and prepare patches
	applyPatchesNode := langgraph.NewBasicNode(
		"apply_patches",
		gg.applyPatchesNode,
		[]string{"generate_config"},
		"Collect and prepare all patches for application",
	)
	if err := graph.AddNode(applyPatchesNode); err != nil {
		return err
	}

	// Node 8: End - Finalize output
	endNode := langgraph.NewBasicNode(
		"end",
		gg.endNode,
		[]string{"apply_patches"},
		"Finalize generation output",
	)
	if err := graph.AddNode(endNode); err != nil {
		return err
	}

	return nil
}

// Execute runs the generation workflow
func (gg *GenerationGraph) Execute(ctx context.Context, fcs *models.FinalClarifiedSpecification, outputDir string) (*models.GenerationOutput, error) {
	// Create initial state
	initialState := langgraph.NewMapState()
	initialState.Set("fcs", fcs)
	initialState.Set("output_dir", outputDir)
	initialState.Set("code_patches", []models.Patch{})
	initialState.Set("test_patches", []models.Patch{})
	initialState.Set("config_patches", []models.Patch{})
	initialState.Set("all_patches", []models.Patch{})
	initialState.Set("completed_phases", []string{})

	log.Info().
		Str("fcs_id", fcs.ID).
		Str("output_dir", outputDir).
		Msg("Starting generation workflow execution")

	// Execute the graph
	finalState, err := gg.graph.Execute(ctx, initialState)
	if err != nil {
		return nil, fmt.Errorf("generation workflow failed: %w", err)
	}

	// Extract patches from final state
	allPatchesVal, ok := finalState.Get("all_patches")
	if !ok {
		return nil, fmt.Errorf("no patches generated")
	}

	allPatches, ok := allPatchesVal.([]models.Patch)
	if !ok {
		return nil, fmt.Errorf("invalid patches type")
	}

	planVal, _ := finalState.Get("plan")
	plan := planVal.(*models.GenerationPlan)

	// Create output structure with patches
	output := &models.GenerationOutput{
		SchemaVersion: "1.0",
		ID:            uuid.New().String(),
		PlanID:        plan.ID,
		Patches:       allPatches,
		Status:        models.OutputStatusInProgress,
	}

	log.Info().
		Str("output_id", output.ID).
		Int("patches", len(allPatches)).
		Msg("Generation workflow completed successfully")

	return output, nil
}

// Node implementations

func (gg *GenerationGraph) startNode(_ context.Context, state langgraph.State) (langgraph.State, error) {
	log.Debug().Msg("Starting generation workflow")
	state.Set("current_phase", "start")
	return state, nil
}

func (gg *GenerationGraph) analyzeFCSNode(_ context.Context, state langgraph.State) (langgraph.State, error) {
	log.Debug().Msg("Analyzing FCS")
	state.Set("current_phase", "analyze_fcs")

	fcsVal, ok := state.Get("fcs")
	if !ok {
		return nil, fmt.Errorf("FCS not found in state")
	}

	fcs, ok := fcsVal.(*models.FinalClarifiedSpecification)
	if !ok {
		return nil, fmt.Errorf("invalid FCS type")
	}

	// Validate FCS
	if err := fcs.Validate(); err != nil {
		return nil, fmt.Errorf("FCS validation failed: %w", err)
	}

	log.Debug().
		Str("fcs_id", fcs.ID).
		Int("packages", len(fcs.Architecture.Packages)).
		Msg("FCS validated successfully")

	// Mark phase as completed
	gg.markPhaseCompleted(state, "analyze_fcs")

	return state, nil
}

func (gg *GenerationGraph) createPlanNode(ctx context.Context, state langgraph.State) (langgraph.State, error) {
	log.Debug().Msg("Creating generation plan")
	state.Set("current_phase", "create_plan")

	fcsVal, _ := state.Get("fcs")
	fcs := fcsVal.(*models.FinalClarifiedSpecification)

	// Create generation plan using planner
	plan, err := gg.planner.Plan(ctx, fcs)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	// Store plan in state
	state.Set("plan", plan)

	log.Debug().
		Str("plan_id", plan.ID).
		Int("phases", len(plan.Phases)).
		Msg("Generation plan created")

	// Extract package list
	packageList := make([]string, len(fcs.Architecture.Packages))
	for i, pkg := range fcs.Architecture.Packages {
		packageList[i] = pkg.Name
	}
	state.Set("package_list", packageList)

	// Mark phase as completed
	gg.markPhaseCompleted(state, "create_plan")

	return state, nil
}

func (gg *GenerationGraph) generatePackagesNode(ctx context.Context, state langgraph.State) (langgraph.State, error) {
	log.Debug().Msg("Generating source code packages")
	state.Set("current_phase", "generate_packages")

	planVal, _ := state.Get("plan")
	plan := planVal.(*models.GenerationPlan)

	// Generate code using coder
	patches, err := gg.coder.Generate(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("failed to generate code: %w", err)
	}

	// Store patches in state
	state.Set("code_patches", patches)

	log.Debug().
		Int("patches", len(patches)).
		Msg("Code generation completed")

	// Mark phase as completed
	gg.markPhaseCompleted(state, "generate_packages")

	return state, nil
}

func (gg *GenerationGraph) generateTestsNode(ctx context.Context, state langgraph.State) (langgraph.State, error) {
	log.Debug().Msg("Generating test files")
	state.Set("current_phase", "generate_tests")

	planVal, _ := state.Get("plan")
	plan := planVal.(*models.GenerationPlan)

	packageListVal, _ := state.Get("package_list")
	packageList := packageListVal.([]string)

	// Generate tests using tester
	patches, err := gg.tester.Generate(ctx, packageList, plan)
	if err != nil {
		// Log error but don't fail - tests are important but not critical
		log.Warn().
			Err(err).
			Msg("Failed to generate some test files")
		patches = []models.Patch{}
	}

	// Store patches in state
	state.Set("test_patches", patches)

	log.Debug().
		Int("patches", len(patches)).
		Msg("Test generation completed")

	// Mark phase as completed
	gg.markPhaseCompleted(state, "generate_tests")

	return state, nil
}

func (gg *GenerationGraph) generateConfigNode(_ context.Context, state langgraph.State) (langgraph.State, error) {
	log.Debug().Msg("Generating configuration files")
	state.Set("current_phase", "generate_config")

	// For now, config generation is handled in the code generation phase
	// This node is a placeholder for future config file generation
	state.Set("config_patches", []models.Patch{})

	log.Debug().Msg("Configuration generation completed")

	// Mark phase as completed
	gg.markPhaseCompleted(state, "generate_config")

	return state, nil
}

func (gg *GenerationGraph) applyPatchesNode(_ context.Context, state langgraph.State) (langgraph.State, error) {
	log.Debug().Msg("Collecting patches for application")
	state.Set("current_phase", "apply_patches")

	// Collect all patches
	codePatchesVal, _ := state.Get("code_patches")
	testPatchesVal, _ := state.Get("test_patches")
	configPatchesVal, _ := state.Get("config_patches")

	codePatches := codePatchesVal.([]models.Patch)
	testPatches := testPatchesVal.([]models.Patch)
	configPatches := configPatchesVal.([]models.Patch)

	allPatches := append([]models.Patch{}, codePatches...)
	allPatches = append(allPatches, testPatches...)
	allPatches = append(allPatches, configPatches...)

	state.Set("all_patches", allPatches)

	log.Debug().
		Int("code_patches", len(codePatches)).
		Int("test_patches", len(testPatches)).
		Int("config_patches", len(configPatches)).
		Int("total_patches", len(allPatches)).
		Msg("Patches collected for application")

	// Mark phase as completed
	gg.markPhaseCompleted(state, "apply_patches")

	return state, nil
}

func (gg *GenerationGraph) endNode(_ context.Context, state langgraph.State) (langgraph.State, error) {
	log.Debug().Msg("Finalizing generation output")
	state.Set("current_phase", "end")

	// This node is handled by the engine which will apply patches
	// and create the final GenerationOutput
	// For now, just mark as completed
	gg.markPhaseCompleted(state, "end")

	return state, nil
}

// markPhaseCompleted marks a phase as completed in the state
func (gg *GenerationGraph) markPhaseCompleted(state langgraph.State, phase string) {
	completedVal, ok := state.Get("completed_phases")
	if !ok {
		completedVal = []string{}
	}

	completed := completedVal.([]string)
	completed = append(completed, phase)
	state.Set("completed_phases", completed)
}
