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

// ensureOwnership sets owner and/or group of path on host when they differ from the
// current ones. It reports whether anything was changed. Become settings from args apply.
func ensureOwnership(ctx context.Context, host types.Host, args map[string]interface{}, path, owner, group string) (bool, error) {
	if owner == "" && group == "" {
		return false, nil
	}

	exec, err := executor.NewCommandExecutor(host)
	if err != nil {
		return false, fmt.Errorf("failed to create executor: %w", err)
	}
	defer exec.Close()

	if become, ok := args["_become"].(bool); ok && become {
		becomeUser, _ := args["_become_user"].(string)
		becomeMethod, _ := args["_become_method"].(string)
		exec.SetBecome(true, becomeUser, becomeMethod)
	}

	// A single command string keeps quoting identical for local and SSH execution
	run := func(script string) (string, error) {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		out, err := exec.Execute("sh -c " + shellQuote(script))
		if err != nil {
			return out, fmt.Errorf("%w: %s", err, strings.TrimSpace(out))
		}
		return strings.TrimSpace(out), nil
	}

	qPath := shellQuote(path)
	// GNU stat first, BSD stat as a fallback
	current, err := run(fmt.Sprintf("stat -c '%%U:%%G' %s 2>/dev/null || stat -f '%%Su:%%Sg' %s", qPath, qPath))
	if err != nil {
		return false, fmt.Errorf("failed to read ownership of %s: %w", path, err)
	}
	currentOwner, currentGroup, _ := strings.Cut(current, ":")

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

	if _, err := run(fmt.Sprintf("chown %s %s", shellQuote(wantOwner+":"+wantGroup), qPath)); err != nil {
		return false, fmt.Errorf("failed to set ownership of %s to %s:%s: %w", path, wantOwner, wantGroup, err)
	}
	return true, nil
}
