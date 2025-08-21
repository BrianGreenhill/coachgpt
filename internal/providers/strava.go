package providers

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/BrianGreenhill/coachgpt/pkg/strava"
)

// StravaProvider implements the Provider interface for Strava
type StravaProvider struct {
	client *strava.Client
	hrmax  int
}

// NewStravaProvider creates a new Strava provider instance
func NewStravaProvider(client *strava.Client, hrmax int) *StravaProvider {
	return &StravaProvider{
		client: client,
		hrmax:  hrmax,
	}
}

// NewStravaProviderForSetup creates a new Strava provider for setup/configuration purposes
func NewStravaProviderForSetup() *StravaProvider {
	return &StravaProvider{
		client: nil, // Not needed for setup
		hrmax:  0,   // Will be read from environment
	}
}

// Name returns the provider name
func (p *StravaProvider) Name() string {
	return "strava"
}

// Description returns a human-readable description
func (p *StravaProvider) Description() string {
	return "Strava (cardio activities - running, cycling, etc.)"
}

// IsConfigured checks if Strava is properly configured
func (p *StravaProvider) IsConfigured() bool {
	clientID := os.Getenv("STRAVA_CLIENT_ID")
	clientSecret := os.Getenv("STRAVA_CLIENT_SECRET")
	hrMaxStr := os.Getenv("STRAVA_HRMAX")

	if clientID == "" || clientSecret == "" || hrMaxStr == "" {
		return false
	}

	hrMax, err := strconv.Atoi(hrMaxStr)
	return err == nil && hrMax > 0
}

// ShowConfig displays current configuration status
func (p *StravaProvider) ShowConfig() string {
	if !p.IsConfigured() {
		return "❌ Strava: Not configured"
	}

	clientID := os.Getenv("STRAVA_CLIENT_ID")
	hrMax := os.Getenv("STRAVA_HRMAX")

	return fmt.Sprintf("✅ Strava: Configured\n   Client ID: %s***\n   Max HR: %s",
		clientID[:min(6, len(clientID))], hrMax)
}

// Setup runs the interactive setup for Strava
func (p *StravaProvider) Setup(reader *bufio.Reader) error {
	fmt.Println()
	fmt.Println("🚴‍♂️ Strava Setup")

	if p.IsConfigured() {
		fmt.Println("Strava is already configured:")
		fmt.Println(p.ShowConfig())
		fmt.Print("Do you want to reconfigure? (y/N): ")

		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Keeping existing Strava configuration.")
			return nil
		}
		fmt.Println()
	}

	fmt.Println("To set up Strava, you need to create a Strava API application:")
	fmt.Println("1. Go to https://www.strava.com/settings/api")
	fmt.Println("2. Create a new application")
	fmt.Println("3. Copy your Client ID and Client Secret")
	fmt.Println()

	// Get Client ID
	currentClientID := os.Getenv("STRAVA_CLIENT_ID")
	clientIDPrompt := "Enter your Strava Client ID: "
	if currentClientID != "" {
		clientIDPrompt = fmt.Sprintf("Enter your Strava Client ID (current: %s***): ", currentClientID[:min(6, len(currentClientID))])
	}
	fmt.Print(clientIDPrompt)
	clientID, _ := reader.ReadString('\n')
	clientID = strings.TrimSpace(clientID)
	if clientID == "" && currentClientID != "" {
		clientID = currentClientID
		fmt.Printf("Using existing Client ID: %s***\n", clientID[:min(6, len(clientID))])
	} else if clientID == "" {
		return fmt.Errorf("client ID is required")
	}

	// Get Client Secret
	currentClientSecret := os.Getenv("STRAVA_CLIENT_SECRET")
	clientSecretPrompt := "Enter your Strava Client Secret: "
	if currentClientSecret != "" {
		clientSecretPrompt = "Enter your Strava Client Secret (current: ***): "
	}
	fmt.Print(clientSecretPrompt)
	clientSecret, _ := reader.ReadString('\n')
	clientSecret = strings.TrimSpace(clientSecret)
	if clientSecret == "" && currentClientSecret != "" {
		clientSecret = currentClientSecret
		fmt.Println("Using existing Client Secret: ***")
	} else if clientSecret == "" {
		return fmt.Errorf("client secret is required")
	}

	// Get HR Max
	currentHRMax := os.Getenv("STRAVA_HRMAX")
	hrMaxPrompt := "Enter your maximum heart rate (e.g., 185): "
	if currentHRMax != "" {
		hrMaxPrompt = fmt.Sprintf("Enter your maximum heart rate (current: %s): ", currentHRMax)
	}
	fmt.Print(hrMaxPrompt)
	hrMaxStr, _ := reader.ReadString('\n')
	hrMaxStr = strings.TrimSpace(hrMaxStr)

	if hrMaxStr == "" && currentHRMax != "" {
		hrMaxStr = currentHRMax
		fmt.Printf("Using existing HR Max: %s\n", hrMaxStr)
	} else if hrMaxStr != "" {
		hrMax, err := strconv.Atoi(hrMaxStr)
		if err != nil || hrMax <= 0 {
			return fmt.Errorf("invalid maximum heart rate: %s", hrMaxStr)
		}
	} else {
		return fmt.Errorf("maximum heart rate is required")
	}

	// Write to shell profile
	return writeEnvVars(map[string]string{
		"STRAVA_CLIENT_ID":     clientID,
		"STRAVA_CLIENT_SECRET": clientSecret,
		"STRAVA_HRMAX":         hrMaxStr,
	}, "Strava")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetLatest retrieves and displays the most recent workout
