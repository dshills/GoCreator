package generate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/pkg/llm"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Planner creates generation plans from FCS
type Planner interface {
	// Plan creates a detailed generation plan from an FCS
	Plan(ctx context.Context, fcs *models.FinalClarifiedSpecification) (*models.GenerationPlan, error)
}

// llmPlanner implements Planner using an LLM to analyze the FCS and create a plan
type llmPlanner struct {
	client llm.Client
}

// PlannerConfig contains configuration for creating a planner
type PlannerConfig struct {
	LLMClient llm.Client
}

// NewPlanner creates a new Planner instance
func NewPlanner(cfg PlannerConfig) (Planner, error) {
	if cfg.LLMClient == nil {
		return nil, fmt.Errorf("LLM client is required")
	}

	return &llmPlanner{
		client: cfg.LLMClient,
	}, nil
}

// Plan creates a detailed generation plan from an FCS
func (p *llmPlanner) Plan(ctx context.Context, fcs *models.FinalClarifiedSpecification) (*models.GenerationPlan, error) {
	log.Info().
		Str("fcs_id", fcs.ID).
		Msg("Starting generation plan creation")

	startTime := time.Now()

	// Generate the plan using LLM
	plan, err := p.generatePlan(ctx, fcs)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	// Validate the plan
	if err := plan.Validate(); err != nil {
		return nil, fmt.Errorf("generated plan is invalid: %w", err)
	}

	duration := time.Since(startTime)
	log.Info().
		Str("plan_id", plan.ID).
		Str("fcs_id", fcs.ID).
		Int("phases", len(plan.Phases)).
		Int("files", len(plan.FileTree.Files)).
		Dur("duration", duration).
		Msg("Generation plan created successfully")

	return plan, nil
}

// generatePlan uses the LLM to analyze the FCS and create a generation plan
func (p *llmPlanner) generatePlan(ctx context.Context, fcs *models.FinalClarifiedSpecification) (*models.GenerationPlan, error) {
	log.Debug().
		Str("fcs_id", fcs.ID).
		Msg("Sending planning request to LLM")

	// Try to use prompt caching if the client supports it (Anthropic only)
	var response string
	var err error

	if cacheableClient, ok := p.client.(llm.CacheableClient); ok {
		// Client supports caching - use cached prompts
		log.Debug().
			Str("provider", p.client.Provider()).
			Str("fcs_id", fcs.ID).
			Msg("Using prompt caching for planning")

		messages := p.buildPlanningPromptWithCache(fcs)
		response, err = cacheableClient.GenerateWithCache(ctx, messages)
	} else {
		// Client doesn't support caching - use standard generation
		prompt := p.buildPlanningPrompt(fcs)
		log.Debug().
			Str("provider", p.client.Provider()).
			Str("fcs_id", fcs.ID).
			Int("prompt_length", len(prompt)).
			Msg("Client doesn't support caching, using standard generation")

		response, err = p.client.Generate(ctx, prompt)
	}

	if err != nil {
		return nil, fmt.Errorf("LLM planning request failed: %w", err)
	}

	log.Debug().
		Str("fcs_id", fcs.ID).
		Int("response_length", len(response)).
		Msg("Received planning response from LLM")

	// Parse the LLM response into a GenerationPlan
	plan, err := p.parsePlanResponse(response, fcs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan response: %w", err)
	}

	// Set plan metadata
	plan.ID = uuid.New().String()
	plan.FCSID = fcs.ID
	plan.SchemaVersion = "1.0"
	plan.CreatedAt = time.Now()

	return plan, nil
}

