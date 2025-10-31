package facts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
)

// TestGathererCreation tests gatherer instantiation
func TestGathererCreation(t *testing.T) {
	g := NewGatherer()
	assert.NotNil(t, g)
	assert.NotNil(t, g.cache)
}

// TestCacheOperations tests basic cache operations
func TestCacheOperations(t *testing.T) {
	g := NewGatherer()

	// Set and get
	fact := &cache.SystemFacts{Hostname: "test"}
	g.cache.Set("host1", fact)
	retrieved, found := g.cache.Get("host1")

	assert.True(t, found)
	assert.Equal(t, "test", retrieved.Hostname)
}

// TestIPValidation tests IP address validation
func TestIPValidation(t *testing.T) {
	testCases := []struct {
		name    string
		ip      string
		isValid bool
	}{
		{"valid_ipv4", "192.168.1.1", true},
		{"loopback", "127.0.0.1", true},
		{"invalid_range", "256.1.1.1", false},
		{"empty", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidIP(tc.ip)
			assert.Equal(t, tc.isValid, result)
		})
	}
}

func isValidIP(ip string) bool {
	if ip == "" {
		return false
	}

	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}

	for _, part := range parts {
		num := 0
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
			num = num*10 + int(c-'0')
		}
		if num < 0 || num > 255 {
			return false
		}
	}
	return true
}

// BenchmarkGatherer benchmarks gatherer creation
func BenchmarkGatherer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewGatherer()
	}
}
