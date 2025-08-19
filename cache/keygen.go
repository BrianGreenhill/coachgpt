package cache

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"strings"
)

// KeyGenerators provides different key generation strategies
type KeyGenerators struct{}

// URLToKey converts a URL to a cache key suitable for Hevy-style caching
func (kg *KeyGenerators) URLToKey(requestURL string) string {
	// Parse URL to extract components
	u, err := url.Parse(requestURL)
	if err != nil {
		// Fallback to sanitized URL
		return kg.sanitizeForFilename(requestURL)
	}

	// Build key from host + path + query
	parts := []string{u.Host}
	if u.Path != "" {
		parts = append(parts, strings.Trim(u.Path, "/"))
	}
	if u.RawQuery != "" {
		// Hash query params if too long
		if len(u.RawQuery) > 100 {
			hash := md5.Sum([]byte(u.RawQuery))
			parts = append(parts, fmt.Sprintf("q_%x", hash))
		} else {
			parts = append(parts, u.RawQuery)
		}
	}

	key := strings.Join(parts, "_")
	return kg.sanitizeForFilename(key)
}

// PathParamsToKey converts path and params to a cache key (Strava-style)
func (kg *KeyGenerators) PathParamsToKey(path string, params map[string]string) string {
	fc := &FileCache{} // Use FileCache's KeyFor method
	return fc.KeyFor(path, params)
}

// sanitizeForFilename makes a string safe for use as a filename
func (kg *KeyGenerators) sanitizeForFilename(s string) string {
	// Replace characters that are problematic for filenames
	replacements := map[string]string{
		"/":  "_",
		"\\": "_",
		":":  "_",
		"*":  "_",
		"?":  "_",
		"\"": "_",
		"<":  "_",
		">":  "_",
		"|":  "_",
		"#":  "_",
		"&":  "_",
		"=":  "_",
		" ":  "_",
	}

	result := s
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}

	// Limit length and use hash for very long keys
	if len(result) > 200 {
		hash := md5.Sum([]byte(s))
		return fmt.Sprintf("long_%x", hash)
	}

	// Ensure it has an extension
	if !strings.HasSuffix(result, ".json") {
		result += ".json"
	}

	return result
}

// DefaultKeyGenerator provides a shared key generator instance
var DefaultKeyGenerator = &KeyGenerators{}
