package validate

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dshills/gocreator/internal/models"
)

// TestValidator validates tests
type TestValidator interface {
	Validate(ctx context.Context, projectRoot string) (*models.TestResult, error)
}

// goTestValidator implements TestValidator using go test
type goTestValidator struct {
	timeout         time.Duration
	coverageProfile string
	additionalFlags []string
}

// TestOption configures the test validator
type TestOption func(*goTestValidator)

// WithTestTimeout sets a custom timeout
func WithTestTimeout(timeout time.Duration) TestOption {
	return func(v *goTestValidator) {
		v.timeout = timeout
	}
}

// WithCoverageProfile sets the coverage profile file path
func WithCoverageProfile(path string) TestOption {
	return func(v *goTestValidator) {
		v.coverageProfile = path
	}
}

// WithTestFlags adds additional flags to go test
func WithTestFlags(flags ...string) TestOption {
	return func(v *goTestValidator) {
		v.additionalFlags = append(v.additionalFlags, flags...)
	}
}

// NewTestValidator creates a new test validator
func NewTestValidator(opts ...TestOption) TestValidator {
	v := &goTestValidator{
		timeout:         5 * time.Minute, // Default 5 minute timeout
		coverageProfile: "coverage.out",
		additionalFlags: []string{},
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// Validate runs go test ./... with coverage and parses results
func (t *goTestValidator) Validate(ctx context.Context, projectRoot string) (*models.TestResult, error) {
	start := time.Now()
	result := &models.TestResult{
		Success:     true,
		TotalTests:  0,
		PassedTests: 0,
		FailedTests: 0,
		Failures:    []models.TestFailure{},
		Coverage:    0.0,
	}

	// Create context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	// Build coverage file path (absolute)
	coverageFile := t.coverageProfile
	if !filepath.IsAbs(coverageFile) {
		coverageFile = filepath.Join(projectRoot, coverageFile)
	}

	// Build command: go test ./... -coverprofile=coverage.out -v
	args := []string{"test", "./...", "-coverprofile=" + coverageFile, "-v"}
	args = append(args, t.additionalFlags...)

	cmd := exec.CommandContext(ctxWithTimeout, "go", args...)
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(start)

	// Check for timeout
	if ctxWithTimeout.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("tests timed out after %v", t.timeout)
	}

	// Parse test output
	totalTests, passedTests, failures := parseTestOutput(string(output))
	result.TotalTests = totalTests
	result.PassedTests = passedTests
	result.FailedTests = len(failures)
	result.Failures = failures

	if err != nil || len(failures) > 0 {
		result.Success = false
	}

	// Parse coverage
	coverage, coverageErr := parseCoverage(coverageFile)
	if coverageErr == nil {
		result.Coverage = coverage
	}
	// Don't fail validation if coverage parsing fails, just set to 0

	return result, nil
}

// parseTestOutput parses go test -v output for test results
// Format:
// === RUN   TestFoo
// --- PASS: TestFoo (0.00s)
// === RUN   TestBar
// --- FAIL: TestBar (0.00s)
//
//	file_test.go:42: error message
func parseTestOutput(output string) (totalTests int, passedTests int, failures []models.TestFailure) {
	scanner := bufio.NewScanner(strings.NewReader(output))

	// Pattern for test results
	resultPattern := regexp.MustCompile(`^--- (PASS|FAIL|SKIP): (.+?) \(`)
	// Pattern for failure location
	locationPattern := regexp.MustCompile(`^\s+([^:]+):(\d+): (.+)$`)

	var currentPackage string
	var currentFailure *models.TestFailure

	for scanner.Scan() {
		line := scanner.Text()

		// Check for package indication
		if strings.HasPrefix(line, "?") || strings.HasPrefix(line, "ok") || strings.HasPrefix(line, "FAIL") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				pkg := parts[1]
				if pkg != "" && !strings.HasPrefix(pkg, "[") {
					currentPackage = pkg
				}
			}
		}

		// Check for test results
		matches := resultPattern.FindStringSubmatch(line)
		if matches != nil {
			// Save previous failure if exists
			if currentFailure != nil {
				failures = append(failures, *currentFailure)
				currentFailure = nil
			}

			status := matches[1]
			testName := matches[2]

			totalTests++

			if status == "PASS" {
				passedTests++
			} else if status == "FAIL" {
				// Start tracking a new failure
				currentFailure = &models.TestFailure{
					Package:  currentPackage,
					Test:     testName,
					Message:  "",
					Location: "",
				}
			}
			// Skip SKIP status from totals
			if status == "SKIP" {
				totalTests-- // Don't count skipped tests
			}
		} else {
			// Check for failure details (location and message) only if not a result line
			if currentFailure != nil {
				locMatches := locationPattern.FindStringSubmatch(line)
				if locMatches != nil {
					file := locMatches[1]
					lineNum := locMatches[2]
					message := locMatches[3]

					currentFailure.Location = fmt.Sprintf("%s:%s", file, lineNum)
					if currentFailure.Message == "" {
						currentFailure.Message = message
					} else {
						currentFailure.Message += "\n" + message
					}
				}
			}
		}
	}

	// Save last failure if exists
	if currentFailure != nil {
		failures = append(failures, *currentFailure)
	}

	return totalTests, passedTests, failures
}

// parseCoverage parses coverage.out file and calculates percentage
// Format:
// mode: set
// package/file.go:10.2,12.3 2 1
// package/file.go:14.5,16.8 3 0
func parseCoverage(coverageFile string) (float64, error) {
	file, err := os.Open(coverageFile)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var totalStatements int
	var coveredStatements int

	scanner := bufio.NewScanner(file)
	// Pattern: file:startLine.startCol,endLine.endCol numStatements count
	coveragePattern := regexp.MustCompile(`^[^:]+:\d+\.\d+,\d+\.\d+ (\d+) (\d+)`)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip mode line
		if strings.HasPrefix(line, "mode:") {
			continue
		}

		matches := coveragePattern.FindStringSubmatch(line)
		if matches != nil {
			stmts, _ := strconv.Atoi(matches[1])
			count, _ := strconv.Atoi(matches[2])

			totalStatements += stmts
			if count > 0 {
				coveredStatements += stmts
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	if totalStatements == 0 {
		return 0, nil
	}

	percentage := (float64(coveredStatements) / float64(totalStatements)) * 100
	return percentage, nil
}
