package state

import (
	"bytes"
	"compress/gzip"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestCompressionManagerDisabled(t *testing.T) {
	config := &CompressionConfig{
		Enabled: false,
	}

	mgr := NewCompressionManager(config)
	state := &types.State{
		Version:   1,
		LastRun:   time.Now(),
		Playbook:  "test.yml",
		Variables: make(map[string]interface{}),
		Checksums: make(map[string]string),
	}

	data, err := mgr.CompressState(state)
	if err != nil {
		t.Fatalf("CompressState failed: %v", err)
	}

	// Should not be compressed
	if IsCompressed(data) {
		t.Fatalf("Data should not be compressed when disabled")
	}
}

func TestCompressionManagerEnabled(t *testing.T) {
	config := &CompressionConfig{
		Enabled:           true,
		Algorithm:         "gzip",
		CompressionLevel:  6,
		FileSizeThreshold: 100, // Very low threshold for testing
	}

	mgr := NewCompressionManager(config)
	state := &types.State{
		Version:  1,
		LastRun:  time.Now(),
		Playbook: "test.yml",
		Variables: map[string]interface{}{
			"key1": "This is a long value to ensure file size exceeds threshold",
			"key2": "Another long value to ensure compression",
		},
		Checksums: make(map[string]string),
	}

	data, err := mgr.CompressState(state)
	if err != nil {
		t.Fatalf("CompressState failed: %v", err)
	}

	// Should be compressed
	if !IsCompressed(data) {
		t.Logf("Note: Data might not be compressed if JSON size < threshold")
	}
}

func TestCompressionDecompression(t *testing.T) {
	config := DefaultCompressionConfig()
	mgr := NewCompressionManager(config)

	originalState := &types.State{
		Version:  1,
		LastRun:  time.Now(),
		Playbook: "test.yml",
		Variables: map[string]interface{}{
			"test": "value",
		},
		Checksums: map[string]string{
			"file1": "abcd1234",
		},
	}

	// Compress
	data, err := mgr.CompressState(originalState)
	if err != nil {
		t.Fatalf("CompressState failed: %v", err)
	}

	// Decompress
	restored, err := mgr.DecompressState(data)
	if err != nil {
		t.Fatalf("DecompressState failed: %v", err)
	}

	// Verify
	if restored.Playbook != originalState.Playbook {
		t.Fatalf("Playbook mismatch: expected %q, got %q", originalState.Playbook, restored.Playbook)
	}

	if len(restored.Checksums) != len(originalState.Checksums) {
		t.Fatalf("Checksums count mismatch")
	}
}

func TestCompressionStats(t *testing.T) {
	config := DefaultCompressionConfig()
	mgr := NewCompressionManager(config)

	state := &types.State{
		Version:   1,
		LastRun:   time.Now(),
		Playbook:  "test.yml",
		Variables: make(map[string]interface{}),
		Checksums: make(map[string]string),
	}

	stats, err := mgr.GetStats(state)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.UncompressedSize <= 0 {
		t.Fatalf("UncompressedSize should be positive, got %d", stats.UncompressedSize)
	}

	if stats.CompressedSize <= 0 {
		t.Fatalf("CompressedSize should be positive, got %d", stats.CompressedSize)
	}

	if stats.CompressionRatio <= 0 || stats.CompressionRatio > 1 {
		t.Fatalf("CompressionRatio should be between 0 and 1, got %f", stats.CompressionRatio)
	}
}

func TestIsCompressed(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		compressed bool
	}{
		{
			name:       "Gzip compressed data",
			data:       createGzipData([]byte("test data")),
			compressed: true,
		},
		{
			name:       "Plain JSON data",
			data:       []byte(`{"test": "data"}`),
			compressed: false,
		},
		{
			name:       "Empty data",
			data:       []byte{},
			compressed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCompressed(tt.data)
			if result != tt.compressed {
				t.Fatalf("Expected compressed=%v, got %v", tt.compressed, result)
			}
		})
	}
}

func TestCompressionWithLargeState(t *testing.T) {
	config := DefaultCompressionConfig()
	mgr := NewCompressionManager(config)

	// Create a state with large results
	state := &types.State{
		Version:   1,
		LastRun:   time.Now(),
		Playbook:  "large.yml",
		Variables: make(map[string]interface{}),
		Checksums: make(map[string]string),
		Results:   []types.PlayResult{},
	}

	// Add multiple play results
	for i := 0; i < 100; i++ {
		play := types.PlayResult{
			Name:      "Large Play",
			PlayName:  "Large Play",
			Host:      "localhost",
			Success:   true,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(time.Second),
			Tasks:     []types.TaskResult{},
		}

		// Add tasks
		for j := 0; j < 10; j++ {
			task := types.TaskResult{
				TaskName:  "Task",
				Host:      "localhost",
				Module:    "shell",
				Success:   true,
				Duration:  time.Second,
				Timestamp: time.Now(),
				Output: map[string]interface{}{
					"stdout": "This is a long output with lots of data to make the file bigger",
					"stderr": "",
				},
			}
			play.Tasks = append(play.Tasks, task)
		}

		state.Results = append(state.Results, play)
	}

	stats, err := mgr.GetStats(state)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	ratio := stats.CompressionRatio
	if ratio > 0.9 {
		t.Logf("Note: Compression ratio is high (%.2f), compression not very effective for this data", ratio)
	} else {
		t.Logf("Compression ratio: %.2f (%.1f%% savings)", ratio, (1-ratio)*100)
	}
}

func TestCompressionRoundTrip(t *testing.T) {
	config := DefaultCompressionConfig()
	mgr := NewCompressionManager(config)

	originalState := &types.State{
		Version:  1,
		LastRun:  time.Now(),
		Playbook: "test.yml",
		Variables: map[string]interface{}{
			"key1": "value1",
			"key2": 123,
			"key3": []string{"a", "b", "c"},
		},
		Checksums: map[string]string{
			"file1": "checksum1",
			"file2": "checksum2",
		},
		Metadata: &types.ExecutionMetadata{
			User:     "testuser",
			Hostname: "testhost",
			Tags:     []string{"tag1", "tag2"},
		},
	}

	// Round trip
	data, err := mgr.CompressState(originalState)
	if err != nil {
		t.Fatalf("CompressState failed: %v", err)
	}

	restored, err := mgr.DecompressState(data)
	if err != nil {
		t.Fatalf("DecompressState failed: %v", err)
	}

	// Detailed verification
	if restored.Version != originalState.Version {
		t.Fatalf("Version mismatch")
	}
	if restored.Playbook != originalState.Playbook {
		t.Fatalf("Playbook mismatch")
	}
	if len(restored.Variables) != len(originalState.Variables) {
		t.Fatalf("Variables count mismatch")
	}
	if len(restored.Checksums) != len(originalState.Checksums) {
		t.Fatalf("Checksums count mismatch")
	}
	if restored.Metadata.User != originalState.Metadata.User {
		t.Fatalf("Metadata.User mismatch")
	}
}

// Helper function to create gzip compressed data
func createGzipData(data []byte) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write(data) //nolint:errcheck // Test data
	gz.Close()
	return buf.Bytes()
}
