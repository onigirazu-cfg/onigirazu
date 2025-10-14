package secrets

import (
	"context"
	"testing"
	"time"
)

func TestSecretCache(t *testing.T) {
	cache := NewSecretCache(100 * time.Millisecond)

	// Test Set and Get
	cache.Set("test-key", "test-value")

	value, found := cache.Get("test-key")
	if !found {
		t.Error("Expected to find cached value")
	}
	if value != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", value)
	}

	// Test expiration
	time.Sleep(150 * time.Millisecond)

	_, found = cache.Get("test-key")
	if found {
		t.Error("Expected cached value to be expired")
	}

	// Test Clear
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Clear()

	_, found = cache.Get("key1")
	if found {
		t.Error("Expected cache to be cleared")
	}
}

func TestBitwardenClient_ExtractField(t *testing.T) {
	client := &BitwardenClient{}

	item := &BitwardenItem{
		Name: "test-item",
		Login: &Login{
			Username: "testuser",
			Password: "testpass",
			Totp:     "123456",
		},
		Notes: "Test notes",
		Fields: []Field{
			{Name: "api_key", Value: "secret123", Type: 1},
			{Name: "custom", Value: "custom_value", Type: 0},
		},
	}

	tests := []struct {
		field    string
		expected string
		hasError bool
	}{
		{"username", "testuser", false},
		{"user", "testuser", false},
		{"password", "testpass", false},
		{"pass", "testpass", false},
		{"totp", "123456", false},
		{"notes", "Test notes", false},
		{"note", "Test notes", false},
		{"api_key", "secret123", false},
		{"custom", "custom_value", false},
		{"nonexistent", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			value, err := client.extractField(item, tt.field)

			if tt.hasError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if value != tt.expected {
					t.Errorf("Expected '%s', got '%s'", tt.expected, value)
				}
			}
		})
	}
}

func TestNewBitwardenClient_WithSessionToken(t *testing.T) {
	config := map[string]interface{}{
		"session_token": "test-session-token",
		"cache_ttl":     300,
	}

	client, err := NewBitwardenClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.sessionToken != "test-session-token" {
		t.Errorf("Expected session token 'test-session-token', got '%s'", client.sessionToken)
	}

	if client.server != "https://vault.bitwarden.com" {
		t.Errorf("Expected default server, got '%s'", client.server)
	}

	if client.cache == nil {
		t.Error("Expected cache to be initialized")
	}
}

func TestNewBitwardenClient_CustomServer(t *testing.T) {
	config := map[string]interface{}{
		"server":        "https://vault.example.com",
		"session_token": "test-token",
	}

	client, err := NewBitwardenClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.server != "https://vault.example.com" {
		t.Errorf("Expected custom server, got '%s'", client.server)
	}
}

func TestBitwardenClient_Name(t *testing.T) {
	client := &BitwardenClient{}

	if client.Name() != "bitwarden" {
		t.Errorf("Expected 'bitwarden', got '%s'", client.Name())
	}
}

func TestBitwardenClient_Close(t *testing.T) {
	config := map[string]interface{}{
		"session_token": "test-token",
	}

	client, err := NewBitwardenClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Add some cached data
	client.cache.Set("test", "value")

	// Close should clear cache
	err = client.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}

	// Verify cache is cleared
	_, found := client.cache.Get("test")
	if found {
		t.Error("Expected cache to be cleared after Close")
	}
}

