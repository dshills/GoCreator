// Package config provides configuration management for GoCreator.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	LLM        LLMConfig        `mapstructure:"llm"`
	Workflow   WorkflowConfig   `mapstructure:"workflow"`
	Validation ValidationConfig `mapstructure:"validation"`
	Logging    LoggingConfig    `mapstructure:"logging"`
}

// LLMConfig configures the LLM provider
type LLMConfig struct {
	Provider    string        `mapstructure:"provider"`
	Model       string        `mapstructure:"model"`
	Temperature float64       `mapstructure:"temperature"`
	APIKey      string        `mapstructure:"api_key"`
	Timeout     time.Duration `mapstructure:"timeout"`
	MaxTokens   int           `mapstructure:"max_tokens"`
}

// WorkflowConfig configures workflow execution
type WorkflowConfig struct {
	RootDir            string   `mapstructure:"root_dir"`
	AllowCommands      []string `mapstructure:"allow_commands"`
	MaxParallel        int      `mapstructure:"max_parallel"`
	CheckpointInterval int      `mapstructure:"checkpoint_interval"`
}

// ValidationConfig configures validation behavior
type ValidationConfig struct {
	EnableLinting    bool          `mapstructure:"enable_linting"`
	LinterConfig     string        `mapstructure:"linter_config"`
	EnableTests      bool          `mapstructure:"enable_tests"`
	TestTimeout      time.Duration `mapstructure:"test_timeout"`
	RequiredCoverage float64       `mapstructure:"required_coverage"`
}

// LoggingConfig configures logging behavior
type LoggingConfig struct {
	Level        string `mapstructure:"level"`
	Format       string `mapstructure:"format"`
	Output       string `mapstructure:"output"`
	ExecutionLog string `mapstructure:"execution_log"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Set config file if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Look for config in standard locations
		v.SetConfigName(".gocreator")
		v.SetConfigType("yaml")

		// Current directory
		v.AddConfigPath(".")

		// Home directory
		if home, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(filepath.Join(home, ".config", "gocreator"))
			v.AddConfigPath(home)
		}
	}

	// Read config file (optional)
	if err := v.ReadInConfig(); err != nil {
		var notFoundErr viper.ConfigFileNotFoundError
		if !errors.As(err, &notFoundErr) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is okay, we'll use defaults
	}

	// Override with environment variables
	v.SetEnvPrefix("GOCREATOR")
	v.AutomaticEnv()

	// Unmarshal into config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// LLM defaults
	v.SetDefault("llm.provider", "anthropic")
	v.SetDefault("llm.model", "claude-sonnet-4")
	v.SetDefault("llm.temperature", 0.0)
	v.SetDefault("llm.timeout", 60*time.Second)
	v.SetDefault("llm.max_tokens", 4096)

	// Workflow defaults
	v.SetDefault("workflow.root_dir", "./generated")
	v.SetDefault("workflow.allow_commands", []string{"go", "git", "golangci-lint"})
	v.SetDefault("workflow.max_parallel", 4)
	v.SetDefault("workflow.checkpoint_interval", 10)

	// Validation defaults
	v.SetDefault("validation.enable_linting", true)
	v.SetDefault("validation.linter_config", ".golangci.yml")
	v.SetDefault("validation.enable_tests", true)
	v.SetDefault("validation.test_timeout", 5*time.Minute)
	v.SetDefault("validation.required_coverage", 80.0)

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "console")
	v.SetDefault("logging.output", "stderr")
	v.SetDefault("logging.execution_log", ".gocreator/execution.jsonl")
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate LLM config
	if c.LLM.Provider == "" {
		return fmt.Errorf("llm.provider is required")
	}
	if c.LLM.Model == "" {
		return fmt.Errorf("llm.model is required")
	}
	if c.LLM.Temperature < 0 || c.LLM.Temperature > 2.0 {
		return fmt.Errorf("llm.temperature must be between 0 and 2.0")
	}
	if c.LLM.MaxTokens <= 0 {
		return fmt.Errorf("llm.max_tokens must be positive")
	}

	// Validate workflow config
	if c.Workflow.MaxParallel <= 0 {
		return fmt.Errorf("workflow.max_parallel must be positive")
	}
	if c.Workflow.CheckpointInterval <= 0 {
		return fmt.Errorf("workflow.checkpoint_interval must be positive")
	}

	// Validate validation config
	if c.Validation.RequiredCoverage < 0 || c.Validation.RequiredCoverage > 100 {
		return fmt.Errorf("validation.required_coverage must be between 0 and 100")
	}

	// Validate logging config
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("logging.level must be one of: debug, info, warn, error")
	}
	validFormats := map[string]bool{"console": true, "json": true}
	if !validFormats[c.Logging.Format] {
		return fmt.Errorf("logging.format must be one of: console, json")
	}

	return nil
}

// GetLogLevel returns the zerolog level based on config
func (c *Config) GetLogLevel() zerolog.Level {
	switch c.Logging.Level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// IsJSONFormat returns true if logging format is JSON
func (c *Config) IsJSONFormat() bool {
	return c.Logging.Format == "json"
}
