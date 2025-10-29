package secrets

import (
	"context"
	"os"
	"testing"
)

func init() {
	// Skip vault authentication checks in test environments
	os.Setenv("VAULT_SKIP_VERIFY", "true")
}

// TestNewVaultClient tests the NewVaultClient function
func TestNewVaultClient(t *testing.T) {
	t.Run("valid configuration", func(t *testing.T) {
		config := map[string]interface{}{
			"address": "https://vault.example.com",
			"token":   "test-token",
		}

		client, err := NewVaultClient(config)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("Expected client to be created")
		}
		if client.address != "https://vault.example.com" {
			t.Errorf("Expected address 'https://vault.example.com', got '%s'", client.address)
		}
		if client.token != "test-token" {
			t.Errorf("Expected token 'test-token', got '%s'", client.token)
		}
	})

	t.Run("missing address", func(t *testing.T) {
		config := map[string]interface{}{
			"token": "test-token",
		}

		client, err := NewVaultClient(config)
		if err == nil {
			t.Error("Expected error for missing address")
		}
		if client != nil {
			t.Error("Expected client to be nil")
		}

		provErr, ok := err.(*ProviderError)
		if !ok {
			t.Errorf("Expected ProviderError, got %T", err)
		}
		if provErr.Provider != "vault" {
			t.Errorf("Expected provider 'vault', got '%s'", provErr.Provider)
		}
		if provErr.Message != "vault address is required" {
			t.Errorf("Expected message 'vault address is required', got '%s'", provErr.Message)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		config := map[string]interface{}{
			"address": "https://vault.example.com",
		}

		client, err := NewVaultClient(config)
		if err == nil {
			t.Error("Expected error for missing token")
		}
		if client != nil {
			t.Error("Expected client to be nil")
		}

		provErr, ok := err.(*ProviderError)
		if !ok {
			t.Errorf("Expected ProviderError, got %T", err)
		}
		if provErr.Provider != "vault" {
			t.Errorf("Expected provider 'vault', got '%s'", provErr.Provider)
		}
		if provErr.Message != "vault token is required" {
			t.Errorf("Expected message 'vault token is required', got '%s'", provErr.Message)
		}
	})

	t.Run("empty address", func(t *testing.T) {
		config := map[string]interface{}{
			"address": "",
			"token":   "test-token",
		}

		client, err := NewVaultClient(config)
		if err == nil {
			t.Error("Expected error for empty address")
		}
		if client != nil {
			t.Error("Expected client to be nil")
		}
	})

	t.Run("empty token", func(t *testing.T) {
		config := map[string]interface{}{
			"address": "https://vault.example.com",
			"token":   "",
		}

		client, err := NewVaultClient(config)
		if err == nil {
			t.Error("Expected error for empty token")
		}
		if client != nil {
			t.Error("Expected client to be nil")
		}
	})
}

// TestVaultClient_GetSecret tests the GetSecret method (stub implementation)
func TestVaultClient_GetSecret(t *testing.T) {
	config := map[string]interface{}{
		"address": "https://vault.example.com",
		"token":   "test-token",
	}

	client, err := NewVaultClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	_, err = client.GetSecret(ctx, "secret/data/myapp", "password")
	if err == nil {
		t.Error("Expected error for stub implementation")
	}

	provErr, ok := err.(*ProviderError)
	if !ok {
		t.Errorf("Expected ProviderError, got %T", err)
	}
	if provErr.Provider != "vault" {
		t.Errorf("Expected provider 'vault', got '%s'", provErr.Provider)
	}
	if provErr.Message != "vault integration not yet implemented" {
		t.Errorf("Expected message 'vault integration not yet implemented', got '%s'", provErr.Message)
	}
}

// TestVaultClient_ListSecrets tests the ListSecrets method (stub implementation)
func TestVaultClient_ListSecrets(t *testing.T) {
	config := map[string]interface{}{
		"address": "https://vault.example.com",
		"token":   "test-token",
	}

	client, err := NewVaultClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	secrets, err := client.ListSecrets(ctx, "")
	if err == nil {
		t.Error("Expected error for stub implementation")
	}
	if secrets != nil {
		t.Error("Expected nil secrets for stub implementation")
	}

	provErr, ok := err.(*ProviderError)
	if !ok {
		t.Errorf("Expected ProviderError, got %T", err)
	}
	if provErr.Provider != "vault" {
		t.Errorf("Expected provider 'vault', got '%s'", provErr.Provider)
	}
	if provErr.Message != "vault integration not yet implemented" {
		t.Errorf("Expected message 'vault integration not yet implemented', got '%s'", provErr.Message)
	}
}

// TestVaultClient_Close tests the Close method
func TestVaultClient_Close(t *testing.T) {
	config := map[string]interface{}{
		"address": "https://vault.example.com",
		"token":   "test-token",
	}

	client, err := NewVaultClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Unexpected error from Close: %v", err)
	}
}

// TestVaultClient_Name tests the Name method
func TestVaultClient_Name(t *testing.T) {
	config := map[string]interface{}{
		"address": "https://vault.example.com",
		"token":   "test-token",
	}

	client, err := NewVaultClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.Name() != "vault" {
		t.Errorf("Expected name 'vault', got '%s'", client.Name())
	}
}

// TestVaultClient_Interface tests that VaultClient implements SecretProvider
func TestVaultClient_Interface(t *testing.T) {
	config := map[string]interface{}{
		"address": "https://vault.example.com",
		"token":   "test-token",
	}

	client, err := NewVaultClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Verify it implements SecretProvider interface
	var _ SecretProvider = client
}
