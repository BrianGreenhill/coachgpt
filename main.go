package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/BrianGreenhill/coachgpt/internal/config"
	"github.com/BrianGreenhill/coachgpt/internal/providers"
)

// Build-time variables
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run(args []string) error {
	// Handle config command first (before loading config)
	if len(args) > 0 && (args[0] == "config" || args[0] == "setup") {
		// Create a temporary registry with all available providers for setup
		registry := providers.NewRegistry()

		// Register all available providers (these will check their own environment)
		registry.Register(providers.NewStravaProviderForSetup())
		registry.Register(providers.NewHevyProviderForSetup())

		return providers.SetupWizard(registry)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	// Parse command line arguments
	providerName, workoutID := parseArgs(args)

	// Setup providers
	registry := providers.Setup(cfg)

	// Execute the request
	return runWithProvider(registry, cfg, providerName, workoutID)
}

func parseArgs(args []string) (providerName, workoutID string) {
	if len(args) > 0 {
		switch args[0] {
		case "help", "--help", "-h":
			fmt.Println("Usage: coachgpt [command] [options]")
			fmt.Println()
			fmt.Println("Commands:")
			fmt.Println("  config, setup       Interactive setup wizard for API credentials")
			fmt.Println("  strength, -s        Use Hevy for strength training data")
			fmt.Println("  version, -v         Show version information")
			fmt.Println("  help, -h            Show this help message")
			fmt.Println()
			fmt.Println("Environment Variables:")
			fmt.Println("  STRAVA_CLIENT_ID    Your Strava client ID (required)")
			fmt.Println("  STRAVA_CLIENT_SECRET Your Strava client secret (required)")
			fmt.Println("  STRAVA_HRMAX        Your maximum heart rate (required, e.g. 185)")
			fmt.Println("  STRAVA_ACTIVITY_ID  Specific activity ID to fetch (optional)")
			fmt.Println("  HEVY_API_KEY        Your Hevy API key (required for strength)")
			fmt.Println()
			fmt.Println("Examples:")
			fmt.Println("  coachgpt config     # Set up API credentials")
			fmt.Println("  coachgpt            # Get latest Strava activity")
			fmt.Println("  coachgpt -s         # Get latest Hevy workout")
			os.Exit(0)
		case "version", "--version", "-v":
			fmt.Printf("CoachGPT %s\n", version)
			fmt.Printf("Commit: %s\n", commit)
			fmt.Printf("Built: %s\n", date)
			os.Exit(0)
		case "config", "setup":
			// This is handled in run() function before we get here
			return "", ""
		case "strength", "--strength", "-s":
			return "hevy", ""
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n", args[0])
			os.Exit(1)
		}
	}

	// Default to Strava with optional activity ID
	return "strava", ""
}

func runWithProvider(registry *providers.Registry, cfg *config.Config, providerName, workoutID string) error {
	provider, exists := registry.Get(providerName)
	if !exists {
		availableProviders := registry.List()
		if len(availableProviders) == 0 {
			return fmt.Errorf("no providers are configured")
		}
		return fmt.Errorf("provider '%s' not found. Available providers: %v", providerName, availableProviders)
	}

	ctx := context.Background()
	var output string
	var err error

	// Use specific activity ID for Strava if provided
	if providerName == "strava" && workoutID == "" && cfg.Strava.ActivityID != "" {
		workoutID = cfg.Strava.ActivityID
	}

	if workoutID != "" {
		output, err = provider.Get(ctx, workoutID)
	} else {
		output, err = provider.GetLatest(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to get workout from %s: %v", providerName, err)
	}

	fmt.Print(output)
	return nil
}
