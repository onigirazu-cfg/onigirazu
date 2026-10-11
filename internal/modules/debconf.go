package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// DebconfModule sets debconf selections (Ansible's debconf): name (the
// package), question, vtype, value, unseen
type DebconfModule struct{ *BaseModule }

func NewDebconfModule() *DebconfModule { return &DebconfModule{BaseModule: NewBaseModule("debconf")} }

func (m *DebconfModule) GetDescription() string { return "Configure a package through debconf" }

func (m *DebconfModule) Validate(args map[string]interface{}) error {
	if getStringArg(args, "name", "") == "" {
		return fmt.Errorf("debconf requires 'name'")
	}
	q := getStringArg(args, "question", "")
	if q != "" && (getStringArg(args, "vtype", "") == "" || args["value"] == nil) {
		return fmt.Errorf("debconf requires 'vtype' and 'value' with a question")
	}
	return nil
}

func (m *DebconfModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success, result.Error, result.Duration = false, msg, time.Since(start)
		return result, nil
	}
	if err := m.Validate(args); err != nil {
		return fail(err.Error())
	}
	name := getStringArg(args, "name", "")
	question := getStringArg(args, "question", "")
	out, err := runShellOnHost(ctx, host, args, "debconf-show "+shellQuote(name)+" 2>/dev/null || true")
	if err != nil {
		return fail("debconf: " + err.Error())
	}
	current := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimLeft(strings.TrimSpace(line), "* ")
		if q, v, ok := strings.Cut(line, ":"); ok {
			current[strings.TrimSpace(q)] = strings.TrimSpace(v)
		}
	}
	result.Output["current"] = current
	if question == "" {
		result.Duration = time.Since(start)
		return result, nil
	}
	value := fmt.Sprint(args["value"])
	if b, ok := args["value"].(bool); ok {
		value = map[bool]string{true: "true", false: "false"}[b]
	}
	if have, ok := current[question]; ok && have == value {
		result.Duration = time.Since(start)
		return result, nil
	}
	result.Changed = true
	if inCheckMode(args) {
		result.Duration = time.Since(start)
		return result, nil
	}
	line := fmt.Sprintf("%s %s %s %s", name, question, getStringArg(args, "vtype", "string"), value)
	cmd := "printf '%s\\n' " + shellQuote(line) + " | debconf-set-selections"
	if getBoolArg(args, "unseen", false) {
		cmd = "printf '%s\\n' " + shellQuote(line) + " | debconf-set-selections && echo " + shellQuote("set "+name+"/"+question+" seen false") + " | debconf-communicate >/dev/null"
	}
	if _, err := runShellOnHost(ctx, host, args, cmd); err != nil {
		return fail("debconf: " + err.Error())
	}
	result.Duration = time.Since(start)
	return result, nil
}
