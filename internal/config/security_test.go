package config

import (
	"testing"
	"time"
)

// ============================================================================
// DEFAULT SECURITY CONFIG TESTS
// ============================================================================

func TestDefaultSecurityConfig(t *testing.T) {
	sc := DefaultSecurityConfig()

	// Test SSH config
	if !sc.SSH.StrictHostKeyChecking {
		t.Error("Expected StrictHostKeyChecking to be true by default")
	}

	if sc.SSH.ConnectionTimeout != 30*time.Second {
		t.Errorf("Expected ConnectionTimeout to be 30s, got %v", sc.SSH.ConnectionTimeout)
	}

	if sc.SSH.CommandTimeout != 300*time.Second {
		t.Errorf("Expected CommandTimeout to be 300s, got %v", sc.SSH.CommandTimeout)
	}

	if sc.SSH.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %d", sc.SSH.MaxRetries)
	}
}

func TestDefaultSecurityConfigCiphers(t *testing.T) {
	sc := DefaultSecurityConfig()

	expectedCiphers := []string{
		"aes128-ctr",
		"aes192-ctr",
		"aes256-ctr",
		"aes128-gcm@openssh.com",
		"aes256-gcm@openssh.com",
	}

	if len(sc.SSH.AllowedCiphers) != len(expectedCiphers) {
		t.Errorf("Expected %d ciphers, got %d", len(expectedCiphers), len(sc.SSH.AllowedCiphers))
	}

	for i, cipher := range expectedCiphers {
		if i >= len(sc.SSH.AllowedCiphers) || sc.SSH.AllowedCiphers[i] != cipher {
			t.Errorf("Expected cipher %d to be %s", i, cipher)
		}
	}
}

func TestDefaultSecurityConfigMACs(t *testing.T) {
	sc := DefaultSecurityConfig()

	expectedMACs := []string{
		"hmac-sha2-256-etm@openssh.com",
		"hmac-sha2-512-etm@openssh.com",
		"hmac-sha2-256",
		"hmac-sha2-512",
	}

	if len(sc.SSH.AllowedMACs) != len(expectedMACs) {
		t.Errorf("Expected %d MACs, got %d", len(expectedMACs), len(sc.SSH.AllowedMACs))
	}

	for i, mac := range expectedMACs {
		if i >= len(sc.SSH.AllowedMACs) || sc.SSH.AllowedMACs[i] != mac {
			t.Errorf("Expected MAC %d to be %s", i, mac)
		}
	}
}

func TestDefaultSecurityConfigKeyExchange(t *testing.T) {
	sc := DefaultSecurityConfig()

	expectedAlgos := []string{
		"curve25519-sha256",
		"curve25519-sha256@libssh.org",
		"ecdh-sha2-nistp256",
		"ecdh-sha2-nistp384",
		"ecdh-sha2-nistp521",
		"diffie-hellman-group14-sha256",
		"diffie-hellman-group16-sha512",
	}

	if len(sc.SSH.AllowedKexAlgos) != len(expectedAlgos) {
		t.Errorf("Expected %d key exchange algorithms, got %d", len(expectedAlgos), len(sc.SSH.AllowedKexAlgos))
	}

	for i, algo := range expectedAlgos {
		if i >= len(sc.SSH.AllowedKexAlgos) || sc.SSH.AllowedKexAlgos[i] != algo {
			t.Errorf("Expected KEX algo %d to be %s", i, algo)
		}
	}
}

// ============================================================================
// SECURITY CONFIG VALIDATION TESTS
// ============================================================================

func TestValidateSecurityConfigValid(t *testing.T) {
	sc := DefaultSecurityConfig()

	err := ValidateSecurityConfig(sc)
	if err != nil {
		t.Errorf("Expected no error for valid config, got %v", err)
	}
}

func TestValidateSecurityConfigInvalidConnectionTimeout(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.SSH.ConnectionTimeout = -1

	err := ValidateSecurityConfig(sc)
	if err == nil {
		t.Error("Expected error for negative ConnectionTimeout")
	}
}

func TestValidateSecurityConfigZeroConnectionTimeout(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.SSH.ConnectionTimeout = 0

	err := ValidateSecurityConfig(sc)
	if err == nil {
		t.Error("Expected error for zero ConnectionTimeout")
	}
}

func TestValidateSecurityConfigInvalidCommandTimeout(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.SSH.CommandTimeout = -5 * time.Second

	err := ValidateSecurityConfig(sc)
	if err == nil {
		t.Error("Expected error for negative CommandTimeout")
	}
}

func TestValidateSecurityConfigZeroCommandTimeout(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.SSH.CommandTimeout = 0

	err := ValidateSecurityConfig(sc)
	if err == nil {
		t.Error("Expected error for zero CommandTimeout")
	}
}

func TestValidateSecurityConfigNegativeMaxRetries(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.SSH.MaxRetries = -1

	err := ValidateSecurityConfig(sc)
	if err == nil {
		t.Error("Expected error for negative MaxRetries")
	}
}

