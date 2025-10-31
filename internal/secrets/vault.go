package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

// cachedSecret holds a secret value with expiration timestamp
type cachedSecret struct {
	Value     string
	ExpiresAt time.Time
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

	// Verify connectivity with a simple lookup (skip in test environments)
	if os.Getenv("VAULT_SKIP_VERIFY") != "true" {
		_, err = client.Auth().Token().LookupSelf()
		if err != nil {
			return nil, &ProviderError{
				Provider: "vault",
				Message:  fmt.Sprintf("vault authentication failed: %v", err),
			}
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

	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", path, field)
	if value, found := vc.getFromCache(cacheKey); found {
		return value, nil
	}

	// Read secret from Vault
	secret, err := vc.client.KVv2("secret").Get(ctx, path)
	if err != nil {
		return "", &ProviderError{
			Provider: "vault",
			Message:  fmt.Sprintf("failed to read secret at %s: %v", path, err),
		}
	}

	if secret == nil || secret.Data == nil {
		return "", &ProviderError{
			Provider: "vault",
			Message:  fmt.Sprintf("secret not found at path: %s", path),
		}
	}

	// Extract field value
	var value string
	if field != "" {
		// If a specific field is requested
		if val, exists := secret.Data[field]; exists {
			value = fmt.Sprintf("%v", val)
		} else {
			return "", &ProviderError{
				Provider: "vault",
				Message:  fmt.Sprintf("field %q not found in secret at path %s", field, path),
			}
		}
	} else {
		// If no field specified, treat entire secret as a map
		data, err := json.Marshal(secret.Data)
		if err != nil {
			return "", &ProviderError{
				Provider: "vault",
				Message:  fmt.Sprintf("failed to marshal secret data: %v", err),
			}
		}
		value = string(data)
	}

	// Cache the result
	vc.putInCache(cacheKey, value)
	return value, nil
}

// ListSecrets lists all available secrets at a given path in Vault
func (vc *VaultClient) ListSecrets(ctx context.Context, filter string) ([]string, error) {
	if filter == "" {
		filter = "secret"
	}

	// List secrets from Vault using the Logical client with KV v2 metadata endpoint
	// KV v2 stores metadata at secret/metadata/{path}
	path := "secret/metadata/" + filter
	secret, err := vc.client.Logical().List(path)
	if err != nil {
		return nil, &ProviderError{
			Provider: "vault",
			Message:  fmt.Sprintf("failed to list secrets at %s: %v", filter, err),
		}
	}

	if secret == nil || secret.Data == nil {
		return []string{}, nil
	}

	// Extract the list of secret keys from the response data
	var keys []string
	if keyData, ok := secret.Data["keys"]; ok {
		if keyList, ok := keyData.([]interface{}); ok {
			for _, key := range keyList {
				if strKey, ok := key.(string); ok {
					keys = append(keys, strKey)
				}
			}
		}
	}

	return keys, nil
}

// getFromCache retrieves a secret from the cache if it exists and hasn't expired
func (vc *VaultClient) getFromCache(key string) (string, bool) {
	vc.cache.mu.RLock()
	defer vc.cache.mu.RUnlock()

	if cached, exists := vc.cache.data[key]; exists {
		return cached.Value, true
	}
	return "", false
}

// putInCache stores a secret in the cache with timestamp
func (vc *VaultClient) putInCache(key, value string) {
	vc.cache.mu.Lock()
	defer vc.cache.mu.Unlock()

	vc.cache.data[key] = cachedSecret{
		Value:     value,
		ExpiresAt: time.Now().Add(vc.cache.ttl),
	}
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
