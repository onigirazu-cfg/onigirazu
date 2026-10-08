package executor

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
	"github.com/onigirazu-cfg/onigirazu/internal/winrm"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// CommandExecutor handles command execution on local or remote hosts
type CommandExecutor struct {
	host         types.Host
	sshClient    *sshpkg.Client
	usePool      bool
	poolReleased bool
	become       bool
	becomeUser   string
	becomeMethod string
	// becomePassword goes to sudo -S on stdin; empty: sudo -n
	becomePassword string
	env            string // "env 'K=V' ..." for the task's environment, or empty
	// container: commands run through "<runtime> exec" (ansible_connection
	// docker or podman) instead of SSH
	runtime, container, containerUser string
}

// containerCmd is the command running a shell line in the host's container
func (e *CommandExecutor) containerCmd(ctx context.Context, line string) *exec.Cmd {
	argv := []string{"exec", "-i"}
	if e.containerUser != "" {
		argv = append(argv, "-u", e.containerUser)
	}
	argv = append(argv, e.container, "sh", "-c", line)
	// #nosec G204 -- the runtime and container come from the inventory
	return exec.CommandContext(ctx, e.runtime, argv...)
}

// NewCommandExecutor creates a new command executor for the given host
// Uses connection pooling for remote hosts by default
func NewCommandExecutor(host types.Host) (*CommandExecutor, error) {
	executor := &CommandExecutor{
		host:    host,
		usePool: true, // Enable pooling by default
	}

	isLocal := sshpkg.IsLocal(host)
	if runtime, name, ok := sshpkg.Container(host); ok {
		executor.runtime, executor.container, executor.containerUser = runtime, name, host.User
		isLocal = true // no SSH client
	}

	// If it's not a local host, get SSH client from pool
	if !isLocal {
		pool := sshpkg.GetGlobalPool()
		client, err := pool.GetConnection(host)
		if err != nil {
			return nil, err
		}
		executor.sshClient = client
	}
	executor.becomePassword = host.BecomePassword
	if host.Become {
		executor.SetBecome(true, host.BecomeUser, host.BecomeMethod)
	}
	executor.setEnvironment(host.Environment)
	executor.prewarm()

	return executor, nil
}

// prewarm starts the connection's command servers in the background, the
// become one too: the first command then does not wait for them one after
// the other (each start is an SSH session, sudo another)
func (e *CommandExecutor) prewarm() {
	// a Windows host over SSH has no sh to start a server with
	if e.sshClient == nil || winrm.IsWindows(e.host) {
		return
	}
	e.sshClient.Prewarm("")
	if e.become && e.becomeMethod == "sudo" && e.becomePassword == "" {
		e.sshClient.Prewarm(e.becomeUser)
	}
}

// NewCommandExecutorWithoutPool creates a new command executor without using connection pool
// Useful for testing or when connection pooling is not desired
func NewCommandExecutorWithoutPool(host types.Host) (*CommandExecutor, error) {
	executor := &CommandExecutor{
		host:    host,
		usePool: false,
	}

	isLocal := sshpkg.IsLocal(host)

	// If it's not a local host, create SSH client directly
	if !isLocal {
		client, err := sshpkg.NewClient(host)
		if err != nil {
			return nil, err
		}
		executor.sshClient = client
	}
	executor.becomePassword = host.BecomePassword
	if host.Become {
		executor.SetBecome(true, host.BecomeUser, host.BecomeMethod)
	}
	executor.setEnvironment(host.Environment)

	return executor, nil
}

// SetBecome enables privilege escalation for command execution
func (e *CommandExecutor) SetBecome(become bool, becomeUser, becomeMethod string) {
	e.become = become
	e.becomeUser = becomeUser
	if becomeMethod == "" {
		e.becomeMethod = "sudo" // Default to sudo
	} else {
		e.becomeMethod = becomeMethod
	}
	if e.becomeUser == "" {
		e.becomeUser = "root" // Default to root
	}
}

