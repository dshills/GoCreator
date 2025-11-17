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

func TestBuildValidator_Success(t *testing.T) {
	// Create a temporary directory with valid Go code
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.25
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create a simple main.go that compiles
	mainGo := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	// Run build validation
	validator := validate.NewBuildValidator(30 * time.Second)
	result, err := validator.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Empty(t, result.Errors)
	assert.Empty(t, result.Warnings)
	assert.Greater(t, result.Duration, time.Duration(0))
}

func TestBuildValidator_CompilationErrors(t *testing.T) {
	// Create a temporary directory with invalid Go code
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.25
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create main.go with compilation errors
	mainGo := `package main

import "fmt"

func main() {
	fmt.Println("Hello"
	// Missing closing parenthesis
	undeclaredVariable
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	// Run build validation
	validator := validate.NewBuildValidator(30 * time.Second)
	result, err := validator.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.NotEmpty(t, result.Errors)
	assert.Greater(t, result.Duration, time.Duration(0))

	// Verify error details contain file and line information
	hasFileInfo := false
	for _, compErr := range result.Errors {
		if compErr.File != "" && compErr.Line > 0 {
			hasFileInfo = true
			break
		}
	}
	assert.True(t, hasFileInfo, "Expected errors to contain file and line information")
}

func TestBuildValidator_MultipleFiles(t *testing.T) {
	// Create a temporary directory with multiple Go files
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

func main() {
	fmt.Println(Greet())
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	// Create helper.go
	helperGo := `package main

func Greet() string {
	return "Hello from helper"
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "helper.go"), []byte(helperGo), 0644)
	require.NoError(t, err)

	// Run build validation
	validator := validate.NewBuildValidator(30 * time.Second)
	result, err := validator.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Empty(t, result.Errors)
}

func TestBuildValidator_SubPackages(t *testing.T) {
	// Create a temporary directory with subpackages
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.25
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create main.go
	mainGo := `package main

import (
	"fmt"
	"testproject/pkg/utils"
)

func main() {
	fmt.Println(utils.Add(1, 2))
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	// Create pkg/utils directory
	err = os.MkdirAll(filepath.Join(tmpDir, "pkg", "utils"), 0755)
	require.NoError(t, err)

	// Create pkg/utils/math.go
	mathGo := `package utils

func Add(a, b int) int {
	return a + b
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "pkg", "utils", "math.go"), []byte(mathGo), 0644)
	require.NoError(t, err)

	// Run build validation
	validator := validate.NewBuildValidator(30 * time.Second)
	result, err := validator.Validate(context.Background(), tmpDir)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Empty(t, result.Errors)
}

func TestBuildValidator_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

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

	// Use a very short timeout (but this might not trigger unless build is slow)
	// This test is more about ensuring timeout mechanism works
	validator := validate.NewBuildValidator(1 * time.Nanosecond)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = validator.Validate(ctx, tmpDir)
	// We expect either success (build was fast) or timeout error
	// This test mainly ensures timeout mechanism doesn't panic
	if err != nil {
		assert.Contains(t, err.Error(), "timed out")
	}
}

func TestBuildValidator_InvalidProjectRoot(t *testing.T) {
	validator := validate.NewBuildValidator(30 * time.Second)

	// Test with non-existent directory
	result, err := validator.Validate(context.Background(), "/nonexistent/path")

	require.NoError(t, err) // Validator doesn't error on invalid path
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.NotEmpty(t, result.Errors)
}
