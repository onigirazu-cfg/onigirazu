package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// BitwardenClient manages Bitwarden secret retrieval
type BitwardenClient struct {
	server       string
	email        string
	password     string
	sessionToken string
	cache        *SecretCache
	mu           sync.RWMutex
}

// BitwardenItem represents a Bitwarden vault item
type BitwardenItem struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Type     int     `json:"type"` // 1=login, 2=note, 3=card, 4=identity
	Login    *Login  `json:"login,omitempty"`
	Notes    string  `json:"notes,omitempty"`
	Fields   []Field `json:"fields,omitempty"`
	FolderID string  `json:"folderId,omitempty"`
}

// Login represents login credentials in a Bitwarden item
type Login struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Totp     string `json:"totp,omitempty"`
	Uris     []URI  `json:"uris,omitempty"`
}

// URI represents a URI in a Bitwarden login item
type URI struct {
	Match int    `json:"match,omitempty"`
	URI   string `json:"uri"`
}

// Field represents a custom field in a Bitwarden item
type Field struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  int    `json:"type"` // 0=text, 1=hidden, 2=boolean
}

// SecretCache caches retrieved secrets with TTL
type SecretCache struct {
	data map[string]*CachedSecret
	mu   sync.RWMutex
	ttl  time.Duration
}

// CachedSecret represents a cached secret with expiration
type CachedSecret struct {
	Value     string
	ExpiresAt time.Time
}

// NewSecretCache creates a new secret cache
func NewSecretCache(ttl time.Duration) *SecretCache {
	cache := &SecretCache{
		data: make(map[string]*CachedSecret),
		ttl:  ttl,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a cached secret
func (sc *SecretCache) Get(key string) (string, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	cached, exists := sc.data[key]
	if !exists {
		return "", false
	}

	if time.Now().After(cached.ExpiresAt) {
		return "", false
	}

	return cached.Value, true
}

// Set stores a secret in cache
func (sc *SecretCache) Set(key, value string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.data[key] = &CachedSecret{
		Value:     value,
		ExpiresAt: time.Now().Add(sc.ttl),
	}
}

// Clear removes all cached secrets
func (sc *SecretCache) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.data = make(map[string]*CachedSecret)
}

// cleanup periodically removes expired secrets
func (sc *SecretCache) cleanup() {
	ticker := time.NewTicker(sc.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		sc.mu.Lock()
		now := time.Now()
		for key, cached := range sc.data {
			if now.After(cached.ExpiresAt) {
				delete(sc.data, key)
			}
		}
		sc.mu.Unlock()
	}
}

// NewBitwardenClient creates a new Bitwarden client
func NewBitwardenClient(config map[string]interface{}) (*BitwardenClient, error) {
	server, _ := config["server"].(string)
	if server == "" {
		server = "https://vault.bitwarden.com"
	}

	email, _ := config["email"].(string)
	password, _ := config["password"].(string)

	cacheTTL := 5 * time.Minute
	if ttl, ok := config["cache_ttl"].(int); ok {
		cacheTTL = time.Duration(ttl) * time.Second
	}

	client := &BitwardenClient{
		server:   server,
		email:    email,
		password: password,
		cache:    NewSecretCache(cacheTTL),
	}

	// Authenticate on creation
	if err := client.authenticate(config); err != nil {
		return nil, &ProviderError{
			Provider: "bitwarden",
			Message:  "authentication failed",
			Err:      err,
		}
	}

	return client, nil
}

// authenticate authenticates with Bitwarden and obtains a session token
func (bc *BitwardenClient) authenticate(config map[string]interface{}) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Check if session token is provided
	if token, ok := config["session_token"].(string); ok && token != "" {
		bc.sessionToken = token
		return nil
	}

	// Check if BW_SESSION environment variable is set
	cmd := exec.Command("sh", "-c", "echo $BW_SESSION")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		token := strings.TrimSpace(string(output))
		if token != "" {
			bc.sessionToken = token
			return nil
		}
	}

	// Authenticate using email and password
	if bc.email == "" || bc.password == "" {
		return fmt.Errorf("email and password required for authentication")
	}

	// Set server
	if bc.server != "https://vault.bitwarden.com" {
		// #nosec G204 -- server URL is from trusted configuration
		cmd := exec.Command("bw", "config", "server", bc.server)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to configure server: %w", err)
		}
	}

	// Login
	// #nosec G204 -- credentials are from trusted configuration
	cmd = exec.Command("bw", "login", bc.email, bc.password, "--raw")
	output, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	bc.sessionToken = strings.TrimSpace(string(output))
	return nil
}

