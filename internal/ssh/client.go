package ssh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Logger interface for dependency injection
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
	Warn(format string, args ...interface{})
}

// Client wraps SSH connection functionality
type Client struct {
	client *ssh.Client
	// jumps: connections to the ProxyJump hosts, closed with the client
	jumps  []*ssh.Client
	host   types.Host
	logger Logger
	shells shellPool
	// asShells: command servers started once with sudo -n -u <user>, so
	// become commands skip a sudo each
	asMu     sync.Mutex
	asShells map[string]*shellPool
	agent    agentState
	// prewarmed: the users whose command server Prewarm started
	prewarmed sync.Map
	// closed: Close was called (the run stops); commands still running
	// then fail with ErrClosed instead of a bare EOF
	closed atomic.Bool
}

// ErrClosed: the connection was closed by onigirazu while a command ran
// (the run was stopped: Ctrl-C, --timeout, a signal)
var ErrClosed = errors.New("connection closed while the command ran (the run was stopped)")

// NewClient creates a new SSH client for the given host
func NewClient(host types.Host) (*Client, error) {
	return NewClientWithLogger(host, logger.New(false))
}

// NewClientWithLogger creates a new SSH client with a custom logger
func NewClientWithLogger(host types.Host, lg Logger) (*Client, error) {
	// the known_hosts file and strict mode of the configuration, as for
	// pooled connections (hostKeyCallback honors the host's insecure flag)
	return NewClientWithHostKeyManagerAndLogger(host, GetGlobalPool().hostKeyMgr, lg)
}

// hostKeyCallback checks the host key with the manager, unless the host
// turns the check off (insecure_ignore_host_key, StrictHostKeyChecking=no)
func hostKeyCallback(host types.Host, m *HostKeyManager) ssh.HostKeyCallback {
	insecure := host.InsecureIgnoreHostKey
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if insecure {
			return nil // the inventory turned the check off for this host
		}
		return m.VerifyHostKey(hostname, remote, key)
	}
}

// HostKeyCallbackFor is the host key check of the configuration (known_hosts,
// strict mode) for a host, honoring its insecure flag: for SSH connections
// made outside the client (a device's CLI, say)
func HostKeyCallbackFor(host types.Host) ssh.HostKeyCallback {
	return hostKeyCallback(host, GetGlobalPool().hostKeyMgr)
}

// NewClientWithHostKeyManager creates a new SSH client with custom host key manager (deprecated, use NewClientWithHostKeyManagerAndLogger)
func NewClientWithHostKeyManager(host types.Host, hostKeyManager *HostKeyManager) (*Client, error) {
	return NewClientWithHostKeyManagerAndLogger(host, hostKeyManager, logger.New(false))
}

