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

// ============================================================================
// EXTENDED OS RELEASE PARSING TESTS
// ============================================================================

func TestParseOSReleaseEdgeCases(t *testing.T) {
	gatherer := NewGatherer()

	tests := []struct {
		name            string
		content         string
		expectedFamily  string
		expectedDistrib string
	}{
		{
			name:            "Empty content",
			content:         "",
			expectedFamily:  "",
			expectedDistrib: "",
		},
		{
			name:            "Only ID field",
			content:         `ID=custom`,
			expectedFamily:  "Linux",
			expectedDistrib: "custom",
		},
		{
			name: "Ubuntu with quotes",
			content: `ID="ubuntu"
VERSION_ID="24.04"`,
			expectedDistrib: "ubuntu",
			expectedFamily:  "Debian",
		},
		{
			name: "Fedora with rhel-like",
			content: `ID="fedora"
ID_LIKE="rhel"`,
			expectedDistrib: "fedora",
			expectedFamily:  "RedHat",
		},
		{
			name: "Alpine (unknown distro)",
			content: `ID="alpine"
VERSION_ID="3.18"`,
			expectedDistrib: "alpine",
			expectedFamily:  "Linux",
		},
		{
			name: "Missing version ID",
			content: `ID="ubuntu"
ID_LIKE="debian"`,
			expectedFamily: "Debian",
		},
		{
			name: "Whitespace and special chars",
			content: `   ID="ubuntu"
		VERSION_ID="24.04"`,
			expectedDistrib: "ubuntu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facts := &cache.SystemFacts{}
			gatherer.parseOSRelease(tt.content, facts)

			if tt.expectedDistrib != "" && facts.Distribution != tt.expectedDistrib {
				t.Errorf("Expected distribution '%s', got '%s'", tt.expectedDistrib, facts.Distribution)
			}

			if tt.expectedFamily != "" && facts.OSFamily != tt.expectedFamily {
				t.Errorf("Expected OS family '%s', got '%s'", tt.expectedFamily, facts.OSFamily)
			}
		})
	}
}

// ============================================================================
// EXTENDED LSB RELEASE PARSING TESTS
// ============================================================================

func TestParseLSBReleaseEdgeCases(t *testing.T) {
	gatherer := NewGatherer()

	tests := []struct {
		name            string
		content         string
		expectedDistrib string
		expectedFamily  string
	}{
		{
			name:            "Empty content",
			content:         "",
			expectedDistrib: "",
			expectedFamily:  "Linux",
		},
		{
			name:            "Only distribution line",
			content:         `Distributor ID:	CustomOS`,
			expectedDistrib: "CustomOS",
			expectedFamily:  "Linux",
		},
		{
			name: "With Release line",
			content: `Distributor ID:	RedHat
Release:	8.5`,
			expectedDistrib: "RedHat",
			expectedFamily:  "RedHat",
		},
		{
			name: "CentOS with various spacing",
			content: `Distributor ID:CentOS
Release:  7.9`,
			expectedDistrib: "CentOS",
			expectedFamily:  "RedHat",
		},
		{
			name: "Unknown Distro",
			content: `Distributor ID:	UnknownOS
Release:	1.0`,
			expectedFamily: "Linux",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facts := &cache.SystemFacts{}
			gatherer.parseLSBRelease(tt.content, facts)

			if tt.expectedDistrib != "" && facts.Distribution != tt.expectedDistrib {
				t.Errorf("Expected distribution '%s', got '%s'", tt.expectedDistrib, facts.Distribution)
			}

			if facts.OSFamily != tt.expectedFamily {
				t.Errorf("Expected OS family '%s', got '%s'", tt.expectedFamily, facts.OSFamily)
			}
		})
	}
}

// ============================================================================
// EXTENDED RED HAT RELEASE PARSING TESTS
// ============================================================================

