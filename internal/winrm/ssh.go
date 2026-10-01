package winrm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	cryptossh "golang.org/x/crypto/ssh"

	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// sshRunner runs on a Windows host through OpenSSH: every command is a
// powershell.exe command line, whatever the server's default shell is
type sshRunner struct {
	host types.Host
}

func (r *sshRunner) run(ctx context.Context, command, stdin string) (Result, error) {
	client, err := sshpkg.GetGlobalPool().GetConnection(r.host)
	if err != nil {
		return Result{}, fmt.Errorf("ssh to %s: %w", r.host.Name, err)
	}
	session, err := client.GetClient().NewSession()
	if err != nil {
		return Result{}, err
	}
	defer session.Close()
	var out, errOut bytes.Buffer
	session.Stdout, session.Stderr = &out, &errOut
	session.Stdin = strings.NewReader(stdin)
	done := make(chan error, 1)
	go func() { done <- session.Run(command) }()
	select {
	case <-ctx.Done():
		_ = session.Signal(cryptossh.SIGKILL)
		_ = session.Close()
		return Result{}, ctx.Err()
	case err = <-done:
	}
	res := Result{Stdout: out.String(), Stderr: cleanCLIXML(errOut.String())}
	var exit *cryptossh.ExitError
	switch {
	case errors.As(err, &exit):
		res.ExitCode = exit.ExitStatus()
	case err != nil:
		return res, err
	}
	return res, nil
}

// RunPS runs the script through the same stdin bootstrap as WinRM
func (r *sshRunner) RunPS(ctx context.Context, script string) (Result, error) {
	return r.run(ctx, EncodedCommand(bootstrap), b64(script))
}

// cmdScript starts cmd.exe /c with the command line as it is: no PowerShell
// parsing of quotes, && or %VAR%
const cmdScript = `$psi = New-Object Diagnostics.ProcessStartInfo 'cmd.exe', ('/c ' + %s)
$psi.UseShellExecute = $false
$psi.RedirectStandardOutput = $true; $psi.RedirectStandardError = $true; $psi.RedirectStandardInput = $true
$p = [Diagnostics.Process]::Start($psi)
$o = $p.StandardOutput.ReadToEndAsync(); $e = $p.StandardError.ReadToEndAsync()
$p.StandardInput.Write(%s); $p.StandardInput.Close()
$p.WaitForExit()
[Console]::Out.Write($o.Result); [Console]::Error.Write($e.Result)
exit $p.ExitCode`

func psString(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// RunCmd runs a command line with cmd.exe
func (r *sshRunner) RunCmd(ctx context.Context, command string) (Result, error) {
	return r.RunCmdInput(ctx, command, "")
}

// RunCmdInput runs a command line with cmd.exe and stdin
func (r *sshRunner) RunCmdInput(ctx context.Context, command, stdin string) (Result, error) {
	return r.RunPS(ctx, fmt.Sprintf(cmdScript, psString(command), psString(stdin)))
}

// Upload writes data to a temporary file in pieces
func (r *sshRunner) Upload(ctx context.Context, data []byte) (string, error) {
	return upload(ctx, r, data)
}