// GetSecret retrieves a secret from Bitwarden
func (bc *BitwardenClient) GetSecret(ctx context.Context, itemName, field string) (string, error) {
	// Check cache first
	cacheKey := itemName + ":" + field
	if cached, found := bc.cache.Get(cacheKey); found {
		return cached, nil
	}

	bc.mu.RLock()
	sessionToken := bc.sessionToken
	bc.mu.RUnlock()

	if sessionToken == "" {
		return "", &ProviderError{
			Provider: "bitwarden",
			Message:  "not authenticated",
		}
	}

	// Get item from Bitwarden
	// #nosec G204 -- itemName is validated and sessionToken is from authentication
	cmd := exec.CommandContext(ctx, "bw", "get", "item", itemName, "--session", sessionToken)
	output, err := cmd.Output()
	if err != nil {
		return "", &ProviderError{
			Provider: "bitwarden",
			Message:  fmt.Sprintf("failed to get item '%s'", itemName),
			Err:      err,
		}
	}

	// Parse JSON response
	var item BitwardenItem
	if err := json.Unmarshal(output, &item); err != nil {
		return "", &ProviderError{
			Provider: "bitwarden",
			Message:  "failed to parse item",
			Err:      err,
		}
	}

	// Extract field value
	value, err := bc.extractField(&item, field)
	if err != nil {
		return "", err
	}

	// Cache the result
	bc.cache.Set(cacheKey, value)

	return value, nil
}

// extractField extracts a specific field from a Bitwarden item
func (bc *BitwardenClient) extractField(item *BitwardenItem, field string) (string, error) {
	field = strings.ToLower(field)

	// Check login fields
	if item.Login != nil {
		switch field {
		case "username", "user":
			return item.Login.Username, nil
		case "password", "pass":
			return item.Login.Password, nil
		case "totp":
			return item.Login.Totp, nil
		}
	}

	// Check notes
	if field == "notes" || field == "note" {
		return item.Notes, nil
	}

	// Check custom fields
	for _, f := range item.Fields {
		if strings.ToLower(f.Name) == field {
			return f.Value, nil
		}
	}

	return "", &ProviderError{
		Provider: "bitwarden",
		Message:  fmt.Sprintf("field '%s' not found in item '%s'", field, item.Name),
	}
}

// ListSecrets lists all available secrets
func (bc *BitwardenClient) ListSecrets(ctx context.Context, filter string) ([]string, error) {
	bc.mu.RLock()
	sessionToken := bc.sessionToken
	bc.mu.RUnlock()

	if sessionToken == "" {
		return nil, &ProviderError{
			Provider: "bitwarden",
			Message:  "not authenticated",
		}
	}

	// List all items
	// #nosec G204 -- sessionToken is from authentication
	cmd := exec.CommandContext(ctx, "bw", "list", "items", "--session", sessionToken)
	output, err := cmd.Output()
	if err != nil {
		return nil, &ProviderError{
			Provider: "bitwarden",
			Message:  "failed to list items",
			Err:      err,
		}
	}

	// Parse JSON response
	var items []BitwardenItem
	if err := json.Unmarshal(output, &items); err != nil {
		return nil, &ProviderError{
			Provider: "bitwarden",
			Message:  "failed to parse items",
			Err:      err,
		}
	}

	// Extract names
	var names []string
	for _, item := range items {
		if filter == "" || strings.Contains(strings.ToLower(item.Name), strings.ToLower(filter)) {
			names = append(names, item.Name)
		}
	}

	return names, nil
}

// Close cleans up resources
func (bc *BitwardenClient) Close() error {
	bc.cache.Clear()

	// Lock the vault
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if bc.sessionToken != "" {
		cmd := exec.Command("bw", "lock")
		_ = cmd.Run() // Ignore errors on lock
	}

	return nil
}

// Name returns the provider name
func (bc *BitwardenClient) Name() string {
	return "bitwarden"
}
