package ssh

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

// A command costs about three round trips as its own SSH session (open the
// channel, exec, exit). A shell kept open on the host runs commands sent over
// its stdin: one round trip each. Commands and their output travel with
// lengths, so any bytes pass.

// ErrNotSent: the command never reached the host (the connection is gone),
// so running it again on a new connection is safe
var ErrNotSent = errors.New("command not sent")

// ExitStatusError is a command that ran and exited non-zero
type ExitStatusError struct{ Status int }

func (e *ExitStatusError) Error() string {
	return fmt.Sprintf("Process exited with status %d", e.Status)
}

// ExitStatus is the exit code (as ssh.ExitError has it)
func (e *ExitStatusError) ExitStatus() int { return e.Status }

// shellScript reads one base64 command per line, runs it with sh (stdin from
// /dev/null, stderr into stdout for "C", apart for "S") and answers
// "ONIGIRAZU <rc> <stdout bytes> <stderr bytes>" followed by the bytes.
// Every command gets its own output files: a background process it leaves
// behind writes to a removed file, not into the next command's output.
// The files live in ~/.onigirazu/tmp, as Ansible keeps its own in
// ~/.ansible/tmp: a task that cleans /tmp neither sees nor removes them, and
// the directory comes back if something removes it anyway (a command that
// removed its own output files reports empty output).
var shellScript = pythonServer() + posixServer

// serverScript is the script server to start: the Python one where python3
// is there, unless remote_server is sh (or ONIGIRAZU_NO_PYTHON=1)
func serverScript() string {
	if serverMode() == "sh" {
		return posixServer
	}
	return shellScript
}

//go:embed shell_server.py
var shellServerPy []byte

// pythonServer starts the Python version of the server when the host has a
// usable python3: one process per command instead of seven. The POSIX
// version below runs otherwise.
func pythonServer() string {
	src := base64.StdEncoding.EncodeToString(shellServerPy)
	return `if command -v python3 >/dev/null 2>&1 && python3 -c 'import subprocess, tempfile' >/dev/null 2>&1; then
  exec python3 -c "import base64; exec(base64.b64decode('` + src + `'))"
fi
`
}

const posixServer = `r="$HOME/.onigirazu/tmp"
[ -O "$HOME" ] || r=/nonexistent/.onigirazu
d=$( (mkdir -p -m 700 "$r" && mktemp -d "$r/sh.XXXXXX") 2>/dev/null || mktemp -d 2>/dev/null) || exit 97
trap 'rm -rf "$d"' EXIT
command -v base64 >/dev/null 2>&1 || exit 98
printf 'ONIGIRAZU-READY\n'
n=0
while IFS= read -r l; do
  [ -d "$d" ] || mkdir -p -m 700 "$d"
  n=$((n+1)); o="$d/o$n"; e="$d/e$n"
  m=${l%% *}; l=${l#* }
  base64 -d > "$d/c" 2>/dev/null <<EOF || { printf 'ONIGIRAZU 255 0 0\n'; continue; }
$l
EOF
  if [ "$m" = C ]; then sh "$d/c" </dev/null >"$o" 2>&1; rc=$?; : >"$e"
  else sh "$d/c" </dev/null >"$o" 2>"$e"; rc=$?; fi
  # one wc for both sizes: "N o", "M e", "T total" (the path may have
  # spaces: stderr's size is the total less stdout's); a command that
  # removed its own output file gets the slow way
  if [ -f "$o" ] && [ -f "$e" ]; then
    set -f; set -- $(wc -c "$o" "$e"); set +f; so=$1; eval "se=\$(( \${$(($# - 1))} - so ))"
  else
    so=$(wc -c <"$o" 2>/dev/null) || so=0; se=$(wc -c <"$e" 2>/dev/null) || se=0
  fi
  printf 'ONIGIRAZU %d %d %d\n' "$rc" $((so)) $((se))
  cat "$o" "$e" 2>/dev/null
  # the outputs of finished commands go in batches, not one rm each
  [ $((n % 64)) -ne 0 ] || rm -f "$d"/o* "$d"/e*
done
`

