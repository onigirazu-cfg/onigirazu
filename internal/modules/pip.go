package modules

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// PipModule manages Python packages with pip (Ansible's pip): name (list or
// comma separated, "pkg==1.2" pins), version, state present/absent/latest/
// forcereinstall, requirements, virtualenv (created with virtualenv_command,
// default "python3 -m venv"), executable, extra_args
type PipModule struct {
	*BaseModule
}

// NewPipModule creates the pip module
func NewPipModule() *PipModule {
	return &PipModule{BaseModule: NewBaseModule("pip")}
}

func (m *PipModule) GetDescription() string { return "Manage Python packages with pip" }

func (m *PipModule) Validate(args map[string]interface{}) error {
	if len(packageNames(args)) == 0 && getStringArg(args, "requirements", "") == "" {
		return fmt.Errorf("pip requires 'name' or 'requirements'")
	}
	switch getStringArg(args, "state", "present") {
	case "present", "absent", "latest", "forcereinstall":
		return nil
	}
	return fmt.Errorf("pip: state must be present, absent, latest or forcereinstall")
}

var pipSpec = regexp.MustCompile(`^([A-Za-z0-9._-]+)(\[[^\]]*\])?\s*(==|>=|<=|~=|!=|>|<)?\s*(.*)$`)

func (m *PipModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name,
		Timestamp: start, Success: true, Output: map[string]interface{}{}}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success, result.Error, result.Duration = false, msg, time.Since(start)
		return result, nil
	}
	if err := m.Validate(args); err != nil {
		return fail(err.Error())
	}
	state := getStringArg(args, "state", "present")
	venv := getStringArg(args, "virtualenv", "")
	pip := getStringArg(args, "executable", "")
	switch {
	case pip != "":
	case venv != "":
		pip = strings.TrimSuffix(venv, "/") + "/bin/pip"
	default:
		pip = "$(command -v pip3 || command -v pip)"
	}
	if venv != "" && pip == strings.TrimSuffix(venv, "/")+"/bin/pip" {
		if _, err := runOnHost(ctx, host, args, "test", "-x", pip); err != nil {
			if inCheckMode(args) {
				result.Changed = true
				result.Output["msg"] = "would create the virtualenv " + venv
				result.Duration = time.Since(start)
				return result, nil
			}
			cmd := getStringArg(args, "virtualenv_command", "python3 -m venv")
			if _, err := runShellOnHost(ctx, host, args, cmd+" "+shellQuote(venv)); err != nil {
				return fail(fmt.Sprintf("failed to create the virtualenv: %v", err))
			}
			result.Changed = true
		}
		pip = shellQuote(pip)
	} else if getStringArg(args, "executable", "") != "" {
		pip = shellQuote(pip)
	}
	extra := getStringArg(args, "extra_args", "")

	if req := getStringArg(args, "requirements", ""); req != "" {
		before, _ := runShellOnHost(ctx, host, args, pip+" freeze 2>/dev/null")
		if inCheckMode(args) {
			result.Output["msg"] = "requirements are not checked in check mode"
			result.Duration = time.Since(start)
			return result, nil
		}
		flags := "-r " + shellQuote(req)
		if state == "latest" {
			flags = "-U " + flags
		}
		if out, err := runShellOnHost(ctx, host, args, pip+" install "+extra+" "+flags); err != nil {
			return fail(fmt.Sprintf("pip install failed: %v %s", err, out))
		}
		after, _ := runShellOnHost(ctx, host, args, pip+" freeze 2>/dev/null")
		result.Changed = result.Changed || before != after
		result.Duration = time.Since(start)
		return result, nil
	}

	installed, err := runShellOnHost(ctx, host, args, pip+" list --format=freeze 2>/dev/null")
	if err != nil {
		return fail(fmt.Sprintf("pip list failed: %v", err))
	}
	have := map[string]string{}
	for _, l := range strings.Split(installed, "\n") {
		if n, v, ok := strings.Cut(strings.TrimSpace(l), "=="); ok {
			have[normalizePip(n)] = v
		}
	}
	var todo []string
	for _, spec := range packageNames(args) {
		if v := getStringArg(args, "version", ""); v != "" && !strings.ContainsAny(spec, "=<>~!") {
			spec += "==" + v
		}
		mt := pipSpec.FindStringSubmatch(spec)
		if mt == nil {
			return fail(fmt.Sprintf("pip: cannot read the package %q", spec))
		}
		name, op, want := normalizePip(mt[1]), mt[3], strings.TrimSpace(mt[4])
		current, ok := have[name]
		switch state {
		case "absent":
			if ok {
				todo = append(todo, mt[1])
			}
		case "latest", "forcereinstall":
			todo = append(todo, spec)
		default:
			if !ok || (op == "==" && current != want) {
				todo = append(todo, spec)
			}
		}
	}
	result.Output["packages"] = todo
	if len(todo) == 0 {
		result.Duration = time.Since(start)
		return result, nil
	}
	quoted := make([]string, len(todo))
	for i, p := range todo {
		quoted[i] = shellQuote(p)
	}
	var cmd string
	switch state {
	case "absent":
		cmd = pip + " uninstall -y " + strings.Join(quoted, " ")
	case "latest":
		cmd = pip + " install -U " + extra + " " + strings.Join(quoted, " ")
	case "forcereinstall":
		cmd = pip + " install --force-reinstall " + extra + " " + strings.Join(quoted, " ")
	default:
		cmd = pip + " install " + extra + " " + strings.Join(quoted, " ")
	}
	if inCheckMode(args) {
		result.Changed = state != "latest" || len(todo) > 0
		result.Duration = time.Since(start)
		return result, nil
	}
	if out, err := runShellOnHost(ctx, host, args, cmd); err != nil {
		return fail(fmt.Sprintf("pip failed: %v %s", err, out))
	}
	if state == "latest" {
		after, _ := runShellOnHost(ctx, host, args, pip+" list --format=freeze 2>/dev/null")
		result.Changed = result.Changed || after != installed
	} else {
		result.Changed = true
	}
	result.Duration = time.Since(start)
	return result, nil
}

// normalizePip compares names as pip does (case, - _ . alike)
func normalizePip(name string) string {
	return strings.NewReplacer("_", "-", ".", "-").Replace(strings.ToLower(name))
}
