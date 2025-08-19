package strava

import (
	"briangreenhill/coachgpt/cache"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Split represents a metric split from a Strava activity
type Split struct {
	Split               int     `json:"split"`                // 1,2,3...
	Distance            float64 `json:"distance"`             // meters (usually ~1000)
	ElapsedTime         int     `json:"elapsed_time"`         // sec
	MovingTime          int64   `json:"moving_time"`          // sec
	AverageSpeed        float64 `json:"average_speed"`        // m/s
	ElevationDifference float64 `json:"elevation_difference"` // meters
	PaceZone            int     `json:"pace_zone"`
}

// Lap represents a lap from a Strava activity
type Lap struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`              // e.g., "Lap 1"
	LapIndex         int     `json:"lap_index"`         // 1-based
	Split            int     `json:"split"`             // sometimes present
	ElapsedTime      int     `json:"elapsed_time"`      // sec
	MovingTime       int64   `json:"moving_time"`       // sec
	Distance         float64 `json:"distance"`          // meters
	AverageSpeed     float64 `json:"average_speed"`     // m/s
	MaxSpeed         float64 `json:"max_speed"`         // m/s
	AverageHeartrate float64 `json:"average_heartrate"` // bpm (may be 0)
	MaxHeartrate     float64 `json:"max_heartrate"`     // bpm
	ElevationGain    float64 `json:"total_elevation_gain"`
	ElevationLoss    float64 `json:"total_elevation_loss"`
	StartIndex       int     `json:"start_index"` // stream index (optional)
	EndIndex         int     `json:"end_index"`   // stream index (optional)
}

// Activity represents a Strava activity
type Activity struct {
	ID                 int64   `json:"id"`
	Name               string  `json:"name"`
	SportType          string  `json:"sport_type"`
	MovingTime         int64   `json:"moving_time"`
	Distance           float64 `json:"distance"`
	AverageSpeed       float64 `json:"average_speed"`
	AverageHeartRate   float64 `json:"average_heartrate"`
	TotalElevationGain float64 `json:"total_elevation_gain"`
	StartDateLocal     string  `json:"start_date_local"` // e.g. "2023-10-01T10:00:00Z"

	SplitsMetric []Split `json:"splits_metric"` // metric splits, may be empty
}

// Stream represents a data stream from Strava
type Stream struct {
	Data []float64 `json:"data"`
}

// Streams represents all available streams for an activity
type Streams struct {
	Time           Stream `json:"time"`
	Heartrate      Stream `json:"heartrate"`
	VelocitySmooth Stream `json:"velocity_smooth"`
	Distance       Stream `json:"distance"`
	Altitude       Stream `json:"altitude"`
}

// apiGETCached performs a cached GET request to the Strava API
func (c *Client) apiGETCached(token, path string, params map[string]string, out any, ttl time.Duration) error {
	u := APIBase + path
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if params != nil {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	var cacheName string
	if c.Cache != nil {
		cacheName = c.Cache.KeyFor(path, params)
	}

	// Try fresh-enough cache
	if !c.NoCache && c.Cache != nil && cacheName != "" {
		if entry, exists := c.Cache.Read(cacheName, ttl); exists && len(entry.Body) > 0 {
			return json.Unmarshal(entry.Body, out)
		}
	}

	// Try ETag (revalidation) if we have any cache (even if stale)
	if c.Cache != nil && cacheName != "" {
		if entry, _ := c.Cache.Read(cacheName, 0); entry != nil && entry.ETag != "" {
			req.Header.Set("If-None-Match", entry.ETag)
			resp, err := http.DefaultClient.Do(req)
			if err == nil && resp.StatusCode == http.StatusNotModified {
				// 304 -> use cached body
				_ = resp.Body.Close()
				return json.Unmarshal(entry.Body, out)
			}
			// fall through to normal fetch if error or not 304
			if resp != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
	}

	// Normal fetch
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s -> %s: %s", path, resp.Status, string(body))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Save cache with new ETag (if any)
	if c.Cache != nil && cacheName != "" {
		entry := &cache.Entry{
			ETag: resp.Header.Get("ETag"),
			Body: json.RawMessage(body),
		}
		_ = c.Cache.Write(cacheName, entry)
	}

	return json.Unmarshal(body, out)
}

// GetLatestRun gets the most recent running activity
func (c *Client) GetLatestRun(token string) (*Activity, error) {
	var activities []Activity
	if err := c.apiGETCached(token, "/athlete/activities", map[string]string{"per_page": "10", "include_all_efforts": "true"}, &activities, 24*time.Hour); err != nil {
		return nil, err
	}
	for _, a := range activities {
		if a.SportType == "Run" || a.SportType == "TrailRun" {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("no recent run activity found")
}

// GetActivity gets a specific activity by ID
func (c *Client) GetActivity(token string, id int64) (*Activity, error) {
	var activity Activity
	err := c.apiGETCached(token, fmt.Sprintf("/activities/%d", id), map[string]string{"include_all_efforts": "true"}, &activity, 24*time.Hour)
	return &activity, err
}

// GetLaps gets the laps for a specific activity
func (c *Client) GetLaps(token string, id int64) ([]Lap, error) {
	var laps []Lap
	err := c.apiGETCached(token, fmt.Sprintf("/activities/%d/laps", id), nil, &laps, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	return laps, nil
}

// GetStreams gets the data streams for a specific activity
func (c *Client) GetStreams(token string, id int64) (*Streams, error) {
	var raw map[string]Stream
	err := c.apiGETCached(token, fmt.Sprintf("/activities/%d/streams", id),
		map[string]string{"keys": "time,heartrate,velocity_smooth,distance,altitude", "key_by_type": "true"}, &raw, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	s := &Streams{}
	if v, ok := raw["time"]; ok {
		s.Time = v
	}
	if v, ok := raw["heartrate"]; ok {
		s.Heartrate = v
	}
	if v, ok := raw["velocity_smooth"]; ok {
		s.VelocitySmooth = v
	}
	if v, ok := raw["distance"]; ok {
		s.Distance = v
	}
	if v, ok := raw["altitude"]; ok {
		s.Altitude = v
	}
	return s, nil
}
