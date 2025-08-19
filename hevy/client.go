package hevy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

const DefaultBaseURL = "https://api.hevyapp.com" // adjust if needed

// Cache interface for HTTP response caching with ETag support
type Cache interface {
	Read(key string, ttl time.Duration) (body []byte, etag string, ok bool)
	Write(key string, body []byte, etag string)
	ETag(key string) string
}

type Client struct {
	http    *http.Client
	baseURL *url.URL
	apiKey  string

	cache Cache // optional; nil means no cache
	ttl   time.Duration
}

type Option func(*Client)

func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}
func WithBaseURL(raw string) Option {
	return func(c *Client) {
		if u, err := url.Parse(raw); err == nil {
			c.baseURL = u
		}
	}
}
func WithCache(cache Cache, ttl time.Duration) Option {
	return func(c *Client) { c.cache, c.ttl = cache, ttl }
}

func New(apiKey string, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, errors.New("apiKey required")
	}
	u, _ := url.Parse(DefaultBaseURL)
	c := &Client{
		http:    http.DefaultClient,
		baseURL: u,
		apiKey:  apiKey,
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

func (c *Client) newReq(ctx context.Context, p string, q map[string]string) (*http.Request, string, error) {
	u := *c.baseURL
	u.Path = path.Join(u.Path, p)
	qq := u.Query()
	for k, v := range q {
		qq.Set(k, v)
	}
	u.RawQuery = qq.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	// Auth header — adjust to the header name your API uses
	req.Header.Set("api-key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	return req, u.String(), nil
}

func (c *Client) doJSON(ctx context.Context, p string, q map[string]string, out any) (*string, error) {
	req, cacheKey, err := c.newReq(ctx, p, q)
	if err != nil {
		return nil, err
	}

	// cache read (fresh)
	if c.cache != nil {
		if body, etag, ok := c.cache.Read(cacheKey, c.ttl); ok {
			if err := json.Unmarshal(body, out); err == nil {
				return &etag, nil
			}
		}
		// try revalidate via If-None-Match
		if etag := c.cache.ETag(cacheKey); etag != "" {
			req.Header.Set("If-None-Match", etag)
		}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified: // 304 revalidate
		if c.cache != nil {
			if body, _, ok := c.cache.Read(cacheKey, 0); ok {
				return nil, json.Unmarshal(body, out)
			}
		}
		return nil, fmt.Errorf("304 but no cached body for %s", cacheKey)
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(body, out); err != nil {
			return nil, err
		}
		if c.cache != nil {
			c.cache.Write(cacheKey, body, resp.Header.Get("ETag"))
		}
		etag := resp.Header.Get("ETag")
		return &etag, nil
	default:
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GET %s: %s: %s", p, resp.Status, string(b))
	}
}

// GetWorkouts returns a page of workouts (page starts at 1)
func (c *Client) GetWorkouts(ctx context.Context, page int) (Body, error) {
	if page <= 0 {
		page = 1
	}
	var b Body
	_, err := c.doJSON(ctx, "/v1/workouts", map[string]string{"page": fmt.Sprint(page)}, &b)
	return b, err
}

// GetLatestWorkout returns the first workout from page 1 (if present)
func (c *Client) GetLatestWorkout(ctx context.Context) (*WorkoutJSON, error) {
	b, err := c.GetWorkouts(ctx, 1)
	if err != nil {
		return nil, err
	}
	if len(b.Workouts) == 0 {
		return nil, errors.New("no workouts")
	}
	return &b.Workouts[0], nil
}
