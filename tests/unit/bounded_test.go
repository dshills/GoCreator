package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dshills/gocreator/pkg/fsops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePath(t *testing.T) {
	rootDir, cleanup := setupTestDir(t)
	defer cleanup()

	ops, err := fsops.New(fsops.Config{
		RootDir: rootDir,
		Logger:  fsops.NewMemoryLogger(),
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid relative path",
			path:    "test.txt",
			wantErr: false,
		},
		{
			name:    "valid nested path",
			path:    "dir/subdir/file.txt",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "path cannot be empty",
		},
		{
			name:    "absolute path",
			path:    "/etc/passwd",
			wantErr: true,
			errMsg:  "must be relative",
		},
		{
			name:    "path with ..",
			path:    "../outside.txt",
			wantErr: true,
			errMsg:  "outside root",
		},
		{
			name:    "path with .. in middle that stays within root",
			path:    "dir/../subdir/file.txt",
			wantErr: false,
		},
		{
			name:    "path with null byte",
			path:    "test\x00.txt",
			wantErr: true,
			errMsg:  "null byte",
		},
		{
			name:    "complex traversal attempt",
			path:    "dir/../../etc/passwd",
			wantErr: true,
			errMsg:  "outside root",
		},
		{
			name:    "path with multiple ..",
			path:    "../../file.txt",
			wantErr: true,
			errMsg:  "outside root",
		},
		{
			name:    "valid filename with double dots",
			path:    "file..txt",
			wantErr: false,
		},
		{
			name:    "valid filename with double dots in middle",
			path:    "my..special..file.txt",
			wantErr: false,
		},
		{
			name:    "valid backup filename",
			path:    "config..backup",
			wantErr: false,
		},
		{
			name:    "valid path with dots (not ..)",
			path:    "my.file.with.dots.txt",
			wantErr: false,
		},
		{
			name:    "valid hidden file",
			path:    ".gitignore",
			wantErr: false,
		},
		{
			name:    "valid path with single dot",
			path:    "./file.txt",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ops.ValidatePath(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsWithinRoot(t *testing.T) {
	rootDir, cleanup := setupTestDir(t)
	defer cleanup()

	ops, err := fsops.New(fsops.Config{
		RootDir: rootDir,
		Logger:  fsops.NewMemoryLogger(),
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		path    string
		within  bool
		wantErr bool
	}{
		{
			name:    "simple file",
			path:    "test.txt",
			within:  true,
			wantErr: false,
		},
		{
			name:    "nested file",
			path:    "dir/subdir/file.txt",
			within:  true,
			wantErr: false,
		},
		{
			name:    "traversal attempt",
			path:    "../outside.txt",
			within:  false,
			wantErr: false,
		},
		{
			name:    "current directory",
			path:    ".",
			within:  true,
			wantErr: false,
		},
		{
			name:    "hidden file",
			path:    ".hidden",
			within:  true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ops.IsWithinRoot(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.within, result)
			}
		})
	}
}

func TestPathTraversalAttacks(t *testing.T) {
	rootDir, cleanup := setupTestDir(t)
	defer cleanup()

	logger := fsops.NewMemoryLogger()
	ops, err := fsops.New(fsops.Config{
		RootDir: rootDir,
		Logger:  logger,
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Various path traversal attack patterns
	attacks := []string{
		"../etc/passwd",
		"../../etc/passwd",
		"dir/../../etc/passwd",
		"./../../etc/passwd",
		"dir/../../../etc/passwd",
		"./../../../etc/passwd",
		"legitimate/../../../etc/passwd",
	}

	for _, attack := range attacks {
		t.Run("Attack: "+attack, func(t *testing.T) {
			// Try to write - should fail
			err := ops.WriteFile(ctx, attack, "malicious content")
			assert.Error(t, err, "Path traversal attack should be blocked")

			// Try to read - should fail
			_, err = ops.ReadFile(ctx, attack)
			assert.Error(t, err, "Path traversal attack should be blocked")

			// Try to delete - should fail
			err = ops.DeleteFile(ctx, attack)
			assert.Error(t, err, "Path traversal attack should be blocked")
		})
	}

	// Verify no files were created outside root
	fileOps := logger.GetFileOperations()
	for _, op := range fileOps {
		// All logged operations should have paths within root
		assert.NotContains(t, op.Path, "..", "No operations should contain ..")
	}
}

func TestAbsolutePathBlocking(t *testing.T) {
	rootDir, cleanup := setupTestDir(t)
	defer cleanup()

	ops, err := fsops.New(fsops.Config{
		RootDir: rootDir,
		Logger:  fsops.NewMemoryLogger(),
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Common absolute paths (Unix-style only, as Windows paths aren't recognized as absolute on Unix)
	absolutePaths := []string{
		"/etc/passwd",
		"/tmp/sensitive",
		"/var/log/system.log",
		"/usr/bin/bash",
		"/root/.ssh/id_rsa",
	}

	for _, absPath := range absolutePaths {
		t.Run("Absolute: "+absPath, func(t *testing.T) {
			// All operations should fail on absolute paths
			err := ops.WriteFile(ctx, absPath, "content")
			assert.Error(t, err, "Absolute paths should be rejected")
			assert.Contains(t, err.Error(), "relative")

			_, err = ops.ReadFile(ctx, absPath)
			assert.Error(t, err, "Absolute paths should be rejected")
		})
	}
}

func TestNullByteInjection(t *testing.T) {
	rootDir, cleanup := setupTestDir(t)
	defer cleanup()

	ops, err := fsops.New(fsops.Config{
		RootDir: rootDir,
		Logger:  fsops.NewMemoryLogger(),
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Null byte injection attempts
	nullBytePaths := []string{
		"test\x00.txt",
		"test.txt\x00.secret",
		"\x00test.txt",
		"dir\x00/file.txt",
	}

	for _, path := range nullBytePaths {
		t.Run("NullByte: "+path, func(t *testing.T) {
			err := ops.WriteFile(ctx, path, "content")
			assert.Error(t, err, "Null byte injection should be blocked")
			assert.Contains(t, err.Error(), "null byte")
		})
	}
}

func TestBoundaryEnforcement(t *testing.T) {
	rootDir, cleanup := setupTestDir(t)
	defer cleanup()

	ops, err := fsops.New(fsops.Config{
		RootDir: rootDir,
		Logger:  fsops.NewMemoryLogger(),
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Write a legitimate file
	err = ops.WriteFile(ctx, "legitimate.txt", "content")
	require.NoError(t, err)

	// Verify it's within root
	legitimatePath := filepath.Join(rootDir, "legitimate.txt")
	assert.FileExists(t, legitimatePath)

	// Try to write outside root (this should fail)
	outsidePath := filepath.Join(rootDir, "..", "outside.txt")
	err = ops.WriteFile(ctx, "../outside.txt", "malicious")
	assert.Error(t, err)

	// Verify the outside file was NOT created
	assert.NoFileExists(t, outsidePath)
}

func TestSymlinkAttacks(t *testing.T) {
	rootDir, cleanup := setupTestDir(t)
	defer cleanup()

	ops, err := fsops.New(fsops.Config{
		RootDir: rootDir,
		Logger:  fsops.NewMemoryLogger(),
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Create a directory outside the root
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "secret.txt")
	err = os.WriteFile(outsideFile, []byte("sensitive data"), 0644)
	require.NoError(t, err)

	// Create a symlink inside root that points outside
	symlinkPath := filepath.Join(rootDir, "evil_link")
	err = os.Symlink(outsideFile, symlinkPath)
	require.NoError(t, err)

	// Try to read through the symlink - should be blocked
	_, err = ops.ReadFile(ctx, "evil_link")
	assert.Error(t, err, "Reading through symlink pointing outside root should be blocked")

	// Try to write through the symlink - should be blocked
	err = ops.WriteFile(ctx, "evil_link", "malicious content")
	assert.Error(t, err, "Writing through symlink pointing outside root should be blocked")

	// Verify the outside file was not modified
	content, err := os.ReadFile(outsideFile)
	require.NoError(t, err)
	assert.Equal(t, "sensitive data", string(content), "Outside file should not be modified")
}

func TestNormalizePath(t *testing.T) {
	rootDir, cleanup := setupTestDir(t)
	defer cleanup()

	ops, err := fsops.New(fsops.Config{
		RootDir: rootDir,
		Logger:  fsops.NewMemoryLogger(),
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name:    "simple path",
			path:    "test.txt",
			want:    "test.txt",
			wantErr: false,
		},
		{
			name:    "path with ./",
			path:    "./test.txt",
			want:    "test.txt",
			wantErr: false,
		},
		{
			name:    "path with redundant slashes",
			path:    "dir//file.txt",
			want:    filepath.Join("dir", "file.txt"),
			wantErr: false,
		},
		{
			name:    "path with ..",
			path:    "../outside.txt",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This tests an internal method if exposed, otherwise test through ValidatePath
			err := ops.ValidatePath(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConcurrentBoundaryChecks(t *testing.T) {
	rootDir, cleanup := setupTestDir(t)
	defer cleanup()

	ops, err := fsops.New(fsops.Config{
		RootDir: rootDir,
		Logger:  fsops.NewMemoryLogger(),
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Multiple goroutines trying different paths concurrently
	done := make(chan bool, 20)

	// 10 legitimate operations
	for i := 0; i < 10; i++ {
		go func(index int) {
			path := filepath.Join("safe", string(rune('a'+index))+".txt")
			err := ops.WriteFile(ctx, path, "content")
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// 10 malicious operations (should all fail)
	for i := 0; i < 10; i++ {
		go func(index int) {
			path := "../attack" + string(rune('a'+index)) + ".txt"
			err := ops.WriteFile(ctx, path, "malicious")
			assert.Error(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify legitimate files exist
	for i := 0; i < 10; i++ {
		path := filepath.Join("safe", string(rune('a'+i))+".txt")
		exists, err := ops.Exists(ctx, path)
		assert.NoError(t, err)
		assert.True(t, exists)
	}

	// Verify attack files don't exist
	parentDir := filepath.Dir(rootDir)
	for i := 0; i < 10; i++ {
		attackPath := filepath.Join(parentDir, "attack"+string(rune('a'+i))+".txt")
		assert.NoFileExists(t, attackPath)
	}
}
