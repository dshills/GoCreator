package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dshills/gocreator/internal/clarify"
	"github.com/dshills/gocreator/internal/cli"
	"github.com/dshills/gocreator/internal/generate"
	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/internal/spec"
	"github.com/dshills/gocreator/pkg/fsops"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	generateOutput      string
	generateResume      bool
	generateBatch       string
	generateDryRun      bool
	generateIncremental bool
)

var generateCmd = &cobra.Command{
	Use:   "generate <spec-file>",
	Short: "Run clarification and generation phases",
	Long: `Run the complete generation workflow including clarification and code generation.

The generation phase:
  1. Clarification: Analyzes specification and resolves ambiguities
  2. Planning: Creates architecture plan and file structure
  3. Code Generation: Generates complete project structure
  4. Finalization: Creates build files, documentation, and metadata

Validation is skipped (use 'full' command to include validation).

Options:
  --resume       Resume from last checkpoint if available
  --batch        Use pre-answered questions from JSON file
  --dry-run      Show what would be generated without writing files
  --incremental  Enable incremental regeneration (only regenerate changed files)

Example:
  # Basic generation
  gocreator generate ./my-project-spec.yaml

  # Specify output directory
  gocreator generate ./my-project-spec.yaml --output ./my-project

  # Resume from checkpoint
  gocreator generate ./my-project-spec.yaml --resume

  # Batch mode
  gocreator generate ./my-project-spec.yaml --batch ./answers.json`,
	Args: cobra.ExactArgs(1),
	RunE: runGenerate,
}

func setupGenerateFlags() {
	generateCmd.Flags().StringVarP(&generateOutput, "output", "o", "./generated", "output directory for generated code")
	generateCmd.Flags().BoolVar(&generateResume, "resume", false, "resume from last checkpoint if available")
	generateCmd.Flags().StringVar(&generateBatch, "batch", "", "path to JSON file with pre-answered questions")
	generateCmd.Flags().BoolVar(&generateDryRun, "dry-run", false, "show what would be generated without writing files")
	generateCmd.Flags().BoolVar(&generateIncremental, "incremental", false, "enable incremental regeneration (only regenerate changed files)")
}

func runGenerate(_ *cobra.Command, args []string) error {
	specFile := args[0]

	log.Info().
		Str("spec_file", specFile).
		Str("output", generateOutput).
		Bool("resume", generateResume).
		Bool("dry_run", generateDryRun).
		Msg("Starting generation phase")

	// Phase 1: Clarification (silent, no progress bar for now)
	fcs, err := runClarificationPhase(specFile, generateBatch)
	if err != nil {
		return err
	}

	// Phase 2: Code Generation with Progress Tracking
	if generateDryRun {
		fmt.Printf("\n[DRY RUN] No files will be written\n\n")
		return nil
	}

	if err := runGenerationWithProgress(fcs, generateOutput, generateIncremental); err != nil {
		return err
	}

	// Show next steps
	fmt.Printf("\nOutput written to: %s\n\n", generateOutput)
	fmt.Printf("Next steps:\n")
	fmt.Printf("  cd %s\n", generateOutput)
	fmt.Printf("  go mod tidy\n")
	fmt.Printf("  make test\n\n")

	return nil
}

func runClarificationPhase(specFile, batchFile string) (*models.FinalClarifiedSpecification, error) {
	// Detect format
	format, err := detectSpecFormat(specFile)
	if err != nil {
		return nil, ExitError{Code: ExitCodeSpecError, Err: err}
	}

	// Read spec file
	//nolint:gosec // G304: Reading user-provided spec file - required for CLI functionality
	content, err := os.ReadFile(specFile)
	if err != nil {
		return nil, ExitError{Code: ExitCodeFileSystemError, Err: fmt.Errorf("failed to read spec file: %w", err)}
	}

	// Parse and validate
	inputSpec, err := spec.ParseAndValidate(format, string(content))
	if err != nil {
		return nil, ExitError{Code: ExitCodeSpecError, Err: fmt.Errorf("specification validation failed: %w", err)}
	}

	// Create LLM client
	llmClient, err := createLLMClient(cfg)
	if err != nil {
		return nil, ExitError{Code: ExitCodeNetworkError, Err: fmt.Errorf("failed to create LLM client: %w", err)}
	}

	// Create clarification engine
	engine, err := clarify.NewEngine(clarify.EngineConfig{
		LLMClient: llmClient,
	})
	if err != nil {
		return nil, ExitError{Code: ExitCodeInternalError, Err: fmt.Errorf("failed to create clarification engine: %w", err)}
	}

	// Determine interactive mode
	interactive := batchFile == ""

	// Run clarification
	ctx := context.Background()
	fcs, err := engine.Clarify(ctx, inputSpec, interactive)
	if err != nil {
		return nil, ExitError{Code: ExitCodeClarificationError, Err: fmt.Errorf("clarification failed: %w", err)}
	}

	return fcs, nil
}

