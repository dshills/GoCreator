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

// setupTestDir creates a temporary directory for testing
func setupTestDir(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "fsops-test-*")
	require.NoError(t, err)
	return tmpDir, func() {
		os.RemoveAll(tmpDir)
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     fsops.Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: fsops.Config{
				RootDir: t.TempDir(),
				Logger:  fsops.NewMemoryLogger(),
			},
			wantErr: false,
		},
		{
			name: "empty root dir",
			cfg: fsops.Config{
				RootDir: "",
				Logger:  fsops.NewMemoryLogger(),
			},
			wantErr: true,
		},
		{
			name: "nil logger (should use noop)",
			cfg: fsops.Config{
				RootDir: t.TempDir(),
				Logger:  nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ops, err := fsops.New(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, ops)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ops)
			}
		})
	}
}

func TestWriteFile(t *testing.T) {
	rootDir, cleanup := setupTestDir(t)
	defer cleanup()

	logger := fsops.NewMemoryLogger()
	ops, err := fsops.New(fsops.Config{
		RootDir: rootDir,
		Logger:  logger,
	})
	require.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name    string
		path    string
		content string
		wantErr bool
	}{
		{
			name:    "simple file",
			path:    "test.txt",
			content: "Hello, World!",
			wantErr: false,
		},
		{
			name:    "nested file",
			path:    "dir1/dir2/test.txt",
			content: "Nested content",
			wantErr: false,
		},
		{
			name:    "empty content",
			path:    "empty.txt",
			content: "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ops.WriteFile(ctx, tt.path, tt.content)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify file exists and has correct content
				content, err := ops.ReadFile(ctx, tt.path)
				assert.NoError(t, err)
				assert.Equal(t, tt.content, content)
			}
		})
	}

	// Verify logging
	fileOps := logger.GetFileOperations()
	assert.NotEmpty(t, fileOps)
}

func TestReadFile(t *testing.T) {
	rootDir, cleanup := setupTestDir(t)
	defer cleanup()

	ops, err := fsops.New(fsops.Config{
		RootDir: rootDir,
		Logger:  fsops.NewMemoryLogger(),
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Write a file first
	testContent := "Test content"
	err = ops.WriteFile(ctx, "test.txt", testContent)
	require.NoError(t, err)

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name:    "existing file",
			path:    "test.txt",
			want:    testContent,
			wantErr: false,
		},
		{
			name:    "non-existent file",
			path:    "nonexistent.txt",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := ops.ReadFile(ctx, tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, content)
			}
		})
	}
}

func TestDeleteFile(t *testing.T) {
	rootDir, cleanup := setupTestDir(t)
	defer cleanup()

	ops, err := fsops.New(fsops.Config{
		RootDir: rootDir,
		Logger:  fsops.NewMemoryLogger(),
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Write a file first
	err = ops.WriteFile(ctx, "test.txt", "content")
	require.NoError(t, err)

	// Verify file exists
	exists, err := ops.Exists(ctx, "test.txt")
	require.NoError(t, err)
	require.True(t, exists)

	// Delete the file
	err = ops.DeleteFile(ctx, "test.txt")
	assert.NoError(t, err)

	// Verify file no longer exists
	exists, err = ops.Exists(ctx, "test.txt")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestMkdirAll(t *testing.T) {
	rootDir, cleanup := setupTestDir(t)
	defer cleanup()

	ops, err := fsops.New(fsops.Config{
		RootDir: rootDir,
		Logger:  fsops.NewMemoryLogger(),
	})
	require.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name    string
		path    string
		perm    os.FileMode
		wantErr bool
	}{
		{
			name:    "simple directory",
			path:    "testdir",
			perm:    0755,
			wantErr: false,
		},
		{
			name:    "nested directory",
			path:    "dir1/dir2/dir3",
			perm:    0755,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ops.MkdirAll(ctx, tt.path, tt.perm)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify directory exists
				fullPath := filepath.Join(rootDir, tt.path)
				info, err := os.Stat(fullPath)
				assert.NoError(t, err)
				assert.True(t, info.IsDir())
			}
		})
	}
}

