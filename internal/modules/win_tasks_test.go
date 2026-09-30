package modules

import (
	"context"
	"strings"
	"testing"
)

// the approval task of the wsus role
func wsusTask() map[string]interface{} {
	return map[string]interface{}{
		"name": "WSUS approval", "path": `\WSUS`, "description": "Approve updates",
		"actions": []interface{}{map[string]interface{}{"path": "powershell.exe",
			"arguments": `-NoProfile -Command "& 'C:\wsus\Invoke-WsusApproval.ps1' *>&1 | Tee-Object -FilePath 'C:\logs\a-$(Get-Date -Format yyyy-MM-dd).log'"`}},
		"triggers": []interface{}{
			map[string]interface{}{"type": "daily", "start_boundary": "2026-01-01T02:00:00"},
			map[string]interface{}{"type": "weekly", "days_of_week": "sunday", "start_boundary": "2026-01-01T03:00:00"},
			map[string]interface{}{"type": "registration"},
		},
		"username": "SYSTEM", "run_level": "highest", "state": "present", "enabled": true,
	}
}

func TestWinScheduledTaskArgs(t *testing.T) {
	script, err := winScheduledTaskScript(context.TODO(), nil, wsusTask())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`$path = '\WSUS\'`, `$user = 'SYSTEM'`, `$runLevel = 'highest'`, `"type":"registration"`, `"days_of_week":"sunday"`} {
		if !strings.Contains(script, want) {
			t.Errorf("script lacks %q", want)
		}
	}
	bad := []map[string]interface{}{
		{},
		{"name": "x", "state": "running"},
		{"name": "x"},
		{"name": "x", "actions": []interface{}{map[string]interface{}{}}},
		{"name": "x", "actions": []interface{}{map[string]interface{}{"path": "a"}}, "triggers": []interface{}{map[string]interface{}{"type": "hourly"}}},
		{"name": "x", "actions": []interface{}{map[string]interface{}{"path": "a"}}, "triggers": []interface{}{map[string]interface{}{"type": "daily"}}},
		{"name": "x", "actions": []interface{}{map[string]interface{}{"path": "a"}}, "triggers": []interface{}{map[string]interface{}{"type": "weekly", "start_boundary": "2026-01-01T00:00:00"}}},
		{"name": "x", "actions": []interface{}{map[string]interface{}{"path": "a"}}, "run_level": "root"},
		{"name": "x", "actions": []interface{}{map[string]interface{}{"path": "a"}}, "logon_type": "magic"},
	}
	for _, b := range bad {
		if _, err := winScheduledTaskScript(context.TODO(), nil, b); err == nil {
			t.Errorf("%v accepted", b)
		}
	}
	if _, err := winScheduledTaskScript(context.TODO(), nil, map[string]interface{}{"name": "x", "state": "absent"}); err != nil {
		t.Errorf("absent needs no actions: %v", err)
	}
}

func TestWinChocolateyArgs(t *testing.T) {
	script, err := winChocolateyScript(context.TODO(), nil, map[string]interface{}{"name": []interface{}{"git", "7zip"}, "state": "upgrade"})
	if err != nil || !strings.Contains(script, `$names = @('git', '7zip')`) || !strings.Contains(script, `$state = 'latest'`) {
		t.Errorf("script: %v\n%s", err, script)
	}
	for _, b := range []map[string]interface{}{{}, {"name": "git", "state": "newest"}} {
		if _, err := winChocolateyScript(context.TODO(), nil, b); err == nil {
			t.Errorf("%v accepted", b)
		}
	}
}

func TestWinTaskScriptsParse(t *testing.T) {
	host := pwshHost(t)
	task, _ := winScheduledTaskScript(context.TODO(), nil, wsusTask())
	choco, _ := winChocolateyScript(context.TODO(), nil, map[string]interface{}{"name": []interface{}{"chocolatey", "git"}, "version": "2.46.0"})
	for name, script := range map[string]string{"win_scheduled_task": task, "win_chocolatey": choco} {
		res := run(t, NewWinPowerShellModule(), host, map[string]interface{}{
			"script":     "param($Text)\n$e = $null\n[void][System.Management.Automation.Language.Parser]::ParseInput($Text, [ref]$null, [ref]$e)\n$Ansible.Result = @($e | ForEach-Object { $_.Message })",
			"parameters": map[string]interface{}{"Text": script}})
		if errs, _ := res.Output["result"].([]interface{}); !res.Success || len(errs) > 0 {
			t.Errorf("%s: %+v", name, res)
		}
	}
}
