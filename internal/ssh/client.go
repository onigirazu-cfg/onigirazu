package ssh

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
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
	// Apply defaults if not specified
	if host.User == "" {
		host.User = os.Getenv("USER")
		if host.User == "" {
			host.User = "root"
		}
	}

	fmt.Printf("[DEBUG SSH] NewClientWithHostKeyManager called for host: %s, Address: %s, User: %s, KeyFile: %s\n",
		host.Name, host.Address, host.User, host.KeyFile)

	var auth []ssh.AuthMethod

	// Try key-based authentication first
	keyFile := host.KeyFile
	useDefaultKey := false
	if keyFile == "" {
		keyFile = getDefaultSSHKey()
		useDefaultKey = true
	}

	if keyFile != "" {
		fmt.Printf("[DEBUG SSH] Reading key file: %s\n", keyFile)
		key, err := os.ReadFile(keyFile)
		if err != nil {
			// If explicitly specified key file fails, return error
			if !useDefaultKey {
				return nil, fmt.Errorf("unable to read private key: %v", err)
			}
			fmt.Printf("[DEBUG SSH] Failed to read default key file: %v\n", err)
		} else {
			fmt.Printf("[DEBUG SSH] Key file read successfully, size: %d bytes\n", len(key))

			signer, err := ssh.ParsePrivateKey(key)
			if err != nil {
				// If explicitly specified key file fails to parse, return error
				if !useDefaultKey {
					return nil, fmt.Errorf("unable to parse private key: %v", err)
				}
				fmt.Printf("[DEBUG SSH] Failed to parse default private key: %v\n", err)
			} else {
				fmt.Printf("[DEBUG SSH] Private key parsed successfully\n")
				auth = append(auth, ssh.PublicKeys(signer))
			}
		}
	} else {
		fmt.Printf("[DEBUG SSH] No KeyFile specified for host %s\n", host.Name)
	}

	// Add password authentication if available
	if host.Password != "" {
		auth = append(auth, ssh.Password(host.Password))
		fmt.Printf("[DEBUG SSH] Password authentication added\n")
	}

	if len(auth) == 0 {
		fmt.Printf("[DEBUG SSH] ERROR: No authentication methods available for host %s\n", host.Name)
		return nil, fmt.Errorf("no authentication method available for host %s", host.Name)
	}

	fmt.Printf("[DEBUG SSH] Total authentication methods: %d\n", len(auth))

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
	fmt.Printf("[DEBUG SSH] Attempting to connect to %s as user %s\n", address, host.User)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		fmt.Printf("[DEBUG SSH] Connection failed: %v\n", err)
		return nil, fmt.Errorf("failed to connect to %s: %v", address, err)
	}
	fmt.Printf("[DEBUG SSH] Connection established successfully to %s\n", address)

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

// WriteFile writes data to a file on the remote host using SFTP
func (c *Client) WriteFile(remotePath string, data []byte, mode os.FileMode) error {
	// Create SFTP client
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %v", err)
	}
	defer sftpClient.Close()

	// Ensure parent directory exists
	remoteDir := filepath.Dir(remotePath)
	if err := sftpClient.MkdirAll(remoteDir); err != nil {
		return fmt.Errorf("failed to create remote directory %s: %v", remoteDir, err)
	}

	// Create remote file
	remoteFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file %s: %v", remotePath, err)
	}
	defer remoteFile.Close()

	// Write data
	if _, err := remoteFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to remote file %s: %v", remotePath, err)
	}

	// Set file permissions
	if err := sftpClient.Chmod(remotePath, mode); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %v", remotePath, err)
	}

	return nil
}

// ReadFile reads a file from the remote host using SFTP
func (c *Client) ReadFile(remotePath string) ([]byte, error) {
	// Create SFTP client
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create SFTP client: %v", err)
	}
	defer sftpClient.Close()

	// Open remote file
	remoteFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open remote file %s: %v", remotePath, err)
	}
	defer remoteFile.Close()

	// Read file contents
	data, err := io.ReadAll(remoteFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read remote file %s: %v", remotePath, err)
	}

	return data, nil
}

// StatFile gets file info from the remote host using SFTP
func (c *Client) StatFile(remotePath string) (os.FileInfo, error) {
	// Create SFTP client
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create SFTP client: %v", err)
	}
	defer sftpClient.Close()

	// Get file info
	fileInfo, err := sftpClient.Stat(remotePath)
	if err != nil {
		return nil, err
	}

	return fileInfo, nil
}

// getDefaultSSHKey returns the path to the default SSH key
func getDefaultSSHKey() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Try common SSH key locations in order of preference
	keyPaths := []string{
		filepath.Join(homeDir, ".ssh", "id_ed25519"),
		filepath.Join(homeDir, ".ssh", "id_rsa"),
		filepath.Join(homeDir, ".ssh", "id_ecdsa"),
		filepath.Join(homeDir, ".ssh", "id_dsa"),
	}

	for _, keyPath := range keyPaths {
		if _, err := os.Stat(keyPath); err == nil {
			return keyPath
		}
	}

	return ""
}

// CopyFile copies a file from local to remote host using SFTP
func (c *Client) CopyFile(localPath, remotePath string, mode os.FileMode) error {
	// Read local file
	// #nosec G304 - localPath is provided by user configuration and is intentional
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file %s: %v", localPath, err)
	}

	// Write to remote
	return c.WriteFile(remotePath, data, mode)
}