// wrapWithBecome runs the whole command line through a root (or become user)
// shell, so redirections, pipes and && chains are escalated too - not only the
// first word, which is all "sudo cmd > file" would cover.
// setEnvironment keeps the task's environment as "env 'K=V' ..." (sorted)
func (e *CommandExecutor) setEnvironment(env map[string]string) {
	if len(env) == 0 {
		e.env = ""
		return
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := []string{"env"}
	for _, k := range keys {
		parts = append(parts, shellQuote(k+"="+env[k]))
	}
	e.env = strings.Join(parts, " ")
}

// withEnvironment runs command under the task's environment. The command
// goes through its own shell, so $VAR in it sees the new value.
func (e *CommandExecutor) withEnvironment(command string) string {
	if e.env == "" {
		return command
	}
	return e.env + " sh -c " + shellQuote(command)
}

func (e *CommandExecutor) wrapWithBecome(command string) string {
	if !e.become {
		return command
	}

	shell := "sh -c " + shellQuote(command)
	switch e.becomeMethod {
	case "su":
		// su -c already hands the whole line to the target user's shell
		if e.becomeUser == "root" {
			return "su -c " + shellQuote(command)
		}
		return fmt.Sprintf("su %s -c %s", e.becomeUser, shellQuote(command))
	case "doas":
		if e.becomeUser == "root" {
			return "doas " + shell
		}
		return fmt.Sprintf("doas -u %s %s", shellQuote(e.becomeUser), shell)
	default: // sudo
		// with a password it reaches sudo on stdin through printf, a shell
		// builtin: it shows in no process list
		sudo := "sudo -n"
		if e.becomePassword != "" {
			sudo = "printf '%s\\n' " + shellQuote(e.becomePassword) + " | sudo -S -p ''"
		}
		if e.becomeUser == "root" {
			return sudo + " " + shell
		}
		return fmt.Sprintf("%s -u %s %s", sudo, shellQuote(e.becomeUser), shell)
	}
}

// withOutput appends the last lines of a failed command's output to its
// error: "exit status 1" alone says nothing about what went wrong
func withOutput(err error, output string) error {
	if err == nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return err
	}
	if len(lines) > 5 {
		lines = lines[len(lines)-5:]
	}
	return fmt.Errorf("%w: %s", err, strings.Join(lines, " | "))
}

// commandLine builds a shell command line: command is used as written (it may
// contain shell syntax), each separate argument is quoted as one word
func commandLine(command string, args []string) string {
	if len(args) == 0 {
		return command
	}
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = shellQuote(a)
	}
	return command + " " + strings.Join(quoted, " ")
}

// shellQuote quotes s as one POSIX shell word
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Execute runs a command on the appropriate host (local or remote)
func (e *CommandExecutor) Execute(command string, args ...string) (string, error) {
	out, err := e.execute(command, args...)
	return out, withOutput(err, out)
}

func (e *CommandExecutor) execute(command string, args ...string) (string, error) {
	fullCommand := e.withEnvironment(commandLine(command, args))

	if e.sshClient != nil {
		// Execute on remote host via SSH; exec adds become
		out, err := e.executeSSHWithContext(context.Background(), fullCommand)
		if err != nil {
			return out, fmt.Errorf("command failed: %w", err)
		}
		return out, nil
	}
	// Wrap with become if enabled
	fullCommand = e.wrapWithBecome(fullCommand)
	if e.container != "" {
		out, err := e.containerCmd(context.Background(), fullCommand).CombinedOutput()
		if err != nil {
			return string(out), fmt.Errorf("command failed: %w", err)
		}
		return string(out), nil
	} else if e.become || e.env != "" {
		// A single string with spaces goes through sh -c
		return e.executeLocal(fullCommand)
	} else {
		// Execute locally
		return e.executeLocal(command, args...)
	}
}

// ExecuteWithContext runs a command with context on the appropriate host
func (e *CommandExecutor) ExecuteWithContext(ctx context.Context, command string, args ...string) (string, error) {
	out, err := e.executeWithContext(ctx, command, args...)
	return out, withOutput(err, out)
}

func (e *CommandExecutor) executeWithContext(ctx context.Context, command string, args ...string) (string, error) {
	fullCommand := e.withEnvironment(commandLine(command, args))

	if e.sshClient != nil {
		// Execute on remote host via SSH with context support; exec adds become
		return e.executeSSHWithContext(ctx, fullCommand)
	}
	// Wrap with become if enabled
	fullCommand = e.wrapWithBecome(fullCommand)
	if e.container != "" {
		output, err := e.containerCmd(ctx, fullCommand).CombinedOutput()
		return string(output), err
	} else if e.become || e.env != "" {
		// #nosec G204 -- privilege escalation wraps the module's own command
		cmd := exec.CommandContext(ctx, "sh", "-c", fullCommand)
		output, err := cmd.CombinedOutput()
		return string(output), err
	} else {
		// Execute locally with context
		cmd := exec.CommandContext(ctx, command, args...)
		output, err := cmd.CombinedOutput()
		return string(output), err
	}
}

// ExecuteContext is an alias for ExecuteWithContext for compatibility with ModuleExecutor interface
func (e *CommandExecutor) ExecuteContext(ctx context.Context, command string, args ...string) (string, error) {
	return e.ExecuteWithContext(ctx, command, args...)
}

