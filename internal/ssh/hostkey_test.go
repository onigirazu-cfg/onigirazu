package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// testingTB is an interface that covers both *testing.T and *testing.B
type testingTB interface {
	Helper()
	Fatalf(format string, args ...interface{})
}

// generateTestKey generates a test SSH key pair and returns the public key
func generateTestKey(t testingTB) ssh.PublicKey {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	return signer.PublicKey()
}

// generateTestPrivateKey generates a test SSH private key and returns it in OpenSSH format
func generateTestPrivateKey(t testingTB) []byte {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Marshal the private key to OpenSSH format
	pemBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		t.Fatalf("failed to marshal private key: %v", err)
	}

	// Encode the PEM block to bytes
	return pem.EncodeToMemory(pemBlock)
}

// TestNewHostKeyManager tests creating a new host key manager
func TestNewHostKeyManager(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	tests := []struct {
		name           string
		knownHostsFile string
		strictMode     bool
	}{
		{
			name:           "with custom file",
			knownHostsFile: knownHostsFile,
			strictMode:     false,
		},
		{
			name:           "with strict mode",
			knownHostsFile: knownHostsFile,
			strictMode:     true,
		},
		{
			name:           "with empty file path (should use default)",
			knownHostsFile: "",
			strictMode:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hkm := NewHostKeyManager(tt.knownHostsFile, tt.strictMode)
			assert.NotNil(t, hkm)
			assert.Equal(t, tt.strictMode, hkm.strictMode)
			assert.NotNil(t, hkm.knownHosts)

			if tt.knownHostsFile == "" {
				// Should use default path
				home, _ := os.UserHomeDir()
				expectedPath := filepath.Join(home, ".ssh", "known_hosts")
				assert.Equal(t, expectedPath, hkm.knownHostsFile)
			} else {
				assert.Equal(t, tt.knownHostsFile, hkm.knownHostsFile)
			}
		})
	}
}

// TestHostKeyManager_LoadKnownHosts tests loading known hosts from file
func TestHostKeyManager_LoadKnownHosts(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	// Create a test known_hosts file
	testKey := generateTestKey(t)
	// MarshalAuthorizedKey already includes the key type
	authorizedKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(testKey)))
	keyLine := fmt.Sprintf("example.com %s\n", authorizedKey)

	err := os.WriteFile(knownHostsFile, []byte(keyLine), 0600)
	require.NoError(t, err)

	hkm := NewHostKeyManager(knownHostsFile, false)
	assert.NotNil(t, hkm)

	// Verify the key was loaded
	assert.Contains(t, hkm.knownHosts, "example.com")
}

// TestHostKeyManager_LoadKnownHosts_NonExistent tests loading from non-existent file
func TestHostKeyManager_LoadKnownHosts_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "nonexistent_known_hosts")

	hkm := NewHostKeyManager(knownHostsFile, false)
	assert.NotNil(t, hkm)
	assert.Empty(t, hkm.knownHosts)
}

// TestHostKeyManager_LoadKnownHosts_MultipleHosts tests loading multiple hosts
func TestHostKeyManager_LoadKnownHosts_MultipleHosts(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	testKey1 := generateTestKey(t)
	testKey2 := generateTestKey(t)

	// MarshalAuthorizedKey already includes the key type
	authorizedKey1 := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(testKey1)))
	authorizedKey2 := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(testKey2)))
	content := fmt.Sprintf("host1.example.com %s\nhost2.example.com %s\n",
		authorizedKey1, authorizedKey2)

	err := os.WriteFile(knownHostsFile, []byte(content), 0600)
	require.NoError(t, err)

	hkm := NewHostKeyManager(knownHostsFile, false)
	assert.NotNil(t, hkm)
	assert.Len(t, hkm.knownHosts, 2)
	assert.Contains(t, hkm.knownHosts, "host1.example.com")
	assert.Contains(t, hkm.knownHosts, "host2.example.com")
}

