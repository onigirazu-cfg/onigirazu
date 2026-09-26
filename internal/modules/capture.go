package modules

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Rollback: before a module changes a file, the registry captures what was
// there (kind, mode, owner, group and, for a text file up to maxCaptureSize,
// the content). A changed task carries it as TaskResult.Before; apply keeps it
// in the run's snapshot and rollback restores it.

const maxCaptureSize = 1 << 20

// captureModules are the modules whose target file is captured, with the
// argument that names it
var captureModules = map[string][]string{
	"copy": {"dest"}, "template": {"dest"}, "lineinfile": {"path", "dest", "name"},
	"blockinfile": {"path", "dest", "name"}, "replace": {"path", "dest", "name"}, "file": {"path", "dest", "name"},
}

// captureBefore describes the target of a file module on the host before the
// task runs; nil when the module is not captured or the target is unknown
func captureBefore(ctx context.Context, host types.Host, module string, args map[string]interface{}) map[string]interface{} {
	switch module {
	case "apt", "yum", "dnf", "package":
		return capturePackages(ctx, host, args)
	case "service", "systemd":
		return captureService(ctx, host, args)
	case "user", "group":
		return captureAccount(ctx, host, module, args)
	}
	keys, ok := captureModules[module]
	if !ok {
		return nil
	}
	path := ""
	for _, k := range keys {
		if path = getStringArg(args, k, ""); path != "" {
			break
		}
	}
	if path == "" || strings.Contains(path, "{{") {
		return nil
	}
	q := shellQuote(path)
	out, err := runShellOnHost(ctx, host, args, fmt.Sprintf(
		`p=%s; if [ -L "$p" ]; then k=link; elif [ -d "$p" ]; then k=directory; elif [ -f "$p" ]; then k=file; elif [ -e "$p" ]; then k=other; else echo absent; exit 0; fi
echo "$k $( (stat -c '%%a %%U %%G %%s' "$p" 2>/dev/null || stat -f '%%Lp %%Su %%Sg %%z' "$p") )"`, q))
	if err != nil {
		return map[string]interface{}{"path": path, "error": err.Error()}
	}
	f := strings.Fields(strings.TrimSpace(out))
	before := map[string]interface{}{"path": path}
	if len(f) == 1 && f[0] == "absent" {
		before["kind"] = "absent"
		return withBecome(before, args)
	}
	if len(f) != 5 {
		return map[string]interface{}{"path": path, "error": fmt.Sprintf("unexpected stat output %q", out)}
	}
	mode := f[1]
	for len(mode) < 4 {
		mode = "0" + mode
	}
	before["kind"], before["mode"], before["owner"], before["group"] = f[0], mode, f[2], f[3]
	if size, _ := strconv.Atoi(f[4]); f[0] == "file" && size <= maxCaptureSize {
		data, exists, err := readHostFile(ctx, host, args, path)
		if err == nil && exists && utf8.Valid(data) && bytes.IndexByte(data, 0) < 0 {
			before["content"] = string(data)
		}
	}
	return withBecome(before, args)
}

// withBecome keeps the task's escalation, so the restore can write where the
// task wrote
func withBecome(before, args map[string]interface{}) map[string]interface{} {
	for _, k := range []string{"_become", "_become_user", "_become_method"} {
		if v, ok := args[k]; ok {
			before[k] = v
		}
	}
	return before
}

// packageNames reads name as a list, a comma separated string or one name
func packageNames(args map[string]interface{}) []string {
	var names []string
	switch v := args["name"].(type) {
	case string:
		for _, n := range strings.Split(v, ",") {
			if n = strings.TrimSpace(n); n != "" {
				names = append(names, n)
			}
		}
	case []interface{}:
		for _, n := range v {
			names = append(names, fmt.Sprint(n))
		}
	}
	return names
}

// capturePackages records which of the task's packages were installed
func capturePackages(ctx context.Context, host types.Host, args map[string]interface{}) map[string]interface{} {
	names := packageNames(args)
	if len(names) == 0 {
		return nil
	}
	var installed []interface{}
	for _, n := range names {
		q := shellQuote(n)
		out, err := runShellOnHost(ctx, host, args, fmt.Sprintf(
			"(dpkg-query -W -f='${Status}' %s 2>/dev/null | grep -q 'install ok installed' || rpm -q %s >/dev/null 2>&1) && echo yes || echo no", q, q))
		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
		if strings.TrimSpace(out) == "yes" {
			installed = append(installed, n)
		}
	}
	all := make([]interface{}, len(names))
	for i, n := range names {
		all[i] = n
	}
	return withBecome(map[string]interface{}{
		"kind": "packages", "names": all, "installed": installed, "state": getStringArg(args, "state", "present"),
	}, args)
}

// captureService records whether a service was running and enabled
func captureService(ctx context.Context, host types.Host, args map[string]interface{}) map[string]interface{} {
	name := getStringArg(args, "name", "")
	if name == "" {
		return nil
	}
	q := shellQuote(name)
	out, err := runShellOnHost(ctx, host, args, fmt.Sprintf(
		"command -v systemctl >/dev/null || exit 3; echo $(systemctl is-active %s 2>/dev/null) $(systemctl is-enabled %s 2>/dev/null)", q, q))
	if err != nil {
		return map[string]interface{}{"error": "no systemctl"}
	}
	f := strings.Fields(out)
	for len(f) < 2 {
		f = append(f, "unknown")
	}
	return withBecome(map[string]interface{}{"kind": "service", "name": name, "active": f[0], "enabled": f[1]}, args)
}

// captureAccount records whether a user or group existed
func captureAccount(ctx context.Context, host types.Host, module string, args map[string]interface{}) map[string]interface{} {
	name := getStringArg(args, "name", "")
	if name == "" {
		return nil
	}
	db := "passwd"
	if module == "group" {
		db = "group"
	}
	_, err := runOnHost(ctx, host, args, "getent", db, name)
	return withBecome(map[string]interface{}{"kind": module, "name": name, "exists": err == nil}, args)
}
