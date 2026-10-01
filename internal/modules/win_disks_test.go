package modules

import (
	"context"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/winrm"
)

func TestPartitionBytes(t *testing.T) {
	for in, want := range map[string]int64{"-1": -1, "1024": 1024, "10 GiB": 10 << 30, "1.5GB": 1500000000, "512 MiB": 512 << 20} {
		if got, err := partitionBytes(in); err != nil || got != want {
			t.Errorf("partitionBytes(%q) = %d, %v", in, got, err)
		}
	}
	if _, err := partitionBytes("big"); err == nil {
		t.Error("big accepted")
	}
}

func TestWinDiskArgs(t *testing.T) {
	type builder func(context.Context, winrm.Runner, map[string]interface{}) (string, error)
	bad := []struct {
		b    builder
		args map[string]interface{}
	}{
		{winOptionalFeatureScript, map[string]interface{}{}},
		{winOptionalFeatureScript, map[string]interface{}{"name": "SMB1Protocol", "state": "removed"}},
		{winInitializeDiskScript, map[string]interface{}{}},
		{winInitializeDiskScript, map[string]interface{}{"disk_number": "one"}},
		{winInitializeDiskScript, map[string]interface{}{"disk_number": 1, "style": "apm"}},
		{winPartitionScript, map[string]interface{}{}},
		{winPartitionScript, map[string]interface{}{"drive_letter": "DE"}},
		{winPartitionScript, map[string]interface{}{"disk_number": 1, "state": "absent"}},
		{winPartitionScript, map[string]interface{}{"drive_letter": "D", "partition_size": "lots"}},
		{winFormatScript, map[string]interface{}{}},
		{winFormatScript, map[string]interface{}{"drive_letter": "D", "file_system": "ext4"}},
		{winFormatScript, map[string]interface{}{"drive_letter": "D", "allocation_unit_size": "4k"}},
	}
	for _, c := range bad {
		if _, err := c.b(context.TODO(), nil, c.args); err == nil {
			t.Errorf("%v accepted", c.args)
		}
	}
	s, err := winPartitionScript(context.TODO(), nil, map[string]interface{}{"disk_number": 1, "drive_letter": "d:", "partition_size": -1})
	if err != nil || !strings.Contains(s, "$letter = 'D'") || !strings.Contains(s, "$size = [long]-1") || !strings.Contains(s, "$disk = '1'") {
		t.Errorf("partition script: %v", err)
	}
}

// the wsus_disk role's scripts parse
func TestWinDiskScriptsParse(t *testing.T) {
	host := pwshHost(t)
	type builder func(context.Context, winrm.Runner, map[string]interface{}) (string, error)
	cases := map[string]struct {
		b    builder
		args map[string]interface{}
	}{
		"optional": {winOptionalFeatureScript, map[string]interface{}{"name": "SMB1Protocol", "state": "absent"}},
		"facts":    {winDiskFactsScript, map[string]interface{}{}},
		"init":     {winInitializeDiskScript, map[string]interface{}{"disk_number": 1, "style": "gpt"}},
		"part":     {winPartitionScript, map[string]interface{}{"disk_number": 1, "partition_size": -1, "drive_letter": "D"}},
		"format":   {winFormatScript, map[string]interface{}{"drive_letter": "D", "file_system": "ntfs", "new_label": "WSUS"}},
	}
	for name, c := range cases {
		script, err := c.b(context.TODO(), nil, c.args)
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
