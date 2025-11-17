package unit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerationPlan_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name string
		plan *models.GenerationPlan
	}{
		{
			name: "complete generation plan",
			plan: &models.GenerationPlan{
				SchemaVersion: "1.0",
				ID:            uuid.New().String(),
				FCSID:         uuid.New().String(),
				Phases: []models.GenerationPhase{
					{
						Name:  "setup",
						Order: 1,
						Tasks: []models.GenerationTask{
							{
								ID:          "task1",
								Type:        "generate_file",
								TargetPath:  "go.mod",
								Inputs:      map[string]interface{}{"module": "example.com/app"},
								CanParallel: false,
							},
						},
						Dependencies: []string{},
					},
					{
						Name:  "generate_models",
						Order: 2,
						Tasks: []models.GenerationTask{
							{
								ID:          "task2",
								Type:        "generate_file",
								TargetPath:  "internal/models/user.go",
								Inputs:      map[string]interface{}{"entity": "User"},
								CanParallel: true,
							},
							{
								ID:          "task3",
								Type:        "generate_file",
								TargetPath:  "internal/models/order.go",
								Inputs:      map[string]interface{}{"entity": "Order"},
								CanParallel: true,
							},
						},
						Dependencies: []string{"setup"},
					},
				},
				FileTree: models.FileTree{
					Root: "/project",
					Directories: []models.Directory{
						{Path: "internal/models", Purpose: "Domain models"},
						{Path: "cmd/app", Purpose: "Application entry point"},
					},
					Files: []models.File{
						{
							Path:        "go.mod",
							Purpose:     "Go module definition",
							GeneratedBy: "task1",
						},
						{
							Path:        "internal/models/user.go",
							Purpose:     "User entity",
							GeneratedBy: "task2",
						},
					},
				},
				CreatedAt: time.Now().UTC(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.plan)
			require.NoError(t, err)

			var unmarshaled models.GenerationPlan
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, tt.plan.ID, unmarshaled.ID)
			assert.Equal(t, tt.plan.FCSID, unmarshaled.FCSID)
			assert.Equal(t, len(tt.plan.Phases), len(unmarshaled.Phases))
			assert.Equal(t, tt.plan.FileTree.Root, unmarshaled.FileTree.Root)
		})
	}
}

