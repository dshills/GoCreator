package unit

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/internal/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportGenerator_Generate_Success(t *testing.T) {
	gen := validate.NewReportGenerator()

	buildResult := &models.BuildResult{
		Success:  true,
		Errors:   []models.CompilationError{},
		Warnings: []models.CompilationWarning{},
		Duration: 1 * time.Second,
	}

	lintResult := &models.LintResult{
		Success:  true,
		Issues:   []models.LintIssue{},
		Duration: 2 * time.Second,
	}

	testResult := &models.TestResult{
		Success:     true,
		TotalTests:  10,
		PassedTests: 10,
		FailedTests: 0,
		Failures:    []models.TestFailure{},
		Coverage:    85.5,
		Duration:    3 * time.Second,
	}

	report, err := gen.Generate(buildResult, lintResult, testResult, "output-123")

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.NotEmpty(t, report.ID)
	assert.Equal(t, "output-123", report.OutputID)
	assert.Equal(t, models.ValidationStatusPass, report.OverallStatus)
	assert.Equal(t, "1.0", report.SchemaVersion)
	assert.NotZero(t, report.CreatedAt)
	assert.True(t, report.BuildResult.Success)
	assert.True(t, report.LintResult.Success)
	assert.True(t, report.TestResult.Success)
}

func TestReportGenerator_Generate_WithFailures(t *testing.T) {
	gen := validate.NewReportGenerator()

	buildResult := &models.BuildResult{
		Success: false,
		Errors: []models.CompilationError{
			{File: "main.go", Line: 10, Message: "undefined: foo"},
		},
		Warnings: []models.CompilationWarning{},
		Duration: 1 * time.Second,
	}

	lintResult := &models.LintResult{
		Success: false,
		Issues: []models.LintIssue{
			{File: "main.go", Line: 5, Severity: "error", Rule: "unused", Message: "unused variable"},
		},
		Duration: 2 * time.Second,
	}

	testResult := &models.TestResult{
		Success:     false,
		TotalTests:  10,
		PassedTests: 8,
		FailedTests: 2,
		Failures: []models.TestFailure{
			{Package: "main", Test: "TestFoo", Message: "expected 5, got 3"},
		},
		Coverage: 65.0,
		Duration: 3 * time.Second,
	}

	report, err := gen.Generate(buildResult, lintResult, testResult, "output-456")

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, models.ValidationStatusFail, report.OverallStatus)
	assert.False(t, report.BuildResult.Success)
	assert.False(t, report.LintResult.Success)
	assert.False(t, report.TestResult.Success)
	assert.Len(t, report.BuildResult.Errors, 1)
	assert.Len(t, report.LintResult.Issues, 1)
	assert.Len(t, report.TestResult.Failures, 1)
}

