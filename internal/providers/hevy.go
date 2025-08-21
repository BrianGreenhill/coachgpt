package providers

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BrianGreenhill/coachgpt/pkg/hevy"
)

// HevyProvider implements the Provider interface for Hevy
type HevyProvider struct {
	client *hevy.Client
}

// NewHevyProvider creates a new Hevy provider instance
func NewHevyProvider(client *hevy.Client) *HevyProvider {
	return &HevyProvider{
		client: client,
	}
}

// NewHevyProviderForSetup creates a new Hevy provider for setup/configuration purposes
func NewHevyProviderForSetup() *HevyProvider {
	return &HevyProvider{
		client: nil, // Not needed for setup
	}
}

// Name returns the provider name
func (p *HevyProvider) Name() string {
	return "hevy"
}

// Description returns a human-readable description
func (p *HevyProvider) Description() string {
	return "Hevy (strength training)"
}

// IsConfigured checks if Hevy is properly configured
func (p *HevyProvider) IsConfigured() bool {
	apiKey := os.Getenv("HEVY_API_KEY")
	return apiKey != ""
}

// ShowConfig displays current configuration status
func (p *HevyProvider) ShowConfig() string {
	if !p.IsConfigured() {
		return "❌ Hevy: Not configured"
	}

	apiKey := os.Getenv("HEVY_API_KEY")
	return fmt.Sprintf("✅ Hevy: Configured\n   API Key: %s***",
		apiKey[:min(8, len(apiKey))])
}

// Setup runs the interactive setup for Hevy
func (p *HevyProvider) Setup(reader *bufio.Reader) error {
	fmt.Println()
	fmt.Println("💪 Hevy Setup")

	if p.IsConfigured() {
		fmt.Println("Hevy is already configured:")
		fmt.Println(p.ShowConfig())
		fmt.Print("Do you want to reconfigure? (y/N): ")

		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Keeping existing Hevy configuration.")
			return nil
		}
		fmt.Println()
	}

	fmt.Println("To set up Hevy, you need your API key:")
	fmt.Println("1. Open the Hevy app")
	fmt.Println("2. Go to Settings > Developer")
	fmt.Println("3. Copy your API key")
	fmt.Println()

	// Get API Key
	currentAPIKey := os.Getenv("HEVY_API_KEY")
	apiKeyPrompt := "Enter your Hevy API key: "
	if currentAPIKey != "" {
		apiKeyPrompt = fmt.Sprintf("Enter your Hevy API key (current: %s***): ", currentAPIKey[:min(8, len(currentAPIKey))])
	}
	fmt.Print(apiKeyPrompt)
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" && currentAPIKey != "" {
		apiKey = currentAPIKey
		fmt.Printf("Using existing API key: %s***\n", apiKey[:min(8, len(apiKey))])
	} else if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	// Write to shell profile
	return writeEnvVars(map[string]string{
		"HEVY_API_KEY": apiKey,
	}, "Hevy")
}

// GetLatest retrieves and displays the most recent workout
func (p *HevyProvider) GetLatest(ctx context.Context) (string, error) {
	if p.client == nil {
		return "", fmt.Errorf("provider not properly initialized - use regular setup, not setup-only instance")
	}

	w, err := p.client.GetLatestWorkout(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get latest Hevy workout: %v", err)
	}

	return p.formatWorkout(w), nil
}

// Get retrieves and displays a specific workout by ID
func (p *HevyProvider) Get(ctx context.Context, workoutID string) (string, error) {
	if p.client == nil {
		return "", fmt.Errorf("provider not properly initialized - use regular setup, not setup-only instance")
	}

	// Note: Hevy API doesn't support getting specific workout by ID in current implementation
	// This could be extended if the API supports it
	return "", fmt.Errorf("getting specific workout by ID not yet supported for Hevy")
}

// formatWorkout generates the markdown output for a Hevy workout
func (p *HevyProvider) formatWorkout(w *hevy.WorkoutJSON) string {
	var output string

	output += "--- Paste below ---\n"
	output += "## Strength Log\n"
	output += fmt.Sprintf("Title: %s\n", w.Title)

	startTime, _ := time.Parse(time.RFC3339, w.StartTime)
	endTime, _ := time.Parse(time.RFC3339, w.EndTime)
	duration := endTime.Sub(startTime)
	output += fmt.Sprintf("Duration: %s\n", formatDuration(duration))

	totalVol := 0.0
	totalReps := 0

	output += "Exercises:\n"
	for _, ex := range w.Exercises {
		exVol := 0.0
		exReps := 0
		output += fmt.Sprintf("- %s\n", ex.Title)
		for _, s := range ex.Sets {
			switch {
			case s.Reps != nil && s.WeightKG != nil:
				exReps += *s.Reps
				totalReps += *s.Reps
				vol := float64(*s.Reps) * *s.WeightKG
				exVol += vol
				totalVol += vol
				output += fmt.Sprintf("  • Set %d: %d reps @ %.1f kg\n", s.Index+1, *s.Reps, *s.WeightKG)
			case s.DurationSeconds != nil:
				output += fmt.Sprintf("  • Set %d: %ds (time)\n", s.Index+1, *s.DurationSeconds)
			default:
				output += fmt.Sprintf("  • Set %d: (type=%s)\n", s.Index+1, s.Type)
			}
		}
	}

	output += fmt.Sprintf("Total Volume: %.1f kg\n", totalVol)
	output += fmt.Sprintf("Total Reps: %d\n", totalReps)

	output += "Notes: []\n"
	output += "RPE: 0-10 (0=rest, 10=max effort)\n"
	output += "Fueling: [pre + during]\n"
	output += "--- End paste ---\n"

	return output
}

// formatDuration converts a time.Duration to HH:MM format
func formatDuration(d time.Duration) string {
	seconds := int64(d.Seconds())
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d", hours, minutes)
	}
	return fmt.Sprintf("%d:%02d", 0, minutes)
}
