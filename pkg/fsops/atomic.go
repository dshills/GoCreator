package fsops

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dshills/gocreator/internal/models"
)

// AtomicWrite writes content to a file atomically using a temp file and rename
// This ensures that the file is either fully written or not written at all
func (f *fileOps) AtomicWrite(ctx context.Context, path, content string) error {
	if err := f.ValidatePath(path); err != nil {
		return err
	}

	absPath, err := f.getAbsolutePath(path)
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create a temporary file in the same directory
	// This ensures the temp file is on the same filesystem as the target
	tempFile, err := os.CreateTemp(dir, ".gocreator-temp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Ensure cleanup on error
	defer func() {
		// Only remove temp file if it still exists (not renamed)
		if _, err := os.Stat(tempPath); err == nil {
			os.Remove(tempPath)
		}
	}()

	// Write content to temp file
	if _, err := tempFile.WriteString(content); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	// Sync to ensure data is written to disk
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close the temp file
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set proper permissions on temp file before rename
	if err := os.Chmod(tempPath, 0644); err != nil {
		return fmt.Errorf("failed to set permissions on temp file: %w", err)
	}

	// Calculate checksum
	checksum := f.GenerateChecksum(content)

	// Log the operation before the atomic rename
	if err := f.logger.LogFileOperation(ctx, models.FileOperationLog{
		LogEntry: models.LogEntry{
			Component: "fsops",
			Operation: "atomic_write",
			Message:   fmt.Sprintf("Atomic write to: %s", path),
		},
		OperationType: "create",
		Path:          path,
		Checksum:      checksum,
	}); err != nil {
		return fmt.Errorf("failed to log operation: %w", err)
	}

	// Atomic rename - this is the atomic operation
	// On POSIX systems, rename is atomic if source and dest are on same filesystem
	if err := os.Rename(tempPath, absPath); err != nil {
		return fmt.Errorf("failed to rename temp file to target: %w", err)
	}

	return nil
}

// AtomicWriteWithBackup writes content atomically and creates a backup of existing file
func (f *fileOps) AtomicWriteWithBackup(ctx context.Context, path, content string) (backupPath string, err error) {
	if err := f.ValidatePath(path); err != nil {
		return "", err
	}

	absPath, err := f.getAbsolutePath(path)
	if err != nil {
		return "", err
	}

	// Check if file exists
	exists, err := f.Exists(ctx, path)
	if err != nil {
		return "", err
	}

	// Create backup if file exists
	if exists {
		// Read existing content
		existingContent, err := f.ReadFile(ctx, path)
		if err != nil {
			return "", fmt.Errorf("failed to read existing file for backup: %w", err)
		}

		// Create backup filename
		backupPath = path + ".backup"
		backupAbsPath := absPath + ".backup"

		// Write backup atomically
		backupTempFile, err := os.CreateTemp(filepath.Dir(absPath), ".gocreator-backup-*")
		if err != nil {
			return "", fmt.Errorf("failed to create backup temp file: %w", err)
		}
		backupTempPath := backupTempFile.Name()

		defer func() {
			if _, err := os.Stat(backupTempPath); err == nil {
				os.Remove(backupTempPath)
			}
		}()

		if _, err := backupTempFile.WriteString(existingContent); err != nil {
			backupTempFile.Close()
			return "", fmt.Errorf("failed to write backup: %w", err)
		}

		if err := backupTempFile.Sync(); err != nil {
			backupTempFile.Close()
			return "", fmt.Errorf("failed to sync backup: %w", err)
		}

		if err := backupTempFile.Close(); err != nil {
			return "", fmt.Errorf("failed to close backup: %w", err)
		}

		if err := os.Chmod(backupTempPath, 0644); err != nil {
			return "", fmt.Errorf("failed to set permissions on backup: %w", err)
		}

		// Rename backup into place
		if err := os.Rename(backupTempPath, backupAbsPath); err != nil {
			return "", fmt.Errorf("failed to rename backup: %w", err)
		}

		// Log backup creation
		checksum := f.GenerateChecksum(existingContent)
		if err := f.logger.LogFileOperation(ctx, models.FileOperationLog{
			LogEntry: models.LogEntry{
				Component: "fsops",
				Operation: "create_backup",
				Message:   fmt.Sprintf("Created backup: %s", backupPath),
			},
			OperationType: "create",
			Path:          backupPath,
			Checksum:      checksum,
		}); err != nil {
			return backupPath, fmt.Errorf("failed to log backup operation: %w", err)
		}
	}

	// Now write the new content atomically
	if err := f.AtomicWrite(ctx, path, content); err != nil {
		return backupPath, err
	}

	return backupPath, nil
}

// RestoreFromBackup restores a file from its backup
func (f *fileOps) RestoreFromBackup(ctx context.Context, path string) error {
	backupPath := path + ".backup"

	// Validate both paths
	if err := f.ValidatePath(path); err != nil {
		return err
	}
	if err := f.ValidatePath(backupPath); err != nil {
		return err
	}

	// Check if backup exists
	exists, err := f.Exists(ctx, backupPath)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	// Read backup content
	backupContent, err := f.ReadFile(ctx, backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	// Restore atomically
	if err := f.AtomicWrite(ctx, path, backupContent); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	// Log the restore operation
	if err := f.logger.LogFileOperation(ctx, models.FileOperationLog{
		LogEntry: models.LogEntry{
			Component: "fsops",
			Operation: "restore_backup",
			Message:   fmt.Sprintf("Restored from backup: %s", backupPath),
		},
		OperationType: "update",
		Path:          path,
	}); err != nil {
		return fmt.Errorf("failed to log restore operation: %w", err)
	}

	return nil
}

// DeleteBackup deletes a backup file
func (f *fileOps) DeleteBackup(ctx context.Context, path string) error {
	backupPath := path + ".backup"

	if err := f.ValidatePath(backupPath); err != nil {
		return err
	}

	// Check if backup exists
	exists, err := f.Exists(ctx, backupPath)
	if err != nil {
		return err
	}
	if !exists {
		// Not an error if backup doesn't exist
		return nil
	}

	return f.DeleteFile(ctx, backupPath)
}

// SafeUpdate performs an atomic update with automatic backup and rollback on error
func (f *fileOps) SafeUpdate(ctx context.Context, path, newContent string) error {
	// Create backup and write atomically
	backupPath, err := f.AtomicWriteWithBackup(ctx, path, newContent)
	if err != nil {
		return err
	}

	// Verify the written content
	writtenContent, err := f.ReadFile(ctx, path)
	if err != nil {
		// Try to restore from backup
		if backupPath != "" {
			if restoreErr := f.RestoreFromBackup(ctx, path); restoreErr != nil {
				return fmt.Errorf("failed to verify write and restore failed: %w (restore error: %v)", err, restoreErr)
			}
		}
		return fmt.Errorf("failed to verify written content: %w", err)
	}

	// Verify checksum
	expectedChecksum := f.GenerateChecksum(newContent)
	actualChecksum := f.GenerateChecksum(writtenContent)

	if expectedChecksum != actualChecksum {
		// Checksums don't match, restore from backup
		if backupPath != "" {
			if restoreErr := f.RestoreFromBackup(ctx, path); restoreErr != nil {
				return fmt.Errorf("checksum mismatch and restore failed: restore error: %v", restoreErr)
			}
		}
		return fmt.Errorf("checksum mismatch after write: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	// Success - optionally delete backup
	// For safety, we keep the backup until explicitly deleted
	return nil
}
