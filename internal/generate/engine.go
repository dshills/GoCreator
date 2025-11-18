package generate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dshills/gocreator/internal/generate/templates"
	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/pkg/fsops"
	"github.com/dshills/gocreator/pkg/llm"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Engine orchestrates the entire code generation process
type Engine interface {
	// Generate creates a complete Go project from an FCS
	Generate(ctx context.Context, fcs *models.FinalClarifiedSpecification, outputDir string) (*models.GenerationOutput, error)
}

// engine implements the Engine interface
type engine struct {
	graph        *GenerationGraph
	fileOps      fsops.FileOps
	logDecisions bool
	eventChan    chan<- models.ProgressEvent
}

// EngineConfig contains configuration for the generation engine
type EngineConfig struct {
	LLMClient    llm.Client
	FileOps      fsops.FileOps
	LogDecisions bool
	EventChan    chan<- models.ProgressEvent
}

// NewEngine creates a new generation engine
func NewEngine(cfg EngineConfig) (Engine, error) {
	if cfg.LLMClient == nil {
		return nil, fmt.Errorf("LLM client is required")
	}
	if cfg.FileOps == nil {
		return nil, fmt.Errorf("file operations handler is required")
	}

	// Create planner
	planner, err := NewPlanner(PlannerConfig{
		LLMClient: cfg.LLMClient,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create planner: %w", err)
	}

	// Create coder
	coder, err := NewCoder(CoderConfig{
		LLMClient: cfg.LLMClient,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create coder: %w", err)
	}

	// Create tester
	tester, err := NewTester(TesterConfig{
		LLMClient: cfg.LLMClient,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tester: %w", err)
	}

	// Create template generator
	templateGen, err := templates.NewTemplateGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to create template generator: %w", err)
	}

	// Create generation graph
	graph, err := NewGenerationGraph(GenerationGraphConfig{
		Planner:           planner,
		Coder:             coder,
		Tester:            tester,
		TemplateGenerator: templateGen,
		EventChan:         cfg.EventChan,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create generation graph: %w", err)
	}

	return &engine{
		graph:        graph,
		fileOps:      cfg.FileOps,
		logDecisions: cfg.LogDecisions,
		eventChan:    cfg.EventChan,
	}, nil
}
// Generate creates a complete Go project from an FCS
func (e *engine) Generate(ctx context.Context, fcs *models.FinalClarifiedSpecification, outputDir string) (*models.GenerationOutput, error) {
	log.Info().
		Str("fcs_id", fcs.ID).
		Str("output_dir", outputDir).
		Msg("Starting autonomous code generation")

	startTime := time.Now()

	// Emit start event
	e.emitEvent(models.NewPhaseStartedEvent("initialization", "Preparing generation workflow"))

	// Create output structure
	output := &models.GenerationOutput{
		SchemaVersion: "1.0",
		ID:            uuid.New().String(),
		Status:        models.OutputStatusPending,
		Metadata: models.OutputMetadata{
			StartedAt: startTime,
		},
	}

	// Transition to in-progress
	if err := output.TransitionTo(models.OutputStatusInProgress); err != nil {
		return nil, fmt.Errorf("failed to transition output status: %w", err)
	}

	// Log decision if enabled
	if e.logDecisions {
		e.logDecision(ctx, "starting_generation", "Beginning autonomous code generation from FCS", map[string]interface{}{
			"fcs_id":     fcs.ID,
			"output_dir": outputDir,
			"output_id":  output.ID,
			"go_version": fcs.BuildConfig.GoVersion,
			"packages":   len(fcs.Architecture.Packages),
		})
	}

	// Execute the generation workflow
	workflowOutput, err := e.graph.Execute(ctx, fcs, outputDir)
	if err != nil {
		output.Status = models.OutputStatusFailed
		e.logDecision(ctx, "generation_failed", "Code generation workflow failed", map[string]interface{}{
			"error": err.Error(),
		})
		return output, fmt.Errorf("generation workflow failed: %w", err)
	}

	// Validate workflow output
	if workflowOutput == nil {
		output.Status = models.OutputStatusFailed
		return output, fmt.Errorf("workflow returned nil output")
	}

	// Get patches from workflow output
	// In a real implementation, the graph would pass patches through state
	// For now, we'll extract them from the workflow output
	patches := workflowOutput.Patches

	// Apply all patches using file operations
	if err := e.applyPatches(ctx, patches, output); err != nil {
		output.Status = models.OutputStatusFailed
		e.logDecision(ctx, "patch_application_failed", "Failed to apply generated patches", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to apply patches: %w", err)
	}

	// Calculate metadata
	output.Metadata.FilesCount = len(output.Files)
	output.Metadata.LinesCount = e.countTotalLines(output.Files)
	output.Metadata.Duration = time.Since(startTime)
	now := time.Now()
	output.Metadata.CompletedAt = &now

	// Validate output
	if err := output.Validate(); err != nil {
		output.Status = models.OutputStatusFailed
		e.logDecision(ctx, "validation_failed", "Generated output validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("output validation failed: %w", err)
	}

	// Transition to completed
	if err := output.TransitionTo(models.OutputStatusCompleted); err != nil {
		return nil, fmt.Errorf("failed to transition to completed status: %w", err)
	}

	// Log successful completion
	if e.logDecisions {
		e.logDecision(ctx, "generation_completed", "Code generation completed successfully", map[string]interface{}{
			"output_id":   output.ID,
			"files":       len(output.Files),
			"lines":       output.Metadata.LinesCount,
			"duration_ms": output.Metadata.Duration.Milliseconds(),
		})
	}

	log.Info().
		Str("output_id", output.ID).
		Int("files", len(output.Files)).
		Int("lines", output.Metadata.LinesCount).
		Dur("duration", output.Metadata.Duration).
		Msg("Code generation completed successfully")

	return output, nil
}

// applyPatches applies all patches to the file system and populates the output
func (e *engine) applyPatches(ctx context.Context, patches []models.Patch, output *models.GenerationOutput) error {
	log.Debug().
		Int("patches", len(patches)).
		Msg("Applying patches to file system")

	// Emit phase started event
	e.emitEvent(models.NewPhaseStartedEvent("file_writing", fmt.Sprintf("Writing %d files to disk", len(patches))))
	phaseStart := time.Now()

	generatedFiles := make([]models.GeneratedFile, 0, len(patches))

	for i, patch := range patches {
		log.Debug().
			Int("patch", i+1).
			Int("total", len(patches)).
			Str("target", patch.TargetFile).
			Msg("Applying patch")

		// Emit file generating event
		e.emitEvent(models.NewFileGeneratingEvent(patch.TargetFile, "file_writing"))
		fileStart := time.Now()

		// Validate patch before applying
		if err := e.fileOps.ValidatePatch(ctx, patch); err != nil {
			log.Warn().
				Err(err).
				Str("target", patch.TargetFile).
				Msg("Patch validation failed, attempting to apply anyway")
		}

		// Apply patch with backup
		if err := e.fileOps.ApplyPatchWithBackup(ctx, patch); err != nil {
			return fmt.Errorf("failed to apply patch to %s: %w", patch.TargetFile, err)
		}

		// Read the file content after applying patch
		content, err := e.fileOps.ReadFile(ctx, patch.TargetFile)
		if err != nil {
			return fmt.Errorf("failed to read file %s after patch: %w", patch.TargetFile, err)
		}

		// Calculate checksum
		checksum := e.fileOps.GenerateChecksum(content)

		// Create generated file entry
		generatedFile := models.GeneratedFile{
			Path:        patch.TargetFile,
			Content:     content,
			Checksum:    checksum,
			GeneratedAt: patch.AppliedAt,
			Generator:   "langgraph-generation-workflow",
		}

		// Verify checksum
		if !generatedFile.VerifyChecksum() {
			return fmt.Errorf("checksum verification failed for %s", patch.TargetFile)
		}

		generatedFiles = append(generatedFiles, generatedFile)

		// Calculate lines and duration
		lines := strings.Count(content, "\n") + 1
		fileDuration := time.Since(fileStart)

		// Emit file completed event
		e.emitEvent(models.NewFileCompletedEvent(patch.TargetFile, "file_writing", lines, fileDuration))

		// Log decision
		if e.logDecisions {
			e.logDecision(ctx, "file_generated", fmt.Sprintf("Generated file: %s", patch.TargetFile), map[string]interface{}{
				"path":     patch.TargetFile,
				"checksum": checksum,
				"lines":    lines,
			})
		}
	}

	// Update output with generated files
	output.Files = generatedFiles
	output.Patches = patches

	// Emit phase completed event
	phaseDuration := time.Since(phaseStart)
	e.emitEvent(models.NewPhaseCompletedEvent("file_writing", phaseDuration, len(generatedFiles)))

	log.Debug().
		Int("files", len(generatedFiles)).
		Msg("All patches applied successfully")

	return nil
}

// countTotalLines counts total lines across all files
func (e *engine) countTotalLines(files []models.GeneratedFile) int {
	total := 0
	for _, file := range files {
		total += strings.Count(file.Content, "\n") + 1
	}
	return total
}

// logDecision logs a generation decision for audit and replay
func (e *engine) logDecision(_ context.Context, decision, rationale string, context map[string]interface{}) {
	log.Info().
		Str("decision", decision).
		Str("rationale", rationale).
		Interface("context", context).
		Msg("Generation decision")

	// In a full implementation, this would write to the execution log
	// using the models.DecisionLog structure
}

// Resume resumes generation from a checkpoint
func (e *engine) Resume(_ context.Context, checkpointID string) (*models.GenerationOutput, error) {
	log.Info().
		Str("checkpoint_id", checkpointID).
		Msg("Resuming generation from checkpoint")

	// This would use the graph's Resume functionality
	// For now, return an error indicating it's not implemented
	return nil, fmt.Errorf("checkpoint resume not yet implemented")
}

// emitEvent sends a progress event to the event channel if configured
func (e *engine) emitEvent(event models.ProgressEvent) {
	if e.eventChan != nil {
		select {
		case e.eventChan <- event:
			// Event sent successfully
		default:
			// Channel full or closed, skip event
			log.Warn().
				Str("event_type", string(event.Type)).
				Msg("Failed to send progress event: channel full or closed")
		}
	}
}
