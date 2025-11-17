package generate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/pkg/llm"
	"github.com/google/uuid"
)

// IncrementalConfig configures the incremental generator
type IncrementalConfig struct {
	LLMClient      llm.Client
	ChangeDetector *ChangeDetector
}

// IncrementalGenerator handles incremental regeneration of code
type IncrementalGenerator struct {
	llmClient      llm.Client
	changeDetector *ChangeDetector
}

// NewIncrementalGenerator creates a new incremental generator
func NewIncrementalGenerator(config IncrementalConfig) (*IncrementalGenerator, error) {
	if config.LLMClient == nil {
		return nil, fmt.Errorf("LLM client is required")
	}
	if config.ChangeDetector == nil {
		return nil, fmt.Errorf("change detector is required")
	}

	return &IncrementalGenerator{
		llmClient:      config.LLMClient,
		changeDetector: config.ChangeDetector,
	}, nil
}

// Regenerate regenerates only the affected portions of code based on FCS changes
func (ig *IncrementalGenerator) Regenerate(ctx context.Context, oldFCS, newFCS *models.FinalClarifiedSpecification, oldOutput *models.GenerationOutput) (*models.GenerationOutput, error) {
	// Detect changes
	changes, err := ig.changeDetector.DetectChanges(oldFCS, newFCS)
	if err != nil {
		return nil, fmt.Errorf("failed to detect changes: %w", err)
	}

	// If no changes and we have existing output, return it
	if !changes.HasChanges && len(oldOutput.Files) > 0 {
		return oldOutput, nil
	}

	// Identify affected packages (or all packages if no old output)
	var affectedPackages []string
	if changes.HasChanges {
		affectedPackages, err = ig.changeDetector.IdentifyAffectedPackages(changes, &newFCS.Architecture)
		if err != nil {
			return nil, fmt.Errorf("failed to identify affected packages: %w", err)
		}
	} else {
		// No changes but no old output - generate all packages (initial generation)
		for _, pkg := range newFCS.Architecture.Packages {
			affectedPackages = append(affectedPackages, pkg.Name)
		}
	}

	// Generate code only for affected packages
	newFiles := []models.GeneratedFile{}
	for _, pkgName := range affectedPackages {
		// Find the package in the new architecture
		var pkg *models.Package
		for i := range newFCS.Architecture.Packages {
			if newFCS.Architecture.Packages[i].Name == pkgName {
				pkg = &newFCS.Architecture.Packages[i]
				break
			}
		}

		if pkg == nil {
			// Package was deleted, skip generation
			continue
		}

		// Generate code for this package
		files, err := ig.generatePackageCode(ctx, pkg, newFCS)
		if err != nil {
			return nil, fmt.Errorf("failed to generate code for package %s: %w", pkgName, err)
		}

		newFiles = append(newFiles, files...)
	}

	// Merge with old output
	mergedOutput := ig.MergeOutputs(oldOutput, newFiles, affectedPackages)

	return mergedOutput, nil
}

// generatePackageCode generates code for a specific package
func (ig *IncrementalGenerator) generatePackageCode(ctx context.Context, pkg *models.Package, fcs *models.FinalClarifiedSpecification) ([]models.GeneratedFile, error) {
	// Build prompt for code generation
	prompt := ig.buildGenerationPrompt(pkg, fcs)

	// Call LLM to generate code
	response, err := ig.llmClient.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Parse response (simple JSON format for this implementation)
	var fileData struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}

	if err := json.Unmarshal([]byte(response), &fileData); err != nil {
		// If it's not JSON, treat the whole response as content
		fileData.Path = fmt.Sprintf("%s/%s.go", pkg.Path, pkg.Name)
		fileData.Content = response
	}

	// Create generated file
	file := models.GeneratedFile{
		Path:        fileData.Path,
		Content:     fileData.Content,
		GeneratedAt: time.Now(),
		Generator:   "incremental-generator",
	}

	// Compute checksum
	hash := sha256.Sum256([]byte(file.Content))
	file.Checksum = hex.EncodeToString(hash[:])

	return []models.GeneratedFile{file}, nil
}

