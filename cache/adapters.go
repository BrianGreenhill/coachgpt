package cache

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	// ErrCacheNotFound is returned when a cache entry is not found or expired
	ErrCacheNotFound = errors.New("cache entry not found or expired")
)

// StravaAdapter adapts the unified cache to work with Strava's expected interface
type StravaAdapter struct {
	cache Cache
}

// NewStravaAdapter creates a new adapter for Strava cache interface
func NewStravaAdapter(cache Cache) *StravaAdapter {
	return &StravaAdapter{cache: cache}
}

// StravaEntry represents a Strava cache entry (for compatibility)
type StravaEntry struct {
	FetchedAt time.Time       `json:"fetched_at"`
	ETag      string          `json:"etag,omitempty"`
	Body      json.RawMessage `json:"body"`
}

// ReadCache implements the Strava CacheReader interface
func (sa *StravaAdapter) ReadCache(name string, maxAge time.Duration) (*StravaEntry, error) {
	entry, exists := sa.cache.Read(name, maxAge)
	if !exists {
		return nil, ErrCacheNotFound
	}

	return &StravaEntry{
		FetchedAt: entry.FetchedAt,
		ETag:      entry.ETag,
		Body:      entry.Body,
	}, nil
}

// WriteCache implements the Strava CacheWriter interface
func (sa *StravaAdapter) WriteCache(name string, se *StravaEntry) error {
	entry := &Entry{
		FetchedAt: se.FetchedAt,
		ETag:      se.ETag,
		Body:      se.Body,
	}
	return sa.cache.Write(name, entry)
}

// KeyFor implements the Strava CacheWriter interface
func (sa *StravaAdapter) KeyFor(path string, params map[string]string) string {
	return sa.cache.KeyFor(path, params)
}

// HevyAdapter adapts the unified cache to work with Hevy's expected interface
type HevyAdapter struct {
	cache Cache
}

// NewHevyAdapter creates a new adapter for Hevy cache interface
func NewHevyAdapter(cache Cache) *HevyAdapter {
	return &HevyAdapter{cache: cache}
}

// Read implements the Hevy Cache interface
func (ha *HevyAdapter) Read(key string, ttl time.Duration) (body []byte, etag string, ok bool) {
	entry, exists := ha.cache.Read(key, ttl)
	if !exists || entry == nil {
		return nil, "", false
	}

	return []byte(entry.Body), entry.ETag, true
}

// Write implements the Hevy Cache interface
func (ha *HevyAdapter) Write(key string, body []byte, etag string) {
	entry := &Entry{
		ETag: etag,
		Body: json.RawMessage(body),
	}
	// Ignore error for Hevy compatibility (original interface doesn't return error)
	_ = ha.cache.Write(key, entry)
}

// ETag implements the Hevy Cache interface
func (ha *HevyAdapter) ETag(key string) string {
	return ha.cache.GetETag(key)
}
