package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/dshills/gocreator/internal/validate"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	validateSkipBuild bool
	validateSkipLint  bool
	validateSkipTests bool
	validateReport    string
)

var validateCmd = &cobra.Command{
	Use:   "validate <project-root>",
	Short: "Run build, lint, and test validation on existing project",
	Long: `Validate an existing project by running build, lint, and test checks.

The validation phase:
  1. Build Validation: Ensures code compiles without errors
  2. Lint Validation: Runs golangci-lint to check code quality
  3. Test Validation: Executes all tests and measures coverage

All checks run by default. Use skip flags to disable specific checks.

Exit codes:
  0 - All validations passed
  5 - One or more validations failed

Options:
  --skip-build    Skip build validation
  --skip-lint     Skip lint validation
  --skip-tests    Skip test validation
  --report PATH   Output validation report to JSON file

Example:
  # Validate all checks
  gocreator validate ./my-project

  # Skip linting
  gocreator validate ./my-project --skip-lint

  # Save report to file
  gocreator validate ./my-project --report ./validation.json`,
	Args: cobra.ExactArgs(1),
	RunE: runValidate,
}

func setupValidateFlags() {
	validateCmd.Flags().BoolVar(&validateSkipBuild, "skip-build", false, "skip build validation")
	validateCmd.Flags().BoolVar(&validateSkipLint, "skip-lint", false, "skip lint validation")
	validateCmd.Flags().BoolVar(&validateSkipTests, "skip-tests", false, "skip test validation")
	validateCmd.Flags().StringVarP(&validateReport, "report", "r", "", "output validation report to file (JSON format)")
}

func runValidate(_ *cobra.Command, args []string) error {
	projectRoot := args[0]

	log.Info().
		Str("project_root", projectRoot).
		Bool("skip_build", validateSkipBuild).
		Bool("skip_lint", validateSkipLint).
		Bool("skip_tests", validateSkipTests).
		Msg("Starting validation phase")

	fmt.Printf("GoCreator v%s - Validation Phase\n\n", version)
	fmt.Printf("Validating project: %s\n\n", projectRoot)

	// Check if project exists
	if _, err := os.Stat(projectRoot); os.IsNotExist(err) {
		log.Error().Err(err).Msg("Project directory does not exist")
		return ExitError{Code: ExitCodeFileSystemError, Err: fmt.Errorf("project directory does not exist: %s", projectRoot)}
	}

	// Configure validators based on skip flags
	logSkippedValidations()

	// Run validation
	ctx := context.Background()

	// Run validations
	buildPassed, err := runBuildValidation(ctx, projectRoot)
	if err != nil {
		return err
	}

	lintPassed, err := runLintValidation(ctx, projectRoot)
	if err != nil {
		return err
	}

	testPassed, err := runTestValidation(ctx, projectRoot)
	if err != nil {
		return err
	}

	// Determine overall result
	checksRun, checksPassed := calculateResults(buildPassed, lintPassed, testPassed)
	allPassed := checksPassed == checksRun

	// Print result
	printValidationResult(allPassed, checksPassed, checksRun)

	// Save report if requested
	if err := saveReport(buildPassed, lintPassed, testPassed, checksRun, checksPassed); err != nil {
		return err
	}

	log.Info().
		Bool("all_passed", allPassed).
		Int("checks_passed", checksPassed).
		Int("checks_run", checksRun).
		Msg("Validation phase completed")

	// Return error if any checks failed
	if !allPassed {
		return ExitError{Code: ExitCodeValidationError, Err: fmt.Errorf("validation failed: %d/%d checks passed", checksPassed, checksRun)}
	}

	return nil
}

func logSkippedValidations() {
	if validateSkipBuild {
		log.Info().Msg("Build validation skipped")
	}
	if validateSkipLint {
		log.Info().Msg("Lint validation skipped")
	}
	if validateSkipTests {
		log.Info().Msg("Test validation skipped")
	}
}

func runBuildValidation(ctx context.Context, projectRoot string) (bool, error) {
	if validateSkipBuild {
		return false, nil
	}

	fmt.Printf("[1/3] Build Validation\n")
	fmt.Printf("  Running: go build ./...\n")

	buildValidator := validate.NewBuildValidator(cfg.Validation.TestTimeout)
	buildResult, err := buildValidator.Validate(ctx, projectRoot)
	if err != nil {
		log.Error().Err(err).Msg("Build validation error")
		return false, ExitError{Code: ExitCodeValidationError, Err: fmt.Errorf("build validation error: %w", err)}
	}

	if buildResult.Success {
		fmt.Printf("  ✓ Build successful [elapsed: %.1fs]\n\n", buildResult.Duration.Seconds())
		return true, nil
	}

	fmt.Printf("  ✗ Build failed:\n")
	for _, buildErr := range buildResult.Errors {
		fmt.Printf("    - %s:%d: %s\n", buildErr.File, buildErr.Line, buildErr.Message)
	}
	fmt.Printf("\n")
	return false, nil
}

