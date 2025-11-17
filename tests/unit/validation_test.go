package unit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationReport_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name   string
		report *models.ValidationReport
	}{
		{
			name: "passing validation report",
			report: &models.ValidationReport{
				SchemaVersion: "1.0",
				ID:            uuid.New().String(),
				OutputID:      uuid.New().String(),
				BuildResult: models.BuildResult{
					Success:  true,
					Errors:   []models.CompilationError{},
					Warnings: []models.CompilationWarning{},
					Duration: 2 * time.Second,
				},
				LintResult: models.LintResult{
					Success:  true,
					Issues:   []models.LintIssue{},
					Duration: 1 * time.Second,
				},
				TestResult: models.TestResult{
					Success:     true,
					TotalTests:  50,
					PassedTests: 50,
					FailedTests: 0,
					Failures:    []models.TestFailure{},
					Coverage:    87.5,
					Duration:    5 * time.Second,
				},
				OverallStatus: models.ValidationStatusPass,
				CreatedAt:     time.Now().UTC(),
			},
		},
		{
			name: "failing validation report",
			report: &models.ValidationReport{
				SchemaVersion: "1.0",
				ID:            uuid.New().String(),
				OutputID:      uuid.New().String(),
				BuildResult: models.BuildResult{
					Success: false,
					Errors: []models.CompilationError{
						{
							File:    "main.go",
							Line:    10,
							Column:  5,
							Message: "undefined: someFunc",
						},
					},
					Warnings: []models.CompilationWarning{
						{
							File:    "utils.go",
							Line:    25,
							Message: "unused variable 'x'",
						},
					},
					Duration: 1 * time.Second,
				},
				LintResult: models.LintResult{
					Success: false,
					Issues: []models.LintIssue{
						{
							File:     "handler.go",
							Line:     42,
							Severity: "error",
							Rule:     "errcheck",
							Message:  "error return value not checked",
						},
					},
					Duration: 500 * time.Millisecond,
				},
				TestResult: models.TestResult{
					Success:     false,
					TotalTests:  50,
					PassedTests: 45,
					FailedTests: 5,
					Failures: []models.TestFailure{
						{
							Package:  "internal/auth",
							Test:     "TestLogin",
							Message:  "expected true, got false",
							Location: "auth_test.go:45",
						},
					},
					Coverage: 75.0,
					Duration: 3 * time.Second,
				},
				OverallStatus: models.ValidationStatusFail,
				CreatedAt:     time.Now().UTC(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.report)
			require.NoError(t, err)

			var unmarshaled models.ValidationReport
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, tt.report.ID, unmarshaled.ID)
			assert.Equal(t, tt.report.OutputID, unmarshaled.OutputID)
			assert.Equal(t, tt.report.OverallStatus, unmarshaled.OverallStatus)
			assert.Equal(t, tt.report.BuildResult.Success, unmarshaled.BuildResult.Success)
			assert.Equal(t, tt.report.LintResult.Success, unmarshaled.LintResult.Success)
			assert.Equal(t, tt.report.TestResult.Success, unmarshaled.TestResult.Success)
		})
	}
}

func TestValidationReport_Validate(t *testing.T) {
	tests := []struct {
		name    string
		report  *models.ValidationReport
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid report - all pass",
			report: &models.ValidationReport{
				ID:       uuid.New().String(),
				OutputID: uuid.New().String(),
				BuildResult: models.BuildResult{
					Success:  true,
					Duration: 1 * time.Second,
				},
				LintResult: models.LintResult{
					Success:  true,
					Duration: 500 * time.Millisecond,
				},
				TestResult: models.TestResult{
					Success:  true,
					Duration: 2 * time.Second,
				},
				OverallStatus: models.ValidationStatusPass,
			},
			wantErr: false,
		},
		{
			name: "invalid - overall status mismatch (should be fail)",
			report: &models.ValidationReport{
				ID:       uuid.New().String(),
				OutputID: uuid.New().String(),
				BuildResult: models.BuildResult{
					Success:  false,
					Duration: 1 * time.Second,
				},
				LintResult: models.LintResult{
					Success:  true,
					Duration: 500 * time.Millisecond,
				},
				TestResult: models.TestResult{
					Success:  true,
					Duration: 2 * time.Second,
				},
				OverallStatus: models.ValidationStatusPass, // Should be fail
			},
			wantErr: true,
			errMsg:  "OverallStatus mismatch",
		},
		{
			name: "invalid - zero duration",
			report: &models.ValidationReport{
				ID:       uuid.New().String(),
				OutputID: uuid.New().String(),
				BuildResult: models.BuildResult{
					Success:  true,
					Duration: 0, // Invalid
				},
				LintResult: models.LintResult{
					Success:  true,
					Duration: 500 * time.Millisecond,
				},
				TestResult: models.TestResult{
					Success:  true,
					Duration: 2 * time.Second,
				},
				OverallStatus: models.ValidationStatusPass,
			},
			wantErr: true,
			errMsg:  "duration must be > 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.report.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidationReport_ComputeOverallStatus(t *testing.T) {
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
			name:           "tests fail",
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
				BuildResult: models.BuildResult{Success: tt.buildSuccess},
				LintResult:  models.LintResult{Success: tt.lintSuccess},
				TestResult:  models.TestResult{Success: tt.testSuccess},
			}

			status := report.ComputeOverallStatus()
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

func TestBuildResult_JSONMarshaling(t *testing.T) {
	result := &models.BuildResult{
		Success: false,
		Errors: []models.CompilationError{
			{File: "main.go", Line: 10, Column: 5, Message: "syntax error"},
		},
		Warnings: []models.CompilationWarning{
			{File: "utils.go", Line: 20, Message: "unused variable"},
		},
		Duration: 2 * time.Second,
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var unmarshaled models.BuildResult
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, result.Success, unmarshaled.Success)
	assert.Equal(t, len(result.Errors), len(unmarshaled.Errors))
	assert.Equal(t, len(result.Warnings), len(unmarshaled.Warnings))
	assert.Equal(t, result.Duration, unmarshaled.Duration)
}

func TestTestResult_JSONMarshaling(t *testing.T) {
	result := &models.TestResult{
		Success:     false,
		TotalTests:  100,
		PassedTests: 95,
		FailedTests: 5,
		Failures: []models.TestFailure{
			{
				Package:  "internal/auth",
				Test:     "TestLogin",
				Message:  "assertion failed",
				Location: "auth_test.go:42",
			},
		},
		Coverage: 85.5,
		Duration: 10 * time.Second,
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var unmarshaled models.TestResult
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, result.Success, unmarshaled.Success)
	assert.Equal(t, result.TotalTests, unmarshaled.TotalTests)
	assert.Equal(t, result.PassedTests, unmarshaled.PassedTests)
	assert.Equal(t, result.Coverage, unmarshaled.Coverage)
	assert.Equal(t, len(result.Failures), len(unmarshaled.Failures))
}
