package modules

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
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
	"ini_file": {"path", "dest"},
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
	path := capturePath(module, args)
	if path == "" {
		return nil
	}
	// taken for the whole loop already
	if out, ok := loopProbe(ctx, path); ok {
		return parseProbe(path, out, args)
	}
	// one round trip: kind, mode, owner, group, size and, for a file up to
	// maxCaptureSize, its content (the sha256 is taken here), else its sha256
	out, served, err := probeOnHost(ctx, host, args, path, maxCaptureSize)
	if !served {
		out, err = shellProbe(ctx, host, args, path)
	}
	if err != nil {
		return map[string]interface{}{"path": path, "error": err.Error()}
	}
	return parseProbe(path, out, args)
}

// shellProbe prints what the server's probe prints, with shell tools
func shellProbe(ctx context.Context, host types.Host, args map[string]interface{}, path string) (string, error) {
	return runShellOnHost(ctx, host, args, shellProbeFunc+"probe "+shellQuote(path))
}

// shellProbeFunc defines probe PATH: what the server's probe prints, with
// shell tools
var shellProbeFunc = fmt.Sprintf(`probe() {
p=$1; if [ -L "$p" ]; then k=link; elif [ -d "$p" ]; then k=directory; elif [ -f "$p" ]; then k=file; elif [ -e "$p" ]; then k=other; else echo absent; return 0; fi
s=$(stat -c '%%a %%U %%G %%s' "$p" 2>/dev/null || stat -f '%%Lp %%Su %%Sg %%z' "$p")
if [ "$k" = file ]; then
  set -- $s
  if [ "$4" -le %d ]; then echo "$k $s +"; printf 'C:'; base64 < "$p"; else
  h=$( (sha256sum "$p" 2>/dev/null || shasum -a 256 "$p") | cut -d' ' -f1); echo "$k $s $h"; fi
else echo "$k $s -"; fi
}
`, maxCaptureSize)

// parseProbe reads the probe's answer: "absent", or "kind mode owner group
// size sha256|+|-" and for "+" a "C:" line with the content in base64
func parseProbe(path, out string, args map[string]interface{}) map[string]interface{} {
	lines := strings.SplitN(strings.TrimSpace(out), "\n", 2)
	f := strings.Fields(lines[0])
	before := map[string]interface{}{"path": path}
	if len(f) == 1 && f[0] == "absent" {
		before["kind"] = "absent"
		return withBecome(before, args)
	}
	if len(f) != 6 {
		return map[string]interface{}{"path": path, "error": fmt.Sprintf("unexpected stat output %q", out)}
	}
	mode := f[1]
	for len(mode) < 4 {
		mode = "0" + mode
	}
	before["kind"], before["mode"], before["owner"], before["group"] = f[0], mode, f[2], f[3]
	if f[5] != "-" && f[5] != "+" {
		before["sha256"] = f[5]
	}
	if len(lines) == 2 && f[5] == "+" {
		encoded, ok := strings.CutPrefix(strings.Join(strings.Fields(lines[1]), ""), "C:")
		data, err := base64.StdEncoding.DecodeString(encoded)
		if !ok || err != nil {
			return map[string]interface{}{"path": path, "error": fmt.Sprintf("unexpected content of %s", path)}
		}
		sum := sha256.Sum256(data)
		before["sha256"] = hex.EncodeToString(sum[:])
		if utf8.Valid(data) && bytes.IndexByte(data, 0) < 0 {
			before["content"] = string(data)
		}
	}
	return withBecome(before, args)
}

// capturePath is the target of a captured file module, "" for others
func capturePath(module string, args map[string]interface{}) string {
	path := ""
	for _, k := range captureModules[module] {
		if path = getStringArg(args, k, ""); path != "" {
			break
		}
	}
	if strings.Contains(path, "{{") {
		return ""
	}
	return path
}

// captureNative is captureBefore of a file module when the host's command
// server describes the file without a process; nil otherwise
func captureNative(ctx context.Context, host types.Host, module string, args map[string]interface{}) map[string]interface{} {
	path := capturePath(module, args)
	if path == "" {
		return nil
	}
	if out, ok := loopProbe(ctx, path); ok {
		return parseProbe(path, out, args)
	}
	out, served, err := probeOnHost(ctx, host, args, path, maxCaptureSize)
	if !served || err != nil {
		return nil
	}
	return parseProbe(path, out, args)
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
	present, err := queryInstalled(ctx, host, args, names)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	var installed []interface{}
	for _, n := range names {
		if present[n] {
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

// queryInstalled asks the host in one round trip which of the packages are
// installed (dpkg, rpm or pacman)
func queryInstalled(ctx context.Context, host types.Host, args map[string]interface{}, names []string) (map[string]bool, error) {
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = shellQuote(n)
	}
	out, err := runShellOnHost(ctx, host, args, fmt.Sprintf(
		`if command -v dpkg-query >/dev/null 2>&1; then q() { dpkg-query -W -f='${Status}' "$1" 2>/dev/null | grep -q 'install ok installed'; }; `+
			`elif command -v rpm >/dev/null 2>&1; then q() { rpm -q "$1" >/dev/null 2>&1; }; `+
			`elif command -v pacman >/dev/null 2>&1; then q() { pacman -Q "$1" >/dev/null 2>&1; }; `+
			`else q() { false; }; fi; for n in %s; do q "$n" && echo "$n"; done; true`,
		strings.Join(quoted, " ")))
	if err != nil {
		return nil, err
	}
	present := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			present[line] = true
		}
	}
	return present, nil
}

// installedPackages reports which of the packages are installed: from the
// capture taken before the task when it covers them all, else in one query
func installedPackages(ctx context.Context, host types.Host, args map[string]interface{}, names []string) (map[string]bool, error) {
	if before, ok := args["_before"].(map[string]interface{}); ok && before["kind"] == "packages" {
		captured := map[string]bool{}
		if list, ok := before["names"].([]interface{}); ok {
			for _, n := range list {
				captured[fmt.Sprint(n)] = true
			}
		}
		all := true
		for _, n := range names {
			all = all && captured[n]
		}
		if all {
			present := map[string]bool{}
			if list, ok := before["installed"].([]interface{}); ok {
				for _, n := range list {
					present[fmt.Sprint(n)] = true
				}
			}
			return present, nil
		}
	}
	return queryInstalled(ctx, host, args, names)
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
	if module == "user" {
		// the whole account in the same round trip: the module reads it
		// from here
		out, err := runShellOnHost(ctx, host, args, accountScript(name))
		if err != nil {
			return map[string]interface{}{"kind": module, "name": name, "error": err.Error()}
		}
		return withBecome(map[string]interface{}{"kind": module, "name": name, "exists": strings.TrimSpace(out) != "", "account": out}, args)
	}
	_, err := runOnHost(ctx, host, args, "getent", db, name)
	return withBecome(map[string]interface{}{"kind": module, "name": name, "exists": err == nil}, args)
}

// capturedAccountExists is whether the account (user or group) existed when the
// capture before the task looked; known is false without that capture
func capturedAccountExists(args map[string]interface{}, kind, name string) (exists, known bool) {
	before, ok := args["_before"].(map[string]interface{})
	if !ok || before["kind"] != kind || before["name"] != name || before["error"] != nil {
		return false, false
	}
	exists, known = before["exists"].(bool)
	return exists, known
}