// NewClientWithHostKeyManagerAndLogger creates a new SSH client with custom host key manager and logger
func NewClientWithHostKeyManagerAndLogger(host types.Host, hostKeyManager *HostKeyManager, lg Logger) (*Client, error) {
	if lg == nil {
		lg = logger.New(false)
	}

	// Apply defaults if not specified
	if host.User == "" {
		host.User = os.Getenv("USER")
		if host.User == "" {
			host.User = "root"
		}
	}

	lg.Debug("NewClientWithHostKeyManager called for host: %s, Address: %s, User: %s, KeyFile: %s",
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
		lg.Debug("Reading key file: %s", keyFile)
		key, err := os.ReadFile(keyFile) // #nosec G304 - keyFile is from trusted inventory configuration
		if err != nil {
			// If explicitly specified key file fails, return error
			if !useDefaultKey {
				return nil, fmt.Errorf("unable to read private key: %v", err)
			}
			lg.Debug("Failed to read default key file: %v", err)
		} else {
			lg.Debug("Key file read successfully, size: %d bytes", len(key))

			signer, err := ssh.ParsePrivateKey(key)
			if err != nil {
				// If explicitly specified key file fails to parse, return error
				if !useDefaultKey {
					return nil, fmt.Errorf("unable to parse private key: %v", err)
				}
				lg.Debug("Failed to parse default private key: %v", err)
			} else {
				lg.Debug("Private key parsed successfully")
				auth = append(auth, ssh.PublicKeys(signer))
			}
		}
	} else {
		lg.Debug("No KeyFile specified for host %s", host.Name)
	}

	// Add password authentication if available
	if host.Password != "" {
		auth = append(auth, ssh.Password(host.Password))
		lg.Debug("Password authentication added")
	}

	if len(auth) == 0 {
		lg.Debug("ERROR: No authentication methods available for host %s", host.Name)
		return nil, fmt.Errorf("no authentication method available for host %s", host.Name)
	}

	lg.Debug("Total authentication methods: %d", len(auth))

	config := &ssh.ClientConfig{
		User:            host.User,
		Auth:            auth,
		HostKeyCallback: hostKeyCallback(host, hostKeyManager),
		Timeout:         dialTimeout,
	}

	opts := parseSSHArgs(host.SSHArgs)
	if opts.ConnectTimeout > 0 {
		config.Timeout = opts.ConnectTimeout
	}
	address := dialAddress(host)
	lg.Debug("Attempting to connect to %s as user %s", address, host.User)
	client, jumps, err := dialHost(host, address, config, auth, hostKeyManager, opts)
	if err != nil {
		lg.Debug("Connection failed: %v", err)
		return nil, fmt.Errorf("failed to connect to %s: %v", address, err)
	}
	lg.Debug("Connection established successfully to %s", address)

	return &Client{
		client: client,
		jumps:  jumps,
		host:   host,
		logger: lg,
	}, nil
}

// ExecuteCommand executes a command on the remote host
func (c *Client) ExecuteCommand(command string) (string, error) {
	out, _, rc, err := c.Exec(context.Background(), command, true)
	if err != nil {
		return string(out), fmt.Errorf("command failed: %w", err)
	}
	if rc != 0 {
		return string(out), fmt.Errorf("command failed: %w", &ExitStatusError{Status: rc})
	}
	return string(out), nil
}

// GetClient returns the underlying SSH client
func (c *Client) GetClient() *ssh.Client {
	return c.client
}

// Close closes the SSH connection
func (c *Client) Close() error {
	c.closed.Store(true)
	c.closeShells()
	var err error
	if c.client != nil {
		err = c.client.Close()
	}
	closeAll(c.jumps)
	return err
}

// HealthCheck sends a ping to verify the connection is alive
// Returns true if connection is healthy, false otherwise
func (c *Client) HealthCheck(timeout time.Duration) bool {
	if c.client == nil {
		return false
	}

	// Create a channel for the result
	result := make(chan bool, 1)

	// Send the health check in a goroutine with timeout
	go func() {
		// Try to send a keepalive message by executing a simple command
		// Using 'true' command which always succeeds and has minimal overhead
		session, err := c.client.NewSession()
		if err != nil {
			result <- isChannelRefused(err)
			return
		}
		defer session.Close()

		// Execute a lightweight command with minimal output
		err = session.Run("true")
		result <- err == nil
	}()

	// Wait for result or timeout
	select {
	case ok := <-result:
		return ok
	case <-time.After(timeout):
		c.logger.Warn("Health check timeout for host %s", c.host.Address)
		return false
	}
}

// IsAlive checks if the client connection is still active
// This is a quick check using SSH channel open
func (c *Client) IsAlive() bool {
	if c.client == nil {
		return false
	}

	// Try to open a channel which will fail immediately if connection is dead
	session, err := c.client.NewSession()
	if err != nil {
		// refused (sshd's MaxSessions): the connection answered
		return isChannelRefused(err)
	}
	_ = session.Close()
	return true
}

