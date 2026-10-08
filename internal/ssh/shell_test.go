package ssh

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// the shell script and its framing, run by a local sh
func TestRemoteShellProtocol(t *testing.T) { forEachServer(t, checkRemoteShellProtocol) }

func checkRemoteShellProtocol(t *testing.T, script string) {
	cmd := exec.Command("sh", "-c", script)
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	out, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	defer func() { _ = stdin.Close(); _ = cmd.Wait() }()
	s := &remoteShell{stdin: stdin, stdout: bufio.NewReader(out)}
	ready, err := s.stdout.ReadString('\n')
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(ready, "ONIGIRAZU-READY"), "%q", ready)

	o, e, rc, err := s.run("echo out; echo err >&2; exit 3", false)
	require.NoError(t, err)
	assert.Equal(t, "out\n", string(o))
	assert.Equal(t, "err\n", string(e))
	assert.Equal(t, 3, rc)

	o, e, rc, err = s.run("echo a; echo b >&2; echo c", true)
	require.NoError(t, err)
	assert.Equal(t, "a\nb\nc\n", string(o))
	assert.Empty(t, e)
	assert.Equal(t, 0, rc)

	// any bytes, quotes, no trailing newline, a command reading stdin
	o, _, rc, err = s.run("printf '%s' \"x'y\\\"z\"; printf '\\001\\377'; cat", false)
	require.NoError(t, err)
	assert.Equal(t, "x'y\"z\x01\xff", string(o))
	assert.Equal(t, 0, rc)

	big := strings.Repeat("0123456789", 20000)
	o, _, _, err = s.run("printf '%s' "+quote(big), false)
	require.NoError(t, err)
	assert.Equal(t, big, string(o))

	// the shell survives a failing command and keeps going
	_, _, rc, err = s.run("false", false)
	require.NoError(t, err)
	assert.Equal(t, 1, rc)
	o, _, _, err = s.run("echo still", false)
	require.NoError(t, err)
	assert.Equal(t, "still\n", string(o))
}

func TestExitStatusError(t *testing.T) {
	var e error = &ExitStatusError{Status: 2}
	assert.Equal(t, "Process exited with status 2", e.Error())
	var es interface{ ExitStatus() int }
	assert.ErrorAs(t, e, &es)
	assert.Equal(t, 2, es.ExitStatus())
}

func TestRemoteShellBackgroundChild(t *testing.T) { forEachServer(t, checkRemoteShellBackgroundChild) }

func checkRemoteShellBackgroundChild(t *testing.T, script string) {
	cmd := exec.Command("sh", "-c", script)
	stdin, _ := cmd.StdinPipe()
	out, _ := cmd.StdoutPipe()
	require.NoError(t, cmd.Start())
	defer func() { _ = stdin.Close(); _ = cmd.Wait() }()
	s := &remoteShell{stdin: stdin, stdout: bufio.NewReader(out)}
	_, _ = s.stdout.ReadString('\n')
	_, _, _, err := s.run("(sleep 0.3; echo late) &", false)
	require.NoError(t, err)
	o, _, _, err := s.run("sleep 0.6; echo mine", false)
	require.NoError(t, err)
	assert.Equal(t, "mine\n", string(o))
}

// A run that is stopped closes the connections: a command still running
// then fails with ErrClosed, not with the bare EOF of the dead shell
func TestExecOnClosedClient(t *testing.T) {
	cmd := exec.Command("sh", "-c", shellScript)
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	out, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	s := &remoteShell{stdin: stdin, stdout: bufio.NewReader(out)}
	_, err = s.stdout.ReadString('\n')
	require.NoError(t, err)

	c := &Client{}
	c.shells.idle = []*remoteShell{s}
	done := make(chan error, 1)
	go func() {
		_, _, _, err := c.Exec(context.Background(), "sleep 10", true)
		done <- err
	}()
	time.Sleep(300 * time.Millisecond)
	require.NoError(t, c.Close())
	_ = cmd.Process.Kill() // what closing the connection does to the shell
	_ = cmd.Wait()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, ErrClosed)
	case <-time.After(5 * time.Second):
		t.Fatal("Exec did not return")
	}
}

// the work files live under ~/.onigirazu/tmp, not in /tmp, and a command
// that removes them does not break the next one
func TestRemoteShellWorkDir(t *testing.T) { forEachServer(t, checkRemoteShellWorkDir) }

func checkRemoteShellWorkDir(t *testing.T, script string) {
	home := t.TempDir()
	cmd := exec.Command("sh", "-c", script)
	cmd.Env = append(cmd.Environ(), "HOME="+home)
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	out, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	defer func() { _ = stdin.Close(); _ = cmd.Wait() }()
	s := &remoteShell{stdin: stdin, stdout: bufio.NewReader(out)}
	_, err = s.stdout.ReadString('\n')
	require.NoError(t, err)

	o, _, _, err := s.run("ls \"$HOME/.onigirazu/tmp\" | grep -cE '^(sh|py)\\.'", false)
	require.NoError(t, err)
	assert.Equal(t, "1\n", string(o))

	_, _, rc, err := s.run("rm -rf \"$HOME/.onigirazu\"", false)
	require.NoError(t, err)
	assert.Equal(t, 0, rc)
	o, _, rc, err = s.run("echo after", false)
	require.NoError(t, err)
	assert.Equal(t, 0, rc)
	assert.Equal(t, "after\n", string(o))
}

// forEachServer runs a check against the POSIX server and, when this
// machine has python3, the Python one
func forEachServer(t *testing.T, check func(*testing.T, string)) {
	t.Run("posix", func(t *testing.T) { check(t, posixServer) })
	t.Run("python", func(t *testing.T) {
		if _, err := exec.LookPath("python3"); err != nil {
			t.Skip("no python3")
		}
		check(t, shellScript)
	})
}