// buildPlanningPrompt constructs the LLM prompt for planning
func (p *llmPlanner) buildPlanningPrompt(fcs *models.FinalClarifiedSpecification) string {
	var sb strings.Builder

	sb.WriteString("You are an expert Go architect creating a detailed generation plan for a Go project.\n\n")
	sb.WriteString("# Task\n")
	sb.WriteString("Analyze the following Final Clarified Specification and create a comprehensive generation plan.\n\n")

	// Include FCS details
	sb.WriteString("# Final Clarified Specification\n\n")

	// Requirements
	sb.WriteString("## Requirements\n")
	sb.WriteString("### Functional Requirements\n")
	for _, req := range fcs.Requirements.Functional {
		sb.WriteString(fmt.Sprintf("- %s: %s (Priority: %s)\n", req.ID, req.Description, req.Priority))
	}
	sb.WriteString("\n")

	// Architecture
	sb.WriteString("## Architecture\n")
	sb.WriteString("### Packages\n")
	for _, pkg := range fcs.Architecture.Packages {
		sb.WriteString(fmt.Sprintf("- %s (%s): %s\n", pkg.Name, pkg.Path, pkg.Purpose))
		if len(pkg.Dependencies) > 0 {
			sb.WriteString(fmt.Sprintf("  Dependencies: %s\n", strings.Join(pkg.Dependencies, ", ")))
		}
	}
	sb.WriteString("\n")

	// Dependencies
	if len(fcs.Architecture.Dependencies) > 0 {
		sb.WriteString("### External Dependencies\n")
		for _, dep := range fcs.Architecture.Dependencies {
			sb.WriteString(fmt.Sprintf("- %s %s: %s\n", dep.Name, dep.Version, dep.Purpose))
		}
		sb.WriteString("\n")
	}

	// Data Model
	if len(fcs.DataModel.Entities) > 0 {
		sb.WriteString("## Data Model\n")
		for _, entity := range fcs.DataModel.Entities {
			sb.WriteString(fmt.Sprintf("- %s (package: %s)\n", entity.Name, entity.Package))
		}
		sb.WriteString("\n")
	}

	// Build Config
	sb.WriteString("## Build Configuration\n")
	sb.WriteString(fmt.Sprintf("- Go Version: %s\n", fcs.BuildConfig.GoVersion))
	sb.WriteString(fmt.Sprintf("- Output Path: %s\n", fcs.BuildConfig.OutputPath))
	sb.WriteString("\n")

	// Testing Strategy
	sb.WriteString("## Testing Strategy\n")
	sb.WriteString(fmt.Sprintf("- Coverage Target: %.1f%%\n", fcs.TestingStrategy.CoverageTarget))
	sb.WriteString(fmt.Sprintf("- Unit Tests: %t\n", fcs.TestingStrategy.UnitTests))
	sb.WriteString(fmt.Sprintf("- Integration Tests: %t\n", fcs.TestingStrategy.IntegrationTests))
	sb.WriteString("\n")

	// Instructions for the plan
	sb.WriteString("# Instructions\n\n")
	sb.WriteString("Create a detailed generation plan in JSON format with the following structure:\n\n")
	sb.WriteString("```json\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"file_tree\": {\n")
	sb.WriteString("    \"root\": \"./output\",\n")
	sb.WriteString("    \"directories\": [{\"path\": \"cmd/app\", \"purpose\": \"Main application entry\"}],\n")
	sb.WriteString("    \"files\": [{\"path\": \"main.go\", \"purpose\": \"Application entry point\", \"generated_by\": \"generate_main\"}]\n")
	sb.WriteString("  },\n")
	sb.WriteString("  \"phases\": [\n")
	sb.WriteString("    {\n")
	sb.WriteString("      \"name\": \"setup\",\n")
	sb.WriteString("      \"order\": 1,\n")
	sb.WriteString("      \"dependencies\": [],\n")
	sb.WriteString("      \"tasks\": [\n")
	sb.WriteString("        {\"id\": \"create_gomod\", \"type\": \"generate_file\", \"target_path\": \"go.mod\", \"can_parallel\": false}\n")
	sb.WriteString("      ]\n")
	sb.WriteString("    }\n")
	sb.WriteString("  ]\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")

	sb.WriteString("# Planning Guidelines\n\n")
	sb.WriteString("1. **Phase Organization**: Create phases in logical order:\n")
	sb.WriteString("   - Phase 1: Project setup (go.mod, directory structure, .gitignore)\n")
	sb.WriteString("   - Phase 2: Domain models and entities\n")
	sb.WriteString("   - Phase 3: Repository interfaces and implementations\n")
	sb.WriteString("   - Phase 4: Service layer and business logic\n")
	sb.WriteString("   - Phase 5: API handlers (if applicable)\n")
	sb.WriteString("   - Phase 6: Configuration files (Dockerfile, Makefile, etc.)\n")
	sb.WriteString("   - Phase 7: Tests for all packages\n")
	sb.WriteString("   - Phase 8: Documentation (README.md, API docs)\n\n")

	sb.WriteString("2. **File Tree**: Include ALL files and directories that will be generated\n\n")

	sb.WriteString("3. **Dependencies**: Ensure phases have correct dependencies (e.g., models before services)\n\n")

	sb.WriteString("4. **Parallelization**: Mark tasks as parallel only if they don't write to the same files\n\n")

	sb.WriteString("5. **Task Types**: Use these task types:\n")
	sb.WriteString("   - generate_file: Create a new source file\n")
	sb.WriteString("   - apply_patch: Modify an existing file\n")
	sb.WriteString("   - run_command: Execute a build/test command\n\n")

	sb.WriteString("6. **Template-based Files**: Mark these files with generated_by=\"template\" (they will be generated from templates, not LLM):\n")
	sb.WriteString("   - go.mod\n")
	sb.WriteString("   - .gitignore\n")
	sb.WriteString("   - Dockerfile\n")
	sb.WriteString("   - Makefile\n")
	sb.WriteString("   - README.md\n\n")

	sb.WriteString("Return ONLY the JSON plan, no additional text or explanation.\n")

	return sb.String()
}