// Container returns the container runtime (docker or podman) and the
// container a host is reached through with ansible_connection docker/podman
// (community.docker.docker, containers.podman.podman): the host's address,
// else its name, as in Ansible. ok is false for any other connection.
func Container(host types.Host) (runtime, name string, ok bool) {
	for _, key := range []string{"onigirazu_connection", "ansible_connection"} {
		switch fmt.Sprint(host.Vars[key]) {
		case "docker", "community.docker.docker":
			runtime = "docker"
		case "podman", "containers.podman.podman":
			runtime = "podman"
		default:
			continue
		}
		name = host.Address
		if name == "" {
			name = host.Name
		}
		return runtime, name, true
	}
	return "", "", false
}

// IsLocal checks if the host is localhost
func IsLocal(host types.Host) bool {
	// An explicit connection type wins
	for _, key := range []string{"onigirazu_connection", "ansible_connection"} {
		switch host.Vars[key] {
		case "local":
			return true
		case "ssh":
			return false
		}
	}
	// a container is not this machine, and has no SSH either
	if _, _, ok := Container(host); ok {
		return false
	}

	// A non-default SSH port on a loopback address is a forwarded remote (container, tunnel)
	if host.Port != 0 && host.Port != 22 {
		return false
	}

	if host.Address == "localhost" || host.Address == "127.0.0.1" || host.Address == "::1" {
		return true
	}

	// Check if it's the local machine's IP
	return localAddrs()[host.Address]
}

// localAddrs are this machine's non-loopback addresses, read once: listing
// the interfaces for every task cost 40% of a 500-host run's CPU
var localAddrs = sync.OnceValue(func() map[string]bool {
	set := map[string]bool{}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return set
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			set[ipnet.IP.String()] = true
		}
	}
	return set
})

// WriteFile writes data to a file on the remote host using SFTP
func (c *Client) WriteFile(remotePath string, data []byte, mode os.FileMode) error {
	// Create SFTP client
	sftpClient, err := c.newSFTP()
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
	if err := remoteFile.Close(); err != nil {
		return fmt.Errorf("failed to close remote file %s: %v", remotePath, err)
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
	sftpClient, err := c.newSFTP()
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
	sftpClient, err := c.newSFTP()
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

// Chmod changes permissions of a file on the remote host using SFTP
func (c *Client) Chmod(remotePath string, mode os.FileMode) error {
	sftpClient, err := c.newSFTP()
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %v", err)
	}
	defer sftpClient.Close()

	if err := sftpClient.Chmod(remotePath, mode); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %v", remotePath, err)
	}
	return nil
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

// dialAddress returns host:port for the SSH connection, defaulting the port to 22.
func dialAddress(host types.Host) string {
	port := host.Port
	if port == 0 {
		port = 22
	}
	return net.JoinHostPort(host.Address, strconv.Itoa(port))
}

// isChannelRefused: the server refused a channel (more sessions than its
// MaxSessions, 10 by default for OpenSSH); the connection itself is fine
func isChannelRefused(err error) bool {
	var refused *ssh.OpenChannelError
	return errors.As(err, &refused)
}

// withChannelRoom opens a channel through open. Hosts that share one
// connection (same user, address and port) share its sessions: when the
// server refuses another one, idle command servers are closed and it is
// tried again while busy ones finish.
func (c *Client) withChannelRoom(open func() error) error {
	for attempt := 1; ; attempt++ {
		err := open()
		if err == nil || !isChannelRefused(err) || attempt == 40 {
			return err
		}
		c.closeShells()
		time.Sleep(time.Duration(attempt) * 25 * time.Millisecond)
	}
}

// newSession opens a session, waiting for room on the connection
func (c *Client) newSession() (*ssh.Session, error) {
	var session *ssh.Session
	err := c.withChannelRoom(func() (err error) {
		session, err = c.client.NewSession()
		return err
	})
	return session, err
}

// newSFTP opens an SFTP client, waiting for room on the connection
func (c *Client) newSFTP() (*sftp.Client, error) {
	var client *sftp.Client
	err := c.withChannelRoom(func() (err error) {
		client, err = sftp.NewClient(c.client)
		return err
	})
	return client, err
}
