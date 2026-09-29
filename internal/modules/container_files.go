package modules

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// inContainer tells whether the host is reached with "<runtime> exec"
// (ansible_connection docker or podman) instead of SSH
func inContainer(host types.Host) bool {
	_, _, ok := sshpkg.Container(host)
	return ok
}

// putContainerFile writes data to path inside the host's container with the
// given mode. The content goes through stdin, never on a command line.
func putContainerFile(ctx context.Context, host types.Host, path string, data []byte, mode os.FileMode) error {
	runtime, name, _ := sshpkg.Container(host)
	argv := []string{"exec", "-i"}
	if host.User != "" {
		argv = append(argv, "-u", host.User)
	}
	argv = append(argv, name, "sh", "-c",
		fmt.Sprintf("umask 077 && cat > %s && chmod %04o %s", shellQuote(path), mode.Perm(), shellQuote(path)))
	// #nosec G204 -- the runtime and container come from the inventory
	cmd := exec.CommandContext(ctx, runtime, argv...)
	cmd.Stdin = bytes.NewReader(data)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to write %s in container %s: %v: %s", path, name, err, bytes.TrimSpace(out))
	}
	return nil
}
