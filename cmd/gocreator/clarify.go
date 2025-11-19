// Package main implements the GoCreator CLI application.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dshills/gocreator/internal/clarify"
	"github.com/dshills/gocreator/internal/config"
	"github.com/dshills/gocreator/internal/models"
	"github.com/dshills/gocreator/internal/spec"
	"github.com/dshills/gocreator/pkg/llm"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	clarifyOutput      string
	clarifyInteractive bool
	clarifyBatch       string
)

var clarifyCmd = &cobra.Command{
	Use:   "clarify <spec-file>",
	Short: "Analyze specification and run clarification phase",
	Long: `Analyze a specification file to identify ambiguities and run the clarification phase.

The clarification phase:
  1. Parses and validates the input specification
  2. Identifies ambiguities, missing constraints, and unclear requirements
  3. Generates targeted questions for resolution
  4. Produces a Final Clarified Specification (FCS)

Interactive mode (default):
  Prompts for answers to clarification questions interactively.

Batch mode (--batch):
  Uses pre-answered questions from a JSON file.

Example:
  # Interactive mode
  gocreator clarify ./my-project-spec.yaml

  # Batch mode
  gocreator clarify ./my-project-spec.yaml --batch ./answers.json

  # Specify output directory
  gocreator clarify ./my-project-spec.yaml --output ./output`,
	Args: cobra.ExactArgs(1),
	RunE: runClarify,
}

func setupClarifyFlags() {
	clarifyCmd.Flags().StringVarP(&clarifyOutput, "output", "o", ".", "output directory for FCS")
	clarifyCmd.Flags().BoolVarP(&clarifyInteractive, "interactive", "i", true, "interactive mode for answering questions")
	clarifyCmd.Flags().StringVar(&clarifyBatch, "batch", "", "path to JSON file with pre-answered questions")
}

func runClarify(_ *cobra.Command, args []string) error {
	specFile := args[0]

	log.Info().
		Str("spec_file", specFile).
		Str("output", clarifyOutput).
		Bool("interactive", clarifyInteractive).
		Msg("Starting clarification phase")

	fmt.Printf("GoCreator v%s - Clarification Phase\n\n", version)
	fmt.Printf("Analyzing specification: %s\n\n", specFile)

	// Detect format from file extension
	format, err := detectSpecFormat(specFile)
	if err != nil {
		log.Error().Err(err).Msg("Failed to detect spec format")
		return ExitError{Code: ExitCodeSpecError, Err: err}
	}

	// Read spec file
	//nolint:gosec // G304: Reading user-provided spec file - required for CLI functionality
	content, err := os.ReadFile(specFile)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read spec file")
		return ExitError{Code: ExitCodeFileSystemError, Err: fmt.Errorf("failed to read spec file: %w", err)}
	}

	// Parse and validate specification
	inputSpec, err := spec.ParseAndValidate(format, string(content))
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse specification")
		return ExitError{Code: ExitCodeSpecError, Err: fmt.Errorf("specification validation failed: %w", err)}
	}

	log.Info().
		Str("spec_id", inputSpec.ID).
		Str("format", string(inputSpec.Format)).
		Msg("Specification parsed and validated")

	// Create LLM client
	llmClient, err := createLLMClient(cfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create LLM client")
		return ExitError{Code: ExitCodeNetworkError, Err: fmt.Errorf("failed to create LLM client: %w", err)}
	}

	// Create clarification engine
	engine, err := clarify.NewEngine(clarify.EngineConfig{
		LLMClient: llmClient,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create clarification engine")
		return ExitError{Code: ExitCodeInternalError, Err: fmt.Errorf("failed to create clarification engine: %w", err)}
	}

	// Determine interactive mode
	interactive := clarifyInteractive && clarifyBatch == ""

	// Load batch answers if provided
	var batchAnswers map[string]string
	if clarifyBatch != "" {
		fmt.Printf("Loading batch answers from: %s\n", clarifyBatch)
		interactive = false

		// Read batch answers file
		//nolint:gosec // G304: Reading user-provided batch file - required for CLI functionality
		batchData, err := os.ReadFile(clarifyBatch)
		if err != nil {
			log.Error().Err(err).Msg("Failed to read batch answers file")
			return ExitError{Code: ExitCodeFileSystemError, Err: fmt.Errorf("failed to read batch answers file: %w", err)}
		}

		// Parse JSON answers
		if err := json.Unmarshal(batchData, &batchAnswers); err != nil {
			log.Error().Err(err).Msg("Failed to parse batch answers JSON")
			return ExitError{Code: ExitCodeSpecError, Err: fmt.Errorf("failed to parse batch answers: %w", err)}
		}

		log.Info().
			Int("answers_count", len(batchAnswers)).
			Msg("Loaded batch answers")

		// Note: The current clarification engine doesn't support batch answers yet.
		// The engine runs in autonomous mode and doesn't prompt for user input.
		// In the future, batch answers will be used to pre-populate responses.
		log.Warn().Msg("Batch answers loaded but not yet integrated with clarification engine (runs autonomously)")
	}

	// Run clarification
	ctx := context.Background()
	fcs, err := engine.Clarify(ctx, inputSpec, interactive)
	if err != nil {
		log.Error().Err(err).Msg("Clarification failed")
		return ExitError{Code: ExitCodeClarificationError, Err: fmt.Errorf("clarification failed: %w", err)}
	}

	// Ensure output directory exists
	fcsDir := filepath.Join(clarifyOutput, ".gocreator")
	if err := os.MkdirAll(fcsDir, 0o750); err != nil {
		log.Error().Err(err).Msg("Failed to create output directory")
		return ExitError{Code: ExitCodeFileSystemError, Err: fmt.Errorf("failed to create output directory: %w", err)}
	}

	// Write FCS to file
	fcsPath := filepath.Join(fcsDir, "fcs.json")
	if err := writeFCS(fcs, fcsPath); err != nil {
		log.Error().Err(err).Msg("Failed to write FCS")
		return ExitError{Code: ExitCodeFileSystemError, Err: fmt.Errorf("failed to write FCS: %w", err)}
	}

	fmt.Printf("\nFinal Clarified Specification written to: %s\n", fcsPath)

	log.Info().
		Str("fcs_id", fcs.ID).
		Str("fcs_path", fcsPath).
		Int("clarifications", len(fcs.Metadata.Clarifications)).
		Msg("Clarification phase completed successfully")

	return nil
}

