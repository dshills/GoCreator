package validate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/google/uuid"
)

const (
	// SchemaVersion is the current validation report schema version
	SchemaVersion = "1.0"
)

// ReportGenerator generates validation reports
type ReportGenerator interface {
	Generate(buildResult *models.BuildResult, lintResult *models.LintResult, testResult *models.TestResult, outputID string) (*models.ValidationReport, error)
	Save(report *models.ValidationReport, outputPath string) error
	Load(reportPath string) (*models.ValidationReport, error)
}

// reportGenerator implements ReportGenerator
type reportGenerator struct{}

// NewReportGenerator creates a new report generator
func NewReportGenerator() ReportGenerator {
	return &reportGenerator{}
}

// Generate creates a validation report from individual validation results
func (r *reportGenerator) Generate(
	buildResult *models.BuildResult,
	lintResult *models.LintResult,
	testResult *models.TestResult,
	outputID string,
) (*models.ValidationReport, error) {
	if buildResult == nil {
		return nil, fmt.Errorf("buildResult cannot be nil")
	}
	if lintResult == nil {
		return nil, fmt.Errorf("lintResult cannot be nil")
	}
	if testResult == nil {
		return nil, fmt.Errorf("testResult cannot be nil")
	}

	report := &models.ValidationReport{
		SchemaVersion: SchemaVersion,
		ID:            uuid.New().String(),
		OutputID:      outputID,
		BuildResult:   *buildResult,
		LintResult:    *lintResult,
		TestResult:    *testResult,
		CreatedAt:     time.Now(),
	}

	// Compute overall status
	report.OverallStatus = report.ComputeOverallStatus()

	// Validate the report
	if err := report.Validate(); err != nil {
		return nil, fmt.Errorf("invalid validation report: %w", err)
	}

	return report, nil
}

// Save writes the validation report to a JSON file
func (r *reportGenerator) Save(report *models.ValidationReport, outputPath string) error {
	if report == nil {
		return fmt.Errorf("report cannot be nil")
	}

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write report to %s: %w", outputPath, err)
	}

	return nil
}

// Load reads a validation report from a JSON file
func (r *reportGenerator) Load(reportPath string) (*models.ValidationReport, error) {
	//nolint:gosec // G304: Reading validation report file - required for report loading
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read report from %s: %w", reportPath, err)
	}

	var report models.ValidationReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal report: %w", err)
	}

	// Validate the loaded report
	if err := report.Validate(); err != nil {
		return nil, fmt.Errorf("loaded report is invalid: %w", err)
	}

	return &report, nil
}

// FileErrorMap creates a map of files to their errors/issues
// This is useful for presenting validation results grouped by file
func FileErrorMap(report *models.ValidationReport) map[string]FileErrors {
	fileMap := make(map[string]FileErrors)

	// Add build errors
	for _, err := range report.BuildResult.Errors {
		fe := fileMap[err.File]
		fe.BuildErrors = append(fe.BuildErrors, err)
		fileMap[err.File] = fe
	}

	// Add build warnings
	for _, warn := range report.BuildResult.Warnings {
		fe := fileMap[warn.File]
		fe.BuildWarnings = append(fe.BuildWarnings, warn)
		fileMap[warn.File] = fe
	}

	// Add lint issues
	for _, issue := range report.LintResult.Issues {
		fe := fileMap[issue.File]
		fe.LintIssues = append(fe.LintIssues, issue)
		fileMap[issue.File] = fe
	}

	return fileMap
}

// FileErrors contains all errors and issues for a specific file
type FileErrors struct {
	BuildErrors   []models.CompilationError   `json:"build_errors,omitempty"`
	BuildWarnings []models.CompilationWarning `json:"build_warnings,omitempty"`
	LintIssues    []models.LintIssue          `json:"lint_issues,omitempty"`
}

// HasErrors returns true if the file has any errors (not warnings)
func (f *FileErrors) HasErrors() bool {
	if len(f.BuildErrors) > 0 {
		return true
	}
	for _, issue := range f.LintIssues {
		if issue.Severity == "error" {
			return true
		}
	}
	return false
}

// Summary provides a human-readable summary of the validation report
type Summary struct {
	OverallStatus   models.ValidationStatus `json:"overall_status"`
	BuildPassed     bool                    `json:"build_passed"`
	LintPassed      bool                    `json:"lint_passed"`
	TestsPassed     bool                    `json:"tests_passed"`
	TotalErrors     int                     `json:"total_errors"`
	TotalWarnings   int                     `json:"total_warnings"`
	TotalLintIssues int                     `json:"total_lint_issues"`
	TestCoverage    float64                 `json:"test_coverage"`
	TotalDuration   time.Duration           `json:"total_duration"`
}

// GenerateSummary creates a summary from a validation report
func GenerateSummary(report *models.ValidationReport) Summary {
	totalDuration := report.BuildResult.Duration +
		report.LintResult.Duration +
		report.TestResult.Duration

	return Summary{
		OverallStatus:   report.OverallStatus,
		BuildPassed:     report.BuildResult.Success,
		LintPassed:      report.LintResult.Success,
		TestsPassed:     report.TestResult.Success,
		TotalErrors:     len(report.BuildResult.Errors),
		TotalWarnings:   len(report.BuildResult.Warnings),
		TotalLintIssues: len(report.LintResult.Issues),
		TestCoverage:    report.TestResult.Coverage,
		TotalDuration:   totalDuration,
	}
}
