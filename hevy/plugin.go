package hevy

import (
	"briangreenhill/coachgpt/plugins"
	"context"
	"fmt"
	"time"
)

// Plugin implements the plugins.Plugin interface for Hevy
type Plugin struct {
	client *Client
}

// NewPlugin creates a new Hevy plugin instance
func NewPlugin(client *Client) *Plugin {
	return &Plugin{
		client: client,
	}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "hevy"
}

// GetLatest retrieves and displays the most recent workout
func (p *Plugin) GetLatest(ctx context.Context) (string, error) {
	w, err := p.client.GetLatestWorkout(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get latest Hevy workout: %v", err)
	}

	return p.formatWorkout(w), nil
}

// Get retrieves and displays a specific workout by ID
func (p *Plugin) Get(ctx context.Context, workoutID string) (string, error) {
	// Note: Hevy API doesn't support getting specific workout by ID in current implementation
	// This could be extended if the API supports it
	return "", fmt.Errorf("getting specific workout by ID not yet supported for Hevy")
}

// formatWorkout generates the markdown output for a Hevy workout
func (p *Plugin) formatWorkout(w *WorkoutJSON) string {
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

// Ensure Plugin implements the plugins.Plugin interface
var _ plugins.Plugin = (*Plugin)(nil)
