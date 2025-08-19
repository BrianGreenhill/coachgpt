// Package plugins defines the common interface for fitness data integrations
package plugins

import "context"

// Plugin defines the minimal interface that all fitness data providers must implement
type Plugin interface {
	// Name returns the name of the plugin (e.g., "strava", "hevy")
	Name() string

	// GetLatest retrieves and displays the most recent workout
	GetLatest(ctx context.Context) (string, error)

	// Get retrieves and displays a specific workout by ID
	Get(ctx context.Context, id string) (string, error)
}

// Registry manages available fitness data plugins
type Registry struct {
	plugins map[string]Plugin
}

// NewRegistry creates a new plugin registry
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]Plugin),
	}
}

// Register adds a plugin to the registry
func (r *Registry) Register(plugin Plugin) {
	r.plugins[plugin.Name()] = plugin
}

// Get retrieves a plugin by name
func (r *Registry) GetPlugin(name string) (Plugin, bool) {
	plugin, exists := r.plugins[name]
	return plugin, exists
}

// List returns all registered plugin names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	return names
}
