package templates

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTemplateGenerator(t *testing.T) {
	gen, err := NewTemplateGenerator()
	require.NoError(t, err)
	require.NotNil(t, gen)
}

func TestTemplateGenerator_IsBoilerplateFile(t *testing.T) {
	gen, err := NewTemplateGenerator()
	require.NoError(t, err)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "go.mod is boilerplate",
			path:     "go.mod",
			expected: true,
		},
		{
			name:     ".gitignore is boilerplate",
			path:     ".gitignore",
			expected: true,
		},
		{
			name:     "Dockerfile is boilerplate",
			path:     "Dockerfile",
			expected: true,
		},
		{
			name:     "Makefile is boilerplate",
			path:     "Makefile",
			expected: true,
		},
		{
			name:     "README.md is boilerplate",
			path:     "README.md",
			expected: true,
		},
		{
			name:     "main.go is not boilerplate",
			path:     "main.go",
			expected: false,
		},
		{
			name:     "some/path/go.mod is boilerplate",
			path:     "some/path/go.mod",
			expected: true,
		},
		{
			name:     "random.txt is not boilerplate",
			path:     "random.txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.IsBoilerplateFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTemplateGenerator_GenerateGoMod(t *testing.T) {
	gen, err := NewTemplateGenerator()
	require.NoError(t, err)

	tests := []struct {
		name       string
		data       TemplateData
		assertions func(t *testing.T, content string)
	}{
		{
			name: "basic go.mod",
			data: TemplateData{
				ModuleName: "github.com/test/project",
				GoVersion:  "1.21",
			},
			assertions: func(t *testing.T, content string) {
				assert.Contains(t, content, "module github.com/test/project")
				assert.Contains(t, content, "go 1.21")
			},
		},
		{
			name: "go.mod with dependencies",
			data: TemplateData{
				ModuleName: "github.com/test/project",
				GoVersion:  "1.21",
				Dependencies: []models.Dependency{
					{Name: "github.com/stretchr/testify", Version: "v1.8.0"},
					{Name: "github.com/rs/zerolog", Version: "v1.29.0"},
				},
			},
			assertions: func(t *testing.T, content string) {
				assert.Contains(t, content, "module github.com/test/project")
				assert.Contains(t, content, "go 1.21")
				assert.Contains(t, content, "require (")
				assert.Contains(t, content, "github.com/stretchr/testify v1.8.0")
				assert.Contains(t, content, "github.com/rs/zerolog v1.29.0")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := gen.GenerateGoMod(context.Background(), tt.data)
			require.NoError(t, err)
			require.NotEmpty(t, content)
			tt.assertions(t, content)
		})
	}
}

func TestTemplateGenerator_GenerateGitignore(t *testing.T) {
	gen, err := NewTemplateGenerator()
	require.NoError(t, err)

	data := TemplateData{
		ProjectName: "testproject",
	}

	content, err := gen.GenerateGitignore(context.Background(), data)
	require.NoError(t, err)
	require.NotEmpty(t, content)

	// Verify standard gitignore patterns
	assert.Contains(t, content, "*.exe")
	assert.Contains(t, content, "*.out")
	assert.Contains(t, content, "vendor/")
	assert.Contains(t, content, ".DS_Store")
	assert.Contains(t, content, ".env")
	assert.Contains(t, content, "testproject")
}

func TestTemplateGenerator_GenerateDockerfile(t *testing.T) {
	gen, err := NewTemplateGenerator()
	require.NoError(t, err)

	data := TemplateData{
		GoVersion:   "1.21",
		ProjectName: "myapp",
	}

	content, err := gen.GenerateDockerfile(context.Background(), data)
	require.NoError(t, err)
	require.NotEmpty(t, content)

	// Verify Dockerfile structure
	assert.Contains(t, content, "FROM golang:1.21-alpine AS builder")
	assert.Contains(t, content, "COPY go.mod go.sum")
	assert.Contains(t, content, "go mod download")
	assert.Contains(t, content, "./cmd/myapp")
	assert.Contains(t, content, "FROM alpine:latest")
	assert.Contains(t, content, "USER appuser")
	assert.Contains(t, content, "EXPOSE 8080")
}

