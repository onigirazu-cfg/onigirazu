package checksum

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestManager_ComputeFile tests file checksum computation
func TestManager_ComputeFile(t *testing.T) {
	testCases := []struct {
		name    string
		content []byte
		desc    string
	}{
		{
			name:    "empty file",
			content: []byte{},
			desc:    "empty content",
		},
		{
			name:    "simple text",
			content: []byte("hello world"),
			desc:    "simple ASCII text",
		},
		{
			name:    "binary content",
			content: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
			desc:    "binary data",
		},
		{
			name:    "large content",
			content: bytes.Repeat([]byte("a"), 10000),
			desc:    "large repeated content",
		},
		{
			name:    "multiline text",
			content: []byte("line1\nline2\nline3\n"),
			desc:    "multiline text",
		},
		{
			name:    "json content",
			content: []byte(`{"key": "value", "number": 42}`),
			desc:    "JSON data",
		},
		{
			name:    "yaml content",
			content: []byte("key: value\nnumber: 42\n"),
			desc:    "YAML data",
		},
	}

	manager := NewManager(SHA256, 5*time.Minute)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "testfile")

			// Write test content
			err := os.WriteFile(filePath, tc.content, 0644)
			require.NoError(t, err, "Failed to write test file: %s", tc.desc)

			// Compute checksum
			result, err := manager.ComputeFile(filePath)
			assert.NoError(t, err, "Failed to compute checksum for: %s", tc.desc)
			assert.NotNil(t, result, "Result should not be nil for: %s", tc.desc)
			assert.NotEmpty(t, result.Checksum, "Checksum should not be empty for: %s", tc.desc)

			// Verify checksum is valid hex
			assert.Len(t, result.Checksum, 64, "SHA256 checksum should be 64 hex characters for: %s", tc.desc)
			for _, c := range result.Checksum {
				assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
					"Checksum should be lowercase hex for: %s", tc.desc)
			}

			// Verify metadata
			assert.Equal(t, SHA256, result.Algorithm, "Should use SHA256 algorithm")
			assert.Equal(t, int64(len(tc.content)), result.Size, "Size should match content length")
			assert.False(t, result.Computed.IsZero(), "Computed timestamp should be set")
		})
	}
}

// TestManager_ComputeFile_Deterministic tests that checksum is deterministic
func TestManager_ComputeFile_Deterministic(t *testing.T) {
	manager := NewManager(SHA256, 5*time.Minute)
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	content := []byte("deterministic test content")

	// Write content
	err := os.WriteFile(filePath, content, 0644)
	require.NoError(t, err)

	// Compute checksum multiple times
	result1, err := manager.ComputeFile(filePath)
	require.NoError(t, err)

	result2, err := manager.ComputeFile(filePath)
	require.NoError(t, err)

	result3, err := manager.ComputeFile(filePath)
	require.NoError(t, err)

	// All checksums should be identical
	assert.Equal(t, result1.Checksum, result2.Checksum, "Checksums should be identical on multiple runs")
	assert.Equal(t, result1.Checksum, result3.Checksum, "Checksums should be identical on multiple runs")
}

// TestManager_ComputeFile_DifferentContent tests that different content produces different checksums
func TestManager_ComputeFile_DifferentContent(t *testing.T) {
	manager := NewManager(SHA256, 5*time.Minute)
	tmpDir := t.TempDir()

	// Create two files with different content
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	err := os.WriteFile(file1, []byte("content 1"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(file2, []byte("content 2"), 0644)
	require.NoError(t, err)

	result1, err := manager.ComputeFile(file1)
	require.NoError(t, err)

	result2, err := manager.ComputeFile(file2)
	require.NoError(t, err)

	assert.NotEqual(t, result1.Checksum, result2.Checksum, "Different content should produce different checksums")
}

// TestManager_ComputeFile_NonexistentFile tests error handling for nonexistent files
func TestManager_ComputeFile_NonexistentFile(t *testing.T) {
	manager := NewManager(SHA256, 5*time.Minute)

	result, err := manager.ComputeFile("/nonexistent/path/to/file.txt")
	assert.Error(t, err, "Should return error for nonexistent file")
	assert.Nil(t, result, "Result should be nil on error")
}

// TestManager_ComputeFile_IsDirectory tests error handling for directories
func TestManager_ComputeFile_IsDirectory(t *testing.T) {
	manager := NewManager(SHA256, 5*time.Minute)
	tmpDir := t.TempDir()

	result, err := manager.ComputeFile(tmpDir)
	assert.Error(t, err, "Should return error for directory")
	assert.Nil(t, result, "Result should be nil on error")
}

// TestManager_ComputeFile_PermissionDenied tests error handling for permission denied
func TestManager_ComputeFile_PermissionDenied(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Test cannot run as root")
	}

	manager := NewManager(SHA256, 5*time.Minute)
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "restricted.txt")

	err := os.WriteFile(filePath, []byte("restricted content"), 0000)
	require.NoError(t, err)
	defer func() {
		_ = os.Chmod(filePath, 0644) // Restore permissions
	}()

	result, err := manager.ComputeFile(filePath)
	assert.Error(t, err, "Should return error for unreadable file")
	assert.Nil(t, result, "Result should be nil on error")
}