func TestValidateSecurityConfigZeroMaxRetries(t *testing.T) {
	sc := DefaultSecurityConfig()
	sc.SSH.MaxRetries = 0

	err := ValidateSecurityConfig(sc)
	if err != nil {
		t.Errorf("Expected no error for zero MaxRetries, got %v", err)
	}
}

// ============================================================================
// SECURITY CONFIG MODIFICATION TESTS
// ============================================================================

func TestModifySecurityConfig(t *testing.T) {
	sc := DefaultSecurityConfig()

	// Modify values
	sc.SSH.StrictHostKeyChecking = false
	sc.SSH.ConnectionTimeout = 60 * time.Second
	sc.SSH.MaxRetries = 5
	sc.SSH.KnownHostsFile = "/etc/ssh/known_hosts"

	// Verify modifications
	if sc.SSH.StrictHostKeyChecking {
		t.Error("Expected StrictHostKeyChecking to be false after modification")
	}

	if sc.SSH.ConnectionTimeout != 60*time.Second {
		t.Errorf("Expected ConnectionTimeout to be 60s, got %v", sc.SSH.ConnectionTimeout)
	}

	if sc.SSH.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries to be 5, got %d", sc.SSH.MaxRetries)
	}

	if sc.SSH.KnownHostsFile != "/etc/ssh/known_hosts" {
		t.Errorf("Expected KnownHostsFile to be /etc/ssh/known_hosts, got %s", sc.SSH.KnownHostsFile)
	}

	// Validation should still work
	err := ValidateSecurityConfig(sc)
	if err != nil {
		t.Errorf("Expected no error for modified valid config, got %v", err)
	}
}

func TestSecurityConfigAddCustomCipher(t *testing.T) {
	sc := DefaultSecurityConfig()
	originalCount := len(sc.SSH.AllowedCiphers)

	sc.SSH.AllowedCiphers = append(sc.SSH.AllowedCiphers, "aes128-gcm@openssh.com")

	if len(sc.SSH.AllowedCiphers) != originalCount+1 {
		t.Errorf("Expected %d ciphers, got %d", originalCount+1, len(sc.SSH.AllowedCiphers))
	}
}

func TestSecurityConfigCustomKnownHostsFile(t *testing.T) {
	sc := DefaultSecurityConfig()
	customPath := "/custom/path/.ssh/known_hosts"

	sc.SSH.KnownHostsFile = customPath

	if sc.SSH.KnownHostsFile != customPath {
		t.Errorf("Expected %s, got %s", customPath, sc.SSH.KnownHostsFile)
	}
}

// ============================================================================
// SSH SECURITY CONFIG EDGE CASES
// ============================================================================

func TestSSHSecurityConfigBoundaryValues(t *testing.T) {
	tests := []struct {
		name      string
		timeout   time.Duration
		expectErr bool
	}{
		{"very small timeout", 1 * time.Millisecond, false},
		{"one second", 1 * time.Second, false},
		{"large timeout", 1 * time.Hour, false},
		{"negative timeout", -1 * time.Second, true},
		{"zero timeout", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := DefaultSecurityConfig()
			sc.SSH.ConnectionTimeout = tt.timeout

			err := ValidateSecurityConfig(sc)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got %v", err)
			}
		})
	}
}

func TestSSHMaxRetriesBoundaryValues(t *testing.T) {
	tests := []struct {
		name      string
		maxRetry  int
		expectErr bool
	}{
		{"zero retries", 0, false},
		{"single retry", 1, false},
		{"many retries", 100, false},
		{"negative retries", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := DefaultSecurityConfig()
			sc.SSH.MaxRetries = tt.maxRetry

			err := ValidateSecurityConfig(sc)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got %v", err)
			}
		})
	}
}

// ============================================================================
// SECURITY CONFIG IMMUTABILITY TESTS
// ============================================================================

func TestDefaultSecurityConfigConsistency(t *testing.T) {
	sc1 := DefaultSecurityConfig()
	sc2 := DefaultSecurityConfig()

	// Both should have same values
	if sc1.SSH.StrictHostKeyChecking != sc2.SSH.StrictHostKeyChecking {
		t.Error("DefaultSecurityConfig should return consistent values")
	}

	if sc1.SSH.ConnectionTimeout != sc2.SSH.ConnectionTimeout {
		t.Error("DefaultSecurityConfig should return consistent values")
	}

	if len(sc1.SSH.AllowedCiphers) != len(sc2.SSH.AllowedCiphers) {
		t.Error("DefaultSecurityConfig should return consistent cipher lists")
	}
}

// ============================================================================
// VALIDATION ERROR TESTS
// ============================================================================

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("test_module", "test_task", "test error message")

	if err == nil {
		t.Fatal("Expected error to be returned")
	}

	errStr := err.Error()
	if len(errStr) == 0 {
		t.Error("Expected error message to contain text")
	}

	// Should contain module name
	if !contains(errStr, "test_module") {
		t.Errorf("Expected error to contain module name, got %s", errStr)
	}

	// Should contain error message
	if !contains(errStr, "test error message") {
		t.Errorf("Expected error to contain error message, got %s", errStr)
	}
}

// Helper function for string contains check
func contains(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
