// Package main implements the GoCreator CLI application.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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

func init() {
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
	checkpointDir := filepath.Join(clarifyOutput, ".gocreator", "checkpoints")
	engine, err := clarify.NewEngine(clarify.EngineConfig{
		LLMClient:        llmClient,
		CheckpointDir:    checkpointDir,
		EnableCheckpoint: true,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create clarification engine")
		return ExitError{Code: ExitCodeInternalError, Err: fmt.Errorf("failed to create clarification engine: %w", err)}
	}

	// Determine interactive mode
	interactive := clarifyInteractive && clarifyBatch == ""

	// Load batch answers if provided
	if clarifyBatch != "" {
		fmt.Printf("Loading batch answers from: %s\n", clarifyBatch)
		interactive = false
		// TODO: Implement batch answer loading
		log.Warn().Msg("Batch mode not yet fully implemented")
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
	if err := os.MkdirAll(fcsDir, 0750); err != nil {
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

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write FCS file: %w", err)
	}

	return nil
}

func createLLMClient(_ *config.Config) (llm.Client, error) {
	// TODO: Implement actual LLM client creation based on config
	// For now, return a placeholder
	log.Warn().Msg("LLM client creation not yet fully implemented")
	return nil, fmt.Errorf("LLM client creation not yet implemented")
}