// buildPlanningPromptWithCache constructs cacheable LLM prompts for planning
// This method separates static (cacheable) planning guidelines from dynamic (project-specific) FCS content
func (p *llmPlanner) buildPlanningPromptWithCache(fcs *models.FinalClarifiedSpecification) []llm.CacheableMessage {
	builder := llm.NewPromptBuilder("5m") // 5-minute cache TTL

	// CACHEABLE PART: Static planning guidelines and schema (same across all projects)
	var guidelines strings.Builder
	guidelines.WriteString("You are an expert Go architect creating a detailed generation plan for a Go project.\n\n")
	guidelines.WriteString("# Instructions\n\n")
	guidelines.WriteString("Create a detailed generation plan in JSON format with the following structure:\n\n")
	guidelines.WriteString("```json\n")
	guidelines.WriteString("{\n")
	guidelines.WriteString("  \"file_tree\": {\n")
	guidelines.WriteString("    \"root\": \"./output\",\n")
	guidelines.WriteString("    \"directories\": [{\"path\": \"cmd/app\", \"purpose\": \"Main application entry\"}],\n")
	guidelines.WriteString("    \"files\": [{\"path\": \"main.go\", \"purpose\": \"Application entry point\", \"generated_by\": \"generate_main\"}]\n")
	guidelines.WriteString("  },\n")
	guidelines.WriteString("  \"phases\": [\n")
	guidelines.WriteString("    {\n")
	guidelines.WriteString("      \"name\": \"setup\",\n")
	guidelines.WriteString("      \"order\": 1,\n")
	guidelines.WriteString("      \"dependencies\": [],\n")
	guidelines.WriteString("      \"tasks\": [\n")
	guidelines.WriteString("        {\"id\": \"create_gomod\", \"type\": \"generate_file\", \"target_path\": \"go.mod\", \"can_parallel\": false}\n")
	guidelines.WriteString("      ]\n")
	guidelines.WriteString("    }\n")
	guidelines.WriteString("  ]\n")
	guidelines.WriteString("}\n")
	guidelines.WriteString("```\n\n")
	guidelines.WriteString("# Planning Guidelines\n\n")
	guidelines.WriteString("1. **Phase Organization**: Create phases in logical order:\n")
	guidelines.WriteString("   - Phase 1: Project setup (go.mod, directory structure, .gitignore)\n")
	guidelines.WriteString("   - Phase 2: Domain models and entities\n")
	guidelines.WriteString("   - Phase 3: Repository interfaces and implementations\n")
	guidelines.WriteString("   - Phase 4: Service layer and business logic\n")
	guidelines.WriteString("   - Phase 5: API handlers (if applicable)\n")
	guidelines.WriteString("   - Phase 6: Configuration files (Dockerfile, Makefile, etc.)\n")
	guidelines.WriteString("   - Phase 7: Tests for all packages\n")
	guidelines.WriteString("   - Phase 8: Documentation (README.md, API docs)\n\n")
	guidelines.WriteString("2. **File Tree**: Include ALL files and directories that will be generated\n\n")
	guidelines.WriteString("3. **Dependencies**: Ensure phases have correct dependencies (e.g., models before services)\n\n")
	guidelines.WriteString("4. **Parallelization**: Mark tasks as parallel only if they don't write to the same files\n\n")
	guidelines.WriteString("5. **Task Types**: Use these task types:\n")
	guidelines.WriteString("   - generate_file: Create a new source file\n")
	guidelines.WriteString("   - apply_patch: Modify an existing file\n")
	guidelines.WriteString("   - run_command: Execute a build/test command\n\n")
	guidelines.WriteString("6. **Template-based Files**: Mark these files with generated_by=\"template\" (they will be generated from templates, not LLM):\n")
	guidelines.WriteString("   - go.mod\n")
	guidelines.WriteString("   - .gitignore\n")
	guidelines.WriteString("   - Dockerfile\n")
	guidelines.WriteString("   - Makefile\n")
	guidelines.WriteString("   - README.md\n\n")

	builder.AddCacheable(guidelines.String())

	// DYNAMIC PART: Project-specific FCS content (changes per project)
	var fcsContent strings.Builder
	fcsContent.WriteString("# Task\n")
	fcsContent.WriteString("Analyze the following Final Clarified Specification and create a comprehensive generation plan.\n\n")
	fcsContent.WriteString("# Final Clarified Specification\n\n")

	// Requirements
	fcsContent.WriteString("## Requirements\n")
	fcsContent.WriteString("### Functional Requirements\n")
	for _, req := range fcs.Requirements.Functional {
		fcsContent.WriteString(fmt.Sprintf("- %s: %s (Priority: %s)\n", req.ID, req.Description, req.Priority))
	}
	fcsContent.WriteString("\n")

	// Architecture
	fcsContent.WriteString("## Architecture\n")
	fcsContent.WriteString("### Packages\n")
	for _, pkg := range fcs.Architecture.Packages {
		fcsContent.WriteString(fmt.Sprintf("- %s (%s): %s\n", pkg.Name, pkg.Path, pkg.Purpose))
		if len(pkg.Dependencies) > 0 {
			fcsContent.WriteString(fmt.Sprintf("  Dependencies: %s\n", strings.Join(pkg.Dependencies, ", ")))
		}
	}
	fcsContent.WriteString("\n")

	// Dependencies
	if len(fcs.Architecture.Dependencies) > 0 {
		fcsContent.WriteString("### External Dependencies\n")
		for _, dep := range fcs.Architecture.Dependencies {
			fcsContent.WriteString(fmt.Sprintf("- %s %s: %s\n", dep.Name, dep.Version, dep.Purpose))
		}
		fcsContent.WriteString("\n")
	}

	// Data Model
	if len(fcs.DataModel.Entities) > 0 {
		fcsContent.WriteString("## Data Model\n")
		for _, entity := range fcs.DataModel.Entities {
			fcsContent.WriteString(fmt.Sprintf("- %s (package: %s)\n", entity.Name, entity.Package))
		}
		fcsContent.WriteString("\n")
	}

	// Build Config
	fcsContent.WriteString("## Build Configuration\n")
	fcsContent.WriteString(fmt.Sprintf("- Go Version: %s\n", fcs.BuildConfig.GoVersion))
	fcsContent.WriteString(fmt.Sprintf("- Output Path: %s\n", fcs.BuildConfig.OutputPath))
	fcsContent.WriteString("\n")

	// Testing Strategy
	fcsContent.WriteString("## Testing Strategy\n")
	fcsContent.WriteString(fmt.Sprintf("- Coverage Target: %.1f%%\n", fcs.TestingStrategy.CoverageTarget))
	fcsContent.WriteString(fmt.Sprintf("- Unit Tests: %t\n", fcs.TestingStrategy.UnitTests))
	fcsContent.WriteString(fmt.Sprintf("- Integration Tests: %t\n", fcs.TestingStrategy.IntegrationTests))
	fcsContent.WriteString("\n")

	fcsContent.WriteString("Return ONLY the JSON plan, no additional text or explanation.\n")

	builder.AddDynamic(fcsContent.String())

	return builder.Build()
}