// remoteShell is one shell kept open on the host
type remoteShell struct {
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	// probe, write: the server answers "P"/"Q" and "W" requests (the
	// Python one and the agent)
	probe, write bool
}

func (s *remoteShell) close() {
	_ = s.stdin.Close()
	if s.session != nil {
		_ = s.session.Close()
	}
}

// shellPool holds the idle shells of a connection
type shellPool struct {
	mu       sync.Mutex
	idle     []*remoteShell
	disabled bool // the host cannot run the shell: use a session per command
}

func (c *Client) startShell(user string) (*remoteShell, error) {
	session, err := c.newSession()
	if err != nil {
		return nil, err
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		return nil, err
	}
	out, err := session.StdoutPipe()
	if err != nil {
		_ = session.Close()
		return nil, err
	}
	command := "sh -c " + quote(serverScript())
	agent := c.agentFor(user)
	if agent != "" {
		command = agent
	}
	if user != "" {
		// sudo once for the server instead of once per command; -n: a
		// password prompt fails the start, and become falls back to sudo
		// per command
		command = "sudo -n -u " + quote(user) + " " + command
	}
	if err := session.Start(command); err != nil {
		_ = session.Close()
		return nil, err
	}
	s := &remoteShell{session: session, stdin: stdin, stdout: bufio.NewReaderSize(out, 64*1024)}
	line, err := s.stdout.ReadString('\n')
	f := strings.Fields(line)
	if err != nil || len(f) == 0 || f[0] != "ONIGIRAZU-READY" {
		s.close()
		if agent != "" {
			// e.g. the become user cannot reach the login user's home
			c.agentFailed(user)
			return c.startShell(user)
		}
		return nil, fmt.Errorf("shell did not start: %q", line)
	}
	for _, flag := range f[1:] {
		s.probe = s.probe || flag == "P"
		s.write = s.write || flag == "W"
	}
	return s, nil
}

func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// run sends one command and reads its answer
func (s *remoteShell) run(command string, combined bool) (stdout, stderr []byte, rc int, err error) {
	mode := "S"
	if combined {
		mode = "C"
	}
	return s.request(mode, command)
}

// request sends one request of mode (S, C: a command; P: a probe)
func (s *remoteShell) request(mode, payload string) (stdout, stderr []byte, rc int, err error) {
	if _, err = io.WriteString(s.stdin, mode+" "+base64.StdEncoding.EncodeToString([]byte(payload))+"\n"); err != nil {
		return nil, nil, 0, fmt.Errorf("%w: %v", ErrNotSent, err)
	}
	header, err := s.stdout.ReadString('\n')
	if err != nil {
		return nil, nil, 0, err
	}
	f := strings.Fields(header)
	if len(f) != 4 || f[0] != "ONIGIRAZU" {
		return nil, nil, 0, fmt.Errorf("unexpected shell answer %q", header)
	}
	rc, _ = strconv.Atoi(f[1])
	no, _ := strconv.Atoi(f[2])
	ne, _ := strconv.Atoi(f[3])
	buf := make([]byte, no+ne)
	if _, err = io.ReadFull(s.stdout, buf); err != nil {
		return nil, nil, 0, err
	}
	return buf[:no], buf[no:], rc, nil
}

// Exec runs a command line on the host and returns stdout, stderr (empty
// when combined: then stderr is in stdout, in order) and the exit code. A
// non-zero exit is not an error; the error is for a command that could not
// run. It uses a shell kept open on the host, or a session of its own when
// the host has no usable shell.
func (c *Client) Exec(ctx context.Context, command string, combined bool) ([]byte, []byte, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, 0, err
	}
	shell, err := c.takeShell("")
	if err != nil || shell == nil {
		return c.execSession(ctx, command, combined)
	}
	return c.runIn(ctx, "", shell, modeOf(combined), command)
}

func modeOf(combined bool) string {
	if combined {
		return "C"
	}
	return "S"
}

