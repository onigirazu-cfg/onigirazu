package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// UfwModule manages the Uncomplicated Firewall (community.general.ufw):
// state enabled/disabled/reloaded/reset, default policy with direction,
// logging, and rules (rule allow/deny/limit/reject with port, proto,
// src/from_ip, dest/to_ip, from_port, interface, direction, route, delete,
// insert, comment)
type UfwModule struct {
	*BaseModule
}

// NewUfwModule creates the ufw module
func NewUfwModule() *UfwModule {
	return &UfwModule{BaseModule: NewBaseModule("ufw")}
}

func (m *UfwModule) GetDescription() string { return "Manage ufw firewall rules and state" }

func (m *UfwModule) Validate(args map[string]interface{}) error {
	if getStringArg(args, "state", "") == "" && getStringArg(args, "rule", "") == "" &&
		getStringArg(args, "policy", getStringArg(args, "default", "")) == "" && getStringArg(args, "logging", "") == "" {
		return fmt.Errorf("ufw requires one of state, rule, policy or logging")
	}
	if r := getStringArg(args, "rule", ""); r != "" && r != "allow" && r != "deny" && r != "limit" && r != "reject" {
		return fmt.Errorf("ufw: rule must be allow, deny, limit or reject")
	}
	return nil
}

func (m *UfwModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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
	check := inCheckMode(args)
	var commands []string

	// the status only for policy, logging and state: a rule alone is one
	// ufw call
	status := ""
	if getStringArg(args, "policy", getStringArg(args, "default", "")) != "" || getStringArg(args, "logging", "") != "" || getStringArg(args, "state", "") != "" {
		out, err := runShellOnHost(ctx, host, args, "ufw status verbose")
		if err != nil {
			return fail(fmt.Sprintf("ufw status failed: %v", err))
		}
		status = out
	}
	active := strings.Contains(status, "Status: active")

	if policy := getStringArg(args, "policy", getStringArg(args, "default", "")); policy != "" {
		direction := getStringArg(args, "direction", "incoming")
		if direction == "in" {
			direction = "incoming"
		} else if direction == "out" {
			direction = "outgoing"
		}
		if !strings.Contains(status, policy+" ("+direction+")") {
			commands = append(commands, "ufw default "+shellQuote(policy)+" "+shellQuote(direction))
		}
	}
	if logging := getStringArg(args, "logging", ""); logging != "" {
		want := map[string]string{"on": "on", "off": "off", "true": "on", "false": "off"}[logging]
		if want == "" {
			want = "on (" + logging + ")"
		}
		if !strings.Contains(status, "Logging: "+want) {
			commands = append(commands, "ufw logging "+shellQuote(logging))
		}
	}

	ruleChanged := false
	if rule := getStringArg(args, "rule", ""); rule != "" {
		cmd := ufwRuleCommand(rule, args)
		if check {
			out, err := runShellOnHost(ctx, host, args, "ufw --dry-run "+cmd)
			if err != nil {
				return fail(fmt.Sprintf("ufw %s: %v", cmd, err))
			}
			ruleChanged = !strings.Contains(out, "Skipping")
		} else {
			out, err := runShellOnHost(ctx, host, args, "ufw "+cmd)
			if err != nil {
				return fail(fmt.Sprintf("ufw %s: %v %s", cmd, err, out))
			}
			ruleChanged = ufwRuleChanged(out)
		}
		result.Output["rule"] = "ufw " + cmd
	}

	switch getStringArg(args, "state", "") {
	case "enabled":
		if !active {
			commands = append(commands, "ufw --force enable")
		}
	case "disabled":
		if active {
			commands = append(commands, "ufw disable")
		}
	case "reloaded":
		commands = append(commands, "ufw reload")
	case "reset":
		commands = append(commands, "ufw --force reset")
	}

	result.Changed = ruleChanged || len(commands) > 0
	result.Output["commands"] = commands
	if check || len(commands) == 0 {
		result.Duration = time.Since(start)
		return result, nil
	}
	for _, c := range commands {
		if out, err := runShellOnHost(ctx, host, args, c); err != nil {
			return fail(fmt.Sprintf("%s: %v %s", c, err, out))
		}
	}
	result.Duration = time.Since(start)
	return result, nil
}

// ufwRuleChanged reads what ufw said about a rule: "Rule added", "Rule
// updated", "Rule deleted", "Rule inserted", "Rules updated" are a change;
// "Skipping adding existing rule", "Could not delete non-existent rule" and
// warnings are not
func ufwRuleChanged(out string) bool {
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Rule") {
			return true
		}
	}
	return false
}

// ufwRuleCommand is the ufw rule in its full syntax
func ufwRuleCommand(rule string, args map[string]interface{}) string {
	var parts []string
	if getBoolArg(args, "route", false) {
		parts = append(parts, "route")
	}
	if getBoolArg(args, "delete", false) {
		parts = append(parts, "delete")
	}
	if n := getStringArg(args, "insert", ""); n != "" {
		parts = append(parts, "insert", shellQuote(n))
	}
	parts = append(parts, rule)
	switch getStringArg(args, "direction", "") {
	case "in", "incoming":
		parts = append(parts, "in")
	case "out", "outgoing":
		parts = append(parts, "out")
	}
	if iface := getStringArg(args, "interface", getStringArg(args, "if", "")); iface != "" {
		parts = append(parts, "on", shellQuote(iface))
	}
	if getBoolArg(args, "log", false) {
		parts = append(parts, "log")
	}
	if p := getStringArg(args, "proto", getStringArg(args, "protocol", "")); p != "" && p != "any" {
		parts = append(parts, "proto", shellQuote(p))
	}
	src := getStringArg(args, "src", getStringArg(args, "from_ip", getStringArg(args, "from", "any")))
	parts = append(parts, "from", shellQuote(src))
	if fp := getStringArg(args, "from_port", ""); fp != "" {
		parts = append(parts, "port", shellQuote(fp))
	}
	dst := getStringArg(args, "dest", getStringArg(args, "to_ip", getStringArg(args, "to", "any")))
	parts = append(parts, "to", shellQuote(dst))
	if p := getStringArg(args, "port", getStringArg(args, "to_port", "")); p != "" {
		parts = append(parts, "port", shellQuote(p))
	}
	if c := getStringArg(args, "comment", ""); c != "" {
		parts = append(parts, "comment", shellQuote(c))
	}
	return strings.Join(parts, " ")
}
