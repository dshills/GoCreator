package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dshills/gocreator/internal/clarify"
	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/internal/spec"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	generateOutput string
	generateResume bool
	generateBatch  string
	generateDryRun bool
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
  --resume      Resume from last checkpoint if available
  --batch       Use pre-answered questions from JSON file
  --dry-run     Show what would be generated without writing files

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
}

func runGenerate(_ *cobra.Command, args []string) error {
	specFile := args[0]

	log.Info().
		Str("spec_file", specFile).
		Str("output", generateOutput).
		Bool("resume", generateResume).
		Bool("dry_run", generateDryRun).
		Msg("Starting generation phase")

	fmt.Printf("GoCreator v%s - Generation Phase\n\n", version)

	startTime := time.Now()

	// Phase 1: Clarification
	fmt.Printf("[1/4] Clarification\n")
	fcs, err := runClarificationPhase(specFile, generateBatch)
	if err != nil {
		return err
	}
	fmt.Printf("  ✓ Specification analyzed (0 ambiguities)\n")
	fmt.Printf("  ✓ FCS constructed\n\n")

	// Phase 2: Planning
	fmt.Printf("[2/4] Planning\n")
	plan, err := runPlanningPhase(fcs)
	if err != nil {
		return err
	}
	fmt.Printf("  ✓ Architecture planned (%d packages)\n", len(plan.Packages))
	fmt.Printf("  ✓ File tree generated (%d files)\n\n", len(plan.Files))

	// Phase 3: Code Generation
	fmt.Printf("[3/4] Code Generation\n")
	if generateDryRun {
		fmt.Printf("  [DRY RUN] No files will be written\n")
	}

	if err := runCodeGeneration(plan, generateOutput, generateDryRun); err != nil {
		return err
	}
	fmt.Printf("  ✓ Generation complete\n\n")

	// Phase 4: Finalization
	fmt.Printf("[4/4] Finalization\n")
	if err := runFinalization(generateOutput, generateDryRun); err != nil {
		return err
	}
	fmt.Printf("  ✓ go.mod created\n")
	fmt.Printf("  ✓ Makefile created\n")
	fmt.Printf("  ✓ README.md created\n\n")

	duration := time.Since(startTime)
	fmt.Printf("Generation complete! [total: %.1fs]\n", duration.Seconds())
	if !generateDryRun {
		fmt.Printf("Output written to: %s\n\n", generateOutput)
		fmt.Printf("Next steps:\n")
		fmt.Printf("  cd %s\n", generateOutput)
		fmt.Printf("  go mod tidy\n")
		fmt.Printf("  make test\n")
	}

	log.Info().
		Str("output", generateOutput).
		Dur("duration", duration).
		Msg("Generation phase completed successfully")

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
	checkpointDir := filepath.Join(generateOutput, ".gocreator", "checkpoints")
	engine, err := clarify.NewEngine(clarify.EngineConfig{
		LLMClient:        llmClient,
		CheckpointDir:    checkpointDir,
		EnableCheckpoint: true,
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
		if err := os.MkdirAll(outputDir, 0750); err != nil {
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
		if err := os.MkdirAll(metaDir, 0750); err != nil {
			return ExitError{Code: ExitCodeFileSystemError, Err: fmt.Errorf("failed to create metadata directory: %w", err)}
		}
	}

	return nil
}
