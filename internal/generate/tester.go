package generate

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/pkg/llm"
	"github.com/rs/zerolog/log"
)

// Tester generates test files for generated code
type Tester interface {
	// Generate creates test files for the specified packages
	Generate(ctx context.Context, packages []string, plan *models.GenerationPlan) ([]models.Patch, error)

	// GenerateTestFile generates a test file for a specific source file
	GenerateTestFile(ctx context.Context, sourceFile string, plan *models.GenerationPlan) (models.Patch, error)
}

// llmTester implements Tester using an LLM to generate tests
type llmTester struct {
	client llm.Client
}

// TesterConfig contains configuration for creating a tester
type TesterConfig struct {
	LLMClient llm.Client
}

// NewTester creates a new Tester instance
func NewTester(cfg TesterConfig) (Tester, error) {
	if cfg.LLMClient == nil {
		return nil, fmt.Errorf("LLM client is required")
	}

	return &llmTester{
		client: cfg.LLMClient,
	}, nil
}

// Generate creates test files for the specified packages
func (t *llmTester) Generate(ctx context.Context, packages []string, plan *models.GenerationPlan) ([]models.Patch, error) {
	log.Info().
		Int("packages", len(packages)).
		Msg("Starting test generation")

	startTime := time.Now()

	// Get all source files from the plan
	sourceFiles := t.getSourceFiles(plan)
	allPatches := make([]models.Patch, 0, len(sourceFiles))

	// Generate tests for each source file
	for _, sourceFile := range sourceFiles {
		// Skip files that are already tests
		if strings.HasSuffix(sourceFile, "_test.go") {
			continue
		}

		// Skip non-Go files
		if !strings.HasSuffix(sourceFile, ".go") {
			continue
		}

		log.Debug().
			Str("source_file", sourceFile).
			Msg("Generating test file")

		patch, err := t.GenerateTestFile(ctx, sourceFile, plan)
		if err != nil {
			// Log error but continue with other files
			log.Warn().
				Err(err).
				Str("source_file", sourceFile).
				Msg("Failed to generate test file")
			continue
		}

		allPatches = append(allPatches, patch)
	}

	duration := time.Since(startTime)
	log.Info().
		Int("test_files_generated", len(allPatches)).
		Dur("duration", duration).
		Msg("Test generation completed")

	return allPatches, nil
}

// GenerateTestFile generates a test file for a specific source file
func (t *llmTester) GenerateTestFile(ctx context.Context, sourceFile string, plan *models.GenerationPlan) (models.Patch, error) {
	// Determine test file path
	testFile := t.getTestFilePath(sourceFile)

	log.Debug().
		Str("source_file", sourceFile).
		Str("test_file", testFile).
		Msg("Generating test file")

	// Build the prompt for test generation
	prompt := t.buildTestGenerationPrompt(sourceFile, plan)

	// Call LLM to generate test code
	response, err := t.client.Generate(ctx, prompt)
	if err != nil {
		return models.Patch{}, fmt.Errorf("LLM test generation failed: %w", err)
	}

	// Clean the response
	testCode := t.cleanTestResponse(response)

	// Create patch for new test file
	patch := models.Patch{
		TargetFile: testFile,
		Diff:       t.createFileDiff(testCode),
		AppliedAt:  time.Now(),
		Reversible: true,
	}

	log.Debug().
		Str("source_file", sourceFile).
		Str("test_file", testFile).
		Int("lines", strings.Count(testCode, "\n")+1).
		Msg("Test file generated successfully")

	return patch, nil
}

// getSourceFiles extracts source file paths from the plan
func (t *llmTester) getSourceFiles(plan *models.GenerationPlan) []string {
	files := make([]string, 0, len(plan.FileTree.Files))
	for _, file := range plan.FileTree.Files {
		files = append(files, file.Path)
	}
	return files
}

