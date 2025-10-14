package secrets

import (
	"context"
	"testing"
)

// MockSecretProvider is a mock implementation for testing
type MockSecretProvider struct {
	name    string
	secrets map[string]map[string]string
}

func NewMockSecretProvider(name string) *MockSecretProvider {
	return &MockSecretProvider{
		name:    name,
		secrets: make(map[string]map[string]string),
	}
}

func (m *MockSecretProvider) AddSecret(itemName, field, value string) {
	if m.secrets[itemName] == nil {
		m.secrets[itemName] = make(map[string]string)
	}
	m.secrets[itemName][field] = value
}

func (m *MockSecretProvider) GetSecret(ctx context.Context, itemName, field string) (string, error) {
	if item, exists := m.secrets[itemName]; exists {
		if value, ok := item[field]; ok {
			return value, nil
		}
	}
	return "", &ProviderError{
		Provider: m.name,
		Message:  "secret not found",
	}
}

func (m *MockSecretProvider) ListSecrets(ctx context.Context, filter string) ([]string, error) {
	var names []string
	for name := range m.secrets {
		names = append(names, name)
	}
	return names, nil
}

func (m *MockSecretProvider) Close() error {
	return nil
}

func (m *MockSecretProvider) Name() string {
	return m.name
}

func TestTemplateSecretManager_RegisterProvider(t *testing.T) {
	tsm := NewTemplateSecretManager()

	provider := NewMockSecretProvider("test")

	err := tsm.RegisterProvider("test", provider)
	if err != nil {
		t.Errorf("Failed to register provider: %v", err)
	}

	// Try to register again - should fail
	err = tsm.RegisterProvider("test", provider)
	if err == nil {
		t.Error("Expected error when registering duplicate provider")
	}
}

func TestTemplateSecretManager_GetProvider(t *testing.T) {
	tsm := NewTemplateSecretManager()

	provider := NewMockSecretProvider("test")
	tsm.RegisterProvider("test", provider)

	retrieved, err := tsm.GetProvider("test")
	if err != nil {
		t.Errorf("Failed to get provider: %v", err)
	}

	if retrieved.Name() != "test" {
		t.Errorf("Expected provider name 'test', got '%s'", retrieved.Name())
	}

	// Try to get non-existent provider
	_, err = tsm.GetProvider("nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent provider")
	}
}

func TestTemplateSecretManager_CreateBitwardenFunc(t *testing.T) {
	tsm := NewTemplateSecretManager()

	// Create and register mock provider
	provider := NewMockSecretProvider("bitwarden")
	provider.AddSecret("test-item", "password", "secret123")
	tsm.RegisterProvider("bitwarden", provider)

	// Create template function
	bitwardenFunc := tsm.CreateBitwardenFunc()

	// Test successful retrieval
	value, err := bitwardenFunc("test-item", "password")
	if err != nil {
		t.Errorf("Failed to get secret: %v", err)
	}
	if value != "secret123" {
		t.Errorf("Expected 'secret123', got '%s'", value)
	}

	// Test non-existent secret
	_, err = bitwardenFunc("nonexistent", "password")
	if err == nil {
		t.Error("Expected error for non-existent secret")
	}
}

func TestTemplateSecretManager_CreateVaultFunc(t *testing.T) {
	tsm := NewTemplateSecretManager()

	// Create and register mock provider
	provider := NewMockSecretProvider("vault")
	provider.AddSecret("secret/database", "password", "vaultpass123")
	tsm.RegisterProvider("vault", provider)

	// Create template function
	vaultFunc := tsm.CreateVaultFunc()

	// Test successful retrieval
	value, err := vaultFunc("secret/database", "password")
	if err != nil {
		t.Errorf("Failed to get secret: %v", err)
	}
	if value != "vaultpass123" {
		t.Errorf("Expected 'vaultpass123', got '%s'", value)
	}
}

func TestTemplateSecretManager_CreateSecretFunc(t *testing.T) {
	tsm := NewTemplateSecretManager()

	// Register multiple providers
	bwProvider := NewMockSecretProvider("bitwarden")
	bwProvider.AddSecret("item1", "field1", "value1")
	tsm.RegisterProvider("bitwarden", bwProvider)

	vaultProvider := NewMockSecretProvider("vault")
	vaultProvider.AddSecret("item2", "field2", "value2")
	tsm.RegisterProvider("vault", vaultProvider)

	// Create generic secret function
	secretFunc := tsm.CreateSecretFunc()

	// Test Bitwarden
	value, err := secretFunc("bitwarden", "item1", "field1")
	if err != nil {
		t.Errorf("Failed to get bitwarden secret: %v", err)
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got '%s'", value)
	}

	// Test Vault
	value, err = secretFunc("vault", "item2", "field2")
	if err != nil {
		t.Errorf("Failed to get vault secret: %v", err)
	}
	if value != "value2" {
		t.Errorf("Expected 'value2', got '%s'", value)
	}

	// Test non-existent provider
	_, err = secretFunc("nonexistent", "item", "field")
	if err == nil {
		t.Error("Expected error for non-existent provider")
	}
}

func TestTemplateSecretManager_GetTemplateFunctions(t *testing.T) {
	tsm := NewTemplateSecretManager()

	// Register provider
	provider := NewMockSecretProvider("bitwarden")
	provider.AddSecret("test", "password", "secret")
	tsm.RegisterProvider("bitwarden", provider)

	// Get template functions
	funcs := tsm.GetTemplateFunctions()

	// Check all functions are present
	expectedFuncs := []string{"bitwarden", "vault", "secret"}
	for _, name := range expectedFuncs {
		if _, exists := funcs[name]; !exists {
			t.Errorf("Expected function '%s' to be present", name)
		}
	}
}

func TestTemplateSecretManager_Close(t *testing.T) {
	tsm := NewTemplateSecretManager()

	// Register multiple providers
	provider1 := NewMockSecretProvider("provider1")
	provider2 := NewMockSecretProvider("provider2")

	tsm.RegisterProvider("provider1", provider1)
	tsm.RegisterProvider("provider2", provider2)

	// Close should not error
	err := tsm.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

func TestTemplateSecretManager_ConcurrentAccess(t *testing.T) {
	tsm := NewTemplateSecretManager()

	provider := NewMockSecretProvider("bitwarden")
	provider.AddSecret("test", "password", "secret123")
	tsm.RegisterProvider("bitwarden", provider)

	bitwardenFunc := tsm.CreateBitwardenFunc()

	// Test concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := bitwardenFunc("test", "password")
			if err != nil {
				t.Errorf("Concurrent access failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Benchmark tests
func BenchmarkBitwardenFunc(b *testing.B) {
	tsm := NewTemplateSecretManager()

	provider := NewMockSecretProvider("bitwarden")
	provider.AddSecret("test", "password", "secret123")
	tsm.RegisterProvider("bitwarden", provider)

	bitwardenFunc := tsm.CreateBitwardenFunc()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bitwardenFunc("test", "password")
	}
}

func BenchmarkSecretFunc(b *testing.B) {
	tsm := NewTemplateSecretManager()

	provider := NewMockSecretProvider("bitwarden")
	provider.AddSecret("test", "password", "secret123")
	tsm.RegisterProvider("bitwarden", provider)

	secretFunc := tsm.CreateSecretFunc()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = secretFunc("bitwarden", "test", "password")
	}
}
