package providers

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"github.com/BrianGreenhill/coachgpt/internal/config"
)

// TestNewUserSetupWorkflow tests the complete setup workflow for a new user
func TestNewUserSetupWorkflow(t *testing.T) {
	// Setup temporary config directory
	cleanup := setupTestConfigForIntegration(t)
	defer cleanup()

	// Clear all environment variables to simulate a new user
	clearAllEnvVars(t)

	// Create a config to test with
	cfg := &config.Config{}

	// Create a Strava provider and set it up with test data
	stravaProvider := NewStravaProviderForSetup()

	// Simulate user input for Strava setup
	userInput := `88986
9771e2cf9661abcb09e55f0b7f490c88e76e481a
202
`
	reader := bufio.NewReader(strings.NewReader(userInput))

	// Test the individual provider setup
	err := stravaProvider.SetupConfig(reader, cfg)
	if err != nil {
		t.Fatalf("Strava SetupConfig failed: %v", err)
	}

	// Save the config
	err = cfg.Save()
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify config file was created in the expected location
	homeDir := os.Getenv("HOME")
	configPath := homeDir + "/.config/coachgpt/config.json"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("Config file was not created at %s", configPath)
	}

	// Load the config and verify it was saved correctly
	loadedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	// Verify Strava configuration
	if !loadedCfg.HasStrava() {
		t.Error("Strava should be configured after setup")
	}
	if loadedCfg.Strava.ClientID != "88986" {
		t.Errorf("Expected ClientID '88986', got '%s'", loadedCfg.Strava.ClientID)
	}
	if loadedCfg.Strava.ClientSecret != "9771e2cf9661abcb09e55f0b7f490c88e76e481a" {
		t.Errorf("Expected specific client secret, got '%s'", loadedCfg.Strava.ClientSecret)
	}
	if loadedCfg.Strava.HRMax != 202 {
		t.Errorf("Expected HRMax 202, got %d", loadedCfg.Strava.HRMax)
	}

	// Verify Hevy is not configured
	if loadedCfg.HasHevy() {
		t.Error("Hevy should not be configured")
	}

	// Verify config file permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}
	expectedMode := os.FileMode(0600)
	if info.Mode().Perm() != expectedMode {
		t.Errorf("Expected config file permissions %v, got %v", expectedMode, info.Mode().Perm())
	}
}

// TestExistingUserReconfiguration tests reconfiguring an existing setup
func TestExistingUserReconfiguration(t *testing.T) {
	// Setup temporary config directory
	cleanup := setupTestConfigForIntegration(t)
	defer cleanup()

	// Create initial config
	initialCfg := &config.Config{
		Strava: config.StravaConfig{
			ClientID:     "old_id",
			ClientSecret: "old_secret",
			HRMax:        180,
		},
		Hevy: config.HevyConfig{
			APIKey: "old_api_key",
		},
	}
	if err := initialCfg.Save(); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Create a Strava provider and test reconfiguration
	stravaProvider := NewStravaProviderForSetup()

	// Simulate user input for reconfiguring Strava (yes to reconfigure)
	userInput := `y
new_client_id
new_client_secret
190
`
	reader := bufio.NewReader(strings.NewReader(userInput))

	// Load the existing config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load initial config: %v", err)
	}

	// Test reconfiguration
	err = stravaProvider.SetupConfig(reader, cfg)
	if err != nil {
		t.Fatalf("Strava SetupConfig failed: %v", err)
	}

	// Save the updated config
	err = cfg.Save()
	if err != nil {
		t.Fatalf("Failed to save updated config: %v", err)
	}

	// Load the updated config to verify
	updatedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load updated config: %v", err)
	}

	// Verify Strava was updated
	if updatedCfg.Strava.ClientID != "new_client_id" {
		t.Errorf("Expected updated ClientID 'new_client_id', got '%s'", updatedCfg.Strava.ClientID)
	}
	if updatedCfg.Strava.ClientSecret != "new_client_secret" {
		t.Errorf("Expected updated ClientSecret 'new_client_secret', got '%s'", updatedCfg.Strava.ClientSecret)
	}
	if updatedCfg.Strava.HRMax != 190 {
		t.Errorf("Expected updated HRMax 190, got %d", updatedCfg.Strava.HRMax)
	}

	// Verify Hevy config was preserved
	if updatedCfg.Hevy.APIKey != "old_api_key" {
		t.Errorf("Expected Hevy API key to be preserved as 'old_api_key', got '%s'", updatedCfg.Hevy.APIKey)
	}
}

