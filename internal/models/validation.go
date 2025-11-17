package models

import (
	"fmt"
	"time"
)

// ValidationStatus represents the overall validation status
type ValidationStatus string

// ValidationStatus constants define the possible outcomes of validation
const (
	ValidationStatusPass ValidationStatus = "pass"
	ValidationStatusFail ValidationStatus = "fail"
)

// CompilationError represents a compilation error
type CompilationError struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column,omitempty"`
	Message string `json:"message"`
}

// CompilationWarning represents a compilation warning
type CompilationWarning struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

// BuildResult represents the result of a build operation
type BuildResult struct {
	Success  bool                 `json:"success"`
	Errors   []CompilationError   `json:"errors,omitempty"`
	Warnings []CompilationWarning `json:"warnings,omitempty"`
	Duration time.Duration        `json:"duration"`
}

// LintIssue represents a linting issue
type LintIssue struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column,omitempty"`
	Severity string `json:"severity"`
	Rule     string `json:"rule"`
	Message  string `json:"message"`
}

// LintResult represents the result of linting
type LintResult struct {
	Success  bool          `json:"success"`
	Issues   []LintIssue   `json:"issues,omitempty"`
	Duration time.Duration `json:"duration"`
}

// TestFailure represents a test failure
type TestFailure struct {
	Package  string `json:"package"`
	Test     string `json:"test"`
	Message  string `json:"message"`
	Location string `json:"location,omitempty"`
}

// TestResult represents the result of test execution
type TestResult struct {
	Success     bool          `json:"success"`
	TotalTests  int           `json:"total_tests"`
	PassedTests int           `json:"passed_tests"`
	FailedTests int           `json:"failed_tests"`
	Failures    []TestFailure `json:"failures,omitempty"`
	Coverage    float64       `json:"coverage"`
	Duration    time.Duration `json:"duration"`
}

// ValidationReport represents a complete validation report
type ValidationReport struct {
	SchemaVersion string           `json:"schema_version"`
	ID            string           `json:"id"`
	OutputID      string           `json:"output_id"`
	BuildResult   BuildResult      `json:"build_result"`
	LintResult    LintResult       `json:"lint_result"`
	TestResult    TestResult       `json:"test_result"`
	OverallStatus ValidationStatus `json:"overall_status"`
	CreatedAt     time.Time        `json:"created_at"`
}

// Validate validates the validation report
func (v *ValidationReport) Validate() error {
	// Check that all durations are positive
	if v.BuildResult.Duration <= 0 {
		return fmt.Errorf("build duration must be > 0")
	}
	if v.LintResult.Duration <= 0 {
		return fmt.Errorf("lint duration must be > 0")
	}
	if v.TestResult.Duration <= 0 {
		return fmt.Errorf("test duration must be > 0")
	}

	// Verify overall status matches individual results
	expectedStatus := v.ComputeOverallStatus()
	if v.OverallStatus != expectedStatus {
		return fmt.Errorf("OverallStatus mismatch: expected %s, got %s", expectedStatus, v.OverallStatus)
	}

	return nil
}

// ComputeOverallStatus computes the overall validation status
func (v *ValidationReport) ComputeOverallStatus() ValidationStatus {
	if v.BuildResult.Success && v.LintResult.Success && v.TestResult.Success {
		return ValidationStatusPass
	}
	return ValidationStatusFail
}
