package strava

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gregjones/httpcache"
)

func TestSecToHHMM(t *testing.T) {
	tests := []struct {
		sec      int64
		expected string
	}{
		{3661, "1:01"}, // 1 hour 1 minute 1 second
		{3600, "1:00"}, // 1 hour
		{61, "0:01"},   // 1 minute 1 second
		{59, "0:00"},   // 59 seconds
		{7200, "2:00"}, // 2 hours
		{0, "0:00"},    // 0 seconds
	}

	for _, tt := range tests {
		result := SecToHHMM(tt.sec)
		if result != tt.expected {
			t.Errorf("SecToHHMM(%d) = %s, want %s", tt.sec, result, tt.expected)
		}
	}
}

func TestPaceFromMoving(t *testing.T) {
	tests := []struct {
		distance float64
		time     int64
		expected string
	}{
		{1000, 300, "5:00"},  // 5 min/km
		{1000, 240, "4:00"},  // 4 min/km
		{5000, 1500, "5:00"}, // 5 min/km over 5km
		{0, 300, "-"},        // zero distance
		{1000, 0, "-"},       // zero time
		{1000, 360, "6:00"},  // 6 min/km
	}

	for _, tt := range tests {
		result := PaceFromMoving(tt.distance, tt.time)
		if result != tt.expected {
			t.Errorf("PaceFromMoving(%.0f, %d) = %s, want %s", tt.distance, tt.time, result, tt.expected)
		}
	}
}

func TestComputeZones(t *testing.T) {
	hr := []float64{120, 140, 160, 180, 160, 140, 120}
	hrmax := 190

	zones := ComputeZones(hr, hrmax)

	// Verify zones array structure
	if len(zones) != 5 {
		t.Errorf("Expected 5 zones, got %d", len(zones))
	}

	// Basic sanity check - should have some distribution
	total := 0
	for _, count := range zones {
		total += count
	}
	if total != len(hr) {
		t.Errorf("Expected total count %d, got %d", len(hr), total)
	}

	// Test with empty data
	emptyZones := ComputeZones([]float64{}, hrmax)
	for i, count := range emptyZones {
		if count != 0 {
			t.Errorf("Expected zone %d to be 0 for empty data, got %d", i+1, count)
		}
	}
}

func TestStravaClientCreation(t *testing.T) {
	// Test basic client creation
	client := NewClient("test_id", "test_secret")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.ClientID != "test_id" {
		t.Errorf("Expected ClientID 'test_id', got %s", client.ClientID)
	}

	if client.ClientSecret != "test_secret" {
		t.Errorf("Expected ClientSecret 'test_secret', got %s", client.ClientSecret)
	}

	// Test with HTTP client (new approach with httpcache)
	transport := httpcache.NewMemoryCacheTransport()
	httpClient := &http.Client{Transport: transport}
	clientWithHTTP := NewClientWithHTTP("test_id", "test_secret", httpClient)
	if clientWithHTTP == nil {
		t.Error("Client with HTTP client should not be nil")
		return
	}

	if clientWithHTTP.HTTPClient != httpClient {
		t.Error("HTTP client not properly assigned")
	}
}

func TestLapElevationFromStreams(t *testing.T) {
	altitude := []float64{100, 105, 110, 108, 115, 112, 120, 118}
	lap := Lap{
		StartIndex: 1,
		EndIndex:   6, // indices 1-5 inclusive
	}

	result := LapElevationFromStreams(lap, altitude)

	// Verify the result structure
	if result.Gain < 0 {
		t.Errorf("Expected non-negative elevation gain, got %d", result.Gain)
	}
	if result.Loss < 0 {
		t.Errorf("Expected non-negative elevation loss, got %d", result.Loss)
	}
}

func TestLapElevationFromStreams_EdgeCases(t *testing.T) {
	altitude := []float64{100, 105, 110}

	// Test with indices out of bounds
	lap := Lap{StartIndex: 10, EndIndex: 20}
	result := LapElevationFromStreams(lap, altitude)
	if result.Gain != 0 || result.Loss != 0 {
		t.Errorf("Expected 0 gain/loss for out of bounds indices, got gain=%d, loss=%d", result.Gain, result.Loss)
	}

	// Test with start >= end
	lap = Lap{StartIndex: 2, EndIndex: 1}
	result = LapElevationFromStreams(lap, altitude)
	if result.Gain != 0 || result.Loss != 0 {
		t.Errorf("Expected 0 gain/loss for start >= end, got gain=%d, loss=%d", result.Gain, result.Loss)
	}

	// Test with empty altitude data
	lap = Lap{StartIndex: 0, EndIndex: 2}
	result = LapElevationFromStreams(lap, []float64{})
	if result.Gain != 0 || result.Loss != 0 {
		t.Errorf("Expected 0 gain/loss for empty altitude data, got gain=%d, loss=%d", result.Gain, result.Loss)
	}
}

func TestActivityDataStructures(t *testing.T) {
	// Test Activity JSON unmarshaling
	activityJSON := `{
		"id": 123456789,
		"name": "Morning Run",
		"sport_type": "Run",
		"moving_time": 1800,
		"distance": 5000.0,
		"average_speed": 2.78,
		"average_heartrate": 150.5,
		"total_elevation_gain": 100.0,
		"start_date_local": "2023-10-01T08:00:00Z",
		"splits_metric": [
			{
				"split": 1,
				"distance": 1000.0,
				"elapsed_time": 360,
				"moving_time": 360,
				"average_speed": 2.78,
				"elevation_difference": 10.0,
				"pace_zone": 2
			}
		]
	}`

	var activity Activity
	err := json.Unmarshal([]byte(activityJSON), &activity)
	if err != nil {
		t.Fatalf("Failed to unmarshal activity JSON: %v", err)
	}

	if activity.ID != 123456789 {
		t.Errorf("Expected ID 123456789, got %d", activity.ID)
	}

	if activity.Name != "Morning Run" {
		t.Errorf("Expected name 'Morning Run', got '%s'", activity.Name)
	}

	if activity.SportType != "Run" {
		t.Errorf("Expected sport_type 'Run', got '%s'", activity.SportType)
	}

	if len(activity.SplitsMetric) != 1 {
		t.Errorf("Expected 1 split, got %d", len(activity.SplitsMetric))
	}

	if activity.SplitsMetric[0].Split != 1 {
		t.Errorf("Expected split number 1, got %d", activity.SplitsMetric[0].Split)
	}
}
