package hevy

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"time"
)

type Cache interface {
	Read(key string, ttl time.Duration) (body []byte, etag string, ok bool)
	Write(key string, body []byte, etag string)
	ETag(key string) string
}

type fileCache struct {
	dir string
}

type cacheEntry struct {
	ETag      string    `json:"etag"`
	FetchedAt time.Time `json:"fetched_at"`
	Body      []byte    `json:"body"`
}

func NewFileCache(subdir string) (Cache, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(usr.HomeDir, ".hevy_cache")
	if subdir != "" {
		dir = filepath.Join(dir, subdir)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &fileCache{dir: dir}, nil
}

func (fc *fileCache) path(key string) string {
	// key is a URL; make it file-safe
	name := key
	for range []string{"/", ":", "?", "&", "=", "#"} {
		name = filepath.Base(name)
	}
	// simplest: hash would be better, but basename works if unique
	return filepath.Join(fc.dir, sanitize(key)+".json")
}
func sanitize(s string) string {
	// very simple sanitizer; replace path separators and query chars
	repl := map[rune]rune{'/': '_', '?': '_', '&': '_', '=': '_', ':': '_', '#': '_'}
	out := []rune{}
	for _, r := range s {
		if v, ok := repl[r]; ok {
			out = append(out, v)
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

func (fc *fileCache) Read(key string, ttl time.Duration) ([]byte, string, bool) {
	fp := fc.path(key)
	b, err := os.ReadFile(fp)
	if err != nil {
		return nil, "", false
	}
	var ce cacheEntry
	if err := json.Unmarshal(b, &ce); err != nil {
		return nil, "", false
	}
	if ttl > 0 && time.Since(ce.FetchedAt) > ttl {
		return nil, "", false
	}
	return ce.Body, ce.ETag, true
}

func (fc *fileCache) Write(key string, body []byte, etag string) {
	fp := fc.path(key)
	ce := cacheEntry{ETag: etag, FetchedAt: time.Now(), Body: body}
	b, _ := json.MarshalIndent(&ce, "", "  ")
	_ = os.WriteFile(fp, b, 0o600)
}

func (fc *fileCache) ETag(key string) string {
	_, etag, ok := fc.Read(key, 0)
	if !ok {
		return ""
	}
	return etag
}
