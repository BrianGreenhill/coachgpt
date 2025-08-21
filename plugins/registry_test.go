package plugins

import (
	"context"
	"testing"
)

// mockPlugin is a test implementation of the Plugin interface
type mockPlugin struct {
	name string
}

func (m *mockPlugin) Name() string {
	return m.name
}

func (m *mockPlugin) GetLatest(ctx context.Context) (string, error) {
	return "latest workout from " + m.name, nil
}

func (m *mockPlugin) Get(ctx context.Context, id string) (string, error) {
	return "workout " + id + " from " + m.name, nil
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry should not return nil")
	}

	plugins := registry.List()
	if len(plugins) != 0 {
		t.Errorf("New registry should be empty, got %d plugins: %v", len(plugins), plugins)
	}
}

func TestRegisterAndGetPlugin(t *testing.T) {
	registry := NewRegistry()

	// Create mock plugins
	stravaPlugin := &mockPlugin{name: "strava"}
	hevyPlugin := &mockPlugin{name: "hevy"}

	// Register plugins
	registry.Register(stravaPlugin)
	registry.Register(hevyPlugin)

	// Test List()
	plugins := registry.List()
	if len(plugins) != 2 {
		t.Errorf("Expected 2 plugins, got %d: %v", len(plugins), plugins)
	}

	// Check that both plugin names are present
	pluginNames := make(map[string]bool)
	for _, name := range plugins {
		pluginNames[name] = true
	}
	if !pluginNames["strava"] {
		t.Error("Expected strava plugin to be listed")
	}
	if !pluginNames["hevy"] {
		t.Error("Expected hevy plugin to be listed")
	}

	// Test GetPlugin()
	plugin, exists := registry.GetPlugin("strava")
	if !exists {
		t.Error("Strava plugin should exist")
	}
	if plugin.Name() != "strava" {
		t.Errorf("Expected plugin name 'strava', got '%s'", plugin.Name())
	}

	plugin, exists = registry.GetPlugin("hevy")
	if !exists {
		t.Error("Hevy plugin should exist")
	}
	if plugin.Name() != "hevy" {
		t.Errorf("Expected plugin name 'hevy', got '%s'", plugin.Name())
	}

	// Test non-existent plugin
	_, exists = registry.GetPlugin("nonexistent")
	if exists {
		t.Error("Non-existent plugin should not exist")
	}
}

func TestPluginInterface(t *testing.T) {
	plugin := &mockPlugin{name: "test"}

	// Test Name()
	if plugin.Name() != "test" {
		t.Errorf("Expected name 'test', got '%s'", plugin.Name())
	}

	// Test GetLatest()
	ctx := context.Background()
	result, err := plugin.GetLatest(ctx)
	if err != nil {
		t.Errorf("GetLatest should not return error: %v", err)
	}
	if result != "latest workout from test" {
		t.Errorf("Expected 'latest workout from test', got '%s'", result)
	}

	// Test Get()
	result, err = plugin.Get(ctx, "123")
	if err != nil {
		t.Errorf("Get should not return error: %v", err)
	}
	if result != "workout 123 from test" {
		t.Errorf("Expected 'workout 123 from test', got '%s'", result)
	}
}

func TestRegistryOverwrite(t *testing.T) {
	registry := NewRegistry()

	// Register a plugin
	plugin1 := &mockPlugin{name: "test"}
	registry.Register(plugin1)

	// Register another plugin with same name
	plugin2 := &mockPlugin{name: "test"}
	registry.Register(plugin2)

	// Should only have one plugin
	plugins := registry.List()
	if len(plugins) != 1 {
		t.Errorf("Expected 1 plugin after overwrite, got %d", len(plugins))
	}

	// Should get the second plugin
	plugin, exists := registry.GetPlugin("test")
	if !exists {
		t.Error("Plugin should exist")
	}
	if plugin != plugin2 {
		t.Error("Should get the second registered plugin")
	}
}
