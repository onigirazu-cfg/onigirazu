package ssh

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/agentbin"
)

// The agent (cmd/onigirazu-agent) is the command server as a Go binary:
// the same protocol, probes in-process, no python3 needed. With
// ONIGIRAZU_AGENT=1 a connection uploads the agent for the host's OS and
// architecture once per version to ~/.onigirazu/bin and starts it instead of
// the shell script. Any trouble (no binary for the platform, noexec home,
// sudo user without access) falls back to the Python or sh server.

// agentState is the agent of one connection
type agentState struct {
	once sync.Once
	path string // on the host; "" when there is none
	mu   sync.Mutex
	off  map[string]bool // users the agent did not start for
}

func agentEnabled() bool { return os.Getenv("ONIGIRAZU_AGENT") == "1" }

// agentFor is the agent command for user ("" when the script server is used)
func (c *Client) agentFor(user string) string {
	if !agentEnabled() {
		return ""
	}
	c.agent.mu.Lock()
	off := c.agent.off[user]
	c.agent.mu.Unlock()
	if off {
		return ""
	}
	c.agent.once.Do(func() { c.agent.path = c.installAgent() })
	if c.agent.path == "" {
		return ""
	}
	return quote(c.agent.path) + " serve"
}

// agentFailed: the agent did not start for user; the script server it is
func (c *Client) agentFailed(user string) {
	c.agent.mu.Lock()
	if c.agent.off == nil {
		c.agent.off = map[string]bool{}
	}
	c.agent.off[user] = true
	c.agent.mu.Unlock()
}

// localAgent is the agent binary for goos/goarch: built into onigirazu
// (internal/agentbin), else in $ONIGIRAZU_AGENT_DIR or next to the
// onigirazu executable, named onigirazu-agent-<os>-<arch>
func localAgent(goos, goarch string) ([]byte, error) {
	if data, ok := agentbin.Get(goos, goarch); ok {
		return data, nil
	}
	name := "onigirazu-agent-" + goos + "-" + goarch
	var dirs []string
	if d := os.Getenv("ONIGIRAZU_AGENT_DIR"); d != "" {
		dirs = append(dirs, d)
	}
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	for _, d := range dirs {
		if data, err := os.ReadFile(filepath.Join(d, name)); err == nil { // #nosec G304 -- our own binary
			return data, nil
		}
	}
	return nil, fmt.Errorf("no %s", name)
}

// platform maps "uname -sm" to GOOS and GOARCH
func platform(uname string) (string, string, bool) {
	f := strings.Fields(uname)
	if len(f) != 2 {
		return "", "", false
	}
	goos := map[string]string{"Linux": "linux", "Darwin": "darwin", "FreeBSD": "freebsd"}[f[0]]
	goarch := map[string]string{"x86_64": "amd64", "amd64": "amd64", "aarch64": "arm64", "arm64": "arm64",
		"armv7l": "arm", "armv6l": "arm", "i686": "386", "i386": "386"}[f[1]]
	return goos, goarch, goos != "" && goarch != ""
}

// installAgent puts the agent on the host if it is not there yet and checks
// that it runs; it returns its path, or "" to use the script server
func (c *Client) installAgent() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, _, rc, err := c.execSession(ctx, `uname -sm; printf '%s\n' "$HOME"`, false)
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if err != nil || rc != 0 || len(lines) != 2 || !strings.HasPrefix(lines[1], "/") {
		return ""
	}
	goos, goarch, ok := platform(lines[0])
	if !ok {
		return ""
	}
	data, err := localAgent(goos, goarch)
	if err != nil {
		c.logger.Debug("agent: %v", err)
		return ""
	}
	sum := sha256.Sum256(data)
	remote := path.Join(lines[1], ".onigirazu", "bin", "onigirazu-agent-"+hex.EncodeToString(sum[:8]))
	if !c.agentRuns(ctx, remote) {
		tmp := fmt.Sprintf("%s.%d.tmp", remote, time.Now().UnixNano())
		if err := c.WriteFile(tmp, data, 0o755); err != nil {
			c.logger.Debug("agent: upload: %v", err)
			return ""
		}
		_, _, rc, err := c.execSession(ctx, "mv -f "+quote(tmp)+" "+quote(remote)+" || rm -f "+quote(tmp), false)
		if err != nil || rc != 0 || !c.agentRuns(ctx, remote) {
			return ""
		}
	}
	return remote
}

// agentRuns starts the agent with no input: it answers ready and ends
func (c *Client) agentRuns(ctx context.Context, remote string) bool {
	out, _, rc, err := c.execSession(ctx, quote(remote)+" serve </dev/null", false)
	return err == nil && rc == 0 && strings.HasPrefix(string(out), "ONIGIRAZU-READY")
}
