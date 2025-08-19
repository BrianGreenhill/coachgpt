// Package cache provides a unified caching interface for HTTP responses
// with support for ETag validation and TTL-based expiration.
package cache

import (
	"encoding/json"
	"time"
)

// Entry represents a cached entry with metadata
type Entry struct {
	ETag      string          `json:"etag,omitempty"`
	FetchedAt time.Time       `json:"fetched_at"`
	Body      json.RawMessage `json:"body"`
}

// Reader defines the interface for reading cache entries
type Reader interface {
	// Read retrieves a cache entry by key with TTL validation
	// Returns the entry and true if found and not expired, false otherwise
	Read(key string, maxAge time.Duration) (*Entry, bool)
}

// Writer defines the interface for writing cache entries
type Writer interface {
	// Write stores a cache entry with the given key
	Write(key string, entry *Entry) error
}

// ReadWriter combines both cache operations
type ReadWriter interface {
	Reader
	Writer
}

// ETagger provides ETag support for conditional requests
type ETagger interface {
	// GetETag returns the ETag for a given key, empty string if not found
	GetETag(key string) string
}

// KeyGenerator generates cache keys from request parameters
type KeyGenerator interface {
	// KeyFor generates a stable cache key from path and parameters
	KeyFor(path string, params map[string]string) string
}

// Cache is the main interface that combines all cache operations
type Cache interface {
	ReadWriter
	ETagger
	KeyGenerator
}
