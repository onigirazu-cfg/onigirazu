package secrets

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestVaultClientCreation tests creating a Vault client
func TestVaultClientCreation(t *testing.T) {
	// Skip if VAULT_SKIP_VERIFY not set (prevents actual Vault connection attempt)
	os.Setenv("VAULT_SKIP_VERIFY", "true")

	config := map[string]interface{}{
		"address": "http://127.0.0.1:8200",
		"token":   "s.test_token_12345",
	}

	client, err := NewVaultClient(config)
	if err != nil {
		t.Fatalf("failed to create vault client: %v", err)
	}

	if client == nil {
		t.Fatal("vault client is nil")
	}

	if client.Name() != "vault" {
		t.Errorf("expected provider name 'vault', got %s", client.Name())
	}
}

// TestVaultClientMissingAddress tests error when address is missing
func TestVaultClientMissingAddress(t *testing.T) {
	os.Setenv("VAULT_SKIP_VERIFY", "true")

	config := map[string]interface{}{
		"token": "s.test_token_12345",
	}

	_, err := NewVaultClient(config)
	if err == nil {
		t.Fatal("expected error for missing address")
	}

	if provErr, ok := err.(*ProviderError); ok {
		if provErr.Provider != "vault" {
			t.Errorf("expected provider 'vault' in error, got %s", provErr.Provider)
		}
	} else {
		t.Errorf("expected ProviderError, got %T", err)
	}
}

// TestVaultClientMissingToken tests error when token is missing
func TestVaultClientMissingToken(t *testing.T) {
	os.Setenv("VAULT_SKIP_VERIFY", "true")

	config := map[string]interface{}{
		"address": "http://127.0.0.1:8200",
	}

	_, err := NewVaultClient(config)
	if err == nil {
		t.Fatal("expected error for missing token")
	}

	if provErr, ok := err.(*ProviderError); ok {
		if provErr.Provider != "vault" {
			t.Errorf("expected provider 'vault' in error, got %s", provErr.Provider)
		}
	} else {
		t.Errorf("expected ProviderError, got %T", err)
	}
}

// TestVaultClientCacheManagement tests cache operations
func TestVaultClientCacheManagement(t *testing.T) {
	os.Setenv("VAULT_SKIP_VERIFY", "true")

	config := map[string]interface{}{
		"address":   "http://127.0.0.1:8200",
		"token":     "s.test_token_12345",
		"cache_ttl": "5s",
	}

	client, err := NewVaultClient(config)
	if err != nil {
		t.Fatalf("failed to create vault client: %v", err)
	}

	// Test putting in cache
	client.putInCache("test_secret", "test_value")

	// Test getting from cache
	value, found := client.getFromCache("test_secret")
	if !found {
		t.Fatal("expected to find value in cache")
	}

	if value != "test_value" {
		t.Errorf("expected 'test_value', got %s", value)
	}

	// Test cache miss
	_, found = client.getFromCache("nonexistent")
	if found {
		t.Fatal("expected cache miss for nonexistent key")
	}

	// Test clear cache
	client.ClearCache()
	_, found = client.getFromCache("test_secret")
	if found {
		t.Fatal("expected cache miss after clear")
	}
}

// TestVaultClientNamespace tests namespace support
func TestVaultClientNamespace(t *testing.T) {
	os.Setenv("VAULT_SKIP_VERIFY", "true")

	config := map[string]interface{}{
		"address":   "http://127.0.0.1:8200",
		"token":     "s.test_token_12345",
		"namespace": "prod",
	}

	client, err := NewVaultClient(config)
	if err != nil {
		t.Fatalf("failed to create vault client: %v", err)
	}

	if client.namespace != "prod" {
		t.Errorf("expected namespace 'prod', got %s", client.namespace)
	}
}

// TestVaultClientCacheTTL tests custom cache TTL
func TestVaultClientCacheTTL(t *testing.T) {
	os.Setenv("VAULT_SKIP_VERIFY", "true")

	config := map[string]interface{}{
		"address":   "http://127.0.0.1:8200",
		"token":     "s.test_token_12345",
		"cache_ttl": "10s",
	}

	client, err := NewVaultClient(config)
	if err != nil {
		t.Fatalf("failed to create vault client: %v", err)
	}

	if client.cache.ttl != 10*time.Second {
		t.Errorf("expected cache TTL of 10s, got %v", client.cache.ttl)
	}
}

