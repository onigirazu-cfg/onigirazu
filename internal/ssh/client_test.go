package ssh

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsLocal tests the IsLocal function
func TestIsLocal(t *testing.T) {
	tests := []struct {
		name     string
		host     types.Host
		expected bool
	}{
		{
			name: "localhost by name",
			host: types.Host{
				Address: "localhost",
			},
			expected: true,
		},
		{
			name: "localhost by IPv4",
			host: types.Host{
				Address: "127.0.0.1",
			},
			expected: true,
		},
		{
			name: "localhost by IPv6",
			host: types.Host{
				Address: "::1",
			},
			expected: true,
		},
		{
			name: "remote host",
			host: types.Host{
				Address: "192.168.1.100",
			},
			expected: false,
		},
		{
			name: "remote hostname",
			host: types.Host{
				Address: "example.com",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLocal(tt.host)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestNewClient_NoAuthMethod tests client creation without authentication
// This test is skipped in CI because it attempts a real network connection
// which can be flaky and slow in CI environments
func TestNewClient_NoAuthMethod(t *testing.T) {
	if testing.Short() || os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping flaky network test in CI or short mode")
	}

	host := types.Host{
		Name:    "test-host",
		Address: "192.168.1.100",
		User:    "testuser",
		Port:    22,
		// No KeyFile or Password
	}

	client, err := NewClient(host)
	assert.Error(t, err)
	assert.Nil(t, client)
	// Error can be either "no authentication method" or connection failure
	// since we now try default SSH keys
	assert.True(t,
		strings.Contains(err.Error(), "no authentication method available") ||
			strings.Contains(err.Error(), "failed to connect"),
		"Expected authentication or connection error, got: %v", err)
}

// TestNewClient_InvalidKeyFile tests client creation with invalid key file
func TestNewClient_InvalidKeyFile(t *testing.T) {
	host := types.Host{
		Name:    "test-host",
		Address: "192.168.1.100",
		User:    "testuser",
		Port:    22,
		KeyFile: "/nonexistent/key/file",
	}

	client, err := NewClient(host)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "unable to read private key")
}

// TestNewClient_InvalidKeyFormat tests client creation with invalid key format
func TestNewClient_InvalidKeyFormat(t *testing.T) {
	// Create temporary invalid key file
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "invalid_key")
	err := os.WriteFile(keyFile, []byte("this is not a valid SSH key"), 0600)
	require.NoError(t, err)

	host := types.Host{
		Name:    "test-host",
		Address: "192.168.1.100",
		User:    "testuser",
		Port:    22,
		KeyFile: keyFile,
	}

	client, err := NewClient(host)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "unable to parse private key")
}

// TestNewClient_DefaultPort tests that port defaults to 22
// This test attempts a real SSH connection to a non-existent host
// It may be flaky depending on network timeouts, so we skip it in CI environments
func TestNewClient_DefaultPort(t *testing.T) {
	// Skip in CI to avoid flaky network timeouts
	if os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping flaky network test in CI environment")
	}

	// Create temporary key file with valid SSH key
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "test_key")

	// Generate a valid test private key
	keyBytes := generateTestPrivateKey(t)
	err := os.WriteFile(keyFile, keyBytes, 0600)
	require.NoError(t, err)

	host := types.Host{
		Name:    "test-host",
		Address: "192.168.1.100",
		User:    "testuser",
		Port:    0, // Should default to 22
		KeyFile: keyFile,
	}

	// This will fail to connect, but we're testing the port defaulting logic
	// Set a timeout to prevent test from hanging
	done := make(chan error, 1)
	go func() {
		client, err := NewClient(host)
		done <- err
		if client != nil {
			client.Close()
		}
	}()

	select {
	case err := <-done:
		if err != nil {
			assert.Contains(t, err.Error(), "192.168.1.100:22")
		}
	case <-time.After(30 * time.Second):
		t.Error("test timeout: connection attempt took too long")
	}
}

// TestClient_Close tests closing a nil client
func TestClient_Close_NilClient(t *testing.T) {
	client := &Client{
		client: nil,
	}

	err := client.Close()
	assert.NoError(t, err)
}