// TestManager_ComputeData tests checksum computation from data
func TestManager_ComputeData(t *testing.T) {
	manager := NewManager(SHA256, 5*time.Minute)

	testCases := []struct {
		name    string
		content []byte
		desc    string
	}{
		{
			name:    "empty data",
			content: []byte{},
			desc:    "empty content",
		},
		{
			name:    "text data",
			content: []byte("test data content"),
			desc:    "text content",
		},
		{
			name:    "binary data",
			content: []byte{0x00, 0x01, 0x02, 0xFF},
			desc:    "binary content",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := manager.ComputeData(tc.content)
			assert.NoError(t, err, "Failed to compute data checksum for: %s", tc.desc)
			assert.NotNil(t, result, "Result should not be nil")
			assert.NotEmpty(t, result.Checksum, "Checksum should not be empty")
			assert.Len(t, result.Checksum, 64, "SHA256 checksum should be 64 hex characters")
			assert.Equal(t, int64(len(tc.content)), result.Size, "Size should match data length")
		})
	}
}

// TestManager_ComputeData_ConsistentWithFile tests consistency between file and data
func TestManager_ComputeData_ConsistentWithFile(t *testing.T) {
	manager := NewManager(SHA256, 5*time.Minute)
	content := []byte("test content for consistency check")

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	// Write content
	err := os.WriteFile(filePath, content, 0644)
	require.NoError(t, err)

	// Get file checksum
	fileResult, err := manager.ComputeFile(filePath)
	require.NoError(t, err)

	// Get data checksum
	dataResult, err := manager.ComputeData(content)
	require.NoError(t, err)

	// Should be identical
	assert.Equal(t, fileResult.Checksum, dataResult.Checksum, "File and data checksums should be identical")
}

// TestManager_Verify tests checksum verification
func TestManager_Verify(t *testing.T) {
	manager := NewManager(SHA256, 5*time.Minute)
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content for verification")

	// Write content
	err := os.WriteFile(filePath, content, 0644)
	require.NoError(t, err)

	// Compute checksum
	result, err := manager.ComputeFile(filePath)
	require.NoError(t, err)

	// Verify correct checksum
	valid, err := manager.Verify(filePath, result.Checksum)
	assert.NoError(t, err, "Verification should not error for correct checksum")
	assert.True(t, valid, "Verification should succeed for correct checksum")

	// Verify incorrect checksum
	wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"
	valid, err = manager.Verify(filePath, wrongChecksum)
	assert.NoError(t, err, "Verification should not error for incorrect checksum")
	assert.False(t, valid, "Verification should fail for incorrect checksum")
}

// TestManager_VerifyData tests data checksum verification
func TestManager_VerifyData(t *testing.T) {
	manager := NewManager(SHA256, 5*time.Minute)
	content := []byte("test data for verification")

	// Compute checksum
	result, err := manager.ComputeData(content)
	require.NoError(t, err)

	// Verify correct checksum
	valid, err := manager.VerifyData(content, result.Checksum)
	assert.NoError(t, err, "Verification should not error for correct checksum")
	assert.True(t, valid, "Verification should succeed for correct checksum")

	// Verify incorrect checksum
	wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"
	valid, err = manager.VerifyData(content, wrongChecksum)
	assert.NoError(t, err, "Verification should not error for incorrect checksum")
	assert.False(t, valid, "Verification should fail for incorrect checksum")
}

// TestManager_Cache tests caching functionality
func TestManager_Cache(t *testing.T) {
	manager := NewManager(SHA256, 5*time.Minute)
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content for caching")

	// Write content
	err := os.WriteFile(filePath, content, 0644)
	require.NoError(t, err)

	// First computation (not cached)
	result1, err := manager.ComputeFile(filePath)
	require.NoError(t, err)

	// Second computation (should be cached)
	result2, err := manager.ComputeFile(filePath)
	require.NoError(t, err)

	// Results should be identical
	assert.Equal(t, result1.Checksum, result2.Checksum, "Cached checksums should be identical")

	// Modify file
	err = os.WriteFile(filePath, []byte("modified content"), 0644)
	require.NoError(t, err)

	// Cache should be invalidated on file modification
	result3, err := manager.ComputeFile(filePath)
	require.NoError(t, err)

	assert.NotEqual(t, result1.Checksum, result3.Checksum, "Checksum should change after file modification")
}

