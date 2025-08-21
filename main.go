package main

import (
	"briangreenhill/coachgpt/hevy"
	"briangreenhill/coachgpt/plugins"
	"briangreenhill/coachgpt/strava"
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
			return runWithPlugin("hevy", "")
		default:
			return fmt.Errorf("unknown command: %s", args[0])
		}
	} else {
		activityID := os.Getenv("STRAVA_ACTIVITY_ID")
		return runWithPlugin("strava", activityID)
	}

	return nil
}

// setupPluginRegistry creates and configures all available plugins
func setupPluginRegistry() *plugins.Registry {
	registry := plugins.NewRegistry()

	// Create HTTP client with caching transport
	transport := httpcache.NewMemoryCacheTransport()
	httpClient := &http.Client{Transport: transport}

	// Register Strava plugin
	if clientID := os.Getenv("STRAVA_CLIENT_ID"); clientID != "" {
		if clientSecret := os.Getenv("STRAVA_CLIENT_SECRET"); clientSecret != "" {
			if hrmaxStr := os.Getenv("STRAVA_HRMAX"); hrmaxStr != "" {
				if hrmax, err := strconv.Atoi(hrmaxStr); err == nil && hrmax >= 120 {
					stravaClient := strava.NewClientWithHTTP(clientID, clientSecret, httpClient)
					stravaPlugin := strava.NewPlugin(stravaClient, hrmax)
					registry.Register(stravaPlugin)
				}
			}
		}
	}

	// Register Hevy plugin
	if apiKey := os.Getenv("HEVY_API_KEY"); apiKey != "" {
		if client, err := hevy.New(apiKey, hevy.WithHTTPClient(httpClient)); err == nil {
			hevyPlugin := hevy.NewPlugin(client)
			registry.Register(hevyPlugin)
		}
	}

	return registry
}

// runWithPlugin executes the specified plugin to fetch and display workout data
func runWithPlugin(pluginName, workoutID string) error {
	registry := setupPluginRegistry()

	plugin, exists := registry.GetPlugin(pluginName)
	if !exists {
		availablePlugins := registry.List()
		if len(availablePlugins) == 0 {
			return fmt.Errorf("no plugins are configured. Please set the required environment variables")
		}
		return fmt.Errorf("plugin '%s' not found. Available plugins: %v", pluginName, availablePlugins)
	}

	ctx := context.Background()
	var output string
	var err error

	if workoutID != "" {
		output, err = plugin.Get(ctx, workoutID)
	} else {
		output, err = plugin.GetLatest(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to get workout from %s: %v", pluginName, err)
	}

	fmt.Print(output)
	return nil
}