func TestReportGenerator_Generate_NilInputs(t *testing.T) {
	gen := validate.NewReportGenerator()

	tests := []struct {
		name        string
		buildResult *models.BuildResult
		lintResult  *models.LintResult
		testResult  *models.TestResult
		wantErr     bool
	}{
		{
			name:        "nil build result",
			buildResult: nil,
			lintResult:  &models.LintResult{Duration: 1 * time.Second},
			testResult:  &models.TestResult{Duration: 1 * time.Second},
			wantErr:     true,
		},
		{
			name:        "nil lint result",
			buildResult: &models.BuildResult{Duration: 1 * time.Second},
			lintResult:  nil,
			testResult:  &models.TestResult{Duration: 1 * time.Second},
			wantErr:     true,
		},
		{
			name:        "nil test result",
			buildResult: &models.BuildResult{Duration: 1 * time.Second},
			lintResult:  &models.LintResult{Duration: 1 * time.Second},
			testResult:  nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := gen.Generate(tt.buildResult, tt.lintResult, tt.testResult, "output-123")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestReportGenerator_SaveAndLoad(t *testing.T) {
	gen := validate.NewReportGenerator()
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "validation_report.json")

	// Create a report
	buildResult := &models.BuildResult{
		Success:  true,
		Duration: 1 * time.Second,
	}
	lintResult := &models.LintResult{
		Success:  true,
		Duration: 1 * time.Second,
	}
	testResult := &models.TestResult{
		Success:  true,
		Duration: 1 * time.Second,
		Coverage: 90.0,
	}

	originalReport, err := gen.Generate(buildResult, lintResult, testResult, "output-789")
	require.NoError(t, err)

	// Save the report
	err = gen.Save(originalReport, reportPath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(reportPath)
	require.NoError(t, err)

	// Load the report
	loadedReport, err := gen.Load(reportPath)
	require.NoError(t, err)
	require.NotNil(t, loadedReport)

	// Compare reports
	assert.Equal(t, originalReport.ID, loadedReport.ID)
	assert.Equal(t, originalReport.OutputID, loadedReport.OutputID)
	assert.Equal(t, originalReport.OverallStatus, loadedReport.OverallStatus)
	assert.Equal(t, originalReport.SchemaVersion, loadedReport.SchemaVersion)
	assert.Equal(t, originalReport.BuildResult.Success, loadedReport.BuildResult.Success)
	assert.Equal(t, originalReport.LintResult.Success, loadedReport.LintResult.Success)
	assert.Equal(t, originalReport.TestResult.Success, loadedReport.TestResult.Success)
}

func TestReportGenerator_Save_NilReport(t *testing.T) {
	gen := validate.NewReportGenerator()
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "report.json")

	err := gen.Save(nil, reportPath)
	assert.Error(t, err)
}

func TestReportGenerator_Load_InvalidFile(t *testing.T) {
	gen := validate.NewReportGenerator()

	// Test with non-existent file
	_, err := gen.Load("/nonexistent/report.json")
	assert.Error(t, err)

	// Test with invalid JSON
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "invalid.json")
	err = os.WriteFile(invalidPath, []byte("not valid json"), 0644)
	require.NoError(t, err)

	_, err = gen.Load(invalidPath)
	assert.Error(t, err)
}

func TestFileErrorMap(t *testing.T) {
	report := &models.ValidationReport{
		BuildResult: models.BuildResult{
			Errors: []models.CompilationError{
				{File: "main.go", Line: 10, Message: "error 1"},
				{File: "utils.go", Line: 20, Message: "error 2"},
				{File: "main.go", Line: 15, Message: "error 3"},
			},
			Warnings: []models.CompilationWarning{
				{File: "main.go", Line: 5, Message: "warning 1"},
			},
		},
		LintResult: models.LintResult{
			Issues: []models.LintIssue{
				{File: "main.go", Line: 8, Severity: "error", Message: "lint issue 1"},
				{File: "helper.go", Line: 3, Severity: "warning", Message: "lint issue 2"},
			},
		},
	}

	fileMap := validate.FileErrorMap(report)

	// Check main.go has errors from multiple sources
	mainErrors := fileMap["main.go"]
	assert.Len(t, mainErrors.BuildErrors, 2)
	assert.Len(t, mainErrors.BuildWarnings, 1)
	assert.Len(t, mainErrors.LintIssues, 1)
	assert.True(t, mainErrors.HasErrors())

	// Check utils.go
	utilsErrors := fileMap["utils.go"]
	assert.Len(t, utilsErrors.BuildErrors, 1)
	assert.True(t, utilsErrors.HasErrors())

	// Check helper.go
	helperErrors := fileMap["helper.go"]
	assert.Len(t, helperErrors.LintIssues, 1)
	assert.False(t, helperErrors.HasErrors()) // Only warning, not error
}

func TestGenerateSummary(t *testing.T) {
	report := &models.ValidationReport{
		OverallStatus: models.ValidationStatusFail,
		BuildResult: models.BuildResult{
			Success: false,
			Errors: []models.CompilationError{
				{File: "main.go", Line: 10, Message: "error 1"},
				{File: "utils.go", Line: 20, Message: "error 2"},
			},
			Warnings: []models.CompilationWarning{
				{File: "main.go", Line: 5, Message: "warning 1"},
			},
			Duration: 2 * time.Second,
		},
		LintResult: models.LintResult{
			Success: false,
			Issues: []models.LintIssue{
				{File: "main.go", Line: 8, Severity: "error", Message: "issue 1"},
				{File: "helper.go", Line: 3, Severity: "warning", Message: "issue 2"},
				{File: "main.go", Line: 12, Severity: "error", Message: "issue 3"},
			},
			Duration: 3 * time.Second,
		},
		TestResult: models.TestResult{
			Success:     false,
			TotalTests:  20,
			PassedTests: 18,
			FailedTests: 2,
			Coverage:    75.5,
			Duration:    5 * time.Second,
		},
	}

	summary := validate.GenerateSummary(report)

	assert.Equal(t, models.ValidationStatusFail, summary.OverallStatus)
	assert.False(t, summary.BuildPassed)
	assert.False(t, summary.LintPassed)
	assert.False(t, summary.TestsPassed)
	assert.Equal(t, 2, summary.TotalErrors)
	assert.Equal(t, 1, summary.TotalWarnings)
	assert.Equal(t, 3, summary.TotalLintIssues)
	assert.Equal(t, 75.5, summary.TestCoverage)
	assert.Equal(t, 10*time.Second, summary.TotalDuration)
}

func TestValidationEngineReport_ComputeOverallStatus(t *testing.T) {
	tests := []struct {
		name           string
		buildSuccess   bool
		lintSuccess    bool
		testSuccess    bool
		expectedStatus models.ValidationStatus
	}{
		{
			name:           "all pass",
			buildSuccess:   true,
			lintSuccess:    true,
			testSuccess:    true,
			expectedStatus: models.ValidationStatusPass,
		},
		{
			name:           "build fails",
			buildSuccess:   false,
			lintSuccess:    true,
			testSuccess:    true,
			expectedStatus: models.ValidationStatusFail,
		},
		{
			name:           "lint fails",
			buildSuccess:   true,
			lintSuccess:    false,
			testSuccess:    true,
			expectedStatus: models.ValidationStatusFail,
		},
		{
			name:           "test fails",
			buildSuccess:   true,
			lintSuccess:    true,
			testSuccess:    false,
			expectedStatus: models.ValidationStatusFail,
		},
		{
			name:           "all fail",
			buildSuccess:   false,
			lintSuccess:    false,
			testSuccess:    false,
			expectedStatus: models.ValidationStatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &models.ValidationReport{
				BuildResult: models.BuildResult{
					Success:  tt.buildSuccess,
					Duration: 1 * time.Second,
				},
				LintResult: models.LintResult{
					Success:  tt.lintSuccess,
					Duration: 1 * time.Second,
				},
				TestResult: models.TestResult{
					Success:  tt.testSuccess,
					Duration: 1 * time.Second,
				},
			}

			status := report.ComputeOverallStatus()
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}