// TestVaultClientClose tests cleanup
func TestVaultClientClose(t *testing.T) {
	os.Setenv("VAULT_SKIP_VERIFY", "true")

	config := map[string]interface{}{
		"address": "http://127.0.0.1:8200",
		"token":   "s.test_token_12345",
	}

	client, err := NewVaultClient(config)
	if err != nil {
		t.Fatalf("failed to create vault client: %v", err)
	}

	// Add something to cache
	client.putInCache("test", "value")

	// Close should clear cache
	err = client.Close()
	if err != nil {
		t.Errorf("unexpected error during close: %v", err)
	}

	_, found := client.getFromCache("test")
	if found {
		t.Fatal("expected cache to be cleared after close")
	}
}

// TestVaultCacheStructure tests cache entry expiration tracking
func TestVaultCacheStructure(t *testing.T) {
	os.Setenv("VAULT_SKIP_VERIFY", "true")

	config := map[string]interface{}{
		"address": "http://127.0.0.1:8200",
		"token":   "s.test_token_12345",
	}

	client, err := NewVaultClient(config)
	if err != nil {
		t.Fatalf("failed to create vault client: %v", err)
	}

	// Put in cache
	client.putInCache("mykey", "myvalue")

	// Verify cache entry has expiration
	client.cache.mu.RLock()
	entry, exists := client.cache.data["mykey"]
	client.cache.mu.RUnlock()

	if !exists {
		t.Fatal("cache entry not found")
	}

	if entry.Value != "myvalue" {
		t.Errorf("expected 'myvalue', got %s", entry.Value)
	}

	if entry.ExpiresAt.IsZero() {
		t.Fatal("expected non-zero expiration time")
	}

	if time.Now().After(entry.ExpiresAt) {
		t.Fatal("cache entry should not be expired immediately")
	}
}

// BenchmarkVaultClientCache benchmarks cache operations
func BenchmarkVaultClientCache(b *testing.B) {
	os.Setenv("VAULT_SKIP_VERIFY", "true")

	config := map[string]interface{}{
		"address": "http://127.0.0.1:8200",
		"token":   "s.test_token_12345",
	}

	client, _ := NewVaultClient(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "secret_" + string(rune(i%100))
		client.putInCache(key, "test_value")
		client.getFromCache(key)
	}
}

// BenchmarkVaultClientCreation benchmarks client creation
func BenchmarkVaultClientCreation(b *testing.B) {
	os.Setenv("VAULT_SKIP_VERIFY", "true")

	config := map[string]interface{}{
		"address": "http://127.0.0.1:8200",
		"token":   "s.test_token_12345",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewVaultClient(config)
	}
}

// TestVaultErrorHandling tests error cases
func TestVaultErrorHandling(t *testing.T) {
	os.Setenv("VAULT_SKIP_VERIFY", "true")

	config := map[string]interface{}{
		"address": "http://127.0.0.1:8200",
		"token":   "s.test_token_12345",
	}

	client, err := NewVaultClient(config)
	if err != nil {
		t.Fatalf("failed to create vault client: %v", err)
	}

	// Test GetSecret with invalid path
	ctx := context.Background()
	_, err = client.GetSecret(ctx, "", "field")
	if err == nil {
		t.Fatal("expected error for empty path")
	}

	if provErr, ok := err.(*ProviderError); ok {
		if provErr.Provider != "vault" {
			t.Errorf("expected vault provider error, got %s", provErr.Provider)
		}
	}
}

// TestVaultClientInterface verifies the client implements the interface
func TestVaultClientInterface(t *testing.T) {
	os.Setenv("VAULT_SKIP_VERIFY", "true")

	config := map[string]interface{}{
		"address": "http://127.0.0.1:8200",
		"token":   "s.test_token_12345",
	}

	client, err := NewVaultClient(config)
	if err != nil {
		t.Fatalf("failed to create vault client: %v", err)
	}

	var _ SecretProvider = client

	// Verify interface methods exist and are callable
	if client.Name() == "" {
		t.Fatal("Name() should return non-empty string")
	}

	// Close should not panic
	err = client.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}
