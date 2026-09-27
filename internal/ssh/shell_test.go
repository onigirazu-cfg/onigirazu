package ssh

import (
	"bufio"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// the shell script and its framing, run by a local sh
func TestRemoteShellProtocol(t *testing.T) {
	cmd := exec.Command("sh", "-c", shellScript)
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	out, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	defer func() { _ = stdin.Close(); _ = cmd.Wait() }()
	s := &remoteShell{stdin: stdin, stdout: bufio.NewReader(out)}
	ready, err := s.stdout.ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "ONIGIRAZU-READY\n", ready)

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

func TestRemoteShellBackgroundChild(t *testing.T) {
	cmd := exec.Command("sh", "-c", shellScript)
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
