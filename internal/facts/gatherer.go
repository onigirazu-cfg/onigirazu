package facts

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/ssh"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Gatherer collects system facts from remote hosts
type Gatherer struct {
	cache *cache.FactsCache
}

// NewGatherer creates a new facts gatherer
func NewGatherer() *Gatherer {
	return &Gatherer{
		cache: cache.GetGlobalFactsCache(),
	}
}

// GatherFacts collects system facts from a host
func (g *Gatherer) GatherFacts(ctx context.Context, host types.Host) (*cache.SystemFacts, error) {
	// Check cache first
	if facts, found := g.cache.Get(host.Name); found {
		return facts, nil
	}

	// Connect to host
	client, err := ssh.NewClient(host)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to host %s: %w", host.Name, err)
	}
	defer client.Close()

	facts := &cache.SystemFacts{}

	// Gather OS information
	if err := g.gatherOSInfo(client, facts); err != nil {
		return nil, fmt.Errorf("failed to gather OS info: %w", err)
	}

	// Gather hardware information
	if err := g.gatherHardwareInfo(client, facts); err != nil {
		return nil, fmt.Errorf("failed to gather hardware info: %w", err)
	}

	// Gather network information
	if err := g.gatherNetworkInfo(client, facts); err != nil {
		return nil, fmt.Errorf("failed to gather network info: %w", err)
	}

	// Gather user and environment information
	if err := g.gatherUserInfo(client, facts); err != nil {
		return nil, fmt.Errorf("failed to gather user info: %w", err)
	}

	// Cache the facts
	g.cache.Set(host.Name, facts)

	return facts, nil
}

// gatherOSInfo collects operating system information
func (g *Gatherer) gatherOSInfo(client *ssh.Client, facts *cache.SystemFacts) error {
	// Get kernel name
	kernel, err := client.ExecuteCommand("uname -s")
	if err != nil {
		return err
	}
	facts.Kernel = strings.TrimSpace(kernel)

	// Get kernel version
	kernelVersion, err := client.ExecuteCommand("uname -r")
	if err != nil {
		return err
	}
	facts.KernelVersion = strings.TrimSpace(kernelVersion)

	// Get architecture
	arch, err := client.ExecuteCommand("uname -m")
	if err != nil {
		return err
	}
	facts.Architecture = strings.TrimSpace(arch)

	// Get hostname
	hostname, err := client.ExecuteCommand("hostname")
	if err != nil {
		return err
	}
	facts.Hostname = strings.TrimSpace(hostname)

	// Get FQDN
	fqdn, err := client.ExecuteCommand("hostname -f 2>/dev/null || hostname")
	if err == nil {
		facts.FQDN = strings.TrimSpace(fqdn)
	} else {
		facts.FQDN = facts.Hostname
	}

	// Detect distribution
	if facts.Kernel == "Linux" {
		g.detectLinuxDistribution(client, facts)
	} else if facts.Kernel == "Darwin" {
		facts.OSFamily = "Darwin"
		facts.Distribution = "MacOSX"
		g.detectMacOSVersion(client, facts)
	} else {
		facts.OSFamily = facts.Kernel
		facts.Distribution = "Unknown"
	}

	return nil
}

// detectLinuxDistribution detects Linux distribution
func (g *Gatherer) detectLinuxDistribution(client *ssh.Client, facts *cache.SystemFacts) {
	// Try /etc/os-release first (modern systems)
	osRelease, err := client.ExecuteCommand("cat /etc/os-release 2>/dev/null")
	if err == nil {
		g.parseOSRelease(osRelease, facts)
		return
	}

	// Try lsb_release
	lsbRelease, err := client.ExecuteCommand("lsb_release -a 2>/dev/null")
	if err == nil {
		g.parseLSBRelease(lsbRelease, facts)
		return
	}

	// Fallback: check for specific distribution files
	if _, err := client.ExecuteCommand("test -f /etc/debian_version"); err == nil {
		facts.OSFamily = "Debian"
		facts.Distribution = "Debian"
		if version, err := client.ExecuteCommand("cat /etc/debian_version"); err == nil {
			facts.OSVersion = strings.TrimSpace(version)
		}
		return
	}

	if _, err := client.ExecuteCommand("test -f /etc/redhat-release"); err == nil {
		facts.OSFamily = "RedHat"
		if release, err := client.ExecuteCommand("cat /etc/redhat-release"); err == nil {
			facts.Distribution = g.parseRedHatRelease(release)
		}
		return
	}

	// Default
	facts.OSFamily = "Linux"
	facts.Distribution = "Unknown"
}

