package facts

import (
	"context"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/winrm/winrmtest"
)

func TestWindowsFacts(t *testing.T) {
	f := &winrmtest.Server{Handle: func(command, stdin string) (string, string, int) {
		if !strings.Contains(winrmtest.Script(command, stdin), "Win32_OperatingSystem") {
			return "", "unexpected", 1
		}
		return `{"hostname":"WIN1","domain":"office.red","caption":"Microsoft Windows Server 2025 Standard ","version":"10.0.26100","arch":"64-bit","cores":4,"mem_mb":8191,"manufacturer":"VMware, Inc.","model":"VMware20,1","ipv4":"192.168.99.23","user":"admin","home":"C:\\Users\\admin","path":"C:\\Windows"}`, "", 0
	}}
	addr, port := winrmtest.Start(t, f)
	g := NewGatherer()
	sf, err := g.Regather(context.Background(), winrmtest.Host("win1", addr, port), "")
	if err != nil {
		t.Fatal(err)
	}
	if sf.OSFamily != "Windows" || sf.Distribution != "Microsoft Windows Server 2025 Standard" || sf.FQDN != "win1.office.red" ||
		sf.VirtualizationType != "VMware" || sf.MemTotalMB != 8191 || sf.DefaultIPv4 != "192.168.99.23" {
		t.Errorf("facts = %+v", sf)
	}
	if _, err := parseWindowsFacts("not json"); err == nil {
		t.Error("bad output is an error")
	}
}
