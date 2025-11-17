package main

import (
	"fmt"
	"os"
	"time"

	"github.com/dshills/gocreator/internal/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	// Version information (set by build flags)
	version   = "0.1.0-dev"
	commit    = "unknown"
	buildDate = "unknown"

	// Global flags
	cfgFile   string
	logLevel  string
	logFormat string

	// Global config
	cfg *config.Config
)

func main() {
	setupCommands()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "gocreator",
	Short: "GoCreator - Autonomous Go Code Generation",
	Long: `GoCreator is an autonomous code generation system that transforms specifications
into complete, functioning Go codebases.

It follows a three-phase workflow:
  1. Clarification: Analyzes specifications and resolves ambiguities
  2. Generation: Autonomously generates complete project structures
  3. Validation: Validates generated code (build, lint, test)`,
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		// Skip for version command
		if cmd.Name() == "version" {
			return nil
		}

		// Initialize logging first
		if err := initLogging(); err != nil {
			return fmt.Errorf("failed to initialize logging: %w", err)
		}

		// Load configuration
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			log.Error().Err(err).Msg("Failed to load configuration")
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Override log level from config if not set via flag
		if cmd.Flags().Changed("log-level") {
			// Use flag value
			level, err := parseLogLevel(logLevel)
			if err != nil {
				return err
			}
			zerolog.SetGlobalLevel(level)
		} else {
			// Use config value
			zerolog.SetGlobalLevel(cfg.GetLogLevel())
		}

		log.Debug().
			Str("version", version).
			Str("config_file", cfgFile).
			Msg("GoCreator initialized")

		return nil
	},
}

func setupCommands() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: .gocreator.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "console", "log format (console, json)")

	// Setup command-specific flags
	setupClarifyFlags()
	setupGenerateFlags()
	setupValidateFlags()
	setupFullFlags()
	setupDumpFCSFlags()

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(clarifyCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(fullCmd)
	rootCmd.AddCommand(dumpFCSCmd)

	// Set version template
	rootCmd.SetVersionTemplate(fmt.Sprintf("GoCreator v%s\n", version))
}

func initLogging() error {
	// Configure output
	if logFormat == "console" {
		output := zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		}
		log.Logger = zerolog.New(output).With().Timestamp().Logger()
	} else {
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	}

	// Set log level
	level, err := parseLogLevel(logLevel)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(level)

	return nil
}

func parseLogLevel(level string) (zerolog.Level, error) {
	switch level {
	case "debug":
		return zerolog.DebugLevel, nil
	case "info":
		return zerolog.InfoLevel, nil
	case "warn":
		return zerolog.WarnLevel, nil
	case "error":
		return zerolog.ErrorLevel, nil
	default:
		return zerolog.InfoLevel, fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", level)
	}
}
