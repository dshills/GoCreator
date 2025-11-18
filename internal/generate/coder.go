// Package generate provides code generation functionality for GoCreator.
package generate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/pkg/llm"
	"github.com/rs/zerolog/log"
)

// Coder generates source code from generation plans
type Coder interface {
	// Generate creates source code files based on the generation plan
	Generate(ctx context.Context, plan *models.GenerationPlan, fcs *models.FinalClarifiedSpecification) ([]models.Patch, error)

	// GenerateFile generates a single file based on task inputs
	GenerateFile(ctx context.Context, task models.GenerationTask, plan *models.GenerationPlan, fcs *models.FinalClarifiedSpecification) (models.Patch, error)
}

// llmCoder implements Coder using an LLM to generate code
type llmCoder struct {
	client        llm.Client
	contextFilter *ContextFilter
	metrics       *models.GenerationMetrics
}

// CoderConfig contains configuration for creating a coder
type CoderConfig struct {
	LLMClient llm.Client
}

// NewCoder creates a new Coder instance
func NewCoder(cfg CoderConfig) (Coder, error) {
	if cfg.LLMClient == nil {
		return nil, fmt.Errorf("LLM client is required")
	}

	return &llmCoder{
		client: cfg.LLMClient,
		metrics: &models.GenerationMetrics{
			PhaseTimings:  make(map[string]time.Duration),
			CostBreakdown: make(map[string]float64),
		},
	}, nil
}

// SetFCS sets the FCS and initializes the context filter
func (c *llmCoder) SetFCS(fcs *models.FinalClarifiedSpecification) {
	c.contextFilter = NewContextFilter(fcs)
}

// GetMetrics returns the generation metrics
func (c *llmCoder) GetMetrics() *models.GenerationMetrics {
	return c.metrics
}

// Generate creates source code files based on the generation plan
func (c *llmCoder) Generate(ctx context.Context, plan *models.GenerationPlan, fcs *models.FinalClarifiedSpecification) ([]models.Patch, error) {
	if plan == nil {
		return nil, fmt.Errorf("generation plan is required")
	}

	// Initialize context filter if FCS is provided
	if fcs != nil {
		c.SetFCS(fcs)
	}

	log.Info().
		Str("plan_id", plan.ID).
		Int("phases", len(plan.Phases)).
		Msg("Starting code generation with smart context filtering")

	startTime := time.Now()
	var allPatches []models.Patch

	// Process each phase in order
	for _, phase := range plan.Phases {
		log.Debug().
			Str("phase", phase.Name).
			Int("tasks", len(phase.Tasks)).
			Msg("Processing generation phase")

		// Process tasks in the phase
		for _, task := range phase.Tasks {
			// Only generate files, skip other task types
			if task.Type != "generate_file" {
				log.Debug().
					Str("task_id", task.ID).
					Str("task_type", task.Type).
					Msg("Skipping non-generate_file task")
				continue
			}

			patch, err := c.GenerateFile(ctx, task, plan, fcs)
			if err != nil {
				return nil, fmt.Errorf("failed to generate file for task %s: %w", task.ID, err)
			}

			allPatches = append(allPatches, patch)
		}
	}

	duration := time.Since(startTime)
	log.Info().
		Str("plan_id", plan.ID).
		Int("files_generated", len(allPatches)).
		Dur("duration", duration).
		Float64("avg_reduction_pct", c.metrics.AvgReductionPercentage).
		Msg("Code generation completed")

	return allPatches, nil
}

