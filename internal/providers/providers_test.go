package providers

import (
	"bufio"
	"context"
	"testing"
)

// mockProvider is a test implementation of the Provider interface
type mockProvider struct {
	name        string
	description string
	configured  bool
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Description() string {
	if m.description != "" {
		return m.description
	}
	return m.name + " provider"
}

func (m *mockProvider) IsConfigured() bool {
	return m.configured
}

func (m *mockProvider) Setup(reader *bufio.Reader) error {
	// Mock setup - just mark as configured
	m.configured = true
	return nil
}

func (m *mockProvider) ShowConfig() string {
	if m.configured {
		return "✅ " + m.name + ": Configured"
	}
	return "❌ " + m.name + ": Not configured"
}

func (m *mockProvider) GetLatest(ctx context.Context) (string, error) {
	return "latest workout from " + m.name, nil
}

func (m *mockProvider) Get(ctx context.Context, id string) (string, error) {
	return "workout " + id + " from " + m.name, nil
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry should not return nil")
	}

	providers := registry.List()
	if len(providers) != 0 {
		t.Errorf("New registry should be empty, got %d providers: %v", len(providers), providers)
	}
}

func TestRegisterAndGetProvider(t *testing.T) {
	registry := NewRegistry()
	provider := &mockProvider{name: "test"}

	// Register provider
	registry.Register(provider)

	// Check it was registered
	retrieved, exists := registry.Get("test")
	if !exists {
		t.Error("Provider should exist after registration")
	}

	if retrieved != provider {
		t.Error("Retrieved provider should be the same as registered")
	}

	// Check non-existent provider
	_, exists = registry.Get("nonexistent")
	if exists {
		t.Error("Non-existent provider should not exist")
	}

	// Check list
	providers := registry.List()
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}

	if providers[0] != "test" {
		t.Errorf("Expected provider name 'test', got '%s'", providers[0])
	}
}

func TestProviderInterface(t *testing.T) {
	provider := &mockProvider{name: "test"}

	// Test Name()
	if provider.Name() != "test" {
		t.Errorf("Expected name 'test', got '%s'", provider.Name())
	}

	// Test GetLatest()
	ctx := context.Background()
	result, err := provider.GetLatest(ctx)
	if err != nil {
		t.Errorf("GetLatest should not return error: %v", err)
	}
	if result != "latest workout from test" {
		t.Errorf("Expected 'latest workout from test', got '%s'", result)
	}

	// Test Get()
	result, err = provider.Get(ctx, "123")
	if err != nil {
		t.Errorf("Get should not return error: %v", err)
	}
	if result != "workout 123 from test" {
		t.Errorf("Expected 'workout 123 from test', got '%s'", result)
	}
}

func TestRegistryOverwrite(t *testing.T) {
	registry := NewRegistry()
	provider1 := &mockProvider{name: "test"}
	provider2 := &mockProvider{name: "test"}

	// Register first provider
	registry.Register(provider1)
	retrieved, _ := registry.Get("test")
	if retrieved != provider1 {
		t.Error("Should get first provider")
	}

	// Register second provider with same name (should overwrite)
	registry.Register(provider2)
	retrieved, _ = registry.Get("test")
	if retrieved != provider2 {
		t.Error("Should get second provider after overwrite")
	}

	// Should still only have one provider
	providers := registry.List()
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider after overwrite, got %d", len(providers))
	}
}
