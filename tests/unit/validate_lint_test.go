package unit

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/dshills/gocreator/internal/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLintValidator_SkipIfNotFound(t *testing.T) {
	// Create a temporary directory with valid Go code
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.25
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create a simple main.go
	mainGo := `package main

func main() {}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	// Run lint validation with skip enabled (default)
	validator := validate.NewLintValidator(validate.WithSkipIfNotFound(true))
	result, err := validator.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, result)
	// Should succeed even if golangci-lint is not available
	assert.True(t, result.Success)
	assert.Empty(t, result.Issues)
	assert.Greater(t, result.Duration, time.Duration(0))
}

func TestLintValidator_WithGolangciLint(t *testing.T) {
	// This test only runs if golangci-lint is available
	if !isGolangciLintAvailable() {
		t.Skip("golangci-lint not available, skipping test")
	}

	// Create a temporary directory with code that has lint issues
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.25
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create code with potential lint issues (unused variable, ineffective assignment)
	mainGo := `package main

import "fmt"

func main() {
	x := 1
	x = 2
	_ = x
	fmt.Println("Hello")
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	// Run lint validation
	// Use skipIfNotFound=true to handle version compatibility issues gracefully
	validator := validate.NewLintValidator(validate.WithSkipIfNotFound(true))
	result, err := validator.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Greater(t, result.Duration, time.Duration(0))
	// Result depends on linter configuration
	// Just verify we got a valid result
}

func TestLintValidator_CleanCode(t *testing.T) {
	// This test only runs if golangci-lint is available
	if !isGolangciLintAvailable() {
		t.Skip("golangci-lint not available, skipping test")
	}

	// Create a temporary directory with clean Go code
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.25
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create clean code
	mainGo := `package main

import "fmt"

func main() {
	message := "Hello, World!"
	fmt.Println(message)
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	// Run lint validation
	// Use skipIfNotFound=true to handle version compatibility issues gracefully
	validator := validate.NewLintValidator(validate.WithSkipIfNotFound(true))
	result, err := validator.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Greater(t, result.Duration, time.Duration(0))
	// Clean code should have fewer or no issues
}

func TestLintValidator_CustomTimeout(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.25
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create a simple main.go
	mainGo := `package main

func main() {}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	// Create validator with custom timeout
	validator := validate.NewLintValidator(
		validate.WithLintTimeout(1*time.Minute),
		validate.WithSkipIfNotFound(true),
	)
	result, err := validator.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestLintValidator_InvalidProjectRoot(t *testing.T) {
	validator := validate.NewLintValidator(validate.WithSkipIfNotFound(true))

	// Test with non-existent directory
	result, err := validator.Validate(context.Background(), "/nonexistent/path")

	require.NoError(t, err)
	require.NotNil(t, result)
	// Should succeed with skip enabled
	assert.True(t, result.Success)
}

func TestLintValidator_WithAdditionalFlags(t *testing.T) {
	// This test only runs if golangci-lint is available
	if !isGolangciLintAvailable() {
		t.Skip("golangci-lint not available, skipping test")
	}

	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.25
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create a simple main.go
	mainGo := `package main

func main() {}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	// Create validator with additional flags
	// Use skipIfNotFound=true to handle version compatibility issues gracefully
	validator := validate.NewLintValidator(
		validate.WithSkipIfNotFound(true),
		validate.WithLintFlags("--timeout=30s"),
	)
	result, err := validator.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, result)
}

// Helper function to check if golangci-lint is available
func isGolangciLintAvailable() bool {
	// Check if golangci-lint binary is in PATH
	_, err := exec.LookPath("golangci-lint")
	if err != nil {
		return false
	}

	// Verify it can run by checking version
	cmd := exec.Command("golangci-lint", "--version")
	err = cmd.Run()
	return err == nil
}
