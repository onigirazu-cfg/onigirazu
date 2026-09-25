package modules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// shellQuote quotes s for safe use as a single POSIX shell word.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// shellJoin quotes every word and joins them into one command line.
func shellJoin(argv ...string) string {
	quoted := make([]string, len(argv))
	for i, a := range argv {
		quoted[i] = shellQuote(a)
	}
	return strings.Join(quoted, " ")
}

// runOnHost runs argv on the task's target host (locally or over SSH), honoring
// the become settings in args. It returns the combined output.
func runOnHost(ctx context.Context, host types.Host, args map[string]interface{}, argv ...string) (string, error) {
	return runShellOnHost(ctx, host, args, shellJoin(argv...))
}

// runShellOnHost runs a shell script on the task's target host, honoring become.
func runShellOnHost(ctx context.Context, host types.Host, args map[string]interface{}, script string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	exec, err := executor.NewCommandExecutor(host)
	if err != nil {
		return "", fmt.Errorf("failed to create executor: %w", err)
	}
	defer exec.Close()

	if become, ok := args["_become"].(bool); ok && become {
		becomeUser, _ := args["_become_user"].(string)
		becomeMethod, _ := args["_become_method"].(string)
		exec.SetBecome(true, becomeUser, becomeMethod)
	}

	// One command string keeps quoting identical for local and SSH execution
	out, err := exec.Execute("sh -c " + shellQuote(script))
	if err != nil {
		return out, fmt.Errorf("%w: %s", err, strings.TrimSpace(out))
	}
	return out, nil
}

// ensureOwnership sets owner and/or group of path on host when they differ from the
// current ones. It reports whether anything was changed. Become settings from args apply.
func ensureOwnership(ctx context.Context, host types.Host, args map[string]interface{}, path, owner, group string) (bool, error) {
	if owner == "" && group == "" {
		return false, nil
	}

	qPath := shellQuote(path)
	// GNU stat first, BSD stat as a fallback
	current, err := runShellOnHost(ctx, host, args, fmt.Sprintf("stat -c '%%U:%%G' %s 2>/dev/null || stat -f '%%Su:%%Sg' %s", qPath, qPath))
	if err != nil {
		return false, fmt.Errorf("failed to read ownership of %s: %w", path, err)
	}
	currentOwner, currentGroup, _ := strings.Cut(strings.TrimSpace(current), ":")

	wantOwner, wantGroup := owner, group
	if wantOwner == "" {
		wantOwner = currentOwner
	}
	if wantGroup == "" {
		wantGroup = currentGroup
	}
	if wantOwner == currentOwner && wantGroup == currentGroup {
		return false, nil
	}

	if _, err := runOnHost(ctx, host, args, "chown", wantOwner+":"+wantGroup, path); err != nil {
		return false, fmt.Errorf("failed to set ownership of %s to %s:%s: %w", path, wantOwner, wantGroup, err)
	}
	return true, nil
}

// remoteFile describes a file on the target host as root (or the become user)
// sees it, so root-only files can be compared without downloading them.
type remoteFile struct {
	Exists bool
	Mode   os.FileMode
	Owner  string
	Group  string
	SHA256 string
}

// statRemoteFile reads mode, owner and content hash of path on the host.
func statRemoteFile(ctx context.Context, host types.Host, args map[string]interface{}, path string) (remoteFile, error) {
	q := shellQuote(path)
	out, err := runShellOnHost(ctx, host, args, fmt.Sprintf(
		"if [ -e %s ]; then stat -c '%%a %%U %%G' %s && sha256sum %s | cut -d' ' -f1; else echo absent; fi", q, q, q))
	if err != nil {
		return remoteFile{}, fmt.Errorf("failed to stat %s: %w", path, err)
	}
	lines := strings.Fields(strings.TrimSpace(out))
	if len(lines) == 1 && lines[0] == "absent" {
		return remoteFile{}, nil
	}
	if len(lines) != 4 {
		return remoteFile{}, fmt.Errorf("unexpected stat output for %s: %q", path, out)
	}
	mode, err := strconv.ParseUint(lines[0], 8, 32)
	if err != nil {
		return remoteFile{}, fmt.Errorf("unexpected mode for %s: %q", path, lines[0])
	}
	return remoteFile{Exists: true, Mode: os.FileMode(mode), Owner: lines[1], Group: lines[2], SHA256: lines[3]}, nil
}

// installRemoteFile writes data to path on the host: it uploads to a private
// temporary file over SFTP and moves it into place with install(1), so with
// become it can write where the SSH user cannot. Parent directories are
// created; the owner of an existing file is kept.
func installRemoteFile(ctx context.Context, host types.Host, args map[string]interface{}, client *sshpkg.Client,
	path string, data []byte, mode os.FileMode, existing remoteFile) error {
	tmp := fmt.Sprintf("/tmp/.onigirazu-%d-%s", time.Now().UnixNano(), filepath.Base(path))
	if err := client.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("failed to upload %s: %w", path, err)
	}
	owner := ""
	if existing.Exists {
		owner = fmt.Sprintf("-o %s -g %s ", shellQuote(existing.Owner), shellQuote(existing.Group))
	}
	qt := shellQuote(tmp)
	_, err := runShellOnHost(ctx, host, args, fmt.Sprintf("install -D %s-m %04o %s %s; rc=$?; rm -f %s; exit $rc",
		owner, mode.Perm(), qt, shellQuote(path), qt))
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}
