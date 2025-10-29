package secrets

import (
	"context"
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
	// Vault integration is not yet fully implemented
	return "", &ProviderError{
		Provider: "vault",
		Message:  "vault integration not yet implemented",
	}
}

// ListSecrets lists all available secrets
func (vc *VaultClient) ListSecrets(ctx context.Context, filter string) ([]string, error) {
	// Vault integration is not yet fully implemented
	return nil, &ProviderError{
		Provider: "vault",
		Message:  "vault integration not yet implemented",
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
