package facts

import (
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
)

func TestNewGatherer(t *testing.T) {
	gatherer := NewGatherer()

	if gatherer == nil {
		t.Fatal("Expected gatherer to be created")
	}

	if gatherer.cache == nil {
		t.Fatal("Expected gatherer to have cache")
	}
}

func TestParseOSRelease(t *testing.T) {
	gatherer := NewGatherer()

	tests := []struct {
		name     string
		content  string
		expected cache.SystemFacts
	}{
		{
			name: "Ubuntu",
			content: `NAME="Ubuntu"
VERSION="24.04 LTS (Noble Numbat)"
ID=ubuntu
ID_LIKE=debian
VERSION_ID="24.04"`,
			expected: cache.SystemFacts{
				Distribution: "ubuntu",
				OSVersion:    "24.04",
				OSFamily:     "Debian",
			},
		},
		{
			name: "CentOS",
			content: `NAME="CentOS Linux"
VERSION="8"
ID="centos"
ID_LIKE="rhel fedora"
VERSION_ID="8"`,
			expected: cache.SystemFacts{
				Distribution: "centos",
				OSVersion:    "8",
				OSFamily:     "RedHat",
			},
		},
		{
			name: "Debian",
			content: `NAME="Debian GNU/Linux"
VERSION="12 (bookworm)"
ID=debian
VERSION_ID="12"`,
			expected: cache.SystemFacts{
				Distribution: "debian",
				OSVersion:    "12",
				OSFamily:     "Debian",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facts := &cache.SystemFacts{}
			gatherer.parseOSRelease(tt.content, facts)

			if facts.Distribution != tt.expected.Distribution {
				t.Errorf("Expected distribution '%s', got '%s'", tt.expected.Distribution, facts.Distribution)
			}

			if facts.OSVersion != tt.expected.OSVersion {
				t.Errorf("Expected OS version '%s', got '%s'", tt.expected.OSVersion, facts.OSVersion)
			}

			if facts.OSFamily != tt.expected.OSFamily {
				t.Errorf("Expected OS family '%s', got '%s'", tt.expected.OSFamily, facts.OSFamily)
			}
		})
	}
}

func TestParseLSBRelease(t *testing.T) {
	gatherer := NewGatherer()

	tests := []struct {
		name     string
		content  string
		expected cache.SystemFacts
	}{
		{
			name: "Ubuntu",
			content: `Distributor ID:	Ubuntu
Description:	Ubuntu 24.04 LTS
Release:	24.04
Codename:	noble`,
			expected: cache.SystemFacts{
				Distribution: "Ubuntu",
				OSVersion:    "24.04",
				OSFamily:     "Debian",
			},
		},
		{
			name: "Debian",
			content: `Distributor ID:	Debian
Description:	Debian GNU/Linux 12 (bookworm)
Release:	12
Codename:	bookworm`,
			expected: cache.SystemFacts{
				Distribution: "Debian",
				OSVersion:    "12",
				OSFamily:     "Debian",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facts := &cache.SystemFacts{}
			gatherer.parseLSBRelease(tt.content, facts)

			if facts.Distribution != tt.expected.Distribution {
				t.Errorf("Expected distribution '%s', got '%s'", tt.expected.Distribution, facts.Distribution)
			}

			if facts.OSVersion != tt.expected.OSVersion {
				t.Errorf("Expected OS version '%s', got '%s'", tt.expected.OSVersion, facts.OSVersion)
			}

			if facts.OSFamily != tt.expected.OSFamily {
				t.Errorf("Expected OS family '%s', got '%s'", tt.expected.OSFamily, facts.OSFamily)
			}
		})
	}
}

func TestParseRedHatRelease(t *testing.T) {
	gatherer := NewGatherer()

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "CentOS",
			content:  "CentOS Linux release 8.5.2111",
			expected: "CentOS",
		},
		{
			name:     "Red Hat",
			content:  "Red Hat Enterprise Linux release 8.5 (Ootpa)",
			expected: "RedHat",
		},
		{
			name:     "Fedora",
			content:  "Fedora release 35 (Thirty Five)",
			expected: "Fedora",
		},
		{
			name:     "Rocky",
			content:  "Rocky Linux release 8.5 (Green Obsidian)",
			expected: "Rocky",
		},
		{
			name:     "AlmaLinux",
			content:  "AlmaLinux release 8.5 (Arctic Sphynx)",
			expected: "AlmaLinux",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gatherer.parseRedHatRelease(tt.content)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestIsValidIPv4(t *testing.T) {
	gatherer := NewGatherer()

	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{
			name:     "Valid IP",
			ip:       "192.168.1.100",
			expected: true,
		},
		{
			name:     "Valid IP - localhost",
			ip:       "127.0.0.1",
			expected: true,
		},
		{
			name:     "Valid IP - zeros",
			ip:       "0.0.0.0",
			expected: true,
		},
		{
			name:     "Valid IP - max",
			ip:       "255.255.255.255",
			expected: true,
		},
		{
			name:     "Invalid IP - too many octets",
			ip:       "192.168.1.1.1",
			expected: false,
		},
		{
			name:     "Invalid IP - too few octets",
			ip:       "192.168.1",
			expected: false,
		},
		{
			name:     "Invalid IP - out of range",
			ip:       "192.168.1.256",
			expected: false,
		},
		{
			name:     "Invalid IP - negative",
			ip:       "192.168.1.-1",
			expected: false,
		},
		{
			name:     "Invalid IP - letters",
			ip:       "192.168.1.abc",
			expected: false,
		},
		{
			name:     "Invalid IP - empty",
			ip:       "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gatherer.isValidIPv4(tt.ip)

			if result != tt.expected {
				t.Errorf("Expected %v for IP '%s', got %v", tt.expected, tt.ip, result)
			}
		})
	}
}

func TestInvalidateCache(t *testing.T) {
	gatherer := NewGatherer()

	// Add some facts to cache
	facts := &cache.SystemFacts{
		OSFamily:     "Debian",
		Distribution: "Ubuntu",
		Hostname:     "test-host",
	}

	gatherer.cache.Set("test-host", facts)

	// Verify it's cached
	_, found := gatherer.cache.Get("test-host")
	if !found {
		t.Error("Expected facts to be cached")
	}

	// Invalidate
	gatherer.InvalidateCache("test-host")

	// Verify it's gone
	_, found = gatherer.cache.Get("test-host")
	if found {
		t.Error("Expected facts to be invalidated")
	}
}

func TestGetCacheStats(t *testing.T) {
	gatherer := NewGatherer()

	// Clear cache to start fresh
	gatherer.cache.Clear()

	// Add some facts
	facts := &cache.SystemFacts{
		OSFamily: "Debian",
		Hostname: "test-host",
	}

	gatherer.cache.Set("test-host", facts)

	// Generate some hits and misses
	gatherer.cache.Get("test-host")    // hit
	gatherer.cache.Get("non-existent") // miss

	stats := gatherer.GetCacheStats()

	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}

	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	if stats.Entries != 1 {
		t.Errorf("Expected 1 entry, got %d", stats.Entries)
	}
}
