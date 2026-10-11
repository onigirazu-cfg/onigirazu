package modules

import (
	"context"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TempfileModule makes a temporary file or directory on the host (Ansible's
// tempfile): state file/directory, path (default the system temp dir),
// prefix, suffix; returns path
type TempfileModule struct{ *BaseModule }

func NewTempfileModule() *TempfileModule {
	return &TempfileModule{BaseModule: NewBaseModule("tempfile")}
}

func (m *TempfileModule) GetDescription() string { return "Create a temporary file or directory" }

func (m *TempfileModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	state := getStringArg(args, "state", "file")
	dir := getStringArg(args, "path", "")
	prefix := getStringArg(args, "prefix", "ansible.")
	suffix := getStringArg(args, "suffix", "")
	if inCheckMode(args) {
		result.Changed = true
		result.Output["msg"] = "would create a temporary " + state
		result.Duration = time.Since(start)
		return result, nil
	}
	flag := ""
	if state == "directory" {
		flag = " -d"
	}
	tmpl := prefix + "XXXXXX" + suffix
	// a bare template lands in the current directory: always say where
	cmd := "mktemp" + flag
	if dir == "" {
		cmd += ` -p "${TMPDIR:-/tmp}"`
	} else {
		cmd += " -p " + shellQuote(dir)
	}
	// mktemp needs the X's at the end: a suffix goes through --suffix on GNU
	if suffix != "" {
		cmd += " --suffix=" + shellQuote(suffix)
		tmpl = prefix + "XXXXXX"
	}
	out, err := runShellOnHost(ctx, host, args, cmd+" "+shellQuote(tmpl))
	if err != nil {
		result.Success, result.Error = false, "tempfile: "+err.Error()
		return result, nil
	}
	result.Changed = true
	result.Output["path"] = strings.TrimSpace(out)
	result.Duration = time.Since(start)
	return result, nil
}