func runLintValidation(ctx context.Context, projectRoot string) (bool, error) {
	if validateSkipLint {
		return false, nil
	}

	fmt.Printf("[2/3] Lint Validation\n")
	fmt.Printf("  Running: golangci-lint run ./...\n")

	lintValidator := validate.NewLintValidator(validate.WithSkipIfNotFound(true))
	lintResult, err := lintValidator.Validate(ctx, projectRoot)
	if err != nil {
		log.Error().Err(err).Msg("Lint validation error")
		return false, ExitError{Code: ExitCodeValidationError, Err: fmt.Errorf("lint validation error: %w", err)}
	}

	if lintResult.Success {
		fmt.Printf("  ✓ No lint issues found [elapsed: %.1fs]\n\n", lintResult.Duration.Seconds())
		return true, nil
	}

	fmt.Printf("  ✗ Found %d issues:\n", len(lintResult.Issues))
	for i, issue := range lintResult.Issues {
		if i < 10 { // Show first 10 issues
			fmt.Printf("    - %s:%d: %s\n", issue.File, issue.Line, issue.Message)
		}
	}
	if len(lintResult.Issues) > 10 {
		fmt.Printf("    ... and %d more issues\n", len(lintResult.Issues)-10)
	}
	fmt.Printf("\n")
	return false, nil
}

func runTestValidation(ctx context.Context, projectRoot string) (bool, error) {
	if validateSkipTests {
		return false, nil
	}

	fmt.Printf("[3/3] Test Validation\n")
	fmt.Printf("  Running: go test ./...\n")

	testValidator := validate.NewTestValidator(validate.WithTestTimeout(cfg.Validation.TestTimeout))
	testResult, err := testValidator.Validate(ctx, projectRoot)
	if err != nil {
		log.Error().Err(err).Msg("Test validation error")
		return false, ExitError{Code: ExitCodeValidationError, Err: fmt.Errorf("test validation error: %w", err)}
	}

	if testResult.Success {
		fmt.Printf("  ✓ All tests passed (%d/%d) [coverage: %.1f%%] [elapsed: %.1fs]\n\n",
			testResult.PassedTests, testResult.TotalTests, testResult.Coverage, testResult.Duration.Seconds())
		return true, nil
	}

	fmt.Printf("  ✗ Tests failed (%d/%d passed) [coverage: %.1f%%]\n",
		testResult.PassedTests, testResult.TotalTests, testResult.Coverage)
	for i, failure := range testResult.Failures {
		if i < 5 { // Show first 5 failures
			fmt.Printf("    - %s: %s\n", failure.Test, failure.Message)
		}
	}
	if len(testResult.Failures) > 5 {
		fmt.Printf("    ... and %d more failures\n", len(testResult.Failures)-5)
	}
	fmt.Printf("\n")
	return false, nil
}

func calculateResults(buildPassed, lintPassed, testPassed bool) (checksRun, checksPassed int) {
	if !validateSkipBuild {
		checksRun++
		if buildPassed {
			checksPassed++
		}
	}
	if !validateSkipLint {
		checksRun++
		if lintPassed {
			checksPassed++
		}
	}
	if !validateSkipTests {
		checksRun++
		if testPassed {
			checksPassed++
		}
	}
	return checksRun, checksPassed
}

func printValidationResult(allPassed bool, checksPassed, checksRun int) {
	if allPassed {
		fmt.Printf("Validation Result: PASSED (%d/%d checks passed)\n", checksPassed, checksRun)
	} else {
		fmt.Printf("Validation Result: FAILED (%d/%d checks passed)\n", checksPassed, checksRun)
	}
}

func saveReport(buildPassed, lintPassed, testPassed bool, checksRun, checksPassed int) error {
	if validateReport == "" {
		return nil
	}

	report := map[string]interface{}{
		"build_passed":  buildPassed,
		"lint_passed":   lintPassed,
		"test_passed":   testPassed,
		"checks_run":    checksRun,
		"checks_passed": checksPassed,
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal report")
		return ExitError{Code: ExitCodeInternalError, Err: fmt.Errorf("failed to marshal report: %w", err)}
	}

	if err := os.WriteFile(validateReport, data, 0o600); err != nil {
		log.Error().Err(err).Msg("Failed to write report")
		return ExitError{Code: ExitCodeFileSystemError, Err: fmt.Errorf("failed to write report: %w", err)}
	}

	log.Info().Str("report_path", validateReport).Msg("Saved validation report")
	fmt.Printf("\nDetailed report written to: %s\n", validateReport)

	return nil
}
