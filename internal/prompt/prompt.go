// Package prompt handles coaching prompt generation and management
package prompt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BrianGreenhill/coachgpt/internal/config"
)

// Generator handles coaching prompt generation
type Generator struct {
	config *config.Config
}

// NewGenerator creates a new prompt generator
func NewGenerator(cfg *config.Config) *Generator {
	return &Generator{config: cfg}
}

// Generate returns the coaching prompt content (custom or default)
func (g *Generator) Generate() (string, error) {
	if g.config == nil {
		return GetDefault(), nil
	}

	return g.config.GetCoachingPrompt()
}

// GenerateWithFallback returns the coaching prompt with error handling fallback
func (g *Generator) GenerateWithFallback() string {
	prompt, err := g.Generate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading coaching prompt: %v\n", err)
		fmt.Fprintf(os.Stderr, "Using default prompt instead.\n\n")
		return GetDefault()
	}
	return prompt
}

// LoadConfigForPromptOnly loads config specifically for prompt functionality
// without requiring full provider validation
func LoadConfigForPromptOnly() (*config.Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return &config.Config{}, nil // Return empty config if path fails
	}

	// If config file doesn't exist, return empty config with env vars
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := &config.Config{}
		applyPromptEnvVars(cfg)
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return &config.Config{}, nil // Return empty config if read fails
	}

	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &config.Config{}, nil // Return empty config if parse fails
	}

	applyPromptEnvVars(&cfg)
	return &cfg, nil
}

// applyPromptEnvVars applies prompt-related environment variable overrides
func applyPromptEnvVars(cfg *config.Config) {
	if promptPath := os.Getenv("COACHING_PROMPT_PATH"); promptPath != "" {
		cfg.Prompt.CustomPath = promptPath
	}
}

// getConfigPath gets config path for prompt functionality
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "coachgpt", "config.json"), nil
}
