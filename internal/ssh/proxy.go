package ssh

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	sshconfig "github.com/kevinburke/ssh_config"
	"golang.org/x/crypto/ssh"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// dialHost connects to the host directly, through ProxyJump hosts or through
// a ProxyCommand. jumps are the clients of the jump hosts, to be closed with
// the connection.
func dialHost(host types.Host, address string, config *ssh.ClientConfig, auth []ssh.AuthMethod,
	m *HostKeyManager, opts sshOptions) (*ssh.Client, []*ssh.Client, error) {
	switch {
	case opts.ProxyCommand != "":
		conn, err := proxyCommandConn(opts.ProxyCommand, host, address)
		if err != nil {
			return nil, nil, err
		}
		client, err := clientOver(conn, address, config)
		return client, nil, err
	case opts.ProxyJump != "":
		var jumps []*ssh.Client
		var via *ssh.Client
		for _, hop := range strings.Split(opts.ProxyJump, ",") {
			jumpHost, jumpAddr, jumpCfg, err := jumpConfig(strings.TrimSpace(hop), host, auth, m, config.Timeout)
			if err != nil {
				closeAll(jumps)
				return nil, nil, err
			}
			var next *ssh.Client
			if via == nil {
				next, err = ssh.Dial("tcp", jumpAddr, jumpCfg)
			} else {
				next, err = dialThrough(via, jumpAddr, jumpCfg)
			}
			if err != nil {
				closeAll(jumps)
				return nil, nil, fmt.Errorf("jump host %s: %w", jumpHost, err)
			}
			jumps = append(jumps, next)
			via = next
		}
		client, err := dialThrough(via, address, config)
		if err != nil {
			closeAll(jumps)
			return nil, nil, err
		}
		return client, jumps, nil
	}
	client, err := dialRetrying(address, config)
	return client, nil, err
}

func dialThrough(via *ssh.Client, address string, config *ssh.ClientConfig) (*ssh.Client, error) {
	conn, err := via.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	return clientOver(conn, address, config)
}

func clientOver(conn net.Conn, address string, config *ssh.ClientConfig) (*ssh.Client, error) {
	c, chans, reqs, err := ssh.NewClientConn(conn, address, config)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}

func closeAll(clients []*ssh.Client) {
	for i := len(clients) - 1; i >= 0; i-- {
		_ = clients[i].Close()
	}
}

// jumpConfig resolves a jump host ([user@]host[:port]) the way ssh does: an
// alias from ~/.ssh/config gives HostName, User, Port and IdentityFile. The
// jump host's key is checked like the target's.
func jumpConfig(hop string, target types.Host, auth []ssh.AuthMethod, m *HostKeyManager,
	timeout time.Duration) (string, string, *ssh.ClientConfig, error) {
	user, hostPort := "", hop
	if at := strings.LastIndex(hop, "@"); at >= 0 {
		user, hostPort = hop[:at], hop[at+1:]
	}
	name, port := hostPort, ""
	if h, p, err := net.SplitHostPort(hostPort); err == nil {
		name, port = h, p
	}
	address := name
	if v := sshconfig.Get(name, "HostName"); v != "" {
		address = v
	}
	if port == "" {
		port = sshconfig.Get(name, "Port")
	}
	if port == "" {
		port = "22"
	}
	if user == "" {
		user = sshconfig.Get(name, "User")
	}
	if user == "" {
		user = target.User
	}
	methods := auth
	if key := sshconfig.Get(name, "IdentityFile"); key != "" && key != "~/.ssh/identity" {
		if signer, err := readSigner(expandHome(key)); err == nil {
			methods = append([]ssh.AuthMethod{ssh.PublicKeys(signer)}, auth...)
		}
	}
	portN, _ := strconv.Atoi(port)
	jump := types.Host{Name: name, Address: address, Port: portN, User: user, InsecureIgnoreHostKey: target.InsecureIgnoreHostKey}
	cfg := &ssh.ClientConfig{User: user, Auth: methods, HostKeyCallback: hostKeyCallback(jump, m), Timeout: timeout}
	return name, net.JoinHostPort(address, port), cfg, nil
}

func readSigner(path string) (ssh.Signer, error) {
	key, err := os.ReadFile(path) // #nosec G304 -- IdentityFile from the user's ssh config
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}
	return withCertificate(signer, path, types.Host{})
}

func expandHome(p string) string {
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, rest)
		}
	}
	return p
}

// proxyCommandConn runs a ProxyCommand (%h, %p, %r replaced) and speaks SSH
// over its stdin and stdout
func proxyCommandConn(command string, host types.Host, address string) (net.Conn, error) {
	h, p, _ := net.SplitHostPort(address)
	r := strings.NewReplacer("%h", h, "%p", p, "%r", host.User, "%%", "%")
	// #nosec G204 -- the ProxyCommand comes from the inventory, as in ssh
	cmd := exec.Command("sh", "-c", r.Replace(command))
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("ProxyCommand: %w", err)
	}
	return &cmdConn{cmd: cmd, in: stdin, out: stdout, addr: pipeAddr(address)}, nil
}

// cmdConn is a net.Conn over a command's stdin and stdout
type cmdConn struct {
	cmd  *exec.Cmd
	in   io.WriteCloser
	out  io.ReadCloser
	addr pipeAddr
}

func (c *cmdConn) Read(b []byte) (int, error)  { return c.out.Read(b) }
func (c *cmdConn) Write(b []byte) (int, error) { return c.in.Write(b) }
func (c *cmdConn) Close() error {
	_ = c.in.Close()
	_ = c.out.Close()
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	_ = c.cmd.Wait()
	return nil
}
func (c *cmdConn) LocalAddr() net.Addr                { return c.addr }
func (c *cmdConn) RemoteAddr() net.Addr               { return c.addr }
func (c *cmdConn) SetDeadline(t time.Time) error      { return nil }
func (c *cmdConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *cmdConn) SetWriteDeadline(t time.Time) error { return nil }

// pipeAddr reports the target host:port, which known_hosts checks need
type pipeAddr string

func (a pipeAddr) Network() string { return "tcp" }
func (a pipeAddr) String() string  { return string(a) }

// dialRetrying dials like ssh.Dial, trying again a few times when the host
// refuses or resets the connection: sshd restarted by the previous task
// (hardening roles) is back within seconds, and Ansible retries as well
func dialRetrying(address string, config *ssh.ClientConfig) (*ssh.Client, error) {
	var client *ssh.Client
	var err error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
		client, err = ssh.Dial("tcp", address, config)
		if err == nil || !isTransientDialError(err) {
			return client, err
		}
	}
	return client, err
}

func isTransientDialError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "connection refused") || strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "unexpected EOF") || strings.Contains(msg, "handshake failed: EOF")
}