func TestParseRedHatReleaseEdgeCases(t *testing.T) {
	gatherer := NewGatherer()

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "Empty string",
			content:  "",
			expected: "RedHat",
		},
		{
			name:     "Only spaces",
			content:  "   ",
			expected: "RedHat",
		},
		{
			name:     "Whitespace with content",
			content:  "  CentOS Linux release 8.5  ",
			expected: "CentOS",
		},
		{
			name:     "Mixed case",
			content:  "rocky linux release 8.5",
			expected: "RedHat", // Should not match Rocky due to case
		},
		{
			name:     "Partial match - should not match",
			content:  "Linux server release",
			expected: "RedHat",
		},
		{
			name:     "Multiple keywords - CentOS checked first",
			content:  "Red Hat CentOS Fedora release",
			expected: "CentOS", // CentOS is checked before RedHat in the code
		},
		{
			name:     "AlmaLinux exact",
			content:  "AlmaLinux release 9.0",
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

// ============================================================================
// IPv4 VALIDATION - EXTENDED EDGE CASES
// ============================================================================

func TestIsValidIPv4ExtendedEdgeCases(t *testing.T) {
	gatherer := NewGatherer()

	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		// Boundary values
		{name: "All zeros", ip: "0.0.0.0", expected: true},
		{name: "All max", ip: "255.255.255.255", expected: true},
		{name: "Localhost", ip: "127.0.0.1", expected: true},
		{name: "Common private", ip: "192.168.1.1", expected: true},
		{name: "Another private", ip: "10.0.0.1", expected: true},
		{name: "Class C private", ip: "172.16.0.1", expected: true},

		// Invalid formats
		{name: "Extra octet", ip: "192.168.1.1.1", expected: false},
		{name: "Missing octet", ip: "192.168.1", expected: false},
		{name: "Single number", ip: "192", expected: false},
		{name: "Too many dots", ip: "192.168...1", expected: false},

		// Out of range
		{name: "First octet 256", ip: "256.168.1.1", expected: false},
		{name: "Last octet 256", ip: "192.168.1.256", expected: false},
		{name: "Negative first", ip: "-1.168.1.1", expected: false},
		{name: "Negative last", ip: "192.168.1.-1", expected: false},

		// Non-numeric
		{name: "Letters", ip: "192.168.a.1", expected: false},
		{name: "Special chars", ip: "192.168.1!1", expected: false},
		{name: "Hexadecimal", ip: "0xC0.0xA8.0x01.0x01", expected: false},

		// Whitespace
		{name: "Leading space", ip: " 192.168.1.1", expected: false},
		{name: "Trailing space", ip: "192.168.1.1 ", expected: false},
		{name: "Middle space", ip: "192 .168.1.1", expected: false},
		{name: "Empty", ip: "", expected: false},

		// Special formats that shouldn't match
		{name: "CIDR notation", ip: "192.168.1.0/24", expected: false},
		{name: "IPv6-like", ip: "::1", expected: false},
		{name: "Domain name", ip: "example.com", expected: false},
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

// ============================================================================
// CACHE OPERATIONS TESTS
// ============================================================================

func TestCacheMultipleHostsIndependence(t *testing.T) {
	gatherer := NewGatherer()
	gatherer.cache.Clear()

	// Add facts for multiple hosts
	facts1 := &cache.SystemFacts{
		Hostname: "host1",
		OSFamily: "Debian",
	}
	facts2 := &cache.SystemFacts{
		Hostname: "host2",
		OSFamily: "RedHat",
	}

	gatherer.cache.Set("host1", facts1)
	gatherer.cache.Set("host2", facts2)

	// Verify both are independent
	retrieved1, found1 := gatherer.cache.Get("host1")
	retrieved2, found2 := gatherer.cache.Get("host2")

	if !found1 || !found2 {
		t.Error("Expected both hosts to be cached")
	}

	if retrieved1.OSFamily != "Debian" {
		t.Errorf("Expected host1 to have Debian family, got %s", retrieved1.OSFamily)
	}

	if retrieved2.OSFamily != "RedHat" {
		t.Errorf("Expected host2 to have RedHat family, got %s", retrieved2.OSFamily)
	}
}

func TestCacheInvalidateSpecificHost(t *testing.T) {
	gatherer := NewGatherer()
	gatherer.cache.Clear()

	facts1 := &cache.SystemFacts{Hostname: "host1"}
	facts2 := &cache.SystemFacts{Hostname: "host2"}

	gatherer.cache.Set("host1", facts1)
	gatherer.cache.Set("host2", facts2)

	// Invalidate only host1
	gatherer.InvalidateCache("host1")

	_, found1 := gatherer.cache.Get("host1")
	_, found2 := gatherer.cache.Get("host2")

	if found1 {
		t.Error("Expected host1 to be invalidated")
	}

	if !found2 {
		t.Error("Expected host2 to still be cached")
	}
}

func TestCacheStatsAfterMultipleOperations(t *testing.T) {
	gatherer := NewGatherer()
	gatherer.cache.Clear()

	facts := &cache.SystemFacts{Hostname: "test"}
	gatherer.cache.Set("test", facts)

	// Perform operations
	for i := 0; i < 5; i++ {
		gatherer.cache.Get("test") // 5 hits
	}

	for i := 0; i < 3; i++ {
		gatherer.cache.Get("non-existent") // 3 misses
	}

	stats := gatherer.GetCacheStats()

	if stats.Hits != 5 {
		t.Errorf("Expected 5 hits, got %d", stats.Hits)
	}

	if stats.Misses != 3 {
		t.Errorf("Expected 3 misses, got %d", stats.Misses)
	}

	if stats.Entries != 1 {
		t.Errorf("Expected 1 entry, got %d", stats.Entries)
	}
}
