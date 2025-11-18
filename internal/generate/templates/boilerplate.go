// Package templates provides template-based boilerplate file generation for Go projects.
package templates

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/rs/zerolog/log"
)

//go:embed files/*.tmpl
var templateFS embed.FS

// TemplateData contains data for template rendering
type TemplateData struct {
	ModuleName     string
	GoVersion      string
	ProjectName    string
	Description    string
	Dependencies   []models.Dependency
	Packages       []models.Package
	BuildFlags     []string
	Year           int
	GeneratedAt    string
	CoverageTarget float64
}

// TemplateGenerator generates boilerplate files from templates without LLM calls
type TemplateGenerator interface {
	// GenerateGoMod generates a go.mod file
	GenerateGoMod(ctx context.Context, data TemplateData) (string, error)

	// GenerateGitignore generates a .gitignore file
	GenerateGitignore(ctx context.Context, data TemplateData) (string, error)

	// GenerateDockerfile generates a Dockerfile
	GenerateDockerfile(ctx context.Context, data TemplateData) (string, error)

	// GenerateMakefile generates a Makefile
	GenerateMakefile(ctx context.Context, data TemplateData) (string, error)

	// GenerateReadme generates a README.md file
	GenerateReadme(ctx context.Context, data TemplateData) (string, error)

	// IsBoilerplateFile returns true if the file should be generated via template
	IsBoilerplateFile(path string) bool

	// GenerateBoilerplate generates any boilerplate file by path
	GenerateBoilerplate(ctx context.Context, path string, data TemplateData) (string, error)
}

// templateGenerator implements TemplateGenerator
type templateGenerator struct {
	templates      map[string]*template.Template
	boilerplateMap map[string]string // maps file paths to template names
}

// NewTemplateGenerator creates a new template-based generator
func NewTemplateGenerator() (TemplateGenerator, error) {
	gen := &templateGenerator{
		templates: make(map[string]*template.Template),
		boilerplateMap: map[string]string{
			"go.mod":     "go.mod.tmpl",
			".gitignore": ".gitignore.tmpl",
			"Dockerfile": "Dockerfile.tmpl",
			"Makefile":   "Makefile.tmpl",
			"README.md":  "README.md.tmpl",
		},
	}

	// Load all templates
	if err := gen.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	log.Info().
		Int("templates", len(gen.templates)).
		Msg("Template generator initialized")

	return gen, nil
}

// loadTemplates loads all template files from the embedded filesystem
func (g *templateGenerator) loadTemplates() error {
	for _, tmplName := range []string{
		"go.mod.tmpl",
		".gitignore.tmpl",
		"Dockerfile.tmpl",
		"Makefile.tmpl",
		"README.md.tmpl",
	} {
		content, err := templateFS.ReadFile("files/" + tmplName)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", tmplName, err)
		}

		tmpl, err := template.New(tmplName).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", tmplName, err)
		}

		g.templates[tmplName] = tmpl
		log.Debug().
			Str("template", tmplName).
			Msg("Template loaded")
	}

	return nil
}

// IsBoilerplateFile returns true if the file should be generated via template
func (g *templateGenerator) IsBoilerplateFile(path string) bool {
	// Normalize path - handle both absolute and relative paths
	normalizedPath := path
	if idx := strings.LastIndex(path, "/"); idx != -1 {
		normalizedPath = path[idx+1:]
	}

	_, exists := g.boilerplateMap[normalizedPath]
	return exists
}

// GenerateGoMod generates a go.mod file
func (g *templateGenerator) GenerateGoMod(ctx context.Context, data TemplateData) (string, error) {
	return g.executeTemplate(ctx, "go.mod.tmpl", data)
}

// GenerateGitignore generates a .gitignore file
func (g *templateGenerator) GenerateGitignore(ctx context.Context, data TemplateData) (string, error) {
	return g.executeTemplate(ctx, ".gitignore.tmpl", data)
}

// GenerateDockerfile generates a Dockerfile
func (g *templateGenerator) GenerateDockerfile(ctx context.Context, data TemplateData) (string, error) {
	return g.executeTemplate(ctx, "Dockerfile.tmpl", data)
}

// GenerateMakefile generates a Makefile
func (g *templateGenerator) GenerateMakefile(ctx context.Context, data TemplateData) (string, error) {
	return g.executeTemplate(ctx, "Makefile.tmpl", data)
}

// GenerateReadme generates a README.md file
func (g *templateGenerator) GenerateReadme(ctx context.Context, data TemplateData) (string, error) {
	return g.executeTemplate(ctx, "README.md.tmpl", data)
}

// GenerateBoilerplate generates any boilerplate file by path
func (g *templateGenerator) GenerateBoilerplate(ctx context.Context, path string, data TemplateData) (string, error) {
	// Normalize path
	normalizedPath := path
	if idx := strings.LastIndex(path, "/"); idx != -1 {
		normalizedPath = path[idx+1:]
	}

	templateName, exists := g.boilerplateMap[normalizedPath]
	if !exists {
		return "", fmt.Errorf("no template found for path: %s", path)
	}

	return g.executeTemplate(ctx, templateName, data)
}

// executeTemplate executes a template with the given data
func (g *templateGenerator) executeTemplate(_ context.Context, templateName string, data TemplateData) (string, error) {
	tmpl, exists := g.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template not found: %s", templateName)
	}

	// Set default values if not provided
	if data.Year == 0 {
		data.Year = time.Now().Year()
	}
	if data.GeneratedAt == "" {
		data.GeneratedAt = time.Now().Format(time.RFC3339)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	log.Debug().
		Str("template", templateName).
		Int("output_length", buf.Len()).
		Msg("Template executed successfully")

	return buf.String(), nil
}

// ExtractTemplateData extracts template data from an FCS
func ExtractTemplateData(fcs *models.FinalClarifiedSpecification) TemplateData {
	// Extract module name from packages or use a default
	moduleName := "github.com/example/project"
	if len(fcs.Architecture.Packages) > 0 {
		// Try to extract from first package path
		firstPath := fcs.Architecture.Packages[0].Path
		if idx := strings.Index(firstPath, "/"); idx > 0 {
			moduleName = firstPath[:strings.LastIndex(firstPath, "/")]
		}
	}

	// Extract project name from module name
	projectName := "project"
	if parts := strings.Split(moduleName, "/"); len(parts) > 0 {
		projectName = parts[len(parts)-1]
	}

	// Build description from requirements
	description := "A Go application"
	if len(fcs.Requirements.Functional) > 0 {
		description = fcs.Requirements.Functional[0].Description
	}

	data := TemplateData{
		ModuleName:     moduleName,
		GoVersion:      fcs.BuildConfig.GoVersion,
		ProjectName:    projectName,
		Description:    description,
		Dependencies:   fcs.Architecture.Dependencies,
		Packages:       fcs.Architecture.Packages,
		BuildFlags:     fcs.BuildConfig.BuildFlags,
		Year:           time.Now().Year(),
		GeneratedAt:    time.Now().Format(time.RFC3339),
		CoverageTarget: fcs.TestingStrategy.CoverageTarget,
	}

	return data
}
