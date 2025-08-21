// Package providers contains workout data provider implementations
package providers

import "context"

// Provider defines the interface that all fitness data providers must implement
type Provider interface {
	// Name returns the name of the provider (e.g., "strava", "hevy")
	Name() string

	// GetLatest retrieves and displays the most recent workout
	GetLatest(ctx context.Context) (string, error)

	// Get retrieves and displays a specific workout by ID
	Get(ctx context.Context, id string) (string, error)
}

// Registry manages available fitness data providers
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry
func (r *Registry) Register(provider Provider) {
	r.providers[provider.Name()] = provider
}

// Get retrieves a provider by name
func (r *Registry) Get(name string) (Provider, bool) {
	provider, exists := r.providers[name]
	return provider, exists
}

// List returns all registered provider names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}
