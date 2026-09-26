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
