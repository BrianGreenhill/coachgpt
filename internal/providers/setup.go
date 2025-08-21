package providers

import (
	"net/http"

	"github.com/BrianGreenhill/coachgpt/internal/config"
	"github.com/BrianGreenhill/coachgpt/pkg/hevy"
	"github.com/BrianGreenhill/coachgpt/pkg/strava"

	"github.com/gregjones/httpcache"
)

// Setup creates a registry with all configured providers
func Setup(cfg *config.Config) *Registry {
	registry := NewRegistry()

	// Create HTTP client with caching transport
	transport := httpcache.NewMemoryCacheTransport()
	httpClient := &http.Client{Transport: transport}

	// Register Strava provider if configured
	if cfg.HasStrava() {
		stravaClient := strava.NewClientWithHTTP(cfg.Strava.ClientID, cfg.Strava.ClientSecret, httpClient)
		stravaProvider := NewStravaProvider(stravaClient, cfg.Strava.HRMax)
		registry.Register(stravaProvider)
	}

	// Register Hevy provider if configured
	if cfg.HasHevy() {
		if client, err := hevy.New(cfg.Hevy.APIKey, hevy.WithHTTPClient(httpClient)); err == nil {
			hevyProvider := NewHevyProvider(client)
			registry.Register(hevyProvider)
		}
	}

	return registry
}
