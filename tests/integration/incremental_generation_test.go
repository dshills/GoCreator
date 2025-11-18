package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dshills/gocreator/internal/generate"
	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/pkg/fsops"
	"github.com/dshills/gocreator/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockIncrementalLLMClient provides deterministic responses for testing incremental regeneration
type mockIncrementalLLMClient struct {
	generateCallCount int
	generatedFiles    []string
}

func (m *mockIncrementalLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	m.generateCallCount++

	// Return simple file content based on what's being requested
	if containsString(prompt, "User") {
		m.generatedFiles = append(m.generatedFiles, "User")
		return `package models

type User struct {
	ID   string
	Name string
}
`, nil
	} else if containsString(prompt, "Product") {
		m.generatedFiles = append(m.generatedFiles, "Product")
		return `package models

type Product struct {
	ID    string
	Name  string
	Price float64
}
`, nil
	} else if containsString(prompt, "Order") {
		m.generatedFiles = append(m.generatedFiles, "Order")
		return `package models

type Order struct {
	ID        string
	UserID    string
	ProductID string
}
`, nil
	}

	return "package test\n", nil
}

func (m *mockIncrementalLLMClient) GenerateStructured(ctx context.Context, prompt string, schema interface{}) (interface{}, error) {
	return nil, nil
}

func (m *mockIncrementalLLMClient) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	return "", nil
}

func (m *mockIncrementalLLMClient) Provider() string {
	return "mock-incremental"
}

func (m *mockIncrementalLLMClient) Model() string {
	return "mock-model"
}