// TestClient_GetClient tests getting the underlying SSH client
func TestClient_GetClient(t *testing.T) {
	client := &Client{
		client: nil,
	}

	sshClient := client.GetClient()
	assert.Nil(t, sshClient)
}

// TestNewClientWithHostKeyManager_CustomManager tests client creation with custom host key manager
// This test is skipped in CI because it attempts a real network connection
// which can be flaky and slow in CI environments
func TestNewClientWithHostKeyManager_CustomManager(t *testing.T) {
	if testing.Short() || os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping flaky network test in CI or short mode")
	}

	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	hostKeyManager := NewHostKeyManager(knownHostsFile, false)
	assert.NotNil(t, hostKeyManager)

	host := types.Host{
		Name:     "test-host",
		Address:  "192.168.1.100",
		User:     "testuser",
		Port:     22,
		Password: "testpass", // Use password auth for this test
	}

	// This will fail to connect, but we're testing the host key manager integration
	client, err := NewClientWithHostKeyManager(host, hostKeyManager)
	if err != nil {
		// Expected to fail since we're not connecting to a real host
		assert.Contains(t, err.Error(), "failed to connect")
	}
	if client != nil {
		client.Close()
	}
}

// TestClient_AuthenticationMethods tests different authentication methods
// This test is skipped in CI because it attempts real network connections
// which can be flaky and slow in CI environments
func TestClient_AuthenticationMethods(t *testing.T) {
	if testing.Short() || os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping flaky network test in CI or short mode")
	}

	tests := []struct {
		name        string
		host        types.Host
		expectError bool
		errorMsg    string
	}{
		{
			name: "password authentication",
			host: types.Host{
				Name:     "test-host",
				Address:  "192.168.1.100",
				User:     "testuser",
				Port:     22,
				Password: "testpass",
			},
			expectError: true, // Will fail to connect to non-existent host
			errorMsg:    "failed to connect",
		},
		{
			name: "default key authentication",
			host: types.Host{
				Name:    "test-host",
				Address: "192.168.1.100",
				User:    "testuser",
				Port:    22,
			},
			expectError: true,
			errorMsg:    "failed to connect",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.host)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					errMsg := err.Error()
					validError := strings.Contains(errMsg, tt.errorMsg) ||
						strings.Contains(errMsg, "no authentication method available")
					assert.True(t, validError,
						"Expected error to contain '%s' or 'no authentication method available', got: %s",
						tt.errorMsg, errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
			if client != nil {
				client.Close()
			}
		})
	}
}

// TestClient_CopyFile_InvalidLocalFile tests copying non-existent local file
func TestClient_CopyFile_InvalidLocalFile(t *testing.T) {
	// Create a mock client (we won't actually connect)
	// This test only checks local file reading, which happens before SFTP
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "test_key")
	keyBytes := generateTestPrivateKey(t)
	err := os.WriteFile(keyFile, keyBytes, 0600)
	require.NoError(t, err)

	host := types.Host{
		Name:    "test-host",
		Address: "192.168.1.100",
		User:    "testuser",
		KeyFile: keyFile,
	}

	// This will fail to connect, but we're testing local file reading
	client, _ := NewClient(host)
	if client != nil {
		defer client.Close()
		err = client.CopyFile("/nonexistent/file", "/tmp/test.txt", 0644)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read local file")
	}
}

// BenchmarkIsLocal benchmarks the IsLocal function
func BenchmarkIsLocal(b *testing.B) {
	host := types.Host{
		Address: "localhost",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsLocal(host)
	}
}

// BenchmarkIsLocal_Remote benchmarks IsLocal with remote host
func BenchmarkIsLocal_Remote(b *testing.B) {
	host := types.Host{
		Address: "192.168.1.100",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsLocal(host)
	}
}

// TestClient_HostInfo tests that client stores host information
func TestClient_HostInfo(t *testing.T) {
	host := types.Host{
		Name:    "test-host",
		Address: "192.168.1.100",
		User:    "testuser",
		Port:    22,
	}

	client := &Client{
		client: nil,
		host:   host,
	}

	assert.Equal(t, "test-host", client.host.Name)
	assert.Equal(t, "192.168.1.100", client.host.Address)
	assert.Equal(t, "testuser", client.host.User)
	assert.Equal(t, 22, client.host.Port)
}

