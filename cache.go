package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type cacheEntry struct {
	FetchedAt time.Time       `json:"fetched_at"`
	ETag      string          `json:"etag,omitempty"`
	Body      json.RawMessage `json:"body"`
}

func cacheDir() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(u.HomeDir, ".strava_cache")
	_ = os.MkdirAll(dir, 0o700)
	return dir, nil
}

func cachePath(name string) (string, error) {
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

func readCache(name string, maxAge time.Duration) (*cacheEntry, error) {
	fp, err := cachePath(name)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	var ce cacheEntry
	if err := json.Unmarshal(b, &ce); err != nil {
		return nil, err
	}
	if maxAge > 0 && time.Since(ce.FetchedAt) > maxAge {
		return nil, errors.New("cache stale")
	}
	return &ce, nil
}

func writeCache(name string, ce *cacheEntry) error {
	fp, err := cachePath(name)
	if err != nil {
		return err
	}
	ce.FetchedAt = time.Now()
	b, _ := json.MarshalIndent(ce, "", "  ")
	tmp := fp + fmt.Sprintf(".tmp.%d", rand.Int())
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, fp)
}

// Build a stable cache key from path + params (sorted)
func keyFor(path string, params map[string]string) string {
	var parts []string
	for k, v := range params {
		parts = append(parts, k+"="+v)
	}
	sort.Strings(parts)
	p := strings.ReplaceAll(path, "/", "_")
	if len(parts) > 0 {
		return fmt.Sprintf("%s__%s.json", p, strings.Join(parts, "__"))
	}
	return fmt.Sprintf("%s.json", p)
}