type generationPlan struct {
	Packages []string
	Files    []string
}

func runPlanningPhase(_ *models.FinalClarifiedSpecification) (*generationPlan, error) {
	// TODO: Implement actual planning logic
	log.Warn().Msg("Planning phase not yet fully implemented")

	// Return placeholder plan
	return &generationPlan{
		Packages: []string{"internal/app", "internal/domain", "cmd/server"},
		Files:    []string{"main.go", "go.mod", "README.md"},
	}, nil
}

func runCodeGeneration(_ *generationPlan, outputDir string, dryRun bool) error {
	// TODO: Implement actual code generation
	log.Warn().Msg("Code generation not yet fully implemented")

	if !dryRun {
		// Create output directory
		if err := os.MkdirAll(outputDir, 0o750); err != nil {
			return ExitError{Code: ExitCodeFileSystemError, Err: fmt.Errorf("failed to create output directory: %w", err)}
		}
	}

	return nil
}

func runFinalization(outputDir string, dryRun bool) error {
	// TODO: Implement finalization logic
	log.Warn().Msg("Finalization not yet fully implemented")

	if !dryRun {
		// Create .gocreator directory
		metaDir := filepath.Join(outputDir, ".gocreator")
		if err := os.MkdirAll(metaDir, 0o750); err != nil {
			return ExitError{Code: ExitCodeFileSystemError, Err: fmt.Errorf("failed to create metadata directory: %w", err)}
		}
	}

	return nil
}

// runGenerationWithProgress runs the generation engine with real-time progress tracking
func runGenerationWithProgress(fcs *models.FinalClarifiedSpecification, outputDir string, incremental bool) error {
	// Create event channel for progress updates
	eventChan := make(chan models.ProgressEvent, 100)

	// Create progress tracker
	progressConfig := cli.ProgressConfig{
		Writer:         os.Stdout,
		ShowTokens:     true,
		ShowCost:       true,
		ShowETA:        true,
		UpdateInterval: 500 * time.Millisecond,
		Quiet:          false,
	}
	tracker := cli.NewProgressTracker(progressConfig)

	// Start progress tracking in background
	done := make(chan struct{})
	go func() {
		defer close(done)
		for event := range eventChan {
			tracker.HandleEvent(event)
		}
	}()

	// Create LLM client
	llmClient, err := createLLMClient(cfg)
	if err != nil {
		return ExitError{Code: ExitCodeNetworkError, Err: fmt.Errorf("failed to create LLM client: %w", err)}
	}

	// Create file operations handler with logger
	logDir := filepath.Join(outputDir, ".gocreator", "logs")
	logger, err := fsops.NewFileLogger(logDir)
	if err != nil {
		return ExitError{Code: ExitCodeInternalError, Err: fmt.Errorf("failed to create file logger: %w", err)}
	}
	defer func() {
		if closeErr := logger.Close(); closeErr != nil {
			log.Warn().Err(closeErr).Msg("Failed to close file logger")
		}
	}()

	fileOps, err := fsops.New(fsops.Config{
		RootDir: outputDir,
		Logger:  logger,
	})
	if err != nil {
		return ExitError{Code: ExitCodeInternalError, Err: fmt.Errorf("failed to create file operations handler: %w", err)}
	}

	// Create generation engine
	engine, err := generate.NewEngine(generate.EngineConfig{
		LLMClient:    llmClient,
		FileOps:      fileOps,
		LogDecisions: true,
		EventChan:    eventChan,
		Incremental:  incremental,
		OutputDir:    outputDir,
	})
	if err != nil {
		return ExitError{Code: ExitCodeInternalError, Err: fmt.Errorf("failed to create generation engine: %w", err)}
	}

	// Start progress tracker with total phases
	// Phases: initialization, analyze_fcs, create_plan, generate_packages, generate_tests, generate_config, file_writing
	tracker.Start(7)

	// Run generation
	ctx := context.Background()
	output, err := engine.Generate(ctx, fcs, outputDir)

	// Close event channel and wait for progress tracker to finish
	close(eventChan)
	<-done

	if err != nil {
		return ExitError{Code: ExitCodeGenerationError, Err: fmt.Errorf("code generation failed: %w", err)}
	}

	// Complete progress tracking
	tracker.Complete()

	// Log summary
	log.Info().
		Str("output_id", output.ID).
		Int("files", len(output.Files)).
		Msg("Generation completed successfully")

	return nil
}