func TestGenerationPlan_Validate(t *testing.T) {
	tests := []struct {
		name    string
		plan    *models.GenerationPlan
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid plan with acyclic phase dependencies",
			plan: &models.GenerationPlan{
				ID:    uuid.New().String(),
				FCSID: uuid.New().String(),
				Phases: []models.GenerationPhase{
					{Name: "phase1", Order: 1, Dependencies: []string{}},
					{Name: "phase2", Order: 2, Dependencies: []string{"phase1"}},
					{Name: "phase3", Order: 3, Dependencies: []string{"phase1", "phase2"}},
				},
				FileTree: models.FileTree{
					Root: "/project",
				},
			},
			wantErr: false,
		},
		{
			name: "valid plan with target paths within root",
			plan: &models.GenerationPlan{
				ID:    uuid.New().String(),
				FCSID: uuid.New().String(),
				Phases: []models.GenerationPhase{
					{
						Name:  "phase1",
						Order: 1,
						Tasks: []models.GenerationTask{
							{ID: "t1", Type: "generate_file", TargetPath: "/project/internal/main.go"},
							{ID: "t2", Type: "generate_file", TargetPath: "/project/cmd/app/main.go"},
						},
					},
				},
				FileTree: models.FileTree{
					Root: "/project",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid - cyclic phase dependencies",
			plan: &models.GenerationPlan{
				ID:    uuid.New().String(),
				FCSID: uuid.New().String(),
				Phases: []models.GenerationPhase{
					{Name: "phase1", Order: 1, Dependencies: []string{"phase2"}},
					{Name: "phase2", Order: 2, Dependencies: []string{"phase1"}},
				},
				FileTree: models.FileTree{Root: "/project"},
			},
			wantErr: true,
			errMsg:  "cyclic dependency",
		},
		{
			name: "invalid - target path outside root",
			plan: &models.GenerationPlan{
				ID:    uuid.New().String(),
				FCSID: uuid.New().String(),
				Phases: []models.GenerationPhase{
					{
						Name:  "phase1",
						Order: 1,
						Tasks: []models.GenerationTask{
							{ID: "t1", Type: "generate_file", TargetPath: "/etc/passwd"},
						},
					},
				},
				FileTree: models.FileTree{
					Root: "/project",
				},
			},
			wantErr: true,
			errMsg:  "target path outside root",
		},
		{
			name: "invalid - parallel tasks writing to same file",
			plan: &models.GenerationPlan{
				ID:    uuid.New().String(),
				FCSID: uuid.New().String(),
				Phases: []models.GenerationPhase{
					{
						Name:  "phase1",
						Order: 1,
						Tasks: []models.GenerationTask{
							{ID: "t1", Type: "generate_file", TargetPath: "/project/main.go", CanParallel: true},
							{ID: "t2", Type: "generate_file", TargetPath: "/project/main.go", CanParallel: true},
						},
					},
				},
				FileTree: models.FileTree{
					Root: "/project",
				},
			},
			wantErr: true,
			errMsg:  "parallel tasks cannot write to same file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plan.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGenerationPlan_DetectCyclicDependencies(t *testing.T) {
	tests := []struct {
		name      string
		phases    []models.GenerationPhase
		hasCycles bool
	}{
		{
			name: "no cycles - linear dependencies",
			phases: []models.GenerationPhase{
				{Name: "p1", Dependencies: []string{}},
				{Name: "p2", Dependencies: []string{"p1"}},
				{Name: "p3", Dependencies: []string{"p2"}},
			},
			hasCycles: false,
		},
		{
			name: "no cycles - parallel phases",
			phases: []models.GenerationPhase{
				{Name: "p1", Dependencies: []string{}},
				{Name: "p2", Dependencies: []string{}},
				{Name: "p3", Dependencies: []string{"p1", "p2"}},
			},
			hasCycles: false,
		},
		{
			name: "simple cycle",
			phases: []models.GenerationPhase{
				{Name: "p1", Dependencies: []string{"p2"}},
				{Name: "p2", Dependencies: []string{"p1"}},
			},
			hasCycles: true,
		},
		{
			name: "three-phase cycle",
			phases: []models.GenerationPhase{
				{Name: "p1", Dependencies: []string{"p2"}},
				{Name: "p2", Dependencies: []string{"p3"}},
				{Name: "p3", Dependencies: []string{"p1"}},
			},
			hasCycles: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &models.GenerationPlan{Phases: tt.phases}
			hasCycles := plan.HasCyclicDependencies()
			assert.Equal(t, tt.hasCycles, hasCycles)
		})
	}
}

func TestGenerationPlan_GetTaskByID(t *testing.T) {
	plan := &models.GenerationPlan{
		Phases: []models.GenerationPhase{
			{
				Name: "phase1",
				Tasks: []models.GenerationTask{
					{ID: "task1", Type: "generate_file"},
					{ID: "task2", Type: "apply_patch"},
				},
			},
			{
				Name: "phase2",
				Tasks: []models.GenerationTask{
					{ID: "task3", Type: "run_command"},
				},
			},
		},
	}

	tests := []struct {
		name       string
		taskID     string
		shouldFind bool
	}{
		{"find task in first phase", "task1", true},
		{"find task in second phase", "task3", true},
		{"task not found", "task999", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := plan.GetTaskByID(tt.taskID)
			if tt.shouldFind {
				require.NotNil(t, task)
				assert.Equal(t, tt.taskID, task.ID)
			} else {
				assert.Nil(t, task)
			}
		})
	}
}

func TestGenerationTask_Validate(t *testing.T) {
	tests := []struct {
		name    string
		task    models.GenerationTask
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid generate_file task",
			task: models.GenerationTask{
				ID:         "task1",
				Type:       "generate_file",
				TargetPath: "/project/main.go",
				Inputs:     map[string]interface{}{"template": "main"},
			},
			wantErr: false,
		},
		{
			name: "valid apply_patch task",
			task: models.GenerationTask{
				ID:         "task2",
				Type:       "apply_patch",
				TargetPath: "/project/existing.go",
				Inputs:     map[string]interface{}{"patch": "diff content"},
			},
			wantErr: false,
		},
		{
			name: "valid run_command task",
			task: models.GenerationTask{
				ID:     "task3",
				Type:   "run_command",
				Inputs: map[string]interface{}{"command": "go build"},
			},
			wantErr: false,
		},
		{
			name: "invalid - unknown task type",
			task: models.GenerationTask{
				ID:   "task4",
				Type: "invalid_type",
			},
			wantErr: true,
			errMsg:  "invalid task type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFileTree_JSONMarshaling(t *testing.T) {
	fileTree := &models.FileTree{
		Root: "/project",
		Directories: []models.Directory{
			{Path: "internal/models", Purpose: "Domain models"},
			{Path: "cmd/app", Purpose: "Application entry"},
		},
		Files: []models.File{
			{Path: "go.mod", Purpose: "Go module", GeneratedBy: "task1"},
			{Path: "main.go", Purpose: "Main entry point", GeneratedBy: "task2"},
		},
	}

	data, err := json.Marshal(fileTree)
	require.NoError(t, err)

	var unmarshaled models.FileTree
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, fileTree.Root, unmarshaled.Root)
	assert.Equal(t, len(fileTree.Directories), len(unmarshaled.Directories))
	assert.Equal(t, len(fileTree.Files), len(unmarshaled.Files))
}