// ExecuteWithTimeout runs a command with timeout
func (e *CommandExecutor) ExecuteWithTimeout(command string, timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return e.ExecuteWithContext(ctx, command, args...)
}

// executeSSHWithContext runs a command on the host; stderr comes with
// stdout, and a non-zero exit is an error, as with CombinedOutput
func (e *CommandExecutor) executeSSHWithContext(ctx context.Context, command string) (string, error) {
	out, _, rc, err := e.exec(ctx, command, true)
	if err != nil {
		return string(out), err
	}
	if rc != 0 {
		return string(out), &sshpkg.ExitStatusError{Status: rc}
	}
	return string(out), nil
}

// exec runs a command (become not applied yet) through the connection's
// shell; a command that never reached a host whose connection died is sent
// again on a new connection
func (e *CommandExecutor) exec(ctx context.Context, command string, combined bool) ([]byte, []byte, int, error) {
	out, errOut, rc, err := e.execOnce(ctx, command, combined)
	if err == nil || !e.usePool || !errors.Is(err, sshpkg.ErrNotSent) {
		return out, errOut, rc, err
	}
	fresh, rerr := sshpkg.GetGlobalPool().Reconnect(e.host)
	if rerr != nil {
		return nil, nil, 0, fmt.Errorf("%w (reconnect: %v)", err, rerr)
	}
	e.sshClient = fresh
	return e.execOnce(ctx, command, combined)
}

// execOnce sends a become command to a command server started with sudo
// once per connection when it can (sudo without a password), else wraps it
// in sudo/su/doas
func (e *CommandExecutor) execOnce(ctx context.Context, command string, combined bool) ([]byte, []byte, int, error) {
	if e.become && e.becomeMethod == "sudo" && e.becomePassword == "" {
		out, errOut, rc, served, err := e.sshClient.ExecAs(ctx, e.becomeUser, command, combined)
		if served {
			return out, errOut, rc, err
		}
	}
	return e.sshClient.Exec(ctx, e.wrapWithBecome(command), combined)
}

// Probe describes path through the host's command server without a
// process (see ssh.Client.Probe), as the become user when become is sudo
// without a password; served is false when that is not possible
func (e *CommandExecutor) Probe(ctx context.Context, path string, limit int) (string, bool, error) {
	if e.sshClient == nil {
		return "", false, nil
	}
	user := ""
	if e.become {
		if e.becomeMethod != "sudo" || e.becomePassword != "" {
			return "", false, nil
		}
		user = e.becomeUser
	}
	out, served, err := e.sshClient.Probe(ctx, user, path, limit)
	if errors.Is(err, sshpkg.ErrNotSent) {
		return "", false, nil // the shell probe reconnects
	}
	return string(out), served, err
}

// ProbeMany is Probe for several paths in one round trip
func (e *CommandExecutor) ProbeMany(ctx context.Context, paths []string, limit int) (string, bool, error) {
	if e.sshClient == nil {
		return "", false, nil
	}
	user := ""
	if e.become {
		if e.becomeMethod != "sudo" || e.becomePassword != "" {
			return "", false, nil
		}
		user = e.becomeUser
	}
	out, served, err := e.sshClient.ProbeMany(ctx, user, paths, limit)
	if errors.Is(err, sshpkg.ErrNotSent) {
		return "", false, nil
	}
	return string(out), served, err
}

// executeLocal executes a command locally
func (e *CommandExecutor) executeLocal(command string, args ...string) (string, error) {
	// Separate arguments are passed as argv, exactly as the remote path quotes them
	if len(args) > 0 {
		// #nosec G204 -- modules run the commands they manage
		output, err := exec.Command(command, args...).CombinedOutput()
		return string(output), err
	}
	// A single command line may use shell syntax (pipes, redirects, ...)
	if strings.ContainsAny(command, "|&;<>()$`\\\"' \t\n*?[]{}") {
		// #nosec G204 - This is intentional: we need shell execution for complex commands with pipes, redirects, etc.
		cmd := exec.Command("sh", "-c", command)
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

	// Simple command without args - execute directly
	cmd := exec.Command(command)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// Close releases the connection back to the pool or closes it if not using pool
func (e *CommandExecutor) Close() error {
	if e.sshClient != nil && !e.poolReleased {
		e.poolReleased = true
		if e.usePool {
			// Return connection to pool for reuse
			pool := sshpkg.GetGlobalPool()
			pool.ReleaseConnection(e.host)
			return nil
		} else {
			// Close connection directly if not using pool
			return e.sshClient.Close()
		}
	}
	return nil
}

// IsRemote returns true if this executor is for a remote host (SSH or a
// container)
func (e *CommandExecutor) IsRemote() bool {
	return e.sshClient != nil || e.container != ""
}