func (p *StravaProvider) GetLatest(ctx context.Context) (string, error) {
	if p.client == nil {
		return "", fmt.Errorf("provider not properly initialized - use regular setup, not setup-only instance")
	}
	return p.Get(ctx, "")
}

// Get retrieves and displays a specific workout by ID (empty string for latest)
func (p *StravaProvider) Get(ctx context.Context, activityID string) (string, error) {
	if p.client == nil {
		return "", fmt.Errorf("provider not properly initialized - use regular setup, not setup-only instance")
	}

	// Get OAuth token
	token, err := p.client.EnsureTokens()
	if err != nil {
		return "", fmt.Errorf("failed to get OAuth token: %v", err)
	}

	// Get the activity
	var act *strava.Activity
	if activityID != "" {
		id, err := strconv.ParseInt(activityID, 10, 64)
		if err != nil {
			return "", fmt.Errorf("invalid activity ID: %v", err)
		}
		act, err = p.client.GetActivity(token, id)
		if err != nil {
			return "", fmt.Errorf("failed to get activity: %v", err)
		}
	} else {
		latest, err := p.client.GetLatestRun(token)
		if err != nil {
			return "", fmt.Errorf("failed to get latest run: %v", err)
		}
		act, err = p.client.GetActivity(token, latest.ID)
		if err != nil {
			return "", fmt.Errorf("failed to get activity: %v", err)
		}
	}

	// Get additional data
	streams, _ := p.client.GetStreams(token, act.ID)
	laps, _ := p.client.GetLaps(token, act.ID)

	// Calculate heart rate zones
	var zones [5]int
	if streams != nil && len(streams.Heartrate.Data) > 0 {
		zones = strava.ComputeZones(streams.Heartrate.Data, p.hrmax)
	}

	// Format the output
	return p.formatOutput(act, streams, laps, zones), nil
}

// formatOutput generates the markdown output for a Strava activity
func (p *StravaProvider) formatOutput(act *strava.Activity, streams *strava.Streams, laps []strava.Lap, zones [5]int) string {
	var output string

	avgHR := "-"
	if act.AverageHeartRate > 0 {
		avgHR = fmt.Sprintf("%d", int(act.AverageHeartRate))
	}

	output += "--- Paste below ---\n"
	output += "## Run Log\n"

	typ := "Run"
	if act.SportType == "TrailRun" {
		typ = "Trail Run"
	}
	output += fmt.Sprintf("- **Type:** [%s] %s\n", typ, act.Name)

	when := act.StartDateLocal
	if when == "" {
		when = time.Now().Format(time.RFC3339)
	}
	output += fmt.Sprintf("- **When:** %s\n", when)
	output += fmt.Sprintf("- **Duration:** %s\n", strava.SecToHHMM(act.MovingTime))
	output += fmt.Sprintf("- **Distance:** %.1f km (elev %d m)\n", act.Distance/1000.0, int(math.Round(act.TotalElevationGain)))
	output += fmt.Sprintf("- **Avg Pace:** %s / km\n", strava.PaceFromMoving(act.Distance, act.MovingTime))
	output += fmt.Sprintf("- **Avg HR:** %s bpm\n", avgHR)

	if streams != nil && len(streams.Heartrate.Data) > 0 {
		total := 0
		for _, v := range zones {
			total += v
		}
		if total == 0 {
			total = 1
		}
		toPct := func(n int) int { return int(math.Round(float64(n) / float64(total) * 100)) }
		output += fmt.Sprintf("- **HR Zones:** Z1: %d%%, Z2: %d%%, Z3: %d%%, Z4: %d%%, Z5: %d%%\n",
			toPct(zones[0]), toPct(zones[1]), toPct(zones[2]), toPct(zones[3]), toPct(zones[4]))
	} else {
		output += "- **HR Zones:** No heart rate data available\n"
	}

	output += "- **Splits:**\n"
	output += strava.FormatSplitsWithHR(act, streams)

	output += "- **Laps:**\n"
	output += strava.FormatLapsWithElevation(laps, streams)

	output += "- **RPE:** 0-10 (0=rest, 10=max effort)\n"
	output += "- **Fueling** [pre + during]\n"
	output += "- **Terrain/Weather:** []\n"
	output += "- **Notes:** []\n"
	output += "--- End paste ---\n"

	return output
}
