package ssh

import (
	"bufio"
	"context"
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
const shellScript = `d=$(mktemp -d 2>/dev/null) || exit 97
trap 'rm -rf "$d"' EXIT
command -v base64 >/dev/null 2>&1 || exit 98
printf 'ONIGIRAZU-READY\n'
n=0
while IFS= read -r l; do
  n=$((n+1)); o="$d/o$n"; e="$d/e$n"
  m=${l%% *}; l=${l#* }
  printf '%s' "$l" | base64 -d > "$d/c" 2>/dev/null || { printf 'ONIGIRAZU 255 0 0\n'; continue; }
  if [ "$m" = C ]; then sh "$d/c" </dev/null >"$o" 2>&1; rc=$?; : >"$e"
  else sh "$d/c" </dev/null >"$o" 2>"$e"; rc=$?; fi
  printf 'ONIGIRAZU %d %d %d\n' "$rc" $(($(wc -c <"$o"))) $(($(wc -c <"$e")))
  cat "$o" "$e"
  rm -f "$o" "$e"
done
`

// remoteShell is one shell kept open on the host
type remoteShell struct {
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  *bufio.Reader
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

func (c *Client) startShell() (*remoteShell, error) {
	session, err := c.client.NewSession()
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
	if err := session.Start("sh -c " + quote(shellScript)); err != nil {
		_ = session.Close()
		return nil, err
	}
	s := &remoteShell{session: session, stdin: stdin, stdout: bufio.NewReaderSize(out, 64*1024)}
	line, err := s.stdout.ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "ONIGIRAZU-READY" {
		s.close()
		return nil, fmt.Errorf("shell did not start: %q", line)
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
	if _, err = io.WriteString(s.stdin, mode+" "+base64.StdEncoding.EncodeToString([]byte(command))+"\n"); err != nil {
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
	shell, err := c.takeShell()
	if err != nil || shell == nil {
		return c.execSession(ctx, command, combined)
	}
	type answer struct {
		out, errOut []byte
		rc          int
		err         error
	}
	done := make(chan answer, 1)
	go func() {
		o, e, rc, err := shell.run(command, combined)
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
		c.returnShell(shell)
		return a.out, a.errOut, a.rc, nil
	case <-ctx.Done():
		// the command may still run: the shell is not reused
		shell.close()
		return nil, nil, 0, fmt.Errorf("command execution canceled: %w", ctx.Err())
	}
}

func (c *Client) takeShell() (*remoteShell, error) {
	c.shells.mu.Lock()
	if c.shells.disabled {
		c.shells.mu.Unlock()
		return nil, nil
	}
	if n := len(c.shells.idle); n > 0 {
		s := c.shells.idle[n-1]
		c.shells.idle = c.shells.idle[:n-1]
		c.shells.mu.Unlock()
		return s, nil
	}
	c.shells.mu.Unlock()
	s, err := c.startShell()
	if err != nil {
		var exitErr *ssh.ExitError
		if errors.As(err, &exitErr) || strings.Contains(err.Error(), "shell did not start") {
			c.shells.mu.Lock()
			c.shells.disabled = true
			c.shells.mu.Unlock()
		}
		return nil, err
	}
	return s, nil
}

func (c *Client) returnShell(s *remoteShell) {
	c.shells.mu.Lock()
	defer c.shells.mu.Unlock()
	if len(c.shells.idle) < 8 {
		c.shells.idle = append(c.shells.idle, s)
		return
	}
	go s.close()
}

func (c *Client) closeShells() {
	c.shells.mu.Lock()
	idle := c.shells.idle
	c.shells.idle = nil
	c.shells.mu.Unlock()
	for _, s := range idle {
		s.close()
	}
}

// execSession runs a command in a session of its own
func (c *Client) execSession(ctx context.Context, command string, combined bool) ([]byte, []byte, int, error) {
	session, err := c.client.NewSession()
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