// parseOSRelease parses /etc/os-release content
func (g *Gatherer) parseOSRelease(content string, facts *cache.SystemFacts) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ID=") {
			facts.Distribution = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		} else if strings.HasPrefix(line, "VERSION_ID=") {
			facts.OSVersion = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
		} else if strings.HasPrefix(line, "ID_LIKE=") {
			idLike := strings.Trim(strings.TrimPrefix(line, "ID_LIKE="), "\"")
			if strings.Contains(idLike, "debian") {
				facts.OSFamily = "Debian"
			} else if strings.Contains(idLike, "rhel") || strings.Contains(idLike, "fedora") {
				facts.OSFamily = "RedHat"
			}
		}
	}

	// Set OS family based on distribution if not set
	if facts.OSFamily == "" {
		dist := strings.ToLower(facts.Distribution)
		if strings.Contains(dist, "ubuntu") || strings.Contains(dist, "debian") {
			facts.OSFamily = "Debian"
		} else if strings.Contains(dist, "centos") || strings.Contains(dist, "rhel") ||
			strings.Contains(dist, "fedora") || strings.Contains(dist, "rocky") ||
			strings.Contains(dist, "alma") {
			facts.OSFamily = "RedHat"
		} else {
			facts.OSFamily = "Linux"
		}
	}
}

// parseLSBRelease parses lsb_release output
func (g *Gatherer) parseLSBRelease(content string, facts *cache.SystemFacts) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Distributor ID:") {
			facts.Distribution = strings.TrimSpace(strings.TrimPrefix(line, "Distributor ID:"))
		} else if strings.HasPrefix(line, "Release:") {
			facts.OSVersion = strings.TrimSpace(strings.TrimPrefix(line, "Release:"))
		}
	}

	// Set OS family
	dist := strings.ToLower(facts.Distribution)
	if strings.Contains(dist, "ubuntu") || strings.Contains(dist, "debian") {
		facts.OSFamily = "Debian"
	} else if strings.Contains(dist, "centos") || strings.Contains(dist, "redhat") || strings.Contains(dist, "fedora") {
		facts.OSFamily = "RedHat"
	} else {
		facts.OSFamily = "Linux"
	}
}

// parseRedHatRelease parses /etc/redhat-release content
func (g *Gatherer) parseRedHatRelease(content string) string {
	content = strings.TrimSpace(content)
	if strings.Contains(content, "CentOS") {
		return "CentOS"
	} else if strings.Contains(content, "Red Hat") {
		return "RedHat"
	} else if strings.Contains(content, "Fedora") {
		return "Fedora"
	} else if strings.Contains(content, "Rocky") {
		return "Rocky"
	} else if strings.Contains(content, "AlmaLinux") {
		return "AlmaLinux"
	}
	return "RedHat"
}

// detectMacOSVersion detects macOS version
func (g *Gatherer) detectMacOSVersion(client *ssh.Client, facts *cache.SystemFacts) {
	version, err := client.ExecuteCommand("sw_vers -productVersion 2>/dev/null")
	if err == nil {
		facts.OSVersion = strings.TrimSpace(version)
	}
}

