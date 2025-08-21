package providers

import (
	"context"
	"fmt"
	"math"
	"strconv"
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

// Name returns the provider name
func (p *StravaProvider) Name() string {
	return "strava"
}

// GetLatest retrieves and displays the most recent workout
func (p *StravaProvider) GetLatest(ctx context.Context) (string, error) {
	return p.Get(ctx, "")
}

// Get retrieves and displays a specific workout by ID (empty string for latest)
func (p *StravaProvider) Get(ctx context.Context, activityID string) (string, error) {
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
