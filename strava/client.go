package strava

import (
	"briangreenhill/coachgpt/cache"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"time"
)

const (
	AuthBase    = "https://www.strava.com/oauth/authorize"
	TokenURL    = "https://www.strava.com/oauth/token"
	APIBase     = "https://www.strava.com/api/v3"
	TokenFile   = "strava_token.json"
	RedirectURI = "http://127.0.0.1:8723/cb"
)

// Client represents a Strava API client
type Client struct {
	ClientID     string
	ClientSecret string
	NoCache      bool
	Cache        cache.Cache
}

// Tokens represents OAuth2 tokens from Strava
type Tokens struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	ExpiresAt    int64  `json:"expires_at"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// NewClient creates a new Strava client
func NewClient(clientID, clientSecret string) *Client {
	return &Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		NoCache:      os.Getenv("STRAVA_NOCACHE") == "1",
	}
}

// NewClientWithCache creates a new Strava client with custom cache implementation
func NewClientWithCache(clientID, clientSecret string, cacheImpl cache.Cache) *Client {
	return &Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		NoCache:      os.Getenv("STRAVA_NOCACHE") == "1",
		Cache:        cacheImpl,
	}
}

// homeFile returns a path to a file in the user's home directory
func homeFile(name string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, name), nil
}

// LoadTokens loads OAuth tokens from the home directory
func (c *Client) LoadTokens() (*Tokens, error) {
	path, err := homeFile(TokenFile)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t Tokens
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// SaveTokens saves OAuth tokens to the home directory
func (c *Client) SaveTokens(t *Tokens) error {
	path, err := homeFile(TokenFile)
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(t, "", "  ")
	return os.WriteFile(path, b, 0600)
}

// EnsureTokens ensures we have valid OAuth tokens, performing OAuth flow if needed
func (c *Client) EnsureTokens() (string, error) {
	// 1) Try to load tokens
	tok, _ := c.LoadTokens()
	now := time.Now().Unix()

	// 2) If we have tokens, refresh if needed; otherwise return
	if tok != nil && tok.RefreshToken != "" {
		// If expiring in < 2 min, refresh
		if tok.ExpiresAt-now < 120 {
			form := url.Values{
				"client_id":     {c.ClientID},
				"client_secret": {c.ClientSecret},
				"grant_type":    {"refresh_token"},
				"refresh_token": {tok.RefreshToken},
			}
			resp, err := http.PostForm(TokenURL, form)
			if err != nil {
				return "", fmt.Errorf("refresh token failed: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return "", fmt.Errorf("refresh token failed: %s", resp.Status)
			}
			var nt Tokens
			if err := json.NewDecoder(resp.Body).Decode(&nt); err != nil {
				return "", fmt.Errorf("decode refresh token failed: %v", err)
			}
			// update and persist
			*tok = nt
			if err := c.SaveTokens(tok); err != nil {
				return "", fmt.Errorf("save token failed: %v", err)
			}
		}
		// Either refreshed or still valid
		return tok.AccessToken, nil
	}

	// 3) No tokens yet → perform OAuth on localhost:8723
	return c.performOAuth()
}

// performOAuth performs the OAuth flow to get initial tokens
func (c *Client) performOAuth() (string, error) {
	type result struct {
		code string
		err  error
	}
	resCh := make(chan result, 1)

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    "127.0.0.1:8723",
		Handler: mux,
	}

	mux.HandleFunc("/cb", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		code := q.Get("code")
		if code == "" {
			http.Error(w, "no code in query", http.StatusBadRequest)
			resCh <- result{"", fmt.Errorf("no code")}
			return
		}
		fmt.Fprintln(w, "Authorized. You can close this window.")
		// return code to main goroutine then shut server down
		resCh <- result{code: code}
		go func() { _ = srv.Shutdown(context.Background()) }()
	})

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return "", fmt.Errorf("listen %s: %v", srv.Addr, err)
	}
	go func() {
		// Serve returns http.ErrServerClosed on Shutdown — that's fine
		_ = srv.Serve(ln)
	}()

	// Build the authorize URL
	authURL := fmt.Sprintf(
		"%s?client_id=%s&response_type=code&redirect_uri=%s&scope=%s&approval_prompt=auto",
		AuthBase,
		url.QueryEscape(c.ClientID),
		url.QueryEscape(RedirectURI),
		url.QueryEscape("read,activity:read_all"),
	)
	fmt.Println("Open in browser:", authURL)
	if err := openBrowser(authURL); err != nil {
		fmt.Println("If the browser didn't open automatically, copy/paste the URL above.")
	}

	// Wait for callback
	res := <-resCh
	if res.err != nil || res.code == "" {
		return "", fmt.Errorf("OAuth failed: %v", res.err)
	}

	// Exchange code for tokens
	form := url.Values{
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
		"code":          {res.code},
		"grant_type":    {"authorization_code"},
	}
	resp, err := http.PostForm(TokenURL, form)
	if err != nil {
		return "", fmt.Errorf("token exchange failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token exchange failed: %s", resp.Status)
	}
	var nt Tokens
	if err := json.NewDecoder(resp.Body).Decode(&nt); err != nil {
		return "", fmt.Errorf("decode tokens failed: %v", err)
	}
	if err := c.SaveTokens(&nt); err != nil {
		return "", fmt.Errorf("save token failed: %v", err)
	}
	return nt.AccessToken, nil
}

// openBrowser opens a URL in the default browser
func openBrowser(u string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", u).Start()
	case "windows":
		// this is the most reliable on modern Windows
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	case "darwin":
		return exec.Command("open", u).Start()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