// the Python server is the one that starts when python3 is there
func TestShellScriptPrefersPython(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("no python3")
	}
	home := t.TempDir()
	cmd := exec.Command("sh", "-c", shellScript)
	cmd.Env = append(cmd.Environ(), "HOME="+home)
	stdin, _ := cmd.StdinPipe()
	out, _ := cmd.StdoutPipe()
	require.NoError(t, cmd.Start())
	defer func() { _ = stdin.Close(); _ = cmd.Wait() }()
	s := &remoteShell{stdin: stdin, stdout: bufio.NewReader(out)}
	_, _ = s.stdout.ReadString('\n')
	o, _, _, err := s.run("ls \"$HOME/.onigirazu/tmp\"", false)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(o), "py."), "work dir %q", o)
}

// the Python server describes a file without a process, as the shell probe
// of the capture prints it
func TestPythonServerProbe(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("no python3")
	}
	home := t.TempDir()
	cmd := exec.Command("sh", "-c", shellScript)
	cmd.Env = append(cmd.Environ(), "HOME="+home)
	stdin, _ := cmd.StdinPipe()
	out, _ := cmd.StdoutPipe()
	require.NoError(t, cmd.Start())
	defer func() { _ = stdin.Close(); _ = cmd.Wait() }()
	s := &remoteShell{stdin: stdin, stdout: bufio.NewReader(out)}
	ready, _ := s.stdout.ReadString('\n')
	assert.Equal(t, "ONIGIRAZU-READY P\n", ready)

	p := filepath.Join(home, "f")
	require.NoError(t, os.WriteFile(p, []byte("v1\n"), 0o640))
	require.NoError(t, os.Chmod(p, 0o640))
	o, _, rc, err := s.request("P", "100 "+p)
	require.NoError(t, err)
	assert.Equal(t, 0, rc)
	lines := strings.Split(string(o), "\n")
	f := strings.Fields(lines[0])
	require.Len(t, f, 6, "%q", o)
	assert.Equal(t, []string{"file", "640"}, f[:2])
	assert.Equal(t, []string{"3", "+"}, f[4:])
	assert.Equal(t, "C:djEK", lines[1])

	o, _, _, _ = s.request("P", "2 "+p)
	// over the limit: the sha256 instead of the content
	f = strings.Fields(string(o))
	require.Len(t, f, 6, "%q", o)
	assert.Len(t, f[5], 64, "sha256 %q", o)

	o, _, _, _ = s.request("P", "100 "+filepath.Join(home, "none"))
	assert.Equal(t, "absent\n", string(o))
	o, _, _, _ = s.request("P", "100 "+home)
	assert.True(t, strings.HasPrefix(string(o), "directory "), "%q", o)

	// commands still work after probes
	o, _, rc, _ = s.run("echo ok", false)
	assert.Equal(t, "ok\n", string(o))
	assert.Equal(t, 0, rc)
}

// the work directory may have spaces and glob characters in its path
func TestPosixServerOddHome(t *testing.T) {
	home := filepath.Join(t.TempDir(), "a b*c")
	require.NoError(t, os.MkdirAll(home, 0o700))
	cmd := exec.Command("sh", "-c", posixServer)
	cmd.Env = append(cmd.Environ(), "HOME="+home)
	stdin, _ := cmd.StdinPipe()
	out, _ := cmd.StdoutPipe()
	require.NoError(t, cmd.Start())
	defer func() { _ = stdin.Close(); _ = cmd.Wait() }()
	s := &remoteShell{stdin: stdin, stdout: bufio.NewReader(out)}
	_, _ = s.stdout.ReadString('\n')
	for i := 0; i < 70; i++ { // past a batch of removals
		o, e, rc, err := s.run("printf out; printf errr >&2; exit 4", false)
		require.NoError(t, err)
		assert.Equal(t, "out", string(o))
		assert.Equal(t, "errr", string(e))
		assert.Equal(t, 4, rc)
	}
	// a command that removes its own output files
	o, e, rc, err := s.run("rm -rf \"$HOME/.onigirazu\"; echo gone", false)
	require.NoError(t, err)
	assert.Equal(t, 0, rc)
	assert.Empty(t, string(o)+string(e))
	o, _, _, err = s.run("echo after", false)
	require.NoError(t, err)
	assert.Equal(t, "after\n", string(o))
}

// several probes in one request
func TestPythonServerProbeMany(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("no python3")
	}
	home := t.TempDir()
	cmd := exec.Command("sh", "-c", shellScript)
	cmd.Env = append(cmd.Environ(), "HOME="+home)
	stdin, _ := cmd.StdinPipe()
	out, _ := cmd.StdoutPipe()
	require.NoError(t, cmd.Start())
	defer func() { _ = stdin.Close(); _ = cmd.Wait() }()
	s := &remoteShell{stdin: stdin, stdout: bufio.NewReader(out)}
	_, _ = s.stdout.ReadString('\n')
	p := filepath.Join(home, "f")
	require.NoError(t, os.WriteFile(p, []byte("v1\n"), 0o600))
	o, _, rc, err := s.request("Q", "100\n"+p+"\n"+filepath.Join(home, "none")+"\n"+home)
	require.NoError(t, err)
	assert.Equal(t, 0, rc)
	records := strings.Split(string(o), "\x1e\n")
	require.Len(t, records, 4, "%q", o)
	assert.True(t, strings.HasPrefix(records[0], "file 600 "), records[0])
	assert.Contains(t, records[0], "\nC:djEK")
	assert.Equal(t, "absent\n", records[1])
	assert.True(t, strings.HasPrefix(records[2], "directory "), records[2])
	assert.Equal(t, "", records[3])
}
