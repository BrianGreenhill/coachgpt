package main

import (
	"briangreenhill/coachgpt/cache"
	"briangreenhill/coachgpt/hevy"
	"briangreenhill/coachgpt/strava"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
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
			fmt.Println("  STRAVA_CLIENT_ID    Your Strava client ID (required)")
			fmt.Println("  STRAVA_CLIENT_SECRET Your Strava client secret (required)")
			fmt.Println("  STRAVA_HRMAX        Your maximum heart rate (required, e.g. 185)")
			fmt.Println("  STRAVA_ACTIVITY_ID  Specific activity ID to fetch (optional)")
			fmt.Println("  STRAVA_NOCACHE      Disable caching (optional)")
		case "version", "--version", "-v":
			fmt.Println("CoachGPT v0.1.0")
		case "strength", "--strength", "-s":
			runStrengthIntegration()
		default:
			return fmt.Errorf("unknown command: %s", args[0])
		}
	} else {
		runStravaIntegration()
	}

	return nil
}

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("environment variable %s is not set", key)
	}
	return val
}

func runStravaIntegration() {
	clientID := mustEnv("STRAVA_CLIENT_ID")
	clientSecret := mustEnv("STRAVA_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		log.Fatal("Please set STRAVA_CLIENT_ID and STRAVA_CLIENT_SECRET environment variables")
	}
	hrmaxStr := os.Getenv("STRAVA_HRMAX")
	if hrmaxStr == "" {
		log.Fatal("Please set STRAVA_HRMAX environment variable")
	}
	hrmax, err := strconv.Atoi(hrmaxStr)
	if err != nil || hrmax < 120 {
		log.Fatalf("HR_MAX must be an int like 185-200")
	}
	var activityID string
	if v := os.Getenv("STRAVA_ACTIVITY_ID"); v != "" {
		if _, err := strconv.ParseInt(v, 10, 64); err != nil {
			log.Fatalf("STRAVA_ACTIVITY_ID must be a positive integer")
		}
		activityID = v
	}

	// Create Strava client with unified cache
	stravaCache, err := cache.NewStravaCache()
	if err != nil {
		log.Fatalf("failed to create cache: %v", err)
	}
	client := strava.NewClientWithCache(clientID, clientSecret, stravaCache)

	// Create Strava plugin
	plugin := strava.NewPlugin(client, hrmax)

	// Use plugin to get and display workout
	ctx := context.Background()
	var output string
	if activityID != "" {
		output, err = plugin.Get(ctx, activityID)
	} else {
		output, err = plugin.GetLatest(ctx)
	}
	if err != nil {
		log.Fatalf("Failed to get workout: %v", err)
	}

	fmt.Print(output)
}

func runStrengthIntegration() {
	apiKey := mustEnv("HEVY_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set HEVY_API_KEY environment variable")
	}

	// Create unified cache and adapt it for Hevy
	baseCache, err := cache.NewHevyCache()
	if err != nil {
		log.Fatalf("failed to create cache: %v", err)
	}
	hevyCache := cache.NewHevyAdapter(baseCache)

	client, err := hevy.New(apiKey, hevy.WithCache(hevyCache, 24*time.Hour))
	if err != nil {
		log.Fatalf("Failed to create Hevy client: %v", err)
	}

	// Create Hevy plugin
	plugin := hevy.NewPlugin(client)

	// Use plugin to get and display workout
	ctx := context.Background()
	output, err := plugin.GetLatest(ctx)
	if err != nil {
		log.Fatalf("Failed to get workout: %v", err)
	}

	fmt.Print(output)
}
