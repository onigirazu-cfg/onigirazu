package modules

import (
	"context"
	"encoding/base64"
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
	// The executor already adds the output to the error
	return exec.Execute("sh -c " + shellQuote(script))
}

// ensureOwnership sets owner and/or group of path on host when they differ from the
// current ones. It reports whether anything was changed. Become settings from args apply.
func ensureOwnership(ctx context.Context, host types.Host, args map[string]interface{}, path, owner, group string) (bool, error) {
	if owner == "" && group == "" {
		return false, nil
	}

	qPath := shellQuote(path)
	// GNU stat first, BSD stat as a fallback
	current, err := runShellOnHost(ctx, host, args, fmt.Sprintf("stat -c '%%U:%%G:%%u:%%g' %s 2>/dev/null || stat -f '%%Su:%%Sg:%%u:%%g' %s", qPath, qPath))
	if err != nil {
		if inCheckMode(args) {
			return true, nil // the file does not exist yet; its ownership would be set
		}
		return false, fmt.Errorf("failed to read ownership of %s: %w", path, err)
	}
	// name:group:uid:gid; owner and group may be given by name or id
	ids := strings.Split(strings.TrimSpace(current), ":")
	for len(ids) < 4 {
		ids = append(ids, "")
	}
	ownerOK := owner == "" || owner == ids[0] || owner == ids[2]
	groupOK := group == "" || group == ids[1] || group == ids[3]
	if ownerOK && groupOK {
		return false, nil
	}
	wantOwner, wantGroup := owner, group
	if wantOwner == "" {
		wantOwner = ids[0]
	}
	if wantGroup == "" {
		wantGroup = ids[1]
	}
	if inCheckMode(args) {
		return true, nil
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
		// GNU stat/sha256sum, with BSD fallbacks (a local macOS host)
		"if [ -e %s ]; then (stat -c '%%a %%U %%G' %s 2>/dev/null || stat -f '%%Lp %%Su %%Sg' %s) && "+
			"(sha256sum %s 2>/dev/null || shasum -a 256 %s) | cut -d' ' -f1; else echo absent; fi", q, q, q, q, q))
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
	// a local host has no SSH client: the temporary file is written directly
	if client == nil {
		if err := os.WriteFile(tmp, data, 0600); err != nil {
			return fmt.Errorf("failed to write %s: %w", tmp, err)
		}
	} else if err := client.WriteFile(tmp, data, 0600); err != nil {
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

// readHostFile returns the content of path on the host (with become), and
// whether it exists. Content travels base64-encoded, so any bytes survive.
func readHostFile(ctx context.Context, host types.Host, args map[string]interface{}, path string) ([]byte, bool, error) {
	q := shellQuote(path)
	out, err := runShellOnHost(ctx, host, args, fmt.Sprintf(
		"if [ -e %s ]; then printf 'present:'; base64 < %s | tr -d '\\n'; else printf absent; fi", q, q))
	if err != nil {
		return nil, false, fmt.Errorf("failed to read %s: %w", path, err)
	}
	out = strings.TrimSpace(out)
	if out == "absent" {
		return nil, false, nil
	}
	encoded, ok := strings.CutPrefix(out, "present:")
	if !ok {
		return nil, false, fmt.Errorf("unexpected output reading %s", path)
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, false, fmt.Errorf("failed to decode %s: %w", path, err)
	}
	return data, true, nil
}

// writeHostFile writes data to path on the host (with become) through a
// temporary file and install(1). mode 0 keeps the mode of an existing file
// (0644 for a new one); the owner of an existing file is kept.
func writeHostFile(ctx context.Context, host types.Host, args map[string]interface{}, path string, data []byte, mode os.FileMode) error {
	q := shellQuote(path)
	script := fmt.Sprintf(`set -e
p=%s
m=%s
if [ -e "$p" ]; then
  [ -n "$m" ] || m=$(stat -c %%a "$p" 2>/dev/null || stat -f %%Lp "$p")
  o=$(stat -c %%U "$p" 2>/dev/null || stat -f %%Su "$p")
  g=$(stat -c %%G "$p" 2>/dev/null || stat -f %%Sg "$p")
fi
[ -n "$m" ] || m=0644
mkdir -p "$(dirname "$p")"
t=$(mktemp)
trap 'rm -f "$t"' EXIT
printf '%%s' %s | base64 -d > "$t"
if [ -n "$o" ]; then install -m "$m" -o "$o" -g "$g" "$t" "$p"; else install -m "$m" "$t" "$p"; fi`,
		q, shellQuote(modeString(mode)), shellQuote(base64.StdEncoding.EncodeToString(data)))
	if _, err := runShellOnHost(ctx, host, args, script); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

func modeString(mode os.FileMode) string {
	if mode == 0 {
		return ""
	}
	return fmt.Sprintf("%04o", mode.Perm())
}

// inCheckMode reports whether the task runs in check mode: the module
// reports what it would change and changes nothing
func inCheckMode(args map[string]interface{}) bool {
	check, _ := args["_check_mode"].(bool)
	return check
}
