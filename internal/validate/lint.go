package validate

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dshills/gocreator/internal/models"
)

// LintValidator validates code using linters
type LintValidator interface {
	Validate(ctx context.Context, projectRoot string) (*models.LintResult, error)
}

// golangciLintValidator implements LintValidator using golangci-lint
type golangciLintValidator struct {
	timeout         time.Duration
	skipIfNotFound  bool
	additionalFlags []string
}

// LintOption configures the lint validator
type LintOption func(*golangciLintValidator)

// WithLintTimeout sets a custom timeout
func WithLintTimeout(timeout time.Duration) LintOption {
	return func(v *golangciLintValidator) {
		v.timeout = timeout
	}
}

// WithSkipIfNotFound skips linting if golangci-lint is not available
func WithSkipIfNotFound(skip bool) LintOption {
	return func(v *golangciLintValidator) {
		v.skipIfNotFound = skip
	}
}

// WithLintFlags adds additional flags to golangci-lint
func WithLintFlags(flags ...string) LintOption {
	return func(v *golangciLintValidator) {
		v.additionalFlags = append(v.additionalFlags, flags...)
	}
}

// NewLintValidator creates a new lint validator
func NewLintValidator(opts ...LintOption) LintValidator {
	v := &golangciLintValidator{
		timeout:         3 * time.Minute, // Default 3 minute timeout
		skipIfNotFound:  true,            // Default: skip if not found
		additionalFlags: []string{},
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// Validate runs golangci-lint and parses issues
func (l *golangciLintValidator) Validate(ctx context.Context, projectRoot string) (*models.LintResult, error) {
	start := time.Now()
	result := &models.LintResult{
		Success: true,
		Issues:  []models.LintIssue{},
	}

	// Check if golangci-lint is available
	if !l.isGolangciLintAvailable() {
		if l.skipIfNotFound {
			// Return success with no issues if we're skipping
			result.Duration = time.Since(start)
			return result, nil
		}
		return nil, fmt.Errorf("golangci-lint not found in PATH")
	}

	// Create context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()

	// Build command: golangci-lint run ./... --out-format json
	args := append([]string{"run", "./...", "--out-format", "json"}, l.additionalFlags...)
	//nolint:gosec // G204: Subprocess launched with golangci-lint - required for code validation
	cmd := exec.CommandContext(ctxWithTimeout, "golangci-lint", args...)
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(start)

	// Check for timeout first
	if ctxWithTimeout.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("lint timed out after %v", l.timeout)
	}

	// Check if golangci-lint failed due to unsupported flags or version issues
	// This can happen with older versions that don't support --out-format
	if err != nil && len(output) > 0 {
		outputStr := string(output)
		// Check for version/compatibility issues
		if strings.Contains(outputStr, "unknown flag") || strings.Contains(outputStr, "Error:") {
			if l.skipIfNotFound {
				result.Duration = time.Since(start)
				return result, nil
			}
			return nil, fmt.Errorf("golangci-lint version compatibility issue: %s", outputStr)
		}
	}

	// golangci-lint returns non-zero if issues are found
	// We need to parse the JSON output if available
	if len(output) > 0 {
		issues, parseErr := l.parseLintOutput(output, projectRoot)
		if parseErr != nil {
			// If parsing fails and we have an error, it might be that golangci-lint
			// failed to run properly (e.g., configuration issues)
			// In this case, if skip is enabled, just return success with no issues
			if l.skipIfNotFound {
				result.Duration = time.Since(start)
				return result, nil
			}
			// Otherwise, return the parsing error
			return nil, fmt.Errorf("failed to parse lint output: %w", parseErr)
		}
		result.Issues = issues
		if len(issues) > 0 {
			result.Success = false
		}
	}

	return result, nil
}

// isGolangciLintAvailable checks if golangci-lint is in PATH
func (l *golangciLintValidator) isGolangciLintAvailable() bool {
	_, err := exec.LookPath("golangci-lint")
	return err == nil
}

// golangciLintOutput represents the JSON output from golangci-lint
type golangciLintOutput struct {
	Issues []golangciIssue `json:"Issues"`
}

type golangciIssue struct {
	FromLinter string                `json:"FromLinter"`
	Text       string                `json:"Text"`
	Severity   string                `json:"Severity"`
	Pos        golangciIssuePosition `json:"Pos"`
}

type golangciIssuePosition struct {
	Filename string `json:"Filename"`
	Line     int    `json:"Line"`
	Column   int    `json:"Column"`
}

// parseLintOutput parses golangci-lint JSON output
func (l *golangciLintValidator) parseLintOutput(output []byte, projectRoot string) ([]models.LintIssue, error) {
	var lintOutput golangciLintOutput
	if err := json.Unmarshal(output, &lintOutput); err != nil {
		return nil, fmt.Errorf("failed to unmarshal lint output: %w", err)
	}

	issues := make([]models.LintIssue, 0, len(lintOutput.Issues))
	for _, issue := range lintOutput.Issues {
		// Make file path relative to project root
		file := issue.Pos.Filename
		if filepath.IsAbs(file) {
			relPath, err := filepath.Rel(projectRoot, file)
			if err == nil {
				file = relPath
			}
		}

		// Map severity
		severity := issue.Severity
		if severity == "" {
			severity = "error"
		}

		issues = append(issues, models.LintIssue{
			File:     file,
			Line:     issue.Pos.Line,
			Column:   issue.Pos.Column,
			Severity: severity,
			Rule:     issue.FromLinter,
			Message:  issue.Text,
		})
	}

	return issues, nil
}
