package winrm

import (
	"context"
	"fmt"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Runner runs commands and PowerShell on a Windows host, over WinRM or SSH
type Runner interface {
	// RunCmd runs a command line with cmd.exe semantics
	RunCmd(ctx context.Context, command string) (Result, error)
	// RunCmdInput runs a command line with stdin
	RunCmdInput(ctx context.Context, command, stdin string) (Result, error)
	// RunPS runs a PowerShell script of any length
	RunPS(ctx context.Context, script string) (Result, error)
	// Upload writes data to a temporary file on the host and returns its path
	Upload(ctx context.Context, data []byte) (string, error)
}

// IsWindowsSSH reports a Windows host reached over SSH: ansible_shell_type
// powershell or cmd, as Ansible marks them
func IsWindowsSSH(host types.Host) bool {
	for _, key := range []string{"onigirazu_shell_type", "ansible_shell_type"} {
		switch strings.ToLower(strings.TrimSpace(fmt.Sprint(host.Vars[key]))) {
		case "powershell", "cmd":
			return true
		}
	}
	return false
}

// IsWindows reports whether the host is a Windows host (WinRM, or SSH with a
// Windows shell type)
func IsWindows(host types.Host) bool {
	return IsWinRM(host) || IsWindowsSSH(host)
}
