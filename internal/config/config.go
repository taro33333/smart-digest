// Package config provides configuration management for smart-digest.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LLMProvider represents the type of LLM backend.
type LLMProvider string

const (
	ProviderOpenAI LLMProvider = "openai"
	ProviderOllama LLMProvider = "ollama"
)

// Config holds all configuration for smart-digest.
type Config struct {
	LLMProvider LLMProvider `yaml:"llm_provider"`
	APIKey      string      `yaml:"api_key"`
	Model       string      `yaml:"model"`
	Interests   []string    `yaml:"interests"`
	Threshold   int         `yaml:"threshold"`
	OllamaURL   string      `yaml:"ollama_url"`
	MaxWorkers  int         `yaml:"max_workers"`
	RateLimit   float64     `yaml:"rate_limit_per_second"`
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		LLMProvider: ProviderOpenAI,
		Model:       "gpt-4o-mini",
		Interests:   []string{"Go", "Rust", "Productivity", "System Design"},
		Threshold:   70,
		OllamaURL:   "http://localhost:11434",
		MaxWorkers:  5,
		RateLimit:   10.0,
	}
}

// Load reads configuration from file, checking multiple locations.
// Priority: ./config.yaml > ~/.config/smart-digest/config.yaml > defaults
func Load(customPath string) (*Config, error) {
	cfg := DefaultConfig()

	// Determine config path
	configPath := customPath
	if configPath == "" {
		configPath = findConfigFile()
	}

	if configPath == "" {
		// No config file found, use defaults
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Override API key from environment if set
	if envKey := os.Getenv("OPENAI_API_KEY"); envKey != "" && cfg.APIKey == "" {
		cfg.APIKey = envKey
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// findConfigFile searches for config.yaml in standard locations.
func findConfigFile() string {
	// Check current directory first
	if _, err := os.Stat("config.yaml"); err == nil {
		return "config.yaml"
	}

	// Check XDG config directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	xdgPath := filepath.Join(homeDir, ".config", "smart-digest", "config.yaml")
	if _, err := os.Stat(xdgPath); err == nil {
		return xdgPath
	}

	return ""
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.LLMProvider != ProviderOpenAI && c.LLMProvider != ProviderOllama {
		return fmt.Errorf("invalid llm_provider: %s (must be 'openai' or 'ollama')", c.LLMProvider)
	}

	if c.LLMProvider == ProviderOpenAI && c.APIKey == "" {
		return fmt.Errorf("api_key is required for OpenAI provider")
	}

	if c.Model == "" {
		return fmt.Errorf("model must be specified")
	}

	if len(c.Interests) == 0 {
		return fmt.Errorf("at least one interest must be specified")
	}

	if c.Threshold < 0 || c.Threshold > 100 {
		return fmt.Errorf("threshold must be between 0 and 100")
	}

	if c.MaxWorkers < 1 {
		c.MaxWorkers = 5
	}

	if c.RateLimit <= 0 {
		c.RateLimit = 10.0
	}

	return nil
}

// InterestsString returns a comma-separated string of interests.
func (c *Config) InterestsString() string {
	result := ""
	for i, interest := range c.Interests {
		if i > 0 {
			result += ", "
		}
		result += interest
	}
	return result
}
