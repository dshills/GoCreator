package validate

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"golang.org/x/sync/errgroup"
)

// Validator orchestrates the complete validation process
type Validator interface {
	Validate(ctx context.Context, projectRoot string) (*models.ValidationReport, error)
	ValidateWithOutputID(ctx context.Context, projectRoot, outputID string) (*models.ValidationReport, error)
}

// Engine orchestrates build, lint, and test validation
type Engine struct {
	buildValidator BuildValidator
	lintValidator  LintValidator
	testValidator  TestValidator
	reportGen      ReportGenerator
	concurrent     bool
}

// EngineOption configures the validation engine
type EngineOption func(*Engine)

// WithBuildValidator sets a custom build validator
func WithBuildValidator(v BuildValidator) EngineOption {
	return func(e *Engine) {
		e.buildValidator = v
	}
}

// WithLintValidator sets a custom lint validator
func WithLintValidator(v LintValidator) EngineOption {
	return func(e *Engine) {
		e.lintValidator = v
	}
}

// WithTestValidator sets a custom test validator
func WithTestValidator(v TestValidator) EngineOption {
	return func(e *Engine) {
		e.testValidator = v
	}
}

// WithReportGenerator sets a custom report generator
func WithReportGenerator(g ReportGenerator) EngineOption {
	return func(e *Engine) {
		e.reportGen = g
	}
}

// WithConcurrentValidation enables/disables concurrent validation
// When true, build, lint, and test run in parallel
// When false, they run sequentially
func WithConcurrentValidation(concurrent bool) EngineOption {
	return func(e *Engine) {
		e.concurrent = concurrent
	}
}

// NewEngine creates a new validation engine with default validators
func NewEngine(opts ...EngineOption) *Engine {
	e := &Engine{
		buildValidator: NewBuildValidator(2 * time.Minute),
		lintValidator:  NewLintValidator(WithSkipIfNotFound(true)),
		testValidator:  NewTestValidator(WithTestTimeout(5 * time.Minute)),
		reportGen:      NewReportGenerator(),
		concurrent:     true, // Default to concurrent execution
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Validate runs complete validation: build, lint, and test
func (e *Engine) Validate(ctx context.Context, projectRoot string) (*models.ValidationReport, error) {
	return e.ValidateWithOutputID(ctx, projectRoot, "")
}

// ValidateWithOutputID runs complete validation with a specific output ID
func (e *Engine) ValidateWithOutputID(ctx context.Context, projectRoot, outputID string) (*models.ValidationReport, error) {
	if projectRoot == "" {
		return nil, fmt.Errorf("projectRoot cannot be empty")
	}

	var buildResult *models.BuildResult
	var lintResult *models.LintResult
	var testResult *models.TestResult

	if e.concurrent {
		// Run validations concurrently
		var err error
		buildResult, lintResult, testResult, err = e.runConcurrent(ctx, projectRoot)
		if err != nil {
			return nil, err
		}
	} else {
		// Run validations sequentially
		var err error
		buildResult, lintResult, testResult, err = e.runSequential(ctx, projectRoot)
		if err != nil {
			return nil, err
		}
	}

	// Generate report
	report, err := e.reportGen.Generate(buildResult, lintResult, testResult, outputID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate report: %w", err)
	}

	return report, nil
}

// runConcurrent runs all validations concurrently using errgroup
func (e *Engine) runConcurrent(ctx context.Context, projectRoot string) (
	*models.BuildResult,
	*models.LintResult,
	*models.TestResult,
	error,
) {
	var buildResult *models.BuildResult
	var lintResult *models.LintResult
	var testResult *models.TestResult
	var mu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)

	// Run build validation
	g.Go(func() error {
		result, err := e.buildValidator.Validate(ctx, projectRoot)
		if err != nil {
			return fmt.Errorf("build validation failed: %w", err)
		}
		mu.Lock()
		buildResult = result
		mu.Unlock()
		return nil
	})

	// Run lint validation
	g.Go(func() error {
		result, err := e.lintValidator.Validate(ctx, projectRoot)
		if err != nil {
			return fmt.Errorf("lint validation failed: %w", err)
		}
		mu.Lock()
		lintResult = result
		mu.Unlock()
		return nil
	})

	// Run test validation
	g.Go(func() error {
		result, err := e.testValidator.Validate(ctx, projectRoot)
		if err != nil {
			return fmt.Errorf("test validation failed: %w", err)
		}
		mu.Lock()
		testResult = result
		mu.Unlock()
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, nil, nil, err
	}

	return buildResult, lintResult, testResult, nil
}

// runSequential runs all validations sequentially
func (e *Engine) runSequential(ctx context.Context, projectRoot string) (
	*models.BuildResult,
	*models.LintResult,
	*models.TestResult,
	error,
) {
	// Run build validation first
	buildResult, err := e.buildValidator.Validate(ctx, projectRoot)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("build validation failed: %w", err)
	}

	// Run lint validation
	lintResult, err := e.lintValidator.Validate(ctx, projectRoot)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("lint validation failed: %w", err)
	}

	// Run test validation
	testResult, err := e.testValidator.Validate(ctx, projectRoot)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("test validation failed: %w", err)
	}

	return buildResult, lintResult, testResult, nil
}

// ValidateAndSave runs validation and saves the report to a file
func (e *Engine) ValidateAndSave(ctx context.Context, projectRoot, reportPath, outputID string) (*models.ValidationReport, error) {
	report, err := e.ValidateWithOutputID(ctx, projectRoot, outputID)
	if err != nil {
		return nil, err
	}

	if err := e.reportGen.Save(report, reportPath); err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	return report, nil
}