// Write writes data to path on the host as user through the command server,
// without a process: in a file next to it moved over it, missing parent
// directories made. mode is octal or "-" (keep, 0644 for a new file),
// owner and group a name, an id or "-" (keep). served is false when the
// server cannot (the sh one): the caller writes with commands.
func (c *Client) Write(ctx context.Context, user, path string, data []byte, mode, owner, group string) (served bool, err error) {
	if err := ctx.Err(); err != nil {
		return true, err
	}
	shell, err := c.takeShell(user)
	if err != nil || shell == nil {
		return false, nil
	}
	if !shell.write || strings.ContainsAny(path, "\n\x00") {
		c.returnShell(user, shell)
		return false, nil
	}
	_, e, rc, err := c.runIn(ctx, user, shell, "W", mode+" "+owner+" "+group+"\n"+path+"\n"+string(data))
	if err != nil {
		return true, err
	}
	if rc != 0 {
		return true, fmt.Errorf("write %s: %s", path, strings.TrimSpace(string(e)))
	}
	return true, nil
}

// ProbeMany is Probe for several paths in one request: the records come in
// the order of paths, each followed by "\x1e\n"
func (c *Client) ProbeMany(ctx context.Context, user string, paths []string, limit int) (out []byte, served bool, err error) {
	if err := ctx.Err(); err != nil {
		return nil, true, err
	}
	shell, err := c.takeShell(user)
	if err != nil || shell == nil {
		return nil, false, nil
	}
	if !shell.probe {
		c.returnShell(user, shell)
		return nil, false, nil
	}
	o, e, rc, err := c.runIn(ctx, user, shell, "Q", strconv.Itoa(limit)+"\n"+strings.Join(paths, "\n"))
	if err != nil {
		return nil, true, err
	}
	if rc != 0 {
		return nil, true, fmt.Errorf("probe: %s", strings.TrimSpace(string(e)))
	}
	return o, true, nil
}

// Probe describes path (kind, mode, owner, group, size and the content up to
// limit bytes, else its sha256) as user ("" is the login user) without
// starting a process on the host, in the format of the shell probe of the
// file modules' capture. served is false when the server cannot (no Python,
// sudo wants a password): the caller runs the shell probe.
func (c *Client) Probe(ctx context.Context, user, path string, limit int) (out []byte, served bool, err error) {
	if err := ctx.Err(); err != nil {
		return nil, true, err
	}
	shell, err := c.takeShell(user)
	if err != nil || shell == nil {
		return nil, false, nil
	}
	if !shell.probe {
		c.returnShell(user, shell)
		return nil, false, nil
	}
	o, e, rc, err := c.runIn(ctx, user, shell, "P", strconv.Itoa(limit)+" "+path)
	if err != nil {
		return nil, true, err
	}
	if rc != 0 {
		return nil, true, fmt.Errorf("probe %s: %s", path, strings.TrimSpace(string(e)))
	}
	return o, true, nil
}

// ExecAs runs a command as user through a command server started with sudo
// -n once per connection. served is false when the host cannot run one
// (sudo wants a password, requiretty, ...): the caller wraps the command in
// sudo and uses Exec.
func (c *Client) ExecAs(ctx context.Context, user, command string, combined bool) (stdout, stderr []byte, rc int, served bool, err error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, 0, true, err
	}
	shell, err := c.takeShell(user)
	if err != nil || shell == nil {
		return nil, nil, 0, false, nil
	}
	stdout, stderr, rc, err = c.runIn(ctx, user, shell, modeOf(combined), command)
	return stdout, stderr, rc, true, err
}

// runIn runs a command on a shell taken from the pool of user
func (c *Client) runIn(ctx context.Context, user string, shell *remoteShell, mode, payload string) ([]byte, []byte, int, error) {
	type answer struct {
		out, errOut []byte
		rc          int
		err         error
	}
	done := make(chan answer, 1)
	go func() {
		o, e, rc, err := shell.request(mode, payload)
		done <- answer{o, e, rc, err}
	}()
	select {
	case a := <-done:
		if a.err != nil {
			shell.close() // the shell is out of step or gone
			if c.closed.Load() {
				return nil, nil, 0, fmt.Errorf("%w: %v", ErrClosed, a.err)
			}
			return nil, nil, 0, a.err
		}
		c.returnShell(user, shell)
		return a.out, a.errOut, a.rc, nil
	case <-ctx.Done():
		// the command may still run: the shell is not reused
		shell.close()
		return nil, nil, 0, fmt.Errorf("command execution canceled: %w", ctx.Err())
	}
}