// TestHostKeyManager_LoadKnownHosts_CommaSeparatedHosts tests loading comma-separated hosts
func TestHostKeyManager_LoadKnownHosts_CommaSeparatedHosts(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	testKey := generateTestKey(t)
	// MarshalAuthorizedKey already includes the key type
	authorizedKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(testKey)))
	content := fmt.Sprintf("host1.example.com,host2.example.com,192.168.1.1 %s\n", authorizedKey)

	err := os.WriteFile(knownHostsFile, []byte(content), 0600)
	require.NoError(t, err)

	hkm := NewHostKeyManager(knownHostsFile, false)
	assert.NotNil(t, hkm)
	assert.Len(t, hkm.knownHosts, 3)
	assert.Contains(t, hkm.knownHosts, "host1.example.com")
	assert.Contains(t, hkm.knownHosts, "host2.example.com")
	assert.Contains(t, hkm.knownHosts, "192.168.1.1")
}

// TestHostKeyManager_LoadKnownHosts_Comments tests loading file with comments
func TestHostKeyManager_LoadKnownHosts_Comments(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	testKey := generateTestKey(t)
	// MarshalAuthorizedKey already includes the key type
	authorizedKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(testKey)))
	content := fmt.Sprintf("# This is a comment\n\nexample.com %s\n# Another comment\n", authorizedKey)

	err := os.WriteFile(knownHostsFile, []byte(content), 0600)
	require.NoError(t, err)

	hkm := NewHostKeyManager(knownHostsFile, false)
	assert.NotNil(t, hkm)
	assert.Len(t, hkm.knownHosts, 1)
	assert.Contains(t, hkm.knownHosts, "example.com")
}

// TestHostKeyManager_VerifyHostKey_KnownHost tests verifying a known host
func TestHostKeyManager_VerifyHostKey_KnownHost(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	testKey := generateTestKey(t)
	hkm := NewHostKeyManager(knownHostsFile, false)

	// Add the key manually
	hkm.knownHosts["example.com"] = testKey

	// Verify with correct key
	addr := &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 22}
	err := hkm.VerifyHostKey("example.com", addr, testKey)
	assert.NoError(t, err)
}

// TestHostKeyManager_VerifyHostKey_KeyMismatch tests verifying with wrong key
func TestHostKeyManager_VerifyHostKey_KeyMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	testKey1 := generateTestKey(t)
	testKey2 := generateTestKey(t)

	hkm := NewHostKeyManager(knownHostsFile, false)
	hkm.knownHosts["example.com"] = testKey1

	// Verify with different key
	addr := &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 22}
	err := hkm.VerifyHostKey("example.com", addr, testKey2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key mismatch")
}

// TestHostKeyManager_VerifyHostKey_UnknownHost_StrictMode tests unknown host in strict mode
func TestHostKeyManager_VerifyHostKey_UnknownHost_StrictMode(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	testKey := generateTestKey(t)
	hkm := NewHostKeyManager(knownHostsFile, true) // Strict mode

	addr := &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 22}
	err := hkm.VerifyHostKey("unknown.example.com", addr, testKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown host")
}

// TestHostKeyManager_VerifyHostKey_UnknownHost_NonStrictMode tests unknown host in non-strict mode
func TestHostKeyManager_VerifyHostKey_UnknownHost_NonStrictMode(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	testKey := generateTestKey(t)
	hkm := NewHostKeyManager(knownHostsFile, false) // Non-strict mode

	addr := &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 22}
	err := hkm.VerifyHostKey("new.example.com", addr, testKey)
	assert.NoError(t, err)

	// Verify the key was added
	assert.Contains(t, hkm.knownHosts, "new.example.com")
}

// TestHostKeyManager_VerifyHostKey_ByIP tests verifying by IP address
func TestHostKeyManager_VerifyHostKey_ByIP(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	testKey := generateTestKey(t)
	hkm := NewHostKeyManager(knownHostsFile, false)

	// Add key by IP
	hkm.knownHosts["192.168.1.1"] = testKey

	// Verify by IP
	addr := &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 22}
	err := hkm.VerifyHostKey("example.com", addr, testKey)
	assert.NoError(t, err)
}