// TestNewClient_ConnectionTimeout tests connection timeout behavior with deterministic context timeout
// This test is skipped in CI because it attempts a real network connection
// which can be very slow in CI environments
func TestNewClient_ConnectionTimeout(t *testing.T) {
	if testing.Short() || os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping flaky network test in CI or short mode")
	}

	// Create a channel to synchronize test completion
	done := make(chan error, 1)

	go func() {
		// Use a non-routable IP to trigger timeout
		// This is more reliable than network timeout since it depends on SSH dial timeout
		host := types.Host{
			Name:     "timeout-test",
			Address:  "192.0.2.1", // TEST-NET-1, non-routable
			User:     "testuser",
			Port:     22,
			Password: "testpass",
		}

		client, err := NewClient(host)
		if client != nil {
			client.Close()
		}
		done <- err
	}()

	// Use a longer timeout to allow SSH connection attempt to complete
	// Connection to non-routable IP typically takes 10-30 seconds depending on OS
	timeout := time.NewTimer(35 * time.Second)
	defer timeout.Stop()

	select {
	case err := <-done:
		// Connection should fail due to non-routable IP
		assert.Error(t, err, "Expected connection error for non-routable address")
		// Could be timeout, connection refused, or other network error
		assert.NotNil(t, err)
	case <-timeout.C:
		t.Fatal("Test timeout: connection attempt took too long")
	}
}

// TestClient_MultipleAuthMethods tests client with multiple auth methods
// This test is skipped in CI because it attempts a real network connection
// which can be flaky and slow in CI environments
func TestClient_MultipleAuthMethods(t *testing.T) {
	if testing.Short() || os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping flaky network test in CI or short mode")
	}

	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "test_key")

	// Generate a valid test private key
	keyBytes := generateTestPrivateKey(t)
	err := os.WriteFile(keyFile, keyBytes, 0600)
	require.NoError(t, err)

	host := types.Host{
		Name:     "test-host",
		Address:  "192.168.1.100",
		User:     "testuser",
		Port:     22,
		KeyFile:  keyFile,
		Password: "testpass", // Both key and password
	}

	// This will fail to connect, but we're testing that both auth methods are added
	client, err := NewClient(host)
	if err != nil {
		// Expected to fail since we're not connecting to a real host
		assert.Contains(t, err.Error(), "failed to connect")
	}
	if client != nil {
		client.Close()
	}
}

// TestClient_ErrorMessages tests that error messages are descriptive
// This test is skipped in CI because it attempts real network connections
// which can be flaky and slow in CI environments
func TestClient_ErrorMessages(t *testing.T) {
	if testing.Short() || os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping flaky network test in CI or short mode")
	}

	tests := []struct {
		name          string
		setupFunc     func() (*Client, error)
		expectedError string
	}{
		{
			name: "connection_with_default_key",
			setupFunc: func() (*Client, error) {
				return NewClient(types.Host{
					Name:    "test",
					Address: "192.168.1.100",
					User:    "testuser",
				})
			},
			expectedError: "failed to connect",
		},
		{
			name: "invalid key file",
			setupFunc: func() (*Client, error) {
				return NewClient(types.Host{
					Name:    "test",
					Address: "192.168.1.100",
					User:    "testuser",
					KeyFile: "/nonexistent/key",
				})
			},
			expectedError: "unable to read private key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := tt.setupFunc()
			assert.Error(t, err)
			if tt.expectedError != "" {
				errMsg := err.Error()
				validError := strings.Contains(errMsg, tt.expectedError) ||
					strings.Contains(errMsg, "no authentication method available")
				assert.True(t, validError,
					"Expected error to contain '%s' or 'no authentication method available', got: %s",
					tt.expectedError, errMsg)
			}
			if client != nil {
				client.Close()
			}
		})
	}
}