// parsePlanResponse parses the LLM response into a GenerationPlan
func (p *llmPlanner) parsePlanResponse(response string, _ *models.FinalClarifiedSpecification) (*models.GenerationPlan, error) {
	// Clean the response (remove markdown code blocks if present)
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// Parse JSON into a temporary structure
	var planData struct {
		FileTree struct {
			Root        string `json:"root"`
			Directories []struct {
				Path    string `json:"path"`
				Purpose string `json:"purpose"`
			} `json:"directories"`
			Files []struct {
				Path        string `json:"path"`
				Purpose     string `json:"purpose"`
				GeneratedBy string `json:"generated_by"`
			} `json:"files"`
		} `json:"file_tree"`
		Phases []struct {
			Name         string   `json:"name"`
			Order        int      `json:"order"`
			Dependencies []string `json:"dependencies"`
			Tasks        []struct {
				ID          string                 `json:"id"`
				Type        string                 `json:"type"`
				TargetPath  string                 `json:"target_path"`
				CanParallel bool                   `json:"can_parallel"`
				Inputs      map[string]interface{} `json:"inputs"`
			} `json:"tasks"`
		} `json:"phases"`
	}

	if err := json.Unmarshal([]byte(response), &planData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Convert to GenerationPlan
	plan := &models.GenerationPlan{
		FileTree: models.FileTree{
			Root:        planData.FileTree.Root,
			Directories: make([]models.Directory, len(planData.FileTree.Directories)),
			Files:       make([]models.File, len(planData.FileTree.Files)),
		},
		Phases: make([]models.GenerationPhase, len(planData.Phases)),
	}

	// Convert directories
	for i, dir := range planData.FileTree.Directories {
		plan.FileTree.Directories[i] = models.Directory{
			Path:    dir.Path,
			Purpose: dir.Purpose,
		}
	}

	// Convert files
	for i, file := range planData.FileTree.Files {
		plan.FileTree.Files[i] = models.File{
			Path:        file.Path,
			Purpose:     file.Purpose,
			GeneratedBy: file.GeneratedBy,
		}
	}

	// Convert phases
	for i, phase := range planData.Phases {
		tasks := make([]models.GenerationTask, len(phase.Tasks))
		for j, task := range phase.Tasks {
			// Keep target path as-is (should be relative to root)
			// FileOps will handle joining with the configured root directory
			tasks[j] = models.GenerationTask{
				ID:          task.ID,
				Type:        task.Type,
				TargetPath:  task.TargetPath,
				Inputs:      task.Inputs,
				CanParallel: task.CanParallel,
			}
		}

		plan.Phases[i] = models.GenerationPhase{
			Name:         phase.Name,
			Order:        phase.Order,
			Tasks:        tasks,
			Dependencies: phase.Dependencies,
		}
	}

	return plan, nil
}