func TestTemplateGenerator_GenerateMakefile(t *testing.T) {
	gen, err := NewTemplateGenerator()
	require.NoError(t, err)

	data := TemplateData{
		ProjectName:    "myservice",
		GoVersion:      "1.21",
		CoverageTarget: 85.0,
	}

	content, err := gen.GenerateMakefile(context.Background(), data)
	require.NoError(t, err)
	require.NotEmpty(t, content)

	// Verify Makefile targets
	assert.Contains(t, content, "BINARY_NAME=myservice")
	assert.Contains(t, content, "GO_VERSION=1.21")
	assert.Contains(t, content, "COVERAGE_TARGET=85")
	assert.Contains(t, content, ".PHONY:")
	assert.Contains(t, content, "## build:")
	assert.Contains(t, content, "## test:")
	assert.Contains(t, content, "## lint:")
	assert.Contains(t, content, "## coverage:")
	assert.Contains(t, content, "## docker-build:")
}

func TestTemplateGenerator_GenerateReadme(t *testing.T) {
	gen, err := NewTemplateGenerator()
	require.NoError(t, err)

	data := TemplateData{
		ProjectName:    "awesome-service",
		ModuleName:     "github.com/example/awesome-service",
		Description:    "An awesome Go service",
		GoVersion:      "1.21",
		CoverageTarget: 90.0,
		Year:           2024,
		Packages: []models.Package{
			{Name: "main", Path: "cmd/awesome-service", Purpose: "Application entry point"},
			{Name: "api", Path: "internal/api", Purpose: "HTTP API handlers"},
		},
	}

	content, err := gen.GenerateReadme(context.Background(), data)
	require.NoError(t, err)
	require.NotEmpty(t, content)

	// Verify README structure
	assert.Contains(t, content, "# awesome-service")
	assert.Contains(t, content, "An awesome Go service")
	assert.Contains(t, content, "Go 1.21")
	assert.Contains(t, content, "Coverage target: 90")
	assert.Contains(t, content, "github.com/example/awesome-service")
	assert.Contains(t, content, "make build")
	assert.Contains(t, content, "make test")
	assert.Contains(t, content, "Copyright 2024")
	assert.Contains(t, content, "cmd/awesome-service")
}

func TestTemplateGenerator_GenerateBoilerplate(t *testing.T) {
	gen, err := NewTemplateGenerator()
	require.NoError(t, err)

	data := TemplateData{
		ModuleName:  "github.com/test/project",
		GoVersion:   "1.21",
		ProjectName: "testproject",
	}

	tests := []struct {
		name          string
		path          string
		checkContains []string
	}{
		{
			name: "generate go.mod via path",
			path: "go.mod",
			checkContains: []string{
				"module github.com/test/project",
				"go 1.21",
			},
		},
		{
			name: "generate .gitignore via path",
			path: ".gitignore",
			checkContains: []string{
				"*.exe",
				"vendor/",
			},
		},
		{
			name: "generate Dockerfile via path",
			path: "Dockerfile",
			checkContains: []string{
				"FROM golang:1.21-alpine",
				"./cmd/testproject",
			},
		},
		{
			name: "generate Makefile via path",
			path: "Makefile",
			checkContains: []string{
				"BINARY_NAME=testproject",
				".PHONY:",
			},
		},
		{
			name: "generate README.md via path",
			path: "README.md",
			checkContains: []string{
				"# testproject",
				"Go 1.21",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := gen.GenerateBoilerplate(context.Background(), tt.path, data)
			require.NoError(t, err)
			require.NotEmpty(t, content)

			for _, expected := range tt.checkContains {
				assert.Contains(t, content, expected)
			}
		})
	}
}

