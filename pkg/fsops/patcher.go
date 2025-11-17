package fsops

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// PatchResult contains the result of applying a patch
type PatchResult struct {
	Success      bool
	OriginalHash string
	NewHash      string
	LinesChanged int
}

// ApplyPatch applies a unified diff patch to a file
func (f *fileOps) ApplyPatch(ctx context.Context, patch models.Patch) error {
	if err := f.ValidatePath(patch.TargetFile); err != nil {
		return fmt.Errorf("invalid target file path: %w", err)
	}

	// Check if file exists
	exists, err := f.Exists(ctx, patch.TargetFile)
	if err != nil {
		return fmt.Errorf("failed to check if target file exists: %w", err)
	}

	// Read the current file content
	var currentContent string
	if exists {
		currentContent, err = f.ReadFile(ctx, patch.TargetFile)
		if err != nil {
			return fmt.Errorf("failed to read target file: %w", err)
		}
	} else {
		currentContent = ""
	}

	// Calculate original checksum
	originalHash := f.GenerateChecksum(currentContent)

	// Apply the patch
	dmp := diffmatchpatch.New()
	patches, err := dmp.PatchFromText(patch.Diff)
	if err != nil {
		return fmt.Errorf("failed to parse patch: %w", err)
	}

	if len(patches) == 0 {
		return fmt.Errorf("no patches found in diff")
	}

	// Apply patches
	newContent, results := dmp.PatchApply(patches, currentContent)

	// Check if all patches applied successfully
	for i, result := range results {
		if !result {
			return fmt.Errorf("failed to apply patch %d of %d", i+1, len(patches))
		}
	}

	// Calculate new checksum
	newHash := f.GenerateChecksum(newContent)

	// Log the operation before applying
	if err := f.logger.LogFileOperation(ctx, models.FileOperationLog{
		LogEntry: models.LogEntry{
			Component: "fsops",
			Operation: "apply_patch",
			Message:   fmt.Sprintf("Applying patch to: %s", patch.TargetFile),
			Context: map[string]interface{}{
				"original_hash": originalHash,
				"new_hash":      newHash,
				"reversible":    patch.Reversible,
			},
		},
		OperationType: "patch",
		Path:          patch.TargetFile,
		Checksum:      newHash,
	}); err != nil {
		return fmt.Errorf("failed to log patch operation: %w", err)
	}

	// Write the patched content
	if err := f.WriteFile(ctx, patch.TargetFile, newContent); err != nil {
		return fmt.Errorf("failed to write patched content: %w", err)
	}

	return nil
}

// GeneratePatch creates a patch from old content to new content
func (f *fileOps) GeneratePatch(ctx context.Context, targetFile, oldContent, newContent string) (models.Patch, error) {
	if err := f.ValidatePath(targetFile); err != nil {
		return models.Patch{}, fmt.Errorf("invalid target file path: %w", err)
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldContent, newContent, false)

	// Optimize the diffs
	diffs = dmp.DiffCleanupSemantic(diffs)

	// Create patches
	patches := dmp.PatchMake(oldContent, diffs)

	// Convert to text format
	patchText := dmp.PatchToText(patches)

	patch := models.Patch{
		TargetFile: targetFile,
		Diff:       patchText,
		AppliedAt:  time.Now(),
		Reversible: true,
	}

	return patch, nil
}

// GeneratePatchFromFiles creates a patch between two file versions
func (f *fileOps) GeneratePatchFromFiles(ctx context.Context, targetFile string) (models.Patch, error) {
	if err := f.ValidatePath(targetFile); err != nil {
		return models.Patch{}, fmt.Errorf("invalid target file path: %w", err)
	}

	// Read the current file
	_, err := f.ReadFile(ctx, targetFile)
	if err != nil {
		return models.Patch{}, fmt.Errorf("failed to read current file: %w", err)
	}

	// For this implementation, we need the previous version to be stored
	// This would typically come from a backup or version control
	// For now, we return an error indicating this operation needs context
	return models.Patch{}, fmt.Errorf("GeneratePatchFromFiles requires previous version context")
}

