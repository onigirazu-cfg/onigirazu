package secrets

import (
	"context"
	"fmt"
	"sync"
)

// TemplateSecretManager manages secret providers for template functions
type TemplateSecretManager struct {
	providers map[string]SecretProvider
	mu        sync.RWMutex
}

// NewTemplateSecretManager creates a new template secret manager
func NewTemplateSecretManager() *TemplateSecretManager {
	return &TemplateSecretManager{
		providers: make(map[string]SecretProvider),
	}
}

// RegisterProvider registers a secret provider
func (tsm *TemplateSecretManager) RegisterProvider(name string, provider SecretProvider) error {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()

	if _, exists := tsm.providers[name]; exists {
		return fmt.Errorf("provider '%s' already registered", name)
	}

	tsm.providers[name] = provider
	return nil
}

// GetProvider retrieves a registered provider
func (tsm *TemplateSecretManager) GetProvider(name string) (SecretProvider, error) {
	tsm.mu.RLock()
	defer tsm.mu.RUnlock()

	provider, exists := tsm.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not registered", name)
	}

	return provider, nil
}

// Close closes all registered providers
func (tsm *TemplateSecretManager) Close() error {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()

	var lastErr error
	for _, provider := range tsm.providers {
		if err := provider.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// CreateBitwardenFunc creates a template function for Bitwarden secrets
func (tsm *TemplateSecretManager) CreateBitwardenFunc() func(itemName, field string) (string, error) {
	return func(itemName, field string) (string, error) {
		provider, err := tsm.GetProvider("bitwarden")
		if err != nil {
			return "", fmt.Errorf("bitwarden provider not configured: %w", err)
		}

		ctx := context.Background()
		value, err := provider.GetSecret(ctx, itemName, field)
		if err != nil {
			return "", fmt.Errorf("failed to get bitwarden secret '%s.%s': %w", itemName, field, err)
		}

		return value, nil
	}
}

// CreateVaultFunc creates a template function for Vault secrets
func (tsm *TemplateSecretManager) CreateVaultFunc() func(path, field string) (string, error) {
	return func(path, field string) (string, error) {
		provider, err := tsm.GetProvider("vault")
		if err != nil {
			return "", fmt.Errorf("vault provider not configured: %w", err)
		}

		ctx := context.Background()
		value, err := provider.GetSecret(ctx, path, field)
		if err != nil {
			return "", fmt.Errorf("failed to get vault secret '%s.%s': %w", path, field, err)
		}

		return value, nil
	}
}

// CreateSecretFunc creates a generic template function for any provider
func (tsm *TemplateSecretManager) CreateSecretFunc() func(providerName, itemName, field string) (string, error) {
	return func(providerName, itemName, field string) (string, error) {
		provider, err := tsm.GetProvider(providerName)
		if err != nil {
			return "", fmt.Errorf("provider '%s' not configured: %w", providerName, err)
		}

		ctx := context.Background()
		value, err := provider.GetSecret(ctx, itemName, field)
		if err != nil {
			return "", fmt.Errorf("failed to get secret '%s.%s' from %s: %w", itemName, field, providerName, err)
		}

		return value, nil
	}
}

// GetTemplateFunctions returns all template functions for secrets
func (tsm *TemplateSecretManager) GetTemplateFunctions() map[string]interface{} {
	return map[string]interface{}{
		"bitwarden": tsm.CreateBitwardenFunc(),
		"vault":     tsm.CreateVaultFunc(),
		"secret":    tsm.CreateSecretFunc(),
	}
}