// TestManager_CacheStats tests cache statistics
func TestManager_CacheStats(t *testing.T) {
	manager := NewManager(SHA256, 5*time.Minute)
	tmpDir := t.TempDir()

	// Create and compute checksums for multiple files
	for i := 1; i <= 3; i++ {
		filePath := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		content := []byte("content " + string(rune('0'+i)))
		err := os.WriteFile(filePath, content, 0644)
		require.NoError(t, err)

		_, err = manager.ComputeFile(filePath)
		require.NoError(t, err)
	}

	// Check cache stats
	stats := manager.GetCacheStats()
	assert.NotNil(t, stats, "Cache stats should not be nil")
	assert.Equal(t, 3, stats.Entries, "Cache should have 3 entries")
}

// TestManager_ClearCache tests cache clearing
func TestManager_ClearCache(t *testing.T) {
	manager := NewManager(SHA256, 5*time.Minute)
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	// Write content and compute checksum
	err := os.WriteFile(filePath, []byte("test"), 0644)
	require.NoError(t, err)

	_, err = manager.ComputeFile(filePath)
	require.NoError(t, err)

	// Verify cache has entry
	stats := manager.GetCacheStats()
	assert.Equal(t, 1, stats.Entries, "Cache should have 1 entry before clearing")

	// Clear cache
	manager.ClearCache()

	// Verify cache is empty
	stats = manager.GetCacheStats()
	assert.Equal(t, 0, stats.Entries, "Cache should be empty after clearing")
}

// TestManager_ClearCacheEntry tests clearing individual cache entries
func TestManager_ClearCacheEntry(t *testing.T) {
	manager := NewManager(SHA256, 5*time.Minute)
	tmpDir := t.TempDir()

	// Create two files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	err := os.WriteFile(file1, []byte("content1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte("content2"), 0644)
	require.NoError(t, err)

	// Compute checksums
	_, err = manager.ComputeFile(file1)
	require.NoError(t, err)
	_, err = manager.ComputeFile(file2)
	require.NoError(t, err)

	// Verify cache has 2 entries
	stats := manager.GetCacheStats()
	assert.Equal(t, 2, stats.Entries, "Cache should have 2 entries")

	// Clear one entry
	manager.ClearCacheEntry(file1)

	// Verify cache has 1 entry
	stats = manager.GetCacheStats()
	assert.Equal(t, 1, stats.Entries, "Cache should have 1 entry after clearing one")
}

// TestManager_CleanupExpiredCache tests cleanup of expired cache entries
func TestManager_CleanupExpiredCache(t *testing.T) {
	manager := NewManager(SHA256, 100*time.Millisecond) // Very short TTL
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	// Write content and compute checksum
	err := os.WriteFile(filePath, []byte("test"), 0644)
	require.NoError(t, err)

	_, err = manager.ComputeFile(filePath)
	require.NoError(t, err)

	// Verify cache has entry
	stats := manager.GetCacheStats()
	assert.Equal(t, 1, stats.Entries, "Cache should have 1 entry before cleanup")

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Cleanup expired entries
	removed := manager.CleanupExpiredCache()
	assert.Equal(t, 1, removed, "Should remove 1 expired entry")

	// Verify cache is empty
	stats = manager.GetCacheStats()
	assert.Equal(t, 0, stats.Entries, "Cache should be empty after cleanup")
}

// TestNewManager_DefaultValues tests that NewManager sets default values correctly
func TestNewManager_DefaultValues(t *testing.T) {
	manager := NewManager("", 0)
	require.NotNil(t, manager, "Manager should not be nil")

	// Verify defaults
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(filePath, []byte("test"), 0644)
	require.NoError(t, err)

	result, err := manager.ComputeFile(filePath)
	require.NoError(t, err)

	// Should use SHA256 by default
	assert.Equal(t, SHA256, result.Algorithm, "Should use SHA256 algorithm by default")
}

// BenchmarkManager_ComputeFile benchmarks file checksum computation
func BenchmarkManager_ComputeFile(b *testing.B) {
	manager := NewManager(SHA256, 5*time.Minute)
	tmpDir := b.TempDir()
	filePath := filepath.Join(tmpDir, "bench.txt")

	// Create a 1MB test file
	testContent := bytes.Repeat([]byte("benchmark test content "), 50000)
	err := os.WriteFile(filePath, testContent, 0644)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.ComputeFile(filePath)
		require.NoError(b, err)
	}
}

// BenchmarkManager_ComputeData benchmarks data checksum computation
func BenchmarkManager_ComputeData(b *testing.B) {
	manager := NewManager(SHA256, 5*time.Minute)
	testContent := bytes.Repeat([]byte("benchmark test content "), 50000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.ComputeData(testContent)
		require.NoError(b, err)
	}
}
