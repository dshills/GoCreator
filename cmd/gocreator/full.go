package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dshills/gocreator/internal/clarify"
	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/internal/spec"
	"github.com/dshills/gocreator/internal/validate"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	fullOutput string
	fullBatch  string
	fullResume bool
	fullReport string
)

var fullCmd = &cobra.Command{
	Use:   "full <spec-file>",
	Short: "Run complete pipeline (clarify + generate + validate)",
	Long: `Run the complete GoCreator pipeline from specification to validated code.

The full pipeline includes:
  1. Clarification: Analyzes specification and resolves ambiguities
  2. Planning: Creates architecture plan and file structure
  3. Code Generation: Generates complete project structure
  4. Finalization: Creates build files, documentation, and metadata
  5. Validation: Validates generated code (build, lint, test)

This is the recommended command for end-to-end code generation.

Options:
  --batch       Use pre-answered questions from JSON file
  --resume      Resume from last checkpoint if available
  --report PATH Output validation report to JSON file

Example:
  # Full pipeline
  gocreator full ./my-project-spec.yaml

  # Specify output directory
  gocreator full ./my-project-spec.yaml --output ./my-project

  # Batch mode with validation report
  gocreator full ./my-project-spec.yaml --batch ./answers.json --report ./validation.json`,
	Args: cobra.ExactArgs(1),
	RunE: runFull,
}

func setupFullFlags() {
	fullCmd.Flags().StringVarP(&fullOutput, "output", "o", "./generated", "output directory")
	fullCmd.Flags().StringVar(&fullBatch, "batch", "", "path to JSON file with pre-answered questions")
	fullCmd.Flags().BoolVar(&fullResume, "resume", false, "resume from last checkpoint")
	fullCmd.Flags().StringVarP(&fullReport, "report", "r", "", "output validation report to file")
}

func runFull(_ *cobra.Command, args []string) error {
	specFile := args[0]

	log.Info().
		Str("spec_file", specFile).
		Str("output", fullOutput).
		Bool("resume", fullResume).
		Msg("Starting full pipeline")

	fmt.Printf("GoCreator v%s - Full Pipeline\n\n", version)

	startTime := time.Now()

	// Phase 1: Clarification
	fmt.Printf("=== Phase 1: Clarification ===\n\n")
	fcs, err := runFullClarification(specFile, fullBatch)
	if err != nil {
		return err
	}
	fmt.Printf("  ✓ Specification analyzed\n")
	fmt.Printf("  ✓ FCS constructed\n\n")

	// Phase 2: Planning
	fmt.Printf("=== Phase 2: Planning ===\n\n")
	plan, err := runPlanningPhase(fcs)
	if err != nil {
		return err
	}
	fmt.Printf("  ✓ Architecture planned (%d packages)\n", len(plan.Packages))
	fmt.Printf("  ✓ File tree generated (%d files)\n\n", len(plan.Files))

	// Phase 3: Code Generation
	fmt.Printf("=== Phase 3: Code Generation ===\n\n")
	if err := runCodeGeneration(plan, fullOutput, false); err != nil {
		return err
	}
	fmt.Printf("  ✓ Code generation complete\n\n")

	// Phase 4: Finalization
	fmt.Printf("=== Phase 4: Finalization ===\n\n")
	if err := runFinalization(fullOutput, false); err != nil {
		return err
	}
	fmt.Printf("  ✓ Build files created\n")
	fmt.Printf("  ✓ Documentation generated\n\n")

	// Phase 5: Validation
	fmt.Printf("=== Phase 5: Validation ===\n\n")
	validationPassed, err := runFullValidation(fullOutput, fullReport)
	if err != nil {
		// Don't return error - validation failure shouldn't fail the entire pipeline
		log.Warn().Err(err).Msg("Validation phase had failures")
	}

	duration := time.Since(startTime)
	fmt.Printf("\n=== Pipeline Complete ===\n\n")
	fmt.Printf("Total time: %.1fs\n", duration.Seconds())
	fmt.Printf("Output directory: %s\n", fullOutput)

	if validationPassed {
		fmt.Printf("Status: SUCCESS (all validations passed)\n\n")
		fmt.Printf("Next steps:\n")
		fmt.Printf("  cd %s\n", fullOutput)
		fmt.Printf("  go mod tidy\n")
		fmt.Printf("  go run ./cmd/...\n")
	} else {
		fmt.Printf("Status: GENERATED (validation failures detected)\n\n")
		fmt.Printf("Code was generated successfully, but validation found issues.\n")
		fmt.Printf("Review the validation output above and update the specification.\n\n")
		fmt.Printf("To regenerate after fixing issues:\n")
		fmt.Printf("  gocreator full %s --output %s\n", specFile, fullOutput)
	}

	log.Info().
		Str("output", fullOutput).
		Bool("validation_passed", validationPassed).
		Dur("duration", duration).
		Msg("Full pipeline completed")

	return nil
}

