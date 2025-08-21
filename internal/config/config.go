// Package config handles application configuration from environment variables
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	Strava StravaConfig
	Hevy   HevyConfig
}

// StravaConfig holds Strava-specific configuration
type StravaConfig struct {
	ClientID     string
	ClientSecret string
	HRMax        int
	ActivityID   string // Optional specific activity ID
}

// HevyConfig holds Hevy-specific configuration
type HevyConfig struct {
	APIKey string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Load Strava config
	cfg.Strava.ClientID = os.Getenv("STRAVA_CLIENT_ID")
	cfg.Strava.ClientSecret = os.Getenv("STRAVA_CLIENT_SECRET")
	cfg.Strava.ActivityID = os.Getenv("STRAVA_ACTIVITY_ID")

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

	// Load Hevy config
	cfg.Hevy.APIKey = os.Getenv("HEVY_API_KEY")

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
		return fmt.Errorf("no providers configured - please set environment variables for at least one provider")
	}
	return nil
}
