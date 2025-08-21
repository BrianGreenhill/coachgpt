// Package config handles application configuration from config file and environment variables
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	Strava StravaConfig `json:"strava"`
	Hevy   HevyConfig   `json:"hevy"`
}

// StravaConfig holds Strava-specific configuration
type StravaConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	HRMax        int    `json:"hr_max"`
	ActivityID   string `json:"activity_id,omitempty"` // Optional specific activity ID
}

// HevyConfig holds Hevy-specific configuration
type HevyConfig struct {
	APIKey string `json:"api_key"`
}

// getConfigPath returns the path to the config file
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %v", err)
	}

	configDir := filepath.Join(homeDir, ".config", "coachgpt")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %v", err)
	}

	return filepath.Join(configDir, "config.json"), nil
}

// loadFromFile loads configuration from the config file
func loadFromFile() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	// If config file doesn't exist, return empty config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &cfg, nil
}

// Save writes the configuration to the config file
func (c *Config) Save() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// Load reads configuration from config file first, then applies environment variable overrides
func Load() (*Config, error) {
	// Load from config file first
	cfg, err := loadFromFile()
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %v", err)
	}

	// Apply environment variable overrides
	if clientID := os.Getenv("STRAVA_CLIENT_ID"); clientID != "" {
		cfg.Strava.ClientID = clientID
	}
	if clientSecret := os.Getenv("STRAVA_CLIENT_SECRET"); clientSecret != "" {
		cfg.Strava.ClientSecret = clientSecret
	}
	if activityID := os.Getenv("STRAVA_ACTIVITY_ID"); activityID != "" {
		cfg.Strava.ActivityID = activityID
	}

	if hrmaxStr := os.Getenv("STRAVA_HRMAX"); hrmaxStr != "" {
		hrmax, err := strconv.Atoi(hrmaxStr)
		if err != nil {
			return nil, fmt.Errorf("invalid STRAVA_HRMAX: %v", err)
		}
		if hrmax < 120 || hrmax > 220 {
			return nil, fmt.Errorf("STRAVA_HRMAX must be between 120-220, got %d", hrmax)
		}
		cfg.Strava.HRMax = hrmax
	}

	if apiKey := os.Getenv("HEVY_API_KEY"); apiKey != "" {
		cfg.Hevy.APIKey = apiKey
	}

	return cfg, nil
}

// HasStrava returns true if Strava configuration is complete
func (c *Config) HasStrava() bool {
	return c.Strava.ClientID != "" && c.Strava.ClientSecret != "" && c.Strava.HRMax > 0
}

// HasHevy returns true if Hevy configuration is complete
func (c *Config) HasHevy() bool {
	return c.Hevy.APIKey != ""
}

// Validate ensures the configuration has at least one provider configured
func (c *Config) Validate() error {
	if !c.HasStrava() && !c.HasHevy() {
		configPath, _ := getConfigPath()
		return fmt.Errorf(`no providers configured

To get started, run the setup wizard to create a config file:
  coachgpt config

This will create a config file at: %s

You can also override settings with environment variables:
  
For Strava:
  export STRAVA_CLIENT_ID="your_client_id"
  export STRAVA_CLIENT_SECRET="your_client_secret"  
  export STRAVA_HRMAX="185"

For Hevy:
  export HEVY_API_KEY="your_api_key"`, configPath)
	}
	return nil
}