func runFullClarification(specFile, batchFile string) (*models.FinalClarifiedSpecification, error) {
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
	checkpointDir := filepath.Join(fullOutput, ".gocreator", "checkpoints")
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

func runFullValidation(projectRoot, reportPath string) (bool, error) {
	ctx := context.Background()

	// Run build validation
	fmt.Printf("[1/3] Build Validation\n")
	buildValidator := validate.NewBuildValidator(cfg.Validation.TestTimeout)
	buildResult, err := buildValidator.Validate(ctx, projectRoot)
	if err != nil {
		log.Error().Err(err).Msg("Build validation error")
		return false, err
	}

	if buildResult.Success {
		fmt.Printf("  ✓ Build successful [elapsed: %.1fs]\n", buildResult.Duration.Seconds())
	} else {
		fmt.Printf("  ✗ Build failed (%d errors)\n", len(buildResult.Errors))
		for i, err := range buildResult.Errors {
			if i < 5 {
				fmt.Printf("    - %s:%d: %s\n", err.File, err.Line, err.Message)
			}
		}
		if len(buildResult.Errors) > 5 {
			fmt.Printf("    ... and %d more errors\n", len(buildResult.Errors)-5)
		}
	}

	// Run lint validation
	fmt.Printf("\n[2/3] Lint Validation\n")
	lintValidator := validate.NewLintValidator(validate.WithSkipIfNotFound(true))
	lintResult, err := lintValidator.Validate(ctx, projectRoot)
	if err != nil {
		log.Error().Err(err).Msg("Lint validation error")
		return false, err
	}

	if lintResult.Success {
		fmt.Printf("  ✓ No lint issues [elapsed: %.1fs]\n", lintResult.Duration.Seconds())
	} else {
		fmt.Printf("  ✗ Found %d lint issues\n", len(lintResult.Issues))
		for i, issue := range lintResult.Issues {
			if i < 5 {
				fmt.Printf("    - %s:%d: %s\n", issue.File, issue.Line, issue.Message)
			}
		}
		if len(lintResult.Issues) > 5 {
			fmt.Printf("    ... and %d more issues\n", len(lintResult.Issues)-5)
		}
	}

	// Run test validation
	fmt.Printf("\n[3/3] Test Validation\n")
	testValidator := validate.NewTestValidator(validate.WithTestTimeout(cfg.Validation.TestTimeout))
	testResult, err := testValidator.Validate(ctx, projectRoot)
	if err != nil {
		log.Error().Err(err).Msg("Test validation error")
		return false, err
	}

	if testResult.Success {
		fmt.Printf("  ✓ All tests passed (%d/%d) [coverage: %.1f%%] [elapsed: %.1fs]\n",
			testResult.PassedTests, testResult.TotalTests, testResult.Coverage, testResult.Duration.Seconds())
	} else {
		fmt.Printf("  ✗ Tests failed (%d/%d passed)\n", testResult.PassedTests, testResult.TotalTests)
		for i, failure := range testResult.Failures {
			if i < 5 {
				fmt.Printf("    - %s: %s\n", failure.Test, failure.Message)
			}
		}
		if len(testResult.Failures) > 5 {
			fmt.Printf("    ... and %d more failures\n", len(testResult.Failures)-5)
		}
	}

	allPassed := buildResult.Success && lintResult.Success && testResult.Success

	// Save report if requested
	if reportPath != "" {
		report := map[string]interface{}{
			"build_passed":  buildResult.Success,
			"lint_passed":   lintResult.Success,
			"test_passed":   testResult.Success,
			"all_passed":    allPassed,
			"build_errors":  len(buildResult.Errors),
			"lint_issues":   len(lintResult.Issues),
			"test_failures": len(testResult.Failures),
			"coverage":      testResult.Coverage,
		}

		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal report")
			return allPassed, fmt.Errorf("failed to marshal report: %w", err)
		}

		if err := os.WriteFile(reportPath, data, 0600); err != nil {
			log.Error().Err(err).Msg("Failed to write report")
			return allPassed, fmt.Errorf("failed to write report: %w", err)
		}

		fmt.Printf("\nValidation report saved to: %s\n", reportPath)
	}

	return allPassed, nil
}