// TestIncrementalGeneration_FirstGeneration tests initial generation with incremental mode
func TestIncrementalGeneration_FirstGeneration(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create FCS with two entities
	fcs := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-incremental-first",
		Version:        "1.0",
		OriginalSpecID: "spec-1",
		DataModel: models.DataModel{
			Entities: []models.Entity{
				{Name: "User", Package: "models", Attributes: map[string]string{"ID": "string", "Name": "string"}},
				{Name: "Product", Package: "models", Attributes: map[string]string{"ID": "string", "Name": "string", "Price": "float64"}},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "models", Path: "internal/models", Purpose: "Data models"},
			},
		},
		BuildConfig: models.BuildConfig{
			GoVersion: "1.21",
		},
	}

	// Create mock LLM client
	mockClient := &mockIncrementalLLMClient{
		generatedFiles: []string{},
	}

	// Create file operations
	logger, err := fsops.NewFileLogger(filepath.Join(tempDir, ".gocreator", "logs"))
	require.NoError(t, err)
	defer logger.Close()

	fileOps, err := fsops.New(fsops.Config{
		RootDir: tempDir,
		Logger:  logger,
	})
	require.NoError(t, err)

	// Create coder with incremental mode enabled
	coder, err := generate.NewCoder(generate.CoderConfig{
		LLMClient:   mockClient,
		OutputDir:   tempDir,
		Incremental: true,
	})
	require.NoError(t, err)

	// Create generation plan
	plan := &models.GenerationPlan{
		SchemaVersion: "1.0",
		ID:            "plan-1",
		FCSID:         "test-incremental-first",
		Phases: []models.GenerationPhase{
			{
				Name:  "entity-generation",
				Order: 1,
				Tasks: []models.GenerationTask{
					{
						ID:          "task-1",
						Type:        "generate_file",
						TargetPath:  "internal/models/user.go",
						CanParallel: true,
					},
					{
						ID:          "task-2",
						Type:        "generate_file",
						TargetPath:  "internal/models/product.go",
						CanParallel: true,
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	// Generate
	ctx := context.Background()
	patches, err := coder.Generate(ctx, plan, fcs)
	require.NoError(t, err)
	require.Len(t, patches, 2, "should generate two files")

	// Apply patches
	for _, patch := range patches {
		err := fileOps.ApplyPatchWithBackup(ctx, patch)
		require.NoError(t, err)
	}

	// Verify state file was created
	stateFile := filepath.Join(tempDir, ".gocreator", "state.json")
	_, err = os.Stat(stateFile)
	require.NoError(t, err, "state file should be created")

	// Verify both files were generated
	assert.Contains(t, mockClient.generatedFiles, "User")
	assert.Contains(t, mockClient.generatedFiles, "Product")
	assert.Equal(t, 2, mockClient.generateCallCount, "should call LLM for both files")
}

// TestIncrementalGeneration_NoChanges tests that no regeneration happens when FCS is unchanged
func TestIncrementalGeneration_NoChanges(t *testing.T) {
	tempDir := t.TempDir()

	// Create initial FCS
	fcs := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-incremental-nochange",
		Version:        "1.0",
		OriginalSpecID: "spec-2",
		DataModel: models.DataModel{
			Entities: []models.Entity{
				{Name: "User", Package: "models", Attributes: map[string]string{"ID": "string"}},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "models", Path: "internal/models", Purpose: "Data models"},
			},
		},
		BuildConfig: models.BuildConfig{
			GoVersion: "1.21",
		},
	}

	mockClient := &mockIncrementalLLMClient{generatedFiles: []string{}}

	logger, err := fsops.NewFileLogger(filepath.Join(tempDir, ".gocreator", "logs"))
	require.NoError(t, err)
	defer logger.Close()

	fileOps, err := fsops.New(fsops.Config{
		RootDir: tempDir,
		Logger:  logger,
	})
	require.NoError(t, err)

	coder, err := generate.NewCoder(generate.CoderConfig{
		LLMClient:   mockClient,
		OutputDir:   tempDir,
		Incremental: true,
	})
	require.NoError(t, err)

	plan := &models.GenerationPlan{
		SchemaVersion: "1.0",
		ID:            "plan-2",
		FCSID:         "test-incremental-nochange",
		Phases: []models.GenerationPhase{
			{
				Name:  "entity-generation",
				Order: 1,
				Tasks: []models.GenerationTask{
					{
						ID:          "task-1",
						Type:        "generate_file",
						TargetPath:  "internal/models/user.go",
						CanParallel: true,
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	// First generation
	ctx := context.Background()
	patches1, err := coder.Generate(ctx, plan, fcs)
	require.NoError(t, err)
	require.Len(t, patches1, 1)

	for _, patch := range patches1 {
		err := fileOps.ApplyPatchWithBackup(ctx, patch)
		require.NoError(t, err)
	}

	firstGenCount := mockClient.generateCallCount
	mockClient.generateCallCount = 0
	mockClient.generatedFiles = []string{}

	// Second generation with same FCS
	patches2, err := coder.Generate(ctx, plan, fcs)
	require.NoError(t, err)

	// Should generate no patches when FCS is unchanged
	assert.Len(t, patches2, 0, "should not regenerate when FCS is unchanged")
	assert.Equal(t, 0, mockClient.generateCallCount, "should not call LLM when nothing changed")
	assert.Greater(t, firstGenCount, 0, "first generation should have called LLM")
}

// TestIncrementalGeneration_EntityModified tests regeneration when entity is modified
func TestIncrementalGeneration_EntityModified(t *testing.T) {
	tempDir := t.TempDir()

	// Initial FCS
	fcs1 := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-incremental-modify",
		Version:        "1.0",
		OriginalSpecID: "spec-3",
		DataModel: models.DataModel{
			Entities: []models.Entity{
				{Name: "User", Package: "models", Attributes: map[string]string{"ID": "string", "Name": "string"}},
				{Name: "Product", Package: "models", Attributes: map[string]string{"ID": "string", "Name": "string"}},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "models", Path: "internal/models", Purpose: "Data models"},
			},
		},
		BuildConfig: models.BuildConfig{
			GoVersion: "1.21",
		},
	}

	mockClient := &mockIncrementalLLMClient{generatedFiles: []string{}}

	logger, err := fsops.NewFileLogger(filepath.Join(tempDir, ".gocreator", "logs"))
	require.NoError(t, err)
	defer logger.Close()

	fileOps, err := fsops.New(fsops.Config{
		RootDir: tempDir,
		Logger:  logger,
	})
	require.NoError(t, err)

	coder, err := generate.NewCoder(generate.CoderConfig{
		LLMClient:   mockClient,
		OutputDir:   tempDir,
		Incremental: true,
	})
	require.NoError(t, err)

	plan := &models.GenerationPlan{
		SchemaVersion: "1.0",
		ID:            "plan-3",
		FCSID:         "test-incremental-modify",
		Phases: []models.GenerationPhase{
			{
				Name:  "entity-generation",
				Order: 1,
				Tasks: []models.GenerationTask{
					{
						ID:          "task-1",
						Type:        "generate_file",
						TargetPath:  "internal/models/user.go",
						CanParallel: true,
					},
					{
						ID:          "task-2",
						Type:        "generate_file",
						TargetPath:  "internal/models/product.go",
						CanParallel: true,
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	// First generation
	ctx := context.Background()
	patches1, err := coder.Generate(ctx, plan, fcs1)
	require.NoError(t, err)
	require.Len(t, patches1, 2)

	for _, patch := range patches1 {
		err := fileOps.ApplyPatchWithBackup(ctx, patch)
		require.NoError(t, err)
	}

	mockClient.generateCallCount = 0
	mockClient.generatedFiles = []string{}

	// Wait a bit to ensure timestamps differ
	time.Sleep(10 * time.Millisecond)

	// Modified FCS - add Email attribute to User
	fcs2 := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-incremental-modify",
		Version:        "1.1",
		OriginalSpecID: "spec-3",
		DataModel: models.DataModel{
			Entities: []models.Entity{
				{Name: "User", Package: "models", Attributes: map[string]string{"ID": "string", "Name": "string", "Email": "string"}}, // Modified
				{Name: "Product", Package: "models", Attributes: map[string]string{"ID": "string", "Name": "string"}},                 // Unchanged
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "models", Path: "internal/models", Purpose: "Data models"},
			},
		},
		BuildConfig: models.BuildConfig{
			GoVersion: "1.21",
		},
	}

	// Second generation with modified User entity
	// Note: Current implementation doesn't have old FCS stored yet, so it will regenerate new files
	patches2, err := coder.Generate(ctx, plan, fcs2)
	require.NoError(t, err)

	// In the current implementation, it should detect the FCS changed and find new files or changes
	// Since we're modifying the task list, it may regenerate
	assert.NotEmpty(t, patches2, "should regenerate when FCS changes")
}

// TestIncrementalGeneration_StateFilePersistence tests state file persistence across runs
func TestIncrementalGeneration_StateFilePersistence(t *testing.T) {
	tempDir := t.TempDir()

	// Create state manager
	stateManager := generate.NewIncrementalStateManager(tempDir)
	var err error

	// Manually create and save state
	state := &generate.IncrementalState{
		FCSChecksum: "test-checksum",
		GeneratedFiles: map[string]generate.FileState{
			"internal/models/user.go": {
				Path:         "internal/models/user.go",
				Checksum:     "file-checksum",
				GeneratedAt:  time.Now(),
				Dependencies: []string{"User"},
			},
		},
		DependencyGraph: map[string][]string{
			"internal/models/user.go": {"User"},
		},
		LastGeneration: time.Now(),
		Version:        "1.0",
	}

	err = stateManager.Save(state)
	require.NoError(t, err)

	// Verify state file exists
	stateFile := filepath.Join(tempDir, ".gocreator", "state.json")
	_, err = os.Stat(stateFile)
	require.NoError(t, err, "state file should exist")

	// Load state in a new manager
	newStateManager := generate.NewIncrementalStateManager(tempDir)
	loadedState, err := newStateManager.Load()
	require.NoError(t, err)

	// Verify loaded state matches
	assert.Equal(t, state.FCSChecksum, loadedState.FCSChecksum)
	assert.Len(t, loadedState.GeneratedFiles, 1)
	assert.Contains(t, loadedState.GeneratedFiles, "internal/models/user.go")
	assert.Equal(t, state.Version, loadedState.Version)
}

// TestIncrementalGeneration_FallbackToFullGeneration tests graceful fallback when state is corrupted
func TestIncrementalGeneration_FallbackToFullGeneration(t *testing.T) {
	tempDir := t.TempDir()

	// Create corrupted state file
	stateDir := filepath.Join(tempDir, ".gocreator")
	err := os.MkdirAll(stateDir, 0750)
	require.NoError(t, err)

	stateFile := filepath.Join(stateDir, "state.json")
	err = os.WriteFile(stateFile, []byte("corrupted json {{{"), 0600)
	require.NoError(t, err)

	fcs := &models.FinalClarifiedSpecification{
		SchemaVersion:  "1.0",
		ID:             "test-fallback",
		Version:        "1.0",
		OriginalSpecID: "spec-5",
		DataModel: models.DataModel{
			Entities: []models.Entity{
				{Name: "User", Package: "models", Attributes: map[string]string{"ID": "string"}},
			},
		},
		Architecture: models.Architecture{
			Packages: []models.Package{
				{Name: "models", Path: "internal/models", Purpose: "Data models"},
			},
		},
		BuildConfig: models.BuildConfig{
			GoVersion: "1.21",
		},
	}

	mockClient := &mockIncrementalLLMClient{generatedFiles: []string{}}

	coder, err := generate.NewCoder(generate.CoderConfig{
		LLMClient:   mockClient,
		OutputDir:   tempDir,
		Incremental: true,
	})
	require.NoError(t, err)

	plan := &models.GenerationPlan{
		SchemaVersion: "1.0",
		ID:            "plan-5",
		FCSID:         "test-fallback",
		Phases: []models.GenerationPhase{
			{
				Name:  "entity-generation",
				Order: 1,
				Tasks: []models.GenerationTask{
					{
						ID:          "task-1",
						Type:        "generate_file",
						TargetPath:  "internal/models/user.go",
						CanParallel: true,
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	// Should fall back to full generation despite corrupted state
	ctx := context.Background()
	patches, err := coder.Generate(ctx, plan, fcs)
	require.NoError(t, err, "should fall back to full generation on corrupted state")
	assert.NotEmpty(t, patches, "should generate files in fallback mode")
	assert.Contains(t, mockClient.generatedFiles, "User", "should generate User entity")
}
