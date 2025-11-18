package generate

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncrementalStateManager_LoadAndSave(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		state     *IncrementalState
		wantLoad  bool
		wantFiles int
	}{
		{
			name: "save and load state",
			state: &IncrementalState{
				FCSChecksum: "abc123",
				GeneratedFiles: map[string]FileState{
					"file1.go": {
						Path:         "file1.go",
						Checksum:     "checksum1",
						GeneratedAt:  time.Now(),
						Dependencies: []string{"User", "Product"},
						Template:     false,
					},
					"file2.go": {
						Path:         "file2.go",
						Checksum:     "checksum2",
						GeneratedAt:  time.Now(),
						Dependencies: []string{"Order"},
						Template:     false,
					},
				},
				DependencyGraph: map[string][]string{
					"file1.go": {"User", "Product"},
					"file2.go": {"Order"},
				},
				LastGeneration: time.Now(),
				Version:        "1.0",
			},
			wantLoad:  true,
			wantFiles: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewIncrementalStateManager(tempDir)

			// Save state
			err := manager.Save(tt.state)
			require.NoError(t, err)

			// Verify file exists
			stateFile := filepath.Join(tempDir, ".gocreator", "state.json")
			_, err = os.Stat(stateFile)
			require.NoError(t, err)

			// Load state
			loadedState, err := manager.Load()
			if tt.wantLoad {
				require.NoError(t, err)
				assert.Equal(t, tt.state.FCSChecksum, loadedState.FCSChecksum)
				assert.Equal(t, tt.state.Version, loadedState.Version)
				assert.Len(t, loadedState.GeneratedFiles, tt.wantFiles)
				assert.Len(t, loadedState.DependencyGraph, tt.wantFiles)

				// Verify file states
				for path, expectedFile := range tt.state.GeneratedFiles {
					loadedFile, exists := loadedState.GeneratedFiles[path]
					assert.True(t, exists, "File %s should exist", path)
					assert.Equal(t, expectedFile.Path, loadedFile.Path)
					assert.Equal(t, expectedFile.Checksum, loadedFile.Checksum)
					assert.Equal(t, expectedFile.Dependencies, loadedFile.Dependencies)
					assert.Equal(t, expectedFile.Template, loadedFile.Template)
				}
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestIncrementalStateManager_LoadNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewIncrementalStateManager(tempDir)

	// Load should create empty state if file doesn't exist
	state, err := manager.Load()
	require.NoError(t, err)
	assert.NotNil(t, state)
	assert.Empty(t, state.FCSChecksum)
	assert.Empty(t, state.GeneratedFiles)
	assert.Empty(t, state.DependencyGraph)
	assert.Equal(t, "1.0", state.Version)
}

func TestIncrementalStateManager_UpdateState(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewIncrementalStateManager(tempDir)

	// Create test FCS
	fcs := &models.FinalClarifiedSpecification{
		ID:      "test-fcs",
		Version: "1.0",
		DataModel: models.DataModel{
			Entities: []models.Entity{
				{Name: "User", Package: "models"},
				{Name: "Product", Package: "models"},
			},
		},
	}

	// Create test patches
	patches := []models.Patch{
		{
			TargetFile: "models/user.go",
			Diff:       "+package models\n+type User struct {}\n",
			AppliedAt:  time.Now(),
			Reversible: true,
		},
		{
			TargetFile: "models/product.go",
			Diff:       "+package models\n+type Product struct {}\n",
			AppliedAt:  time.Now(),
			Reversible: true,
		},
	}

	// Create dependency graph
	dependencyGraph := map[string][]string{
		"models/user.go":    {"User"},
		"models/product.go": {"Product"},
	}

	// Update state
	err := manager.UpdateState(fcs, patches, dependencyGraph)
	require.NoError(t, err)

	// Load and verify
	state, err := manager.Load()
	require.NoError(t, err)

	assert.NotEmpty(t, state.FCSChecksum)
	assert.Len(t, state.GeneratedFiles, 2)
	assert.Len(t, state.DependencyGraph, 2)
	assert.False(t, state.LastGeneration.IsZero())

	// Verify file states
	userFile, exists := state.GeneratedFiles["models/user.go"]
	assert.True(t, exists)
	assert.Equal(t, "models/user.go", userFile.Path)
	assert.NotEmpty(t, userFile.Checksum)
	assert.Equal(t, []string{"User"}, userFile.Dependencies)

	productFile, exists := state.GeneratedFiles["models/product.go"]
	assert.True(t, exists)
	assert.Equal(t, "models/product.go", productFile.Path)
	assert.NotEmpty(t, productFile.Checksum)
	assert.Equal(t, []string{"Product"}, productFile.Dependencies)
}

func TestIncrementalStateManager_Clear(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewIncrementalStateManager(tempDir)

	// Create and save state
	state := &IncrementalState{
		FCSChecksum:     "test",
		GeneratedFiles:  make(map[string]FileState),
		DependencyGraph: make(map[string][]string),
		Version:         "1.0",
	}
	err := manager.Save(state)
	require.NoError(t, err)

	// Verify file exists
	stateFile := filepath.Join(tempDir, ".gocreator", "state.json")
	_, err = os.Stat(stateFile)
	require.NoError(t, err)

	// Clear state
	err = manager.Clear()
	require.NoError(t, err)

	// Verify file is deleted
	_, err = os.Stat(stateFile)
	assert.True(t, os.IsNotExist(err))
}

func TestComputeFCSChecksum(t *testing.T) {
	fcs1 := &models.FinalClarifiedSpecification{
		ID:      "test",
		Version: "1.0",
		DataModel: models.DataModel{
			Entities: []models.Entity{
				{Name: "User", Package: "models"},
			},
		},
	}

	fcs2 := &models.FinalClarifiedSpecification{
		ID:      "test",
		Version: "1.0",
		DataModel: models.DataModel{
			Entities: []models.Entity{
				{Name: "User", Package: "models"},
			},
		},
	}

	fcs3 := &models.FinalClarifiedSpecification{
		ID:      "test",
		Version: "1.0",
		DataModel: models.DataModel{
			Entities: []models.Entity{
				{Name: "User", Package: "models"},
				{Name: "Product", Package: "models"},
			},
		},
	}

	// Same FCS should produce same checksum
	checksum1, err := ComputeFCSChecksum(fcs1)
	require.NoError(t, err)
	assert.NotEmpty(t, checksum1)

	checksum2, err := ComputeFCSChecksum(fcs2)
	require.NoError(t, err)
	assert.Equal(t, checksum1, checksum2)

	// Different FCS should produce different checksum
	checksum3, err := ComputeFCSChecksum(fcs3)
	require.NoError(t, err)
	assert.NotEqual(t, checksum1, checksum3)
}

func TestComputeFileChecksum(t *testing.T) {
	content1 := "package main\n\nfunc main() {}\n"
	content2 := "package main\n\nfunc main() {}\n"
	content3 := "package main\n\nfunc main() { println(\"hello\") }\n"

	checksum1 := ComputeFileChecksum(content1)
	checksum2 := ComputeFileChecksum(content2)
	checksum3 := ComputeFileChecksum(content3)

	// Same content should produce same checksum
	assert.Equal(t, checksum1, checksum2)

	// Different content should produce different checksum
	assert.NotEqual(t, checksum1, checksum3)

	// Checksums should be hex strings
	assert.Len(t, checksum1, 64) // SHA-256 produces 64 hex characters
}

func TestExtractContentFromDiff(t *testing.T) {
	tests := []struct {
		name string
		diff string
		want string
	}{
		{
			name: "simple new file diff",
			diff: "@@ -0,0 +1,3 @@\n+package main\n+\n+func main() {}\n",
			want: "package main\n\nfunc main() {}\n",
		},
		{
			name: "diff with header",
			diff: "+++ b/file.go\n@@ -0,0 +1,2 @@\n+package test\n+\n",
			want: "package test\n\n",
		},
		{
			name: "empty diff",
			diff: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractContentFromDiff(tt.diff)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIncrementalStateManager_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewIncrementalStateManager(tempDir)

	// Create initial state
	state := &IncrementalState{
		FCSChecksum:     "initial",
		GeneratedFiles:  make(map[string]FileState),
		DependencyGraph: make(map[string][]string),
		Version:         "1.0",
	}

	// Save initial state
	err := manager.Save(state)
	require.NoError(t, err)

	// Multiple concurrent reads should work
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := manager.Load()
			assert.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all reads to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
