package secrets

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Config is the secrets block of onigirazu.yml. Credentials never come from
// it: Bitwarden uses the session of an unlocked vault (BW_SESSION), Vault
// its token (VAULT_TOKEN or ~/.vault-token).
type Config struct {
	// CacheTTL is how long a value is reused within a run (default 5m)
	CacheTTL time.Duration `yaml:"cache_ttl" json:"cache_ttl"`
	Vault    VaultConfig   `yaml:"vault" json:"vault"`
}

// VaultConfig locates a HashiCorp Vault server
type VaultConfig struct {
	Address   string `yaml:"address" json:"address"`     // default VAULT_ADDR
	Namespace string `yaml:"namespace" json:"namespace"` // default VAULT_NAMESPACE
	Mount     string `yaml:"mount" json:"mount"`         // KV v2 engine, default "secret"
}

// Resolver reads secrets for templates; each provider is created when a
// template first uses it, so runs without secrets need neither bw nor Vault
type Resolver struct {
	cfg       Config
	mu        sync.Mutex
	providers map[string]SecretProvider
	// newProvider is NewProvider; tests replace it
	newProvider func(ProviderConfig) (SecretProvider, error)
}

// NewResolver creates a resolver for the configuration
func NewResolver(cfg Config) *Resolver {
	return &Resolver{cfg: cfg, providers: map[string]SecretProvider{}, newProvider: NewProvider}
}

// Get reads a field of a secret: provider "bitwarden" (item name or id,
// field password, username, totp, notes or a custom field) or "vault" (path
// in the KV engine, field of its data)
func (r *Resolver) Get(provider, item, field string) (string, error) {
	p, err := r.provider(provider)
	if err != nil {
		return "", err
	}
	return p.GetSecret(context.Background(), item, field)
}

func (r *Resolver) provider(name string) (SecretProvider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.providers[name]; ok {
		return p, nil
	}
	ttl := r.cfg.CacheTTL
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	var config map[string]interface{}
	switch name {
	case "bitwarden":
		session := strings.TrimSpace(os.Getenv("BW_SESSION"))
		if session == "" {
			return nil, &ProviderError{Provider: "bitwarden",
				Message: "the vault is locked: run export BW_SESSION=$(bw unlock --raw)"}
		}
		config = map[string]interface{}{"session_token": session, "cache_ttl": ttl}
	case "vault":
		v := r.cfg.Vault
		address := firstNonEmpty(v.Address, os.Getenv("VAULT_ADDR"))
		token := strings.TrimSpace(os.Getenv("VAULT_TOKEN"))
		if token == "" {
			if home, err := os.UserHomeDir(); err == nil {
				if data, err := os.ReadFile(filepath.Join(home, ".vault-token")); err == nil { // #nosec G304 -- the vault CLI's token file
					token = strings.TrimSpace(string(data))
				}
			}
		}
		if token == "" {
			return nil, &ProviderError{Provider: "vault", Message: "no token: set VAULT_TOKEN or run vault login"}
		}
		config = map[string]interface{}{"address": address, "token": token,
			"namespace": firstNonEmpty(v.Namespace, os.Getenv("VAULT_NAMESPACE")),
			"mount":     v.Mount, "cache_ttl": ttl.String()}
	default:
		return nil, fmt.Errorf("unknown secret provider %q (bitwarden, vault)", name)
	}
	p, err := r.newProvider(ProviderConfig{Type: name, Config: config})
	if err != nil {
		return nil, err
	}
	r.providers[name] = p
	return p, nil
}

// Close closes the providers that were used
func (r *Resolver) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for name, p := range r.providers {
		_ = p.Close()
		delete(r.providers, name)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