// TestHevySetupWorkflow tests the Hevy provider setup workflow
func TestHevySetupWorkflow(t *testing.T) {
	// Setup temporary config directory
	cleanup := setupTestConfigForIntegration(t)
	defer cleanup()

	// Clear all environment variables
	clearAllEnvVars(t)

	// Create a config to test with
	cfg := &config.Config{}

	// Create a Hevy provider and set it up with test data
	hevyProvider := NewHevyProviderForSetup()

	// Simulate user input for Hevy setup
	userInput := `84e82e4a-a709-457c-9777-1d39d1fdb23e
`
	reader := bufio.NewReader(strings.NewReader(userInput))

	// Test the Hevy provider setup
	err := hevyProvider.SetupConfig(reader, cfg)
	if err != nil {
		t.Fatalf("Hevy SetupConfig failed: %v", err)
	}

	// Save the config
	err = cfg.Save()
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load and verify the config
	loadedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	// Verify Hevy configuration
	if !loadedCfg.HasHevy() {
		t.Error("Hevy should be configured after setup")
	}
	if loadedCfg.Hevy.APIKey != "84e82e4a-a709-457c-9777-1d39d1fdb23e" {
		t.Errorf("Expected specific API key, got '%s'", loadedCfg.Hevy.APIKey)
	}

	// Verify Strava is not configured
	if loadedCfg.HasStrava() {
		t.Error("Strava should not be configured")
	}
}

// TestEnvironmentVariableOverrides tests that env vars still override config file
func TestEnvironmentVariableOverrides(t *testing.T) {
	// Setup temporary config directory
	cleanup := setupTestConfigForIntegration(t)
	defer cleanup()

	// Create config file
	cfg := &config.Config{
		Strava: config.StravaConfig{
			ClientID:     "file_id",
			ClientSecret: "file_secret",
			HRMax:        180,
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Set environment variable overrides
	_ = os.Setenv("STRAVA_CLIENT_ID", "env_id")
	_ = os.Setenv("STRAVA_HRMAX", "200")
	defer func() {
		_ = os.Unsetenv("STRAVA_CLIENT_ID")
		_ = os.Unsetenv("STRAVA_HRMAX")
	}()

	// Load config
	loadedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify environment variables override config file
	if loadedCfg.Strava.ClientID != "env_id" {
		t.Errorf("Expected env var to override: 'env_id', got '%s'", loadedCfg.Strava.ClientID)
	}
	if loadedCfg.Strava.HRMax != 200 {
		t.Errorf("Expected env var to override: 200, got %d", loadedCfg.Strava.HRMax)
	}
	// ClientSecret should come from file since no env override
	if loadedCfg.Strava.ClientSecret != "file_secret" {
		t.Errorf("Expected file value for ClientSecret: 'file_secret', got '%s'", loadedCfg.Strava.ClientSecret)
	}
}

// setupTestConfigForIntegration creates a temporary config directory for integration testing
func setupTestConfigForIntegration(t *testing.T) func() {
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

// clearAllEnvVars clears all CoachGPT-related environment variables
func clearAllEnvVars(t *testing.T) {
	envVars := []string{
		"STRAVA_CLIENT_ID",
		"STRAVA_CLIENT_SECRET",
		"STRAVA_HRMAX",
		"STRAVA_ACTIVITY_ID",
		"HEVY_API_KEY",
	}

	for _, envVar := range envVars {
		_ = os.Unsetenv(envVar)
	}
}