// pool is the shell pool of user ("" is the login user)
func (c *Client) pool(user string) *shellPool {
	if user == "" {
		return &c.shells
	}
	c.asMu.Lock()
	defer c.asMu.Unlock()
	if c.asShells == nil {
		c.asShells = map[string]*shellPool{}
	}
	p, ok := c.asShells[user]
	if !ok {
		p = &shellPool{}
		c.asShells[user] = p
	}
	return p
}

func (c *Client) takeShell(user string) (*remoteShell, error) {
	p := c.pool(user)
	p.mu.Lock()
	if p.disabled {
		p.mu.Unlock()
		return nil, nil
	}
	if n := len(p.idle); n > 0 {
		s := p.idle[n-1]
		p.idle = p.idle[:n-1]
		p.mu.Unlock()
		return s, nil
	}
	p.mu.Unlock()
	s, err := c.startShell(user)
	if err != nil {
		var exitErr *ssh.ExitError
		// a server as another user that does not start (sudo wants a
		// password) is not tried again on this connection
		if user != "" || errors.As(err, &exitErr) || strings.Contains(err.Error(), "shell did not start") {
			p.mu.Lock()
			p.disabled = true
			p.mu.Unlock()
		}
		return nil, err
	}
	return s, nil
}

func (c *Client) returnShell(user string, s *remoteShell) {
	p := c.pool(user)
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.idle) < 8 {
		p.idle = append(p.idle, s)
		return
	}
	go s.close()
}

func (c *Client) closeShells() {
	pools := []*shellPool{&c.shells}
	c.asMu.Lock()
	for _, p := range c.asShells {
		pools = append(pools, p)
	}
	c.asMu.Unlock()
	for _, p := range pools {
		p.mu.Lock()
		idle := p.idle
		p.idle = nil
		p.mu.Unlock()
		for _, s := range idle {
			s.close()
		}
	}
}

// execSession runs a command in a session of its own
func (c *Client) execSession(ctx context.Context, command string, combined bool) ([]byte, []byte, int, error) {
	session, err := c.newSession()
	if err != nil {
		return nil, nil, 0, fmt.Errorf("%w: %v", ErrNotSent, err)
	}
	defer session.Close()
	var stdout, stderr safeBuffer
	session.Stdout = &stdout
	if combined {
		session.Stderr = &stdout
	} else {
		session.Stderr = &stderr
	}
	done := make(chan error, 1)
	go func() { done <- session.Run(command) }()
	select {
	case err = <-done:
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGTERM)
		return nil, nil, 0, fmt.Errorf("command execution canceled: %w", ctx.Err())
	}
	var exitErr *ssh.ExitError
	if errors.As(err, &exitErr) {
		return stdout.Bytes(), stderr.Bytes(), exitErr.ExitStatus(), nil
	}
	return stdout.Bytes(), stderr.Bytes(), 0, err
}

// safeBuffer is a buffer written by stdout and stderr of one session
type safeBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *safeBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.buf...)
}

// Prewarm starts a command server for user ("" is the login user) in the
// background, once per connection, and puts it in the pool
func (c *Client) Prewarm(user string) {
	if _, started := c.prewarmed.LoadOrStore(user, true); started {
		return
	}
	go func() {
		p := c.pool(user)
		p.mu.Lock()
		skip := p.disabled || len(p.idle) > 0
		p.mu.Unlock()
		if skip || c.closed.Load() {
			return
		}
		s, err := c.startShell(user)
		if err != nil {
			if user != "" {
				p.mu.Lock()
				p.disabled = true // as takeShell: no second try with sudo
				p.mu.Unlock()
			}
			return
		}
		c.returnShell(user, s)
	}()
}