// TestHostKeyManager_AddHostKey tests adding a new host key
func TestHostKeyManager_AddHostKey(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	testKey := generateTestKey(t)
	hkm := NewHostKeyManager(knownHostsFile, false)

	err := hkm.addHostKey("example.com", testKey)
	assert.NoError(t, err)

	// Verify key was added to memory
	assert.Contains(t, hkm.knownHosts, "example.com")

	// Verify key was written to file
	content, err := os.ReadFile(knownHostsFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "example.com")
	assert.Contains(t, string(content), testKey.Type())
}

// TestHostKeyManager_AddHostKey_CreatesDirectory tests that directory is created
func TestHostKeyManager_AddHostKey_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "subdir", "known_hosts")

	testKey := generateTestKey(t)
	hkm := NewHostKeyManager(knownHostsFile, false)

	err := hkm.addHostKey("example.com", testKey)
	assert.NoError(t, err)

	// Verify directory was created
	_, err = os.Stat(filepath.Dir(knownHostsFile))
	assert.NoError(t, err)
}

// TestHostKeyManager_GetFingerprint tests getting key fingerprint
func TestHostKeyManager_GetFingerprint(t *testing.T) {
	hkm := NewHostKeyManager("", false)
	testKey := generateTestKey(t)

	fingerprint := hkm.GetFingerprint(testKey)
	assert.NotEmpty(t, fingerprint)
	assert.True(t, strings.HasPrefix(fingerprint, "SHA256:"))
}

// TestHostKeyManager_GetFingerprint_Consistency tests fingerprint consistency
func TestHostKeyManager_GetFingerprint_Consistency(t *testing.T) {
	hkm := NewHostKeyManager("", false)
	testKey := generateTestKey(t)

	fingerprint1 := hkm.GetFingerprint(testKey)
	fingerprint2 := hkm.GetFingerprint(testKey)

	assert.Equal(t, fingerprint1, fingerprint2)
}

// TestHostKeyManager_IsStrictMode tests getting strict mode
func TestHostKeyManager_IsStrictMode(t *testing.T) {
	tests := []struct {
		name       string
		strictMode bool
	}{
		{"strict mode enabled", true},
		{"strict mode disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hkm := NewHostKeyManager("", tt.strictMode)
			assert.Equal(t, tt.strictMode, hkm.IsStrictMode())
		})
	}
}

// TestHostKeyManager_SetStrictMode tests setting strict mode
func TestHostKeyManager_SetStrictMode(t *testing.T) {
	hkm := NewHostKeyManager("", false)
	assert.False(t, hkm.IsStrictMode())

	hkm.SetStrictMode(true)
	assert.True(t, hkm.IsStrictMode())

	hkm.SetStrictMode(false)
	assert.False(t, hkm.IsStrictMode())
}

// TestKeysEqual tests the keysEqual helper function
func TestKeysEqual(t *testing.T) {
	key1 := generateTestKey(t)
	key2 := generateTestKey(t)

	// Same key should be equal
	assert.True(t, keysEqual(key1, key1))

	// Different keys should not be equal
	assert.False(t, keysEqual(key1, key2))
}

// TestHostKeyManager_ConcurrentAccess tests concurrent access to host key manager
func TestHostKeyManager_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	hkm := NewHostKeyManager(knownHostsFile, false)
	testKey := generateTestKey(t)

	// Add initial key
	hkm.knownHosts["example.com"] = testKey

	// Concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			addr := &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 22}
			_ = hkm.VerifyHostKey("example.com", addr, testKey)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestHostKeyManager_LoadKnownHosts_InvalidLines tests handling invalid lines
