package fsops

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidatePath validates that a path is safe and within the root directory
// This is a critical security function that prevents path traversal attacks
// Uses proper canonicalization and symlink resolution for security
func (f *fileOps) ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check for null bytes (security issue)
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("path contains null byte")
	}

	// Check for absolute paths (should be relative to root)
	if filepath.IsAbs(path) {
		return fmt.Errorf("path must be relative, not absolute")
	}

	// Clean the path to resolve . and .. segments
	cleaned := filepath.Clean(path)

	// Verify the final path is within root using proper canonicalization
	isWithin, err := f.IsWithinRoot(cleaned)
	if err != nil {
		return fmt.Errorf("failed to validate path: %w", err)
	}
	if !isWithin {
		return fmt.Errorf("path is outside root directory")
	}

	return nil
}

// IsWithinRoot checks if a path is within the root directory
// Uses proper canonicalization and symlink resolution for security
// Returns (isWithin bool, error)
func (f *fileOps) IsWithinRoot(path string) (bool, error) {
	// Canonicalize root directory
	rootAbs, err := filepath.Abs(f.rootDir)
	if err != nil {
		return false, fmt.Errorf("failed to get absolute root path: %w", err)
	}

	// Resolve symlinks in root directory
	// Note: If symlinks cannot be resolved, we fall back to the absolute path
	rootEval, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		// If EvalSymlinks fails, use the cleaned absolute path
		rootEval = filepath.Clean(rootAbs)
	}

	// Construct the target absolute path
	targetAbs := filepath.Join(rootEval, path)
	targetAbs = filepath.Clean(targetAbs)

	// Resolve symlinks in target path
	// Note: If the target doesn't exist yet or symlinks can't be resolved,
	// we use the cleaned path (this is okay for validation before creation)
	targetEval, err := filepath.EvalSymlinks(targetAbs)
	if err != nil {
		// If EvalSymlinks fails (e.g., file doesn't exist yet), use cleaned path
		targetEval = filepath.Clean(targetAbs)
	}

	// Use filepath.Rel to determine if target is within root
	rel, err := filepath.Rel(rootEval, targetEval)
	if err != nil {
		return false, fmt.Errorf("failed to compute relative path: %w", err)
	}

	// Path is outside if it starts with ".." or is exactly ".."
	isOutside := strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".."

	return !isOutside, nil
}

// GetRootDir returns the root directory path
func (f *fileOps) GetRootDir() string {
	return f.rootDir
}

// NormalizePath normalizes a path relative to the root directory
// Returns an error if the path is outside the root
func (f *fileOps) NormalizePath(path string) (string, error) {
	if err := f.ValidatePath(path); err != nil {
		return "", err
	}

	// Clean and return the path
	return filepath.Clean(path), nil
}

// RelativePath returns the path relative to the root directory
// If the path is already relative, it's validated and returned
// If the path is absolute and within root, it's converted to relative
func (f *fileOps) RelativePath(path string) (string, error) {
	// If already relative, validate and return
	if !filepath.IsAbs(path) {
		if err := f.ValidatePath(path); err != nil {
			return "", err
		}
		return filepath.Clean(path), nil
	}

	// For absolute paths, ensure they're within root
	cleanPath := filepath.Clean(path)
	cleanRoot := filepath.Clean(f.rootDir)

	// Check if path is within root
	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	// Validate the resulting relative path
	if err := f.ValidatePath(rel); err != nil {
		return "", err
	}

	return rel, nil
}

// sanitizePath performs additional sanitization checks
// This is called internally before file operations
// Note: With proper root containment, this is defense-in-depth
func (f *fileOps) sanitizePath(path string) error {
	// Normalize the path for checking
	absPath, err := f.getAbsolutePath(path)
	if err != nil {
		return err
	}

	// Clean and normalize the path
	cleanPath := filepath.Clean(absPath)

	// Normalize path separators for consistent checking
	normalizedPath := filepath.ToSlash(cleanPath)

	// Convert to lowercase on Windows for case-insensitive checking
	// On Unix, we keep the original case
	checkPath := normalizedPath
	if filepath.Separator == '\\' {
		checkPath = strings.ToLower(normalizedPath)
	}

	// Check for common dangerous directory prefixes
	// These are checked as a defense-in-depth measure
	dangerousPrefixes := []string{
		"/etc/",
		"/var/",
		"/usr/",
		"/bin/",
		"/sbin/",
		"/sys/",
		"/proc/",
		"/dev/",
		"c:/windows/",
		"c:/program files/",
	}

	for _, prefix := range dangerousPrefixes {
		if strings.HasPrefix(checkPath, prefix) {
			return fmt.Errorf("path targets system directory")
		}
	}

	return nil
}
