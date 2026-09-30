package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/winrm/winrmtest"
)

// a playbook against a Windows host over WinRM: facts, win_ping, win_shell
func TestApplyOnWindowsHost(t *testing.T) {
	f := &winrmtest.Server{Handle: func(command, stdin string) (string, string, int) {
		script := winrmtest.Script(command, stdin)
		switch {
		case strings.Contains(script, "Win32_OperatingSystem"):
			return `{"hostname":"WIN1","caption":"Microsoft Windows Server 2025 Standard","version":"10.0.26100","arch":"64-bit","cores":2,"mem_mb":4096}`, "", 0
		case strings.HasPrefix(script, "Write-Output"):
			return "pong\r\n", "", 0
		case strings.Contains(script, "Get-Item"):
			return "C:\\Windows\r\n", "", 0
		case strings.Contains(script, "Get-Disk"):
			return "@@ONIGIRAZU-RESULT@@\n" + `{"changed":false,"ansible_facts":{"ansible_disks":[{"number":1,"partition_style":"RAW"}]}}` + "\n@@ONIGIRAZU-RESULT@@\n", "", 0
		}
		return "", "unexpected: " + script + command, 1
	}}
	addr, port := winrmtest.Start(t, f)
	dir := t.TempDir()
	inv := fmt.Sprintf(`all:
  hosts:
    win1:
      ansible_host: %s
      ansible_port: %d
      ansible_user: admin
      ansible_password: secret
      ansible_connection: winrm
      ansible_winrm_transport: basic
      ansible_winrm_scheme: http
      ansible_winrm_message_encryption: never
`, addr, port)
	pb := `- hosts: all
  tasks:
    - ansible.windows.win_ping:
    - win_shell: Get-Item C:\Windows | Select-Object -Expand FullName
      register: w
    - assert:
        that:
          - ansible_os_family == "Windows"
          - ansible_distribution == "Microsoft Windows Server 2025 Standard"
          - w.stdout_lines[0] == "C:\\Windows"
    - community.windows.win_disk_facts:
    - assert:
        that: ["ansible_disks[0].partition_style == 'RAW'"]
    - copy: {dest: C:\x.txt, content: x}
      ignore_errors: true
      register: c
    - assert:
        that: ["c is failed", "'win_' in c.msg"]
`
	for name, body := range map[string]string{"inv.yml": inv, "pb.yml": pb} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)
	t.Setenv("HOME", dir)
	root := NewRootCommand()
	root.SetArgs([]string{"apply", "pb.yml", "-i", "inv.yml", "--no-color"})
	if err := root.Execute(); err != nil {
		t.Fatalf("apply: %v (commands: %d)", err, len(f.Commands))
	}
	// facts, win_ping, win_shell, win_disk_facts
	if len(f.Commands) != 4 {
		t.Errorf("%d commands ran on the host", len(f.Commands))
	}
}
