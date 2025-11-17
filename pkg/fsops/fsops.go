package fsops

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dshills/gocreator/internal/models"
)

// FileOps defines the interface for safe file operations
// All operations are bounded to a root directory and logged
type FileOps interface {
	// WriteFile writes content to a file within the bounded root
	// Returns error if path is outside root or write fails
	WriteFile(ctx context.Context, path, content string) error

	// ReadFile reads content from a file within the bounded root
	// Returns error if path is outside root or read fails
	ReadFile(ctx context.Context, path string) (string, error)

	// ApplyPatch applies a patch to a file within the bounded root
	// Returns error if patch application fails or path is outside root
	ApplyPatch(ctx context.Context, patch models.Patch) error

	// ValidatePath checks if a path is valid and within root
	// Returns error if path is invalid or outside root
	ValidatePath(path string) error

	// IsWithinRoot checks if a path is within the root directory
	// Uses proper canonicalization and symlink resolution for security
	// Returns (isWithin bool, error)
	IsWithinRoot(path string) (bool, error)

	// AtomicWrite writes content atomically using temp file + rename
	// Returns error if write fails or path is outside root
	AtomicWrite(ctx context.Context, path, content string) error

	// DeleteFile deletes a file within the bounded root
	// Returns error if path is outside root or deletion fails
	DeleteFile(ctx context.Context, path string) error

	// MkdirAll creates directories within the bounded root
	// Returns error if path is outside root or creation fails
	MkdirAll(ctx context.Context, path string, perm os.FileMode) error

	// Exists checks if a file or directory exists within the bounded root
	// Returns error if path is outside root
	Exists(ctx context.Context, path string) (bool, error)

	// Checksum calculates SHA-256 checksum of a file
	// Returns error if path is outside root or read fails
	Checksum(ctx context.Context, path string) (string, error)

	// GenerateChecksum calculates SHA-256 checksum of content
	GenerateChecksum(content string) string

	// GeneratePatch creates a patch from old content to new content
	GeneratePatch(ctx context.Context, targetFile, oldContent, newContent string) (models.Patch, error)

	// ValidatePatch validates a patch without applying it
	ValidatePatch(ctx context.Context, patch models.Patch) error

	// ReversePatch creates a reverse patch that undoes the given patch
	ReversePatch(ctx context.Context, patch models.Patch) (models.Patch, error)

	// CreateFilePatch creates a patch for creating a new file
	CreateFilePatch(ctx context.Context, targetFile, content string) (models.Patch, error)

	// DeleteFilePatch creates a patch for deleting a file
	DeleteFilePatch(ctx context.Context, targetFile string) (models.Patch, error)

	// GetPatchStats returns statistics about a patch
	GetPatchStats(patch models.Patch) (added, removed, modified int, err error)

	// ApplyPatchWithBackup applies a patch and creates a backup for reversal
	ApplyPatchWithBackup(ctx context.Context, patch models.Patch) error
}

// fileOps implements the FileOps interface
type fileOps struct {
	rootDir string
	logger  Logger
}

// Config holds configuration for FileOps
type Config struct {
	RootDir string
	Logger  Logger
}

// New creates a new FileOps instance with the given configuration
func New(cfg Config) (FileOps, error) {
	if cfg.RootDir == "" {
		return nil, fmt.Errorf("root directory cannot be empty")
	}

	// Get absolute path of root directory
	absRoot, err := filepath.Abs(cfg.RootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path of root: %w", err)
	}

	// Ensure root directory exists
	if err := os.MkdirAll(absRoot, 0755); err != nil {
		return nil, fmt.Errorf("failed to create root directory: %w", err)
	}

	// Clean the path to resolve any . or .. segments
	absRoot = filepath.Clean(absRoot)

	logger := cfg.Logger
	if logger == nil {
		logger = &noopLogger{}
	}

	return &fileOps{
		rootDir: absRoot,
		logger:  logger,
	}, nil
}

