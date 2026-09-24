package modules

import (
	"context"
	"fmt"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
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
