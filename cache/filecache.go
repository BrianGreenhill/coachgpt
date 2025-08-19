package cache

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileCache implements the Cache interface using filesystem storage
type FileCache struct {
	dir string
}

// NewFileCache creates a new file-based cache in the specified subdirectory
// If subdir is empty, uses a default cache directory
func NewFileCache(subdir string) (*FileCache, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	baseDir := filepath.Join(usr.HomeDir, ".coach_cache")
	if subdir != "" {
		baseDir = filepath.Join(baseDir, subdir)
	}

	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return nil, err
	}

	return &FileCache{dir: baseDir}, nil
}

// NewStravaCache creates a cache specifically for Strava API calls
func NewStravaCache() (*FileCache, error) {
	return NewFileCache("strava")
}

// NewHevyCache creates a cache specifically for Hevy API calls
func NewHevyCache() (*FileCache, error) {
	return NewFileCache("hevy")
}

// Read implements Reader interface
func (fc *FileCache) Read(key string, maxAge time.Duration) (*Entry, bool) {
	path := fc.path(key)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}

	// Check if expired
	if maxAge > 0 && time.Since(entry.FetchedAt) > maxAge {
		return &entry, false // Return entry but mark as expired
	}

	return &entry, true
}

// Write implements Writer interface
func (fc *FileCache) Write(key string, entry *Entry) error {
	path := fc.path(key)
	entry.FetchedAt = time.Now()

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	// Write to temporary file first, then rename (atomic operation)
	tmpPath := path + fmt.Sprintf(".tmp.%d", rand.Int())
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

// GetETag implements ETagger interface
func (fc *FileCache) GetETag(key string) string {
	entry, exists := fc.Read(key, 0) // Read without TTL check
	if !exists || entry == nil {
		return ""
	}
	return entry.ETag
}

// KeyFor implements KeyGenerator interface
func (fc *FileCache) KeyFor(path string, params map[string]string) string {
	// Build stable key from path + sorted params
	var parts []string
	for k, v := range params {
		parts = append(parts, k+"="+v)
	}
	sort.Strings(parts)

	// Clean path for filename
	cleanPath := strings.ReplaceAll(path, "/", "_")

	if len(parts) > 0 {
		key := fmt.Sprintf("%s__%s", cleanPath, strings.Join(parts, "__"))
		return fc.sanitizeKey(key) + ".json"
	}

	return fc.sanitizeKey(cleanPath) + ".json"
}

// path generates the full filesystem path for a cache key
func (fc *FileCache) path(key string) string {
	return filepath.Join(fc.dir, key)
}

// sanitizeKey ensures the key is safe for use as a filename
func (fc *FileCache) sanitizeKey(key string) string {
	// For very long keys, use hash to avoid filesystem limits
	if len(key) > 200 {
		hash := md5.Sum([]byte(key))
		return fmt.Sprintf("hash_%x", hash)
	}

	// Replace unsafe characters
	unsafe := []string{":", "?", "&", "=", "#", "<", ">", "|", "*", "\""}
	result := key
	for _, char := range unsafe {
		result = strings.ReplaceAll(result, char, "_")
	}

	return result
}
