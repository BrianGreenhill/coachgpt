package config

import (
	"os"
	"testing"
)

// setupTestConfig creates a temporary config directory for testing
func setupTestConfig(t *testing.T) func() {
	// Create a temporary directory for config
	tempDir := t.TempDir()

	// Save original HOME
	originalHome := os.Getenv("HOME")

	// Set HOME to our temp directory
	_ = os.Setenv("HOME", tempDir)

	// Return cleanup function
	return func() {
		if originalHome == "" {
			_ = os.Unsetenv("HOME")
		} else {
			_ = os.Setenv("HOME", originalHome)
		}
	}
}

func TestLoad(t *testing.T) {
	// Setup temporary config directory
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Save original env vars
	original := map[string]string{
		"STRAVA_CLIENT_ID":     os.Getenv("STRAVA_CLIENT_ID"),
		"STRAVA_CLIENT_SECRET": os.Getenv("STRAVA_CLIENT_SECRET"),
		"STRAVA_HRMAX":         os.Getenv("STRAVA_HRMAX"),
		"STRAVA_ACTIVITY_ID":   os.Getenv("STRAVA_ACTIVITY_ID"),
		"HEVY_API_KEY":         os.Getenv("HEVY_API_KEY"),
	}

	// Clean up after test
	defer func() {
		for key, value := range original {
			if value == "" {
				_ = os.Unsetenv(key)
			} else {
				_ = os.Setenv(key, value)
			}
		}
	}()

	// Test with valid Strava config
	_ = os.Setenv("STRAVA_CLIENT_ID", "test_id")
	_ = os.Setenv("STRAVA_CLIENT_SECRET", "test_secret")
	_ = os.Setenv("STRAVA_HRMAX", "185")
	_ = os.Setenv("STRAVA_ACTIVITY_ID", "123456")
	_ = os.Unsetenv("HEVY_API_KEY")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Strava.ClientID != "test_id" {
		t.Errorf("Expected ClientID 'test_id', got '%s'", cfg.Strava.ClientID)
	}

	if cfg.Strava.ClientSecret != "test_secret" {
		t.Errorf("Expected ClientSecret 'test_secret', got '%s'", cfg.Strava.ClientSecret)
	}

	if cfg.Strava.HRMax != 185 {
		t.Errorf("Expected HRMax 185, got %d", cfg.Strava.HRMax)
	}

	if cfg.Strava.ActivityID != "123456" {
		t.Errorf("Expected ActivityID '123456', got '%s'", cfg.Strava.ActivityID)
	}

	if !cfg.HasStrava() {
		t.Error("Should have Strava configured")
	}

	if cfg.HasHevy() {
		t.Error("Should not have Hevy configured")
	}
}

func TestLoadInvalidHRMax(t *testing.T) {
	// Setup temporary config directory
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Clean up after test
	defer func() {
		_ = os.Unsetenv("STRAVA_HRMAX")
	}()

	// Test with invalid HR Max
	_ = os.Setenv("STRAVA_HRMAX", "invalid")

	_, err := Load()
	if err == nil {
		t.Error("Expected error for invalid STRAVA_HRMAX")
	}

	// Test with HR Max out of range
	_ = os.Setenv("STRAVA_HRMAX", "50")

	_, err = Load()
	if err == nil {
		t.Error("Expected error for STRAVA_HRMAX out of range")
	}
}

func TestValidate(t *testing.T) {
	// Test with no providers
	cfg := &Config{}
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error when no providers configured")
	}

	// Test with Strava only
	cfg.Strava.ClientID = "test"
	cfg.Strava.ClientSecret = "test"
	cfg.Strava.HRMax = 180
	err = cfg.Validate()
	if err != nil {
		t.Errorf("Should not error with Strava configured: %v", err)
	}

	// Test with Hevy only
	cfg = &Config{}
	cfg.Hevy.APIKey = "test"
	err = cfg.Validate()
	if err != nil {
		t.Errorf("Should not error with Hevy configured: %v", err)
	}
}