// ReversePatch creates a reverse patch that undoes the given patch
// It requires a backup file to exist (created by ApplyPatchWithBackup)
func (f *fileOps) ReversePatch(ctx context.Context, patch models.Patch) (models.Patch, error) {
	if !patch.Reversible {
		return models.Patch{}, fmt.Errorf("patch is not reversible")
	}

	// Check if backup exists
	backupPath := patch.TargetFile + ".backup"
	exists, err := f.Exists(ctx, backupPath)
	if err != nil {
		return models.Patch{}, fmt.Errorf("failed to check for backup: %w", err)
	}

	if !exists {
		// If no backup, we can't reverse - this is a limitation
		// In a real implementation, you'd store the original content with the patch
		return models.Patch{}, fmt.Errorf("no backup file found for reversal - use ApplyPatchWithBackup")
	}

	// Read backup content (original before patch)
	originalContent, err := f.ReadFile(ctx, backupPath)
	if err != nil {
		return models.Patch{}, fmt.Errorf("failed to read backup: %w", err)
	}

	// Read current content (after patch)
	currentContent, err := f.ReadFile(ctx, patch.TargetFile)
	if err != nil {
		return models.Patch{}, fmt.Errorf("failed to read current file: %w", err)
	}

	// Generate reverse patch (from current back to original)
	return f.GeneratePatch(ctx, patch.TargetFile, currentContent, originalContent)
}

// ApplyPatchWithBackup applies a patch and creates a backup for reversal
func (f *fileOps) ApplyPatchWithBackup(ctx context.Context, patch models.Patch) error {
	// First check if file exists and back it up
	exists, err := f.Exists(ctx, patch.TargetFile)
	if err != nil {
		return err
	}

	if exists {
		// Read current content for backup
		content, err := f.ReadFile(ctx, patch.TargetFile)
		if err != nil {
			return fmt.Errorf("failed to read file for backup: %w", err)
		}

		// Write backup
		backupPath := patch.TargetFile + ".backup"
		if err := f.WriteFile(ctx, backupPath, content); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Apply the patch
	return f.ApplyPatch(ctx, patch)
}

// ValidatePatch validates a patch without applying it
func (f *fileOps) ValidatePatch(ctx context.Context, patch models.Patch) error {
	if err := f.ValidatePath(patch.TargetFile); err != nil {
		return fmt.Errorf("invalid target file path: %w", err)
	}

	if patch.Diff == "" {
		return fmt.Errorf("patch diff is empty")
	}

	// Try to parse the patch
	dmp := diffmatchpatch.New()
	patches, err := dmp.PatchFromText(patch.Diff)
	if err != nil {
		return fmt.Errorf("failed to parse patch: %w", err)
	}

	if len(patches) == 0 {
		return fmt.Errorf("no valid patches found")
	}

	// Optionally, try to apply to current file to see if it would succeed
	exists, err := f.Exists(ctx, patch.TargetFile)
	if err != nil {
		return fmt.Errorf("failed to check if target exists: %w", err)
	}

	if exists {
		content, err := f.ReadFile(ctx, patch.TargetFile)
		if err != nil {
			return fmt.Errorf("failed to read target file: %w", err)
		}

		_, results := dmp.PatchApply(patches, content)
		if !allTrue(results) {
			return fmt.Errorf("patch would fail to apply to current file state")
		}
	}

	return nil
}

// CreateFilePatch creates a patch for creating a new file
func (f *fileOps) CreateFilePatch(ctx context.Context, targetFile, content string) (models.Patch, error) {
	return f.GeneratePatch(ctx, targetFile, "", content)
}

// DeleteFilePatch creates a patch for deleting a file
func (f *fileOps) DeleteFilePatch(ctx context.Context, targetFile string) (models.Patch, error) {
	currentContent, err := f.ReadFile(ctx, targetFile)
	if err != nil {
		return models.Patch{}, fmt.Errorf("failed to read file for deletion patch: %w", err)
	}

	return f.GeneratePatch(ctx, targetFile, currentContent, "")
}

// GetPatchStats returns statistics about a patch
// Note: This provides approximate statistics by analyzing the diff text
func (f *fileOps) GetPatchStats(patch models.Patch) (added, removed, modified int, err error) {
	if patch.Diff == "" {
		return 0, 0, 0, nil
	}

	// Parse the patch text line by line to count changes
	lines := strings.Split(patch.Diff, "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		// Unified diff format: lines starting with + are additions, - are deletions
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			added++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			removed++
		}
	}

	// Modified lines are the minimum of added and removed
	modified = min(added, removed)
	added -= modified
	removed -= modified

	return added, removed, modified, nil
}

// allTrue checks if all boolean values in a slice are true
func allTrue(values []bool) bool {
	for _, v := range values {
		if !v {
			return false
		}
	}
	return true
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
