package modules

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ReplaceModule replaces every match of a regular expression in a file on
// the host, as Ansible's replace does (multiline mode, \1 back references)
type ReplaceModule struct {
	*BaseModule
}

// NewReplaceModule creates the replace module
func NewReplaceModule() *ReplaceModule {
	return &ReplaceModule{BaseModule: NewBaseModule("replace")}
}

func (m *ReplaceModule) GetDescription() string {
	return "Replace all matches of a regular expression in a file"
}

var pythonBackref = regexp.MustCompile(`\\(\d+)|\\g<(\w+)>`)

func (m *ReplaceModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{
		TaskName: taskName(args), Host: host.Name, Module: m.name,
		Timestamp: start, Success: true, Output: map[string]interface{}{},
	}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success, result.Error = false, msg
		result.Duration = time.Since(start)
		return result, nil
	}
	if err := m.Validate(args); err != nil {
		return fail(err.Error())
	}
	path := filePath(args)
	re, err := regexp.Compile("(?m)" + getStringArg(args, "regexp", ""))
	if err != nil {
		return fail(fmt.Sprintf("invalid regexp: %v", err))
	}
	// Python's \1 and \g<name> are $1 and ${name} in Go
	repl := pythonBackref.ReplaceAllStringFunc(getStringArg(args, "replace", ""), func(ref string) string {
		sub := pythonBackref.FindStringSubmatch(ref)
		if sub[1] != "" {
			return "${" + sub[1] + "}"
		}
		return "${" + sub[2] + "}"
	})

	data, exists, err := readHostFile(ctx, host, args, path)
	if err != nil {
		return fail(err.Error())
	}
	if !exists {
		return fail(fmt.Sprintf("path %s does not exist", path))
	}
	before := string(data)
	after := re.ReplaceAllString(before, repl)
	count := len(re.FindAllStringIndex(before, -1))
	result.Output["msg"] = fmt.Sprintf("%d replacements made", count)
	if after == before {
		result.Duration = time.Since(start)
		return result, nil
	}
	result.Changed = true
	addDiff(args, &result, path, diffText(data, true), diffText([]byte(after), true))
	if inCheckMode(args) {
		result.Duration = time.Since(start)
		return result, nil
	}
	if getBoolArg(args, "backup", false) {
		backup := fmt.Sprintf("%s.%s~", path, time.Now().Format("20060102150405"))
		if err := writeHostFile(ctx, host, args, backup, data, 0); err != nil {
			return fail(err.Error())
		}
		result.Output["backup_file"] = backup
	}
	if err := writeHostFile(ctx, host, args, path, []byte(after), 0); err != nil {
		return fail(err.Error())
	}
	result.Duration = time.Since(start)
	return result, nil
}

func (m *ReplaceModule) Validate(args map[string]interface{}) error {
	if filePath(args) == "" {
		return fmt.Errorf("argument 'path' is required")
	}
	if getStringArg(args, "regexp", "") == "" {
		return fmt.Errorf("argument 'regexp' is required")
	}
	return nil
}
