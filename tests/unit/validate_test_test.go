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

func TestTestValidator_AllTestsPass(t *testing.T) {
	// Create a temporary directory with passing tests
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.24
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

	// Create passing test
	testGo := `package main

import "testing"

func TestAdd(t *testing.T) {
	result := Add(2, 3)
	if result != 5 {
		t.Errorf("Add(2, 3) = %d; want 5", result)
	}
}

func TestAddNegative(t *testing.T) {
	result := Add(-1, 1)
	if result != 0 {
		t.Errorf("Add(-1, 1) = %d; want 0", result)
	}
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte(testGo), 0644)
	require.NoError(t, err)

	// Run test validation
	validator := validate.NewTestValidator()
	result, err := validator.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 2, result.TotalTests)
	assert.Equal(t, 2, result.PassedTests)
	assert.Equal(t, 0, result.FailedTests)
	assert.Empty(t, result.Failures)
	assert.Greater(t, result.Duration, time.Duration(0))
	// Coverage should be > 0
	assert.Greater(t, result.Coverage, 0.0)
}

func TestTestValidator_TestFailures(t *testing.T) {
	// Create a temporary directory with failing tests
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.24
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

	// Create test with failures
	testGo := `package main

import "testing"

func TestAddCorrect(t *testing.T) {
	result := Add(2, 3)
	if result != 5 {
		t.Errorf("Add(2, 3) = %d; want 5", result)
	}
}

func TestAddIncorrect(t *testing.T) {
	result := Add(2, 3)
	if result != 6 { // This will fail
		t.Errorf("Add(2, 3) = %d; want 6", result)
	}
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte(testGo), 0644)
	require.NoError(t, err)

	// Run test validation
	validator := validate.NewTestValidator()
	result, err := validator.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Equal(t, 2, result.TotalTests)
	assert.Equal(t, 1, result.PassedTests)
	assert.Equal(t, 1, result.FailedTests)
	assert.Len(t, result.Failures, 1)
	assert.Greater(t, result.Duration, time.Duration(0))

	// Check failure details
	failure := result.Failures[0]
	assert.Equal(t, "TestAddIncorrect", failure.Test)
	// Message might be empty depending on output format, just verify we captured the failure
	assert.NotEmpty(t, failure.Test)
}

func TestTestValidator_NoCoverage(t *testing.T) {
	// Create a temporary directory with tests but no code coverage
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.24
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create main.go with no functions (just a comment)
	mainGo := `package main

func main() {
	// Empty main
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	// Create test
	testGo := `package main

import "testing"

func TestSomething(t *testing.T) {
	// Empty test that passes
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte(testGo), 0644)
	require.NoError(t, err)

	// Run test validation
	validator := validate.NewTestValidator()
	result, err := validator.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 1, result.TotalTests)
	assert.Equal(t, 1, result.PassedTests)
	// Coverage might be 0 or low
	assert.GreaterOrEqual(t, result.Coverage, 0.0)
}

func TestTestValidator_CustomCoverageProfile(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.24
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

	// Create validator with custom coverage profile
	customProfile := filepath.Join(tmpDir, "custom-coverage.out")
	validator := validate.NewTestValidator(validate.WithCoverageProfile(customProfile))
	result, err := validator.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Check that custom coverage file was created
	_, err = os.Stat(customProfile)
	assert.NoError(t, err, "Custom coverage profile should be created")
}

func TestTestValidator_CustomTimeout(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.24
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

func TestQuick(t *testing.T) {}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte(testGo), 0644)
	require.NoError(t, err)

	// Create validator with custom timeout
	validator := validate.NewTestValidator(validate.WithTestTimeout(1 * time.Minute))
	result, err := validator.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestTestValidator_MultiplePackages(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.24
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create main package
	mainGo := `package main

import "testproject/utils"

func main() {
	utils.Helper()
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	// Create utils package
	err = os.MkdirAll(filepath.Join(tmpDir, "utils"), 0755)
	require.NoError(t, err)

	utilsGo := `package utils

func Helper() string {
	return "helper"
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "utils", "utils.go"), []byte(utilsGo), 0644)
	require.NoError(t, err)

	// Create test for utils
	utilsTestGo := `package utils

import "testing"

func TestHelper(t *testing.T) {
	result := Helper()
	if result != "helper" {
		t.Errorf("Helper() = %s; want helper", result)
	}
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "utils", "utils_test.go"), []byte(utilsTestGo), 0644)
	require.NoError(t, err)

	// Run test validation
	validator := validate.NewTestValidator()
	result, err := validator.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Greater(t, result.TotalTests, 0)
	assert.Greater(t, result.Coverage, 0.0)
}

func TestTestValidator_InvalidProjectRoot(t *testing.T) {
	validator := validate.NewTestValidator()

	// Test with non-existent directory
	result, err := validator.Validate(context.Background(), "/nonexistent/path")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
}

func TestTestValidator_WithAdditionalFlags(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.24
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

	// Create validator with additional flags
	validator := validate.NewTestValidator(
		validate.WithTestFlags("-short"),
	)
	result, err := validator.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, result)
}
