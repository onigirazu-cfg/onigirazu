package ssh

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Client wraps SSH connection functionality
type Client struct {
	client *ssh.Client
	host   types.Host
}

// NewClient creates a new SSH client for the given host
func NewClient(host types.Host) (*Client, error) {
	return NewClientWithHostKeyManager(host, NewHostKeyManager("", false))
}

// NewClientWithHostKeyManager creates a new SSH client with custom host key manager
func NewClientWithHostKeyManager(host types.Host, hostKeyManager *HostKeyManager) (*Client, error) {
	var auth []ssh.AuthMethod

	// Try key-based authentication first
	if host.KeyFile != "" {
		key, err := os.ReadFile(host.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("unable to read private key: %v", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("unable to parse private key: %v", err)
		}

		auth = append(auth, ssh.PublicKeys(signer))
	}

	// Add password authentication if available
	if host.Password != "" {
		auth = append(auth, ssh.Password(host.Password))
	}

	if len(auth) == 0 {
		return nil, fmt.Errorf("no authentication method available for host %s", host.Name)
	}

	config := &ssh.ClientConfig{
		User:            host.User,
		Auth:            auth,
		HostKeyCallback: hostKeyManager.VerifyHostKey, // ✅ БЕЗОПАСНАЯ ПРОВЕРКА HOST KEY
		Timeout:         30 * time.Second,
	}

	// Default port to 22 if not specified
	port := host.Port
	if port == 0 {
		port = 22
	}

	address := fmt.Sprintf("%s:%d", host.Address, port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %v", address, err)
	}

	return &Client{
		client: client,
		host:   host,
	}, nil
}

// ExecuteCommand executes a command on the remote host
func (c *Client) ExecuteCommand(command string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %v", err)
	}

	return string(output), nil
}

// GetClient returns the underlying SSH client
func (c *Client) GetClient() *ssh.Client {
	return c.client
}

// Close closes the SSH connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// IsLocal checks if the host is localhost
func IsLocal(host types.Host) bool {
	if host.Address == "localhost" || host.Address == "127.0.0.1" || host.Address == "::1" {
		return true
	}

	// Check if it's the local machine's IP
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.String() == host.Address {
				return true
			}
		}
	}

	return false
}