func TestExists(t *testing.T) {
	rootDir, cleanup := setupTestDir(t)
	defer cleanup()

	ops, err := fsops.New(fsops.Config{
		RootDir: rootDir,
		Logger:  fsops.NewMemoryLogger(),
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Create a test file
	err = ops.WriteFile(ctx, "test.txt", "content")
	require.NoError(t, err)

	// Create a test directory
	err = ops.MkdirAll(ctx, "testdir", 0755)
	require.NoError(t, err)

	tests := []struct {
		name       string
		path       string
		wantExists bool
		wantErr    bool
	}{
		{
			name:       "existing file",
			path:       "test.txt",
			wantExists: true,
			wantErr:    false,
		},
		{
			name:       "existing directory",
			path:       "testdir",
			wantExists: true,
			wantErr:    false,
		},
		{
			name:       "non-existent path",
			path:       "nonexistent.txt",
			wantExists: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := ops.Exists(ctx, tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantExists, exists)
			}
		})
	}
}

func TestChecksum(t *testing.T) {
	rootDir, cleanup := setupTestDir(t)
	defer cleanup()

	ops, err := fsops.New(fsops.Config{
		RootDir: rootDir,
		Logger:  fsops.NewMemoryLogger(),
	})
	require.NoError(t, err)

	ctx := context.Background()

	content := "Hello, World!"
	path := "test.txt"

	// Write file
	err = ops.WriteFile(ctx, path, content)
	require.NoError(t, err)

	// Calculate checksum
	checksum, err := ops.Checksum(ctx, path)
	assert.NoError(t, err)
	assert.NotEmpty(t, checksum)

	// Verify it matches GenerateChecksum
	expectedChecksum := ops.GenerateChecksum(content)
	assert.Equal(t, expectedChecksum, checksum)

	// Verify checksum is SHA-256 (64 hex characters)
	assert.Len(t, checksum, 64)
}

func TestGenerateChecksum(t *testing.T) {
	rootDir, cleanup := setupTestDir(t)
	defer cleanup()

	ops, err := fsops.New(fsops.Config{
		RootDir: rootDir,
		Logger:  fsops.NewMemoryLogger(),
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "simple content",
			content: "Hello, World!",
		},
		{
			name:    "empty content",
			content: "",
		},
		{
			name:    "multiline content",
			content: "Line 1\nLine 2\nLine 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checksum1 := ops.GenerateChecksum(tt.content)
			checksum2 := ops.GenerateChecksum(tt.content)

			// Checksums should be deterministic
			assert.Equal(t, checksum1, checksum2)

			// Should be 64 hex characters (SHA-256)
			assert.Len(t, checksum1, 64)

			// Different content should produce different checksums
			differentChecksum := ops.GenerateChecksum(tt.content + "different")
			if tt.content != "" {
				assert.NotEqual(t, checksum1, differentChecksum)
			}
		})
	}
}

func TestMemoryLogger(t *testing.T) {
	logger := fsops.NewMemoryLogger()
	ctx := context.Background()

	// Test that logger captures operations
	rootDir, cleanup := setupTestDir(t)
	defer cleanup()

	ops, err := fsops.New(fsops.Config{
		RootDir: rootDir,
		Logger:  logger,
	})
	require.NoError(t, err)

	// Perform some operations
	err = ops.WriteFile(ctx, "test1.txt", "content1")
	require.NoError(t, err)

	err = ops.WriteFile(ctx, "test2.txt", "content2")
	require.NoError(t, err)

	_, err = ops.ReadFile(ctx, "test1.txt")
	require.NoError(t, err)

	// Check logged operations
	fileOps := logger.GetFileOperations()
	assert.GreaterOrEqual(t, len(fileOps), 3, "Should have at least 3 operations logged")

	// Verify operations are logged with correct data
	for _, op := range fileOps {
		assert.NotEmpty(t, op.Path)
		assert.NotEmpty(t, op.OperationType)
	}

	// Test clear
	logger.Clear()
	assert.Empty(t, logger.GetEntries())
}

func TestConcurrentOperations(t *testing.T) {
	rootDir, cleanup := setupTestDir(t)
	defer cleanup()

	logger := fsops.NewMemoryLogger()
	ops, err := fsops.New(fsops.Config{
		RootDir: rootDir,
		Logger:  logger,
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Write multiple files concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(index int) {
			path := filepath.Join("concurrent", "file"+string(rune('0'+index))+".txt")
			content := "Content " + string(rune('0'+index))
			err := ops.WriteFile(ctx, path, content)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all files were created
	for i := 0; i < 10; i++ {
		path := filepath.Join("concurrent", "file"+string(rune('0'+i))+".txt")
		exists, err := ops.Exists(ctx, path)
		assert.NoError(t, err)
		assert.True(t, exists)
	}
}