func TestProviderError(t *testing.T) {
	err := &ProviderError{
		Provider: "bitwarden",
		Message:  "test error",
	}

	expected := "bitwarden: test error"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}

	// Test with wrapped error
	innerErr := context.DeadlineExceeded
	err = &ProviderError{
		Provider: "bitwarden",
		Message:  "timeout",
		Err:      innerErr,
	}

	if err.Unwrap() != innerErr {
		t.Error("Expected Unwrap to return inner error")
	}
}

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name      string
		config    ProviderConfig
		expectErr bool
	}{
		{
			name: "bitwarden provider",
			config: ProviderConfig{
				Type: "bitwarden",
				Config: map[string]interface{}{
					"session_token": "test-token",
				},
			},
			expectErr: false,
		},
		{
			name: "unsupported provider",
			config: ProviderConfig{
				Type:   "unknown",
				Config: map[string]interface{}{},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.config)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if provider == nil {
					t.Error("Expected provider to be created")
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkSecretCache_Set(b *testing.B) {
	cache := NewSecretCache(5 * time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("key", "value")
	}
}

func BenchmarkSecretCache_Get(b *testing.B) {
	cache := NewSecretCache(5 * time.Minute)
	cache.Set("key", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("key")
	}
}

func BenchmarkExtractField(b *testing.B) {
	client := &BitwardenClient{}
	item := &BitwardenItem{
		Login: &Login{
			Username: "testuser",
			Password: "testpass",
		},
		Fields: []Field{
			{Name: "field1", Value: "value1"},
			{Name: "field2", Value: "value2"},
			{Name: "field3", Value: "value3"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.extractField(item, "password")
	}
}

// TestBitwardenClient_GetSecret tests the GetSecret method
func TestBitwardenClient_GetSecret(t *testing.T) {
	t.Run("not authenticated", func(t *testing.T) {
		// Create client directly without authentication
		client := &BitwardenClient{
			cache: NewSecretCache(5 * time.Minute),
		}

		ctx := context.Background()
		_, err := client.GetSecret(ctx, "test-item", "password")
		if err == nil {
			t.Error("Expected error for unauthenticated client")
		}

		provErr, ok := err.(*ProviderError)
		if !ok {
			t.Errorf("Expected ProviderError, got %T", err)
		}
		if provErr.Provider != "bitwarden" {
			t.Errorf("Expected provider 'bitwarden', got '%s'", provErr.Provider)
		}
		if provErr.Message != "not authenticated" {
			t.Errorf("Expected message 'not authenticated', got '%s'", provErr.Message)
		}
	})

	t.Run("cache hit", func(t *testing.T) {
		config := map[string]interface{}{
			"session_token": "test-token",
			"cache_ttl":     300,
		}
		client, err := NewBitwardenClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		// Pre-populate cache
		cacheKey := "test-item:password"
		client.cache.Set(cacheKey, "cached-password")

		ctx := context.Background()
		value, err := client.GetSecret(ctx, "test-item", "password")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if value != "cached-password" {
			t.Errorf("Expected 'cached-password', got '%s'", value)
		}
	})
}

// TestBitwardenClient_ListSecrets tests the ListSecrets method
func TestBitwardenClient_ListSecrets(t *testing.T) {
	t.Run("not authenticated", func(t *testing.T) {
		// Create client directly without authentication
		client := &BitwardenClient{
			cache: NewSecretCache(5 * time.Minute),
		}

		ctx := context.Background()
		_, err := client.ListSecrets(ctx, "")
		if err == nil {
			t.Error("Expected error for unauthenticated client")
		}

		provErr, ok := err.(*ProviderError)
		if !ok {
			t.Errorf("Expected ProviderError, got %T", err)
		}
		if provErr.Provider != "bitwarden" {
			t.Errorf("Expected provider 'bitwarden', got '%s'", provErr.Provider)
		}
		if provErr.Message != "not authenticated" {
			t.Errorf("Expected message 'not authenticated', got '%s'", provErr.Message)
		}
	})
}

// TestBitwardenClient_Authenticate tests the authenticate method
func TestBitwardenClient_Authenticate(t *testing.T) {
	t.Run("with session token in config", func(t *testing.T) {
		config := map[string]interface{}{
			"session_token": "test-session-token",
		}
		client := &BitwardenClient{
			cache: NewSecretCache(5 * time.Minute),
		}

		err := client.authenticate(config)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if client.sessionToken != "test-session-token" {
			t.Errorf("Expected session token 'test-session-token', got '%s'", client.sessionToken)
		}
	})

	t.Run("missing credentials", func(t *testing.T) {
		config := map[string]interface{}{}
		client := &BitwardenClient{
			cache: NewSecretCache(5 * time.Minute),
		}

		err := client.authenticate(config)
		if err == nil {
			t.Error("Expected error for missing credentials")
		}
	})
}

// TestProviderError_Error tests the Error method with wrapped error
func TestProviderError_Error(t *testing.T) {
	t.Run("without wrapped error", func(t *testing.T) {
		err := &ProviderError{
			Provider: "test",
			Message:  "test message",
		}
		expected := "test: test message"
		if err.Error() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("with wrapped error", func(t *testing.T) {
		innerErr := context.Canceled
		err := &ProviderError{
			Provider: "test",
			Message:  "operation canceled",
			Err:      innerErr,
		}
		expected := "test: operation canceled: context canceled"
		if err.Error() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, err.Error())
		}
	})
}

// TestNewProvider_VaultProvider tests creating a Vault provider
func TestNewProvider_VaultProvider(t *testing.T) {
	config := ProviderConfig{
		Type: "vault",
		Config: map[string]interface{}{
			"address": "https://vault.example.com",
			"token":   "test-token",
		},
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if provider == nil {
		t.Error("Expected provider to be created")
	}
	if provider.Name() != "vault" {
		t.Errorf("Expected provider name 'vault', got '%s'", provider.Name())
	}
}