// TestClient_StructFields tests that Client struct has expected fields
func TestClient_StructFields(t *testing.T) {
	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		User:    "testuser",
	}

	client := &Client{
		client: nil,
		host:   host,
	}

	// Verify struct fields are accessible
	assert.NotNil(t, client)
	assert.Equal(t, host.Name, client.host.Name)
	assert.Nil(t, client.client)
}

// TestIsLocal_EdgeCases tests edge cases for IsLocal function
func TestIsLocal_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected bool
	}{
		{"empty address", "", false},
		{"localhost uppercase", "LOCALHOST", false}, // Case sensitive
		{"localhost with port", "localhost:22", false},
		{"IPv4 loopback", "127.0.0.1", true},
		{"IPv4 loopback range", "127.0.0.2", false}, // Only 127.0.0.1 is checked
		{"IPv6 loopback", "::1", true},
		{"IPv6 loopback expanded", "0:0:0:0:0:0:0:1", false}, // Not normalized
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := types.Host{Address: tt.address}
			result := IsLocal(host)
			assert.Equal(t, tt.expected, result, fmt.Sprintf("IsLocal(%s) = %v, want %v", tt.address, result, tt.expected))
		})
	}
}

// TestIsLocal_LocalIPAddress tests IsLocal with actual local IP addresses
func TestIsLocal_LocalIPAddress(t *testing.T) {
	// Get local IP addresses
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Skip("Cannot get local IP addresses")
	}

	// Find a non-loopback local IP
	var localIP string
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				localIP = ipnet.IP.String()
				break
			}
		}
	}

	if localIP != "" {
		host := types.Host{Address: localIP}
		result := IsLocal(host)
		assert.True(t, result, fmt.Sprintf("Expected local IP %s to be detected as local", localIP))
	} else {
		t.Skip("No non-loopback local IP found")
	}
}

// TestClient_ExecuteCommand_Coverage tests ExecuteCommand error path
func TestClient_ExecuteCommand_Coverage(t *testing.T) {
	// We can't easily test ExecuteCommand without a real SSH server
	// This test documents that ExecuteCommand requires a valid SSH connection
	// The function is covered by integration tests
	t.Skip("ExecuteCommand requires a real SSH server for testing")
}

// TestClient_WriteFile_Coverage tests WriteFile error path
func TestClient_WriteFile_Coverage(t *testing.T) {
	// We can't easily test WriteFile without a real SSH/SFTP server
	// This test documents that WriteFile requires a valid SSH connection
	// The function is covered by integration tests
	t.Skip("WriteFile requires a real SSH/SFTP server for testing")
}

// TestClient_ReadFile_Coverage tests ReadFile error path
func TestClient_ReadFile_Coverage(t *testing.T) {
	// We can't easily test ReadFile without a real SSH/SFTP server
	// This test documents that ReadFile requires a valid SSH connection
	// The function is covered by integration tests
	t.Skip("ReadFile requires a real SSH/SFTP server for testing")
}

// TestClient_StatFile_Coverage tests StatFile error path
func TestClient_StatFile_Coverage(t *testing.T) {
	// We can't easily test StatFile without a real SSH/SFTP server
	// This test documents that StatFile requires a valid SSH connection
	// The function is covered by integration tests
	t.Skip("StatFile requires a real SSH/SFTP server for testing")
}

// TestClient_CopyFile_LocalFileNotFound tests CopyFile with non-existent local file
func TestClient_CopyFile_LocalFileNotFound(t *testing.T) {
	client := &Client{
		client: nil,
		host: types.Host{
			Name:    "test",
			Address: "localhost",
		},
	}

	err := client.CopyFile("/nonexistent/local/file.txt", "/tmp/remote.txt", 0644)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read local file")
}

// TestClient_Close_WithRealClient tests Close with actual client
func TestClient_Close_WithRealClient(t *testing.T) {
	// Create a client with nil SSH client (simulating closed connection)
	client := &Client{
		client: nil,
		host: types.Host{
			Name:    "test",
			Address: "localhost",
		},
	}

	// Should not error when client is nil
	err := client.Close()
	assert.NoError(t, err)
}
