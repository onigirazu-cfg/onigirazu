package modules

import (
	"reflect"
	"testing"
)

func TestParseAptAvailableVersion(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name: "valid output with candidate",
			output: `nginx:
  Installed: 1.18.0-0ubuntu1
  Candidate: 1.18.0-0ubuntu1.4
  Version table:`,
			expected: "1.18.0-0ubuntu1.4",
		},
		{
			name: "candidate on separate line",
			output: `Package: nginx
Installed: (none)
Candidate: 1.20.1-1
Version table:`,
			expected: "1.20.1-1",
		},
		{
			name: "no candidate line",
			output: `nginx:
  Installed: 1.18.0-0ubuntu1
  Version table:`,
			expected: "",
		},
		{
			name:     "empty output",
			output:   "",
			expected: "",
		},
		{
			name: "candidate with no version",
			output: `nginx:
  Candidate:
  Version table:`,
			expected: "",
		},
		{
			name: "multiple packages",
			output: `nginx:
  Installed: 1.18.0
  Candidate: 1.20.0
apache2:
  Installed: 2.4.41
  Candidate: 2.4.52`,
			expected: "1.20.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAptAvailableVersion(tt.output)
			if result != tt.expected {
				t.Errorf("parseAptAvailableVersion() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseAptRepository(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name: "valid repository line",
			output: `Package: nginx
Version: 1.18.0-0ubuntu1
APT-Sources: http://archive.ubuntu.com/ubuntu focal/main amd64 Packages
Description: small, powerful, scalable web/proxy server`,
			expected: "http://archive.ubuntu.com/ubuntu focal/main amd64 Packages",
		},
		{
			name: "repository at start",
			output: `APT-Sources: http://ppa.launchpad.net/nginx/stable/ubuntu focal/main amd64 Packages
Package: nginx
Version: 1.20.1-1`,
			expected: "http://ppa.launchpad.net/nginx/stable/ubuntu focal/main amd64 Packages",
		},
		{
			name: "no repository line",
			output: `Package: nginx
Version: 1.18.0-0ubuntu1
Description: web server`,
			expected: "",
		},
		{
			name:     "empty output",
			output:   "",
			expected: "",
		},
		{
			name: "empty repository value",
			output: `Package: nginx
APT-Sources:
Version: 1.18.0`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAptRepository(tt.output)
			if result != tt.expected {
				t.Errorf("parseAptRepository() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseAptDependencies(t *testing.T) {
	tests := []struct {
		name     string
		deps     string
		expected []string
	}{
		{
			name:     "single dependency",
			deps:     "libc6",
			expected: []string{"libc6"},
		},
		{
			name:     "multiple dependencies",
			deps:     "libc6, libssl1.1, zlib1g",
			expected: []string{"libc6", "libssl1.1", "zlib1g"},
		},
		{
			name:     "dependencies with version constraints",
			deps:     "libc6 (>= 2.27), libssl1.1 (>= 1.1.0), zlib1g (>= 1:1.2.0)",
			expected: []string{"libc6", "libssl1.1", "zlib1g"},
		},
		{
			name:     "mixed dependencies",
			deps:     "libc6 (>= 2.27), libssl1.1, zlib1g (>= 1:1.2.0)",
			expected: []string{"libc6", "libssl1.1", "zlib1g"},
		},
		{
			name:     "empty string",
			deps:     "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			deps:     "   ",
			expected: nil,
		},
		{
			name:     "trailing comma",
			deps:     "libc6, libssl1.1,",
			expected: []string{"libc6", "libssl1.1"},
		},
		{
			name:     "extra whitespace",
			deps:     "libc6  ,   libssl1.1   ,  zlib1g",
			expected: []string{"libc6", "libssl1.1", "zlib1g"},
		},
		{
			name:     "complex version constraints",
			deps:     "libc6 (>= 2.27), libssl1.1 (<< 1.2.0), zlib1g (= 1:1.2.11)",
			expected: []string{"libc6", "libssl1.1", "zlib1g"},
		},
		{
			name:     "single dependency with version",
			deps:     "nginx-common (= 1.18.0-0ubuntu1)",
			expected: []string{"nginx-common"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAptDependencies(tt.deps)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseAptDependencies() = %v, want %v", result, tt.expected)
			}
		})
	}
}