func detectSpecFormat(filename string) (models.SpecFormat, error) {
	ext := filepath.Ext(filename)
	switch ext {
	case ".yaml", ".yml":
		return models.FormatYAML, nil
	case ".json":
		return models.FormatJSON, nil
	case ".md", ".markdown":
		return models.FormatMarkdown, nil
	default:
		return "", fmt.Errorf("unsupported file extension: %s (must be .yaml, .json, or .md)", ext)
	}
}

func writeFCS(fcs *models.FinalClarifiedSpecification, path string) error {
	data, err := json.MarshalIndent(fcs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal FCS: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write FCS file: %w", err)
	}

	return nil
}

func createLLMClient(cfg *config.Config) (llm.Client, error) {
	// Validate config
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Get API key from config or environment variable
	apiKey := cfg.LLM.APIKey
	if apiKey == "" {
		// Fall back to environment variable based on provider
		switch cfg.LLM.Provider {
		case "anthropic":
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		case "openai":
			apiKey = os.Getenv("OPENAI_API_KEY")
		case "google":
			apiKey = os.Getenv("GOOGLE_API_KEY")
		}
	}

	if apiKey == "" {
		return nil, fmt.Errorf("API key not found in config or environment variable for provider: %s", cfg.LLM.Provider)
	}

	// Create LLM client configuration
	llmConfig := llm.Config{
		Provider:      llm.Provider(cfg.LLM.Provider),
		Model:         cfg.LLM.Model,
		Temperature:   0.0, // Force 0.0 for deterministic output (required by spec)
		APIKey:        apiKey,
		Timeout:       cfg.LLM.Timeout,
		MaxTokens:     cfg.LLM.MaxTokens,
		MaxRetries:    3,
		RetryDelay:    time.Second * 2,
		EnableCaching: true, // Enable prompt caching for cost savings
		CacheTTL:      "5m",
	}

	// Create and return LLM client
	client, err := llm.NewClient(llmConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LLM client: %w", err)
	}

	log.Info().
		Str("provider", cfg.LLM.Provider).
		Str("model", cfg.LLM.Model).
		Msg("LLM client created successfully")

	return client, nil
}
