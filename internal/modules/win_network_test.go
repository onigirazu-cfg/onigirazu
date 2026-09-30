package modules

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/winrm/winrmtest"
)

func TestWinRebootWaitsForANewBootTime(t *testing.T) {
	var mu sync.Mutex
	polls, rebooted := 0, false
	addr, port := winrmtest.Start(t, &winrmtest.Server{Handle: func(command, stdin string) (string, string, int) {
		mu.Lock()
		defer mu.Unlock()
		script := winrmtest.Script(command, stdin)
		switch {
		case strings.HasPrefix(script, "shutdown.exe /r /t 0"):
			rebooted = true
			return "", "", 0
		case strings.Contains(script, "LastBootUpTime"):
			if !rebooted {
				return "2026-09-30T08:00:00Z\n", "", 0
			}
			polls++
			if polls < 3 {
				return "", "going down", 1
			}
			return "2026-09-30T09:00:00Z\n", "", 0
		case script == "whoami":
			return "w1\\admin", "", 0
		}
		return "", "unexpected " + script, 1
	}})
	m := NewWinRebootModule()
	m.poll = time.Millisecond
	res := run(t, m, winrmtest.Host("w1", addr, port), map[string]interface{}{"pre_reboot_delay": 0, "reboot_timeout": 5})
	if !res.Success || !res.Changed || res.Output["rebooted"] != true || polls != 3 {
		t.Errorf("win_reboot = %+v (polls %d)", res, polls)
	}
	res = run(t, m, winrmtest.Host("w1", addr, port), map[string]interface{}{"_check_mode": true})
	if !res.Changed || res.Output["rebooted"] != false {
		t.Errorf("check mode = %+v", res)
	}
}

func TestWinRebootTimesOut(t *testing.T) {
	addr, port := winrmtest.Start(t, &winrmtest.Server{Handle: func(command, stdin string) (string, string, int) {
		if strings.Contains(winrmtest.Script(command, stdin), "LastBootUpTime") {
			return "2026-09-30T08:00:00Z\n", "", 0 // never reboots
		}
		return "", "", 0
	}})
	m := NewWinRebootModule()
	m.poll = time.Millisecond
	res := run(t, m, winrmtest.Host("w1", addr, port), map[string]interface{}{"pre_reboot_delay": 0, "reboot_timeout": 0})
	if !res.Failed || !strings.Contains(res.Error, "did not come back") {
		t.Errorf("timeout = %+v", res)
	}
}

func TestWinNetworkArgs(t *testing.T) {
	host := cannedHost(t, "")
	bad := []struct {
		module string
		args   map[string]interface{}
	}{
		{"win_firewall_rule", map[string]interface{}{}},
		{"win_firewall_rule", map[string]interface{}{"name": "x", "action": "maybe"}},
		{"win_firewall_rule", map[string]interface{}{"name": "x", "direction": "up"}},
		{"win_firewall_rule", map[string]interface{}{"name": "x", "state": "gone"}},
		{"win_firewall", map[string]interface{}{"state": "on"}},
		{"win_firewall", map[string]interface{}{"inbound_action": "drop"}},
		{"win_group_membership", map[string]interface{}{"members": []interface{}{"a"}}},
		{"win_group_membership", map[string]interface{}{"name": "Administrators"}},
		{"win_group_membership", map[string]interface{}{"name": "A", "members": []interface{}{"a"}, "state": "some"}},
		{"win_feature", map[string]interface{}{}},
		{"win_feature", map[string]interface{}{"name": "X", "state": "latest"}},
	}
	for _, b := range bad {
		m, _ := NewRegistry().GetModule(b.module)
		if res := run(t, m, host, b.args); !res.Failed {
			t.Errorf("%s %v accepted", b.module, b.args)
		}
	}
	if got := psList("80, 443,"); got != "@('80', '443')" {
		t.Errorf("psList = %s", got)
	}
}

func TestWinNetworkScriptsParse(t *testing.T) {
	host := pwshHost(t)
	scripts := map[string]map[string]interface{}{
		"win_firewall_rule": {"name": "WSUS (8530)", "localport": "8530,8531", "remoteip": "any", "action": "allow",
			"direction": "in", "protocol": "tcp", "profiles": []interface{}{"domain", "private"}, "enabled": true},
		"win_firewall":         {"state": "enabled", "profiles": []interface{}{"Domain", "Private", "Public"}},
		"win_group_membership": {"name": "Remote Desktop Users", "members": []interface{}{`OFFICE\bob`}},
		"win_feature":          {"name": "UpdateServices", "include_management_tools": true},
	}
	builders := map[string]func(context.Context, map[string]interface{}) (string, error){
		"win_firewall_rule": func(ctx context.Context, a map[string]interface{}) (string, error) {
			return winFirewallRuleScript(ctx, nil, a)
		},
		"win_firewall": func(ctx context.Context, a map[string]interface{}) (string, error) {
			return winFirewallScript(ctx, nil, a)
		},
		"win_group_membership": func(ctx context.Context, a map[string]interface{}) (string, error) {
			return winGroupMembershipScript(ctx, nil, a)
		},
		"win_feature": func(ctx context.Context, a map[string]interface{}) (string, error) {
			return winFeatureScript(ctx, nil, a)
		},
	}
	for name, args := range scripts {
		script, err := builders[name](context.TODO(), args)
		if err != nil {
			t.Fatal(err)
		}
		res := run(t, NewWinPowerShellModule(), host, map[string]interface{}{
			"script":     "param($Text)\n$e = $null\n[void][System.Management.Automation.Language.Parser]::ParseInput($Text, [ref]$null, [ref]$e)\n$Ansible.Result = @($e | ForEach-Object { $_.Message })",
			"parameters": map[string]interface{}{"Text": script}})
		if errs, _ := res.Output["result"].([]interface{}); !res.Success || len(errs) > 0 {
			t.Errorf("%s: %+v", name, res)
		}
	}
}
