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

	"sync/atomic"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"

	"github.com/onigirazu-cfg/onigirazu/internal/agentbin"
)

// The agent (cmd/onigirazu-agent) is the command server as a Go binary:
// the same protocol, probes in-process, no python3 needed. With
// remote_server auto (the default) a connection uploads the agent for the
// host's OS and architecture once per version to ~/.onigirazu/bin and starts
// it instead of the shell script. Any trouble (no binary for the platform, noexec home,
// sudo user without access) falls back to the Python or sh server.

// agentState is the agent of one connection
type agentState struct {
	once sync.Once
	path string // on the host; "" when there is none
	mu   sync.Mutex
	off  map[string]bool // users the agent did not start for
}

// remoteServer is the remote_server setting: auto (agent, Python, sh),
// python (Python, sh) or sh
var remoteServer atomic.Value

// SetRemoteServer sets the command server mode (remote_server)
func SetRemoteServer(mode string) { remoteServer.Store(mode) }

// serverMode is remote_server; ONIGIRAZU_NO_PYTHON=1 still means sh
func serverMode() string {
	if os.Getenv("ONIGIRAZU_NO_PYTHON") == "1" {
		return "sh"
	}
	if mode, _ := remoteServer.Load().(string); mode != "" {
		return mode
	}
	return "auto"
}

func agentEnabled() bool { return serverMode() == "auto" }

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
	if isWindowsSSH(c.host) {
		return windowsAgentCommand(c.agent.path)
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

// agentImage is an agent binary and the id in its remote name
type agentImage struct {
	data []byte
	gz   []byte // the same gzipped, when the build embeds it
	id   string
	err  error
}

// agentImages: each platform's agent is unpacked and hashed once per run,
// not once per host (500 hosts spent a third of the run's CPU on it)
var agentImages sync.Map

func cachedAgent(goos, goarch string) agentImage {
	load, _ := agentImages.LoadOrStore(goos+"/"+goarch, sync.OnceValue(func() agentImage {
		data, err := localAgent(goos, goarch)
		if err != nil {
			return agentImage{err: err}
		}
		sum := sha256.Sum256(data)
		gz, _ := agentbin.Compressed(goos, goarch)
		return agentImage{data: data, gz: gz, id: hex.EncodeToString(sum[:8])}
	}))
	get, _ := load.(func() agentImage)
	return get()
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

// isWindowsSSH reports a Windows host reached over OpenSSH (ansible_shell_type
// powershell or cmd, as Ansible marks them)
func isWindowsSSH(host types.Host) bool {
	for _, key := range []string{"onigirazu_shell_type", "ansible_shell_type"} {
		switch host.Vars[key] {
		case "powershell", "cmd":
			return true
		}
	}
	return false
}

// installAgent puts the agent on the host if it is not there yet and checks
// that it runs; it returns its path, or "" to use the script server
func (c *Client) installAgent() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if isWindowsSSH(c.host) {
		return c.installWindowsAgent(ctx)
	}
	out, _, rc, err := c.execSession(ctx, `uname -sm; printf '%s\n' "$HOME"; command -v gzip || true`, false)
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if err != nil || rc != 0 || len(lines) < 2 || !strings.HasPrefix(lines[1], "/") {
		return ""
	}
	hasGzip := len(lines) > 2 && lines[2] != ""
	goos, goarch, ok := platform(lines[0])
	if !ok {
		return ""
	}
	img := cachedAgent(goos, goarch)
	if img.err != nil {
		c.logger.Debug("agent: %v", img.err)
		return ""
	}
	data := img.data
	remote := path.Join(lines[1], ".onigirazu", "bin", "onigirazu-agent-"+img.id)
	if !c.agentRuns(ctx, remote) {
		tmp := fmt.Sprintf("%s.%d.tmp", remote, time.Now().UnixNano())
		install := "mv -f " + quote(tmp) + " " + quote(remote) + " || rm -f " + quote(tmp)
		upload, mode := data, os.FileMode(0o755)
		// the gzipped agent is less than half the bytes: unpacked on the host
		if hasGzip && img.gz != nil {
			upload, mode = img.gz, 0o600
			install = "gzip -dc " + quote(tmp+".gz") + " > " + quote(tmp) + " && chmod 755 " + quote(tmp) +
				" && mv -f " + quote(tmp) + " " + quote(remote) + "; rc=$?; rm -f " + quote(tmp+".gz") + " " + quote(tmp) + "; exit $rc"
			tmp += ".gz"
		}
		if err := c.WriteFile(tmp, upload, mode); err != nil {
			c.logger.Debug("agent: upload: %v", err)
			return ""
		}
		_, _, rc, err := c.execSession(ctx, install, false)
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

// windowsAgentCommand starts the agent whatever the server's default shell
// is (cmd or PowerShell): through powershell.exe, the path quoted its way
func windowsAgentCommand(remote string) string {
	return `powershell.exe -NoProfile -NonInteractive -Command "& '` + strings.ReplaceAll(remote, "'", "''") + `' serve"`
}

// installWindowsAgent: the host is Windows with OpenSSH; the agent goes to
// %USERPROFILE%\.onigirazu\bin, uploaded whole (no gzip there), and runs
// PowerShell scripts for the commands
func (c *Client) installWindowsAgent(ctx context.Context) string {
	out, _, rc, err := c.execSession(ctx, `powershell.exe -NoProfile -NonInteractive -Command "Write-Output $env:PROCESSOR_ARCHITECTURE; Write-Output $env:USERPROFILE"`, false)
	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(string(out), "\r", "")), "\n")
	if err != nil || rc != 0 || len(lines) < 2 || !strings.Contains(lines[1], ":\\") {
		c.logger.Debug("agent: windows: no platform answer: %v %q", err, out)
		return ""
	}
	goarch := map[string]string{"AMD64": "amd64", "ARM64": "arm64"}[strings.ToUpper(strings.TrimSpace(lines[0]))]
	if goarch == "" {
		return ""
	}
	img := cachedAgent("windows", goarch)
	if img.err != nil {
		c.logger.Debug("agent: %v", img.err)
		return ""
	}
	home := strings.TrimSpace(lines[1])
	dir := home + `\.onigirazu\bin`
	remote := dir + `\onigirazu-agent-` + img.id + ".exe"
	if c.windowsAgentRuns(ctx, remote) {
		return remote
	}
	if _, _, rc, err := c.execSession(ctx, `powershell.exe -NoProfile -NonInteractive -Command "New-Item -ItemType Directory -Force -Path '`+strings.ReplaceAll(dir, "'", "''")+`' | Out-Null"`, false); err != nil || rc != 0 {
		return ""
	}
	// SFTP takes the path with forward slashes
	if err := c.WriteFile(strings.ReplaceAll(remote, `\`, "/"), img.data, 0o755); err != nil {
		c.logger.Debug("agent: windows upload: %v", err)
		return ""
	}
	if !c.windowsAgentRuns(ctx, remote) {
		return ""
	}
	return remote
}

func (c *Client) windowsAgentRuns(ctx context.Context, remote string) bool {
	out, _, rc, err := c.execSession(ctx, windowsAgentCommand(remote), false)
	return err == nil && rc == 0 && strings.HasPrefix(strings.TrimSpace(string(out)), "ONIGIRAZU-READY")
}
