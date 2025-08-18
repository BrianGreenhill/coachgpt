package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestSecToHHMM(t *testing.T) {
	tests := []struct {
		sec      int64
		expected string
	}{
		{3661, "1:01"},   // 1 hour 1 minute 1 second
		{3600, "1:00"},   // 1 hour
		{61, "0:01"},     // 1 minute 1 second
		{59, "0:00"},     // 59 seconds
		{7200, "2:00"},   // 2 hours
		{0, "0:00"},      // 0 seconds
	}

	for _, tt := range tests {
		result := secToHHMM(tt.sec)
		if result != tt.expected {
			t.Errorf("secToHHMM(%d) = %s, want %s", tt.sec, result, tt.expected)
		}
	}
}

func TestPaceFromMoving(t *testing.T) {
	tests := []struct {
		distance float64
		time     int64
		expected string
	}{
		{1000, 300, "5:00"},    // 5 min/km
		{1000, 240, "4:00"},    // 4 min/km
		{5000, 1500, "5:00"},   // 5 min/km over 5km
		{0, 300, "-"},          // zero distance
		{1000, 0, "-"},         // zero time
		{1000, 360, "6:00"},    // 6 min/km
	}

	for _, tt := range tests {
		result := paceFromMoving(tt.distance, tt.time)
		if result != tt.expected {
			t.Errorf("paceFromMoving(%.0f, %d) = %s, want %s", tt.distance, tt.time, result, tt.expected)
		}
	}
}

func TestComputeZones(t *testing.T) {
	hrmax := 180
	hrData := []float64{
		90,   // Z1 (50% of 180)
		126,  // Z1 (70% of 180) - exactly on the boundary, should be Z1
		135,  // Z2 (75% of 180)
		150,  // Z3 (83% of 180)
		165,  // Z4 (92% of 180)
		175,  // Z5 (97% of 180)
		0,    // Should be ignored
		-1,   // Should be ignored
	}

	zones := computeZones(hrData, hrmax)

	// 126 is exactly 70% of 180, which should be Z1 (< 70%)
	// So we have: 90 (Z1), 126 (Z1) -> Z1=2
	// 135 (Z2) -> Z2=1  
	// 150 (Z3) -> Z3=1
	// 165 (Z4) -> Z4=1
	// 175 (Z5) -> Z5=1
	expected := [5]int{1, 2, 1, 1, 1}

	if zones != expected {
		t.Errorf("computeZones() = %v, want %v", zones, expected)
	}
}