func TestHostKeyManager_LoadKnownHosts_InvalidLines(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	testKey := generateTestKey(t)
	// MarshalAuthorizedKey returns "keytype base64data comment\n"
	// We need to extract just the keytype and base64data parts
	authorizedKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(testKey)))
	parts := strings.Fields(authorizedKey)
	validLine := fmt.Sprintf("example.com %s %s\n", parts[0], parts[1])

	content := "invalid line\n" +
		"too few fields\n" +
		validLine +
		"host invalid-key-type invalid-key-data\n"

	err := os.WriteFile(knownHostsFile, []byte(content), 0600)
	require.NoError(t, err)

	hkm := NewHostKeyManager(knownHostsFile, false)
	assert.NotNil(t, hkm)

	// Should only load the valid line
	assert.Len(t, hkm.knownHosts, 1)
	assert.Contains(t, hkm.knownHosts, "example.com")
}

// TestHostKeyManager_VerifyHostKey_NilAddr tests verification with nil address
func TestHostKeyManager_VerifyHostKey_NilAddr(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	testKey := generateTestKey(t)
	hkm := NewHostKeyManager(knownHostsFile, false)
	hkm.knownHosts["example.com"] = testKey

	// Verify with nil address (should still work if hostname matches)
	err := hkm.VerifyHostKey("example.com", nil, testKey)
	assert.NoError(t, err)
}

// customAddr is a custom address type for testing
type customAddr struct{}

func (customAddr) Network() string { return "custom" }
func (customAddr) String() string  { return "custom:0" }

// TestHostKeyManager_VerifyHostKey_NonTCPAddr tests verification with non-TCP address
func TestHostKeyManager_VerifyHostKey_NonTCPAddr(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	testKey := generateTestKey(t)
	hkm := NewHostKeyManager(knownHostsFile, false)
	hkm.knownHosts["example.com"] = testKey

	// Use a non-TCP address type
	err := hkm.VerifyHostKey("example.com", customAddr{}, testKey)
	assert.NoError(t, err)
}

// BenchmarkHostKeyManager_VerifyHostKey benchmarks host key verification
func BenchmarkHostKeyManager_VerifyHostKey(b *testing.B) {
	tmpDir := b.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	testKey := generateTestKey(b)
	hkm := NewHostKeyManager(knownHostsFile, false)
	hkm.knownHosts["example.com"] = testKey

	addr := &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 22}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hkm.VerifyHostKey("example.com", addr, testKey)
	}
}

// BenchmarkHostKeyManager_GetFingerprint benchmarks fingerprint generation
func BenchmarkHostKeyManager_GetFingerprint(b *testing.B) {
	hkm := NewHostKeyManager("", false)
	testKey := generateTestKey(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hkm.GetFingerprint(testKey)
	}
}

// TestHostKeyManager_AddHostKey_FilePermissions tests file permissions
func TestHostKeyManager_AddHostKey_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	testKey := generateTestKey(t)
	hkm := NewHostKeyManager(knownHostsFile, false)

	err := hkm.addHostKey("example.com", testKey)
	assert.NoError(t, err)

	// Check file permissions
	info, err := os.Stat(knownHostsFile)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

// TestHostKeyManager_AddHostKey_DirectoryPermissions tests directory permissions
func TestHostKeyManager_AddHostKey_DirectoryPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "subdir", "known_hosts")

	testKey := generateTestKey(t)
	hkm := NewHostKeyManager(knownHostsFile, false)

	err := hkm.addHostKey("example.com", testKey)
	assert.NoError(t, err)

	// Check directory permissions
	info, err := os.Stat(filepath.Dir(knownHostsFile))
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
}

// TestHostKeyManager_VerifyHostKey_IPMismatch tests IP address key mismatch
func TestHostKeyManager_VerifyHostKey_IPMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	testKey1 := generateTestKey(t)
	testKey2 := generateTestKey(t)

	hkm := NewHostKeyManager(knownHostsFile, false)
	hkm.knownHosts["192.168.1.1"] = testKey1

	// Verify with different key for same IP
	addr := &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 22}
	err := hkm.VerifyHostKey("example.com", addr, testKey2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key mismatch")
}