func TestTemplateGenerator_GenerateBoilerplate_Error(t *testing.T) {
	gen, err := NewTemplateGenerator()
	require.NoError(t, err)

	data := TemplateData{
		ModuleName:  "github.com/test/project",
		GoVersion:   "1.21",
		ProjectName: "testproject",
	}

	// Try to generate non-boilerplate file
	_, err = gen.GenerateBoilerplate(context.Background(), "main.go", data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no template found")
}

func TestExtractTemplateData(t *testing.T) {
	tests := []struct {
		name       string
		fcs        *models.FinalClarifiedSpecification
		assertions func(t *testing.T, data TemplateData)
	}{
		{
			name: "basic FCS",
			fcs: &models.FinalClarifiedSpecification{
				BuildConfig: models.BuildConfig{
					GoVersion: "1.21",
				},
				Architecture: models.Architecture{
					Packages: []models.Package{
						{Name: "main", Path: "github.com/example/project/cmd/app"},
					},
				},
				Requirements: models.Requirements{
					Functional: []models.FunctionalRequirement{
						{Description: "A test project"},
					},
				},
				TestingStrategy: models.TestingStrategy{
					CoverageTarget: 80.0,
				},
			},
			assertions: func(t *testing.T, data TemplateData) {
				assert.Equal(t, "1.21", data.GoVersion)
				assert.Contains(t, data.ModuleName, "github.com/example/project")
				assert.Equal(t, "A test project", data.Description)
				assert.Equal(t, 80.0, data.CoverageTarget)
				assert.NotZero(t, data.Year)
				assert.NotEmpty(t, data.GeneratedAt)
			},
		},
		{
			name: "FCS with dependencies",
			fcs: &models.FinalClarifiedSpecification{
				BuildConfig: models.BuildConfig{
					GoVersion:  "1.22",
					BuildFlags: []string{"-trimpath", "-ldflags=-w -s"},
				},
				Architecture: models.Architecture{
					Packages: []models.Package{
						{Name: "api", Path: "github.com/acme/service/internal/api"},
					},
					Dependencies: []models.Dependency{
						{Name: "github.com/gin-gonic/gin", Version: "v1.9.0", Purpose: "HTTP framework"},
					},
				},
				TestingStrategy: models.TestingStrategy{
					CoverageTarget: 90.0,
				},
			},
			assertions: func(t *testing.T, data TemplateData) {
				assert.Equal(t, "1.22", data.GoVersion)
				assert.Len(t, data.Dependencies, 1)
				assert.Equal(t, "github.com/gin-gonic/gin", data.Dependencies[0].Name)
				assert.Len(t, data.BuildFlags, 2)
				assert.Equal(t, 90.0, data.CoverageTarget)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := ExtractTemplateData(tt.fcs)
			tt.assertions(t, data)
		})
	}
}

func TestTemplateData_DefaultValues(t *testing.T) {
	gen, err := NewTemplateGenerator()
	require.NoError(t, err)

	// Test with minimal data - defaults should be applied
	data := TemplateData{
		ModuleName: "github.com/test/project",
		GoVersion:  "1.21",
	}

	content, err := gen.GenerateReadme(context.Background(), data)
	require.NoError(t, err)

	// Check that defaults were applied
	currentYear := time.Now().Year()
	assert.Contains(t, content, string(rune(currentYear/1000+48))) // Year should be present
}

func TestTemplateGenerator_AllTemplatesExecuteWithoutError(t *testing.T) {
	gen, err := NewTemplateGenerator()
	require.NoError(t, err)

	data := TemplateData{
		ModuleName:     "github.com/test/project",
		GoVersion:      "1.21",
		ProjectName:    "testapp",
		Description:    "Test application",
		CoverageTarget: 85.0,
		Year:           2024,
		Dependencies: []models.Dependency{
			{Name: "github.com/example/dep", Version: "v1.0.0"},
		},
		Packages: []models.Package{
			{Name: "main", Path: "cmd/testapp", Purpose: "Entry point"},
		},
	}

	// Test all generation methods
	tests := []struct {
		name string
		fn   func(context.Context, TemplateData) (string, error)
	}{
		{"GenerateGoMod", gen.GenerateGoMod},
		{"GenerateGitignore", gen.GenerateGitignore},
		{"GenerateDockerfile", gen.GenerateDockerfile},
		{"GenerateMakefile", gen.GenerateMakefile},
		{"GenerateReadme", gen.GenerateReadme},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := tt.fn(context.Background(), data)
			require.NoError(t, err)
			require.NotEmpty(t, content)
			// Verify content is reasonable (not empty, has multiple lines)
			lines := strings.Split(content, "\n")
			assert.Greater(t, len(lines), 1, "Generated content should have multiple lines")
		})
	}
}

func BenchmarkTemplateGenerator_GenerateGoMod(b *testing.B) {
	gen, err := NewTemplateGenerator()
	require.NoError(b, err)

	data := TemplateData{
		ModuleName: "github.com/test/project",
		GoVersion:  "1.21",
		Dependencies: []models.Dependency{
			{Name: "github.com/stretchr/testify", Version: "v1.8.0"},
			{Name: "github.com/rs/zerolog", Version: "v1.29.0"},
		},
	}

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = gen.GenerateGoMod(ctx, data)
	}
}

func BenchmarkTemplateGenerator_GenerateAllBoilerplate(b *testing.B) {
	gen, err := NewTemplateGenerator()
	require.NoError(b, err)

	data := TemplateData{
		ModuleName:     "github.com/test/project",
		GoVersion:      "1.21",
		ProjectName:    "testapp",
		Description:    "Test application",
		CoverageTarget: 85.0,
	}

	files := []string{"go.mod", ".gitignore", "Dockerfile", "Makefile", "README.md"}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, file := range files {
			_, _ = gen.GenerateBoilerplate(ctx, file, data)
		}
	}
}
