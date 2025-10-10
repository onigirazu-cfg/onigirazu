package secrets

import (
	"context"
)

// SecretProvider defines the interface for secret management providers
type SecretProvider interface {
	// GetSecret retrieves a secret value by item name and field
	GetSecret(ctx context.Context, itemName, field string) (string, error)

	// ListSecrets lists all available secrets (optionally filtered)
	ListSecrets(ctx context.Context, filter string) ([]string, error)

	// Close cleans up resources
	Close() error

	// Name returns the provider name
	Name() string
}

// ProviderConfig holds configuration for secret providers
type ProviderConfig struct {
	Type   string                 `yaml:"type" json:"type"`     // "bitwarden" or "vault"
	Config map[string]interface{} `yaml:"config" json:"config"` // Provider-specific config
}

// NewProvider creates a new secret provider based on configuration
func NewProvider(config ProviderConfig) (SecretProvider, error) {
	switch config.Type {
	case "bitwarden":
		return NewBitwardenClient(config.Config)
	case "vault":
		return NewVaultClient(config.Config)
	default:
		return nil, &ProviderError{
			Provider: config.Type,
			Message:  "unsupported provider type",
		}
	}
}

// ProviderError represents a provider-specific error
type ProviderError struct {
	Provider string
	Message  string
	Err      error
}

func (e *ProviderError) Error() string {
	if e.Err != nil {
		return e.Provider + ": " + e.Message + ": " + e.Err.Error()
	}
	return e.Provider + ": " + e.Message
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}
