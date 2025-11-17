// Package validate provides validation services for generated code.
package validate

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dshills/gocreator/internal/models"
)

// BuildValidator validates Go compilation
type BuildValidator interface {
	Validate(ctx context.Context, projectRoot string) (*models.BuildResult, error)
}

// goBuildValidator implements BuildValidator using go build
type goBuildValidator struct {
	timeout time.Duration
}

// NewBuildValidator creates a new build validator
func NewBuildValidator(timeout time.Duration) BuildValidator {
	if timeout == 0 {
		timeout = 2 * time.Minute // Default 2 minute timeout
	}
	return &goBuildValidator{
		timeout: timeout,
	}
}

// Validate runs go build ./... and parses compilation errors
func (b *goBuildValidator) Validate(ctx context.Context, projectRoot string) (*models.BuildResult, error) {
	start := time.Now()
	result := &models.BuildResult{
		Success:  true,
		Errors:   []models.CompilationError{},
		Warnings: []models.CompilationWarning{},
	}

	// Create context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	// Run go build ./...
	cmd := exec.CommandContext(ctxWithTimeout, "go", "build", "./...")
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(start)

	if err != nil {
		// Check if it's a timeout
		if ctxWithTimeout.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("build timed out after %v", b.timeout)
		}

		// Parse compilation errors from output
		result.Success = false
		errors, warnings := parseCompilationOutput(string(output), projectRoot)
		result.Errors = errors
		result.Warnings = warnings

		// If we couldn't parse any errors, create a generic one
		if len(errors) == 0 && len(warnings) == 0 {
			result.Errors = []models.CompilationError{
				{
					File:    "unknown",
					Line:    0,
					Message: fmt.Sprintf("build failed: %v\nOutput: %s", err, string(output)),
				},
			}
		}
	}

	return result, nil
}

// parseCompilationOutput parses go build output for errors and warnings
// Format: path/to/file.go:line:column: error message
// or: path/to/file.go:line: error message
func parseCompilationOutput(output, projectRoot string) ([]models.CompilationError, []models.CompilationWarning) {
	var errors []models.CompilationError
	var warnings []models.CompilationWarning

	// Regex patterns for Go compiler output
	// Format: ./path/file.go:line:column: message
	// or: ./path/file.go:line: message
	errorPattern := regexp.MustCompile(`^([^:]+):(\d+)(?::(\d+))?: (.+)$`)

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		matches := errorPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		file := matches[1]
		lineNum, _ := strconv.Atoi(matches[2])
		columnNum := 0
		if matches[3] != "" {
			columnNum, _ = strconv.Atoi(matches[3])
		}
		message := matches[4]

		// Make file path relative to project root
		if !filepath.IsAbs(file) {
			file = filepath.Join(projectRoot, file)
		}
		relPath, err := filepath.Rel(projectRoot, file)
		if err == nil {
			file = relPath
		}

		// Determine if it's a warning or error
		// Go compiler warnings are rare, but check for common patterns
		isWarning := strings.Contains(strings.ToLower(message), "warning:") ||
			strings.Contains(strings.ToLower(message), "deprecated")

		if isWarning {
			warnings = append(warnings, models.CompilationWarning{
				File:    file,
				Line:    lineNum,
				Message: message,
			})
		} else {
			errors = append(errors, models.CompilationError{
				File:    file,
				Line:    lineNum,
				Column:  columnNum,
				Message: message,
			})
		}
	}

	return errors, warnings
}