// GenerateFile generates a single file based on task inputs
func (c *llmCoder) GenerateFile(ctx context.Context, task models.GenerationTask, plan *models.GenerationPlan, fcs *models.FinalClarifiedSpecification) (models.Patch, error) {
	log.Debug().
		Str("task_id", task.ID).
		Str("target_path", task.TargetPath).
		Msg("Generating file with filtered context")

	startTime := time.Now()

	// Filter FCS for this specific file
	var filteredFCS *FilteredFCS
	if c.contextFilter != nil {
		filteredFCS = c.contextFilter.FilterForFile(task.TargetPath, plan, fcs)

		// Track metrics
		metric := models.ContextFilterMetrics{
			FilePath:             task.TargetPath,
			OriginalEntityCount:  filteredFCS.OriginalEntityCount,
			FilteredEntityCount:  filteredFCS.FilteredEntityCount,
			OriginalPackageCount: filteredFCS.OriginalPackageCount,
			FilteredPackageCount: filteredFCS.FilteredPackageCount,
			ReductionPercentage:  filteredFCS.ReductionPercentage,
			FilterDuration:       time.Since(startTime),
		}
		c.metrics.AddContextFilterMetrics(metric)
	}

	// Build the prompt for code generation with filtered context
	prompt := c.buildCodeGenerationPrompt(task, plan, filteredFCS)

	// Call LLM to generate code
	response, err := c.client.Generate(ctx, prompt)
	if err != nil {
		return models.Patch{}, fmt.Errorf("LLM code generation failed: %w", err)
	}

	// Clean the response (remove markdown code blocks if present)
	code := c.cleanCodeResponse(response)

	// Calculate checksum
	hash := sha256.Sum256([]byte(code))
	checksum := hex.EncodeToString(hash[:])

	// Create patch for new file creation
	patch := models.Patch{
		TargetFile: task.TargetPath,
		Diff:       c.createFileDiff(code),
		AppliedAt:  time.Now(),
		Reversible: true,
	}

	logEvent := log.Debug().
		Str("task_id", task.ID).
		Str("target_path", task.TargetPath).
		Str("checksum", checksum).
		Int("lines", strings.Count(code, "\n")+1)

	if filteredFCS != nil {
		logEvent.Float64("context_reduction_pct", filteredFCS.ReductionPercentage)
	}

	logEvent.Msg("File generated successfully")

	return patch, nil
}