// buildGenerationPrompt builds the LLM prompt for generating package code
func (ig *IncrementalGenerator) buildGenerationPrompt(pkg *models.Package, fcs *models.FinalClarifiedSpecification) string {
	return fmt.Sprintf(`Generate Go code for the following package:

Package Name: %s
Package Path: %s
Purpose: %s
Dependencies: %v

Requirements:
%s

Output the code as JSON with 'path' and 'content' fields.`,
		pkg.Name,
		pkg.Path,
		pkg.Purpose,
		pkg.Dependencies,
		ig.formatRequirements(fcs.Requirements),
	)
}

// formatRequirements formats requirements for the prompt
func (ig *IncrementalGenerator) formatRequirements(reqs models.Requirements) string {
	result := ""
	for _, req := range reqs.Functional {
		result += fmt.Sprintf("- %s: %s\n", req.ID, req.Description)
	}
	return result
}

// ShouldRegenerate determines if a package should be regenerated
func (ig *IncrementalGenerator) ShouldRegenerate(packageName string, changes *FCSChanges, architecture *models.Architecture) bool {
	if !changes.HasChanges {
		return false
	}

	// Check if package was added
	for _, pkg := range changes.AddedPackages {
		if pkg.Name == packageName {
			return true
		}
	}

	// Check if package was modified
	for _, pkg := range changes.ModifiedPackages {
		if pkg.Name == packageName {
			return true
		}
	}

	// Check if any dependency was modified
	affectedPackages, err := ig.changeDetector.IdentifyAffectedPackages(changes, architecture)
	if err != nil {
		return false
	}

	for _, affectedPkg := range affectedPackages {
		if affectedPkg == packageName {
			return true
		}
	}

	return false
}

// MergeOutputs merges new files with old output, replacing files for affected packages
func (ig *IncrementalGenerator) MergeOutputs(oldOutput *models.GenerationOutput, newFiles []models.GeneratedFile, affectedPackages []string) *models.GenerationOutput {
	// Handle nil oldOutput (treat as empty output)
	if oldOutput == nil {
		oldOutput = &models.GenerationOutput{
			SchemaVersion: "1.0",
			Files:         []models.GeneratedFile{},
			Patches:       []models.Patch{},
			PlanID:        "",
		}
	}

	// Create a map of new files by path
	newFileMap := make(map[string]models.GeneratedFile)
	for _, file := range newFiles {
		newFileMap[file.Path] = file
	}

	// Create a set of affected packages for quick lookup
	affectedSet := make(map[string]bool)
	for _, pkg := range affectedPackages {
		affectedSet[pkg] = true
	}

	// Start with old files, replacing those affected by changes
	mergedFiles := []models.GeneratedFile{}
	for _, oldFile := range oldOutput.Files {
		// Check if this file should be replaced by a new version
		if newFile, exists := newFileMap[oldFile.Path]; exists {
			mergedFiles = append(mergedFiles, newFile)
			delete(newFileMap, oldFile.Path) // Mark as processed
		} else {
			// Check if file belongs to an affected package
			pkgName := ig.extractPackageNameFromPath(oldFile.Path)
			if !affectedSet[pkgName] {
				// Keep old file if not in affected packages
				mergedFiles = append(mergedFiles, oldFile)
			}
			// Otherwise, file is in affected package but not in new files (deleted/not regenerated)
		}
	}

	// Add any remaining new files that weren't replacements
	for _, newFile := range newFileMap {
		mergedFiles = append(mergedFiles, newFile)
	}

	// Create new output
	return &models.GenerationOutput{
		SchemaVersion: "1.0",
		ID:            uuid.New().String(),
		PlanID:        oldOutput.PlanID,
		Files:         mergedFiles,
		Patches:       oldOutput.Patches,
		Metadata: models.OutputMetadata{
			StartedAt:  time.Now(),
			FilesCount: len(mergedFiles),
		},
		Status: models.OutputStatusCompleted,
	}
}

// extractPackageNameFromPath extracts the package name from a file path
// For example: "internal/auth/auth.go" -> "auth"
func (ig *IncrementalGenerator) extractPackageNameFromPath(filePath string) string {
	// Split path by /
	parts := splitPath(filePath)
	if len(parts) >= 2 {
		// Return the last directory before the file
		return parts[len(parts)-2]
	}
	return ""
}

// splitPath splits a path by /
func splitPath(path string) []string {
	var result []string
	current := ""
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(path[i])
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
