package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/hashicorp/vault/api"
)

// VaultClient manages HashiCorp Vault secret retrieval
type VaultClient struct {
	address   string
	token     string
	client    *api.Client
	cache     *secretCache
	namespace string
}

// secretCache holds cached secrets with TTL
type secretCache struct {
	data   map[string]cachedSecret
	mu     sync.RWMutex
	ttl    time.Duration
	maxAge time.Duration
}

type cachedSecret struct {
	value     string
	timestamp time.Time
}

// NewVaultClient creates a new Vault client
func NewVaultClient(config map[string]interface{}) (*VaultClient, error) {
	address, _ := config["address"].(string)
	if address == "" {
		return nil, &ProviderError{
			Provider: "vault",
			Message:  "vault address is required",
		}
	}

	token, _ := config["token"].(string)
	if token == "" {
		return nil, &ProviderError{
			Provider: "vault",
			Message:  "vault token is required",
		}
	}

	// Optional namespace
	namespace, _ := config["namespace"].(string)

	// Create Vault API client
	vaultConfig := &api.Config{
		Address: address,
		HttpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	client, err := api.NewClient(vaultConfig)
	if err != nil {
		return nil, &ProviderError{
			Provider: "vault",
			Message:  fmt.Sprintf("failed to create vault client: %v", err),
		}
	}

	// Set token and namespace
	client.SetToken(token)
	if namespace != "" {
		client.SetNamespace(namespace)
	}

	// Verify connectivity with a simple lookup
	_, err = client.Auth().Token().LookupSelf()
	if err != nil {
		return nil, &ProviderError{
			Provider: "vault",
			Message:  fmt.Sprintf("vault authentication failed: %v", err),
		}
	}

	cacheTTL := 5 * time.Minute // Cache secrets for 5 minutes by default
	if ttlVal, ok := config["cache_ttl"].(string); ok {
		if d, err := time.ParseDuration(ttlVal); err == nil {
			cacheTTL = d
		}
	}

	return &VaultClient{
		address:   address,
		token:     token,
		client:    client,
		namespace: namespace,
		cache: &secretCache{
			data:   make(map[string]cachedSecret),
			ttl:    cacheTTL,
			maxAge: cacheTTL,
		},
	}, nil
}

// GetSecret retrieves a secret from Vault
func (vc *VaultClient) GetSecret(ctx context.Context, path, field string) (string, error) {
	if path == "" {
		return "", &ProviderError{
			Provider: "vault",
			Message:  "secret path is required",
		}
	}

	cacheKey := fmt.Sprintf("%s:%s", path, field)

	// Check cache first
	vc.cache.mu.RLock()
	if cached, ok := vc.cache.data[cacheKey]; ok {
		if time.Since(cached.timestamp) < vc.cache.ttl {
			vc.cache.mu.RUnlock()
			return cached.value, nil
		}
	}
	vc.cache.mu.RUnlock()

	// Read from Vault
	secret, err := vc.client.Logical().Read(path)
	if err != nil {
		return "", &ProviderError{
			Provider: "vault",
			Message:  fmt.Sprintf("failed to read secret from path %s: %v", path, err),
		}
	}

	if secret == nil {
		return "", &ProviderError{
			Provider: "vault",
			Message:  fmt.Sprintf("secret not found at path %s", path),
		}
	}

	// Handle KV v2 secrets (they're nested under "data" key)
	var data map[string]interface{}
	if dataMap, ok := secret.Data["data"].(map[string]interface{}); ok {
		// KV v2 format
		data = dataMap
	} else {
		// KV v1 format or direct response
		data = secret.Data
	}

	// Extract field
	if field == "" {
		// Return the entire secret as JSON if no field specified
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return "", &ProviderError{
				Provider: "vault",
				Message:  fmt.Sprintf("failed to marshal secret: %v", err),
			}
		}
		value := string(jsonBytes)

		// Cache the result
		vc.cache.mu.Lock()
		vc.cache.data[cacheKey] = cachedSecret{
			value:     value,
			timestamp: time.Now(),
		}
		vc.cache.mu.Unlock()

		return value, nil
	}

	// Extract specific field
	value, ok := data[field].(string)
	if !ok {
		// Try to convert to string
		valueInterface := data[field]
		if valueInterface == nil {
			return "", &ProviderError{
				Provider: "vault",
				Message:  fmt.Sprintf("field %s not found in secret at path %s", field, path),
			}
		}

		// Convert non-string values to JSON
		jsonBytes, err := json.Marshal(valueInterface)
		if err != nil {
			return "", &ProviderError{
				Provider: "vault",
				Message:  fmt.Sprintf("failed to convert field %s to string: %v", field, err),
			}
		}
		value = string(jsonBytes)
	}

	// Cache the result
	vc.cache.mu.Lock()
	vc.cache.data[cacheKey] = cachedSecret{
		value:     value,
		timestamp: time.Now(),
	}
	vc.cache.mu.Unlock()

	return value, nil
}

// ListSecrets lists all available secrets
func (vc *VaultClient) ListSecrets(ctx context.Context, filter string) ([]string, error) {
	if filter == "" {
		return nil, &ProviderError{
			Provider: "vault",
			Message:  "secret path filter is required",
		}
	}

	// List secrets at the given path
	secret, err := vc.client.Logical().List(filter)
	if err != nil {
		return nil, &ProviderError{
			Provider: "vault",
			Message:  fmt.Sprintf("failed to list secrets at path %s: %v", filter, err),
		}
	}

	if secret == nil {
		return []string{}, nil
	}

	// Extract keys from response
	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return []string{}, nil
	}

	result := make([]string, 0, len(keys))
	for _, key := range keys {
		if keyStr, ok := key.(string); ok {
			result = append(result, keyStr)
		}
	}

	return result, nil
}

// ClearCache clears the secret cache
func (vc *VaultClient) ClearCache() {
	vc.cache.mu.Lock()
	defer vc.cache.mu.Unlock()
	vc.cache.data = make(map[string]cachedSecret)
}

// Close cleans up resources
func (vc *VaultClient) Close() error {
	vc.ClearCache()
	return nil
}

// Name returns the provider name
func (vc *VaultClient) Name() string {
	return "vault"
}

// Placeholder for Vault implementation
var _ SecretProvider = (*VaultClient)(nil)