// gatherHardwareInfo collects hardware information
func (g *Gatherer) gatherHardwareInfo(client *ssh.Client, facts *cache.SystemFacts) error {
	// Get CPU cores
	if facts.Kernel == "Linux" {
		cpuInfo, err := client.ExecuteCommand("nproc 2>/dev/null || grep -c ^processor /proc/cpuinfo")
		if err == nil {
			if cores, err := strconv.Atoi(strings.TrimSpace(cpuInfo)); err == nil {
				facts.CPUCores = cores
			}
		}

		// Get memory
		memInfo, err := client.ExecuteCommand("free -h | grep Mem: | awk '{print $2}'")
		if err == nil {
			facts.MemoryTotal = strings.TrimSpace(memInfo)
		}
	} else if facts.Kernel == "Darwin" {
		// macOS CPU cores
		cpuInfo, err := client.ExecuteCommand("sysctl -n hw.ncpu")
		if err == nil {
			if cores, err := strconv.Atoi(strings.TrimSpace(cpuInfo)); err == nil {
				facts.CPUCores = cores
			}
		}

		// macOS memory
		memInfo, err := client.ExecuteCommand("sysctl -n hw.memsize")
		if err == nil {
			if memBytes, err := strconv.ParseInt(strings.TrimSpace(memInfo), 10, 64); err == nil {
				facts.MemoryTotal = fmt.Sprintf("%.1fG", float64(memBytes)/(1024*1024*1024))
			}
		}
	}

	return nil
}

// gatherNetworkInfo collects network information
func (g *Gatherer) gatherNetworkInfo(client *ssh.Client, facts *cache.SystemFacts) error {
	// Get default IPv4 address
	if facts.Kernel == "Linux" {
		// Try to get the IP of the default route interface
		ipOutput, err := client.ExecuteCommand("ip route get 1.1.1.1 2>/dev/null | grep -oP 'src \\K\\S+' || hostname -I | awk '{print $1}'")
		if err == nil {
			ip := strings.TrimSpace(ipOutput)
			if ip != "" {
				facts.DefaultIPv4 = ip
			}
		}
	} else if facts.Kernel == "Darwin" {
		// macOS
		ipOutput, err := client.ExecuteCommand("route -n get default 2>/dev/null | grep 'interface:' | awk '{print $2}' | xargs ifconfig | grep 'inet ' | grep -v 127.0.0.1 | awk '{print $2}' | head -1")
		if err == nil {
			ip := strings.TrimSpace(ipOutput)
			if ip != "" {
				facts.DefaultIPv4 = ip
			}
		}
	}

	// Fallback: try to extract from SSH connection
	if facts.DefaultIPv4 == "" {
		// Try hostname -I
		ipOutput, err := client.ExecuteCommand("hostname -I 2>/dev/null | awk '{print $1}'")
		if err == nil {
			ip := strings.TrimSpace(ipOutput)
			if ip != "" && g.isValidIPv4(ip) {
				facts.DefaultIPv4 = ip
			}
		}
	}

	return nil
}

// isValidIPv4 checks if a string is a valid IPv4 address
func (g *Gatherer) isValidIPv4(ip string) bool {
	ipv4Regex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	if !ipv4Regex.MatchString(ip) {
		return false
	}

	parts := strings.Split(ip, ".")
	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil || num < 0 || num > 255 {
			return false
		}
	}

	return true
}

// gatherUserInfo collects user and environment information
func (g *Gatherer) gatherUserInfo(client *ssh.Client, facts *cache.SystemFacts) error {
	// Get current username
	username, err := client.ExecuteCommand("whoami")
	if err == nil {
		facts.Username = strings.TrimSpace(username)
	}

	// Get home directory
	homeDir, err := client.ExecuteCommand("echo $HOME")
	if err == nil {
		facts.HomeDir = strings.TrimSpace(homeDir)
	}

	// Get PATH
	path, err := client.ExecuteCommand("echo $PATH")
	if err == nil {
		facts.Path = strings.TrimSpace(path)
	}

	return nil
}

// InvalidateCache invalidates cached facts for a host
func (g *Gatherer) InvalidateCache(hostname string) {
	g.cache.Invalidate(hostname)
}

// GetCacheStats returns cache statistics
func (g *Gatherer) GetCacheStats() cache.FactsCacheStats {
	return g.cache.GetStats()
}
