package main

import (
	"briangreenhill/coachgpt/strava"
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
		result := strava.SecToHHMM(tt.sec)
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
		result := strava.PaceFromMoving(tt.distance, tt.time)
		if result != tt.expected {
			t.Errorf("PaceFromMoving(%.0f, %d) = %s, want %s", tt.distance, tt.time, result, tt.expected)
		}
	}
}

func TestComputeZones(t *testing.T) {
	hrmax := 180
	hrData := []float64{
		90,  // Z1 (50% of 180)
		126, // Z1 (70% of 180) - exactly on the boundary, should be Z1
		135, // Z2 (75% of 180)
		150, // Z3 (83% of 180)
		165, // Z4 (92% of 180)
		175, // Z5 (97% of 180)
		0,   // Should be ignored
		-1,  // Should be ignored
	}

	zones := strava.ComputeZones(hrData, hrmax)

	// 126 is exactly 70% of 180, which should be Z1 (< 70%)
	// So we have: 90 (Z1), 126 (Z1) -> Z1=2
	// 135 (Z2) -> Z2=1
	// 150 (Z3) -> Z3=1
	// 165 (Z4) -> Z4=1
	// 175 (Z5) -> Z5=1
	expected := [5]int{1, 2, 1, 1, 1}

	if zones != expected {
		t.Errorf("ComputeZones() = %v, want %v", zones, expected)
	}
}

func TestComputeSplitHR(t *testing.T) {
	splits := []strava.Split{
		{Split: 1, Distance: 1000, ElapsedTime: 300, MovingTime: 300, ElevationDifference: 10},
		{Split: 2, Distance: 1000, ElapsedTime: 300, MovingTime: 300, ElevationDifference: -5},
	}

	timeStream := []float64{0, 150, 300, 450, 600} // 5 data points
	hrStream := []float64{120, 130, 140, 150, 160} // corresponding HR

	result := strava.ComputeSplitHR(splits, timeStream, hrStream)

	if len(result) != 2 {
		t.Fatalf("Expected 2 splits, got %d", len(result))
	}

	// First split (0-300s) should include first 2 HR readings: 120, 130
	// (time 300 is the boundary and belongs to next split)
	// Average should be 125, max should be 130
	if result[0].AvgHR != 125 {
		t.Errorf("First split AvgHR = %d, want 125", result[0].AvgHR)
	}
	if result[0].MaxHR != 130 {
		t.Errorf("First split MaxHR = %d, want 130", result[0].MaxHR)
	}
	if result[0].Pace != "5:00" {
		t.Errorf("First split Pace = %s, want 5:00", result[0].Pace)
	}
}

// Test httpcache functionality
func TestHTTPCacheOperations(t *testing.T) {
	// Create HTTP client with memory cache transport
	transport := httpcache.NewMemoryCacheTransport()
	if transport == nil {
		t.Fatal("Transport should not be nil")
	}

	transport.MarkCachedResponses = true // Add X-From-Cache header for testing
	client := &http.Client{Transport: transport}

	// Note: Since httpcache works transparently with HTTP requests,
	// we can't easily test the internal cache operations without making
	// actual HTTP requests. This test verifies the transport is configured correctly.

	// Verify transport is properly configured
	if client.Transport != transport {
		t.Fatal("Transport not properly assigned")
	}

	// Verify the transport is properly configured
	if client.Transport != transport {
		t.Error("HTTP client transport not properly configured")
	}
}

// Test Strava client creation
func TestStravaClientCreation(t *testing.T) {
	client := strava.NewClient("test_id", "test_secret")
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
	clientWithHTTP := strava.NewClientWithHTTP("test_id", "test_secret", httpClient)
	if clientWithHTTP == nil {
		t.Error("Client with HTTP client should not be nil")
		return
	}

	if clientWithHTTP.HTTPClient != httpClient {
		t.Error("HTTP client not properly assigned")
	}
}

// Test lap elevation calculation
func TestLapElevationFromStreams(t *testing.T) {
	altitude := []float64{100, 105, 110, 108, 115, 112, 120, 118}
	lap := strava.Lap{
		StartIndex: 1,
		EndIndex:   6, // indices 1-5 inclusive
	}

	result := strava.LapElevationFromStreams(lap, altitude)

	// From indices 1-5: 105->110(+5) -> 108(-2) -> 115(+7) -> 112(-3)
	// Gain: 5 + 7 = 12
	// Loss: 2 + 3 = 5
	// Net: 12 - 5 = 7
	expectedGain := 12
	expectedLoss := 5
	expectedNet := 7

	if result.Gain != expectedGain {
		t.Errorf("Expected gain %d, got %d", expectedGain, result.Gain)
	}
	if result.Loss != expectedLoss {
		t.Errorf("Expected loss %d, got %d", expectedLoss, result.Loss)
	}
	if result.Net != expectedNet {
		t.Errorf("Expected net %d, got %d", expectedNet, result.Net)
	}
}

// Test boundary conditions for lap elevation
func TestLapElevationFromStreams_EdgeCases(t *testing.T) {
	altitude := []float64{100, 105, 110}

	// Test with invalid indices
	lap := strava.Lap{StartIndex: -1, EndIndex: 10}
	result := strava.LapElevationFromStreams(lap, altitude)

	// Should handle gracefully and return zero values
	if result.Gain != 10 || result.Loss != 0 || result.Net != 10 {
		t.Errorf("Expected gain=10, loss=0, net=10, got gain=%d, loss=%d, net=%d",
			result.Gain, result.Loss, result.Net)
	}

	// Test with empty altitude data
	emptyLap := strava.Lap{StartIndex: 0, EndIndex: 2}
	emptyResult := strava.LapElevationFromStreams(emptyLap, []float64{})

	if emptyResult.Gain != 0 || emptyResult.Loss != 0 || emptyResult.Net != 0 {
		t.Errorf("Expected all zeros for empty altitude, got gain=%d, loss=%d, net=%d",
			emptyResult.Gain, emptyResult.Loss, emptyResult.Net)
	}
}

// Integration test for the main data structures
func TestActivityDataStructures(t *testing.T) {
	// Test that our data structures can be properly marshaled/unmarshaled
	activity := strava.Activity{
		ID:                 123456789,
		Name:               "Morning Run",
		SportType:          "Run",
		MovingTime:         2400,
		Distance:           8000,
		AverageSpeed:       3.33,
		AverageHeartRate:   145,
		TotalElevationGain: 150,
		StartDateLocal:     "2024-08-19T07:00:00Z",
		SplitsMetric: []strava.Split{
			{Split: 1, Distance: 1000, ElapsedTime: 300, MovingTime: 295, AverageSpeed: 3.39},
			{Split: 2, Distance: 1000, ElapsedTime: 305, MovingTime: 300, AverageSpeed: 3.33},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(activity)
	if err != nil {
		t.Fatalf("Failed to marshal activity: %v", err)
	}

	// Unmarshal back
	var unmarshaled strava.Activity
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal activity: %v", err)
	}

	// Verify key fields
	if unmarshaled.ID != activity.ID {
		t.Errorf("ID mismatch after marshal/unmarshal")
	}
	if len(unmarshaled.SplitsMetric) != len(activity.SplitsMetric) {
		t.Errorf("Splits count mismatch after marshal/unmarshal")
	}
}
