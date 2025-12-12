package modules

import (
	"testing"
)

// TestGenerateStateHash tests the generateStateHash helper function
func TestGenerateStateHash(t *testing.T) {
	tests := []struct {
		name       string
		pkgName    string
		version    string
		repository string
		wantSame   bool
		compareTo  struct {
			pkgName    string
			version    string
			repository string
		}
	}{
		{
			name:       "identical inputs produce same hash",
			pkgName:    "nginx",
			version:    "1.18.0",
			repository: "main",
			wantSame:   true,
			compareTo: struct {
				pkgName    string
				version    string
				repository string
			}{
				pkgName:    "nginx",
				version:    "1.18.0",
				repository: "main",
			},
		},
		{
			name:       "different package names produce different hashes",
			pkgName:    "nginx",
			version:    "1.18.0",
			repository: "main",
			wantSame:   false,
			compareTo: struct {
				pkgName    string
				version    string
				repository string
			}{
				pkgName:    "apache2",
				version:    "1.18.0",
				repository: "main",
			},
		},
		{
			name:       "different versions produce different hashes",
			pkgName:    "nginx",
			version:    "1.18.0",
			repository: "main",
			wantSame:   false,
			compareTo: struct {
				pkgName    string
				version    string
				repository string
			}{
				pkgName:    "nginx",
				version:    "1.20.0",
				repository: "main",
			},
		},
		{
			name:       "different repositories produce different hashes",
			pkgName:    "nginx",
			version:    "1.18.0",
			repository: "main",
			wantSame:   false,
			compareTo: struct {
				pkgName    string
				version    string
				repository string
			}{
				pkgName:    "nginx",
				version:    "1.18.0",
				repository: "universe",
			},
		},
		{
			name:       "empty values produce consistent hash",
			pkgName:    "",
			version:    "",
			repository: "",
			wantSame:   true,
			compareTo: struct {
				pkgName    string
				version    string
				repository string
			}{
				pkgName:    "",
				version:    "",
				repository: "",
			},
		},
		{
			name:       "hash is deterministic",
			pkgName:    "postgresql",
			version:    "13.4",
			repository: "main",
			wantSame:   true,
			compareTo: struct {
				pkgName    string
				version    string
				repository string
			}{
				pkgName:    "postgresql",
				version:    "13.4",
				repository: "main",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := generateStateHash(tt.pkgName, tt.version, tt.repository)
			hash2 := generateStateHash(tt.compareTo.pkgName, tt.compareTo.version, tt.compareTo.repository)

			// Verify hash is not empty
			if hash1 == "" {
				t.Error("generateStateHash returned empty string")
			}

			// Verify hash is hex string (64 characters for SHA256)
			if len(hash1) != 64 {
				t.Errorf("Expected hash length 64, got %d", len(hash1))
			}

			// Verify hash contains only hex characters
			for _, c := range hash1 {
				if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
					t.Errorf("Hash contains non-hex character: %c", c)
				}
			}

			// Compare hashes based on expectation
			if tt.wantSame {
				if hash1 != hash2 {
					t.Errorf("Expected same hash for identical inputs, got:\n  hash1: %s\n  hash2: %s", hash1, hash2)
				}
			} else {
				if hash1 == hash2 {
					t.Errorf("Expected different hashes for different inputs, but both are: %s", hash1)
				}
			}
		})
	}

	// Additional test: verify hash is deterministic
	t.Run("hash is deterministic across multiple calls", func(t *testing.T) {
		hash1 := generateStateHash("test-package", "1.0.0", "test-repo")
		hash2 := generateStateHash("test-package", "1.0.0", "test-repo")
		hash3 := generateStateHash("test-package", "1.0.0", "test-repo")

		if hash1 != hash2 || hash2 != hash3 {
			t.Errorf("Hash function is not deterministic:\n  hash1: %s\n  hash2: %s\n  hash3: %s", hash1, hash2, hash3)
		}

		// Verify it's a valid hex string of correct length
		if len(hash1) != 64 {
			t.Errorf("Expected hash length 64, got %d", len(hash1))
		}
	})
}