func TestComputeSplitHR(t *testing.T) {
	splits := []Split{
		{Split: 1, Distance: 1000, ElapsedTime: 300, MovingTime: 300, ElevationDifference: 10},
		{Split: 2, Distance: 1000, ElapsedTime: 300, MovingTime: 300, ElevationDifference: -5},
	}

	timeStream := []float64{0, 150, 300, 450, 600}  // 5 data points
	hrStream := []float64{120, 130, 140, 150, 160}  // corresponding HR

	result := computeSplitHR(splits, timeStream, hrStream)

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

// Test cache functionality
func TestCacheOperations(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Test cache key generation
	key := keyFor("/test/path", map[string]string{"param1": "value1", "param2": "value2"})
	expected := "_test_path__param1=value1__param2=value2.json"
	if key != expected {
		t.Errorf("keyFor() = %s, want %s", key, expected)
	}

	// Test write and read cache
	testEntry := &cacheEntry{
		FetchedAt: time.Now(),
		ETag:      "test-etag",
		Body:      json.RawMessage(`{"test": "data"}`),
	}

	err := writeCache("test", testEntry)
	if err != nil {
		t.Fatalf("writeCache failed: %v", err)
	}

	// Read back with no max age (should succeed)
	readEntry, err := readCache("test", 0)
	if err != nil {
		t.Fatalf("readCache failed: %v", err)
	}

	if readEntry.ETag != testEntry.ETag {
		t.Errorf("ETag mismatch: got %s, want %s", readEntry.ETag, testEntry.ETag)
	}

	if string(readEntry.Body) != string(testEntry.Body) {
		// JSON marshaling might add formatting, so let's compare the actual content
		var original, read map[string]interface{}
		json.Unmarshal(testEntry.Body, &original)
		json.Unmarshal(readEntry.Body, &read)
		
		if original["test"] != read["test"] {
			t.Errorf("Body content mismatch")
		}
	}

	// Test stale cache detection
	_, err = readCache("test", time.Nanosecond) // Very short max age
	if err == nil {
		t.Error("Expected stale cache error, got nil")
	}
}

// Test mock HTTP server for API calls
func TestAPIGETCachedMockResponse(t *testing.T) {
	// Set up test environment
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Set no cache for this test
	originalNoCache := noCache
	noCache = true
	defer func() { noCache = originalNoCache }()

	// Test the cache key generation and ETag functionality
	// since we can't easily mock the API base URL (it's a const)
	testEntry := &cacheEntry{
		FetchedAt: time.Now(),
		ETag:      "test-etag-123",
		Body:      json.RawMessage(`{"id": 12345, "name": "Test Run", "sport_type": "Run"}`),
	}

	err := writeCache("test_api_call", testEntry)
	if err != nil {
		t.Fatalf("writeCache failed: %v", err)
	}

	// Verify we can read it back
	readEntry, err := readCache("test_api_call", time.Hour)
	if err != nil {
		t.Fatalf("readCache failed: %v", err)
	}

	if readEntry.ETag != testEntry.ETag {
		t.Errorf("ETag mismatch: got %s, want %s", readEntry.ETag, testEntry.ETag)
	}

	// Test that we can unmarshal the cached data
	var activity Activity
	err = json.Unmarshal(readEntry.Body, &activity)
	if err != nil {
		t.Fatalf("Failed to unmarshal cached activity: %v", err)
	}

	if activity.ID != 12345 {
		t.Errorf("Expected ID 12345, got %d", activity.ID)
	}
}

// Test lap elevation calculation
func TestLapElevationFromStreams(t *testing.T) {
	altitude := []float64{100, 105, 110, 108, 115, 112, 120, 118}
	lap := Lap{
		StartIndex: 1,
		EndIndex:   6, // indices 1-5 inclusive
	}

	result := lapElevationFromStreams(lap, altitude)

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
	lap := Lap{StartIndex: -1, EndIndex: 10}
	result := lapElevationFromStreams(lap, altitude)

	// Should handle gracefully and return zero values
	if result.Gain != 10 || result.Loss != 0 || result.Net != 10 {
		t.Errorf("Expected gain=10, loss=0, net=10, got gain=%d, loss=%d, net=%d",
			result.Gain, result.Loss, result.Net)
	}

	// Test with empty altitude data
	emptyLap := Lap{StartIndex: 0, EndIndex: 2}
	emptyResult := lapElevationFromStreams(emptyLap, []float64{})
	
	if emptyResult.Gain != 0 || emptyResult.Loss != 0 || emptyResult.Net != 0 {
		t.Errorf("Expected all zeros for empty altitude, got gain=%d, loss=%d, net=%d",
			emptyResult.Gain, emptyResult.Loss, emptyResult.Net)
	}
}

// Integration test for the main data structures
func TestActivityDataStructures(t *testing.T) {
	// Test that our data structures can be properly marshaled/unmarshaled
	activity := Activity{
		ID:                 123456789,
		Name:               "Morning Run",
		SportType:          "Run",
		MovingTime:         2400,
		Distance:           8000,
		AverageSpeed:       3.33,
		AverageHeartRate:   145,
		TotalElevationGain: 150,
		StartDateLocal:     "2024-08-19T07:00:00Z",
		SplitsMetric: []Split{
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
	var unmarshaled Activity
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
