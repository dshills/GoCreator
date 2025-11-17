package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dshills/gocreator/internal/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_Validate_SuccessfulProject(t *testing.T) {
	// Create a complete, valid Go project
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.25
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create main.go
	mainGo := `package main

import "fmt"

func Add(a, b int) int {
	return a + b
}

func main() {
	result := Add(2, 3)
	fmt.Printf("Result: %d\n", result)
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	// Create tests
	testGo := `package main

import "testing"

func TestAdd(t *testing.T) {
	result := Add(2, 3)
	if result != 5 {
		t.Errorf("Add(2, 3) = %d; want 5", result)
	}
}

func TestAddZero(t *testing.T) {
	result := Add(0, 0)
	if result != 0 {
		t.Errorf("Add(0, 0) = %d; want 0", result)
	}
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte(testGo), 0644)
	require.NoError(t, err)

	// Create engine and run validation
	engine := validate.NewEngine()
	report, err := engine.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.NotEmpty(t, report.ID)
	assert.True(t, report.BuildResult.Success)
	assert.True(t, report.LintResult.Success)
	assert.True(t, report.TestResult.Success)
	assert.Equal(t, "pass", string(report.OverallStatus))
}

func TestEngine_Validate_BuildFailure(t *testing.T) {
	// Create a project with build errors
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.25
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create invalid main.go
	mainGo := `package main

func main() {
	undefinedFunction()
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	// Create engine and run validation
	engine := validate.NewEngine()
	report, err := engine.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.False(t, report.BuildResult.Success)
	assert.NotEmpty(t, report.BuildResult.Errors)
	assert.Equal(t, "fail", string(report.OverallStatus))
}

func TestEngine_Validate_TestFailure(t *testing.T) {
	// Create a project with failing tests
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.25
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create main.go
	mainGo := `package main

func Add(a, b int) int {
	return a + b
}

func main() {}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	// Create failing test
	testGo := `package main

import "testing"

func TestAddFail(t *testing.T) {
	result := Add(2, 3)
	if result != 6 { // This will fail
		t.Errorf("Add(2, 3) = %d; want 6", result)
	}
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte(testGo), 0644)
	require.NoError(t, err)

	// Create engine and run validation
	engine := validate.NewEngine()
	report, err := engine.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.True(t, report.BuildResult.Success)
	assert.False(t, report.TestResult.Success)
	assert.NotEmpty(t, report.TestResult.Failures)
	assert.Equal(t, "fail", string(report.OverallStatus))
}

func TestEngine_ValidateAndSave(t *testing.T) {
	// Create a valid project
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.25
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create main.go
	mainGo := `package main

func main() {}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	// Create test
	testGo := `package main

import "testing"

func TestMain(t *testing.T) {}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte(testGo), 0644)
	require.NoError(t, err)

	// Create engine and run validation with save
	reportPath := filepath.Join(tmpDir, "validation_report.json")
	engine := validate.NewEngine()
	report, err := engine.ValidateAndSave(context.Background(), tmpDir, reportPath, "output-123")

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, "output-123", report.OutputID)

	// Verify file was created
	_, err = os.Stat(reportPath)
	require.NoError(t, err)

	// Load and verify
	gen := validate.NewReportGenerator()
	loadedReport, err := gen.Load(reportPath)
	require.NoError(t, err)
	assert.Equal(t, report.ID, loadedReport.ID)
}

func TestEngine_WithCustomValidators(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.25
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create main.go
	mainGo := `package main

func main() {}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	// Create test
	testGo := `package main

import "testing"

func TestMain(t *testing.T) {}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte(testGo), 0644)
	require.NoError(t, err)

	// Create engine with custom validators
	engine := validate.NewEngine(
		validate.WithBuildValidator(validate.NewBuildValidator(1*time.Minute)),
		validate.WithLintValidator(validate.NewLintValidator(validate.WithSkipIfNotFound(true))),
		validate.WithTestValidator(validate.NewTestValidator(validate.WithTestTimeout(2*time.Minute))),
	)

	report, err := engine.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, report)
}

func TestEngine_ConcurrentValidation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.25
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create main.go
	mainGo := `package main

func main() {}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	// Create test
	testGo := `package main

import "testing"

func TestMain(t *testing.T) {}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte(testGo), 0644)
	require.NoError(t, err)

	// Test with concurrent validation enabled
	engineConcurrent := validate.NewEngine(validate.WithConcurrentValidation(true))
	reportConcurrent, err := engineConcurrent.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, reportConcurrent)

	// Test with concurrent validation disabled
	engineSequential := validate.NewEngine(validate.WithConcurrentValidation(false))
	reportSequential, err := engineSequential.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, reportSequential)

	// Both should produce the same result
	assert.Equal(t, reportConcurrent.BuildResult.Success, reportSequential.BuildResult.Success)
	assert.Equal(t, reportConcurrent.LintResult.Success, reportSequential.LintResult.Success)
	assert.Equal(t, reportConcurrent.TestResult.Success, reportSequential.TestResult.Success)
}

func TestEngine_EmptyProjectRoot(t *testing.T) {
	engine := validate.NewEngine()
	_, err := engine.Validate(context.Background(), "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "projectRoot cannot be empty")
}

func TestEngine_ValidateWithOutputID(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.25
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create main.go
	mainGo := `package main

func main() {}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	// Create test
	testGo := `package main

import "testing"

func TestMain(t *testing.T) {}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte(testGo), 0644)
	require.NoError(t, err)

	// Validate with specific output ID
	engine := validate.NewEngine()
	report, err := engine.ValidateWithOutputID(context.Background(), tmpDir, "custom-output-id")

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, "custom-output-id", report.OutputID)
}
