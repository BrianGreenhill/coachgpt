package main

import (
	"briangreenhill/coachgpt/hevy"
	"briangreenhill/coachgpt/strava"
	"briangreenhill/coachgpt/workout"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gregjones/httpcache"
)

func main() {
	if err := runCLI(os.Args[1:]); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func runCLI(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "help", "--help", "-h":
			fmt.Println("Usage: coachgpt [options]")
			fmt.Println("Options:")
			fmt.Println("  --help, -h          Show this help message")
			fmt.Println("  --strength, -s      Use Hevy for strength training data")
			fmt.Println("  STRAVA_CLIENT_ID    Your Strava client ID (required)")
			fmt.Println("  STRAVA_CLIENT_SECRET Your Strava client secret (required)")
			fmt.Println("  STRAVA_HRMAX        Your maximum heart rate (required, e.g. 185)")
			fmt.Println("  STRAVA_ACTIVITY_ID  Specific activity ID to fetch (optional)")
			fmt.Println("  HEVY_API_KEY        Your Hevy API key (required for strength)")
		case "version", "--version", "-v":
			fmt.Println("CoachGPT v0.1.0")
		case "strength", "--strength", "-s":
			return runWithProvider("hevy", "")
		default:
			return fmt.Errorf("unknown command: %s", args[0])
		}
	} else {
		activityID := os.Getenv("STRAVA_ACTIVITY_ID")
		return runWithProvider("strava", activityID)
	}

	return nil
}

// setupWorkoutRegistry creates and configures all available workout providers
func setupWorkoutRegistry() *workout.Registry {
	registry := workout.NewRegistry()

	// Create HTTP client with caching transport
	transport := httpcache.NewMemoryCacheTransport()
	httpClient := &http.Client{Transport: transport}

	// Register Strava provider
	if clientID := os.Getenv("STRAVA_CLIENT_ID"); clientID != "" {
		if clientSecret := os.Getenv("STRAVA_CLIENT_SECRET"); clientSecret != "" {
			if hrmaxStr := os.Getenv("STRAVA_HRMAX"); hrmaxStr != "" {
				if hrmax, err := strconv.Atoi(hrmaxStr); err == nil && hrmax >= 120 {
					stravaClient := strava.NewClientWithHTTP(clientID, clientSecret, httpClient)
					stravaProvider := strava.NewProvider(stravaClient, hrmax)
					registry.Register(stravaProvider)
				}
			}
		}
	}

	// Register Hevy provider
	if apiKey := os.Getenv("HEVY_API_KEY"); apiKey != "" {
		if client, err := hevy.New(apiKey, hevy.WithHTTPClient(httpClient)); err == nil {
			hevyProvider := hevy.NewProvider(client)
			registry.Register(hevyProvider)
		}
	}

	return registry
}

// runWithProvider executes the specified provider to fetch and display workout data
func runWithProvider(providerName, workoutID string) error {
	registry := setupWorkoutRegistry()

	provider, exists := registry.Get(providerName)
	if !exists {
		availableProviders := registry.List()
		if len(availableProviders) == 0 {
			return fmt.Errorf("no providers are configured. Please set the required environment variables")
		}
		return fmt.Errorf("provider '%s' not found. Available providers: %v", providerName, availableProviders)
	}

	ctx := context.Background()
	var output string
	var err error

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
