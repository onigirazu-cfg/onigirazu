package facts

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/winrm"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// windowsFactsScript prints the facts of a Windows host as JSON
const windowsFactsScript = `$os = Get-CimInstance Win32_OperatingSystem
$cs = Get-CimInstance Win32_ComputerSystem
$ip = (Get-NetIPConfiguration -ErrorAction SilentlyContinue | Where-Object { $_.IPv4DefaultGateway } | Select-Object -First 1).IPv4Address.IPAddress
[pscustomobject]@{
  hostname = $env:COMPUTERNAME
  domain = $cs.Domain
  caption = $os.Caption
  version = $os.Version
  arch = $os.OSArchitecture
  cores = [int]$cs.NumberOfLogicalProcessors
  mem_mb = [int]($cs.TotalPhysicalMemory / 1MB)
  manufacturer = $cs.Manufacturer
  model = $cs.Model
  ipv4 = "$ip"
  user = $env:USERNAME
  home = $env:USERPROFILE
  path = $env:Path
} | ConvertTo-Json -Compress`

type windowsFacts struct {
	Hostname     string `json:"hostname"`
	Domain       string `json:"domain"`
	Caption      string `json:"caption"`
	Version      string `json:"version"`
	Arch         string `json:"arch"`
	Cores        int    `json:"cores"`
	MemMB        int    `json:"mem_mb"`
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	IPv4         string `json:"ipv4"`
	User         string `json:"user"`
	Home         string `json:"home"`
	Path         string `json:"path"`
}

// gatherWindows reads the facts of a WinRM host with one PowerShell run
func gatherWindows(ctx context.Context, host types.Host) (*cache.SystemFacts, error) {
	c, err := winrm.For(host)
	if err != nil {
		return nil, err
	}
	res, err := c.RunPS(ctx, windowsFactsScript)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to host %s: %w", host.Name, err)
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("facts of %s: %s", host.Name, strings.TrimSpace(res.Stderr))
	}
	return parseWindowsFacts(res.Stdout)
}

func parseWindowsFacts(out string) (*cache.SystemFacts, error) {
	var w windowsFacts
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &w); err != nil {
		return nil, fmt.Errorf("reading Windows facts: %w", err)
	}
	f := &cache.SystemFacts{
		OSFamily: "Windows", Distribution: strings.TrimSpace(w.Caption), OSVersion: w.Version,
		Architecture: w.Arch, Kernel: "Win32NT", KernelVersion: w.Version,
		Hostname: w.Hostname, CPUCores: w.Cores, MemTotalMB: w.MemMB,
		DefaultIPv4: w.IPv4, Username: w.User, HomeDir: w.Home, Path: w.Path,
		VirtualizationType: "NA", VirtualizationRole: "NA",
	}
	f.MemoryTotal = fmt.Sprintf("%d MB", w.MemMB)
	f.FQDN = w.Hostname
	if w.Domain != "" && !strings.EqualFold(w.Domain, "WORKGROUP") {
		f.FQDN = strings.ToLower(w.Hostname) + "." + w.Domain
	}
	switch m := strings.ToLower(w.Manufacturer + " " + w.Model); {
	case strings.Contains(m, "vmware"):
		f.VirtualizationType, f.VirtualizationRole = "VMware", "guest"
	case strings.Contains(m, "qemu") || strings.Contains(m, "kvm"):
		f.VirtualizationType, f.VirtualizationRole = "kvm", "guest"
	case strings.Contains(m, "virtual machine") && strings.Contains(m, "microsoft"):
		f.VirtualizationType, f.VirtualizationRole = "Hyper-V", "guest"
	}
	return f, nil
}