// getTestFilePath converts a source file path to its corresponding test file path
func (t *llmTester) getTestFilePath(sourceFile string) string {
	dir := filepath.Dir(sourceFile)
	base := filepath.Base(sourceFile)

	// Remove .go extension and add _test.go
	if strings.HasSuffix(base, ".go") {
		base = strings.TrimSuffix(base, ".go")
		base += "_test.go"
	}

	return filepath.Join(dir, base)
}

// buildTestGenerationPrompt constructs the LLM prompt for test generation
func (t *llmTester) buildTestGenerationPrompt(sourceFile string, plan *models.GenerationPlan) string {
	var sb strings.Builder

	sb.WriteString("You are an expert Go developer writing comprehensive tests.\n\n")
	sb.WriteString("# Task\n")
	sb.WriteString(fmt.Sprintf("Generate comprehensive tests for the source file: %s\n\n", sourceFile))

	// Get file purpose from plan
	filePurpose := t.getFilePurpose(sourceFile, plan)
	if filePurpose != "" {
		sb.WriteString(fmt.Sprintf("# Source File Purpose\n%s\n\n", filePurpose))
	}

	sb.WriteString("# Test Requirements\n\n")
	sb.WriteString("Generate a complete test file that includes:\n\n")

	sb.WriteString("1. **Table-Driven Tests**:\n")
	sb.WriteString("   - Use table-driven test pattern for functions with multiple cases\n")
	sb.WriteString("   - Test both success and failure scenarios\n")
	sb.WriteString("   - Include edge cases and boundary conditions\n\n")

	sb.WriteString("2. **Test Organization**:\n")
	sb.WriteString("   - One test function per public function/method\n")
	sb.WriteString("   - Use descriptive test names (TestFunctionName_Scenario)\n")
	sb.WriteString("   - Group related test cases in subtests using t.Run()\n\n")

	sb.WriteString("3. **Mocking**:\n")
	sb.WriteString("   - Create mock implementations for interfaces\n")
	sb.WriteString("   - Use dependency injection for testability\n")
	sb.WriteString("   - Mock external dependencies (databases, APIs, etc.)\n\n")

	sb.WriteString("4. **Assertions**:\n")
	sb.WriteString("   - Use testify/assert for clean assertions\n")
	sb.WriteString("   - Check all return values (including errors)\n")
	sb.WriteString("   - Verify state changes when applicable\n\n")

	sb.WriteString("5. **Test Coverage**:\n")
	sb.WriteString("   - Test all exported functions and methods\n")
	sb.WriteString("   - Cover success paths\n")
	sb.WriteString("   - Cover error paths\n")
	sb.WriteString("   - Cover edge cases (nil values, empty inputs, etc.)\n\n")

	sb.WriteString("6. **Setup and Teardown**:\n")
	sb.WriteString("   - Use setup functions for test initialization\n")
	sb.WriteString("   - Clean up resources in defer statements or teardown functions\n")
	sb.WriteString("   - Use test helpers for common setup patterns\n\n")

	// Specific test patterns based on file type
	fileName := filepath.Base(sourceFile)
	switch {
	case strings.Contains(fileName, "repository"):
		sb.WriteString("# Repository-Specific Tests\n\n")
		sb.WriteString("- Test all CRUD operations\n")
		sb.WriteString("- Test error handling (not found, duplicate, etc.)\n")
		sb.WriteString("- Test context cancellation\n")
		sb.WriteString("- Mock database interactions\n\n")

	case strings.Contains(fileName, "service"):
		sb.WriteString("# Service-Specific Tests\n\n")
		sb.WriteString("- Test business logic validation\n")
		sb.WriteString("- Test integration with repository layer\n")
		sb.WriteString("- Test error propagation and wrapping\n")
		sb.WriteString("- Mock all dependencies\n\n")

	case strings.Contains(fileName, "handler"):
		sb.WriteString("# Handler-Specific Tests\n\n")
		sb.WriteString("- Test HTTP request handling\n")
		sb.WriteString("- Test request validation\n")
		sb.WriteString("- Test response status codes\n")
		sb.WriteString("- Test JSON encoding/decoding\n")
		sb.WriteString("- Use httptest.ResponseRecorder\n\n")

	case strings.Contains(fileName, "model") || strings.Contains(fileName, "entity"):
		sb.WriteString("# Model-Specific Tests\n\n")
		sb.WriteString("- Test validation methods\n")
		sb.WriteString("- Test constructors\n")
		sb.WriteString("- Test value object behavior\n")
		sb.WriteString("- Test JSON marshaling/unmarshaling\n\n")
	}

	sb.WriteString("# Code Quality\n\n")
	sb.WriteString("1. Follow Go testing best practices\n")
	sb.WriteString("2. Use meaningful test names that describe what is being tested\n")
	sb.WriteString("3. Keep tests simple and focused\n")
	sb.WriteString("4. Avoid testing implementation details\n")
	sb.WriteString("5. Make tests readable and maintainable\n\n")

	sb.WriteString("# Example Test Structure\n\n")
	sb.WriteString("```go\n")
	sb.WriteString("func TestFunctionName(t *testing.T) {\n")
	sb.WriteString("    tests := []struct {\n")
	sb.WriteString("        name    string\n")
	sb.WriteString("        input   InputType\n")
	sb.WriteString("        want    OutputType\n")
	sb.WriteString("        wantErr bool\n")
	sb.WriteString("    }{\n")
	sb.WriteString("        {\n")
	sb.WriteString("            name:    \"successful case\",\n")
	sb.WriteString("            input:   validInput,\n")
	sb.WriteString("            want:    expectedOutput,\n")
	sb.WriteString("            wantErr: false,\n")
	sb.WriteString("        },\n")
	sb.WriteString("        {\n")
	sb.WriteString("            name:    \"error case\",\n")
	sb.WriteString("            input:   invalidInput,\n")
	sb.WriteString("            want:    OutputType{},\n")
	sb.WriteString("            wantErr: true,\n")
	sb.WriteString("        },\n")
	sb.WriteString("    }\n\n")
	sb.WriteString("    for _, tt := range tests {\n")
	sb.WriteString("        t.Run(tt.name, func(t *testing.T) {\n")
	sb.WriteString("            got, err := FunctionName(tt.input)\n")
	sb.WriteString("            if (err != nil) != tt.wantErr {\n")
	sb.WriteString("                t.Errorf(\"FunctionName() error = %v, wantErr %v\", err, tt.wantErr)\n")
	sb.WriteString("                return\n")
	sb.WriteString("            }\n")
	sb.WriteString("            if !reflect.DeepEqual(got, tt.want) {\n")
	sb.WriteString("                t.Errorf(\"FunctionName() = %v, want %v\", got, tt.want)\n")
	sb.WriteString("            }\n")
	sb.WriteString("        })\n")
	sb.WriteString("    }\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")

	sb.WriteString("# Output Format\n\n")
	sb.WriteString("Return ONLY the Go test code, no additional explanation or markdown.\n")
	sb.WriteString("The code should be complete, correctly formatted, and ready to run.\n")
	sb.WriteString("Include all necessary imports.\n")

	return sb.String()
}

// getFilePurpose retrieves the purpose of a file from the plan
func (t *llmTester) getFilePurpose(targetPath string, plan *models.GenerationPlan) string {
	for _, file := range plan.FileTree.Files {
		if file.Path == targetPath || filepath.Base(file.Path) == filepath.Base(targetPath) {
			return file.Purpose
		}
	}
	return ""
}

// cleanTestResponse removes markdown formatting and extracts the code
func (t *llmTester) cleanTestResponse(response string) string {
	response = strings.TrimSpace(response)

	// Remove markdown code blocks
	if strings.HasPrefix(response, "```go") {
		response = strings.TrimPrefix(response, "```go")
		response = strings.TrimSuffix(response, "```")
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
	}

	return strings.TrimSpace(response)
}

// createFileDiff creates a unified diff for creating a new file
func (t *llmTester) createFileDiff(content string) string {
	var sb strings.Builder

	lines := strings.Split(content, "\n")
	sb.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(lines)))

	for _, line := range lines {
		sb.WriteString("+" + line + "\n")
	}

	return sb.String()
}
