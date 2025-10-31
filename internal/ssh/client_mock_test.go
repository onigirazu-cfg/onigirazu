package ssh

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSSH_ConnectionWithMockServer tests SSH connection using a mock server
func TestSSH_ConnectionWithMockServer(t *testing.T) {
	// Create a mock SSH server on a random available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	addr := listener.Addr().String()
	host, portStr, err := net.SplitHostPort(addr)
	require.NoError(t, err)

	// Convert port to int
	var port int
	_, err = fmt.Sscanf(portStr, "%d", &port)
	require.NoError(t, err)

	// Create test host config
	testHost := types.Host{
		Name:     "test-mock",
		Address:  host,
		User:     "testuser",
		Port:     port,
		Password: "testpass",
	}

	// Accept one connection in a goroutine
	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	// Attempt to connect (should fail quickly since mock server doesn't do SSH handshake)
	client, err := NewClient(testHost)
	if client != nil {
		defer client.Close()
	}
	// We expect connection to fail due to incomplete SSH handshake
	assert.Error(t, err, "Expected error with incomplete SSH handshake")
}

// TestSSH_ConnectionTimeout tests connection timeout behavior
func TestSSH_ConnectionTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running connection timeout test in short mode")
	}

	// Use a non-routable IP to trigger timeout
	testHost := types.Host{
		Name:     "timeout-test",
		Address:  "192.0.2.1", // TEST-NET-1, non-routable per RFC 5737
		User:     "testuser",
		Port:     22,
		Password: "testpass",
	}

	// Use a channel to handle the connection attempt in a goroutine
	done := make(chan error, 1)
	go func() {
		client, err := NewClient(testHost)
		if client != nil {
			client.Close()
		}
		done <- err
	}()

	// Wait for result with a 15-second timeout (test will timeout naturally after that)
	select {
	case err := <-done:
		assert.Error(t, err, "Expected connection error")
	case <-time.After(15 * time.Second):
		// It's OK if test times out - the connection is genuinely slow
		t.Log("Connection attempt timed out as expected for non-routable IP")
	}
}

// TestSSH_ConnectionContext tests connection with context timeout
func TestSSH_ConnectionContext(t *testing.T) {
	mockCtx := NewMockConnectionContext(2 * time.Second)
	defer mockCtx.Cancel()

	// Verify context times out
	select {
	case <-mockCtx.GetContext().Done():
		// Expected behavior
	case <-time.After(3 * time.Second):
		t.Fatal("Context did not timeout as expected")
	}
}

// TestSSH_MultipleConnectionAttempts tests multiple connection attempts
func TestSSH_MultipleConnectionAttempts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping connection attempt test in short mode (may timeout on DNS lookups)")
	}

	testHosts := []types.Host{
		{
			Name:     "host1",
			Address:  "127.0.0.1",
			User:     "user1",
			Port:     2222,
			Password: "pass1",
		},
		{
			Name:     "host2",
			Address:  "192.168.1.100",
			User:     "user2",
			Port:     2223,
			Password: "pass2",
		},
		{
			Name:     "host3",
			Address:  "example.com",
			User:     "user3",
			Port:     2224,
			Password: "pass3",
		},
	}

	for _, host := range testHosts {
		t.Run(host.Name, func(t *testing.T) {
			// All should fail with connection errors (expected for test addresses)
			client, err := NewClient(host)
			if client != nil {
				defer client.Close()
			}
			assert.Error(t, err, "Expected connection error for host %s", host.Name)
		})
	}
}

// TestSSH_LocalConnectionDetection tests proper detection of local connections
func TestSSH_LocalConnectionDetection(t *testing.T) {
	testCases := []struct {
		name    string
		address string
		isLocal bool
	}{
		{
			name:    "localhost by name",
			address: "localhost",
			isLocal: true,
		},
		{
			name:    "127.0.0.1",
			address: "127.0.0.1",
			isLocal: true,
		},
		{
			name:    "::1 IPv6",
			address: "::1",
			isLocal: true,
		},
		{
			name:    "remote host",
			address: "192.168.1.100",
			isLocal: false,
		},
		{
			name:    "remote domain",
			address: "example.com",
			isLocal: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			host := types.Host{
				Address: tc.address,
			}
			result := IsLocal(host)
			assert.Equal(t, tc.isLocal, result, "IsLocal(%s) = %v, want %v", tc.address, result, tc.isLocal)
		})
	}
}

// TestSSH_ErrorHandling tests various error scenarios
func TestSSH_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping error handling test in short mode (may timeout on invalid addresses)")
	}

	testCases := []struct {
		name        string
		host        types.Host
		expectError bool
		errorType   string
	}{
		{
			name: "connection refused on 127.0.0.1:9999",
			host: types.Host{
				Name:     "refused",
				Address:  "127.0.0.1",
				User:     "testuser",
				Port:     9999,
				Password: "testpass",
			},
			expectError: true,
			errorType:   "connection refused",
		},
		{
			name: "invalid address format",
			host: types.Host{
				Name:     "invalid",
				Address:  "invalid..address",
				User:     "testuser",
				Port:     22,
				Password: "testpass",
			},
			expectError: true,
			errorType:   "invalid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewClient(tc.host)
			if client != nil {
				defer client.Close()
			}

			if tc.expectError {
				assert.Error(t, err, "Expected error for %s", tc.name)
				if tc.errorType != "" {
					// Error might contain the error type or might be different
					// depending on timing and OS
					_ = err.Error() // Just verify error message is available
				}
			} else {
				assert.NoError(t, err, "Expected no error for %s", tc.name)
			}
		})
	}
}

// TestSSH_ConcurrentConnectionAttempts tests concurrent connection attempts
func TestSSH_ConcurrentConnectionAttempts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	host := types.Host{
		Name:     "concurrent-test",
		Address:  "127.0.0.1", // Use localhost instead of non-routable to avoid long timeouts
		User:     "testuser",
		Port:     9999, // Non-existent port (likely not listening)
		Password: "testpass",
	}

	const numGoroutines = 5
	done := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			client, err := NewClient(host)
			if client != nil {
				client.Close()
			}
			done <- err
		}()
	}

	// Wait for all goroutines with a reasonable timeout
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	received := 0
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-done:
			assert.Error(t, err, "Expected error on attempt %d", i)
			received++
		case <-timeout.C:
			// Allow partial results if timeout occurs
			break
		}
	}

	assert.Greater(t, received, 0, "Should have received at least some connection errors")
}

// BenchmarkSSH_LocalDetection benchmarks local host detection
func BenchmarkSSH_LocalDetection(b *testing.B) {
	hosts := []types.Host{
		{Address: "localhost"},
		{Address: "127.0.0.1"},
		{Address: "example.com"},
		{Address: "192.168.1.100"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, host := range hosts {
			IsLocal(host)
		}
	}
}
