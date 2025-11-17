package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dshills/gocreator/internal/clarify"
	"github.com/dshills/gocreator/internal/spec"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	dumpFCSOutput string
	dumpFCSBatch  string
	dumpFCSPretty bool
)

var dumpFCSCmd = &cobra.Command{
	Use:   "dump-fcs <spec-file>",
	Short: "Output Final Clarified Specification as JSON",
	Long: `Generate and output the Final Clarified Specification (FCS) as JSON.

This command runs the clarification phase and outputs the resulting FCS
without generating any code. Useful for:
  - Inspecting the clarified specification
  - Debugging clarification issues
  - Validating specification interpretation
  - Extracting machine-readable requirements

The FCS contains:
  - Fully resolved requirements
  - Clarification decisions
  - Architectural constraints
  - Implementation details

Output:
  By default, outputs to stdout (can be redirected)
  Use --output to write to a file instead

Example:
  # Output to stdout
  gocreator dump-fcs ./my-project-spec.yaml

  # Save to file
  gocreator dump-fcs ./my-project-spec.yaml --output ./fcs.json

  # Compact JSON
  gocreator dump-fcs ./my-project-spec.yaml --pretty=false

  # Batch mode
  gocreator dump-fcs ./my-project-spec.yaml --batch ./answers.json`,
	Args: cobra.ExactArgs(1),
	RunE: runDumpFCS,
}

func setupDumpFCSFlags() {
	dumpFCSCmd.Flags().StringVarP(&dumpFCSOutput, "output", "o", "", "output file path (default: stdout)")
	dumpFCSCmd.Flags().StringVar(&dumpFCSBatch, "batch", "", "path to JSON file with pre-answered questions")
	dumpFCSCmd.Flags().BoolVar(&dumpFCSPretty, "pretty", true, "pretty-print JSON")
}

func runDumpFCS(_ *cobra.Command, args []string) error {
	specFile := args[0]

	log.Info().
		Str("spec_file", specFile).
		Str("output", dumpFCSOutput).
		Bool("pretty", dumpFCSPretty).
		Msg("Dumping FCS")

	// Detect format
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

	// Parse and validate
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
	// Use temp directory for checkpoints since we're just dumping
	tempDir := os.TempDir()
	checkpointDir := filepath.Join(tempDir, ".gocreator-dump", "checkpoints")
	engine, err := clarify.NewEngine(clarify.EngineConfig{
		LLMClient:        llmClient,
		CheckpointDir:    checkpointDir,
		EnableCheckpoint: false, // No need for checkpoints when dumping
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create clarification engine")
		return ExitError{Code: ExitCodeInternalError, Err: fmt.Errorf("failed to create clarification engine: %w", err)}
	}

	// Determine interactive mode
	interactive := dumpFCSBatch == ""

	// Run clarification
	ctx := context.Background()
	fcs, err := engine.Clarify(ctx, inputSpec, interactive)
	if err != nil {
		log.Error().Err(err).Msg("Clarification failed")
		return ExitError{Code: ExitCodeClarificationError, Err: fmt.Errorf("clarification failed: %w", err)}
	}

	// Marshal FCS to JSON
	var jsonData []byte
	if dumpFCSPretty {
		jsonData, err = json.MarshalIndent(fcs, "", "  ")
	} else {
		jsonData, err = json.Marshal(fcs)
	}
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal FCS")
		return ExitError{Code: ExitCodeInternalError, Err: fmt.Errorf("failed to marshal FCS: %w", err)}
	}

	// Output FCS
	if dumpFCSOutput != "" {
		// Write to file
		if err := os.WriteFile(dumpFCSOutput, jsonData, 0600); err != nil {
			log.Error().Err(err).Msg("Failed to write FCS file")
			return ExitError{Code: ExitCodeFileSystemError, Err: fmt.Errorf("failed to write FCS file: %w", err)}
		}
		fmt.Printf("FCS written to: %s\n", dumpFCSOutput)
		log.Info().Str("output", dumpFCSOutput).Msg("FCS dumped to file")
	} else {
		// Write to stdout
		fmt.Println(string(jsonData))
		log.Info().Msg("FCS dumped to stdout")
	}

	return nil
}