// WriteFile writes content to a file within the bounded root
func (f *fileOps) WriteFile(ctx context.Context, path, content string) error {
	if err := f.ValidatePath(path); err != nil {
		return err
	}

	absPath, err := f.getAbsolutePath(path)
	if err != nil {
		return err
	}

	// Log the operation before execution
	checksum := f.GenerateChecksum(content)
	if err := f.logger.LogFileOperation(ctx, models.FileOperationLog{
		LogEntry: models.LogEntry{
			Component: "fsops",
			Operation: "write_file",
			Message:   fmt.Sprintf("Writing file: %s", path),
		},
		OperationType: "create",
		Path:          path,
		Checksum:      checksum,
	}); err != nil {
		return fmt.Errorf("failed to log operation: %w", err)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write the file
	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	return nil
}

// ReadFile reads content from a file within the bounded root
func (f *fileOps) ReadFile(ctx context.Context, path string) (string, error) {
	if err := f.ValidatePath(path); err != nil {
		return "", err
	}

	absPath, err := f.getAbsolutePath(path)
	if err != nil {
		return "", err
	}

	// Log the operation
	if err := f.logger.LogFileOperation(ctx, models.FileOperationLog{
		LogEntry: models.LogEntry{
			Component: "fsops",
			Operation: "read_file",
			Message:   fmt.Sprintf("Reading file: %s", path),
		},
		OperationType: "read",
		Path:          path,
	}); err != nil {
		return "", fmt.Errorf("failed to log operation: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return string(data), nil
}

// DeleteFile deletes a file within the bounded root
func (f *fileOps) DeleteFile(ctx context.Context, path string) error {
	if err := f.ValidatePath(path); err != nil {
		return err
	}

	absPath, err := f.getAbsolutePath(path)
	if err != nil {
		return err
	}

	// Log the operation before execution
	if err := f.logger.LogFileOperation(ctx, models.FileOperationLog{
		LogEntry: models.LogEntry{
			Component: "fsops",
			Operation: "delete_file",
			Message:   fmt.Sprintf("Deleting file: %s", path),
		},
		OperationType: "delete",
		Path:          path,
	}); err != nil {
		return fmt.Errorf("failed to log operation: %w", err)
	}

	if err := os.Remove(absPath); err != nil {
		return fmt.Errorf("failed to delete file %s: %w", path, err)
	}

	return nil
}

// MkdirAll creates directories within the bounded root
func (f *fileOps) MkdirAll(ctx context.Context, path string, perm os.FileMode) error {
	if err := f.ValidatePath(path); err != nil {
		return err
	}

	absPath, err := f.getAbsolutePath(path)
	if err != nil {
		return err
	}

	// Log the operation
	if err := f.logger.LogFileOperation(ctx, models.FileOperationLog{
		LogEntry: models.LogEntry{
			Component: "fsops",
			Operation: "mkdir_all",
			Message:   fmt.Sprintf("Creating directory: %s", path),
		},
		OperationType: "create",
		Path:          path,
	}); err != nil {
		return fmt.Errorf("failed to log operation: %w", err)
	}

	if err := os.MkdirAll(absPath, perm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	return nil
}

// Exists checks if a file or directory exists within the bounded root
func (f *fileOps) Exists(ctx context.Context, path string) (bool, error) {
	if err := f.ValidatePath(path); err != nil {
		return false, err
	}

	absPath, err := f.getAbsolutePath(path)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(absPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check existence of %s: %w", path, err)
}

// Checksum calculates SHA-256 checksum of a file
func (f *fileOps) Checksum(ctx context.Context, path string) (string, error) {
	content, err := f.ReadFile(ctx, path)
	if err != nil {
		return "", err
	}
	return f.GenerateChecksum(content), nil
}

// GenerateChecksum calculates SHA-256 checksum of content
func (f *fileOps) GenerateChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// getAbsolutePath returns the absolute path for a given relative path
func (f *fileOps) getAbsolutePath(path string) (string, error) {
	// Join with root directory
	joined := filepath.Join(f.rootDir, path)

	// Clean the path to resolve any . or .. segments
	cleaned := filepath.Clean(joined)

	return cleaned, nil
}
