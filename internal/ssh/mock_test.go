package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// MockSSHServer provides a test SSH server for unit testing SSH client functionality
// without requiring a real SSH server or network connection
type MockSSHServer struct {
	listener net.Listener
	config   *ssh.ServerConfig
	done     chan bool
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
}

// MockSSHSession represents a mock SSH session
type MockSSHSession struct {
	cmd   string
	env   map[string]string
	stdin io.Reader
}

// NewMockSSHServer creates a new mock SSH server for testing
func NewMockSSHServer(port int) (*MockSSHServer, error) {
	config := &ssh.ServerConfig{
		// Accept any authentication method for testing
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			return &ssh.Permissions{}, nil
		},
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			return &ssh.Permissions{}, nil
		},
	}

	// Generate a test private key for the server
	privateKey, err := generateTestPrivateKeyForServer()
	if err != nil {
		return nil, fmt.Errorf("failed to generate server key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server key: %w", err)
	}

	config.AddHostKey(signer)

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	srv := &MockSSHServer{
		listener: listener,
		config:   config,
		done:     make(chan bool),
		running:  true,
	}

	// Start accepting connections in a goroutine
	srv.wg.Add(1)
	go srv.acceptConnections()

	return srv, nil
}

// acceptConnections handles incoming SSH connections
func (m *MockSSHServer) acceptConnections() {
	defer m.wg.Done()

	for {
		select {
		case <-m.done:
			return
		default:
		}

		conn, err := m.listener.Accept()
		if err != nil {
			continue
		}

		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.handleConnection(conn)
		}()
	}
}

// handleConnection handles a single SSH connection
func (m *MockSSHServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	_, _, _, err := ssh.NewServerConn(conn, m.config)
	if err != nil {
		return
	}
}

// Close closes the mock SSH server
func (m *MockSSHServer) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	m.running = false
	close(m.done)

	err := m.listener.Close()
	m.wg.Wait()

	return err
}

// GetAddress returns the address the mock server is listening on
func (m *MockSSHServer) GetAddress() string {
	return m.listener.Addr().String()
}

// MockSSHConnector provides a mock SSH connector for testing
type MockSSHConnector struct {
	shouldFail bool
	failMsg    string
}

// NewMockSSHConnector creates a new mock SSH connector
func NewMockSSHConnector() *MockSSHConnector {
	return &MockSSHConnector{
		shouldFail: false,
	}
}

// SetShouldFail configures the connector to fail connections
func (m *MockSSHConnector) SetShouldFail(fail bool, msg string) {
	m.shouldFail = fail
	m.failMsg = msg
}

// WithContext returns a context-aware mock connection for testing timeouts
type MockConnectionContext struct {
	ctx      context.Context
	cancelFn context.CancelFunc
	timeout  time.Duration
}

// NewMockConnectionContext creates a new mock connection context
func NewMockConnectionContext(timeout time.Duration) *MockConnectionContext {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return &MockConnectionContext{
		ctx:      ctx,
		cancelFn: cancel,
		timeout:  timeout,
	}
}

// Cancel cancels the mock context
func (m *MockConnectionContext) Cancel() {
	m.cancelFn()
}

// GetContext returns the context
func (m *MockConnectionContext) GetContext() context.Context {
	return m.ctx
}

// GetTimeout returns the timeout duration
func (m *MockConnectionContext) GetTimeout() time.Duration {
	return m.timeout
}

// generateTestPrivateKeyForServer generates a test private key for the mock server
func generateTestPrivateKeyForServer() ([]byte, error) {
	// This is a test key - never use in production!
	// This is a minimal ED25519 test key
	const testKey = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUtbm9uZS1ub25lAAAAAEAAAAzAAAAC2VjZHNhLXNoYTIt
bmlzdHAyNTYAAAAIbmlzdHAyNTYAAABIBCA7P+h3TLjWJi3Y5uB+vLo4P1rqmhYB/wt7BsZA2lLR
/wEAAaCAAAAAw1CAEADAAAA0AAAAL2qIUXH6WH2J/vBcXvD8kKtpKCNCPWJmKXQKXKbx0AAAA0
Lw4dMQgGHDEkwgQlEBgCAAAAAQAAADMAAAAAy+TJhCpLmhX7r5g7Z1qK6Jx3ygHDpJ0bVTQ6HnCa
kNgAEAAAADQAAAA9VZLJ9RgGx2g4AEA==
-----END OPENSSH PRIVATE KEY-----`
	return []byte(testKey), nil
}
