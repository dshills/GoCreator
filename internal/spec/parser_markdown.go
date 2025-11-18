package spec

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/internal/yamlutil"
	"github.com/google/uuid"
)

// MarkdownParser implements the Parser interface for Markdown with YAML frontmatter
type MarkdownParser struct{}

var (
	// frontmatterRegex matches YAML frontmatter delimited by ---
	frontmatterRegex = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n`)
)

// Parse parses Markdown content with YAML frontmatter into an InputSpecification
func (p *MarkdownParser) Parse(content string) (*models.InputSpecification, error) {
	// Validate content is not empty
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("empty content provided")
	}

	// Extract frontmatter
	frontmatter, err := p.extractFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract frontmatter: %w", err)
	}

	// Parse frontmatter as YAML with enhanced error reporting
	var data map[string]interface{}
	if err := yamlutil.Unmarshal([]byte(frontmatter), &data); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Create InputSpecification
	spec := &models.InputSpecification{
		SchemaVersion: "1.0",
		ID:            uuid.New().String(),
		Format:        models.FormatMarkdown,
		Content:       content,
		ParsedData:    data,
		Metadata: models.SpecMetadata{
			CreatedAt: time.Now(),
			Version:   "1.0",
		},
		State: models.SpecStateUnparsed,
	}

	// Transition to parsed state
	if err := spec.TransitionTo(models.SpecStateParsed); err != nil {
		return nil, fmt.Errorf("failed to transition to parsed state: %w", err)
	}

	return spec, nil
}

// extractFrontmatter extracts the YAML frontmatter from markdown content
func (p *MarkdownParser) extractFrontmatter(content string) (string, error) {
	// Check if content has frontmatter
	if !strings.HasPrefix(content, "---") {
		return "", fmt.Errorf("markdown content missing frontmatter (must start with ---)")
	}

	matches := frontmatterRegex.FindStringSubmatch(content)
	if len(matches) < 2 {
		return "", fmt.Errorf("invalid frontmatter format (must be enclosed in --- delimiters)")
	}

	frontmatter := strings.TrimSpace(matches[1])
	if frontmatter == "" {
		return "", fmt.Errorf("frontmatter is empty")
	}

	return frontmatter, nil
}

// ExtractBody extracts the markdown body (content after frontmatter)
func (p *MarkdownParser) ExtractBody(content string) (string, error) {
	if !strings.HasPrefix(content, "---") {
		return "", fmt.Errorf("markdown content missing frontmatter")
	}

	// Find the end of frontmatter
	body := frontmatterRegex.ReplaceAllString(content, "")
	return strings.TrimSpace(body), nil
}

// RenderMarkdown renders the markdown body to HTML (optional, for future use)
func (p *MarkdownParser) RenderMarkdown(content string) (string, error) {
	body, err := p.ExtractBody(content)
	if err != nil {
		return "", err
	}

	// Simple conversion for now - can be enhanced with goldmark later
	var buf bytes.Buffer
	buf.WriteString(body)
	return buf.String(), nil
}

// ParseMarkdownFile is a helper for testing or direct file parsing
func ParseMarkdownFile(content string) (*models.InputSpecification, error) {
	parser := &MarkdownParser{}
	return parser.Parse(content)
}