// buildCodeGenerationPrompt constructs the LLM prompt for code generation
func (c *llmCoder) buildCodeGenerationPrompt(task models.GenerationTask, plan *models.GenerationPlan, filteredFCS *FilteredFCS) string {
	var sb strings.Builder

	sb.WriteString("You are an expert Go developer generating production-ready code.\n\n")
	sb.WriteString("# Task\n")
	sb.WriteString(fmt.Sprintf("Generate a Go source file for: %s\n\n", task.TargetPath))

	// Include filtered FCS context if available
	if filteredFCS != nil {
		sb.WriteString("# Project Context (Filtered)\n\n")
		sb.WriteString(c.contextFilter.FormatFilteredFCS(filteredFCS))
		sb.WriteString("\n")
	}

	// Determine file type and provide specific instructions
	fileName := filepath.Base(task.TargetPath)
	fileType := c.determineFileType(fileName)

	sb.WriteString(fmt.Sprintf("# File Type: %s\n\n", fileType))

	// Get file purpose from plan
	filePurpose := c.getFilePurpose(task.TargetPath, plan)
	if filePurpose != "" {
		sb.WriteString(fmt.Sprintf("# Purpose\n%s\n\n", filePurpose))
	}

	// Add context from task inputs
	if task.Inputs != nil {
		sb.WriteString("# Context\n")
		if pkg, ok := task.Inputs["package"].(string); ok {
			sb.WriteString(fmt.Sprintf("Package: %s\n", pkg))
		}
		if entities, ok := task.Inputs["entities"].([]interface{}); ok {
			sb.WriteString("Entities: ")
			for i, e := range entities {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("%v", e))
			}
			sb.WriteString("\n")
		}
		if deps, ok := task.Inputs["dependencies"].([]interface{}); ok {
			sb.WriteString("Dependencies: ")
			for i, d := range deps {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("%v", d))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Type-specific instructions
	sb.WriteString("# Requirements\n\n")

	switch fileType {
	case "go.mod":
		sb.WriteString("Generate a go.mod file with:\n")
		sb.WriteString("- Correct module path\n")
		sb.WriteString("- Go version from build config\n")
		sb.WriteString("- Required dependencies with versions\n")
		sb.WriteString("- Proper formatting\n\n")

	case "main.go":
		sb.WriteString("Generate a main.go file with:\n")
		sb.WriteString("- package main declaration\n")
		sb.WriteString("- Proper imports\n")
		sb.WriteString("- main() function with initialization\n")
		sb.WriteString("- Error handling and logging\n")
		sb.WriteString("- Graceful shutdown handling\n\n")

	case "model":
		sb.WriteString("Generate a model/entity file with:\n")
		sb.WriteString("- Proper package declaration\n")
		sb.WriteString("- Struct definitions with JSON tags\n")
		sb.WriteString("- Validation methods\n")
		sb.WriteString("- Constructor functions\n")
		sb.WriteString("- Godoc comments for all exported types and functions\n\n")

	case "repository":
		sb.WriteString("Generate a repository file with:\n")
		sb.WriteString("- Interface definition for repository contract\n")
		sb.WriteString("- Concrete implementation struct\n")
		sb.WriteString("- Constructor function\n")
		sb.WriteString("- All CRUD methods with proper error handling\n")
		sb.WriteString("- Context support for cancellation\n\n")

	case "service":
		sb.WriteString("Generate a service file with:\n")
		sb.WriteString("- Service interface definition\n")
		sb.WriteString("- Service struct with dependencies\n")
		sb.WriteString("- Constructor with dependency injection\n")
		sb.WriteString("- Business logic methods\n")
		sb.WriteString("- Proper error handling and logging\n\n")

	case "handler":
		sb.WriteString("Generate an HTTP handler file with:\n")
		sb.WriteString("- Handler struct with service dependencies\n")
		sb.WriteString("- HTTP handler functions\n")
		sb.WriteString("- Request validation\n")
		sb.WriteString("- Proper HTTP status codes\n")
		sb.WriteString("- JSON encoding/decoding\n\n")

	case "test":
		sb.WriteString("Generate a test file with:\n")
		sb.WriteString("- Table-driven tests using testing package\n")
		sb.WriteString("- Test setup and teardown\n")
		sb.WriteString("- Mocks for dependencies\n")
		sb.WriteString("- Comprehensive test cases including edge cases\n")
		sb.WriteString("- Proper assertions\n\n")

	default:
		sb.WriteString("Generate a well-structured Go file with:\n")
		sb.WriteString("- Proper package declaration\n")
		sb.WriteString("- Clear, idiomatic Go code\n")
		sb.WriteString("- Proper error handling\n")
		sb.WriteString("- Comprehensive documentation\n\n")
	}

	// General coding standards
	sb.WriteString("# Coding Standards\n\n")
	sb.WriteString("1. **Go Best Practices**:\n")
	sb.WriteString("   - Follow Go idioms and conventions\n")
	sb.WriteString("   - Accept interfaces, return structs\n")
	sb.WriteString("   - Use meaningful variable names\n")
	sb.WriteString("   - Keep functions small and focused\n\n")

	sb.WriteString("2. **Error Handling**:\n")
	sb.WriteString("   - Return errors, don't panic\n")
	sb.WriteString("   - Wrap errors with context using fmt.Errorf\n")
	sb.WriteString("   - Use sentinel errors for known conditions\n\n")

	sb.WriteString("3. **Documentation**:\n")
	sb.WriteString("   - Add godoc comments for all exported symbols\n")
	sb.WriteString("   - Comments should explain why, not what\n")
	sb.WriteString("   - Keep line length under 100 characters\n\n")

	sb.WriteString("4. **Testing**:\n")
	sb.WriteString("   - Write testable code\n")
	sb.WriteString("   - Use dependency injection\n")
	sb.WriteString("   - Avoid global state\n\n")

	sb.WriteString("# Output Format\n\n")
	sb.WriteString("Return ONLY the Go source code, no additional explanation or markdown.\n")
	sb.WriteString("The code should be complete, correctly formatted, and ready to use.\n")

	return sb.String()
}

// determineFileType determines the type of file being generated
func (c *llmCoder) determineFileType(fileName string) string {
	switch {
	case fileName == "go.mod":
		return "go.mod"
	case fileName == "main.go":
		return "main.go"
	case strings.HasSuffix(fileName, "_test.go"):
		return "test"
	case strings.Contains(fileName, "model") || strings.Contains(fileName, "entity"):
		return "model"
	case strings.Contains(fileName, "repository") || strings.Contains(fileName, "repo"):
		return "repository"
	case strings.Contains(fileName, "service"):
		return "service"
	case strings.Contains(fileName, "handler"):
		return "handler"
	case fileName == "Makefile":
		return "Makefile"
	case fileName == "Dockerfile":
		return "Dockerfile"
	case fileName == "README.md":
		return "documentation"
	default:
		return "source"
	}
}

// getFilePurpose retrieves the purpose of a file from the plan
func (c *llmCoder) getFilePurpose(targetPath string, plan *models.GenerationPlan) string {
	for _, file := range plan.FileTree.Files {
		if file.Path == targetPath || filepath.Base(file.Path) == filepath.Base(targetPath) {
			return file.Purpose
		}
	}
	return ""
}

// cleanCodeResponse removes markdown formatting and extracts the code
func (c *llmCoder) cleanCodeResponse(response string) string {
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
func (c *llmCoder) createFileDiff(content string) string {
	// For new files, we create a simple diff format
	// In a real implementation, you'd use a proper diff library
	var sb strings.Builder

	lines := strings.Split(content, "\n")
	sb.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(lines)))

	for _, line := range lines {
		sb.WriteString("+" + line + "\n")
	}

	return sb.String()
}
